package logger

import (
	"io"
	"os"
	"sync"

	"github.com/fatih/color"
)

type Option struct {
	Level        int32
	LevelName    string
	Console      io.Writer
	ConsoleLevel int32 // -1 follow with Level
	ConsoleColor bool
	formater     *Formater
	lmux         sync.RWMutex
	levelMaps    map[int32]*level
}

var defaultOption = &Option{
	TRACE,
	LevelTRACE,
	os.Stdout,
	-1,
	true,
	MFormater("yyyy-MM-dd HH:mm:ss.SSSSSS [pid] [level] module/file:line msg", "\n"),
	sync.RWMutex{},
	defaultLevelMaps}

func (opt *Option) Merge(aos ...*Option) *Option {
	oo := &Option{
		opt.Level,
		opt.LevelName,
		opt.Console,
		opt.ConsoleLevel,
		opt.ConsoleColor,
		nil,
		sync.RWMutex{},
		nil,
	}
	oo.SetFormat(opt.formater.format, opt.formater.eol)
	opt.lmux.RLock()
	for _, l := range opt.levelMaps {
		oo.SetLevelAtrribute(l.id, l.name, l.flag, l.color)
	}
	opt.lmux.RUnlock()
	for _, a := range aos {
		oo.Level = a.Level
		oo.LevelName = a.LevelName
		if a.Console != nil {
			oo.Console = a.Console
		}
		oo.ConsoleLevel = a.ConsoleLevel
		oo.ConsoleColor = a.ConsoleColor
		if a.formater != nil {
			oo.SetFormat(a.formater.format, a.formater.eol)
		}
		a.lmux.RLock()
		for _, l := range a.levelMaps {
			oo.SetLevelAtrribute(l.id, l.name, l.flag, l.color)
		}
		a.lmux.RUnlock()
	}
	return oo
}

func (opt *Option) SetFormat(fmt string, eol string) {
	if fmt == "" {
		fmt = defaultOption.formater.format
	}
	if eol == "" {
		eol = defaultOption.formater.eol
	}
	if opt.formater == nil {
		opt.formater = MFormater(fmt, eol)
	} else {
		opt.formater.SetFormat(fmt, eol)
	}
}

func (opt *Option) GetFormat() (fmt string, eol string) {
	if opt.formater == nil {
		return defaultOption.GetFormat()
	}
	return opt.formater.GetFormat()
}

func (opt *Option) SetLevelAtrribute(id int32, name string, flag string, colours []color.Attribute) {
	opt.lmux.Lock()
	defer opt.lmux.Unlock()
	if opt.levelMaps == nil {
		opt.levelMaps = map[int32]*level{}
		for n, l := range defaultLevelMaps {
			opt.levelMaps[n] = l
		}
	}
	opt.levelMaps[id] = &level{id, name, flag, colours}
}

func (opt *Option) level(id int32) *level {
	opt.lmux.RLock()
	defer opt.lmux.RUnlock()
	if opt.levelMaps == nil {
		return defaultLevelMaps[id]
	}
	return opt.levelMaps[id]
}
