package deepcopy

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/spf13/cast"
)

func MapToObject(m map[string]interface{}, obj interface{}, ignoreTypeMatchError bool) error {
	objp := reflect.ValueOf(obj)
	if objp.Kind() != reflect.Ptr {
		return errors.New("参数必须为指针")
	}
	if objp.IsNil() {
		return errors.New("参数不能为空")
	}
	objv := objp.Elem()
	if objv.Kind() != reflect.Struct {
		return errors.New("参数必须为指向结构的指针")
	}
	return toObject(m, objv, ignoreTypeMatchError)
}

func toObject(m map[string]interface{}, objv reflect.Value, ignoreTypeMatchError bool) error {
	i := 0
	objt := objv.Type()
	if objv.NumField() > 0 {
		ovf := objv.Field(i)
		ovft := ovf.Type()
		if ovft.Kind() == reflect.Ptr && ovf.Elem().Kind() == reflect.Struct {
			err := toObject(m, ovf.Elem(), ignoreTypeMatchError)
			if err != nil {
				return err
			}
			i++
		}
	}
	for ; i < objv.NumField(); i++ {
		ovf := objv.Field(i)
		otf := objt.Field(i)
		ovft := ovf.Type()
		key := otf.Name
		if ovf.CanSet() && ovf.CanInterface() {
			// tag := otf.Tag.Get("json")
			// fmt.Println(key, "[", ovft.Kind(), ",", tag, "]", ovf.Interface(), " -> ")
			v, b := m[key]
			if b {
				vv, e := CastValue(ovft, v)
				if e == nil {
					ovf.Set(vv)
				} else if !ignoreTypeMatchError {
					return e
				} // else continue
			}
		}
	}
	return nil
}

// Interface for delegating copy process to type
type DeepCopyInterface interface {
	DeepCopy() DeepCopyInterface
}

type DeepCopyOption int

// 针对包含私有属性，同时又没有提供DeepCopy接口的结构体的处理方式
const (
	DC_OVPA_COPY_ADDRESS      DeepCopyOption = iota // 不新建对象，保留原对象，不做深度复制处理
	DC_OVPA_SKIP_PRIVATE_ATTR                       // 新建对象，跳过私有属性的处理
	DC_OVPA_PANIC                                   // 异常退出
)

var DC_OVPA_DEFAULT = DC_OVPA_SKIP_PRIVATE_ATTR

type Receiver map[string]map[string]string

func (r Receiver) String() string {
	s := ""
	for k, m := range r {
		if s != "" {
			s += "\n"
		}
		s += k + ":"
		for k, v := range m {
			s += "\n"
			s += "  " + k + ": " + v
		}
	}
	return s
}

func dcoption(option ...interface{}) (DeepCopyOption, Receiver) {
	opt := DC_OVPA_DEFAULT
	receiver := Receiver{}
	for _, o := range option {
		switch ov := o.(type) {
		case DeepCopyOption:
			opt = ov
		case *Receiver:
			receiver = *ov
		case Receiver:
			receiver = ov
		case map[string]map[string]string:
			receiver = ov
		}
	}
	return opt, receiver
}

// DeepCopy creates a deep copy of whatever is passed to it and returns the copy
// in an interface{}.  The returned value will need to be asserted to the
// correct type.
// 可选参数：
// option    可指定 DeepCopyOption 选项，DC_OVPA_COPY_ADDRESS，DC_OVPA_SKIP_PRIVATE_ATTR，DC_OVPA_PANIC
// receiver  可指定 map[string]string，收集包含私有属性的处理对象相关信息
func DeepCopy(src interface{}, option ...interface{}) interface{} {
	if src == nil {
		return nil
	}

	// Make the interface a reflect.Value
	original := reflect.ValueOf(src)

	// Make a copy of the same type as the original.
	cpy := reflect.New(original.Type()).Elem()
	// cpyo := cpy.Interface()

	opt, receiver := dcoption(option...)

	// Recursively copy the original.
	CopyRecursive(original, cpy, opt, receiver)

	if opt == DC_OVPA_PANIC && len(receiver) > 0 {
		panic(fmt.Sprint(receiver))
	}
	// Return the copy as an interface.
	return cpy.Interface()
}

