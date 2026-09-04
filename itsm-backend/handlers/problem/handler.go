package problem

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"

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
		ID:            p.ID,
		ProblemNumber: p.ProblemNumber,
		Title:         p.Title,
		Description:   p.Description,
		Status:        p.Status,
		Priority:      p.Priority,
		Category:      p.Category,
		RootCause:     p.RootCause,
		Workaround:    p.Workaround,
		Resolution:    p.Resolution,
		Impact:        p.Impact,
		CreatedBy:     p.CreatedBy,
		TenantID:      p.TenantID,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
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

// Create 问题管理-创建问题
// @Summary 创建问题
// @Description 创建新的问题记录
// @Tags 问题管理
// @Accept json
// @Produce json
// @Param request body dto.CreateProblemRequest true "创建问题请求"
// @Success 200 {object} common.Response{data=dto.ProblemResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
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
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, h.toDTO(created))
}

// Get 问题管理-获取问题详情
// @Summary 获取问题详情
// @Description 根据ID获取问题详情（含关联数据）
// @Tags 问题管理
// @Produce json
// @Param id path int true "问题ID"
// @Success 200 {object} common.Response{data=dto.ProblemResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	p, err := h.service.GetWithAssociations(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if h.service.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}
	common.Success(c, h.toDTO(p))
}

// GetAssociations 获取问题的关联项
// @Summary 获取问题关联项
// @Description 获取问题关联的事件、工单和变更列表
// @Tags 问题管理
// @Produce json
// @Param id path int true "问题ID"
// @Success 200 {object} common.Response{data=dto.ProblemAssociationResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id}/associations [get]
func (h *Handler) GetAssociations(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	p, err := h.service.GetWithAssociations(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if h.service.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
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
// @Summary 添加问题关联
// @Description 为问题添加关联的事件、工单或变更
// @Tags 问题管理
// @Accept json
// @Produce json
// @Param id path int true "问题ID"
// @Param request body dto.ProblemAssociationRequest true "关联请求"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id}/associations [post]
func (h *Handler) AddAssociation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req dto.ProblemAssociationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	// 验证问题存在
	_, err = h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if h.service.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}

	if err := h.service.AddAssociations(c.Request.Context(), tenantID.(int), id, req.RelatedType, req.RelatedIDs); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// RemoveAssociation 移除关联
// @Summary 移除问题关联
// @Description 移除问题的一个关联项
// @Tags 问题管理
// @Accept json
// @Produce json
// @Param id path int true "问题ID"
// @Param request body dto.ProblemRemoveAssociationRequest true "移除关联请求"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id}/associations [delete]
func (h *Handler) RemoveAssociation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req dto.ProblemRemoveAssociationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	// 验证问题存在
	_, err = h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if h.service.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}

	if err := h.service.RemoveAssociation(c.Request.Context(), tenantID.(int), id, req.RelatedType, req.RelatedID); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// List 问题管理-获取问题列表
// @Summary 获取问题列表
// @Description 分页获取问题列表，支持状态、优先级、分类和关键词过滤
// @Tags 问题管理
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param status query string false "状态过滤"
// @Param priority query string false "优先级过滤"
// @Param category query string false "分类过滤"
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} common.Response{data=dto.ListProblemsResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems [get]
func (h *Handler) List(c *gin.Context) {
	var req dto.ListProblemsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	// 行级数据权限：从鉴权中间件注入的 user_id/role 取得，下传给 service 判定 DataScope。
	currentUserID := c.GetInt("user_id")
	currentRole := c.GetString("role")

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

	list, total, err := h.service.List(c.Request.Context(), tenantID.(int), req.Page, req.PageSize, filters, currentUserID, currentRole)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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

	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	common.Success(c, &dto.ListProblemsResponse{
		Problems:   dtoProblems,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	})
}

// Update 问题管理-更新问题
// @Summary 更新问题
// @Description 更新问题信息（部分字段可选），状态流转违规返回409
// @Tags 问题管理
// @Accept json
// @Produce json
// @Param id path int true "问题ID"
// @Param request body dto.UpdateProblemRequest true "更新问题请求"
// @Success 200 {object} common.Response{data=dto.ProblemResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 409 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req dto.UpdateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
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
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, h.toDTO(updated))
}

