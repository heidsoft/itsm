package ai

import (
	"context"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/aianalysisresult"
	"itsm-backend/ent/conversation"
	"itsm-backend/ent/message"
	"itsm-backend/ent/rootcauseanalysis"
	"itsm-backend/ent/toolinvocation"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

// Conversations

func toConversationDomain(e *ent.Conversation) *Conversation {
	if e == nil {
		return nil
	}
	return &Conversation{
		ID:        e.ID,
		Title:     e.Title,
		UserID:    e.UserID,
		TenantID:  e.TenantID,
		CreatedAt: e.CreatedAt,
	}
}

func (r *EntRepository) CreateConversation(ctx context.Context, c *Conversation) (*Conversation, error) {
	e, err := r.client.Conversation.Create().
		SetTitle(c.Title).
		SetUserID(c.UserID).
		SetTenantID(c.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toConversationDomain(e), nil
}

func (r *EntRepository) GetConversation(ctx context.Context, id int, tenantID int) (*Conversation, error) {
	e, err := r.client.Conversation.Query().
		Where(conversation.ID(id), conversation.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toConversationDomain(e), nil
}

func (r *EntRepository) ListConversations(ctx context.Context, tenantID int, userID int) ([]*Conversation, error) {
	es, err := r.client.Conversation.Query().
		Where(conversation.TenantID(tenantID), conversation.UserID(userID)).
		Order(ent.Desc(conversation.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	var res []*Conversation
	for _, e := range es {
		res = append(res, toConversationDomain(e))
	}
	return res, nil
}

func (r *EntRepository) DeleteConversation(ctx context.Context, id int, tenantID int) error {
	return r.client.Conversation.DeleteOneID(id).
		Where(conversation.TenantID(tenantID)).
		Exec(ctx)
}

// Messages

func toMessageDomain(e *ent.Message) *Message {
	if e == nil {
		return nil
	}
	return &Message{
		ID:             e.ID,
		ConversationID: e.ConversationID,
		Role:           e.Role,
		Content:        e.Content,
		RequestID:      e.RequestID,
		CreatedAt:      e.CreatedAt,
	}
}

func (r *EntRepository) CreateMessage(ctx context.Context, m *Message) (*Message, error) {
	e, err := r.client.Message.Create().
		SetConversationID(m.ConversationID).
		SetRole(m.Role).
		SetContent(m.Content).
		SetRequestID(m.RequestID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toMessageDomain(e), nil
}

func (r *EntRepository) GetMessages(ctx context.Context, conversationID int) ([]*Message, error) {
	es, err := r.client.Message.Query().
		Where(message.ConversationID(conversationID)).
		Order(ent.Asc(message.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	var res []*Message
	for _, e := range es {
		res = append(res, toMessageDomain(e))
	}
	return res, nil
}

// Tool Invocations

func toToolInvocationDomain(e *ent.ToolInvocation) *ToolInvocation {
	if e == nil {
		return nil
	}
	var approvedAt *time.Time
	if !e.ApprovedAt.IsZero() {
		t := e.ApprovedAt
		approvedAt = &t
	}
	return &ToolInvocation{
		ID:               e.ID,
		TenantID:         e.TenantID,
		ConversationID:   e.ConversationID,
		ToolName:         e.ToolName,
		Arguments:        e.Arguments,
		Status:           e.Status,
		Result:           e.Result,
		Error:            e.Error,
		NeedsApproval:    e.NeedsApproval,
		ApprovalState:    e.ApprovalState,
		ApprovedBy:       e.ApprovedBy,
		ApprovalReason:   e.ApprovalReason,
		ApprovedAt:       approvedAt,
		RequestID:        e.RequestID,
		CreatedAt:        e.CreatedAt,
		UserID:           e.UserID,
		PermissionCheck:  e.PermissionCheck,
		PermissionReason: e.PermissionReason,
		RoleSnapshot:     e.RoleSnapshot,
	}
}

func (r *EntRepository) CreateToolInvocation(ctx context.Context, i *ToolInvocation) (*ToolInvocation, error) {
	e, err := r.client.ToolInvocation.Create().
		SetTenantID(i.TenantID).
		SetToolName(i.ToolName).
		SetArguments(i.Arguments).
		SetStatus(i.Status).
		SetNeedsApproval(i.NeedsApproval).
		SetApprovalState(i.ApprovalState).
		SetRequestID(i.RequestID).
		SetUserID(i.UserID).
		SetPermissionCheck(i.PermissionCheck).
		SetPermissionReason(i.PermissionReason).
		SetRoleSnapshot(i.RoleSnapshot).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toToolInvocationDomain(e), nil
}

func (r *EntRepository) GetToolInvocation(ctx context.Context, id int, tenantID int) (*ToolInvocation, error) {
	e, err := r.client.ToolInvocation.Query().
		Where(toolinvocation.ID(id), toolinvocation.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toToolInvocationDomain(e), nil
}

// ListToolInvocations 按租户 + 审批状态列出工具调用记录，按创建时间倒序。
// state 为空时返回该租户全部调用；传 "pending" 即审批人待办。
func (r *EntRepository) ListToolInvocations(ctx context.Context, tenantID int, state string) ([]*ToolInvocation, error) {
	q := r.client.ToolInvocation.Query().
		Where(toolinvocation.TenantID(tenantID))
	if state != "" {
		q = q.Where(toolinvocation.ApprovalStateEQ(state))
	}
	invocations, err := q.
		Order(ent.Desc(toolinvocation.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*ToolInvocation, 0, len(invocations))
	for _, e := range invocations {
		out = append(out, toToolInvocationDomain(e))
	}
	return out, nil
}

func (r *EntRepository) UpdateToolInvocation(ctx context.Context, i *ToolInvocation) (*ToolInvocation, error) {
	update := r.client.ToolInvocation.UpdateOneID(i.ID).
		SetStatus(i.Status).
		SetApprovalState(i.ApprovalState).
		SetApprovalReason(i.ApprovalReason).
		SetApprovedBy(i.ApprovedBy).
		SetPermissionCheck(i.PermissionCheck).
		SetPermissionReason(i.PermissionReason)

	if i.Result != nil {
		update.SetResult(*i.Result)
	}
	if i.Error != nil {
		update.SetError(*i.Error)
	}
	if i.ApprovedAt != nil {
		update.SetApprovedAt(*i.ApprovedAt)
	}

	e, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toToolInvocationDomain(e), nil
}

// Root Cause Analysis

func toRCADomain(e *ent.RootCauseAnalysis) *RootCauseAnalysis {
	if e == nil {
		return nil
	}
	return &RootCauseAnalysis{
		ID:              e.ID,
		TicketID:        e.TicketID,
		TicketNumber:    e.TicketNumber,
		TicketTitle:     e.TicketTitle,
		AnalysisDate:    e.AnalysisDate,
		RootCauses:      e.RootCauses,
		AnalysisSummary: e.AnalysisSummary,
		ConfidenceScore: e.ConfidenceScore,
		AnalysisMethod:  e.AnalysisMethod,
		TenantID:        e.TenantID,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func (r *EntRepository) CreateRCA(ctx context.Context, rca *RootCauseAnalysis) (*RootCauseAnalysis, error) {
	e, err := r.client.RootCauseAnalysis.Create().
		SetTicketID(rca.TicketID).
		SetTicketNumber(rca.TicketNumber).
		SetTicketTitle(rca.TicketTitle).
		SetAnalysisDate(rca.AnalysisDate).
		SetRootCauses(rca.RootCauses).
		SetAnalysisSummary(rca.AnalysisSummary).
		SetConfidenceScore(rca.ConfidenceScore).
		SetAnalysisMethod(rca.AnalysisMethod).
		SetTenantID(rca.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toRCADomain(e), nil
}

func (r *EntRepository) GetRCAByTicket(ctx context.Context, ticketID int, tenantID int) (*RootCauseAnalysis, error) {
	e, err := r.client.RootCauseAnalysis.Query().
		Where(rootcauseanalysis.TicketID(ticketID), rootcauseanalysis.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toRCADomain(e), nil
}

func (r *EntRepository) UpdateRCA(ctx context.Context, rca *RootCauseAnalysis) (*RootCauseAnalysis, error) {
	e, err := r.client.RootCauseAnalysis.UpdateOneID(rca.ID).
		SetRootCauses(rca.RootCauses).
		SetAnalysisSummary(rca.AnalysisSummary).
		SetConfidenceScore(rca.ConfidenceScore).
		SetAnalysisMethod(rca.AnalysisMethod).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toRCADomain(e), nil
}

// AI Analysis Results

func toAIAnalysisResultDomain(e *ent.AIAnalysisResult) *AIAnalysisResult {
	return &AIAnalysisResult{
		ID:              e.ID,
		TenantID:        e.TenantID,
		UserID:          e.UserID,
		AnalysisType:    e.AnalysisType,
		TicketID:        e.TicketID,
		IncidentID:      e.IncidentID,
		TicketNumber:    e.TicketNumber,
		TicketTitle:     e.TicketTitle,
		RequestPrompt:   e.RequestPrompt,
		ResultJSON:      e.ResultJSON,
		Model:           e.Model,
		LatencyMs:       e.LatencyMs,
		TotalTokens:     e.TotalTokens,
		CostUSD:         e.CostUsd,
		ConfidenceScore: e.ConfidenceScore,
		Degraded:        e.Degraded,
		CreatedAt:       e.CreatedAt,
	}
}

func (r *EntRepository) SaveAIAnalysisResult(ctx context.Context, a *AIAnalysisResult) (*AIAnalysisResult, error) {
	create := r.client.AIAnalysisResult.Create().
		SetTenantID(a.TenantID).
		SetUserID(a.UserID).
		SetAnalysisType(a.AnalysisType).
		SetRequestPrompt(a.RequestPrompt).
		SetResultJSON(a.ResultJSON).
		SetModel(a.Model).
		SetDegraded(a.Degraded)
	if a.TicketID > 0 {
		create.SetTicketID(a.TicketID)
	}
	if a.IncidentID > 0 {
		create.SetIncidentID(a.IncidentID)
	}
	if a.TicketNumber != "" {
		create.SetTicketNumber(a.TicketNumber)
	}
	if a.TicketTitle != "" {
		create.SetTicketTitle(a.TicketTitle)
	}
	if a.LatencyMs > 0 {
		create.SetLatencyMs(a.LatencyMs)
	}
	if a.TotalTokens > 0 {
		create.SetTotalTokens(a.TotalTokens)
	}
	if a.CostUSD > 0 {
		create.SetCostUsd(a.CostUSD)
	}
	if a.ConfidenceScore > 0 {
		create.SetConfidenceScore(a.ConfidenceScore)
	}
	e, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toAIAnalysisResultDomain(e), nil
}

func (r *EntRepository) ListAIAnalysisResults(ctx context.Context, tenantID int, analysisType string, limit int) ([]*AIAnalysisResult, error) {
	q := r.client.AIAnalysisResult.Query().
		Where(aianalysisresult.TenantID(tenantID))
	if analysisType != "" {
		q = q.Where(aianalysisresult.AnalysisType(analysisType))
	}
	if limit <= 0 {
		limit = 20
	}
	es, err := q.Order(ent.Desc(aianalysisresult.FieldCreatedAt)).Limit(limit).All(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]*AIAnalysisResult, 0, len(es))
	for _, e := range es {
		res = append(res, toAIAnalysisResultDomain(e))
	}
	return res, nil
}

func (r *EntRepository) GetAIAnalysisResult(ctx context.Context, id int, tenantID int) (*AIAnalysisResult, error) {
	e, err := r.client.AIAnalysisResult.Query().
		Where(aianalysisresult.ID(id), aianalysisresult.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toAIAnalysisResultDomain(e), nil
}

func (r *EntRepository) DeleteAIAnalysisResult(ctx context.Context, id int, tenantID int) error {
	return r.client.AIAnalysisResult.DeleteOneID(id).
		Where(aianalysisresult.TenantID(tenantID)).
		Exec(ctx)
}
