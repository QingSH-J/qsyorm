package main

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"qsyorm/qsyengine"
	"qsyorm/qsylog"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {
	newlogger := qsylog.New(log.New(os.Stdout, " ", log.LstdFlags), qsylog.Config{
		Colorful: true,
		Loglevel: qsylog.Info,
	})
	e, _ := qsyengine.NewQSyEngine("sqlite3", "qsy.db", newlogger)
	defer e.Close()
	s := e.NewSession()
	_, _ = s.Raw("DROP TABLE IF EXISTS User;").Exec()
	_, _ = s.Raw("CREATE TABLE User(Name text);").Exec()
	_, _ = s.Raw("CREATE TABLE User(Name text);").Exec()
	result, _ := s.Raw("INSERT INTO User(`Name`) values (?), (?)", "QingShiyu", "TangZhuozhi").Exec()
	count, _ := result.RowsAffected()
	row, _ := s.Raw("SELECT * FROM User").QueryRows()
	for row.Next() {
		var name string
		err := row.Scan(&name)
		if err != nil {
			s.Logger.Error(err.Error())
		}
		fmt.Printf("%s\n", name)
	}

	fmt.Printf("Exec success, %d affected\n", count)
}
