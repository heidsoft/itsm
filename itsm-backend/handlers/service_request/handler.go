package service_request

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/internal/datascope"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func failServiceRequest(c *gin.Context, err error) {
	var appErr *common.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case common.ErrCodeBadRequest, common.ErrCodeValidation:
			common.Fail(c, common.ParamErrorCode, appErr.Message)
		case common.ErrCodeUnauthorized:
			common.Fail(c, common.UnauthorizedCode, appErr.Message)
		case common.ErrCodeForbidden:
			common.Fail(c, common.ForbiddenErrorCode, appErr.Message)
		case common.ErrCodeNotFound:
			common.Fail(c, common.NotFoundErrorCode, appErr.Message)
		case common.ErrCodeConflict:
			// Never render AppError.Error() here: it carries the debug prefix
			// ("[CONFLICT] ") plus the internal detail suffix. Classification has
			// exactly one owner and clients get the domain message.
			common.Fail(c, common.ConflictCode, appErr.Message)
		default:
			common.Fail(c, common.InternalErrorCode, appErr.Message)
		}
		return
	}
	if ent.IsNotFound(err) {
		common.Fail(c, common.NotFoundErrorCode, "Service request not found")
		return
	}
	common.RespondError(c, err, "操作失败")
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Map Domain to DTO
func (h *Handler) toDTO(req *ServiceRequest, approvals []*ServiceRequestApproval) *dto.ServiceRequestResponse {
	if req == nil {
		return nil
	}
	resp := &dto.ServiceRequestResponse{
		ID:                 req.ID,
		RequestNumber:      dto.GenerateServiceRequestNumber(req.ID, req.CreatedAt),
		TicketNumber:       dto.GenerateServiceRequestNumber(req.ID, req.CreatedAt),
		CatalogID:          req.CatalogID,
		RequesterID:        req.RequesterID,
		CIID:               req.CiID,
		Status:             req.Status,
		Title:              req.Title,
		Reason:             req.Reason,
		FormData:           req.FormData,
		FormFields:         req.FormData,
		CostCenter:         req.CostCenter,
		DataClassification: req.DataClassification,
		NeedsPublicIP:      req.NeedsPublicIP,
		SourceIPWhitelist:  req.SourceIPWhitelist,
		ComplianceAck:      req.ComplianceAck,
		CurrentLevel:       req.CurrentLevel,
		TotalLevels:        req.TotalLevels,
		Version:            req.Version,
		ProcessorID:        req.ProcessorID,
		ApprovedAt:         req.ApprovedAt,
		StartedAt:          req.StartedAt,
		CompletedAt:        req.CompletedAt,
		CompletionNote:     req.CompletionNote,
		LastError:          req.LastError,
		CreatedAt:          req.CreatedAt,
		UpdatedAt:          req.UpdatedAt,
	}
	if req.ExpireAt != nil {
		t := *req.ExpireAt
		resp.ExpireAt = &t
	}

	if approvals != nil {
		resp.Approvals = toApprovalDTOs(approvals)
	}
	return resp
}

