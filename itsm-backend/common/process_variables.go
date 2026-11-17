package common

import (
	"fmt"
	"sort"
	"strings"
)

// 引擎内部控制变量以 `_` 前缀写入 ProcessInstance.Variables（完成位图、幂等键、
// 补偿顺序、service task 重试计数等）。读取侧若不过滤，流程详情/列表/监控会把
// 调度器内部状态原样吐给前端和 CLI；写入侧由 ValidateClientVariables 统一拒绝。
//
// 业务变量（含连接器写入的 snake_case 键）保持原样返回，避免改变现有流程语义。
const engineVariablePrefix = "_"

// ValidateClientVariables 拒绝调用方写入引擎控制命名空间。启动流程与变量 PATCH
// 共用这一条规则：控制键一旦能被外部写进去，完成位图和幂等键就可被调用方操纵。
// 返回 400 的 AppError 而非裸 error，否则 handler 会把它兜底成 500「操作失败」。
// 多个非法键时按字典序报第一个，保证同样的请求得到同样的文案。
func ValidateClientVariables(variables map[string]interface{}) error {
	var rejected []string
	for key := range variables {
		if strings.HasPrefix(key, engineVariablePrefix) {
			rejected = append(rejected, key)
		}
	}
	if len(rejected) == 0 {
		return nil
	}
	sort.Strings(rejected)
	return NewValidationError(fmt.Sprintf("流程内部变量 %s 不允许通过 API 覆盖", rejected[0]), nil)
}

// FilterEngineVariables 返回去掉引擎内部控制键后的变量副本；输入为 nil 时返回 nil。
func FilterEngineVariables(variables map[string]interface{}) map[string]interface{} {
	if variables == nil {
		return nil
	}
	filtered := make(map[string]interface{}, len(variables))
	for key, value := range variables {
		if len(key) > 0 && key[0] == engineVariablePrefix[0] {
			continue
		}
		filtered[key] = value
	}
	return filtered
}
