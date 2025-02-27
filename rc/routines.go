package rc

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wecisecode/util/merrs"
)

var mutex sync.Mutex
var routinesinfo = map[string]int{}

func RoutinesInfo() string {
	ss := []string{}
	mutex.Lock()
	for k, v := range routinesinfo {
		ss = append(ss, fmt.Sprint(v, ":", k, "\n"))
	}
	mutex.Unlock()
	return strings.Join(ss, "")
}

var ErrQueueClosed = errors.New("Queue has been closed")

type RoutinesController struct {
	concurlimitCount    int
	requestwaitingcount int64
	chqueuelimit        chan byte
	rcname              string
	rcnamecount         int
	srcline             string
	queueMutex          sync.RWMutex
	queueCount          int // 排队中的任务数量 queueCount 和 len(queueCalling) 不一样，queueCount >= len(queueCalling)
	concurCount         int // 执行中的任务数量 queueCount 包含 concurCount，任务从队列中取出执行，不减少 queueCount，执行完成后才会减少 queueCount
	maxqueueCount       int
	maxconcurCount      int
	goroutineCount      int
	concurqueue         *ConcurQueue
	queueCalling        <-chan func()
	onqueuechanged      map[int64]func()
	stop                chan bool
	lastlogtime         time.Time
	lastloginfo         string
	lastactivetime      time.Time
	waitNewJobTimeout   *time.Duration
}

var WarningQueueSize = 64 * 1024
var DefaultWaitNewJobTimeout = 1 * time.Minute

var rcnamecountmutex sync.Mutex
var rcnamecount = map[string]int{}

// ConcurrencyLimitCount 限制并发数，0 默认为 runtime.GOMAXPROCS(0)，-1 不限制
func NewRoutinesController(rcname string, ConcurrencyLimitCount int) (rl *RoutinesController) {
	_, src, call := GetCaller(2, 2)
	if rcname == "" {
		rcname = call
	}
	rcnamecountmutex.Lock()
	n := rcnamecount[rcname]
	n++
	rcnamecount[rcname] = n
	rcnamecountmutex.Unlock()
	return newRoutinesController(rcname, n, src, ConcurrencyLimitCount, 0)
}

