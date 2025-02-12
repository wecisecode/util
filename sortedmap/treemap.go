package sortedmap

import (
	"encoding/json"
	"sync"
)

type TreeMap struct {
	sync.RWMutex
	tree *redBlackTree
	mm   map[interface{}]interface{}
}

var _ SortedMap = &TreeMap{}

func NewTreeMap() *TreeMap {
	return &TreeMap{tree: NewRedBlackTree(MapItemCompare), mm: map[interface{}]interface{}{}}
}

func (m *TreeMap) PutAll(amap interface{}) SortedMap {
	if sm, ok := amap.(SortedMap); ok {
		Merge(m, sm)
	} else {
		MergeMap(m, amap)
	}
	return m
}

func (m *TreeMap) Copy() SortedMap {
	sm := NewTreapMap()
	Merge(sm, m)
	return sm
}

func (m *TreeMap) DeepCopy() SortedMap {
	sm := NewTreapMap()
	DeepMerge(sm, m, true)
	return sm
}

func (m *TreeMap) FirstItem() *MapItem {
	mi := m.tree.Min()
	if mi == nil {
		return nil
	}
	return mi.(*MapItem)
}

func (m *TreeMap) LastItem() *MapItem {
	mi := m.tree.Max()
	if mi == nil {
		return nil
	}
	return mi.(*MapItem)
}

func (m *TreeMap) Get(key interface{}, defaultValue ...interface{}) (interface{}, bool) {
	m.RLock()
	defer m.RUnlock()
	value, ok := m.mm[key]
	if ok {
		return value, true
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], false
	}
	return nil, false
}

func (m *TreeMap) GetValue(key interface{}, defaultValue ...interface{}) interface{} {
	m.RLock()
	defer m.RUnlock()
	value, ok := m.mm[key]
	if ok {
		return value
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return nil
}

func (m *TreeMap) Has(key interface{}) bool {
	m.RLock()
	defer m.RUnlock()
	_, ok := m.mm[key]
	return ok
}

func (m *TreeMap) Put(key interface{}, value interface{}) bool {
	m.Lock()
	defer m.Unlock()
	_, exist := m.mm[key]
	m.tree.Insert(Item(&MapItem{Key: key, Value: value}))
	m.mm[key] = value
	return !exist
}

func (m *TreeMap) Delete(key interface{}) (didDeleted bool) {
	m.Lock()
	defer m.Unlock()
	_, didDeleted = m.mm[key]
	if didDeleted {
		m.tree.Delete(Item(key))
		delete(m.mm, key)
	}
	return
}

func (m *TreeMap) Clear() {
	m.Lock()
	defer m.Unlock()
	m.tree = NewRedBlackTree(MapItemCompare)
	m.mm = map[interface{}]interface{}{}
}

func (m *TreeMap) Len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.mm)
}

func (m *TreeMap) Keys() []interface{} {
	m.RLock()
	defer m.RUnlock()
	keys := []interface{}{}
	m.tree.VisitAscend(m.tree.Min(), func(i Item) bool {
		if mi, ok := i.(*MapItem); ok {
			keys = append(keys, mi.Key)
		}
		return true
	})
	return keys
}

func (m *TreeMap) Values() []interface{} {
	m.RLock()
	defer m.RUnlock()
	vals := []interface{}{}
	m.tree.VisitAscend(m.tree.Min(), func(i Item) bool {
		if mi, ok := i.(*MapItem); ok {
			vals = append(vals, mi.Value)
		}
		return true
	})
	return vals
}

func (m *TreeMap) Fetch(p func(key interface{}, value interface{}) bool) {
	m.RLock()
	defer m.RUnlock()
	m.fetch(p, false)
}

func (m *TreeMap) FetchReverse(p func(key interface{}, value interface{}) bool) {
	m.RLock()
	defer m.RUnlock()
	m.fetch(p, true)
}

func (m *TreeMap) fetch(p func(key interface{}, value interface{}) bool, reverse bool) {
	if len(m.mm) == 0 {
		return
	}
	if reverse {
		m.tree.VisitDescend(m.tree.Max(), func(i Item) bool {
			if mi, ok := i.(*MapItem); ok {
				return p(mi.Key, mi.Value)
			}
			return true
		})
	} else {
		m.tree.VisitAscend(m.tree.Min(), func(i Item) bool {
			if mi, ok := i.(*MapItem); ok {
				return p(mi.Key, mi.Value)
			}
			return true
		})
	}
}

func (m *TreeMap) FetchRange(from interface{}, to interface{}, p func(key interface{}, value interface{}) bool, reverse bool) {
	m.RLock()
	defer m.RUnlock()
	if len(m.mm) == 0 {
		return
	}
	if reverse {
		if to == nil {
			to = m.tree.Max()
		}
		m.tree.VisitDescend(to, func(i Item) bool {
			if from != nil && m.tree.compare(i, from) < 0 {
				return false
			}
			if mi, ok := i.(*MapItem); ok {
				return p(mi.Key, mi.Value)
			}
			return true
		})
	} else {
		if from == nil {
			from = m.tree.Min()
		}
		m.tree.VisitAscend(from, func(i Item) bool {
			if to != nil && m.tree.compare(i, to) > 0 {
				return false
			}
			if mi, ok := i.(*MapItem); ok {
				return p(mi.Key, mi.Value)
			}
			return true
		})
	}
}

func (m *TreeMap) UnmarshalJSON(bs []byte) (err error) {
	err = UnmarshalJSON(m, bs)
	return
}

func (m *TreeMap) MarshalJSON() ([]byte, error) {
	return MarshalJSON(m)
}

func (m *TreeMap) String() string {
	bs, _ := json.MarshalIndent(m, "", "    ")
	return string(bs)
}
