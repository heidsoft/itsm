package dto

import "encoding/json"

// StructToMap 辅助方法：结构体转Map
func StructToMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	return m
}

// StructSliceToMapSlice 辅助方法：结构体切片转Map切片
func StructSliceToMapSlice(v interface{}) []map[string]interface{} {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m []map[string]interface{}
	_ = json.Unmarshal(data, &m)
	return m
}

// MapToStruct 辅助方法：Map转结构体
func MapToStruct(m map[string]interface{}, v interface{}) {
	if m == nil {
		return
	}
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, v)
}

// MapSliceToStructSlice 辅助方法：Map切片转结构体切片
func MapSliceToStructSlice(m []map[string]interface{}, v interface{}) {
	if m == nil {
		return
	}
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, v)
}
