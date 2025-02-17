package sortedmap

import (
	"encoding/json"
	"math/rand"
	"sync"
)

type TreapMap struct {
	treap *Treap
	sync.RWMutex
	mm map[interface{}]interface{}
}

var _ SortedMap = &TreapMap{}

func NewTreapMap() *TreapMap {
	return &TreapMap{treap: NewTreap(MapItemCompare), mm: map[interface{}]interface{}{}}
}

func (m *TreapMap) PutAll(amap interface{}) SortedMap {
	if sm, ok := amap.(SortedMap); ok {
		Merge(m, sm)
	} else {
		MergeMap(m, amap)
	}
	return m
}

func (m *TreapMap) Copy() SortedMap {
	sm := NewTreapMap()
	Merge(sm, m)
	return sm
}

func (m *TreapMap) DeepCopy() SortedMap {
	sm := NewTreapMap()
	DeepMerge(sm, m, true)
	return sm
}

func (m *TreapMap) FirstItem() *MapItem {
	mi := m.treap.Min()
	if mi == nil {
		return nil
	}
	return mi.(*MapItem)
}

func (m *TreapMap) LastItem() *MapItem {
	mi := m.treap.Max()
	if mi == nil {
		return nil
	}
	return mi.(*MapItem)
}

func (m *TreapMap) Get(key interface{}, defaultValue ...interface{}) (interface{}, bool) {
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

func (m *TreapMap) GetValue(key interface{}, defaultValue ...interface{}) interface{} {
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

func (m *TreapMap) Has(key interface{}) bool {
	m.RLock()
	defer m.RUnlock()
	_, ok := m.mm[key]
	return ok
}

func (m *TreapMap) Put(key interface{}, value interface{}) bool {
	m.Lock()
	defer m.Unlock()
	_, exist := m.mm[key]
	m.treap = m.treap.Upsert(Item(&MapItem{key, value}), rand.Int())
	m.mm[key] = value
	return !exist
}

func (m *TreapMap) Delete(key interface{}) (didDeleted bool) {
	m.Lock()
	defer m.Unlock()
	_, didDeleted = m.mm[key]
	if didDeleted {
		m.treap = m.treap.Delete(key)
		delete(m.mm, key)
	}
	return
}

func (m *TreapMap) Clear() {
	m.Lock()
	defer m.Unlock()
	m.treap = NewTreap(MapItemCompare)
	m.mm = map[interface{}]interface{}{}
}

func (m *TreapMap) Len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.mm)
}

func (m *TreapMap) Keys() []interface{} {
	keys := []interface{}{}
	m.treap.VisitAscend(m.treap.Min, func(i Item) bool {
		if mi, ok := i.(*MapItem); ok {
			keys = append(keys, mi.Key)
		}
		return true
	})
	return keys
}

func (m *TreapMap) Values() []interface{} {
	vals := []interface{}{}
	m.treap.VisitAscend(m.treap.Min, func(i Item) bool {
		if mi, ok := i.(*MapItem); ok {
			vals = append(vals, mi.Value)
		}
		return true
	})
	return vals
}

func (m *TreapMap) Fetch(p func(key interface{}, value interface{}) bool) {
	m.fetch(p, false)
}

func (m *TreapMap) FetchReverse(p func(key interface{}, value interface{}) bool) {
	m.fetch(p, true)
}

func (m *TreapMap) fetch(p func(key interface{}, value interface{}) bool, reverse bool) {
	if len(m.mm) == 0 {
		return
	}
	if reverse {
		m.treap.VisitDescend(m.treap.Max, func(i Item) bool {
			if mi, ok := i.(*MapItem); ok {
				return p(mi.Key, mi.Value)
			}
			return true
		})
	} else {
		m.treap.VisitAscend(m.treap.Min, func(i Item) bool {
			if mi, ok := i.(*MapItem); ok {
				return p(mi.Key, mi.Value)
			}
			return true
		})
	}
}

func (m *TreapMap) FetchRange(from interface{}, to interface{}, p func(key interface{}, value interface{}) bool, reverse bool) {
	if len(m.mm) == 0 {
		return
	}
	if reverse {
		if to == nil {
			to = m.treap.Max()
		}
		m.treap.VisitDescend(to, func(i Item) bool {
			if from != nil && m.treap.compare(i, from) < 0 {
				return false
			}
			if mi, ok := i.(*MapItem); ok {
				return p(mi.Key, mi.Value)
			}
			return true
		})
	} else {
		if from == nil {
			from = m.treap.Min()
		}
		m.treap.VisitAscend(from, func(i Item) bool {
			if to != nil && m.treap.compare(i, to) > 0 {
				return false
			}
			if mi, ok := i.(*MapItem); ok {
				return p(mi.Key, mi.Value)
			}
			return true
		})
	}
}

func (m *TreapMap) UnmarshalJSON(bs []byte) (err error) {
	err = UnmarshalJSON(m, bs)
	return
}

func (m *TreapMap) MarshalJSON() ([]byte, error) {
	return MarshalJSON(m)
}

func (m *TreapMap) String() string {
	bs, _ := json.MarshalIndent(m, "", "    ")
	return string(bs)
}
