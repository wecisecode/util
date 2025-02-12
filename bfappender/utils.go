package bfappender

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"sort"
	"time"
)

// 时间取整字符串
func timeFixString(t time.Time, d time.Duration) string {
	// 默认格式
	layout := "20060102"
	length := len(layout)
	s := int64(d.Seconds())
	if s <= 30*24*3600 {
		if s%10 != 0 {
			// 秒级取整
			layout = "20060102150405"
			length = len(layout)
		} else if s%60 != 0 {
			// 10秒级取整
			layout = "20060102150405"
			length = len(layout) - 1
		} else if s%600 != 0 {
			// 分钟级取整
			layout = "200601021504"
			length = len(layout)
		} else if s%3600 != 0 {
			// 10分钟级取整
			layout = "200601021504"
			length = len(layout) - 1
		} else if s%(24*3600) != 0 {
			// 小时级取整
			layout = "2006010215"
			length = len(layout)
		} else {
			// 天级取整
			layout = "20060102"
			length = len(layout)
		}
	} else if s <= 300*24*3600 {
		// 月级取整
		layout = "200601"
		length = len(layout)
	} else if s <= 366*24*3600 {
		// 年级取整
		layout = "2006"
		length = len(layout)
	} else if s <= 36600*24*3600 {
		// 世纪级取整
		layout = "2006"
		length = len(layout) - 2
	} else {
		// 与时间无关
		return ""
	}
	tt, _ := time.Parse("2006-01-02 15:04:05.000000000", t.Format("2006-01-02 15:04:05.000000000"))
	return tt.Truncate(d).Format(layout)[:length]
}

func archiveFiles(filename string) (afs []string, aft map[string]time.Time) {
	dir, fname := path.Split(filename)
	ext := path.Ext(fname)
	fnbase := fname[:len(fname)-len(ext)]
	des, _ := os.ReadDir(dir)
	fis := []fs.FileInfo{}
	for _, de := range des {
		fi, _ := de.Info()
		if fi != nil && !fi.IsDir() {
			fnbs := []byte(fi.Name())
			if bytes.HasPrefix(fnbs, []byte(fnbase)) && bytes.HasSuffix(fnbs, []byte(ext)) && fi.Name() != fname {
				fis = append(fis, fi)
			}
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
	aft = make(map[string]time.Time)
	for _, fi := range fis {
		fp := path.Join(dir, fi.Name())
		afs = append(afs, fp)
		aft[fp] = fi.ModTime()
	}
	return
}
