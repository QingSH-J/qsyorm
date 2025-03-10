package qsysession

import (
	"fmt"
	"qsyorm/qsyschema"
	"reflect"
	"strings"
)

func (s *Session) Model(model interface{}) *Session {
	if s.Schema == nil || reflect.TypeOf(model).Kind() != reflect.TypeOf(s.Schema).Kind() {
		s.Schema = qsyschema.Parse(model, s.dialect)
	}
	return s
}

func (s *Session) CreateTable() error {
	table := s.Ref()
	var columns []string
	for _, field := range table.Fields {
		columnss := []string{field.Name, field.Type}
		if field.IsPrimaryKey {
			columnss = append(columns, "PRIMARY KEY")
		}
		if field.IsAutoIncrement {
			columnss = append(columns, "AUTOINCREMENT")
		}
		if field.Index {
			columnss = append(columns, "INDEX")
		}
		if field.Unique {
			columnss = append(columns, "UNIQUE")
		}
		columns = append(columns, strings.Join(columnss, " "))
	}
	createtablesql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", table.Name, strings.Join(columns, ","))
	_, err := s.Raw(createtablesql).Exec()
	return err
}

func (s *Session) DropTable() error {
	droptable := fmt.Sprintf("DROP TABLE IF EXISTS %s", s.Schema.GetTableName())
	_, err := s.Raw(droptable).Exec()
	return err
}
