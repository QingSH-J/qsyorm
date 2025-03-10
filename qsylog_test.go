package main

import (
	"log"
	"os"
	"qsyorm/qsylog"
)

func test() {
	newlogger := qsylog.New(log.New(os.Stdout, "", log.LstdFlags), qsylog.Config{
		Colorful: true,
		Loglevel: qsylog.Info,
	})

	newlogger.Info("Hello World")

}

func main() {
	test()
}
