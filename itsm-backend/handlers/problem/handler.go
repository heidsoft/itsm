package problem

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) toDTO(p *Problem) *dto.ProblemResponse {
	if p == nil {
		return nil
	}

	resp := dto.ProblemResponse{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		Status:      p.Status,
		Priority:    p.Priority,
		Category:    p.Category,
		RootCause:   p.RootCause,
		Impact:      p.Impact,
		CreatedBy:   p.CreatedBy,
		TenantID:    p.TenantID,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
	if p.AssigneeID != nil {
		resp.AssigneeID = p.AssigneeID
	}

	// 映射关联数据
	if p.Tickets != nil {
		resp.AssociatedTickets = make([]*dto.AssociatedItemResponse, 0, len(p.Tickets))
		for _, t := range p.Tickets {
			resp.AssociatedTickets = append(resp.AssociatedTickets, &dto.AssociatedItemResponse{
				ID:     t.ID,
				Title:  t.Title,
				Status: t.Status,
				Number: t.Number,
				Type:   t.Type,
			})
		}
	}
	if p.Incidents != nil {
		resp.AssociatedIncidents = make([]*dto.AssociatedItemResponse, 0, len(p.Incidents))
		for _, inc := range p.Incidents {
			resp.AssociatedIncidents = append(resp.AssociatedIncidents, &dto.AssociatedItemResponse{
				ID:     inc.ID,
				Title:  inc.Title,
				Status: inc.Status,
				Number: inc.Number,
				Type:   inc.Type,
			})
		}
	}
	if p.Changes != nil {
		resp.AssociatedChanges = make([]*dto.AssociatedItemResponse, 0, len(p.Changes))
		for _, ch := range p.Changes {
			resp.AssociatedChanges = append(resp.AssociatedChanges, &dto.AssociatedItemResponse{
				ID:     ch.ID,
				Title:  ch.Title,
				Status: ch.Status,
				Number: ch.Number,
				Type:   ch.Type,
			})
		}
	}

	return &resp
}

func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")
	userID, _ := c.Get("user_id") // Override req.CreatedBy with actual user?

	// Legacy DTO has CreatedBy in request, but better to enforce from context
	createdBy := userID.(int)

	problem := &Problem{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Status:      "open",
		Category:    req.Category,
		RootCause:   req.RootCause,
		Impact:      req.Impact,
		CreatedBy:   createdBy,
	}

	created, err := h.service.Create(c.Request.Context(), tenantID.(int), problem)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, h.toDTO(created))
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	p, err := h.service.GetWithAssociations(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		} else {
			common.Fail(c, common.InternalErrorCode, err.Error())
		}
		return
	}
	common.Success(c, h.toDTO(p))
}

// GetAssociations 获取问题的关联项
func (h *Handler) GetAssociations(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	p, err := h.service.GetWithAssociations(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		} else {
			common.Fail(c, common.InternalErrorCode, err.Error())
		}
		return
	}

	resp := &dto.ProblemAssociationResponse{
		Tickets:   make([]*dto.AssociatedItemResponse, 0),
		Incidents: make([]*dto.AssociatedItemResponse, 0),
		Changes:   make([]*dto.AssociatedItemResponse, 0),
	}
	for _, t := range p.Tickets {
		resp.Tickets = append(resp.Tickets, &dto.AssociatedItemResponse{
			ID: t.ID, Title: t.Title, Status: t.Status, Number: t.Number, Type: t.Type,
		})
	}
	for _, inc := range p.Incidents {
		resp.Incidents = append(resp.Incidents, &dto.AssociatedItemResponse{
			ID: inc.ID, Title: inc.Title, Status: inc.Status, Number: inc.Number, Type: inc.Type,
		})
	}
	for _, ch := range p.Changes {
		resp.Changes = append(resp.Changes, &dto.AssociatedItemResponse{
			ID: ch.ID, Title: ch.Title, Status: ch.Status, Number: ch.Number, Type: ch.Type,
		})
	}
	common.Success(c, resp)
}

// AddAssociation 添加关联
func (h *Handler) AddAssociation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req dto.ProblemAssociationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")
	// 验证问题存在
	_, err = h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		} else {
			common.Fail(c, common.InternalErrorCode, err.Error())
		}
		return
	}

	if err := h.service.AddAssociations(c.Request.Context(), id, req.RelatedType, req.RelatedIDs); err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

// RemoveAssociation 移除关联
func (h *Handler) RemoveAssociation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req dto.ProblemRemoveAssociationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")
	// 验证问题存在
	_, err = h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		} else {
			common.Fail(c, common.InternalErrorCode, err.Error())
		}
		return
	}

	if err := h.service.RemoveAssociation(c.Request.Context(), id, req.RelatedType, req.RelatedID); err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

func (h *Handler) List(c *gin.Context) {
	var req dto.ListProblemsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")

	// Convert DTO filters to map
	filters := make(map[string]interface{})
	if req.Status != "" {
		filters["status"] = req.Status
	}
	if req.Priority != "" {
		filters["priority"] = req.Priority
	}
	if req.Category != "" {
		filters["category"] = req.Category
	}
	if req.Keyword != "" {
		filters["keyword"] = req.Keyword
	}

	list, total, err := h.service.List(c.Request.Context(), tenantID.(int), req.Page, req.PageSize, filters)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	// Map to DTO response
	dtoProblems := make([]*dto.ProblemResponse, 0, len(list))
	for _, p := range list {
		item := &dto.ProblemResponse{
			ID:          p.ID,
			Title:       p.Title,
			Description: p.Description,
			Status:      p.Status,
			Priority:    p.Priority,
			Category:    p.Category,
			RootCause:   p.RootCause,
			Impact:      p.Impact,
			CreatedBy:   p.CreatedBy,
			TenantID:    p.TenantID,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		}
		if p.AssigneeID != nil {
			item.AssigneeID = p.AssigneeID
		}
		dtoProblems = append(dtoProblems, item)
	}

	common.Success(c, &dto.ListProblemsResponse{
		Problems: dtoProblems,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req dto.UpdateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")

	// 将 DTO 指针字段转换为 domain entity
	updates := &Problem{}
	if req.Title != nil {
		updates.Title = *req.Title
	}
	if req.Description != nil {
		updates.Description = *req.Description
	}
	if req.Status != nil {
		updates.Status = *req.Status
	}
	if req.Priority != nil {
		updates.Priority = *req.Priority
	}
	if req.Category != nil {
		updates.Category = *req.Category
	}
	if req.RootCause != nil {
		updates.RootCause = *req.RootCause
	}
	if req.Impact != nil {
		updates.Impact = *req.Impact
	}

	updated, err := h.service.Update(c.Request.Context(), tenantID.(int), id, updates)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, h.toDTO(updated))
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	err = h.service.Delete(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

func (h *Handler) GetStats(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	stats, err := h.service.GetStats(c.Request.Context(), tenantID.(int))
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	// Map domain stats to DTO
	resp := &dto.ProblemStatsResponse{
		Total:        stats.Total,
		Open:         stats.Open,
		InProgress:   stats.InProgress,
		Resolved:     stats.Resolved,
		Closed:       stats.Closed,
		HighPriority: stats.HighPriority,
	}
	common.Success(c, resp)
}
