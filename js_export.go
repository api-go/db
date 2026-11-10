package db

import (
	"context"
	"errors"
	"strings"

	"github.com/api-go/cast"
	"github.com/api-go/file"
	"github.com/api-go/jsmod"
)

func init() {
	jsmod.Register("db", map[string]any{
		// 入口：支持别名获取，不传则默认 "default"
		"Get":     jsGet,
		"Query":   jsQuery,
		"Exec":    jsExec,
		"Insert":  jsInsert,
		"Update":  jsUpdate,
		"Delete":  jsDelete,
		"Replace": jsReplace,
		"Begin":   jsBegin,
	})
}

func jsGet(ctx context.Context, name *string) (*jsDB, error) {
	target := "default"
	if name != nil {
		target = *name
	}
	db, err := getJSDB(ctx, target)
	if err != nil {
		return nil, jsmod.MakeError(err)
	}
	return db, nil
}

func jsQuery(ctx context.Context, query string, args ...any) (*QueryResult, error) {
	jd, err := getJSDB(ctx, "default")
	if err != nil {
		return nil, jsmod.MakeError(err)
	}
	res := jd.Query(query, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res, nil
}

func jsExec(ctx context.Context, query string, args ...any) (*ExecResult, error) {
	jd, err := getJSDB(ctx, "default")
	if err != nil {
		return nil, jsmod.MakeError(err)
	}
	res := jd.Exec(query, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res, nil
}

func jsInsert(ctx context.Context, table string, data any) (*ExecResult, error) {
	jd, err := getJSDB(ctx, "default")
	if err != nil {
		return nil, jsmod.MakeError(err)
	}
	res := jd.Insert(table, data)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res, nil
}

func jsUpdate(ctx context.Context, table string, data any, conditions string, args ...any) (*ExecResult, error) {
	jd, err := getJSDB(ctx, "default")
	if err != nil {
		return nil, jsmod.MakeError(err)
	}
	res := jd.Update(table, data, conditions, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res, nil
}

func jsDelete(ctx context.Context, table string, conditions string, args ...any) (*ExecResult, error) {
	jd, err := getJSDB(ctx, "default")
	if err != nil {
		return nil, jsmod.MakeError(err)
	}
	res := jd.Delete(table, conditions, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res, nil
}

func jsReplace(ctx context.Context, table string, data any) (*ExecResult, error) {
	jd, err := getJSDB(ctx, "default")
	if err != nil {
		return nil, jsmod.MakeError(err)
	}
	res := jd.Replace(table, data)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res, nil
}

func jsBegin(ctx context.Context) (*jsTx, error) {
	jd, err := getJSDB(ctx, "default")
	if err != nil {
		return nil, jsmod.MakeError(err)
	}
	return jd.Begin()
}

func getJSDB(ctx context.Context, name string) (*jsDB, error) {
	isReadOnly := false

	// 安全模式下的动态判定
	if jsmod.IsSafeMode(ctx) {
		if strings.HasPrefix(name, "sqlite://") {
			// 1. SQLite 路径校验
			path := strings.TrimPrefix(name, "sqlite://")
			if idx := strings.Index(path, "?"); idx != -1 {
				path = path[:idx]
			}
			_, err := file.VerifyPathForSafeMode(ctx, path)
			if err != nil {
				return nil, jsmod.MakeError(err)
			}
			isReadOnly = false
		} else {
			// 2. 远程数据库前缀校验
			var allowedDSNs []string
			cast.Convert(&allowedDSNs, jsmod.Get(ctx, "AllowedDSNs"))
			matched := false
			for _, prefix := range allowedDSNs {
				if strings.HasPrefix(name, prefix) {
					matched = true
					break
				}
			}
			if !matched {
				isReadOnly = true
			}
		}
	}

	d := GetDB(name, nil)
	if d.Error != nil {
		return nil, jsmod.MakeError(d.Error)
	}
	return &jsDB{db: d, ctx: ctx, isReadOnly: isReadOnly}, nil
}

type jsDB struct {
	db         *DB
	ctx        context.Context
	isReadOnly bool
}

var errReadOnly = errors.New("database is in read-only mode for this context")

func (jd *jsDB) checkWrite() error {
	if jd.isReadOnly {
		return errReadOnly
	}
	return nil
}

// Read Operations
func (jd *jsDB) Query(query string, args ...any) *QueryResult {
	res := jd.db.Query(query, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

// Write Operations
func (jd *jsDB) Exec(query string, args ...any) *ExecResult {
	if err := jd.checkWrite(); err != nil {
		return &ExecResult{Error: jsmod.MakeError(err)}
	}
	res := jd.db.Exec(query, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

func (jd *jsDB) Insert(table string, data any) *ExecResult {
	if err := jd.checkWrite(); err != nil {
		return &ExecResult{Error: jsmod.MakeError(err)}
	}
	res := jd.db.Insert(table, data)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

func (jd *jsDB) Update(table string, data any, conditions string, args ...any) *ExecResult {
	if err := jd.checkWrite(); err != nil {
		return &ExecResult{Error: jsmod.MakeError(err)}
	}
	res := jd.db.Update(table, data, conditions, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

func (jd *jsDB) Delete(table string, conditions string, args ...any) *ExecResult {
	if err := jd.checkWrite(); err != nil {
		return &ExecResult{Error: jsmod.MakeError(err)}
	}
	res := jd.db.Delete(table, conditions, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

func (jd *jsDB) Replace(table string, data any) *ExecResult {
	if err := jd.checkWrite(); err != nil {
		return &ExecResult{Error: jsmod.MakeError(err)}
	}
	res := jd.db.Replace(table, data)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

// Transaction Support
func (jd *jsDB) Begin() (*jsTx, error) {
	tx := jd.db.Begin()
	if tx.Error != nil {
		return nil, jsmod.MakeError(tx.Error)
	}
	return &jsTx{tx: tx, ctx: jd.ctx, isReadOnly: jd.isReadOnly}, nil
}

// Metadata
func (jd *jsDB) InKeys(numArgs int) string { return jd.db.InKeys(numArgs) }
func (jd *jsDB) Quote(text string) string  { return jd.db.Quote(text) }

// jsTx wraps *Tx for JS environment
type jsTx struct {
	tx         *Tx
	ctx        context.Context
	isReadOnly bool
}

func (jt *jsTx) checkWrite() error {
	if jt.isReadOnly {
		return errReadOnly
	}
	return nil
}

func (jt *jsTx) Query(query string, args ...any) *QueryResult {
	res := jt.tx.Query(query, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

func (jt *jsTx) Exec(query string, args ...any) *ExecResult {
	if err := jt.checkWrite(); err != nil {
		return &ExecResult{Error: jsmod.MakeError(err)}
	}
	res := jt.tx.Exec(query, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

func (jt *jsTx) Insert(table string, data any) *ExecResult {
	if err := jt.checkWrite(); err != nil {
		return &ExecResult{Error: jsmod.MakeError(err)}
	}
	res := jt.tx.Insert(table, data)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

func (jt *jsTx) Update(table string, data any, conditions string, args ...any) *ExecResult {
	if err := jt.checkWrite(); err != nil {
		return &ExecResult{Error: jsmod.MakeError(err)}
	}
	res := jt.tx.Update(table, data, conditions, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

func (jt *jsTx) Delete(table string, conditions string, args ...any) *ExecResult {
	if err := jt.checkWrite(); err != nil {
		return &ExecResult{Error: jsmod.MakeError(err)}
	}
	res := jt.tx.Delete(table, conditions, args...)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

func (jt *jsTx) Replace(table string, data any) *ExecResult {
	if err := jt.checkWrite(); err != nil {
		return &ExecResult{Error: jsmod.MakeError(err)}
	}
	res := jt.tx.Replace(table, data)
	if res != nil && res.Error != nil {
		res.Error = jsmod.MakeError(res.Error)
	}
	return res
}

func (jt *jsTx) Commit() error {
	if err := jt.checkWrite(); err != nil {
		return jsmod.MakeError(err)
	}
	if err := jt.tx.Commit(); err != nil {
		return jsmod.MakeError(err)
	}
	return nil
}

func (jt *jsTx) Rollback() error {
	if err := jt.tx.Rollback(); err != nil {
		return jsmod.MakeError(err)
	}
	return nil
}
