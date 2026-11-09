package dto

import "errors"

// 交付任务的执行权冲突与终止态哨兵（2026-10-21）。
// 这组错误文案由 handler 映射为 409/4004 等稳定响应，禁止随意改写；
// 交付会对真实基础设施产生副作用，调用方必须能区分「已完成」「他人正在做」
// 「重试次数用尽」「当前状态不可重试」，而不是统一压成 500。
var (
	// ErrProvisioningTaskSucceeded 任务已交付成功，拒绝再次执行（幂等空操作）。
	ErrProvisioningTaskSucceeded = errors.New("交付任务已完成，拒绝重复执行")
	// ErrProvisioningTaskInProgress 任务正在执行中（或刚被其他 worker 领取）。
	ErrProvisioningTaskInProgress = errors.New("交付任务正在执行中")
	// ErrProvisioningTaskAttemptsExhausted 重试次数已达上限，必须转人工判定。
	ErrProvisioningTaskAttemptsExhausted = errors.New("交付任务重试次数已达上限，需人工接管")
	// ErrProvisioningTaskNotRetryable 当前状态不具备重试语义（如尚未失败的 pending）。
	ErrProvisioningTaskNotRetryable = errors.New("交付任务当前状态不允许重试")
	// ErrProvisioningTaskNotResolvable 当前状态不允许人工判定（running/succeeded）。
	ErrProvisioningTaskNotResolvable = errors.New("交付任务当前状态不允许人工判定")
	// ErrProvisioningNotExecutable 交付参数或云账号配置不完整，自动重试不可能成功。
	ErrProvisioningNotExecutable = errors.New("交付配置不完整，自动重试不会成功")
	// ErrProvisioningPermanentlyFailed 云侧业务拒绝（余额不足/配额封禁/参数不存在）。
	ErrProvisioningPermanentlyFailed = errors.New("云侧拒绝且自动重试不会成功")
)
