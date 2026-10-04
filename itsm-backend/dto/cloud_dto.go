package dto

// 本文件只声明云账号/云服务/云资源三个列表端点的响应契约与云资源列表的查询参数。
//
// 2026-10-04：`/api/v1/cloud/*` 那套与 `/api/v1/cmdb/cloud-*` 平行的表面（15 条已注册路由、
// handlers/cloud、service/cloud_service.go）已删除，它是同一用例的第二套业务规则所有者，
// 实测零前端调用方。它的请求 DTO 因此成为零生产引用的死契约，随该表面一并删除；
// 活表面的写入请求 DTO 是 cmdb_dto.go 里的 CMDB* 系列。

// ListCloudSelectorRequest 云账号/云服务两个选择器列表的查询参数
// （GET /api/v1/cmdb/cloud-accounts、GET /api/v1/cmdb/cloud-services）。
//
// 两个端点是同一份契约（只有一个 provider 过滤），所以共用一个 DTO，而不是各写一遍。
// Provider 走枚举校验：过去 handler 直接 c.Query("provider") 把任意字符串塞进精确匹配，
// 拼错的厂商名表现为「这个租户没有云账号」的空列表，而不是请求错误。
type ListCloudSelectorRequest struct {
	Provider string `form:"provider" binding:"omitempty,oneof=aliyun tencent huawei aws azure onprem"`
}

// ListCloudResourcesRequest 云资源列表查询参数（GET /api/v1/cmdb/cloud-resources）。
//
// 这里刻意不声明 page/pageSize：CMDB 列表页长的唯一所有者是 HTTP 入口的
// common.GetPaginationFromQuery（缺省 1/20，越界回落默认页长）。在请求 DTO 上再挂一套
// binding 夹紧就是第三套页长真相，两套规则会分叉。
type ListCloudResourcesRequest struct {
	Provider       string `form:"provider" binding:"omitempty,oneof=aliyun tencent huawei aws azure onprem" comment:"云厂商过滤"`
	CloudAccountID int    `form:"cloudAccountId" comment:"云账号ID过滤"`
	ServiceID      int    `form:"serviceId" comment:"云服务类型ID过滤"`
	Region         string `form:"region" comment:"Region过滤"`
	Status         string `form:"status" comment:"资源状态过滤"`
	Search         string `form:"search" comment:"搜索关键词（云资源唯一ID前缀）"`
}

// CreateCloudResourceRequest 手工登记云资源（POST /api/v1/cmdb/cloud-resources）。
//
// 归属租户、identity 相关字段和 firstSeenAt/lastSeenAt 由 service 层从认证上下文与
// 云账号/云服务派生，请求体无法自报；这里只能声明调用方真正可写的字段。
// 三份字段与 handler 内联匿名 struct 一致，抽到 DTO 层是为了让契约可被 swagger 引用。
type CreateCloudResourceRequest struct {
	CloudAccountID int                    `json:"cloudAccountId" binding:"required"`
	ServiceID      int                    `json:"serviceId" binding:"required"`
	ResourceID     string                 `json:"resourceId" binding:"required"`
	ResourceName   string                 `json:"resourceName"`
	Region         string                 `json:"region"`
	Zone           string                 `json:"zone"`
	Status         string                 `json:"status"`
	Tags           map[string]string      `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata"`
	LifecycleState string                 `json:"lifecycleState"`
}

// UpdateCloudResourceRequest 更新云资源（PUT /api/v1/cmdb/cloud-resources/:id）。
//
// 与 Create 的区别只有必填约束：本端点沿用 PUT 全量覆盖语义，缺席字段按零值写入，
// 「未传」与「传零值」的区分属于 PATCH 契约，尚未拍板（见 ledger E4-26 同族问题）。
type UpdateCloudResourceRequest struct {
	CloudAccountID int                    `json:"cloudAccountId"`
	ServiceID      int                    `json:"serviceId"`
	ResourceID     string                 `json:"resourceId"`
	ResourceName   string                 `json:"resourceName"`
	Region         string                 `json:"region"`
	Zone           string                 `json:"zone"`
	Status         string                 `json:"status"`
	Tags           map[string]string      `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata"`
	LifecycleState string                 `json:"lifecycleState"`
}

// CloudResourceListResponse 云资源列表响应。
//
// 真实分页：total 来自仓储的 Count，items 是当前页；page/pageSize/totalPages 反映生效值。
type CloudResourceListResponse struct {
	Items      []*CloudResourceResponse `json:"items"`
	Total      int                      `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"pageSize"`
	TotalPages int                      `json:"totalPages"`
}

// CloudAccountListResponse 云账号列表响应。
//
// 实测不分页：本端点是 CI 表单与云资源页的选择器数据源，整份返回，因此只保留诚实的
// {items,total} 两键，不得补 page/pageSize/totalPages 假键。
type CloudAccountListResponse struct {
	Items []*CloudAccountResponse `json:"items"`
	Total int                     `json:"total"`
}

// CloudServiceListResponse 云服务列表响应。
//
// 同 CloudAccountListResponse：作为 CI 表单的级联选择数据源整份返回，不分页。
type CloudServiceListResponse struct {
	Items []*CloudServiceResponse `json:"items"`
	Total int                     `json:"total"`
}
