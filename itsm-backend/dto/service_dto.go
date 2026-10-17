package dto

import (
	"fmt"
	"strconv"
	"time"

	"itsm-backend/ent"
)

// UserResponse 用户响应
type UserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// CreateServiceRequestRequest 创建服务请求请求
type CreateServiceRequestRequest struct {
	CatalogID int            `json:"catalogId" binding:"omitempty,min=1"`
	Title     string         `json:"title" binding:"omitempty,max=255"`
	Reason    string         `json:"reason" binding:"omitempty,max=500"`
	FormData  map[string]any `json:"formData" binding:"omitempty"`

	CostCenter         string     `json:"costCenter" binding:"omitempty,max=100"`
	DataClassification string     `json:"dataClassification" binding:"omitempty,oneof=public internal confidential restricted"`
	NeedsPublicIP      bool       `json:"needsPublicIp"`
	SourceIPWhitelist  []string   `json:"sourceIpWhitelist" binding:"omitempty"`
	ExpireAt           *time.Time `json:"expireAt" binding:"omitempty"`
	ComplianceAck      bool       `json:"complianceAck"`
}

// UpdateServiceRequestStatusRequest 更新服务请求状态请求
type UpdateServiceRequestStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateServiceRequestRequest 更新服务请求请求
type UpdateServiceRequestRequest struct {
	Title    string         `json:"title" binding:"omitempty,max=255"`
	Reason   string         `json:"reason" binding:"omitempty,max=500"`
	FormData map[string]any `json:"formData" binding:"omitempty"`

	CostCenter         string     `json:"costCenter" binding:"omitempty,max=100"`
	DataClassification string     `json:"dataClassification" binding:"omitempty,oneof=public internal confidential restricted"`
	NeedsPublicIP      *bool      `json:"needsPublicIp"`
	SourceIPWhitelist  []string   `json:"sourceIpWhitelist" binding:"omitempty"`
	ExpireAt           *time.Time `json:"expireAt" binding:"omitempty"`
	ComplianceAck      *bool      `json:"complianceAck"`
}

// GetServiceCatalogsRequest 服务目录列表的过滤条件。
//
// 这里不声明 page/pageSize：分页夹紧的唯一所有者是 HTTP 入口的
// common.GetPaginationFromQuery（缺省 1/20，只采纳 (0,100]，越界回落默认页长）。
// DTO 上再加 binding 上限会形成第二套真相：越界输入先返回业务 code 1001 参数错误，
// 而不是回落默认页长；键名 size 也和平台请求契约 pageSize 不同名。
type GetServiceCatalogsRequest struct {
	Category string `json:"category" form:"category"`
	Status   string `json:"status" form:"status" binding:"omitempty,oneof=enabled disabled"`
	// Keyword 服务端过滤 name/description，与 /search 同一套匹配口径。
	// 管理页需要「按状态 + 关键词」检索，/search 固定只看已发布目录，替代不了它。
	Keyword        string `json:"q" form:"q" binding:"omitempty,max=100"`
	CiTypeID       int    `json:"ciTypeId" form:"ciTypeId"`
	CloudServiceID int    `json:"cloudServiceId" form:"cloudServiceId"`
}

// GetServiceRequestsRequest 获取服务请求列表请求。
//
// page/pageSize 不在这里绑定：分页夹紧的唯一所有者是 HTTP 入口的
// common.GetPaginationFromQuery（缺省 1/20，只采纳 (0,100]，越界回落默认页长）。
// DTO 上的 binding 上限同样会造成两套真相：越界先返回 1001 参数错误而不是回落默认页长。
// 这里没有 UserID 字段，操作者身份只取认证上下文。
type GetServiceRequestsRequest struct {
	Status string `json:"status" form:"status" binding:"omitempty"`
}

