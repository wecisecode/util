package cfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cast"
	"github.com/wecisecode/util/cfg/parser"
	"github.com/wecisecode/util/cmap"
	"github.com/wecisecode/util/etcd"
	"github.com/wecisecode/util/logger"
	"github.com/wecisecode/util/mfmt"
	"github.com/wecisecode/util/mid"
	"github.com/wecisecode/util/sortedmap"
)

type CfgType int

const (
	baseCfgType CfgType = -1
	UserDefined CfgType = iota // 主要用于环境变量和命令行参数的解析
	KVS_TEXT                   // 主要用于环境变量和命令行参数的解析
	INI_TEXT
	INI_FILE
	INI_ETCD
	JSON_TEXT
	JSON_FILE
	JSON_ETCD
	YAML_TEXT
	YAML_FILE
	YAML_ETCD
	OLOG_CONF
)

type CfgParser func(values ...string) (sm *sortedmap.LinkedMap, err error)
type CfgInfo map[string]*sortedmap.LinkedMap
type CfgLoader func(parser CfgParser, values ...string) (sm <-chan *CfgInfo, err error)

type CfgOption struct {
	Name   string
	Type   CfgType
	Values []string
	Loader CfgLoader
	Parser CfgParser
	log    *mConfLog
	id     int32
}

var cfgOptionsmu sync.Mutex
var cfgOptions = map[string]*CfgOption{}
var cfgOptionId = int32(0x10000)

func GetLogConfCfgOption(filename string) (co *CfgOption) {
	co = getCfgOptionByKey("m:file:/" + filename)
	co.Type = OLOG_CONF
	co.Values = append(co.Values, filename)
	return
}

func GetIniFileCfgOption(filename string) (co *CfgOption) {
	co = getCfgOptionByKey("m:file:/" + filename)
	co.Type = INI_FILE
	co.Values = append(co.Values, filename)
	return
}

func getCfgOptionByKey(key string) (co *CfgOption) {
	cfgOptionsmu.Lock()
	defer cfgOptionsmu.Unlock()
	co = cfgOptions[key]
	if co == nil {
		co = &CfgOption{Name: key, Type: baseCfgType}
		cfgOptions[key] = co
	}
	return
}

func (co *CfgOption) String() string {
	if co.id == 0 {
		co.id = atomic.AddInt32(&cfgOptionId, 1)
	}
	s := co.Name
	if len(co.Values) == 1 && len(co.Values[0]) > 0 && len(co.Values[0]) < 100 && regexp.MustCompile(`^\S+$`).MatchString(co.Values[0]) {
		s += ":/" + co.Values[0]
	} else if len(co.Values) > 0 {
		s = fmt.Sprintf("%s:/%X", s, co.id)
	}
	return s
}

var absAppPath, _ = filepath.Abs(os.Args[0])
var DefaultAppName = regexp.MustCompile(`^(?:.*\/)?([^\/]+)(?:\.[^\.]*)?$`).ReplaceAllString(os.Args[0], "$1")
var DefaultAppDir = filepath.Dir(absAppPath)

var DEFAULT_OPTION = &CfgOption{"cfg", baseCfgType, nil, nil, nil, nil, 0}

//	func LogConf() *CfgOption {
//		return GetLogConfCfgOption(filepath.Join(mdir.GetConfDir(), "log.conf"))
//	}
//
//	func AppLogConf(appname string) *CfgOption {
//		if appname == "" {
//			appname = DefaultAppName
//		}
//		return GetLogConfCfgOption(filepath.Join(mdir.GetConfDir(), appname, "log.conf"))
//	}
// func AppConf(appname string) *CfgOption {
// 	if appname == "" {
// 		appname = DefaultAppName
// 	}

// 	return GetIniFileCfgOption(filepath.Join(mdir.GetConfDir(), appname, fmt.Sprint(filepath.Base(appname), ".conf")))
// }

func CwdAppConf(appname string) *CfgOption {
	if appname == "" {
		appname = DefaultAppName
	}
	return GetIniFileCfgOption(filepath.Join(fmt.Sprint(appname, ".conf")))
}

var CFGOPTION_ARGS = &CfgOption{Name: "m:args", Type: KVS_TEXT, Values: os.Args}
var CFGOPTION_ENVS = &CfgOption{Name: "m:envs", Type: KVS_TEXT, Values: os.Environ()}

