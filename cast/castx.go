package cast

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cast"
)

func ToStr(v any) (rets string) {
	if v == nil {
		return ""
	}
	switch r := v.(type) {
	case []any:
		for _, v := range r {
			rets += ToStr(v)
		}
	case []byte:
		return string(r)
	case string:
		return r
	default:
		return cast.ToString(r)
	}
	return
}

func ToStrs(v any) (rets []string) {
	switch r := v.(type) {
	case []any:
		for _, v := range r {
			rets = append(rets, ToStr(v))
		}
	case []byte:
		return []string{string(r)}
	case string:
		return []string{r}
	case []string:
		return r
	default:
		return []string{cast.ToString(r)}
	}
	return
}

func Unquote(ss string) string {
	if len(ss) >= 2 && ss[0] == '`' && ss[len(ss)-1] == '`' {
		ss = ss[1 : len(ss)-1]
		return ss
	}
	if len(ss) < 2 || ss[0] != '"' || ss[len(ss)-1] != '"' {
		ss = `"` + ss + `"`
	}
	s, e := strconv.Unquote(ss)
	if e != nil {
		ss := strings.ReplaceAll(strings.ReplaceAll(ss, "\n", `\n`), "\r", `\r`)
		s, e = strconv.Unquote(ss)
		if e != nil {
			panic(fmt.Errorf("cannot parse string, %s", ss))
		}
	}
	return s
}

func UnmarshalJsonString(encodedstring string) string {
	var s string
	e := json.Unmarshal([]byte(encodedstring), &s)
	if e != nil {
		s = Unquote(encodedstring)
	}
	return s
}
