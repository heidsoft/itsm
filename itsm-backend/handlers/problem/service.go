package problem

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/handlers/knowledge"
	"itsm-backend/internal/datascope"
	"itsm-backend/service"
	"itsm-backend/service/sla"

	"go.uber.org/zap"
)

type Service struct {
	repo      Repository
	validator AssigneeValidator
	logger    *zap.SugaredLogger
	// slaSvc 提供 SLA 截止时间计算与 sla_states 持久化（2026-10-21 GA 阻断项落地）。
	// 与 incident 域同一装配模式：handler 域不自己算 deadline，词汇表与存储收敛在
	// service 层单一实现；nil 时（单测/旧组装）问题照常创建，只是不带 SLA 绑定。
	slaSvc *service.TicketSLAService
	// kbRepo 知识沉淀目标仓储（2026-10-22 沉淀闭环落地）。
	// nil 时（单测/旧组装）沉淀端点显式 503，反查端点返回空列表。
	kbRepo KnowledgeSink
}

// AssigneeValidator 校验受指人属于同一租户且处于在职可用状态。
// 实现由 bootstrap 注入（当前复用 service.TicketService.ValidateAssignee，
// 与 ticket 域同一实现）——同一校验用例不复制第二份实现；
// placeholders 为 nil 时受指人引用必须 fail-closed，禁止跨租户关系注入。
type AssigneeValidator interface {
	ValidateAssignee(ctx context.Context, assigneeID, tenantID int) error
}

// IsNotFound keeps persistence error classification behind the domain service.
func (s *Service) IsNotFound(err error) bool {
	return ent.IsNotFound(err)
}

func NewService(repo Repository, validator AssigneeValidator, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
		logger:    logger,
	}
}

// SetSLAService 注入 SLA 服务（截止时间计算 + sla_states 持久化）。
// 装配缺失时问题域降级为「无 SLA 绑定」而非拒绝创建——能力缺失不得阻断主流程。
func (s *Service) SetSLAService(slaSvc *service.TicketSLAService) {
	s.slaSvc = slaSvc
}

// KnowledgeSink 知识沉淀目标的窄接口。由 bootstrap 注入 handlers/knowledge
// 的 EntRepository 实现——问题域只依赖「建文章 + 按来源反查」两个用例，
// 不感知知识域的评审/版本/RAG 细节，知识域演进不牵连问题域。
type KnowledgeSink interface {
	Create(ctx context.Context, a *knowledge.Article) (*knowledge.Article, error)
	ListBySourceProblem(ctx context.Context, tenantID, problemID int) ([]*knowledge.Article, error)
}

// SetKnowledgeRepository 注入知识沉淀目标仓储（装配缺失时沉淀端点显式 503）。
func (s *Service) SetKnowledgeRepository(kb KnowledgeSink) {
	s.kbRepo = kb
}

// bindSLAOnCreate 问题创建后绑定 SLA：按 problem 服务类型 + 优先级匹配 SLA 定义，
// 计算响应/解决截止时间并写入 sla_states（aggregate_type="problem"）。
// 与工单/事件同一 Store、同一多态表，不复制第二套计时实现。
//
// 与工单的刻意差异：工单在无匹配定义时套用默认 SLA（60/480 分钟），问题不套用。
// 问题域投产初期若给存量问题静默绑上隐藏默认值，下一轮 watcher 会让 8 小时前的
// 存量问题全部违规——那是配置噪音不是治理信号。无匹配定义即不绑定，
// 由 GET /problems/:id/sla 的 bound=false 如实回答「没配 SLA」。
func (s *Service) bindSLAOnCreate(ctx context.Context, tenantID, problemID int, priority string) {
	if s.slaSvc == nil {
		return
	}
	slaResult, err := s.slaSvc.CalculateSLADeadlineFromRequest(ctx, tenantID, "problem", priority)
	if err != nil {
		// 查询失败只是「该问题没有 SLA」，不是创建失败。
		s.logger.Warnw("Failed to calculate problem SLA deadline, continuing without SLA binding",
			"tenant_id", tenantID, "problem_id", problemID, "error", err)
		return
	}
	if slaResult == nil || slaResult.SLADefinitionID == 0 {
		// 无匹配的 problem 类型 SLA 定义（或走了默认兜底）：不绑定。
		return
	}
	store := s.slaSvc.SLAStore()
	if store == nil {
		return
	}
	if err := store.SaveDeadlines(ctx, tenantID, "problem", problemID, slaResult.SLADefinitionID, slaResult.ResponseDeadline, slaResult.ResolutionDeadline); err != nil {
		s.logger.Warnw("Failed to save problem SLA state, continuing", "problem_id", problemID, "error", err)
	}
}

