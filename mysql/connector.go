package mysql

import (
	"context"
	"crypto/tls"
	"database/sql/driver"
	"errors"
	"net/url"
	"os"
	"regexp"

	"apigo.cc/go/db"
	"apigo.cc/go/safe"
	"github.com/go-sql-driver/mysql"
)

type SecureMysqlConnector struct {
	conf *db.Config
	pwd  *safe.SafeBuf
	tls  *tls.Config
}

var databaseNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*$`)

func (c *SecureMysqlConnector) Connect(ctx context.Context) (driver.Conn, error) {
	cfg := mysql.NewConfig()
	params, _ := url.ParseQuery(c.conf.Args)
	cfg.User = c.conf.User
	cfg.Net, cfg.Addr = params.Get("net"), params.Get("addr")
	params.Del("net")
	params.Del("addr")
	if cfg.Net == "" {
		cfg.Net = "tcp"
	}
	if cfg.Addr == "" {
		cfg.Addr = c.conf.Host
	}
	for key, values := range params {
		if len(values) > 0 {
			cfg.Params[key] = values[len(values)-1]
		}
	}
	cfg.DBName = c.conf.DB
	cfg.TLS = c.tls
	pwdBuf := c.pwd.Open()
	defer pwdBuf.Close()
	cfg.Passwd = pwdBuf.String()

	conn, err := connectWithAutoCreate(ctx, cfg)
	var mysqlErr *mysql.MySQLError
	localUser, allowLocal := c.conf.LocalAuthFallback()
	if !allowLocal || !errors.As(err, &mysqlErr) || mysqlErr.Number != 1044 && mysqlErr.Number != 1045 && mysqlErr.Number != 1698 {
		return conn, err
	}
	if cfg.DBName != "" && !databaseNamePattern.MatchString(cfg.DBName) {
		return nil, err
	}
	for _, socket := range []string{"/tmp/mysql.sock", "/run/mysqld/mysqld.sock", "/var/run/mysqld/mysqld.sock"} {
		if _, statErr := os.Stat(socket); statErr != nil {
			continue
		}
		for _, bootstrapUser := range []string{localUser, "root"} {
			if bootstrapLocalTCP(ctx, cfg, socket, bootstrapUser) == nil {
				return connectWithAutoCreate(ctx, cfg)
			}
		}
	}
	return nil, err
}

func bootstrapLocalTCP(ctx context.Context, target *mysql.Config, socket, bootstrapUser string) error {
	local := target.Clone()
	local.User = bootstrapUser
	local.Passwd = ""
	local.Net = "unix"
	local.Addr = socket
	local.DBName = ""
	connector, err := mysql.NewConnector(local)
	if err != nil {
		return err
	}
	conn, err := connector.Connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	execer, ok := conn.(driver.ExecerContext)
	if !ok {
		return driver.ErrSkip
	}
	statements := []string{
		"CREATE USER IF NOT EXISTS 'apigo'@'localhost' IDENTIFIED BY ''",
		"CREATE USER IF NOT EXISTS 'apigo'@'127.0.0.1' IDENTIFIED BY ''",
	}
	if target.DBName != "" {
		statements = append(statements,
			"CREATE DATABASE IF NOT EXISTS `"+target.DBName+"`",
			"GRANT ALL PRIVILEGES ON `"+target.DBName+"`.* TO 'apigo'@'localhost'",
			"GRANT ALL PRIVILEGES ON `"+target.DBName+"`.* TO 'apigo'@'127.0.0.1'",
		)
	}
	for _, statement := range statements {
		if _, err = execer.ExecContext(ctx, statement, nil); err != nil {
			return err
		}
	}
	return nil
}

func connectWithAutoCreate(ctx context.Context, cfg *mysql.Config) (driver.Conn, error) {
	tmpConnector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	conn, err := tmpConnector.Connect(ctx)
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1049 || !databaseNamePattern.MatchString(cfg.DBName) {
		return conn, err
	}

	maintenance := cfg.Clone()
	maintenance.DBName = ""
	maintenanceConnector, createErr := mysql.NewConnector(maintenance)
	if createErr != nil {
		return nil, err
	}
	maintenanceConn, createErr := maintenanceConnector.Connect(ctx)
	if createErr != nil {
		return nil, err
	}
	execer, ok := maintenanceConn.(driver.ExecerContext)
	if !ok {
		_ = maintenanceConn.Close()
		return nil, err
	}
	_, createErr = execer.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+cfg.DBName+"`", nil)
	_ = maintenanceConn.Close()
	if createErr != nil {
		return nil, err
	}
	return tmpConnector.Connect(ctx)
}

func (c *SecureMysqlConnector) Driver() driver.Driver {
	return &mysql.MySQLDriver{}
}

func init() {
	db.RegisterConnector("mysql", func(conf *db.Config, pwd *safe.SafeBuf, tls *tls.Config) driver.Connector {
		return &SecureMysqlConnector{conf: conf, pwd: pwd, tls: tls}
	})
}