// ServiceCatalogResponse 服务目录响应
type ServiceCatalogResponse struct {
	ID                int                    `json:"id"`
	Name              string                 `json:"name"`
	Category          string                 `json:"category"`
	Description       string                 `json:"description"`
	Icon              string                 `json:"icon,omitempty"`
	ServiceType       string                 `json:"serviceType,omitempty"`
	Price             float64                `json:"price,omitempty"`
	DeliveryTime      string                 `json:"deliveryTime"`
	Unit              string                 `json:"unit,omitempty"`
	RequiresApproval  bool                   `json:"requiresApproval"`
	ApprovalLevel     int                    `json:"approvalLevel,omitempty"`
	Approvers         []int                  `json:"approvers,omitempty"`
	SLAResponseTime   int                    `json:"slaResponseTime,omitempty"`
	SLAResolutionTime int                    `json:"slaResolutionTime,omitempty"`
	SLAID             int                    `json:"slaId,omitempty"`
	SLAName           string                 `json:"slaName,omitempty"`
	CITypeID          int                    `json:"ciTypeId"`
	CloudServiceID    int                    `json:"cloudServiceId"`
	FormSchema        map[string]interface{} `json:"formSchema,omitempty"`
	AvailableRegions  []string               `json:"availableRegions,omitempty"`
	AvailableSpecs    []string               `json:"availableSpecs,omitempty"`
	Status            string                 `json:"status"`
	IsActive          bool                   `json:"isActive"`
	SortOrder         int                    `json:"sortOrder,omitempty"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
}

// ServiceRequestResponse 服务请求响应
//
// 字段命名约定（与工单对齐）：
//   - RequestNumber: 服务请求标准编号（推荐使用，前端接口字段）
//   - TicketNumber: 兼容别名（与工单 TicketNumber 命名一致）
//   - FormData:    原生气表字段（DB 映射）
//   - FormFields:  兼容别名（与工单 formFields 命名一致）
type ServiceRequestResponse struct {
	ID            int            `json:"id"`
	RequestNumber string         `json:"requestNumber"`
	TicketNumber  string         `json:"ticketNumber,omitempty"`
	CatalogID     int            `json:"catalogId"`
	RequesterID   int            `json:"requesterId"`
	CIID          int            `json:"ciId,omitempty"`
	Status        string         `json:"status"`
	Title         string         `json:"title,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	FormData      map[string]any `json:"formData,omitempty"`
	FormFields    map[string]any `json:"formFields,omitempty"`

	CostCenter         string     `json:"costCenter,omitempty"`
	DataClassification string     `json:"dataClassification,omitempty"`
	NeedsPublicIP      bool       `json:"needsPublicIp"`
	SourceIPWhitelist  []string   `json:"sourceIpWhitelist,omitempty"`
	ExpireAt           *time.Time `json:"expireAt,omitempty"`
	ComplianceAck      bool       `json:"complianceAck"`

	CurrentLevel   int        `json:"currentLevel"`
	TotalLevels    int        `json:"totalLevels"`
	Version        int        `json:"version"`
	ProcessorID    *int       `json:"processorId,omitempty"`
	ApprovedAt     *time.Time `json:"approvedAt,omitempty"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	CompletionNote string     `json:"completionNote,omitempty"`
	LastError      string     `json:"lastError,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`

	Approvals []ServiceRequestApprovalResponse `json:"approvals,omitempty"`
	Catalog   *ServiceCatalogResponse          `json:"catalog,omitempty"`
	Requester *UserResponse                    `json:"requester,omitempty"`
}

