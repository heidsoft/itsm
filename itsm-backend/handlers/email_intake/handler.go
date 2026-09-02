package email_intake

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/middleware"
)

type Handler struct {
	svc          *Service
	onCall       *OnCallService
	orchestrator *EmailIntakeOrchestrator
}

func (h *Handler) SetOrchestrator(orchestrator *EmailIntakeOrchestrator) {
	h.orchestrator = orchestrator
}

func NewHandler(client *ent.Client) *Handler {
	return &Handler{svc: NewService(client), onCall: NewOnCallService(client)}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	root := rg.Group("/email-intake")
	root.GET("/conversations", middleware.RequirePermission("email_intake", "read"), h.ListConversations)
	// Conversation detail contains customer email content and therefore requires
	// the stronger NOC review permission; list metadata remains readable.
	root.GET("/conversations/:id", middleware.RequirePermission("email_intake", "review"), h.GetConversation)
	root.POST("/conversations/:id/revalidate", middleware.RequirePermission("email_intake", "review"), h.RevalidateConversation)
	root.POST("/conversations/:id/corrections", middleware.RequirePermission("email_intake", "review"), h.CorrectConversation)
	root.POST("/conversations/:id/confirm", middleware.RequirePermission("email_intake", "review"), h.ConfirmConversation)
	root.POST("/conversations/:id/reject", middleware.RequirePermission("email_intake", "review"), h.RejectConversation)
	root.POST("/conversations/:id/retry", middleware.RequirePermission("email_intake", "retry"), h.RetryConversation)
	root.POST("/conversations/:id/override", middleware.RequirePermission("email_intake", "override"), h.OverrideConversation)

	customers := root.Group("/customers")
	customers.GET("", middleware.RequirePermission("customer_master", "read"), h.ListCustomers)
	customers.POST("", middleware.RequirePermission("customer_master", "write"), h.CreateCustomer)
	customers.PUT("/:id", middleware.RequirePermission("customer_master", "write"), h.UpdateCustomer)
	customers.DELETE("/:id", middleware.RequirePermission("customer_master", "write"), h.DisableCustomer)
	branches := root.Group("/branches")
	branches.GET("", middleware.RequirePermission("customer_master", "read"), h.ListBranches)
	branches.POST("", middleware.RequirePermission("customer_master", "write"), h.CreateBranch)
	branches.PUT("/:id", middleware.RequirePermission("customer_master", "write"), h.UpdateBranch)
	branches.DELETE("/:id", middleware.RequirePermission("customer_master", "write"), h.DisableBranch)
	sources := root.Group("/source-organizations")
	sources.GET("", middleware.RequirePermission("customer_master", "read"), h.ListSourceOrganizations)
	sources.POST("", middleware.RequirePermission("customer_master", "write"), h.CreateSourceOrganization)
	sources.PUT("/:id", middleware.RequirePermission("customer_master", "write"), h.UpdateSourceOrganization)
	sources.DELETE("/:id", middleware.RequirePermission("customer_master", "write"), h.DisableSourceOrganization)
	contracts := root.Group("/support-contracts")
	contracts.GET("", middleware.RequirePermission("support_contract", "read"), h.ListSupportContracts)
	contracts.POST("", middleware.RequirePermission("support_contract", "write"), h.CreateSupportContract)
	contracts.PUT("/:id", middleware.RequirePermission("support_contract", "write"), h.UpdateSupportContract)
	contracts.DELETE("/:id", middleware.RequirePermission("support_contract", "write"), h.TerminateSupportContract)
	references := root.Group("/external-contract-references")
	references.GET("", middleware.RequirePermission("support_contract", "read"), h.ListExternalContractReferences)
	references.POST("", middleware.RequirePermission("support_contract", "write"), h.CreateExternalContractReference)
	references.PUT("/:id", middleware.RequirePermission("support_contract", "write"), h.UpdateExternalContractReference)
	references.DELETE("/:id", middleware.RequirePermission("support_contract", "write"), h.DeleteExternalContractReference)

	onCall := root.Group("/on-call")
	onCall.GET("/schedules", middleware.RequirePermission("on_call", "read"), h.ListSchedules)
	onCall.POST("/schedules", middleware.RequirePermission("on_call", "write"), h.CreateSchedule)
	onCall.POST("/shifts", middleware.RequirePermission("on_call", "write"), h.CreateShift)
	onCall.GET("/shifts", middleware.RequirePermission("on_call", "read"), h.ListShifts)
	onCall.PUT("/shifts/:id", middleware.RequirePermission("on_call", "write"), h.UpdateShift)
	onCall.DELETE("/shifts/:id", middleware.RequirePermission("on_call", "write"), h.DeleteShift)
	onCall.GET("/current", middleware.RequirePermission("on_call", "read"), h.CurrentOnCall)
}

