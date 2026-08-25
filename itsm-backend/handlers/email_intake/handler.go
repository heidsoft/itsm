package email_intake

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/customerbranch"
	"itsm-backend/ent/emailconversation"
	"itsm-backend/ent/externalcontractreference"
	"itsm-backend/ent/group"
	"itsm-backend/ent/oncallschedule"
	"itsm-backend/ent/servicecustomer"
	"itsm-backend/ent/sourceorganization"
	"itsm-backend/ent/supportcontract"
	"itsm-backend/middleware"
)

type Handler struct {
	client       *ent.Client
	resolver     *Resolver
	onCall       *OnCallService
	orchestrator *EmailIntakeOrchestrator
}

func (h *Handler) SetOrchestrator(orchestrator *EmailIntakeOrchestrator) {
	h.orchestrator = orchestrator
}

func NewHandler(client *ent.Client) *Handler {
	return &Handler{client: client, resolver: NewResolver(client), onCall: NewOnCallService(client)}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	root := rg.Group("/email-intake")
	root.GET("/conversations", middleware.RequirePermission("email_intake", "read"), h.ListConversations)
	root.GET("/conversations/:id", middleware.RequirePermission("email_intake", "read"), h.GetConversation)
	root.POST("/conversations/:id/revalidate", middleware.RequirePermission("email_intake", "review"), h.RevalidateConversation)
	root.POST("/conversations/:id/corrections", middleware.RequirePermission("email_intake", "review"), h.CorrectConversation)
	root.POST("/conversations/:id/confirm", middleware.RequirePermission("email_intake", "review"), h.ConfirmConversation)
	root.POST("/conversations/:id/reject", middleware.RequirePermission("email_intake", "review"), h.RejectConversation)
	root.POST("/conversations/:id/retry", middleware.RequirePermission("email_intake", "retry"), h.RevalidateConversation)
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
	status := req.Status
	if status == "" {
		status = "active"
	}
	entity, err := h.client.ServiceCustomer.Create().SetTenantID(tenantID).SetName(strings.TrimSpace(req.Name)).
		SetNormalizedName(NormalizeName(req.Name)).SetShortName(req.ShortName).SetAliases(req.Aliases).
		SetHistoricalNames(req.HistoricalNames).SetStatus(status).SetNillableLinkedCustomerTenantID(req.LinkedCustomerTenantID).Save(c)
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
	items, err := h.client.ServiceCustomer.Query().Where(servicecustomer.TenantIDEQ(tenantID)).Order(ent.Desc(servicecustomer.FieldUpdatedAt)).All(c)
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
	entity, err := h.client.ServiceCustomer.UpdateOneID(id).Where(servicecustomer.TenantIDEQ(tenantID)).SetName(strings.TrimSpace(req.Name)).SetNormalizedName(NormalizeName(req.Name)).SetShortName(req.ShortName).SetAliases(req.Aliases).SetHistoricalNames(req.HistoricalNames).SetStatus(defaultString(req.Status, "active")).SetNillableLinkedCustomerTenantID(req.LinkedCustomerTenantID).Save(c)
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
	entity, err := h.client.ServiceCustomer.UpdateOneID(id).Where(servicecustomer.TenantIDEQ(tenantID)).SetStatus("inactive").Save(c)
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
	if exists, err := h.client.ServiceCustomer.Query().Where(servicecustomer.IDEQ(req.CustomerID), servicecustomer.TenantIDEQ(tenantID)).Exist(c); err != nil || !exists {
		common.Fail(c, common.ParamErrorCode, "customer not found")
		return
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	entity, err := h.client.CustomerBranch.Create().SetTenantID(tenantID).SetCustomerID(req.CustomerID).SetName(strings.TrimSpace(req.Name)).SetNormalizedName(NormalizeName(req.Name)).SetAliases(req.Aliases).SetStatus(status).Save(c)
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
	query := h.client.CustomerBranch.Query().Where(customerbranch.TenantIDEQ(tenantID))
	if id, err := strconv.Atoi(c.Query("customerId")); err == nil && id > 0 {
		query.Where(customerbranch.CustomerIDEQ(id))
	}
	items, err := query.All(c)
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
	if exists, err := h.client.ServiceCustomer.Query().Where(servicecustomer.IDEQ(req.CustomerID), servicecustomer.TenantIDEQ(tenantID)).Exist(c); err != nil || !exists {
		common.Fail(c, common.ParamErrorCode, "customer not found")
		return
	}
	entity, err := h.client.CustomerBranch.UpdateOneID(id).Where(customerbranch.TenantIDEQ(tenantID)).SetCustomerID(req.CustomerID).SetName(strings.TrimSpace(req.Name)).SetNormalizedName(NormalizeName(req.Name)).SetAliases(req.Aliases).SetStatus(defaultString(req.Status, "active")).Save(c)
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
	entity, err := h.client.CustomerBranch.UpdateOneID(id).Where(customerbranch.TenantIDEQ(tenantID)).SetStatus("inactive").Save(c)
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
	status := req.Status
	if status == "" {
		status = "active"
	}
	entity, err := h.client.SourceOrganization.Create().SetTenantID(tenantID).SetName(strings.TrimSpace(req.Name)).SetNormalizedName(NormalizeName(req.Name)).SetEmailAddresses(req.EmailAddresses).SetEmailDomains(req.EmailDomains).SetStatus(status).Save(c)
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
	items, err := h.client.SourceOrganization.Query().Where(sourceorganization.TenantIDEQ(tenantID)).All(c)
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
	entity, err := h.client.SourceOrganization.UpdateOneID(id).Where(sourceorganization.TenantIDEQ(tenantID)).SetName(strings.TrimSpace(req.Name)).SetNormalizedName(NormalizeName(req.Name)).SetEmailAddresses(req.EmailAddresses).SetEmailDomains(req.EmailDomains).SetStatus(defaultString(req.Status, "active")).Save(c)
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
	entity, err := h.client.SourceOrganization.UpdateOneID(id).Where(sourceorganization.TenantIDEQ(tenantID)).SetStatus("inactive").Save(c)
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
	if exists, err := h.client.ServiceCustomer.Query().Where(servicecustomer.IDEQ(req.CustomerID), servicecustomer.TenantIDEQ(tenantID)).Exist(c); err != nil || !exists {
		common.Fail(c, common.ParamErrorCode, "customer not found")
		return
	}
	if req.BranchID != nil {
		if exists, err := h.client.CustomerBranch.Query().Where(customerbranch.IDEQ(*req.BranchID), customerbranch.CustomerIDEQ(req.CustomerID), customerbranch.TenantIDEQ(tenantID)).Exist(c); err != nil || !exists {
			common.Fail(c, common.ParamErrorCode, "branch not found for customer")
			return
		}
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	entity, err := h.client.SupportContract.Create().SetTenantID(tenantID).SetCustomerID(req.CustomerID).SetNillableBranchID(req.BranchID).SetContractNumber(strings.TrimSpace(req.ContractNumber)).SetNormalizedContractNumber(NormalizeContractNumber(req.ContractNumber)).SetStatus(status).SetNillableStartAt(req.StartAt).SetNillableEndAt(req.EndAt).Save(c)
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
	items, err := h.client.SupportContract.Query().Where(supportcontract.TenantIDEQ(tenantID)).All(c)
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
	if exists, err := h.client.ServiceCustomer.Query().Where(servicecustomer.IDEQ(req.CustomerID), servicecustomer.TenantIDEQ(tenantID)).Exist(c); err != nil || !exists {
		common.Fail(c, common.ParamErrorCode, "customer not found")
		return
	}
	if req.BranchID != nil {
		if exists, err := h.client.CustomerBranch.Query().Where(customerbranch.IDEQ(*req.BranchID), customerbranch.CustomerIDEQ(req.CustomerID), customerbranch.TenantIDEQ(tenantID)).Exist(c); err != nil || !exists {
			common.Fail(c, common.ParamErrorCode, "branch not found for customer")
			return
		}
	}
	entity, err := h.client.SupportContract.UpdateOneID(id).Where(supportcontract.TenantIDEQ(tenantID)).SetCustomerID(req.CustomerID).SetNillableBranchID(req.BranchID).SetContractNumber(strings.TrimSpace(req.ContractNumber)).SetNormalizedContractNumber(NormalizeContractNumber(req.ContractNumber)).SetStatus(defaultString(req.Status, "active")).SetNillableStartAt(req.StartAt).SetNillableEndAt(req.EndAt).Save(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, mapSupportContract(entity))
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
	entity, err := h.client.SupportContract.UpdateOneID(id).Where(supportcontract.TenantIDEQ(tenantID)).SetStatus("terminated").Save(c)
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
	source, err := h.client.SourceOrganization.Query().Where(sourceorganization.IDEQ(req.SourceOrganizationID), sourceorganization.TenantIDEQ(tenantID)).Only(c)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "source organization not found")
		return
	}
	_ = source
	contract, err := h.client.SupportContract.Query().Where(supportcontract.IDEQ(req.SupportContractID), supportcontract.TenantIDEQ(tenantID)).Only(c)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "support contract not found")
		return
	}
	entity, err := h.client.ExternalContractReference.Create().SetTenantID(tenantID).SetSourceOrganizationID(req.SourceOrganizationID).SetSupportContractID(contract.ID).SetCustomerID(contract.CustomerID).SetNillableBranchID(contract.BranchID).SetExternalContractNumber(strings.TrimSpace(req.ExternalContractNumber)).SetNormalizedExternalContractNumber(NormalizeContractNumber(req.ExternalContractNumber)).Save(c)
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
	items, err := h.client.ExternalContractReference.Query().Where(externalcontractreference.TenantIDEQ(tenantID)).All(c)
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
	contract, err := h.client.SupportContract.Query().Where(supportcontract.IDEQ(req.SupportContractID), supportcontract.TenantIDEQ(tenantID)).Only(c)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "support contract not found")
		return
	}
	if exists, sourceErr := h.client.SourceOrganization.Query().Where(sourceorganization.IDEQ(req.SourceOrganizationID), sourceorganization.TenantIDEQ(tenantID)).Exist(c); sourceErr != nil || !exists {
		common.Fail(c, common.ParamErrorCode, "source organization not found")
		return
	}
	entity, err := h.client.ExternalContractReference.UpdateOneID(id).Where(externalcontractreference.TenantIDEQ(tenantID)).SetSourceOrganizationID(req.SourceOrganizationID).SetSupportContractID(req.SupportContractID).SetCustomerID(contract.CustomerID).SetNillableBranchID(contract.BranchID).SetExternalContractNumber(strings.TrimSpace(req.ExternalContractNumber)).SetNormalizedExternalContractNumber(NormalizeContractNumber(req.ExternalContractNumber)).Save(c)
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
	deleted, err := h.client.ExternalContractReference.Delete().Where(externalcontractreference.IDEQ(id), externalcontractreference.TenantIDEQ(tenantID)).Exec(c)
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
	timezone := req.Timezone
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	groupExists, err := h.client.Group.Query().Where(group.IDEQ(req.GroupID), group.TenantIDEQ(tenantID)).Exist(c)
	if err != nil || !groupExists {
		common.Fail(c, common.ParamErrorCode, "group not found in tenant")
		return
	}
	entity, err := h.client.OnCallSchedule.Create().SetTenantID(tenantID).SetGroupID(req.GroupID).SetName(req.Name).SetTimezone(timezone).SetStatus(status).Save(c)
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
	items, err := h.client.OnCallSchedule.Query().Where(oncallschedule.TenantIDEQ(tenantID)).All(c)
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
	query := h.client.EmailConversation.Query().Where(emailconversation.TenantIDEQ(tenantID))
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query.Where(emailconversation.StatusEQ(status))
	}
	// Pagination: page (1-based) and page_size (default 20, max 100)
	page := 1
	pageSize := 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 {
		if ps > 100 {
			ps = 100
		}
		pageSize = ps
	}
	total, err := query.Count(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	items, err := query.
		WithCustomer().WithBranch().WithSupportContract().WithIncidents().
		Order(ent.Desc(emailconversation.FieldLastMessageAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(c)
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
		"page_size": pageSize,
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
	item, err := h.client.EmailConversation.Query().Where(emailconversation.IDEQ(id), emailconversation.TenantIDEQ(tenantID)).WithCustomer().WithBranch().WithSupportContract().WithMessages().WithAnalyses().WithOutboundMessages().WithIncidents().Only(c)
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
	Version int    `json:"version" binding:"required,min=1"`
	Reason  string `json:"reason" binding:"required,min=5,max=1000"`
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
		common.Fail(c, common.ConflictCode, err.Error())
		return
	}
	item, loadErr := h.client.EmailConversation.Query().Where(emailconversation.IDEQ(updated.ID), emailconversation.TenantIDEQ(updated.TenantID)).WithCustomer().WithBranch().WithSupportContract().WithIncidents().Only(c)
	if loadErr != nil {
		common.Fail(c, common.InternalErrorCode, loadErr.Error())
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