// attachCatalog 补齐服务请求所属的服务目录项。
//
// 为什么必须下发：前端要据此判断「这个目录项是否支持自动云资源交付」，
// 才能区分「审批通过后启动交付创建云资源」与「人工履约型目录项」。
// 此前该字段恒为空，前端只能凭状态猜测，于是把「IT服务台」咨询类请求
// 也渲染成可启动云资源交付（prod 实测 O1）。
//
// 目录项缺失不影响主流程：详情照常返回，只是前端拿不到交付类型。
func (h *Handler) attachCatalog(ctx context.Context, tenantID int, resp *dto.ServiceRequestResponse) {
	if resp == nil || resp.CatalogID <= 0 {
		return
	}
	catalog, err := h.service.scRepo.Get(ctx, tenantID, resp.CatalogID)
	if err != nil || catalog == nil {
		return
	}
	resp.Catalog = &dto.ServiceCatalogResponse{
		ID:                catalog.ID,
		Name:              catalog.Name,
		Category:          catalog.Category,
		Description:       catalog.Description,
		Icon:              catalog.Icon,
		ServiceType:       catalog.ServiceType,
		Price:             catalog.Price,
		DeliveryTime:      strconv.Itoa(catalog.DeliveryTime),
		Unit:              catalog.Unit,
		RequiresApproval:  catalog.RequiresApproval,
		ApprovalLevel:     catalog.ApprovalLevel,
		Approvers:         catalog.Approvers,
		SLAResponseTime:   catalog.SLAResponseTime,
		SLAResolutionTime: catalog.SLAResolutionTime,
		CITypeID:          catalog.CITypeID,
		CloudServiceID:    catalog.CloudServiceID,
		FormSchema:        catalog.FormSchema,
		AvailableRegions:  catalog.AvailableRegions,
		AvailableSpecs:    catalog.AvailableSpecs,
		Status:            catalog.Status,
		IsActive:          catalog.IsActive,
		SortOrder:         catalog.SortOrder,
		CreatedAt:         catalog.CreatedAt,
		UpdatedAt:         catalog.UpdatedAt,
	}
}

func toApprovalDTOs(approvals []*ServiceRequestApproval) []dto.ServiceRequestApprovalResponse {
	responses := make([]dto.ServiceRequestApprovalResponse, len(approvals))
	for i, app := range approvals {
		responses[i] = dto.ServiceRequestApprovalResponse{
			ID:               app.ID,
			ServiceRequestID: app.ServiceRequestID,
			Level:            app.Level,
			Step:             app.Step,
			Status:           app.Status,
			ApproverID:       app.ApproverID,
			ApproverName:     app.ApproverName,
			Comment:          app.Comment,
			Action:           app.Action,
			TimeoutHours:     app.TimeoutHours,
			DueAt:            app.DueAt,
			ProcessedAt:      app.ProcessedAt,
			CreatedAt:        app.CreatedAt,
		}
	}
	return responses
}

// Create handles POST /api/v1/service-requests
//
//	@Summary	创建服务请求
//	@Description	基于服务目录创建服务请求，初始状态为 submitted。若目录要求审批则同步解析审批链并起审批流；合规确认（complianceAck）与公网 IP 白名单按目录规则校验。读回失败时退回创建返回的权威实体，不返回 data:null 的假成功。
//	@Tags	服务请求
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	request	body	dto.CreateServiceRequestRequest	true	"创建服务请求"
//	@Success	200	{object}	common.Response{data=dto.ServiceRequestResponse}
//	@Failure	400	{object}	common.Response	"参数错误或合规校验失败"
//	@Failure	401	{object}	common.Response
//	@Failure	404	{object}	common.Response	"服务目录不存在"
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/service-requests [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateServiceRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	normalizeCreateServiceRequest(&req)
	if req.CatalogID == 0 {
		common.Fail(c, 1001, "catalogId is required")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "Tenant ID missing")
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, 2001, "User ID missing")
		return
	}

	expireAt := req.ExpireAt

	domainReq := &ServiceRequest{
		ComplianceAck:      req.ComplianceAck,
		NeedsPublicIP:      req.NeedsPublicIP,
		DataClassification: req.DataClassification,
		Title:              req.Title,
		Reason:             req.Reason,
		FormData:           req.FormData,
		CostCenter:         req.CostCenter,
		SourceIPWhitelist:  req.SourceIPWhitelist,
		ExpireAt:           expireAt,
	}

	created, err := h.service.Create(c.Request.Context(), tenantID, userID, req.CatalogID, domainReq)
	if err != nil {
		failServiceRequest(c, err)
		return
	}

	fullReq, approvals, err := h.service.Get(c.Request.Context(), created.ID, tenantID)
	if err != nil {
		h.service.logger.Errorw("Create: failed to get created service request", "error", err, "id", created.ID)
		// Return the created object even if Get fails - created.ID is valid
		createdDTO := h.toDTO(created, nil)
		h.attachCatalog(c.Request.Context(), tenantID, createdDTO)
		common.Success(c, createdDTO)
		return
	}
	createdDTO := h.toDTO(fullReq, approvals)
	h.attachCatalog(c.Request.Context(), tenantID, createdDTO)
	common.Success(c, createdDTO)
}

