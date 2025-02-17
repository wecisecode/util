package cast

import (
	"fmt"
	"reflect"
	"time"

	"github.com/wecisecode/util/cast/dateparse"
	"github.com/wecisecode/util/merrs"
)

// local zone utc+8
// -292277022399-01-01 00:00:00.000000000         -9223372028715350743 + 0
// 292277026596-12-04 23:35:51.000000000          -9223372036854775808 + 0
// 292277026596-12-04 23:30:07.000000000          9223372036854775807  + 0
// 292277026854-11-09 07:00:15.999999999          -9223372028715350744 + 999999999
// 0001-01-01 00:00:00.000000000    -62135596800s , -6795364578871345152ns    time.Time{}.Unix() , time.Time{}.UnixNano()
// 1754-08-31 06:49:24.128654848    0 + -6795364578871345152
// 1677-09-21 08:18:26.145224192    0 + -9223372036854775808    math.MinInt64
// 2262-04-12 07:47:16.854775807    0 + 9223372036854775807     math.MaxInt64
// 1970-01-01 08:00:00.000000000    0 + 0
// 1970-04-27 01:46:40.000000000    0 + 1e16
// 1973-03-03 17:46:40.000000000    0 + 1e17
// 2001-09-09 09:46:40.000000000    0 + 1e18
// 2033-05-18 11:33:20.000000000    0 + 2e18
// 2065-01-24 13:20:00.000000000    0 + 3e18
// 2096-10-02 15:06:40.000000000    0 + 4e18
// 2128-06-11 16:53:20.000000000    0 + 5e18
// 2160-02-18 18:40:00.000000000    0 + 6e18
// 2191-10-27 20:26:40.000000000    0 + 7e18
// 2223-07-06 22:13:20.000000000    0 + 8e18
// 2255-03-15 00:00:00.000000000    0 + 9e18

// 默认时间范围 北京时间 1973-03-03 17:46:40 ~ 2255-03-15 00:00:00
func TimeAssume(ctime int64) (time.Time, error) {
	if ctime >= 1e8 && ctime <= 9e9 {
		// 秒值
		return time.Unix(0, ctime*1e9), nil
	} else if ctime >= 1e11 && ctime <= 9e12 {
		// 毫秒值
		return time.Unix(0, ctime*1e6), nil
	} else if ctime >= 1e14 && ctime <= 9e15 {
		// 微妙值
		return time.Unix(0, ctime*1e3), nil
	} else if ctime >= 1e17 && ctime <= 9e18 {
		// 纳秒值
		return time.Unix(0, ctime), nil
	} else {
		return time.Time{}, merrs.NewError(fmt.Sprintf("unknown time format %d", ctime))
	}
}

// 默认时间范围 北京时间 1973-03-03 17:46:40 ~ 2255-03-15 00:00:00
func ToUnixNano(ctime int64) (int64, error) {
	t, e := TimeAssume(ctime)
	if e != nil {
		return 0, e
	}
	return t.UnixNano(), nil
}

func IntToTime[I int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64](v I) (t time.Time, err error) {
	return TimeAssume(int64(v))
}

func IntsToTimes[I int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64](vs []I) (ts []time.Time, err error) {
	for _, av := range vs {
		t, e := TimeAssume(int64(av))
		if e != nil {
			return nil, e
		}
		ts = append(ts, t)
	}
	return
}

func ToDateTimeE(v interface{}, layouts ...string) (time.Time, error) {
	var t time.Time
	var e error
	switch vv := v.(type) {
	case string:
		for _, layout := range layouts {
			t, e = time.Parse(layout, vv)
			if e == nil {
				return t, nil
			}
		}
		t, e = dateparse.ParseLocal(vv)
	case uint32:
		t, e = IntToTime(int64(vv))
	case uint64:
		t, e = TimeAssume(int64(vv))
	case int32:
		t, e = TimeAssume(int64(vv))
	case int64:
		t, e = TimeAssume(int64(vv))
	case float32:
		t, e = TimeAssume(int64(vv))
	case float64:
		t, e = TimeAssume(int64(vv))
	case int:
		t, e = TimeAssume(int64(vv))
	case time.Time:
		t = vv
	default:
		return time.Time{}, merrs.NewError(fmt.Sprintf("unknown timestamp format :%v, type: %v", v, reflect.TypeOf(v)))
	}
	if e != nil {
		return time.Time{}, e
	}
	return t, nil
}