// recordResolutionOnSLA 问题进入 resolved/closed 时在 sla_states 打解决时间戳。
// 迟到解决不抹除违规事实（已 breached 的状态保持 breached），未违才转 met。
func (s *Service) recordResolutionOnSLA(ctx context.Context, tenantID, problemID int, at time.Time) {
	if s.slaSvc == nil {
		return
	}
	store := s.slaSvc.SLAStore()
	if store == nil {
		return
	}
	if err := store.RecordResolution(ctx, tenantID, "problem", problemID, at); err != nil {
		s.logger.Warnw("Failed to record problem SLA resolution", "problem_id", problemID, "error", err)
	}
}

// GetProblemSLA 读取问题 SLA 状态（GET /api/v1/problems/:id/sla 的真实实现）。
//
// 取代此前的显式 503 桩： Bound=false 如实回答「该问题没有绑定 SLA 定义」，
// 与「绑定了且未超时」是两种语义，调用方不得混淆（旧桩注释里的契约理由）。
// 状态由存储态 + 纯函数推导合成：库里只有 active/met，实时 ok/warning/breached
// 由 sla.SLAStatus 按截止时间与当前时刻推导——该词汇表单一来源在 service/sla，
// 此处不复制第二套判定。
func (s *Service) GetProblemSLA(ctx context.Context, id, tenantID int) (*ProblemSLAInfo, error) {
	p, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	info := &ProblemSLAInfo{
		ProblemID:     p.ID,
		ProblemNumber: p.ProblemNumber,
		Priority:      p.Priority,
		Bound:         false,
		Status:        "unbound",
	}

	if s.slaSvc == nil {
		return info, nil
	}
	store := s.slaSvc.SLAStore()
	if store == nil {
		return info, nil
	}
	state, err := store.GetState(ctx, tenantID, "problem", id)
	if err != nil || state == nil {
		// 无 sla_state 行 = 未绑定（租户没配 problem 定义或优先级无匹配）。
		return info, nil
	}

	info.Bound = true
	info.SLADefinitionID = state.SLADefinitionID
	info.Status = state.Status
	info.PausedDurationSec = state.PausedDurationSeconds
	if !state.ResponseDeadline.IsZero() {
		t := state.ResponseDeadline
		info.ResponseDeadline = &t
	}
	if !state.ResolutionDeadline.IsZero() {
		t := state.ResolutionDeadline
		info.ResolutionDeadline = &t
	}
	if !state.FirstResponseAt.IsZero() {
		t := state.FirstResponseAt
		info.FirstResponseAt = &t
	}
	if !state.ResolvedAt.IsZero() {
		t := state.ResolvedAt
		info.ResolvedAt = &t
	}

	// 暂停中的计时冻结与已解决都不做实时推导，直接信任存储态。
	if state.Status != "paused" && state.ResolvedAt.IsZero() {
		derived := sla.SLAStatus(time.Now(),
			info.ResponseDeadline, info.ResolutionDeadline,
			state.FirstResponseAt, state.ResolvedAt)
		if derived == "breached" {
			info.Status = "breached"
		} else if derived == "warning" && state.Status == "active" {
			info.Status = "warning"
		}
	}
	return info, nil
}

