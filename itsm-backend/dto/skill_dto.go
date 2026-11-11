package dto

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// Skill 注册表的值类型、接口与 BaseSkill 骨架（原定义在共享层 service 包）。
// 整组闭包一起下沉：ComputeChecksum / ValidateForRegistration / BuildManifest
// 都是值类型方法，不能与类型分家；TrackedSkill / RecordResult 原本是私有钩子，
// 跨包后必须导出，否则 skill.SkillRegistry 无法再识别并累计指标。

// SkillManifest 技能自描述信息，用于市场展示和自动装配
type SkillManifest struct {
	Name                string      `json:"name"`                          // 唯一标识名称
	Version             string      `json:"version"`                       // 版本号
	Title               string      `json:"title"`                         // 显示名称
	Provider            string      `json:"provider"`                      // 提供商
	Description         string      `json:"description"`                   // 描述
	LongDescription     string      `json:"longDescription,omitempty"`     // 详细描述
	IconURL             string      `json:"iconUrl,omitempty"`             // 图标URL
	Screenshots         []string    `json:"screenshots,omitempty"`         // 截图列表
	Tags                []string    `json:"tags,omitempty"`                // 标签列表
	Category            string      `json:"category,omitempty"`            // 分类
	Capabilities        []string    `json:"capabilities,omitempty"`        // 支持的能力列表
	InputSchema         interface{} `json:"inputSchema,omitempty"`         // 输入参数JSON Schema
	OutputSchema        interface{} `json:"outputSchema,omitempty"`        // 输出结果JSON Schema
	RequiredPermissions []string    `json:"requiredPermissions,omitempty"` // 需要的系统权限
	MinSystemVersion    string      `json:"minSystemVersion,omitempty"`    // 最低支持的系统版本
	Author              string      `json:"author,omitempty"`              // 作者
	Rating              float64     `json:"rating,omitempty"`              // 评分
	InstallCount        int         `json:"installCount,omitempty"`        // 安装次数
	IsOfficial          bool        `json:"isOfficial,omitempty"`          // 是否是官方技能
	Checksum            string      `json:"checksum,omitempty"`            // manifest 完整性校验和（注册时自动计算，sha256:...）
}

// ComputeChecksum 基于身份与安全关键字段计算确定性校验和，
// 与 connector.Manifest.ComputeChecksum 保持同样的覆盖范围约定。
func (m SkillManifest) ComputeChecksum() string {
	caps := append([]string(nil), m.Capabilities...)
	sort.Strings(caps)
	perms := append([]string(nil), m.RequiredPermissions...)
	sort.Strings(perms)
	payload := strings.Join([]string{
		m.Name,
		m.Version,
		m.Provider,
		strings.Join(caps, ","),
		strings.Join(perms, ","),
		m.MinSystemVersion,
	}, "|")
	sum := sha256.Sum256([]byte(payload))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ValidateForRegistration 注册前的 manifest 完整性校验（fail closed）：
// 所有技能必须声明 name、version 和所需系统权限，否则拒绝注册。
func (m SkillManifest) ValidateForRegistration() error {
	if m.Name == "" {
		return errors.New("skill manifest: name must not be empty")
	}
	if m.Version == "" {
		return errors.New("skill manifest: version must not be empty for " + m.Name)
	}
	if len(m.RequiredPermissions) == 0 {
		return errors.New("skill manifest: requiredPermissions must be declared for " + m.Name)
	}
	return nil
}

// SkillMetrics 技能运行指标
type SkillMetrics struct {
	TotalCalls   int64   `json:"totalCalls"`   // 总调用次数
	SuccessRate  float64 `json:"successRate"`  // 成功率
	AvgLatencyMs int64   `json:"avgLatencyMs"` // 平均延迟(ms)
	ErrorCount   int64   `json:"errorCount"`   // 错误次数
	LastUsedAt   string  `json:"lastUsedAt"`   // 最后使用时间
}

// Skill 技能接口（运行时能力抽象）
//
// Skill Registry v1：恢复该接口定义，修复 skill_manifest.go 中
// ExtendedSkill 引用未定义 Skill 的预存编译错误。
// 所有 AI 能力（chat/triage/summarize/analyze/rag_search/agent_tool/...）
// 的运行时抽象都应该实现该接口，由 SkillRegistry 统一管理。
type Skill interface {
	// Code 技能唯一标识（如 ai.triage）
	Code() string
	// Name 显示名称
	Name() string
	// Execute 执行技能调用；input/executor-specific 协议由各 skill 自行定义
	Execute(ctx context.Context, input interface{}) (interface{}, error)
	// Validate 验证输入是否合法
	Validate(input interface{}) error
	// Tags 技能标签（用于发现、统计、过滤）
	Tags() []string
	// Manifest 返回技能的市场自描述信息
	Manifest() SkillManifest
	// GetMetrics 获取运行指标
	GetMetrics() SkillMetrics
}

// BaseSkill 提供 Skill 接口中无需每个 skill 单独实现的"通用字段 + 指标累计"骨架。
//
// 推荐用法：每个 builtin Skill 通过嵌入 *BaseSkill 复写 Code/Name/Tags/Manifest/Execute/Validate，
// 不必关注计数 / 平均时延 / LastUsed 等指标统计。
//
// 注意：BaseSkill 通过 trackedSkill 内部接口与 SkillRegistry 对接，仅当 Skill 通过
// Register() 注入到 SkillRegistry 后，Invoke 的调用结果才会被自动累计；脱离 Registry
// 的 Skill（如测试桩）不会自动累计指标。
type BaseSkill struct {
	code            string
	name            string
	tags            []string
	version         string
	category        string // ga / pilot / experimental
	requiredPerms   []string
	capabilities    []string
	provider        string
	author          string
	description     string
	longDescription string

	mu      sync.Mutex
	metrics SkillMetrics
}

// NewBaseSkill 构造一个 BaseSkill。
// category 取值约定：ga / pilot / experimental。
// requiredPermissions 至少声明一项（manifest 校验硬约束）。
func NewBaseSkill(code, name, version, category string, tags []string, requiredPermissions []string, capabilities []string) *BaseSkill {
	return &BaseSkill{
		code:          code,
		name:          name,
		tags:          append([]string(nil), tags...),
		version:       version,
		category:      category,
		requiredPerms: append([]string(nil), requiredPermissions...),
		capabilities:  append([]string(nil), capabilities...),
	}
}

// SkillIdentity 返回 BaseSkill 的标识字段，便于具体 skill 在 Manifest() 覆写时复用。
type SkillIdentity struct {
	Code            string
	Name            string
	Version         string
	Category        string
	Tags            []string
	RequiredPerms   []string
	Capabilities    []string
	Provider        string
	Author          string
	Description     string
	LongDescription string
}

// Identity 返回当前 BaseSkill 的全部标识字段（拷贝）。
func (b *BaseSkill) Identity() SkillIdentity {
	b.mu.Lock()
	defer b.mu.Unlock()
	return SkillIdentity{
		Code:            b.code,
		Name:            b.name,
		Version:         b.version,
		Category:        b.category,
		Tags:            append([]string(nil), b.tags...),
		RequiredPerms:   append([]string(nil), b.requiredPerms...),
		Capabilities:    append([]string(nil), b.capabilities...),
		Provider:        b.provider,
		Author:          b.author,
		Description:     b.description,
		LongDescription: b.longDescription,
	}
}

// With* 系列配置器（链式可选），便于在 builtin 注册处一行完成元数据补全。

func (b *BaseSkill) WithProvider(p string) *BaseSkill { b.provider = p; return b }
func (b *BaseSkill) WithAuthor(a string) *BaseSkill   { b.author = a; return b }
func (b *BaseSkill) WithDescription(d string) *BaseSkill {
	b.description = d
	return b
}

func (b *BaseSkill) WithLongDescription(d string) *BaseSkill {
	b.longDescription = d
	return b
}

// Code skill 唯一标识（e.g. ai.triage）。
func (b *BaseSkill) Code() string { return b.code }

// Name skill 显示名。
func (b *BaseSkill) Name() string { return b.name }

// Tags skill 标签快照。
func (b *BaseSkill) Tags() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.tags...)
}

