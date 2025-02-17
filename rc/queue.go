package rc

import (
	"runtime"
	"sort"
	"sync"
)

type ConcurQueue struct {
	limitcount      int
	mutex           sync.Mutex
	priortityweight []int
	priortityqueue  map[int][]func()
	queuecount      int
	changemutex     sync.Mutex
	change          chan byte
	outqclosed      bool
	outqclosechan   chan bool
	outproccount    map[int]int
	// maxoutweight     int
	last_outprocchan chan func()
}

// 创建并发队列
func NewConcurQueue(LimitCount int) (cq *ConcurQueue) {
	return &ConcurQueue{
		limitcount:      LimitCount,
		priortityweight: []int{},
		priortityqueue:  map[int][]func(){},
		queuecount:      0,
		change:          make(chan byte, 1),
		outqclosed:      true,
		outqclosechan:   nil,
		outproccount:    map[int]int{},
		// maxoutweight:     math.MaxInt,
		last_outprocchan: nil,
	}
}

// 按指定优先权重 weight 插入并发请求 proc
// 超出 LimitCount 时，仍然可以插入，但change会被阻塞，无法返回
func (cq *ConcurQueue) Push(weight int, proc func()) {
	cq.push(weight, proc)
	cq.changenotify()
}

func (cq *ConcurQueue) changenotify() {
	cq.changemutex.Lock()
	if len(cq.change) == cap(cq.change) {
		cq.ChanCapGrowth(&cq.change)
	}
	cq.changemutex.Unlock()
	cq.change <- 1 // 产生新的队列改变标记
}

func (cq *ConcurQueue) ChanCapGrowth(ch *chan byte) {
	qc := *ch
	capqc := cap(qc)
	// if cq.limitcount > 0 && capqc >= cq.limitcount {
	// 	return
	// }
	nqcsize := capqc * 2
	// if cq.limitcount > 0 && nqcsize > cq.limitcount {
	// 	nqcsize = cq.limitcount
	// }
	new_queueChange := make(chan byte, nqcsize)
	done := false
	for !done {
		select {
		case x := <-qc:
			new_queueChange <- x
		default:
			done = true
		}
	}
	// 关闭请求队列，执行器获取队列元素时，会得到空值
	// 自增过程在锁控制中，执行器重新获取队列时，已经是新队列
	*ch = new_queueChange
	close(qc)
	return
}

func (cq *ConcurQueue) push(weight int, proc func()) {
	if weight < 1 {
		weight = 1
	}
	cq.mutex.Lock()
	defer cq.mutex.Unlock()
	procs := cq.priortityqueue[weight]
	if procs == nil {
		cq.priortityweight = append(cq.priortityweight, weight)
		sort.Ints(cq.priortityweight)
	}
	cq.priortityqueue[weight] = append(procs, proc)
	cq.queuecount++
}

func (cq *ConcurQueue) pop() (weight int, proc func(), hasnext bool) {
	cq.mutex.Lock()
	defer cq.mutex.Unlock()
	var procs []func()
	for i := len(cq.priortityweight) - 1; i >= 0; i-- {
		weight = cq.priortityweight[i]
		if cq.outproccount[weight] == weight {
			// 当前 weight 输出已满
			// 继续下一 weight
			continue
		}
		procs = cq.priortityqueue[weight]
		if len(procs) == 0 {
			// 当前 weight 没有更多请求
			// 继续下一 weight
			continue
		}
		break
	}
	if len(procs) > 0 {
		proc = procs[0]
		procs = procs[1:]
		cq.priortityqueue[weight] = procs
		cq.queuecount--
		cq.outproccount[weight]++
		// if cq.outproccount[cq.maxoutweight] == cq.maxoutweight {
		// 	cq.maxoutweight = weight - 1
		// }
		hasnext = cq.outproccount[weight] != weight || cq.priortityweight[0] != weight
	}
	if !hasnext {
		// 没有更多请求 或最小 weight 输出请求数已满
		// 等待有新的
		// 重置输出计数，重新开始按权重输出
		cq.outproccount = map[int]int{}
		// cq.maxoutweight = math.MaxInt
	}
	return
}

// 根据指定并发数生成输出队列
// 队列优先获取权重值更大的请求
// 当权重较大的请求输出数量与权重值一致 或 没有更多请求 时，输出下一权重的请求
// 直至最小权重请求输出数量与权重值一致 或 没有更多请求 时，重新开始计数
func (cq *ConcurQueue) Output(concurcount int) <-chan func() {
	if concurcount <= 0 {
		concurcount = runtime.GOMAXPROCS(0) * 10
	}
	cq.mutex.Lock()
	last_outprocchan := cq.last_outprocchan
	cq.mutex.Unlock()
	if last_outprocchan != nil {
		cq.CloseOutput()
	}
	cq.mutex.Lock()
	defer cq.mutex.Unlock()
	// 确保 CloseOutput 只会关闭最后开启的 Output
	// 二次开启 Output 会自动关闭上一个 Output，并且接续上一个 Output 队列中的内容
	if !cq.outqclosed {
		panic("不能同时生成多个输出队列")
	}
	cq.outqclosed = false
	outqclosechan := make(chan bool, 1)
	cq.outqclosechan = outqclosechan
	//
	outprocchan := make(chan func(), concurcount)
	cq.last_outprocchan = outprocchan
	go func() {
		k := "ConcurQueue.Output"
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
		if last_outprocchan != nil {
			done := false
			for !done {
				select {
				case last_outproc := <-last_outprocchan:
					if last_outproc != nil {
						outprocchan <- last_outproc
					} else {
						done = true
					}
				default:
					done = true
				}
			}
		}
		var proc func()
		// var hasnext bool = false
		var outqclosed = false
		for !outqclosed {
			if cq.QueueCount() == 0 {
				// 没有更多请求
				// 等待队列改变 或 输出关闭信号
				select {
				case <-outqclosechan:
					outqclosed = true
					continue
				case <-cq.change:
				}
			}
			// 获取一个 proc
			_, proc, _ = cq.pop()
			if proc != nil {
				outprocchan <- proc
			}
		}
		close(outprocchan)
	}()
	return outprocchan
}

// 队列中请求数量
func (cq *ConcurQueue) QueueCount() int {
	cq.mutex.Lock()
	defer cq.mutex.Unlock()
	return cq.queuecount
}

func (cq *ConcurQueue) isOutputClosed() bool {
	cq.mutex.Lock()
	defer cq.mutex.Unlock()
	return cq.outqclosed
}

func (cq *ConcurQueue) CloseOutput() {
	cq.mutex.Lock()
	closed := cq.outqclosed
	cq.outqclosed = true
	cq.mutex.Unlock()
	if closed {
		return
	}
	if cq.outqclosechan != nil {
		cq.outqclosechan <- true
	}
}