// ── 评论（2026-10-21 GA 阻断项落地）──────────────────────────────────────────

// AddComment 新增问题评论。content 由 handler 侧已做 Trim；此处做领域校验
// （非空 + 问题可见性由 repository 的 JOIN 校验保证）。
func (s *Service) AddComment(ctx context.Context, tenantID, problemID, userID int, content string, isInternal bool, mentions, attachments []int) (*ProblemComment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, common.NewBusinessError(common.ValidationError, "评论内容不能为空", "empty content")
	}
	return s.repo.CreateComment(ctx, &ProblemComment{
		ProblemID:   problemID,
		UserID:      userID,
		Content:     content,
		IsInternal:  isInternal,
		Mentions:    mentions,
		Attachments: attachments,
		TenantID:    tenantID,
	})
}

// ListComments 分页列出问题评论（时间正序）。
func (s *Service) ListComments(ctx context.Context, tenantID, problemID int, page, size int) ([]*ProblemComment, int, error) {
	return s.repo.ListComments(ctx, tenantID, problemID, page, size)
}

// ── 知识沉淀闭环（2026-10-22 落地）────────────────────────────────────────────
//
// 问题域与知识域的双向断链是知识域预览期的核心缺口：
//   - 正向：问题解决后没有「沉淀为知识」的动作，根因/方案散落在问题单里，
//     RAG 检索不到，同一个故障换个人处理还得重新踩坑；
//   - 反向：调查摘要里的 KnowledgeArticles 字段恒为空数组（契约存在、实现缺席）。
//
// 沉淀产物按知识域既有治理流走：创建即 draft，发布必须过评审——
// 沉淀降低的是记录成本，不是审核成本。

// PromoteKnowledgeInput 沉淀请求的可选覆盖项。nil/空串字段用问题单内容兜底。
type PromoteKnowledgeInput struct {
	Title    string   // 空 = 问题标题
	Category string   // 空 = 故障排查
	Tags     []string // 追加在自动标签（问题沉淀 + 问题编号）之后，去重
}

// PromoteToKnowledge 把问题的根因/解决方案沉淀为知识草稿（POST /problems/:id/knowledge）。
//
// 准入门槛：至少要有根因或解决方案之一。没有可复用结论的问题沉淀出来
// 只是一篇空壳文章，污染 RAG 语料——这是质量门，不是流程绊脚石。
// 允许同一问题沉淀多篇（不同侧面，如「根因复盘」与「预防指南」），不做唯一性约束。
func (s *Service) PromoteToKnowledge(ctx context.Context, tenantID, problemID, userID int, in PromoteKnowledgeInput) (*knowledge.Article, error) {
	p, err := s.repo.Get(ctx, problemID, tenantID)
	if err != nil {
		return nil, err
	}

	rootCause := strings.TrimSpace(p.RootCause)
	resolution := strings.TrimSpace(p.Resolution)
	if rootCause == "" && resolution == "" {
		return nil, common.NewBusinessError(common.UnprocessableEntityCode,
			"问题尚无根因或解决方案，沉淀为知识没有可复用价值", "root_cause and resolution both empty")
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = p.Title
	}
	category := strings.TrimSpace(in.Category)
	if category == "" {
		category = "故障排查"
	}

	// 内容模板按问题单字段分段组装，空段跳过。
	var b strings.Builder
	b.WriteString("## 问题现象\n\n")
	b.WriteString(strings.TrimSpace(p.Description))
	b.WriteString("\n")
	if rootCause != "" {
		b.WriteString("\n## 根因分析\n\n")
		b.WriteString(rootCause)
		b.WriteString("\n")
	}
	if wa := strings.TrimSpace(p.Workaround); wa != "" {
		b.WriteString("\n## 临时措施\n\n")
		b.WriteString(wa)
		b.WriteString("\n")
	}
	if resolution != "" {
		b.WriteString("\n## 解决方案\n\n")
		b.WriteString(resolution)
		b.WriteString("\n")
	}

	// 自动标签标明沉淀来源；来源追溯的权威字段是 source_problem_id，标签只服务检索。
	tags := []string{"问题沉淀", p.ProblemNumber}
	for _, t := range in.Tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		dup := false
		for _, existing := range tags {
			if existing == t {
				dup = true
				break
			}
		}
		if !dup {
			tags = append(tags, t)
		}
	}

	if s.kbRepo == nil {
		return nil, common.NewBusinessError(common.ServiceUnavailableCode,
			"知识沉淀能力未装配", "knowledge repository not wired in bootstrap")
	}

	return s.kbRepo.Create(ctx, &knowledge.Article{
		Title:           title,
		Content:         b.String(),
		Category:        category,
		Tags:            tags,
		AuthorID:        userID,
		TenantID:        tenantID,
		IsPublished:     false, // 草稿进评审流，沉淀不等于发布
		SourceProblemID: &problemID,
	})
}

