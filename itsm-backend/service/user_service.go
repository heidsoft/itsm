package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/user"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewUserService(client *ent.Client, logger *zap.SugaredLogger) *UserService {
	return &UserService{
		client: client,
		logger: logger,
	}
}

// CreateUser 创建用户
func (s *UserService) CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*ent.User, error) {
	s.logger.Infof("创建用户: %s", req.Username)

	// 检查用户名是否已存在
	exists, err := s.client.User.Query().
		Where(user.UsernameEQ(req.Username)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查用户名失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("用户名已存在: %s", req.Username)
	}

	// 检查邮箱是否已存在
	exists, err = s.client.User.Query().
		Where(user.EmailEQ(req.Email)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查邮箱失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("邮箱已存在: %s", req.Email)
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 创建用户
	userEntity, err := s.client.User.Create().
		SetUsername(req.Username).
		SetEmail(req.Email).
		SetName(req.Name).
		SetDepartment(req.Department).
		SetPhone(req.Phone).
		SetPasswordHash(string(hashedPassword)).
		SetActive(true).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	s.logger.Infof("用户创建成功: ID=%d, Username=%s", userEntity.ID, userEntity.Username)
	return userEntity, nil
}

// ListUsers 获取用户列表
func (s *UserService) ListUsers(ctx context.Context, req *dto.ListUsersRequest) (*dto.PagedUsersResponse, error) {
	s.logger.Infof("获取用户列表: page=%d, pageSize=%d", req.Page, req.PageSize)

	query := s.client.User.Query()

	// 按租户过滤
	if req.TenantID > 0 {
		query = query.Where(user.TenantIDEQ(req.TenantID))
	}

	// 按状态过滤
	if req.Status != "" {
		active := req.Status == "active"
		query = query.Where(user.ActiveEQ(active))
	}

	// 按部门过滤
	if req.Department != "" {
		query = query.Where(user.DepartmentContainsFold(req.Department))
	}

	// 搜索过滤
	if req.Search != "" {
		search := strings.TrimSpace(req.Search)
		query = query.Where(
			user.Or(
				user.UsernameContainsFold(search),
				user.NameContainsFold(search),
				user.EmailContainsFold(search),
			),
		)
	}

	// 计算总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计用户总数失败: %w", err)
	}

	// 分页查询
	users, err := query.
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Order(ent.Desc(user.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	// 转换为响应格式
	userResponses := make([]*dto.UserDetailResponse, 0, len(users))
	for _, u := range users {
		userResponses = append(userResponses, &dto.UserDetailResponse{
			ID:         u.ID,
			Username:   u.Username,
			Email:      u.Email,
			Name:       u.Name,
			Department: u.Department,
			Phone:      u.Phone,
			Active:     u.Active,
			TenantID:   u.TenantID,
			CreatedAt:  u.CreatedAt,
			UpdatedAt:  u.UpdatedAt,
		})
	}

	response := &dto.PagedUsersResponse{
		Users: userResponses,
		Pagination: dto.PaginationResponse{
			Page:      req.Page,
			PageSize:  req.PageSize,
			Total:     total,
			TotalPage: (total + req.PageSize - 1) / req.PageSize,
		},
	}

	s.logger.Infof("用户列表查询成功: total=%d, returned=%d", total, len(users))
	return response, nil
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(ctx context.Context, id int) (*ent.User, error) {
	s.logger.Infof("获取用户详情: ID=%d", id)

	userEntity, err := s.client.User.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("用户不存在: ID=%d", id)
		}
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	return userEntity, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, id int, req *dto.UpdateUserRequest) (*ent.User, error) {
	s.logger.Infof("更新用户: ID=%d", id)

	// 检查用户是否存在
	existingUser, err := s.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	update := s.client.User.UpdateOneID(id)

	// 检查用户名是否已被其他用户使用
	if req.Username != "" && req.Username != existingUser.Username {
		exists, err := s.client.User.Query().
			Where(
				user.And(
					user.UsernameEQ(req.Username),
					user.IDNEQ(id),
				),
			).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("检查用户名失败: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("用户名已存在: %s", req.Username)
		}
		update = update.SetUsername(req.Username)
	}

	// 检查邮箱是否已被其他用户使用
	if req.Email != "" && req.Email != existingUser.Email {
		exists, err := s.client.User.Query().
			Where(
				user.And(
					user.EmailEQ(req.Email),
					user.IDNEQ(id),
				),
			).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("检查邮箱失败: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("邮箱已存在: %s", req.Email)
		}
		update = update.SetEmail(req.Email)
	}

	// 更新其他字段
	if req.Name != "" {
		update = update.SetName(req.Name)
	}
	if req.Department != "" {
		update = update.SetDepartment(req.Department)
	}
	if req.Phone != "" {
		update = update.SetPhone(req.Phone)
	}

	userEntity, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	s.logger.Infof("用户更新成功: ID=%d", id)
	return userEntity, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	s.logger.Infof("删除用户: ID=%d", id)

	// 检查用户是否存在
	_, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	// 软删除 - 设置为非激活状态
	err = s.client.User.UpdateOneID(id).
		SetActive(false).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	s.logger.Infof("用户删除成功: ID=%d", id)
	return nil
}

// ChangeUserStatus 更改用户状态
func (s *UserService) ChangeUserStatus(ctx context.Context, id int, active bool) error {
	s.logger.Infof("更改用户状态: ID=%d, active=%t", id, active)

	// 检查用户是否存在
	_, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.client.User.UpdateOneID(id).
		SetActive(active).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("更改用户状态失败: %w", err)
	}

	s.logger.Infof("用户状态更改成功: ID=%d, active=%t", id, active)
	return nil
}

// ResetPassword 重置用户密码
func (s *UserService) ResetPassword(ctx context.Context, id int, newPassword string) error {
	s.logger.Infof("重置用户密码: ID=%d", id)

	// 检查用户是否存在
	_, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	err = s.client.User.UpdateOneID(id).
		SetPasswordHash(string(hashedPassword)).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("重置密码失败: %w", err)
	}

	s.logger.Infof("用户密码重置成功: ID=%d", id)
	return nil
}

