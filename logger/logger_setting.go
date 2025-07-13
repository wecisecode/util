package logger

import (
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/wecisecode/util/cast"
	"github.com/wecisecode/util/mfmt"
)

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
		log.setConsole(mcfg.GetBool(cfgkey(keyprefix, "console"), true))
		log.setColor(mcfg.GetBool(cfgkey(keyprefix, "color"), true))
		log.setConsoleLevel(mcfg.GetString(cfgkey(keyprefix, "consolelevel"), LevelINFO))
		log.setLevel(mcfg.GetString(cfgkey(keyprefix, "level"), LevelTRACE))
		log.setFormat(format(mcfg.GetString(cfgkey(keyprefix, "format"), "")), format(mcfg.GetString(cfgkey(keyprefix, "eol"), "")))
		log.setRollingFile("",
			logfilepath,
			mcfg.GetDuration(cfgkey(keyprefix, "scroll"), scroll),
			mcfg.GetBytsCount(cfgkey(keyprefix, "size"), 0)*bsunit,
			mcfg.GetDuration(cfgkey(keyprefix, "expire"), 0),
			mcfg.GetInt(cfgkey(keyprefix, "count"), 0))
	})
	return log
}

type Setting struct {
	module          *string
	filepath        *string
	ScrollByTime    *time.Duration
	ScrollBySize    *int64
	ScrollKeepTime  *time.Duration
	ScrollKeepCount *int
	level           any
	isConsole       *bool
	consolelevel    any
	isColor         *bool
	fmt             *string
	eol             *string
}

// ScrollByTime        time.Duration // 滚动时间，-1 无限，0 默认 1天
// ScrollBySize        int64         // 滚动尺寸，-1 无限，0 默认 5MB
// ScrollKeepTime      time.Duration // 滚动文件保留最长时间，-1 不留，math.MaxInt64 长期保留，0 默认 14天
// ScrollKeepCount     int           // 滚动文件保留最多数量，-1 不留，math.MaxInt64 长期保留，0 默认 20
func (l *Logger) SetRollingFile(module string, filepath string, ScrollByTime time.Duration, ScrollBySize int64, ScrollKeepTime time.Duration, ScrollKeepCount int) {
	l.setting.module = &module
	l.setting.filepath = &filepath
	l.setting.ScrollByTime = &ScrollByTime
	l.setting.ScrollBySize = &ScrollBySize
	l.setting.ScrollKeepTime = &ScrollKeepTime
	l.setting.ScrollKeepCount = &ScrollKeepCount
	l.setRollingFile(module, filepath, ScrollByTime, ScrollBySize, ScrollKeepTime, ScrollKeepCount)
}

func (l *Logger) SetLevel(level interface{}) {
	l.setting.level = level
	l.setLevel(level)
}

func (l *Logger) SetConsole(isConsole bool) {
	l.setting.isConsole = &isConsole
	l.setConsole(isConsole)
}

func (l *Logger) SetConsoleLevel(level interface{}) {
	l.setting.consolelevel = level
	l.setConsoleLevel(level)
}

func (l *Logger) SetColor(isColor bool) {
	l.setting.isColor = &isColor
	l.setColor(isColor)
}

func (l *Logger) SetFormat(fmt string, eol string) {
	l.setting.fmt = &fmt
	l.setting.eol = &eol
	l.setFormat(fmt, eol)
}
