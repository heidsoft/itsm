package service

import (
	"sort"
	"sync"
	"time"
)

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

// trackedSkill 暴露给 SkillRegistry 的内部钩子。
func (b *BaseSkill) trackedSkill() {}

// recordResult Registry 在每次 Invoke 后自动调用以累计指标。
// 计算：
//   - TotalCalls 自增
//   - ErrorCount 仅在 err != nil 时自增
//   - SuccessRate = successCount / TotalCalls
//   - AvgLatencyMs 使用指数移动平均（alpha=0.2）以平衡近期与历史权重
func (b *BaseSkill) recordResult(elapsed time.Duration, err error) {
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

// sortedTags 工具：返回排序后的 tag 列表（用于不区分调用顺序的相等比较）。
func sortedTags(tags []string) []string {
	out := append([]string(nil), tags...)
	sort.Strings(out)
	return out
}

// tagSetEqual 比较两个 tag 切片是否代表同一集合。
func tagSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := sortedTags(a)
	bs := sortedTags(b)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}
