package pattern

func Wildcard2RegexpString(wildcard string) string {
	pattern := []rune(wildcard)
	prunes := []rune{}
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c > 0x00FF {
			if i == 0 {
				prunes = append(prunes, '^')
			}
			prunes = append(prunes, c)
			if i == len(pattern)-1 {
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
	return "(?s)" + string(prunes)
}

func PathWildcard2RegexpString(wildcardpath string) string {
	pattern := []rune(wildcardpath)
	prunes := []rune{}
	if len(wildcardpath) > 0 && wildcardpath[0] == '/' || len(wildcardpath) > 2 && wildcardpath[1:3] == `:\` {
		prunes = append(prunes, '^')
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
	return "(?s)" + string(prunes)
}

func Equal2RegexpString(equal string) string {
	pattern := []rune(equal)
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
	return "(?s)^" + string(prunes) + "$"
}

func Contain2RegexpString(contain string) string {
	pattern := []rune(contain)
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
	return "(?s)" + string(prunes) + ""
}