type Configure interface {
	Name() string
	// 配置最后改变的时间
	Stamp() time.Time
	Option() *CfgOption
	// 通过程序设置改变配置信息
	Set(key string, value interface{})
	Get(key string, defaultvalue ...interface{}) interface{}
	GetStrings(key string, defaultvalue ...string) []string
	GetString(key string, defaultvalue ...string) string
	GetInt(key string, defaultvalue ...int) int
	GetBool(key string, defaultvalue ...bool) bool
	GetFloat(key string, defaultvalue ...float64) float64
	// 支持单位 byte kb mb gb tb pb eb， 默认为 byte
	GetBytsCount(key string, defaultvalue ...interface{}) int64
	// 支持单位 d 天 h 小时 m 分钟 s 秒 ms 毫秒 us 微秒 ns 纳秒，默认毫秒
	GetDuration(key string, defaultvalue ...interface{}) time.Duration
	Keys() []string
	Map() map[string]interface{}
	LinkedMap() *sortedmap.LinkedMap
	// 扁平化处理前的原始配置信息
	Original() *sortedmap.LinkedMap
	//
	LastConfig() Configure
	//
	Merge(cfg Configure)
	// 所有配置信息
	Info() string
	//
	OnChange(func()) int64
	RemoveChangeHandler(int64)
	// 通过 cfg.WithLogger 调整日志输出相关配置
	// 默认 cfg.log 不输出任何信息，仅缓存最后100条信息，待通过 cfg.WithLogger 配置日志时一起输出
	WithLogger(log ConfLog) Configure
	withLogger(log ConfLog, clearbuffer bool) Configure
	WithETCD(cli etcd.Client) Configure
	Load() Configure
	LogError(e error)
}

var CommandArgs Configure
var Environs Configure
var DefaultConfig Configure

func init() {
	CommandArgs = MConfig(CFGOPTION_ARGS)
	Environs = MConfig(CFGOPTION_ENVS)
	DefaultConfig = MConfig(CwdAppConf(DefaultAppName), CFGOPTION_ENVS, CFGOPTION_ARGS)
	DefaultConfig.withLogger(logger.New().WithConfig(DefaultConfig), false)
}

var clog = &mConfLog{}

var cfgmu sync.Mutex
var cfgmm = map[*CfgOption]*mConfig{}

func cachedConfigure(option *CfgOption, in_newcfg bool) *mConfig {
	if !in_newcfg {
		cfgmu.Lock()
		defer cfgmu.Unlock()
	}
	cfg := cfgmm[option]
	if cfg == nil {
		cfg = newConfig(option)
		cfgmm[option] = cfg
	}
	return cfg
}

// 获取配置信息
//
//	可以在调用此函数前通过 cfg.Logger 调整日志输出相关配置
//	默认配置：
//		当前工作目录下 与应用同名的 .conf 文件
//		环境变量
//		命令行参数
func MConfig(option ...*CfgOption) Configure {
	if len(option) == 0 {
		return DefaultConfig
	}
	return NewConfig(option...).Load()
}

func NewConfig(option ...*CfgOption) Configure {
	if len(option) == 0 {
		return DefaultConfig
	}
	cfg := newConfig(DEFAULT_OPTION)
	for i := 0; i < len(option); i++ {
		cfg.subConfigs = append(cfg.subConfigs, cachedConfigure(option[i], false))
	}
	return cfg
}

func (mc *mConfig) Load() Configure {
	mc.load()
	for _, cfg := range mc.subConfigs {
		mc.Merge(cfg.load())
	}
	mc.merge()
	return mc
}

type mChangeHandler struct {
	name string
	proc func()
}
type mConfig struct {
	name           string
	stamp          time.Time
	option         *CfgOption
	subConfigs     []*mConfig
	chetcdclient   chan etcd.Client
	etcdclient     etcd.Client
	stopped        chan struct{}
	loaded         bool
	basecfg        *sortedmap.LinkedMap // 基础配置信息，通过UnmarshalJSON导入
	mergeConfigure *sortedmap.LinkedMap // 其它配置信息，通过MConfig初始化或Merge并入
	setcfg         *sortedmap.LinkedMap // 程序设置的配置信息，通过Set设置
	allConfig      *sortedmap.LinkedMap // 融合后的配置信息，经过扁平化处理
	orgconfig      *sortedmap.LinkedMap // 原始配置信息，扁平化处理前的配置信息
	changehandlers cmap.ConcurrentMap[int64, *mChangeHandler]
	lastConfig     *mConfig
	log            *mConfLog
	applog         ConfLog
}

