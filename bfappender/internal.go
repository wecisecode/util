package bfappender

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wecisecode/util/merrs"
)

type Option struct {
	RecordEndFlag       []byte        // 记录结束标记，默认空，达到在滚动尺寸时直接截断，设置此值可保持记录完整性
	BackwardFindEndFlag bool          // true 超过滚动尺寸后逆向查找最后一个结束标记，找不到在顺向找，默认 false 超过滚动尺寸后先顺向查找第一个结束标记，找不到再逆向找
	force_end           bool          // 强制结束，always true，找不到结束标记且数据尺寸超过 ScrollBySize 强制截断写入
	FlushAtLeastTime    time.Duration // 写入时间，-1 立即写入，0 默认 1秒
	FlushOverSize       int           // 写入尺寸，-1 立即写入，0 默认 64K
	ScrollByTime        time.Duration // 滚动时间，-1 无限，0 默认 -1
	ScrollBySize        int64         // 滚动尺寸，-1 无限，0 默认 -1
	ScrollKeepTime      time.Duration // 滚动文件保留最长时间，-1 不留，math.MaxInt64 长期保留，0 默认 math.MaxInt64
	ScrollKeepCount     int           // 滚动文件保留最多数量，-1 不留，math.MaxInt 长期保留，0 默认 math.MaxInt
	UseGoBufIOWriter    bool          // 是否使用 go 语言提供的 bufio.Writer，区别不大，默认 false 稍快一点
	ErrorLog            string        // 默认空字符串，接口返回错误信息，设置文件路径名将错误信息写入文件，写入失败返回错误信息
}

// ScrollByTime        time.Duration // 滚动时间，-1 无限，0 默认 -1
// ScrollBySize        int64         // 滚动尺寸，-1 无限，0 默认 -1
// ScrollKeepTime      time.Duration // 滚动文件保留最长时间，-1 不留，math.MaxInt64 长期保留，0 默认 math.MaxInt64
// ScrollKeepCount     int           // 滚动文件保留最多数量，-1 不留，math.MaxInt 长期保留，0 默认 math.MaxInt
var defaultOption = &Option{
	RecordEndFlag:    nil,
	FlushAtLeastTime: time.Duration(1 * time.Second),
	FlushOverSize:    4096 * 16,
	ScrollByTime:     -1,
	ScrollBySize:     -1,
	ScrollKeepTime:   math.MaxInt64,
	ScrollKeepCount:  math.MaxInt,
	UseGoBufIOWriter: false,
	ErrorLog:         "",
}

func (opt *Option) Merge(aos ...*Option) *Option {
	oo := *opt
	for _, a := range aos {
		oo.force_end = true // always true
		if len(a.RecordEndFlag) > 0 {
			oo.RecordEndFlag = a.RecordEndFlag
		}
		oo.BackwardFindEndFlag = a.BackwardFindEndFlag
		if a.FlushAtLeastTime != 0 {
			oo.FlushAtLeastTime = a.FlushAtLeastTime
		}
		if a.FlushOverSize != 0 {
			oo.FlushOverSize = a.FlushOverSize
		}
		if a.ScrollByTime != 0 {
			oo.ScrollByTime = a.ScrollByTime
		}
		if a.ScrollBySize != 0 {
			oo.ScrollBySize = a.ScrollBySize
		}
		if a.ScrollKeepTime != 0 {
			oo.ScrollKeepTime = a.ScrollKeepTime
		}
		if a.ScrollKeepCount != 0 {
			oo.ScrollKeepCount = a.ScrollKeepCount
		}
		if a.ErrorLog != "" {
			oo.ErrorLog = a.ErrorLog
		}
		oo.UseGoBufIOWriter = a.UseGoBufIOWriter
	}
	return &oo
}

type bufferedFileAppender struct {
	option           *Option
	bufmutex         sync.Mutex
	buffer           []byte
	filemutex        sync.Mutex
	filename         string
	file             *os.File
	fsize            int64
	writeBuffer      *bufio.Writer
	fwrcount         int32
	flushTimer       *time.Timer
	flushTimerOpen   bool
	lastFlushTime    time.Time
	lastScrollTime   string
	lastScrollIdx    int
	lastError        error
	archivefilenames []string
	archivefiletime  map[string]time.Time
	referscount      int32
	errorlog         *BufferedFileAppender
	onscrollid       int
	onscroll         map[int]func(string)
}

