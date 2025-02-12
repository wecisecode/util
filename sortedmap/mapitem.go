package sortedmap

import (
	"bytes"
	"reflect"
	"time"

	"github.com/spf13/cast"
)

// ItemVistor callback should return true to keep going on the visitation.
type ItemVisitor func(i Item) bool

// Compare returns an integer comparing the two items
// lexicographically. The result will be 0 if a==b, -1 if a < b, and
// +1 if a > b.
// likly return a - b
type Compare func(a, b interface{}) int

// Item can be anything.
type Item interface{}

type MapItem struct {
	Key   interface{}
	Value interface{}
}

func StringCompare(a, b string) int {
	return bytes.Compare([]byte(a), []byte(b))
}

func IntCompare(a, b int) int {
	return a - b
}

func Int64Compare(a, b int64) int {
	return int(a - b)
}

func KeyCompare(a, b interface{}) int {
	ta := reflect.TypeOf(a).Kind()
	if ta != reflect.TypeOf(b).Kind() {
		return bytes.Compare([]byte(cast.ToString(a)), []byte(cast.ToString(b)))
	}
	switch aa := a.(type) {
	case string:
		return StringCompare(aa, b.(string))
	case int:
		return IntCompare(aa, b.(int))
	case int8, int16, int32, uint8, uint16, uint32, uint64:
		return IntCompare(cast.ToInt(aa), cast.ToInt(b))
	case int64:
		return Int64Compare(aa, b.(int64))
	case time.Time:
		return Int64Compare(aa.UnixNano(), b.(time.Time).UnixNano())
	case *time.Time:
		return Int64Compare(aa.UnixNano(), b.(*time.Time).UnixNano())
	default:
		panic("不支持的数据类型，需增加相应类型的比较函数")
	}
}

func MapItemCompare(a, b interface{}) int {
	var ka, kb interface{}
	if mi, ok := a.(*MapItem); ok {
		ka = mi.Key
	} else {
		ka = a
	}
	if mi, ok := b.(*MapItem); ok {
		kb = mi.Key
	} else {
		kb = b
	}
	return KeyCompare(ka, kb)
}
