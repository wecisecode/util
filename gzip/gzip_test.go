package gzip_test

import (
	"testing"

	"github.com/wecisecode/util/gzip"
)

func TestGzipx(t *testing.T) {
	b, e := gzip.Encode([]byte("Hello"))
	if e != nil {
		t.Fatal(e)
	}
	t.Log("Encode successful")

	b2, e := gzip.Decode(b)
	if e != nil {
		t.Fatal(e)
	}
	if string(b2) == "Hello" {
		t.Log("Decode successful")
	} else {
		t.Error("Decode not match")
	}
}
