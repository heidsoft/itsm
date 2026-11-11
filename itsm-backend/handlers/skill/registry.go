// Skill Registry v1.
//
// SkillRegistry 是运行期 Skill 一等公民的中央索引：
//   - 启动时由 internal/bootstrap 通过 Register 注入 built-in Skill
//   - 通过 Get / List / ListByTag 提供给管理面板/路由发现
//   - 通过 Invoke 提供统一的运行时调用入口（含指标累计）
//
// 该注册表是单进程内存表，未直接持久化自定义 Skill。本期范围聚焦"统一抽象"
// 与"管理/发现"，DB-backed Skill 元数据存储留给后续 Sprint。
package skill

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"itsm-backend/dto"
)

// Skill 接口与值类型的唯一实现在 dto 包；域内沿用短名（域包别名，
// 既有调用点零改动）。
type Skill = dto.Skill

// SkillRegistry 注册表（运行期内存中心）。
//
// 租户维度：注册表是单进程共享的，但「共享」不等于「互通」。内置技能由
// bootstrap 注册，owner 记为 0（全局可见，所有租户都能发现/调用）；管理端
// 注册的自定义技能记录创建它的租户 ID，只有该租户能看到、改、删、调用。
//
// 没有这层归属判定时，任一租户调 GET /api/v1/skills 就能列出别人注册的技能，
// 也能凭 code 直接 PUT/DELETE 覆盖或删掉别的租户的技能——注册表成了跨租户
// 横向移动的通道。
type SkillRegistry struct {
	mu       sync.RWMutex
	skills   map[string]Skill
	owners   map[string]int                 // code -> 创建租户 ID；0 = 内置/全局
	tagIndex map[string]map[string]struct{} // tag -> codes 集合
}

func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills:   make(map[string]Skill),
		owners:   make(map[string]int),
		tagIndex: make(map[string]map[string]struct{}),
	}
}

// ErrSkillAlreadyRegistered skill code 已注册。
var ErrSkillAlreadyRegistered = dto.ErrSkillAlreadyRegistered

// ErrSkillNotFound skill code 未注册。
var ErrSkillNotFound = dto.ErrSkillNotFound

// ErrSkillValidation 注册或调用前的 manifest/input 校验失败。
var ErrSkillValidation = dto.ErrSkillValidation

// ErrSkillInvoke 调用期失败。
var ErrSkillInvoke = dto.ErrSkillInvoke

// Register 注册一个 Skill。code 必须唯一，且 manifest 必须通过基本完整性校验
// （name/version/permissions 三项必填），以保持 marketplace 视图的有效性。
//
// 用于**内置技能**（bootstrap 阶段）：owner 记为 0，所有租户可见。
// 租户注册自定义技能请用 RegisterScoped。
func (r *SkillRegistry) Register(skill Skill) error {
	return r.register(0, skill)
}

// RegisterScoped 以租户身份注册自定义技能。
//
// code 是全局唯一标识：已被内置技能或其它租户占用时返回
// ErrSkillAlreadyRegistered（handler 映射为 409），不会覆盖或抢占。
func (r *SkillRegistry) RegisterScoped(tenantID int, skill Skill) error {
	return r.register(tenantID, skill)
}

