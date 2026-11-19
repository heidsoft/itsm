package cmdb

// CI 数据认证任务的 HTTP 适配层。
// 权限复用 cmdb:read/cmdb:write（路由层 RequirePermission），不新增权限码。

import (
	"errors"
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CICertificationHandler 认证任务 HTTP handler。
type CICertificationHandler struct {
	svc    *CICertificationService
	logger *zap.SugaredLogger
}

// NewCICertificationHandler 构造 handler。
func NewCICertificationHandler(svc *CICertificationService, logger *zap.SugaredLogger) *CICertificationHandler {
	return &CICertificationHandler{svc: svc, logger: logger}
}

// nilIfZero 把 ent 的零值时间转成 nil：time.Time 的 omitempty 不生效，
// 零值会被序列化成 0001-01-01T00:00:00Z，前端无法区分「从未确认/未设截止」
// 与「真的在那一天发生」。ent 的 Optional+Time 字段在 Go 侧仍是值类型，
// 故这里按 IsZero 判定而非指针判空。
func nilIfZero(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func toCICertificationTaskResponse(t *ent.CICertificationTask, ciNames map[int]string) *dto.CICertificationTaskResponse {
	return &dto.CICertificationTaskResponse{
		ID: t.ID, CIID: t.CiID, CIName: ciNames[t.CiID], Status: t.Status,
		AssigneeID: t.AssigneeID, ConfirmedBy: t.ConfirmedBy,
		ConfirmedAt: nilIfZero(t.ConfirmedAt), DueAt: nilIfZero(t.DueAt),
		Evidence: t.Evidence, Source: t.Source,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

// CreateTask POST /cmdb/certification/tasks
func (h *CICertificationHandler) CreateTask(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	var req dto.CreateCICertificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "请求参数错误")
		return
	}
	task, err := h.svc.CreateTask(c.Request.Context(), tenantID, c.GetInt("user_id"), &req)
	if err != nil {
		h.respondError(c, err, "创建认证任务失败")
		return
	}
	common.Success(c, toCICertificationTaskResponse(task, nil))
}

// ConfirmTask POST /cmdb/certification/tasks/:id/confirm
func (h *CICertificationHandler) ConfirmTask(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ParamError(c, "任务ID非法")
		return
	}
	var req dto.ConfirmCICertificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "请求参数错误")
		return
	}
	task, err := h.svc.ConfirmTask(c.Request.Context(), tenantID, c.GetInt("user_id"), id, &req)
	if err != nil {
		h.respondError(c, err, "完成认证失败")
		return
	}
	common.Success(c, toCICertificationTaskResponse(task, nil))
}

// ListTasks GET /cmdb/certification/tasks
func (h *CICertificationHandler) ListTasks(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	pg := common.GetPaginationFromQuery(c)
	items, total, pending, overdue, confirmed, err := h.svc.ListTasks(c.Request.Context(), tenantID, pg.Page, pg.PageSize)
	if err != nil {
		h.respondError(c, err, "查询认证任务失败")
		return
	}

	// 批量取 CI 名称（一次 IN 查询，避免 N+1）；失败时降级为空名，不影响任务列表本身。
	ciIDs := make([]int, 0, len(items))
	for _, t := range items {
		ciIDs = append(ciIDs, t.CiID)
	}
	ciNames := make(map[int]string)
	if len(ciIDs) > 0 {
		if cis, qerr := h.svc.client.ConfigurationItem.Query().
			Where(configurationitem.TenantID(tenantID), configurationitem.IDIn(ciIDs...)).
			All(c.Request.Context()); qerr == nil {
			for _, ci := range cis {
				ciNames[ci.ID] = ci.Name
			}
		} else if h.logger != nil {
			h.logger.Warnw("认证任务列表取 CI 名称失败，降级为空名", "error", qerr)
		}
	}

	responses := make([]*dto.CICertificationTaskResponse, 0, len(items))
	for _, t := range items {
		responses = append(responses, toCICertificationTaskResponse(t, ciNames))
	}
	// 统计随列表下发，前端不必为三个计数再发请求。
	result := &dto.CICertificationListResponse{
		Items: responses, Total: total, Pending: pending, Overdue: overdue, Confirmed: confirmed,
		Page: pg.Page, PageSize: pg.PageSize,
	}
	if pg.PageSize > 0 {
		result.TotalPages = (total + pg.PageSize - 1) / pg.PageSize
	}
	common.Success(c, result)
}

// respondError 把领域 sentinel 映射成 400/4000，其余按 classifyError 兜底。
//
// 用 errors.Is 而非 ==：服务层用 fmt.Errorf("%w") 包装过 sentinel，
// 直接比较会漏判并退化成 500，客户端拿到「服务器错误」而实际上是自己传错了。
//
// 每个 sentinel 用**各自的具体消息**而不是共用调用点的 msg：曾统一传 msg，
// 导致「该任务已完成」被改写为「完成认证失败」——客户端看到的是笼统的失败，
// 无法区分「操作出错该重试」与「状态已变更别重试」。状态码同为 400，
// 但 message 是唯一能让人（和前端提示文案）区分二者的字段。
func (h *CICertificationHandler) respondError(c *gin.Context, err error, msg string) {
	var publicMsg string
	switch {
	case errors.Is(err, ErrCITaskAlreadyPending):
		publicMsg = ErrCITaskAlreadyPending.Error()
	case errors.Is(err, ErrCINotOwned):
		publicMsg = ErrCINotOwned.Error()
	case errors.Is(err, ErrCITaskAlreadyConfirmed):
		publicMsg = ErrCITaskAlreadyConfirmed.Error()
	case errors.Is(err, ErrCICertificationActorMissing):
		publicMsg = ErrCICertificationActorMissing.Error()
	default:
		if h.logger != nil {
			h.logger.Errorw(msg, "error", err)
		}
		common.RespondError(c, err, msg)
		return
	}
	common.FailWithErr(c, common.NewValidationError(publicMsg, err), publicMsg)
}
