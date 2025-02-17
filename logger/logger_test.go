package logger_test

import (
	"math"
	"testing"
	"time"

	"github.com/wecisecode/util/cfg"
	"github.com/wecisecode/util/logger"
	"github.com/wecisecode/util/mfmt"
)

type DiDi struct {
	log *logger.Logger
}

func (dd *DiDi) String() string {
	dd.log.Info("haha")
	return "didi"
}

func TestUserdefineFormat(t *testing.T) {
	log := logger.New()
	log.SetRollingFile("", "_test/test.log", 1*time.Minute, 2*mfmt.MB, math.MaxInt64, 2)
	log.SetFormat("yyyy-MM-dd HH:mm:ss.SSSSSS [pid] [level] file:line [module] msg", "\n")
	// test add userdefined formater
	log.AddFormat("ssssss", func(buf *[]byte, fa *logger.FmtArgs) {
		*buf = append(*buf, "***"...)
	})
	log.AddFormat("module", func(buf *[]byte, fa *logger.FmtArgs) {
		*buf = append(*buf, "test"...)
	})
	didi := &DiDi{log}
	log.Info("hello")
	log.Info(didi)
}

func TestConfig(t *testing.T) {
	mc := cfg.MConfig(cfg.GetLogConfCfgOption("log.conf"),
		&cfg.CfgOption{
			Name: "test",
			Type: cfg.INI_TEXT,
			Values: []string{
				`[log]
level=trace         ; 日志级别 trace，debug，info，warn，error，fatal，默认 trace
console=true        ; 是否控制台输出，默认 true
color=true          ; 控制台输出是否根据级别区分颜色，默认 true
consolelevel=debug  ; 控制台显示级别，-1 跟随主级别定义，默认 info
format=             ; 默认 yyyy-MM-dd HH:mm:ss.SSSSSS [pid] [level] file:line msg
eol=                ; 默认 \n
dir=                ; 默认 /opt/matrix/var/logs
file=               ; /opt/matrix/var/logs/<app>/log.log，默认不输出文件，相对路径相对于由 dir 指定的目录
size=2m             ; 尺寸滚动，默认 5MB
unit=               ; deprecated:
count=10            ; 保留数量，默认 20
dialy=              ; deprecated: false 相当于 scroll=-1， true 相当于 scroll=24h 或 1天
scroll=1天          ; 时间滚动 scroll，覆盖 dialy 设置，默认 1 天
expire=14d          ; 保留时间，默认 14 天
`,
			},
		})
	log := logger.New().WithConfig(mc, "log")
	mc.WithLogger(log)
	log.Info("test")
}

func TestDebug(t *testing.T) {
	a := []string{
		"Hello",
	}
	logger.SetLevel(logger.TRACE)
	for _, s := range a {
		logger.Trace(s)
		logger.Debug(s)
		logger.Info(s)
		logger.Warn(s)
		logger.Error(s)
		logger.Fatal(s)
		s = s + " F"
		logger.Tracef(s)
		logger.Debugf(s)
		logger.Infof(s)
		logger.Warnf(s)
		logger.Errorf(s)
		logger.Fatalf(s)
	}

	lg := logger.New()
	lg.SetLevel(logger.LevelDEBUG)
	for _, s := range a {
		s = "New " + s
		lg.Trace(s)
		lg.Debug(s)
		lg.Info(s)
		lg.Warn(s)
		lg.Error(s)
		lg.Fatal(s)
		s = s + " F"
		lg.Tracef(s)
		lg.Debugf(s)
		lg.Infof(s)
		lg.Warnf(s)
		lg.Errorf(s)
		lg.Fatalf(s)
	}
}
