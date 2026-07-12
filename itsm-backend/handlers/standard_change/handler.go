package standard_change

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	entstandardchange "itsm-backend/ent/standardchange"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewHandler(client *ent.Client, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		client: client,
		logger: logger,
	}
}

// toResponse converts ent StandardChange to DTO response
func toResponse(sc *ent.StandardChange) *dto.StandardChangeResponse {
	if sc == nil {
		return nil
	}
	return &dto.StandardChangeResponse{
		ID:                 sc.ID,
		Title:              sc.Title,
		Description:        sc.Description,
		ImplementationPlan: sc.ImplementationPlan,
		RollbackPlan:       sc.RollbackPlan,
		Justification:      sc.Justification,
		Category:           sc.Category,
		RiskLevel:          sc.RiskLevel,
		ImpactScope:        sc.ImpactScope,
		ExpectedDuration:   sc.ExpectedDuration,
		ApprovalRequired:   sc.ApprovalRequired,
		AffectedCis:        sc.AffectedCis,
		Prerequisites:      sc.Prerequisites,
		Remarks:            sc.Remarks,
		CreatedBy:          sc.CreatedBy,
		TenantID:           sc.TenantID,
		IsActive:           sc.IsActive,
		CreatedAt:          sc.CreatedAt,
		UpdatedAt:          sc.UpdatedAt,
	}
}

// ListStandardChanges handles GET /api/v1/standard-changes
func (h *Handler) ListStandardChanges(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	category := c.Query("category")
	search := c.Query("search")
	activeOnly := c.Query("active_only") == "true"

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ctx := c.Request.Context()

	// Build query
	query := h.client.StandardChange.Query().Where(entstandardchange.TenantID(tenantID))

	if activeOnly {
		query = query.Where(entstandardchange.IsActive(true))
	}

	if category != "" {
		query = query.Where(entstandardchange.Category(category))
	}

	if search != "" {
		query = query.Where(
			entstandardchange.Or(
				entstandardchange.TitleContains(search),
				entstandardchange.DescriptionContains(search),
			),
		)
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		h.logger.Warnw("Failed to count standard changes", "error", err)
		total = 0
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	results, err := query.
		Order(ent.Desc(entstandardchange.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		h.logger.Warnw("Failed to list standard changes", "error", err)
		common.InternalError(c, "Failed to list standard changes")
		return
	}

	// Convert to DTOs
	templates := make([]dto.StandardChangeResponse, 0, len(results))
	for _, sc := range results {
		templates = append(templates, *toResponse(sc))
	}

	common.Success(c, gin.H{
		"templates": templates,
		"total":     total,
		"page":      page,
		"pageSize": pageSize,
	})
}

// GetStandardChange handles GET /api/v1/standard-changes/:id
func (h *Handler) GetStandardChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ctx := c.Request.Context()

	sc, err := h.client.StandardChange.Query().
		Where(
			entstandardchange.ID(id),
			entstandardchange.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			common.NotFound(c, "Standard change template not found")
			return
		}
		h.logger.Warnw("Failed to get standard change", "error", err, "id", id)
		common.InternalError(c, "Failed to get standard change")
		return
	}

	common.Success(c, toResponse(sc))
}

// CreateStandardChange handles POST /api/v1/standard-changes
func (h *Handler) CreateStandardChange(c *gin.Context) {
	var req dto.CreateStandardChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body: "+err.Error())
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	userIDVal, _ := c.Get("user_id")
	userID := userIDVal.(int)

	ctx := c.Request.Context()

	// Set defaults
	riskLevel := req.RiskLevel
	if riskLevel == "" {
		riskLevel = "low"
	}
	impactScope := req.ImpactScope
	if impactScope == "" {
		impactScope = "low"
	}
	category := req.Category
	if category == "" {
		category = "general"
	}

	// expected_duration: when omitted the JSON zero value (0) would be set
	// explicitly and override the schema default (30). Apply the default for
	// any non-positive value so templates keep a sane estimated duration.
	expectedDuration := req.ExpectedDuration
	if expectedDuration <= 0 {
		expectedDuration = 30
	}

	sc, err := h.client.StandardChange.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetImplementationPlan(req.ImplementationPlan).
		SetRollbackPlan(req.RollbackPlan).
		SetJustification(req.Justification).
		SetCategory(category).
		SetRiskLevel(riskLevel).
		SetImpactScope(impactScope).
		SetExpectedDuration(expectedDuration).
		SetApprovalRequired(req.ApprovalRequired).
		SetAffectedCis(req.AffectedCis).
		SetPrerequisites(req.Prerequisites).
		SetRemarks(req.Remarks).
		SetCreatedBy(userID).
		SetTenantID(tenantID).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		h.logger.Warnw("Failed to create standard change", "error", err)
		common.InternalError(c, "Failed to create standard change template")
		return
	}

	common.Success(c, toResponse(sc))
}

