package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"qsyorm/qsyengine"
	"qsyorm/qsylog"

	"github.com/gin-gonic/gin"
)

// User 用户模型
type User struct {
	ID       int64  `qsy:"name:ID;primarykey;autoincrement"`
	Username string `qsy:"name:Username;unique"`
	Password string `qsy:"name:Password"`
	Age      int    `qsy:"name:Age;index"`
	Created  string `qsy:"name:Created"`
}

// Article 文章模型
type Article struct {
	ID        int64  `qsy:"name:ID;primarykey;autoincrement"`
	Title     string `qsy:"name:Title;index"`
	Content   string `qsy:"name:Content"`
	UserID    int64  `qsy:"name:UserID;index"`
	CreatedAt string `qsy:"name:CreatedAt"`
}

var (
	dbEngine *qsyengine.QSyEngine
	dbPath   = "qsyorm.db"
)

// 初始化数据库引擎
func initEngine() error {
	var err error

	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前工作目录失败: %v", err)
	}
	log.Printf("当前工作目录: %s", currentDir)

	fullDbPath := filepath.Join(currentDir, dbPath)
	log.Printf("数据库路径: %s", fullDbPath)

	// 删除可能存在的测试数据库
	if _, err := os.Stat(fullDbPath); err == nil {
		log.Printf("删除已存在的数据库文件: %s", fullDbPath)
		os.Remove(fullDbPath)
	}

	// 创建数据库引擎
	log.Printf("初始化数据库引擎: sqlite3, %s", fullDbPath)
	dbEngine, err = qsyengine.NewQSyEngine("sqlite3", fullDbPath, qsylog.Default)
	if err != nil {
		return fmt.Errorf("初始化数据库引擎失败: %v", err)
	}

	// 迁移表结构
	log.Println("开始迁移User表")
	err = dbEngine.Migrate(&User{})
	if err != nil {
		return fmt.Errorf("迁移User表失败: %v", err)
	}

	log.Println("开始迁移Article表")
	err = dbEngine.Migrate(&Article{})
	if err != nil {
		return fmt.Errorf("迁移Article表失败: %v", err)
	}

	// 插入测试数据
	log.Println("开始插入测试数据")
	err = insertTestData()
	if err != nil {
		return fmt.Errorf("插入测试数据失败: %v", err)
	}

	log.Println("数据库初始化完成")
	return nil
}

// 插入测试数据
func insertTestData() error {
	session := dbEngine.NewSession()

	// 插入用户数据
	for i := 1; i <= 10; i++ {
		user := &User{
			Username: fmt.Sprintf("user%d", i),
			Password: fmt.Sprintf("pass%d", i),
			Age:      20 + i,
			Created:  time.Now().Format("2006-01-02 15:04:05"),
		}

		// 先设置Model以提供Schema
		session = session.Model(user)
		_, err := session.Insert(user)
		if err != nil {
			return err
		}
	}

	// 插入文章数据
	for i := 1; i <= 20; i++ {
		article := &Article{
			Title:     fmt.Sprintf("Article Title %d", i),
			Content:   fmt.Sprintf("This is the content of article %d", i),
			UserID:    int64((i % 10) + 1),
			CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
		}

		// 先设置Model以提供Schema
		session = session.Model(article)
		_, err := session.Insert(article)
		if err != nil {
			return err
		}
	}

	return nil
}

// 获取所有表名
func getTableNames() ([]string, error) {
	log.Println("开始获取表名列表")
	session := dbEngine.NewSession()

	// 使用适当的SQL查询获取SQLite中的表名
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';"
	log.Printf("执行SQL: %s", query)

	rows, err := session.Raw(query).QueryRows()
	if err != nil {
		log.Printf("查询表名失败: %v", err)
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Printf("扫描表名失败: %v", err)
			return nil, err
		}
		tables = append(tables, tableName)
		log.Printf("找到表: %s", tableName)
	}

	log.Printf("总共找到 %d 个表", len(tables))
	return tables, nil
}

// 获取表结构
func getTableStructure(tableName string) ([]map[string]interface{}, error) {
	session := dbEngine.NewSession()

	// 使用PRAGMA获取表结构信息
	rows, err := session.Raw(fmt.Sprintf("PRAGMA table_info(%s);", tableName)).QueryRows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var cid int
		var name, type_name string
		var notnull, pk int
		var dflt_value interface{}

		if err := rows.Scan(&cid, &name, &type_name, &notnull, &dflt_value, &pk); err != nil {
			return nil, err
		}

		column := map[string]interface{}{
			"cid":        cid,
			"name":       name,
			"type":       type_name,
			"notnull":    notnull == 1,
			"default":    dflt_value,
			"primaryKey": pk == 1,
		}

		columns = append(columns, column)
	}

	return columns, nil
}

// 获取表数据
func getTableData(tableName string, limit, offset int) ([]map[string]interface{}, []string, error) {
	session := dbEngine.NewSession()

	// 查询表结构以获取列名
	structRows, err := session.Raw(fmt.Sprintf("PRAGMA table_info(%s);", tableName)).QueryRows()
	if err != nil {
		return nil, nil, err
	}
	defer structRows.Close()

	var columnNames []string
	for structRows.Next() {
		var cid int
		var name, type_name string
		var notnull, pk int
		var dflt_value interface{}

		if err := structRows.Scan(&cid, &name, &type_name, &notnull, &dflt_value, &pk); err != nil {
			return nil, nil, err
		}

		columnNames = append(columnNames, name)
	}

	// 查询表数据
	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d;", tableName, limit, offset)
	rows, err := session.Raw(query).QueryRows()
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var data []map[string]interface{}
	for rows.Next() {
		// 为每一行创建一个动态的值数组
		values := make([]interface{}, len(columnNames))
		valuePtrs := make([]interface{}, len(columnNames))

		for i := range columnNames {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, err
		}

		// 将扫描值转换为map
		rowData := make(map[string]interface{})
		for i, col := range columnNames {
			val := values[i]
			rowData[col] = val
		}

		data = append(data, rowData)
	}

	return data, columnNames, nil
}