// ListProblemKnowledge 反查问题沉淀的知识文章（GET /problems/:id/knowledge）。
// 先过问题可见性校验：跨租户问题 ID 与不存在一样返回 not visible，
// 不给「这篇知识来自别家租户的哪个问题」留探测面。
func (s *Service) ListProblemKnowledge(ctx context.Context, tenantID, problemID int) ([]*knowledge.Article, error) {
	if _, err := s.repo.Get(ctx, problemID, tenantID); err != nil {
		return nil, err
	}
	if s.kbRepo == nil {
		return []*knowledge.Article{}, nil
	}
	return s.kbRepo.ListBySourceProblem(ctx, tenantID, problemID)
}

// validateAssignee 统一受指人同租户/在职校验入口。
func (s *Service) validateAssignee(ctx context.Context, assigneeID, tenantID int) error {
	if assigneeID <= 0 {
		return fmt.Errorf("受理人 ID 无效")
	}
	if s.validator == nil {
		return fmt.Errorf("受理人校验器未配置，拒绝写入受指人引用")
	}
	if err := s.validator.ValidateAssignee(ctx, assigneeID, tenantID); err != nil {
		return fmt.Errorf("处理人不存在或不可用: %w", err)
	}
	return nil
}

func (s *Service) Create(ctx context.Context, tenantID int, p *Problem) (*Problem, error) {
	if strings.TrimSpace(p.Title) == "" {
		return nil, fmt.Errorf("problem title is required")
	}
	if !isValidProblemPriority(p.Priority) {
		return nil, fmt.Errorf("invalid problem priority: %s", p.Priority)
	}
	// AssigneeID 可指向任意租户的用户：写入前必须校验同租户 +在职，
	// 否则问题单（含根因等敏感内容）可以被静默指给别家租户员工。
	if p.AssigneeID != nil && *p.AssigneeID > 0 {
		if err := s.validateAssignee(ctx, *p.AssigneeID, tenantID); err != nil {
			return nil, err
		}
	}
	p.Title = strings.TrimSpace(p.Title)
	p.Status = "open"
	p.TenantID = tenantID
	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	// SLA 绑定在问题落库后进行：查不到定义只是「无 SLA」，不阻断创建
	// （与工单/事件同一口径——能力缺失降级，主流程不失败）。
	s.bindSLAOnCreate(ctx, tenantID, created.ID, created.Priority)
	return created, nil
}

func (s *Service) Get(ctx context.Context, id int, tenantID int) (*Problem, error) {
	return s.repo.Get(ctx, id, tenantID)
}

func (s *Service) GetWithAssociations(ctx context.Context, id int, tenantID int) (*Problem, error) {
	return s.repo.GetWithAssociations(ctx, id, tenantID)
}

