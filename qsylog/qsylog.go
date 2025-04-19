//first version qsylog 1.0
//package qsylog
//
//import (
//	"io/ioutil"
//	"log"
//	"os"
//	"sync"
//)
//
//var (
//	errorlog = log.New(os.Stdout, "\033[1;31;40m[error]\033[0m ", log.LstdFlags|log.Lmicroseconds)
//	infolog  = log.New(os.Stdout, "\033[1;33;40m[info]\033[0m ", log.LstdFlags|log.Lmicroseconds)
//	loggers  = []*log.Logger{errorlog, infolog}
//	mulock   sync.Mutex
//)
//
//// log methods
//var (
//	ERROR = errorlog.Println
//	INFO  = infolog.Println
//)
//
//const (
//	InfoLevel = iota
//	ErrorLevel
//	Disabled
//)
//
//func setLevel(level int) {
//	mulock.Lock()
//	defer mulock.Unlock()
//	for _, logger := range loggers {
//		logger.SetOutput(os.Stdout)
//	}
//	if ErrorLevel < level {
//		errorlog.SetOutput(ioutil.Discard)
//	}
//	if InfoLevel < level {
//		infolog.SetOutput(ioutil.Discard)
//	}
//
//}

// second log version 2.0
package qsylog

//the package used in qsylog
import (
	"fmt"
	"io"
	"log"
	"os"
)

// color
const (
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Reset  = "\033[0m"
)

// log level
type Loglevel int

const (
	Silent Loglevel = iota
	Error
	Warning
	Info
)

// interface to printf
type Write interface {
	Printf(string, ...interface{})
}

// log config
type Config struct {
	Colorful bool
	Loglevel Loglevel
}

// log interface
type Interface interface {
	Info(s string, v ...interface{})
	Warn(s string, v ...interface{})
	Error(s string, v ...interface{})
}

var (
	//for the logger which is discard
	Discard = New(log.New(io.Discard, "", log.LstdFlags), Config{})

	//for the logger which is default
	Default = New(log.New(os.Stdout, "\r\n", log.LstdFlags), Config{
		Colorful: true,
		Loglevel: Warning,
	})
)

func New(writer Write, config Config) Interface {
	var (
		errorPrefix = "[error] "
		warnPrefix  = "[warn] "
		infoPrefix  = "[info] "
	)

	//if config.colorful == 1, then set the color
	if config.Colorful {
		errorPrefix = Red + "[error] " + Reset
		warnPrefix = Yellow + "[warn] " + Reset
		infoPrefix = Green + "[info] " + Reset
	}

	//return a looger(struct) that can implement the Interface
	return &logger{
		Writer:      writer,
		Config:      config,
		errorPrefix: errorPrefix,
		warnPrefix:  warnPrefix,
		infoPrefix:  infoPrefix,
	}
}

type logger struct {
	Writer                              Write
	Config                              Config
	errorPrefix, warnPrefix, infoPrefix string
}

// now implement the methods in Interface
func (l *logger) Info(s string, v ...interface{}) {
	if l.Config.Loglevel >= Info {
		var msg string
		if len(v) > 0 {
			msg = fmt.Sprintf(s, v...)
		} else {
			msg = s
		}
		l.Writer.Printf("%s%s\n", l.infoPrefix, msg)
	}
}

func (l *logger) Warn(s string, v ...interface{}) {
	if l.Config.Loglevel >= Warning {
		var msg string
		if len(v) > 0 {
			msg = fmt.Sprintf(s, v...)
		} else {
			msg = s
		}
		l.Writer.Printf("%s%s\n", l.warnPrefix, msg)
	}
}

func (l *logger) Error(s string, v ...interface{}) {
	if l.Config.Loglevel >= Error {
		var msg string
		if len(v) > 0 {
			msg = fmt.Sprintf(s, v...)
		} else {
			msg = s
		}
		l.Writer.Printf("%s%s\n", l.errorPrefix, msg)
	}
}
