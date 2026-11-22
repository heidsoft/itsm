package skill

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"itsm-backend/dto"
)

// validateSkillInput 按 manifest 声明的 InputSchema 校验调用方入参。
//
// 字段即声明：InputSchema 此前只在 manifests 里声明、持久化并在发现面暴露，
// 执行入口从不消费它——技能作者写下的 required/type 约束等于装饰。本函数在
// 执行前消费该声明，让「声明了必须有 query 字段」的技能在缺字段时真的拒绝。
//
// 边界（刻意保持最小，避免引入完整 JSON Schema 依赖）：
//   - 支持 type / required / properties / enum 四个关键字，properties 可递归；
//   - schema 为 nil 或空 object 时放行（向后兼容：多数内置技能未声明 schema）；
//   - schema 非空但无法解析为 object 时 **fail-closed**——声明存在却读不懂，
//     静默放行会让校验形同虚设，与策略信封「读不出即拒绝」口径一致；
//   - 校验对象必须是调用方原始入参：身份字段（tenantId/userId/role）由 handler
//     在校验之后注入，不参与校验，避免额外字段触发意外的 additional 约束。
//
// 返回的错误统一包装 dto.ErrSkillValidation，由 failInvoke 映射为 400。
func validateSkillInput(schema interface{}, input interface{}) error {
	root, declared := normalizeSchema(schema)
	if !declared {
		// schema 为 nil / 空 —— 未声明约束，放行。
		return nil
	}
	if root == nil {
		return fmt.Errorf("%w: inputSchema 声明存在但无法解析为 JSON object", dto.ErrSkillValidation)
	}
	return validateSchemaNode(root, input, "$")
}

// normalizeSchema 把 interface{} 形态的 schema 归一为 map。
// 返回 (nil, false) 表示未声明；(nil, true) 表示声明了但解析失败（调用方须 fail-closed）。
func normalizeSchema(schema interface{}) (map[string]interface{}, bool) {
	if schema == nil {
		return nil, false
	}
	switch v := schema.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			return nil, false
		}
		return v, true
	default:
		// manifest 经 JSON 往返后可能是 json.RawMessage 或其他标量形态；
		// 尝试 marshal→unmarshal 得到规范 map。
		raw, err := json.Marshal(schema)
		if err != nil {
			return nil, true
		}
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, true
		}
		if len(m) == 0 {
			return nil, false
		}
		return m, true
	}
}

func validateSchemaNode(node interface{}, value interface{}, path string) error {
	schema, ok := toStringMap(node)
	if !ok {
		// 子节点格式异常：fail-closed，不静默跳过。
		return fmt.Errorf("%w: %s 的 schema 节点格式非法", dto.ErrSkillValidation, path)
	}

	if err := validateType(schema["type"], value, path); err != nil {
		return err
	}
	if err := validateEnum(schema["enum"], value, path); err != nil {
		return err
	}
	if err := validateRequiredAndProperties(schema, value, path); err != nil {
		return err
	}
	return nil
}

// validateType 支持 type 为字符串或字符串数组（JSON Schema 允许 union，如 ["string","null"]）。
// type 缺省时不校验类型。
func validateType(typeSpec interface{}, value interface{}, path string) error {
	if typeSpec == nil {
		return nil
	}
	types := normalizeStringSlice(typeSpec)
	if len(types) == 0 {
		return fmt.Errorf("%w: %s 的 type 声明格式非法", dto.ErrSkillValidation, path)
	}
	for _, t := range types {
		if valueMatchesType(t, value) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s 类型不符，期望 %s，实际 %s",
		dto.ErrSkillValidation, path, strings.Join(types, "|"), valueTypeName(value))
}

func valueMatchesType(t string, value interface{}) bool {
	if value == nil {
		return t == "null"
	}
	switch t {
	case "object":
		_, ok := toStringMap(value)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number":
		return isNumber(value)
	case "integer":
		return isInteger(value)
	case "null":
		return false
	default:
		// 未知 type 不可静默当作通过——声明方写错时应当响亮失败。
		return false
	}
}

func validateEnum(enumSpec interface{}, value interface{}, path string) error {
	if enumSpec == nil {
		return nil
	}
	items, ok := enumSpec.([]interface{})
	if !ok {
		return fmt.Errorf("%w: %s 的 enum 声明格式非法", dto.ErrSkillValidation, path)
	}
	for _, candidate := range items {
		if jsonEqual(candidate, value) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s 取值不在声明的 enum 内", dto.ErrSkillValidation, path)
}

func validateRequiredAndProperties(schema map[string]interface{}, value interface{}, path string) error {
	obj, isObject := toStringMap(value)
	if !isObject {
		// required/properties 只对 object 生效；非 object 已由 type 校验兜底。
		return nil
	}
	if required, ok := schema["required"]; ok {
		fields := normalizeStringSlice(required)
		if len(fields) == 0 {
			return fmt.Errorf("%w: %s 的 required 声明格式非法", dto.ErrSkillValidation, path)
		}
		for _, field := range fields {
			v, present := obj[field]
			if !present || v == nil {
				return fmt.Errorf("%w: %s 缺少必填字段 %q", dto.ErrSkillValidation, path, field)
			}
		}
	}
	if propsRaw, ok := schema["properties"]; ok {
		props, ok := toStringMap(propsRaw)
		if !ok {
			return fmt.Errorf("%w: %s 的 properties 声明格式非法", dto.ErrSkillValidation, path)
		}
		for field, childSchema := range props {
			childValue, present := obj[field]
			if !present {
				continue // 非必填字段缺席不校验
			}
			if err := validateSchemaNode(childSchema, childValue, path+"."+field); err != nil {
				return err
			}
		}
	}
	return nil
}

// ---- helpers ----

func toStringMap(v interface{}) (map[string]interface{}, bool) {
	switch m := v.(type) {
	case map[string]interface{}:
		return m, true
	default:
		return nil, false
	}
}

func normalizeStringSlice(v interface{}) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []string:
		return x
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, item := range x {
			s, ok := item.(string)
			if !ok {
				return nil
			}
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

func isNumber(v interface{}) bool {
	switch v.(type) {
	case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func isInteger(v interface{}) bool {
	if isNumber(v) {
		return true
	}
	// JSON 反序列化后整数也是 float64，判断是否为整数值。
	if f, ok := v.(float64); ok {
		return f == float64(int64(f))
	}
	return false
}

func valueTypeName(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		if isNumber(v) {
			return "number"
		}
		return reflect.TypeOf(v).String()
	}
}

// jsonEqual 通过 JSON 往返归一化数字类型后比较（enum 的 1 与 int(1) 应相等）。
func jsonEqual(a, b interface{}) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	ra, errA := json.Marshal(a)
	rb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(ra) == string(rb)
}
