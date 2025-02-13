package rc_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/wecisecode/util/rc"
)

func TestRC(t *testing.T) {
	var wg sync.WaitGroup
	rc := rc.NewRoutinesControllerLimit("", 1, 2)
	fmt.Println(1)
	wg.Add(1)
	rc.ConcurCall(1, func() {
		defer wg.Done()
		fmt.Println(1, "s")
		time.Sleep(3 * time.Second)
		fmt.Println(1, "e")
	})
	fmt.Println(2)
	wg.Add(1)
	rc.ConcurCall(1, func() {
		defer wg.Done()
		fmt.Println(2, "s")
		time.Sleep(3 * time.Second)
		fmt.Println(2, "e")
	})
	fmt.Println(3)
	wg.Add(1)
	rc.ConcurCall(1, func() {
		defer wg.Done()
		fmt.Println(3, "s")
		time.Sleep(3 * time.Second)
		fmt.Println(3, "e")
	})
	fmt.Println("S")
	rc.SetConcurrencyLimitCount(3)
	wg.Wait()
}
