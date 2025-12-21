package service_request

import (
	"strconv"
	"strings"

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

// Map Domain to DTO
func (h *Handler) toDTO(req *ServiceRequest, approvals []*ServiceRequestApproval) *dto.ServiceRequestResponse {
	if req == nil {
		return nil
	}
	resp := &dto.ServiceRequestResponse{
		ID:                 req.ID,
		CatalogID:          req.CatalogID,
		RequesterID:        req.RequesterID,
		Status:             req.Status,
		Title:              req.Title,
		Reason:             req.Reason,
		FormData:           req.FormData,
		CostCenter:         req.CostCenter,
		DataClassification: req.DataClassification,
		NeedsPublicIP:      req.NeedsPublicIP,
		SourceIPWhitelist:  req.SourceIPWhitelist,
		ComplianceAck:      req.ComplianceAck,
		CurrentLevel:       req.CurrentLevel,
		TotalLevels:        req.TotalLevels,
		CreatedAt:          req.CreatedAt,
	}
	if req.ExpireAt != nil {
		t := *req.ExpireAt
		resp.ExpireAt = &t
	}

	if approvals != nil {
		resp.Approvals = make([]dto.ServiceRequestApprovalResponse, len(approvals))
		for i, app := range approvals {
			resp.Approvals[i] = dto.ServiceRequestApprovalResponse{
				ID:               app.ID,
				ServiceRequestID: app.ServiceRequestID,
				Level:            app.Level,
				Step:             app.Step,
				Status:           app.Status,
				ApproverName:     app.ApproverName,
				Comment:          app.Comment,
				Action:           app.Action,
				TimeoutHours:     app.TimeoutHours,
			}
			if app.DueAt != nil {
				t := *app.DueAt
				resp.Approvals[i].DueAt = &t
			}
			if app.ProcessedAt != nil {
				t := *app.ProcessedAt
				resp.Approvals[i].ProcessedAt = &t
			}
			if app.ApproverID != nil {
				resp.Approvals[i].ApproverID = app.ApproverID
			}
		}
	}
	return resp
}

func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateServiceRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "Invalid parameters: "+err.Error())
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "Tenant ID missing")
		return
	}
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, 2001, "User ID missing")
		return
	}

	expireAt := req.ExpireAt

	domainReq := &ServiceRequest{
		ComplianceAck:      req.ComplianceAck,
		NeedsPublicIP:      req.NeedsPublicIP,
		DataClassification: req.DataClassification,
		Title:              req.Title,
		Reason:             req.Reason,
		FormData:           req.FormData,
		CostCenter:         req.CostCenter,
		SourceIPWhitelist:  req.SourceIPWhitelist,
		ExpireAt:           expireAt,
	}

	created, err := h.service.Create(c.Request.Context(), tenantID.(int), userID.(int), req.CatalogID, domainReq)
	if err != nil {
		// Error mapping could be improved
		common.Fail(c, 5001, err.Error())
		return
	}

	fullReq, approvals, _ := h.service.Get(c.Request.Context(), created.ID, tenantID.(int))
	common.Success(c, h.toDTO(fullReq, approvals))
}

func (h *Handler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "Invalid ID")
		return
	}
	tenantID, _ := c.Get("tenant_id")

	req, approvals, err := h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, 404, "Not Found")
		} else if common.IsAppError(err) {
			common.Fail(c, 5001, err.Error())
		} else {
			common.Fail(c, 5001, err.Error())
		}
		return
	}
	common.Success(c, h.toDTO(req, approvals))
}

func (h *Handler) List(c *gin.Context) {
	var req dto.GetServiceRequestsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, 1001, "Invalid parameters")
		return
	}
	tenantID, _ := c.Get("tenant_id")

	// If listing "me", we need user ID
	userID := 0
	if c.Request.URL.Path == "/me" || c.Query("scope") == "me" {
		uid, exists := c.Get("user_id")
		if exists {
			userID = uid.(int)
		}
	}
	// For compatibility with legacy controller which injects UserID from token into DTO if needed
	if req.UserID == 0 && (c.Request.URL.Path == "/api/v1/service-requests/me" || strings.Contains(c.Request.URL.Path, "/me")) {
		uid, _ := c.Get("user_id")
		userID = uid.(int)
	}

	filters := ListFilters{
		Status: req.Status,
		UserID: userID,
		Page:   req.Page,
		Size:   req.Size,
	}
	if filters.Page == 0 {
		filters.Page = 1
	}
	if filters.Size == 0 {
		filters.Size = 10
	}

	list, total, err := h.service.List(c.Request.Context(), tenantID.(int), filters)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	dtos := make([]dto.ServiceRequestResponse, len(list))
	for i, v := range list {
		dtos[i] = *h.toDTO(v, nil)
	}

	common.Success(c, map[string]interface{}{
		"requests": dtos,
		"total":    total,
		"page":     filters.Page,
		"size":     filters.Size,
	})
}

func (h *Handler) ApplyApproval(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "Invalid ID")
		return
	}

	var req dto.ServiceRequestApprovalActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	dept, _ := c.Get("department")

	roleStr, _ := role.(string)
	deptStr, _ := dept.(string)

	res, approvals, err := h.service.ApplyApproval(c.Request.Context(), id, tenantID.(int), userID.(int), req.Action, req.Comment, roleStr, deptStr)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, h.toDTO(res, approvals))
}

func (h *Handler) ListPending(c *gin.Context) {
	var req dto.GetServiceRequestsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, 1001, "Invalid parameters")
		return
	}
	tenantID, _ := c.Get("tenant_id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	roleStr, _ := role.(string)

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 10
	}

	list, total, err := h.service.ListPendingApprovals(c.Request.Context(), tenantID.(int), userID.(int), roleStr, req.Page, req.Size)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	dtos := make([]dto.ServiceRequestResponse, len(list))
	for i, v := range list {
		// ListPendingApprovals in Service returns []*ServiceRequest.
		// Approvals are pre-loaded? Repository implementation says:
		// return list, total, nil
		// It uses request.QueryApprovals()... but we need to check if toDTO handles nil approvals gracefully (it currently allocates generic slice if not nil, else leaves field empty).
		// For pending list, likely we want to see approvals?
		// The Service implementation of ListPendingApprovals calls repo.ListPendingApprovals.
		// Let's assume for now we return the request details.
		dtos[i] = *h.toDTO(v, nil)
	}

	common.Success(c, map[string]interface{}{
		"requests": dtos,
		"total":    total,
		"page":     req.Page,
		"size":     req.Size,
	})
}
