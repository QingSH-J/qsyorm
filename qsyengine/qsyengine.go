package qsyengine

import (
	"database/sql"
	"qsyorm/qsydialect"
	"qsyorm/qsylog"
	"qsyorm/qsysession"
)

type QSyEngine struct {
	db      *sql.DB
	logger  qsylog.Interface
	dialect qsydialect.Dialect
}

func NewQSyEngine(driver, source string, log qsylog.Interface) (e *QSyEngine, err error) {
	db, err := sql.Open(driver, source)
	Dialect, _ := qsydialect.GetDialect(driver)
	e = &QSyEngine{logger: log, dialect: Dialect, db: db}
	if err != nil {
		e.logger.Error(err.Error())
		return
	}

	if err = db.Ping(); err != nil {
		e.logger.Error(err.Error())
		return
	}
	e.logger.Info("Successfully connected to QSyEngine")
	e = &QSyEngine{db: db, dialect: Dialect, logger: log}
	return
}

func (engine *QSyEngine) Close() {
	engine.logger.Info("Close QSyEngine")
	engine.db.Close()
}

func (engine *QSyEngine) NewSession() *qsysession.Session {
	return qsysession.NewSession(engine.db, engine.logger, engine.dialect)
}