func DeepCopyE(src interface{}, option ...interface{}) (interface{}, error) {
	if src == nil {
		return nil, nil
	}

	// Make the interface a reflect.Value
	original := reflect.ValueOf(src)

	// Make a copy of the same type as the original.
	cpy := reflect.New(original.Type()).Elem()
	// cpyo := cpy.Interface()

	opt, receiver := dcoption(option...)

	// Recursively copy the original.
	e := CopyRecursiveE(original, cpy, opt, receiver)
	if e != nil {
		return nil, e
	}
	// Return the copy as an interface.
	return cpy.Interface(), nil
}

func DeepCopy2(src interface{}, dest interface{}, option ...interface{}) error {
	if src == nil || dest == nil {
		return errors.New("src和dest都不能为空")
	}
	if reflect.TypeOf(src) != reflect.TypeOf(dest) {
		return errors.New("src和dest数据类型必须一致")
	}
	// Make the interface a reflect.Value
	org := reflect.ValueOf(src)
	cpy := reflect.ValueOf(dest)
	if reflect.TypeOf(src).Kind() == reflect.Ptr {
		org = org.Elem()
		cpy = cpy.Elem()
		if reflect.TypeOf(org) != reflect.TypeOf(cpy) {
			return errors.New("src和dest数据类型必须一致")
		}
	} else {
		return errors.New("src和dest数据类型必须是指针")
	}
	opt, receiver := dcoption(option...)

	// Recursively copy the original.
	e := CopyRecursiveE(org, cpy, opt, receiver)
	if e != nil {
		return e
	}
	return nil
}

func CopyRecursiveE(original, cpy reflect.Value, opt DeepCopyOption, receiver map[string]map[string]string) error {
	CopyRecursive(original, cpy, opt, receiver)
	if len(receiver) > 0 {
		bs, _ := json.MarshalIndent(receiver, "", "  ")
		return fmt.Errorf("%s%s", "DeepCopy encounter with private field, change it to public or implement DeepCopy interface\n", string(bs))
	}
	return nil
}

