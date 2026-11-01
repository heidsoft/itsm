package cloudsignal

import (
	"context"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
)

// Handler 暴露云监控告警与安全漏洞的查询、本地状态流转和同步健康面。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListAlerts 云监控告警列表。
//
// @Summary  云监控告警列表
// @Description 按状态、级别、云账号与关键字过滤。默认按触发时间倒序，分页口径与全站一致。
// @Tags 云信号
// @Produce json
// @Security BearerAuth
// @Param status query []string false "告警状态" Enums(firing,acknowledged,resolved)
// @Param severity query []string false "告警级别" Enums(P1,P2,P3,P4)
// @Param cloudAccountId query int false "云账号 ID"
// @Param search query string false "按名称、告警 ID、描述搜索"
// @Param page query int false "页码，从 1 开始"
// @Param pageSize query int false "每页条数，最大 200"
// @Success 200 {object} common.Response{data=common.ListResponse}
// @Router /api/v1/cloud-signals/alerts [get]
func (h *Handler) ListAlerts(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	var query AlertListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	pagination := common.GetPaginationFromQuery(c)
	filter := AlertFilter{
		Statuses:       query.Statuses,
		Severities:     query.Severities,
		CloudAccountID: query.CloudAccountID,
		Search:         query.Search,
		Page:           pagination.Page,
		PageSize:       pagination.PageSize,
	}
	records, total, err := h.svc.ListAlerts(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.respondError(c, err, "查询云监控告警失败", "告警过滤条件不合法")
		return
	}
	common.SuccessWithPagination(c, toAlertDTOList(records), pagination.Page, pagination.PageSize, int64(total))
}

// AcknowledgeAlert 认领一条仍在触发的告警。
//
// @Summary  认领云监控告警
// @Tags 云信号
// @Produce json
// @Security BearerAuth
// @Param id path int true "告警记录 ID"
// @Success 200 {object} common.Response{data=AlertResponse}
// @Failure 404 {object} common.Response "告警不存在"
// @Failure 409 {object} common.Response "告警已不是触发状态"
// @Router /api/v1/cloud-signals/alerts/{id}/acknowledge [post]
func (h *Handler) AcknowledgeAlert(c *gin.Context) {
	h.mutateAlert(c, h.svc.AcknowledgeAlert, "认领告警失败")
}

// ResolveAlert 关闭一条触发中或已认领的告警。
//
// @Summary  关闭云监控告警
// @Tags 云信号
// @Produce json
// @Security BearerAuth
// @Param id path int true "告警记录 ID"
// @Success 200 {object} common.Response{data=AlertResponse}
// @Failure 404 {object} common.Response "告警不存在"
// @Failure 409 {object} common.Response "告警已关闭"
// @Router /api/v1/cloud-signals/alerts/{id}/resolve [post]
func (h *Handler) ResolveAlert(c *gin.Context) {
	h.mutateAlert(c, h.svc.ResolveAlert, "关闭告警失败")
}

func (h *Handler) mutateAlert(
	c *gin.Context,
	action func(context.Context, int, int64) (*AlertRecord, error),
	publicMsg string,
) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	id, ok := parseSignalID(c)
	if !ok {
		return
	}
	record, err := action(c.Request.Context(), tenantID, id)
	if err != nil {
		h.respondError(c, err, publicMsg, "告警 ID 不合法")
		return
	}
	common.Success(c, toAlertDTO(record))
}

// ListFindings 安全漏洞列表。
//
// @Summary  安全漏洞列表
// @Description 按状态、严重度、漏洞类型、实例与云账号过滤。默认按严重度、最近发现时间排序。
// @Tags 云信号
// @Produce json
// @Security BearerAuth
// @Param status query []string false "漏洞状态" Enums(open,fixing,fixed,ignored)
// @Param severity query []string false "严重度" Enums(critical,high,medium,low,unknown)
// @Param vulnType query string false "漏洞类型" Enums(cve,sys,cms,app,emg)
// @Param instanceId query string false "ECS 实例 ID"
// @Param cloudAccountId query int false "云账号 ID"
// @Param search query string false "按名称、CVE 编号搜索"
// @Param page query int false "页码"
// @Param pageSize query int false "每页条数，最大 200"
// @Success 200 {object} common.Response{data=common.ListResponse}
// @Router /api/v1/cloud-signals/vulnerabilities [get]
func (h *Handler) ListFindings(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	var query FindingListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	pagination := common.GetPaginationFromQuery(c)
	filter := FindingFilter{
		Statuses:       query.Statuses,
		Severities:     query.Severities,
		VulnType:       query.VulnType,
		InstanceID:     query.InstanceID,
		CloudAccountID: query.CloudAccountID,
		Search:         query.Search,
		Page:           pagination.Page,
		PageSize:       pagination.PageSize,
	}
	findings, total, err := h.svc.ListFindings(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.respondError(c, err, "查询安全漏洞失败", "漏洞过滤条件不合法")
		return
	}
	common.SuccessWithPagination(c, toFindingDTOList(findings), pagination.Page, pagination.PageSize, int64(total))
}

