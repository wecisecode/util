package sortedmap

import (
	"reflect"

	"github.com/spf13/cast"
)

func ToLinkedMap(amap map[interface{}]interface{}) *LinkedMap {
	return NewLinkedMap().PutAll(amap).(*LinkedMap)
}

func ToTreeMap(amap map[interface{}]interface{}) *TreeMap {
	return NewTreeMap().PutAll(amap).(*TreeMap)
}

func ToTreapMap(amap map[interface{}]interface{}) *TreapMap {
	return NewTreapMap().PutAll(amap).(*TreapMap)
}

func ToMap(smf Map) map[interface{}]interface{} {
	mm := map[interface{}]interface{}{}
	smf.Fetch(func(key, value interface{}) bool {
		if sm, ok := value.(Map); ok {
			value = ToMap(sm)
		}
		mm[key] = value
		return true
	})
	return mm
}

func ToStringMap(smf Map) map[string]interface{} {
	mm := map[string]interface{}{}
	smf.Fetch(func(key, value interface{}) bool {
		if sm, ok := value.(Map); ok {
			value = ToStringMap(sm)
		}
		mm[cast.ToString(key)] = value
		return true
	})
	return mm
}

func Merge(smt Map, smf Map) {
	smf.Fetch(func(key, value interface{}) bool {
		smt.Put(key, value)
		return true
	})
}

func MergeMap(smt Map, smfmap interface{}) {
	if reflect.TypeOf(smfmap).Kind() == reflect.Map {
		smfv := reflect.ValueOf(smfmap)
		for _, k := range smfv.MapKeys() {
			v := smfv.MapIndex(k).Interface()
			smt.Put(k.Interface(), v)
		}
	} else {
		panic("smftmap should be map only")
	}
}

// force 强制合并不同子类型的 slice,map 为 []interface{},LinkedMap[interface{}]interface{}
// 涉及深度 map 的并发，应用需自行处理同步控制问题
func DeepCopyValue(smfvalue interface{}, force bool) interface{} {
	if smfvalue == nil {
		return nil
	}
	if reflect.TypeOf(smfvalue).Kind() == reflect.Slice {
		smfv := reflect.ValueOf(smfvalue)
		if force {
			nv := []interface{}{}
			for i := 0; i < smfv.Len(); i++ {
				v := smfv.Index(i).Interface()
				v = DeepCopyValue(v, force)
				nv = append(nv, v)
			}
			return nv
		} else {
			nv := reflect.MakeSlice(reflect.TypeOf(smfvalue), smfv.Len(), smfv.Len())
			for i := 0; i < smfv.Len(); i++ {
				v := smfv.Index(i).Interface()
				v = DeepCopyValue(v, force)
				nv.Index(i).Set(reflect.ValueOf(v))
			}
			return nv.Interface()
		}
	} else if reflect.TypeOf(smfvalue).Kind() == reflect.Map {
		smfv := reflect.ValueOf(smfvalue)
		if force {
			nv := NewLinkedMap()
			for _, k := range smfv.MapKeys() {
				v := smfv.MapIndex(k).Interface()
				v = DeepCopyValue(v, force)
				nv.Put(k.Interface(), v)
			}
			return nv
		} else {
			nv := reflect.MakeMap(reflect.TypeOf(smfvalue))
			for _, k := range smfv.MapKeys() {
				v := smfv.MapIndex(k).Interface()
				v = DeepCopyValue(v, force)
				nv.SetMapIndex(k, reflect.ValueOf(v))
			}
			return nv.Interface()
		}
	} else if smfv, ok := smfvalue.(SortedMap); ok {
		nv := smfv.DeepCopy()
		return nv
	} else {
		smfv := reflect.ValueOf(smfvalue)
		if smfv.CanInterface() {
			tm, b := smfv.Type().MethodByName("DeepCopy")
			if b {
				// 只有返回值类型与接口类型一致，没有参数的DeepCopy方法才能得到正确结果
				if tm.PkgPath == "" && tm.Type.NumIn() == 1 && tm.Type.NumOut() == 1 &&
					(smfv.Type() == tm.Type.Out(0) ||
						smfv.Type().AssignableTo(tm.Type.Out(0))) {
					om := smfv.MethodByName(tm.Name)
					if !om.IsNil() {
						args := []reflect.Value{}
						out := om.Call(args)
						nv := out[0].Interface()
						return nv
					}
				}
			}
		}
	}
	return smfvalue
}

