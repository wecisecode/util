package filewalker

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type FileWalker struct {
	WalkDir []string         // 与 RePath 一一对应，指定遍历目录
	RePath  []*regexp.Regexp // 与 WalkDir 一一对应，指定遍历文件路径要匹配的正则表达式
	ReFile  *regexp.Regexp   // 遍历文件名匹配的正则表达式
}

// 遍历文件，
// 支持通配符 ** 表示任意字符，* 表示除分隔符以外的任意字符, . 表示递归当前目录下的所有子目录
// 为避免 shell 自动将 * 转换为文件名列表，可以将指定的 path 用引号包含
func NewFileWalker(walkpaths []string, fnpattern string) (fw *FileWalker, err error) {
	walkdirs := []string{}
	repaths := []*regexp.Regexp{}
	for _, walkpath := range walkpaths {
		if walkpath == "" {
			continue
		}
		if walkpath[len(walkpath)-1:] == string(os.PathSeparator) {
			walkpath += "**"
		}
		wps := strings.Split(walkpath, string(os.PathSeparator))
		rwps := []string{} // 没有通配符的部分，作为遍历的根目录
		pwps := []string{} // 有通配符的部分，转换为正则表达式，在遍历过程中匹配
		for _, wpi := range wps {
			if strings.Index(wpi, "*") >= 0 || len(pwps) > 0 {
				pwps = append(pwps, wpi)
			} else {
				rwps = append(rwps, wpi)
			}
		}
		walkdir := "."
		if len(rwps) == 0 {
		} else if len(rwps) == 1 && rwps[0] == "" {
			walkdir = string(os.PathSeparator)
		} else {
			walkdir = strings.Join(rwps, string(os.PathSeparator))
		}
		info, _ := os.Stat(walkdir)
		if info != nil && !info.IsDir() {
			pwps = append([]string{filepath.Base(walkdir)}, pwps...)
			walkdir = filepath.Dir(walkdir)
		}
		pattern := []rune(strings.Join(pwps, string(os.PathSeparator)))
		prunes := []rune{}
		for i := 0; i < len(pattern); i++ {
			c := pattern[i]
			if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c > 0x00FF {
				if i == 0 {
					prunes = append(prunes, '^')
				}
				prunes = append(prunes, c)
			} else if c == '*' {
				if i+1 < len(pattern) && pattern[i+1] == '*' {
					// ** 变 .*
					i++
					prunes = append(prunes, '.', '*')
				} else {
					// * 变 [^\/]*
					prunes = append(prunes, []rune("[^\\"+string(os.PathSeparator)+"]*")...)
				}
			} else {
				// 转义特殊字符
				prunes = append(prunes, '\\', c)
			}
		}
		spattern := "(?s)" + string(prunes) + ".*"
		repath, err := regexp.Compile(spattern)
		if err != nil {
			return nil, fmt.Errorf("path pattern format error, %v", err)
		}
		walkdirs = append(walkdirs, walkdir)
		repaths = append(repaths, repath)
	}
	refile, err := regexp.Compile(fnpattern)
	if err != nil {
		return nil, fmt.Errorf("file pattern format error, %v", err)
	}
	fw = &FileWalker{
		WalkDir: walkdirs,
		RePath:  repaths,
		ReFile:  refile,
	}
	return
}

func (fw *FileWalker) Walk(walkdir string, proc func(fpath string, info fs.FileInfo, e error) error) (err error) {
	des, e := os.ReadDir(walkdir)
	if e != nil {
		return e
	}
	sort.Slice(des, func(i, j int) bool {
		fi1, e := des[i].Info()
		if e == nil {
			fi2, e := des[j].Info()
			if e == nil {
				if !fi1.IsDir() && fi2.IsDir() {
					return true
				}
				if fi1.IsDir() && !fi2.IsDir() {
					return false
				}
			}
		}
		return des[i].Name() < des[j].Name()
	})
	for _, de := range des {
		fn := de.Name()
		if fn == "." || fn == ".." {
			continue
		}
		fi, e := de.Info()
		fpath := strings.Join([]string{walkdir, fn}, string(os.PathSeparator))
		proc(fpath, fi, e)
		if fi != nil && fi.IsDir() {
			e := fw.Walk(fpath, proc)
			if e != nil {
				proc(fpath, fi, e)
			}
		}
	}
	return nil
}

func (fw *FileWalker) List(proc func(basedir, fpath string) bool) (err error) {
	if len(fw.WalkDir) != len(fw.RePath) {
		return fmt.Errorf("len(fw.WalkDir)[%d] != len(fw.RePath)[%d]", len(fw.WalkDir), len(fw.RePath))
	}
	for i, walkdir := range fw.WalkDir {
		e := fw.Walk(walkdir, func(fpath string, info fs.FileInfo, e error) error {
			if info != nil && !info.IsDir() {
				prefix := walkdir + string(os.PathSeparator)
				if strings.HasPrefix(fpath, prefix) {
					fpath = fpath[len(prefix):]
				}
				if fw.RePath[i].MatchString(fpath) {
					fn := filepath.Base(fpath)
					if fw.ReFile.MatchString(fn) {
						if !proc(walkdir, fpath) {
							e = filepath.SkipAll
							err = e
							return err
						}
					}
				}
			}
			return nil
		})
		if e != nil {
			return e
		}
		if err != nil {
			if err == filepath.SkipAll {
				err = nil
			}
			return
		}
	}
	return
}
