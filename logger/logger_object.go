package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cast"
	"github.com/wecisecode/util/bfappender"
	"github.com/wecisecode/util/mfmt"
)

type Configure interface {
	GetString(key string, defaultvalue ...string) string
	GetInt(key string, defaultvalue ...int) int
	GetBool(key string, defaultvalue ...bool) bool
	GetFloat(key string, defaultvalue ...float64) float64
	GetBytsCount(key string, defaultvalue ...interface{}) int64
	GetDuration(key string, defaultvalue ...interface{}) time.Duration
	OnChange(func()) int64
}

type Logger struct {
	option      *Option
	lc          sync.Mutex
	bfasmu      sync.RWMutex
	bfAppenders map[string]*bfappender.BufferedFileAppender
	setting     Setting // 代码设置的选项，优先于option
}

func New(opt ...*Option) *Logger {
	option := defaultOption.Merge(opt...)
	return &Logger{option: option}
}

var DefaultLogsDir = filepath.Join("/", "opt", "matrix", "var", "logs")

func cfgkey(keyprefixs []string, key string) (cfgkey string) {
	if len(keyprefixs) == 0 {
		return key
	}
	cfgkey = ""
	for _, keyprefix := range keyprefixs {
		kpfs := strings.Split(keyprefix, "|")
		for _, kpf := range kpfs {
			keyprefix := strings.TrimSpace(kpf)
			if keyprefix != "" && keyprefix[len(keyprefix)-1] != '.' {
				keyprefix += "."
			}
			if cfgkey != "" {
				cfgkey += "|"
			}
			cfgkey += keyprefix + key
		}
	}
	return
}

var DefaultLogFileOption = &bfappender.Option{
	RecordEndFlag:    []byte("\n"),
	FlushAtLeastTime: -1,
	FlushOverSize:    -1,
	ScrollByTime:     24 * time.Hour,
	ScrollBySize:     5 * mfmt.MB,
	ScrollKeepTime:   14 * 24 * time.Hour,
	ScrollKeepCount:  20,
	UseGoBufIOWriter: false,
	ErrorLog:         filepath.Join("error.log"),
}

// ScrollByTime        time.Duration // 滚动时间，-1 无限，0 默认 1天
// ScrollBySize        int64         // 滚动尺寸，-1 无限，0 默认 5MB
// ScrollKeepTime      time.Duration // 滚动文件保留最长时间，-1 不留，math.MaxInt64 长期保留，0 默认 14天
// ScrollKeepCount     int           // 滚动文件保留最多数量，-1 不留，math.MaxInt64 长期保留，0 默认 20
func (l *Logger) setRollingFile(module string, filepath string, ScrollByTime time.Duration, ScrollBySize int64, ScrollKeepTime time.Duration, ScrollKeepCount int) {
	if l.setting.module != nil && *l.setting.module != module ||
		l.setting.filepath != nil && *l.setting.filepath != filepath ||
		l.setting.ScrollByTime != nil && *l.setting.ScrollByTime != ScrollByTime ||
		l.setting.ScrollBySize != nil && *l.setting.ScrollBySize != ScrollBySize ||
		l.setting.ScrollKeepTime != nil && *l.setting.ScrollKeepTime != ScrollKeepTime ||
		l.setting.ScrollKeepCount != nil && *l.setting.ScrollKeepCount != ScrollKeepCount {
		// 代码设置优先
		return
	}
	l.bfasmu.Lock()
	defer l.bfasmu.Unlock()
	if l.bfAppenders == nil {
		l.bfAppenders = make(map[string]*bfappender.BufferedFileAppender)
	}
	if l.bfAppenders[module] != nil {
		l.bfAppenders[module].Close()
		l.bfAppenders[module] = nil
	}
	if filepath == "" {
		return
	}
	l.option.fileoutpath = filepath
	l.bfAppenders[module] = bfappender.MBufferedFileAppender(filepath, DefaultLogFileOption).WithOption(&bfappender.Option{
		ScrollByTime:    ScrollByTime,
		ScrollBySize:    ScrollBySize,
		ScrollKeepTime:  ScrollKeepTime,
		ScrollKeepCount: ScrollKeepCount,
	})
}

func (l *Logger) FileOutPath() string {
	return l.option.fileoutpath
}

func (l *Logger) SetDepth(depth int) {
	l.option.depth = depth
}

func (l *Logger) Level() (int32, string) {
	return l.option.Level, l.option.LevelName
}

func (l *Logger) FileOutLevel() int32 {
	return l.option.Level
}

func (l *Logger) ConsoleLevel() int32 {
	return l.option.ConsoleLevel
}

func (l *Logger) setLevel(level interface{}) {
	if l.setting.level != nil && l.setting.level != level {
		// 代码设置优先
		return
	}
	switch lv := level.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		lvl := l.option.level(cast.ToInt32(lv))
		if lvl != nil {
			l.option.Level = lvl.id
			l.option.LevelName = lvl.name
		}
	case string:
		l.option.LevelName = lv
		l.option.Level = string2Level(lv)
	}
}

func (l *Logger) SetConsoleOut(consoleout io.Writer) {
	l.option.Console = consoleout
}

func (l *Logger) setConsole(isConsole bool) {
	if l.setting.isConsole != nil && *l.setting.isConsole != isConsole {
		// 代码设置优先
		return
	}
	if isConsole {
		l.SetConsoleOut(os.Stdout)
	} else {
		l.SetConsoleOut(nil)
	}
}