// CopyRecursive does the actual copying of the interface. It currently has
// limited support for what it can handle. Add as needed.
func CopyRecursive(original, cpy reflect.Value, opt DeepCopyOption, receiver map[string]map[string]string) {
	// check for implement deepcopy.Interface
	if original.CanInterface() {
		tm, b := original.Type().MethodByName("DeepCopy")
		if b {
			// fmt.Println(m)
			// 只有返回值类型与接口类型的DeepCopy方法才能得到正确结果，否则无法Set
			if tm.PkgPath == "" && tm.Type.NumOut() == 1 &&
				(original.Type() == tm.Type.Out(0) ||
					original.Type().AssignableTo(tm.Type.Out(0))) {
				om := original.MethodByName(tm.Name)
				if !om.IsNil() {
					args := []reflect.Value{}
					for i := 0; i < tm.Type.NumIn(); i++ {
						intn := tm.Type.In(i).Name()
						if intn == "DeepCopyOption" {
							args = append(args, reflect.ValueOf(opt))
						} else if intn == "map[string]map[string]string" {
							args = append(args, reflect.ValueOf(receiver))
						} else if intn == "" && tm.Type.IsVariadic() {
							args = append(args, reflect.ValueOf(opt))
							args = append(args, reflect.ValueOf(receiver))
						} else if intn != "" {
							args = append(args, reflect.New(tm.Type.In(i)))
						}
					}
					out := om.Call(args)
					// fmt.Println(out)
					cpy.Set(out[0])
					return
				}
			}
		}
	}

	// handle according to original's Kind
	switch original.Kind() {
	case reflect.Ptr:
		// Get the actual value being pointed to.
		originalValue := original.Elem()

		// if  it isn't valid, return.
		if !originalValue.IsValid() {
			return
		}
		cpy.Set(reflect.New(originalValue.Type()))
		CopyRecursive(originalValue, cpy.Elem(), opt, receiver)

	case reflect.Interface:
		// If this is a nil, don't do anything
		if original.IsNil() {
			return
		}
		// Get the value for the interface, not the pointer.
		originalValue := original.Elem()

		// Get the value by calling Elem().
		copyValue := reflect.New(originalValue.Type()).Elem()
		CopyRecursive(originalValue, copyValue, opt, receiver)
		cpy.Set(copyValue)

	case reflect.Struct:
		t, ok := original.Interface().(time.Time)
		if ok {
			cpy.Set(reflect.ValueOf(t))
			return
		}
		// Go through each field of the struct and copy it.
		fieldidx := []int{}
		hasPrivateField := false
		for i := 0; i < original.NumField(); i++ {
			// The Type's StructField for a given field is checked to see if StructField.PkgPath
			// is set to determine if the field is exported or not because CanSet() returns false
			// for settable fields.  I'm not sure why.  -mohae
			field := original.Type().Field(i)
			pkg := field.PkgPath
			if pkg != "" {
				key := pkg + "/" + original.Type().Name()
				if receiver != nil {
					if receiver[key] == nil {
						receiver[key] = map[string]string{
							field.Name: field.Type.Name(),
						}
					} else {
						receiver[key][field.Name] = field.Type.Name()
					}
				}
				hasPrivateField = true
			} else {
				fieldidx = append(fieldidx, i)
			}
		}
		if hasPrivateField {
			switch opt {
			case DC_OVPA_COPY_ADDRESS:
				cpy.Set(original)
				return
			case DC_OVPA_SKIP_PRIVATE_ATTR:
			case DC_OVPA_PANIC:
			}
		}
		for _, i := range fieldidx {
			CopyRecursive(original.Field(i), cpy.Field(i), opt, receiver)
		}

	case reflect.Slice:
		if original.IsNil() {
			return
		}
		// Make a new slice and copy each element.
		cpy.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
		for i := 0; i < original.Len(); i++ {
			CopyRecursive(original.Index(i), cpy.Index(i), opt, receiver)
		}

	case reflect.Map:
		if original.IsNil() {
			return
		}
		cpy.Set(reflect.MakeMap(original.Type()))
		for _, key := range original.MapKeys() {
			originalValue := original.MapIndex(key)
			copyValue := reflect.New(originalValue.Type()).Elem()
			CopyRecursive(originalValue, copyValue, opt, receiver)
			copyKey := DeepCopy(key.Interface())
			cpy.SetMapIndex(reflect.ValueOf(copyKey), copyValue)
		}

	default:
		cpy.Set(original)
	}
}

func CastValue(vt reflect.Type, vi interface{}) (vv reflect.Value, ee error) {
	switch vt.Kind() {
	case reflect.Bool:
		v, e := cast.ToStringE(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Int:
		v, e := cast.ToIntE(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Int8:
		v, e := cast.ToInt8E(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Int16:
		v, e := cast.ToInt16E(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Int32:
		v, e := cast.ToInt32E(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Int64:
		v, e := cast.ToInt64E(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Uint:
		v, e := cast.ToUintE(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Uint8:
		v, e := cast.ToUint8E(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Uint16:
		v, e := cast.ToUint16E(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Uint32:
		v, e := cast.ToUint32E(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Uint64:
		v, e := cast.ToUint64E(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Float32:
		v, e := cast.ToFloat32E(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Float64:
		v, e := cast.ToFloat64E(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.Slice:
		v, e := cast.ToSliceE(vi)
		ee = e
		vv = reflect.ValueOf(v)
	case reflect.String:
		v, e := cast.ToStringE(vi)
		ee = e
		vv = reflect.ValueOf(v)
	default:
		pfinfo := map[string]map[string]string{}
		vv = reflect.New(vt).Elem()
		CopyRecursive(reflect.ValueOf(vi), vv, DC_OVPA_DEFAULT, pfinfo)
		if len(pfinfo) > 0 {
			bs, _ := json.MarshalIndent(pfinfo, "", "  ")
			s := fmt.Sprint("DeepCopy encounter with private field, change it to public or implement DeepCopy interface\n", string(bs))
			ee = errors.New("不支持的数据类型，" + s)
		}
	}
	return
}
