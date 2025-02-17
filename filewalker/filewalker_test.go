package filewalker_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wecisecode/util/filewalker"
)

func TestFileWalker(t *testing.T) {
	fw, e := filewalker.NewFileWalker([]string{".."}, ".*")
	if e != nil {
		assert.Nil(t, e)
	}
	fw.List(func(basedir, fpath string) bool {
		fmt.Println(filepath.Join(basedir, fpath))
		return true
	})
}
