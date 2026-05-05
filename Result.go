package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"apigo.cc/go/cast"
	"github.com/mitchellh/mapstructure"
)

type QueryResult struct {
	rows      *sql.Rows
	Sql       *string
	Args      []any
	Error     error
	logger    *dbLogger
	usedTime  float32
	completed bool
}

type ExecResult struct {
	result   sql.Result
	Sql      *string
	Args     []any
	Error    error
	logger   *dbLogger
	usedTime float32
}

func (r *ExecResult) Changes() int64 {
	if r.result == nil {
		return 0
	}
	numChanges, err := r.result.RowsAffected()
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
		return 0
	}
	return numChanges
}

func (r *ExecResult) Id() int64 {
	if r.result == nil {
		return 0
	}
	insertId, err := r.result.LastInsertId()
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
		return 0
	}
	return insertId
}

func (r *QueryResult) Complete() {
	if !r.completed {
		if r.rows != nil {
			r.rows.Close()
		}
		r.completed = true
	}
}

func (r *QueryResult) To(result any) error {
	if r.rows == nil {
		return errors.New("operate on a bad query")
	}
	return r.makeResults(result, r.rows)
}

func ToSlice[T any](r *QueryResult) ([]T, error) {
	var result []T
	err := r.To(&result)
	return result, err
}

func To[T any](r *QueryResult) (T, error) {
	var result T
	err := r.To(&result)
	return result, err
}

func (r *QueryResult) MapResults() []map[string]any {
	result := make([]map[string]any, 0)
	err := r.makeResults(&result, r.rows)
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
	}
	return result
}

func (r *QueryResult) SliceResults() [][]any {
	result := make([][]any, 0)
	err := r.makeResults(&result, r.rows)
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
	}
	return result
}

func (r *QueryResult) StringMapResults() []map[string]string {
	result := make([]map[string]string, 0)
	err := r.makeResults(&result, r.rows)
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
	}
	return result
}

func (r *QueryResult) StringSliceResults() [][]string {
	result := make([][]string, 0)
	err := r.makeResults(&result, r.rows)
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
	}
	return result
}

func (r *QueryResult) MapOnR1() map[string]any {
	result := make(map[string]any)
	err := r.makeResults(&result, r.rows)
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
	}
	return result
}

func (r *QueryResult) StringMapOnR1() map[string]string {
	result := make(map[string]string)
	err := r.makeResults(&result, r.rows)
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
	}
	return result
}

func (r *QueryResult) IntsOnC1() []int64 {
	result := make([]int64, 0)
	err := r.makeResults(&result, r.rows)
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
	}
	return result
}

func (r *QueryResult) StringsOnC1() []string {
	result := make([]string, 0)
	err := r.makeResults(&result, r.rows)
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
	}
	return result
}

func (r *QueryResult) IntOnR1C1() int64 {
	var result int64 = 0
	err := r.makeResults(&result, r.rows)
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
	}
	return result
}

func (r *QueryResult) FloatOnR1C1() float64 {
	var result float64 = 0
	err := r.makeResults(&result, r.rows)
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
	}
	return result
}

func (r *QueryResult) StringOnR1C1() string {
	result := ""
	err := r.makeResults(&result, r.rows)
	if err != nil {
		r.Error = err
		r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
	}
	return result
}

