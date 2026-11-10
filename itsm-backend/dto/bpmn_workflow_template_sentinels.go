package dto

import "errors"

// BPMN 模板目录与存量盘点共用的纯值：下沉到 dto 后，handler 侧可以直接读 sentinel
// 与默认阈值，不必为几个值类型重新 import 共享 service 包。
// service 侧保留同名别名，errors.Is 与常量值身份都指向这里的唯一定义。

var (
	ErrWorkflowTemplateNotFound = errors.New("workflow template not found")
	ErrWorkflowTemplateConflict = errors.New("workflow template conflict")
	ErrWorkflowTemplateInvalid  = errors.New("workflow template invalid")
)

// DefaultStaleThresholdHours 存量待办盘点的默认陈旧阈值（小时）。
// 与 GA 就绪检查的 24h 口径一致；service 侧以常量别名暴露，避免两处漂移。
const DefaultStaleThresholdHours = 24
