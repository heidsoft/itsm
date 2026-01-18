package service_request

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/servicerequest"
	"itsm-backend/ent/servicerequestapproval"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

// toDomain converts Ent model to Domain entity
func (r *EntRepository) toDomain(req *ent.ServiceRequest) *ServiceRequest {
	if req == nil {
		return nil
	}
	return &ServiceRequest{
		ID:                 req.ID,
		TenantID:           req.TenantID,
		CatalogID:          req.CatalogID,
		RequesterID:        req.RequesterID,
		CiID:               req.CiID,
		Status:             req.Status,
		Title:              req.Title,
		Reason:             req.Reason,
		FormData:           req.FormData,
		CostCenter:         req.CostCenter,
		DataClassification: req.DataClassification,
		NeedsPublicIP:      req.NeedsPublicIP,
		SourceIPWhitelist:  req.SourceIPWhitelist,
		ExpireAt:           itemOrNil(req.ExpireAt),
		ComplianceAck:      req.ComplianceAck,
		CurrentLevel:       req.CurrentLevel,
		TotalLevels:        req.TotalLevels,
		CreatedAt:          req.CreatedAt,
		UpdatedAt:          req.UpdatedAt,
	}
}

func itemOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func itemIntOrNil(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// toDomainApproval converts Ent approval model to Domain entity
func (r *EntRepository) toDomainApproval(app *ent.ServiceRequestApproval) *ServiceRequestApproval {
	if app == nil {
		return nil
	}
	return &ServiceRequestApproval{
		ID:               app.ID,
		TenantID:         app.TenantID,
		ServiceRequestID: app.ServiceRequestID,
		Level:            app.Level,
		Step:             app.Step,
		Status:           app.Status,
		ApproverID:       app.ApproverID,
		ApproverName:     app.ApproverName,
		Comment:          app.Comment,
		Action:           app.Action,
		TimeoutHours:     app.TimeoutHours,
		DueAt:            itemOrNil(app.DueAt),
		ProcessedAt:      itemOrNil(app.ProcessedAt),
		CreatedAt:        app.CreatedAt,
		// UpdatedAt not in ent schema
	}
}

func (r *EntRepository) Create(ctx context.Context, req *ServiceRequest, approvals []*ServiceRequestApproval) (*ServiceRequest, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Create Service Request
	create := tx.ServiceRequest.Create().
		SetTenantID(req.TenantID).
		SetCatalogID(req.CatalogID).
		SetRequesterID(req.RequesterID).
		SetStatus(req.Status).
		SetCurrentLevel(req.CurrentLevel).
		SetTotalLevels(req.TotalLevels).
		SetComplianceAck(req.ComplianceAck).
		SetNeedsPublicIP(req.NeedsPublicIP).
		SetDataClassification(req.DataClassification)

	if req.Title != "" {
		create.SetTitle(req.Title)
	}
	if req.Reason != "" {
		create.SetReason(req.Reason)
	}
	if req.FormData != nil {
		create.SetFormData(req.FormData)
	}
	if req.CostCenter != "" {
		create.SetCostCenter(req.CostCenter)
	}
	if req.SourceIPWhitelist != nil {
		create.SetSourceIPWhitelist(req.SourceIPWhitelist)
	}
	if req.ExpireAt != nil {
		create.SetExpireAt(*req.ExpireAt)
	}
	if req.CiID > 0 {
		create.SetCiID(req.CiID)
	}

	savedReq, err := create.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating service request: %w", err)
	}

	// 2. Create Approvals
	if len(approvals) > 0 {
		bulk := make([]*ent.ServiceRequestApprovalCreate, len(approvals))
		for i, app := range approvals {
			bulk[i] = tx.ServiceRequestApproval.Create().
				SetTenantID(req.TenantID).
				SetServiceRequestID(savedReq.ID).
				SetLevel(app.Level).
				SetStep(app.Step).
				SetStatus(app.Status).
				SetTimeoutHours(app.TimeoutHours)

			if app.DueAt != nil {
				bulk[i].SetDueAt(*app.DueAt)
			}
		}
		if _, err := tx.ServiceRequestApproval.CreateBulk(bulk...).Save(ctx); err != nil {
			return nil, fmt.Errorf("creating approvals: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return r.toDomain(savedReq), nil
}

func (r *EntRepository) Get(ctx context.Context, id int) (*ServiceRequest, error) {
	req, err := r.client.ServiceRequest.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.toDomain(req), nil
}

func (r *EntRepository) GetWithApprovals(ctx context.Context, id int) (*ServiceRequest, []*ServiceRequestApproval, error) {
	req, err := r.client.ServiceRequest.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	approvals, err := r.client.ServiceRequestApproval.Query().
		Where(servicerequestapproval.ServiceRequestID(id)).
		Order(ent.Asc(servicerequestapproval.FieldLevel)).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("querying approvals: %w", err)
	}

	domainApprovals := make([]*ServiceRequestApproval, len(approvals))
	for i, app := range approvals {
		domainApprovals[i] = r.toDomainApproval(app)
	}

	return r.toDomain(req), domainApprovals, nil
}

func (r *EntRepository) List(ctx context.Context, tenantID int, filters ListFilters) ([]*ServiceRequest, int, error) {
	query := r.client.ServiceRequest.Query().
		Where(servicerequest.TenantID(tenantID))

	if filters.Status != "" {
		query.Where(servicerequest.Status(filters.Status))
	}
	if filters.UserID > 0 {
		query.Where(servicerequest.RequesterID(filters.UserID))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	if filters.Size > 0 {
		query.Limit(filters.Size).Offset((filters.Page - 1) * filters.Size)
	}

	// Default sort by CreatedAt DESC
	rows, err := query.Order(ent.Desc(servicerequest.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	results := make([]*ServiceRequest, len(rows))
	for i, row := range rows {
		results[i] = r.toDomain(row)
	}

	return results, total, nil
}

func (r *EntRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	return r.client.ServiceRequest.UpdateOneID(id).SetStatus(status).Exec(ctx)
}

func (r *EntRepository) GetApproval(ctx context.Context, requestID int, level int) (*ServiceRequestApproval, error) {
	app, err := r.client.ServiceRequestApproval.Query().
		Where(servicerequestapproval.ServiceRequestID(requestID)).
		Where(servicerequestapproval.Level(level)).
		Where(servicerequestapproval.Status("pending")). // Ideally we query by ID or just logic, but this matches logic
		First(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomainApproval(app), nil
}

func (r *EntRepository) UpdateApproval(ctx context.Context, approval *ServiceRequestApproval) error {
	update := r.client.ServiceRequestApproval.UpdateOneID(approval.ID).
		SetStatus(approval.Status).
		SetAction(approval.Action).
		SetComment(approval.Comment)

	if approval.ApproverID != nil {
		update.SetApproverID(*approval.ApproverID)
	}
	if approval.ApproverName != "" {
		update.SetApproverName(approval.ApproverName)
	}
	if approval.ProcessedAt != nil {
		update.SetProcessedAt(*approval.ProcessedAt)
	}

	return update.Exec(ctx)
}

func (r *EntRepository) UpdateRequestAndApproval(ctx context.Context, req *ServiceRequest, approval *ServiceRequestApproval) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update Request
	reqUpdate := tx.ServiceRequest.UpdateOneID(req.ID).
		SetStatus(req.Status).
		SetCurrentLevel(req.CurrentLevel)
	if err := reqUpdate.Exec(ctx); err != nil {
		return err
	}

	// Update Approval
	appUpdate := tx.ServiceRequestApproval.UpdateOneID(approval.ID).
		SetStatus(approval.Status).
		SetAction(approval.Action).
		SetComment(approval.Comment)

	if approval.ApproverID != nil {
		appUpdate.SetApproverID(*approval.ApproverID)
	}
	if approval.ApproverName != "" {
		appUpdate.SetApproverName(approval.ApproverName)
	}
	if approval.ProcessedAt != nil {
		appUpdate.SetProcessedAt(*approval.ProcessedAt)
	}
	if err := appUpdate.Exec(ctx); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *EntRepository) ListPendingApprovals(ctx context.Context, tenantID int, targetLevel int, requiredStatus string, page, size int) ([]*ServiceRequest, int, error) {
	query := r.client.ServiceRequest.Query().
		Where(servicerequest.TenantID(tenantID))

	if targetLevel > 0 {
		query = query.Where(
			servicerequest.CurrentLevel(targetLevel),
			servicerequest.Status(requiredStatus),
		)
	} else {
		// Admin/Super Admin: See all active pending states
		// This logic mimics the original service
		query = query.Where(servicerequest.StatusIn(
			"submitted",
			"manager_approved",
			"it_approved",
		))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := query.
		Order(ent.Desc(servicerequest.FieldCreatedAt)).
		Offset((page - 1) * size).
		Limit(size).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	results := make([]*ServiceRequest, len(rows))
	for i, row := range rows {
		results[i] = r.toDomain(row)
	}
	return results, total, nil
}

// Helper to get User department (needed for filtering)
// Note: This leaks abstraction slightly by querying User, but practical.
func (r *EntRepository) GetUserDepartment(ctx context.Context, userID int) (string, error) {
	u, err := r.client.User.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	return u.Department, nil
}
