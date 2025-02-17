package mfmt_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wecisecode/util/mfmt"
)

func TestParseDuration(t *testing.T) {
	go fmt.Println(func() error { return mfmt.ErrorWithSourceLine("test") }())
	fmt.Println(func() error { return mfmt.ErrorWithSourceLine("test") }())
	assert.Equal(t, time.Duration(34*time.Hour), mfmt.ParseDuration("1天10h"))
	assert.Equal(t, time.Duration(34*time.Hour+50*time.Minute), mfmt.ParseDuration("1d10h50m"))
}
