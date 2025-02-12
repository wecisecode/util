package sortedmap

import (
	"container/list"
	"encoding/json"
	"sync"
)

type mapping map[interface{}]*list.Element

type Element struct {
	item    *MapItem
	element *list.Element
}

func newElement(e *list.Element) *Element {
	if e == nil {
		return nil
	}
	return &Element{
		element: e,
		item:    e.Value.(*MapItem),
	}
}

// Next returns the next element, or nil if it finished.
func (e *Element) Next() *Element {
	return newElement(e.element.Next())
}

// Prev returns the previous element, or nil if it finished.
func (e *Element) Prev() *Element {
	return newElement(e.element.Prev())
}

type LinkedMap struct {
	sync.RWMutex
	mapping
	entries *list.List
}

var _ SortedMap = &LinkedMap{}

func NewLinkedMap() *LinkedMap {
	sm := &LinkedMap{
		mapping: map[interface{}]*list.Element{},
		entries: list.New(),
	}
	return sm
}

func (m *LinkedMap) PutAll(amap interface{}) SortedMap {
	if sm, ok := amap.(Map); ok {
		Merge(m, sm)
	} else {
		MergeMap(m, amap)
	}
	return m
}

func (m *LinkedMap) Copy() SortedMap {
	sm := NewLinkedMap()
	Merge(sm, m)
	return sm
}

func (m *LinkedMap) DeepCopy() SortedMap {
	sm := NewLinkedMap()
	DeepMerge(sm, m, true)
	return sm
}

// Get returns the value for a key. If the key does not exist, the second return
// parameter will be false and the value will be nil or defaultValue.
func (m *LinkedMap) Get(key interface{}, defaultValue ...interface{}) (interface{}, bool) {
	m.RLock()
	defer m.RUnlock()
	value, ok := m.mapping[key]
	if ok {
		return value.Value.(*MapItem).Value, true
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], false
	}
	return nil, false
}

func (m *LinkedMap) GetValue(key interface{}, defaultValue ...interface{}) interface{} {
	m.RLock()
	defer m.RUnlock()
	value, ok := m.mapping[key]
	if ok {
		return value.Value.(*MapItem).Value
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return nil
}

// Put will set (or replace) a value for a key. If the key was new, then true
// will be returned. The returned value will be false if the value was replaced
// (even if the value was the same).
func (m *LinkedMap) Put(key, value interface{}) bool {
	m.Lock()
	defer m.Unlock()
	_, didExist := m.mapping[key]
	if !didExist {
		element := m.entries.PushBack(&MapItem{key, value})
		m.mapping[key] = element
	} else {
		m.mapping[key].Value.(*MapItem).Value = value
	}
	return !didExist
}

func (m *LinkedMap) Has(key interface{}) bool {
	m.RLock()
	defer m.RUnlock()
	_, ok := m.mapping[key]
	return ok
}

// GetElement returns the element for a key. If the key does not exist, the
// pointer will be nil.
func (m *LinkedMap) GetElement(key interface{}) *Element {
	m.RLock()
	defer m.RUnlock()
	return m.getElement(key)
}

func (m *LinkedMap) getElement(key interface{}) *Element {
	value, ok := m.mapping[key]
	if ok {
		element := value.Value.(*MapItem)
		return &Element{
			element: value,
			item:    element,
		}
	}
	return nil
}

// Len returns the number of elements in the map.
func (m *LinkedMap) Len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.mapping)
}

// Keys returns all of the keys in the order they were inserted. If a key was
// replaced it will retain the same position. To ensure most recently set keys
// are always at the end you must always Delete before Set.
func (m *LinkedMap) Keys() (keys []interface{}) {
	m.RLock()
	defer m.RUnlock()
	keys = make([]interface{}, m.Len())
	element := m.entries.Front()
	for i := 0; element != nil; i++ {
		keys[i] = element.Value.(*MapItem).Key
		element = element.Next()
	}
	return keys
}

// Values returns all of the values in the order they were inserted.
func (m *LinkedMap) Values() (values []interface{}) {
	m.RLock()
	defer m.RUnlock()
	values = make([]interface{}, m.Len())
	element := m.entries.Front()
	for i := 0; element != nil; i++ {
		values[i] = element.Value.(*MapItem).Value
		element = element.Next()
	}
	return values
}

// Delete will remove a key from the map. It will return true if the key was
// removed (the key did exist).
func (m *LinkedMap) Delete(key interface{}) (didDelete bool) {
	m.Lock()
	defer m.Unlock()
	element, ok := m.mapping[key]
	if ok {
		m.entries.Remove(element)
		delete(m.mapping, key)
	}
	return ok
}

func (m *LinkedMap) Clear() {
	m.Lock()
	defer m.Unlock()
	m.mapping = map[interface{}]*list.Element{}
	m.entries = list.New()
}

func (m *LinkedMap) FirstItem() *MapItem {
	m.RLock()
	defer m.RUnlock()
	el := m.first()
	if el == nil {
		return nil
	}
	return el.item
}

func (m *LinkedMap) LastItem() *MapItem {
	m.RLock()
	defer m.RUnlock()
	el := m.last()
	if el == nil {
		return nil
	}
	return el.item
}

// First will return the element that is the first (oldest Set element). If
// there are no elements this will return nil.
func (m *LinkedMap) First() *Element {
	m.RLock()
	defer m.RUnlock()
	return m.first()
}

func (m *LinkedMap) first() *Element {
	front := m.entries.Front()
	if front == nil {
		return nil
	}
	element := front.Value.(*MapItem)
	return &Element{
		element: front,
		item:    element,
	}
}

// Last will return the element that is the last (most recent Set element). If
// there are no elements this will return nil.
func (m *LinkedMap) Last() *Element {
	m.RLock()
	defer m.RUnlock()
	return m.last()
}

func (m *LinkedMap) last() *Element {
	back := m.entries.Back()
	if back == nil {
		return nil
	}
	element := back.Value.(*MapItem)
	return &Element{
		element: back,
		item:    element,
	}
}

func (m *LinkedMap) Fetch(p func(key interface{}, value interface{}) bool) {
	m.FetchRange(nil, nil, p, false)
}

func (m *LinkedMap) FetchReverse(p func(key interface{}, value interface{}) bool) {
	m.FetchRange(nil, nil, p, true)
}

func (m *LinkedMap) FetchRange(from interface{}, to interface{}, p func(key interface{}, value interface{}) bool, reverse bool) {
	m.RLock()
	defer m.RUnlock()
	if len(m.mapping) == 0 {
		return
	}
	var elto *Element
	if to == nil {
		elto = m.last()
	} else {
		elto = m.getElement(to)
	}
	var elfrom *Element
	if from == nil {
		elfrom = m.first()
	} else {
		elfrom = m.getElement(from)
	}
	if reverse {
		for elto != nil {
			if !p(elto.item.Key, elto.item.Value) {
				return
			}
			if elfrom.item.Key == elto.item.Key {
				return
			}
			elto = elto.Prev()
		}
	} else {
		for elfrom != nil {
			if !p(elfrom.item.Key, elfrom.item.Value) {
				return
			}
			if elfrom.item.Key == elto.item.Key {
				return
			}
			elfrom = elfrom.Next()
		}
	}
}

func (m *LinkedMap) UnmarshalJSON(bs []byte) (err error) {
	err = UnmarshalJSON(m, bs)
	return
}

func (m *LinkedMap) MarshalJSON() ([]byte, error) {
	return MarshalJSON(m)
}

func (m *LinkedMap) String() string {
	bs, _ := json.MarshalIndent(m, "", "    ")
	return string(bs)
}
