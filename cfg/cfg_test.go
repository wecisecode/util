package cfg_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	mc "github.com/wecisecode/util/cfg"
	"github.com/wecisecode/util/logger"
	"github.com/wecisecode/util/sortedmap"
)

var log = logger.New(&logger.Option{ConsoleLevel: logger.DEBUG})

func TestMConfig(t *testing.T) {

	var DEFAULT_CONFIG = &mc.CfgOption{Name: "m:default", Type: mc.INI_TEXT, Values: []string{`
[allows]
ip=127.0.0.1
`}}

	var etcd_env_file = "/matrix/etc/env"
	var ETCD_ENV_INI = &mc.CfgOption{Name: "m:etcd", Type: mc.INI_ETCD, Values: []string{etcd_env_file}}

	var etcd_omdb_file = "/matrix/etc/omdb"
	var ETCD_OMDB_INI = &mc.CfgOption{Name: "m:etcd", Type: mc.INI_ETCD, Values: []string{etcd_omdb_file}}

	var ini_file = filepath.Join("/opt/matrix", "odbserver", "omdb.ini")
	var FILE_OMDB_INI = &mc.CfgOption{Name: "m:file", Type: mc.INI_FILE, Values: []string{ini_file}}

	var Environs = mc.Environs

	options := []*mc.CfgOption{
		DEFAULT_CONFIG,
		ETCD_ENV_INI,
		ETCD_OMDB_INI,
		mc.GetIniFileCfgOption("/opt/matrix/conf/log.conf"),
		mc.GetIniFileCfgOption("/opt/matrix/conf/odbserver/log.conf"),
		FILE_OMDB_INI,
	}
	var work_dir_ini_file = Environs.GetString("MCONFIG", "")
	if work_dir_ini_file != "" {
		options = append(options, mc.GetIniFileCfgOption(work_dir_ini_file))
	}
	options = append(options, mc.CFGOPTION_ENVS, mc.CFGOPTION_ARGS)
	//
	cfg := mc.MConfig(options...).
		WithLogger(log)

	cfg.OnChange(func() {
		log.Info(cfg.Info())
	})

	var ETCD_APP_JSON = &mc.CfgOption{Name: "m:etcd", Type: mc.JSON_ETCD, Values: []string{"/matrix/apps/telemetry/*/*.js*n"},
		Parser: func(values ...string) (sm *sortedmap.LinkedMap, err error) {
			sm = sortedmap.NewLinkedMap()
			for _, json_str := range values {
				if json_str != "" {
					// 配置文件定义的结构
					type aserver struct {
						IP          string   `json:"ip"`
						Enable      bool     `json:"enable"`
						PerfEnable  bool     `json:"perfenable"`
						SensorPaths []string `json:"sensorpaths"`
					}
					type aconfig struct {
						Domain   string     `json:"domain"`
						Rule     string     `json:"rule"`
						PerfRule string     `json:"perfrule"`
						Servers  []*aserver `json:"servers"`
					}
					acfg := &aconfig{}
					err = json.Unmarshal([]byte(json_str), acfg)
					if err != nil {
						return
					}
					// 将配置文件定义的结构转换为程序逻辑使用的配置索引结构
					rm := sortedmap.NewLinkedMap()
					dm := sortedmap.NewLinkedMap()
					for _, server := range acfg.Servers {
						if server.Enable {
							cm := sortedmap.NewLinkedMap()
							for _, sensorpath := range server.SensorPaths {
								rules := []string{acfg.Rule}
								if server.PerfEnable && acfg.PerfRule != "" && acfg.PerfRule != acfg.Rule {
									rules = append(rules, acfg.PerfRule)
								}
								cm.Put(sensorpath, rules)
							}
							dm.Put(server.IP, cm)
						}
					}
					rm.Put(acfg.Domain, dm)
					jm := sortedmap.NewLinkedMap()
					err = jm.UnmarshalJSON([]byte(json_str))
					if err != nil {
						return
					}
					am := sortedmap.NewLinkedMap()
					am.Put("rule", rm)
					am.Put("ocfg", jm)
					sortedmap.DeepMerge(sm, am, true)
				}
			}
			return
		},
	}

	cfg = mc.MConfig(ETCD_APP_JSON).
		WithLogger(log)

	cfg.OnChange(func() {
		log.Info(cfg.Info())
	})

	fmt.Println("ok")
	time.Sleep(1 * time.Hour)
}
