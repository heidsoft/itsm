package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/knownerror"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

// RootCauseAnalysisService 根因分析服务
type RootCauseAnalysisService struct {
	client *ent.Client
	// slaSvc 与 incident_service 同构，直接持有共享层的 *TicketSLAService。
	// 本服务就住在 internal/services（共享层），不需要像 handlers/problem 那样
	// 再套一层窄契约——那层窄契约是为了让 handler 域不引用共享层具体类型。
	slaSvc  *TicketSLAService
	gateway *LLMGateway
	logger  *zap.SugaredLogger
}

// NewRootCauseAnalysisService 创建根因分析服务
func NewRootCauseAnalysisService(client *ent.Client) *RootCauseAnalysisService {
	return &RootCauseAnalysisService{client: client}
}

// SetGateway 注入 LLM Gateway，启用后 Perform5WhysAnalysis / AnalyzeWithAI 将使用 AI
func (s *RootCauseAnalysisService) SetGateway(gateway *LLMGateway) {
	s.gateway = gateway
}

// SetSLAService 注入 SLA 服务，供事件转问题时按 problem 类型匹配定义并写 sla_states。
// 用 setter 而非改构造函数签名：构造函数被 bootstrap 与 incident_service_test 共用，
// 加参数会传染既有装配与测试，而 SetGateway/SetLogger 已是本服务的注入范式。
func (s *RootCauseAnalysisService) SetSLAService(slaSvc *TicketSLAService) {
	s.slaSvc = slaSvc
}

// SetLogger 注入 logger
func (s *RootCauseAnalysisService) SetLogger(logger *zap.SugaredLogger) {
	s.logger = logger
}

// RCAContext RCA分析上下文
type RCAContext struct {
	ProblemID       int
	IncidentIDs     []int
	AnalysisType    string // "5_whys", "fishbone", "ai_assisted"
	Findings        []string
	RootCause       string
	Recommendations []string
}

// Perform5WhysAnalysis 执行5 Whys分析
func (s *RootCauseAnalysisService) Perform5WhysAnalysis(ctx context.Context, problemID int, initialQuestion string) (*RCAContext, error) {
	// 创建RCA上下文
	rca := &RCAContext{
		ProblemID:    problemID,
		AnalysisType: "5_whys",
		Findings:     []string{},
	}

	// 获取问题信息
	problem, err := s.client.Problem.Get(ctx, problemID)
	if err != nil {
		return nil, err
	}

	// 记录初始问题
	rca.Findings = append(rca.Findings, fmt.Sprintf("问题: %s", problem.Title))

	// 如果 LLM Gateway 可用，使用 AI 辅助 5-Whys 分析
	if s.gateway != nil {
		aiFindings, rootCause, recommendations, aiErr := s.performAI5Whys(ctx, problem, initialQuestion)
		if aiErr == nil && len(aiFindings) > 0 {
			rca.Findings = append(rca.Findings, aiFindings...)
			rca.RootCause = rootCause
			rca.Recommendations = recommendations
			return rca, nil
		}
		// AI 分析失败，回退到基础逻辑
		if s.logger != nil {
			s.logger.Warnw("AI 5-Whys analysis failed, falling back to basic", "error", aiErr)
		}
	}

	// 基础回退逻辑
	rca.Findings = append(rca.Findings, fmt.Sprintf("初始问题: %s", initialQuestion))
	rca.RootCause = "需要进一步人工分析"
	return rca, nil
}

