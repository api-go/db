package db

import (
	"context"
	"errors"

	"apigo.cc/go/jsmod"
)

func init() {
	jsmod.Register("db", map[string]any{
		"get": func(ctx context.Context, name string) (*jsDB, error) {
			d := GetDB(name, nil)
			if d.Error != nil {
				return nil, d.Error
			}
			return &jsDB{db: d, ctx: ctx}, nil
		},
	})
}

type jsDB struct {
	db  *DB
	ctx context.Context
}

var errSafeMode = errors.New("database write operation is restricted in safe mode")

func (jd *jsDB) checkSafe() error {
	if jsmod.IsSafeMode(jd.ctx) {
		return errSafeMode
	}
	return nil
}

// Read Operations (Always allowed)
func (jd *jsDB) Query(query string, args ...any) *QueryResult {
	return jd.db.Query(query, args...)
}

// Write Operations (Restricted in SafeMode)
func (jd *jsDB) Exec(query string, args ...any) *ExecResult {
	if jd.checkSafe() != nil {
		return &ExecResult{Error: errSafeMode}
	}
	return jd.db.Exec(query, args...)
}

func (jd *jsDB) Insert(table string, data any) *ExecResult {
	if jd.checkSafe() != nil {
		return &ExecResult{Error: errSafeMode}
	}
	return jd.db.Insert(table, data)
}

func (jd *jsDB) Update(table string, data any, conditions string, args ...any) *ExecResult {
	if jd.checkSafe() != nil {
		return &ExecResult{Error: errSafeMode}
	}
	return jd.db.Update(table, data, conditions, args...)
}

func (jd *jsDB) Delete(table string, conditions string, args ...any) *ExecResult {
	if jd.checkSafe() != nil {
		return &ExecResult{Error: errSafeMode}
	}
	return jd.db.Delete(table, conditions, args...)
}

func (jd *jsDB) Replace(table string, data any) *ExecResult {
	if jd.checkSafe() != nil {
		return &ExecResult{Error: errSafeMode}
	}
	return jd.db.Replace(table, data)
}

// Metadata
func (jd *jsDB) InKeys(numArgs int) string {
	return jd.db.InKeys(numArgs)
}

func (jd *jsDB) Quote(text string) string {
	return jd.db.Quote(text)
}