func (r *QueryResult) ToKV(target any) error {
	v := reflect.ValueOf(target)
	t := v.Type()
	for t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = v.Type()
	}

	if t.Kind() != reflect.Map {
		r.logger.LogQueryError("target not a map", *r.Sql, r.Args, r.usedTime)
		return errors.New("target not a map")
	}

	vt := t.Elem()
	finalVt := vt
	for finalVt.Kind() == reflect.Ptr {
		finalVt = finalVt.Elem()
	}
	if finalVt.Kind() == reflect.Map || finalVt.Kind() == reflect.Struct {
		colTypes, err := r.getColumnTypes()
		list := r.MapResults()
		if err != nil {
			r.logger.LogQueryError(err.Error(), *r.Sql, r.Args, r.usedTime)
			return err
		}
		for _, item := range list {
			newKey := reflect.ValueOf(reflect.New(t.Key()).Interface()).Elem()
			cast.Convert(newKey.Addr().Interface(), item[colTypes[0].Name()])

			newValue := v.MapIndex(newKey)
			isNew := false
			if !newValue.IsValid() {
				newValue = reflect.New(vt)
				isNew = true
			}

			err := mapstructure.WeakDecode(item, newValue.Interface())
			if err != nil {
				r.logger.LogError(err.Error())
			}

			if isNew {
				v.SetMapIndex(newKey, newValue.Elem())
			}
		}
	} else {
		list := r.SliceResults()
		for _, item := range list {
			if len(item) < 2 {
				continue
			}
			switch vt.Kind() {
			case reflect.Int:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(cast.Int(item[1])))
			case reflect.Int8:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(int8(cast.Int(item[1]))))
			case reflect.Int16:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(int16(cast.Int(item[1]))))
			case reflect.Int32:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(int32(cast.Int(item[1]))))
			case reflect.Int64:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(cast.Int64(item[1])))
			case reflect.Uint:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(uint(cast.Int(item[1]))))
			case reflect.Uint8:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(uint8(cast.Int(item[1]))))
			case reflect.Uint16:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(uint16(cast.Int(item[1]))))
			case reflect.Uint32:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(uint32(cast.Int(item[1]))))
			case reflect.Uint64:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(cast.Uint64(item[1])))
			case reflect.Float32:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(cast.Float(item[1])))
			case reflect.Float64:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(cast.Float64(item[1])))
			case reflect.Bool:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(cast.Bool(item[1])))
			case reflect.String:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(cast.String(item[1])))
			case reflect.Interface:
				v.SetMapIndex(reflect.ValueOf(cast.String(item[0])), reflect.ValueOf(item[1]))
			}
		}
	}

	return nil
}

