package qsyschema

import (
	"go/ast"
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
	Model      interface{}
	Name       string
	Fields     []*Field
	FieldNames []string
	FieldMap   map[string]*Field
	Dialect    qsydialect.Dialect
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
		Model:    dest,
		Name:     modelType.Name(),
		FieldMap: make(map[string]*Field),
		Dialect:  d,
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
			field := &Field{
				Name: p.Name,
				Type: d.DataTypeOf(reflect.Indirect(reflect.New(p.Type))),
			}
			if v, ok := p.Tag.Lookup("qsy"); ok {
				field.Tag = v
				tags := schema.parseTag(v)
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
			}
			schema.Fields = append(schema.Fields, field)
			schema.FieldNames = append(schema.FieldNames, p.Name)
			schema.FieldMap[p.Name] = field
		}
	}
	return schema
}

func (s *Schema) GetTableName() string {
	return strings.ToLower(s.Name)
}