// 执行SQL查询并返回结果（适用于SELECT语句）
func executeQuerySQL(query string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// 创建数据库会话
	db := dbEngine.NewSession()
	defer db.Close()

	// 执行查询
	log.Printf("执行SQL查询: %s", query)
	rows, err := db.Raw(query).QueryRows()
	if err != nil {
		log.Printf("执行查询失败: %v", err)
		return nil, fmt.Errorf("执行查询失败: %v", err)
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		log.Printf("获取列名失败: %v", err)
		return nil, fmt.Errorf("获取列名失败: %v", err)
	}
	log.Printf("查询结果包含列: %v", columns)

	// 准备接收查询结果的变量
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// 遍历结果集
	for rows.Next() {
		err = rows.Scan(valuePtrs...)
		if err != nil {
			log.Printf("扫描行数据失败: %v", err)
			return nil, fmt.Errorf("扫描行数据失败: %v", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]

			// 根据不同类型进行处理
			if val == nil {
				row[col] = nil
			} else {
				switch v := val.(type) {
				case []byte:
					// 字节数组转换为字符串
					row[col] = string(v)
				case int64, float64, bool, string, time.Time:
					// 这些类型可以直接使用
					row[col] = v
				default:
					// 其他类型尝试转为字符串
					row[col] = fmt.Sprintf("%v", v)
				}
			}
		}

		results = append(results, row)
	}

	if err = rows.Err(); err != nil {
		log.Printf("读取结果集时发生错误: %v", err)
		return nil, fmt.Errorf("读取结果集时发生错误: %v", err)
	}

	log.Printf("查询返回 %d 行结果", len(results))
	return results, nil
}

// 执行非查询SQL语句并返回影响的行数
func executeSQL(query string) (int64, error) {
	// 创建数据库会话
	db := dbEngine.NewSession()
	defer db.Close()

	// 执行语句
	result, err := db.Raw(query).Exec()
	if err != nil {
		return 0, fmt.Errorf("执行SQL失败: %v", err)
	}

	// 获取影响的行数
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %v", err)
	}

	return rowsAffected, nil
}

// 创建新表
func createTable(tableName string, columns []map[string]string) error {
	// 构建创建表的SQL
	var columnDefs []string

	log.Printf("创建表 %s，列定义: %v", tableName, columns)

	for _, col := range columns {
		columnName := col["name"]
		columnType := col["type"]

		log.Printf("处理列: %s, 类型: %s", columnName, columnType)

		// 构建列定义
		var constraints []string

		if col["primaryKey"] == "true" {
			constraints = append(constraints, "PRIMARY KEY")
			log.Printf("列 %s 设为主键", columnName)
		}

		if col["autoIncrement"] == "true" {
			constraints = append(constraints, "AUTOINCREMENT")
			log.Printf("列 %s 设为自增", columnName)
		}

		if col["notNull"] == "true" {
			constraints = append(constraints, "NOT NULL")
			log.Printf("列 %s 设为非空", columnName)
		}

		if col["unique"] == "true" {
			constraints = append(constraints, "UNIQUE")
			log.Printf("列 %s 设为唯一", columnName)
		}

		definition := fmt.Sprintf("%s %s %s", columnName, columnType, strings.Join(constraints, " "))
		columnDefs = append(columnDefs, strings.TrimSpace(definition))
		log.Printf("最终列定义: %s", strings.TrimSpace(definition))
	}

	// 创建表的SQL
	createSQL := fmt.Sprintf("CREATE TABLE %s (%s);", tableName, strings.Join(columnDefs, ", "))
	log.Printf("执行SQL: %s", createSQL)

	// 执行创建表操作
	session := dbEngine.NewSession()
	_, err := session.Raw(createSQL).Exec()
	if err != nil {
		log.Printf("创建表失败: %v", err)
		return err
	}

	log.Printf("成功创建表 %s", tableName)
	return nil
}

