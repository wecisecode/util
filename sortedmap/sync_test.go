package sortedmap_test

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"
	"testing"

	gtreap "github.com/wecisecode/util/sortedmap"
)

func BenchmarkSyncWrite1(t *testing.B) {
	t.StartTimer()
	m := map[int]string{}
	i := 0
	w := 0
	for ; i < 10000; i++ {
		w++
		m[w] = strconv.Itoa(w)
	}
	t.StopTimer()
	fmt.Printf("i=%d, len(m)=%d\n", i, len(m))
	// i=10000, len(m)=10000
	// 0.002602 ns/op
}

func BenchmarkSyncWrite2(t *testing.B) {
	t.StartTimer()
	m := map[int]string{}
	i := 0
	w := 0
	for ; i < 10000; i++ {
		go func() {
			w++
			m[w] = strconv.Itoa(w)
		}()
	}
	t.StopTimer()
	fmt.Printf("i=%d, len(m)=%d\n", i, len(m))
	// Crash: Throw reason unavailable, see https://github.com/golang/go/issues/46425
}

func BenchmarkSyncWrite3(t *testing.B) {
	t.StartTimer()
	m := map[int]string{}
	i := 0
	w := 0
	var mutex sync.RWMutex
	for ; i < 10000; i++ {
		go func() {
			mutex.Lock()
			defer mutex.Unlock()
			w++
			m[w] = strconv.Itoa(w)
		}()
	}
	t.StopTimer()
	fmt.Printf("i=%d, len(m)=%d\n", i, len(m))
	// i=10000, len(m)=5449
	// 0.01489 ns/op
	// i=10000, len(m)=6714
	// 0.006816 ns/op
}

func BenchmarkSyncWrite4(t *testing.B) {
	t.StartTimer()
	m := map[int]string{}
	i := 0
	w := 0
	var mutex sync.RWMutex
	var wg sync.WaitGroup
	for ; i < 10000; i++ {
		wg.Add(1)
		go func() {
			mutex.Lock()
			defer func() {
				mutex.Unlock()
				wg.Done()
			}()
			w++
			m[w] = strconv.Itoa(w)
		}()
	}
	wg.Wait()
	t.StopTimer()
	fmt.Printf("i=%d, len(m)=%d\n", i, len(m))
	// i=10000, len(m)=10000
	// 0.01763 ns/op
	// 0.01725 ns/op
	// 0.01987 ns/op
}

func BenchmarkSyncWrite5(t *testing.B) {
	t.StartTimer()
	m := gtreap.NewTreap(func(a, b interface{}) int {
		return bytes.Compare([]byte(a.(string)), []byte(b.(string)))
	})
	i := 0
	w := 0
	var wg sync.WaitGroup
	for ; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
			}()
			w++
			m = m.Upsert(strconv.Itoa(w), w)
		}()
	}
	wg.Wait()
	t.StopTimer()
	fmt.Printf("i=%d\n", i)
	// i=10000
	// 0.1156 ns/op
	// 0.1591 ns/op
	// 0.1439 ns/op
}
