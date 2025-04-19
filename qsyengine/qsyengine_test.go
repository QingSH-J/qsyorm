package qsyengine

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"qsyorm/qsylog"

	_ "github.com/mattn/go-sqlite3"
)

// 测试用的模型结构体
type User struct {
	ID   int `qsy:"primarykey;autoincrement"`
	Name string
	Age  int `qsy:"index"`
}

type Article struct {
	ID       int    `qsy:"primarykey;autoincrement"`
	Title    string `qsy:"unique"`
	Content  string
	AuthorID int
}

func TestMigrate(t *testing.T) {
	// 确保每个子测试使用不同的数据库文件
	t.Run("TestMigrateSingleModel", func(t *testing.T) {
		testMigrateSingleModel(t)
	})

	// 等待确保上一个测试完全释放资源
	time.Sleep(100 * time.Millisecond)

	t.Run("TestMigrateAllModels", func(t *testing.T) {
		testMigrateAllModels(t)
	})

	// 等待确保上一个测试完全释放资源
	time.Sleep(100 * time.Millisecond)

	t.Run("TestMigrateExistingTable", func(t *testing.T) {
		testMigrateExistingTable(t)
	})
}

func testMigrateSingleModel(t *testing.T) {
	// 创建临时数据库文件，确保唯一性
	dbFile := fmt.Sprintf("test_migrate_single_%d.db", time.Now().UnixNano())
	t.Logf("Using database file: %s", dbFile)

	// 在测试结束后删除临时数据库
	defer func() {
		if err := os.Remove(dbFile); err != nil {
			t.Logf("Warning: could not remove test database file: %v", err)
		}
	}()

	// 初始化日志
	logger := qsylog.New(log.New(os.Stdout, "", log.LstdFlags), qsylog.Config{
		Colorful: true,
		Loglevel: qsylog.Info,
	})

	// 创建引擎
	engine, err := NewQSyEngine("sqlite3", dbFile, logger)
	if err != nil {
		t.Fatal("failed to create engine:", err)
	}
	defer engine.Close()

	// 确认数据库连接正常
	testDb(t, engine)

	err = engine.Migrate(&User{})
	if err != nil {
		t.Fatal("failed to migrate User model:", err)
	}

	// 打印模型信息
	session := engine.NewSession()
	session.Model(&User{})
	schema := session.Schema

	t.Logf("Migrated model - Name: %s, Table: %s", schema.Name, schema.GetTableName())
	for i, field := range schema.Fields {
		t.Logf("Field %d: Name=%s, Type=%s, PK=%v, Auto=%v, Index=%v, Unique=%v",
			i, field.Name, field.Type, field.IsPrimaryKey, field.IsAutoIncrement, field.Index, field.Unique)
	}

	// 验证表是否创建成功
	rows, err := session.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name=?", "user").QueryRows()
	if err != nil {
		t.Fatal("failed to query tables:", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("user table not created")
	}

	// 验证表结构是否正确 - 使用带引号的表名
	var columns []string
	colRows, err := session.Raw("PRAGMA table_info(user)").QueryRows()
	if err != nil {
		t.Fatal("failed to query table info:", err)
	}
	defer colRows.Close()

	for colRows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var dflt_value sql.NullString // 使用sql.NullString处理可能为NULL的值

		if err := colRows.Scan(&cid, &name, &typ, &notnull, &dflt_value, &pk); err != nil {
			t.Fatal("failed to scan column info:", err)
		}

		defaultVal := "NULL"
		if dflt_value.Valid {
			defaultVal = dflt_value.String
		}

		t.Logf("Found column: %s (type: %s, default: %s)", name, typ, defaultVal)
		columns = append(columns, strings.ToUpper(name)) // 转换为大写以进行不区分大小写的比较
	}

	// 检查必要的列是否存在（不区分大小写）
	requiredColumns := []string{"ID", "NAME", "AGE"} // 全部大写
	for _, col := range requiredColumns {
		found := false
		for _, column := range columns {
			if strings.ToUpper(column) == col {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("column %s not found in user table, available columns: %v", col, columns)
		}
	}

	// 额外打印表的DDL
	ddlRows, err := session.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", "user").QueryRows()
	if err != nil {
		t.Fatal("failed to query table DDL:", err)
	}
	defer ddlRows.Close()

	if ddlRows.Next() {
		var ddl string
		if err := ddlRows.Scan(&ddl); err != nil {
			t.Fatal("failed to scan DDL:", err)
		}
		t.Logf("Table DDL: %s", ddl)
	}
}

func testMigrateAllModels(t *testing.T) {
	// 创建临时数据库文件，确保唯一性
	dbFile := fmt.Sprintf("test_migrate_all_%d.db", time.Now().UnixNano())
	t.Logf("Using database file: %s", dbFile)

	// 在测试结束后删除临时数据库
	defer func() {
		if err := os.Remove(dbFile); err != nil {
			t.Logf("Warning: could not remove test database file: %v", err)
		}
	}()

	// 初始化日志
	logger := qsylog.New(log.New(os.Stdout, "", log.LstdFlags), qsylog.Config{
		Colorful: true,
		Loglevel: qsylog.Info,
	})

	// 创建引擎 - 使用更稳定的SQLite配置
	engine, err := NewQSyEngine("sqlite3", dbFile+"?_timeout=5000&_journal=WAL&_sync=NORMAL", logger)
	if err != nil {
		t.Fatal("failed to create engine:", err)
	}
	defer engine.Close()

	// 确认数据库连接正常
	testDb(t, engine)

	err = engine.MigrateAll(&User{}, &Article{})
	if err != nil {
		t.Fatal("failed to migrate models:", err)
	}

	// 验证所有表是否创建成功
	session := engine.NewSession()
	tables := []string{"user", "article"}

	for _, table := range tables {
		rows, err := session.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).QueryRows()
		if err != nil {
			t.Fatalf("failed to query table %s: %v", table, err)
		}
		defer rows.Close()

		if !rows.Next() {
			t.Fatalf("%s table not created", table)
		}

		// 打印表结构 - 直接将表名嵌入SQL中
		t.Logf("Table %s structure:", table)
		var pragmaSQL string
		if table == "user" {
			pragmaSQL = "PRAGMA table_info(user)"
		} else {
			pragmaSQL = "PRAGMA table_info(article)"
		}

		colRows, err := session.Raw(pragmaSQL).QueryRows()
		if err != nil {
			t.Logf("Warning: failed to query %s table info: %v", table, err)
			continue
		}
		defer colRows.Close()

		for colRows.Next() {
			var cid, notnull, pk int
			var name, typ string
			var dflt_value sql.NullString // 使用sql.NullString处理可能为NULL的值

			if err := colRows.Scan(&cid, &name, &typ, &notnull, &dflt_value, &pk); err != nil {
				t.Logf("Warning: failed to scan column info: %v", err)
				continue
			}

			defaultVal := "NULL"
			if dflt_value.Valid {
				defaultVal = dflt_value.String
			}

			t.Logf("  Column: %s (type: %s, default: %s)", name, typ, defaultVal)
		}
	}
}

