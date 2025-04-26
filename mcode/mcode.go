package mcode

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
)

func SourceCodeLine() string {
	_, f, n, ok := runtime.Caller(1)
	if !ok {
		return ""
	}
	return fmt.Sprint(filepath.Base(f), ":", n)
}

type routine struct {
	stack string
	count int
}

func (me *routine) String() string {
	return fmt.Sprint(me.count, ": ", me.stack)
}

type routines []*routine

func (me routines) String() string {
	s := ""
	for _, r := range me {
		if len(s) > 0 {
			s += "\n"
		}
		s += r.String()
	}
	return s
}

var re_goroutine_number = regexp.MustCompile(`(in goroutine \d+)`)

func RoutinesCreaterInfo() (routines routines) {
	n := 1024 * runtime.NumGoroutine()
	bs := make([]byte, n)
	n = runtime.Stack(bs, true)
	for n == len(bs) {
		bs = make([]byte, len(bs)*2)
		n = runtime.Stack(bs, true)
	}
	bs = bs[:n]
	// println(string(bs))
	stackcount := map[string]int{}
	for is, ie := 0, 0; is >= 0 && ie >= 0; {
		is = bytes.Index(bs[ie:], []byte("created by "))
		if is >= 0 {
			is += ie
			ie = bytes.Index(bs[is:], []byte(" +0x"))
			if ie >= 0 {
				ie += is
				nbs := bs[is:ie]
				nbs = bytes.ReplaceAll(nbs, []byte("\n\t"), []byte(" "))
				nbs = bytes.ReplaceAll(nbs, []byte("\r"), []byte(""))
				nbs = bytes.ReplaceAll(nbs, []byte(":"), []byte("."))
				screatedby := string(nbs)
				screatedby = re_goroutine_number.ReplaceAllString(screatedby, "at")
				stackcount[screatedby] += 1
			}
		}
	}
	for k, v := range stackcount {
		routines = append(routines, &routine{k, v})
	}
	sort.Slice(routines, func(i, j int) bool {
		if routines[i].count == routines[j].count {
			return routines[i].stack < routines[i].stack
		}
		return routines[i].count > routines[j].count
	})
	return
}

func bytesIndex(bs []byte, seps [][]byte) int {
	for i := 0; i < len(bs); i++ {
		for _, sep := range seps {
			if bytes.HasPrefix(bs[i:], sep) {
				return i
			}
		}
	}
	return -1
}

func StackCount() (routines routines) {
	n := 1024 * runtime.NumGoroutine()
	bs := make([]byte, n)
	n = runtime.Stack(bs, true)
	for n == len(bs) {
		bs = make([]byte, len(bs)*2)
		n = runtime.Stack(bs, true)
	}
	bs = bs[:n]
	// println(string(bs))
	stackcount := map[string]int{}
	for is, ie := 0, 0; is >= 0 && ie >= 0; {
		is = bytes.Index(bs[ie:], []byte("goroutine "))
		if is >= 0 {
			is += ie
			ie = bytes.Index(bs[is:], []byte("\n\n"))
			if ie >= 0 {
				ie += is + 1
				nbs := bs[is:ie]
				stack := []byte{}
				for nis, nie := 0, 0; nis >= 0 && nie >= 0; {
					nis = bytes.Index(nbs[nie:], []byte("goroutine "))
					if nis >= 0 {
						nis += nie
						nie = bytes.Index(nbs[nis:], []byte("\n"))
						if nie > 0 {
							nie += nis + 1
							stack = append(stack, nbs[nis:nie]...)
						}
					} else {
						nis = bytesIndex(nbs[nie:], [][]byte{
							[]byte("git.wecise.com"),
							[]byte("google.golang."),
							[]byte("golang."),
							[]byte("created by "),
							[]byte("main.main"),
						})
						if nis >= 0 {
							nis += nie
							nie = bytes.Index(nbs[nis:], []byte("\n"))
							if nie > 0 {
								nit := nis + nie + 1
								nie = bytes.Index(nbs[nit:], []byte("\n"))
								if nie > 0 {
									nie += nit + 1
									stack = append(stack, nbs[nis:nie]...)
								}
							}
						}
					}
				}
				stack = regexp.MustCompile(`goroutine \d+ \[`).ReplaceAll(stack, []byte(`goroutine [`))
				stack = regexp.MustCompile(`([^\.])\([^\)]*\)`).ReplaceAll(stack, []byte(`$1`))
				stack = regexp.MustCompile(` \+0x\w+`).ReplaceAll(stack, []byte(``))
				if regexp.MustCompile(`(?s)^goroutine [\w*]:\n$`).Match(stack) {
					stack = nbs
				}
				stackcount[string(stack)] += 1
			}
		}
	}
	for k, v := range stackcount {
		routines = append(routines, &routine{k, v})
	}
	sort.Slice(routines, func(i, j int) bool {
		if routines[i].count == routines[j].count {
			return routines[i].stack < routines[i].stack
		}
		return routines[i].count > routines[j].count
	})
	return
}
