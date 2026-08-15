// Package service - Sprint C Skill Registry v1.
//
// SkillRegistry 是运行期 Skill 一等公民的中央索引：
//   - 启动时由 internal/bootstrap 通过 Register 注入 built-in Skill
//   - 通过 Get / List / ListByTag 提供给管理面板/路由发现
//   - 通过 Invoke 提供统一的运行时调用入口（含指标累计）
//
// 该注册表是单进程内存表，未直接持久化自定义 Skill。本期范围聚焦"统一抽象"
// 与"管理/发现"，DB-backed Skill 元数据存储留给后续 Sprint。
package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// SkillRegistry 注册表（运行期内存中心）。
type SkillRegistry struct {
	mu       sync.RWMutex
	skills   map[string]Skill
	tagIndex map[string]map[string]struct{} // tag -> codes 集合
}

func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills:   make(map[string]Skill),
		tagIndex: make(map[string]map[string]struct{}),
	}
}

// ErrSkillAlreadyRegistered skill code 已注册。
var ErrSkillAlreadyRegistered = errors.New("skill already registered")

// ErrSkillNotFound skill code 未注册。
var ErrSkillNotFound = errors.New("skill not found")

// ErrSkillValidation 注册或调用前的 manifest/input 校验失败。
var ErrSkillValidation = errors.New("skill validation failed")

// ErrSkillInvoke 调用期失败。
var ErrSkillInvoke = errors.New("skill invocation failed")

// Register 注册一个 Skill。code 必须唯一，且 manifest 必须通过基本完整性校验
// （name/version/permissions 三项必填），以保持 marketplace 视图的有效性。
func (r *SkillRegistry) Register(skill Skill) error {
	if skill == nil {
		return fmt.Errorf("%w: nil skill", ErrSkillValidation)
	}
	code := skill.Code()
	if code == "" {
		return fmt.Errorf("%w: empty skill code", ErrSkillValidation)
	}
	manifest := skill.Manifest()
	if err := manifest.ValidateForRegistration(); err != nil {
		return fmt.Errorf("%w: %v", ErrSkillValidation, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.skills[code]; exists {
		return fmt.Errorf("%w: %s", ErrSkillAlreadyRegistered, code)
	}
	r.skills[code] = skill
	for _, tag := range skill.Tags() {
		if tag == "" {
			continue
		}
		if r.tagIndex[tag] == nil {
			r.tagIndex[tag] = make(map[string]struct{})
		}
		r.tagIndex[tag][code] = struct{}{}
	}
	return nil
}

// Unregister 删除注册项（用于管理 API 的"禁用/删除"动作）。
// 注意：审计记录与历史调用仍保留在 DB 中，仅运行时不再可见。
func (r *SkillRegistry) Unregister(code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.skills[code]
	if !ok {
		return fmt.Errorf("%w: %s", ErrSkillNotFound, code)
	}
	delete(r.skills, code)
	for _, tag := range s.Tags() {
		if set, ok := r.tagIndex[tag]; ok {
			delete(set, code)
			if len(set) == 0 {
				delete(r.tagIndex, tag)
			}
		}
	}
	return nil
}

// Get 按 code 取出 Skill。
func (r *SkillRegistry) Get(code string) (Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[code]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSkillNotFound, code)
	}
	return s, nil
}

// MustGet 取出 Skill；找不到则 panic。仅在 bootstrap 期间使用，handler 不应调用。
func (r *SkillRegistry) MustGet(code string) Skill {
	s, err := r.Get(code)
	if err != nil {
		panic(err)
	}
	return s
}

// List 全部 Skill 的快照（code 字典序）。
func (r *SkillRegistry) List() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Skill, 0, len(r.skills))
	for _, s := range r.skills {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code() < out[j].Code() })
	return out
}

// ListByTag 按 tag 过滤 Skill。
func (r *SkillRegistry) ListByTag(tag string) []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	codes := r.tagIndex[tag]
	if len(codes) == 0 {
		return nil
	}
	out := make([]Skill, 0, len(codes))
	for code := range codes {
		if s, ok := r.skills[code]; ok {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code() < out[j].Code() })
	return out
}

// Codes 返回全部注册的 code 列表（按字典序）。
func (r *SkillRegistry) Codes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	codes := make([]string, 0, len(r.skills))
	for c := range r.skills {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	return codes
}

// Count 返回注册数量。
func (r *SkillRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// SkillEntry 管理面板 / API 视图：Skill 的扁平可序列化快照。
//
// 与 SkillManifest 的区别：SkillEntry 折叠了 Code/Name/Metrics 等从 Skill
// 接口直接派生出的字段，方便前端直接渲染而不必再下钻 Manifest。
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
	Status          string        `json:"status"` // active / disabled
	Manifest        SkillManifest `json:"manifest"`
	Metrics         SkillMetrics  `json:"metrics"`
	LongDescription string        `json:"longDescription,omitempty"`
}

