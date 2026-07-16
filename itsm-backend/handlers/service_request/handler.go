package service_request

import (
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func failServiceRequest(c *gin.Context, err error) {
	if appErr, ok := common.AsAppError(err); ok {
		switch appErr.Code {
		case common.ErrCodeBadRequest, common.ErrCodeValidation:
			common.Fail(c, common.ParamErrorCode, appErr.Message)
		case common.ErrCodeUnauthorized:
			common.Fail(c, common.UnauthorizedCode, appErr.Message)
		case common.ErrCodeForbidden:
			common.Fail(c, common.ForbiddenErrorCode, appErr.Message)
		case common.ErrCodeNotFound:
			common.Fail(c, common.NotFoundErrorCode, appErr.Message)
		case common.ErrCodeConflict:
			common.Fail(c, common.ConflictCode, appErr.Error())
		default:
			common.Fail(c, common.InternalErrorCode, appErr.Message)
		}
		return
	}
	if ent.IsNotFound(err) {
		common.Fail(c, common.NotFoundErrorCode, "Service request not found")
		return
	}
	common.Fail(c, common.InternalErrorCode, err.Error())
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
		CIID:               req.CiID,
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
		Version:            req.Version,
		ProcessorID:        req.ProcessorID,
		ApprovedAt:         req.ApprovedAt,
		StartedAt:          req.StartedAt,
		CompletedAt:        req.CompletedAt,
		CompletionNote:     req.CompletionNote,
		LastError:          req.LastError,
		CreatedAt:          req.CreatedAt,
		UpdatedAt:          req.UpdatedAt,
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
	normalizeCreateServiceRequest(&req)
	if req.CatalogID == 0 {
		common.Fail(c, 1001, "catalogId is required")
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
		failServiceRequest(c, err)
		return
	}

	fullReq, approvals, err := h.service.Get(c.Request.Context(), created.ID, tenantID.(int))
	if err != nil {
		h.service.logger.Errorw("Create: failed to get created service request", "error", err, "id", created.ID)
		// Return the created object even if Get fails - created.ID is valid
		common.Success(c, h.toDTO(created, nil))
		return
	}
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
		Status: normalizeServiceRequestStatus(req.Status),
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
		failServiceRequest(c, err)
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
		failServiceRequest(c, err)
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

func (h *Handler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "Invalid ID")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "Tenant ID missing")
		return
	}

	var req dto.UpdateServiceRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "Invalid parameters: "+err.Error())
		return
	}
	normalizeUpdateServiceRequest(&req)

	domainReq := &ServiceRequest{
		Title:              req.Title,
		Reason:             req.Reason,
		FormData:           req.FormData,
		CostCenter:         req.CostCenter,
		DataClassification: req.DataClassification,
		NeedsPublicIPSet:   req.NeedsPublicIP != nil,
		SourceIPWhitelist:  req.SourceIPWhitelist,
		ComplianceAckSet:   req.ComplianceAck != nil,
		ExpireAt:           req.ExpireAt,
	}
	if req.NeedsPublicIP != nil {
		domainReq.NeedsPublicIP = *req.NeedsPublicIP
	}
	if req.ComplianceAck != nil {
		domainReq.ComplianceAck = *req.ComplianceAck
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")
	updated, err := h.service.Update(c.Request.Context(), id, tenantID.(int), userID, role, domainReq)
	if err != nil {
		failServiceRequest(c, err)
		return
	}

	fullReq, approvals, _ := h.service.Get(c.Request.Context(), updated.ID, tenantID.(int))
	common.Success(c, h.toDTO(fullReq, approvals))
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "Invalid ID")
		return
	}

	var req dto.UpdateServiceRequestStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "Invalid parameters: "+err.Error())
		return
	}
	status := normalizeServiceRequestStatus(req.Status)
	if status == "" {
		common.Fail(c, 1001, "status is required")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "Tenant ID missing")
		return
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")
	if err := h.service.UpdateStatus(c.Request.Context(), id, tenantID.(int), userID, role, status); err != nil {
		failServiceRequest(c, err)
		return
	}

	fullReq, approvals, err := h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Success(c, gin.H{"id": id, "status": status})
		return
	}
	common.Success(c, h.toDTO(fullReq, approvals))
}

func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "Invalid ID")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "Tenant ID missing")
		return
	}

	err = h.service.Delete(c.Request.Context(), id, tenantID.(int), c.GetInt("user_id"), c.GetString("role"))
	if err != nil {
		failServiceRequest(c, err)
		return
	}

	common.Success(c, nil)
}

func normalizeCreateServiceRequest(req *dto.CreateServiceRequestRequest) {
	if req.FormData == nil {
		req.FormData = map[string]any{}
	}
	if req.Title == "" {
		if title, ok := req.FormData["title"].(string); ok {
			req.Title = title
		}
	}
	if req.Reason == "" {
		if reason, ok := req.FormData["reason"].(string); ok {
			req.Reason = reason
		}
	}
	if req.CostCenter == "" {
		if costCenter, ok := req.FormData["cost_center"].(string); ok {
			req.CostCenter = costCenter
		}
	}
	if req.DataClassification == "" {
		if classification, ok := req.FormData["data_classification"].(string); ok {
			req.DataClassification = classification
		}
	}
	if req.DataClassification == "" {
		req.DataClassification = "internal"
	}
	if len(req.SourceIPWhitelist) == 0 {
		if whitelist, ok := req.FormData["source_ip_whitelist"].([]string); ok {
			req.SourceIPWhitelist = whitelist
		}
	}
	if req.ExpireAt == nil {
		if expireAt, ok := req.FormData["expire_at"].(string); ok {
			if parsed, err := time.Parse(time.RFC3339, expireAt); err == nil {
				req.ExpireAt = &parsed
			}
		}
	}
	if req.ExpireAt == nil {
		defaultExpireAt := time.Now().Add(30 * 24 * time.Hour)
		req.ExpireAt = &defaultExpireAt
	}
	if ack, ok := req.FormData["compliance_ack"].(bool); ok {
		req.ComplianceAck = ack
	}
}

func normalizeUpdateServiceRequest(req *dto.UpdateServiceRequestRequest) {
}

func normalizeServiceRequestStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "pending_approval", "pending":
		return "submitted"
	case "approved":
		return "security_approved"
	case "in_progress":
		return "provisioning"
	case "completed":
		return "delivered"
	default:
		return strings.TrimSpace(status)
	}
}
