package service_request

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/internal/domain/cmdb"
	"itsm-backend/internal/domain/service_catalog"

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
	// 1. Validate Service Catalog
	cat, err := s.scRepo.Get(ctx, catalogID)
	if err != nil {
		return nil, common.NewNotFoundError("Service Catalog not found")
	}
	if cat.TenantID != tenantID {
		return nil, common.NewNotFoundError("Service Catalog not found")
	}
	if cat.CloudServiceID > 0 && cat.CITypeID == 0 {
		return nil, common.NewBadRequestError("关联云服务时必须配置CI类型", nil)
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
		newReq.CIID = ciID
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

	ciName := strings.TrimSpace(reqData.Title)
	if ciName == "" {
		if nameVal, ok := reqData.FormData["name"].(string); ok {
			ciName = strings.TrimSpace(nameVal)
		}
	}
	if ciName == "" {
		if titleVal, ok := reqData.FormData["title"].(string); ok {
			ciName = strings.TrimSpace(titleVal)
		}
	}
	if ciName == "" {
		ciName = fmt.Sprintf("%s-%s", cat.Name, time.Now().Format("20060102150405"))
	}

	ci := &cmdb.ConfigurationItem{
		Name:            ciName,
		CITypeID:        cat.CITypeID,
		Status:          "active",
		Source:          "service_catalog",
		TenantID:        tenantID,
		CloudResourceID: parseStringField(reqData.FormData, "cloud_resource_id"),
		CloudAccountID:  parseStringField(reqData.FormData, "cloud_account_id"),
		CloudRegion:     parseStringField(reqData.FormData, "cloud_region"),
		CloudZone:       parseStringField(reqData.FormData, "cloud_zone"),
		CloudProvider:   parseStringField(reqData.FormData, "cloud_provider"),
	}

	if cloudResourceRefID > 0 {
		ci.CloudResourceRefID = cloudResourceRefID
	}

	if cat.CloudServiceID > 0 {
		cs, err := s.cmdbRepo.GetCloudService(ctx, tenantID, cat.CloudServiceID)
		if err != nil && !ent.IsNotFound(err) {
			return 0, common.NewInternalError("获取云服务目录失败", err)
		}
		if cs != nil {
			if ci.CloudProvider == "" {
				ci.CloudProvider = cs.Provider
			}
			if ci.CloudResourceType == "" {
				ci.CloudResourceType = cs.ResourceTypeCode
			}
		}
	}

	if ci.CloudResourceType == "" {
		ci.CloudResourceType = parseStringField(reqData.FormData, "cloud_resource_type")
	}

	created, err := s.cmdbRepo.CreateCI(ctx, ci)
	if err != nil {
		return 0, common.NewInternalError("创建关联CI失败", err)
	}
	return created.ID, nil
}

func parseStringField(formData map[string]interface{}, key string) string {
	if formData == nil {
		return ""
	}
	if v, ok := formData[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
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
	req, approvals, err := s.repo.GetWithApprovals(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if req.TenantID != tenantID {
		return nil, nil, common.NewNotFoundError("Service Request not found")
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
	req, approvals, err := s.repo.GetWithApprovals(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if req.TenantID != tenantID {
		return nil, nil, common.NewNotFoundError("Service Request not found")
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
	if err := s.checkEligibility(userRole, userDept, currentApproval.Step); err != nil {
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
		nextLevel = req.CurrentLevel + 1
	}

	// 6. Update Entities
	currentApproval.Status = status
	currentApproval.Action = action
	currentApproval.Comment = comment
	currentApproval.ApproverID = &actorID
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

func (s *Service) checkEligibility(actorRole, actorDept, step string) error {
	actorRole = strings.ToLower(actorRole)
	if actorRole == RoleAdmin || actorRole == RoleSuperAdmin {
		return nil
	}

	switch step {
	case ApprovalStepManager:
		if actorRole == RoleManager {
			// Missing department check against requester for now
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

	switch role {
	case ApprovalStepManager:
		targetLevel = 1
		requiredStatus = SRStatusSubmitted
	case ApprovalStepIT, RoleAgent, RoleTechnician:
		targetLevel = 2
		requiredStatus = SRStatusManagerApproved
	case ApprovalStepSecurity:
		targetLevel = 3
		requiredStatus = SRStatusITApproved
	case RoleAdmin, RoleSuperAdmin:
		targetLevel = 0
	}

	return s.repo.ListPendingApprovals(ctx, tenantID, targetLevel, requiredStatus, page, size)
}

func (s *Service) List(ctx context.Context, tenantID int, filters ListFilters) ([]*ServiceRequest, int, error) {
	return s.repo.List(ctx, tenantID, filters)
}