func (r *SkillRegistry) register(owner int, skill Skill) error {
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
	r.owners[code] = owner
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
//
// 该函数不校验归属，仅供 bootstrap/测试使用；租户面删除请用 UnregisterScoped。
func (r *SkillRegistry) Unregister(code string) error {
	return r.unregister(0, code, true)
}

// UnregisterScoped 以租户身份删除技能。
//
// 只能删自己注册的：内置技能（owner=0）与其它租户的技能一律返回
// ErrSkillNotFound —— 用「不存在」而不是「无权限」，避免通过错误码探测
// 某个 code 是否被别的租户注册。
func (r *SkillRegistry) UnregisterScoped(tenantID int, code string) error {
	return r.unregister(tenantID, code, false)
}

func (r *SkillRegistry) unregister(tenantID int, code string, allowAnyOwner bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.skills[code]
	if !ok {
		return fmt.Errorf("%w: %s", ErrSkillNotFound, code)
	}
	if !allowAnyOwner && r.owners[code] != tenantID {
		return fmt.Errorf("%w: %s", ErrSkillNotFound, code)
	}
	delete(r.skills, code)
	delete(r.owners, code)
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

// GetScoped 按租户视角取 Skill：内置技能（owner=0）与本人租户注册的技能可见，
// 其它租户的同 code 技能按「不存在」处理。
func (r *SkillRegistry) GetScoped(tenantID int, code string) (Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[code]
	if !ok || !r.visibleLocked(tenantID, code) {
		return nil, fmt.Errorf("%w: %s", ErrSkillNotFound, code)
	}
	return s, nil
}

// visibleLocked 判定某 code 对租户是否可见。调用方必须已持有读锁。
func (r *SkillRegistry) visibleLocked(tenantID int, code string) bool {
	owner, ok := r.owners[code]
	if !ok {
		return false
	}
	return owner == 0 || owner == tenantID
}

// OwnerOf 返回 code 的归属租户（0 表示内置/全局）。第二个返回值为 false 表示未注册。
func (r *SkillRegistry) OwnerOf(code string) (int, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	owner, ok := r.owners[code]
	return owner, ok
}

// Has 判断 code 是否已有运行实现。
//
// 市场侧交付校验用这个而不是 Get：技能商品指向一个编译不进本进程的 code 时，
// 「查不到」就是它无法交付的原因，不需要把整个 Skill 接口暴露给 marketplace 包。
func (r *SkillRegistry) Has(code string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.skills[code]
	return ok
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

// ListScoped 返回某租户可见的 Skill 快照（内置 + 本租户），code 字典序。
func (r *SkillRegistry) ListScoped(tenantID int) []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Skill, 0, len(r.skills))
	for code, s := range r.skills {
		if r.visibleLocked(tenantID, code) {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code() < out[j].Code() })
	return out
}

// ListByTagScoped 按 tag 过滤某租户可见的 Skill。
func (r *SkillRegistry) ListByTagScoped(tenantID int, tag string) []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	codes := r.tagIndex[tag]
	if len(codes) == 0 {
		return nil
	}
	out := make([]Skill, 0, len(codes))
	for code := range codes {
		if !r.visibleLocked(tenantID, code) {
			continue
		}
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

// SkillEntry 值类型唯一实现在 dto 包；此处保留同名别名以保持域包公开 API。
type SkillEntry = dto.SkillEntry

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

// ListEntriesScoped 返回某租户可见的可序列化视图（内置 + 本租户）。
func (r *SkillRegistry) ListEntriesScoped(tenantID int) []SkillEntry {
	return r.entries(r.ListScoped(tenantID))
}

// ListEntriesByTagScoped 按 tag 过滤某租户可见的可序列化视图。
func (r *SkillRegistry) ListEntriesByTagScoped(tenantID int, tag string) []SkillEntry {
	return r.entries(r.ListByTagScoped(tenantID, tag))
}

// GetEntryScoped 取单个技能的可序列化视图；不属于本租户时返回 ErrSkillNotFound。
func (r *SkillRegistry) GetEntryScoped(tenantID int, code string) (SkillEntry, error) {
	s, err := r.GetScoped(tenantID, code)
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
		// owner 由注册表维护；查不到时按内置处理（bootstrap 之前注册的路径）。
		owner, ok := r.OwnerOf(s.Code())
		if !ok {
			owner = 0
		}
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
			IsBuiltin:       owner == 0,
			TenantID:        owner,
			Status:          "active",
			Manifest:        m,
			Metrics:         s.GetMetrics(),
		})
	}
	return out
}

// SkillInvokeResult 值类型唯一实现在 dto 包；此处保留同名别名以保持域包公开 API。
type SkillInvokeResult = dto.SkillInvokeResult

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
		bs.RecordResult(elapsed, execErr)
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

// trackedSkill 内部接口：BaseSkill 以导出钩子实现，供 Registry 累计指标。
// 接口本身不导出，避免被外部 Skill 实现误用导致统计口径分裂。
//
// 同时承担：身份钩子 TrackedSkill() + 单次结果统计 RecordResult(elapsed, err)。
// 之所以要求 skill 同时实现这两个方法，是为了让 Registry 知道"哪些调用会被
// 自动统计"，并避免为不嵌入 BaseSkill 的自定义 Skill 引入静默指标丢失。
type trackedSkill interface {
	TrackedSkill()
	RecordResult(elapsed time.Duration, err error)
}
