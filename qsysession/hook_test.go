package qsysession_test

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"qsyorm/qsydialect"
	"qsyorm/qsylog"
	"qsyorm/qsysession"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// HookTestUser 是一个实现了 Hook 接口的测试模型
type HookTestUser struct {
	ID        int `qsy:"primarykey;autoincrement"`
	Name      string
	Age       int
	CreatedAt string // 存储为字符串格式的时间戳，更适合 SQLite
	UpdatedAt string // 存储为字符串格式的时间戳，更适合 SQLite
}

// 格式化当前时间为标准格式字符串
func formattedTime() string {
	return time.Now().Format(time.RFC3339)
}

// BeforeInsert 在插入前被调用
func (u *HookTestUser) BeforeInsert() error {
	fmt.Println("BeforeInsert 被调用")
	u.CreatedAt = formattedTime()
	u.UpdatedAt = formattedTime()
	return nil
}

// AfterInsert 在插入后被调用
func (u *HookTestUser) AfterInsert() error {
	fmt.Println("AfterInsert 被调用，记录 ID:", u.ID)
	return nil
}

// BeforeUpdate 在更新前被调用
func (u *HookTestUser) BeforeUpdate() error {
	fmt.Println("BeforeUpdate 被调用")
	u.UpdatedAt = formattedTime()
	fmt.Printf("BeforeUpdate 更新的时间戳: %v\n", u.UpdatedAt)
	return nil
}

// AfterUpdate 在更新后被调用
func (u *HookTestUser) AfterUpdate() error {
	fmt.Println("AfterUpdate 被调用")
	return nil
}

// BeforeDelete 在删除前被调用
func (u *HookTestUser) BeforeDelete() error {
	fmt.Println("BeforeDelete 被调用")
	return nil
}

// AfterDelete 在删除后被调用
func (u *HookTestUser) AfterDelete() error {
	fmt.Println("AfterDelete 被调用")
	return nil
}

// BeforeQuery 在查询前被调用
func (u *HookTestUser) BeforeQuery() error {
	fmt.Println("BeforeQuery 被调用")
	return nil
}

// AfterQuery 在查询后被调用
func (u *HookTestUser) AfterQuery() error {
	fmt.Printf("AfterQuery 被调用，加载记录: %s, 时间戳: %v\n", u.Name, u.UpdatedAt)
	return nil
}

func TestHook(t *testing.T) {
	// 初始化数据库和日志
	os.Remove("./test_hook.db")
	db, err := sql.Open("sqlite3", "./test_hook.db")
	if err != nil {
		t.Fatal("打开数据库失败:", err)
	}
	defer db.Close()
	defer os.Remove("./test_hook.db")

	logger := qsylog.New(log.New(os.Stdout, "", log.LstdFlags), qsylog.Config{
		Colorful: true,
		Loglevel: qsylog.Info,
	})

	dialect, _ := qsydialect.GetDialect("sqlite3")
	session := qsysession.NewSession(db, logger, dialect)

	// 创建表
	session.Model(&HookTestUser{})
	err = session.CreateTable()
	if err != nil {
		t.Fatal("创建表失败:", err)
	}

	t.Log("表创建成功")

	// 1. 测试插入钩子
	t.Log("测试插入钩子...")
	user := &HookTestUser{Name: "张三", Age: 25}

	id, err := session.Insert(user)
	if err != nil {
		t.Fatal("插入记录失败:", err)
	}
	t.Logf("插入成功，ID: %d, CreatedAt: %s", id, user.CreatedAt)

	// 让我们直接用SQL检查创建的记录
	var initialUpdatedAt string
	rawCheckSQL := "SELECT UpdatedAt FROM hooktestuser WHERE ID = ?"
	if err := db.QueryRow(rawCheckSQL, id).Scan(&initialUpdatedAt); err != nil {
		t.Fatalf("无法查询初始时间戳: %v", err)
	}
	t.Logf("初始的UpdatedAt: %s", initialUpdatedAt)

	// 等待一会儿确保时间戳会不同
	time.Sleep(1 * time.Second)

	// 3. 测试更新钩子 - 手动构建一个更新对象
	t.Log("测试更新钩子...")
	updateObj := &HookTestUser{
		ID:        int(id),
		Name:      "张三",
		Age:       26, // 更新年龄
		CreatedAt: initialUpdatedAt,
		UpdatedAt: initialUpdatedAt,
	}

	// 打印更新前的信息
	t.Logf("更新前 - ID: %d, Age: %d, UpdatedAt: %s",
		updateObj.ID, updateObj.Age, updateObj.UpdatedAt)

	// 执行更新
	affected, err := session.Update(updateObj, "ID = ?", updateObj.ID)
	if err != nil {
		t.Fatal("更新记录失败:", err)
	}
	t.Logf("更新影响的行数: %d", affected)

	// 直接从数据库读取更新后的值
	var newAge int
	var newUpdatedAt string
	if err := db.QueryRow("SELECT Age, UpdatedAt FROM hooktestuser WHERE ID = ?", id).Scan(&newAge, &newUpdatedAt); err != nil {
		t.Fatalf("无法读取更新后的记录: %v", err)
	}

	t.Logf("数据库中的值 - Age: %d, UpdatedAt: %s", newAge, newUpdatedAt)

	// 验证年龄是否已更新
	if newAge != 26 {
		t.Fatalf("年龄未更新，期望: 26, 实际: %d", newAge)
	}

	// 验证时间戳是否已更新
	if newUpdatedAt == initialUpdatedAt {
		t.Fatal("更新时间戳未被钩子修改")
	}

	t.Logf("更新前的时间戳: %s", initialUpdatedAt)
	t.Logf("更新后的时间戳: %s", newUpdatedAt)
	t.Log("更新钩子测试成功")

	// 4. 测试删除钩子
	t.Log("测试删除钩子...")
	_, err = session.Delete("ID = ?", id)
	if err != nil {
		t.Fatal("删除记录失败:", err)
	}

	// 验证记录是否已删除
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM hooktestuser WHERE ID = ?", id).Scan(&count); err != nil {
		t.Fatalf("验证删除失败: %v", err)
	}

	if count != 0 {
		t.Fatal("记录未被成功删除")
	}

	t.Log("删除钩子测试成功")
	t.Log("所有钩子测试通过!")
}
