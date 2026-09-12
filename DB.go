package db

import (
	"crypto/tls"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os/user"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"apigo.cc/go/cast"
	"apigo.cc/go/config"
	"apigo.cc/go/crypto"
	"apigo.cc/go/id"
	"apigo.cc/go/log"
	"apigo.cc/go/redis"
	"apigo.cc/go/safe"
)

type Config struct {
	Type          string
	User          string
	Password      string
	pwd           *safe.SafeBuf
	Host          string
	ReadonlyHosts []string
	DB            string
	SSL           string
	tls           *tls.Config
	Args          string
	MaxOpens      int
	MaxIdles      int
	MaxLifeTime   int
	LogSlow       config.Duration
	Redis         string
	autoLocal     bool
	localUser     string
	logger        *log.Logger
}

type SSL struct {
	Ca       string
	Cert     string
	Key      string
	Insecure bool
}

type ConnectorFunc func(*Config, *safe.SafeBuf, *tls.Config) driver.Connector

var connectors = map[string]ConnectorFunc{}

func RegisterConnector(typ string, conn ConnectorFunc) {
	connectors[typ] = conn
}

var sqlite3PwdMatcher = regexp.MustCompile(`x'\w+'`)

func (dbInfo *Config) Dsn() string {
	args := make([]string, 0)
	if dbInfo.SSL != "" {
		args = append(args, "tls="+dbInfo.SSL)
	}
	if dbInfo.Args != "" {
		args = append(args, dbInfo.Args)
	}
	argsStr := ""
	if len(args) > 0 {
		argsStr = "&" + strings.Join(args, "&")
	}

	if isFileDB(dbInfo.Type) {
		argsStr = sqlite3PwdMatcher.ReplaceAllString(argsStr, "******")
		return fmt.Sprintf("%s://%s?logSlow=%s"+argsStr, dbInfo.Type, dbInfo.Host, dbInfo.LogSlow.TimeDuration())
	} else {
		hosts := []string{dbInfo.Host}
		if dbInfo.ReadonlyHosts != nil {
			hosts = append(hosts, dbInfo.ReadonlyHosts...)
		}
		return fmt.Sprintf("%s://%s:****@%s/%s?logSlow=%s"+argsStr, dbInfo.Type, dbInfo.User, strings.Join(hosts, ","), dbInfo.DB, dbInfo.LogSlow.TimeDuration())
	}
}

var fileDBs = map[string]bool{
	"sqlite":     true,
	"sqlite3":    true,
	"chai":       true,
	"access":     true,
	"mdb":        true,
	"accdb":      true,
	"h2":         true,
	"hsqldb":     true,
	"derby":      true,
	"sqlce":      true,
	"sdf":        true,
	"firebird":   true,
	"fdb":        true,
	"dbase":      true,
	"dbf":        true,
	"berkeleydb": true,
	"bdb":        true,
}

func isFileDB(typ string) bool {
	return fileDBs[typ]
}

func isNetworkDBShorthand(setting string) bool {
	switch setting {
	case "mysql", "pg", "pgsql", "postgres", "postgresql", "pgx":
		return true
	default:
		return false
	}
}