// GetUserStats 获取用户统计信息
func (s *UserService) GetUserStats(ctx context.Context, tenantID int) (*dto.UserStatsResponse, error) {
	s.logger.Infof("获取用户统计: tenantID=%d", tenantID)

	query := s.client.User.Query()
	if tenantID > 0 {
		query = query.Where(user.TenantIDEQ(tenantID))
	}

	// 总用户数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计总用户数失败: %w", err)
	}

	// 活跃用户数
	active, err := query.Where(user.ActiveEQ(true)).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计活跃用户数失败: %w", err)
	}

	// 非活跃用户数
	inactive := total - active

	response := &dto.UserStatsResponse{
		Total:    total,
		Active:   active,
		Inactive: inactive,
	}

	s.logger.Infof("用户统计获取成功: total=%d, active=%d, inactive=%d", total, active, inactive)
	return response, nil
}

// BatchUpdateUsers 批量更新用户
func (s *UserService) BatchUpdateUsers(ctx context.Context, req *dto.BatchUpdateUsersRequest) error {
	s.logger.Infof("批量更新用户: count=%d", len(req.UserIDs))

	if len(req.UserIDs) == 0 {
		return fmt.Errorf("用户ID列表不能为空")
	}

	update := s.client.User.Update().Where(user.IDIn(req.UserIDs...))

	// 根据操作类型更新
	switch req.Action {
	case "activate":
		update = update.SetActive(true)
	case "deactivate":
		update = update.SetActive(false)
	case "department":
		if req.Department == "" {
			return fmt.Errorf("部门不能为空")
		}
		update = update.SetDepartment(req.Department)
	default:
		return fmt.Errorf("不支持的操作类型: %s", req.Action)
	}

	count, err := update.Save(ctx)
	if err != nil {
		return fmt.Errorf("批量更新用户失败: %w", err)
	}

	s.logger.Infof("批量更新用户成功: updated=%d", count)
	return nil
}

// SearchUsers 搜索用户
func (s *UserService) SearchUsers(ctx context.Context, req *dto.SearchUsersRequest) ([]*dto.UserDetailResponse, error) {
	s.logger.Infof("搜索用户: keyword=%s", req.Keyword)

	if req.Keyword == "" {
		return []*dto.UserDetailResponse{}, nil
	}

	query := s.client.User.Query().
		Where(
			user.Or(
				user.UsernameContainsFold(req.Keyword),
				user.NameContainsFold(req.Keyword),
				user.EmailContainsFold(req.Keyword),
			),
		)

	// 按租户过滤
	if req.TenantID > 0 {
		query = query.Where(user.TenantIDEQ(req.TenantID))
	}

	// 只返回活跃用户
	query = query.Where(user.ActiveEQ(true))

	users, err := query.
		Limit(req.Limit).
		Order(ent.Asc(user.FieldName)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("搜索用户失败: %w", err)
	}

	// 转换为响应格式
	userResponses := make([]*dto.UserDetailResponse, 0, len(users))
	for _, u := range users {
		userResponses = append(userResponses, &dto.UserDetailResponse{
			ID:         u.ID,
			Username:   u.Username,
			Email:      u.Email,
			Name:       u.Name,
			Department: u.Department,
			Phone:      u.Phone,
			Active:     u.Active,
			TenantID:   u.TenantID,
			CreatedAt:  u.CreatedAt,
			UpdatedAt:  u.UpdatedAt,
		})
	}

	s.logger.Infof("用户搜索成功: found=%d", len(users))
	return userResponses, nil
}