// QueuelimitCount 不能低于 ConcurrencyLimitCount
func NewRoutinesControllerLimit(rcname string, ConcurrencyLimitCount int, QueuelimitCount int) (rl *RoutinesController) {
	_, src, call := GetCaller(2, 2)
	if rcname == "" {
		rcname = call
	}
	rcnamecountmutex.Lock()
	n := rcnamecount[rcname]
	n++
	rcnamecount[rcname] = n
	rcnamecountmutex.Unlock()
	return newRoutinesController(rcname, n, src, ConcurrencyLimitCount, QueuelimitCount)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func initLimitCount(limitcount int) int {
	if limitcount == 0 {
		return runtime.GOMAXPROCS(0)
	}
	return limitcount
}

func newRoutinesController(rcname string, rcnamecount int, srcline string, ConcurrencyLimitCount int, QueueLimitCount int) (rl *RoutinesController) {
	rl = &RoutinesController{
		rcname:            rcname,
		rcnamecount:       rcnamecount,
		srcline:           srcline,
		concurqueue:       NewConcurQueue(QueueLimitCount), // make(chan func(), initQueueSize(initLimitCount(ConcurrencyLimitCount))),
		onqueuechanged:    map[int64]func(){},
		lastactivetime:    time.Now(),
		waitNewJobTimeout: &DefaultWaitNewJobTimeout,
		stop:              make(chan bool, 1),
	}
	rl.stop <- false
	rl.SetConcurQueueLimit(ConcurrencyLimitCount, QueueLimitCount)
	return
}

func (rl *RoutinesController) OnQueueChanged(proc func()) int64 {
	rl.queueMutex.Lock()
	defer rl.queueMutex.Unlock()
	id := time.Now().UnixNano()
	for rl.onqueuechanged[id] != nil {
		id++
	}
	rl.onqueuechanged[id] = proc
	go proc()
	return id
}

func (rl *RoutinesController) RemoveQueueChangedHandler(id int64) {
	rl.queueMutex.Lock()
	defer rl.queueMutex.Unlock()
	delete(rl.onqueuechanged, id)
}

func (rl *RoutinesController) onQueueChanged() {
	for _, proc := range rl.onqueuechanged {
		go proc()
	}
}

func (rl *RoutinesController) SetConcurrencyLimitCount(ConcurrencyLimitCount int) {
	rl.SetConcurQueueLimit(ConcurrencyLimitCount, cap(rl.chqueuelimit))
}

func (rl *RoutinesController) SetConcurQueueLimit(ConcurrencyLimitCount, QueueLimitCount int) {
	rl.queueMutex.Lock()
	defer rl.queueMutex.Unlock()
	newLimitCount := initLimitCount(ConcurrencyLimitCount)
	if newLimitCount >= 1 && newLimitCount <= 2 && rl.waitNewJobTimeout == &DefaultWaitNewJobTimeout {
		// 限制不大，避免频繁重启
		// TODO 记录使用时间间隔，自动调整 WaitNewJobTimeout
		WaitNewJobTimeout := time.Duration(math.MaxInt64)
		rl.waitNewJobTimeout = &WaitNewJobTimeout
	}
	// 重新设置 rl.concurlimitCount 后，超出限制的已运行协程会自动停止
	rl.concurlimitCount = newLimitCount
	if QueueLimitCount > 0 {
		if QueueLimitCount < newLimitCount*2 {
			QueueLimitCount = newLimitCount * 2
		}
		// 限制 队列插入
		if rl.chqueuelimit == nil {
			// 新建限制队列
			rl.chqueuelimit = make(chan byte, QueueLimitCount)
		} else if cap(rl.chqueuelimit) != QueueLimitCount {
			// 迁移原来的限制队列
			oldchqueuelimit := rl.chqueuelimit
			rl.chqueuelimit = make(chan byte, QueueLimitCount)
			qmove(oldchqueuelimit, rl.chqueuelimit)
		}
	} else {
		// 不限制 队列插入
		if rl.chqueuelimit != nil {
			// 解除原来的限制
			// relase all
			oldchqueuelimit := rl.chqueuelimit
			rl.chqueuelimit = nil
			qmove(oldchqueuelimit, rl.chqueuelimit)
		}
	}
	rl.prepare_gorun()
}

func (rl *RoutinesController) RequestWaitingCount() int {
	return int(rl.requestwaitingcount)
}

func (rl *RoutinesController) qrequest1() {
	oql := rl.chqueuelimit
	if oql != nil {
		atomic.AddInt64(&rl.requestwaitingcount, 1)
		defer atomic.AddInt64(&rl.requestwaitingcount, -1)
		oql <- 1
	}
}

func (rl *RoutinesController) qrelease1() {
	oql := rl.chqueuelimit
	if oql != nil {
		<-oql
	}
}

func qmove(oldchqueue, newchqueue chan byte) {
	go func() {
		// 已经在队列中的，直接迁入新队列
		if newchqueue != nil {
			for i := 0; i < len(oldchqueue); i++ {
				newchqueue <- 1
			}
		}
		// 正在排队等待，尚未进入队列的
		done := 2 // 2:队列中有数据 1:等待队列中新入数据 0:延迟等待后，队列中无数据
		for done != 0 {
			// 先占新队
			if newchqueue != nil {
				newchqueue <- 1
			}
			select {
			case <-oldchqueue:
				// 再从旧队中清除
				done = 2
			default:
				if done == 2 {
					time.Sleep(1 * time.Second)
					done = 1
				} else {
					// 延迟等待，已经清除干净，结束迁移
					done = 0
				}
				// 退回多占的位置
				if newchqueue != nil {
					<-newchqueue
				}
			}
		}
	}()
}

// 关闭协程控制，队列中未执行完成的协程将全部被取消
func (rl *RoutinesController) Close() {
	rl.CloseWaitDone(false)
}

// var cccc int32

// 关闭协程控制，如果 run_job_in_queue == true，队列中未执行完成的协程将全部进入后台执行
func (rl *RoutinesController) CloseWaitDone(run_job_in_queue bool) {
	stopped := <-rl.stop
	rl.stop <- true
	if stopped {
		// 只执行一次stop
		return
	}
	rl.queueMutex.Lock()
	defer rl.queueMutex.Unlock()
	rl.clearQueue(run_job_in_queue)
}

func (rl *RoutinesController) WaitDone() {
	done := make(chan any)
	oqcid := rl.OnQueueChanged(func() {
		if rl.queueCount == 0 {
			done <- nil
		}
	})
	<-done
	rl.RemoveQueueChangedHandler(oqcid)
}

// 清除队列中未执行完成的任务
func (rl *RoutinesController) ClearQueue() {
	rl.queueMutex.Lock()
	defer rl.queueMutex.Unlock()
	rl.clearQueue(false)
}

func (rl *RoutinesController) clearQueue(run_job_in_queue bool) {
	rl.concurqueue.CloseOutput()
	qc := rl.queueCalling
	rl.queueCalling = nil
	if qc != nil {
		// 清除队列中的ConcurCall请求
		// n := atomic.AddInt32(&cccc, 1)
		// Logger.Info(n, rl.rcname+" "+rl.srcline, "close")
		done := false
		for !done {
			select {
			case f := <-qc:
				if run_job_in_queue {
					go rl.run_job_1(f)
				} else {
					// 直接清空
					rl.queueCount--
					rl.qrelease1()
				}
				if len(qc) == 0 {
					done = true
				}
				// Logger.Info(n, rl.rcname+" "+rl.srcline, "close", len(qc), f)
			default:
				done = true
			}
		}
		// Logger.Info(n, rl.rcname+" "+rl.srcline, "close done")
	}
}

// 锁内执行，预启动处理协程
func (rl *RoutinesController) prepare_gorun() {
	if rl.concurlimitCount > 0 {
		for rl.goroutineCount < rl.concurlimitCount {
			rl.goroutineCount++
			go rl.run()
		}
	}
}

// 锁内执行，启动新协程
func (rl *RoutinesController) gorun() {
	if rl.concurlimitCount < 0 || rl.goroutineCount < rl.concurlimitCount {
		if rl.goroutineCount < rl.queueCount*2+1 {
			rl.goroutineCount++
			go rl.run()
		}
	}
}

func (rl *RoutinesController) run() {
	defer func() {
		rl.queueMutex.Lock()
		rl.goroutineCount--
		if rl.goroutineCount == 0 {
			if rl.queueCount > 0 {
				// 全部处理过程退出，发现新作业，重新启动协程
				rl.gorun()
			} else {
				rl.concurqueue.CloseOutput()
				rl.queueCalling = nil
			}
		}
		rl.queueMutex.Unlock()
	}()
	k := fmt.Sprint(rl.rcname+" "+rl.srcline, " Limit ", rl.concurlimitCount)
	mutex.Lock()
	routinesinfo[k] = routinesinfo[k] + 1
	mutex.Unlock()
	defer func() {
		mutex.Lock()
		routinesinfo[k] = routinesinfo[k] - 1
		if routinesinfo[k] == 0 {
			delete(routinesinfo, k)
		}
		mutex.Unlock()
	}()
	wait_new_job := time.NewTimer(math.MaxInt64)
	defer wait_new_job.Stop()
	for rl.run_job(wait_new_job) {
		// timesharing
		// 队列有剩余空间，还有请求排队等待，分时给队列请求协程执行机会
		if cap(rl.chqueuelimit) > rl.queueCount && rl.requestwaitingcount > 0 {
			time.Sleep(time.Duration(rl.requestwaitingcount) * time.Microsecond)
		}
	}
}

func (rl *RoutinesController) run_job(wait_new_job *time.Timer) bool {
	stopped := <-rl.stop
	rl.stop <- stopped
	if stopped {
		return false
	}
	rl.queueMutex.Lock()
	if rl.queueCalling == nil {
		rl.queueCalling = rl.concurqueue.Output(rl.concurlimitCount)
	}
	qc := rl.queueCalling
	rl.queueMutex.Unlock()
	if qc == nil {
		return false
	}
	wait_new_job.Stop()
	wait_new_job.Reset(*rl.waitNewJobTimeout)
	select {
	case f := <-qc:
		// 等到新作业
		rl.run_job_1(f)
		return true
	case <-wait_new_job.C:
		// 	// 没等到新作业
		return false
	}
}

func (rl *RoutinesController) run_job_1(f func()) {
	if f == nil {
		// close chan
		return
	}
	rl.queueMutex.Lock()
	rl.concurCount++
	if rl.concurCount > rl.maxconcurCount {
		rl.maxconcurCount = rl.concurCount
	}
	rl.lastactivetime = time.Now()
	rl.queueMutex.Unlock()
	defer func() {
		rl.qrelease1()
		rl.queueMutex.Lock()
		rl.loginfo()
		rl.queueCount--
		rl.concurCount--
		rl.onQueueChanged()
		rl.lastactivetime = time.Now()
		rl.queueMutex.Unlock()
	}()
	defer func() {
		x := recover()
		if x != nil {
			if Logger != nil {
				Logger.Error(merrs.NewError(x))
			}
		}
	}()
	f()
}

// 队列中请求数
func (rl *RoutinesController) QueueCount() (n int) {
	rl.queueMutex.RLock()
	n = rl.queueCount
	rl.queueMutex.RUnlock()
	return
}

// 实际的并发计数
func (rl *RoutinesController) ConcurCount() (n int) {
	rl.queueMutex.RLock()
	n = rl.concurCount
	rl.queueMutex.RUnlock()
	return
}

// 并发限制数
func (rl *RoutinesController) LimitCount() (n int) {
	rl.queueMutex.RLock()
	n = rl.concurlimitCount
	rl.queueMutex.RUnlock()
	return
}

// 最后一次激活时间
func (rl *RoutinesController) LastActiveTime() time.Time {
	rl.queueMutex.RLock()
	t := rl.lastactivetime
	rl.queueMutex.RUnlock()
	return t
}

// 锁内调用，日志输出
func (rl *RoutinesController) loginfo() {
	if Logger == nil {
		return
	}
	if time.Since(rl.lastlogtime) < 1*time.Minute {
		return
	}
	defer func() { rl.lastlogtime = time.Now() }()
	s := fmt.Sprint(rl.srcline, " ", rl.rcname, "[", rl.rcnamecount, "]", " QueueSize C/M ", rl.queueCount, "/", rl.maxqueueCount,
		" ConcurCount C/M/L ", rl.concurCount, "/", rl.maxconcurCount, "/", rl.concurlimitCount,
	)
	if rl.lastloginfo != s {
		if rl.queueCount < WarningQueueSize {
			Logger.Info(s)
		} else {
			Logger.Warn(s)
		}
		rl.lastloginfo = s
	}
}

func (rl *RoutinesController) push(priortity int, f func()) {
	rl.gorun()
	rl.queueCount++
	if rl.queueCount > rl.maxqueueCount {
		rl.maxqueueCount = rl.queueCount
	}
	rl.lastactivetime = time.Now()
	rl.concurqueue.Push(priortity, f)
	rl.loginfo()
	rl.onQueueChanged()
}

// 队列长度自动增加，达到最大队列长度 MaxQueueSize 后，入列过程将被阻塞
//
//	所有优先级不为 1 的情况，都在最外层代码以 “priortity :=” 的形式设置优先级，方便查找
func (rl *RoutinesController) ConcurCall(priortity int, f func()) error {
	rl.qrequest1()
	// chqueuelimit 阻塞期间 stopped 可能会改变
	stopped := <-rl.stop
	rl.stop <- stopped
	if stopped {
		rl.qrelease1()
		return ErrQueueClosed
	}
	rl.queueMutex.Lock()
	defer rl.queueMutex.Unlock()
	rl.push(priortity, f)
	return nil
}

func (rl *RoutinesController) CallLast2Only(f func()) error {
	rl.qrequest1()
	// chqueuelimit 阻塞期间 stopped 可能会改变
	stopped := <-rl.stop
	rl.stop <- stopped
	if stopped {
		// 取消 push，清除站位标记
		rl.qrelease1()
		return ErrQueueClosed
	}
	rl.queueMutex.Lock()
	defer rl.queueMutex.Unlock()
	if rl.queueCount >= 2 {
		// 取消 push，清除站位标记
		rl.qrelease1()
		return nil
	}
	rl.push(1, f)
	return nil
}