// ListEntries 全部 Skill 的可序列化视图。
func (r *SkillRegistry) ListEntries() []SkillEntry {
	return r.entries(r.List())
}

// ListEntriesByTag 按 tag 过滤的可序列化视图。
func (r *SkillRegistry) ListEntriesByTag(tag string) []SkillEntry {
	return r.entries(r.ListByTag(tag))
}

// GetEntry 单个 Skill 的可序列化视图。
func (r *SkillRegistry) GetEntry(code string) (SkillEntry, error) {
	s, err := r.Get(code)
	if err != nil {
		return SkillEntry{}, err
	}
	entries := r.entries([]Skill{s})
	return entries[0], nil
}

func (r *SkillRegistry) entries(skills []Skill) []SkillEntry {
	out := make([]SkillEntry, 0, len(skills))
	for _, s := range skills {
		m := s.Manifest()
		out = append(out, SkillEntry{
			Code:            s.Code(),
			Name:            s.Name(),
			Version:         m.Version,
			Title:           m.Title,
			Provider:        m.Provider,
			Description:     m.Description,
			LongDescription: m.LongDescription,
			Category:        m.Category,
			Tags:            s.Tags(),
			Capabilities:    m.Capabilities,
			RequiredPerms:   m.RequiredPermissions,
			Author:          m.Author,
			IsOfficial:      m.IsOfficial,
			Checksum:        m.Checksum,
			IsBuiltin:       true,
			Status:          "active",
			Manifest:        m,
			Metrics:         s.GetMetrics(),
		})
	}
	return out
}

// SkillInvokeResult Invoke 的结构化返回，包含 latency / 错误码 / 是否被 metrics 追踪。
type SkillInvokeResult struct {
	Output         interface{}
	LatencyMs      int64
	SkippedMetrics bool
}

// Invoke 是 Skill 的统一运行时调用入口。
//
// 调用流程：
//  1. Get(code) 查找 Skill
//  2. Validate(input) 校验输入
//  3. 计时 + 调用 Execute
//  4. 若 Skill 嵌入了 *BaseSkill，自动累计 TotalCalls / SuccessRate / AvgLatencyMs
//  5. 返回结果
//
// 返回错误：
//   - code 不存在：ErrSkillNotFound
//   - input 不合法：ErrSkillValidation + 详细原因
//   - Execute 抛错：ErrSkillInvoke 包装
func (r *SkillRegistry) Invoke(ctx context.Context, code string, input interface{}) (interface{}, error) {
	return r.invokeWithMetrics(ctx, code, input)
}

// InvokeWithMetrics 同 Invoke，但额外返回 latency 与 metrics 记录状态（用于管理面板调试）。
func (r *SkillRegistry) InvokeWithMetrics(ctx context.Context, code string, input interface{}) (SkillInvokeResult, error) {
	s, err := r.Get(code)
	if err != nil {
		return SkillInvokeResult{}, err
	}
	if err := s.Validate(input); err != nil {
		return SkillInvokeResult{}, fmt.Errorf("%w: %v", ErrSkillValidation, err)
	}
	bs, hasBase := s.(trackedSkill)

	start := time.Now()
	out, execErr := s.Execute(ctx, input)
	elapsed := time.Since(start)
	tracked := false
	if hasBase {
		bs.recordResult(elapsed, execErr)
		tracked = true
	}
	if execErr != nil {
		return SkillInvokeResult{
			Output:         out,
			LatencyMs:      elapsed.Milliseconds(),
			SkippedMetrics: !tracked,
		}, fmt.Errorf("%w: %v", ErrSkillInvoke, execErr)
	}
	return SkillInvokeResult{
		Output:         out,
		LatencyMs:      elapsed.Milliseconds(),
		SkippedMetrics: !tracked,
	}, nil
}

func (r *SkillRegistry) invokeWithMetrics(ctx context.Context, code string, input interface{}) (interface{}, error) {
	res, err := r.InvokeWithMetrics(ctx, code, input)
	return res.Output, err
}

// trackedSkill 内部接口：BaseSkill 私有地实现以让 Registry 累计指标。
// 不导出，避免被外部 Skill 实现误用导致统计口径分裂。
//
// 同时承担：身份钩子 trackedSkill() + 单次结果统计 recordResult(elapsed, err)。
// 之所以要求 skill 同时实现这两个方法，是为了让 Registry 知道"哪些调用会被
// 自动统计"，并避免为不嵌入 BaseSkill 的自定义 Skill 引入静默指标丢失。
type trackedSkill interface {
	trackedSkill()
	recordResult(elapsed time.Duration, err error)
}