func (dbInfo *Config) ConfigureBy(setting string) {
	setting = strings.TrimSpace(setting)
	if isNetworkDBShorthand(setting) {
		setting += "://"
	}
	urlInfo, err := url.Parse(setting)
	if err != nil {
		if dbInfo.logger != nil {
			dbInfo.logger.Error(err.Error(), "url", setting)
		}
		return
	}

	dbInfo.Type = urlInfo.Scheme
	if isFileDB(dbInfo.Type) {
		dbInfo.Host = urlInfo.Host + urlInfo.Path
		dbInfo.DB = strings.SplitN(urlInfo.Host, ".", 2)[0]
	} else {
		if strings.ContainsRune(urlInfo.Host, ',') {
			a := strings.Split(urlInfo.Host, ",")
			dbInfo.Host = a[0]
			dbInfo.ReadonlyHosts = a[1:]
		} else {
			dbInfo.Host = urlInfo.Host
			dbInfo.ReadonlyHosts = nil
		}
		if len(urlInfo.Path) > 1 {
			dbInfo.DB = urlInfo.Path[1:]
		}
	}
	dbInfo.User = urlInfo.User.Username()
	dbInfo.Password, _ = urlInfo.User.Password()
	if isPostgresType(dbInfo.Type) {
		if dbInfo.Host == "" {
			dbInfo.Host = "127.0.0.1:5432"
		} else if urlInfo.Port() == "" {
			dbInfo.Host = net.JoinHostPort(urlInfo.Hostname(), "5432")
		}
		if dbInfo.DB == "" {
			dbInfo.DB = "apigo"
		}
		if dbInfo.User == "" {
			if current, currentErr := user.Current(); currentErr == nil {
				dbInfo.User = current.Username
			}
		}
	} else if dbInfo.Type == "mysql" {
		dbInfo.autoLocal = urlInfo.Host == "" && dbInfo.User == ""
		if dbInfo.Host == "" {
			dbInfo.Host = "127.0.0.1:3306"
		} else if urlInfo.Port() == "" {
			dbInfo.Host = net.JoinHostPort(urlInfo.Hostname(), "3306")
		}
		if dbInfo.DB == "" {
			dbInfo.DB = "apigo"
		}
		if dbInfo.User == "" {
			dbInfo.User = "apigo"
			if current, currentErr := user.Current(); currentErr == nil {
				dbInfo.localUser = current.Username
			}
		}
	}

	q := urlInfo.Query()
	dbInfo.MaxIdles = cast.Int(q.Get("maxIdles"))
	dbInfo.MaxLifeTime = cast.Int(q.Get("maxLifeTime"))
	dbInfo.MaxOpens = cast.Int(q.Get("maxOpens"))
	dbInfo.LogSlow = config.Duration(cast.Duration(q.Get("logSlow")))
	dbInfo.Redis = q.Get("redis")
	dbInfo.SSL = q.Get("tls")

	sslCa := q.Get("sslCA")
	sslCert := q.Get("sslCert")
	sslKey := q.Get("sslKey")
	sslSkipVerify := cast.Bool(q.Get("sslSkipVerify"))
	if sslCa == "" || sslCert == "" || sslKey == "" {
		if dbSSL, ok := dbSSLs[dbInfo.SSL]; ok {
			sslCa = dbSSL.Ca
			sslCert = dbSSL.Cert
			sslKey = dbSSL.Key
			sslSkipVerify = dbSSL.Insecure
		}
	}
	if sslCa != "" && sslCert != "" && sslKey != "" {
		sslName := id.Get12BytesUltraPerSecond()
		dbInfo.SSL = sslName
		decryptedCa, _ := confAES.DecryptBytes([]byte(sslCa))
		decryptedCert, _ := confAES.DecryptBytes([]byte(sslCert))
		decryptedKey, _ := confAES.DecryptBytes([]byte(sslKey))
		tlsConf := BuildTLSConfig(decryptedCa, decryptedCert, decryptedKey, sslSkipVerify)
		if tlsConf != nil {
			dbInfo.tls = tlsConf
		}
		safe.ZeroMemory(decryptedCa)
		safe.ZeroMemory(decryptedCert)
		safe.ZeroMemory(decryptedKey)
	}

	args := make([]string, 0)
	for k := range q {
		if k != "maxIdles" && k != "maxLifeTime" && k != "maxOpens" && k != "logSlow" && k != "tls" && k != "redis" {
			args = append(args, k+"="+q.Get(k))
		}
	}
	if len(args) > 0 {
		dbInfo.Args = strings.Join(args, "&")
	}
}

// LocalAuthFallback reports whether an abbreviated local MySQL configuration
// may try the operating-system user over a local Unix socket.
func (dbInfo *Config) LocalAuthFallback() (string, bool) {
	return dbInfo.localUser, dbInfo.autoLocal && dbInfo.localUser != ""
}

type DB struct {
	schemaTx            *Tx // private transaction-bound schema runner
	name                string
	conn                *sql.DB
	readonlyConnections []*sql.DB
	Config              *Config
	logger              *dbLogger
	Error               error
	QuoteTag            string
	tables              map[string]*TableStruct
	tablesLock          *sync.RWMutex
	sqliteMu            *sync.Mutex // Serial lock for SQLite writers
}

type TableStruct struct {
	Name           string
	Indexes        []TableIndex
	indexError     error
	Comment        string
	Fields         []TableField
	Columns        []string
	ShadowDelete   bool
	HasShadowTable bool
	VersionField   string
	IdField        string
	IdSize         int
}

// TableIndex defines an independently ordered index. A leading '-' on a field
// selects descending order; names are generated from the complete definition.
type TableIndex struct {
	Type   string
	Fields []string
}

type TableField struct {
	parseError error
	Name       string
	Type       string
	Index      string
	IndexGroup string
	Default    string
	Comment    string
	Null       string
	Extra      string
	Desc       string
	IsVersion  bool
	IsObject   bool
}

var confAES *crypto.Symmetric

func init() {
	crypto.OnSetDefaultAES(func(aes *crypto.Symmetric) {
		confAES = aes
	})
}

func SetEncryptKeys(key, iv []byte) {
	crypto.SetDefaultAES(key, iv)
}

type dbLogger struct {
	config *Config
	logger *log.Logger
}

func (dl *dbLogger) LogError(errStr string) {
	dl.LogDB("", nil, 0, errors.New(errStr))
}

func (dl *dbLogger) LogQuery(query string, args []any, usedTime float32) {
	dl.LogDB(query, args, usedTime, nil)
}

func (dl *dbLogger) LogQueryError(errStr string, query string, args []any, usedTime float32) {
	dl.LogDB(query, args, usedTime, errors.New(errStr))
}