func testMigrateExistingTable(t *testing.T) {
	// 创建临时数据库文件，确保唯一性
	dbFile := fmt.Sprintf("test_migrate_existing_%d.db", time.Now().UnixNano())
	t.Logf("Using database file: %s", dbFile)

	// 在测试结束后删除临时数据库
	defer func() {
		if err := os.Remove(dbFile); err != nil {
			t.Logf("Warning: could not remove test database file: %v", err)
		}
	}()

	// 初始化日志
	logger := qsylog.New(log.New(os.Stdout, "", log.LstdFlags), qsylog.Config{
		Colorful: true,
		Loglevel: qsylog.Info,
	})

	// 创建引擎
	engine, err := NewQSyEngine("sqlite3", dbFile, logger)
	if err != nil {
		t.Fatal("failed to create engine:", err)
	}
	defer engine.Close()

	// 确认数据库连接正常
	testDb(t, engine)

	// 第一次迁移
	err = engine.Migrate(&User{})
	if err != nil {
		t.Fatal("failed to migrate User model:", err)
	}

	// 再次迁移应该不会出错
	err = engine.Migrate(&User{})
	if err != nil {
		t.Fatal("failed to re-migrate User model again:", err)
	}

	t.Log("Successfully re-migrated existing table")
}

// 测试数据库连接
func testDb(t *testing.T, engine *QSyEngine) {
	session := engine.NewSession()
	// 执行一个简单的SQL语句，验证数据库连接正常
	_, err := session.Raw("SELECT 1").Exec()
	if err != nil {
		t.Fatal("database connection test failed:", err)
	}
	t.Log("Database connection successful")
}

// 测试数据库操作
func TestDatabaseOperations(t *testing.T) {
	// 创建临时数据库文件，确保唯一性
	dbFile := fmt.Sprintf("test_operations_%d.db", time.Now().UnixNano())
	t.Logf("Using database file: %s", dbFile)

	// 在测试结束后删除临时数据库
	defer func() {
		if err := os.Remove(dbFile); err != nil {
			t.Logf("Warning: could not remove test database file: %v", err)
		}
	}()

	// 初始化日志
	logger := qsylog.New(log.New(os.Stdout, "", log.LstdFlags), qsylog.Config{
		Colorful: true,
		Loglevel: qsylog.Info,
	})

	// 创建引擎
	engine, err := NewQSyEngine("sqlite3", dbFile, logger)
	if err != nil {
		t.Fatal("failed to create engine:", err)
	}
	defer engine.Close()

	// 确认数据库连接正常
	testDb(t, engine)

	// 打印模型信息
	printModelInfo(t, engine, &User{})

	// 迁移模型
	if err := engine.Migrate(&User{}); err != nil {
		t.Fatal("failed to migrate User model:", err)
	}

	// 创建会话
	session := engine.NewSession()

	// 插入数据
	_, err = session.Raw("INSERT INTO user(Name, Age) VALUES(?, ?)", "Tom", 18).Exec()
	if err != nil {
		t.Fatal("failed to insert data:", err)
	}

	// 查询数据
	var user struct {
		ID   int
		Name string
		Age  int
	}

	row := session.Raw("SELECT * FROM user WHERE Name = ?", "Tom").QueryRow()
	if err := row.Scan(&user.ID, &user.Name, &user.Age); err != nil {
		t.Fatal("failed to query data:", err)
	}

	// 验证数据
	if user.Name != "Tom" || user.Age != 18 {
		t.Fatalf("unexpected data: got name=%s age=%d, want name=Tom age=18", user.Name, user.Age)
	}

	t.Logf("Successfully inserted and retrieved data: ID=%d, Name=%s, Age=%d", user.ID, user.Name, user.Age)
}

// 打印模型信息
func printModelInfo(t *testing.T, engine *QSyEngine, model interface{}) {
	session := engine.NewSession()
	session.Model(model)
	schema := session.Schema

	t.Logf("Model name: %s", schema.Name)
	t.Logf("Table name: %s", schema.GetTableName())
	t.Logf("Fields count: %d", len(schema.Fields))

	for i, field := range schema.Fields {
		t.Logf("Field %d: Name=%s, Type=%s, PrimaryKey=%v, AutoIncrement=%v, Index=%v, Unique=%v",
			i, field.Name, field.Type, field.IsPrimaryKey, field.IsAutoIncrement, field.Index, field.Unique)
	}
}