func newConfig(co *CfgOption) (mc *mConfig) {
	mc = &mConfig{
		name:           co.Name,
		stamp:          time.Now(),
		option:         co,
		subConfigs:     []*mConfig{},
		basecfg:        sortedmap.NewLinkedMap(),
		mergeConfigure: sortedmap.NewLinkedMap(),
		setcfg:         sortedmap.NewLinkedMap(),
		allConfig:      sortedmap.NewLinkedMap(),
		orgconfig:      nil,
		changehandlers: cmap.New[int64, *mChangeHandler](),
		lastConfig:     nil,
		log:            clog,
		applog:         nil,
	}
	return
}

func (mc *mConfig) load() Configure {
	if mc.option.Type == baseCfgType || mc.loaded {
		return mc
	}
	mc.loaded = true
	err := mc.loading()
	if err != nil {
		mc.log.Warn(mc.name, err)
	}
	return mc
}

func (mc *mConfig) loading() (err error) {
	loaderf, parserf := mc.getLoader(), mc.getParser()
	cfginfo, e := loaderf(parserf, mc.option.Values...)
	if e != nil {
		return e
	}
	// 等待首次加载完成
	cfg := <-cfginfo
	if cfg == nil {
		return
	}
	mc.cfgloaded(cfg, true)
	// 后续变化加载
	go func() {
		for {
			select {
			case cfg := <-cfginfo:
				if cfg == nil {
					// 终止
					return
				}
				mc.cfgloaded(cfg, false)
			}
		}
	}()
	return
}
func (mc *mConfig) cfgloaded(cfg *CfgInfo, newcfg bool) {
	if len(*cfg) == 0 {
		return
	}
	for k, v := range *cfg {
		if v != nil {
			if k == "" {
				mc.setbase(k, v)
			} else {
				topt := getCfgOptionByKey(k)
				tcfg := cachedConfigure(topt, newcfg)
				if tcfg.basecfg == nil || tcfg.basecfg.String() != v.String() {
					tcfg.setbase(k, v)
					tcfg.onChanged() // 首次执行不存在已注册的changeEvent，后续变化会激活已注册的changeEvent
				}
				mc.Merge(tcfg) // 首次Merge会注册并激活changeEvent，后续变化不会重复注册和激活changeEvent
			}
		} else {
			mc.mergeConfigure.Delete(getCfgOptionByKey(k))
		}
	}
	// 加载过程中的多个配置信息，以对应的Option.Name重新排序
	topts := mc.mergeConfigure.Keys()
	sort.Slice(topts, func(i, j int) bool {
		if topts[i].(*CfgOption).Name == topts[j].(*CfgOption).Name {
			return topts[i].(*CfgOption).String() < topts[j].(*CfgOption).String()
		}
		return topts[i].(*CfgOption).Name < topts[j].(*CfgOption).Name
	})
	for _, topt := range topts {
		tcfg, _ := mc.mergeConfigure.Get(topt)
		if tcfg != nil {
			mc.mergeConfigure.Delete(topt)
			mc.mergeConfigure.Put(topt, tcfg)
		}
	}
	mc.onChanged()
}

func (mc *mConfig) setbase(key string, info *sortedmap.LinkedMap) {
	if key != "" {
		mc.name = key
	}
	if info != nil {
		mc.basecfg = info
	}
}

func (mc *mConfig) getLoader() (loaderf CfgLoader) {
	loaderf = mc.option.Loader
	if loaderf == nil {
		switch mc.option.Type {
		case KVS_TEXT, INI_TEXT, JSON_TEXT, YAML_TEXT:
			loaderf = mc.loadFromText
		case OLOG_CONF, INI_FILE, JSON_FILE, YAML_FILE:
			loaderf = mc.loadFromFile
		case INI_ETCD, JSON_ETCD, YAML_ETCD:
			loaderf = mc.loadFromETCD
		default:
			panic("没有指定配置信息加载器")
		}
	}
	return
}

