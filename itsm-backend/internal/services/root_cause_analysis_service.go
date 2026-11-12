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
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

// RootCauseAnalysisService 根因分析服务
type RootCauseAnalysisService struct {
	client  *ent.Client
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

	return newProblem, nil
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
