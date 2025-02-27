package rc_test

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/wecisecode/util/logger"
	"github.com/wecisecode/util/rc"
)

func TestRC(t *testing.T) {
	var wg sync.WaitGroup
	var testnilpanic *testing.T
	rc := rc.NewRoutinesControllerLimit("", 1, 2)
	logger.Info(1)
	wg.Add(1)
	rc.ConcurCall(1, func() {
		defer wg.Done()
		logger.Info(1, "s")
		time.Sleep(3 * time.Second)
		logger.Info(1, "e")
		testnilpanic.Fail()
	})
	logger.Info(2)
	wg.Add(1)
	rc.ConcurCall(1, func() {
		defer wg.Done()
		logger.Info(2, "s")
		time.Sleep(3 * time.Second)
		logger.Info(2, "e")
	})
	logger.Info(3)
	wg.Add(1)
	rc.ConcurCall(1, func() {
		defer wg.Done()
		logger.Info(3, "s")
		time.Sleep(3 * time.Second)
		logger.Info(3, "e")
	})
	logger.Info("S")
	rc.SetConcurrencyLimitCount(3)
	wg.Wait()
	time.Sleep(10 * time.Second)
}

func TestRC0(t *testing.T) {
	var wg sync.WaitGroup
	rc := rc.NewRoutinesController("", 1)
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(i int) {
			rc.ConcurCall(1, func() {
				defer wg.Done()
				logger.Info(i, "start")
				logger.Info(i, "C", rc.ConcurCount(), "Q", rc.QueueCount())
				time.Sleep(3 * time.Second)
				logger.Info(i, "C", rc.ConcurCount(), "Q", rc.QueueCount())
				logger.Info(i, "end")
			})
			logger.Info(i, "pushed")
		}(i)
	}
	wg.Wait()
}

func TestRC1(t *testing.T) {
	var wg sync.WaitGroup
	rc := rc.NewRoutinesControllerLimit("", 1, 3)
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(i int) {
			rc.ConcurCall(1, func() {
				defer wg.Done()
				logger.Info(i, "start")
				logger.Info(i, "C", rc.ConcurCount(), "Q", rc.QueueCount())
				time.Sleep(3 * time.Second)
				logger.Info(i, "C", rc.ConcurCount(), "Q", rc.QueueCount())
				logger.Info(i, "end")
			})
			logger.Info(i, "pushed")
		}(i)
	}
	wg.Wait()
}

func TestRC2(t *testing.T) {
	var wg sync.WaitGroup
	rc := rc.NewRoutinesControllerLimit("", 1, 1)
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(i int) {
			rc.ConcurCall(1, func() {
				defer wg.Done()
				logger.Info(i, "start")
				logger.Info(i, "C", rc.ConcurCount(), "Q", rc.QueueCount())
				time.Sleep(3 * time.Second)
				logger.Info(i, "C", rc.ConcurCount(), "Q", rc.QueueCount())
				logger.Info(i, "end")
				rc.SetConcurQueueLimit(1, 4)
			})
			logger.Info(i, "pushed")
		}(i)
	}
	wg.Wait()
}

func TestRC3(t *testing.T) {
	var wg sync.WaitGroup
	rc := rc.NewRoutinesControllerLimit("", 2, 4)
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(i int) {
			rc.ConcurCall(1, func() {
				defer wg.Done()
				logger.Info(i, "start")
				logger.Info(i, "C", rc.ConcurCount(), "Q", rc.QueueCount())
				time.Sleep(3 * time.Second)
				logger.Info(i, "C", rc.ConcurCount(), "Q", rc.QueueCount())
				logger.Info(i, "end")
				rc.SetConcurQueueLimit(1, 1)
			})
			logger.Info(i, "pushed")
		}(i)
	}
	wg.Wait()
}

func TestRC4(t *testing.T) {
	runtime.GOMAXPROCS(1)
	var wg sync.WaitGroup
	arc := rc.NewRoutinesControllerLimit("", 3, 10)
	st := time.Now()
	for i := 1; i <= 1000000; i++ {
		wg.Add(1)
		n := i
		arc.ConcurCall(1, func() {
			defer wg.Done()
			logger.Info(n, "start")
			logger.Info(n, "C", arc.ConcurCount(), "Q", arc.QueueCount(), "W", arc.RequestWaitingCount())
			// time.Sleep(1 * time.Second)
			logger.Info(n, "end")
		})
		logger.Info(n, "pushed")
	}
	wg.Wait()
	logger.Info(time.Since(st))
}