func (s *Service) AddAssociations(ctx context.Context, tenantID, problemID int, relatedType string, relatedIDs []int) error {
	relatedIDs = uniquePositiveIDs(relatedIDs)
	if len(relatedIDs) == 0 {
		return fmt.Errorf("at least one related id is required")
	}
	return s.repo.AddAssociations(ctx, tenantID, problemID, relatedType, relatedIDs)
}

func (s *Service) RemoveAssociation(ctx context.Context, tenantID, problemID int, relatedType string, relatedID int) error {
	if relatedID <= 0 {
		return fmt.Errorf("invalid related id")
	}
	return s.repo.RemoveAssociation(ctx, tenantID, problemID, relatedType, relatedID)
}

// List 列出问题单。推广 ticket 的 DataScope 行级权限：
// 管理角色可见全租户，其余角色仅可见本人创建或分配给自己的问题单。
// currentUserID/currentRole 由 handler 从鉴权中间件注入的 user_id/role 取得。
func (s *Service) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, currentUserID int, currentRole string) ([]*Problem, int, error) {
	dataScope := datascope.DataScopeAll
	if !datascope.IsDataScopeAllRole(currentRole) {
		dataScope = datascope.DataScopeOwnedOrAssigned
	}
	return s.repo.List(ctx, tenantID, page, size, filters, dataScope, currentUserID)
}

// LoadUserNames 委托 repo 批量加载 user 显示名映射。查询失败仅记日志并返回空映射，
// 由调用方决定是否降级（一般继续返回带 ID 的响应，让前端降级展示）。
func (s *Service) LoadUserNames(ctx context.Context, tenantID int, ids []int) map[int]string {
	m, err := s.repo.LoadUserNames(ctx, tenantID, ids)
	if err != nil {
		s.logger.Warnw("LoadUserNames failed", "error", err, "tenant_id", tenantID)
		return map[int]string{}
	}
	return m
}

