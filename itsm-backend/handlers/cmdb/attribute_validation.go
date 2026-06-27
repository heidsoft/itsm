package cmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/ciattributedefinition"
	"itsm-backend/ent/configurationitem"
)

func (r *EntRepository) normalizeCIAttributes(ctx context.Context, ci *ConfigurationItem, existingID int) (map[string]interface{}, error) {
	defs, err := r.client.CIAttributeDefinition.Query().
		Where(
			ciattributedefinition.CiTypeID(ci.CITypeID),
			ciattributedefinition.IsActive(true),
			ciattributedefinition.Or(
				ciattributedefinition.TenantID(ci.TenantID),
				ciattributedefinition.TenantID(1),
			),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询CI属性定义失败: %w", err)
	}

	if len(defs) == 0 {
		if ci.Attributes == nil {
			return map[string]interface{}{}, nil
		}
		return ci.Attributes, nil
	}

	definitionMap := make(map[string]*ent.CIAttributeDefinition, len(defs))
	for _, def := range defs {
		current, exists := definitionMap[def.Name]
		if !exists || (def.TenantID == ci.TenantID && current.TenantID != ci.TenantID) {
			definitionMap[def.Name] = def
		}
	}

	normalized := make(map[string]interface{}, len(ci.Attributes))
	for k, v := range ci.Attributes {
		normalized[k] = v
	}

	for name, def := range definitionMap {
		value, exists := normalized[name]
		if (!exists || value == nil || isBlankString(value)) && def.DefaultValue != "" {
			parsedDefault, err := parseAttributeValue(def.Type, def.DefaultValue)
			if err != nil {
				return nil, fmt.Errorf("属性 %s 默认值无效: %w", def.DisplayName, err)
			}
			normalized[name] = parsedDefault
			value = parsedDefault
			exists = true
		}

		if def.Required && (!exists || value == nil || isBlankString(value)) {
			return nil, fmt.Errorf("属性 %s 为必填项", def.DisplayName)
		}
		if !exists || value == nil {
			continue
		}

		normalizedValue, err := normalizeAttributeValue(def, value)
		if err != nil {
			return nil, fmt.Errorf("属性 %s 校验失败: %w", def.DisplayName, err)
		}
		normalized[name] = normalizedValue

		if def.Unique {
			duplicate, err := r.hasDuplicateAttributeValue(ctx, ci.TenantID, ci.CITypeID, name, normalizedValue, existingID)
			if err != nil {
				return nil, fmt.Errorf("校验属性 %s 唯一性失败: %w", def.DisplayName, err)
			}
			if duplicate {
				return nil, fmt.Errorf("属性 %s 的值必须唯一", def.DisplayName)
			}
		}
	}

	return normalized, nil
}

func (r *EntRepository) hasDuplicateAttributeValue(ctx context.Context, tenantID, ciTypeID int, attributeName string, value interface{}, existingID int) (bool, error) {
	items, err := r.client.ConfigurationItem.Query().
		Where(
			configurationitem.TenantID(tenantID),
			configurationitem.CiTypeID(ciTypeID),
		).
		All(ctx)
	if err != nil {
		return false, err
	}

	target := canonicalAttributeValue(value)
	for _, item := range items {
		if existingID > 0 && item.ID == existingID {
			continue
		}
		current, ok := item.Attributes[attributeName]
		if !ok {
			continue
		}
		if canonicalAttributeValue(current) == target {
			return true, nil
		}
	}

	return false, nil
}

func normalizeAttributeValue(def *ent.CIAttributeDefinition, value interface{}) (interface{}, error) {
	normalized, err := coerceAttributeValue(def.Type, value)
	if err != nil {
		return nil, err
	}

	rules, err := parseValidationRules(def.ValidationRules)
	if err != nil {
		return nil, fmt.Errorf("验证规则格式错误: %w", err)
	}

	if err := applyValidationRules(def.Type, normalized, rules); err != nil {
		return nil, err
	}

	return normalized, nil
}

func parseValidationRules(raw string) (map[string]interface{}, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]interface{}{}, nil
	}

	var rules map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func parseAttributeValue(attributeType string, raw string) (interface{}, error) {
	return coerceAttributeValue(attributeType, raw)
}