func (r *QueryResult) makeResults(results any, rows *sql.Rows) error {
	if rows == nil {
		return errors.New("not a valid query result")
	}

	defer func() {
		_ = rows.Close()
		r.completed = true
	}()
	resultsValue := reflect.ValueOf(results)
	if resultsValue.Kind() != reflect.Ptr {
		return fmt.Errorf("results must be a pointer")
	}

	for resultsValue.Kind() == reflect.Ptr {
		resultsValue = resultsValue.Elem()
	}
	rowType := resultsValue.Type()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	colNum := len(colTypes)
	originRowType := rowType
	if rowType.Kind() == reflect.Slice {
		rowType = rowType.Elem()
		originRowType = rowType
		for rowType.Kind() == reflect.Ptr {
			rowType = rowType.Elem()
		}
	}

	scanValues := make([]any, colNum)
	var fieldInfos []struct {
		index []int
		typ   reflect.Type
		name  string
	}

	if rowType.Kind() == reflect.Struct {
		fieldInfos = make([]struct {
			index []int
			typ   reflect.Type
			name  string
		}, colNum)
		for colIndex, col := range colTypes {
			publicColName := makePublicVarName(col.Name())
			field, found := rowType.FieldByName(publicColName)
			if found {
				fieldInfos[colIndex].index = field.Index
				fieldInfos[colIndex].typ = field.Type
				fieldInfos[colIndex].name = publicColName
				if field.Type.Kind() == reflect.Interface {
					scanValues[colIndex] = makeValue(colTypes[colIndex].ScanType())
				} else {
					scanValues[colIndex] = makeValue(field.Type)
				}
			} else {
				fieldInfos[colIndex].index = nil
				scanValues[colIndex] = makeValue(nil)
			}
		}
	} else if rowType.Kind() == reflect.Map {
		for colIndex := range colTypes {
			if rowType.Elem().Kind() == reflect.Interface {
				scanValues[colIndex] = makeValue(colTypes[colIndex].ScanType())
			} else {
				scanValues[colIndex] = makeValue(rowType.Elem())
			}
		}
	} else if rowType.Kind() == reflect.Slice {
		for colIndex := range colTypes {
			if rowType.Elem().Kind() == reflect.Interface {
				scanValues[colIndex] = makeValue(colTypes[colIndex].ScanType())
			} else {
				scanValues[colIndex] = makeValue(rowType.Elem())
			}
		}
	} else {
		if rowType.Kind() == reflect.Interface {
			scanValues[0] = makeValue(colTypes[0].ScanType())
		} else {
			scanValues[0] = makeValue(rowType)
		}
		for colIndex := 1; colIndex < colNum; colIndex++ {
			scanValues[colIndex] = makeValue(nil)
		}
	}

	var data reflect.Value
	isNew := true
	for rows.Next() {
		err = rows.Scan(scanValues...)
		if err != nil {
			return err
		}
		if rowType.Kind() == reflect.Struct {
			if resultsValue.Kind() == reflect.Slice {
				data = reflect.New(rowType).Elem()
			} else {
				data = resultsValue
				isNew = false
			}

			for colIndex, col := range colTypes {
				fInfo := fieldInfos[colIndex]
				if fInfo.index == nil {
					continue
				}
				field := data.FieldByIndex(fInfo.index)
				valuePtr := reflect.ValueOf(scanValues[colIndex]).Elem()
				if !valuePtr.IsNil() {
					val := valuePtr.Elem()
					if fInfo.typ.String() == "time.Time" {
						str := val.String()
						tm, err := time.Parse("2006-01-02 15:04:05.000000", str)
						if err != nil {
							tm, err = time.Parse("2006-01-02 15:04:05", str)
						}
						if err == nil {
							field.Set(reflect.ValueOf(tm))
						}
					} else if val.Kind() != field.Kind() && field.Kind() != reflect.Interface {
						if field.Kind() == reflect.Ptr && val.Kind() == field.Type().Elem().Kind() {
							if val.CanAddr() {
								if field.Type().AssignableTo(val.Type()) {
									field.Set(val.Addr())
								} else if val.Type().String() == "string" {
									strVal := fixValue(col.DatabaseTypeName(), val)
									field.Set(reflect.New(field.Type().Elem()))
									field.Elem().SetString(cast.String(strVal.Interface()))
								} else if strings.Contains(field.Type().String(), "uint") {
									field.Set(reflect.New(field.Type().Elem()))
									field.Elem().SetUint(cast.Uint64(val.Interface()))
								} else if strings.Contains(field.Type().String(), "int") {
									field.Set(reflect.New(field.Type().Elem()))
									field.Elem().SetInt(cast.Int64(val.Interface()))
								} else if strings.Contains(field.Type().String(), "float") {
									field.Set(reflect.New(field.Type().Elem()))
									field.Elem().SetFloat(cast.Float64(val.Interface()))
								} else {
									field.Set(val.Addr())
								}
							}
						} else {
							convertedObject := reflect.New(field.Type())
							if s, ok := val.Interface().(string); ok {
								storedValue := new(any)
								if s != "" {
									_ = json.Unmarshal([]byte(s), storedValue)
								}
								cast.Convert(convertedObject.Interface(), storedValue)
								field.Set(convertedObject.Elem())
							} else {
								cast.Convert(convertedObject.Interface(), val.Interface())
							}
						}
					} else if field.Type().AssignableTo(val.Type()) {
						if val.Kind() == reflect.String {
							field.Set(fixValue(col.DatabaseTypeName(), val))
						} else {
							field.Set(val)
						}
					} else if val.Type().String() == "string" {
						field.Set(fixValue(col.DatabaseTypeName(), val))
					} else if strings.Contains(val.Type().String(), "int") {
						field.SetInt(val.Int())
					} else if strings.Contains(val.Type().String(), "float") {
						field.SetFloat(val.Float())
					} else {
						field.Set(val)
					}
				}
			}
		} else if rowType.Kind() == reflect.Map {
			if resultsValue.Kind() == reflect.Slice {
				data = reflect.MakeMap(rowType)
			} else {
				data = resultsValue
				isNew = false
			}
			for colIndex, col := range colTypes {
				valuePtr := reflect.ValueOf(scanValues[colIndex]).Elem()
				if !valuePtr.IsNil() {
					data.SetMapIndex(reflect.ValueOf(col.Name()), fixValue(col.DatabaseTypeName(), valuePtr.Elem()))
				} else {
					data.SetMapIndex(reflect.ValueOf(col.Name()), fixValue(col.DatabaseTypeName(), reflect.New(rowType.Elem()).Elem()))
				}
			}
		} else if rowType.Kind() == reflect.Slice {
			data = reflect.MakeSlice(rowType, colNum, colNum)
			for colIndex, col := range colTypes {
				valuePtr := reflect.ValueOf(scanValues[colIndex]).Elem()
				if !valuePtr.IsNil() {
					data.Index(colIndex).Set(fixValue(col.DatabaseTypeName(), valuePtr.Elem()))
				} else {
					data.Index(colIndex).Set(fixValue(col.DatabaseTypeName(), reflect.New(rowType.Elem()).Elem()))
				}
			}
		} else {
			valuePtr := reflect.ValueOf(scanValues[0]).Elem()
			if !valuePtr.IsNil() {
				data = fixValue(colTypes[0].DatabaseTypeName(), valuePtr.Elem())
			}
		}

		if resultsValue.Kind() == reflect.Slice {
			if originRowType.Kind() == reflect.Ptr {
				resultsValue = reflect.Append(resultsValue, data.Addr())
			} else {
				resultsValue = reflect.Append(resultsValue, data)
			}
		} else {
			resultsValue = data
			break
		}
	}

	if isNew && resultsValue.IsValid() {
		reflect.ValueOf(results).Elem().Set(resultsValue)
	}
	return nil
}