var dbConfigs = make(map[string]*Config)
var dbConfigsLock = sync.RWMutex{}
var dbSSLs = make(map[string]*SSL)
var dbInstances = make(map[string]*DB)
var dbInstancesLock = sync.RWMutex{}
var globalVersionMap = sync.Map{}
var globalIdMakers = sync.Map{}
var versionInited = sync.Map{}
var once sync.Once

func (db *DB) NextVersion(table string) int64 {
	ts := db.getTable(table)
	if ts.VersionField == "" {
		return 0
	}

	if _, inited := versionInited.Load(table); !inited {
		db.syncVersionFromDB(table, ts.VersionField)
		versionInited.Store(table, true)
	}

	if db.Config.Redis != "" {
		r := redis.GetRedis(db.Config.Redis, db.logger.logger)
		if r != nil {
			return r.INCR("db_ver_" + table)
		}
	}

	v, _ := globalVersionMap.LoadOrStore(table, new(int64))
	return atomic.AddInt64(v.(*int64), 1)
}

func (db *DB) NextID(table string) string {
	ts := db.getTable(table)
	if ts.IdField == "" || ts.IdSize == 0 {
		return ""
	}

	var maker *id.IDMaker
	if db.Config.Redis != "" {
		if v, ok := globalIdMakers.Load(db.Config.Redis); ok {
			maker = v.(*id.IDMaker)
		} else {
			r := redis.GetRedis(db.Config.Redis, db.logger.logger)
			if r != nil {
				maker = redis.NewIDMaker(r)
				globalIdMakers.Store(db.Config.Redis, maker)
			}
		}
	}

	if maker == nil {
		maker = id.DefaultIDMaker
	}

	switch db.Config.Type {
	case "mysql":
		return maker.GetForMysql(ts.IdSize)
	case "pg", "pgsql", "postgres", "postgresql", "pgx":
		return maker.GetForPostgreSQL(ts.IdSize)
	default:
		return maker.Get(ts.IdSize)
	}
}

func (db *DB) syncVersionFromDB(table, versionField string) {
	query := fmt.Sprintf("SELECT MAX(%s) FROM %s", db.Quote(versionField), db.Quote(table))
	maxVer := db.rawQuery(query).IntOnR1C1()

	if db.Config.Redis != "" {
		r := redis.GetRedis(db.Config.Redis, db.logger.logger)
		if r != nil {
			r.Do("SETNX", "db_ver_"+table, maxVer)
			return
		}
	}

	v, _ := globalVersionMap.LoadOrStore(table, new(int64))
	ptr := v.(*int64)
	for {
		current := atomic.LoadInt64(ptr)
		if current >= maxVer {
			break
		}
		if atomic.CompareAndSwapInt64(ptr, current, maxVer) {
			break
		}
	}
}

func GetDBWithoutCache(name string, logger *log.Logger) *DB {
	return getDB(name, logger, false)
}

func GetDB(name string, logger *log.Logger) *DB {
	return getDB(name, logger, true)
}

// Sync 同步数据库结构 (使用默认实例 "default")
func Sync(desc string) error {
	d := GetDB("default", nil)
	if d == nil {
		return errors.New("default db not configured")
	}
	return d.Sync(desc)
}

// Insert 插入数据 (使用默认实例 "default")
func Insert(table string, data any) *ExecResult {
	d := GetDB("default", nil)
	if d == nil {
		return &ExecResult{Error: errors.New("default db not configured")}
	}
	return d.Insert(table, data)
}

// Update 更新数据 (使用默认实例 "default")
func Update(table string, data any, conditions string, args ...any) *ExecResult {
	d := GetDB("default", nil)
	if d == nil {
		return &ExecResult{Error: errors.New("default db not configured")}
	}
	return d.Update(table, data, conditions, args...)
}

// Delete 删除数据 (使用默认实例 "default")
func Delete(table string, conditions string, args ...any) *ExecResult {
	d := GetDB("default", nil)
	if d == nil {
		return &ExecResult{Error: errors.New("default db not configured")}
	}
	return d.Delete(table, conditions, args...)
}

// Query 查询数据 (使用默认实例 "default")
func Query(query string, args ...any) *QueryResult {
	d := GetDB("default", nil)
	if d == nil {
		return &QueryResult{Error: errors.New("default db not configured")}
	}
	return d.Query(query, args...)
}

// Exec 执行 SQL (使用默认实例 "default")
func Exec(query string, args ...any) *ExecResult {
	d := GetDB("default", nil)
	if d == nil {
		return &ExecResult{Error: errors.New("default db not configured")}
	}
	return d.Exec(query, args...)
}

// Begin 开始事务 (使用默认实例 "default")
func Begin() *Tx {
	d := GetDB("default", nil)
	if d == nil {
		return &Tx{Error: errors.New("default db not configured")}
	}
	return d.Begin()
}

func SetConfig(name, setting string) {
	loadConfigs(nil)
	conf := new(Config)
	conf.ConfigureBy(setting)
	dbConfigsLock.Lock()
	dbConfigs[name] = conf
	dbConfigsLock.Unlock()
}