func (l *Logger) setConsoleLevel(level interface{}) {
	if l.setting.consolelevel != nil && l.setting.consolelevel != level {
		// 代码设置优先
		return
	}
	switch lv := level.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		l.option.ConsoleLevel = cast.ToInt32(lv)
	case string:
		l.option.ConsoleLevel = string2Level(lv)
	}
}

func (l *Logger) setColor(isColor bool) {
	if l.setting.isColor != nil && *l.setting.isColor != isColor {
		// 代码设置优先
		return
	}
	l.option.ConsoleColor = isColor
}

func (l *Logger) SetLevelAtrribute(id int32, name string, flag string, colours []color.Attribute) {
	l.option.SetLevelAtrribute(id, name, flag, colours)
}

func (l *Logger) setFormat(fmt string, eol string) {
	if l.setting.fmt != nil && *l.setting.fmt != fmt ||
		l.setting.eol != nil && *l.setting.eol != eol {
		// 代码设置优先
		return
	}
	l.option.SetFormat(fmt, eol)
}

func (l *Logger) AddFormat(name string, f func(buf *[]byte, fa *FmtArgs)) {
	l.option.formater.AddFormat(name, f)
}

func (l *Logger) Format(t time.Time, level string, module string, file string, line int, pc uintptr, fmtf string, args ...interface{}) string {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	fa := &FmtArgs{
		year,
		int(month),
		day,
		hour,
		min,
		sec,
		t.Nanosecond(),
		level,
		module,
		file,
		line,
		pc,
		fmtf,
		args,
	}
	return l.option.formater.Format(fa)
}

func (l *Logger) Fatal(a ...interface{}) {
	l.PrintOut(FATAL, "", a...)
}

func (l *Logger) Fatalf(format string, a ...interface{}) {
	l.PrintOut(FATAL, format, a...)
}

func (l *Logger) Error(a ...interface{}) {
	l.PrintOut(ERROR, "", a...)
}

func (l *Logger) Errorf(format string, a ...interface{}) {
	l.PrintOut(ERROR, format, a...)
}

func (l *Logger) Warn(a ...interface{}) {
	l.PrintOut(WARN, "", a...)
}

func (l *Logger) Warnf(format string, a ...interface{}) {
	l.PrintOut(WARN, format, a...)
}

func (l *Logger) Info(a ...interface{}) {
	l.PrintOut(INFO, "", a...)
}

func (l *Logger) Infof(format string, a ...interface{}) {
	l.PrintOut(INFO, format, a...)
}

func (l *Logger) Debug(a ...interface{}) {
	l.PrintOut(DEBUG, "", a...)
}

func (l *Logger) Debugf(format string, a ...interface{}) {
	l.PrintOut(DEBUG, format, a...)
}

func (l *Logger) Trace(a ...interface{}) {
	l.PrintOut(TRACE, "", a...)
}

func (l *Logger) Tracef(format string, a ...interface{}) {
	l.PrintOut(TRACE, format, a...)
}

func (l *Logger) Print(a ...interface{}) {
	l.PrintOut(INFO, "", a...)
}

func (l *Logger) Printf(format string, a ...interface{}) {
	l.PrintOut(INFO, format, a...)
}

func (lg *Logger) PrintOut(level interface{}, format string, v ...interface{}) bool {
	var calldepth = 2
	if lg.option.depth != 0 {
		calldepth = lg.option.depth
	}
	return lg.Output(calldepth+1, castToLevel(level), false, format, v...)
}

func (l *Logger) Output(calldepth int, level int32, force bool, format string, v ...interface{}) bool {
	pc, file, line, _ := runtime.Caller(calldepth)
	lv := l.option.level(level)
	if lv == nil {
		lv = l.option.level(INFO)
	}
	return l.writeLog(false, force, level, lv.flag, lv.color, file, line, pc, format, v...)
}

func (lg *Logger) getOutputFileForModule(module string) (bfa *bfappender.BufferedFileAppender) {
	lg.bfasmu.RLock()
	defer lg.bfasmu.RUnlock()
	bfa = lg.bfAppenders[module]
	if bfa == nil {
		bfa = lg.bfAppenders[""]
	}
	return
}

func (lg *Logger) WriteLog(consoleonly bool, level int32, levelName string, colours []color.Attribute, file string, line int, fmtf string, args ...interface{}) {
	lg.writeLog(consoleonly, false, level, levelName, colours, file, line, 0, fmtf, args...)
}

func (lg *Logger) writeLog(consoleonly bool, force bool, level int32, levelName string, colours []color.Attribute, filepath string, line int, pc uintptr, fmtf string, args ...interface{}) (output bool) {
	defer func() {
		if x := recover(); x != nil {
			fmt.Println("log output error:", x)
		}
	}()
	var bs []byte

	_, module, shortfile := splitFile(filepath)
	if !consoleonly {
		bfa := lg.getOutputFileForModule(module)
		if bfa != nil && (lg.option.Level <= level || force) {
			bs = []byte(lg.Format(time.Now(), levelName, module, shortfile, line, pc, fmtf, args...))
			bfa.Write(bs)
			output = true
		}
	}

	if lg.option.Console != nil && (lg.option.ConsoleLevel >= 0 && lg.option.ConsoleLevel <= level ||
		lg.option.ConsoleLevel < 0 && lg.option.Level <= level) {
		if bs == nil {
			bs = []byte(lg.Format(time.Now(), levelName, module, shortfile, line, pc, fmtf, args...))
		}
		lg.lc.Lock()
		defer lg.lc.Unlock()
		if lg.option.ConsoleColor && colours != nil {
			color.Set(colours...)
		}
		lg.option.Console.Write(bs)
		if lg.option.ConsoleColor && colours != nil {
			color.Unset()
		}
		output = true
	}

	return
}
