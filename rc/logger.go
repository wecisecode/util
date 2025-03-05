package rc

import (
	ulog "github.com/wecisecode/util/logger"
)

type logger interface {
	Trace(args ...interface{})
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
}

var defaultlogger = ulog.New()

func init() {
	defaultlogger.SetConsoleLevel(ulog.ERROR)
}

var Logger logger = defaultlogger
