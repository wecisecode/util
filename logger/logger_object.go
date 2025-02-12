package logger

import (
	"encoding/json"
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
	depth       int
	fileoutpath string
	bfasmu      sync.RWMutex
	bfAppenders map[string]*bfappender.BufferedFileAppender
}

func New(opt ...*Option) *Logger {
	option := defaultOption.Merge(opt...)
	return &Logger{option: option}
}

var DefaultLogsDir = filepath.Join("/", "opt", "matrix", "var", "logs")

// [log]              ; 日志配置参数
// level=trace        ; 日志级别 trace，debug，info，warn，error，fatal，默认 trace
// console=true       ; 是否控制台输出，默认 true
// color=true         ; 控制台输出是否根据级别区分颜色，默认 true
// consolelevel=info  ; 控制台显示级别，-1 跟随主级别定义，默认 info
// format=            ; 默认 yyyy-MM-dd HH:mm:ss.SSSSSS [pid] [level] file:line msg
// eol=\r\n           ; 默认 \n
// file=              ; /opt/matrix/var/logs/<app>/log.log，默认不输出文件
// size=5m            ; 尺寸滚动，默认 5MB
// unit=              ; deprecated:
// count=20           ; 保留数量，默认 20
// dialy=             ; deprecated: false 相当于 scroll=-1， true 相当于 scroll=24h 或 1天
// scroll=1天         ; 时间滚动 scroll，覆盖 dialy 设置，默认 1 天
// expire=14d         ; 保留时间，默认 14 天
func (log *Logger) WithConfig(mcfg Configure, keyprefix ...string) *Logger {
	mcfg.OnChange(func() {
		scroll := time.Duration(0)
		if daily := mcfg.GetString(cfgkey(keyprefix, "daily"), ""); daily != "" {
			if cast.ToBool(daily) {
				scroll = 24 * time.Hour
			} else {
				scroll = -1
			}
		}
		bsunit := int64(1)
		if unit := mcfg.GetString(cfgkey(keyprefix, "unit"), ""); unit != "" {
			bsunit = mfmt.ParseBytesCount("1" + unit)
		}
		format := func(s string) (rs string) {
			e := json.Unmarshal([]byte(`"`+s+`"`), &rs)
			if e != nil {
				rs = s
			}
			return
		}
		logfilepath := mcfg.GetString(cfgkey(keyprefix, "file"), "")
		if len(logfilepath) > 0 && !filepath.IsAbs(logfilepath) {
			dir := mcfg.GetString(cfgkey(keyprefix, "dir"), DefaultLogsDir)
			logfilepath = filepath.Join(dir, logfilepath)
		}
		log.SetConsole(mcfg.GetBool(cfgkey(keyprefix, "console"), true))
		log.SetColor(mcfg.GetBool(cfgkey(keyprefix, "color"), true))
		log.SetConsoleLevel(mcfg.GetString(cfgkey(keyprefix, "consolelevel"), LevelINFO))
		log.SetLevel(mcfg.GetString(cfgkey(keyprefix, "level"), LevelTRACE))
		log.SetFormat(format(mcfg.GetString(cfgkey(keyprefix, "format"), "")), format(mcfg.GetString(cfgkey(keyprefix, "eol"), "")))
		log.SetRollingFile("",
			logfilepath,
			mcfg.GetDuration(cfgkey(keyprefix, "scroll"), scroll),
			mcfg.GetBytsCount(cfgkey(keyprefix, "size"), 0)*bsunit,
			mcfg.GetDuration(cfgkey(keyprefix, "expire"), 0),
			mcfg.GetInt(cfgkey(keyprefix, "count"), 0))
	})
	return log
}

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
func (l *Logger) SetRollingFile(module string, filepath string, ScrollByTime time.Duration, ScrollBySize int64, ScrollKeepTime time.Duration, ScrollKeepCount int) {
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
	l.fileoutpath = filepath
	l.bfAppenders[module] = bfappender.MBufferedFileAppender(filepath, DefaultLogFileOption).WithOption(&bfappender.Option{
		ScrollByTime:    ScrollByTime,
		ScrollBySize:    ScrollBySize,
		ScrollKeepTime:  ScrollKeepTime,
		ScrollKeepCount: ScrollKeepCount,
	})
}

func (l *Logger) FileOutPath() string {
	return l.fileoutpath
}

func (l *Logger) SetDepth(depth int) {
	l.depth = depth
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

func (l *Logger) SetLevel(level interface{}) {
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

func (l *Logger) SetConsole(isConsole bool) {
	if isConsole {
		l.SetConsoleOut(os.Stdout)
	} else {
		l.SetConsoleOut(nil)
	}
}

func (l *Logger) SetConsoleLevel(level interface{}) {
	switch lv := level.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		l.option.ConsoleLevel = cast.ToInt32(lv)
	case string:
		l.option.ConsoleLevel = string2Level(lv)
	}
}

func (l *Logger) SetColor(isColor bool) {
	l.option.ConsoleColor = isColor
}

func (l *Logger) SetLevelAtrribute(id int32, name string, flag string, colours []color.Attribute) {
	l.option.SetLevelAtrribute(id, name, flag, colours)
}

func (l *Logger) SetFormat(fmt string, eol string) {
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
	if lg.depth != 0 {
		calldepth = lg.depth
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