type customerRequest struct {
	Name                   string   `json:"name" binding:"required"`
	ShortName              string   `json:"shortName"`
	Aliases                []string `json:"aliases"`
	HistoricalNames        []string `json:"historicalNames"`
	Status                 string   `json:"status"`
	LinkedCustomerTenantID *int     `json:"linkedCustomerTenantId"`
}

func (h *Handler) CreateCustomer(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid customer")
		return
	}
	entity, err := h.svc.CreateCustomer(c, tenantID, &req)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapCustomer(entity))
}

func (h *Handler) ListCustomers(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	items, err := h.svc.ListCustomers(c, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	result := make([]customerResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapCustomer(item))
	}
	common.Success(c, gin.H{"items": result, "total": len(result)})
}

func (h *Handler) UpdateCustomer(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid customer")
		return
	}
	entity, err := h.svc.UpdateCustomer(c, tenantID, id, &req)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapCustomer(entity))
}

func (h *Handler) DisableCustomer(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	entity, err := h.svc.DisableCustomer(c, tenantID, id)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapCustomer(entity))
}

type branchRequest struct {
	CustomerID int      `json:"customerId" binding:"required"`
	Name       string   `json:"name" binding:"required"`
	Aliases    []string `json:"aliases"`
	Status     string   `json:"status"`
}

func (h *Handler) CreateBranch(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	var req branchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid branch")
		return
	}
	if exists, err := h.svc.CustomerExists(c, tenantID, req.CustomerID); err != nil || !exists {
		common.Fail(c, common.ParamErrorCode, "customer not found")
		return
	}
	entity, err := h.svc.CreateBranch(c, tenantID, &req)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapBranch(entity))
}

func (h *Handler) ListBranches(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	customerID, _ := strconv.Atoi(c.Query("customerId"))
	items, err := h.svc.ListBranches(c, tenantID, customerID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	result := make([]branchResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapBranch(item))
	}
	common.Success(c, gin.H{"items": result, "total": len(result)})
}

func (h *Handler) UpdateBranch(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req branchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid branch")
		return
	}
	if exists, err := h.svc.CustomerExists(c, tenantID, req.CustomerID); err != nil || !exists {
		common.Fail(c, common.ParamErrorCode, "customer not found")
		return
	}
	entity, err := h.svc.UpdateBranch(c, tenantID, id, &req)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapBranch(entity))
}

func (h *Handler) DisableBranch(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	entity, err := h.svc.DisableBranch(c, tenantID, id)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapBranch(entity))
}

type sourceOrganizationRequest struct {
	Name           string   `json:"name" binding:"required"`
	EmailAddresses []string `json:"emailAddresses"`
	EmailDomains   []string `json:"emailDomains"`
	Status         string   `json:"status"`
}

func (h *Handler) CreateSourceOrganization(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	var req sourceOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid source organization")
		return
	}
	entity, err := h.svc.CreateSourceOrganization(c, tenantID, &req)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapSourceOrganization(entity))
}

func (h *Handler) ListSourceOrganizations(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	items, err := h.svc.ListSourceOrganizations(c, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	result := make([]sourceOrganizationResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapSourceOrganization(item))
	}
	common.Success(c, gin.H{"items": result, "total": len(result)})
}

func (h *Handler) UpdateSourceOrganization(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req sourceOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid source organization")
		return
	}
	entity, err := h.svc.UpdateSourceOrganization(c, tenantID, id, &req)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapSourceOrganization(entity))
}

func (h *Handler) DisableSourceOrganization(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	entity, err := h.svc.DisableSourceOrganization(c, tenantID, id)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapSourceOrganization(entity))
}

type supportContractRequest struct {
	CustomerID     int        `json:"customerId" binding:"required"`
	BranchID       *int       `json:"branchId"`
	ContractNumber string     `json:"contractNumber" binding:"required"`
	Status         string     `json:"status"`
	StartAt        *time.Time `json:"startAt"`
	EndAt          *time.Time `json:"endAt"`
}

