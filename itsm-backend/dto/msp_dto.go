package dto

import "time"

// ==================== MSP 租户类型 ====================

// TenantType 租户类型
type TenantType string

const (
	TenantTypeInternal     TenantType = "internal"
	TenantTypeCustomer     TenantType = "saas_customer"
	TenantTypeMSPProvider  TenantType = "msp_provider"
	TenantTypeMSPCustomer  TenantType = "msp_customer"
	TenantTypeLegacyMSP    TenantType = "msp"
	TenantTypeLegacyClient TenantType = "customer"
	TenantTypeStandard     TenantType = "standard"
)

// ==================== MSP 角色 ====================

// MSPRole MSP 员工角色
type MSPRole string

const (
	MSPRoleTech       MSPRole = "msp_tech"       // 技术支持
	MSPRoleSpecialist MSPRole = "msp_specialist" // 专家
	MSPRoleManager    MSPRole = "msp_manager"    // 经理
	MSPRoleViewer     MSPRole = "msp_viewer"     // 观察者
)

// AllocationRole 分配角色
type AllocationRole string

const (
	AllocationRolePrimary    AllocationRole = "primary"    // 主支持
	AllocationRoleBackup     AllocationRole = "backup"     // 备份
	AllocationRoleSpecialist AllocationRole = "specialist" // 专家支持
)

// ==================== MSP 分配相关 DTO ====================

// MSPAllocationDTO MSP 分配数据传输对象
type MSPAllocationDTO struct {
	ID                 int        `json:"id"`
	MSPUserID          int        `json:"mspUserId"`
	MSPUsername        string     `json:"mspUsername,omitempty"`
	CustomerTenantID   int        `json:"customerTenantId"`
	CustomerTenantName string     `json:"customerTenantName,omitempty"`
	Role               string     `json:"role"` // primary|backup|specialist
	AssignedAt         time.Time  `json:"assignedAt"`
	DeassignedAt       *time.Time `json:"deassignedAt,omitempty"`
}

// CreateAllocationRequest 创建分配请求
type CreateAllocationRequest struct {
	MSPUserID        int    `json:"mspUserId" binding:"required"`
	CustomerTenantID int    `json:"customerTenantId" binding:"required"`
	Role             string `json:"role" binding:"omitempty,oneof=primary backup specialist"`
}

// DeallocateRequest 解除分配请求
type DeallocateRequest struct {
	MSPUserID        int    `json:"mspUserId" binding:"required"`
	CustomerTenantID int    `json:"customerTenantId" binding:"required"`
	Reason           string `json:"reason,omitempty"`
}

// ==================== 工单 MSP 信息 DTO ====================

// TicketMSPInfo 工单的 MSP 关联信息
type TicketMSPInfo struct {
	IsManagedByMSP    bool   `json:"isManagedByMsp"`
	MSPProviderID     *int   `json:"mspProviderId,omitempty"`
	MSPProviderName   string `json:"mspProviderName,omitempty"`
	ManagedByUserID   *int   `json:"managedByUserId,omitempty"`
	ManagedByUsername string `json:"managedByUsername,omitempty"`
	MSPTicketID       string `json:"mspTicketId,omitempty"`
}

// AssignMSPTechnicianRequest 为工单分配 MSP 技术员请求
type AssignMSPTechnicianRequest struct {
	CustomerTenantID int `json:"customerTenantId" binding:"required"`
	AssignerUserID   int `json:"assignerUserId,omitempty"`
}

// ==================== MSP 上下文 DTO ====================

// MSPContext MSP 访问上下文
type MSPContext struct {
	IsMSP            bool   `json:"isMsp"`
	MSPUserID        int    `json:"mspUserId,omitempty"`
	CustomerTenantID *int   `json:"customerTenantId,omitempty"`
	Role             string `json:"role,omitempty"`
	AllowedCustomers []int  `json:"allowedCustomers,omitempty"`
}

// ==================== 客户与报表 DTO ====================

// CustomerDTO 客户租户简略信息
type CustomerDTO struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// MSPCustomerReport MSP 客户服务报表
type MSPCustomerReport struct {
	CustomerTenantID   int     `json:"customerTenantId"`
	CustomerName       string  `json:"customerName"`
	Period             string  `json:"period"`
	TotalTickets       int     `json:"totalTickets"`
	ResolvedTickets    int     `json:"resolvedTickets"`
	MSPHandlingTimeAvg float64 `json:"mspHandlingTimeAvg"` // 小时
	SLAComplianceRate  float64 `json:"slaComplianceRate"`  // 0.0-1.0
}

