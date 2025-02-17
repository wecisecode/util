package sortedmap_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wecisecode/util/sortedmap"
)

type deepCopyable struct {
	private_value string
}

func (a *deepCopyable) DeepCopy() *deepCopyable {
	return &deepCopyable{a.private_value}
}

func (a *deepCopyable) String() string {
	return a.private_value
}

func (a *deepCopyable) UnmarshalJSON(bs []byte) (err error) {
	a.private_value = string(bs)
	return
}

func (a *deepCopyable) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.private_value)
}

func TestDeepMerge(t *testing.T) {
	m := sortedmap.NewLinkedMap()
	m.Put("a", &deepCopyable{"aaaa"})
	m.Put("b", []*deepCopyable{{"bbbb"}})
	m.Put("c", []interface{}{&deepCopyable{"cccc"}})
	c := m.DeepCopy()
	c.GetValue("a").(*deepCopyable).private_value = "xxxx"
	{
		b := c.GetValue("b")
		if bs, ok := b.([]*deepCopyable); ok {
			bs[0].private_value = "yyyy"
		} else if bs, ok := b.([]interface{}); ok {
			bs[0].(*deepCopyable).private_value = "yyyy"
		} else {
			assert.NoError(t, fmt.Errorf("%s", "b.(type) neither []*deepCopyable nor []interface{}"))
		}
	}
	c.GetValue("c").([]interface{})[0].(*deepCopyable).private_value = "zzzz"
	fmt.Println(m)
	fmt.Println(c)

	sortedmap.DeepMerge(m, c, true)
	fmt.Println(m)
	{
		b := m.GetValue("b")
		if bs, ok := b.([]*deepCopyable); ok {
			assert.Equal(t, "bbbb", bs[0].private_value)
			assert.Equal(t, "yyyy", bs[1].private_value)
		} else if bs, ok := b.([]interface{}); ok {
			assert.Equal(t, "bbbb", bs[0].(*deepCopyable).private_value)
			assert.Equal(t, "yyyy", bs[1].(*deepCopyable).private_value)
		} else {
			assert.NoError(t, fmt.Errorf("%s", "b.(type) neither []*deepCopyable nor []interface{}"))
		}
	}
	assert.Equal(t, "cccc", m.GetValue("c").([]interface{})[0].(*deepCopyable).private_value)
	assert.Equal(t, "zzzz", m.GetValue("c").([]interface{})[1].(*deepCopyable).private_value)
}

func TestUnmarshalJSON(t *testing.T) {
	m := sortedmap.NewLinkedMap()
	e := m.UnmarshalJSON([]byte(fmt.Sprint(`{
		"node_id_str":"HUAWEI",
		"subscription_id_str":"s4",
		"sensor_path":"huawei-ifm:ifm/interfaces/interface",
		"collection_id":46,
		"collection_start_time":`, time.Now().UnixNano(), `,
		"msg_timestamp":`, time.Now().UnixNano(), `,
		"data_gpb":{
			"row":[{
				"timestamp":`, time.Now().UnixNano(), `,
				"content":{
					"interfaces":{
						"interface":[{
							"ifAdminStatus":1,
							"ifIndex":2,
							"ifName":"Eth-Trunk1"
						}]
					}
				}
			}]
		},
		"collection_end_time":`, time.Now().UnixNano(), `,
		"current_period":10000,
		"except_desc":"OK",
		"product_name":"CE6881",
		"encoding":`, 1, `,
		"software_version":"V200R020C10"
	}`)))
	assert.NoError(t, e)
}

func TestMarshalJSON(t *testing.T) {
	m := map[string]interface{}{"a": []int{1}}
	tsmcount := sortedmap.NewLinkedMap()
	tsmcount.Put("test", m)
	dm := sortedmap.NewLinkedMap()
	sortedmap.DeepMerge(dm, tsmcount, true)
	m["b"] = []int{2}
	bs, e := tsmcount.MarshalJSON()
	assert.NoError(t, e)
	fmt.Println(string(bs))
	assert.Equal(t, `{"test":{"a":[1],"b":[2]}}`, string(bs))
	bs, e = dm.MarshalJSON()
	assert.NoError(t, e)
	fmt.Println(string(bs))
	assert.Equal(t, `{"test":{"a":[1]}}`, string(bs))
	sortedmap.DeepMerge(dm, tsmcount, true)
	bs, e = dm.MarshalJSON()
	assert.NoError(t, e)
	fmt.Println(string(bs))
	assert.Equal(t, `{"test":{"a":[1,1],"b":[2]}}`, string(bs))
	//// merge 不同map KV类型，按直接覆盖不同类型的数据处理
	{
		dmx := dm.DeepCopy()
		mx := map[int]string{1: "x"}
		tsmx := sortedmap.NewLinkedMap()
		tsmx.Put("test", mx)
		sortedmap.DeepMerge(dmx, tsmx, false)
		bs, e = dmx.MarshalJSON()
		assert.NoError(t, e)
		fmt.Println(string(bs))
		assert.Equal(t, `{"test":{"1":"x"}}`, string(bs))
	}
	//// force merge 不同map KV类型，slice,map 强制转换为 []interface{},LinkedMap[interface{}]interface{}
	{
		dmx := dm.DeepCopy()
		mx := map[int]string{1: "x"}
		tsmx := sortedmap.NewLinkedMap()
		tsmx.Put("test", mx)
		sortedmap.DeepMerge(dmx, tsmx, true)
		bs, e = dmx.MarshalJSON()
		assert.NoError(t, e)
		fmt.Println(string(bs))
		assert.Equal(t, `{"test":{"a":[1,1],"b":[2],"1":"x"}}`, string(bs))
	}
	// json.Marshal 不支持 map[interface{}]interface{}
	bs, e = json.Marshal(map[interface{}]interface{}{"": 0})
	assert.Error(t, e)
	assert.Nil(t, bs)
	fmt.Println("json.Marshal 不支持 map[interface{}]interface{}")
}