func loadConfigs(logger *log.Logger) {
	once.Do(func() {
		if logger == nil {
			logger = log.DefaultLogger
		}
		dbConfigs1 := make(map[string]*Config)
		if err := config.Load(&dbConfigs1, "db"); err == nil {
			for k, v := range dbConfigs1 {
				if v.Host != "" {
					dbConfigsLock.Lock()
					dbConfigs[k] = v
					dbConfigsLock.Unlock()
				}
			}
		} else {
			logger.Error(err.Error())
		}
		dbConfigs2 := make(map[string]string)
		if err := config.Load(&dbConfigs2, "db"); err == nil {
			for k, v := range dbConfigs2 {
				if strings.Contains(v, "://") {
					v2 := new(Config)
					v2.ConfigureBy(v)
					if v2.Host != "" {
						v2.logger = logger
						dbConfigsLock.Lock()
						dbConfigs[k] = v2
						dbConfigsLock.Unlock()
					}
				} else {
					dbConfigsLock.Lock()
					v2 := dbConfigs[v]
					if v2 != nil && v2.Host != "" {
						dbConfigs[k] = v2
					}
					dbConfigsLock.Unlock()
				}
			}
		} else {
			logger.Error(err.Error())
		}
	})
}

func getDB(name string, logger *log.Logger, useCache bool) *DB {
	if logger == nil {
		logger = log.DefaultLogger
	}

	if useCache {
		dbInstancesLock.RLock()
		oldConn := dbInstances[name]
		dbInstancesLock.RUnlock()
		if oldConn != nil {
			return oldConn.CopyByLogger(logger)
		}
	}

	var conf *Config
	if strings.Contains(name, "://") || isNetworkDBShorthand(name) {
		conf = new(Config)
		conf.logger = logger
		conf.ConfigureBy(name)
	} else {
		loadConfigs(logger)
		dbConfigsLock.RLock()
		conf = dbConfigs[name]
		dbConfigsLock.RUnlock()
		if conf == nil {
			conf = new(Config)
			dbConfigsLock.Lock()
			dbConfigs[name] = conf
			dbConfigsLock.Unlock()
		}
	}

	if conf.Host == "" {
		if name != "default" {
			logger.Error("db config not exists", "name", name)
		}
		return nil
	}

	if conf.SSL != "" && len(dbSSLs) == 0 {
		_ = config.Load(&dbSSLs, "dbssl")
	}

	if conf.SSL != "" && dbSSLs[conf.SSL] == nil {
		logger.Error("dbssl config lost")
	}

	if strings.ContainsRune(conf.Host, ',') {
		a := strings.Split(conf.Host, ",")
		conf.Host = a[0]
		conf.ReadonlyHosts = a[1:]
	} else {
		conf.ReadonlyHosts = nil
	}

	if conf.Password != "" {
		if encryptedPassword, err := base64.URLEncoding.DecodeString(conf.Password); err == nil {
			if pwdSafeBuf, err := confAES.Decrypt(encryptedPassword); err == nil {
				conf.pwd = pwdSafeBuf
			}
		}
		if conf.pwd == nil {
			conf.pwd = safe.NewSafeBuf([]byte(conf.Password))
		}
	} else {
		if !isFileDB(conf.Type) && conf.Host != "127.0.0.1:3306" && conf.User == "root" {
			logger.Warning("password is empty")
		}
		conf.pwd = safe.NewSafeBuf(nil)
	}
	conf.Password = ""

	conn, err := getPool(conf)
	if err != nil {
		LogDB(logger, conf, "", nil, 0, err)
		return &DB{conn: nil, QuoteTag: "\"", Error: err}
	}

	db := new(DB)
	db.QuoteTag = cast.If(conf.Type == "mysql", "`", "\"")
	db.name = name
	db.conn = conn
	db.tables = make(map[string]*TableStruct)
	db.tablesLock = new(sync.RWMutex)
	if conf.Type == "sqlite" || conf.Type == "sqlite3" {
		db.sqliteMu = new(sync.Mutex)
	}

	if conf.ReadonlyHosts != nil {
		readonlyConnections := make([]*sql.DB, 0)
		for _, host := range conf.ReadonlyHosts {
			conn, err := getPoolForHost(conf, host)
			if err != nil {
				LogDB(logger, conf, "", nil, 0, err)
			} else {
				readonlyConnections = append(readonlyConnections, conn)
			}
		}
		if len(readonlyConnections) > 0 {
			db.readonlyConnections = readonlyConnections
		}
	}

	db.Error = nil
	db.Config = conf
	if (conf.Type == "sqlite" || conf.Type == "sqlite3") && conf.MaxOpens == 0 {
		conf.MaxOpens = 100
	}
	if conf.MaxIdles > 0 {
		conn.SetMaxIdleConns(conf.MaxIdles)
	}
	if conf.MaxOpens > 0 {
		conn.SetMaxOpenConns(conf.MaxOpens)
	}
	if conf.MaxLifeTime > 0 {
		conn.SetConnMaxLifetime(time.Second * time.Duration(conf.MaxLifeTime))
	}
	if conf.LogSlow == 0 {
		conf.LogSlow = config.Duration(1000 * time.Millisecond)
	}

	if conf.Type == "sqlite" || conf.Type == "sqlite3" {
		baseExecRaw(conn, nil, "PRAGMA journal_mode=WAL")
		baseExecRaw(conn, nil, "PRAGMA synchronous=NORMAL")
		baseExecRaw(conn, nil, "PRAGMA busy_timeout=5000")
		baseExecRaw(conn, nil, "PRAGMA temp_store=MEMORY")
		baseExecRaw(conn, nil, "PRAGMA cache_size=-2000")

		// Dynamic mmap_size: 1/4 of system memory, max 30GB
		mmapLimit := int64(30000000000)
		sysMemStr := runShell("sysctl -n hw.memsize || free -b | awk '/Mem:/ {print $2}'")
		if sysMem := cast.Int64(sysMemStr); sysMem > 0 {
			if mmapLimit > sysMem/4 {
				mmapLimit = sysMem / 4
			}
		}
		baseExecRaw(conn, nil, fmt.Sprintf("PRAGMA mmap_size=%d", mmapLimit))
	}

	if useCache {
		dbInstancesLock.Lock()
		dbInstances[name] = db
		dbInstancesLock.Unlock()
	}
	return db.CopyByLogger(logger)
}

