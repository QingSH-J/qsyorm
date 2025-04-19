package qsyengine

import (
	"database/sql"
	"fmt"
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
	if err != nil {
		// 如果log非空则记录日志
		if log != nil {
			log.Error("Error opening database connection: %s", err.Error())
		}
		return nil, err
	}

	Dialect, ok := qsydialect.GetDialect(driver)
	if !ok {
		if log != nil {
			log.Error("Dialect not found for driver: %s", driver)
		}
		return nil, fmt.Errorf("dialect not found for driver: %s", driver)
	}

	e = &QSyEngine{logger: log, dialect: Dialect, db: db}

	if err = db.Ping(); err != nil {
		e.logger.Error("Error pinging database: %s", err.Error())
		return nil, err
	}
	e.logger.Info("Successfully connected to QSyEngine")
	return
}

func (engine *QSyEngine) Close() {
	engine.logger.Info("Close QSyEngine")
	_ = engine.db.Close()
}

func (engine *QSyEngine) NewSession() *qsysession.Session {
	return qsysession.NewSession(engine.db, engine.logger, engine.dialect)
}

// Migrate 自动将结构体映射为数据库表
// 如果表不存在，则创建表；如果表存在且结构有变化，则更新表结构
func (engine *QSyEngine) Migrate(value interface{}) error {
	// 创建一个新的会话
	session := engine.NewSession()
	// 使用Model方法设置Schema，而不是直接设置
	session = session.Model(value)

	if session.Schema == nil {
		return fmt.Errorf("failed to parse model schema")
	}

	// 记录模型信息
	engine.logger.Info("Model name: %s, Field count: %d", session.Schema.Name, len(session.Schema.Fields))
	for i, field := range session.Schema.Fields {
		engine.logger.Info("Field %d: %s, Type: %s, PK: %v, Auto: %v, Index: %v, Unique: %v",
			i, field.Name, field.Type, field.IsPrimaryKey, field.IsAutoIncrement, field.Index, field.Unique)
	}

	// 输出DbFieldToGo映射内容
	engine.logger.Info("DbFieldToGo mapping:")
	for dbField, goField := range session.Schema.DbFieldToGo {
		engine.logger.Info("  DB field '%s' -> Go field '%s'", dbField, goField)
	}

	// 获取表存在性检查的SQL语句和参数
	tableName := session.Schema.GetTableName()
	engine.logger.Info("Migrating table %s", tableName)

	tableExistSQL, arg := engine.dialect.TableExist(tableName)
	engine.logger.Info("Check table exists: %s [%v]", tableExistSQL, arg)

	// 判断表是否存在
	rows, err := session.Raw(tableExistSQL, arg).QueryRows()
	if err != nil {
		engine.logger.Error("Error checking if table exists: %s", err.Error())
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		// 如果表不存在，则创建表
		engine.logger.Info("Table %s doesn't exist, creating...", tableName)
		return session.CreateTable()
	}

	// 如果表存在，记录到日志中
	engine.logger.Info("Table %s already exists", tableName)
	return nil
}

// MigrateAll 批量迁移多个结构体到数据库
func (engine *QSyEngine) MigrateAll(values ...interface{}) error {
	// 遍历所有传入的结构体，依次进行迁移
	for _, value := range values {
		if err := engine.Migrate(value); err != nil {
			return err
		}
	}
	return nil
}
