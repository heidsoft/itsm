package service_request

import (
	"context"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/handlers/cmdb"
	"itsm-backend/handlers/service_catalog"

	"go.uber.org/zap"
)

// Constants for approval steps
const (
	ApprovalStepManager  = "manager"
	ApprovalStepIT       = "it"
	ApprovalStepSecurity = "security"

	SRStatusSubmitted        = "submitted"
	SRStatusManagerApproved  = "manager_approved"
	SRStatusITApproved       = "it_approved"
	SRStatusSecurityApproved = "security_approved"
	SRStatusRejected         = "rejected"
	SRStatusProvisioning     = "provisioning"
	SRStatusDelivered        = "delivered"
	SRStatusFailed           = "failed"
	SRStatusCancelled        = "cancelled"

	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
	ApprovalStatusRejected = "rejected"

	// V1 审批时限配置（小时）
	ApprovalTimeoutManager  = 24
	ApprovalTimeoutIT       = 48
	ApprovalTimeoutSecurity = 72

	// Roles
	RoleAdmin      = "admin"
	RoleSuperAdmin = "super_admin"
	RoleManager    = "manager"
	RoleAgent      = "agent"
	RoleTechnician = "technician"
	RoleSecurity   = "security"
)

type Service struct {
	repo     Repository
	scRepo   service_catalog.Repository
	cmdbRepo cmdb.Repository
	logger   *zap.SugaredLogger
}

func NewService(repo Repository, scRepo service_catalog.Repository, cmdbRepo cmdb.Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:     repo,
		scRepo:   scRepo,
		cmdbRepo: cmdbRepo,
		logger:   logger,
	}
}

// Create submits a new service request
func (s *Service) Create(ctx context.Context, tenantID, requesterID int, catalogID int, reqData *ServiceRequest) (*ServiceRequest, error) {
	if _, _, err := s.repo.GetUserContext(ctx, requesterID, tenantID); err != nil {
		return nil, common.NewBadRequestError("Requester not found or inactive", err)
	}
	// 1. Validate Service Catalog
	cat, err := s.scRepo.Get(ctx, tenantID, catalogID)
	if err != nil {
		return nil, common.NewNotFoundError("Service Catalog not found")
	}
	if cat.CloudServiceID > 0 && cat.CITypeID == 0 {
		return nil, common.NewBadRequestError("关联云服务时必须配置CI类型", nil)
	}
	if cat.Status != "enabled" && cat.Status != "active" {
		return nil, common.NewBadRequestError("Service Catalog is not enabled", nil)
	}

	// 2. Validate Request Data
	if !reqData.ComplianceAck {
		return nil, common.NewBadRequestError("Compliance acknowledgement required", nil)
	}
	if reqData.NeedsPublicIP && len(reqData.SourceIPWhitelist) == 0 {
		return nil, common.NewBadRequestError("Source IP whitelist required for public IP", nil)
	}
	if reqData.ExpireAt == nil {
		return nil, common.NewBadRequestError("Expiration date required", nil)
	}
	if !reqData.ExpireAt.After(time.Now()) {
		return nil, common.NewBadRequestError("Expiration date must be in the future", nil)
	}
	if strings.TrimSpace(reqData.Title) == "" {
		return nil, common.NewBadRequestError("Request title is required", nil)
	}
	switch reqData.DataClassification {
	case "public", "internal", "confidential", "restricted":
	default:
		return nil, common.NewBadRequestError("Invalid data classification", nil)
	}

	// 3. Prepare Service Request
	newReq := &ServiceRequest{
		TenantID:           tenantID,
		CatalogID:          catalogID,
		RequesterID:        requesterID,
		Status:             SRStatusSubmitted,
		CurrentLevel:       1,
		TotalLevels:        3,
		ComplianceAck:      reqData.ComplianceAck,
		NeedsPublicIP:      reqData.NeedsPublicIP,
		DataClassification: reqData.DataClassification,
		Title:              reqData.Title,
		Reason:             reqData.Reason,
		FormData:           reqData.FormData,
		CostCenter:         reqData.CostCenter,
		SourceIPWhitelist:  reqData.SourceIPWhitelist,
		ExpireAt:           reqData.ExpireAt,
	}

	if cat.CITypeID > 0 {
		ciID, err := s.ensureLinkedCI(ctx, tenantID, cat, reqData)
		if err != nil {
			return nil, err
		}
		newReq.CiID = ciID
	}

	// 4. Create Approval Steps
	steps := []struct {
		level        int
		step         string
		timeoutHours int
	}{
		{1, ApprovalStepManager, ApprovalTimeoutManager},
		{2, ApprovalStepIT, ApprovalTimeoutIT},
		{3, ApprovalStepSecurity, ApprovalTimeoutSecurity},
	}

	approvals := make([]*ServiceRequestApproval, len(steps))
	now := time.Now()
	for i, st := range steps {
		dueAt := now.Add(time.Duration(st.timeoutHours) * time.Hour)
		approvals[i] = &ServiceRequestApproval{
			TenantID:     tenantID,
			Level:        st.level,
			Step:         st.step,
			Status:       ApprovalStatusPending,
			TimeoutHours: st.timeoutHours,
			DueAt:        &dueAt,
		}
	}

	// 5. Save
	created, err := s.repo.Create(ctx, newReq, approvals)
	if err != nil {
		s.logger.Errorw("Failed to create service request", "error", err)
		return nil, common.NewInternalError("Failed to create service request", err)
	}

	return created, nil
}