func getPool(conf *Config) (*sql.DB, error) {
	return getPoolForHost(conf, "")
}

func getPoolForHost(conf *Config, host string) (*sql.DB, error) {
	connectType := "tcp"
	if host == "" {
		host = conf.Host
	}
	if len(host) > 0 && host[0] == '/' {
		connectType = "unix"
	}

	if connector := connectors[conf.Type]; connector != nil {
		return sql.OpenDB(connector(conf, conf.pwd, conf.tls)), nil
	} else {
		if (conf.Type == "sqlite" || conf.Type == "sqlite3") && !strings.Contains(conf.Args, "journal_mode") {
			if conf.Args != "" {
				conf.Args += "&"
			}
			conf.Args += "_journal_mode=WAL&_busy_timeout=5000&_pragma=synchronous(1)&_pragma=cache_size(-2000)"
		}

		dsn := ""
		args := make([]string, 0)
		if conf.SSL != "" {
			args = append(args, "tls="+conf.SSL)
		}
		if conf.Args != "" {
			args = append(args, conf.Args)
		}
		argsStr := ""
		if len(args) > 0 {
			argsStr = "?" + strings.Join(args, "&")
		}

		if isFileDB(conf.Type) {
			dsn = host + argsStr
		} else {
			pwdBuf := conf.pwd.Open()
			dsn = fmt.Sprintf("%s:%s@%s(%s)/%s"+argsStr, conf.User, pwdBuf.String(), connectType, host, conf.DB)
			pwdBuf.Close()
		}
		return sql.Open(conf.Type, dsn)
	}
}

func (db *DB) CopyByLogger(logger *log.Logger) *DB {
	newDB := new(DB)
	newDB.QuoteTag = db.QuoteTag
	newDB.name = db.name
	newDB.conn = db.conn
	newDB.readonlyConnections = db.readonlyConnections
	newDB.Config = db.Config
	newDB.tables = db.tables
	newDB.tablesLock = db.tablesLock
	newDB.sqliteMu = db.sqliteMu
	if logger == nil {
		logger = log.DefaultLogger
	}
	newDB.logger = &dbLogger{logger: logger, config: db.Config}
	return newDB
}

func (db *DB) SetLogger(logger *log.Logger) {
	db.logger.logger = logger
}

func (db *DB) GetLogger() *log.Logger {
	return db.logger.logger
}

func (db *DB) Destroy() error {
	if db.conn == nil {
		return errors.New("operate on a bad connection")
	}
	err := db.conn.Close()
	if err != nil {
		db.logger.LogError(err.Error())
	}
	dbInstancesLock.Lock()
	delete(dbInstances, db.name)
	dbInstancesLock.Unlock()
	return err
}

func (db *DB) GetOriginDB() *sql.DB {
	return db.conn
}

func (db *DB) Prepare(query string) *Stmt {
	query = db.prepareQuery(query)
	stmt := basePrepare(db.conn, nil, query)
	stmt.logger = db.logger
	stmt.sqliteMu = db.sqliteMu
	if stmt.Error != nil {
		db.logger.LogError(stmt.Error.Error())
	}
	return stmt
}

func (db *DB) Quote(text string) string {
	return quote(db.QuoteTag, text)
}

func (db *DB) Quotes(texts []string) string {
	return quotes(db.QuoteTag, texts)
}

