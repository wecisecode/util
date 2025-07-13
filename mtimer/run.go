package mtimer

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wecisecode/util/cast"
	"github.com/wecisecode/util/sortedmap"
)

type Option struct {
	Timeout time.Duration
}

var tidx = int32(0)
var timeoutidx = map[int32]int64{}
var timeoutxid = map[int64]int32{}
var timeoutqueue = sortedmap.NewTreapMap()
var mutex sync.Mutex
var timer *time.Timer

func ClearAllTimeout() {
	mutex.Lock()
	defer mutex.Unlock()
	timeoutqueue.Clear()
	timeoutidx = map[int32]int64{}
	timeoutxid = map[int64]int32{}
	if timer != nil {
		timer.Reset(time.Duration(1)) // 激活timer.C，结束timer处理
	}
}

func ClearTimeout(tid int32) {
	mutex.Lock()
	defer mutex.Unlock()
	ttime := timeoutidx[tidx]
	timeoutqueue.Delete(ttime)
	delete(timeoutxid, ttime)
	delete(timeoutidx, tidx)
	if timeoutqueue.Len() == 0 && timer != nil {
		timer.Reset(time.Duration(1)) // 激活timer.C，结束timer处理
	}
}

func checkTimeoutQueue() bool {
	mutex.Lock()
	defer mutex.Unlock()
	for {
		if timeoutqueue.Len() == 0 {
			timer = nil
			return false
		} else {
			firstitem := timeoutqueue.FirstItem()
			tfirst := firstitem.Key.(int64)
			tnow := time.Now().UnixNano()
			if tfirst <= tnow {
				ff := firstitem.Value.(func())
				go ff()
				tidx := timeoutxid[tfirst]
				timeoutqueue.Delete(tfirst)
				delete(timeoutxid, tfirst)
				delete(timeoutidx, tidx)
			} else {
				timer.Reset(time.Duration(tfirst - tnow))
				return true
			}
		}
	}
}

// TODO 替代 time.After
func SetTimeout(td time.Duration, f func()) int32 {
	if td <= 0 {
		go f()
		return 0
	}
	tnow := time.Now().UnixNano()
	tnano := tnow + td.Nanoseconds()
	mutex.Lock()
	defer mutex.Unlock()
	for timeoutqueue.Has(tnano) {
		tnano++
	}
	timeoutqueue.Put(tnano, f)
	tidx++
	timeoutidx[tidx] = tnano
	timeoutxid[tnano] = tidx
	if timer == nil {
		timer = time.NewTimer(math.MaxInt64)
		timer.Stop()
		go func() {
			run := true
			for run {
				select {
				case <-timer.C:
					run = checkTimeoutQueue()
				}
			}
		}()
	}
	tfirst := timeoutqueue.FirstItem().Key.(int64)
	if tnano == tfirst {
		timer.Reset(time.Duration(tfirst - tnow))
	}
	return tidx
}

// TODO 替代 time.After
func Run(f func(), opt *Option) (err error) {
	var tid int32
	cherr := make(chan error, 1)

	if opt.Timeout > 0 {
		_, file, line, _ := runtime.Caller(1)
		file = file[strings.LastIndex(file, string(os.PathSeparator))+1:]
		tid = SetTimeout(opt.Timeout, func() {
			cherr <- fmt.Errorf(fmt.Sprint("background routine running timeout (", file, ":", line, ")"))
		})
	}

	go func() {
		defer func() {
			if tid != 0 {
				ClearTimeout(tid)
			}
			x := recover()
			if x != nil {
				if e, ok := x.(error); ok {
					cherr <- e
				} else {
					cherr <- fmt.Errorf(cast.ToString(x))
				}
			} else {
				cherr <- nil
			}
		}()
		f()
	}()

	return <-cherr
}