// InvestigateProblem 问题管理-开始调查问题
// @Summary 开始调查问题
// @Description 将问题流转为调查中状态，状态流转违规返回409
// @Tags 问题管理
// @Produce json
// @Param id path int true "问题ID"
// @Success 200 {object} common.Response{data=dto.ProblemResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 409 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id}/investigate [post]
func (h *Handler) InvestigateProblem(c *gin.Context) {
	id, tenantID, ok := problemRequestContext(c)
	if !ok {
		return
	}
	updated, err := h.service.InvestigateProblem(c.Request.Context(), tenantID, id)
	h.respondProblemMutation(c, updated, err)
}

// UpdateRootCause 问题管理-记录问题根因
// @Summary 更新问题根因
// @Description 记录问题的根因分析，状态流转违规返回409
// @Tags 问题管理
// @Accept json
// @Produce json
// @Param id path int true "问题ID"
// @Param request body dto.UpdateProblemRootCauseRequest true "根因请求"
// @Success 200 {object} common.Response{data=dto.ProblemResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 409 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id}/root-cause [put]
func (h *Handler) UpdateRootCause(c *gin.Context) {
	id, tenantID, ok := problemRequestContext(c)
	if !ok {
		return
	}
	var req dto.UpdateProblemRootCauseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	updated, err := h.service.UpdateRootCause(c.Request.Context(), tenantID, id, req.RootCause)
	h.respondProblemMutation(c, updated, err)
}

// UpdateSolution 问题管理-更新问题解决方案
// @Summary 更新问题解决方案
// @Description 记录问题的临时解决方案（变通方法）或最终解决方案，状态流转违规返回409
// @Tags 问题管理
// @Accept json
// @Produce json
// @Param id path int true "问题ID"
// @Param request body dto.UpdateProblemResolutionRequest true "解决方案请求"
// @Success 200 {object} common.Response{data=dto.ProblemResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 409 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id}/solution [put]
func (h *Handler) UpdateSolution(c *gin.Context) {
	id, tenantID, ok := problemRequestContext(c)
	if !ok {
		return
	}
	var req dto.UpdateProblemResolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	resolution := req.Resolution
	if resolution == "" {
		resolution = req.Solution
	}
	updated, err := h.service.UpdateSolution(c.Request.Context(), tenantID, id, req.Workaround, resolution)
	h.respondProblemMutation(c, updated, err)
}

// CloseProblem 问题管理-关闭问题
// @Description 关闭问题并可同时记录最终解决方案，状态流转违规返回409
// @Summary 关闭问题
// @Tags 问题管理
// @Accept json
// @Produce json
// @Param id path int true "问题ID"
// @Param request body dto.CloseProblemRequest true "关闭问题请求"
// @Success 200 {object} common.Response{data=dto.ProblemResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 409 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id}/close [post]
func (h *Handler) CloseProblem(c *gin.Context) {
	id, tenantID, ok := problemRequestContext(c)
	if !ok {
		return
	}
	var req dto.CloseProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	updated, err := h.service.CloseProblem(c.Request.Context(), tenantID, id, req.Resolution)
	h.respondProblemMutation(c, updated, err)
}

func problemRequestContext(c *gin.Context) (int, int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return 0, 0, false
	}
	tenantValue, exists := c.Get("tenant_id")
	tenantID, valid := tenantValue.(int)
	if !exists || !valid || tenantID <= 0 {
		common.Fail(c, common.AuthErrorCode, "invalid tenant context")
		return 0, 0, false
	}
	return id, tenantID, true
}

