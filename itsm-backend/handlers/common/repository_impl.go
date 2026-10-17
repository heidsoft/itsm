package common

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"itsm-backend/common"
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
	mspRole := ""
	if e.MspRole != "" {
		mspRole = string(e.MspRole)
	}
	return &User{
		ID:           e.ID,
		Username:     e.Username,
		Email:        e.Email,
		Name:         e.Name,
		Role:         string(e.Role),
		MSPRole:      &mspRole,
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
		Members:     toTeamMembers(e.Edges.Users),
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func toTeamMembers(users []*ent.User) []*TeamMember {
	members := make([]*TeamMember, 0, len(users))
	for _, u := range users {
		members = append(members, &TeamMember{
			ID:       u.ID,
			Username: u.Username,
			Name:     u.Name,
			Email:    u.Email,
		})
	}
	return members
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

func (r *EntRepository) GetUserByID(ctx context.Context, id int, tenantID int) (*User, error) {
	e, err := r.client.User.Query().
		Where(user.ID(id), user.TenantID(tenantID)).
		Only(ctx)
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
		Where(user.TenantID(u.TenantID)).
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
	exists, err := r.client.Department.Query().
		Where(department.Code(d.Code), department.TenantID(d.TenantID), department.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("department code already exists: %s", d.Code)
	}
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
		Where(department.ID(id), department.TenantID(tenantID), department.DeletedAtIsNil()).
		WithChildren(func(q *ent.DepartmentQuery) { q.Where(department.DeletedAtIsNil()) }).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toDeptDomain(e), nil
}

func (r *EntRepository) ListDepartments(ctx context.Context, tenantID int) ([]*Department, error) {
	es, err := r.client.Department.Query().Where(department.TenantID(tenantID), department.DeletedAtIsNil()).All(ctx)
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
		Where(department.TenantID(tenantID), department.ParentIDIsNil(), department.DeletedAtIsNil()).
		WithChildren(func(q *ent.DepartmentQuery) { q.Where(department.DeletedAtIsNil()) }).
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
	exists, err := r.client.Department.Query().
		Where(department.Code(d.Code), department.TenantID(d.TenantID), department.DeletedAtIsNil(), department.IDNEQ(d.ID)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("department code already exists: %s", d.Code)
	}
	builder := r.client.Department.UpdateOneID(d.ID).
		Where(department.TenantID(d.TenantID), department.DeletedAtIsNil()).
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
	_, err := r.client.Department.Update().
		Where(department.ID(id), department.TenantID(tenantID), department.DeletedAtIsNil()).
		SetDeletedAt(time.Now()).
		Save(ctx)
	return err
}

// Team methods

// withTeamMembers 让团队查询一次性带回成员投影。列清单必须显式：整行加载会把
// password_hash 之类的字段带进进程，而接口只需要 id/username/name/email。
func withTeamMembers() func(*ent.UserQuery) {
	return func(q *ent.UserQuery) {
		q.Select(user.FieldID, user.FieldUsername, user.FieldName, user.FieldEmail).
			Order(ent.Asc(user.FieldUsername))
	}
}

func (r *EntRepository) findTeam(ctx context.Context, teamID int, tenantID int) (*ent.Team, error) {
	t, err := r.client.Team.Query().
		Where(team.ID(teamID), team.TenantID(tenantID), team.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, common.NewNotFoundError("team")
		}
		return nil, fmt.Errorf("查询团队失败: %w", err)
	}
	return t, nil
}

// assertActiveTenantUser 校验关联用户属于当前租户且处于启用状态。
// 负责人和成员都是跨资源引用，只校验主资源的租户不足以防止跨租户注入。
func (r *EntRepository) assertActiveTenantUser(ctx context.Context, userID int, tenantID int) error {
	exists, err := r.client.User.Query().
		Where(user.ID(userID), user.TenantID(tenantID), user.ActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if !exists {
		return common.NewValidationError("用户不存在、已停用或不属于当前租户", nil)
	}
	return nil
}

// teamOfUser 返回用户当前归属的团队（没有归属返回 nil）。
// users.team_users 是单向 To 边的外键，Ent 不生成 user 侧的 team 谓词，
// 因此按租户一次取回「团队 → 成员 ID」再判定；租户内团队数量是有界的。
func (r *EntRepository) teamOfUser(ctx context.Context, userID int, tenantID int) (*ent.Team, error) {
	teams, err := r.client.Team.Query().
		Where(team.TenantID(tenantID), team.DeletedAtIsNil()).
		WithUsers(func(q *ent.UserQuery) { q.Select(user.FieldID) }).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询团队成员关系失败: %w", err)
	}
	for _, t := range teams {
		for _, u := range t.Edges.Users {
			if u.ID == userID {
				return t, nil
			}
		}
	}
	return nil, nil
}

// attachManagerNames 按租户一次查回负责人姓名并写入投影。
// teams.manager_id 没有指向 users 的 Ent 边，接口只回带 ID 时页面只能拿用户目录比对，
// 目录被分页截断后负责人就显示为空，因此姓名由服务端补齐。
// 负责人被停用或已从其他租户残留时按姓名缺失处理，不回带其他租户的用户信息。
func (r *EntRepository) attachManagerNames(ctx context.Context, tenantID int, teams []*Team) error {
	ids := make([]int, 0, len(teams))
	seen := make(map[int]struct{}, len(teams))
	for _, t := range teams {
		if t == nil || t.ManagerID <= 0 {
			continue
		}
		if _, ok := seen[t.ManagerID]; ok {
			continue
		}
		seen[t.ManagerID] = struct{}{}
		ids = append(ids, t.ManagerID)
	}
	if len(ids) == 0 {
		return nil
	}
	users, err := r.client.User.Query().
		Where(user.IDIn(ids...), user.TenantID(tenantID)).
		Select(user.FieldID, user.FieldName, user.FieldUsername).
		All(ctx)
	if err != nil {
		return fmt.Errorf("查询团队负责人失败: %w", err)
	}
	names := make(map[int]string, len(users))
	for _, u := range users {
		if u.Name != "" {
			names[u.ID] = u.Name
			continue
		}
		names[u.ID] = u.Username
	}
	for _, t := range teams {
		if t != nil {
			t.ManagerName = names[t.ManagerID]
		}
	}
	return nil
}

func (r *EntRepository) CreateTeam(ctx context.Context, t *Team) (*Team, error) {
	exists, err := r.client.Team.Query().
		Where(team.Code(t.Code), team.TenantID(t.TenantID), team.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, common.NewConflictError("team", t.Code)
	}
	if t.ManagerID > 0 {
		if err := r.assertActiveTenantUser(ctx, t.ManagerID, t.TenantID); err != nil {
			return nil, err
		}
	}
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
	domain := toTeamDomain(e)
	if err := r.attachManagerNames(ctx, t.TenantID, []*Team{domain}); err != nil {
		return nil, err
	}
	return domain, nil
}

func (r *EntRepository) GetTeam(ctx context.Context, id int, tenantID int) (*Team, error) {
	e, err := r.client.Team.Query().
		Where(team.ID(id), team.TenantID(tenantID), team.DeletedAtIsNil()).
		WithUsers(withTeamMembers()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, common.NewNotFoundError("team")
		}
		return nil, err
	}
	domain := toTeamDomain(e)
	if err := r.attachManagerNames(ctx, tenantID, []*Team{domain}); err != nil {
		return nil, err
	}
	return domain, nil
}