// 获取map的所有键
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func main() {
	// 初始化数据库引擎
	if err := initEngine(); err != nil {
		log.Fatalf("初始化数据库引擎失败: %v", err)
	}

	// 创建Gin路由
	r := gin.Default()

	// 设置静态文件服务
	r.Static("/static", "./static")

	// 设置HTML模板
	tmpl := createTemplate()
	// 尝试验证模板是否有"content"模板
	log.Println("验证模板是否正确解析...")
	for _, name := range []string{"layout.html", "index.html", "table.html", "structure.html", "sql.html", "create_table.html", "error.html"} {
		if t := tmpl.Lookup(name); t == nil {
			log.Printf("警告: 没有找到模板 %s", name)
		} else {
			log.Printf("找到模板 %s", name)
		}
	}

	// 验证content定义是否存在
	for _, name := range []string{"index.html", "table.html", "structure.html", "sql.html", "create_table.html", "error.html"} {
		if t := tmpl.Lookup(fmt.Sprintf("content.%s", name)); t == nil {
			log.Printf("警告: 没有找到content定义 %s", name)
		} else {
			log.Printf("找到content定义 %s", name)
		}
	}

	r.SetHTMLTemplate(tmpl)

	// 主页 - 显示所有表
	r.GET("/", func(c *gin.Context) {
		log.Println("访问主页路由")
		tables, err := getTableNames()
		if err != nil {
			log.Printf("获取表列表失败: %v", err)
			html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>QSyORM 数据库管理器</title>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<style>
					body {
						font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
						line-height: 1.5;
						color: #333;
						max-width: 1200px;
						margin: 0 auto;
						padding: 20px;
					}
					h1, h2 {
						color: #333;
					}
					.error {
						color: #721c24;
						background-color: #f8d7da;
						border: 1px solid #f5c6cb;
						padding: 10px;
						border-radius: 4px;
						margin-bottom: 20px;
					}
					.nav {
						display: flex;
						margin-bottom: 20px;
						gap: 10px;
						align-items: center;
					}
					.nav a {
						padding: 8px 16px;
						background-color: #f1f1f1;
						color: #333;
						text-decoration: none;
						border-radius: 4px;
					}
					.nav a:hover {
						background-color: #ddd;
					}
				</style>
			</head>
			<body>
				<h1>QSyORM 数据库管理器</h1>
				<div class="nav">
					<a href="/">表列表</a>
					<a href="/sql">执行SQL</a>
					<a href="/create">创建表</a>
				</div>
				
				<div class="error">获取表列表失败: %v</div>
			</body>
			</html>
			`, err)
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(html))
			return
		}

		log.Printf("获取到表列表: %v", tables)

		// 生成表格内容
		var tableRows string
		if len(tables) == 0 {
			tableRows = `<tr><td colspan="2">没有找到表</td></tr>`
		} else {
			for _, tableName := range tables {
				tableRows += fmt.Sprintf(`<tr>
					<td>%s</td>
					<td>
						<a href="/table/%s">查看数据</a> | 
						<a href="/structure/%s">表结构</a>
					</td>
				</tr>`, tableName, tableName, tableName)
			}
		}

		html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>QSyORM 数据库管理器</title>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>
				body {
					font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
					line-height: 1.5;
					color: #333;
					max-width: 1200px;
					margin: 0 auto;
					padding: 20px;
				}
				h1, h2 {
					color: #333;
				}
				table {
					width: 100%%;
					border-collapse: collapse;
					margin-bottom: 20px;
				}
				th, td {
					border: 1px solid #ddd;
					padding: 8px 12px;
					text-align: left;
				}
				th {
					background-color: #f5f5f5;
					font-weight: bold;
				}
				tr:nth-child(even) {
					background-color: #f9f9f9;
				}
				tr:hover {
					background-color: #f1f1f1;
				}
				.nav {
					display: flex;
					margin-bottom: 20px;
					gap: 10px;
					align-items: center;
				}
				.nav a {
					padding: 8px 16px;
					background-color: #f1f1f1;
					color: #333;
					text-decoration: none;
					border-radius: 4px;
				}
				.nav a:hover {
					background-color: #ddd;
				}
				.logo {
					max-width: 300px;
					height: auto;
					display: block;
					margin-bottom: 20px;
				}
			</style>
		</head>
		<body>
			<h1>QSyORM 数据库管理器</h1>
			<img src="/static/image.png" alt="QSyORM Logo" class="logo">
			<div class="nav">
				<a href="/">表列表</a>
				<a href="/sql">执行SQL</a>
				<a href="/create">创建表</a>
			</div>
			
			<h2>数据库表</h2>
			
			<table>
				<thead>
					<tr>
						<th>表名</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					%s
				</tbody>
			</table>
		</body>
		</html>
		`, tableRows)

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})

	// 查看表数据
	r.GET("/table/:name", func(c *gin.Context) {
		tableName := c.Param("name")

		// 获取分页参数
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		if page < 1 {
			page = 1
		}

		limit := 10
		offset := (page - 1) * limit

		// 获取表数据
		data, columns, err := getTableData(tableName, limit, offset)
		if err != nil {
			html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>QSyORM 数据库管理器</title>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<style>
					body {
						font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
						line-height: 1.5;
						color: #333;
						max-width: 1200px;
						margin: 0 auto;
						padding: 20px;
					}
					h1, h2 {
						color: #333;
					}
					.error {
						color: #721c24;
						background-color: #f8d7da;
						border: 1px solid #f5c6cb;
						padding: 10px;
						border-radius: 4px;
						margin-bottom: 20px;
					}
					.nav {
						display: flex;
						margin-bottom: 20px;
						gap: 10px;
						align-items: center;
					}
					.nav a {
						padding: 8px 16px;
						background-color: #f1f1f1;
						color: #333;
						text-decoration: none;
						border-radius: 4px;
					}
					.nav a:hover {
						background-color: #ddd;
					}
				</style>
			</head>
			<body>
				<h1>QSyORM 数据库管理器</h1>
				<div class="nav">
					<a href="/">表列表</a>
					<a href="/sql">执行SQL</a>
					<a href="/create">创建表</a>
				</div>
				
				<div class="error">获取表数据失败: %v</div>
			</body>
			</html>
			`, err)
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(html))
			return
		}

		// 获取表的总行数
		session := dbEngine.NewSession()
		countRow := session.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).QueryRow()

		var total int
		err = countRow.Scan(&total)
		if err != nil {
			html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>QSyORM 数据库管理器</title>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<style>
					body {
						font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
						line-height: 1.5;
						color: #333;
						max-width: 1200px;
						margin: 0 auto;
						padding: 20px;
					}
					h1, h2 {
						color: #333;
					}
					.error {
						color: #721c24;
						background-color: #f8d7da;
						border: 1px solid #f5c6cb;
						padding: 10px;
						border-radius: 4px;
						margin-bottom: 20px;
					}
					.nav {
						display: flex;
						margin-bottom: 20px;
						gap: 10px;
						align-items: center;
					}
					.nav a {
						padding: 8px 16px;
						background-color: #f1f1f1;
						color: #333;
						text-decoration: none;
						border-radius: 4px;
					}
					.nav a:hover {
						background-color: #ddd;
					}
				</style>
			</head>
			<body>
				<h1>QSyORM 数据库管理器</h1>
				<div class="nav">
					<a href="/">表列表</a>
					<a href="/sql">执行SQL</a>
					<a href="/create">创建表</a>
				</div>
				
				<div class="error">获取表行数失败: %v</div>
			</body>
			</html>
			`, err)
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(html))
			return
		}

		// 计算总页数
		totalPages := (total + limit - 1) / limit

		// 构建表头
		var tableHead string
		for _, col := range columns {
			tableHead += fmt.Sprintf("<th>%s</th>", col)
		}

		// 构建表格主体
		var tableBody string
		if len(data) == 0 {
			tableBody = fmt.Sprintf("<tr><td colspan=\"%d\">没有数据</td></tr>", len(columns))
		} else {
			for _, row := range data {
				tableBody += "<tr>"
				for _, col := range columns {
					tableBody += fmt.Sprintf("<td>%v</td>", row[col])
				}
				tableBody += "</tr>"
			}
		}

		// 生成分页控件
		var pagination string
		if totalPages > 1 {
			pagination = "<ul class=\"pagination\">"
			if page > 1 {
				pagination += fmt.Sprintf("<li><a href=\"/table/%s?page=%d\">&laquo; 上一页</a></li>", tableName, page-1)
			}

			for i := 1; i <= totalPages; i++ {
				activeClass := ""
				if i == page {
					activeClass = " class=\"active\""
				}
				pagination += fmt.Sprintf("<li%s><a href=\"/table/%s?page=%d\">%d</a></li>", activeClass, tableName, i, i)
			}

			if page < totalPages {
				pagination += fmt.Sprintf("<li><a href=\"/table/%s?page=%d\">下一页 &raquo;</a></li>", tableName, page+1)
			}
			pagination += "</ul>"
		}

		html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>QSyORM 数据库管理器</title>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>
				body {
					font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
					line-height: 1.5;
					color: #333;
					max-width: 1200px;
					margin: 0 auto;
					padding: 20px;
				}
				h1, h2 {
					color: #333;
				}
				table {
					width: 100%%;
					border-collapse: collapse;
					margin-bottom: 20px;
				}
				th, td {
					border: 1px solid #ddd;
					padding: 8px 12px;
					text-align: left;
				}
				th {
					background-color: #f5f5f5;
					font-weight: bold;
				}
				tr:nth-child(even) {
					background-color: #f9f9f9;
				}
				tr:hover {
					background-color: #f1f1f1;
				}
				.nav {
					display: flex;
					margin-bottom: 20px;
					gap: 10px;
					align-items: center;
				}
				.nav a {
					padding: 8px 16px;
					background-color: #f1f1f1;
					color: #333;
					text-decoration: none;
					border-radius: 4px;
				}
				.nav a:hover {
					background-color: #ddd;
				}
				.pagination {
					display: flex;
					list-style: none;
					padding: 0;
					margin: 20px 0;
				}
				.pagination li {
					margin-right: 5px;
				}
				.pagination a {
					padding: 5px 10px;
					border: 1px solid #ddd;
					text-decoration: none;
					color: #333;
				}
				.pagination .active a {
					background-color: #007bff;
					color: white;
					border-color: #007bff;
				}
			</style>
		</head>
		<body>
			<h1>QSyORM 数据库管理器</h1>
			<div class="nav">
				<a href="/">表列表</a>
				<a href="/sql">执行SQL</a>
				<a href="/create">创建表</a>
			</div>
			
			<h2>表数据: %s</h2>
			
			<div class="nav">
				<a href="/structure/%s">查看表结构</a>
			</div>
			
			<table>
				<thead>
					<tr>
						%s
					</tr>
				</thead>
				<tbody>
					%s
				</tbody>
			</table>
			
			%s
			
			<p>共 %d 条记录，共 %d 页</p>
		</body>
		</html>
		`, tableName, tableName, tableHead, tableBody, pagination, total, totalPages)

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})

	// 查看表结构
	r.GET("/structure/:name", func(c *gin.Context) {
		tableName := c.Param("name")

		session := dbEngine.NewSession()
		rows, err := session.Raw(fmt.Sprintf("PRAGMA table_info(%s)", tableName)).QueryRows()
		if err != nil {
			html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>QSyORM 数据库管理器</title>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<style>
					body {
						font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
						line-height: 1.5;
						color: #333;
						max-width: 1200px;
						margin: 0 auto;
						padding: 20px;
					}
					h1, h2 {
						color: #333;
					}
					.error {
						color: #721c24;
						background-color: #f8d7da;
						border: 1px solid #f5c6cb;
						padding: 10px;
						border-radius: 4px;
						margin-bottom: 20px;
					}
					.nav {
						display: flex;
						margin-bottom: 20px;
						gap: 10px;
						align-items: center;
					}
					.nav a {
						padding: 8px 16px;
						background-color: #f1f1f1;
						color: #333;
						text-decoration: none;
						border-radius: 4px;
					}
					.nav a:hover {
						background-color: #ddd;
					}
				</style>
			</head>
			<body>
				<h1>QSyORM 数据库管理器</h1>
				<div class="nav">
					<a href="/">表列表</a>
					<a href="/sql">执行SQL</a>
					<a href="/create">创建表</a>
				</div>
				
				<div class="error">获取表结构失败: %v</div>
			</body>
			</html>
			`, err)
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(html))
			return
		}

		defer rows.Close()

		// 获取列名
		columns, err := rows.Columns()
		if err != nil {
			html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>QSyORM 数据库管理器</title>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<style>
					body {
						font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
						line-height: 1.5;
						color: #333;
						max-width: 1200px;
						margin: 0 auto;
						padding: 20px;
					}
					h1, h2 {
						color: #333;
					}
					.error {
						color: #721c24;
						background-color: #f8d7da;
						border: 1px solid #f5c6cb;
						padding: 10px;
						border-radius: 4px;
						margin-bottom: 20px;
					}
					.nav {
						display: flex;
						margin-bottom: 20px;
						gap: 10px;
						align-items: center;
					}
					.nav a {
						padding: 8px 16px;
						background-color: #f1f1f1;
						color: #333;
						text-decoration: none;
						border-radius: 4px;
					}
					.nav a:hover {
						background-color: #ddd;
					}
				</style>
			</head>
			<body>
				<h1>QSyORM 数据库管理器</h1>
				<div class="nav">
					<a href="/">表列表</a>
					<a href="/sql">执行SQL</a>
					<a href="/create">创建表</a>
				</div>
				
				<div class="error">获取列名失败: %v</div>
			</body>
			</html>
			`, err)
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(html))
			return
		}

		var structure []map[string]interface{}

		// 准备存储数据的切片
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		// 读取数据行
		for rows.Next() {
			// 为每一列创建一个指针
			for i := range columns {
				valuePtrs[i] = &values[i]
			}

			// 扫描当前行数据到指针
			if err := rows.Scan(valuePtrs...); err != nil {
				html := fmt.Sprintf(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>QSyORM 数据库管理器</title>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<style>
						body {
							font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
							line-height: 1.5;
							color: #333;
							max-width: 1200px;
							margin: 0 auto;
							padding: 20px;
						}
						h1, h2 {
							color: #333;
						}
						.error {
							color: #721c24;
							background-color: #f8d7da;
							border: 1px solid #f5c6cb;
							padding: 10px;
							border-radius: 4px;
							margin-bottom: 20px;
						}
						.nav {
							display: flex;
							margin-bottom: 20px;
							gap: 10px;
							align-items: center;
						}
						.nav a {
							padding: 8px 16px;
							background-color: #f1f1f1;
							color: #333;
							text-decoration: none;
							border-radius: 4px;
						}
						.nav a:hover {
							background-color: #ddd;
						}
					</style>
				</head>
				<body>
					<h1>QSyORM 数据库管理器</h1>
					<div class="nav">
						<a href="/">表列表</a>
						<a href="/sql">执行SQL</a>
						<a href="/create">创建表</a>
					</div>
					
					<div class="error">读取行数据失败: %v</div>
				</body>
				</html>
				`, err)
				c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(html))
				return
			}

			// 将数据转换为map
			row := make(map[string]interface{})
			for i, colName := range columns {
				val := values[i]
				row[colName] = val
			}

			structure = append(structure, row)
		}

		if err = rows.Err(); err != nil {
			html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>QSyORM 数据库管理器</title>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<style>
					body {
						font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
						line-height: 1.5;
						color: #333;
						max-width: 1200px;
						margin: 0 auto;
						padding: 20px;
					}
					h1, h2 {
						color: #333;
					}
					.error {
						color: #721c24;
						background-color: #f8d7da;
						border: 1px solid #f5c6cb;
						padding: 10px;
						border-radius: 4px;
						margin-bottom: 20px;
					}
					.nav {
						display: flex;
						margin-bottom: 20px;
						gap: 10px;
						align-items: center;
					}
					.nav a {
						padding: 8px 16px;
						background-color: #f1f1f1;
						color: #333;
						text-decoration: none;
						border-radius: 4px;
					}
					.nav a:hover {
						background-color: #ddd;
					}
				</style>
			</head>
			<body>
				<h1>QSyORM 数据库管理器</h1>
				<div class="nav">
					<a href="/">表列表</a>
					<a href="/sql">执行SQL</a>
					<a href="/create">创建表</a>
				</div>
				
				<div class="error">处理表结构数据失败: %v</div>
			</body>
			</html>
			`, err)
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(html))
			return
		}

		// 构建表结构表格
		var tableBody string
		for _, column := range structure {
			cid := column["cid"]
			name := column["name"]
			typ := column["type"]
			notnull := column["notnull"]
			dfltValue := column["default"]
			pk := column["primaryKey"]

			var notNullStr, pkStr string

			// 类型转换
			switch n := notnull.(type) {
			case int64:
				if n == 1 {
					notNullStr = "是"
				} else {
					notNullStr = "否"
				}
			case []byte:
				if string(n) == "1" {
					notNullStr = "是"
				} else {
					notNullStr = "否"
				}
			default:
				notNullStr = "否"
			}

			switch p := pk.(type) {
			case int64:
				if p == 1 {
					pkStr = "是"
				} else {
					pkStr = "否"
				}
			case []byte:
				if string(p) == "1" {
					pkStr = "是"
				} else {
					pkStr = "否"
				}
			default:
				pkStr = "否"
			}

			dfltValueStr := ""
			if dfltValue != nil {
				dfltValueStr = fmt.Sprintf("%v", dfltValue)
			}

			tableBody += fmt.Sprintf("<tr><td>%v</td><td>%v</td><td>%v</td><td>%s</td><td>%s</td><td>%s</td></tr>",
				cid, name, typ, notNullStr, dfltValueStr, pkStr)
		}

		html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>QSyORM 数据库管理器</title>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>
				body {
					font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
					line-height: 1.5;
					color: #333;
					max-width: 1200px;
					margin: 0 auto;
					padding: 20px;
				}
				h1, h2 {
					color: #333;
				}
				table {
					width: 100%%;
					border-collapse: collapse;
					margin-bottom: 20px;
				}
				th, td {
					border: 1px solid #ddd;
					padding: 8px 12px;
					text-align: left;
				}
				th {
					background-color: #f5f5f5;
					font-weight: bold;
				}
				tr:nth-child(even) {
					background-color: #f9f9f9;
				}
				tr:hover {
					background-color: #f1f1f1;
				}
				.nav {
					display: flex;
					margin-bottom: 20px;
					gap: 10px;
					align-items: center;
				}
				.nav a {
					padding: 8px 16px;
					background-color: #f1f1f1;
					color: #333;
					text-decoration: none;
					border-radius: 4px;
				}
				.nav a:hover {
					background-color: #ddd;
				}
			</style>
		</head>
		<body>
			<h1>QSyORM 数据库管理器</h1>
			<div class="nav">
				<a href="/">表列表</a>
				<a href="/sql">执行SQL</a>
				<a href="/create">创建表</a>
			</div>
			
			<h2>表结构: %s</h2>
			
			<div class="nav">
				<a href="/table/%s">查看表数据</a>
			</div>
			
			<table>
				<thead>
					<tr>
						<th>ID</th>
						<th>字段名</th>
						<th>类型</th>
						<th>非空</th>
						<th>默认值</th>
						<th>主键</th>
					</tr>
				</thead>
				<tbody>
					%s
				</tbody>
			</table>
		</body>
		</html>
		`, tableName, tableName, tableBody)

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})

	// GET handler for SQL page
	r.GET("/sql", func(c *gin.Context) {
		// 注意这里使用的是"sql.html"，它会自动包含layout和content.sql.html
		c.HTML(http.StatusOK, "sql.html", gin.H{
			"sqlText":  "",
			"executed": false,
		})
	})

	// GET handler for Create Table page
	r.GET("/create", func(c *gin.Context) {
		// 使用"create_table.html"，它会自动包含layout和content.create_table.html
		c.HTML(http.StatusOK, "create_table.html", gin.H{})
	})

	// 执行SQL查询
	r.POST("/sql", func(c *gin.Context) {
		sqlText := c.PostForm("sql")
		if sqlText == "" {
			c.Redirect(http.StatusFound, "/")
			return
		}

		// 判断SQL类型：是SELECT查询还是非查询语句
		sqlType := "query"
		sqlLower := strings.ToLower(strings.TrimSpace(sqlText))
		if strings.HasPrefix(sqlLower, "insert") ||
			strings.HasPrefix(sqlLower, "update") ||
			strings.HasPrefix(sqlLower, "delete") ||
			strings.HasPrefix(sqlLower, "create") ||
			strings.HasPrefix(sqlLower, "drop") ||
			strings.HasPrefix(sqlLower, "alter") {
			sqlType = "execute"
		}

		if sqlType == "query" {
			// 处理查询SQL（SELECT）
			results, err := executeQuerySQL(sqlText)
			if err != nil {
				// 直接渲染错误页面
				errorHTML := fmt.Sprintf(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>QSyORM 数据库管理器</title>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<style>
						body {
							font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
							line-height: 1.5;
							color: #333;
							max-width: 1200px;
							margin: 0 auto;
							padding: 20px;
						}
						h1, h2, h3 {
							color: #333;
						}
						.error {
							color: #721c24;
							background-color: #f8d7da;
							border: 1px solid #f5c6cb;
							padding: 10px;
							border-radius: 4px;
							margin-bottom: 20px;
						}
						.nav {
							display: flex;
							margin-bottom: 20px;
							gap: 10px;
							align-items: center;
						}
						.nav a {
							padding: 8px 16px;
							background-color: #f1f1f1;
							color: #333;
							text-decoration: none;
							border-radius: 4px;
						}
						.nav a:hover {
							background-color: #ddd;
						}
					</style>
				</head>
				<body>
					<h1>QSyORM 数据库管理器</h1>
					<div class="nav">
						<a href="/">表列表</a>
						<a href="/sql">执行SQL</a>
						<a href="/create">创建表</a>
					</div>
					
					<h2>执行SQL</h2>
					<div class="error">%v</div>
					<div>
						<a href="javascript:history.back()">返回上一页</a>
					</div>
				</body>
				</html>
				`, err)
				c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(errorHTML))
				return
			}

			if len(results) == 0 {
				// 如果查询结果为空
				emptyHTML := fmt.Sprintf(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>QSyORM 数据库管理器</title>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<style>
						body {
							font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
							line-height: 1.5;
							color: #333;
							max-width: 1200px;
							margin: 0 auto;
							padding: 20px;
						}
						h1, h2, h3 {
							color: #333;
						}
						.alert {
							padding: 10px;
							border-radius: 4px;
							margin-bottom: 20px;
						}
						.success {
							color: #155724;
							background-color: #d4edda;
							border: 1px solid #c3e6cb;
						}
						.nav {
							display: flex;
							margin-bottom: 20px;
							gap: 10px;
							align-items: center;
						}
						.nav a {
							padding: 8px 16px;
							background-color: #f1f1f1;
							color: #333;
							text-decoration: none;
							border-radius: 4px;
						}
						.nav a:hover {
							background-color: #ddd;
						}
						pre {
							background-color: #f8f9fa;
							padding: 10px;
							border-radius: 4px;
							border: 1px solid #e9ecef;
							margin-bottom: 20px;
							overflow-x: auto;
						}
						textarea {
							width: 100%%;
							height: 150px;
							padding: 10px;
							margin-bottom: 10px;
							font-family: monospace;
							border: 1px solid #ced4da;
							border-radius: 4px;
						}
						button {
							padding: 8px 16px;
							background-color: #4CAF50;
							color: white;
							border: none;
							border-radius: 4px;
							cursor: pointer;
						}
						button:hover {
							background-color: #45a049;
						}
					</style>
				</head>
				<body>
					<h1>QSyORM 数据库管理器</h1>
					<div class="nav">
						<a href="/">表列表</a>
						<a href="/sql">执行SQL</a>
						<a href="/create">创建表</a>
					</div>
					
					<h2>执行SQL</h2>
					
					<form method="post" action="/sql">
						<textarea name="sql" placeholder="输入SQL查询...">%s</textarea>
						<button type="submit">执行</button>
					</form>
					
					<div class="alert success">
						查询执行成功，返回0行结果。
					</div>
					<pre>%s</pre>
				</body>
				</html>
				`, sqlText, sqlText)
				c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(emptyHTML))
				return
			}

			// 获取列名（字段名）
			columns := getMapKeys(results[0])
			log.Printf("查询结果列名: %v", columns)

			// 构建表头HTML
			var tableHead string
			for _, col := range columns {
				tableHead += fmt.Sprintf("<th>%s</th>", col)
			}

			// 构建表体HTML
			var tableRows string
			for _, row := range results {
				tableRows += "<tr>"
				for _, col := range columns {
					// 确保安全访问map
					val, exists := row[col]
					if !exists {
						tableRows += "<td></td>"
					} else if val == nil {
						tableRows += "<td>NULL</td>"
					} else {
						tableRows += fmt.Sprintf("<td>%v</td>", val)
					}
				}
				tableRows += "</tr>"
			}

			// 构建完整HTML页面
			resultHTML := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>QSyORM 数据库管理器</title>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<style>
					body {
						font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
						line-height: 1.5;
						color: #333;
						max-width: 1200px;
						margin: 0 auto;
						padding: 20px;
					}
					h1, h2, h3 {
						color: #333;
					}
					.nav {
						display: flex;
						margin-bottom: 20px;
						gap: 10px;
						align-items: center;
					}
					.nav a {
						padding: 8px 16px;
						background-color: #f1f1f1;
						color: #333;
						text-decoration: none;
						border-radius: 4px;
					}
					.nav a:hover {
						background-color: #ddd;
					}
					textarea {
						width: 100%%;
						height: 150px;
						padding: 10px;
						margin-bottom: 10px;
						font-family: monospace;
						border: 1px solid #ced4da;
						border-radius: 4px;
					}
					button {
						padding: 8px 16px;
						background-color: #4CAF50;
						color: white;
						border: none;
						border-radius: 4px;
						cursor: pointer;
					}
					button:hover {
						background-color: #45a049;
					}
					table {
						width: 100%%;
						border-collapse: collapse;
						margin-bottom: 20px;
					}
					th, td {
						border: 1px solid #ddd;
						padding: 8px 12px;
						text-align: left;
					}
					th {
						background-color: #f5f5f5;
						font-weight: bold;
					}
					tr:nth-child(even) {
						background-color: #f9f9f9;
					}
					tr:hover {
						background-color: #f1f1f1;
					}
				</style>
			</head>
			<body>
				<h1>QSyORM 数据库管理器</h1>
				<div class="nav">
					<a href="/">表列表</a>
					<a href="/sql">执行SQL</a>
					<a href="/create">创建表</a>
				</div>
				
				<h2>执行SQL</h2>
				
				<form method="post" action="/sql">
					<textarea name="sql" placeholder="输入SQL查询...">%s</textarea>
					<button type="submit">执行</button>
				</form>
				
				<h3>结果</h3>
				<table>
					<thead>
						<tr>
							%s
						</tr>
					</thead>
					<tbody>
						%s
					</tbody>
				</table>
			</body>
			</html>
			`, sqlText, tableHead, tableRows)

			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(resultHTML))
		} else {
			// 处理非查询SQL（INSERT/UPDATE/DELETE）
			rowsAffected, err := executeSQL(sqlText)
			if err != nil {
				// 直接渲染错误页面
				errorHTML := fmt.Sprintf(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>QSyORM 数据库管理器</title>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<style>
						body {
							font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
							line-height: 1.5;
							color: #333;
							max-width: 1200px;
							margin: 0 auto;
							padding: 20px;
						}
						h1, h2 {
							color: #333;
						}
						.error {
							color: #721c24;
							background-color: #f8d7da;
							border: 1px solid #f5c6cb;
							padding: 10px;
							border-radius: 4px;
							margin-bottom: 20px;
						}
						.nav {
							display: flex;
							margin-bottom: 20px;
							gap: 10px;
							align-items: center;
						}
						.nav a {
							padding: 8px 16px;
							background-color: #f1f1f1;
							color: #333;
							text-decoration: none;
							border-radius: 4px;
						}
						.nav a:hover {
							background-color: #ddd;
						}
					</style>
				</head>
				<body>
					<h1>QSyORM 数据库管理器</h1>
					<div class="nav">
						<a href="/">表列表</a>
						<a href="/sql">执行SQL</a>
						<a href="/create">创建表</a>
					</div>
					
					<h2>错误</h2>
					<div class="error">%v</div>
					<div>
						<a href="javascript:history.back()">返回上一页</a>
					</div>
				</body>
				</html>
				`, err)
				c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(errorHTML))
				return
			}

			// 渲染成功页面
			successHTML := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>QSyORM 数据库管理器</title>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<style>
					body {
						font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
						line-height: 1.5;
						color: #333;
						max-width: 1200px;
						margin: 0 auto;
						padding: 20px;
					}
					h1, h2, h3 {
						color: #333;
					}
					.alert {
						padding: 10px;
						border-radius: 4px;
						margin-bottom: 20px;
					}
					.success {
						color: #155724;
						background-color: #d4edda;
						border: 1px solid #c3e6cb;
					}
					.nav {
						display: flex;
						margin-bottom: 20px;
						gap: 10px;
						align-items: center;
					}
					.nav a {
						padding: 8px 16px;
						background-color: #f1f1f1;
						color: #333;
						text-decoration: none;
						border-radius: 4px;
					}
					.nav a:hover {
						background-color: #ddd;
					}
					pre {
						background-color: #f8f9fa;
						padding: 10px;
						border-radius: 4px;
						border: 1px solid #e9ecef;
						margin-bottom: 20px;
						overflow-x: auto;
					}
					textarea {
						width: 100%%;
						height: 150px;
						padding: 10px;
						margin-bottom: 10px;
						font-family: monospace;
						border: 1px solid #ced4da;
						border-radius: 4px;
					}
					button {
						padding: 8px 16px;
						background-color: #4CAF50;
						color: white;
						border: none;
						border-radius: 4px;
						cursor: pointer;
					}
					button:hover {
						background-color: #45a049;
					}
				</style>
			</head>
			<body>
				<h1>QSyORM 数据库管理器</h1>
				<div class="nav">
					<a href="/">表列表</a>
					<a href="/sql">执行SQL</a>
					<a href="/create">创建表</a>
				</div>
				
				<h2>执行SQL</h2>
				
				<form method="post" action="/sql">
					<textarea name="sql" placeholder="输入SQL查询...">%s</textarea>
					<button type="submit">执行</button>
				</form>
				
				<div class="alert success">
					SQL执行成功，影响了 %d 行记录。
				</div>
				<pre>%s</pre>
			</body>
			</html>
			`, sqlText, rowsAffected, sqlText)

			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(successHTML))
		}
	})

	r.Run()
}

func createTemplate() *template.Template {
	tmplStr := `
{{define "layout"}}
<!DOCTYPE html>
<html>
<head>
	<title>QSyORM 数据库管理器</title>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			line-height: 1.5;
			color: #333;
			max-width: 1200px;
			margin: 0 auto;
			padding: 20px;
		}
		h1, h2, h3 {
			color: #333;
		}
		.error {
			color: #721c24;
			background-color: #f8d7da;
			border: 1px solid #f5c6cb;
			padding: 10px;
			border-radius: 4px;
			margin-bottom: 20px;
		}
		.alert {
			padding: 10px;
			border-radius: 4px;
			margin-bottom: 20px;
		}
		.success {
			color: #155724;
			background-color: #d4edda;
			border: 1px solid #c3e6cb;
		}
		.nav {
			display: flex;
			margin-bottom: 20px;
			gap: 10px;
			align-items: center;
		}
		.nav a {
			padding: 8px 16px;
			background-color: #f1f1f1;
			color: #333;
			text-decoration: none;
			border-radius: 4px;
		}
		.nav a:hover {
			background-color: #ddd;
		}
		pre {
			background-color: #f8f9fa;
			padding: 10px;
			border-radius: 4px;
			border: 1px solid #e9ecef;
			margin-bottom: 20px;
			overflow-x: auto;
		}
		textarea {
			width: 100%;
			height: 150px;
			padding: 10px;
			margin-bottom: 10px;
			font-family: monospace;
			border: 1px solid #ced4da;
			border-radius: 4px;
		}
		button {
			padding: 8px 16px;
			background-color: #4CAF50;
			color: white;
			border: none;
			border-radius: 4px;
			cursor: pointer;
		}
		button:hover {
			background-color: #45a049;
		}
		table {
			width: 100%;
			border-collapse: collapse;
			margin-bottom: 20px;
		}
		th, td {
			border: 1px solid #ddd;
			padding: 8px 12px;
			text-align: left;
		}
		th {
			background-color: #f5f5f5;
			font-weight: bold;
		}
		tr:nth-child(even) {
			background-color: #f9f9f9;
		}
		tr:hover {
			background-color: #f1f1f1;
		}
		form {
			margin-bottom: 20px;
		}
		.column-form {
			margin-bottom: 15px;
			padding: 10px;
			border: 1px solid #ddd;
			border-radius: 4px;
			background-color: #f9f9f9;
		}
		label {
			display: inline-block;
			width: 80px;
			margin-bottom: 5px;
		}
		input[type="text"], select {
			width: 200px;
			padding: 5px;
			margin-bottom: 10px;
			border: 1px solid #ced4da;
			border-radius: 4px;
		}
		.checkbox-group {
			margin-top: 10px;
		}
		.checkbox-group label {
			width: auto;
			margin-right: 15px;
		}
	</style>