// FindingSummary 未修复漏洞的数量分布。
//
// @Summary  安全漏洞统计
// @Description 只统计 open/fixing 的漏洞，已修复存量不计入。
// @Tags 云信号
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=FindingSummary}
// @Router /api/v1/cloud-signals/vulnerabilities/summary [get]
func (h *Handler) FindingSummary(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	summary, err := h.svc.FindingSummary(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "查询安全漏洞统计失败")
		return
	}
	if summary == nil {
		summary = &FindingSummary{BySeverity: map[string]int{}, ByStatus: map[string]int{}}
	}
	common.Success(c, summary)
}

// IgnoreFinding 忽略一条未修复漏洞。
//
// @Summary  忽略安全漏洞
// @Tags 云信号
// @Produce json
// @Security BearerAuth
// @Param id path int true "漏洞记录 ID"
// @Success 200 {object} common.Response{data=FindingResponse}
// @Failure 404 {object} common.Response "漏洞不存在"
// @Failure 409 {object} common.Response "漏洞已修复或已忽略"
// @Router /api/v1/cloud-signals/vulnerabilities/{id}/ignore [post]
func (h *Handler) IgnoreFinding(c *gin.Context) {
	h.mutateFinding(c, h.svc.IgnoreFinding, "忽略漏洞失败")
}

// ReopenFinding 把已修复或已忽略的漏洞放回未修复队列。
//
// @Summary  重新打开安全漏洞
// @Tags 云信号
// @Produce json
// @Security BearerAuth
// @Param id path int true "漏洞记录 ID"
// @Success 200 {object} common.Response{data=FindingResponse}
// @Failure 404 {object} common.Response "漏洞不存在"
// @Failure 409 {object} common.Response "漏洞仍在未修复队列"
// @Router /api/v1/cloud-signals/vulnerabilities/{id}/reopen [post]
func (h *Handler) ReopenFinding(c *gin.Context) {
	h.mutateFinding(c, h.svc.ReopenFinding, "重新打开漏洞失败")
}

func (h *Handler) mutateFinding(
	c *gin.Context,
	action func(context.Context, int, int64) (*SecurityFinding, error),
	publicMsg string,
) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	id, ok := parseSignalID(c)
	if !ok {
		return
	}
	finding, err := action(c.Request.Context(), tenantID, id)
	if err != nil {
		h.respondError(c, err, publicMsg, "漏洞 ID 不合法")
		return
	}
	common.Success(c, toFindingDTO(finding))
}

// SyncStatus 每个云账号的告警与漏洞同步状态。
//
// @Summary  云信号同步状态
// @Description 未配置凭据的账号返回 unconfigured，同步失败返回 unavailable，最近一次成功为 ready。
// @Tags 云信号
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=common.ListResponse}
// @Router /api/v1/cloud-signals/sync-status [get]
func (h *Handler) SyncStatus(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	states, err := h.svc.SyncStatus(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "查询云信号同步状态失败")
		return
	}
	items := toSyncStateDTOList(states)
	common.SuccessWithPagination(c, items, 1, len(items), int64(len(items)))
}

func parseSignalID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ParamError(c, "ID 不合法")
		return 0, false
	}
	return id, true
}

// respondError 把服务层的三类失败语义映射到既有业务码，其余错误按通用分类处理。
func (h *Handler) respondError(c *gin.Context, err error, internalMsg, invalidFilterMsg string) {
	switch {
	case errors.Is(err, ErrSignalNotFound):
		common.NotFound(c, "云信号不存在")
	case errors.Is(err, ErrSignalStateConflict):
		common.Conflict(c, "云信号状态已变化，请刷新后重试", nil)
	case errors.Is(err, ErrSignalInvalidFilter):
		common.ParamErrorWithErr(c, err, invalidFilterMsg)
	default:
		common.FailWithErr(c, err, internalMsg)
	}
}
