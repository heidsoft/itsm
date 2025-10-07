package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UserController struct {
	userService *service.UserService
	logger      *zap.SugaredLogger
}

func NewUserController(userService *service.UserService, logger *zap.SugaredLogger) *UserController {
	return &UserController{
		userService: userService,
		logger:      logger,
	}
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param user body dto.CreateUserRequest true "用户信息"
// @Success 200 {object} common.Response{data=dto.UserResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/users [post]
func (uc *UserController) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	user, err := uc.userService.CreateUser(c.Request.Context(), &req)
	if err != nil {
		uc.logger.Errorf("创建用户失败: %v", err)
		common.Fail(c, 5001, "创建用户失败: "+err.Error())
		return
	}

	response := &dto.UserDetailResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Name:       user.Name,
		Department: user.Department,
		Phone:      user.Phone,
		Active:     user.Active,
		TenantID:   user.TenantID,
    	Role:       string(user.Role),
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}

	common.Success(c, response)
}

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 分页获取用户列表
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param tenant_id query int false "租户ID"
// @Param status query string false "状态过滤" Enums(active, inactive)
// @Param department query string false "部门过滤"
// @Param search query string false "搜索关键词"
// @Success 200 {object} common.Response{data=dto.PagedUsersResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/users [get]
func (uc *UserController) ListUsers(c *gin.Context) {
	var req dto.ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		uc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	result, err := uc.userService.ListUsers(c.Request.Context(), &req)
	if err != nil {
		uc.logger.Errorf("获取用户列表失败: %v", err)
		common.Fail(c, 5001, "获取用户列表失败: "+err.Error())
		return
	}

	common.Success(c, result)
}

// GetUser 获取用户详情
// @Summary 获取用户详情
// @Description 根据ID获取用户详情
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} common.Response{data=dto.UserResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/users/{id} [get]
func (uc *UserController) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "用户ID格式错误")
		return
	}

	user, err := uc.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		uc.logger.Errorf("获取用户详情失败: %v", err)
		common.Fail(c, 5001, "获取用户详情失败: "+err.Error())
		return
	}

	response := &dto.UserDetailResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Name:       user.Name,
		Department: user.Department,
		Phone:      user.Phone,
		Active:     user.Active,
		TenantID:   user.TenantID,
    	Role:       string(user.Role),
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}

	common.Success(c, response)
}

// UpdateUser 更新用户信息
// @Summary 更新用户信息
// @Description 更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param user body dto.UpdateUserRequest true "用户信息"
// @Success 200 {object} common.Response{data=dto.UserResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/users/{id} [put]
func (uc *UserController) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "用户ID格式错误")
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	user, err := uc.userService.UpdateUser(c.Request.Context(), id, &req)
	if err != nil {
		uc.logger.Errorf("更新用户失败: %v", err)
		common.Fail(c, 5001, "更新用户失败: "+err.Error())
		return
	}

	response := &dto.UserDetailResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Name:       user.Name,
		Department: user.Department,
		Phone:      user.Phone,
		Active:     user.Active,
		TenantID:   user.TenantID,
    	Role:       string(user.Role),
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}

	common.Success(c, response)
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除用户（软删除）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/users/{id} [delete]
func (uc *UserController) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "用户ID格式错误")
		return
	}

	err = uc.userService.DeleteUser(c.Request.Context(), id)
	if err != nil {
		uc.logger.Errorf("删除用户失败: %v", err)
		common.Fail(c, 5001, "删除用户失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}

// ChangeUserStatus 更改用户状态
// @Summary 更改用户状态
// @Description 激活或禁用用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param status body dto.ChangeUserStatusRequest true "状态信息"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/users/{id}/status [put]
func (uc *UserController) ChangeUserStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "用户ID格式错误")
		return
	}

	var req dto.ChangeUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	err = uc.userService.ChangeUserStatus(c.Request.Context(), id, req.Active)
	if err != nil {
		uc.logger.Errorf("更改用户状态失败: %v", err)
		common.Fail(c, 5001, "更改用户状态失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}

// ResetPassword 重置用户密码
// @Summary 重置用户密码
// @Description 管理员重置用户密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param password body dto.ResetPasswordRequest true "新密码"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/users/{id}/reset-password [put]
func (uc *UserController) ResetPassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "用户ID格式错误")
		return
	}

	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	err = uc.userService.ResetPassword(c.Request.Context(), id, req.NewPassword)
	if err != nil {
		uc.logger.Errorf("重置密码失败: %v", err)
		common.Fail(c, 5001, "重置密码失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}

// GetUserStats 获取用户统计
// @Summary 获取用户统计
// @Description 获取用户统计信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param tenant_id query int false "租户ID"
// @Success 200 {object} common.Response{data=dto.UserStatsResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/users/stats [get]
func (uc *UserController) GetUserStats(c *gin.Context) {
	tenantIDStr := c.Query("tenant_id")
	var tenantID int
	if tenantIDStr != "" {
		var err error
		tenantID, err = strconv.Atoi(tenantIDStr)
		if err != nil {
			common.Fail(c, 1001, "租户ID格式错误")
			return
		}
	}

	stats, err := uc.userService.GetUserStats(c.Request.Context(), tenantID)
	if err != nil {
		uc.logger.Errorf("获取用户统计失败: %v", err)
		common.Fail(c, 5001, "获取用户统计失败: "+err.Error())
		return
	}

	common.Success(c, stats)
}

// BatchUpdateUsers 批量更新用户
// @Summary 批量更新用户
// @Description 批量更新用户状态或部门
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body dto.BatchUpdateUsersRequest true "批量更新请求"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Router /api/v1/users/batch [put]
func (uc *UserController) BatchUpdateUsers(c *gin.Context) {
	var req dto.BatchUpdateUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	err := uc.userService.BatchUpdateUsers(c.Request.Context(), &req)
	if err != nil {
		uc.logger.Errorf("批量更新用户失败: %v", err)
		common.Fail(c, 5001, "批量更新用户失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}

// SearchUsers 搜索用户
// @Summary 搜索用户
// @Description 搜索用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param keyword query string true "搜索关键词"
// @Param tenant_id query int false "租户ID"
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} common.Response{data=[]dto.UserResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/users/search [get]
func (uc *UserController) SearchUsers(c *gin.Context) {
	var req dto.SearchUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		uc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 设置默认限制
	if req.Limit <= 0 {
		req.Limit = 10
	}

	users, err := uc.userService.SearchUsers(c.Request.Context(), &req)
	if err != nil {
		uc.logger.Errorf("搜索用户失败: %v", err)
		common.Fail(c, 5001, "搜索用户失败: "+err.Error())
		return
	}

	common.Success(c, users)
}