func mBufferedFileAppender(filename string, option *Option) (bfa *bufferedFileAppender) {
	bfa = &bufferedFileAppender{
		filename: filename,
		option:   option,
		onscroll: map[int]func(string){},
	}
	// 归档文件列表，按最后修改时间+size+name排序
	bfa.archivefilenames, bfa.archivefiletime = archiveFiles(filename)
	fname := filepath.Base(filename)
	ext := filepath.Ext(fname)
	lastfname := ""
	lastftime := time.Time{}
	if len(bfa.archivefilenames) > 0 {
		if filepath.Base(bfa.archivefilenames[len(bfa.archivefilenames)-1]) == fname {
			if len(bfa.archivefilenames) > 1 {
				lastfname = bfa.archivefilenames[len(bfa.archivefilenames)-2]
				lastftime = bfa.archivefiletime[lastfname]
			}
		} else {
			lastfname = bfa.archivefilenames[len(bfa.archivefilenames)-1]
			lastftime = bfa.archivefiletime[lastfname]
		}
	}
	if len(lastfname) > len(fname) {
		idx := filepath.Ext(lastfname[:len(lastfname)-len(ext)])
		if len(idx) > 0 && regexp.MustCompile(`^\.\d+$`).MatchString(idx) {
			bfa.lastScrollIdx, _ = strconv.Atoi(idx[1:])
		}
		tim := filepath.Ext(lastfname[:len(lastfname)-len(ext)-len(idx)])
		if len(tim) > 0 && regexp.MustCompile(`^\.\d+$`).MatchString(tim) {
			bfa.lastScrollTime = tim[1:]
		}
	}
	if bfa.option.ScrollByTime > 0 {
		// 开始写入新内容之前，如果当前时间已经与文件最后修改时间不在同一滚动周期，立即滚动文件
		fi, _ := os.Stat(filename)
		if fi != nil && fi.Size() > 0 {
			lastftime = fi.ModTime()
			bfa.lastScrollTime = timeFixString(lastftime, bfa.option.ScrollByTime)
			if timeFixString(time.Now(), bfa.option.ScrollByTime) != bfa.lastScrollTime {
				// 开始写入新内容之前，如果当前时间已经与滚动时间不一致，立即滚动文件
				err := bfa.scrolling()
				if err != nil {
					bfa.lastError = err
				} else {
					// 当前文件不存在，初始化为当前时间
					bfa.lastScrollTime = timeFixString(time.Now(), bfa.option.ScrollByTime)
				}
			}
		} else {
			// 当前文件不存在，或内容为空，初始化为当前时间
			bfa.lastScrollTime = timeFixString(time.Now(), bfa.option.ScrollByTime)
		}
	}
	if bfa.lastScrollTime == "" {
		// 即使当前配置不需要按时间滚动，也必须初始化上次滚动时间，初始化为当前时间，以备配置变化时的判断需要
		bfa.lastScrollTime = timeFixString(time.Now(), bfa.option.ScrollByTime)
	}
	return
}

// buffer

func (me *bufferedFileAppender) buffersize() (n int) {
	me.bufmutex.Lock()
	n = len(me.buffer)
	me.bufmutex.Unlock()
	return
}

func (me *bufferedFileAppender) putbuffer(record []byte) {
	me.bufmutex.Lock()
	me.buffer = append(me.buffer, record...)
	me.bufmutex.Unlock()
}

func (me *bufferedFileAppender) peekbuffer() (wbs []byte) {
	me.bufmutex.Lock()
	wbs = me.buffer
	me.bufmutex.Unlock()
	return
}

func (me *bufferedFileAppender) shrinkbuffer(size int) {
	me.bufmutex.Lock()
	me.buffer = me.buffer[size:]
	me.bufmutex.Unlock()
}

// 取出后立即清除
func (me *bufferedFileAppender) getbuffer(size int) (wbs []byte) {
	me.bufmutex.Lock()
	wbs = me.buffer
	if size >= 0 && size < len(me.buffer) {
		wbs = me.buffer[:size]
		me.buffer = me.buffer[size:]
	} else {
		me.buffer = nil
	}
	me.bufmutex.Unlock()
	return
}

// 接口

