package qsyschema

import (
	"go/ast"
	"qsyorm/qsyclause"
	"qsyorm/qsydialect"
	"reflect"
	"strings"
)

type Field struct {
	Name            string
	Type            string
	Tag             string
	IsPrimaryKey    bool
	IsAutoIncrement bool
	Index           bool
	Unique          bool
}

type Schema struct {
	Model       interface{}
	Name        string
	Fields      []*Field
	FieldNames  []string          // 存储数据库列名
	FieldMap    map[string]*Field // 数据库列名到Field的映射
	DbFieldToGo map[string]string // 数据库列名到Go字段名的映射
	Dialect     qsydialect.Dialect
	Clause      *qsyclause.Builder
}

func (s *Schema) GetField(name string) *Field {
	return s.FieldMap[name]
}

// parse tag
// "primarykey;not null" is a tag
func (s *Schema) parseTag(tag string) map[string]string {
	result := make(map[string]string)
	if tag == "" {
		return result
	}

	for _, field := range strings.Split(tag, ";") {
		if field == "" {
			continue
		}

		kv := strings.Split(field, ":")
		if len(kv) == 0 {
			continue
		}

		if kv[0] == "" {
			continue
		}
		switch len(kv) {
		case 1:
			result[kv[0]] = ""
		case 2:
			result[kv[0]] = kv[1]
		default:
			result[kv[0]] = strings.Join(kv[1:], ":")
		}
	}
	return result
}

func Parse(dest interface{}, d qsydialect.Dialect) *Schema {
	modelType := reflect.Indirect(reflect.ValueOf(dest)).Type()
	schema := &Schema{
		Model:       dest,
		Name:        modelType.Name(),
		FieldMap:    make(map[string]*Field),
		DbFieldToGo: make(map[string]string),
		Dialect:     d,
	}
	if d == nil {
		panic("qsydialect: nil Dialect")
	}
	for i := 0; i < modelType.NumField(); i++ {
		p := modelType.Field(i)
		if !p.IsExported() {
			continue
		}
		if !p.Anonymous && ast.IsExported(p.Name) {
			// 使用原始Go字段名作为Schema.Field.Name，这样插入时使用这个作为数据库列名
			// 确保大小写一致性
			field := &Field{
				Name: p.Name, // 使用结构体原始字段名
				Type: d.DataTypeOf(reflect.Indirect(reflect.New(p.Type))),
			}

			if v, ok := p.Tag.Lookup("qsy"); ok {
				field.Tag = v
				tags := schema.parseTag(v)

				// name不再覆盖默认字段名，但建立Go字段名到数据库列名的双向映射
				// 这样在ORM内部使用Field.Name（Go字段名）时，数据库操作也能正常工作
				if name, ok := tags["name"]; ok && name != "" {
					// 在DbFieldToGo中保存映射，但Field.Name仍使用Go字段名
					schema.DbFieldToGo[name] = p.Name   // 数据库列名 -> Go字段名
					schema.DbFieldToGo[p.Name] = p.Name // Go字段名 -> Go字段名（自映射，保证查找一致）
				} else {
					// 没有name标签，Go字段名映射到自身
					schema.DbFieldToGo[p.Name] = p.Name
				}

				if _, ok := tags["primarykey"]; ok {
					field.IsPrimaryKey = true
				}
				if _, ok := tags["autoincrement"]; ok {
					field.IsAutoIncrement = true
				}
				if _, ok := tags["unique"]; ok {
					field.Unique = true
				}
				if _, ok := tags["index"]; ok {
					field.Index = true
				}
			} else {
				// 如果没有qsy标签，Go字段名映射到自身
				schema.DbFieldToGo[p.Name] = p.Name
			}

			schema.Fields = append(schema.Fields, field)
			schema.FieldNames = append(schema.FieldNames, field.Name)
			schema.FieldMap[field.Name] = field
		}
	}
	return schema
}

// GetTableName 返回结构体对应的表名，默认使用结构体名称的小写形式
func (s *Schema) GetTableName() string {
	// 保持使用小写表名，与之前一致
	return strings.ToLower(s.Name)
}