func ToDateTimes(v interface{}) (ts []time.Time, err error) {
	var t time.Time
	var e error
	switch vv := v.(type) {
	case []string:
		for _, av := range vv {
			t, e = dateparse.ParseLocal(av)
			if e != nil {
				return nil, e
			}
			ts = append(ts, t)
		}
	case []uint32:
		ats, e := IntsToTimes(vv)
		if e != nil {
			return nil, e
		}
		ts = append(ts, ats...)
	case []uint64:
		ats, e := IntsToTimes(vv)
		if e != nil {
			return nil, e
		}
		ts = append(ts, ats...)
	case []int32:
		ats, e := IntsToTimes(vv)
		if e != nil {
			return nil, e
		}
		ts = append(ts, ats...)
	case []int64:
		ats, e := IntsToTimes(vv)
		if e != nil {
			return nil, e
		}
		ts = append(ts, ats...)
	case []float32:
		ats, e := IntsToTimes(vv)
		if e != nil {
			return nil, e
		}
		ts = append(ts, ats...)
	case []float64:
		ats, e := IntsToTimes(vv)
		if e != nil {
			return nil, e
		}
		ts = append(ts, ats...)
	case []int:
		ats, e := IntsToTimes(vv)
		if e != nil {
			return nil, e
		}
		ts = append(ts, ats...)
	case []time.Time:
		for _, av := range vv {
			ts = append(ts, av)
		}
	default:
		t, e = ToDateTimeE(v)
		if e != nil {
			return nil, e
		}
		ts = append(ts, t)
	}
	return ts, nil
}

func ToUnixNanos(v interface{}) (ts []int64, err error) {
	var t time.Time
	var e error
	switch vv := v.(type) {
	case []string:
		for _, av := range vv {
			t, e = dateparse.ParseLocal(av)
			if e != nil {
				return nil, e
			}
			ts = append(ts, t.UnixNano())
		}
	case []uint32:
		for _, av := range vv {
			t, e := TimeAssume(int64(av))
			if e != nil {
				return nil, e
			}
			ts = append(ts, t.UnixNano())
		}
	case []uint64:
		for _, av := range vv {
			t, e := TimeAssume(int64(av))
			if e != nil {
				return nil, e
			}
			ts = append(ts, t.UnixNano())
		}
	case []int32:
		for _, av := range vv {
			t, e := TimeAssume(int64(av))
			if e != nil {
				return nil, e
			}
			ts = append(ts, t.UnixNano())
		}
	case []int64:
		for _, av := range vv {
			t, e := TimeAssume(int64(av))
			if e != nil {
				return nil, e
			}
			ts = append(ts, t.UnixNano())
		}
	case []float32:
		for _, av := range vv {
			t, e := TimeAssume(int64(av))
			if e != nil {
				return nil, e
			}
			ts = append(ts, t.UnixNano())
		}
	case []float64:
		for _, av := range vv {
			t, e := TimeAssume(int64(av))
			if e != nil {
				return nil, e
			}
			ts = append(ts, t.UnixNano())
		}
	case []int:
		for _, av := range vv {
			t, e := TimeAssume(int64(av))
			if e != nil {
				return nil, e
			}
			ts = append(ts, t.UnixNano())
		}
	case []time.Time:
		for _, av := range vv {
			ts = append(ts, av.UnixNano())
		}
	default:
		t, e = ToDateTimeE(v)
		if e != nil {
			return nil, e
		}
		ts = append(ts, t.UnixNano())
	}
	return ts, nil
}
