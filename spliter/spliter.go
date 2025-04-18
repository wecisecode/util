package spliter

import (
	"bufio"
	"io"
	"regexp"
	"strings"
	"sync"
)

// regexp 不支持预断言，如：END(?=\W)，所以只能自己做预判

// 排除内容标记
type ExcludeFlag struct {
	PrevChar  *regexp.Regexp // 排除内容起始标记前，前置字符，比如：关键字单词前必须是非单词字符或者是整体内容的开头，正则表达式应只尝试匹配前一个字符
	NextChar  *regexp.Regexp // 排除内容结束标记后，后置字符，比如：关键字单词后必须是非单词字符或者是整体内容的结尾，正则表达式应只尝试匹配后一个字符
	Begin     string
	End       string
	Escape    string
	RegxBegin *regexp.Regexp
	RegxEnd   *regexp.Regexp
	Nestable  bool
	Remove    bool // 剔除排除内容，如注释信息
}

var (
	regx_delimeters_mtx  sync.Mutex
	regx_delimeters      = map[string]*regexp.Regexp{}
	regx_delimeter_comma = regexpDelimeter("(?:;|；)")
	regx_batch_prev      = regexp.MustCompile(`\W`)
	regx_batch_begin     = regexp.MustCompile(`(?is)^BEGIN\s+`)
	regx_batch_end       = regexp.MustCompile(`(?is)^\s+END`)
	regx_batch_next      = regexp.MustCompile(`\W`)
)

func regexpDelimeter(delimeter string) *regexp.Regexp {
	regx_delimeters_mtx.Lock()
	regxdelimeter := regx_delimeters[delimeter]
	if regxdelimeter == nil {
		// delimeter为正则表达式，用(?s)取代原表达式中的前置标记
		regxdelimeter = regexp.MustCompile("(?s)^(" + regexp.MustCompile(`^\(\?[iLmsuUx]+\)`).ReplaceAllString(delimeter, "") + ")")
		regx_delimeters[delimeter] = regxdelimeter
	}
	regx_delimeters_mtx.Unlock()
	return regxdelimeter
}

// 分号分隔多条MQL语句
// 支持 begin batch - end
// 已知问题：begin batch - end 内，如果单条语句中包含 end 必须用双引号括起来，如：begin batch select "end" from table end
func MQLSplit(text string) (mqls []string) {
	mqs := NewMQLSpliter(bufio.NewReader(strings.NewReader(text)))
	for {
		s, b, _ := mqs.Next()
		if !b {
			return
		}
		mqls = append(mqls, s)
	}
}

