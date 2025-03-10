package qsydialect

import (
	_ "github.com/mattn/go-sqlite3"
	"reflect"
)

var dialectMap = map[string]Dialect{}

type Dialect interface {

	//SQL
	DataTypeOf(typ reflect.Value) string
	TableExist(tableName string) (string, interface{})
}

func RegisterDialect(name string, d Dialect) {
	dialectMap[name] = d
}

func GetDialect(name string) (d Dialect, ok bool) {
	d, ok = dialectMap[name]
	return
}
