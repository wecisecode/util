package parser

import (
	"github.com/wecisecode/util/sortedmap"
	"gopkg.in/yaml.v3"
)

func YamlParse(yaml_format_str ...string) (sm *sortedmap.LinkedMap, err error) {
	sm = sortedmap.NewLinkedMap()
	for _, yaml_str := range yaml_format_str {
		m := map[string]interface{}{}
		err = yaml.Unmarshal([]byte(yaml_str), &m)
		if err != nil {
			return
		}
		sortedmap.DeepMerge(sm, sortedmap.NewLinkedMap().PutAll(m), true)
	}
	return
}
