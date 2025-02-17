package logger

import (
	"os"
	"time"
)

const DATEFORMAT = "2006-01-02"

const (
	UNKNOWN int32 = iota
	TRACE
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	OFF
)

const (
	LevelTRACE     = "TRACE"
	LevelDEBUG     = "DEBUG"
	LevelINFO      = "INFO"
	LevelWARN      = "WARN"
	LevelERROR     = "ERROR"
	LevelFATAL     = "FATAL"
	LevelFlagTRACE = "T"
	LevelFlagDEBUG = "D"
	LevelFlagINFO  = "I"
	LevelFlagWARN  = "W"
	LevelFlagERROR = "E"
	LevelFlagFATAL = "F"
)

var defaultLogger = New()

func DefaultLogger() *Logger {
	return defaultLogger
}

func SetConsole(isConsole bool) {
	if isConsole {
		defaultLogger.SetConsoleOut(os.Stdout)
	} else {
		defaultLogger.SetConsoleOut(nil)
	}
}

func SetColor(isColor bool) {
	defaultLogger.SetColor(isColor)
}

func SetLevel(level interface{}) {
	defaultLogger.SetLevel(level)
}

func SetConsoleLevel(level interface{}) {
	defaultLogger.SetConsoleLevel(level)
}

func SetFormat(fmt string, eol string) {
	defaultLogger.SetFormat(fmt, eol)
}

func SetRollingFile(local string, filepath string, ScrollByTime time.Duration, ScrollBySize int64, ScrollKeepTime time.Duration, ScrollKeepCount int) {
	defaultLogger.SetRollingFile(local, filepath, ScrollByTime, ScrollBySize, ScrollKeepTime, ScrollKeepCount)
}

func Trace(a ...interface{}) {
	defaultLogger.PrintOut(TRACE, "", a...)
}

func Tracef(format string, a ...interface{}) {
	defaultLogger.PrintOut(TRACE, format, a...)
}

func Debug(a ...interface{}) {
	defaultLogger.PrintOut(DEBUG, "", a...)
}

func Debugf(format string, a ...interface{}) {
	defaultLogger.PrintOut(DEBUG, format, a...)
}

func Info(a ...interface{}) {
	defaultLogger.PrintOut(INFO, "", a...)
}

func Infof(format string, a ...interface{}) {
	defaultLogger.PrintOut(INFO, format, a...)
}

func Warn(a ...interface{}) {
	defaultLogger.PrintOut(WARN, "", a...)
}

func Warnf(format string, a ...interface{}) {
	defaultLogger.PrintOut(WARN, format, a...)
}

func Error(a ...interface{}) {
	defaultLogger.PrintOut(ERROR, "", a...)
}

func Errorf(format string, a ...interface{}) {
	defaultLogger.PrintOut(ERROR, format, a...)
}

func Fatal(a ...interface{}) {
	defaultLogger.PrintOut(FATAL, "", a...)
}

func Fatalf(format string, a ...interface{}) {
	defaultLogger.PrintOut(FATAL, format, a...)
}

// Deprecated: 不建议使用
// func Write(l *Logger, s string, colour color.Attribute, level int32, levelName string, file string, line int, logObj interface{}) {
// 	// logObj *file 参数是包内类型，没有公开函数返回此类型变量，所以外部直接调用 Write 时 logObj 一定为空，即内容只会输出到 console
// 	// 输出文件相关信息转移到Logger内部，内部调用 logObj *file 参数从 lg.config.file[file] 获取或使用 lg.config.Default.LogObj
// 	l.WriteLog(true, level, levelName, colour, file, line, s)
// }

// // Deprecated: 建议使用 Logger.Output
// func Output(l *Logger, depth int, level int32, format string, isFormat, isForce bool, v ...interface{}) {
// 	if !isFormat {
// 		format = ""
// 	}
// 	l.Output(depth+1, level, isForce, format, v...)
// }