func (mc *mConfig) getParser() (parserf CfgParser) {
	parserf = mc.option.Parser
	if parserf == nil {
		switch mc.option.Type {
		case KVS_TEXT:
			parserf = parser.KVmParse
		case OLOG_CONF:
			parserf = parser.OLogConfParse
		case INI_TEXT, INI_FILE, INI_ETCD:
			parserf = parser.IniParse
		case JSON_TEXT, JSON_FILE, JSON_ETCD:
			parserf = parser.JsonParse
		case YAML_TEXT, YAML_FILE, YAML_ETCD:
			parserf = parser.YamlParse
		default:
			panic("没有指定配置信息解析器")
		}
	}
	return
}

func (mc *mConfig) Name() string {
	return mc.name
}

func (mc *mConfig) Stamp() time.Time {
	return mc.stamp
}

func (mc *mConfig) Option() *CfgOption {
	return mc.option
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	s, e := cast.ToStringE(v)
	if e != nil {
		return fmt.Sprint(v)
	}
	return s
}

func toStrings(v interface{}) []string {
	var a []string
	switch v := v.(type) {
	case []interface{}:
		for _, u := range v {
			a = append(a, toString(u))
		}
		return a
	case []string:
		return v
	case string:
		return []string{v}
	default:
		str := toString(v)
		return []string{str}
	}
}

func (mc *mConfig) Set(key string, value interface{}) {
	mc.setcfg.Put(key, value)
	mc.onChanged()
}

func (mc *mConfig) get(key string, defaultvalue ...interface{}) (v interface{}, ok bool) {
	keys := strings.Split(key, "|")
	for _, k := range keys {
		v, ok = mc.allConfig.Get(k)
		if ok {
			return
		}
	}
	return defaultvalue, false
}

func (mc *mConfig) Get(key string, defaultvalue ...interface{}) interface{} {
	v, _ := mc.get(key, defaultvalue...)
	return v
}

func (mc *mConfig) GetStrings(key string, defaultvalue ...string) []string {
	v, ok := mc.get(key)
	if !ok {
		return defaultvalue
	}
	return toStrings(v)
}

func (mc *mConfig) GetString(key string, defaultvalue ...string) string {
	v, ok := mc.get(key)
	if !ok {
		if len(defaultvalue) > 0 {
			return defaultvalue[len(defaultvalue)-1]
		}
	}
	switch v := v.(type) {
	case []interface{}:
		if len(v) > 0 {
			return toString(v[len(v)-1])
		}
		return ""
	case []string:
		if len(v) > 0 {
			return v[len(v)-1]
		}
		return ""
	}
	return toString(v)
}

func (mc *mConfig) GetInt(key string, defaultvalue ...int) int {
	v, ok := mc.get(key)
	if !ok {
		if len(defaultvalue) > 0 {
			return defaultvalue[len(defaultvalue)-1]
		}
	}
	switch v := v.(type) {
	case []interface{}:
		if len(v) > 0 {
			return cast.ToInt(v[len(v)-1])
		}
	case []string:
		if len(v) > 0 {
			return cast.ToInt(v[len(v)-1])
		}
	}
	return cast.ToInt(v)
}

func (mc *mConfig) GetFloat(key string, defaultvalue ...float64) float64 {
	v, ok := mc.get(key)
	if !ok {
		if len(defaultvalue) > 0 {
			return defaultvalue[len(defaultvalue)-1]
		}
	}
	switch v := v.(type) {
	case []interface{}:
		if len(v) > 0 {
			return cast.ToFloat64(v[len(v)-1])
		}
	case []string:
		if len(v) > 0 {
			return cast.ToFloat64(v[len(v)-1])
		}
	}
	return cast.ToFloat64(v)
}

func (mc *mConfig) GetBool(key string, defaultvalue ...bool) bool {
	v, ok := mc.get(key)
	if !ok {
		if len(defaultvalue) > 0 {
			return defaultvalue[len(defaultvalue)-1]
		}
	}
	switch v := v.(type) {
	case []interface{}:
		if len(v) > 0 {
			return cast.ToBool(v[len(v)-1])
		}
	case []string:
		if len(v) > 0 {
			return cast.ToBool(v[len(v)-1])
		}
	}
	return cast.ToBool(v)
}