func (s *Service) Update(ctx context.Context, tenantID int, id int, p *Problem, actorID int, actorRole string) (*Problem, error) {
	existing, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	// 写权限 ⊆ 读权限：普通角色仅可修改本人创建或受理的问题单。
	if !datascope.CanWriteResource(actorID, actorRole, existing.CreatedBy, existing.AssigneeID) {
		return nil, common.NewForbiddenError("无权限修改该问题单：仅创建人、受理人或管理员可操作")
	}

	// Update fields if they are set (non-zero/non-empty check in Handler or here)
	// Here assuming 'p' contains only fields to update usually, but domain entity isn't partial.
	// We merge changes here.
	// CAS 预期：读取时的状态。service 校验与仓储写入之间的并发窗口里若状态被
	// 推进（如另一请求已完成 resolved→closed），条件更新未命中 → 409 冲突，
	// 绝不允许用本次校验结论覆盖推进后的状态。
	expectedStatus := existing.Status
	// enteredResolvedFreshly 捕获「本次调用首次把问题推进到 resolved/closed」，
	// 供仓储写入成功后打 SLA 解决时间戳（重复进入同一状态不打）。
	enteredResolvedFreshly := false

	if p.Title != "" {
		existing.Title = p.Title
	}
	if p.Description != "" {
		existing.Description = p.Description
	}
	if p.Status != "" {
		if !isValidProblemStatusTransition(existing.Status, p.Status) {
			// 业务状态机违规必须用 BusinessError(409) 表达，否则 handler 会误判为
			// 内部错误返回 500，前端会把合法的流程防护当成服务器故障。
			return nil, common.NewBusinessError(
				common.ConflictCode,
				"当前问题状态不允许此操作",
				fmt.Sprintf("%s -> %s", existing.Status, p.Status),
			)
		}
		// 先捕获前态再修改，时间戳分支才能正确判定"是否首次进入"。
		// 旧实现在状态写入后才比对 `existing.Status != "resolved"`，此时两侧永远相等，
		// 导致 ResolvedAt 从未被设置、MTTR 永远为 0、closed_at 派生也走错分支。
		prevStatus := existing.Status
		existing.Status = p.Status
		now := time.Now()
		switch p.Status {
		case "resolved":
			if prevStatus != "resolved" {
				existing.ResolvedAt = &now
				enteredResolvedFreshly = true
			}
			existing.ClosedAt = nil
		case "closed":
			existing.ClosedAt = &now
			if prevStatus != "closed" {
				enteredResolvedFreshly = true
			}
		case "investigating":
			if prevStatus == "resolved" || prevStatus == "closed" {
				existing.ResolvedAt = nil
				existing.ClosedAt = nil
			}
		}
	}
	if p.Priority != "" {
		if !isValidProblemPriority(p.Priority) {
			return nil, common.NewBusinessError(
				common.ParamErrorCode,
				"无效的问题优先级",
				p.Priority,
			)
		}
		existing.Priority = p.Priority
	}
	if p.Category != "" {
		existing.Category = p.Category
	}
	if p.RootCause != "" {
		existing.RootCause = p.RootCause
	}
	if p.Workaround != "" {
		existing.Workaround = p.Workaround
	}
	if p.Resolution != "" {
		existing.Resolution = p.Resolution
	}
	if p.Impact != "" {
		existing.Impact = p.Impact
	}
	if p.AssigneeID != nil {
		if *p.AssigneeID > 0 && existing.AssigneeID != nil && *existing.AssigneeID == *p.AssigneeID {
			// 与当前受理人相同：跳过校验，允许仅变更其他字段。
		} else {
			// AssigneeID 可指向任意租户的用户，必须与 Create 同强度的同租户校验。
			if err := s.validateAssignee(ctx, *p.AssigneeID, tenantID); err != nil {
				return nil, err
			}
		}
		existing.AssigneeID = p.AssigneeID
	}

	// 关闭是唯一需要「证据」的状态跃迁：必须已有解决方案与根因结论。
	// 放在字段合并之后校验，因为 CloseProblem 会在同一次调用里写入 resolution。
	if existing.Status == "closed" {
		if err := validateCloseEvidence(existing); err != nil {
			return nil, err
		}
	}

	result, err := s.repo.Update(ctx, existing, expectedStatus)
	if err != nil {
		if errors.Is(err, ErrProblemStatusMoved) {
			return nil, common.NewBusinessError(
				common.ConflictCode,
				"问题单状态已被并发修改，请刷新后重试",
				fmt.Sprintf("期望状态 '%s'", expectedStatus),
			)
		}
		return nil, err
	}
	// 首次进入 resolved/closed 时在 sla_states 打解决时间戳（MTTR 与达成率的事实来源）。
	// 重复进入同一状态不打（enteredResolvedFreshly 为 false）。
	if enteredResolvedFreshly {
		s.recordResolutionOnSLA(ctx, tenantID, id, time.Now())
	}
	return result, nil
}

// InvestigateProblem starts the investigation lifecycle for a problem.
func (s *Service) InvestigateProblem(ctx context.Context, tenantID, id, actorID int, actorRole string) (*Problem, error) {
	return s.Update(ctx, tenantID, id, &Problem{Status: "investigating"}, actorID, actorRole)
}

// UpdateRootCause records the confirmed root cause.
func (s *Service) UpdateRootCause(ctx context.Context, tenantID, id, actorID int, actorRole string, rootCause string) (*Problem, error) {
	rootCause = strings.TrimSpace(rootCause)
	if rootCause == "" {
		return nil, fmt.Errorf("rootCause is required")
	}
	return s.Update(ctx, tenantID, id, &Problem{RootCause: rootCause}, actorID, actorRole)
}

// UpdateSolution records a workaround and/or final resolution.
func (s *Service) UpdateSolution(ctx context.Context, tenantID, id, actorID int, actorRole string, workaround, resolution string) (*Problem, error) {
	workaround = strings.TrimSpace(workaround)
	resolution = strings.TrimSpace(resolution)
	if workaround == "" && resolution == "" {
		return nil, fmt.Errorf("solution, workaround or resolution is required")
	}
	return s.Update(ctx, tenantID, id, &Problem{Workaround: workaround, Resolution: resolution}, actorID, actorRole)
}

