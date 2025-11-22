package dto

import "itsm-backend/ent"

// ===================================
// User Mappers
// ===================================

// ToUserDetailResponse converts an ent.User to UserDetailResponse
func ToUserDetailResponse(user *ent.User) *UserDetailResponse {
	if user == nil {
		return nil
	}
	return &UserDetailResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Name:       user.Name,
		Department: user.Department,
		Phone:      user.Phone,
		Active:     user.Active,
		TenantID:   user.TenantID,
		Role:       string(user.Role),
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}
}

// ToUserDetailResponseList converts a slice of ent.User to UserDetailResponse slice
func ToUserDetailResponseList(users []*ent.User) []*UserDetailResponse {
	if users == nil {
		return nil
	}
	responses := make([]*UserDetailResponse, 0, len(users))
	for _, user := range users {
		if user != nil {
			responses = append(responses, ToUserDetailResponse(user))
		}
	}
	return responses
}

// ===================================
// Ticket Mappers
// ===================================

// ToTicketResponse converts an ent.Ticket to TicketResponse
func ToTicketResponse(ticket *ent.Ticket) *TicketResponse {
	if ticket == nil {
		return nil
	}

	response := &TicketResponse{
		ID:           ticket.ID,
		Title:        ticket.Title,
		Description:  ticket.Description,
		Status:       ticket.Status,
		Priority:     ticket.Priority,
		TicketNumber: ticket.TicketNumber,
		RequesterID:  ticket.RequesterID,
		AssigneeID:   ticket.AssigneeID,
		TenantID:     ticket.TenantID,
		CategoryID:   ticket.CategoryID,
		CreatedAt:    ticket.CreatedAt,
		UpdatedAt:    ticket.UpdatedAt,
	}

	return response
}

// ToTicketResponseList converts a slice of ent.Ticket to TicketResponse slice
func ToTicketResponseList(tickets []*ent.Ticket) []*TicketResponse {
	if tickets == nil {
		return nil
	}
	responses := make([]*TicketResponse, 0, len(tickets))
	for _, ticket := range tickets {
		if ticket != nil {
			responses = append(responses, ToTicketResponse(ticket))
		}
	}
	return responses
}

// ===================================
// Incident Mappers
// ===================================

// ToIncidentResponse converts an ent.Incident to IncidentResponse
func ToIncidentResponse(incident *ent.Incident) *IncidentResponse {
	if incident == nil {
		return nil
	}

	response := &IncidentResponse{
		ID:              incident.ID,
		Title:           incident.Title,
		Description:     incident.Description,
		Status:          incident.Status,
		Priority:        incident.Priority,
		Severity:        incident.Severity,
		IncidentNumber:  incident.IncidentNumber,
		ReporterID:      incident.ReporterID,
		Category:        incident.Category,
		Subcategory:     incident.Subcategory,
		ImpactAnalysis:  incident.ImpactAnalysis,
		RootCause:       incident.RootCause,
		ResolutionSteps: []map[string]interface{}{}, // Initialize empty slice
		TenantID:        incident.TenantID,
		CreatedAt:       incident.CreatedAt,
		UpdatedAt:       incident.UpdatedAt,
	}

	// Add optional fields if present
	if incident.AssigneeID > 0 {
		response.AssigneeID = &incident.AssigneeID
	}

	if incident.ConfigurationItemID > 0 {
		response.ConfigurationItemID = &incident.ConfigurationItemID
	}

	// Handle time fields - convert to pointer if not zero
	if !incident.ResolvedAt.IsZero() {
		response.ResolvedAt = &incident.ResolvedAt
	}

	if !incident.ClosedAt.IsZero() {
		response.ClosedAt = &incident.ClosedAt
	}

	return response
}

// ToIncidentResponseList converts a slice of ent.Incident to IncidentResponse slice
func ToIncidentResponseList(incidents []*ent.Incident) []*IncidentResponse {
	if incidents == nil {
		return nil
	}
	responses := make([]*IncidentResponse, 0, len(incidents))
	for _, incident := range incidents {
		if incident != nil {
			responses = append(responses, ToIncidentResponse(incident))
		}
	}
	return responses
}

