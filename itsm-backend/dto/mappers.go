package dto

import (
	"context"
	"strconv"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"itsm-backend/ent"
)

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
		MSPRole:    func() *string { s := string(user.MspRole); return &s }(),
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
		ID:             ticket.ID,
		Title:          ticket.Title,
		Description:    ticket.Description,
		Status:         ticket.Status,
		Priority:       ticket.Priority,
		Type:           ticket.Type,
		TicketNumber:   ticket.TicketNumber,
		RequesterID:    ticket.RequesterID,
		AssigneeID:     ticket.AssigneeID,
		TenantID:       ticket.TenantID,
		CategoryID:     ticket.CategoryID,
		DepartmentID:   ticket.DepartmentID,
		ParentTicketID: ticket.ParentTicketID,
		Version:        ticket.Version, // 乐观锁版本号，前端用于并发冲突检测
		CreatedAt:      ticket.CreatedAt,
		UpdatedAt:      ticket.UpdatedAt,
		Resolution:     ticket.Resolution,
		Rating:         ticket.Rating,
	}
	// 时间戳字段：避免 ent 零值（0001-01-01）污染 JSON 输出
	if !ticket.ResolvedAt.IsZero() {
		resolved := ticket.ResolvedAt
		response.ResolvedAt = &resolved
	}
	if !ticket.FirstResponseAt.IsZero() {
		first := ticket.FirstResponseAt
		response.FirstResponseAt = &first
	}
	// ResolutionCategory 和 ClosedAt 是后期 raw SQL 添加的列，ent 不知道
	// 在控制器中调用 PopulateTicketExtraFields 补全

	return response
}

