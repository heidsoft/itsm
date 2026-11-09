package dto

import "errors"

// 事件领域哨兵错误。handler 侧通过 errors.Is 做精确错误分类，
// 因此定义必须放在双方都能引用的公共值类型包，service 侧以变量别名保留原公开名。
// 别名与此处指向同一个 error 实例，errors.Is 可命中包装链。
var (
	ErrIncidentNotFound                  = errors.New("incident not found")
	ErrIncidentTerminal                  = errors.New("terminal incident cannot be escalated")
	ErrIncidentEscalationLevelInvalid    = errors.New("escalation level must be between 1 and 5")
	ErrIncidentEscalationLevelNotGreater = errors.New("escalation level must be greater than current level")
	ErrIncidentEscalationReasonRequired  = errors.New("escalation reason is required")
	ErrEmailContractNotActive            = errors.New("support contract is not active")
	ErrIncidentInvalidTransition         = errors.New("invalid incident status transition")
	ErrIncidentResolutionRequired        = errors.New("incident resolution is required")
	ErrIncidentCloseNotesRequired        = errors.New("incident close notes are required")
	ErrIncidentVersionConflict           = errors.New("incident version conflict")
)
