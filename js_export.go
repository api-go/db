package db

import (
	"context"
	"errors"
	"strings"

	"apigo.cc/go/cast"
	"apigo.cc/go/file"
	"apigo.cc/go/jsmod"
)

func init() {
	jsmod.Register("db", map[string]any{
		"get": func(ctx context.Context, name string) (*jsDB, error) {
			isReadOnly := false

			// 安全模式下的动态判定
			if jsmod.IsSafeMode(ctx) {
				if strings.HasPrefix(name, "sqlite://") {
					// 1. SQLite 路径校验
					path := strings.TrimPrefix(name, "sqlite://")
					if idx := strings.Index(path, "?"); idx != -1 {
						path = path[:idx]
					}
					// 如果路径不在沙箱内，VerifyPathForSafeMode 会直接返回 error，阻止连接
					_, err := file.VerifyPathForSafeMode(ctx, path)
					if err != nil {
						return nil, err
					}
					isReadOnly = false
				} else {
					// 2. 远程数据库前缀校验
					var allowedDSNs []string
					cast.Convert(&allowedDSNs, ctx.Value("AllowedDSNs"))
					matched := false
					for _, prefix := range allowedDSNs {
						if strings.HasPrefix(name, prefix) {
							matched = true
							break
						}
					}
					if !matched {
						// 未命中白名单，降级为只读模式
						isReadOnly = true
					}
				}
			}

			d := GetDB(name, nil)
			if d.Error != nil {
				return nil, d.Error
			}
			return &jsDB{db: d, ctx: ctx, isReadOnly: isReadOnly}, nil
		},
	})
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
	return jd.db.Query(query, args...)
}

// Write Operations
func (jd *jsDB) Exec(query string, args ...any) *ExecResult {
	if err := jd.checkWrite(); err != nil {
		return &ExecResult{Error: err}
	}
	return jd.db.Exec(query, args...)
}

func (jd *jsDB) Insert(table string, data any) *ExecResult {
	if err := jd.checkWrite(); err != nil {
		return &ExecResult{Error: err}
	}
	return jd.db.Insert(table, data)
}

func (jd *jsDB) Update(table string, data any, conditions string, args ...any) *ExecResult {
	if err := jd.checkWrite(); err != nil {
		return &ExecResult{Error: err}
	}
	return jd.db.Update(table, data, conditions, args...)
}

func (jd *jsDB) Delete(table string, conditions string, args ...any) *ExecResult {
	if err := jd.checkWrite(); err != nil {
		return &ExecResult{Error: err}
	}
	return jd.db.Delete(table, conditions, args...)
}

func (jd *jsDB) Replace(table string, data any) *ExecResult {
	if err := jd.checkWrite(); err != nil {
		return &ExecResult{Error: err}
	}
	return jd.db.Replace(table, data)
}

// Transaction Support
func (jd *jsDB) Begin() (*jsTx, error) {
	tx := jd.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &jsTx{tx: tx, ctx: jd.ctx, isReadOnly: jd.isReadOnly}, nil
}

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
	return jt.tx.Query(query, args...)
}

func (jt *jsTx) Exec(query string, args ...any) *ExecResult {
	if err := jt.checkWrite(); err != nil {
		return &ExecResult{Error: err}
	}
	return jt.tx.Exec(query, args...)
}

func (jt *jsTx) Insert(table string, data any) *ExecResult {
	if err := jt.checkWrite(); err != nil {
		return &ExecResult{Error: err}
	}
	return jt.tx.Insert(table, data)
}

func (jt *jsTx) Update(table string, data any, conditions string, args ...any) *ExecResult {
	if err := jt.checkWrite(); err != nil {
		return &ExecResult{Error: err}
	}
	return jt.tx.Update(table, data, conditions, args...)
}

func (jt *jsTx) Delete(table string, conditions string, args ...any) *ExecResult {
	if err := jt.checkWrite(); err != nil {
		return &ExecResult{Error: err}
	}
	return jt.tx.Delete(table, conditions, args...)
}

func (jt *jsTx) Replace(table string, data any) *ExecResult {
	if err := jt.checkWrite(); err != nil {
		return &ExecResult{Error: err}
	}
	return jt.tx.Replace(table, data)
}

func (jt *jsTx) Commit() error {
	if err := jt.checkWrite(); err != nil {
		return err
	}
	return jt.tx.Commit()
}

func (jt *jsTx) Rollback() error { return jt.tx.Rollback() }

// Metadata
func (jd *jsDB) InKeys(numArgs int) string { return jd.db.InKeys(numArgs) }
func (jd *jsDB) Quote(text string) string  { return jd.db.Quote(text) }
