package known_error

import (
	"fmt"
	"strconv"
	"sync"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	entknownerror "itsm-backend/ent/knownerror"

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

// toResponse converts ent KnownError to DTO response
func (h *Handler) toResponse(ke *ent.KnownError) *dto.KEDBResponse {
	if ke == nil {
		return nil
	}
	return &dto.KEDBResponse{
		ID:               ke.ID,
		Title:            ke.Title,
		Description:      ke.Description,
		Symptoms:         ke.Symptoms,
		RootCause:        ke.RootCause,
		Workaround:       ke.Workaround,
		Resolution:       ke.Resolution,
		Status:           ke.Status,
		Category:         ke.Category,
		Severity:         ke.Severity,
		AffectedProducts: ke.AffectedProducts,
		AffectedCIs:      ke.AffectedCis,
		Keywords:         ke.Keywords,
		OccurrenceCount:  ke.OccurrenceCount,
		CreatedBy:        ke.CreatedBy,
		TenantID:         ke.TenantID,
		CreatedAt:        ke.CreatedAt,
		UpdatedAt:        ke.UpdatedAt,
	}
}

// ListKnownErrors handles GET /api/v1/known-errors
func (h *Handler) ListKnownErrors(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status := c.Query("status")
	category := c.Query("category")
	severity := c.Query("severity")
	keyword := c.Query("keyword")

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ctx := c.Request.Context()

	// Build query
	query := h.client.KnownError.Query().Where(entknownerror.TenantID(tenantID))

	if status != "" {
		query = query.Where(entknownerror.Status(status))
	}
	if category != "" {
		query = query.Where(entknownerror.Category(category))
	}
	if severity != "" {
		query = query.Where(entknownerror.Severity(severity))
	}
	if keyword != "" {
		query = query.Where(
			entknownerror.Or(
				entknownerror.TitleContains(keyword),
				entknownerror.DescriptionContains(keyword),
				entknownerror.SymptomsContains(keyword),
			),
		)
	}

	// Get total
	total, err := query.Count(ctx)
	if err != nil {
		h.logger.Warnw("Failed to count known errors", "error", err)
		total = 0
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	results, err := query.
		Order(ent.Desc(entknownerror.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		h.logger.Warnw("Failed to list known errors", "error", err)
		common.InternalError(c, "Failed to list known errors")
		return
	}

	// Convert to DTOs
	items := make([]*dto.KEDBResponse, 0, len(results))
	for _, ke := range results {
		items = append(items, h.toResponse(ke))
	}

	common.Success(c, &dto.KEDBListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetKnownError handles GET /api/v1/known-errors/:id
func (h *Handler) GetKnownError(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ctx := c.Request.Context()

	ke, err := h.client.KnownError.Query().
		Where(
			entknownerror.ID(id),
			entknownerror.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			common.NotFound(c, "Known error not found")
			return
		}
		h.logger.Warnw("Failed to get known error", "error", err, "id", id)
		common.InternalError(c, "Failed to get known error")
		return
	}

	common.Success(c, h.toResponse(ke))
}

// CreateKnownError handles POST /api/v1/known-errors
func (h *Handler) CreateKnownError(c *gin.Context) {
	var req dto.KEDBCreateRequest
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
	status := "draft"
	severity := req.Severity
	if severity == "" {
		severity = "medium"
	}
	category := req.Category
	if category == "" {
		category = "general"
	}

	builder := h.client.KnownError.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetSymptoms(req.Symptoms).
		SetRootCause(req.RootCause).
		SetWorkaround(req.Workaround).
		SetResolution(req.Resolution).
		SetStatus(status).
		SetCategory(category).
		SetSeverity(severity).
		SetAffectedProducts(req.AffectedProducts).
		SetAffectedCis(req.AffectedCIs).
		SetKeywords(req.Keywords).
		SetCreatedBy(userID).
		SetTenantID(tenantID)

	ke, err := builder.Save(ctx)
	if err != nil {
		h.logger.Warnw("Failed to create known error", "error", err)
		common.InternalError(c, "Failed to create known error")
		return
	}

	common.Success(c, h.toResponse(ke))
}

// UpdateKnownError handles PUT /api/v1/known-errors/:id
func (h *Handler) UpdateKnownError(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req dto.KEDBUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}

	ctx := c.Request.Context()

	// Get existing
	ke, err := h.client.KnownError.Query().
		Where(
			entknownerror.ID(id),
			entknownerror.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			common.NotFound(c, "Known error not found")
			return
		}
		h.logger.Warnw("Failed to get known error", "error", err, "id", id)
		common.InternalError(c, "Failed to get known error")
		return
	}

	// Build update
	update := ke.Update()

	if req.Title != nil {
		update = update.SetTitle(*req.Title)
	}
	if req.Description != nil {
		update = update.SetDescription(*req.Description)
	}
	if req.Symptoms != nil {
		update = update.SetSymptoms(*req.Symptoms)
	}
	if req.RootCause != nil {
		update = update.SetRootCause(*req.RootCause)
	}
	if req.Workaround != nil {
		update = update.SetWorkaround(*req.Workaround)
	}
	if req.Resolution != nil {
		update = update.SetResolution(*req.Resolution)
	}
	if req.Category != nil {
		update = update.SetCategory(*req.Category)
	}
	if req.Severity != nil {
		update = update.SetSeverity(*req.Severity)
	}
	if req.Status != nil {
		update = update.SetStatus(*req.Status)
	}
	if req.AffectedProducts != nil {
		update = update.SetAffectedProducts(req.AffectedProducts)
	}
	if req.AffectedCIs != nil {
		update = update.SetAffectedCis(req.AffectedCIs)
	}
	if req.Keywords != nil {
		update = update.SetKeywords(req.Keywords)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		h.logger.Warnw("Failed to update known error", "error", err, "id", id)
		common.InternalError(c, "Failed to update known error")
		return
	}

	common.Success(c, h.toResponse(updated))
}

// DeleteKnownError handles DELETE /api/v1/known-errors/:id
func (h *Handler) DeleteKnownError(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ctx := c.Request.Context()

	// Verify exists
	_, err := h.client.KnownError.Query().
		Where(
			entknownerror.ID(id),
			entknownerror.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			common.NotFound(c, "Known error not found")
			return
		}
		h.logger.Warnw("Failed to get known error", "error", err, "id", id)
		common.InternalError(c, "Failed to get known error")
		return
	}

	// Delete
	_, err = h.client.KnownError.Delete().
		Where(entknownerror.ID(id)).
		Exec(ctx)
	if err != nil {
		h.logger.Warnw("Failed to delete known error", "error", err, "id", id)
		common.InternalError(c, "Failed to delete known error")
		return
	}

	common.Success(c, gin.H{"message": "deleted"})
}

// GetStats handles GET /api/v1/known-errors/stats
func (h *Handler) GetStats(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ctx := c.Request.Context()

	// Run total count and all group counts in parallel with error collection
	var total, active, resolved, deprecated, critical, high, medium, low int
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make([]error, 0)

	wg.Add(8)
	go func() {
		defer wg.Done()
		n, err := h.client.KnownError.Query().Where(entknownerror.TenantID(tenantID)).Count(ctx)
		mu.Lock()
		total = n
		if err != nil {
			errs = append(errs, fmt.Errorf("total count: %w", err))
		}
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		n, err := h.client.KnownError.Query().Where(entknownerror.TenantID(tenantID), entknownerror.Status("active")).Count(ctx)
		mu.Lock()
		active = n
		if err != nil {
			errs = append(errs, fmt.Errorf("active count: %w", err))
		}
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		n, err := h.client.KnownError.Query().Where(entknownerror.TenantID(tenantID), entknownerror.Status("resolved")).Count(ctx)
		mu.Lock()
		resolved = n
		if err != nil {
			errs = append(errs, fmt.Errorf("resolved count: %w", err))
		}
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		n, err := h.client.KnownError.Query().Where(entknownerror.TenantID(tenantID), entknownerror.Status("deprecated")).Count(ctx)
		mu.Lock()
		deprecated = n
		if err != nil {
			errs = append(errs, fmt.Errorf("deprecated count: %w", err))
		}
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		n, err := h.client.KnownError.Query().Where(entknownerror.TenantID(tenantID), entknownerror.Severity("critical")).Count(ctx)
		mu.Lock()
		critical = n
		if err != nil {
			errs = append(errs, fmt.Errorf("critical count: %w", err))
		}
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		n, err := h.client.KnownError.Query().Where(entknownerror.TenantID(tenantID), entknownerror.Severity("high")).Count(ctx)
		mu.Lock()
		high = n
		if err != nil {
			errs = append(errs, fmt.Errorf("high count: %w", err))
		}
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		n, err := h.client.KnownError.Query().Where(entknownerror.TenantID(tenantID), entknownerror.Severity("medium")).Count(ctx)
		mu.Lock()
		medium = n
		if err != nil {
			errs = append(errs, fmt.Errorf("medium count: %w", err))
		}
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		n, err := h.client.KnownError.Query().Where(entknownerror.TenantID(tenantID), entknownerror.Severity("low")).Count(ctx)
		mu.Lock()
		low = n
		if err != nil {
			errs = append(errs, fmt.Errorf("low count: %w", err))
		}
		mu.Unlock()
	}()
	wg.Wait()

	if len(errs) > 0 {
		h.logger.Errorw("GetStats: DB queries failed", "errors", errs)
		common.Fail(c, 5001, "Failed to retrieve known error statistics")
		return
	}

	common.Success(c, &dto.KEDBStatsResponse{
		Total:      total,
		Active:     active,
		Resolved:   resolved,
		Deprecated: deprecated,
		Critical:   critical,
		High:       high,
		Medium:     medium,
		Low:        low,
	})
}

// SearchKnownErrors handles GET /api/v1/known-errors/search
// Full-text search across title, description, symptoms, root_cause, workaround
func (h *Handler) SearchKnownErrors(c *gin.Context) {
	queryStr := c.Query("q")
	if queryStr == "" {
		common.ParamError(c, "Query parameter 'q' is required")
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ctx := c.Request.Context()

	results, err := h.client.KnownError.Query().
		Where(
			entknownerror.TenantID(tenantID),
			entknownerror.Or(
				entknownerror.TitleContains(queryStr),
				entknownerror.DescriptionContains(queryStr),
				entknownerror.SymptomsContains(queryStr),
				entknownerror.RootCauseContains(queryStr),
				entknownerror.WorkaroundContains(queryStr),
			),
		).
		Limit(20).
		All(ctx)
	if err != nil {
		h.logger.Warnw("Failed to search known errors", "error", err)
		common.InternalError(c, "Failed to search known errors")
		return
	}

	knownErrors := make([]*dto.KEDBResponse, 0, len(results))
	for _, ke := range results {
		knownErrors = append(knownErrors, h.toResponse(ke))
	}

	common.Success(c, gin.H{
		"knownErrors": knownErrors,
		"total":       len(knownErrors),
	})
}

// GetCategories handles GET /api/v1/known-errors/categories
func (h *Handler) GetCategories(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ctx := c.Request.Context()

	results, err := h.client.KnownError.Query().
		Select(entknownerror.FieldCategory).
		Where(entknownerror.TenantID(tenantID)).
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

// PromoteToKnownError handles POST /api/v1/known-errors/:id/promote
// Promotes a draft to active known error status
func (h *Handler) PromoteToKnownError(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ctx := c.Request.Context()

	ke, err := h.client.KnownError.Query().
		Where(
			entknownerror.ID(id),
			entknownerror.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			common.NotFound(c, "Known error not found")
			return
		}
		h.logger.Warnw("Failed to get known error", "error", err, "id", id)
		common.InternalError(c, "Failed to get known error")
		return
	}

	// Update to active status
	_, err = ke.Update().
		SetStatus("active").
		Save(ctx)
	if err != nil {
		h.logger.Warnw("Failed to promote known error", "error", err, "id", id)
		common.InternalError(c, "Failed to promote known error")
		return
	}

	common.Success(c, gin.H{"message": "promoted to active"})
}

// RegisterRoutes registers the known error routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	knownErrors := r.Group("/known-errors")
	{
		knownErrors.GET("", h.ListKnownErrors)
		knownErrors.GET("/stats", h.GetStats)
		knownErrors.GET("/categories", h.GetCategories)
		knownErrors.GET("/search", h.SearchKnownErrors)
		knownErrors.GET("/:id", h.GetKnownError)
		knownErrors.POST("", h.CreateKnownError)
		knownErrors.PUT("/:id", h.UpdateKnownError)
		knownErrors.DELETE("/:id", h.DeleteKnownError)
		knownErrors.POST("/:id/promote", h.PromoteToKnownError)
	}
}
