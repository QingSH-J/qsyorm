package qsyclause

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	builder := New()
	if builder == nil {
		t.Fatal("New() returned nil")
	}
	if builder.clauses == nil {
		t.Fatal("New() returned builder with nil clauses map")
	}
}

func TestSet(t *testing.T) {
	builder := New()
	builder.Set(INSERT, "INSERT INTO users", 1, 2)

	clause, ok := builder.clauses[INSERT]
	if !ok {
		t.Fatal("Set() did not add clause to builder")
	}

	if clause.Type != INSERT {
		t.Errorf("Expected Type to be INSERT, got %v", clause.Type)
	}

	if clause.SQL != "INSERT INTO users" {
		t.Errorf("Expected SQL to be 'INSERT INTO users', got '%s'", clause.SQL)
	}

	expectedVars := []interface{}{1, 2}
	if !reflect.DeepEqual(clause.Vars, expectedVars) {
		t.Errorf("Expected Vars to be %v, got %v", expectedVars, clause.Vars)
	}
}

func TestBuild(t *testing.T) {
	builder := New()
	builder.Set(SELECT, "SELECT * FROM users")
	builder.Set(WHERE, "WHERE age > ?", 18)
	builder.Set(ORDERBY, "ORDER BY name")

	sql, vars := builder.Build(SELECT, WHERE, ORDERBY)

	expectedSQL := "SELECT * FROM users WHERE age > ? ORDER BY name"
	if sql != expectedSQL {
		t.Errorf("Expected SQL to be '%s', got '%s'", expectedSQL, sql)
	}

	expectedVars := []interface{}{18}
	if !reflect.DeepEqual(vars, expectedVars) {
		t.Errorf("Expected Vars to be %v, got %v", expectedVars, vars)
	}
}

func TestBuildInsert(t *testing.T) {
	sql, vars := BuildInsert("users", []string{"name", "age"})

	expectedSQL := "INSERT INTO users (name, age) VALUES (?, ?)"
	if sql != expectedSQL {
		t.Errorf("Expected SQL to be '%s', got '%s'", expectedSQL, sql)
	}

	if vars != nil {
		t.Errorf("Expected Vars to be nil, got %v", vars)
	}
}

func TestBuildValues(t *testing.T) {
	sql, vars := BuildValues("John", 25)

	expectedSQL := "VALUES (?, ?)"
	if sql != expectedSQL {
		t.Errorf("Expected SQL to be '%s', got '%s'", expectedSQL, sql)
	}

	expectedVars := []interface{}{"John", 25}
	if !reflect.DeepEqual(vars, expectedVars) {
		t.Errorf("Expected Vars to be %v, got %v", expectedVars, vars)
	}
}

func TestBuildSelect(t *testing.T) {
	sql, vars := BuildSelect("users", []string{"name", "age"}, "WHERE age > 18")

	expectedSQL := "SELECT name, age FROM users WHERE age > 18"
	if sql != expectedSQL {
		t.Errorf("Expected SQL to be '%s', got '%s'", expectedSQL, sql)
	}

	if vars != nil {
		t.Errorf("Expected Vars to be nil, got %v", vars)
	}
}

func TestBuildWhere(t *testing.T) {
	sql, vars := BuildWhere("age > ? AND name = ?", 18, "John")

	expectedSQL := "WHERE age > ? AND name = ?"
	if sql != expectedSQL {
		t.Errorf("Expected SQL to be '%s', got '%s'", expectedSQL, sql)
	}

	expectedVars := []interface{}{18, "John"}
	if !reflect.DeepEqual(vars, expectedVars) {
		t.Errorf("Expected Vars to be %v, got %v", expectedVars, vars)
	}
}

func TestBuildLimit(t *testing.T) {
	sql, vars := BuildLimit(10)

	expectedSQL := "LIMIT ?"
	if sql != expectedSQL {
		t.Errorf("Expected SQL to be '%s', got '%s'", expectedSQL, sql)
	}

	expectedVars := []interface{}{10}
	if !reflect.DeepEqual(vars, expectedVars) {
		t.Errorf("Expected Vars to be %v, got %v", expectedVars, vars)
	}
}

func TestBuildOrderBy(t *testing.T) {
	sql, vars := BuildOrderBy("age", true)

	expectedSQL := "ORDER BY age DESC"
	if sql != expectedSQL {
		t.Errorf("Expected SQL to be '%s', got '%s'", expectedSQL, sql)
	}

	if vars != nil {
		t.Errorf("Expected Vars to be nil, got %v", vars)
	}

	sql, vars = BuildOrderBy("name", false)

	expectedSQL = "ORDER BY name ASC"
	if sql != expectedSQL {
		t.Errorf("Expected SQL to be '%s', got '%s'", expectedSQL, sql)
	}
}

func TestBuildUpdate(t *testing.T) {
	sql, vars := BuildUpdate("users", []string{"name", "age"})

	expectedSQL := "UPDATE users SET name = ?, age = ?"
	if sql != expectedSQL {
		t.Errorf("Expected SQL to be '%s', got '%s'", expectedSQL, sql)
	}

	if vars != nil {
		t.Errorf("Expected Vars to be nil, got %v", vars)
	}
}

func TestBuildDelete(t *testing.T) {
	sql, vars := BuildDelete("users")

	expectedSQL := "DELETE FROM users"
	if sql != expectedSQL {
		t.Errorf("Expected SQL to be '%s', got '%s'", expectedSQL, sql)
	}

	if vars != nil {
		t.Errorf("Expected Vars to be nil, got %v", vars)
	}
}
