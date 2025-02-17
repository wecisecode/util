package parser

import (
	"fmt"

	"github.com/spf13/cast"
	"github.com/wecisecode/util/sortedmap"
	"gopkg.in/ini.v1"
)

// 为兼容已有 log.conf 的配置参数定义，将原参数映射为新参数
func OLogConfParse(ini_format_str ...string) (sm *sortedmap.LinkedMap, err error) {
	sm = sortedmap.NewLinkedMap()
	for _, ini_str := range ini_format_str {
		fcfg, err := ini.ShadowLoad([]byte(ini_str))
		if err != nil {
			return nil, err
		}
		var default_sm_sect = sortedmap.NewLinkedMap()
		var log_sm_sect *sortedmap.LinkedMap
		for _, section_name := range fcfg.SectionStrings() {
			section, _ := fcfg.GetSection(section_name)
			sm_sect := sortedmap.NewLinkedMap()
			for _, key := range section.KeyStrings() {
				values := section.Key(key).ValueWithShadows()
				sm_sect.Put(key, values)
			}
			if section_name == "default" {
				// 合并 size 和 unit
				if unit := cast.ToString(sm_sect.GetValue("unit")); unit != "" {
					if size := cast.ToString(sm_sect.GetValue("size")); size != "" {
						sm_sect.Put("size", fmt.Sprint(size, unit))
					}
					sm_sect.Delete("unit")
				}
				// 转换 daliy 为 scroll
				if daily := cast.ToString(sm_sect.GetValue("daily")); daily != "" {
					if cast.ToBool(daily) {
						sm_sect.Put("scroll", "24h")
					} else {
						sm_sect.Put("scroll", "-1")
					}
					sm_sect.Delete("daily")
				}
				default_sm_sect = sm_sect
			} else if section_name == "log" {
				log_sm_sect = sm_sect
			} else {
				sm.Put(section_name, sm_sect)
			}
		}
		if log_sm_sect != nil {
			default_sm_sect.PutAll(log_sm_sect)
		}
		if default_sm_sect.Len() > 0 {
			sm.Put("log", default_sm_sect)
		}
	}
	return sm, nil
}
