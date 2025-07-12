package dto

import (
	"itsm-backend/ent"
	"time"
)

// CreateServiceRequestRequest 创建服务请求请求
type CreateServiceRequestRequest struct {
	CatalogID int    `json:"catalog_id" binding:"required,min=1"`
	Reason    string `json:"reason" binding:"omitempty,max=500"`
}

// UpdateServiceRequestStatusRequest 更新服务请求状态请求
type UpdateServiceRequestStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending in_progress completed rejected"`
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
	Page      int    `json:"page" form:"page" binding:"omitempty,min=1"`
	Size      int    `json:"size" form:"size" binding:"omitempty,min=1,max=100"`
	Status    string `json:"status" form:"status" binding:"omitempty,oneof=pending in_progress completed rejected"`
	UserID    int    `json:"-"` // 从认证中间件获取
}

// ServiceCatalogResponse 服务目录响应
type ServiceCatalogResponse struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Category     string    `json:"category"`
	Description  string    `json:"description"`
	DeliveryTime string    `json:"delivery_time"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ServiceRequestResponse 服务请求响应
type ServiceRequestResponse struct {
	ID          int                      `json:"id"`
	CatalogID   int                      `json:"catalog_id"`
	RequesterID int                      `json:"requester_id"`
	Status      string                   `json:"status"`
	Reason      string                   `json:"reason"`
	CreatedAt   time.Time                `json:"created_at"`
	Catalog     *ServiceCatalogResponse  `json:"catalog,omitempty"`
	Requester   *UserResponse            `json:"requester,omitempty"`
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
	Requests []ServiceRequestResponse `json:"requests"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	Size     int                      `json:"size"`
}

// ToServiceCatalogResponse 转换为服务目录响应
func ToServiceCatalogResponse(catalog *ent.ServiceCatalog) *ServiceCatalogResponse {
	return &ServiceCatalogResponse{
		ID:           catalog.ID,
		Name:         catalog.Name,
		Category:     catalog.Category,
		Description:  catalog.Description,
		DeliveryTime: catalog.DeliveryTime,
		Status:       string(catalog.Status),
		CreatedAt:    catalog.CreatedAt,
		UpdatedAt:    catalog.UpdatedAt,
	}
}

// ToServiceRequestResponse 转换为服务请求响应
func ToServiceRequestResponse(request *ent.ServiceRequest) *ServiceRequestResponse {
	resp := &ServiceRequestResponse{
		ID:          request.ID,
		CatalogID:   request.CatalogID,
		RequesterID: request.RequesterID,
		Status:      string(request.Status),
		Reason:      request.Reason,
		CreatedAt:   request.CreatedAt,
	}
	
	// 关联数据
	if request.Edges.Catalog != nil {
		resp.Catalog = ToServiceCatalogResponse(request.Edges.Catalog)
	}
	
	if request.Edges.Requester != nil {
		resp.Requester = &UserResponse{
			ID:         request.Edges.Requester.ID,
			Username:   request.Edges.Requester.Username,
			Name:       request.Edges.Requester.Name,
			Email:      request.Edges.Requester.Email,
			Department: request.Edges.Requester.Department,
		}
	}
	
	return resp
}

// CreateServiceCatalogRequest 创建服务目录请求
type CreateServiceCatalogRequest struct {
	Name         string `json:"name" binding:"required,max=255"`
	Category     string `json:"category" binding:"required,max=100"`
	Description  string `json:"description" binding:"omitempty,max=1000"`
	DeliveryTime string `json:"delivery_time" binding:"required,max=50"`
	Status       string `json:"status" binding:"omitempty,oneof=enabled disabled"`
}

// UpdateServiceCatalogRequest 更新服务目录请求
type UpdateServiceCatalogRequest struct {
	Name         string `json:"name" binding:"omitempty,max=255"`
	Category     string `json:"category" binding:"omitempty,max=100"`
	Description  string `json:"description" binding:"omitempty,max=1000"`
	DeliveryTime string `json:"delivery_time" binding:"omitempty,max=50"`
	Status       string `json:"status" binding:"omitempty,oneof=enabled disabled"`
}

