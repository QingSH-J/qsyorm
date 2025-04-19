package qsysession_test

import (
	"log"
	"os"
	"qsyorm/qsyengine"
	"qsyorm/qsylog"
	"testing"
)

type User struct {
	Name string `qsy:"PRIMARY KEY"`
	Age  int
}

func Test(t *testing.T) {
	newlogger := qsylog.New(log.New(os.Stdout, " ", log.LstdFlags), qsylog.Config{
		Colorful: true,
		Loglevel: qsylog.Info,
	})
	e, _ := qsyengine.NewQSyEngine("sqlite3", "qsy.db", newlogger)
	s := e.NewSession().Model(&User{})
	err := s.DropTable()
	if err != nil {
		t.Fatal("error")
	}
	err1 := s.CreateTable()
	if err1 != nil {
		t.Fatal("error")
	}

}
