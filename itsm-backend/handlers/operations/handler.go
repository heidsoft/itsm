package operations

import (
	"encoding/json"
	"errors"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	pg := common.GetPaginationFromQuery(c)
	result, err := h.service.List(c.Request.Context(), ListRequest{
		TenantID:      tenantID,
		Status:        c.Query("status"),
		CommandType:   c.Query("commandType"),
		AggregateType: c.Query("aggregateType"),
		Page:          pg.Page,
		PageSize:      pg.PageSize,
	})
	if err != nil {
		common.InternalError(c, "failed to list operational commands")
		return
	}
	common.Success(c, result)
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := commandID(c)
	if !ok {
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	result, err := h.service.Get(c.Request.Context(), tenantID, id)
	h.respond(c, result, err)
}

func (h *Handler) Replay(c *gin.Context) { h.mutate(c, true) }
func (h *Handler) Cancel(c *gin.Context) { h.mutate(c, false) }

func (h *Handler) BulkReplay(c *gin.Context) { h.bulk(c, true) }
func (h *Handler) BulkCancel(c *gin.Context) { h.bulk(c, false) }

func (h *Handler) mutate(c *gin.Context, replay bool) {
	id, ok := commandID(c)
	if !ok {
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	actor := Actor{
		UserID: c.GetInt("user_id"), RequestID: c.GetString("request_id"),
		IP: c.ClientIP(), Path: c.Request.URL.Path, Method: c.Request.Method,
	}
	var result *CommandDTO
	var err error
	if replay {
		result, err = h.service.Replay(c.Request.Context(), tenantID, id, actor)
	} else {
		result, err = h.service.Cancel(c.Request.Context(), tenantID, id, actor)
	}
	h.respond(c, result, err)
}

type bulkRequestBody struct {
	Status        string `json:"status"`
	CommandType   string `json:"commandType"`
	AggregateType string `json:"aggregateType"`
	LeaseExpired  bool   `json:"leaseExpired"`
	PageSize      int    `json:"pageSize"`
}

func (h *Handler) bulk(c *gin.Context, replay bool) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	actor := Actor{
		UserID: c.GetInt("user_id"), RequestID: c.GetString("request_id"),
		IP: c.ClientIP(), Path: c.Request.URL.Path, Method: c.Request.Method,
	}
	var body bulkRequestBody
	if c.Request.ContentLength > 0 {
		if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
			common.ParamError(c, "批量请求体格式不正确")
			return
		}
	}
	filter := BulkFilter{
		TenantID:      tenantID,
		Status:        body.Status,
		CommandType:   body.CommandType,
		AggregateType: body.AggregateType,
		LeaseExpired:  body.LeaseExpired,
		PageSize:      body.PageSize,
	}
	var (
		result *BulkResult
		err    error
	)
	if replay {
		result, err = h.service.BulkReplay(c.Request.Context(), filter, actor)
	} else {
		result, err = h.service.BulkCancel(c.Request.Context(), filter, actor)
	}
	if err != nil {
		switch {
		case errors.Is(err, ErrBulkEmpty):
			common.NotFound(c, "没有匹配筛选条件的操作命令")
		case errors.Is(err, ErrInvalidState):
			common.Fail(c, common.ConflictCode,
				"批量筛选状态不合法（重放支持 dead_letter/cancelled，取消支持 pending/processing/dead_letter）")
		default:
			common.InternalError(c, "批量操作命令执行失败")
		}
		return
	}
	common.Success(c, result)
}

func (h *Handler) respond(c *gin.Context, result *CommandDTO, err error) {
	switch {
	case err == nil:
		common.Success(c, result)
	case errors.Is(err, ErrCommandNotFound):
		common.NotFound(c, "操作命令不存在")
	case errors.Is(err, ErrInvalidState):
		// 状态不满足不是一个「并发变化」：重放只接受 dead_letter/cancelled，
		// 取消只接受 pending/processing/dead_letter。让调用方知道下一步可以做什么，
		// 而不是笼统地说命令状态变了。
		common.Fail(c, common.ConflictCode,
			"当前命令状态不允许该操作（重放支持 dead_letter/cancelled，取消支持 pending/processing/dead_letter）")
	case errors.Is(err, ErrConcurrentWrite):
		common.Fail(c, common.ConflictCode, "命令已被其他操作修改，请刷新后重试")
	default:
		common.InternalError(c, "操作命令执行失败")
	}
}

func commandID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ParamError(c, "命令 ID 不合法")
		return 0, false
	}
	return id, true
}