// CloseProblem closes a problem and optionally records its final resolution.
func (s *Service) CloseProblem(ctx context.Context, tenantID, id, actorID int, actorRole string, resolution string) (*Problem, error) {
	return s.Update(ctx, tenantID, id, &Problem{
		Status:     "closed",
		Resolution: strings.TrimSpace(resolution),
	}, actorID, actorRole)
}

// validateCloseEvidence 落实「问题关闭前必须明确根因和解决方案」这一商业不变量。
//
// 旧实现允许 investigating -> resolved -> closed 全程不填任何分析结论，
// 结果 KEDB 里堆满「已关闭但什么都没有」的问题单，既不能复用也不能审计。
// 根因确实无法确定时，要求把判定依据写进根因字段（例如「无法确定：日志已轮转」），
// 这样「无法确定」本身也是可审计的结论，而不是留空。
func validateCloseEvidence(p *Problem) error {
	if strings.TrimSpace(p.Resolution) == "" {
		return common.NewBusinessError(
			common.ConflictCode,
			"问题关闭前必须填写解决方案",
			"resolution is required before closing a problem",
		)
	}
	if strings.TrimSpace(p.RootCause) == "" {
		return common.NewBusinessError(
			common.ConflictCode,
			"问题关闭前必须明确根因；若无法确定，请在根因中写明判定依据",
			"root_cause is required before closing a problem",
		)
	}
	return nil
}