// Get handles GET /api/v1/service-requests/:id
//
//	@Summary	获取服务请求详情
//	@Description	按 ID 获取服务请求详情及审批列表。行级可见性：同租户内非相关人员也返回 404（不泄漏存在性）；跨租户统一 404。
//	@Tags	服务请求
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"服务请求ID"
//	@Success	200	{object}	common.Response{data=dto.ServiceRequestResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	401	{object}	common.Response
//	@Failure	404	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/service-requests/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "Invalid ID")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	req, approvals, err := h.service.GetForActor(c.Request.Context(), id, tenantID, c.GetInt("user_id"), c.GetString("role"))
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Not Found")
		} else if common.IsAppError(err) {
			common.FailWithErr(c, err, "操作失败")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}
	common.Success(c, h.toDTOForActorWithCatalog(c.Request.Context(), tenantID, req, approvals, c.GetInt("user_id")))
}

// toDTOForActorWithCatalog 在通用 DTO 之上补上目录项，供前端判定交付类型。
func (h *Handler) toDTOForActorWithCatalog(ctx context.Context, tenantID int, req *ServiceRequest, approvals []*ServiceRequestApproval, actorID int) *dto.ServiceRequestResponse {
	resp := h.toDTOForActor(req, approvals, actorID)
	h.attachCatalog(ctx, tenantID, resp)
	return resp
}

// toDTOForActor 在通用 DTO 之上补上「当前用户是否具备审批资格」。
// 前端据此决定是否渲染「通过/拒绝」按钮——审批人集合只在审批行的 node 里，
// 不下发到前端的话，前端只能按状态渲染，导致越权者看到必然失败的审批入口。
func (h *Handler) toDTOForActor(req *ServiceRequest, approvals []*ServiceRequestApproval, actorID int) *dto.ServiceRequestResponse {
	resp := h.toDTO(req, approvals)
	if resp == nil {
		return nil
	}
	resp.CanApprove = CanActorApproveCurrentLevel(req, approvals, actorID)
	return resp
}

// ListApprovals returns the approval history for one tenant-scoped service request.
//
//	@Summary	获取服务请求审批历史
//	@Description	按请求 ID 返回该租户内服务请求的审批步骤列表（审批人、状态、意见、时间）。跨租户/不存在的请求返回 404，不暴露 ID 可枚举信号。
//	@Tags	服务请求
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"服务请求ID"
//	@Success	200	{object}	common.Response{data=[]dto.ServiceRequestApprovalResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	401	{object}	common.Response
//	@Failure	404	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/service-requests/{id}/approvals [get]
func (h *Handler) ListApprovals(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的服务请求ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "租户信息缺失")
		return
	}

	_, approvals, err := h.service.GetForActor(c.Request.Context(), id, tenantID, c.GetInt("user_id"), c.GetString("role"))
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Not Found")
			return
		}
		failServiceRequest(c, err)
		return
	}

	common.Success(c, toApprovalDTOs(approvals))
}

