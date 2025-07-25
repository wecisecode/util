package pattern

func wildcard2RegexpString(wildcard string, regular bool) string {
	pattern := []rune(wildcard)
	prunes := []rune{}
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c > 0x00FF {
			if i == 0 && regular {
				prunes = append(prunes, '^')
			}
			prunes = append(prunes, c)
			if i == len(pattern)-1 && regular {
				prunes = append(prunes, '$')
			}
		} else if c == '*' {
			// * 变 .*
			prunes = append(prunes, '.', '*')
		} else if c == '?' {
			// ? 变 .
			prunes = append(prunes, '.')
		} else {
			// 转义特殊字符
			prunes = append(prunes, '\\', c)
		}
	}
	if regular {
		return "(?s)" + string(prunes)
	}
	return string(prunes)
}

func Wildcard2RegexpString(wildcard string) string {
	return wildcard2RegexpString(wildcard, true)
}

func Wildcard2SimpleRegexpString(wildcard string) string {
	return wildcard2RegexpString(wildcard, false)
}

func pathWildcard2RegexpString(wildcardpath string, regular bool) string {
	pattern := []rune(wildcardpath)
	prunes := []rune{}
	if len(wildcardpath) > 0 && wildcardpath[0] == '/' || len(wildcardpath) > 2 && wildcardpath[1:3] == `:\` {
		if regular {
			prunes = append(prunes, '^')
		}
	}
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c > 0x00FF {
			prunes = append(prunes, c)
		} else if c == '*' {
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				// ** 变 .*
				i++
				prunes = append(prunes, '.', '*')
			} else {
				// * 变 [^\/\\]*
				prunes = append(prunes, []rune(`[^\/\\]*`)...)
			}
		} else if c == '?' {
			// ? 变 .
			prunes = append(prunes, '.')
		} else {
			// 转义特殊字符
			prunes = append(prunes, '\\', c)
		}
	}
	if regular {
		return "(?s)" + string(prunes)
	}
	return string(prunes)
}

func PathWildcard2RegexpString(wildcardpath string) string {
	return pathWildcard2RegexpString(wildcardpath, true)
}

func PathWildcard2SimpleRegexpString(wildcardpath string) string {
	return pathWildcard2RegexpString(wildcardpath, false)
}

func text2RegexpString(text string) string {
	pattern := []rune(text)
	prunes := []rune{}
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c > 0x00FF {
			prunes = append(prunes, c)
		} else {
			// 转义特殊字符
			prunes = append(prunes, '\\', c)
		}
	}
	return string(prunes)
}

func Equal2RegexpString(equal string) string {
	return "(?s)^" + text2RegexpString(equal) + "$"
}

func Contain2RegexpString(contain string) string {
	return "(?s)" + text2RegexpString(contain) + ""
}