func (mc *mConfig) GetBytsCount(key string, defaultvalue ...interface{}) (nv int64) {
	v := mc.GetString(key)
	nv = mfmt.ParseBytesCount(v)
	if nv == 0 {
		if len(defaultvalue) > 0 {
			v = cast.ToString(defaultvalue[len(defaultvalue)-1])
			nv = mfmt.ParseBytesCount(v)
		}
	}
	return
}

func (mc *mConfig) GetDuration(key string, defaultvalue ...interface{}) (nv time.Duration) {
	v := mc.GetString(key)
	nv = mfmt.ParseDuration(v)
	if nv == 0 {
		if len(defaultvalue) > 0 {
			v = cast.ToString(defaultvalue[len(defaultvalue)-1])
			nv = mfmt.ParseDuration(v)
		}
	}
	return
}

func (mc *mConfig) Keys() []string {
	return toStrings(mc.allConfig.Keys())
}

func (mc *mConfig) Map() map[string]interface{} {
	return sortedmap.ToStringMap(mc.allConfig)
}

func (mc *mConfig) LinkedMap() *sortedmap.LinkedMap {
	nsm := sortedmap.NewLinkedMap()
	sortedmap.DeepMerge(nsm, mc.allConfig, true)
	return nsm
}

func (mc *mConfig) Original() *sortedmap.LinkedMap {
	stamp := mc.Stamp().Format("2006-01-02 15:04:05.000000000")
	if mc.orgconfig == nil || mc.orgconfig.GetValue("time") != stamp {
		orgconfig := sortedmap.NewLinkedMap()
		orgconfig.Put("time", mc.Stamp().Format("2006-01-02 15:04:05.000000000"))
		orgconfig.Put("m:base", mc.basecfg)
		mc.mergeConfigure.Fetch(func(key, value interface{}) bool {
			orgconfig.Put(key, value.(Configure).Original())
			return true
		})
		// sortedmap.DeepMerge(orgconfig, mc.mergeConfigure, true)
		orgconfig.Put("m:set", mc.setcfg)
		mc.orgconfig = orgconfig
	}
	return mc.orgconfig
}

func (mc *mConfig) UnmarshalJSON(bs []byte) (err error) {
	sm := sortedmap.NewLinkedMap()
	err = sortedmap.UnmarshalJSON(sm, bs)
	if err == nil {
		mc.basecfg = sm
		mc.merge()
	}
	return
}

func (mc *mConfig) MarshalJSON() ([]byte, error) {
	sm := sortedmap.NewLinkedMap()
	sm.Put("time", mc.stamp.Format("2006-01-02 15:04:05.000000000"))
	sm.Put(mc.name, mc.basecfg)
	if mc.setcfg.Len() > 0 {
		sm.Put(mc.name+"+", mc.setcfg)
	}
	return sortedmap.MarshalJSON(sm)
}

func (mc *mConfig) copy() *mConfig {
	nmc := &mConfig{
		option:         mc.option,
		stamp:          mc.stamp,
		basecfg:        nil,
		mergeConfigure: nil,
		setcfg:         nil,
		allConfig:      mc.LinkedMap(),
		changehandlers: nil,
		lastConfig:     nil,
	}
	return nmc
}

