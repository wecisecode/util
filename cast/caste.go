package cast

import (
	"time"

	"github.com/spf13/cast"
)

// ToBool casts an interface to a bool type.
func ToBoolE(i interface{}) (bool, error) {
	v, e := cast.ToBoolE(i)
	return v, e
}

// ToTime casts an interface to a time.Time type.
func ToTimeE(i interface{}) (time.Time, error) {
	v, e := cast.ToTimeE(i)
	return v, e
}

func ToTimeInDefaultLocationE(i interface{}, location *time.Location) (time.Time, error) {
	v, e := cast.ToTimeInDefaultLocationE(i, location)
	return v, e
}

// ToDuration casts an interface to a time.Duration type.
func ToDurationE(i interface{}) (time.Duration, error) {
	v, e := cast.ToDurationE(i)
	return v, e
}

func ToFloat64E(i interface{}) (float64, error) {
	switch r := i.(type) {
	case []byte:
		return cast.ToFloat64E(string(r))
	}
	v, e := cast.ToFloat64E(i)
	return v, e
}

// ToFloat32 casts an interface to a float32 type.
func ToFloat32E(i interface{}) (float32, error) {
	v, e := cast.ToFloat32E(i)
	return v, e
}

// ToInt64 casts an interface to an int64 type.
func ToInt64E(i interface{}) (int64, error) {
	v, e := cast.ToInt64E(i)
	return v, e
}

// ToInt32 casts an interface to an int32 type.
func ToInt32E(i interface{}) (int32, error) {
	v, e := cast.ToInt32E(i)
	return v, e
}

// ToInt16 casts an interface to an int16 type.
func ToInt16E(i interface{}) (int16, error) {
	v, e := cast.ToInt16E(i)
	return v, e
}

// ToInt8 casts an interface to an int8 type.
func ToInt8E(i interface{}) (int8, error) {
	v, e := cast.ToInt8E(i)
	return v, e
}

func ToIntE(i interface{}) (int, error) {
	switch v := i.(type) {
	case []byte:
		return cast.ToIntE(string(v))
	}
	v, e := cast.ToIntE(i)
	return v, e
}

// ToUint casts an interface to a uint type.
func ToUintE(i interface{}) (uint, error) {
	v, e := cast.ToUintE(i)
	return v, e
}

// ToUint64 casts an interface to a uint64 type.
func ToUint64E(i interface{}) (uint64, error) {
	v, e := cast.ToUint64E(i)
	return v, e
}

// ToUint32 casts an interface to a uint32 type.
func ToUint32E(i interface{}) (uint32, error) {
	v, e := cast.ToUint32E(i)
	return v, e
}

// ToUint16 casts an interface to a uint16 type.
func ToUint16E(i interface{}) (uint16, error) {
	v, e := cast.ToUint16E(i)
	return v, e
}

// ToUint8 casts an interface to a uint8 type.
func ToUint8E(i interface{}) (uint8, error) {
	v, e := cast.ToUint8E(i)
	return v, e
}

// ToString casts an interface to a string type.
func ToStringE(i interface{}) (string, error) {
	v, e := cast.ToStringE(i)
	return v, e
}

// ToStringMapString casts an interface to a map[string]string type.
func ToStringMapStringE(i interface{}) (map[string]string, error) {
	v, e := cast.ToStringMapStringE(i)
	return v, e
}

// ToStringMapStringSlice casts an interface to a map[string][]string type.
func ToStringMapStringSliceE(i interface{}) (map[string][]string, error) {
	v, e := cast.ToStringMapStringSliceE(i)
	return v, e
}

// ToStringMapBool casts an interface to a map[string]bool type.
func ToStringMapBoolE(i interface{}) (map[string]bool, error) {
	v, e := cast.ToStringMapBoolE(i)
	return v, e
}

// ToStringMapInt casts an interface to a map[string]int type.
func ToStringMapIntE(i interface{}) (map[string]int, error) {
	v, e := cast.ToStringMapIntE(i)
	return v, e
}

// ToStringMapInt64 casts an interface to a map[string]int64 type.
func ToStringMapInt64E(i interface{}) (map[string]int64, error) {
	v, e := cast.ToStringMapInt64E(i)
	return v, e
}

// ToStringMap casts an interface to a map[string]interface{} type.
func ToStringMapE(i interface{}) (map[string]interface{}, error) {
	v, e := cast.ToStringMapE(i)
	return v, e
}

// ToSlice casts an interface to a []interface{} type.
func ToSliceE(i interface{}) ([]interface{}, error) {
	v, e := cast.ToSliceE(i)
	return v, e
}

// ToBoolSlice casts an interface to a []bool type.
func ToBoolSliceE(i interface{}) ([]bool, error) {
	v, e := cast.ToBoolSliceE(i)
	return v, e
}

// ToStringSlice casts an interface to a []string type.
func ToStringSliceE(i interface{}) ([]string, error) {
	v, e := cast.ToStringSliceE(i)
	return v, e
}

// ToIntSlice casts an interface to a []int type.
func ToIntSliceE(i interface{}) ([]int, error) {
	v, e := cast.ToIntSliceE(i)
	return v, e
}

// ToDurationSlice casts an interface to a []time.Duration type.
func ToDurationSliceE(i interface{}) ([]time.Duration, error) {
	v, e := cast.ToDurationSliceE(i)
	return v, e
}
