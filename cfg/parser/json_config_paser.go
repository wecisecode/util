package parser

import "github.com/wecisecode/util/sortedmap"

func JsonParse(json_format_str ...string) (sm *sortedmap.LinkedMap, err error) {
	sm = sortedmap.NewLinkedMap()
	for _, json_str := range json_format_str {
		m := sortedmap.NewLinkedMap()
		err = m.UnmarshalJSON([]byte(json_str))
		if err != nil {
			return
		}
		sortedmap.DeepMerge(sm, m, true)
	}
	return
}
