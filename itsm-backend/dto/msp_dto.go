package dto

import "time"

// ==================== MSP 租户类型 ====================

// TenantType 租户类型
type TenantType string

const (
	TenantTypeInternal   TenantType = "internal"   // MSP 公司内部租户
	TenantTypeCustomer   TenantType = "customer"   // 客户租户
	TenantTypePartner    TenantType = "partner"    // 合作伙伴
	TenantTypeStandard   TenantType = "standard"   // 标准租户
)

// ==================== MSP 角色 ====================

// MSPRole MSP 员工角色
type MSPRole string

const (
	MSPRoleTech        MSPRole = "msp_tech"         // 技术支持
	MSPRoleSpecialist  MSPRole = "msp_specialist"  // 专家
	MSPRoleManager     MSPRole = "msp_manager"     // 经理
	MSPRoleViewer      MSPRole = "msp_viewer"      // 观察者
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
	MSPUserID          int        `json:"msp_user_id"`
	MSPUsername        string     `json:"msp_username,omitempty"`
	CustomerTenantID   int        `json:"customer_tenant_id"`
	CustomerTenantName string     `json:"customer_tenant_name,omitempty"`
	Role               string     `json:"role"` // primary|backup|specialist
	AssignedAt         time.Time  `json:"assigned_at"`
	DeassignedAt       *time.Time `json:"deassigned_at,omitempty"`
}

// CreateAllocationRequest 创建分配请求
type CreateAllocationRequest struct {
	MSPUserID          int    `json:"msp_user_id" binding:"required"`
	CustomerTenantID   int    `json:"customer_tenant_id" binding:"required"`
	Role               string `json:"role" binding:"omitempty,oneof=primary backup specialist"`
}

// DeallocateRequest 解除分配请求
type DeallocateRequest struct {
	MSPUserID          int    `json:"msp_user_id" binding:"required"`
	CustomerTenantID   int    `json:"customer_tenant_id" binding:"required"`
	Reason             string `json:"reason,omitempty"`
}

// ==================== 工单 MSP 信息 DTO ====================

// TicketMSPInfo 工单的 MSP 关联信息
type TicketMSPInfo struct {
	IsManagedByMSP    bool   `json:"is_managed_by_msp"`
	MSPProviderID     *int   `json:"msp_provider_id,omitempty"`
	MSPProviderName   string `json:"msp_provider_name,omitempty"`
	ManagedByUserID   *int   `json:"managed_by_user_id,omitempty"`
	ManagedByUsername string `json:"managed_by_username,omitempty"`
	MSPTicketID       string `json:"msp_ticket_id,omitempty"`
}

// AssignMSPTechnicianRequest 为工单分配 MSP 技术员请求
type AssignMSPTechnicianRequest struct {
	CustomerTenantID int `json:"customer_tenant_id" binding:"required"`
	AssignerUserID   int `json:"assigner_user_id,omitempty"`
}

// ==================== MSP 上下文 DTO ====================

// MSPContext MSP 访问上下文
type MSPContext struct {
	IsMSP            bool     `json:"is_msp"`
	MSPUserID        int      `json:"msp_user_id,omitempty"`
	CustomerTenantID *int     `json:"customer_tenant_id,omitempty"`
	Role             string   `json:"role,omitempty"`
	AllowedCustomers []int    `json:"allowed_customers,omitempty"`
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
	CustomerTenantID      int     `json:"customer_tenant_id"`
	CustomerName          string  `json:"customer_name"`
	Period                string  `json:"period"`
	TotalTickets          int     `json:"total_tickets"`
	ResolvedTickets       int     `json:"resolved_tickets"`
	MSPHandlingTimeAvg    float64 `json:"msp_handling_time_avg"`    // 小时
	SLAComplianceRate     float64 `json:"sla_compliance_rate"`     // 0.0-1.0
}

// MSPPerformanceReport MSP 员工绩效报表
type MSPPerformanceReport struct {
	MSPUserID      int     `json:"msp_user_id"`
	MSPUsername    string  `json:"msp_username"`
	Period         string  `json:"period"`
	TotalTickets   int     `json:"total_tickets"`
	ResolvedTickets  int   `json:"resolved_tickets"`
	AvgHandleTime  float64 `json:"avg_handle_time"`
	CustomerSatisfaction float64 `json:"customer_satisfaction,omitempty"`
}

// ==================== 分配历史 DTO ====================

// MSPAllocationHistory 分配历史记录
type MSPAllocationHistory struct {
	ID                  int       `json:"id"`
	MSPUserID           int       `json:"msp_user_id"`
	MSPUsername         string    `json:"msp_username"`
	CustomerTenantID    int       `json:"customer_tenant_id"`
	CustomerName        string    `json:"customer_name"`
	Role                string    `json:"role"`
	AssignedAt          time.Time `json:"assigned_at"`
	DeassignedAt        *time.Time `json:"deassigned_at,omitempty"`
	DeallocationReason  string    `json:"deallocation_reason,omitempty"`
	CreatedBy           int       `json:"created_by"`
	CreatedByName       string    `json:"created_by_name,omitempty"`
}

// ==================== 工作流节点 MSP 配置 DTO ====================

// WorkflowNodeMSPConfig 工作流节点的 MSP 配置
type WorkflowNodeMSPConfig struct {
	EnableMSPSupport   bool     `json:"enable_msp_support"`
	AllowedMSPRoles    []string `json:"allowed_msp_roles,omitempty"`
	RequireMSPApproval bool     `json:"require_msp_approval"`
}

// ==================== 查询参数 DTO ====================

// MSPAllocationQueryParam MSP 分配查询参数
type MSPAllocationQueryParam struct {
	Page              int    `json:"page,omitempty" form:"page"`
	PageSize          int    `json:"page_size,omitempty" form:"page_size"`
	MSPUserID         int    `json:"msp_user_id,omitempty" form:"msp_user_id"`
	CustomerTenantID  int    `json:"customer_tenant_id,omitempty" form:"customer_tenant_id"`
	Role              string `json:"role,omitempty" form:"role"`
	IncludeDeassigned bool   `json:"include_deassigned,omitempty" form:"include_deassigned"`
}

// MSPTicketQueryParam MSP 工单查询参数
type MSPTicketQueryParam struct {
	Page              int    `json:"page,omitempty" form:"page"`
	PageSize          int    `json:"page_size,omitempty" form:"page_size"`
	CustomerTenantID  int    `json:"customer_tenant_id,omitempty" form:"customer_tenant_id" binding:"required"`
	Status            string `json:"status,omitempty" form:"status"`
	AssignedToMSP     bool   `json:"assigned_to_msp,omitempty" form:"assigned_to_msp"`
	ManagedByUserID   int    `json:"managed_by_user_id,omitempty" form:"managed_by_user_id"`
}

// MSPReportQueryParam MSP 报表查询参数
type MSPReportQueryParam struct {
	StartDate        string `json:"start_date" form:"start_date" binding:"required"`
	EndDate          string `json:"end_date" form:"end_date" binding:"required"`
	CustomerTenantID *int   `json:"customer_tenant_id,omitempty" form:"customer_tenant_id"`
	MSPUserID        *int   `json:"msp_user_id,omitempty" form:"msp_user_id"`
}