func (s *Service) ensureLinkedCI(ctx context.Context, tenantID int, cat *service_catalog.ServiceCatalog, reqData *ServiceRequest) (int, error) {
	_ = cat
	cloudResourceRefID := parseIntField(reqData.FormData, "cloud_resource_ref_id")
	if cloudResourceRefID > 0 {
		existing, err := s.cmdbRepo.GetCIByCloudResourceRefID(ctx, tenantID, cloudResourceRefID)
		if err == nil && existing != nil {
			return existing.ID, nil
		}
		if err != nil && !ent.IsNotFound(err) {
			return 0, common.NewInternalError("查询关联CI失败", err)
		}
	}
	// 新 CI 必须在审批完成后的 provisioning 阶段由履约器/连接器创建，
	// 提交申请时不能提前向 CMDB 写入 active 资产。
	return 0, nil
}

func parseIntField(formData map[string]interface{}, key string) int {
	if formData == nil {
		return 0
	}
	switch v := formData[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return 0
		}
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return 0
}

// Get retrieves a service request with approvals
func (s *Service) Get(ctx context.Context, id, tenantID int) (*ServiceRequest, []*ServiceRequestApproval, error) {
	req, approvals, err := s.repo.GetWithApprovals(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}
	return req, approvals, nil
}

// ApplyApproval processes an approval action
func (s *Service) ApplyApproval(ctx context.Context, id, tenantID, actorID int, action, comment string, userRole, userDept string) (*ServiceRequest, []*ServiceRequestApproval, error) {
	// 1. Validate Inputs
	if action != "approve" && action != "reject" {
		return nil, nil, common.NewBadRequestError("Invalid action: "+action, nil)
	}
	if action == "reject" && strings.TrimSpace(comment) == "" {
		return nil, nil, common.NewBadRequestError("Comment required for rejection", nil)
	}

	// 2. Get Request
	req, approvals, err := s.repo.GetWithApprovals(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}
	actorDept, actorName, err := s.repo.GetUserContext(ctx, actorID, tenantID)
	if err != nil {
		return nil, nil, common.NewForbiddenError("Approver not found or inactive")
	}
	if actorID == req.RequesterID && !isServiceRequestAdmin(userRole) {
		return nil, nil, common.NewForbiddenError("Requesters cannot approve their own service requests")
	}

	// 3. Find Pending Approval for Current Level
	var currentApproval *ServiceRequestApproval
	for _, app := range approvals {
		if app.Level == req.CurrentLevel && app.Status == ApprovalStatusPending {
			currentApproval = app
			break
		}
	}
	if currentApproval == nil {
		return nil, nil, common.NewConflictError("No pending approval found for current level", "")
	}

	// 4. Check Permissions
	requesterDept, _, err := s.repo.GetUserContext(ctx, req.RequesterID, tenantID)
	if err != nil {
		return nil, nil, common.NewConflictError("Requester is no longer available", "")
	}
	if actorDept == "" {
		actorDept = userDept
	}
	if err := s.checkEligibility(userRole, actorDept, requesterDept, currentApproval.Step); err != nil {
		return nil, nil, err
	}

	// 5. Process
	now := time.Now()
	status := ApprovalStatusApproved
	nextReqStatus := req.Status
	nextLevel := req.CurrentLevel

	if action == "reject" {
		status = ApprovalStatusRejected
		nextReqStatus = SRStatusRejected
	} else {
		// Approve logic
		switch currentApproval.Step {
		case ApprovalStepManager:
			nextReqStatus = SRStatusManagerApproved
		case ApprovalStepIT:
			nextReqStatus = SRStatusITApproved
		case ApprovalStepSecurity:
			nextReqStatus = SRStatusSecurityApproved
		}
		if req.CurrentLevel < req.TotalLevels {
			nextLevel = req.CurrentLevel + 1
		}
	}

	// 6. Update Entities
	currentApproval.Status = status
	currentApproval.Action = action
	currentApproval.Comment = comment
	currentApproval.ApproverID = &actorID
	currentApproval.ApproverName = actorName
	currentApproval.ProcessedAt = &now

	req.Status = nextReqStatus
	if action == "approve" {
		req.CurrentLevel = nextLevel
	}

	if err := s.repo.UpdateRequestAndApproval(ctx, req, currentApproval); err != nil {
		return nil, nil, common.NewDatabaseError("Failed to update request", err)
	}

	return s.Get(ctx, id, tenantID)
}