func (me *bufferedFileAppender) Write(opt *Option) error {
	if me.lastError != nil {
		return me.lastError
	}
	n := atomic.AddInt32(&me.fwrcount, 1)
	defer atomic.AddInt32(&me.fwrcount, -1)
	if n > 2 {
		return nil
	}
	me.filemutex.Lock()
	defer me.filemutex.Unlock()
	me.option = me.option.Merge(opt)
	me.lastError = me.writefile()
	return me.errlog(me.lastError)
}

func (me *bufferedFileAppender) Flush() error {
	if me.lastError != nil {
		return me.lastError
	}
	me.filemutex.Lock()
	defer me.filemutex.Unlock()
	_, me.lastError = me.flushfile(-1)
	return me.errlog(me.lastError)
}

func (me *bufferedFileAppender) Close() error {
	me.filemutex.Lock()
	defer me.filemutex.Unlock()
	_, e := me.closefile(-1)
	me.lastError = nil
	e = me.errlog(e)
	if me.errorlog != nil {
		me.errorlog.Close()
		me.errorlog = nil
	}
	return e
}

type fileScrolling struct {
	me *bufferedFileAppender
	id int
}

func (fs *fileScrolling) Close() {
	fs.me.filemutex.Lock()
	defer fs.me.filemutex.Unlock()
	delete(fs.me.onscroll, fs.id)
}

func (me *bufferedFileAppender) OnScroll(f func(string)) *fileScrolling {
	me.filemutex.Lock()
	defer me.filemutex.Unlock()
	me.onscrollid++
	id := me.onscrollid
	me.onscroll[id] = f
	return &fileScrolling{me, id}
}

// error 处理

func (me *bufferedFileAppender) errlog(e error) (err error) {
	if e == nil {
		return nil
	}
	errlogfn := me.option.ErrorLog
	if errlogfn == "" {
		return e
	}
	if me.errorlog != nil && me.errorlog.filename != errlogfn {
		me.errorlog.Close()
		me.errorlog = nil
	}
	if me.errorlog == nil {
		me.errorlog = MBufferedFileAppender(errlogfn, &Option{
			RecordEndFlag:    []byte("\n"),
			FlushAtLeastTime: -1,
			FlushOverSize:    -1,
			ScrollByTime:     -1,
			ScrollBySize:     1024 * 1024 * 5,
			ScrollKeepTime:   math.MaxInt64,
			ScrollKeepCount:  1,
			UseGoBufIOWriter: false,
			ErrorLog:         "",
		})
	}
	data := []byte(fmt.Sprintln(time.Now().Format("2006-01-02 15:04:05.000000"), "[BufferedFileAppender]", e))
	err = me.errorlog.Write(data)
	if err != nil {
		me.errorlog.Close()
		me.errorlog = nil
	}
	return
}

// flush timer

func (me *bufferedFileAppender) activeFlushTimer() {
	// 与 deactiveFlushTimer 在 me.filemutex 锁内执行
	me.lastFlushTime = time.Now()
	if me.flushTimer == nil {
		me.flushTimer = time.AfterFunc(me.option.FlushAtLeastTime, func() {
			me.filemutex.Lock()
			defer me.filemutex.Unlock()
			if me.flushTimerOpen {
				_, me.lastError = me.flushfile(-1)
				me.errlog(me.lastError)
			}
		})
	} else {
		me.flushTimer.Reset(me.option.FlushAtLeastTime)
	}
	me.flushTimerOpen = true
}

func (me *bufferedFileAppender) deactiveFlushTimer() {
	// 与 activeFlushTimer 在 me.filemutex 锁内执行
	me.lastFlushTime = time.Now()
	if me.flushTimer != nil {
		me.flushTimer.Stop()
	}
	me.flushTimerOpen = false
}

// 文件操作

func (me *bufferedFileAppender) closefile(flushbufsize int) (writtencount int, err error) {
	defer func() {
		if me.writeBuffer != nil {
			me.writeBuffer.Flush()
			me.writeBuffer = nil
		}
		if me.file != nil {
			me.file.Close()
			me.file = nil
		}
		me.fsize = 0
	}()
	return me.flushfile(flushbufsize)
}