// ServiceRequestApprovalResponse 服务请求审批记录响应
type ServiceRequestApprovalResponse struct {
	ID               int        `json:"id"`
	ServiceRequestID int        `json:"serviceRequestId"`
	Level            int        `json:"level"`
	Step             string     `json:"step"`
	Status           string     `json:"status"`
	ApproverID       *int       `json:"approverId,omitempty"`
	ApproverName     string     `json:"approverName,omitempty"`
	Action           string     `json:"action,omitempty"`
	Comment          string     `json:"comment,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	ProcessedAt      *time.Time `json:"processedAt,omitempty"`

	// V1 新增字段
	TimeoutHours     int        `json:"timeoutHours,omitempty"`     // 审批时限（小时）
	DueAt            *time.Time `json:"dueAt,omitempty"`            // 到期时间
	IsEscalated      bool       `json:"isEscalated,omitempty"`      // 是否已升级
	DelegatedToID    *int       `json:"delegatedToId,omitempty"`    // 转交审批人ID
	EscalationReason string     `json:"escalationReason,omitempty"` // 升级原因
}

// ServiceRequestApprovalActionRequest 审批动作请求
type ServiceRequestApprovalActionRequest struct {
	Action  string `json:"action" binding:"required,oneof=approve reject"`
	Comment string `json:"comment" binding:"omitempty,max=2000"`
}

// ServiceCatalogListResponse 服务目录列表响应
//
// 平台五键信封：集合键 items、页长键 pageSize（不接受 size 别名）、带 totalPages ——
// GET /service-catalogs、/service-catalog、/service-catalog-services 与
// /service-catalogs/search 都做 Count + Offset/Limit 真分页，缺 totalPages 会让调用方
// 把当前页当成整表。
type ServiceCatalogListResponse struct {
	Items      []ServiceCatalogResponse `json:"items"`
	Total      int                      `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"pageSize"`
	TotalPages int                      `json:"totalPages"`
}

// ToServiceCatalogResponse 转换为服务目录响应
func ToServiceCatalogResponse(catalog *ent.ServiceCatalog) *ServiceCatalogResponse {
	return &ServiceCatalogResponse{
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
		CITypeID:          catalog.CiTypeID,
		CloudServiceID:    catalog.CloudServiceID,
		FormSchema:        catalog.FormSchema,
		AvailableRegions:  catalog.AvailableRegions,
		AvailableSpecs:    catalog.AvailableSpecs,
		Status:            string(catalog.Status),
		IsActive:          catalog.IsActive,
		SortOrder:         catalog.SortOrder,
		CreatedAt:         catalog.CreatedAt,
		UpdatedAt:         catalog.UpdatedAt,
	}
}

// ToServiceRequestResponse 转换为服务请求响应
//
// RequestNumber / TicketNumber 生成策略：
//
//	数据库 service_requests 表当前未持久化 request_number 列
//	（避免 schema 迁移带来的发布与回滚成本），采用「派生编号」：
//	格式 SR-YYYYMM-NNNNNN，基于 ID + CreatedAt 生成。
//	后续如需“按实体持久化”可只补一列 request_number，不影响调用契约。
func ToServiceRequestResponse(request *ent.ServiceRequest) *ServiceRequestResponse {
	var expireAt *time.Time
	if !request.ExpireAt.IsZero() {
		t := request.ExpireAt
		expireAt = &t
	}
	resp := &ServiceRequestResponse{
		ID:                 request.ID,
		RequestNumber:      GenerateServiceRequestNumber(request.ID, request.CreatedAt),
		TicketNumber:       GenerateServiceRequestNumber(request.ID, request.CreatedAt),
		CatalogID:          request.CatalogID,
		RequesterID:        request.RequesterID,
		CIID:               request.CiID,
		Status:             string(request.Status),
		Title:              request.Title,
		Reason:             request.Reason,
		FormData:           request.FormData,
		FormFields:         request.FormData,
		CostCenter:         request.CostCenter,
		DataClassification: request.DataClassification,
		NeedsPublicIP:      request.NeedsPublicIP,
		SourceIPWhitelist:  request.SourceIPWhitelist,
		ExpireAt:           expireAt,
		ComplianceAck:      request.ComplianceAck,
		CurrentLevel:       request.CurrentLevel,
		TotalLevels:        request.TotalLevels,
		CreatedAt:          request.CreatedAt,
		UpdatedAt:          request.UpdatedAt,
	}

	return resp
}

// GenerateServiceRequestNumber 生成服务请求编号（派生，不写入数据库）。
// 格式：SR-YYYYMM-NNNNNN（N 不足 6 位补零）。
// CreatedAt 为零值时退回到当前月份。
func GenerateServiceRequestNumber(id int, createdAt time.Time) string {
	t := createdAt
	if t.IsZero() {
		t = time.Now()
	}
	return fmt.Sprintf("SR-%04d%02d-%06d", t.Year(), int(t.Month()), id)
}

func ToServiceRequestApprovalResponse(a *ent.ServiceRequestApproval) ServiceRequestApprovalResponse {
	var processedAt *time.Time
	if !a.ProcessedAt.IsZero() {
		t := a.ProcessedAt
		processedAt = &t
	}
	return ServiceRequestApprovalResponse{
		ID:               a.ID,
		ServiceRequestID: a.ServiceRequestID,
		Level:            a.Level,
		Step:             a.Step,
		Status:           a.Status,
		ApproverID:       a.ApproverID,
		ApproverName:     a.ApproverName,
		Action:           a.Action,
		Comment:          a.Comment,
		CreatedAt:        a.CreatedAt,
		ProcessedAt:      processedAt,
	}
}

// CreateServiceCatalogRequest 创建服务目录请求
type CreateServiceCatalogRequest struct {
	Name             string  `json:"name" binding:"required,max=255"`
	Category         string  `json:"category" binding:"required,max=100"`
	Description      string  `json:"description" binding:"omitempty,max=1000"`
	Icon             string  `json:"icon" binding:"omitempty,max=255"`
	ServiceType      string  `json:"serviceType" binding:"omitempty,oneof=vm rds oss network storage security custom"`
	Price            float64 `json:"price" binding:"omitempty"`
	DeliveryTime     string  `json:"deliveryTime" binding:"omitempty,max=50"`
	Unit             string  `json:"unit" binding:"omitempty,oneof=月 次 用户"`
	RequiresApproval *bool   `json:"requiresApproval"`
	// 0 是合法值：审批级别 0 表示「未配置」。此前 min=1 让新建目录无法停留在未配置。
	ApprovalLevel     int                    `json:"approvalLevel" binding:"omitempty,min=0,max=3"`
	Approvers         []int                  `json:"approvers"`
	SLAResponseTime   int                    `json:"slaResponseTime" binding:"omitempty"`
	SLAResolutionTime int                    `json:"slaResolutionTime" binding:"omitempty"`
	CITypeID          int                    `json:"ciTypeId,omitempty"`
	CloudServiceID    int                    `json:"cloudServiceId,omitempty"`
	FormSchema        map[string]interface{} `json:"formSchema"`
	AvailableRegions  []string               `json:"availableRegions"`
	AvailableSpecs    []string               `json:"availableSpecs"`
	Status            string                 `json:"status" binding:"omitempty,oneof=enabled disabled"`
	SortOrder         int                    `json:"sortOrder"`
}

// UpdateServiceCatalogRequest 更新服务目录请求
//
// 指针 = 调用方显式提交该字段，因此枚举/范围校验放在 service（nil 表示不动，
// ""/0/false 表示清空或关闭）。绑定层这里不能挂 oneof/min/max：gin 的校验器对
// 非空指针仍会跑 oneof，于是「清空计价单位」这种合法 PATCH 会被拦成通用
// 「请求参数错误」，字段级中文提示也就丢了。
type UpdateServiceCatalogRequest struct {
	Name              *string                 `json:"name" binding:"omitempty,max=255"`
	Category          *string                 `json:"category" binding:"omitempty,max=100"`
	Description       *string                 `json:"description" binding:"omitempty,max=1000"`
	Icon              *string                 `json:"icon" binding:"omitempty,max=255"`
	ServiceType       *string                 `json:"serviceType" binding:"omitempty,max=50"`
	Price             *float64                `json:"price"`
	DeliveryTime      *string                 `json:"deliveryTime" binding:"omitempty,max=50"`
	Unit              *string                 `json:"unit" binding:"omitempty,max=20"`
	RequiresApproval  *bool                   `json:"requiresApproval"`
	ApprovalLevel     *int                    `json:"approvalLevel"`
	Approvers         []int                   `json:"approvers"`
	SLAResponseTime   *int                    `json:"slaResponseTime"`
	SLAResolutionTime *int                    `json:"slaResolutionTime"`
	CITypeID          *int                    `json:"ciTypeId"`
	CloudServiceID    *int                    `json:"cloudServiceId"`
	FormSchema        *map[string]interface{} `json:"formSchema"`
	AvailableRegions  []string                `json:"availableRegions"`
	AvailableSpecs    []string                `json:"availableSpecs"`
	Status            *string                 `json:"status" binding:"omitempty,max=20"`
	SortOrder         *int                    `json:"sortOrder"`
}