// performAI5Whys 使用 LLM 执行 5-Whys 分析
func (s *RootCauseAnalysisService) performAI5Whys(ctx context.Context, problem *ent.Problem, initialQuestion string) (findings []string, rootCause string, recommendations []string, err error) {
	prompt := fmt.Sprintf(`你是一个资深的 IT 运维专家，正在对问题执行 5-Whys 根因分析。

问题信息：
- 标题：%s
- 描述：%s
- 优先级：%s
- 分类：%s
- 初始追问：%s

请执行 5-Whys 分析，逐层追问根本原因。使用以下 JSON 格式返回结果（只返回 JSON）：
{
  "whys": [
    "Why 1: ...",
    "Why 2: ...",
    "Why 3: ...",
    "Why 4: ...",
    "Why 5: ..."
  ],
  "root_cause": "最终识别的根本原因",
  "recommendations": ["建议1", "建议2", "建议3"]
}`,
		problem.Title, problem.Description, problem.Priority, problem.Category, initialQuestion)

	messages := []LLMMessage{
		{Role: "system", Content: "你是一个专业的 IT 服务管理根因分析助手，擅长使用 5-Whys 方法逐层分析问题根因。"},
		{Role: "user", Content: prompt},
	}

	response, err := s.gateway.Chat(ctx, "", messages)
	if err != nil {
		return nil, "", nil, fmt.Errorf("LLM 5-Whys analysis failed: %w", err)
	}

	// 提取 JSON 部分
	jsonStr := response
	if idx := strings.Index(jsonStr, "{"); idx >= 0 {
		jsonStr = jsonStr[idx:]
	}
	if idx := strings.LastIndex(jsonStr, "}"); idx >= 0 {
		jsonStr = jsonStr[:idx+1]
	}

	var result struct {
		Whys            []string `json:"whys"`
		RootCause       string   `json:"root_cause"`
		Recommendations []string `json:"recommendations"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, "", nil, fmt.Errorf("failed to parse LLM 5-Whys response: %w", err)
	}

	return result.Whys, result.RootCause, result.Recommendations, nil
}

// AnalyzeProblemFromIncidents 从关联事件分析问题根因
func (s *RootCauseAnalysisService) AnalyzeProblemFromIncidents(ctx context.Context, problemID int) (*RCAContext, error) {
	// 获取问题关联的事件
	problem, err := s.client.Problem.Get(ctx, problemID)
	if err != nil {
		return nil, err
	}

	rca := &RCAContext{
		ProblemID:    problemID,
		AnalysisType: "incident_analysis",
		Findings:     []string{},
	}

	// 获取相关事件（数据库层按租户过滤，避免跨租户数据加载）
	tenantIncidents, err := s.client.Incident.Query().
		Where(incident.TenantIDEQ(problem.TenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// 分析事件的共同特征
	var commonCategories []string

	for _, inc := range tenantIncidents {
		if inc.Category != "" {
			commonCategories = append(commonCategories, inc.Category)
		}
	}

	// 统计最常见的问题分类
	if len(commonCategories) > 0 {
		rca.Findings = append(rca.Findings, fmt.Sprintf("最常见的事件分类: %v", commonCategories))
	}

	// 识别可能的根因
	rca.RootCause = "基于事件分析得出可能的根因"

	return rca, nil
}

// MatchKnownErrors 匹配已知错误库
func (s *RootCauseAnalysisService) MatchKnownErrors(ctx context.Context, tenantID int, keywords []string) ([]*ent.KnownError, error) {
	// 数据库层按租户和状态过滤，避免跨租户数据加载
	all, err := s.client.KnownError.Query().
		Where(knownerror.TenantIDEQ(tenantID), knownerror.StatusEQ("active")).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*ent.KnownError
	for _, ke := range all {
		// Simple keyword match
		for _, kw := range keywords {
			if ke.Keywords != nil {
				for _, keKw := range ke.Keywords {
					if keKw == kw {
						result = append(result, ke)
						break
					}
				}
			}
		}
	}

	// Limit to 10 results
	if len(result) > 10 {
		result = result[:10]
	}

	return result, nil
}

// CreateProblemFromIncident 从事件创建问题
func (s *RootCauseAnalysisService) CreateProblemFromIncident(
	ctx context.Context,
	incidentID, createdBy, tenantID int,
	req *dto.ConvertIncidentToProblemRequest,
) (*ent.Problem, error) {
	creatorExists, err := s.client.User.Query().
		Where(user.IDEQ(createdBy), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate problem creator: %w", err)
	}
	if !creatorExists {
		return nil, fmt.Errorf("problem creator not found or inactive")
	}
	incidentEntity, err := s.client.Incident.Query().
		Where(incident.IDEQ(incidentID), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found")
		}
		return nil, err
	}

	title := fmt.Sprintf("问题-%s", incidentEntity.Title)
	description := incidentEntity.Description
	rootCause := ""
	if req != nil {
		if strings.TrimSpace(req.Title) != "" {
			title = strings.TrimSpace(req.Title)
		}
		if strings.TrimSpace(req.Description) != "" {
			description = strings.TrimSpace(req.Description)
		}
		rootCause = strings.TrimSpace(req.RootCause)
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start conversion transaction: %w", err)
	}
	rollback := func(cause error) (*ent.Problem, error) {
		_ = tx.Rollback()
		return nil, cause
	}
	newProblem, err := tx.Problem.Create().
		SetTitle(title).
		SetDescription(description).
		SetPriority(incidentEntity.Priority).
		SetCategory(incidentEntity.Category).
		SetRootCause(rootCause).
		SetStatus("open").
		SetCreatedBy(createdBy).
		SetTenantID(tenantID).
		AddIncidentIDs(incidentID).
		Save(ctx)
	if err != nil {
		return rollback(err)
	}

	_, err = tx.IncidentEvent.Create().
		SetIncidentID(incidentID).
		SetEventType("problem_linked").
		SetEventName("关联问题").
		SetDescription(fmt.Sprintf("已关联到问题 #%d", newProblem.ID)).
		SetStatus("active").
		SetSeverity("info").
		SetSource("system").
		SetUserID(createdBy).
		SetOccurredAt(time.Now()).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return rollback(err)
	}
	if err := tx.Commit(); err != nil {
		return rollback(err)
	}

	// SLA 绑定刻意放在事务提交之后，不放进事务内：
	// sla.Store.SaveDeadlines 用的是它自己的 s.client 句柄（不是 tx 句柄），
	// 放进事务里会绕开事务——表面上「有事务保护」，实则写库完全不受该事务
	// 回滚约束，SQLite 下还可能因两个句柄抢同一张表而锁住。
	// 与 handlers/problem 的既有顺序一致：repo.Create 之后才 bindSLAOnCreate。
	//
	// 问题已提交，绑定失败不能伪装成转换失败；另写审计告警区分未配置与依赖故障。
	s.bindSLAOnProblem(ctx, tenantID, newProblem.ID, createdBy, newProblem.Priority)

	return newProblem, nil
}

// bindSLAOnProblem 转换出的问题按 problem 服务类型 + 优先级匹配 SLA 定义，
// 计算响应/解决截止时间并写入 sla_states（aggregate_type="problem"）。
//
// 与 handlers/problem.bindSLAOnCreate 同语义（两者必须保持一致，否则「问题管理页
// 新建」与「事件转问题」会给出两套事实），其中最关键的一条是：
// 问题域刻意不套用默认 SLA。TicketSLAService.CalculateSLADeadlineFromRequest 在
// 没有匹配定义时会返回 SLADefinitionID=0 却仍然塞上 60/480 分钟的默认截止时间；
// 若这里只看 slaResult 非空就绑定，等于给存量问题静默绑隐藏默认值，下一轮 watcher
// 会让它们集体违规——那是配置噪音不是治理信号。故必须显式判 SLADefinitionID == 0
// 即视为「没配 SLA」直接返回，由 GET /problems/:id/sla 的 bound=false 如实回答。
func (s *RootCauseAnalysisService) bindSLAOnProblem(ctx context.Context, tenantID, problemID, actorID int, priority string) {
	if s.slaSvc == nil {
		return
	}
	slaResult, err := s.slaSvc.CalculateSLADeadlineFromRequest(ctx, tenantID, "problem", priority)
	if err != nil {
		s.recordSLABindFailure(ctx, tenantID, problemID, actorID, "deadline_query_failed")
		return
	}
	if slaResult == nil || slaResult.SLADefinitionID == 0 {
		// 共享计算器会把查询故障也降级成 ID=0；核验查询是否可用，避免误称“未配置”。
		// 不重新匹配策略，正常无配置仍不绑定、不记失败。
		if _, err := s.slaSvc.client.SLADefinition.Query().Where(
			sladefinition.TenantID(tenantID), sladefinition.ServiceType("problem"),
			sladefinition.IsActive(true),
		).Exist(ctx); err != nil {
			s.recordSLABindFailure(ctx, tenantID, problemID, actorID, "deadline_query_failed")
		}
		return
	}
	store := s.slaSvc.SLAStore()
	if store == nil {
		s.recordSLABindFailure(ctx, tenantID, problemID, actorID, "store_unavailable")
		return
	}
	if err := store.SaveDeadlines(ctx, tenantID, "problem", problemID, slaResult.SLADefinitionID, slaResult.ResponseDeadline, slaResult.ResolutionDeadline); err != nil {
		s.recordSLABindFailure(ctx, tenantID, problemID, actorID, "state_write_failed")
	}
}

// 复用结构化审计留下可检索的故障记录；只存稳定分类，避免泄漏数据库原始错误。
func (s *RootCauseAnalysisService) recordSLABindFailure(ctx context.Context, tenantID, problemID, actorID int, errorClass string) {
	details, _ := json.Marshal(map[string]interface{}{"slaBindFailed": true, "errorClass": errorClass, "problemId": problemID})
	logger := s.logger
	if logger == nil {
		logger = zap.S()
	}
	logger.Errorw("problem SLA binding failed", "tenant_id", tenantID, "problem_id", problemID, "actor_id", actorID, "error_class", errorClass, "slaBindFailed", true)
	_, err := s.client.AuditLog.Create().SetTenantID(tenantID).SetUserID(actorID).
		SetResource(fmt.Sprintf("problem:%d", problemID)).SetAction("problem.sla_bind_failed").
		SetPath(fmt.Sprintf("service://problems/%d/sla", problemID)).SetMethod("DOMAIN").
		SetStatusCode(503).SetRequestBody(string(details)).Save(ctx)
	if err != nil {
		// 请求取消或数据库同时不可用时审计也可能失败，必须保留独立告警信号。
		logger.Errorw("problem SLA binding audit unavailable", "tenant_id", tenantID, "problem_id", problemID, "error_class", "audit_write_failed")
	}
}

// warnw 在 logger 未注入时静默降级：本服务在生产由 bootstrap 注入 logger，
// 但多处单测只 NewRootCauseAnalysisService(client) 就直接调业务方法，
// 直呼 s.logger.Warnw 会在 nil 上解引用 panic，把「没有 SLA」变成测试崩溃。
func (s *RootCauseAnalysisService) warnw(msg string, kv ...interface{}) {
	if s.logger == nil {
		return
	}
	s.logger.Warnw(msg, kv...)
}

// AnalyzeWithAI 使用AI辅助分析根因
func (s *RootCauseAnalysisService) AnalyzeWithAI(ctx context.Context, problemID int) (*RCAContext, error) {
	// 获取问题信息
	problem, err := s.client.Problem.Get(ctx, problemID)
	if err != nil {
		return nil, err
	}

	rca := &RCAContext{
		ProblemID:    problemID,
		AnalysisType: "ai_assisted",
		Findings:     []string{},
	}

	// 如果 LLM Gateway 不可用，返回提示信息
	if s.gateway == nil {
		rca.Findings = append(rca.Findings, "AI 服务未配置，无法执行 AI 辅助分析")
		rca.Recommendations = append(rca.Recommendations, "请配置 LLM Gateway 后重试")
		return rca, nil
	}

	// 获取问题关联的事件用于上下文
	incidents, err := s.client.Incident.Query().
		Where(incident.TenantIDEQ(problem.TenantID)).
		Limit(20).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incidents: %w", err)
	}

	// 构建 AI 分析 prompt
	incidentSummary := ""
	for i, inc := range incidents {
		if i >= 5 {
			incidentSummary += fmt.Sprintf("\n... 及其他 %d 个事件", len(incidents)-5)
			break
		}
		incidentSummary += fmt.Sprintf("\n- 事件 #%d: %s (优先级: %s, 分类: %s)", inc.ID, inc.Title, inc.Priority, inc.Category)
	}

	prompt := fmt.Sprintf(`你是一个资深的 IT 运维专家，负责分析问题的根本原因。

问题信息：
- 问题ID: %d
- 标题: %s
- 描述: %s
- 优先级: %s
- 分类: %s
- 当前根因: %s

关联事件：%s

请分析这个问题的根本原因，并给出建议。使用以下 JSON 格式返回结果（只返回 JSON）：
{
  "findings": ["发现1", "发现2", "发现3"],
  "root_cause": "根本原因分析",
  "recommendations": ["建议1", "建议2", "建议3"]
}`,
		problem.ID, problem.Title, problem.Description, problem.Priority, problem.Category,
		problem.RootCause, incidentSummary)

	messages := []LLMMessage{
		{Role: "system", Content: "你是一个专业的 IT 服务管理根因分析助手，擅长从多个事件中识别共同模式并推断根本原因。"},
		{Role: "user", Content: prompt},
	}

	response, err := s.gateway.Chat(ctx, "", messages)
	if err != nil {
		if s.logger != nil {
			s.logger.Errorw("AI root cause analysis failed", "error", err, "problem_id", problemID)
		}
		rca.Findings = append(rca.Findings, fmt.Sprintf("AI 分析失败: %v", err))
		return rca, nil
	}

	// 解析 JSON 响应
	jsonStr := response
	if idx := strings.Index(jsonStr, "{"); idx >= 0 {
		jsonStr = jsonStr[idx:]
	}
	if idx := strings.LastIndex(jsonStr, "}"); idx >= 0 {
		jsonStr = jsonStr[:idx+1]
	}

	var result struct {
		Findings        []string `json:"findings"`
		RootCause       string   `json:"root_cause"`
		Recommendations []string `json:"recommendations"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		if s.logger != nil {
			s.logger.Warnw("Failed to parse AI response, using raw text", "error", err)
		}
		rca.Findings = append(rca.Findings, response)
		rca.RootCause = "AI 分析完成（解析失败，见 findings 原文）"
	} else {
		rca.Findings = result.Findings
		rca.RootCause = result.RootCause
		rca.Recommendations = result.Recommendations
	}

	// 更新问题的根因
	_, err = s.client.Problem.UpdateOneID(problemID).
		SetRootCause(rca.RootCause).
		Save(ctx)

	return rca, err
}