// force 强制合并不同子类型的 slice,map 为 []interface{},LinkedMap[interface{}]interface{}
// 涉及深度 map 的并发，应用需自行处理同步控制问题
func DeepMergeValue(smtvalue interface{}, smfvalue interface{}, force bool) interface{} {
	if smfvalue == nil {
		return DeepCopyValue(smtvalue, force)
	}
	if smtvalue == nil {
		return DeepCopyValue(smfvalue, force)
	}
	if reflect.TypeOf(smfvalue).Kind() == reflect.Slice {
		smfv := reflect.ValueOf(smfvalue)
		if reflect.TypeOf(smtvalue).Kind() == reflect.Slice {
			smtv := reflect.ValueOf(smtvalue)
			if force {
				nv := []interface{}{}
				for i := 0; i < smtv.Len(); i++ {
					v := smtv.Index(i).Interface()
					v = DeepCopyValue(v, force)
					nv = append(nv, v)
				}
				for i := 0; i < smfv.Len(); i++ {
					v := smfv.Index(i).Interface()
					v = DeepCopyValue(v, force)
					nv = append(nv, v)
				}
				return nv
			} else {
				nv := reflect.MakeSlice(reflect.TypeOf(smfvalue), smtv.Len()+smfv.Len(), smtv.Len()+smfv.Len())
				for i := 0; i < smtv.Len(); i++ {
					v := smtv.Index(i).Interface()
					v = DeepCopyValue(v, force)
					nv.Index(i).Set(reflect.ValueOf(v))
				}
				for i := 0; i < smfv.Len(); i++ {
					v := smfv.Index(i).Interface()
					v = DeepCopyValue(v, force)
					nv.Index(smtv.Len() + i).Set(reflect.ValueOf(v))
				}
				return nv.Interface()
			}
		}
	} else if reflect.TypeOf(smfvalue).Kind() == reflect.Map {
		smfv := reflect.ValueOf(smfvalue)
		if reflect.TypeOf(smtvalue).Kind() == reflect.Map {
			smtv := reflect.ValueOf(smtvalue)
			if force {
				nv := NewLinkedMap()
				for _, k := range smtv.MapKeys() {
					v := smtv.MapIndex(k).Interface()
					v = DeepCopyValue(v, force)
					nv.Put(k.Interface(), v)
				}
				for _, k := range smfv.MapKeys() {
					v := smfv.MapIndex(k).Interface()
					if ov, ok := nv.Get(k.Interface()); ok {
						v = DeepMergeValue(ov, v, force)
					} else {
						v = DeepCopyValue(v, force)
					}
					nv.Put(k.Interface(), v)
				}
				return nv
			} else {
				nv := reflect.MakeMap(reflect.TypeOf(smfvalue)) // 以新数据的KV类型创建新的map
				for _, k := range smtv.MapKeys() {              // 尽量保留原有KV数据
					v := smtv.MapIndex(k).Interface()
					v = DeepCopyValue(v, force)
					func() {
						defer func() {
							x := recover() // 忽略类型不匹配的原有KV数据
							if x != nil {
								// fmt.Println(x)
							}
						}()
						nv.SetMapIndex(k, reflect.ValueOf(v))
					}()
				}
				for _, k := range smfv.MapKeys() { // 新数据合并到旧数据
					v := smfv.MapIndex(k).Interface()
					ov := nv.MapIndex(k)
					if ov.IsValid() {
						v = DeepMergeValue(ov.Interface(), v, force)
					} else {
						v = DeepCopyValue(v, force)
					}
					nv.SetMapIndex(k, reflect.ValueOf(v))
				}
				return nv.Interface()
			}
		} else if smtv, ok := smtvalue.(SortedMap); force && ok {
			nv := NewLinkedMap()
			for _, k := range smtv.Keys() {
				v := smtv.GetValue(k)
				v = DeepCopyValue(v, force)
				nv.Put(k, v)
			}
			for _, k := range smfv.MapKeys() {
				v := smfv.MapIndex(k).Interface()
				if ov, ok := nv.Get(k.Interface()); ok {
					v = DeepMergeValue(ov, v, force)
				} else {
					v = DeepCopyValue(v, force)
				}
				nv.Put(k.Interface(), v)
			}
			return nv
		}
	} else if smfv, ok := smfvalue.(SortedMap); ok {
		if smtv, ok := smtvalue.(SortedMap); ok {
			nv := smtv.DeepCopy()
			DeepMerge(nv, smfv, force)
			return nv
		} else if force && reflect.TypeOf(smtvalue).Kind() == reflect.Map {
			smtv := reflect.ValueOf(smtvalue)
			nv := NewLinkedMap()
			for _, k := range smtv.MapKeys() {
				v := smtv.MapIndex(k).Interface()
				v = DeepCopyValue(v, force)
				nv.Put(k.Interface(), v)
			}
			for _, k := range smfv.Keys() {
				v := smfv.GetValue(k)
				if ov, ok := nv.Get(k); ok {
					v = DeepMergeValue(ov, v, force)
				} else {
					v = DeepCopyValue(v, force)
				}
				nv.Put(k, v)
			}
			return nv
		}
	}
	return DeepCopyValue(smfvalue, force)
}

// force 强制合并不同子类型的 slice,map 为 []interface{},LinkedMap[interface{}]interface{}
// 涉及深度 map 的并发，应用需自行处理同步控制问题
func DeepMerge(smt SortedMap, smf SortedMap, force bool) {
	smf.Fetch(func(key, value interface{}) bool {
		if smtvalue, ok := smt.Get(key); ok {
			value = DeepMergeValue(smtvalue, value, force)
		} else {
			value = DeepCopyValue(value, force)
		}
		smt.Put(key, value)
		return true
	})
}
