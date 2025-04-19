package qsydialect

import (
	"fmt"
	"reflect"
	"time"
)

type sqlite3 struct{}

var _ Dialect = (*sqlite3)(nil)

func init() {
	RegisterDialect("sqlite3", &sqlite3{})
}

func (s *sqlite3) DataTypeOf(d reflect.Value) string {
	switch d.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "INTEGER"
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Int64, reflect.Uint64:
		return "BIGINT"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.String:
		return "TEXT"
	//tackle with the []byte
	case reflect.Array, reflect.Slice:
		if d.Type().Elem().Kind() == reflect.Uint8 {
			return "BINARY"
		}
	//tackle with the time.Time Struct
	case reflect.Struct:
		if d.Type() == reflect.TypeOf(time.Time{}) {
			return "DATETIME"
		}
		panic(fmt.Sprintf("invalid sql type %s (%s)", d.Type().Name(), d.Kind()))

	}
	return "TEXT"
}

func (s *sqlite3) TableExist(tableName string) (string, interface{}) {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
	return query, tableName
}