// ToTicketResponseWithUsers converts an ent.Ticket to TicketResponse with user info
func ToTicketResponseWithUsers(ticket *ent.Ticket, requester *ent.User, assignee *ent.User) *TicketResponse {
	if ticket == nil {
		return nil
	}

	response := ToTicketResponse(ticket)

	if requester != nil {
		response.Requester = &UserBasicInfo{
			ID:       requester.ID,
			Username: requester.Username,
			Name:     requester.Name,
			Email:    requester.Email,
			Role:     string(requester.Role),
		}
	}

	if assignee != nil && assignee.ID > 0 {
		response.Assignee = &UserBasicInfo{
			ID:       assignee.ID,
			Username: assignee.Username,
			Name:     assignee.Name,
			Email:    assignee.Email,
			Role:     string(assignee.Role),
		}
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

	var impactAnalysis *ImpactAnalysis
	if incident.ImpactAnalysis != nil {
		impactAnalysis = &ImpactAnalysis{}
		MapToStruct(incident.ImpactAnalysis, impactAnalysis)
	}

	var rootCause *RootCause
	if incident.RootCause != nil {
		rootCause = &RootCause{}
		MapToStruct(incident.RootCause, rootCause)
	}

	var resolutionSteps []ResolutionStep
	if incident.ResolutionSteps != nil {
		MapSliceToStructSlice(incident.ResolutionSteps, &resolutionSteps)
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
		ImpactAnalysis:  impactAnalysis,
		RootCause:       rootCause,
		ResolutionSteps: resolutionSteps,
		IsAutomated:     incident.IsAutomated,
		IsMajorIncident: incident.IsMajorIncident,
		TenantID:        incident.TenantID,
		Version:         incident.Version, // 乐观锁版本号
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
	if !incident.DetectedAt.IsZero() {
		response.DetectedAt = incident.DetectedAt
	}

	if !incident.EscalatedAt.IsZero() {
		response.EscalatedAt = &incident.EscalatedAt
	}

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

	// Parse tags string to slice (comma-separated)
	tags := []string{}
	if article.Tags != "" {
		parts := strings.Split(article.Tags, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
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

	response := &TenantResponse{
		ID:             tenant.ID,
		Name:           tenant.Name,
		Code:           tenant.Code,
		Status:         string(tenant.Status),
		Type:           string(tenant.Type),
		BillingEnabled: tenant.BillingEnabled,
		CreatedAt:      tenant.CreatedAt,
		UpdatedAt:      tenant.UpdatedAt,
	}

	if tenant.Domain != "" {
		response.Domain = &tenant.Domain
	}

	if !tenant.ExpiresAt.IsZero() {
		response.ExpiresAt = &tenant.ExpiresAt
	}
	if tenant.ParentTenantID != 0 {
		response.ParentTenantID = &tenant.ParentTenantID
	}
	if tenant.MspProviderID != 0 {
		response.MSPProviderID = &tenant.MspProviderID
	}
	if tenant.PlanCode != "" {
		response.PlanCode = &tenant.PlanCode
	}
	if tenant.CostCenterCode != "" {
		response.CostCenterCode = &tenant.CostCenterCode
	}
	if tenant.LegalEntityCode != "" {
		response.LegalEntityCode = &tenant.LegalEntityCode
	}
	if tenant.Currency != "" {
		response.Currency = &tenant.Currency
	}
	if tenant.ServiceTier != "" {
		response.ServiceTier = &tenant.ServiceTier
	}
	if tenant.OwnerContact != "" {
		response.OwnerContact = &tenant.OwnerContact
	}

	return response
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

// ===================================
// Change Mappers
// ===================================

// ToChangeResponse converts an ent.Change to ChangeResponse
func ToChangeResponse(change *ent.Change) *ChangeResponse {
	if change == nil {
		return nil
	}

	response := &ChangeResponse{
		ID:                 change.ID,
		Title:              change.Title,
		Description:        change.Description,
		Justification:      change.Justification,
		Type:               ChangeType(change.Type),
		Status:             ChangeStatus(change.Status),
		Priority:           ChangePriority(change.Priority),
		ImpactScope:        ChangeImpact(change.ImpactScope),
		RiskLevel:          ChangeRisk(change.RiskLevel),
		CreatedBy:          change.CreatedBy,
		TenantID:           change.TenantID,
		ImplementationPlan: change.ImplementationPlan,
		RollbackPlan:       change.RollbackPlan,
		AffectedCIs:        change.AffectedCis,
		RelatedTickets:     change.RelatedTickets,
		CreatedAt:          change.CreatedAt,
		UpdatedAt:          change.UpdatedAt,
		PlannedStartDate:   &change.PlannedStartDate,
		PlannedEndDate:     &change.PlannedEndDate,
		ActualStartDate:    &change.ActualStartDate,
		ActualEndDate:      &change.ActualEndDate,
	}

	if change.AssigneeID > 0 {
		response.AssigneeID = &change.AssigneeID
	}

	return response
}

// ToChangeResponseList converts a slice of ent.Change to ChangeResponse slice
func ToChangeResponseList(changes []*ent.Change) []*ChangeResponse {
	if changes == nil {
		return nil
	}
	responses := make([]*ChangeResponse, 0, len(changes))
	for _, change := range changes {
		if change != nil {
			responses = append(responses, ToChangeResponse(change))
		}
	}
	return responses
}

// ===================================
// Problem Mappers
// ===================================

// ToProblemResponse converts an ent.Problem to ProblemResponse
func ToProblemResponse(problem *ent.Problem) *ProblemResponse {
	if problem == nil {
		return nil
	}

	response := &ProblemResponse{
		ID:          problem.ID,
		Title:       problem.Title,
		Description: problem.Description,
		Status:      problem.Status,
		Priority:    problem.Priority,
		Category:    problem.Category,
		RootCause:   problem.RootCause,
		Impact:      problem.Impact,
		CreatedBy:   problem.CreatedBy,
		TenantID:    problem.TenantID,
		CreatedAt:   problem.CreatedAt,
		UpdatedAt:   problem.UpdatedAt,
	}

	if problem.AssigneeID > 0 {
		response.AssigneeID = &problem.AssigneeID
	}

	return response
}

// ToProblemResponseList converts a slice of ent.Problem to ProblemResponse slice
func ToProblemResponseList(problems []*ent.Problem) []*ProblemResponse {
	if problems == nil {
		return nil
	}
	responses := make([]*ProblemResponse, 0, len(problems))
	for _, problem := range problems {
		if problem != nil {
			responses = append(responses, ToProblemResponse(problem))
		}
	}
	return responses
}

// ===================================
// Project Mappers
// ===================================

// ToProjectResponse converts an ent.Project to ProjectResponse
func ToProjectResponse(project *ent.Project) *ProjectResponse {
	if project == nil {
		return nil
	}

	response := &ProjectResponse{
		ID:          project.ID,
		Name:        project.Name,
		Code:        project.Code,
		Description: project.Description,
		Status:      project.Status,
		TenantID:    project.TenantID,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}

	if project.ManagerID > 0 {
		response.ManagerID = &project.ManagerID
	}

	if project.DepartmentID > 0 {
		response.DepartmentID = &project.DepartmentID
	}

	if !project.StartDate.IsZero() {
		response.StartDate = &project.StartDate
	}

	if !project.EndDate.IsZero() {
		response.EndDate = &project.EndDate
	}

	return response
}

// ToProjectResponseList converts a slice of ent.Project to ProjectResponse slice
func ToProjectResponseList(projects []*ent.Project) []*ProjectResponse {
	if projects == nil {
		return nil
	}
	responses := make([]*ProjectResponse, 0, len(projects))
	for _, project := range projects {
		if project != nil {
			responses = append(responses, ToProjectResponse(project))
		}
	}
	return responses
}

// ===================================
// ApprovalChain Mappers
// ===================================

// ToApprovalChainResponse converts an ent.ApprovalChain to ApprovalChainResponse
func ToApprovalChainResponse(chain *ent.ApprovalChain) *ApprovalChainResponse {
	if chain == nil {
		return nil
	}

	// Convert schema.ApprovalChainStep to dto.ApprovalChainStepDTO
	chainDTO := make([]ApprovalChainStepDTO, len(chain.Chain))
	for i, step := range chain.Chain {
		chainDTO[i] = ApprovalChainStepDTO{
			Level:      step.Level,
			ApproverID: step.ApproverID,
			Role:       step.Role,
			Name:       step.Name,
			IsRequired: step.IsRequired,
		}
	}

	return &ApprovalChainResponse{
		ID:          chain.ID,
		Name:        chain.Name,
		Description: chain.Description,
		EntityType:  chain.EntityType,
		Chain:       chainDTO,
		Status:      chain.Status,
		CreatedBy:   chain.CreatedBy,
		TenantID:    chain.TenantID,
		CreatedAt:   chain.CreatedAt,
		UpdatedAt:   chain.UpdatedAt,
	}
}

// ToApprovalChainResponseList converts a slice of ent.ApprovalChain to ApprovalChainResponse slice
func ToApprovalChainResponseList(chains []*ent.ApprovalChain) []ApprovalChainResponse {
	if chains == nil {
		return nil
	}
	responses := make([]ApprovalChainResponse, 0, len(chains))
	for _, chain := range chains {
		if chain != nil {
			resp := ToApprovalChainResponse(chain)
			if resp != nil {
				responses = append(responses, *resp)
			}
		}
	}
	return responses
}

// ===================================
// Group Mappers
// ===================================

// ToGroupResponse converts an ent.Group to GroupResponse
func ToGroupResponse(group *ent.Group) *GroupResponse {
	if group == nil {
		return nil
	}

	return &GroupResponse{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		TenantID:    group.TenantID,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	}
}

// ToGroupResponseList converts a slice of ent.Group to GroupResponse slice
func ToGroupResponseList(groups []*ent.Group) []*GroupResponse {
	if groups == nil {
		return nil
	}
	responses := make([]*GroupResponse, 0, len(groups))
	for _, group := range groups {
		if group != nil {
			responses = append(responses, ToGroupResponse(group))
		}
	}
	return responses
}

// ===================================
// SLAPolicy Mappers
// ===================================

// ToSLAPolicyResponse converts an ent.SLAPolicy to SLAPolicyResponse
func ToSLAPolicyResponse(policy *ent.SLAPolicy) *SLAPolicyResponse {
	if policy == nil {
		return nil
	}

	return &SLAPolicyResponse{
		ID:                    policy.ID,
		Name:                  policy.Name,
		Description:           policy.Description,
		CustomerTier:          policy.CustomerTier,
		TicketType:            policy.TicketType,
		Priority:              policy.Priority,
		ResponseTimeMinutes:   policy.ResponseTimeMinutes,
		ResolutionTimeMinutes: policy.ResolutionTimeMinutes,
		BusinessHours:         policy.BusinessHours,
		ExcludeWeekends:       policy.ExcludeWeekends,
		ExcludeHolidays:       policy.ExcludeHolidays,
		IsActive:              policy.IsActive,
		PriorityScore:         policy.PriorityScore,
		TenantID:              policy.TenantID,
		CreatedAt:             policy.CreatedAt,
		UpdatedAt:             policy.UpdatedAt,
	}
}

// ToSLAPolicyResponseList converts a slice of ent.SLAPolicy to SLAPolicyResponse slice
func ToSLAPolicyResponseList(policies []*ent.SLAPolicy) []*SLAPolicyResponse {
	if policies == nil {
		return nil
	}
	responses := make([]*SLAPolicyResponse, 0, len(policies))
	for _, policy := range policies {
		if policy != nil {
			responses = append(responses, ToSLAPolicyResponse(policy))
		}
	}
	return responses
}

// ===================================
// Workflow Mappers
// ===================================

// WorkflowResponse 工作流响应
type WorkflowResponse struct {
	ID           int                    `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         string                 `json:"type"`
	Definition   map[string]interface{} `json:"definition"`
	Version      string                 `json:"version"`
	IsActive     bool                   `json:"is_active"`
	TenantID     int                    `json:"tenant_id"`
	DepartmentID *int                   `json:"department_id,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// WorkflowInstanceResponse 工作流实例响应
type WorkflowInstanceResponse struct {
	ID          int                    `json:"id"`
	Status      string                 `json:"status"`
	CurrentStep string                 `json:"current_step"`
	Context     map[string]interface{} `json:"context"`
	WorkflowID  int                    `json:"workflow_id"`
	EntityID    int                    `json:"entity_id"`
	EntityType  string                 `json:"entity_type"`
	TenantID    int                    `json:"tenant_id"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ToWorkflowResponse converts an ent.Workflow to WorkflowResponse
func ToWorkflowResponse(workflow *ent.Workflow) *WorkflowResponse {
	if workflow == nil {
		return nil
	}

	// Parse definition JSON bytes to map
	var definition map[string]interface{}
	if workflow.Definition != nil {
		_ = json.Unmarshal(workflow.Definition, &definition)
	}

	response := &WorkflowResponse{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Type:        workflow.Type,
		Definition:  definition,
		Version:     workflow.Version,
		IsActive:    workflow.IsActive,
		TenantID:    workflow.TenantID,
		CreatedAt:   workflow.CreatedAt,
		UpdatedAt:   workflow.UpdatedAt,
	}

	if workflow.DepartmentID > 0 {
		response.DepartmentID = &workflow.DepartmentID
	}

	return response
}

// ToWorkflowResponseList converts a slice of ent.Workflow to WorkflowResponse slice
func ToWorkflowResponseList(workflows []*ent.Workflow) []*WorkflowResponse {
	if workflows == nil {
		return nil
	}
	responses := make([]*WorkflowResponse, 0, len(workflows))
	for _, workflow := range workflows {
		if workflow != nil {
			responses = append(responses, ToWorkflowResponse(workflow))
		}
	}
	return responses
}

// ToWorkflowInstanceResponse converts an ent.WorkflowInstance to WorkflowInstanceResponse
func ToWorkflowInstanceResponse(instance *ent.WorkflowInstance) *WorkflowInstanceResponse {
	if instance == nil {
		return nil
	}

	// Parse context JSON bytes to map
	var context map[string]interface{}
	if instance.Context != nil {
		_ = json.Unmarshal(instance.Context, &context)
	}

	response := &WorkflowInstanceResponse{
		ID:          instance.ID,
		Status:      instance.Status,
		CurrentStep: instance.CurrentStep,
		Context:     context,
		WorkflowID:  instance.WorkflowID,
		EntityID:    instance.EntityID,
		EntityType:  instance.EntityType,
		TenantID:    instance.TenantID,
		StartedAt:   instance.StartedAt,
		CreatedAt:   instance.CreatedAt,
		UpdatedAt:   instance.UpdatedAt,
	}

	if !instance.CompletedAt.IsZero() {
		response.CompletedAt = &instance.CompletedAt
	}

	return response
}

// ToWorkflowInstanceResponseList converts a slice of ent.WorkflowInstance to WorkflowInstanceResponse slice
func ToWorkflowInstanceResponseList(instances []*ent.WorkflowInstance) []*WorkflowInstanceResponse {
	if instances == nil {
		return nil
	}
	responses := make([]*WorkflowInstanceResponse, 0, len(instances))
	for _, instance := range instances {
		if instance != nil {
			responses = append(responses, ToWorkflowInstanceResponse(instance))
		}
	}
	return responses
}

// ===================================
// TicketTag Mappers
// ===================================

// ToTicketTagResponse converts an ent.TicketTag to TicketTagResponse
func ToTicketTagResponse(tag *ent.TicketTag) *TicketTagResponse {
	if tag == nil {
		return nil
	}

	return &TicketTagResponse{
		ID:          tag.ID,
		Name:        tag.Name,
		Color:       tag.Color,
		Description: tag.Description,
		IsActive:    tag.IsActive,
		TenantID:    tag.TenantID,
		CreatedAt:   tag.CreatedAt,
		UpdatedAt:   tag.UpdatedAt,
	}
}

// ToTicketTagResponseList converts a slice of ent.TicketTag to TicketTagResponse slice
func ToTicketTagResponseList(tags []*ent.TicketTag) []*TicketTagResponse {
	if tags == nil {
		return nil
	}
	responses := make([]*TicketTagResponse, 0, len(tags))
	for _, tag := range tags {
		if tag != nil {
			responses = append(responses, ToTicketTagResponse(tag))
		}
	}
	return responses
}

// ===================================
// TicketCategory Mappers
// ===================================

// ToTicketCategoryResponse converts an ent.TicketCategory to TicketCategoryResponse
func ToTicketCategoryResponse(category *ent.TicketCategory) *TicketCategoryResponse {
	if category == nil {
		return nil
	}

	response := &TicketCategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Code:        category.Code,
		Description: category.Description,
		SortOrder:   category.SortOrder,
		IsActive:    category.IsActive,
		TenantID:    category.TenantID,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
	}

	if category.ParentID > 0 {
		response.ParentID = &category.ParentID
	}

	return response
}

// ToTicketCategoryResponseList converts a slice of ent.TicketCategory to TicketCategoryResponse slice
func ToTicketCategoryResponseList(categories []*ent.TicketCategory) []*TicketCategoryResponse {
	if categories == nil {
		return nil
	}
	responses := make([]*TicketCategoryResponse, 0, len(categories))
	for _, category := range categories {
		if category != nil {
			responses = append(responses, ToTicketCategoryResponse(category))
		}
	}
	return responses
}


// TicketExtraFields 后期通过 raw SQL 添加的列
type TicketExtraFields struct {
	ResolutionCategory sql.NullString
	ClosedAt           sql.NullTime
}

// EnrichTicketResponses 批量从 raw DB 拉取 ent 不知道的字段并 patch 进响应
// 用于 ListTickets 等返回多条记录的场景
func EnrichTicketResponses(ctx context.Context, db *sql.DB, responses []*TicketResponse, tenantID int) {
	if db == nil || len(responses) == 0 {
		return
	}
	ids := make([]int, 0, len(responses))
	idSet := make(map[int]struct{}, len(responses))
	for _, r := range responses {
		if r == nil {
			continue
		}
		if _, ok := idSet[r.ID]; ok {
			continue
		}
		idSet[r.ID] = struct{}{}
		ids = append(ids, r.ID)
	}
	if len(ids) == 0 {
		return
	}
	// 用 ANY($1) 批量查询
	query := `SELECT id, resolution_category, closed_at FROM tickets WHERE tenant_id = $1 AND id = ANY($2::int[])`
	rows, err := db.QueryContext(ctx, query, tenantID, pqIntArray(ids))
	if err != nil {
		return
	}
	defer rows.Close()
	indexByID := make(map[int]int, len(responses))
	for i, r := range responses {
		if r != nil {
			indexByID[r.ID] = i
		}
	}
	for rows.Next() {
		var id int
		var rc sql.NullString
		var ca sql.NullTime
		if err := rows.Scan(&id, &rc, &ca); err == nil {
			if i, ok := indexByID[id]; ok {
				if rc.Valid {
					responses[i].ResolutionCategory = rc.String
				}
				if ca.Valid {
					t := ca.Time
					responses[i].ClosedAt = &t
				}
			}
		}
	}
}

// EnrichTicketResponse 单条 ticket 富化
func EnrichTicketResponse(ctx context.Context, db *sql.DB, response *TicketResponse, tenantID int) {
	if db == nil || response == nil {
		return
	}
	query := `SELECT resolution_category, closed_at FROM tickets WHERE id = $1 AND tenant_id = $2`
	row := db.QueryRowContext(ctx, query, response.ID, tenantID)
	var rc sql.NullString
	var ca sql.NullTime
	if err := row.Scan(&rc, &ca); err != nil {
		return
	}
	if rc.Valid {
		response.ResolutionCategory = rc.String
	}
	if ca.Valid {
		t := ca.Time
		response.ClosedAt = &t
	}
}

// pqIntArray 把 []int 序列化为 PostgreSQL int[] 文本
func pqIntArray(ids []int) string {
	if len(ids) == 0 {
		return "{}"
	}
	var b []byte
	b = append(b, '{')
	for i, v := range ids {
		if i > 0 {
			b = append(b, ',')
		}
		b = strconv.AppendInt(b, int64(v), 10)
	}
	b = append(b, '}')
	return string(b)
}