func fixValue(colType string, v reflect.Value) reflect.Value {
	if v.Kind() == reflect.String {
		str := v.String()
		switch colType {
		case "DATE":
			if len(str) >= 10 && str[4] == '-' && str[7] == '-' {
				return reflect.ValueOf(str[:10])
			}
		case "DATETIME":
			if len(str) >= 19 && str[10] == 'T' && str[4] == '-' && str[7] == '-' && str[13] == ':' && str[16] == ':' {
				str = strings.TrimRight(str, "Z")
				if len(str) > 19 && str[19] == '.' {
					return reflect.ValueOf(str[:10] + " " + str[11:])
				}
				return reflect.ValueOf(str[:10] + " " + str[11:19])
			}
		case "TIME":
			if len(str) >= 8 && str[2] == ':' && str[4] == ':' {
				if len(str) >= 15 && str[8] == '.' {
					return reflect.ValueOf(str[0:15])
				}
				return reflect.ValueOf(str[0:8])
			}
		}
	}
	return v
}

func (r *QueryResult) getColumnTypes() ([]*sql.ColumnType, error) {
	if r.rows == nil {
		return nil, errors.New("not a valid query result")
	}

	return r.rows.ColumnTypes()
}

func makePublicVarName(name string) string {
	if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
		return string(name[0]-32) + name[1:]
	}
	return name
}

func makeValue(t reflect.Type) any {
	if t == nil {
		return new(*string)
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Int:
		return new(*int)
	case reflect.Int8:
		return new(*int8)
	case reflect.Int16:
		return new(*int16)
	case reflect.Int32:
		return new(*int32)
	case reflect.Int64:
		return new(*int64)
	case reflect.Uint:
		return new(*uint)
	case reflect.Uint8:
		return new(*uint8)
	case reflect.Uint16:
		return new(*uint16)
	case reflect.Uint32:
		return new(*uint32)
	case reflect.Uint64:
		return new(*uint64)
	case reflect.Float32:
		return new(*float32)
	case reflect.Float64:
		return new(*float64)
	case reflect.Bool:
		return new(*bool)
	case reflect.String:
		return new(*string)
	}

	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		return new(*[]byte)
	}

	return new(*string)
}
