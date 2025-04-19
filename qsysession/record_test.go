package qsysession_test

import (
	"database/sql"
	"log"
	"os"
	"qsyorm/qsydialect"
	"qsyorm/qsylog"
	"qsyorm/qsysession"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// 测试用结构体
type TestUser struct {
	ID   int `qsy:"primarykey;autoincrement"`
	Name string
	Age  int
}

func TestCRUD(t *testing.T) {
	// 初始化数据库和日志
	os.Remove("./test.db")
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		t.Fatal("打开数据库失败:", err)
	}
	defer db.Close()
	defer os.Remove("./test.db")

	logger := qsylog.New(log.New(os.Stdout, "", log.LstdFlags), qsylog.Config{
		Colorful: true,
		Loglevel: qsylog.Info,
	})

	dialect, _ := qsydialect.GetDialect("sqlite3")
	session := qsysession.NewSession(db, logger, dialect)

	// 创建表
	session.Model(&TestUser{})
	err = session.CreateTable()
	if err != nil {
		t.Fatal("创建表失败:", err)
	}

	t.Log("表创建成功")

	// 1. 测试插入
	t.Log("测试插入操作...")
	user1 := &TestUser{Name: "张三", Age: 25}
	user2 := &TestUser{Name: "李四", Age: 30}

	id1, err := session.Insert(user1)
	if err != nil {
		t.Fatal("插入记录失败:", err)
	}
	t.Logf("插入成功，ID: %d", id1)

	id2, err := session.Insert(user2)
	if err != nil {
		t.Fatal("插入记录失败:", err)
	}
	t.Logf("插入成功，ID: %d", id2)

	// 2. 测试查询
	t.Log("测试查询操作...")
	var users []TestUser
	err = session.Find(&users, "")
	if err != nil {
		t.Fatal("查询所有记录失败:", err)
	}

	if len(users) != 2 {
		t.Fatalf("期望查询到2条记录，实际查询到%d条记录", len(users))
	}

	t.Logf("查询到的用户: %v", users)

	// 测试条件查询
	var youngUsers []TestUser
	err = session.Find(&youngUsers, "Age < ?", 30)
	if err != nil {
		t.Fatal("条件查询失败:", err)
	}

	if len(youngUsers) != 1 {
		t.Fatalf("期望查询到1条记录，实际查询到%d条记录", len(youngUsers))
	}

	if youngUsers[0].Name != "张三" {
		t.Fatalf("查询结果错误，期望 '张三'，实际为 '%s'", youngUsers[0].Name)
	}

	t.Log("条件查询成功")

	// 3. 测试更新
	t.Log("测试更新操作...")
	updateUser := users[0]
	updateUser.Age = 26

	affected, err := session.Update(updateUser, "Name = ?", updateUser.Name)
	if err != nil {
		t.Fatal("更新记录失败:", err)
	}

	if affected != 1 {
		t.Fatalf("期望更新1条记录，实际更新%d条记录", affected)
	}

	// 验证更新
	var updatedUsers []TestUser
	err = session.Find(&updatedUsers, "Name = ?", updateUser.Name)
	if err != nil {
		t.Fatal("查询更新后的记录失败:", err)
	}

	if len(updatedUsers) != 1 || updatedUsers[0].Age != 26 {
		t.Fatal("更新操作验证失败")
	}

	t.Log("更新操作成功")

	// 4. 测试计数
	t.Log("测试计数操作...")
	count, err := session.Count("")
	if err != nil {
		t.Fatal("计数操作失败:", err)
	}

	if count != 2 {
		t.Fatalf("期望记录总数为2，实际为%d", count)
	}

	t.Log("计数操作成功")

	// 5. 测试删除
	t.Log("测试删除操作...")
	affected, err = session.Delete("Name = ?", "李四")
	if err != nil {
		t.Fatal("删除操作失败:", err)
	}

	if affected != 1 {
		t.Fatalf("期望删除1条记录，实际删除%d条记录", affected)
	}

	// 验证删除
	count, err = session.Count("")
	if err != nil {
		t.Fatal("删除后计数失败:", err)
	}

	if count != 1 {
		t.Fatalf("期望删除后记录总数为1，实际为%d", count)
	}

	t.Log("删除操作成功")
	t.Log("所有测试通过!")
}
