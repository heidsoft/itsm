package ai

import (
	"context"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/conversation"
	"itsm-backend/ent/message"
	"itsm-backend/ent/rootcauseanalysis"
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
		ID:             e.ID,
		ConversationID: e.ConversationID,
		ToolName:       e.ToolName,
		Arguments:      e.Arguments,
		Status:         e.Status,
		Result:         e.Result,
		Error:          e.Error,
		NeedsApproval:  e.NeedsApproval,
		ApprovalState:  e.ApprovalState,
		ApprovedBy:     e.ApprovedBy,
		ApprovalReason: e.ApprovalReason,
		ApprovedAt:     approvedAt,
		RequestID:      e.RequestID,
		CreatedAt:      e.CreatedAt,
	}
}

func (r *EntRepository) CreateToolInvocation(ctx context.Context, i *ToolInvocation) (*ToolInvocation, error) {
	e, err := r.client.ToolInvocation.Create().
		SetConversationID(i.ConversationID).
		SetToolName(i.ToolName).
		SetArguments(i.Arguments).
		SetStatus(i.Status).
		SetNeedsApproval(i.NeedsApproval).
		SetApprovalState(i.ApprovalState).
		SetRequestID(i.RequestID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toToolInvocationDomain(e), nil
}

func (r *EntRepository) GetToolInvocation(ctx context.Context, id int) (*ToolInvocation, error) {
	e, err := r.client.ToolInvocation.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return toToolInvocationDomain(e), nil
}

func (r *EntRepository) UpdateToolInvocation(ctx context.Context, i *ToolInvocation) (*ToolInvocation, error) {
	update := r.client.ToolInvocation.UpdateOneID(i.ID).
		SetStatus(i.Status).
		SetApprovalState(i.ApprovalState).
		SetApprovalReason(i.ApprovalReason).
		SetApprovedBy(i.ApprovedBy)

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