// 将有层次的配置信息扁平化，子对象以 root.parent.child.etc 的形式扁平化
// 如有特别需求，可以通过Original获取原始配置信息
// 数组保留至最后一层，如下示意两种表达方式等价：
//
//	var _ = `{
//		a: [{
//			b: [{
//				c: [{
//					d: 1,
//				},{
//					d: 2,
//				}],
//				e: xx,
//				f: [[
//					111,
//					222,
//				],[
//					333,
//					444,
//				]],
//			},
//		},{
//			b: [{
//				c: [{
//					d: 3,
//				},{
//					d: 4,
//				}],
//				e: [
//					yy,
//					zz,
//				],
//				f: [[[
//					555,
//					666,
//				]]],
//			}],
//		}],
//	}` == `{
//		a.b.c.d: [
//			1,
//			2,
//			3,
//			4,
//		],
//		a.b.e: [
//			xx,
//			yy,
//			zz,
//		],
//		a.b.f: [
//			111,
//			222,
//			333,
//			444,
//			555,
//			666,
//		],
//	}`
func (mc *mConfig) mergeFlatting(retsm sortedmap.SortedMap, key string, value interface{}) {
	if value == nil {
		return
	}
	if sm, ok := value.(sortedmap.SortedMap); ok {
		if sm == nil {
			return
		}
		keyprefix := key
		if keyprefix != "" && keyprefix[len(keyprefix)-1:] != "." {
			keyprefix += "."
		}
		sm.Fetch(func(k, v interface{}) bool {
			skey := keyprefix + cast.ToString(k)
			mc.mergeFlatting(retsm, skey, v)
			return true
		})
	} else if vs, ok := value.([]string); ok {
		for _, v := range vs {
			mc.mergeFlatting(retsm, key, v)
		}
	} else if vs, ok := value.([]interface{}); ok {
		for _, v := range vs {
			mc.mergeFlatting(retsm, key, v)
		}
	} else {
		if ov, ok := retsm.Get(key); ok {
			v := append(ov.([]interface{}), value)
			retsm.Put(key, v)
		} else {
			v := []interface{}{value}
			retsm.Put(key, v)
		}
	}
}

func (mc *mConfig) merge() {
	mc.stamp = time.Now()
	sm := sortedmap.NewLinkedMap()
	mc.mergeFlatting(sm, "", mc.basecfg)
	mc.mergeConfigure.Fetch(func(k, v interface{}) bool {
		cfg := v.(Configure)
		mc.mergeFlatting(sm, "", cfg.LinkedMap())
		return true
	})
	mc.mergeFlatting(sm, "", mc.setcfg)
	mc.allConfig = sm
}

func (mc *mConfig) Merge(cfg Configure) {
	if mc.mergeConfigure.GetValue(cfg.Option()) == cfg {
		return
	}
	cfg.OnChange(func() {
		mc.onChanged()
	})
	mc.mergeConfigure.Put(cfg.Option(), cfg)
}

func (mc *mConfig) LastConfig() (cfg Configure) {
	return mc.lastConfig
}

func (mc *mConfig) RemoveChangeHandler(key int64) {
	mc.changehandlers.Remove(key)
}

func (mc *mConfig) OnChange(och func()) int64 {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		panic("为啥不ok")
	}
	if och != nil {
		fn := runtime.FuncForPC(reflect.ValueOf(och).Pointer()).Name()
		name := fmt.Sprint(fn, "[", path.Base(file), ":", line, "]")
		key := mid.UnixNano()
		mc.changehandlers.Set(key, &mChangeHandler{name: name, proc: och})
		och()
		return key
	}
	return 0
}

func (mc *mConfig) onChanged() {
	mc.merge()
	mc.changehandlers.IterCb(func(key int64, ch *mChangeHandler) {
		mc.log.Debug(mc.Option().String(), "notify on config changed to", ch.name)
		go ch.proc()
	})
}

func (mc *mConfig) Info() string {
	outputconfig := sortedmap.NewLinkedMap()
	outputconfig.Put("time", mc.Stamp().Format("2006-01-02 15:04:05.000000000"))
	outputconfig.Put(mc.name, mc.allConfig)
	// sortedmap.DeepMerge(outputconfig, mc.mergeConfigure, true)
	sortedmap.DeepMerge(outputconfig, mc.Original(), true)
	bs, err := json.MarshalIndent(outputconfig, "", "    ")
	if err != nil {
		mc.log.Error(err)
	}
	return string(bs)
}

func (mc *mConfig) WithLogger(log ConfLog) Configure {
	return mc.withLogger(log, true)
}

func (mc *mConfig) withLogger(log ConfLog, forceclearbuffer bool) Configure {
	if mc.applog != log {
		if mc.applog != nil {
			mc.log.RemoveLog(mc.applog)
		}
		if log != nil {
			mc.log.AppLog(log, forceclearbuffer)
		}
		mc.applog = log
	}
	return mc
}
func (mc *mConfig) LogError(e error) {
	mc.log.Error(e)
}
