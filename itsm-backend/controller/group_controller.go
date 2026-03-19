package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GroupController struct {
	groupService *service.GroupService
	logger       *zap.SugaredLogger
}

func NewGroupController(groupService *service.GroupService, logger *zap.SugaredLogger) *GroupController {
	return &GroupController{
		groupService: groupService,
		logger:       logger,
	}
}

// CreateGroup 创建组
// @Summary 创建组
// @Description 创建新的组
// @Tags 组管理
// @Accept json
// @Produce json
// @Param request body dto.CreateGroupRequest true "组信息"
// @Success 200 {object} common.Response{data=dto.GroupResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/groups [post]
func (gc *GroupController) CreateGroup(c *gin.Context) {
	var req dto.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		gc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	// 获取当前用户租户ID并自动填充
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}
	// 自动填充租户ID，避免前端传入
	req.TenantID = tenantID

	group, err := gc.groupService.CreateGroup(c.Request.Context(), &req)
	if err != nil {
		gc.logger.Errorf("创建组失败: %v", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.ToGroupResponse(group))
}

// ListGroups 获取组列表
// @Summary 获取组列表
// @Description 分页获取组列表
// @Tags 组管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param tenant_id query int false "租户ID"
// @Param search query string false "搜索关键词"
// @Success 200 {object} common.Response{data=dto.PagedGroupsResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/groups [get]
func (gc *GroupController) ListGroups(c *gin.Context) {
	var req dto.ListGroupsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		gc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	// 获取租户ID（优先使用查询参数，否则从上下文中获取）
	tenantID := req.TenantID
	if tenantID == 0 {
		var err error
		tenantID, err = middleware.GetTenantID(c)
		if err != nil {
			common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
			return
		}
	}
	req.TenantID = tenantID

	groups, total, err := gc.groupService.ListGroups(c.Request.Context(), &req)
	if err != nil {
		gc.logger.Errorf("查询组列表失败: %v", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	// 转换为响应DTO
	groupResponses := make([]*dto.GroupResponse, 0, len(groups))
	for _, g := range groups {
		groupResponses = append(groupResponses, &dto.GroupResponse{
			ID:          g.ID,
			Name:        g.Name,
			Description: g.Description,
			TenantID:    g.TenantID,
			CreatedAt:   g.CreatedAt,
			UpdatedAt:   g.UpdatedAt,
		})
	}

	response := &dto.PagedGroupsResponse{
		Groups: groupResponses,
		Pagination: dto.PaginationResponse{
			Page:      req.Page,
			PageSize:  req.PageSize,
			Total:     total,
			TotalPage: (total + req.PageSize - 1) / req.PageSize,
		},
	}

	common.Success(c, response)
}

// GetGroup 获取组详情
// @Summary 获取组详情
// @Description 获取组的详细信息，包括成员列表
// @Tags 组管理
// @Accept json
// @Produce json
// @Param id path int true "组ID"
// @Success 200 {object} common.Response{data=dto.GroupDetailResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/groups/{id} [get]
func (gc *GroupController) GetGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的组ID")
		return
	}

	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	group, err := gc.groupService.GetGroup(c.Request.Context(), id, tenantID)
	if err != nil {
		gc.logger.Errorf("查询组失败: %v", err)
		common.Fail(c, common.NotFoundCode, err.Error())
		return
	}

	// 转换为响应DTO
	members := make([]*dto.UserDTO, 0, len(group.Edges.Members))
	for _, u := range group.Edges.Members {
		members = append(members, &dto.UserDTO{
			ID:       u.ID,
			Username: u.Username,
			Name:     u.Name,
			Email:    u.Email,
			Role:     string(u.Role),
		})
	}

	response := &dto.GroupDetailResponse{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		TenantID:    group.TenantID,
		Members:     members,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	}

	common.Success(c, response)
}

// UpdateGroup 更新组
// @Summary 更新组
// @Description 更新组信息
// @Tags 组管理
// @Accept json
// @Produce json
// @Param id path int true "组ID"
// @Param request body dto.UpdateGroupRequest true "组信息"
// @Success 200 {object} common.Response{data=dto.GroupResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/groups/{id} [put]
func (gc *GroupController) UpdateGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的组ID")
		return
	}

	var req dto.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		gc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	group, err := gc.groupService.UpdateGroup(c.Request.Context(), id, &req, tenantID)
	if err != nil {
		gc.logger.Errorf("更新组失败: %v", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.ToGroupResponse(group))
}

// DeleteGroup 删除组
// @Summary 删除组
// @Description 删除组及其所有关联关系
// @Tags 组管理
// @Accept json
// @Produce json
// @Param id path int true "组ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/groups/{id} [delete]
func (gc *GroupController) DeleteGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的组ID")
		return
	}

	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	err = gc.groupService.DeleteGroup(c.Request.Context(), id, tenantID)
	if err != nil {
		gc.logger.Errorf("删除组失败: %v", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "删除成功"})
}

// AddUserToGroup 添加用户到组
// @Summary 添加用户到组
// @Description 将用户添加到指定的组
// @Tags 组管理
// @Accept json
// @Produce json
// @Param id path int true "组ID"
// @Param request body dto.AddUserToGroupRequest true "用户信息"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/groups/{id}/members [post]
func (gc *GroupController) AddUserToGroup(c *gin.Context) {
	idStr := c.Param("id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的组ID")
		return
	}

	var req dto.AddUserToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		gc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	err = gc.groupService.AddUserToGroup(c.Request.Context(), groupID, req.UserID, tenantID)
	if err != nil {
		gc.logger.Errorf("添加用户到组失败: %v", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "添加成功"})
}

// RemoveUserFromGroup 从组移除用户
// @Summary 从组移除用户
// @Description 从指定组中移除用户
// @Tags 组管理
// @Accept json
// @Produce json
// @Param id path int true "组ID"
// @Param request body dto.RemoveUserFromGroupRequest true "用户信息"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/groups/{id}/members [delete]
func (gc *GroupController) RemoveUserFromGroup(c *gin.Context) {
	idStr := c.Param("id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的组ID")
		return
	}

	var req dto.RemoveUserFromGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		gc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	err = gc.groupService.RemoveUserFromGroup(c.Request.Context(), groupID, req.UserID, tenantID)
	if err != nil {
		gc.logger.Errorf("从组移除用户失败: %v", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "移除成功"})
}

// GetGroupMembers 获取组成员
// @Summary 获取组成员列表
// @Description 分页获取组的成员列表
// @Tags 组管理
// @Accept json
// @Produce json
// @Param id path int true "组ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} common.Response{data=dto.PagedUsersResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/groups/{id}/members [get]
func (gc *GroupController) GetGroupMembers(c *gin.Context) {
	idStr := c.Param("id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的组ID")
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	users, total, err := gc.groupService.GetGroupMembers(c.Request.Context(), groupID, tenantID, page, pageSize)
	if err != nil {
		gc.logger.Errorf("查询组成员失败: %v", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	// 转换为用户DTO
	userDTOs := make([]*dto.UserDTO, 0, len(users))
	for _, u := range users {
		userDTOs = append(userDTOs, &dto.UserDTO{
			ID:       u.ID,
			Username: u.Username,
			Name:     u.Name,
			Email:    u.Email,
			Role:     string(u.Role),
		})
	}

	response := &dto.GroupMembersResponse{
		Members: userDTOs,
		Pagination: dto.PaginationResponse{
			Page:      page,
			PageSize:  pageSize,
			Total:     total,
			TotalPage: (total + pageSize - 1) / pageSize,
		},
	}

	common.Success(c, response)
}
