package on_call

import (
	"errors"
	"strconv"
	"time"

	"itsm-backend/common"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 暴露值班排班的 HTTP 面。
//
// 权限码沿用迁移 20260824 已种下的 on_call:{read,write}，与菜单基线一致。
type Handler struct {
	service *Service
	logger  *zap.SugaredLogger
}

func NewHandler(service *Service, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{service: service, logger: logger}
}

type scheduleRequest struct {
	GroupID  int    `json:"groupId"`
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
	Status   string `json:"status"`
}

type shiftRequest struct {
	ScheduleID int       `json:"scheduleId"`
	UserID     int       `json:"userId"`
	StartAt    time.Time `json:"startAt"`
	EndAt      time.Time `json:"endAt"`
}

// ListSchedules 列出值班排班表。
// @Summary 值班排班列表
// @Tags 值班管理
// @Param groupId query int false "支持组ID"
// @Router /api/v1/on-call/schedules [get]
func (h *Handler) ListSchedules(c *gin.Context) {
	tenantID, ok := h.tenant(c)
	if !ok {
		return
	}
	groupID, _ := strconv.Atoi(c.DefaultQuery("groupId", "0"))
	schedules, err := h.service.ListSchedules(c.Request.Context(), tenantID, groupID)
	if err != nil {
		h.fail(c, err, "查询值班排班失败")
		return
	}
	common.Success(c, gin.H{"items": schedules})
}

// CreateSchedule 创建值班排班表。
// @Summary 创建值班排班
// @Tags 值班管理
// @Router /api/v1/on-call/schedules [post]
func (h *Handler) CreateSchedule(c *gin.Context) {
	tenantID, ok := h.tenant(c)
	if !ok {
		return
	}
	var req scheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	schedule, err := h.service.CreateSchedule(c.Request.Context(), tenantID, ScheduleInput(req))
	if err != nil {
		h.fail(c, err, "创建值班排班失败")
		return
	}
	common.Success(c, schedule)
}

// GetSchedule 读取单个排班表。
// @Summary 值班排班详情
// @Tags 值班管理
// @Router /api/v1/on-call/schedules/{id} [get]
func (h *Handler) GetSchedule(c *gin.Context) {
	tenantID, ok := h.tenant(c)
	if !ok {
		return
	}
	scheduleID, ok := h.pathID(c, "排班ID")
	if !ok {
		return
	}
	schedule, err := h.service.GetSchedule(c.Request.Context(), tenantID, scheduleID)
	if err != nil {
		h.fail(c, err, "查询值班排班失败")
		return
	}
	common.Success(c, schedule)
}

// UpdateSchedule 更新排班表（名称/时区/状态）。
// @Summary 更新值班排班
// @Tags 值班管理
// @Router /api/v1/on-call/schedules/{id} [put]
func (h *Handler) UpdateSchedule(c *gin.Context) {
	tenantID, ok := h.tenant(c)
	if !ok {
		return
	}
	scheduleID, ok := h.pathID(c, "排班ID")
	if !ok {
		return
	}
	var req scheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	schedule, err := h.service.UpdateSchedule(c.Request.Context(), tenantID, scheduleID, ScheduleInput{
		Name: req.Name, Timezone: req.Timezone, Status: req.Status,
	})
	if err != nil {
		h.fail(c, err, "更新值班排班失败")
		return
	}
	common.Success(c, schedule)
}

// DeleteSchedule 删除排班表；仍有班次时返回 409。
// @Summary 删除值班排班
// @Tags 值班管理
// @Router /api/v1/on-call/schedules/{id} [delete]
func (h *Handler) DeleteSchedule(c *gin.Context) {
	tenantID, ok := h.tenant(c)
	if !ok {
		return
	}
	scheduleID, ok := h.pathID(c, "排班ID")
	if !ok {
		return
	}
	if err := h.service.DeleteSchedule(c.Request.Context(), tenantID, scheduleID); err != nil {
		h.fail(c, err, "删除值班排班失败")
		return
	}
	common.SuccessWithMessage(c, "排班已删除", nil)
}

// ListShifts 列出班次，可按排班表与时间窗过滤。
// @Summary 班次列表
// @Tags 值班管理
// @Param scheduleId query int false "排班表ID"
// @Param from query string false "起始时间（RFC3339）"
// @Param to query string false "结束时间（RFC3339）"
// @Router /api/v1/on-call/shifts [get]
func (h *Handler) ListShifts(c *gin.Context) {
	tenantID, ok := h.tenant(c)
	if !ok {
		return
	}
	scheduleID, _ := strconv.Atoi(c.DefaultQuery("scheduleId", "0"))
	from, ok := h.queryTime(c, "from")
	if !ok {
		return
	}
	to, ok := h.queryTime(c, "to")
	if !ok {
		return
	}
	shifts, err := h.service.ListShifts(c.Request.Context(), tenantID, scheduleID, from, to)
	if err != nil {
		h.fail(c, err, "查询班次失败")
		return
	}
	common.Success(c, gin.H{"items": shifts})
}

// CreateShift 新增班次。
// @Summary 新增值班班次
// @Tags 值班管理
// @Router /api/v1/on-call/shifts [post]
func (h *Handler) CreateShift(c *gin.Context) {
	tenantID, ok := h.tenant(c)
	if !ok {
		return
	}
	var req shiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	shift, err := h.service.CreateShift(c.Request.Context(), tenantID, req.ScheduleID, req.UserID, req.StartAt, req.EndAt)
	if err != nil {
		h.fail(c, err, "新增班次失败")
		return
	}
	common.Success(c, shift)
}

// UpdateShift 更新班次。
// @Summary 更新值班班次
// @Tags 值班管理
// @Router /api/v1/on-call/shifts/{id} [put]
func (h *Handler) UpdateShift(c *gin.Context) {
	tenantID, ok := h.tenant(c)
	if !ok {
		return
	}
	shiftID, ok := h.pathID(c, "班次ID")
	if !ok {
		return
	}
	var req shiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	shift, err := h.service.UpdateShift(c.Request.Context(), tenantID, shiftID, req.ScheduleID, req.UserID, req.StartAt, req.EndAt)
	if err != nil {
		h.fail(c, err, "更新班次失败")
		return
	}
	common.Success(c, shift)
}

// DeleteShift 删除班次。
// @Summary 删除值班班次
// @Tags 值班管理
// @Router /api/v1/on-call/shifts/{id} [delete]
func (h *Handler) DeleteShift(c *gin.Context) {
	tenantID, ok := h.tenant(c)
	if !ok {
		return
	}
	shiftID, ok := h.pathID(c, "班次ID")
	if !ok {
		return
	}
	if err := h.service.DeleteShift(c.Request.Context(), tenantID, shiftID); err != nil {
		h.fail(c, err, "删除班次失败")
		return
	}
	common.SuccessWithMessage(c, "班次已删除", nil)
}

// Current 返回某支持组此刻的值班人。邮件受理的兜底派单与告警路由都依赖它。
// @Summary 当前值班人
// @Tags 值班管理
// @Param groupId query int true "支持组ID"
// @Param at query string false "指定时刻（RFC3339，缺省为当前时间）"
// @Router /api/v1/on-call/current [get]
func (h *Handler) Current(c *gin.Context) {
	tenantID, ok := h.tenant(c)
	if !ok {
		return
	}
	groupID, err := strconv.Atoi(c.Query("groupId"))
	if err != nil || groupID <= 0 {
		common.ParamError(c, "groupId 必须是正整数")
		return
	}
	at := time.Now()
	if raw := c.Query("at"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			common.ParamError(c, "at 必须是 RFC3339 时间")
			return
		}
		at = parsed
	}
	current, err := h.service.CurrentResolver(c.Request.Context(), tenantID, groupID, at)
	if err != nil {
		h.fail(c, err, "查询当前值班人失败")
		return
	}
	common.Success(c, current)
}