func (me *bufferedFileAppender) open() (err error) {
	os.MkdirAll(filepath.Dir(me.filename), 0777)
	if me.file, err = os.OpenFile(me.filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666); err != nil {
		return merrs.NewError(err)
	}
	fi, e := me.file.Stat()
	if e != nil {
		me.file.Close()
		me.file = nil
		return merrs.NewError(e)
	}
	me.fsize = fi.Size()
	if me.option.UseGoBufIOWriter && me.option.FlushOverSize > 0 && me.option.FlushAtLeastTime > 0 {
		me.writeBuffer = bufio.NewWriterSize(me.file, me.option.FlushOverSize)
	}
	return
}

func (me *bufferedFileAppender) rename(oldpath, newdir, newfname, newext string) (newpath string, modtime time.Time, err error) {
	if me.lastScrollIdx == 0 {
		newpath = filepath.Join(newdir, fmt.Sprint(newfname, newext))
	} else {
		newpath = filepath.Join(newdir, fmt.Sprint(newfname, ".", me.lastScrollIdx, newext))
	}
	fi, e := os.Stat(newpath)
	if e == nil {
		me.lastScrollIdx += 1
		return me.rename(oldpath, newdir, newfname, newext)
	}
	err = os.Rename(oldpath, newpath)
	if err == nil {
		fi, err = os.Stat(newpath)
		if err == nil {
			modtime = fi.ModTime()
		}
	}
	if err != nil && !os.IsNotExist(err) {
		err = merrs.NewError(err)
		return
	}
	err = nil // 忽略文件不存在错误
	return
}

func (me *bufferedFileAppender) scrolling() (err error) {
	dir, fname := filepath.Split(me.filename)
	ext := filepath.Ext(fname)
	fname = fname[:len(fname)-len(ext)]
	timefix := ""
	if me.option.ScrollByTime > 0 {
		stimefix := timeFixString(time.Now(), me.option.ScrollByTime)
		if stimefix != me.lastScrollTime {
			me.lastScrollTime = stimefix
			if me.option.ScrollBySize > 0 {
				me.lastScrollIdx = 1
			} else {
				me.lastScrollIdx = 0
			}
		} else { // me.option.ScrollBySize > 0 && me.fsize >= me.option.ScrollBySize
			me.lastScrollIdx += 1
		}
		if stimefix != "" {
			timefix = "." + stimefix
		}
	} else { // me.option.ScrollBySize > 0 && me.fsize >= me.option.ScrollBySize
		me.lastScrollIdx += 1
	}
	fi, e := os.Stat(me.filename)
	if e != nil && !os.IsNotExist(e) {
		err = merrs.NewError(e)
		return
	}
	if fi == nil || fi.Size() == 0 {
		return
	}
	archivefilename, archivefiletime, e := me.rename(me.filename, dir, fmt.Sprint(fname, timefix), ext)
	if e != nil {
		err = e
		return
	}
	for _, f := range me.onscroll {
		go func(f func(string)) {
			defer func() {
				x := recover()
				if x != nil {
					me.errlog(merrs.NewError(err))
				}
			}()
			f(archivefilename)
		}(f)
	}
	me.archivefilenames = append(me.archivefilenames, archivefilename)
	me.archivefiletime[archivefilename] = archivefiletime
	if me.option.ScrollKeepTime != 0 {
		for len(me.archivefilenames) > 0 && time.Since(me.archivefiletime[me.archivefilenames[0]]) > me.option.ScrollKeepTime {
			err = os.Remove(me.archivefilenames[0])
			if err != nil && !os.IsNotExist(err) {
				err = merrs.NewError(err)
				return
			}
			err = nil
			delete(me.archivefiletime, me.archivefilenames[0])
			me.archivefilenames = me.archivefilenames[1:]
		}
	}
	if me.option.ScrollKeepCount != 0 {
		for len(me.archivefilenames) > 0 && len(me.archivefilenames) > me.option.ScrollKeepCount {
			err = os.Remove(me.archivefilenames[0])
			if err != nil && !os.IsNotExist(err) {
				err = merrs.NewError(err)
				return
			}
			err = nil
			delete(me.archivefiletime, me.archivefilenames[0])
			me.archivefilenames = me.archivefilenames[1:]
		}
	}
	return
}