// GetProblemAnalysis 获取问题的根因分析
func (s *RootCauseAnalysisService) GetProblemAnalysis(ctx context.Context, problemID int) (*RCAContext, error) {
	problem, err := s.client.Problem.Get(ctx, problemID)
	if err != nil {
		return nil, err
	}

	rca := &RCAContext{
		ProblemID: problemID,
		RootCause: problem.RootCause,
	}

	// 获取关联的事件
	incidents, err := s.client.Incident.Query().
		Limit(100).
		All(ctx)
	if err != nil {
		return nil, err
	}

	rca.IncidentIDs = make([]int, 0)
	for _, inc := range incidents {
		if inc.TenantID == problem.TenantID {
			rca.IncidentIDs = append(rca.IncidentIDs, inc.ID)
		}
	}

	return rca, nil
}

// LinkProblemIncident 关联问题与事件
func (s *RootCauseAnalysisService) LinkProblemIncident(ctx context.Context, problemID, incidentID int) error {
	// 验证事件存在
	_, err := s.client.Incident.Get(ctx, incidentID)
	if err != nil {
		return err
	}

	// 记录事件关联
	_, err = s.client.IncidentEvent.Create().
		SetIncidentID(incidentID).
		SetEventType("problem_linked").
		SetDescription(fmt.Sprintf("已关联到问题 #%d", problemID)).
		SetCreatedAt(time.Now()).
		Save(ctx)

	return err
}

// UnlinkProblemIncident 解除问题与事件关联
func (s *RootCauseAnalysisService) UnlinkProblemIncident(ctx context.Context, problemID, incidentID int) error {
	incident, err := s.client.Incident.Get(ctx, incidentID)
	if err != nil {
		return err
	}
	_ = incident // use the variable

	// 记录解除关联
	_, err = s.client.IncidentEvent.Create().
		SetIncidentID(incidentID).
		SetEventType("problem_unlinked").
		SetDescription(fmt.Sprintf("已解除与问题 #%d 的关联", problemID)).
		SetCreatedAt(time.Now()).
		Save(ctx)

	return err
}

// ResolveProblemWithSolution 使用解决方案解决问题
func (s *RootCauseAnalysisService) ResolveProblemWithSolution(ctx context.Context, problemID int, resolution string) error {
	// 获取问题信息用于日志
	problem, err := s.client.Problem.Get(ctx, problemID)
	if err != nil {
		return err
	}
	_ = problem // 问题存在性已验证

	_, err = s.client.Problem.UpdateOneID(problemID).
		SetStatus("resolved").
		SetRootCause(resolution).
		Save(ctx)

	return err
}
