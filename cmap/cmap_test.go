package cmap_test

import (
	"fmt"
	"testing"

	"github.com/wecisecode/util/cmap"
)

func TestRun(t *testing.T) {
	a := cmap.New(map[string]interface{}{"k": "v", "x": 1, "z": nil, "s": []string{"a", "b", "c"}, "m": map[string]string{"中": "🀄️"}})
	fmt.Println(a)
	fmt.Println(a.Count())
}
