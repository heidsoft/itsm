package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

type TeamController struct {
	service *service.TeamService
}

func NewTeamController(client *ent.Client) *TeamController {
	return &TeamController{
		service: service.NewTeamService(client),
	}
}

// CreateTeam 创建团队
func (c *TeamController) CreateTeam(ctx *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
		ManagerID   int    `json:"manager_id"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}
	team, err := c.service.CreateTeam(ctx.Request.Context(), req.Name, req.Code, req.Description, req.ManagerID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, team)
}

// AddMember 添加成员
func (c *TeamController) AddMember(ctx *gin.Context) {
	var req struct {
		TeamID int `json:"team_id" binding:"required"`
		UserID int `json:"user_id" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}

	err := c.service.AddMember(ctx.Request.Context(), req.TeamID, req.UserID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, nil)
}

// ListTeams 获取团队列表
func (c *TeamController) ListTeams(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}
	teams, err := c.service.ListTeams(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, teams)
}

// UpdateTeam 更新团队
func (c *TeamController) UpdateTeam(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的团队ID")
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Code        *string `json:"code"`
		Description *string `json:"description"`
		ManagerID   *int    `json:"manager_id"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	team, err := c.service.UpdateTeam(ctx.Request.Context(), id, req.Name, req.Code, req.Description, req.ManagerID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, team)
}

// DeleteTeam 删除团队
func (c *TeamController) DeleteTeam(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的团队ID")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	err = c.service.DeleteTeam(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除成功"})
}

