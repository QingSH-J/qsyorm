package qsysession

import (
	"database/sql"
	"log"
	"qsyorm/qsydialect"
	"qsyorm/qsylog"
	"qsyorm/qsyschema"
	"strings"
)

type Session struct {
	db          *sql.DB
	sql         strings.Builder
	sqlvars     []interface{}
	Schema      *qsyschema.Schema
	Logger      qsylog.Interface
	dialect     qsydialect.Dialect
	schemaCache map[string]*qsyschema.Schema
}

func NewSession(db *sql.DB, log qsylog.Interface, d qsydialect.Dialect) *Session {
	return &Session{db: db, Logger: log, dialect: d}
}

func (s *Session) Clear() {
	s.sql.Reset()
	s.sqlvars = nil
}

func (s *Session) DB() *sql.DB {
	return s.db
}

func (s *Session) Raw(sql string, value ...interface{}) *Session {
	s.sql.WriteString(sql)
	s.sql.WriteString(" ")
	s.sqlvars = append(s.sqlvars, value...)
	return s
}

func (s *Session) Exec() (result sql.Result, err error) {
	defer s.Clear()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	s.Logger.Info(s.sql.String(), s.sqlvars...)
	if result, err = s.DB().Exec(s.sql.String(), s.sqlvars...); err != nil {
		s.Logger.Error(err.Error())
	}
	return
}
func (s *Session) QueryRow() *sql.Row {
	defer s.Clear()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	s.Logger.Info(s.sql.String(), s.sqlvars...)
	return s.DB().QueryRow(s.sql.String(), s.sqlvars...)
}
func (s *Session) QueryRows() (rows *sql.Rows, err error) {
	defer s.Clear()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	s.Logger.Info(s.sql.String(), s.sqlvars...)
	if rows, err = s.DB().Query(s.sql.String(), s.sqlvars...); err != nil {
		s.Logger.Error(err.Error())
	}
	return
}

func (s *Session) Ref() *qsyschema.Schema {
	if s.Schema == nil {
		s.Logger.Error("Ref schema is nil")
	}
	return s.Schema
}