func (s *Service) checkEligibility(actorRole, actorDept, requesterDept, step string) error {
	actorRole = strings.ToLower(actorRole)
	if actorRole == RoleAdmin || actorRole == RoleSuperAdmin {
		return nil
	}

	switch step {
	case ApprovalStepManager:
		if actorRole == RoleManager && actorDept != "" && strings.EqualFold(actorDept, requesterDept) {
			return nil
		}
	case ApprovalStepIT:
		if actorRole == RoleAgent || actorRole == RoleTechnician {
			return nil
		}
	case ApprovalStepSecurity:
		if actorRole == RoleSecurity {
			return nil
		}
	}
	return common.NewForbiddenError("Permission denied for this approval step")
}

// ListPendingApprovals lists pending approvals for current user
func (s *Service) ListPendingApprovals(ctx context.Context, tenantID, userID int, role string, page, size int) ([]*ServiceRequest, int, error) {
	role = strings.ToLower(role)
	var targetLevel int
	var requiredStatus string
	var requesterDept string
	actorDept, _, err := s.repo.GetUserContext(ctx, userID, tenantID)
	if err != nil {
		return nil, 0, common.NewForbiddenError("Approver not found or inactive")
	}

	switch role {
	case ApprovalStepManager:
		targetLevel = 1
		requiredStatus = SRStatusSubmitted
		requesterDept = actorDept
	case ApprovalStepIT, RoleAgent, RoleTechnician:
		targetLevel = 2
		requiredStatus = SRStatusManagerApproved
	case ApprovalStepSecurity:
		targetLevel = 3
		requiredStatus = SRStatusITApproved
	case RoleAdmin, RoleSuperAdmin:
		targetLevel = 0
	default:
		return nil, 0, common.NewForbiddenError("Role is not eligible to approve service requests")
	}

	return s.repo.ListPendingApprovals(ctx, tenantID, targetLevel, requiredStatus, requesterDept, page, size)
}

func (s *Service) List(ctx context.Context, tenantID int, filters ListFilters) ([]*ServiceRequest, int, error) {
	return s.repo.List(ctx, tenantID, filters)
}