</head>
<body>
	<h1>QSyORM 数据库管理器</h1>
	<div class="nav">
		<a href="/">表列表</a>
		<a href="/sql">执行SQL</a>
		<a href="/create">创建表</a>
	</div>
{{end}}

{{define "sql.html"}}
{{template "layout" .}}
{{template "content.sql.html" .}}
{{end}}

{{define "content.sql.html"}}
    <h2>执行SQL</h2>
    {{if .error}}
        <div class="error">{{.error}}</div>
    {{end}}
    
    <form method="post" action="/sql">
        <textarea name="sql" placeholder="输入SQL查询...">{{.sqlText}}</textarea>
        <button type="submit">执行</button>
    </form>
    
    {{if .executed}}
        {{if .success}}
            <div class="alert success">
                {{.success}}
            </div>
            <pre>{{.sqlText}}</pre>
        {{else}}
            <h3>结果</h3>
            <table>
                <thead>
                    <tr>
                        {{range .columns}}
                        <th>{{.}}</th>
                        {{end}}
                    </tr>
                </thead>
                <tbody>
                    {{range .results}}
                    <tr>
                        {{range $column := $.columns}}
                        <td>{{index . $column}}</td>
                        {{end}}
                    </tr>
                    {{else}}
                    <tr>
                        <td colspan="{{len .columns}}">没有数据</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        {{end}}
    {{end}}
{{end}}

