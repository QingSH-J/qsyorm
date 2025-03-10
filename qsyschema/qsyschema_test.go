package qsyschema

import (
	"qsyorm/qsydialect"
	"testing"
)

type User struct {
	Name string `qsy:"primarykey"`
	Age  int
}

var testDialect, _ = qsydialect.GetDialect("sqlite3")

func TestParse(t *testing.T) {
	schema := Parse(&User{}, testDialect)
	if schema.Name != "User" {
		t.Fatal("schema.Name is not User")
	}
	if field, ok := schema.FieldMap["Name"]; !ok || field.Tag != "primarykey" {
		t.Fatal("Name field tag is not primarykey")
	}
	if !schema.FieldMap["Name"].IsPrimaryKey {
		t.Fatal("Name field is not marked as primary key")
	}
	ageField, ok := schema.FieldMap["Age"]
	if !ok {
		t.Fatal("Age field not found in schema")
	}
	t.Logf("Age field type: %s", ageField.Type)
}