// MSPPerformanceReport MSP 员工绩效报表
type MSPPerformanceReport struct {
	MSPUserID            int     `json:"mspUserId"`
	MSPUsername          string  `json:"mspUsername"`
	Period               string  `json:"period"`
	TotalTickets         int     `json:"totalTickets"`
	ResolvedTickets      int     `json:"resolvedTickets"`
	AvgHandleTime        float64 `json:"avgHandleTime"`
	CustomerSatisfaction float64 `json:"customerSatisfaction,omitempty"`
}

// ==================== 分配历史 DTO ====================

// MSPAllocationHistory 分配历史记录
type MSPAllocationHistory struct {
	ID                 int        `json:"id"`
	MSPUserID          int        `json:"mspUserId"`
	MSPUsername        string     `json:"mspUsername"`
	CustomerTenantID   int        `json:"customerTenantId"`
	CustomerName       string     `json:"customerName"`
	Role               string     `json:"role"`
	AssignedAt         time.Time  `json:"assignedAt"`
	DeassignedAt       *time.Time `json:"deassignedAt,omitempty"`
	DeallocationReason string     `json:"deallocationReason,omitempty"`
	CreatedBy          int        `json:"createdBy"`
	CreatedByName      string     `json:"createdByName,omitempty"`
}

// ==================== 工作流节点 MSP 配置 DTO ====================

// WorkflowNodeMSPConfig 工作流节点的 MSP 配置
type WorkflowNodeMSPConfig struct {
	EnableMSPSupport   bool     `json:"enableMspSupport"`
	AllowedMSPRoles    []string `json:"allowedMspRoles,omitempty"`
	RequireMSPApproval bool     `json:"requireMspApproval"`
}

type MSPStatusResponse struct {
	IsMSP     bool   `json:"isMsp"`
	MSPUserID int    `json:"mspUserId,omitempty"`
	Role      string `json:"role,omitempty"`
	IsAdmin   bool   `json:"isAdmin"`
	Message   string `json:"message,omitempty"`
}

type MSPAllocationListResponse struct {
	Allocations []*MSPAllocationDTO `json:"allocations"`
	Total       int                 `json:"total"`
}

type MSPCustomerListResponse struct {
	Customers []*CustomerDTO `json:"customers"`
	Total     int            `json:"total"`
}

type MSPReportListResponse[T any] struct {
	Reports []T `json:"reports"`
	Total   int `json:"total"`
}

// ==================== 查询参数 DTO ====================

// MSPAllocationQueryParam MSP 分配查询参数
type MSPAllocationQueryParam struct {
	Page              int    `json:"page,omitempty" form:"page"`
	PageSize          int    `json:"pageSize,omitempty" form:"page_size"`
	MSPUserID         int    `json:"mspUserId,omitempty" form:"msp_user_id"`
	CustomerTenantID  int    `json:"customerTenantId,omitempty" form:"customer_tenant_id"`
	Role              string `json:"role,omitempty" form:"role"`
	IncludeDeassigned bool   `json:"includeDeassigned,omitempty" form:"include_deassigned"`
}

// MSPTicketQueryParam MSP 工单查询参数
type MSPTicketQueryParam struct {
	Page             int    `json:"page,omitempty" form:"page"`
	PageSize         int    `json:"pageSize,omitempty" form:"page_size"`
	CustomerTenantID int    `json:"customerTenantId,omitempty" form:"customer_tenant_id" binding:"required"`
	Status           string `json:"status,omitempty" form:"status"`
	AssignedToMSP    bool   `json:"assignedToMsp,omitempty" form:"assigned_to_msp"`
	ManagedByUserID  int    `json:"managedByUserId,omitempty" form:"managed_by_user_id"`
}

// MSPReportQueryParam MSP 报表查询参数
type MSPReportQueryParam struct {
	StartDate        string `json:"startDate" form:"start_date" binding:"required"`
	EndDate          string `json:"endDate" form:"end_date" binding:"required"`
	CustomerTenantID *int   `json:"customerTenantId,omitempty" form:"customer_tenant_id"`
	MSPUserID        *int   `json:"mspUserId,omitempty" form:"msp_user_id"`
}
