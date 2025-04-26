package mio

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/wecisecode/util/mid"
)

func last_filename(filename string) string {
	i := strings.LastIndex(filename, ".")
	if i > 0 {
		return filename[:i] + ".last" + filename[i:]
	}
	return filename + ".last"
}

func temp_filename(filename string) string {
	dir, file := filepath.Split(filename)
	return filepath.Join(dir, fmt.Sprint(".", file, ".", mid.MTimeStamp().UnixNano(), ".tmp"))
}

func ReadFile(filename string) ([]byte, error) {
	bs, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			bs, err = os.ReadFile(last_filename(filename))
			if os.IsNotExist(err) {
				return []byte{}, nil
			}
		}
	}
	if err != nil {
		return []byte{}, fmt.Errorf("file read %s error: %v", filename, err)
	}
	return bs, nil
}

func WriteFile(filename string, content []byte, keeplast bool) error {
	os.MkdirAll(filepath.Dir(filename), 0775)
	tempfn := filename + ".tmp"
	f, err := os.OpenFile(tempfn, os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		return fmt.Errorf("open file %s error: %v", tempfn, err)
	}
	defer func() {
		e := f.Close()
		if e == nil && err == nil {
			lastfn := last_filename(filename)
			os.Remove(lastfn)
			os.Rename(filename, lastfn)
			os.Rename(tempfn, filename)
			if !keeplast {
				os.Remove(lastfn)
			}
		} else {
			os.Remove(tempfn)
		}
	}()
	_, err = f.Write(content)
	if err != nil {
		return fmt.Errorf("file write %s error: %v", filename, err)
	}
	return nil
}

type ClearFiles struct {
	Dir       string
	NameMatch string
	// OrderBy   string // time,size
	// Recursive bool
	KeepLast int
}

func (o *ClearFiles) Do() (err error) {
	if o.Dir == "" {
		return fmt.Errorf("目录不能为空")
	}
	if o.NameMatch == "" {
		return fmt.Errorf("文件名匹配表达式不能为空")
	}
	namematch, e := regexp.Compile(o.NameMatch)
	if e != nil {
		return e
	}
	keeplast := o.KeepLast
	if keeplast < 0 {
		return fmt.Errorf("保留文件数不能小于0")
	}
	des, e := os.ReadDir(o.Dir)
	if e != nil {
		return e
	}
	fis := []fs.FileInfo{}
	for _, de := range des {
		fi, e := de.Info()
		if e != nil {
			return e
		}
		if !fi.IsDir() && namematch.MatchString(fi.Name()) {
			fis = append(fis, fi)
		}
	}
	sort.Slice(fis, func(i, j int) bool {
		if fis[i].ModTime().Equal(fis[j].ModTime()) {
			if fis[i].Size() == fis[j].Size() {
				return fis[i].Name() < fis[j].Name()
			}
			return fis[i].Size() > fis[j].Size()
		}
		return fis[i].ModTime().Before(fis[j].ModTime())
	})

	for i := 0; len(fis) > o.KeepLast && i < len(fis)-o.KeepLast; {
		fp := filepath.Join(o.Dir, fis[i].Name())
		os.Remove(fp)
		fis = fis[1:]
	}
	return
}
