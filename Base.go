package db

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"apigo.cc/go/cast"
)

var structFieldsCache = sync.Map{}

type structFieldInfo struct {
	name  string
	index []int
}

func getStructFields(typ reflect.Type) []structFieldInfo {
	if v, ok := structFieldsCache.Load(typ); ok {
		return v.([]structFieldInfo)
	}
	var fields []structFieldInfo
	flattenFields(typ, nil, &fields)
	structFieldsCache.Store(typ, fields)
	return fields
}

func flattenFields(typ reflect.Type, index []int, fields *[]structFieldInfo) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		newIndex := make([]int, len(index)+len(f.Index))
		copy(newIndex, index)
		copy(newIndex[len(index):], f.Index)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			flattenFields(f.Type, newIndex, fields)
		} else {
			if f.Name[0] >= 'A' && f.Name[0] <= 'Z' {
				*fields = append(*fields, structFieldInfo{name: f.Name, index: newIndex})
			}
		}
	}
}

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
	return baseExecRaw(db, tx, query, flatArgs(args)...)
}

func baseExecRaw(db *sql.DB, tx *sql.Tx, query string, args ...any) *ExecResult {
	var r sql.Result
	var err error
	startTime := time.Now()
	if tx != nil {
		r, err = tx.Exec(query, args...)
	} else if db != nil {
		r, err = db.Exec(query, args...)
	} else {
		return &ExecResult{Sql: &query, Args: args, usedTime: makeUsedTime(startTime, time.Now()), Error: errors.New("operate on a bad connection")}
	}
	endTime := time.Now()
	usedTime := makeUsedTime(startTime, endTime)

	if err != nil {
		return &ExecResult{Sql: &query, Args: args, usedTime: usedTime, Error: err}
	}
	return &ExecResult{Sql: &query, Args: args, usedTime: usedTime, result: r}
}

func makeUsedTime(startTime, endTime time.Time) float32 {
	return float32(endTime.UnixNano()-startTime.UnixNano()) / 1e6
}

func flatArgs(args []any) []any {
	for i, arg := range args {
		if arg == nil {
			continue
		}
		argValue := reflect.ValueOf(arg)
		kind := argValue.Kind()
		if kind == reflect.Map || kind == reflect.Struct || (kind == reflect.Slice && argValue.Type().Elem().Kind() != reflect.Uint8) {
			args[i] = cast.As(cast.ToJSON(arg))
		}
	}
	return args
}

func baseQuery(db *sql.DB, tx *sql.Tx, query string, args ...any) *QueryResult {
	return baseQueryRaw(db, tx, query, flatArgs(args)...)
}

