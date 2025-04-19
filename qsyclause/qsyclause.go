package qsyclause

import (
	"fmt"
	"strings"
)

// Type represents the type of SQL clause
type Type int

// SQL clause types
const (
	INSERT Type = iota
	VALUES
	SELECT
	LIMIT
	WHERE
	ORDERBY
	UPDATE
	DELETE
	COUNT
	SET
	FROM
	JOIN
)

// Clause represents a SQL clause with its values
type Clause struct {
	Type Type
	SQL  string
	Vars []interface{}
}

// Builder builds SQL statements by combining clauses
type Builder struct {
	clauses map[Type]Clause
}

// New creates a new clause builder
func New() *Builder {
	return &Builder{
		clauses: make(map[Type]Clause),
	}
}

// Set adds a clause to the builder
func (b *Builder) Set(name Type, clause ...interface{}) {
	var vars []interface{}
	var sql string

	// Extract SQL and variables from the clause
	if len(clause) > 0 {
		sql = clause[0].(string)
		vars = clause[1:]
	}

	b.clauses[name] = Clause{
		Type: name,
		SQL:  sql,
		Vars: vars,
	}
}

// Build constructs the final SQL statement based on specified clause types
func (b *Builder) Build(orders ...Type) (string, []interface{}) {
	var sqls []string
	var vars []interface{}

	for _, order := range orders {
		if clause, ok := b.clauses[order]; ok {
			sqls = append(sqls, clause.SQL)
			vars = append(vars, clause.Vars...)
		}
	}

	return strings.Join(sqls, " "), vars
}

// BuildInsert builds an INSERT statement
func BuildInsert(table string, fields []string) (string, []interface{}) {
	var placeholders []string
	for range fields {
		placeholders = append(placeholders, "?")
	}

	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", ")), nil
}

// BuildValues builds a VALUES clause for batch insert
func BuildValues(values ...interface{}) (string, []interface{}) {
	var bindStr string
	if len(values) > 0 {
		bindStr = strings.Repeat("?, ", len(values))
		bindStr = bindStr[:len(bindStr)-2] // Remove trailing ", "
	}
	return fmt.Sprintf("VALUES (%s)", bindStr), values
}

// BuildSelect builds a SELECT statement
func BuildSelect(table string, fields []string, where string) (string, []interface{}) {
	return fmt.Sprintf("SELECT %s FROM %s %s",
		strings.Join(fields, ", "),
		table,
		where), nil
}

// BuildWhere builds a WHERE clause
func BuildWhere(desc string, vars ...interface{}) (string, []interface{}) {
	return fmt.Sprintf("WHERE %s", desc), vars
}

// BuildLimit builds a LIMIT clause
func BuildLimit(limit int) (string, []interface{}) {
	return "LIMIT ?", []interface{}{limit}
}

// BuildOrderBy builds an ORDER BY clause
func BuildOrderBy(field string, desc bool) (string, []interface{}) {
	order := "ASC"
	if desc {
		order = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s", field, order), nil
}

// BuildUpdate builds an UPDATE statement
func BuildUpdate(table string, fields []string) (string, []interface{}) {
	var setStrs []string
	for _, field := range fields {
		setStrs = append(setStrs, fmt.Sprintf("%s = ?", field))
	}
	return fmt.Sprintf("UPDATE %s SET %s", table, strings.Join(setStrs, ", ")), nil
}

// BuildDelete builds a DELETE statement
func BuildDelete(table string) (string, []interface{}) {
	return fmt.Sprintf("DELETE FROM %s", table), nil
}