func (h *Handler) CreateSupportContract(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	var req supportContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid support contract")
		return
	}
	if err := h.ensureContractRefs(c, tenantID, &req); err != nil {
		return
	}
	entity, err := h.svc.CreateSupportContract(c, tenantID, &req)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapSupportContract(entity))
}

func (h *Handler) ListSupportContracts(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	items, err := h.svc.ListSupportContracts(c, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	result := make([]supportContractResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapSupportContract(item))
	}
	common.Success(c, gin.H{"items": result, "total": len(result)})
}

func (h *Handler) UpdateSupportContract(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req supportContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid support contract")
		return
	}
	if err := h.ensureContractRefs(c, tenantID, &req); err != nil {
		return
	}
	entity, err := h.svc.UpdateSupportContract(c, tenantID, id, &req)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapSupportContract(entity))
}

// ensureContractRefs 校验合同引用的 customer/branch 存在且属于当前租户
func (h *Handler) ensureContractRefs(c *gin.Context, tenantID int, req *supportContractRequest) error {
	if exists, err := h.svc.CustomerExists(c, tenantID, req.CustomerID); err != nil || !exists {
		common.Fail(c, common.ParamErrorCode, "customer not found")
		return errMissingRef
	}
	if req.BranchID != nil {
		if exists, err := h.svc.BranchExistsForCustomer(c, tenantID, *req.BranchID, req.CustomerID); err != nil || !exists {
			common.Fail(c, common.ParamErrorCode, "branch not found for customer")
			return errMissingRef
		}
	}
	return nil
}

func (h *Handler) TerminateSupportContract(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	entity, err := h.svc.TerminateSupportContract(c, tenantID, id)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapSupportContract(entity))
}

type externalReferenceRequest struct {
	SourceOrganizationID   int    `json:"sourceOrganizationId" binding:"required"`
	SupportContractID      int    `json:"supportContractId" binding:"required"`
	ExternalContractNumber string `json:"externalContractNumber" binding:"required"`
}

var errMissingRef = errors.New("missing reference")

func (h *Handler) CreateExternalContractReference(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	var req externalReferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid external reference")
		return
	}
	if exists, err := h.svc.SourceOrganizationExists(c, tenantID, req.SourceOrganizationID); err != nil || !exists {
		common.Fail(c, common.ParamErrorCode, "source organization not found")
		return
	}
	entity, err := h.svc.CreateExternalContractReference(c, tenantID, &req)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapExternalReference(entity))
}

func (h *Handler) ListExternalContractReferences(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	items, err := h.svc.ListExternalContractReferences(c, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	result := make([]externalReferenceResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapExternalReference(item))
	}
	common.Success(c, gin.H{"items": result, "total": len(result)})
}

func (h *Handler) UpdateExternalContractReference(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req externalReferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid external reference")
		return
	}
	if exists, err := h.svc.SourceOrganizationExists(c, tenantID, req.SourceOrganizationID); err != nil || !exists {
		common.Fail(c, common.ParamErrorCode, "source organization not found")
		return
	}
	entity, err := h.svc.UpdateExternalContractReference(c, tenantID, id, &req)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapExternalReference(entity))
}

func (h *Handler) DeleteExternalContractReference(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	deleted, err := h.svc.DeleteExternalContractReference(c, tenantID, id)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, gin.H{"deleted": deleted > 0})
}

