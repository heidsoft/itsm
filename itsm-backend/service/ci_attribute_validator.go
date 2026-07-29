package service

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
	"itsm-backend/ent/citype"
	"itsm-backend/ent/configurationitem"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
)

// globalAttributeDefinitionTenantID 全局属性定义所属的租户ID。
// 系统预置的 CI 属性定义统一挂在该租户下，供所有租户共享；
// 查询时显式带上该租户作为兜底，而非隐式回退。
const globalAttributeDefinitionTenantID = 1

// CIAttributeValidator 基于 CIAttributeDefinition 元数据对 CI 动态属性做归一化与校验。
// 校验内容：类型强转、默认值填充、必填、validation_rules（枚举/长度/数值范围/pattern）、唯一性。
type CIAttributeValidator struct {
	client *ent.Client
}

// NewCIAttributeValidator 创建 CI 属性校验器
func NewCIAttributeValidator(client *ent.Client) *CIAttributeValidator {
	return &CIAttributeValidator{client: client}
}

// NormalizeAttributes 按 CI 类型（含父类型继承链）的属性定义归一化并校验 attributes。
// existingID 为更新场景下当前 CI 的 ID（用于唯一性校验时排除自身），创建场景传 0。
// 返回归一化后的属性集合；校验失败返回面向用户的中文错误。
func (v *CIAttributeValidator) NormalizeAttributes(ctx context.Context, tenantID, ciTypeID int, attributes map[string]interface{}, existingID int) (map[string]interface{}, error) {
	typeIDs, err := v.resolveCITypeChain(ctx, ciTypeID, tenantID)
	if err != nil {
		return nil, err
	}
	definitionMap := make(map[string]*ent.CIAttributeDefinition)
	// 从继承链根部向叶子遍历，子类型定义覆盖父类型同名定义
	for i := len(typeIDs) - 1; i >= 0; i-- {
		defs, queryErr := v.client.CIAttributeDefinition.Query().
			Where(
				ciattributedefinition.CiTypeID(typeIDs[i]),
				ciattributedefinition.IsActive(true),
				ciattributedefinition.Or(
					ciattributedefinition.TenantID(tenantID),
					ciattributedefinition.TenantID(globalAttributeDefinitionTenantID),
				),
			).
			// 全局定义排前，租户自定义同名定义覆盖全局定义
			Order(ent.Asc(ciattributedefinition.FieldTenantID)).
			All(ctx)
		if queryErr != nil {
			return nil, fmt.Errorf("查询CI属性定义失败: %w", queryErr)
		}
		for _, def := range defs {
			definitionMap[def.Name] = def
		}
	}
	if len(definitionMap) == 0 {
		if attributes == nil {
			return map[string]interface{}{}, nil
		}
		return attributes, nil
	}

	normalized := make(map[string]interface{}, len(attributes))
	for k, val := range attributes {
		normalized[k] = val
	}

	for name, def := range definitionMap {
		value, exists := normalized[name]
		if (!exists || value == nil || isBlankAttributeString(value)) && def.DefaultValue != "" {
			parsedDefault, err := parseAttributeValue(def.Type, def.DefaultValue)
			if err != nil {
				return nil, fmt.Errorf("属性 %s 默认值无效: %w", def.DisplayName, err)
			}
			normalized[name] = parsedDefault
			value = parsedDefault
			exists = true
		}

		if def.Required && (!exists || value == nil || isBlankAttributeString(value)) {
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
			duplicate, err := v.hasDuplicateAttributeValue(ctx, tenantID, ciTypeID, name, normalizedValue, existingID)
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

// resolveCITypeChain 沿 parent_type_id 解析 CI 类型继承链（含自身），带环检测
func (v *CIAttributeValidator) resolveCITypeChain(ctx context.Context, ciTypeID, tenantID int) ([]int, error) {
	visited := make(map[int]struct{})
	ids := make([]int, 0, 4)
	currentID := ciTypeID
	for currentID != 0 {
		if _, exists := visited[currentID]; exists {
			return nil, fmt.Errorf("CI类型继承关系存在循环")
		}
		visited[currentID] = struct{}{}
		current, err := v.client.CIType.Query().
			Where(citype.ID(currentID), citype.TenantID(tenantID)).
			First(ctx)
		if err != nil {
			return nil, fmt.Errorf("查询CI类型继承关系失败: %w", err)
		}
		ids = append(ids, current.ID)
		if current.ParentTypeID == nil {
			break
		}
		currentID = *current.ParentTypeID
	}
	return ids, nil
}

// hasDuplicateAttributeValue 检查同租户同类型下是否已有 CI 使用相同属性值。
// 使用 JSON 谓词下推到数据库（PostgreSQL: attributes->>name = ?），避免全表加载内存比对。
// 传入的 value 已经过 normalizeAttributeValue 类型归一化，可直接按类型比对。
func (v *CIAttributeValidator) hasDuplicateAttributeValue(ctx context.Context, tenantID, ciTypeID int, attributeName string, value interface{}, existingID int) (bool, error) {
	query := v.client.ConfigurationItem.Query().
		Where(
			configurationitem.TenantID(tenantID),
			configurationitem.CiTypeID(ciTypeID),
			func(s *sql.Selector) {
				s.Where(sqljson.ValueEQ(configurationitem.FieldAttributes, value, sqljson.Path(attributeName)))
			},
		)
	if existingID > 0 {
		query = query.Where(configurationitem.IDNEQ(existingID))
	}
	return query.Exist(ctx)
}

// normalizeAttributeValue 对单个属性值做类型强转并应用 validation_rules
func normalizeAttributeValue(def *ent.CIAttributeDefinition, value interface{}) (interface{}, error) {
	normalized, err := coerceAttributeValue(def.Type, value)
	if err != nil {
		return nil, err
	}

	rules, err := parseValidationRules(def.ValidationRules)
	if err != nil {
		return nil, fmt.Errorf("验证规则格式错误: %w", err)
	}

	if err := applyAttributeValidationRules(def.Type, normalized, rules); err != nil {
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
		return attrToInt(value)
	case "float", "number":
		return attrToFloat(value)
	case "boolean", "bool":
		return attrToBool(value)
	case "date":
		return attrToTimeString(value, "2006-01-02")
	case "datetime":
		return attrToTimeString(value, time.RFC3339)
	case "json":
		return value, nil
	case "enum":
		return fmt.Sprint(value), nil
	default:
		return value, nil
	}
}

func applyAttributeValidationRules(attributeType string, value interface{}, rules map[string]interface{}) error {
	if len(rules) == 0 {
		return nil
	}

	if allowed := stringListFromRule(rules, "enum_values"); len(allowed) > 0 {
		actual := fmt.Sprint(value)
		matched := false
		for _, item := range allowed {
			if actual == item {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("值 %q 不在允许范围内", actual)
		}
	}

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
		actual, err := attrToFloat(value)
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
	value, err := attrToFloat(raw)
	if err != nil {
		return 0, false
	}
	return value, true
}

func attrToInt(value interface{}) (int, error) {
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

func attrToFloat(value interface{}) (float64, error) {
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

func attrToBool(value interface{}) (bool, error) {
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

func attrToTimeString(value interface{}, layout string) (string, error) {
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

func isBlankAttributeString(value interface{}) bool {
	text, ok := value.(string)
	return ok && strings.TrimSpace(text) == ""
}
