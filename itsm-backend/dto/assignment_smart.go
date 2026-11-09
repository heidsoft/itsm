package dto

import "errors"

// 智能分配规则域的稳定错误语义。放在 dto 包是为了让 handlers/assignment_smart
// 的窄契约与错误映射能脱离共享层；service 侧以变量别名指向同一 error 实例，
// 因此 errors.Is 的包装链判定不受影响。
var (
	ErrAssignmentRuleNotFound = errors.New("assignment rule not found")
	ErrTestTicketNotFound     = errors.New("assignment test ticket not found")
)
