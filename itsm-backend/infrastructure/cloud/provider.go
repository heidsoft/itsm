package cloud

import (
	"context"
	"errors"
)

// ErrPermanent 表示云侧明确拒绝、且重试不会改变结果（余额不足、配额封禁、
// 交换機/安全组/镜像不存在、参数非法等业务错误）。
//
// 与网络抖动/限流/MQ 超时的区别在于「重试的期望收益为零」：自动重投只会
// 重复打出同样的请求、刷出同样的告警，并把真正的处置动作（充值、改参数、
// 人工裁决）埋在一堆重试记录里。调用方拿到它必须停止自动重试。
var ErrPermanent = errors.New("云侧拒绝且自动重试不会成功")

// ExecuteResult 交付执行结果。
type ExecuteResult struct {
	Resources map[string]string `json:"resources,omitempty"`
}

// Credential 一次交付调用使用的运行时凭据。
// 只存在于请求内存中，不落库、不写日志。
type Credential struct {
	AccessKeyID     string
	AccessKeySecret string
	SessionToken    string
}

// ExecuteRequest 携带一次交付所需的请求级上下文。
//
// Credential 由应用层按租户解析后传入，Provider 不自行读取配置或数据库；
// ClientToken 由任务唯一标识推导，重试同一任务时必须保持不变，否则云侧
// 幂等会退化成「每次重试都新建一台机器」。
type ExecuteRequest struct {
	TenantID         int
	TaskID           int
	ServiceRequestID int
	AccountID        string
	Region           string
	ClientToken      string
	Payload          map[string]any
	Credential       Credential
}

// Provider 交付 Provider 抽象。
type Provider interface {
	Name() string
	// RequiresCloudAccount 为 true 时，应用层必须先解析云账号与凭据，
	// 否则 Provider 无法执行。Stub 实现返回 false。
	RequiresCloudAccount() bool
	Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error)
}
