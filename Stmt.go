package db

import (
	"database/sql"
	"errors"
	"time"

	"apigo.cc/go/log"
)

type Stmt struct {
	conn     *sql.Stmt
	lastSql  *string
	lastArgs []any
	Error    error
	logger   *dbLogger
}

func (stmt *Stmt) Exec(args ...any) *ExecResult {
	stmt.lastArgs = args
	if stmt.conn == nil {
		return &ExecResult{Sql: stmt.lastSql, Args: stmt.lastArgs, usedTime: -1, logger: stmt.logger, Error: errors.New("operate on a bad connection")}
	}
	startTime := time.Now()
	r, err := stmt.conn.Exec(args...)
	endTime := time.Now()
	usedTime := log.MakeUsedTime(startTime, endTime)
	if err != nil {
		stmt.logger.LogQueryError(err.Error(), *stmt.lastSql, stmt.lastArgs, usedTime)
		return &ExecResult{Sql: stmt.lastSql, Args: stmt.lastArgs, usedTime: usedTime, logger: stmt.logger, Error: err}
	}
	return &ExecResult{Sql: stmt.lastSql, Args: stmt.lastArgs, usedTime: usedTime, logger: stmt.logger, result: r}
}

func (stmt *Stmt) Close() error {
	if stmt.conn == nil {
		return errors.New("operate on a bad connection")
	}
	err := stmt.conn.Close()
	if err != nil {
		stmt.logger.LogQueryError(err.Error(), *stmt.lastSql, stmt.lastArgs, -1)
	}
	return err
}