// ===================================
// SLA Mappers
// ===================================

// ToSLADefinitionResponse converts an ent.SLADefinition to SLADefinitionResponse
func ToSLADefinitionResponse(sla *ent.SLADefinition) *SLADefinitionResponse {
	if sla == nil {
		return nil
	}

	return &SLADefinitionResponse{
		ID:              sla.ID,
		Name:            sla.Name,
		Description:     sla.Description,
		ServiceType:     sla.ServiceType,
		Priority:        sla.Priority,
		ResponseTime:    sla.ResponseTime,
		ResolutionTime:  sla.ResolutionTime,
		BusinessHours:   sla.BusinessHours,
		EscalationRules: sla.EscalationRules,
		Conditions:      sla.Conditions,
		IsActive:        sla.IsActive,
		TenantID:        sla.TenantID,
		CreatedAt:       sla.CreatedAt,
		UpdatedAt:       sla.UpdatedAt,
	}
}

// ToSLADefinitionResponseList converts a slice of ent.SLADefinition to response slice
func ToSLADefinitionResponseList(slas []*ent.SLADefinition) []*SLADefinitionResponse {
	if slas == nil {
		return nil
	}
	responses := make([]*SLADefinitionResponse, 0, len(slas))
	for _, sla := range slas {
		if sla != nil {
			responses = append(responses, ToSLADefinitionResponse(sla))
		}
	}
	return responses
}

// ===================================
// Knowledge Article Mappers
// ===================================

// ToKnowledgeArticleResponse converts an ent.KnowledgeArticle to response
func ToKnowledgeArticleResponse(article *ent.KnowledgeArticle) *KnowledgeArticleResponse {
	if article == nil {
		return nil
	}

	// Convert status from IsPublished boolean
	status := "draft"
	if article.IsPublished {
		status = "published"
	}

	// Parse tags string to slice (assuming comma-separated)
	tags := []string{}
	if article.Tags != "" {
		// Simple split by comma, trim spaces
		// You might want to use a more robust parsing
		tags = []string{article.Tags}
	}

	response := &KnowledgeArticleResponse{
		ID:        article.ID,
		Title:     article.Title,
		Content:   article.Content,
		Category:  article.Category,
		Status:    status,
		Author:    "", // Default, could be populated from authorID if needed
		Views:     0,  // Default value, could be added to schema if needed
		Tags:      tags,
		TenantID:  article.TenantID,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}

	// Note: If you need to populate Author name, you'll need to query the user separately
	// or use edges if they're available in the schema

	return response
}

// ToKnowledgeArticleResponseList converts a slice of articles to response slice
func ToKnowledgeArticleResponseList(articles []*ent.KnowledgeArticle) []*KnowledgeArticleResponse {
	if articles == nil {
		return nil
	}
	responses := make([]*KnowledgeArticleResponse, 0, len(articles))
	for _, article := range articles {
		if article != nil {
			responses = append(responses, ToKnowledgeArticleResponse(article))
		}
	}
	return responses
}

// ===================================
// Tenant Mappers
// ===================================

// ToTenantResponse converts an ent.Tenant to TenantResponse
func ToTenantResponse(tenant *ent.Tenant) *TenantResponse {
	if tenant == nil {
		return nil
	}

	return &TenantResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		Code:      tenant.Code,
		Status:    string(tenant.Status),
		Type:      string(tenant.Type),
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
	}
}

// ToTenantResponseList converts a slice of ent.Tenant to response slice
func ToTenantResponseList(tenants []*ent.Tenant) []*TenantResponse {
	if tenants == nil {
		return nil
	}
	responses := make([]*TenantResponse, 0, len(tenants))
	for _, tenant := range tenants {
		if tenant != nil {
			responses = append(responses, ToTenantResponse(tenant))
		}
	}
	return responses
}
