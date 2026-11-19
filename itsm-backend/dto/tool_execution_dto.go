package dto

import "errors"

// ErrToolPermissionDenied 表示 AI 工具执行被运行期策略或身份守卫拒绝。
//
// 定义在 dto 层是为了让 handlers/skill 能在不依赖 handlers/ai 的前提下用
// errors.Is 识别拒绝语义。此前 registry 用 %v 包装执行错误，链路断开，
// 权限拒绝被统一兜底成 500/5001，操作者无法区分「无授权」与「服务异常」。
var ErrToolPermissionDenied = errors.New("tool permission denied")

// ToolExecutionResponse distinguishes an accepted approval request from execution.
type ToolExecutionResponse struct {
	Status        string      `json:"status"`
	Summary       string      `json:"summary"`
	InvocationID  int         `json:"invocationId"`
	ApprovalState string      `json:"approvalState"`
	NextActions   []string    `json:"nextActions"`
	Artifacts     []string    `json:"artifacts"`
	Data          interface{} `json:"data"`
}