func isValidProblemPriority(priority string) bool {
	switch priority {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func isValidProblemStatusTransition(current, next string) bool {
	if current == next {
		return true
	}
	transitions := map[string]map[string]struct{}{
		"open":          {"investigating": {}, "identified": {}, "resolved": {}},
		"investigating": {"identified": {}, "resolved": {}},
		"identified":    {"investigating": {}, "resolved": {}},
		"resolved":      {"investigating": {}, "closed": {}},
		"closed":        {},
		// 兼容存量 in_progress 数据，仅允许进入规范状态。
		"in_progress": {"identified": {}, "resolved": {}},
	}
	allowed, ok := transitions[current]
	if !ok {
		return false
	}
	_, ok = allowed[next]
	return ok
}

func uniquePositiveIDs(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

// Delete 删除问题单，受行级写权限约束（写权限 ⊆ 读权限）。
func (s *Service) Delete(ctx context.Context, id int, tenantID int, actorID int, actorRole string) error {
	existing, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if !datascope.CanWriteResource(actorID, actorRole, existing.CreatedBy, existing.AssigneeID) {
		return common.NewForbiddenError("无权限删除该问题单：仅创建人、受理人或管理员可操作")
	}
	return s.repo.Delete(ctx, id, tenantID)
}

func (s *Service) GetStats(ctx context.Context, tenantID int) (*ProblemStats, error) {
	return s.repo.GetStats(ctx, tenantID)
}

// GetTrend computes problem trend data for the given period.
func (s *Service) GetTrend(ctx context.Context, tenantID int, startDate, endDate time.Time) (*ProblemTrendData, error) {
	problems, err := s.repo.GetAllForAnalytics(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	total := len(problems)
	resolved := 0
	open := 0
	categoryMap := make(map[string]int)
	priorityMap := make(map[string]int)
	monthlyMap := make(map[string]*MonthlyCount)
	var totalResolutionHours float64
	resolvedCount := 0

	for _, p := range problems {
		if p.Category != "" {
			categoryMap[p.Category]++
		}
		if p.Priority != "" {
			priorityMap[p.Priority]++
		}
		month := p.CreatedAt.Format("2006-01")
		mc, ok := monthlyMap[month]
		if !ok {
			mc = &MonthlyCount{Month: month}
			monthlyMap[month] = mc
		}
		mc.Count++
		switch p.Status {
		case "resolved", "closed":
			resolved++
			mc.Resolved++
			if p.ResolvedAt != nil {
				hours := p.ResolvedAt.Sub(p.CreatedAt).Hours()
				totalResolutionHours += hours
				resolvedCount++
			}
		case "open", "investigating", "identified", "in_progress":
			open++
			mc.Open++
		}
	}

	resolutionRate := 0.0
	if total > 0 {
		resolutionRate = float64(resolved) / float64(total) * 100
	}
	avgResolutionHours := 0.0
	if resolvedCount > 0 {
		avgResolutionHours = totalResolutionHours / float64(resolvedCount)
	}

	// Build top categories
	topCategories := make([]CategoryCount, 0, len(categoryMap))
	for cat, cnt := range categoryMap {
		topCategories = append(topCategories, CategoryCount{Category: cat, Count: cnt})
	}
	for i := 0; i < len(topCategories); i++ {
		for j := i + 1; j < len(topCategories); j++ {
			if topCategories[j].Count > topCategories[i].Count {
				topCategories[i], topCategories[j] = topCategories[j], topCategories[i]
			}
		}
	}
	if len(topCategories) > 5 {
		topCategories = topCategories[:5]
	}

	// Build monthly trend sorted
	monthlyTrend := make([]MonthlyCount, 0, len(monthlyMap))
	for _, mc := range monthlyMap {
		monthlyTrend = append(monthlyTrend, *mc)
	}
	for i := 0; i < len(monthlyTrend); i++ {
		for j := i + 1; j < len(monthlyTrend); j++ {
			if monthlyTrend[j].Month < monthlyTrend[i].Month {
				monthlyTrend[i], monthlyTrend[j] = monthlyTrend[j], monthlyTrend[i]
			}
		}
	}

	trendDirection := "stable"
	if len(monthlyTrend) >= 2 {
		last := monthlyTrend[len(monthlyTrend)-1].Count
		prev := monthlyTrend[len(monthlyTrend)-2].Count
		if last > prev {
			trendDirection = "increasing"
		} else if last < prev {
			trendDirection = "decreasing"
		}
	}

	return &ProblemTrendData{
		Period:                 startDate.Format("2006-01-02") + " ~ " + endDate.Format("2006-01-02"),
		TotalProblems:          total,
		ResolvedProblems:       resolved,
		OpenProblems:           open,
		ResolutionRate:         resolutionRate,
		AvgResolutionTimeHours: avgResolutionHours,
		CategoryBreakdown:      categoryMap,
		PriorityBreakdown:      priorityMap,
		TrendDirection:         trendDirection,
		TopCategories:          topCategories,
		MonthlyTrend:           monthlyTrend,
	}, nil
}

// GetHotspot computes problem hotspot data for the given period.
func (s *Service) GetHotspot(ctx context.Context, tenantID int, startDate, endDate time.Time) (*ProblemHotspotData, error) {
	problems, err := s.repo.GetAllForAnalytics(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	categoryMap := make(map[string]int)
	priorityMap := make(map[string]int)
	for _, p := range problems {
		if p.Category != "" {
			categoryMap[p.Category]++
		}
		if p.Priority != "" {
			priorityMap[p.Priority]++
		}
	}

	hotspots := make([]string, 0)
	avgPerCategory := 0.0
	if len(categoryMap) > 0 {
		totalCount := 0
		for cat, cnt := range categoryMap {
			totalCount += cnt
			if cnt > 1 {
				hotspots = append(hotspots, cat)
			}
		}
		avgPerCategory = float64(totalCount) / float64(len(categoryMap))
	}

	return &ProblemHotspotData{
		PeriodStart:       startDate.Format("2006-01-02"),
		PeriodEnd:         endDate.Format("2006-01-02"),
		CategoryBreakdown: categoryMap,
		PriorityBreakdown: priorityMap,
		Hotspots:          hotspots,
		AvgPerCategory:    avgPerCategory,
	}, nil
}
