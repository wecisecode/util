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

var Logger logger = ulog.DefaultLogger()
