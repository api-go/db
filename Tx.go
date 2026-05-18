package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Tx struct {
	conn                    *sql.Tx
	db                      *DB
	sqliteMu                *sync.Mutex
	hasLock                 bool
	lastSql                 *string
	lastArgs                []any
	Error                   error
	logger                  *dbLogger
	logSlow                 time.Duration
	isCommittedOrRollbacked bool
	QuoteTag                string
}

func (tx *Tx) Quote(text string) string {
	return quote(tx.QuoteTag, text)
}

func (tx *Tx) Quotes(texts []string) string {
	return quotes(tx.QuoteTag, texts)
}

func (tx *Tx) Commit() error {
	defer tx.unlock()
	if tx.isCommittedOrRollbacked {
		return nil
	}
	if tx.conn == nil {
		return errors.New("operate on a bad connection")
	}
	err := tx.conn.Commit()
	if err != nil {
		tx.logger.LogQueryError(err.Error(), *tx.lastSql, tx.lastArgs, -1)
	} else {
		tx.isCommittedOrRollbacked = true
	}
	return err
}

func (tx *Tx) Rollback() error {
	defer tx.unlock()
	if tx.isCommittedOrRollbacked {
		return nil
	}
	if tx.conn == nil {
		return errors.New("operate on a bad connection")
	}
	err := tx.conn.Rollback()
	if err != nil {
		tx.logger.LogQueryError(err.Error(), *tx.lastSql, tx.lastArgs, -1)
	} else {
		tx.isCommittedOrRollbacked = true
	}
	return err
}

func (tx *Tx) Finish(ok bool) error {
	if tx.isCommittedOrRollbacked {
		return nil
	}
	if ok {
		return tx.Commit()
	}
	return tx.Rollback()
}

func (tx *Tx) CheckFinished() error {
	if tx.isCommittedOrRollbacked {
		return nil
	}
	return tx.Rollback()
}

func (tx *Tx) Prepare(query string) *Stmt {
	tx.lock()
	tx.lastSql = &query
	r := basePrepare(nil, tx.conn, query)
	r.logger = tx.logger
	if r.Error != nil {
		tx.logger.LogQueryError(r.Error.Error(), *tx.lastSql, tx.lastArgs, -1)
	}
	return r
}

func (tx *Tx) Exec(query string, args ...any) *ExecResult {
	query, args = tx.db.rewriteFTS(query, args)
	args = flatArgs(args)
	tx.lock()
	tx.lastSql = &query
	tx.lastArgs = args
	r := baseExecRaw(nil, tx.conn, query, args...)
	r.logger = tx.logger
	if r.Error != nil {
		tx.logger.LogQueryError(r.Error.Error(), *tx.lastSql, tx.lastArgs, r.usedTime)
	} else {
		if tx.logSlow > 0 && r.usedTime >= float32(tx.logSlow/time.Millisecond) {
			tx.logger.LogQuery(*tx.lastSql, tx.lastArgs, r.usedTime)
		}
	}
	return r
}

func (tx *Tx) Query(query string, args ...any) *QueryResult {
	query, args = tx.db.rewriteFTS(query, args)
	args = flatArgs(args)
	// Query in Tx doesn't acquire lock unless it's already held by a previous write
	tx.lastSql = &query
	tx.lastArgs = args
	r := baseQueryRaw(nil, tx.conn, query, args...)
	r.logger = tx.logger
	if r.Error != nil {
		tx.logger.LogQueryError(r.Error.Error(), *tx.lastSql, tx.lastArgs, r.usedTime)
	} else {
		if tx.logSlow > 0 && r.usedTime >= float32(tx.logSlow/time.Millisecond) {
			tx.logger.LogQuery(*tx.lastSql, tx.lastArgs, r.usedTime)
		}
	}
	return r
}


func (tx *Tx) Insert(table string, data any) *ExecResult {
	query, values := tx.MakeInsertSql(table, data, false)
	return tx.Exec(query, values...)
}

func (tx *Tx) Replace(table string, data any) *ExecResult {
	query, values := tx.MakeInsertSql(table, data, true)
	return tx.Exec(query, values...)
}

func (tx *Tx) Update(table string, data any, conditions string, args ...any) *ExecResult {
	query, values := tx.MakeUpdateSql(table, data, conditions, args...)
	return tx.Exec(query, values...)
}

func (tx *Tx) Delete(table string, conditions string, args ...any) *ExecResult {
	ts := tx.db.getTable(table)
	where := ""
	if conditions != "" {
		where = " where " + conditions
	}

	if ts.HasShadowTable {
		// Move to shadow table
		colList := ""
		if len(ts.Columns) > 0 {
			quotedCols := make([]string, len(ts.Columns))
			for i, c := range ts.Columns {
				quotedCols[i] = tx.Quote(c)
			}
			colList = fmt.Sprintf(" (%s) select %s", strings.Join(quotedCols, ","), strings.Join(quotedCols, ","))
		} else {
			colList = " select *"
		}
		moveQuery := fmt.Sprintf("insert into %s%s from %s%s", tx.Quote(table+"_deleted"), colList, tx.Quote(table), where)
		// Use Exec to handle locking
		r := tx.Exec(moveQuery, args...)
		if r.Error != nil {
			return r
		}
	}

	query := fmt.Sprintf("delete from %s%s", tx.Quote(table), where)
	return tx.Exec(query, args...)
}

func (tx *Tx) lock() {
	if tx.sqliteMu == nil || tx.hasLock {
		return
	}
	tx.sqliteMu.Lock()
	tx.hasLock = true
}

func (tx *Tx) unlock() {
	if tx.sqliteMu == nil || !tx.hasLock {
		return
	}
	tx.sqliteMu.Unlock()
	tx.hasLock = false
}


