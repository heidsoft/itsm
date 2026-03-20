package common

import (
	"context"

	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/department"
	"itsm-backend/ent/tag"
	"itsm-backend/ent/team"
	"itsm-backend/ent/user"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

// Mappings

func toUserDomain(e *ent.User) *User {
	if e == nil {
		return nil
	}
	return &User{
		ID:           e.ID,
		Username:     e.Username,
		Email:        e.Email,
		Name:         e.Name,
		Role:         string(e.Role),
		Department:   e.Department,
		DepartmentID: e.DepartmentID,
		Phone:        e.Phone,
		Active:       e.Active,
		TenantID:     e.TenantID,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

func toDeptDomain(e *ent.Department) *Department {
	if e == nil {
		return nil
	}
	d := &Department{
		ID:          e.ID,
		Name:        e.Name,
		Code:        e.Code,
		Description: e.Description,
		ManagerID:   e.ManagerID,
		ParentID:    e.ParentID,
		TenantID:    e.TenantID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
	for _, child := range e.Edges.Children {
		d.Children = append(d.Children, toDeptDomain(child))
	}
	return d
}

func toTeamDomain(e *ent.Team) *Team {
	if e == nil {
		return nil
	}
	return &Team{
		ID:          e.ID,
		Name:        e.Name,
		Code:        e.Code,
		Description: e.Description,
		Status:      e.Status,
		ManagerID:   e.ManagerID,
		TenantID:    e.TenantID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func toTagDomain(e *ent.Tag) *Tag {
	if e == nil {
		return nil
	}
	return &Tag{
		ID:          e.ID,
		Name:        e.Name,
		Code:        e.Code,
		Description: e.Description,
		Color:       e.Color,
		TenantID:    e.TenantID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func toAuditLogDomain(e *ent.AuditLog) *AuditLog {
	if e == nil {
		return nil
	}
	body := ""
	if e.RequestBody != nil {
		body = *e.RequestBody
	}
	return &AuditLog{
		ID:          e.ID,
		CreatedAt:   e.CreatedAt,
		TenantID:    e.TenantID,
		UserID:      e.UserID,
		RequestID:   e.RequestID,
		IP:          e.IP,
		Resource:    e.Resource,
		Action:      e.Action,
		Path:        e.Path,
		Method:      e.Method,
		StatusCode:  e.StatusCode,
		RequestBody: body,
	}
}

// User methods

func (r *EntRepository) GetUserByUsername(ctx context.Context, username string, tenantID int) (*User, error) {
	e, err := r.client.User.Query().
		Where(user.Username(username), user.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toUserDomain(e), nil
}

func (r *EntRepository) GetUserByID(ctx context.Context, id int) (*User, error) {
	e, err := r.client.User.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return toUserDomain(e), nil
}

func (r *EntRepository) ListUsers(ctx context.Context, tenantID int) ([]*User, error) {
	es, err := r.client.User.Query().Where(user.TenantID(tenantID)).All(ctx)
	if err != nil {
		return nil, err
	}
	var res []*User
	for _, e := range es {
		res = append(res, toUserDomain(e))
	}
	return res, nil
}

func (r *EntRepository) CreateUser(ctx context.Context, u *User) (*User, error) {
	e, err := r.client.User.Create().
		SetUsername(u.Username).
		SetEmail(u.Email).
		SetName(u.Name).
		SetRole(user.Role(u.Role)).
		SetDepartment(u.Department).
		SetDepartmentID(u.DepartmentID).
		SetPhone(u.Phone).
		SetPasswordHash(""). // Service will set this
		SetTenantID(u.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toUserDomain(e), nil
}

func (r *EntRepository) UpdateUser(ctx context.Context, u *User) (*User, error) {
	e, err := r.client.User.UpdateOneID(u.ID).
		SetEmail(u.Email).
		SetName(u.Name).
		SetRole(user.Role(u.Role)).
		SetDepartment(u.Department).
		SetDepartmentID(u.DepartmentID).
		SetPhone(u.Phone).
		SetActive(u.Active).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toUserDomain(e), nil
}

// Department methods

func (r *EntRepository) CreateDepartment(ctx context.Context, d *Department) (*Department, error) {
	builder := r.client.Department.Create().
		SetName(d.Name).
		SetCode(d.Code).
		SetDescription(d.Description).
		SetManagerID(d.ManagerID).
		SetTenantID(d.TenantID)

	if d.ParentID != 0 {
		builder.SetParentID(d.ParentID)
	}

	e, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDeptDomain(e), nil
}

func (r *EntRepository) GetDepartment(ctx context.Context, id int, tenantID int) (*Department, error) {
	e, err := r.client.Department.Query().
		Where(department.ID(id), department.TenantID(tenantID)).
		WithChildren().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toDeptDomain(e), nil
}

func (r *EntRepository) ListDepartments(ctx context.Context, tenantID int) ([]*Department, error) {
	es, err := r.client.Department.Query().Where(department.TenantID(tenantID)).All(ctx)
	if err != nil {
		return nil, err
	}
	var res []*Department
	for _, e := range es {
		res = append(res, toDeptDomain(e))
	}
	return res, nil
}

func (r *EntRepository) GetDepartmentTree(ctx context.Context, tenantID int) ([]*Department, error) {
	es, err := r.client.Department.Query().
		Where(department.TenantID(tenantID), department.ParentIDIsNil()).
		WithChildren().
		All(ctx)
	if err != nil {
		return nil, err
	}
	var res []*Department
	for _, e := range es {
		res = append(res, toDeptDomain(e))
	}
	return res, nil
}

func (r *EntRepository) UpdateDepartment(ctx context.Context, d *Department) (*Department, error) {
	builder := r.client.Department.UpdateOneID(d.ID).
		SetName(d.Name).
		SetCode(d.Code).
		SetDescription(d.Description).
		SetManagerID(d.ManagerID)

	if d.ParentID != 0 {
		builder.SetParentID(d.ParentID)
	} else {
		builder.ClearParent()
	}

	e, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDeptDomain(e), nil
}

func (r *EntRepository) DeleteDepartment(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.Department.Delete().Where(department.ID(id), department.TenantID(tenantID)).Exec(ctx)
	return err
}

// Team methods

func (r *EntRepository) CreateTeam(ctx context.Context, t *Team) (*Team, error) {
	e, err := r.client.Team.Create().
		SetName(t.Name).
		SetCode(t.Code).
		SetDescription(t.Description).
		SetManagerID(t.ManagerID).
		SetTenantID(t.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toTeamDomain(e), nil
}

func (r *EntRepository) GetTeam(ctx context.Context, id int, tenantID int) (*Team, error) {
	e, err := r.client.Team.Query().Where(team.ID(id), team.TenantID(tenantID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return toTeamDomain(e), nil
}

func (r *EntRepository) ListTeams(ctx context.Context, tenantID int) ([]*Team, error) {
	es, err := r.client.Team.Query().Where(team.TenantID(tenantID)).All(ctx)
	if err != nil {
		return nil, err
	}
	var res []*Team
	for _, e := range es {
		res = append(res, toTeamDomain(e))
	}
	return res, nil
}

func (r *EntRepository) UpdateTeam(ctx context.Context, t *Team) (*Team, error) {
	e, err := r.client.Team.UpdateOneID(t.ID).
		SetName(t.Name).
		SetCode(t.Code).
		SetDescription(t.Description).
		SetStatus(t.Status).
		SetManagerID(t.ManagerID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toTeamDomain(e), nil
}

func (r *EntRepository) DeleteTeam(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.Team.Delete().Where(team.ID(id), team.TenantID(tenantID)).Exec(ctx)
	return err
}

func (r *EntRepository) AddTeamMember(ctx context.Context, teamID int, userID int) error {
	return r.client.Team.UpdateOneID(teamID).AddUserIDs(userID).Exec(ctx)
}

// Tag methods

func (r *EntRepository) CreateTag(ctx context.Context, t *Tag) (*Tag, error) {
	e, err := r.client.Tag.Create().
		SetName(t.Name).
		SetCode(t.Code).
		SetDescription(t.Description).
		SetColor(t.Color).
		SetTenantID(t.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toTagDomain(e), nil
}

func (r *EntRepository) ListTags(ctx context.Context, tenantID int) ([]*Tag, error) {
	es, err := r.client.Tag.Query().Where(tag.TenantID(tenantID)).All(ctx)
	if err != nil {
		return nil, err
	}
	var res []*Tag
	for _, e := range es {
		res = append(res, toTagDomain(e))
	}
	return res, nil
}

func (r *EntRepository) DeleteTag(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.Tag.Delete().Where(tag.ID(id), tag.TenantID(tenantID)).Exec(ctx)
	return err
}

// Audit Log methods

func (r *EntRepository) CreateAuditLog(ctx context.Context, l *AuditLog) error {
	return r.client.AuditLog.Create().
		SetTenantID(l.TenantID).
		SetUserID(l.UserID).
		SetRequestID(l.RequestID).
		SetIP(l.IP).
		SetResource(l.Resource).
		SetAction(l.Action).
		SetPath(l.Path).
		SetMethod(l.Method).
		SetStatusCode(l.StatusCode).
		SetNillableRequestBody(&l.RequestBody).
		Exec(ctx)
}

func (r *EntRepository) ListAuditLogs(ctx context.Context, tenantID int, userID int, limit int) ([]*AuditLog, error) {
	q := r.client.AuditLog.Query().Where(auditlog.TenantID(tenantID))
	if userID != 0 {
		q = q.Where(auditlog.UserID(userID))
	}
	es, err := q.Order(ent.Desc(auditlog.FieldCreatedAt)).Limit(limit).All(ctx)
	if err != nil {
		return nil, err
	}
	var res []*AuditLog
	for _, e := range es {
		res = append(res, toAuditLogDomain(e))
	}
	return res, nil
}