func (db *DB) Begin() *Tx {
	if db.conn == nil {
		return &Tx{db: db, sqliteMu: db.sqliteMu, QuoteTag: db.QuoteTag, logSlow: db.Config.LogSlow.TimeDuration(), Error: errors.New("operate on a bad connection"), logger: db.logger}
	}
	sqlTx, err := db.conn.Begin()
	if err != nil {
		db.logger.LogError(err.Error())
		return &Tx{db: db, sqliteMu: db.sqliteMu, QuoteTag: db.QuoteTag, logSlow: db.Config.LogSlow.TimeDuration(), Error: err, logger: db.logger}
	}
	return &Tx{db: db, sqliteMu: db.sqliteMu, QuoteTag: db.QuoteTag, logSlow: db.Config.LogSlow.TimeDuration(), conn: sqlTx, logger: db.logger}
}

func (db *DB) Exec(query string, args ...any) *ExecResult {
	if db.schemaTx != nil {
		return db.schemaTx.Exec(query, args...)
	}
	query, args = db.rewriteFTS(query, args)
	query = db.prepareQuery(query)
	args = flatArgs(args)
	db.lock()
	r := baseExecRaw(db.conn, nil, query, args...)
	db.unlock()
	r.logger = db.logger
	if r.Error != nil {
		db.logger.LogQueryError(r.Error.Error(), query, args, r.usedTime)
	} else {
		if db.Config.LogSlow > 0 && r.usedTime >= float32(db.Config.LogSlow.TimeDuration()/time.Millisecond) {
			db.logger.LogQuery(query, args, r.usedTime)
		}
	}
	return r
}

func (db *DB) rawExec(query string, args ...any) *ExecResult {
	return db.Exec(query, args...)
}

func (db *DB) Query(query string, args ...any) *QueryResult {
	if db.schemaTx != nil {
		return db.schemaTx.Query(query, args...)
	}
	query, args = db.rewriteFTS(query, args)
	query = db.prepareQuery(query)
	args = flatArgs(args)
	r := baseQueryRaw(db.conn, nil, query, args...)
	r.logger = db.logger
	if r.Error != nil {
		db.logger.LogQueryError(r.Error.Error(), query, args, r.usedTime)
	} else {
		if db.Config.LogSlow > 0 && r.usedTime >= float32(db.Config.LogSlow.TimeDuration()/time.Millisecond) {
			db.logger.LogQuery(query, args, r.usedTime)
		}
	}
	return r
}

func (db *DB) rawQuery(query string, args ...any) *QueryResult {
	return db.Query(query, args...)
}

func (db *DB) lock() {
	if db.sqliteMu != nil {
		db.sqliteMu.Lock()
	}
}

func (db *DB) unlock() {
	if db.sqliteMu != nil {
		db.sqliteMu.Unlock()
	}
}

var identifierRegex = `(?:['"` + "`" + `][^'"` + "`" + `]+['"` + "`" + `]|[\w\-]+)`
var likeFieldReg = regexp.MustCompile(`(?i)(` + identifierRegex + `(?:\.` + identifierRegex + `)*)\s+LIKE\s*$`)
var likeLiteralReg = regexp.MustCompile(`(?i)(` + identifierRegex + `(?:\.` + identifierRegex + `)*)\s+LIKE\s+(['"])(%?[^'"]*?%?)(['"])`)

func cleanIdentifier(s string) string {
	parts := strings.Split(s, ".")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) >= 2 {
			if (p[0] == '`' && p[len(p)-1] == '`') ||
				(p[0] == '"' && p[len(p)-1] == '"') ||
				(p[0] == '\'' && p[len(p)-1] == '\'') ||
				(p[0] == '[' && p[len(p)-1] == ']') {
				parts[i] = p[1 : len(p)-1]
				continue
			}
		}
		parts[i] = p
	}
	return strings.Join(parts, ".")
}