func baseQueryRaw(db *sql.DB, tx *sql.Tx, query string, args ...any) *QueryResult {
	var rows *sql.Rows
	var err error
	startTime := time.Now()
	if tx != nil {
		rows, err = tx.Query(query, args...)
	} else if db != nil {
		rows, err = db.Query(query, args...)
	} else {
		return &QueryResult{Sql: &query, Args: args, usedTime: makeUsedTime(startTime, time.Now()), Error: errors.New("operate on a bad connection")}
	}
	endTime := time.Now()
	usedTime := makeUsedTime(startTime, endTime)

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

func makeInsertSql(quoteTag string, table string, data any, useReplace bool, versionField string, nextVer int64, idField string, nextId string, ts *TableStruct) (string, []any) {
	keys, vars, values := MakeKeysVarsValues(data)
	keys, vars, values = canonicalColumnValues(keys, vars, values, ts)

	// 全文检索影子列自动分词处理
	if ts != nil {
		for _, col := range ts.Columns {
			if strings.HasSuffix(col, "_tokens") {
				originCol := strings.TrimSuffix(col, "_tokens")
				for i, k := range keys {
					if k == originCol {
						found := false
						for _, k2 := range keys {
							if k2 == col {
								found = true
								break
							}
						}
						if !found {
							keys = append(keys, col)
							vars = append(vars, "?")
							values = append(values, BigramTokenize(cast.String(values[i])))
						}
						break
					}
				}
			}
		}
	}

	if versionField != "" {
		found := false
		for _, k := range keys {
			if k == versionField {
				found = true
				break
			}
		}
		if !found {
			keys = append(keys, versionField)
			vars = append(vars, "?")
			values = append(values, nextVer)
		}
	}
	if idField != "" && nextId != "" {
		found := false
		for i, k := range keys {
			if k == idField {
				found = true
				if cast.String(values[i]) == "" {
					values[i] = nextId
				}
				break
			}
		}
		if !found {
			keys = append(keys, idField)
			vars = append(vars, "?")
			values = append(values, nextId)
		}
	}
	operation := "insert"
	if useReplace {
		operation = "replace"
	}
	query := fmt.Sprintf("%s into %s (%s) values (%s)", operation, quote(quoteTag, table), quotes(quoteTag, keys), strings.Join(vars, ","))
	return query, values
}

func makeUpdateSql(quoteTag string, table string, data any, conditions string, versionField string, nextVer int64, ts *TableStruct, args ...any) (string, []any) {
	args = flatArgs(args)
	keys, vars, values := MakeKeysVarsValues(data)
	keys, vars, values = canonicalColumnValues(keys, vars, values, ts)

	// 全文检索影子列自动分词处理
	if ts != nil {
		for _, col := range ts.Columns {
			if strings.HasSuffix(col, "_tokens") {
				originCol := strings.TrimSuffix(col, "_tokens")
				for i, k := range keys {
					if k == originCol {
						found := false
						for _, k2 := range keys {
							if k2 == col {
								found = true
								break
							}
						}
						if !found {
							keys = append(keys, col)
							vars = append(vars, "?")
							values = append(values, BigramTokenize(cast.String(values[i])))
						}
						break
					}
				}
			}
		}
	}

	newKeys := make([]string, 0, len(keys))
	newValues := make([]any, 0, len(values))
	var oldVersion any
	for i, k := range keys {
		if k == versionField {
			oldVersion = values[i]
			continue
		}
		newKeys = append(newKeys, fmt.Sprintf("%s=%s", quote(quoteTag, k), vars[i]))
		newValues = append(newValues, values[i])
	}
	if versionField != "" {
		newKeys = append(newKeys, fmt.Sprintf("%s=?", quote(quoteTag, versionField)))
		newValues = append(newValues, nextVer)
	}

	if oldVersion != nil {
		if conditions != "" {
			conditions = fmt.Sprintf("(%s) and %s=?", conditions, quote(quoteTag, versionField))
		} else {
			conditions = fmt.Sprintf("%s=?", quote(quoteTag, versionField))
		}
		args = append(args, oldVersion)
	}

	newValues = append(newValues, args...)
	if conditions != "" {
		conditions = " where " + conditions
	}
	query := fmt.Sprintf("update %s set %s%s", quote(quoteTag, table), strings.Join(newKeys, ","), conditions)
	return query, newValues
}

func canonicalColumnValues(keys, vars []string, values []any, table *TableStruct) ([]string, []string, []any) {
	if table == nil || len(table.Columns) == 0 {
		return keys, vars, values
	}
	canonical := make(map[string]string, len(table.Columns))
	for _, column := range table.Columns {
		canonical[strings.ToLower(column)] = column
	}
	outKeys := make([]string, 0, len(keys))
	outVars := make([]string, 0, len(vars))
	outValuesByKey := make([]any, 0, len(keys))
	outHasValue := make([]bool, 0, len(keys))
	positions := make(map[string]int, len(keys))
	aliases := make(map[string]bool, len(keys))
	valueIndex := 0
	for i, key := range keys {
		hasValue := vars[i] == "?"
		var value any
		if hasValue {
			value = values[valueIndex]
			valueIndex++
		}
		column := canonical[strings.ToLower(key)]
		if column == "" {
			column = key
		}
		folded := strings.ToLower(column)
		isAlias := key != column
		if position, exists := positions[folded]; exists {
			if isAlias || !aliases[folded] {
				outVars[position], outValuesByKey[position], outHasValue[position], aliases[folded] = vars[i], value, hasValue, isAlias
			}
			continue
		}
		positions[folded] = len(outKeys)
		aliases[folded] = isAlias
		outKeys = append(outKeys, column)
		outVars = append(outVars, vars[i])
		outValuesByKey = append(outValuesByKey, value)
		outHasValue = append(outHasValue, hasValue)
	}
	outValues := make([]any, 0, len(values))
	for i, value := range outValuesByKey {
		if outHasValue[i] {
			outValues = append(outValues, value)
		}
	}
	return outKeys, outVars, outValues
}

func (db *DB) MakeInsertSql(table string, data any, useReplace bool) (string, []any) {
	ts := db.getTable(table)
	nextVer := int64(0)
	if ts.VersionField != "" {
		nextVer = db.NextVersion(table)
	}
	nextId := ""
	if ts.IdField != "" {
		nextId = db.NextID(table)
	}
	return makeInsertSql(db.QuoteTag, table, data, useReplace, ts.VersionField, nextVer, ts.IdField, nextId, ts)
}

func (db *DB) MakeUpdateSql(table string, data any, conditions string, args ...any) (string, []any) {
	ts := db.getTable(table)
	nextVer := int64(0)
	if ts.VersionField != "" {
		nextVer = db.NextVersion(table)
	}
	return makeUpdateSql(db.QuoteTag, table, data, conditions, ts.VersionField, nextVer, ts, args...)
}

func (tx *Tx) MakeInsertSql(table string, data any, useReplace bool) (string, []any) {
	ts := tx.db.getTable(table)
	nextVer := int64(0)
	if ts.VersionField != "" {
		nextVer = tx.db.NextVersion(table)
	}
	nextId := ""
	if ts.IdField != "" {
		nextId = tx.db.NextID(table)
	}
	return makeInsertSql(tx.QuoteTag, table, data, useReplace, ts.VersionField, nextVer, ts.IdField, nextId, ts)
}

func (tx *Tx) MakeUpdateSql(table string, data any, conditions string, args ...any) (string, []any) {
	ts := tx.db.getTable(table)
	nextVer := int64(0)
	if ts.VersionField != "" {
		nextVer = tx.db.NextVersion(table)
	}
	return makeUpdateSql(tx.QuoteTag, table, data, conditions, ts.VersionField, nextVer, ts, args...)
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
		fields := getStructFields(dataType)
		for _, f := range fields {
			v := dataValue.FieldByIndex(f.index)
			if v.Kind() == reflect.Interface {
				v = v.Elem()
			}
			keys = append(keys, f.name)
			if isSQLExpression(v) {
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
			if isSQLExpression(v) {
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

func isSQLExpression(value reflect.Value) bool {
	return value.IsValid() && value.Kind() == reflect.String && value.Len() > 1 &&
		value.String()[0] == ':' && value.String()[1] != ':'
}
