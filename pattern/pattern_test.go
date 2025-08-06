package pattern_test

import (
	"fmt"
	"testing"

	"github.com/wecisecode/util/pattern"
)

func TestGlob2Regexp(t *testing.T) {
	fmt.Println(pattern.Glob2SimpleRegexpString("[ab-d]"))
	fmt.Println(pattern.Glob2RegexpString("xxx*xxx[ab-d]xxx?xxx"))
	fmt.Println(pattern.Glob2SimpleRegexpString("xxx*xxx[ab-d]xxx?xxx"))
}
