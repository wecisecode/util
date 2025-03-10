package deepcopy_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wecisecode/util/deepcopy"
	"github.com/wecisecode/util/msgpack"
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

func TestDeepCopyMap(t *testing.T) {
	m1 := map[string]any{}
	for i := 0; i < 1000000; i++ {
		m1[fmt.Sprint("X", i)] = fmt.Sprint("A", i)
	}
	st := time.Now()
	m2 := deepcopy.DeepCopy(m1).(map[string]any)
	fmt.Println("deepcopy", time.Since(st))
	assert.Equal(t, msgpack.MustEncodeString(m1), msgpack.MustEncodeString(m2))
	st = time.Now()
	m3 := map[string]any{}
	for k, v := range m1 {
		m3[k] = v
	}
	fmt.Println("duplicate", time.Since(st))
	assert.Equal(t, msgpack.MustEncodeString(m1), msgpack.MustEncodeString(m3))
	// deepcopy 2.687850414s
	// duplicate 694.957856ms
}
