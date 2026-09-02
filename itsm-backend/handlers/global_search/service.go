package global_search

import (
	"context"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/ticket"
)

const perTypeLimit = 10

// Service owns the tenant-scoped global-search use case.
type Service struct {
	client *ent.Client
}

func NewService(client *ent.Client) *Service { return &Service{client: client} }

func (s *Service) Search(ctx context.Context, tenantID int, keyword string) (*SearchResponse, error) {
	results := make([]*SearchResult, 0)

	tickets, err := s.client.Ticket.Query().Where(ticket.TenantID(tenantID), ticket.Or(
		ticket.TitleContainsFold(keyword), ticket.DescriptionContainsFold(keyword), ticket.TicketNumberContainsFold(keyword),
	)).Limit(perTypeLimit).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range tickets {
		results = append(results, &SearchResult{ID: item.ID, Type: "ticket", Title: item.Title, Description: item.Description, Status: item.Status, TicketNumber: item.TicketNumber})
	}

	incidents, err := s.client.Incident.Query().Where(incident.TenantID(tenantID), incident.Or(
		incident.TitleContainsFold(keyword), incident.DescriptionContainsFold(keyword), incident.IncidentNumberContainsFold(keyword),
	)).Limit(perTypeLimit).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range incidents {
		results = append(results, &SearchResult{ID: item.ID, Type: "incident", Title: item.Title, Description: item.Description, Status: item.Status, TicketNumber: item.IncidentNumber})
	}

	problems, err := s.client.Problem.Query().Where(problem.TenantID(tenantID), problem.Or(
		problem.TitleContainsFold(keyword), problem.DescriptionContainsFold(keyword),
	)).Limit(perTypeLimit).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range problems {
		results = append(results, &SearchResult{ID: item.ID, Type: "problem", Title: item.Title, Description: item.Description, Status: item.Status})
	}

	changes, err := s.client.Change.Query().Where(change.TenantID(tenantID), change.Or(
		change.TitleContainsFold(keyword), change.DescriptionContainsFold(keyword),
	)).Limit(perTypeLimit).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range changes {
		results = append(results, &SearchResult{ID: item.ID, Type: "change", Title: item.Title, Description: item.Description, Status: item.Status})
	}

	articles, err := s.client.KnowledgeArticle.Query().Where(knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil(), knowledgearticle.Or(
		knowledgearticle.TitleContainsFold(keyword), knowledgearticle.ContentContainsFold(keyword),
	)).Limit(perTypeLimit).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range articles {
		status := "draft"
		if item.IsPublished {
			status = "published"
		}
		results = append(results, &SearchResult{ID: item.ID, Type: "knowledge", Title: item.Title, Description: item.Content, Status: status})
	}

	return &SearchResponse{Results: results, Total: len(results)}, nil
}