// List handles GET /api/v1/service-requests and GET /api/v1/service-requests/me
//
//	@Summary	获取服务请求列表
//	@Description	分页获取服务请求列表，支持状态/关键字/目录等过滤。行级数据权限：管理角色可见全租户，其余角色仅可见本人发起或分配的请求。/me 别名同样走本 handler，按认证身份收紧到「我的请求」。
//	@Tags	服务请求
//	@Produce	json
//	@Security	BearerAuth
//	@Param	page	query	int	false	"页码"	default(1)
//	@Param	pageSize	query	int	false	"每页数量"	default(20)
//	@Param	status	query	string	false	"按状态过滤"
//	@Param	keyword	query	string	false	"关键字（标题/编号）"
//	@Success	200	{object}	common.Response
//	@Failure	400	{object}	common.Response
//	@Failure	401	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/service-requests [get]
func (h *Handler) List(c *gin.Context) {
	var req dto.GetServiceRequestsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, 1001, "Invalid parameters")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	// 行级数据权限：从鉴权中间件注入的 user_id/role 取得。
	currentUserID := c.GetInt("user_id")
	currentRole := c.GetString("role")

	// 「我的请求」由路径或显式 scope=me 触发；操作者身份只取认证上下文，
	// 从不读取查询参数里的 userId（旧代码在此还读过 req.UserID，那个字段其实从未生效）。
	userID := 0
	if strings.Contains(c.Request.URL.Path, "/me") || c.Query("scope") == "me" {
		userID = currentUserID
	}

	// 行级数据权限（推广自 ticket DataScope 模式）：管理角色可见全租户，
	// 其余角色仅可见本人创建或处理的请求单。
	dataScope := datascope.DataScopeAll
	if !datascope.IsDataScopeAllRole(currentRole) {
		dataScope = datascope.DataScopeOwnedOrAssigned
	}

	// 分页只有一个所有者：HTTP 入口夹紧（缺省 1/20，只采纳 (0,100]，越界回落默认页长）。
	// 响应侧的分页元数据由 common.SuccessWithPagination 用同一组值算出。
	pagination := common.GetPaginationFromQuery(c)
	page, pageSize := pagination.Page, pagination.PageSize

	filters := ListFilters{
		Status:        normalizeServiceRequestStatus(req.Status),
		Keyword:       strings.TrimSpace(req.Keyword),
		UserID:        userID,
		Page:          page,
		PageSize:      pageSize,
		CurrentUserID: currentUserID,
		DataScope:     dataScope,
	}

	list, total, err := h.service.List(c.Request.Context(), tenantID, filters)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	dtos := make([]dto.ServiceRequestResponse, len(list))
	for i, v := range list {
		dtos[i] = *h.toDTO(v, nil)
	}

	common.SuccessWithPagination(c, dtos, page, pageSize, int64(total))
}

// ApplyApproval handles POST /api/v1/service-requests/:id/approval (and /approvals)
//
//	@Summary	审批服务请求
//	@Description	对服务请求执行同意/拒绝（approve/reject）。审批人须为租户内有效用户且不得自审；本级为会签时需满足法定人数才自动进入下一级。BPMN 待办任务与业务状态在同一事务内推进，无关联待办或并发冲突返回 409。两个路由别名（/approval 与 /approvals）共用本 handler。
//	@Tags	服务请求
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"服务请求ID"
//	@Param	request	body	dto.ServiceRequestApprovalActionRequest	true	"审批动作（action: approve/reject，comment 意见）"
//	@Success	200	{object}	common.Response{data=dto.ServiceRequestResponse}
//	@Failure	400	{object}	common.Response	"参数错误或审批意见缺失"
//	@Failure	401	{object}	common.Response
//	@Failure	403	{object}	common.Response	"审批人无权限或自审"
//	@Failure	404	{object}	common.Response
//	@Failure	409	{object}	common.Response	"状态冲突/并发冲突/无 BPMN 待办"
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/service-requests/{id}/approval [post]
func (h *Handler) ApplyApproval(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "Invalid ID")
		return
	}

	var req dto.ServiceRequestApprovalActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	userID := c.GetInt("user_id")
	role, _ := c.Get("role")
	dept, _ := c.Get("department")

	roleStr, _ := role.(string)
	deptStr, _ := dept.(string)

	res, approvals, err := h.service.ApplyApproval(c.Request.Context(), id, tenantID, userID, req.Action, req.Comment, roleStr, deptStr)
	if err != nil {
		failServiceRequest(c, err)
		return
	}

	// 会签未满足时服务端返回 200 但请求状态不变（票已记、级未进）。若只回
	// Success，运维连点「审批」看到的都是成功、状态却原地不动，既不知道
	// 还需几个人批，也无从判断该继续等人还是去修审批链。把「已批 X/Y 人」
	// 显式放到 message 里；已推进或串行审批（无会签配置）时 message 为空，
	// 走普通 Success，响应形状不变。
	if req.Action == "approve" && res != nil {
		if msg := quorumProgressMessage(res, approvals); msg != "" {
			common.SuccessWithMessage(c, msg, h.toDTO(res, approvals))
			return
		}
	}

	common.Success(c, h.toDTO(res, approvals))
}