// return writtencount int // 已写入文件的数据尺寸或已写入 bufio.Writer 的数据尺寸
// return remainsize int // 针对一个完整数据块，实际尚未写入文件的数据尺寸，包括 bufio.Writer.Buffered()
func (me *bufferedFileAppender) scrollwritefile() (writtencount int, remainsize int, err error) {
	filescrolled := true
	for filescrolled {
		if me.file == nil {
			// 准备写文件
			err = me.open()
			if err != nil {
				return
			}
		}
		var wbn int // 已写入文件的数据尺寸或已写入 bufio.Writer 的数据尺寸
		wbs, force := me.getwritingdata()
		wbn, remainsize, err = me.writechunk(wbs, force)
		writtencount += wbn
		if err != nil {
			return
		}
		// 如果filescrolled == false，没有关闭前写动作，remainsize 不变，
		// 如果filescrolled == true，将进入下一写循环，remainsize 值会被覆盖，失去意义
		wbn, filescrolled, err = me.scrollfile(remainsize, force)
		writtencount += wbn
		if err != nil {
			return
		}
	}
	return
}

func (me *bufferedFileAppender) writefile() (err error) {
	var writtencount int // 已写入文件的数据尺寸或已写入 bufio.Writer 的数据尺寸
	var remainsize int   // 实际尚未写入文件的数据尺寸
	writtencount, remainsize, err = me.scrollwritefile()
	if err != nil {
		return
	}
	if remainsize == 0 {
		// 没有剩余数据，不再需要FlushTimer
		me.deactiveFlushTimer()
	} else if writtencount > 0 || !me.flushTimerOpen {
		// 有剩余数据
		// 有新的写入重新开始计时
		// 之前没有打开FlushTimer，现在打开
		me.activeFlushTimer()
	}
	return
}

func (me *bufferedFileAppender) getwritingdata() (wbs []byte, scrolling bool) {
	wbs = me.peekbuffer()
	if me.option.ScrollBySize > 0 && me.fsize+int64(len(wbs)) > me.option.ScrollBySize {
		scrolling = true
		// 预计写入超长
		if me.fsize >= me.option.ScrollBySize {
			// 文件已经超长，不写了，直接滚动文件
			wbs = wbs[:0]
		} else if len(me.option.RecordEndFlag) == 0 {
			// 未设置结束标记，截断写入
			wbs = wbs[:me.option.ScrollBySize-me.fsize]
		} else if me.option.BackwardFindEndFlag {
			// 先逆向查找结束标记
			endidx := bytes.LastIndex(wbs[:me.option.ScrollBySize-me.fsize], me.option.RecordEndFlag)
			if endidx >= 0 {
				wbs = wbs[:endidx+len(me.option.RecordEndFlag)]
				// 文件长度不够滚动，需要强制写入并切换文件
			} else {
				// 再顺向查找结束标记
				endidx = bytes.Index(wbs[me.option.ScrollBySize-me.fsize:], me.option.RecordEndFlag)
				if endidx >= 0 {
					wbs = wbs[:me.option.ScrollBySize-me.fsize+int64(endidx+len(me.option.RecordEndFlag))]
				} else {
					// 找不到结束标记，强制截断
					wbs = wbs[:me.option.ScrollBySize-me.fsize]
				}
			}
		} else {
			// 先顺向查找结束标记，me.option.ScrollBySize-me.fsize是要截断的位置
			endidx := bytes.Index(wbs[me.option.ScrollBySize-me.fsize:], me.option.RecordEndFlag)
			if endidx >= 0 {
				wbs = wbs[:me.option.ScrollBySize-me.fsize+int64(endidx+len(me.option.RecordEndFlag))]
			} else {
				// 再逆向查找结束标记
				endidx = bytes.LastIndex(wbs[:me.option.ScrollBySize-me.fsize], me.option.RecordEndFlag)
				if endidx >= 0 {
					wbs = wbs[:endidx+len(me.option.RecordEndFlag)]
					// 文件长度不够滚动，需要强制写入并切换文件
				} else {
					// 找不到结束标记，强制截断
					wbs = wbs[:me.option.ScrollBySize-me.fsize]
				}
			}
		}
	}
	return
}

func (me *bufferedFileAppender) writechunk(wbs []byte, force bool) (writtencount int, remainsize int, err error) {
	if len(wbs) > 0 {
		var wbn int
		if me.option.FlushOverSize > 0 && me.option.FlushAtLeastTime > 0 {
			wbn, remainsize, err = me.bufferedWrite(wbs, force)
		} else {
			wbn, err = me.file.Write(wbs)
			remainsize = len(wbs) - wbn
			if err != nil {
				err = merrs.NewError(err)
			}
		}
		writtencount += wbn
		me.fsize += int64(wbn)
		// 清除已写入文件的缓存
		me.shrinkbuffer(wbn)
		if err != nil {
			return
		}
	}
	return
}

