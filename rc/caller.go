package rc

import (
	"regexp"
	"runtime"
	"strings"

	"github.com/wecisecode/util/cast"
)

var regxfuncname = regexp.MustCompile(`([^/]+)\((?:[^\(]+)`)
var regxfileline = regexp.MustCompile(`([^/]+\/[^/]+\:\d+)`)
var regxgroutine = regexp.MustCompile(`(\d+)`)

func FuncName() (funcname string) {
	_, _, funcname = GetCaller(0, 2)
	return
}

func GetCaller(depthsrc int, depthcall int) (routine int, srcline string, funcname string) {
	var sline [][]string
	var stack []string
	var n = 1024
	buf := make([]byte, n)
	for {
		n = runtime.Stack(buf, false)
		stack = strings.Split(string(buf), "\n")
		if len(stack) > 0 {
			sroutine := regxgroutine.FindString(stack[0])
			routine = cast.ToInt(sroutine)
		}
		if depthsrc > 0 && len(stack) > depthsrc*2+2 {
			sline = regxfileline.FindAllStringSubmatch(strings.TrimSpace(stack[depthsrc*2+2]), -1)
			if len(sline) > 0 && len(sline[0]) > 1 {
				srcline = sline[0][1]
			}
		}
		if depthcall > 0 && len(stack) > depthcall*2+1 {
			sline = regxfuncname.FindAllStringSubmatch(strings.TrimSpace(stack[depthcall*2+1]), -1)
			if len(sline) > 0 && len(sline[0]) > 1 {
				funcname = sline[0][1]
			}
		}
		if (depthsrc <= 0 || srcline != "") && (depthcall <= 0 || funcname != "") {
			return
		}
		if n < len(buf) {
			return
		}
		buf = make([]byte, 2*len(buf))
	}
}
