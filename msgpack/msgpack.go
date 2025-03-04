package msgpack

import (
	"bytes"

	"github.com/vmihailenco/msgpack/v5"
)

// msgpack BUG：解码过程，结构中多个空值时，第二个匿名指针空值会自动被初始化，所以不能使用多个匿名指针
// msgpack 字符串/字节数组转换过程使用 unsafepointer 同地址快速转换方法，可能会因内存被回收，编码失败
// invalid code=cf decoding string/bytes length 结构嵌套或map嵌套，decode时可能会报错，不稳定
//
// 不能动态创建结构，指向结构的 interface{} 解码后会变成 map
// 需要相应的结构实现扩展接口 EncodeMsgpack(enc *msgpack.Encoder) error / DecodeMsgpack(dec *msgpack.Decoder) error
// 或 MarshalMsgpack() ([]byte, error) / UnmarshalMsgpack(b []byte) error
// 通过 msgpack.RegisterExt 注册可以提高效率
//
// encode map[string]interface{}，如果数据中值为 int 类型，decode 后类型会根据数值的大小改变，如 int(0) 变为 int8(0)
// 注意：
// 必须明确类型，接口类型不能正确编码
// 隐藏属性会被忽略
// 最多只能有一个匿名属性，更多的匿名属性会被忽略
func Encode(v interface{}) ([]byte, error) {
	enc := msgpack.GetEncoder()

	var buf bytes.Buffer
	enc.Reset(&buf)

	enc.UseCompactFloats(false)
	enc.UseCompactInts(false)
	enc.SetSortMapKeys(true)
	err := enc.Encode(v)
	b := buf.Bytes()

	msgpack.PutEncoder(enc)

	if err != nil {
		return nil, err
	}
	return b, err
}

func Decode(data []byte, v interface{}) error {
	dec := msgpack.GetDecoder()

	dec.Reset(bytes.NewReader(data))
	err := dec.Decode(v)

	msgpack.PutDecoder(dec)

	return err
}

func EncodeString(v interface{}) (string, error) {
	bs, e := Encode(v)
	return string(bs), e
}

func DecodeString(s string, v interface{}) error {
	return Decode([]byte(s), v)
}

func MustEncode(v interface{}) []byte {
	bs, e := Encode(v)
	if e != nil {
		panic(e)
	}
	return bs
}

func MustDecode(bs []byte, v interface{}) {
	e := Decode(bs, v)
	if e != nil {
		panic(e)
	}
}

func MustEncodeString(v interface{}) string {
	return string(MustEncode(v))
}

func MustDecodeString(s string, v interface{}) {
	MustDecode([]byte(s), v)
}
