package cmap

import (
	"encoding/json"
	"sync"

	"github.com/spf13/cast"
)

var SHARD_COUNT = 32

// A "thread" safe map of type string:Anything.
// To avoid lock bottlenecks this map is dived to several (SHARD_COUNT) map shards.
type ConcurrentMap[K comparable, V any] []*ConcurrentMapShared[K, V]

// A "thread" safe string to anything map.
type ConcurrentMapShared[K comparable, V any] struct {
	items        map[K]V
	sync.RWMutex // Read Write mutex, guards access to internal map.
}

// Creates a new 32 shards concurrent map.
// 生成 32 个碎片化的 支持并发的 map，适用于高并发大数据场景
func New[K comparable, V any](data ...map[K]V) ConcurrentMap[K, V] {
	m := NewShards[K, V](SHARD_COUNT, data...)
	return m
}

// Creates a new shards concurrent map.
// 生成指定碎片数量的碎片化的 支持并发的 map，根据需要定制使用
func NewShards[K comparable, V any](shardcount int, data ...map[K]V) ConcurrentMap[K, V] {
	m := make(ConcurrentMap[K, V], shardcount)
	for i := 0; i < shardcount; i++ {
		m[i] = &ConcurrentMapShared[K, V]{items: make(map[K]V)}
	}
	m.PutAll(data...)
	return m
}

// Creates a new single concurrent map.
// 生成单个 支持并发的 map，适用于小数据量、低并发度的场景
func NewSingle[K comparable, V any](data ...map[K]V) ConcurrentMap[K, V] {
	m := NewShards[K, V](1, data...)
	return m
}

// GetShard returns shard under given key
func (m ConcurrentMap[K, V]) GetShard(key K) *ConcurrentMapShared[K, V] {
	lenm := len(m)
	if lenm == 1 {
		return m[0]
	}
	return m[uint(fnv32(key))%uint(lenm)]
}

func (m ConcurrentMap[K, V]) MSet(data map[K]V) {
	for key, value := range data {
		shard := m.GetShard(key)
		shard.Lock()
		shard.items[key] = value
		shard.Unlock()
	}
}

func (m ConcurrentMap[K, V]) PutAll(data ...map[K]V) {
	for _, md := range data {
		m.MSet(md)
	}
}

// Sets the given value under the specified key.
func (m ConcurrentMap[K, V]) Set(key K, value V) {
	// Get map shard.
	shard := m.GetShard(key)
	shard.Lock()
	shard.items[key] = value
	shard.Unlock()
}

// Insert or Update - updates existing element or inserts a new one using UpsertCb
func (m ConcurrentMap[K, V]) Upsert(key K, value V, cb func(exist bool, valueInMap V, newValue V) V) (res V) {
	shard := m.GetShard(key)
	shard.Lock()
	v, ok := shard.items[key]
	res = cb(ok, v, value)
	shard.items[key] = res
	shard.Unlock()
	return res
}

// Sets the given value under the specified key if no value was associated with it.
func (m ConcurrentMap[K, V]) SetIfAbsent(key K, new_vlaue_func func() V) bool {
	// Get map shard.
	shard := m.GetShard(key)
	shard.Lock()
	_, ok := shard.items[key]
	if !ok {
		shard.items[key] = new_vlaue_func()
	}
	shard.Unlock()
	return !ok
}

// 返回值 err 新建函数返回错误
func (m ConcurrentMap[K, V]) GetWithNew(key K, new_vlaue_func func() (V, error)) (v V, err error) {
	// Get map shard.
	shard := m.GetShard(key)
	shard.RLock()
	v, ok := shard.items[key]
	shard.RUnlock()
	if !ok {
		shard.Lock()
		defer shard.Unlock()
		v, ok = shard.items[key]
		if !ok {
			v, err = new_vlaue_func()
			if err != nil {
				return
			}
			shard.items[key] = v
		}
	}
	return
}