// quorumProgressMessage 在「审批动作落在本级、已记录投票但未达阈值」时给出
// 会签进度文案，其余情况返回空串。判定与服务层同一口径：node 无 approver_ids
// 即串行审批（一次即过），不适用会签语义。
//
// 为什么要求 approved > 0：本级 pending 且一票未收，说明请求刚从上一级推进
// 过来（或尚未有人投票），此时提示「会签未满足」会让刚刚批准通过的审批人
// 误以为自己的票没生效——那句话只属于「我这一级还差人」的场景。
func quorumProgressMessage(req *ServiceRequest, approvals []*ServiceRequestApproval) string {
	var current *ServiceRequestApproval
	for _, a := range approvals {
		if a.Level == req.CurrentLevel && a.Status == ApprovalStatusPending {
			current = a
			break
		}
	}
	if current == nil || current.Node == nil {
		return ""
	}
	nodeIDs, ok := current.Node["approver_ids"].([]interface{})
	if !ok || len(nodeIDs) == 0 {
		return ""
	}
	// 阈值取不到时按「候选审批人数」兜底，与会签求值处同一口径。
	threshold := len(nodeIDs)
	if t, ok := current.Node["quorum_threshold"].(float64); ok && int(t) > 0 {
		threshold = int(t)
	}
	approved := 0
	if ids, ok := current.Node["approved_ids"].([]interface{}); ok {
		approved = len(ids)
	}
	if approved == 0 || approved >= threshold {
		return ""
	}
	return fmt.Sprintf("本级会签未满足（已批 %d/%d 人），还需 %d 人审批后自动进入下一级", approved, threshold, threshold-approved)
}

// ListPending handles GET /api/v1/service-requests/approvals/pending
//
//	@Summary	获取我的待审服务请求
//	@Description	返回当前认证审批人的待办收件箱。不接受任何查询过滤条件：状态范围由审批记录的 pending 决定，审批人身份取认证上下文；分页参数由 HTTP 入口夹紧。
//	@Tags	服务请求
//	@Produce	json
//	@Security	BearerAuth
//	@Param	page	query	int	false	"页码"	default(1)
//	@Param	pageSize	query	int	false	"每页数量"	default(20)
//	@Success	200	{object}	common.Response{data=[]dto.ServiceRequestResponse}
//	@Failure	401	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/service-requests/approvals/pending [get]
func (h *Handler) ListPending(c *gin.Context) {
	// 待办审批收件箱不接受任何查询过滤条件：状态范围由审批记录的 pending 决定，
	// 审批人身份取认证上下文，因此这里没有请求 DTO 可绑定。
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	userID := c.GetInt("user_id")
	role, _ := c.Get("role")
	roleStr, _ := role.(string)

	// 分页与 List 同一个所有者：HTTP 入口夹紧，响应元数据用同一组值算出。
	pagination := common.GetPaginationFromQuery(c)

	list, total, err := h.service.ListPendingApprovals(c.Request.Context(), tenantID, userID, roleStr, pagination.Page, pagination.PageSize)
	if err != nil {
		failServiceRequest(c, err)
		return
	}

	dtos := make([]dto.ServiceRequestResponse, len(list))
	for i, v := range list {
		// repo.ListPendingApprovals 只返回请求单本身，不带审批步骤；
		// 待办明细由 GET /service-requests/:id/approvals 单独取。
		dtos[i] = *h.toDTO(v, nil)
	}

	common.SuccessWithPagination(c, dtos, pagination.Page, pagination.PageSize, int64(total))
}

