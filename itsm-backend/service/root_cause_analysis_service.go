package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
)

// RootCauseAnalysisService 根因分析服务
type RootCauseAnalysisService struct {
	client *ent.Client
}

// NewRootCauseAnalysisService 创建根因分析服务
func NewRootCauseAnalysisService(client *ent.Client) *RootCauseAnalysisService {
	return &RootCauseAnalysisService{client: client}
}

// RCAContext RCA分析上下文
type RCAContext struct {
	ProblemID    int
	IncidentIDs  []int
	AnalysisType string // "5_whys", "fishbone", "ai_assisted"
	Findings     []string
	RootCause    string
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

	// 5 Whys 分析逻辑
	// 这里可以集成AI来辅助分析
	// TODO: 集成AI分析服务

	return rca, nil
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

	// 获取相关事件 - filter manually
	allIncidents, err := s.client.Incident.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	// 分析事件的共同特征
	var commonCategories []string

	for _, incident := range allIncidents {
		if incident.TenantID != problem.TenantID {
			continue
		}
		if incident.Category != "" {
			commonCategories = append(commonCategories, incident.Category)
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
	// Get all known errors and filter manually
	all, err := s.client.KnownError.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*ent.KnownError
	for _, ke := range all {
		if ke.TenantID != tenantID || ke.Status != "active" {
			continue
		}
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
func (s *RootCauseAnalysisService) CreateProblemFromIncident(ctx context.Context, incidentID int, createdBy int) (*ent.Problem, error) {
	incident, err := s.client.Incident.Get(ctx, incidentID)
	if err != nil {
		return nil, err
	}

	// 创建问题
	newProblem, err := s.client.Problem.Create().
		SetTitle(fmt.Sprintf("问题-%s", incident.Title)).
		SetDescription(incident.Description).
		SetPriority(incident.Priority).
		SetCategory(incident.Category).
		SetStatus("open").
		SetCreatedBy(createdBy).
		SetTenantID(incident.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// 记录关联
	_, err = s.client.IncidentEvent.Create().
		SetIncidentID(incidentID).
		SetEventType("problem_linked").
		SetDescription(fmt.Sprintf("已关联到问题 #%d", newProblem.ID)).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return newProblem, nil
}

// AnalyzeWithAI 使用AI辅助分析根因
func (s *RootCauseAnalysisService) AnalyzeWithAI(ctx context.Context, problemID int) (*RCAContext, error) {
	// 获取问题信息用于日志
	_, err := s.client.Problem.Get(ctx, problemID)
	if err != nil {
		return nil, err
	}

	rca := &RCAContext{
		ProblemID:    problemID,
		AnalysisType: "ai_assisted",
	}

	// TODO: 集成AI分析服务
	// 可以调用 llm_gateway 或者 Python AI Service
	// 这里返回模拟结果
	rca.Findings = append(rca.Findings, "AI分析完成")
	rca.Recommendations = append(rca.Recommendations, "建议检查相关服务的日志")

	// 更新问题的根因
	_, err = s.client.Problem.UpdateOneID(problemID).
		SetRootCause("AI分析中 - 待完善").
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