{{define "create_table.html"}}
{{template "layout" .}}
{{template "content.create_table.html" .}}
{{end}}

{{define "content.create_table.html"}}
    <h2>创建新表</h2>
    {{if .error}}
        <div class="error">{{.error}}</div>
    {{end}}
    
    <form id="createTableForm" method="post" action="/create">
        <div>
            <label for="tableName">表名:</label>
            <input type="text" id="tableName" name="tableName" required>
        </div>
        
        <h3>列定义</h3>
        <div id="columns"></div>
        
        <div class="add-column">
            <button type="button" id="addColumn">添加列</button>
        </div>
        
        <input type="hidden" id="columnCount" name="columnCount" value="0">
        <button type="submit">创建表</button>
    </form>
    
    <script>
        let columnCount = 0;
        
        function addColumn() {
            const columnsDiv = document.getElementById('columns');
            const columnDiv = document.createElement('div');
            columnDiv.className = 'column-form';
            
            columnDiv.innerHTML = '' +
                '<div>' +
                    '<label for="column[' + columnCount + '][name]">列名:</label>' +
                    '<input type="text" name="column[' + columnCount + '][name]" required>' +
                '</div>' +
                '<div>' +
                    '<label for="column[' + columnCount + '][type]">类型:</label>' +
                    '<select name="column[' + columnCount + '][type]" required>' +
                        '<option value="INTEGER">INTEGER</option>' +
                        '<option value="TEXT">TEXT</option>' +
                        '<option value="REAL">REAL</option>' +
                        '<option value="BLOB">BLOB</option>' +
                        '<option value="BOOLEAN">BOOLEAN</option>' +
                    '</select>' +
                '</div>' +
                '<div class="checkbox-group">' +
                    '<label>' +
                        '<input type="checkbox" name="column[' + columnCount + '][primaryKey]" value="true">' +
                        '主键' +
                    '</label>' +
                    '<label>' +
                        '<input type="checkbox" name="column[' + columnCount + '][autoIncrement]" value="true">' +
                        '自增' +
                    '</label>' +
                    '<label>' +
                        '<input type="checkbox" name="column[' + columnCount + '][notNull]" value="true">' +
                        '非空' +
                    '</label>' +
                    '<label>' +
                        '<input type="checkbox" name="column[' + columnCount + '][unique]" value="true">' +
                        '唯一' +
                    '</label>' +
                '</div>';
            
            columnsDiv.appendChild(columnDiv);
            columnCount++;
            document.getElementById('columnCount').value = columnCount;
        }
        
        document.addEventListener("DOMContentLoaded", function() {
            document.getElementById('addColumn').addEventListener('click', addColumn);
            
            // 初始添加一列
            addColumn();
            
            // 提交表单前验证
            document.getElementById('createTableForm').addEventListener('submit', function(e) {
                const tableName = document.getElementById('tableName').value;
                if (!tableName) {
                    alert('请输入表名');
                    e.preventDefault();
                    return false;
                }
                
                const columns = document.querySelectorAll('.column-form');
                if (columns.length === 0) {
                    alert('请至少添加一列');
                    e.preventDefault();
                    return false;
                }
                
                return true;
            });
        });
    </script>
{{end}}

{{define "error.html"}}
{{template "layout" .}}
{{end}}

{{define "content.error.html"}}
    <h2>错误</h2>
    <div class="error">{{.error}}</div>
    <div>
        <a href="javascript:history.back()">返回上一页</a>
    </div>
{{end}}
`

	// 创建一个新的模板，注册函数映射
	tmpl := template.New("")
	tmpl.Funcs(template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
		"seq": func(start, end int) []int {
			var result []int
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
			return result
		},
	})

	// 解析整个模板集合
	tmpl, err := tmpl.Parse(tmplStr)
	if err != nil {
		log.Fatalf("解析模板失败: %v", err)
	}

	return tmpl
}