// GetMetrics 线程安全地返回运行时指标快照。
func (b *BaseSkill) GetMetrics() SkillMetrics {
	b.mu.Lock()
	defer b.mu.Unlock()
	m := b.metrics
	return m
}

// TrackedSkill 供 SkillRegistry 判定是否自动累计指标。
func (b *BaseSkill) TrackedSkill() {}

// RecordResult Registry 在每次 Invoke 后自动调用以累计指标。
// 计算：
//   - TotalCalls 自增
//   - ErrorCount 仅在 err != nil 时自增
//   - SuccessRate = successCount / TotalCalls
//   - AvgLatencyMs 使用指数移动平均（alpha=0.2）以平衡近期与历史权重
func (b *BaseSkill) RecordResult(elapsed time.Duration, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.metrics.TotalCalls++
	b.metrics.LastUsedAt = time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		b.metrics.ErrorCount++
	}
	success := b.metrics.TotalCalls - b.metrics.ErrorCount
	b.metrics.SuccessRate = successRate(success, b.metrics.TotalCalls)
	b.metrics.AvgLatencyMs = updateEMA(b.metrics.AvgLatencyMs, elapsed.Milliseconds(), 0.2, b.metrics.TotalCalls)
}

func successRate(successes, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(successes) / float64(total)
}

// updateEMA 指数移动平均；第一次样本取该值直接作为基线。
func updateEMA(prev, sample int64, alpha float64, count int64) int64 {
	if count <= 1 {
		return sample
	}
	return int64(float64(prev)*(1-alpha) + float64(sample)*alpha)
}

// BuildManifest 由 BaseSkill 的字段合成 SkillManifest，自动计算 checksum。
//
// 通常在具体 skill 的 Manifest() 内部调用，再覆盖部分特有字段（如 InputSchema/OutputSchema）。
func (b *BaseSkill) BuildManifest() SkillManifest {
	id := b.Identity()
	m := SkillManifest{
		Name:                id.Code,
		Version:             id.Version,
		Title:               id.Name,
		Provider:            id.Provider,
		Description:         id.Description,
		LongDescription:     id.LongDescription,
		Tags:                id.Tags,
		Category:            id.Category,
		Capabilities:        id.Capabilities,
		RequiredPermissions: id.RequiredPerms,
		Author:              id.Author,
		IsOfficial:          id.Author == "itsm-backend" || id.Author == "official",
	}
	m.Checksum = m.ComputeChecksum()
	return m
}

// _sortedTags 工具：返回排序后的 tag 列表（用于不区分调用顺序的相等比较，保留用于将来使用）。
//
//lint:ignore U1000 utility reserved for future use
func _sortedTags(tags []string) []string {
	out := append([]string(nil), tags...)
	sort.Strings(out)
	return out
}

// _tagSetEqual 比较两个 tag 切片是否代表同一集合（保留用于将来使用）。
//
//lint:ignore U1000 utility reserved for future use
func _tagSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := _sortedTags(a)
	bs := _sortedTags(b)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}