// Update handles PUT /api/v1/service-requests/:id
//
//	@Summary	更新服务请求
//	@Description	部分更新服务请求字段，仅 submitted 状态可编辑，且仅发起人或管理员可改。合规确认与公网 IP 白名单按创建时同等规则校验（写路径一致性）。写提交后读回失败时退回 Update 返回的权威实体，不返回 data:null 的假成功。
//	@Tags	服务请求
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"服务请求ID"
//	@Param	request	body	dto.UpdateServiceRequestRequest	true	"更新请求"
//	@Success	200	{object}	common.Response{data=dto.ServiceRequestResponse}
//	@Failure	400	{object}	common.Response	"参数或合规校验失败"
//	@Failure	401	{object}	common.Response
//	@Failure	403	{object}	common.Response	"非发起人或管理员"
//	@Failure	404	{object}	common.Response
//	@Failure	409	{object}	common.Response	"非 submitted 状态或版本冲突"
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/service-requests/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "Invalid ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "Tenant ID missing")
		return
	}

	var req dto.UpdateServiceRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	normalizeUpdateServiceRequest(&req)

	domainReq := &ServiceRequest{
		Title:              req.Title,
		Reason:             req.Reason,
		FormData:           req.FormData,
		CostCenter:         req.CostCenter,
		DataClassification: req.DataClassification,
		NeedsPublicIPSet:   req.NeedsPublicIP != nil,
		SourceIPWhitelist:  req.SourceIPWhitelist,
		ComplianceAckSet:   req.ComplianceAck != nil,
		ExpireAt:           req.ExpireAt,
	}
	if req.NeedsPublicIP != nil {
		domainReq.NeedsPublicIP = *req.NeedsPublicIP
	}
	if req.ComplianceAck != nil {
		domainReq.ComplianceAck = *req.ComplianceAck
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")
	updated, err := h.service.Update(c.Request.Context(), id, tenantID, userID, role, domainReq)
	if err != nil {
		failServiceRequest(c, err)
		return
	}

	// 读回仅用于拼装响应里的审批列表。写已提交，读回失败不能吞成
	// 200 + data:null 的假成功——客户端会以为更新成功却拿不到任何实体，
	// 也无法判断审批列表是否刷新。与 Create 端点同口径：读回失败时退回
	// Update 返回的权威实体（updated），主数据保证完整，仅审批列表降级为空。
	fullReq, approvals, getErr := h.service.Get(c.Request.Context(), updated.ID, tenantID)
	if getErr != nil {
		h.service.logger.Errorw("Update: failed to get updated service request, falling back to updated entity",
			"error", getErr, "id", updated.ID)
		common.Success(c, h.toDTO(updated, nil))
		return
	}
	common.Success(c, h.toDTO(fullReq, approvals))
}

// UpdateStatus handles PUT /api/v1/service-requests/:id/status
//
//	@Summary	更新服务请求状态
//	@Description	推进服务请求状态。不得经本接口绕过审批门禁：审批等效的状态跃迁由 /approval 端点负责（需 service_request:approve 权限）；本端点仅处理普通状态推进。并发版本冲突按 409 返回。
//	@Tags	服务请求
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"服务请求ID"
//	@Param	request	body	dto.UpdateServiceRequestStatusRequest	true	"状态更新请求"
//	@Success	200	{object}	common.Response{data=dto.ServiceRequestResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	401	{object}	common.Response
//	@Failure	403	{object}	common.Response
//	@Failure	404	{object}	common.Response
//	@Failure	409	{object}	common.Response	"非法状态流转或并发冲突"
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/service-requests/{id}/status [put]
func (h *Handler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "Invalid ID")
		return
	}

	var req dto.UpdateServiceRequestStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	status := normalizeServiceRequestStatus(req.Status)
	if status == "" {
		common.Fail(c, 1001, "status is required")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "Tenant ID missing")
		return
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")
	if err := h.service.UpdateStatus(c.Request.Context(), id, tenantID, userID, role, status); err != nil {
		failServiceRequest(c, err)
		return
	}

	fullReq, approvals, err := h.service.Get(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Success(c, gin.H{"id": id, "status": status})
		return
	}
	common.Success(c, h.toDTO(fullReq, approvals))
}