func (db *DB) rewriteFTS(query string, args []any) (string, []any) {
	// 1. 处理硬编码的 LIKE 'literal'
	query = likeLiteralReg.ReplaceAllStringFunc(query, func(m string) string {
		matches := likeLiteralReg.FindStringSubmatch(m)
		if matches[2] != matches[4] {
			return m // 引号不匹配，跳过
		}
		field := matches[1]
		quoteMark := matches[2]
		literal := matches[3]

		cleanField := cleanIdentifier(field)
		tableName := db.extractTableName(query, field)
		if tableName != "" {
			ts := db.getTable(tableName)
			colParts := strings.Split(cleanField, ".")
			colName := colParts[len(colParts)-1]
			tokensCol := colName + "_tokens"

			hasTokens := false
			for _, c := range ts.Columns {
				if c == tokensCol {
					hasTokens = true
					break
				}
			}

			if hasTokens {
				searchTerm := strings.Trim(literal, "% ")
				tokens := BigramTokenize(searchTerm)
				if db.Config.Type == "pg" || db.Config.Type == "pgsql" || db.Config.Type == "postgres" {
					tokens = strings.ReplaceAll(tokens, " ", " & ")
				}
				pre, suf := db.getFTSMatchSQLParts(query, field)
				return pre + quoteMark + tokens + quoteMark + suf
			}
		}
		return m
	})

	if len(args) == 0 || !strings.Contains(strings.ToUpper(query), " LIKE ") {
		return query, args
	}

	parts := strings.Split(query, "?")
	if len(parts)-1 != len(args) {
		// 存在误伤风险，安全降级
		return query, args
	}

	newArgs := make([]any, len(args))
	copy(newArgs, args)

	isModified := false
	for i := 0; i < len(args); i++ {
		match := likeFieldReg.FindStringSubmatch(parts[i])
		if len(match) > 1 {
			field := match[1]
			cleanField := cleanIdentifier(field)
			tableName := db.extractTableName(query, field)
			if tableName != "" {
				ts := db.getTable(tableName)
				colParts := strings.Split(cleanField, ".")
				colName := colParts[len(colParts)-1]
				tokensCol := colName + "_tokens"

				hasTokens := false
				for _, c := range ts.Columns {
					if c == tokensCol {
						hasTokens = true
						break
					}
				}

				if hasTokens {
					// 命中影子列，执行替换
					ftsPre, ftsSuf := db.getFTSMatchSQLParts(query, field)
					parts[i] = strings.Replace(parts[i], match[0], ftsPre, 1)
					parts[i+1] = ftsSuf + parts[i+1]

					// 处理参数
					searchTerm := cast.String(args[i])
					searchTerm = strings.Trim(searchTerm, "% ")
					tokens := BigramTokenize(searchTerm)

					if db.Config.Type == "pg" || db.Config.Type == "pgsql" || db.Config.Type == "postgres" {
						tokens = strings.ReplaceAll(tokens, " ", " & ")
					}
					newArgs[i] = tokens
					isModified = true
				}
			}
		}
	}

	if isModified {
		return strings.Join(parts, "?"), newArgs
	}

	return query, args
}

func (db *DB) getFTSMatchSQLParts(query string, field string) (string, string) {
	cleanField := cleanIdentifier(field)
	parts := strings.Split(cleanField, ".")
	colName := parts[len(parts)-1]

	// 保持原字段引用方式（带引号或别名）
	tokensField := field + "_tokens"
	lastPart := field
	prefix := ""
	if idx := strings.LastIndex(field, "."); idx != -1 {
		prefix = field[:idx+1]
		lastPart = field[idx+1:]
	}

	if len(lastPart) >= 2 && ((lastPart[0] == '`' && lastPart[len(lastPart)-1] == '`') ||
		(lastPart[0] == '"' && lastPart[len(lastPart)-1] == '"') ||
		(lastPart[0] == '[' && lastPart[len(lastPart)-1] == ']')) {
		tokensField = prefix + lastPart[:len(lastPart)-1] + "_tokens" + lastPart[len(lastPart)-1:]
	}

	switch db.Config.Type {
	case "mysql":
		return fmt.Sprintf("MATCH(%s) AGAINST(", tokensField), " IN BOOLEAN MODE)"
	case "pg", "pgsql", "postgres":
		return fmt.Sprintf("%s @@ to_tsquery('simple', ", tokensField), ")"
	case "sqlite", "sqlite3":
		tableName := db.extractTableName(query, field)
		idField := "id"
		ts := db.getTable(tableName)
		if ts.IdField != "" {
			idField = ts.IdField
		}
		prefix := ""
		dotParts := strings.Split(field, ".")
		if len(dotParts) > 1 {
			prefix = dotParts[0] + "."
		}
		return fmt.Sprintf("%s%s IN (SELECT rowid FROM \"%s_fts\" WHERE \"%s_tokens\" MATCH ", prefix, idField, tableName, colName), ")"
	default:
		return fmt.Sprintf("%s LIKE ", field), ""
	}
}

