package dto

// 本文件只声明云账号/云服务/云资源三个列表端点的响应契约与云资源列表的查询参数。
//
// 2026-10-04：`/api/v1/cloud/*` 那套与 `/api/v1/cmdb/cloud-*` 平行的表面（15 条已注册路由、
// handlers/cloud、service/cloud_service.go）已删除，它是同一用例的第二套业务规则所有者，
// 实测零前端调用方。它的请求 DTO 因此成为零生产引用的死契约，随该表面一并删除；
// 活表面的写入请求 DTO 是 cmdb_dto.go 里的 CMDB* 系列。

// ListCloudResourcesRequest 云资源列表查询参数（GET /api/v1/cmdb/cloud-resources）。
//
// 这里刻意不声明 page/pageSize：CMDB 列表页长的唯一所有者是 HTTP 入口的
// common.GetPaginationFromQuery（缺省 1/20，越界回落默认页长）。在请求 DTO 上再挂一套
// binding 夹紧就是第三套页长真相，两套规则会分叉。
type ListCloudResourcesRequest struct {
	Provider       string `form:"provider" comment:"云厂商过滤"`
	CloudAccountID int    `form:"cloudAccountId" comment:"云账号ID过滤"`
	ServiceID      int    `form:"serviceId" comment:"云服务类型ID过滤"`
	Region         string `form:"region" comment:"Region过滤"`
	Status         string `form:"status" comment:"资源状态过滤"`
	Search         string `form:"search" comment:"搜索关键词（云资源唯一ID前缀）"`
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