// ListTeams 按租户分页列出团队，search 对名称/编码/描述做不区分大小写的包含匹配。
// 搜索由服务端负责：前端只拿到当前页，不能对半截数据做二次过滤后冒充全量结果。
func (r *EntRepository) ListTeams(ctx context.Context, tenantID int, page int, pageSize int, search string) ([]*Team, int, error) {
	query := r.client.Team.Query().Where(team.TenantID(tenantID), team.DeletedAtIsNil())
	if keyword := strings.TrimSpace(search); keyword != "" {
		query = query.Where(team.Or(
			team.NameContainsFold(keyword),
			team.CodeContainsFold(keyword),
			team.DescriptionContainsFold(keyword),
		))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	es, err := query.
		Order(ent.Asc(team.FieldID)).
		WithUsers(withTeamMembers()).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	res := make([]*Team, 0, len(es))
	for _, e := range es {
		res = append(res, toTeamDomain(e))
	}
	if err := r.attachManagerNames(ctx, tenantID, res); err != nil {
		return nil, 0, err
	}
	return res, total, nil
}

func (r *EntRepository) UpdateTeam(ctx context.Context, t *Team) (*Team, error) {
	if _, err := r.findTeam(ctx, t.ID, t.TenantID); err != nil {
		return nil, err
	}
	exists, err := r.client.Team.Query().
		Where(team.Code(t.Code), team.TenantID(t.TenantID), team.DeletedAtIsNil(), team.IDNEQ(t.ID)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, common.NewConflictError("team", t.Code)
	}
	if t.ManagerID > 0 {
		if err := r.assertActiveTenantUser(ctx, t.ManagerID, t.TenantID); err != nil {
			return nil, err
		}
	}
	e, err := r.client.Team.UpdateOneID(t.ID).
		Where(team.TenantID(t.TenantID), team.DeletedAtIsNil()).
		SetName(t.Name).
		SetCode(t.Code).
		SetDescription(t.Description).
		SetStatus(t.Status).
		SetManagerID(t.ManagerID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	updated, err := r.GetTeam(ctx, e.ID, t.TenantID)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *EntRepository) DeleteTeam(ctx context.Context, id int, tenantID int) error {
	if _, err := r.findTeam(ctx, id, tenantID); err != nil {
		return err
	}
	_, err := r.client.Team.UpdateOneID(id).
		Where(team.TenantID(tenantID), team.DeletedAtIsNil()).
		SetDeletedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *EntRepository) ListTeamMembers(ctx context.Context, teamID int, tenantID int, page int, pageSize int) ([]*TeamMember, int, error) {
	if _, err := r.findTeam(ctx, teamID, tenantID); err != nil {
		return nil, 0, err
	}
	query := r.client.Team.Query().
		Where(team.ID(teamID), team.TenantID(tenantID), team.DeletedAtIsNil()).
		QueryUsers()
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("统计团队成员失败: %w", err)
	}
	users, err := query.
		Select(user.FieldID, user.FieldUsername, user.FieldName, user.FieldEmail).
		Order(ent.Asc(user.FieldUsername)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询团队成员失败: %w", err)
	}
	return toTeamMembers(users), total, nil
}

// AddTeamMember 把用户归属到团队。库里是 users.team_users 外键，一个用户同一时刻
// 只能属于一个团队：直接 AddUserIDs 会把人从原团队静默搬走，所以先查归属——
// 已在目标团队按幂等成功，已在别的团队返回冲突。
func (r *EntRepository) AddTeamMember(ctx context.Context, teamID int, userID int, tenantID int) error {
	if _, err := r.findTeam(ctx, teamID, tenantID); err != nil {
		return err
	}
	if err := r.assertActiveTenantUser(ctx, userID, tenantID); err != nil {
		return err
	}
	owner, err := r.teamOfUser(ctx, userID, tenantID)
	if err != nil {
		return err
	}
	if owner != nil {
		if owner.ID == teamID {
			return nil
		}
		return common.NewAppError(
			common.ErrCodeConflict,
			fmt.Sprintf("用户已属于团队「%s」，请先从该团队移出", owner.Name),
			http.StatusConflict,
			nil,
		)
	}
	return r.client.Team.UpdateOneID(teamID).
		Where(team.TenantID(tenantID), team.DeletedAtIsNil()).
		AddUserIDs(userID).
		Exec(ctx)
}

func (r *EntRepository) RemoveTeamMember(ctx context.Context, teamID int, userID int, tenantID int) error {
	if _, err := r.findTeam(ctx, teamID, tenantID); err != nil {
		return err
	}
	owner, err := r.teamOfUser(ctx, userID, tenantID)
	if err != nil {
		return err
	}
	if owner == nil || owner.ID != teamID {
		return common.NewNotFoundError("team member")
	}
	return r.client.Team.UpdateOneID(teamID).
		Where(team.TenantID(tenantID), team.DeletedAtIsNil()).
		RemoveUserIDs(userID).
		Exec(ctx)
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
	// 返回空切片而非 nil，避免 JSON 序列化为 null 导致前端崩溃
	res := make([]*AuditLog, 0, len(es))
	for _, e := range es {
		res = append(res, toAuditLogDomain(e))
	}
	return res, nil
}