// 分号分隔多条MQL语句
// 支持 begin batch - end
// 已知问题：begin batch - end 内，如果单条语句中包含 end 必须用双引号括起来，如：begin batch select "end" from table end
func MQLSplitClean(text string) (mqls []string) {
	mqs := NewMQLSpliterWithOption(bufio.NewReader(strings.NewReader(text)), regx_delimeter_comma, []*ExcludeFlag{
		{Begin: `"`, End: `"`, Escape: `\`},
		{Begin: `'`, End: `'`},
		{Begin: `--`, End: "\n", Remove: true},
		{Begin: `//`, End: "\n", Remove: true},
		{Begin: `/*`, End: `*/`, Remove: true},
		{PrevChar: regx_batch_prev, RegxBegin: regx_batch_begin, RegxEnd: regx_batch_end, NextChar: regx_batch_next, Nestable: true},
	})
	for {
		s, b, _ := mqs.Next()
		if !b {
			return
		}
		mqls = append(mqls, s)
	}
}

type MQLSpliter struct {
	// 参数变量

	reader        bufio.Reader
	regxdelimeter *regexp.Regexp
	excludeflags  []*ExcludeFlag

	// 过程变量

	i              int // 当前字符位置
	n              int // 当前行号
	ret            string
	err            error
	line           string
	inexcludeflags []*ExcludeFlag
}

// 分号分隔多条MQL语句
// 支持 begin batch - end
// 已知问题：begin batch - end 内，如果单条语句中包含 end 必须用双引号括起来，如：begin batch select "end" from table end
func NewMQLSpliter(reader io.Reader) *MQLSpliter {
	return NewMQLSpliterWithOption(reader,
		regx_delimeter_comma,
		[]*ExcludeFlag{
			{Begin: `"`, End: `"`, Escape: `\`},
			{Begin: `'`, End: `'`},
			{Begin: `--`, End: "\n"},
			{Begin: `//`, End: "\n"},
			{Begin: `/*`, End: `*/`},
			{PrevChar: regx_batch_prev, RegxBegin: regx_batch_begin, RegxEnd: regx_batch_end, NextChar: regx_batch_next, Nestable: true},
		},
	)
}

// 分号分隔多条MQL语句
// 支持 begin batch - end
// 已知问题：begin batch - end 内，如果单条语句中包含 end 必须用双引号括起来，如：begin batch select "end" from table end
func NewMQLSpliterWithOption(reader io.Reader, regxdelimeter *regexp.Regexp, excludeflags []*ExcludeFlag) *MQLSpliter {
	return &MQLSpliter{
		reader:        *bufio.NewReader(reader),
		regxdelimeter: regxdelimeter,
		excludeflags:  excludeflags,
		//
		n:              1,
		inexcludeflags: []*ExcludeFlag{},
	}
}

func (me *MQLSpliter) nextChar() string {
	s := me.preloadChars(1)
	me.i += len(s)
	return s
}

func (me *MQLSpliter) prevChar() string {
	if len(me.line) >= me.i && me.i > 0 {
		return me.line[me.i-1 : me.i]
	}
	return ""
}

func (me *MQLSpliter) preloadChars(n int) string {
	for len(me.line) < me.i+n {
		if me.err != nil {
			if me.i >= len(me.line) {
				return ""
			}
			return me.line[me.i:]
		}
		s, err := me.reader.ReadString('\n')
		me.err = err
		me.line += s
		if me.i > 256 && len(me.line) > 256 {
			me.i -= 256
			me.line = me.line[256:]
		}
	}
	return me.line[me.i : me.i+n]
}

func (me *MQLSpliter) lookaheadEndFlag(n int, inexcludeflag *ExcludeFlag) bool {
	if inexcludeflag == nil || inexcludeflag.NextChar == nil {
		return true
	}
	s := me.preloadChars(n + 1)
	if len(s) <= n {
		return true
	}
	return inexcludeflag.NextChar.MatchString(s[n : n+1])
}

func (me *MQLSpliter) Next() (mql string, hasnext bool, err error) {
	mql, _, _, _, _, hasnext, err = me.NextMQL()
	return
}

func (me *MQLSpliter) NextMQL() (mql string, fromline, toline, fromchar, tochar int, hasnext bool, err error) {
	fromline = me.n
	fromchar = me.i
	me.ret = ""
	gotchar := false
	for i := 0; ; i++ {
		var inexcludeflag *ExcludeFlag
		if len(me.inexcludeflags) > 0 {
			inexcludeflag = me.inexcludeflags[len(me.inexcludeflags)-1]
		}
		if gotchar {
			s := me.nextChar()
			if s == "" {
				if me.ret == "" {
					return "", 0, 0, 0, 0, false, me.err
				}
				return me.ret, fromline, me.n, fromchar, me.i, true, nil
			}
			me.n += strings.Count(s, "\n")
			if inexcludeflag == nil || !inexcludeflag.Remove {
				me.ret += s
			}
		}
		gotchar = true
		if inexcludeflag != nil {
			if len(inexcludeflag.Escape) > 0 {
				if s := me.preloadChars(len(inexcludeflag.Escape)); s == inexcludeflag.Escape {
					//skip one char
					me.i += len(inexcludeflag.Escape)
					s += me.nextChar()
					me.n += strings.Count(s, "\n")
					if !inexcludeflag.Remove {
						me.ret += s
					}
					gotchar = false
					continue // 跳过转义字符，继续扫描
				}
			}
			if len(inexcludeflag.End) > 0 {
				if s := me.preloadChars(len(inexcludeflag.End)); s == inexcludeflag.End && me.lookaheadEndFlag(len(inexcludeflag.End), inexcludeflag) {
					me.i += len(inexcludeflag.End)
					me.n += strings.Count(s, "\n")
					if !inexcludeflag.Remove {
						me.ret += s
					}
					gotchar = false
					me.inexcludeflags = me.inexcludeflags[:len(me.inexcludeflags)-1]
					continue // 排除内容完毕，返回上一层，继续扫描
				}
			} else if inexcludeflag.RegxEnd != nil {
				smatch := inexcludeflag.RegxEnd.FindString(me.preloadChars(256))
				if len(smatch) > 0 && me.lookaheadEndFlag(len(smatch), inexcludeflag) {
					me.i += len(smatch)
					me.n += strings.Count(smatch, "\n")
					if !inexcludeflag.Remove {
						me.ret += smatch
					}
					gotchar = false
					me.inexcludeflags = me.inexcludeflags[:len(me.inexcludeflags)-1]
					continue // 排除内容完毕，返回上一层，继续扫描
				}
			}
		}
		if len(me.inexcludeflags) == 0 {
			// 不在排除内容中，判断是否为分隔符，regxdelimeter表达式一定是以 ^ 开始的，也就是说只判断开头，不会继续后面内容的查找
			smatch := me.regxdelimeter.FindString(me.preloadChars(256))
			if len(smatch) > 0 {
				me.i += len(smatch)
				return me.ret, fromline, me.n, fromchar, me.i, true, nil
			}
		}
		if inexcludeflag == nil || inexcludeflag.Nestable {
			// 没有在排除内容中，或在允许嵌套的排除内容中
			// 检查是否进入新的排除内容
			for _, excludeflag := range me.excludeflags {
				if excludeflag != nil {
					if excludeflag.PrevChar == nil || i == 0 || excludeflag.PrevChar.MatchString(me.prevChar()) {
						if len(excludeflag.Begin) > 0 {
							if smatch := me.preloadChars(len(excludeflag.Begin)); smatch == excludeflag.Begin {
								me.inexcludeflags = append(me.inexcludeflags, excludeflag)
								me.i += len(excludeflag.Begin)
								me.n += strings.Count(smatch, "\n")
								if !excludeflag.Remove {
									me.ret += smatch
								}
								gotchar = false
								break // 进入排除内容，继续扫描
							}
						} else if excludeflag.RegxBegin != nil {
							smatch := excludeflag.RegxBegin.FindString(me.preloadChars(256))
							if len(smatch) > 0 {
								me.inexcludeflags = append(me.inexcludeflags, excludeflag)
								me.i += len(smatch)
								me.n += strings.Count(smatch, "\n")
								if !excludeflag.Remove {
									me.ret += smatch
								}
								gotchar = false
								break // 进入排除内容，继续扫描
							}
						}
					}
				}
			}
		}
	}
}
