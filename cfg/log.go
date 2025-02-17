package cfg

import (
	"fmt"
	"strings"
	"sync"
)

// 避免与日志循环引用
// 通过 cfg.WithLogger 调整日志输出相关配置
// 默认 cfg.log 不输出任何信息，仅缓存最后100条信息，待通过 cfg.WithLogger 配置日志时输出

type ConfLog interface {
	PrintOut(level interface{}, format string, v ...interface{}) bool
	Debug(...interface{})
	Warn(...interface{})
	Error(...interface{})
}

type mConfLog struct {
	bufmux sync.Mutex
	buffer [][]string
	applog map[ConfLog]ConfLog
}

func (mc *mConfLog) RemoveLog(log ConfLog) {
	mc.bufmux.Lock()
	if mc.applog != nil {
		delete(mc.applog, log)
	}
	mc.bufmux.Unlock()
}
func (mc *mConfLog) AppLog(log ConfLog, forceclearbuffer bool) {
	mc.bufmux.Lock()
	if mc.applog == nil {
		mc.applog = map[ConfLog]ConfLog{}
	}
	mc.applog[log] = log
	nbuf := [][]string{}
	for _, info := range mc.buffer {
		if !log.PrintOut(info[0], "", info[1]) && !forceclearbuffer {
			nbuf = append(nbuf, info)
		}
	}
	mc.buffer = nbuf
	mc.bufmux.Unlock()
}
func (mc *mConfLog) PrintOut(level interface{}, format string, a ...interface{}) bool {
	mc.bufmux.Lock()
	output := false
	for applog := range mc.applog {
		output = output || applog.PrintOut(level, format, a...)
	}
	if !output {
		s := fmt.Sprintln(a...)
		s = strings.TrimRight(s, "\r\n")
		mc.buffer = append(mc.buffer, []string{level.(string), s})
		if len(mc.buffer) > 100 {
			mc.buffer = mc.buffer[len(mc.buffer)-100:]
		}
	}
	mc.bufmux.Unlock()
	return output
}
func (mc *mConfLog) Debug(a ...interface{}) {
	mc.PrintOut("D", "", a...)
}
func (mc *mConfLog) Warn(a ...interface{}) {
	mc.PrintOut("W", "", a...)
}
func (mc *mConfLog) Error(a ...interface{}) {
	mc.PrintOut("E", "", a...)
}