// Delete handles DELETE /api/v1/service-requests/:id
//
//	@Summary	删除服务请求
//	@Description	删除服务请求。DELETE 幂等：已删除/不存在/跨租户均视为成功，不误报服务器错误。行级写守卫仍生效——非发起人或管理员删除他人请求返回 403。
//	@Tags	服务请求
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"服务请求ID"
//	@Success	200	{object}	common.Response
//	@Failure	400	{object}	common.Response
//	@Failure	401	{object}	common.Response
//	@Failure	403	{object}	common.Response	"非发起人或管理员"
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/service-requests/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "Invalid ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, 2001, "Tenant ID missing")
		return
	}

	err = h.service.Delete(c.Request.Context(), id, tenantID, c.GetInt("user_id"), c.GetString("role"))
	if err != nil {
		failServiceRequest(c, err)
		return
	}

	common.Success(c, nil)
}

func normalizeCreateServiceRequest(req *dto.CreateServiceRequestRequest) {
	if req.FormData == nil {
		req.FormData = map[string]any{}
	}
	if req.Title == "" {
		if title, ok := req.FormData["title"].(string); ok {
			req.Title = title
		}
	}
	if req.Reason == "" {
		if reason, ok := req.FormData["reason"].(string); ok {
			req.Reason = reason
		}
	}
	if req.CostCenter == "" {
		if costCenter, ok := req.FormData["cost_center"].(string); ok {
			req.CostCenter = costCenter
		}
	}
	if req.DataClassification == "" {
		if classification, ok := req.FormData["data_classification"].(string); ok {
			req.DataClassification = classification
		}
	}
	if req.DataClassification == "" {
		req.DataClassification = "internal"
	}
	if len(req.SourceIPWhitelist) == 0 {
		if whitelist, ok := req.FormData["source_ip_whitelist"].([]string); ok {
			req.SourceIPWhitelist = whitelist
		}
	}
	if req.ExpireAt == nil {
		if expireAt, ok := req.FormData["expire_at"].(string); ok {
			if parsed, err := time.Parse(time.RFC3339, expireAt); err == nil {
				req.ExpireAt = &parsed
			}
		}
	}
	if req.ExpireAt == nil {
		defaultExpireAt := time.Now().Add(30 * 24 * time.Hour)
		req.ExpireAt = &defaultExpireAt
	}
	if ack, ok := req.FormData["compliance_ack"].(bool); ok {
		req.ComplianceAck = ack
	}
}

func normalizeUpdateServiceRequest(req *dto.UpdateServiceRequestRequest) {
}

func normalizeServiceRequestStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "pending_approval", "pending":
		return "submitted"
	case "approved":
		return "security_approved"
	case "in_progress":
		return "provisioning"
	case "completed":
		return "delivered"
	default:
		return strings.TrimSpace(status)
	}
}

// GetServiceRequestSLA handles GET /api/v1/service-requests/:id/sla
// @Summary 获取服务请求SLA
// @Description 读取服务请求的 SLA 状态（2026-10-21 统一绑定落地）。bound=false 表示该请求没有绑定任何 SLA 定义（租户未配置 service_request 类型定义）——「没配 SLA」与「SLA 正常且未超时」是两种语义，调用方不得混淆；slaStatus 取值 unbound/active/warning/breached/paused/met
// @Tags 服务请求
// @Produce json
// @Param id path int true "请求ID"
// @Success 200 {object} common.Response{data=dto.ServiceRequestSLAResponse}
// @Failure 400 {object} common.Response
// @Failure 401 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-requests/{id}/sla [get]
func (h *Handler) GetServiceRequestSLA(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, 1001, "Invalid ID")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	info, err := h.service.GetServiceRequestSLA(c.Request.Context(), id, tenantID, c.GetInt("user_id"), c.GetString("role"))
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Not Found")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}
	common.Success(c, info)
}
