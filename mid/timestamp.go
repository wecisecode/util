package mid

import (
	"sync"
	"time"
)

// 确保每次取出的本地纳秒时间值不同，主要用于不支持NanoSecond的操作系统
// time.Time 表示的时间精度比NanoSecond更小，所以有可能 t.Equal(lvt) == false 但是 t.UnixNano() == lvt.UnixNano()
type TimeStamp time.Time

var lvt_mutex sync.Mutex
var lvtns = time.Now().UnixNano()

// 确保每次取出的本地纳秒时间值不同，主要用于不支持NanoSecond的操作系统
// time.Time 表示的时间精度比NanoSecond更小，所以有可能 t.Equal(lvt) == false 但是 t.UnixNano() == lvt.UnixNano()
func MTimeStamp() TimeStamp {
	lvt_mutex.Lock()
	t := time.Now().UnixNano()
	if t <= lvtns {
		t = lvtns + 1
	}
	lvtns = t
	lvt_mutex.Unlock()
	return MTimeStampV(t)
}

func MTimeStampV(vtun int64) TimeStamp {
	return TimeStamp(time.Unix(0, vtun))
}

func (t TimeStamp) UnixNano() int64 {
	return time.Time(t).UnixNano()
}

func (t TimeStamp) Time() time.Time {
	return time.Time(t)
}

// 确保每次取出的本地纳秒时间值不同，主要用于不支持NanoSecond的操作系统
func UnixNano() int64 {
	return MTimeStamp().UnixNano()
}