// Update updates a service request
func (s *Service) Update(ctx context.Context, id, tenantID, actorID int, actorRole string, reqData *ServiceRequest) (*ServiceRequest, error) {
	// 1. Get existing request
	req, _, err := s.repo.GetWithApprovals(ctx, id, tenantID)
	if err != nil {
		return nil, common.NewNotFoundError("Service Request not found")
	}
	if req.Status != SRStatusSubmitted {
		return nil, common.NewConflictError("Only submitted requests can be edited", req.Status)
	}
	if actorID != req.RequesterID && !isServiceRequestAdmin(actorRole) {
		return nil, common.NewForbiddenError("Only the requester or an administrator can edit this request")
	}

	// 2. Update fields
	if reqData.Title != "" {
		req.Title = reqData.Title
	}
	if reqData.Reason != "" {
		req.Reason = reqData.Reason
	}
	if reqData.FormData != nil {
		req.FormData = reqData.FormData
	}
	if reqData.CostCenter != "" {
		req.CostCenter = reqData.CostCenter
	}
	if reqData.DataClassification != "" {
		req.DataClassification = reqData.DataClassification
	}
	if reqData.NeedsPublicIPSet {
		req.NeedsPublicIP = reqData.NeedsPublicIP
	}
	if reqData.SourceIPWhitelist != nil {
		req.SourceIPWhitelist = reqData.SourceIPWhitelist
	}
	if reqData.ExpireAt != nil {
		req.ExpireAt = reqData.ExpireAt
	}
	if reqData.ComplianceAckSet {
		req.ComplianceAck = reqData.ComplianceAck
	}

	// 3. Save
	if err := s.repo.Update(ctx, req); err != nil {
		s.logger.Errorw("Failed to update service request", "error", err)
		return nil, common.NewInternalError("Failed to update service request", err)
	}

	return req, nil
}

// UpdateStatus updates a service request status with tenant isolation.
func (s *Service) UpdateStatus(ctx context.Context, id, tenantID, actorID int, actorRole, status string) error {
	req, _, err := s.repo.GetWithApprovals(ctx, id, tenantID)
	if err != nil {
		return common.NewNotFoundError("Service Request not found")
	}
	if strings.TrimSpace(status) == "" {
		return common.NewBadRequestError("Status is required", nil)
	}
	if !isValidServiceRequestOperationalTransition(req.Status, status) {
		return common.NewConflictError("Invalid service request status transition", req.Status+" -> "+status)
	}
	if status == SRStatusCancelled {
		if actorID != req.RequesterID && !isServiceRequestAdmin(actorRole) {
			return common.NewForbiddenError("Only the requester or an administrator can cancel this request")
		}
	} else if !isServiceRequestOperator(actorRole) {
		return common.NewForbiddenError("Only service operators can update fulfillment status")
	}
	if err := s.repo.UpdateStatus(ctx, req, status, actorID); err != nil {
		s.logger.Errorw("Failed to update service request status", "error", err, "id", id, "status", status)
		return common.NewInternalError("Failed to update service request status", err)
	}
	return nil
}

// Delete deletes a service request
func (s *Service) Delete(ctx context.Context, id, tenantID, actorID int, actorRole string) error {
	// 1. Get existing request
	req, _, err := s.repo.GetWithApprovals(ctx, id, tenantID)
	if err != nil {
		return common.NewNotFoundError("Service Request not found")
	}
	if actorID != req.RequesterID && !isServiceRequestAdmin(actorRole) {
		return common.NewForbiddenError("Only the requester or an administrator can delete this request")
	}
	if req.Status != SRStatusSubmitted && req.Status != SRStatusRejected && req.Status != SRStatusCancelled {
		return common.NewConflictError("Only submitted, rejected, or cancelled requests can be deleted", req.Status)
	}

	// 2. Delete
	if err := s.repo.Delete(ctx, req); err != nil {
		s.logger.Errorw("Failed to delete service request", "error", err)
		return common.NewInternalError("Failed to delete service request", err)
	}

	return nil
}

func isServiceRequestAdmin(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	return role == RoleAdmin || role == RoleSuperAdmin
}

func isServiceRequestOperator(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	return isServiceRequestAdmin(role) || role == RoleAgent || role == RoleTechnician
}

func isValidServiceRequestOperationalTransition(current, next string) bool {
	if current == next {
		return true
	}
	transitions := map[string]map[string]struct{}{
		SRStatusSubmitted:        {SRStatusCancelled: {}},
		SRStatusManagerApproved:  {SRStatusCancelled: {}},
		SRStatusITApproved:       {SRStatusCancelled: {}},
		SRStatusSecurityApproved: {SRStatusProvisioning: {}, SRStatusCancelled: {}},
		SRStatusProvisioning:     {SRStatusDelivered: {}, SRStatusFailed: {}},
		SRStatusFailed:           {SRStatusProvisioning: {}, SRStatusCancelled: {}},
		SRStatusRejected:         {},
		SRStatusDelivered:        {},
		SRStatusCancelled:        {},
	}
	allowed, ok := transitions[current]
	if !ok {
		return false
	}
	_, ok = allowed[next]
	return ok
}
