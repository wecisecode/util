package parser

import (
	"regexp"

	"github.com/wecisecode/util/sortedmap"
	"gopkg.in/ini.v1"
)

var regxvalueskey = regexp.MustCompile(`^var\s*\"([^\"]*)\"$`)

func IniParse(ini_format_str ...string) (sm *sortedmap.LinkedMap, err error) {
	sm = sortedmap.NewLinkedMap()
	for _, ini_str := range ini_format_str {
		fcfg, err := ini.ShadowLoad([]byte(ini_str))
		if err != nil {
			return nil, err
		}
		for _, section_name := range fcfg.SectionStrings() {
			section, _ := fcfg.GetSection(section_name)
			sm_sect := sortedmap.NewLinkedMap()
			for _, key := range section.KeyStrings() {
				values := section.Key(key).ValueWithShadows()
				sm_sect.Put(key, values)
			}
			valueskey := regxvalueskey.FindStringSubmatch(section_name)
			if values := sm_sect.GetValue("value"); values != nil && len(valueskey) > 1 && valueskey[1] != "" {
				sm.Put(valueskey[1], values)
				if sm_sect.Len() > 1 {
					sm.Put(section_name, sm_sect)
				}
			} else {
				sm.Put(section_name, sm_sect)
			}
		}
	}
	return sm, nil
}
