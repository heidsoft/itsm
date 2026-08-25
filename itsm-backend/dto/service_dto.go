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

// GetServiceCatalogsRequest 获取服务目录请求
type GetServiceCatalogsRequest struct {
	Page     int    `json:"page" form:"page" binding:"omitempty,min=1"`
	Size     int    `json:"size" form:"size" binding:"omitempty,min=1,max=100"`
	Category string `json:"category" form:"category"`
	Status   string `json:"status" form:"status" binding:"omitempty,oneof=enabled disabled"`
}

// GetServiceRequestsRequest 获取服务请求列表请求
type GetServiceRequestsRequest struct {
	Page   int    `json:"page" form:"page" binding:"omitempty,min=1"`
	Size   int    `json:"size" form:"size" binding:"omitempty,min=1,max=100"`
	Status string `json:"status" form:"status" binding:"omitempty"`
	UserID int    `json:"-"` // 从认证中间件获取
}

// ServiceCatalogResponse 服务目录响应
type ServiceCatalogResponse struct {
	ID                 int                   `json:"id"`
	Name               string                `json:"name"`
	Category           string                `json:"category"`
	Description        string                `json:"description"`
	Icon               string                `json:"icon,omitempty"`
	ServiceType        string                `json:"serviceType,omitempty"`
	Price              float64               `json:"price,omitempty"`
	DeliveryTime       string                `json:"deliveryTime"`
	Unit               string                `json:"unit,omitempty"`
	RequiresApproval   bool                  `json:"requiresApproval,omitempty"`
	ApprovalLevel      int                   `json:"approvalLevel,omitempty"`
	Approvers          []int                 `json:"approvers,omitempty"`
	SLAResponseTime    int                   `json:"slaResponseTime,omitempty"`
	SLAResolutionTime  int                   `json:"slaResolutionTime,omitempty"`
	SLAID              int                   `json:"slaId,omitempty"`
	SLAName            string                `json:"slaName,omitempty"`
	CITypeID           int                   `json:"ciTypeId,omitempty"`
	CloudServiceID     int                   `json:"cloudServiceId,omitempty"`
	FormSchema         map[string]interface{} `json:"formSchema,omitempty"`
	AvailableRegions   []string              `json:"availableRegions,omitempty"`
	AvailableSpecs     []string              `json:"availableSpecs,omitempty"`
	Status             string                `json:"status"`
	IsActive           bool                  `json:"isActive,omitempty"`
	SortOrder          int                   `json:"sortOrder,omitempty"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
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
type ServiceCatalogListResponse struct {
	Catalogs []ServiceCatalogResponse `json:"catalogs"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	Size     int                      `json:"size"`
}

// ServiceRequestListResponse 服务请求列表响应
type ServiceRequestListResponse struct {
	Items []ServiceRequestResponse `json:"items"`
	Total int                      `json:"total"`
	Page  int                      `json:"page"`
	Size  int                      `json:"size"`
}

// ToServiceCatalogResponse 转换为服务目录响应
func ToServiceCatalogResponse(catalog *ent.ServiceCatalog) *ServiceCatalogResponse {
	return &ServiceCatalogResponse{
		ID:                 catalog.ID,
		Name:               catalog.Name,
		Category:           catalog.Category,
		Description:        catalog.Description,
		Icon:               catalog.Icon,
		ServiceType:        catalog.ServiceType,
		Price:              catalog.Price,
		DeliveryTime:       strconv.Itoa(catalog.DeliveryTime),
		Unit:               catalog.Unit,
		RequiresApproval:   catalog.RequiresApproval,
		ApprovalLevel:      catalog.ApprovalLevel,
		Approvers:          catalog.Approvers,
		SLAResponseTime:    catalog.SLAResponseTime,
		SLAResolutionTime:  catalog.SLAResolutionTime,
		CITypeID:           catalog.CiTypeID,
		CloudServiceID:     catalog.CloudServiceID,
		FormSchema:         catalog.FormSchema,
		AvailableRegions:   catalog.AvailableRegions,
		AvailableSpecs:     catalog.AvailableSpecs,
		Status:             string(catalog.Status),
		IsActive:           catalog.IsActive,
		SortOrder:          catalog.SortOrder,
		CreatedAt:          catalog.CreatedAt,
		UpdatedAt:          catalog.UpdatedAt,
	}
}

// ToServiceRequestResponse 转换为服务请求响应
//
// RequestNumber / TicketNumber 生成策略：
//   数据库 service_requests 表当前未持久化 request_number 列
//   （避免 schema 迁移带来的发布与回滚成本），采用「派生编号」：
//   格式 SR-YYYYMM-NNNNNN，基于 ID + CreatedAt 生成。
//   后续如需“按实体持久化”可只补一列 request_number，不影响调用契约。
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
	Name              string                 `json:"name" binding:"required,max=255"`
	Category          string                 `json:"category" binding:"required,max=100"`
	Description       string                 `json:"description" binding:"omitempty,max=1000"`
	Icon              string                 `json:"icon" binding:"omitempty,max=255"`
	ServiceType       string                 `json:"serviceType" binding:"omitempty,oneof=vm rds oss network storage security custom"`
	Price             float64                `json:"price" binding:"omitempty"`
	DeliveryTime      string                 `json:"deliveryTime" binding:"omitempty,max=50"`
	Unit              string                 `json:"unit" binding:"omitempty,oneof=月 次 用户"`
	RequiresApproval  *bool                  `json:"requiresApproval"`
	ApprovalLevel     int                    `json:"approvalLevel" binding:"omitempty,min=1,max=3"`
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
type UpdateServiceCatalogRequest struct {
	Name              *string                 `json:"name" binding:"omitempty,max=255"`
	Category          *string                 `json:"category" binding:"omitempty,max=100"`
	Description       *string                 `json:"description" binding:"omitempty,max=1000"`
	Icon              *string                 `json:"icon" binding:"omitempty,max=255"`
	ServiceType       *string                 `json:"serviceType" binding:"omitempty,oneof=vm rds oss network storage security custom"`
	Price             *float64                `json:"price"`
	DeliveryTime      *string                 `json:"deliveryTime" binding:"omitempty,max=50"`
	Unit              *string                 `json:"unit" binding:"omitempty,oneof=月 次 用户"`
	RequiresApproval  *bool                   `json:"requiresApproval"`
	ApprovalLevel     *int                    `json:"approvalLevel" binding:"omitempty,min=1,max=3"`
	Approvers         []int                   `json:"approvers"`
	SLAResponseTime   *int                    `json:"slaResponseTime"`
	SLAResolutionTime *int                    `json:"slaResolutionTime"`
	CITypeID          *int                    `json:"ciTypeId"`
	CloudServiceID    *int                    `json:"cloudServiceId"`
	FormSchema        *map[string]interface{} `json:"formSchema"`
	AvailableRegions  []string                `json:"availableRegions"`
	AvailableSpecs    []string                `json:"availableSpecs"`
	Status            *string                 `json:"status" binding:"omitempty,oneof=enabled disabled"`
	SortOrder         *int                    `json:"sortOrder"`
}