func (db *DB) extractTableName(query string, field string) string {
	cleanField := cleanIdentifier(field)
	parts := strings.Split(cleanField, ".")
	if len(parts) > 1 {
		alias := parts[0]
		reg := regexp.MustCompile(fmt.Sprintf(`(?i)FROM\s+(%s)\s+(?:AS\s+)?["\`+"`"+`]?%s["\`+"`"+`]?|JOIN\s+(%s)\s+(?:AS\s+)?["\`+"`"+`]?%s["\`+"`"+`]?`, identifierRegex, alias, identifierRegex, alias))
		match := reg.FindStringSubmatch(query)
		if len(match) > 1 {
			if match[1] != "" {
				return cleanIdentifier(match[1])
			}
			return cleanIdentifier(match[2])
		}
		return alias
	}
	reg := regexp.MustCompile(`(?i)FROM\s+(` + identifierRegex + `)`)
	match := reg.FindStringSubmatch(query)
	if len(match) > 1 {
		return cleanIdentifier(match[1])
	}
	return ""
}

func (db *DB) Insert(table string, data any) *ExecResult {
	query, values := db.MakeInsertSql(table, data, false)
	return db.Exec(query, values...)
}

func (db *DB) Replace(table string, data any) *ExecResult {
	query, values := db.MakeInsertSql(table, data, true)
	return db.Exec(query, values...)
}

func (db *DB) Update(table string, data any, conditions string, args ...any) *ExecResult {
	query, values := db.MakeUpdateSql(table, data, conditions, args...)
	return db.Exec(query, values...)
}

func (db *DB) Delete(table string, conditions string, args ...any) *ExecResult {
	ts := db.getTable(table)
	if !ts.HasShadowTable {
		if conditions != "" {
			conditions = " where " + conditions
		}
		query := fmt.Sprintf("delete from %s%s", db.Quote(table), conditions)
		return db.Exec(query, args...)
	}

	// Shadow delete
	tx := db.Begin()
	defer tx.CheckFinished()
	r := tx.Delete(table, conditions, args...)
	if r.Error == nil {
		tx.Commit()
	}
	return r
}

func (db *DB) getTable(table string) *TableStruct {
	db.tablesLock.RLock()
	ts, ok := db.tables[table]
	db.tablesLock.RUnlock()
	if ok {
		return ts
	}

	db.tablesLock.Lock()
	defer db.tablesLock.Unlock()

	// Double check
	if ts, ok = db.tables[table]; ok {
		return ts
	}

	ts = &TableStruct{Name: table}
	// Probe columns and autoVersion
	var query string
	if db.Config.Type == "mysql" {
		query = "SELECT COLUMN_NAME, DATA_TYPE, CHARACTER_MAXIMUM_LENGTH, COLUMN_KEY FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?"
		res := db.rawQuery(query, db.Config.DB, table)
		rows := res.MapResults()
		for _, row := range rows {
			col := cast.String(row["COLUMN_NAME"])
			dataType := cast.String(row["DATA_TYPE"])
			charLen := cast.Int(row["CHARACTER_MAXIMUM_LENGTH"])
			colKey := cast.String(row["COLUMN_KEY"])

			ts.Columns = append(ts.Columns, col)
			if col == "autoVersion" {
				ts.VersionField = "autoVersion"
			}
			if (colKey == "PRI" || colKey == "UNI") && strings.ToLower(dataType) == "char" && (charLen >= 8 && charLen <= 16) {
				ts.IdField = col
				ts.IdSize = charLen
			}
		}
	} else if db.Config.Type == "postgres" || db.Config.Type == "pgx" {
		query = "SELECT column_name, data_type, character_maximum_length FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ?"
		res := db.rawQuery(query, table)
		rows := res.MapResults()
		for _, row := range rows {
			col := cast.String(row["column_name"])
			dataType := cast.String(row["data_type"])
			charLen := cast.Int(row["character_maximum_length"])

			ts.Columns = append(ts.Columns, col)
			if col == "autoVersion" {
				ts.VersionField = "autoVersion"
			}
			// PostgreSQL PK/Unique check is complex, we use column name 'id' and char type as a heuristic or check constraints if needed.
			// To keep it simple and efficient as requested:
			if (col == "id" || col == "ID") && (strings.Contains(strings.ToLower(dataType), "char")) && (charLen >= 8 && charLen <= 16) {
				ts.IdField = col
				ts.IdSize = charLen
			}
		}
	} else if isFileDB(db.Config.Type) {
		// For SQLite
		query := fmt.Sprintf("PRAGMA table_info(%s)", db.Quote(table))
		res := db.rawQuery(query)
		rows := res.MapResults()
		for _, row := range rows {
			colName := cast.String(row["name"])
			colType := strings.ToUpper(cast.String(row["type"]))
			isPk := cast.Int(row["pk"]) > 0

			ts.Columns = append(ts.Columns, colName)
			if colName == "autoVersion" {
				ts.VersionField = "autoVersion"
			}

			if isPk && strings.Contains(colType, "CHAR") {
				// Extract length from CHAR(N)
				charLen := 0
				fmt.Sscanf(colType, "CHAR(%d)", &charLen)
				if charLen == 0 {
					fmt.Sscanf(colType, "CHARACTER(%d)", &charLen)
				}
				if charLen >= 8 && charLen <= 16 {
					ts.IdField = colName
					ts.IdSize = charLen
				}
			}
		}
	}

	// Probe shadow table
	shadowTable := table + "_deleted"
	if db.Config.Type == "mysql" {
		query = "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?"
		res := db.rawQuery(query, db.Config.DB, shadowTable)
		if res.StringOnR1C1() != "" {
			ts.HasShadowTable = true
		}
	} else if db.Config.Type == "postgres" || db.Config.Type == "pgx" {
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?"
		res := db.rawQuery(query, shadowTable)
		if res.StringOnR1C1() != "" {
			ts.HasShadowTable = true
		}
	} else if isFileDB(db.Config.Type) {
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
		res := db.rawQuery(query, shadowTable)
		if res.StringOnR1C1() != "" {
			ts.HasShadowTable = true
		}
	}

	db.tables[table] = ts
	return ts
}

func (db *DB) InKeys(numArgs int) string {
	return InKeys(numArgs)
}

func InKeys(numArgs int) string {
	a := make([]string, numArgs)
	for i := 0; i < numArgs; i++ {
		a[i] = "?"
	}
	return fmt.Sprintf("(%s)", strings.Join(a, ","))
}
