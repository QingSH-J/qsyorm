package qsysession

import (
	"fmt"
	"qsyorm/qsyschema"
	"reflect"
	"strings"
)

func (s *Session) Model(model interface{}) *Session {
	// 获取模型的具体类型
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	// 获取Schema模型的类型（如果存在）
	var schemaType reflect.Type
	if s.Schema != nil && s.Schema.Model != nil {
		schemaType = reflect.TypeOf(s.Schema.Model)
		if schemaType.Kind() == reflect.Ptr {
			schemaType = schemaType.Elem()
		}
	}

	// 如果Schema为nil或者模型类型不同，则重新解析
	if s.Schema == nil || schemaType != modelType {
		s.Logger.Info("Creating new Schema for type: %s", modelType.Name())
		s.Schema = qsyschema.Parse(model, s.dialect)
	} else {
		s.Logger.Info("Reusing existing Schema for type: %s", modelType.Name())
	}

	return s
}

func (s *Session) CreateTable() error {
	table := s.Ref()
	var columns []string
	var indexes []string

	// 检查是否有字段
	if len(table.Fields) == 0 {
		return fmt.Errorf("no fields in model %s", table.Name)
	}

	for _, field := range table.Fields {
		// SQLite对字段名不区分大小写，但在创建时保留原始大小写
		fieldName := field.Name

		// 处理 SQLite 的特殊情况
		if field.Type == "INTEGER" && field.IsPrimaryKey && field.IsAutoIncrement {
			// SQLite 要求 AUTOINCREMENT 必须按照 INTEGER PRIMARY KEY AUTOINCREMENT 顺序
			columns = append(columns, fmt.Sprintf("%s INTEGER PRIMARY KEY AUTOINCREMENT", fieldName))
		} else {
			columnss := []string{fieldName, field.Type}
			if field.IsPrimaryKey && !field.IsAutoIncrement {
				columnss = append(columnss, "PRIMARY KEY")
			}
			if field.Unique {
				columnss = append(columnss, "UNIQUE")
			}
			columns = append(columns, strings.Join(columnss, " "))
		}

		// 对需要创建索引的字段，添加到索引列表
		if field.Index {
			indexSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s(%s);",
				strings.ToLower(table.Name), strings.ToLower(field.Name),
				strings.ToLower(table.Name), field.Name)
			indexes = append(indexes, indexSQL)
		}
	}

	// 确保表名使用的是小写
	tableName := strings.ToLower(table.Name)

	// 使用表名的小写形式
	createtablesql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(columns, ", "))
	s.Logger.Info("SQL: %s", createtablesql)
	if _, err := s.Raw(createtablesql).Exec(); err != nil {
		s.Logger.Error("Failed to create table: %s", err.Error())
		return err
	}

	// 创建索引
	for _, indexSQL := range indexes {
		s.Logger.Info("SQL: %s", indexSQL)
		if _, err := s.Raw(indexSQL).Exec(); err != nil {
			s.Logger.Error("Failed to create index: %s", err.Error())
			return err
		}
	}

	return nil
}

func (s *Session) DropTable() error {
	droptable := fmt.Sprintf("DROP TABLE IF EXISTS %s", s.Schema.GetTableName())
	_, err := s.Raw(droptable).Exec()
	return err
}
