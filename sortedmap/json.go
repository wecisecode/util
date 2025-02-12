package sortedmap

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/spf13/cast"
)

var JSONParserError = fmt.Errorf("parser error")

type JSONFormatError struct {
	IndexStart int
	IndexCur   int
	Runes      []rune
	Msg        string
}

func (jfe *JSONFormatError) Error() string {
	s := string(jfe.Runes[jfe.IndexStart:jfe.IndexCur])
	if jfe.IndexStart > 0 {
		s = "..." + s
	}
	if jfe.IndexCur < len(jfe.Runes) {
		s += "..."
	}
	if len(s) < 30 {
		s = string(jfe.Runes[jfe.IndexStart:])
		if len(s) > 30 {
			s = s[:30] + "..."
		}
	}
	if len(s) > 150 {
		s = s[:150] + "..."
	}
	return fmt.Sprint("json format error[", jfe.IndexStart, ",", jfe.IndexCur, "] ", s, " ", jfe.Msg)
}

func unmarshalJSON(sm SortedMap, rs []rune, i, n int) int {
	e := i
	skipSpace := func() {
		for i < n && unicode.IsSpace(rs[i]) {
			i++
		}
	}
	isRune := func(mr ...rune) bool {
		for _, r := range mr {
			if i < n && rs[i] == r {
				return true
			}
		}
		return false
	}
	needRune := func(r rune) {
		skipSpace()
		if isRune(r) {
			i++
			return
		}
		panic(&JSONFormatError{e, i, rs, ""})
	}
	getToRune := func(mr ...rune) string {
		mrb := map[rune]bool{}
		for _, r := range mr {
			mrb[r] = true
		}
		vis := i
		for i < n && !mrb[rs[i]] {
			i++
		}
		if i < n && mrb[rs[i]] {
			return string(rs[vis:i])
		}
		panic(&JSONFormatError{e, i, rs, ""})
	}
	getString := func() string {
		needRune('"')
		ors := []rune{}
		for i < n && rs[i] != '"' {
			if rs[i] == '\\' {
				i++
			}
			ors = append(ors, rs[i])
			i++
		}
		if isRune('"') {
			i++
			return string(ors)
		}
		panic(&JSONFormatError{e, i, rs, ""})
	}
	var getItem func(endflag ...rune) (interface{}, bool)
	getList := func() []interface{} {
		as := []interface{}{}
		needRune('[')
		for i < n && rs[i] != ']' {
			item, ok := getItem(',', ']')
			if ok {
				as = append(as, item)
			}
			if isRune(',') {
				i++
			}
		}
		if isRune(']') {
			i++
			return as
		}
		panic(&JSONFormatError{e, i, rs, ""})
	}
	getItem = func(endflag ...rune) (interface{}, bool) {
		skipSpace()
		if isRune(endflag...) {
			return nil, false
		} else if isRune('{') {
			nsm := NewLinkedMap()
			i = unmarshalJSON(nsm, rs, i, n)
			return nsm, true
		} else if isRune('[') {
			return getList(), true
		} else if isRune('"') {
			return getString(), true
		} else {
			s := getToRune(endflag...)
			s = strings.TrimSpace(s)
			s = strings.ToLower(s)
			switch s {
			case "true":
				return true, true
			case "false":
				return false, true
			case "null", "nil":
				return nil, true
			default:
				intnumber, e := cast.ToInt64E(s)
				if e == nil {
					return intnumber, true
				}
				floatnumber, e := cast.ToFloat64E(s)
				if e == nil {
					return floatnumber, true
				}
				// TODO 二进制编码扩展，不能通过json.Unmarshal的格式检查，暂时不起作用
				if s[:2] == "0x" {
					bs, e := hex.DecodeString(s[2:])
					if e == nil {
						return bs, true
					}
				} else {
					bs, e := base64.RawStdEncoding.DecodeString(s)
					if e == nil {
						return bs, true
					}
				}
			}
		}
		panic(&JSONFormatError{e, i, rs, ""})
	}
	// getMap
	needRune('{')
	for i < n && rs[i] != '}' {
		key, _ := getItem(':', '}')
		skipSpace()
		if isRune(':') {
			i++
			val, _ := getItem(',', '}')
			sm.Put(key, val)
			if isRune(',') {
				i++
			}
		}
	}
	if isRune('}') {
		i++
		skipSpace()
		return i
	}
	panic(&JSONFormatError{e, i, rs, ""})
}

func UnmarshalJSON(sm SortedMap, bs []byte) (err error) {
	rs := []rune(string(bs))
	n := len(rs)
	i := 0
	defer func() {
		if x := recover(); x != nil {
			e, ok := x.(error)
			if ok {
				err = e
			} else if err == nil {
				err = JSONParserError
			} // else return existed err
		}
	}()
	i = unmarshalJSON(sm, rs, i, n)
	if i < n {
		panic(&JSONFormatError{i, n, rs, "expect EOF"})
	}
	return nil
}

func MarshalJSON(sm SortedMap) (rbs []byte, err error) {
	rbs = []byte("{")
	sm.Fetch(func(key, value interface{}) bool {
		if len(rbs) > 1 {
			rbs = append(rbs, ',')
		}
		s1, e := json.Marshal(map[string]interface{}{cast.ToString(key): value})
		if e != nil {
			err = e
			return false
		}
		rbs = append(rbs, s1[1:len(s1)-1]...)
		return true
	})
	rbs = append(rbs, '}')
	return
}
