package parser

import (
	"regexp"
	"strings"

	"github.com/wecisecode/util/sortedmap"
)

func KVmParse(kvs ...string) (sm *sortedmap.LinkedMap, err error) {
	sm = sortedmap.NewLinkedMap()
	kvlist := ArgsParse(kvs)
	kvms := []interface{}{}
	for _, kv := range kvlist {
		k := kv.Key
		if k == "" {
			k = kv.Val
		}
		kvms = append(kvms, sortedmap.NewLinkedMap().PutAll(map[string]string{k: kv.Val}))
	}
	sm.Put("", kvms)
	return
}

type KV struct{ Key, Val string }

func ArgsParse(args []string) (kvs []*KV) {
	argk := ""
	argv := ""
	for _, arg := range args {
		if argk != "" {
			argv = arg
		} else if regexp.MustCompile(`^\--[^=]+=.*$`).MatchString(arg) {
			kv := strings.SplitN(arg[2:], "=", 2)
			argk = kv[0]
			argv = kv[1]
		} else if regexp.MustCompile(`^\--[^=]+$`).MatchString(arg) {
			argk = arg[2:]
			continue
		} else if regexp.MustCompile(`^\-[^=]+=.*$`).MatchString(arg) {
			kv := strings.SplitN(arg[1:], "=", 2)
			argk = kv[0]
			argv = kv[1]
		} else if regexp.MustCompile(`^\-[^=]+$`).MatchString(arg) {
			argk = arg[1:]
			continue
		} else {
			kv := strings.SplitN(arg, "=", 2)
			if len(kv) == 2 {
				argk = kv[0]
				argv = kv[1]
			} else {
				argk = ""
				argv = arg
			}
		}
		kvs = append(kvs, &KV{argk, argv})
		argk, argv = "", ""
	}
	if argk != "" {
		kvs = append(kvs, &KV{argk, argv})
	}
	return
}
