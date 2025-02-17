package deepcopy_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wecisecode/util/deepcopy"
)

type TestStructA struct {
	A int
	B string
	C *TestStructA
	p string
}

func NewTestStructA(a int, b string, c *TestStructA, p string) (ts *TestStructA) {
	return &TestStructA{
		A: a,
		B: b,
		C: c,
		p: p,
	}
}

func (ts *TestStructA) DeepCopy() *TestStructA {
	if ts == nil {
		return nil
	}
	return &TestStructA{
		A: ts.A,
		B: ts.B,
		C: ts.C.DeepCopy(),
		p: ts.p,
	}
}

func (ts *TestStructA) String() string {
	return ts.string("")
}

func (ts *TestStructA) string(indent string) string {
	if ts == nil {
		return "nil"
	}
	return fmt.Sprint(
		indent, "A: ", ts.A, "\n",
		indent, "B: ", ts.B, "\n",
		indent, "C: ", ts.C.string("  "), "\n",
		indent, "p: ", ts.p,
	)
}

type TestStructB struct {
	A int
	B string
	C *TestStructA
	p string
}

func NewTestStructB(a int, b string, c *TestStructA, p string) (ts *TestStructB) {
	return &TestStructB{
		A: a,
		B: b,
		C: c,
		p: p,
	}
}

func (ts *TestStructB) String() string {
	return ts.string("")
}

func (ts *TestStructB) string(indent string) string {
	if ts == nil {
		return "nil"
	}
	return fmt.Sprint(
		indent, "A: ", ts.A, "\n",
		indent, "B: ", ts.B, "\n",
		indent, "C: ", ts.C.string("  "), "\n",
		indent, "p: ", ts.p,
	)
}

func TestDeepCopy(t *testing.T) {
	x := deepcopy.DeepCopy(NewTestStructB(1, "xx", NewTestStructA(2, "yy", nil, "---"), "..."))
	receiver := deepcopy.Receiver{}
	z := deepcopy.DeepCopy(x, receiver)
	fmt.Println(z)
	fmt.Println(receiver)
	assert.Equal(t, `A: 1
B: xx
C:   A: 2
  B: yy
  C: nil
  p: ---
p: `, fmt.Sprint(z))
	assert.Equal(t, `github.com/wecisecode/util/deepcopy_test/TestStructB:
  p: string`, fmt.Sprint(receiver))
}