func (h *Handler) tenant(c *gin.Context) (int, bool) {
	tenantID := c.GetInt("tenant_id")
	if tenantID <= 0 {
		common.ParamError(c, "租户信息错误")
		return 0, false
	}
	return tenantID, true
}

func (h *Handler) pathID(c *gin.Context, label string) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ParamError(c, label+"格式错误")
		return 0, false
	}
	return id, true
}

func (h *Handler) queryTime(c *gin.Context, key string) (*time.Time, bool) {
	raw := c.Query(key)
	if raw == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		common.ParamError(c, key+" 必须是 RFC3339 时间")
		return nil, false
	}
	return &parsed, true
}

// fail 把领域错误映射为稳定的 HTTP 语义：
// 参数类 → 400，冲突类 → 409，不存在 → 404，其余 → 500（不泄露内部细节）。
func (h *Handler) fail(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, ErrInvalidShift):
		common.ParamErrorWithErr(c, err, "值班班次参数无效")
	case errors.Is(err, ErrInvalidSchedule):
		common.ParamErrorWithErr(c, err, "值班计划参数无效")
	case errors.Is(err, ErrOverlappingShift):
		common.Conflict(c, "班次时间与已有班次重叠", nil)
	case errors.Is(err, ErrScheduleHasShift):
		common.Conflict(c, "排班仍包含班次，无法删除", nil)
	case errors.Is(err, ErrScheduleNotFound), errors.Is(err, ErrShiftNotFound):
		common.NotFound(c, "值班计划或班次不存在")
	case errors.Is(err, ErrNoOnCall):
		common.NotFound(c, "该支持组当前没有值班人")
	default:
		h.logger.Errorw("on-call request failed", "error", err)
		common.InternalError(c, fallback)
	}
}
