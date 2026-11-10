package dto

import "errors"

// SkillEntry 是 Skill 的可序列化视图（市场发现 / 管理端列表共用）。
// 与 SkillManifest 的区别：SkillEntry 折叠了 Code/Name/Metrics 等从 Skill
// 接口直接派生出的字段，方便前端直接渲染而不必再下钻 Manifest。
//
// TenantID 是技能归属租户：0 表示内置/全局技能，>0 表示该租户自定义技能。
// 管理端据此区分「平台提供」与「本租户注册」。
type SkillEntry struct {
	Code            string        `json:"code"`
	Name            string        `json:"name"`
	Version         string        `json:"version"`
	Title           string        `json:"title"`
	Provider        string        `json:"provider"`
	Description     string        `json:"description"`
	Category        string        `json:"category"` // ga / pilot / experimental
	Tags            []string      `json:"tags"`
	Capabilities    []string      `json:"capabilities"`
	RequiredPerms   []string      `json:"requiredPermissions"`
	Author          string        `json:"author,omitempty"`
	IsOfficial      bool          `json:"isOfficial"`
	Checksum        string        `json:"checksum,omitempty"`
	IsBuiltin       bool          `json:"isBuiltin"`
	TenantID        int           `json:"tenantId"`
	Status          string        `json:"status"` // active / disabled
	Manifest        SkillManifest `json:"manifest"`
	Metrics         SkillMetrics  `json:"metrics"`
	LongDescription string        `json:"longDescription,omitempty"`
}

// SkillInvokeResult 是 Invoke 的结构化返回，包含 latency / 错误码 / 是否被 metrics 追踪。
type SkillInvokeResult struct {
	Output         interface{}
	LatencyMs      int64
	SkippedMetrics bool
}

// 技能注册表的四个 sentinel。唯一实例在本包；service 侧保留同名别名，
// 既有 errors.Is 判定不发生改变。
var (
	ErrSkillAlreadyRegistered = errors.New("skill already registered")
	ErrSkillNotFound          = errors.New("skill not found")
	ErrSkillValidation        = errors.New("skill validation failed")
	ErrSkillInvoke            = errors.New("skill invocation failed")
)