// UpdateStandardChange handles PUT /api/v1/standard-changes/:id
func (h *Handler) UpdateStandardChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req dto.UpdateStandardChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}

	ctx := c.Request.Context()

	// Get existing
	sc, err := h.client.StandardChange.Query().
		Where(
			entstandardchange.ID(id),
			entstandardchange.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			common.NotFound(c, "Standard change template not found")
			return
		}
		h.logger.Warnw("Failed to get standard change", "error", err, "id", id)
		common.InternalError(c, "Failed to get standard change")
		return
	}

	// Build update builder
	update := sc.Update()

	if req.Title != nil {
		update = update.SetTitle(*req.Title)
	}
	if req.Description != nil {
		update = update.SetDescription(*req.Description)
	}
	if req.ImplementationPlan != nil {
		update = update.SetImplementationPlan(*req.ImplementationPlan)
	}
	if req.RollbackPlan != nil {
		update = update.SetRollbackPlan(*req.RollbackPlan)
	}
	if req.Justification != nil {
		update = update.SetJustification(*req.Justification)
	}
	if req.Category != nil {
		update = update.SetCategory(*req.Category)
	}
	if req.RiskLevel != nil {
		update = update.SetRiskLevel(*req.RiskLevel)
	}
	if req.ImpactScope != nil {
		update = update.SetImpactScope(*req.ImpactScope)
	}
	if req.ExpectedDuration != nil {
		update = update.SetExpectedDuration(*req.ExpectedDuration)
	}
	if req.ApprovalRequired != nil {
		update = update.SetApprovalRequired(*req.ApprovalRequired)
	}
	if req.AffectedCis != nil {
		update = update.SetAffectedCis(req.AffectedCis)
	}
	if req.Prerequisites != nil {
		update = update.SetPrerequisites(req.Prerequisites)
	}
	if req.Remarks != nil {
		update = update.SetRemarks(*req.Remarks)
	}
	if req.IsActive != nil {
		update = update.SetIsActive(*req.IsActive)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		h.logger.Warnw("Failed to update standard change", "error", err, "id", id)
		common.InternalError(c, "Failed to update standard change template")
		return
	}

	common.Success(c, toResponse(updated))
}

// DeleteStandardChange handles DELETE /api/v1/standard-changes/:id
func (h *Handler) DeleteStandardChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ctx := c.Request.Context()

	// Verify exists
	_, err := h.client.StandardChange.Query().
		Where(
			entstandardchange.ID(id),
			entstandardchange.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			common.NotFound(c, "Standard change template not found")
			return
		}
		h.logger.Warnw("Failed to get standard change", "error", err, "id", id)
		common.InternalError(c, "Failed to get standard change")
		return
	}

	// Soft delete - just deactivate
	_, err = h.client.StandardChange.Update().
		Where(entstandardchange.ID(id)).
		SetIsActive(false).
		Save(ctx)
	if err != nil {
		h.logger.Warnw("Failed to delete standard change", "error", err, "id", id)
		common.InternalError(c, "Failed to delete standard change template")
		return
	}

	common.Success(c, gin.H{"message": "deleted"})
}

// GetCategories handles GET /api/v1/standard-changes/categories
func (h *Handler) GetCategories(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ctx := c.Request.Context()

	// Get all templates and extract distinct categories
	results, err := h.client.StandardChange.Query().
		Select(entstandardchange.FieldCategory).
		Where(entstandardchange.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		h.logger.Warnw("Failed to get categories", "error", err)
		common.InternalError(c, "Failed to get categories")
		return
	}

	// Extract distinct categories
	categorySet := make(map[string]bool)
	for _, r := range results {
		if r.Category != "" {
			categorySet[r.Category] = true
		}
	}

	categories := make([]string, 0, len(categorySet))
	for cat := range categorySet {
		categories = append(categories, cat)
	}

	common.Success(c, gin.H{"categories": categories})
}

// InstantiateStandardChange handles POST /api/v1/standard-changes/:id/instantiate
// Creates a new Change from a standard change template
func (h *Handler) InstantiateStandardChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	userIDVal, _ := c.Get("user_id")
	userID := userIDVal.(int)

	var req dto.InstantiateStandardChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Body is optional
		req = dto.InstantiateStandardChangeRequest{}
	}

	ctx := c.Request.Context()

	// Get the template
	template, err := h.client.StandardChange.Query().
		Where(
			entstandardchange.ID(id),
			entstandardchange.TenantID(tenantID),
			entstandardchange.IsActive(true),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			common.NotFound(c, "Standard change template not found")
			return
		}
		h.logger.Warnw("Failed to get standard change template", "error", err, "id", id)
		common.InternalError(c, "Failed to get standard change template")
		return
	}

	// Determine title
	title := template.Title
	if req.Title != "" {
		title = req.Title
	}

	// Determine affected CIs
	affectedCIs := template.AffectedCis
	if len(req.AffectedCis) > 0 {
		affectedCIs = req.AffectedCis
	}

	// Create change from template
	change, err := h.client.Change.Create().
		SetTitle(title).
		SetDescription(template.Description).
		SetJustification(template.Justification).
		SetType("standard").
		SetStatus("draft").
		SetPriority("medium").
		SetImpactScope(template.ImpactScope).
		SetRiskLevel(template.RiskLevel).
		SetImplementationPlan(template.ImplementationPlan).
		SetRollbackPlan(template.RollbackPlan).
		SetAffectedCis(affectedCIs).
		SetCreatedBy(userID).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		h.logger.Warnw("Failed to create change from template", "error", err)
		common.InternalError(c, "Failed to create change from template")
		return
	}

	h.logger.Infow("Created change from standard change template",
		"template_id", id, "change_id", change.ID, "title", change.Title)

	common.Success(c, gin.H{
		"change_id": change.ID,
		"change":    change,
	})
}

// RegisterRoutes registers the standard change routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	standardChanges := r.Group("/standard-changes")
	{
		standardChanges.GET("", h.ListStandardChanges)
		standardChanges.GET("/categories", h.GetCategories)
		standardChanges.GET("/:id", h.GetStandardChange)
		standardChanges.POST("", h.CreateStandardChange)
		standardChanges.PUT("/:id", h.UpdateStandardChange)
		standardChanges.DELETE("/:id", h.DeleteStandardChange)
		standardChanges.POST("/:id/instantiate", h.InstantiateStandardChange)
	}
}
