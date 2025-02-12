package bfappender

import (
	"sync"
)

type BufferedFileAppender struct {
	filename string
	option   *Option
	bfa      *bufferedFileAppender
}

var bfasmu = sync.Mutex{}
var bfas = map[string]*bufferedFileAppender{}

// 同一 filename，option 的最后设置值在每次实际执行写文件操作前生效
func MBufferedFileAppender(filename string, opt ...*Option) (bfa *BufferedFileAppender) {
	bfa = &BufferedFileAppender{
		filename: filename,
		option:   defaultOption.Merge(opt...),
	}
	return
}

func (me *BufferedFileAppender) WithOption(opt ...*Option) (bfa *BufferedFileAppender) {
	me.option = me.option.Merge(opt...)
	return me
}

func (me *BufferedFileAppender) OnScroll(f func(string)) *fileScrolling {
	bfa := me.bfa
	if bfa == nil {
		bfasmu.Lock()
		if me.bfa == nil {
			me.bfa = bfas[me.filename]
			if me.bfa == nil {
				me.bfa = mBufferedFileAppender(me.filename, me.option)
				bfas[me.filename] = me.bfa
			}
			me.bfa.referscount++
		}
		bfa = me.bfa
		bfasmu.Unlock()
	}
	return bfa.OnScroll(f)
}

func (me *BufferedFileAppender) Write(record []byte) error {
	bfa := me.bfa
	if bfa == nil {
		bfasmu.Lock()
		if me.bfa == nil {
			me.bfa = bfas[me.filename]
			if me.bfa == nil {
				me.bfa = mBufferedFileAppender(me.filename, me.option)
				bfas[me.filename] = me.bfa
			}
			me.bfa.referscount++
		}
		bfa = me.bfa
		bfasmu.Unlock()
	}
	if bfa.lastError != nil {
		return bfa.lastError
	}
	bfa.putbuffer(record)
	if me.option.FlushOverSize > 0 {
		go bfa.Write(me.option)
	} else {
		bfa.Write(me.option)
	}
	return bfa.lastError
}

func (me *BufferedFileAppender) Flush() error {
	bfa := me.bfa
	if bfa == nil {
		return nil
	}
	if bfa.lastError != nil {
		return bfa.lastError
	}
	return bfa.Flush()
}

func (me *BufferedFileAppender) Close() error {
	bfa := me.bfa
	if bfa == nil {
		return nil
	}
	defer func() {
		bfasmu.Lock()
		if me.bfa != nil {
			me.bfa.referscount--
			if me.bfa.referscount == 0 {
				delete(bfas, me.filename)
			}
			me.bfa = nil
		}
		bfasmu.Unlock()
	}()
	return bfa.Close()
}