type scheduleRequest struct {
	GroupID  int    `json:"groupId" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Timezone string `json:"timezone"`
	Status   string `json:"status"`
}

func (h *Handler) CreateSchedule(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	var req scheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid schedule")
		return
	}
	groupExists, err := h.svc.GroupExistsInTenant(c, tenantID, req.GroupID)
	if err != nil || !groupExists {
		common.Fail(c, common.ParamErrorCode, "group not found in tenant")
		return
	}
	entity, err := h.svc.CreateOnCallSchedule(c, tenantID, &req)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapSchedule(entity))
}

func (h *Handler) ListSchedules(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	items, err := h.svc.ListOnCallSchedules(c, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	result := make([]scheduleResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapSchedule(item))
	}
	common.Success(c, gin.H{"items": result, "total": len(result)})
}

type shiftRequest struct {
	ScheduleID int       `json:"scheduleId" binding:"required"`
	UserID     int       `json:"userId" binding:"required"`
	StartAt    time.Time `json:"startAt" binding:"required"`
	EndAt      time.Time `json:"endAt" binding:"required"`
}

func (h *Handler) CreateShift(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	var req shiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid shift")
		return
	}
	entity, err := h.onCall.CreateShift(c, tenantID, req.ScheduleID, req.UserID, req.StartAt, req.EndAt)
	if err != nil {
		code := common.ParamErrorCode
		if !errors.Is(err, ErrOverlappingShift) && !errors.Is(err, ErrInvalidShift) {
			code = common.InternalErrorCode
		}
		common.Fail(c, code, err.Error())
		return
	}
	common.Success(c, shiftResponse{ID: entity.ID, ScheduleID: entity.ScheduleID, UserID: entity.UserID, StartAt: entity.StartAt, EndAt: entity.EndAt})
}

func (h *Handler) ListShifts(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	scheduleID, _ := strconv.Atoi(c.Query("scheduleId"))
	items, err := h.onCall.ListShifts(c, tenantID, scheduleID)
	if err != nil {
		common.InternalError(c, "failed to list shifts")
		return
	}
	result := make([]shiftResponse, 0, len(items))
	for _, item := range items {
		result = append(result, shiftResponse{ID: item.ID, ScheduleID: item.ScheduleID, UserID: item.UserID, StartAt: item.StartAt, EndAt: item.EndAt})
	}
	common.Success(c, gin.H{"items": result, "total": len(result)})
}

func (h *Handler) UpdateShift(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ParamError(c, "invalid shift id")
		return
	}
	var req shiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid shift")
		return
	}
	entity, err := h.onCall.UpdateShift(c, tenantID, id, req.ScheduleID, req.UserID, req.StartAt, req.EndAt)
	if err != nil {
		code := common.ParamErrorCode
		if errors.Is(err, ErrShiftNotFound) {
			code = common.NotFoundCode
		} else if !errors.Is(err, ErrOverlappingShift) && !errors.Is(err, ErrInvalidShift) {
			code = common.InternalErrorCode
		}
		common.Fail(c, code, err.Error())
		return
	}
	common.Success(c, shiftResponse{ID: entity.ID, ScheduleID: entity.ScheduleID, UserID: entity.UserID, StartAt: entity.StartAt, EndAt: entity.EndAt})
}

func (h *Handler) DeleteShift(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ParamError(c, "invalid shift id")
		return
	}
	err = h.onCall.DeleteShift(c, tenantID, id)
	if err != nil {
		if errors.Is(err, ErrShiftNotFound) {
			common.NotFound(c, "shift not found")
		} else {
			common.InternalError(c, "failed to delete shift")
		}
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *Handler) CurrentOnCall(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	groupID, err := strconv.Atoi(c.Query("groupId"))
	if err != nil || groupID < 1 {
		common.Fail(c, common.ParamErrorCode, "groupId is required")
		return
	}
	current, err := h.onCall.CurrentResolver(c, tenantID, groupID, time.Now())
	if err != nil {
		if errors.Is(err, ErrNoOnCall) {
			common.Fail(c, common.NotFoundCode, err.Error())
		} else {
			common.Fail(c, common.InternalErrorCode, err.Error())
		}
		return
	}
	common.Success(c, current)
}

func (h *Handler) ListConversations(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	// Pagination: page (1-based) and page_size (default 20, max 100)
	page := 1
	pageSize := 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	pageSizeValue := c.Query("pageSize")
	if pageSizeValue == "" {
		pageSizeValue = c.Query("page_size") // temporary backward compatibility
	}
	if ps, err := strconv.Atoi(pageSizeValue); err == nil && ps > 0 {
		if ps > 100 {
			ps = 100
		}
		pageSize = ps
	}
	items, total, err := h.svc.ListConversations(c, tenantID, status, page, pageSize)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	result := make([]conversationResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapConversation(item))
	}
	common.Success(c, gin.H{
		"items":    result,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handler) GetConversation(c *gin.Context) {
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid conversation id")
		return
	}
	item, err := h.svc.GetConversation(c, tenantID, id)
	if ent.IsNotFound(err) {
		common.Fail(c, common.NotFoundCode, "conversation not found")
		return
	}
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapConversationDetail(item))
}

type conversationVersionRequest struct {
	Version int `json:"version" binding:"required,min=1"`
}
type correctionRequest struct {
	Version int          `json:"version" binding:"required,min=1"`
	Fields  IntakeFields `json:"fields" binding:"required"`
}
type overrideRequest struct {
	Version   int    `json:"version" binding:"required,min=1"`
	Reason    string `json:"reason" binding:"required,min=5,max=1000"`
	Confirmed bool   `json:"confirmed" binding:"required"`
}

func (h *Handler) RevalidateConversation(c *gin.Context) {
	tenantID, id, version, ok := h.conversationAction(c)
	if !ok {
		return
	}
	updated, err := h.orchestrator.Revalidate(c, tenantID, id, version)
	h.actionResult(c, updated, err)
}

func (h *Handler) ConfirmConversation(c *gin.Context) {
	tenantID, id, version, ok := h.conversationAction(c)
	if !ok {
		return
	}
	updated, err := h.orchestrator.Confirm(c, tenantID, id, version)
	h.actionResult(c, updated, err)
}

func (h *Handler) RetryConversation(c *gin.Context) {
	tenantID, id, version, ok := h.conversationAction(c)
	if !ok {
		return
	}
	updated, err := h.orchestrator.Retry(c, tenantID, id, version)
	h.actionResult(c, updated, err)
}

func (h *Handler) RejectConversation(c *gin.Context) {
	tenantID, id, version, ok := h.conversationAction(c)
	if !ok {
		return
	}
	updated, err := h.orchestrator.Reject(c, tenantID, id, version)
	h.actionResult(c, updated, err)
}

func (h *Handler) CorrectConversation(c *gin.Context) {
	if h.orchestrator == nil {
		common.Fail(c, common.InternalErrorCode, "email intake orchestrator is unavailable")
		return
	}
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid conversation id")
		return
	}
	var req correctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid corrections")
		return
	}
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.Fail(c, common.AuthFailedCode, "user context required")
		return
	}
	updated, err := h.orchestrator.ApplyCorrections(c, tenantID, id, req.Version, userID, req.Fields)
	h.actionResult(c, updated, err)
}

func (h *Handler) OverrideConversation(c *gin.Context) {
	if h.orchestrator == nil {
		common.Fail(c, common.InternalErrorCode, "email intake orchestrator is unavailable")
		return
	}
	tenantID, ok := tenant(c)
	if !ok {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid conversation id")
		return
	}
	var req overrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid override request")
		return
	}
	if !req.Confirmed {
		common.ParamError(c, "explicit override confirmation is required")
		return
	}
	actorID, actorErr := middleware.GetUserID(c)
	if actorErr != nil {
		common.Fail(c, common.AuthFailedCode, "user context required")
		return
	}
	updated, err := h.orchestrator.Override(c, tenantID, id, req.Version, actorID, req.Reason)
	h.actionResult(c, updated, err)
}

func (h *Handler) conversationAction(c *gin.Context) (int, int, int, bool) {
	if h.orchestrator == nil {
		common.Fail(c, common.InternalErrorCode, "email intake orchestrator is unavailable")
		return 0, 0, 0, false
	}
	tenantID, ok := tenant(c)
	if !ok {
		return 0, 0, 0, false
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid conversation id")
		return 0, 0, 0, false
	}
	var req conversationVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "invalid conversation version")
		return 0, 0, 0, false
	}
	return tenantID, id, req.Version, true
}

func (h *Handler) actionResult(c *gin.Context, updated *ent.EmailConversation, err error) {
	if err != nil {
		common.Fail(c, common.ConflictCode, "email intake action conflicts with the current state")
		return
	}
	item, loadErr := h.svc.ReloadConversation(c, updated.ID, updated.TenantID)
	if loadErr != nil {
		common.InternalError(c, "failed to load updated email conversation")
		return
	}
	common.Success(c, mapConversation(item))
}

func tenant(c *gin.Context) (int, bool) {
	id, err := middleware.GetTenantID(c)
	if err != nil || id < 1 {
		common.Fail(c, common.AuthFailedCode, "tenant context required")
		return 0, false
	}
	return id, true
}

func pathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return 0, false
	}
	return id, true
}
