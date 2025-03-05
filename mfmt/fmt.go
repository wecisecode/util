package mfmt

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cast"
)

const (
	_        = iota
	KB int64 = 1 << (iota * 10)
	MB
	GB
	TB
	PB
	EB
)

var reBytes = regexp.MustCompile(`^(\d+)\s*(.*)$`)

// 支持的单位：
// "K", "KB",
// "M", "MB",
// "G", "GB",
// "T", "TB",
// "P", "PB",
// "E", "EB",
// "B", "BYTE", "BYTES"
// 默认 BYTE
func ParseBytesCount(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	sign := int64(1)
	if s[0] == '-' {
		sign = -1
		s = s[1:]
	}
	ss := reBytes.FindStringSubmatch(s)
	if len(ss) < 3 {
		return 0
	}
	unit := int64(0)
	switch strings.ToUpper(ss[2]) {
	case "K", "KB":
		unit = KB
	case "M", "MB":
		unit = MB
	case "G", "GB":
		unit = GB
	case "T", "TB":
		unit = TB
	case "P", "PB":
		unit = PB
	case "E", "EB":
		unit = EB
	case "", "B", "BYTE", "BYTES":
		unit = 1
	}
	return sign * cast.ToInt64(ss[1]) * unit
}

func BytesSize(b int64) string {
	if b <= -10000 {
		b = (b - 512) / 1024
		if b > -10000 {
			return fmt.Sprintf("%dK", b)
		}
		b = (b - 512) / 1024
		if b > -10000 {
			return fmt.Sprintf("%dM", b)
		}
		b = (b - 512) / 1024
		if b > -10000 {
			return fmt.Sprintf("%dG", b)
		}
		b = (b - 512) / 1024
		if b > -10000 {
			return fmt.Sprintf("%dT", b)
		}
		b = (b - 512) / 1024
		return fmt.Sprintf("%dP", b)
	}
	if b >= 10000 {
		b = (b + 512) / 1024
		if b < 10000 {
			return fmt.Sprintf("%dK", b)
		}
		b = (b + 512) / 1024
		if b < 10000 {
			return fmt.Sprintf("%dM", b)
		}
		b = (b + 512) / 1024
		if b < 10000 {
			return fmt.Sprintf("%dG", b)
		}
		b = (b + 512) / 1024
		if b < 10000 {
			return fmt.Sprintf("%dT", b)
		}
		b = (b + 512) / 1024
		return fmt.Sprintf("%dP", b)
	}
	return fmt.Sprintf("%d", b)
}

// __d__h__m__.__s | __µs | __ns
func FormatDuration(td time.Duration) (ret string) {
	ms := int(td.Milliseconds())
	if ms > 86400000 || len(ret) > 0 {
		ret += strconv.Itoa(ms/86400000) + "d"
		ms %= 86400000
	}
	if ms > 3600000 || len(ret) > 0 {
		ret += strconv.Itoa(ms/3600000) + "h"
		ms %= 3600000
	}
	if ms > 60000 || len(ret) > 0 {
		ret += strconv.Itoa(ms/60000) + "m"
		ms %= 60000
	}
	if len(ret) > 0 {
		ret += fmt.Sprintf("%d", ms/1000) + "s"
	} else {
		// 不足一分钟
		if td >= 10000*time.Millisecond {
			// 十秒以上
			ret = fmt.Sprintf("%0.2f", float32(ms)/1000) + "s"
		} else if td >= 100*time.Millisecond {
			// 不足十秒，100毫秒以上
			ret = fmt.Sprintf("%0.3f", float32(ms)/1000) + "s"
		} else {
			// 不足100毫秒
			ret = td.String()
		}
	}
	return
}

var reDuration = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(\D+)`)

// 默认单位为毫秒，可识别的单位包括：
// "d", "day", "days", "天",
// "h", "hour", "hours", "点", "时", "小时",
// "m", "min", "minute", "minutes", "分", "分钟",
// "s", "sec", "second", "seconds", "秒",
// "ms", "millisecond", "milliseconds", "毫秒",
// "us", "µs", "微秒",
// "ns", "纳秒"
func ParseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	sign := int64(1)
	if s[0] == '-' {
		sign = -1
		s = s[1:]
	}
	sss := reDuration.FindAllStringSubmatch(s, -1)
	d := int64(0)
	for _, ss := range sss {
		if len(ss) != 3 {
			continue
		}
		switch strings.ToLower(ss[2]) {
		case "d", "day", "days", "天":
			d += int64(cast.ToFloat64(ss[1]) * float64(24*time.Hour))
		case "h", "hour", "hours", "点", "时", "小时":
			d += int64(cast.ToFloat64(ss[1]) * float64(time.Hour))
		case "m", "min", "minute", "minutes", "分", "分钟":
			d += int64(cast.ToFloat64(ss[1]) * float64(time.Minute))
		case "s", "sec", "second", "seconds", "秒":
			d += int64(cast.ToFloat64(ss[1]) * float64(time.Second))
		case "ms", "millisecond", "milliseconds", "毫秒":
			d += int64(cast.ToFloat64(ss[1]) * float64(time.Millisecond))
		case "us", "µs", "微秒":
			d += int64(cast.ToFloat64(ss[1]) * float64(time.Microsecond))
		case "ns", "纳秒":
			d += int64(cast.ToFloat64(ss[1]) * float64(time.Nanosecond))
		}
	}
	if d == 0 {
		// 默认毫秒
		d += int64(cast.ToFloat64(s) * float64(time.Millisecond))
	}
	return time.Duration(sign * d)
}
