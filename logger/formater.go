package logger

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
)

func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

var pid = os.Getpid()

type FmtArgs struct {
	Year   int
	Month  int
	Day    int
	Hour   int
	Min    int
	Sec    int
	Ns     int
	Level  string
	Module string
	File   string
	Line   int
	Pc     uintptr
	Fmtf   string
	Args   []interface{}
}

type formater struct {
	idx    byte
	name   string
	fnl    int
	format func(buf *[]byte, fa *FmtArgs)
}

type formaters map[byte][]*formater

func (fs formaters) copy() (nfs formaters) {
	nfs = formaters{}
	for k, v := range fs {
		nfs[k] = append(nfs[k], v...)
	}
	return nfs
}

func (fs formaters) newformater(name string, format func(buf *[]byte, fa *FmtArgs)) (fmt *formater) {
	fmt = &formater{
		idx:    name[0],
		name:   name,
		fnl:    len(name),
		format: format,
	}
	// 同名格式，后来的覆盖已有的
	for i, ofmt := range fs[fmt.idx] {
		if ofmt.name == name {
			fs[fmt.idx][i] = fmt
			return
		}
	}
	// 追加新格式
	fs[fmt.idx] = append(fs[fmt.idx], fmt)
	sort.Slice(fs[fmt.idx], func(i, j int) bool {
		// 名字长的排前面，以保证匹配的准确性
		if fs[fmt.idx][i].fnl == fs[fmt.idx][j].fnl {
			return fs[fmt.idx][i].name < fs[fmt.idx][j].name
		}
		return fs[fmt.idx][i].fnl > fs[fmt.idx][j].fnl
	})
	return fmt
}

var default_formaters = formaters{}
var formaters_list = []*formater{
	default_formaters.newformater("msg", func(buf *[]byte, fa *FmtArgs) {
		if fa.Fmtf == "" {
			for i, arg := range fa.Args {
				if i > 0 {
					*buf = append(*buf, " "...)
				}
				*buf = append(*buf, fmt.Sprint(arg)...)
			}
		} else if len(fa.Args) == 0 {
			*buf = append(*buf, fa.Fmtf...)
		} else {
			*buf = append(*buf, fmt.Sprintf(fa.Fmtf, fa.Args...)...)
		}
	}),
	default_formaters.newformater("module", func(buf *[]byte, fa *FmtArgs) {
		*buf = append(*buf, fa.Module...)
	}),
	default_formaters.newformater("file", func(buf *[]byte, fa *FmtArgs) {
		*buf = append(*buf, fa.File...)
	}),
	default_formaters.newformater("line", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Line, -1)
	}),
	default_formaters.newformater("func", func(buf *[]byte, fa *FmtArgs) {
		fn := runtime.FuncForPC(fa.Pc).Name()
		*buf = append(*buf, fn...)
	}),
	default_formaters.newformater("level", func(buf *[]byte, fa *FmtArgs) {
		*buf = append(*buf, fa.Level...)
	}),
	default_formaters.newformater("pid", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, pid, -1)
	}),
	default_formaters.newformater("yyyy", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Year, 4)
	}),
	default_formaters.newformater("MM", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Month, 2)
	}),
	default_formaters.newformater("dd", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Day, 2)
	}),
	default_formaters.newformater("HH", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Hour, 2)
	}),
	default_formaters.newformater("mm", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Min, 2)
	}),
	default_formaters.newformater("ss", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Sec, 2)
	}),
	default_formaters.newformater("SSSSSS", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Ns/1e3, 6)
	}),
	default_formaters.newformater("SSSSS", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Ns/1e4, 5)
	}),
	default_formaters.newformater("SSSS", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Ns/1e5, 4)
	}),
	default_formaters.newformater("SSS", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Ns/1e6, 3)
	}),
	default_formaters.newformater("SS", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Ns/1e7, 2)
	}),
	default_formaters.newformater("S", func(buf *[]byte, fa *FmtArgs) {
		itoa(buf, fa.Ns/1e8, 1)
	}),
}

type Formater struct {
	fmtsmu sync.Mutex
	fmts   formaters
	format string
	eol    string
	bufmu  sync.Mutex
	bufs   [][]byte
}

func MFormater(format string, eol string) *Formater {
	return &Formater{fmts: default_formaters.copy(), format: format, eol: eol, bufs: make([][]byte, 1)}
}

// 设置输出格式
func (l *Formater) SetFormat(format string, eol string) {
	l.format = format
	l.eol = eol
}

func (l *Formater) GetFormat() (format string, eol string) {
	return l.format, l.eol
}

// 新增格式定义
func (l *Formater) AddFormat(name string, f func(buf *[]byte, fa *FmtArgs)) {
	l.fmtsmu.Lock()
	defer l.fmtsmu.Unlock()
	l.fmts.newformater(name, f)
}

// 格式化输出内容
func (l *Formater) Format(fa *FmtArgs) (s string) {
	format, eol := l.format, l.eol
	var buf []byte
	l.bufmu.Lock()
	n := len(l.bufs)
	if n == 0 {
		buf = []byte{}
	} else {
		buf = l.bufs[n-1][:0]
		l.bufs = l.bufs[:n-1]
		if n > 1024 {
			l.bufs = l.bufs[:n/2]
		}
	}
	l.bufmu.Unlock()
	for i := 0; i < len(format); {
		b := l.format[i]
		if fmts, ok := l.fmts[b]; ok {
			ok = false
			for _, fmt := range fmts {
				ie := i + fmt.fnl
				if len(format) >= ie && fmt.name == format[i:ie] {
					fmt.format(&buf, fa)
					i += len(fmt.name)
					ok = true
					break
				}
			}
			if !ok {
				buf = append(buf, format[i])
				i += 1
			}
		} else {
			buf = append(buf, format[i])
			i += 1
		}
	}
	if len(buf) < len(eol) || string(buf[len(buf)-len(l.eol):]) != eol {
		buf = append(buf, eol...)
	}
	s = string(buf)
	l.bufmu.Lock()
	l.bufs = append(l.bufs, buf[:0])
	l.bufmu.Unlock()
	return
}
