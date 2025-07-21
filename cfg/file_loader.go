package cfg

import (
	"errors"
	"os"

	"github.com/wecisecode/util/filewatcher"
)

type configFileLoader struct {
	*mConfig
	parserf   CfgParser
	chcfginfo chan *CfgInfo
}

func (mc *configFileLoader) LoadConfigFile(filename string, err error) {
	key := "file:/" + filename
	if err != nil {
		if errors.Is(err, os.ErrInvalid) || errors.Is(err, os.ErrNotExist) {
			// 文件不存在按空文件处理
			sm, err := mc.parserf("")
			if err != nil {
				mc.log.Error("watching config file", filename, "error:", err)
				return
			}
			// 清空err
			err = nil
			mc.chcfginfo <- &CfgInfo{key: sm}
		} else {
			mc.log.Error("watching config file", filename, "error:", err)
			// 其它文件错误，保持原有配置信息不变
		}
		return
	}
	mc.log.Debug("watching", filename, "changed")
	bytes, err := os.ReadFile(filename)
	if err != nil {
		mc.log.Error("load config file", filename, "error:", err)
		// 读文件出错，保持原有配置信息不变
		return
	}
	mc.log.Debug("load config from", filename)
	sm, err := mc.parserf(string(bytes))
	if err != nil {
		mc.log.Error("watching config file", filename, "error:", err)
		return
	}
	mc.chcfginfo <- &CfgInfo{key: sm}
}

func (mc *mConfig) loadFromFile(parserf CfgParser, filenames ...string) (<-chan *CfgInfo, error) {
	chcfginfo := make(chan *CfgInfo, len(filenames))
	for _, filename := range filenames {
		var cfl = &configFileLoader{}
		cfl.mConfig = mc
		cfl.parserf = parserf
		cfl.chcfginfo = chcfginfo
		err := filewatcher.PollingWatchFile(filename, cfl.LoadConfigFile)
		if err != nil {
			return nil, err
		}
	}
	return chcfginfo, nil
}
