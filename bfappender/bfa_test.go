package bfappender_test

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/wecisecode/util/bfappender"
	"github.com/wecisecode/util/cfg"
	"github.com/wecisecode/util/mfmt"
)

func init() {
	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGTERM)
	go func() {
		s := <-exitChan
		fmt.Println("收到退出信号", s)
	}()
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var kvm = cfg.MConfig()
var size = kvm.GetInt("size", 100)
var count = kvm.GetInt("count", 1000)
var loop = kvm.GetInt("loop", 10)
var parallel = kvm.GetInt("parallel", 1000)

func run(name string, bfa *bfappender.BufferedFileAppender) (bytesize int64) {
	for n := 1; n <= count; n++ {
		bs := []byte(fmt.Sprint(time.Now().Format("2006-01-02 15:04:05.000000"), " ", name, ".", n, strings.Repeat(".", size+rand.Intn(20)-10)+"!\r\n"))
		e := bfa.Write(bs)
		if e != nil {
			fmt.Println(e)
		}
		bytesize += int64(len(bs))
	}
	return
}

func parallelrun(name string, bfa *bfappender.BufferedFileAppender) (writtencount int64) {
	st := time.Now()
	bytes := int64(0)
	for i := 1; i <= loop; i++ {
		wg := sync.WaitGroup{}
		for n := 1; n <= parallel; n++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				bn := run(fmt.Sprint(name, " ", i, ".", n), bfa)
				atomic.AddInt64(&bytes, bn)
			}(n)
		}
		wg.Wait()
		ut := time.Since(st)
		fmt.Println(name,
			"并发", parallel,
			"记录", fmt.Sprint(i*parallel*count, "/", bytes, "/", mfmt.BytesSize(bytes)),
			"耗时", ut,
			"平均", fmt.Sprint(ut/time.Duration(bytes)*1024*1024*1024, "/GB"),
		)
	}
	return bytes
}

var filesizestart int64
var filesizesum int64

func Summary(filename string) (filechangecount int64) {
	fi, _ := os.Stat(filename)
	filesizeend := fi.Size()
	filechangecount = filesizesum + filesizeend - filesizestart
	fmt.Println("共写入文件", filechangecount)
	return
}

func Init(filename string, opt *bfappender.Option) *bfappender.BufferedFileAppender {
	st := time.Now()
	bfa := bfappender.MBufferedFileAppender(filename, opt)
	fi, _ := os.Stat(filename)
	if fi != nil {
		filesizestart = fi.Size()
	} else {
		filesizestart = 0
	}
	filesizesum = 0
	bfa.OnScroll(func(archivedfilename string) {
		fi, _ := os.Stat(archivedfilename)
		filesizesum += fi.Size()
		if fi.Size() > opt.ScrollBySize+1024 {
			fmt.Println(archivedfilename, fi.Size(), fi.ModTime())
		}
	})
	bs := []byte(fmt.Sprint(time.Now().Format("2006-01-02 15:04:05.000000"), " ", "Initialize", ".", 0, ".", 0, strings.Repeat(".", size+rand.Intn(20)-10)+"!\r\n"))
	filesizestart += int64(len(bs))
	e := bfa.Write(bs)
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println("Initialize:", time.Since(st))
	return bfa
}

func TestBufferedFileAppenderSimplyWrite(t *testing.T) {
	opt := &bfappender.Option{RecordEndFlag: []byte("\r\n"), FlushOverSize: -1, FlushAtLeastTime: -1, ScrollBySize: 1024 * 1024 * 5, ScrollByTime: 60 * time.Minute}
	bfa := Init("_test/test.Simpl.txt", opt)
	writtencount := parallelrun("SimplyWrite:", bfa)
	bfa.Close()
	filechangecount := Summary("_test/test.Simpl.txt")
	if writtencount != filechangecount {
		t.Error("writtencount != filechangecount")
	}
}

func TestBufferedFileAppenderSizeAlign(t *testing.T) {
	opt := &bfappender.Option{RecordEndFlag: []byte("\r\n"), ScrollBySize: 1024 * 1024 * 5, ScrollByTime: 60 * time.Minute}
	bfa := Init("_test/test.Align.txt", opt)
	writtencount := parallelrun("SizeAlign:  ", bfa)
	bfa.Close()
	filechangecount := Summary("_test/test.Align.txt")
	if writtencount != filechangecount {
		t.Error("writtencount != filechangecount")
	}
}

func TestBufferedFileAppenderUseGoBufIOWriter(t *testing.T) {
	opt := &bfappender.Option{RecordEndFlag: []byte("\r\n"), ScrollBySize: 1024 * 1024 * 5, ScrollByTime: 60 * time.Minute, UseGoBufIOWriter: true}
	bfa := Init("_test/test.Bufio.txt", opt)
	writtencount := parallelrun("BufIOWriter:", bfa)
	bfa.Close()
	filechangecount := Summary("_test/test.Bufio.txt")
	if writtencount != filechangecount {
		t.Error("writtencount != filechangecount")
	}
}
