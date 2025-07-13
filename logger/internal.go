package logger

import (
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/wecisecode/util/cast"
)

var defaultLevelMaps = map[int32]*level{
	TRACE: {TRACE, LevelTRACE, "T", []color.Attribute{color.FgCyan}},
	DEBUG: {DEBUG, LevelDEBUG, "D", []color.Attribute{color.FgGreen}},
	INFO:  {INFO, LevelINFO, "I", nil},
	WARN:  {WARN, LevelWARN, "W", []color.Attribute{color.FgYellow}},
	ERROR: {ERROR, LevelERROR, "E", []color.Attribute{color.FgRed}},
	FATAL: {FATAL, LevelFATAL, "F", []color.Attribute{color.FgMagenta}},
}

type level struct {
	id    int32
	name  string
	flag  string
	color []color.Attribute
}

func castToLevel(level interface{}) int32 {
	switch lv := level.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return cast.ToInt32(lv)
	case string:
		return string2Level(lv)
	}
	return 0
}

func string2Level(level string) int32 {
	switch strings.ToUpper(level) {
	case LevelTRACE, LevelFlagTRACE:
		return TRACE
	case LevelDEBUG, LevelFlagDEBUG:
		return DEBUG
	case LevelINFO, LevelFlagINFO:
		return INFO
	case LevelWARN, LevelFlagWARN:
		return WARN
	case LevelERROR, LevelFlagERROR:
		return ERROR
	case LevelFATAL, LevelFlagFATAL:
		return FATAL
	default:
		return UNKNOWN
	}
}

func splitFile(path string) (dir, module, file string) {
	dir, file = filepath.Split(path)
	if dir != "" && dir[len(dir)-1] == '/' {
		dir, module = filepath.Split(dir[:len(dir)-1])
	}
	return
}