// Get retrieves an element from map under given key.
func (m ConcurrentMap[K, V]) GetIFPresent(key K) V {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	// Get item from shard.
	val, _ := shard.items[key]
	shard.RUnlock()
	return val
}

// Get retrieves an element from map under given key.
func (m ConcurrentMap[K, V]) Get(key K) (V, bool) {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	// Get item from shard.
	val, ok := shard.items[key]
	shard.RUnlock()
	return val, ok
}

// Count returns the number of elements within the map.
func (m ConcurrentMap[K, V]) Count() int {
	return m.Len()
}

// returns the number of elements within the map.
func (m ConcurrentMap[K, V]) Len() int {
	count := 0
	for _, shard := range m {
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// Looks up an item under specified key
func (m ConcurrentMap[K, V]) Has(key K) bool {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	// See if element is within shard.
	_, ok := shard.items[key]
	shard.RUnlock()
	return ok
}

// Remove removes an element from the map.
func (m ConcurrentMap[K, V]) Remove(key K) (v V) {
	// Try to get shard.
	shard := m.GetShard(key)
	shard.Lock()
	v = shard.items[key]
	delete(shard.items, key)
	shard.Unlock()
	return
}

// RemoveCb locks the shard containing the key, retrieves its current value and calls the callback with those params
// If callback returns true and element exists, it will remove it from the map
// Returns the value returned by the callback (even if element was not present in the map)
func (m ConcurrentMap[K, V]) RemoveCb(key K, cb func(key K, v V, exists bool) bool) bool {
	// Try to get shard.
	shard := m.GetShard(key)
	shard.Lock()
	v, ok := shard.items[key]
	remove := cb(key, v, ok)
	if remove && ok {
		delete(shard.items, key)
	}
	shard.Unlock()
	return remove
}

// Pop removes an element from the map and returns it
func (m ConcurrentMap[K, V]) Pop(key K) (v V, exists bool) {
	// Try to get shard.
	shard := m.GetShard(key)
	shard.Lock()
	v, exists = shard.items[key]
	delete(shard.items, key)
	shard.Unlock()
	return v, exists
}

// IsEmpty checks if map is empty.
func (m ConcurrentMap[K, V]) IsEmpty() bool {
	return m.Count() == 0
}

// Used by the Iter & IterBuffered functions to wrap two variables together over a channel,
type Tuple[K comparable, V any] struct {
	Key K
	Val V
}

// IterBuffered returns a buffered iterator which could be used in a for range loop.
func (m ConcurrentMap[K, V]) IterBuffered() <-chan Tuple[K, V] {
	chans := snapshot(m)
	total := 0
	for _, c := range chans {
		total += cap(c)
	}
	ch := make(chan Tuple[K, V], total)
	go fanIn(chans, ch)
	return ch
}

// Clear removes all items from map.
func (m ConcurrentMap[K, V]) Clear() {
	for item := range m.IterBuffered() {
		m.Remove(item.Key)
	}
}

// Returns a array of channels that contains elements in each shard,
// which likely takes a snapshot of `m`.
// It returns once the size of each buffered channel is determined,
// before all the channels are populated using goroutines.
func snapshot[K comparable, V any](m ConcurrentMap[K, V]) (chans []chan Tuple[K, V]) {
	chans = make([]chan Tuple[K, V], len(m))
	wg := sync.WaitGroup{}
	wg.Add(len(m))
	// Foreach shard.
	for index, shard := range m {
		go func(index int, shard *ConcurrentMapShared[K, V]) {
			// Foreach key, value pair.
			shard.RLock()
			chans[index] = make(chan Tuple[K, V], len(shard.items))
			wg.Done()
			for key, val := range shard.items {
				chans[index] <- Tuple[K, V]{key, val}
			}
			shard.RUnlock()
			close(chans[index])
		}(index, shard)
	}
	wg.Wait()
	return chans
}

// fanIn reads elements from channels `chans` into channel `out`
func fanIn[K comparable, V any](chans []chan Tuple[K, V], out chan Tuple[K, V]) {
	wg := sync.WaitGroup{}
	wg.Add(len(chans))
	for _, ch := range chans {
		go func(ch chan Tuple[K, V]) {
			for t := range ch {
				out <- t
			}
			wg.Done()
		}(ch)
	}
	wg.Wait()
	close(out)
}

// Items returns all items as map[string]interface{}
func (m ConcurrentMap[K, V]) Items() map[K]V {
	tmp := make(map[K]V)

	// Insert items to temporary map.
	for item := range m.IterBuffered() {
		tmp[item.Key] = item.Val
	}

	return tmp
}

// Callback based iterator, cheapest way to read
// all elements in a map.
func (m ConcurrentMap[K, V]) IterCb(fn func(key K, v V)) {
	for idx := range m {
		shard := (m)[idx]
		shard.RLock()
		for key, value := range shard.items {
			fn(key, value)
		}
		shard.RUnlock()
	}
}

func (m ConcurrentMap[K, V]) Fetch(fn func(key K, v V) bool) {
	for idx := range m {
		shard := (m)[idx]
		shard.RLock()
		for key, value := range shard.items {
			if !fn(key, value) {
				break
			}
		}
		shard.RUnlock()
	}
}

// Keys returns all keys as []string
func (m ConcurrentMap[K, V]) Keys() []K {
	count := m.Count()
	ch := make(chan K, count)
	go func() {
		// Foreach shard.
		wg := sync.WaitGroup{}
		wg.Add(len(m))
		for _, shard := range m {
			go func(shard *ConcurrentMapShared[K, V]) {
				// Foreach key, value pair.
				shard.RLock()
				for key := range shard.items {
					ch <- key
				}
				shard.RUnlock()
				wg.Done()
			}(shard)
		}
		wg.Wait()
		close(ch)
	}()

	// Generate keys
	keys := make([]K, 0, count)
	for k := range ch {
		keys = append(keys, k)
	}
	return keys
}

// Reviles ConcurrentMap "private" variables to json marshal.
func (m ConcurrentMap[K, V]) MarshalJSON() ([]byte, error) {
	// Create a temporary map, which will hold all item spread across shards.
	tmp := make(map[K]V)

	// Insert items to temporary map.
	for item := range m.IterBuffered() {
		tmp[item.Key] = item.Val
	}
	return json.Marshal(tmp)
}

func (m ConcurrentMap[K, V]) String() string {
	bs, _ := json.Marshal(m)
	return string(bs)
}

func fnv32(key any) uint32 {
	var skey string
	switch key := key.(type) {
	case int:
		return uint32(key)
	case uint:
		return uint32(key)
	case int64:
		return uint32(key)
	case int32:
		return uint32(key)
	case int16:
		return uint32(key)
	case int8:
		return uint32(key)
	case uint64:
		return uint32(key)
	case uint32:
		return key
	case uint16:
		return uint32(key)
	case uint8:
		return uint32(key)
	case float64:
		return uint32(key)
	case float32:
		return uint32(key)
	case bool:
		if key {
			return 1
		}
		return 0
	case nil:
		return 0
	case string:
		skey = key
	default:
		skey = cast.ToString(key)
	}
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	keyLength := len(skey)
	for i := 0; i < keyLength; i++ {
		hash *= prime32
		hash ^= uint32(skey[i])
	}
	return hash
}

// Concurrent map uses Interface{} as its value, therefor JSON Unmarshal
// will probably won't know which to type to unmarshal into, in such case
// we'll end up with a value of type map[string]interface{}, In most cases this isn't
// out value type, this is why we've decided to remove this functionality.
func (m *ConcurrentMap[K, V]) UnmarshalJSON(b []byte) (err error) {
	// Reverse process of Marshal.
	tmp := make(map[K]V)

	// Unmarshal into a single map.
	if err := json.Unmarshal(b, &tmp); err != nil {
		return nil
	}

	// foreach key,value pair in temporary map insert into our concurrent map.
	for key, val := range tmp {
		m.Set(key, val)
	}
	return nil
}
