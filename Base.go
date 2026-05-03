package db

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"apigo.cc/go/cast"
	"apigo.cc/go/log"
)

func basePrepare(db *sql.DB, tx *sql.Tx, query string) *Stmt {
	var sqlStmt *sql.Stmt
	var err error
	if tx != nil {
		sqlStmt, err = tx.Prepare(query)
	} else if db != nil {
		sqlStmt, err = db.Prepare(query)
	} else {
		return &Stmt{Error: errors.New("operate on a bad connection")}
	}
	if err != nil {
		return &Stmt{Error: err}
	}
	return &Stmt{conn: sqlStmt, lastSql: &query}
}

func baseExec(db *sql.DB, tx *sql.Tx, query string, args ...any) *ExecResult {
	args = flatArgs(args)
	var r sql.Result
	var err error
	startTime := time.Now()
	if tx != nil {
		r, err = tx.Exec(query, args...)
	} else if db != nil {
		r, err = db.Exec(query, args...)
	} else {
		return &ExecResult{Sql: &query, Args: args, usedTime: log.MakeUsedTime(startTime, time.Now()), Error: errors.New("operate on a bad connection")}
	}
	endTime := time.Now()
	usedTime := log.MakeUsedTime(startTime, endTime)

	if err != nil {
		return &ExecResult{Sql: &query, Args: args, usedTime: usedTime, Error: err}
	}
	return &ExecResult{Sql: &query, Args: args, usedTime: usedTime, result: r}
}

func flatArgs(args []any) []any {
	for i, arg := range args {
		if arg == nil {
			continue
		}
		argValue := reflect.ValueOf(arg)
		kind := argValue.Kind()
		if kind == reflect.Map || kind == reflect.Struct || (kind == reflect.Slice && argValue.Type().Elem().Kind() != reflect.Uint8) {
			args[i] = cast.MustToJSON(arg)
		}
	}
	return args
}

func baseQuery(db *sql.DB, tx *sql.Tx, query string, args ...any) *QueryResult {
	args = flatArgs(args)

	var rows *sql.Rows
	var err error
	startTime := time.Now()
	if tx != nil {
		rows, err = tx.Query(query, args...)
	} else if db != nil {
		rows, err = db.Query(query, args...)
	} else {
		return &QueryResult{Sql: &query, Args: args, usedTime: log.MakeUsedTime(startTime, time.Now()), Error: errors.New("operate on a bad connection")}
	}
	endTime := time.Now()
	usedTime := log.MakeUsedTime(startTime, endTime)

	if err != nil {
		return &QueryResult{Sql: &query, Args: args, usedTime: usedTime, Error: err}
	}
	return &QueryResult{Sql: &query, Args: args, usedTime: usedTime, rows: rows}
}

func quote(quoteTag string, text string) string {
	a := strings.Split(text, ".")
	for i, v := range a {
		a[i] = quoteTag + strings.ReplaceAll(v, quoteTag, "\\"+quoteTag) + quoteTag
	}
	return strings.Join(a, ".")
}

func quotes(quoteTag string, texts []string) string {
	for i, v := range texts {
		texts[i] = quote(quoteTag, v)
	}
	return strings.Join(texts, ",")
}

func makeInsertSql(quoteTag string, table string, data any, useReplace bool) (string, []any) {
	keys, vars, values := MakeKeysVarsValues(data)
	operation := "insert"
	if useReplace {
		operation = "replace"
	}
	query := fmt.Sprintf("%s into %s (%s) values (%s)", operation, quote(quoteTag, table), quotes(quoteTag, keys), strings.Join(vars, ","))
	return query, values
}

func makeUpdateSql(quoteTag string, table string, data any, conditions string, args ...any) (string, []any) {
	args = flatArgs(args)
	keys, vars, values := MakeKeysVarsValues(data)
	for i, k := range keys {
		keys[i] = fmt.Sprintf("%s=%s", quote(quoteTag, k), vars[i])
	}
	values = append(values, args...)
	if conditions != "" {
		conditions = " where " + conditions
	}
	query := fmt.Sprintf("update %s set %s%s", quote(quoteTag, table), strings.Join(keys, ","), conditions)
	return query, values
}

func (db *DB) MakeInsertSql(table string, data any, useReplace bool) (string, []any) {
	return makeInsertSql(db.QuoteTag, table, data, useReplace)
}

func (db *DB) MakeUpdateSql(table string, data any, conditions string, args ...any) (string, []any) {
	return makeUpdateSql(db.QuoteTag, table, data, conditions, args...)
}

func (tx *Tx) MakeInsertSql(table string, data any, useReplace bool) (string, []any) {
	return makeInsertSql(tx.QuoteTag, table, data, useReplace)
}

func (tx *Tx) MakeUpdateSql(table string, data any, conditions string, args ...any) (string, []any) {
	return makeUpdateSql(tx.QuoteTag, table, data, conditions, args...)
}

func getFlatFields(fields map[string]reflect.Value, fieldKeys *[]string, value reflect.Value) {
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		v := value.Field(i)
		if valueType.Field(i).Anonymous {
			getFlatFields(fields, fieldKeys, v)
		} else {
			*fieldKeys = append(*fieldKeys, valueType.Field(i).Name)
			fields[valueType.Field(i).Name] = v
		}
	}
}

func MakeKeysVarsValues(data any) ([]string, []string, []any) {
	keys := make([]string, 0)
	vars := make([]string, 0)
	values := make([]any, 0)

	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)
	for dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
		dataValue = dataValue.Elem()
	}

	if dataType.Kind() == reflect.Struct {
		fields := make(map[string]reflect.Value)
		fieldKeys := make([]string, 0)
		getFlatFields(fields, &fieldKeys, dataValue)
		for _, k := range fieldKeys {
			if k[0] >= 'a' && k[0] <= 'z' {
				continue
			}
			v := fields[k]
			if v.Kind() == reflect.Interface {
				v = v.Elem()
			}
			keys = append(keys, k)
			if v.Kind() == reflect.String && v.Len() > 0 && v.String()[0] == ':' {
				vars = append(vars, v.String()[1:])
			} else {
				vars = append(vars, "?")
				if !v.IsValid() || !v.CanInterface() {
					values = append(values, nil)
				} else {
					values = append(values, v.Interface())
				}
			}
		}
	} else if dataType.Kind() == reflect.Map {
		for _, k := range dataValue.MapKeys() {
			v := dataValue.MapIndex(k)
			if v.Kind() == reflect.Interface {
				v = v.Elem()
			}
			keys = append(keys, cast.String(k.Interface()))
			if v.Kind() == reflect.String && v.Len() > 0 && v.String()[0] == ':' {
				vars = append(vars, v.String()[1:])
			} else {
				vars = append(vars, "?")
				if !v.IsValid() || !v.CanInterface() {
					values = append(values, nil)
				} else {
					values = append(values, v.Interface())
				}
			}
		}
	}

	return keys, vars, values
}