func (h *Handler) respondProblemMutation(c *gin.Context, updated *Problem, err error) {
	if err != nil {
		var bizErr *common.BusinessError
		switch {
		case h.service.IsNotFound(err):
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		case errors.As(err, &bizErr):
			// 业务规则错误（状态机违规/参数校验）按声明的业务码响应，不得落为 500
			common.Fail(c, bizErr.Code, bizErr.Message)
		case strings.Contains(err.Error(), "required"):
			common.ParamErrorWithErr(c, err, "请求参数错误")
		default:
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}
	common.Success(c, h.toDTO(updated))
}

// Delete 问题管理-删除问题
// @Summary 删除问题
// @Description 根据ID删除问题
// @Tags 问题管理
// @Produce json
// @Param id path int true "问题ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	err = h.service.Delete(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// GetStats 问题管理-获取问题统计
// @Summary 获取问题统计
// @Description 获取问题各状态数量统计
// @Tags 问题管理
// @Produce json
// @Success 200 {object} common.Response{data=dto.ProblemStatsResponse}
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	stats, err := h.service.GetStats(c.Request.Context(), tenantID.(int))
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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

// GetTrends handles GET /api/v1/problems/trend
// @Summary 获取问题趋势
// @Description 按日期范围获取问题创建趋势，默认最近6个月
// @Tags 问题管理
// @Produce json
// @Param startDate query string false "开始日期（YYYY-MM-DD）"
// @Param endDate query string false "结束日期（YYYY-MM-DD）"
// @Success 200 {object} common.Response{data=object}
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/trend [get]
func (h *Handler) GetTrends(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	startDateStr := c.DefaultQuery("startDate", "")
	endDateStr := c.DefaultQuery("endDate", "")

	now := time.Now()
	endDate := now
	startDate := now.AddDate(0, -6, 0)

	if startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = parsed
		}
	}
	if endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = parsed
		}
	}

	data, err := h.service.GetTrend(c.Request.Context(), tenantID.(int), startDate, endDate)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, data)
}

// GetHotspots handles GET /api/v1/problems/hotspots
// @Summary 获取问题热点
// @Description 按日期范围获取问题热点分析，默认最近3个月
// @Tags 问题管理
// @Produce json
// @Param startDate query string false "开始日期（YYYY-MM-DD）"
// @Param endDate query string false "结束日期（YYYY-MM-DD）"
// @Success 200 {object} common.Response{data=object}
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/hotspots [get]
func (h *Handler) GetHotspots(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	startDateStr := c.DefaultQuery("startDate", "")
	endDateStr := c.DefaultQuery("endDate", "")

	now := time.Now()
	endDate := now
	startDate := now.AddDate(0, -3, 0)

	if startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = parsed
		}
	}
	if endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = parsed
		}
	}

	data, err := h.service.GetHotspot(c.Request.Context(), tenantID.(int), startDate, endDate)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, data)
}

// GetProblemSLA handles GET /api/v1/problems/:id/sla
// @Summary 获取问题SLA
// @Description 获取问题SLA状态（当前问题暂未配置SLA，返回默认无SLA状态）
// @Tags 问题管理
// @Produce json
// @Param id path int true "问题ID"
// @Success 200 {object} common.Response{data=object}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id}/sla [get]
func (h *Handler) GetProblemSLA(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	_, err = h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if h.service.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}

	// Problems don't have SLA tracking in the current schema;
	// return a sensible default indicating no SLA configured.
	common.Success(c, gin.H{
		"slaStatus":          "none",
		"responseTimeUsed":   0,
		"resolutionTimeUsed": 0,
		"responseBreached":   false,
		"resolutionBreached": false,
	})
}

// GetProblemComments handles GET /api/v1/problems/:id/comments
// @Summary 获取问题评论
// @Description 获取问题评论列表（评论功能暂未实现，返回空列表）
// @Tags 问题管理
// @Produce json
// @Param id path int true "问题ID"
// @Success 200 {object} common.Response{data=object}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id}/comments [get]
func (h *Handler) GetProblemComments(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	_, err = h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if h.service.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}

	// Problem comments are not yet stored in a dedicated table;
	// return an empty list to satisfy the API contract.
	common.Success(c, gin.H{
		"comments": []interface{}{},
		"total":    0,
	})
}

// AddProblemComment handles POST /api/v1/problems/:id/comments
// @Summary 添加问题评论
// @Description 为问题添加评论（评论功能暂未实现，返回501错误）
// @Tags 问题管理
// @Accept json
// @Produce json
// @Param id path int true "问题ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/problems/{id}/comments [post]
func (h *Handler) AddProblemComment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	_, err = h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if h.service.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Problem not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}

	// Problem comments are not yet stored in a dedicated table.
	common.Fail(c, common.InternalErrorCode, "problem comments are not yet supported")
}