func coerceAttributeValue(attributeType string, value interface{}) (interface{}, error) {
	switch strings.ToLower(attributeType) {
	case "string", "reference":
		return fmt.Sprint(value), nil
	case "integer", "int":
		return toInt(value)
	case "float", "number":
		return toFloat(value)
	case "boolean", "bool":
		return toBool(value)
	case "date":
		return toTimeString(value, "2006-01-02")
	case "datetime":
		return toTimeString(value, time.RFC3339)
	case "json":
		return value, nil
	case "enum":
		return fmt.Sprint(value), nil
	default:
		return value, nil
	}
}

func applyValidationRules(attributeType string, value interface{}, rules map[string]interface{}) error {
	if len(rules) == 0 {
		return nil
	}

	if allowed := stringListFromRule(rules, "enum_values"); len(allowed) > 0 {
		actual := fmt.Sprint(value)
		for _, item := range allowed {
			if actual == item {
				goto patternCheck
			}
		}
		return fmt.Errorf("值 %q 不在允许范围内", actual)
	}

patternCheck:
	if pattern, ok := rules["pattern"].(string); ok && pattern != "" {
		actual := fmt.Sprint(value)
		if !strings.Contains(actual, pattern) && actual != pattern {
			return fmt.Errorf("值 %q 不符合约束 %q", actual, pattern)
		}
	}

	switch strings.ToLower(attributeType) {
	case "string", "reference", "enum":
		actual := fmt.Sprint(value)
		if minLen, ok := numericRuleValue(rules, "min_length"); ok && float64(len(actual)) < minLen {
			return fmt.Errorf("长度不能小于 %.0f", minLen)
		}
		if maxLen, ok := numericRuleValue(rules, "max_length"); ok && float64(len(actual)) > maxLen {
			return fmt.Errorf("长度不能大于 %.0f", maxLen)
		}
	case "integer", "int", "float", "number":
		actual, err := toFloat(value)
		if err != nil {
			return err
		}
		if minValue, ok := numericRuleValue(rules, "min"); ok && actual < minValue {
			return fmt.Errorf("不能小于 %v", minValue)
		}
		if maxValue, ok := numericRuleValue(rules, "max"); ok && actual > maxValue {
			return fmt.Errorf("不能大于 %v", maxValue)
		}
	}

	return nil
}

func stringListFromRule(rules map[string]interface{}, key string) []string {
	raw, ok := rules[key]
	if !ok {
		return nil
	}

	switch typed := raw.(type) {
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, fmt.Sprint(item))
		}
		return out
	case []string:
		return typed
	default:
		return nil
	}
}

func numericRuleValue(rules map[string]interface{}, key string) (float64, bool) {
	raw, ok := rules[key]
	if !ok {
		return 0, false
	}
	value, err := toFloat(raw)
	if err != nil {
		return 0, false
	}
	return value, true
}

func toInt(value interface{}) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float32:
		if math.Mod(float64(typed), 1) != 0 {
			return 0, fmt.Errorf("需要整数")
		}
		return int(typed), nil
	case float64:
		if math.Mod(typed, 1) != 0 {
			return 0, fmt.Errorf("需要整数")
		}
		return int(typed), nil
	case json.Number:
		i, err := typed.Int64()
		return int(i), err
	case string:
		v, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, fmt.Errorf("需要整数")
		}
		return v, nil
	default:
		return 0, fmt.Errorf("需要整数")
	}
}

func toFloat(value interface{}) (float64, error) {
	switch typed := value.(type) {
	case int:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case float32:
		return float64(typed), nil
	case float64:
		return typed, nil
	case json.Number:
		return typed.Float64()
	case string:
		v, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, fmt.Errorf("需要数值")
		}
		return v, nil
	default:
		return 0, fmt.Errorf("需要数值")
	}
}

func toBool(value interface{}) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "y":
			return true, nil
		case "false", "0", "no", "n":
			return false, nil
		default:
			return false, fmt.Errorf("需要布尔值")
		}
	default:
		return false, fmt.Errorf("需要布尔值")
	}
}

func toTimeString(value interface{}, layout string) (string, error) {
	switch typed := value.(type) {
	case time.Time:
		return typed.Format(layout), nil
	case string:
		parsed, err := time.Parse(layout, strings.TrimSpace(typed))
		if err != nil {
			return "", fmt.Errorf("时间格式无效")
		}
		return parsed.Format(layout), nil
	default:
		return "", fmt.Errorf("需要时间字符串")
	}
}

func canonicalAttributeValue(value interface{}) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}

func isBlankString(value interface{}) bool {
	text, ok := value.(string)
	return ok && strings.TrimSpace(text) == ""
}
