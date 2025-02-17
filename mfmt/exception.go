package mfmt

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"runtime"
	"strings"
)

var regx_sourcecodeline = regexp.MustCompile(`^.*\([^:]+\/[^:]+\:\d+\)$`)

func ErrorWithSourceLine(xrecover ...interface{}) error {
	if len(xrecover) == 0 || len(xrecover) == 1 && xrecover[0] == nil {
		return nil
	}
	s := fmt.Sprint(xrecover...)
	if !regx_sourcecodeline.MatchString(s) {
		defaultSourceCodeLine := "unkown/source:0"
		sourceCodeLine := ""
		for skip := 1; sourceCodeLine == ""; skip++ {
			_, f, n, ok := runtime.Caller(skip)
			if !ok {
				sourceCodeLine = defaultSourceCodeLine
				break
			}
			if base := path.Base(f); base != "" {
				sourceCodeLine = fmt.Sprint(path.Base(path.Dir(f)), "/", base, ":", n)
				if skip == 1 {
					defaultSourceCodeLine = sourceCodeLine
				}
			}
			if strings.HasPrefix(f, "/usr/local/go/src/runtime/") {
				sourceCodeLine = ""
			}
		}
		s += "(" + sourceCodeLine + ")"
	}
	return errors.New(s)
}
