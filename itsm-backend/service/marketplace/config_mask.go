package marketplace

// maskPlaceholder 是安装配置出站脱敏的占位值，与 handlers/marketplace
// 出站 DTO 的脱敏值保持一致。配置更新收到该值时还原为库中现值，
// 防止前端把脱敏后的配置原样回写导致真实密钥被掩码覆盖。
const maskPlaceholder = "***"

// RestoreMaskedConfig 把 incoming 中值为掩码的叶子还原为 existing 的现值，
// 未命中掩码的键原样保留，逐路径递归 merge。
func RestoreMaskedConfig(incoming, existing map[string]interface{}) map[string]interface{} {
	if incoming == nil {
		return nil
	}
	out := make(map[string]interface{}, len(incoming))
	for k, v := range incoming {
		if nested, ok := v.(map[string]interface{}); ok {
			if existingNested, ok := existing[k].(map[string]interface{}); ok {
				out[k] = RestoreMaskedConfig(nested, existingNested)
			} else {
				out[k] = nested
			}
			continue
		}
		if s, ok := v.(string); ok && s == maskPlaceholder {
			if existingValue, exists := existing[k]; exists {
				out[k] = existingValue
				continue
			}
		}
		out[k] = v
	}
	return out
}
