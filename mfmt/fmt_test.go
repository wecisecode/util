package mfmt_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wecisecode/util/mfmt"
)

func TestParseDuration(t *testing.T) {
	assert.Equal(t, time.Duration(34*time.Hour), mfmt.ParseDuration("1天10h"))
	assert.Equal(t, time.Duration(34*time.Hour+50*time.Minute), mfmt.ParseDuration("1d10h50m"))
}