func (me *bufferedFileAppender) scrollfile(remainsize int, force bool) (writtencount int, scrolled bool, err error) {
	// 判断是否需要进行文件滚动处理，处理之前必须关闭当前文件
	if force || me.option.ScrollBySize > 0 && me.fsize >= me.option.ScrollBySize ||
		me.option.ScrollByTime > 0 && timeFixString(time.Now(), me.option.ScrollByTime) != me.lastScrollTime {
		var wbn int
		wbn, err = me.closefile(remainsize)
		// 关闭之后，wbs中的数据都已写入文件
		writtencount += wbn
		if err != nil {
			return
		}
		// 文件滚动处理
		err = me.scrolling()
		if err != nil {
			return
		}
		scrolled = true
	}
	return
}

func (me *bufferedFileAppender) bufferedWrite(wbs []byte, force bool) (wbn int, remainsize int, err error) {
	if me.writeBuffer != nil {
		wbn, err = me.writeBuffer.Write(wbs)
		remainsize = me.writeBuffer.Buffered()
		if err != nil {
			err = merrs.NewError(err)
			return
		}
		if force {
			err = me.writeBuffer.Flush()
			remainsize = me.writeBuffer.Buffered()
			if err != nil {
				err = merrs.NewError(err)
			}
			return
		}
	} else {
		if force {
			wbn, err = me.file.Write(wbs)
			remainsize = len(wbs) - wbn
			if err != nil {
				err = merrs.NewError(err)
			}
			return
		}
		// buffersize 对齐处理，按 buffersize 的整倍数写入
		// 第一次写入将已有文件对齐
		// 超时没有新数据定时写入剩余数据
		// 超时写入后重新对齐
		// 需要补齐的尺寸
		paddingsize := me.option.FlushOverSize - int(me.fsize%int64(me.option.FlushOverSize))
		if paddingsize > 0 && len(wbs) < paddingsize {
			// 不够补齐，先暂存
			remainsize = len(wbs)
		} else {
			// 补齐后剩余的零头
			remainsize = (len(wbs) - paddingsize) % me.option.FlushOverSize
			// 对齐尺寸 = 完整尺寸 - 补齐后剩余的零头
			alignsize := len(wbs) - remainsize
			if alignsize > 0 {
				pwbs := wbs[:alignsize]
				wbn, err = me.file.Write(pwbs)
				remainsize = len(wbs) - wbn
				if err != nil {
					err = merrs.NewError(err)
				}
				return
			}
		}
	}
	return
}

func (me *bufferedFileAppender) flushfile(flushbufsize int) (writtencount int, err error) {
	defer func() {
		me.deactiveFlushTimer()
	}()
	if flushbufsize < 0 {
		var wbn int
		var remainsize int
		wbn, remainsize, err = me.scrollwritefile()
		writtencount += wbn
		if err != nil {
			return
		}
		flushbufsize = remainsize
	}
	if me.writeBuffer != nil {
		flushbufsize -= me.writeBuffer.Buffered()
		if flushbufsize < 0 {
			// 不应该小于零
			flushbufsize = 0
		}
	}
	// 取出数据同时清除buffer
	wbs := me.getbuffer(flushbufsize)
	//
	if me.file == nil {
		if len(wbs) == 0 && (me.writeBuffer == nil || me.writeBuffer.Buffered() == 0) {
			return
		}
		e := me.open()
		if e != nil {
			err = merrs.NewError(e)
			return
		}
	}
	if me.writeBuffer != nil {
		nn, e := me.writeBuffer.Write(wbs)
		writtencount += nn
		me.fsize += int64(nn)
		if e != nil {
			err = merrs.NewError(e)
		}
		e = me.writeBuffer.Flush()
		if e != nil {
			err = merrs.NewError(e)
		}
	} else {
		if len(wbs) > 0 {
			nn, e := me.file.Write(wbs)
			writtencount += nn
			me.fsize += int64(nn)
			if e != nil {
				err = merrs.NewError(e)
			}
		}
	}
	return
}
