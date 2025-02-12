package sortedmap

type Map interface {
	// Put will set (or replace) a value for a key. If the key was new, then true
	// will be returned. The returned value will be false if the value was replaced
	// (even if the value was the same).
	Put(key interface{}, value interface{}) bool
	Delete(key interface{}) bool
	Get(key interface{}, defaultValue ...interface{}) (interface{}, bool)
	GetValue(key interface{}, defaultValue ...interface{}) interface{}
	Has(key interface{}) bool
	Len() int
	Keys() []interface{}
	Values() []interface{}
	Fetch(p func(key interface{}, value interface{}) bool)
}

type SortedMap interface {
	Map
	//
	Copy() SortedMap
	// 深层复制，所有 slice，map，及所有实现了 DeepCopy 接口函数的数据 都将被递归复制
	DeepCopy() SortedMap
	// 参数 amap 只能是 map 或 SortedMap 类型, 为初始化使用方便返回当前对象自身
	PutAll(amap interface{}) SortedMap
	Clear()
	FirstItem() *MapItem
	LastItem() *MapItem
	FetchReverse(p func(key interface{}, value interface{}) bool)
	FetchRange(from interface{}, to interface{}, p func(key interface{}, value interface{}) bool, reverse bool)
	//
	UnmarshalJSON(bs []byte) (err error)
	MarshalJSON() ([]byte, error)
	String() string
}
