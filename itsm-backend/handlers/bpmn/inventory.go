package bpmn

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// InventoryHandler 存量待办盘点控制器（管理面只读）。
// 路由面刻意只有 GET：终结/改派都是写操作，必须经既有写端点人工执行，
// 本控制器不提供任何自动迁移通道（盘点与修复分离）。
type InventoryHandler struct {
	client *ent.Client
}

// NewInventoryHandler 创建存量待办盘点控制器
func NewInventoryHandler(client *ent.Client) *InventoryHandler {
	return &InventoryHandler{client: client}
}

// RegisterRoutes 注册路由。
//
// 授权契约：租户级全量待办与孤儿实例属于跨用户管理面数据，
// 与 /bpmn/tasks/all（task:admin）同一权限档；任务处理人用
// /bpmn/tasks 看自己的待办即可，不需要本端点。
func (c *InventoryHandler) RegisterRoutes(r *gin.RouterGroup) {
	if c == nil {
		return
	}
	inventory := r.Group("/bpmn/pending-inventory")
	inventory.GET("", middleware.RequirePermission("task", "admin"), c.GetPendingInventory)
}

// GetPendingInventory 生成租户级存量待办盘点。
//
// Query: staleThresholdHours（可选，默认 24，上限 720）。
// 响应：summary + pendingTasks（截断至 200 条，总数见 summary）+
// processGroups + orphanInstances + recommendations。
func (c *InventoryHandler) GetPendingInventory(ctx *gin.Context) {
	tenantID := ctx.GetInt("tenant_id")
	if tenantID <= 0 {
		common.AuthFailed(ctx, "未授权访问：缺少有效租户上下文")
		return
	}

	staleThresholdHours := service.DefaultStaleThresholdHours
	if raw := ctx.Query("staleThresholdHours"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			common.ParamError(ctx, "staleThresholdHours 需为正整数（小时）")
			return
		}
		staleThresholdHours = parsed
	}

	inventory, err := service.BuildPendingWorkflowInventory(ctx.Request.Context(), c.client, tenantID, staleThresholdHours)
	if err != nil {
		common.RespondError(ctx, err, "生成存量待办盘点失败")
		return
	}
	common.Success(ctx, inventory)
}
