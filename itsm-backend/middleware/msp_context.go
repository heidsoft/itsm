package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
)

// MSPContextKey 是存放在 gin.Context 上的 MSP 上下文键名。
const MSPContextKey = "msp_context"

// MSPContext 描述了当前请求的 MSP 访问上下文。MSPMiddleware 在请求进入
// 时构造它并放入 gin.Context，下游中间件和服务层通过 GetMSPContext 读取。
type MSPContext struct {
	// IsMSP 表示当前用户是否为 MSP 员工。仅当用户租户类型为 msp_provider
	// 且 msp_role 非空时才为 true，admin 角色不再自动授予 MSP 访问权。
	IsMSP bool
	// MSPUserID 是发起请求的 MSP 员工 ID。
	MSPUserID int
	// CustomerTenantID 是当前请求锁定的客户租户。当 MSP 员工通过
	// X-Customer-Tenant-ID 头或 URL 参数指定客户后被设置；nil 表示未限定。
	CustomerTenantID *int
	// Role 是用户原始 msp_role 字段值（provider_admin / provider_agent / ...）。
	Role string
	// AllowedCustomers 是该 MSP 员工当前所有有效分配的客户租户 ID 列表。
	AllowedCustomers []int
}

// GetMSPContext 从 gin.Context 取出 MSP 上下文。返回 (nil, false) 时表示
// 当前请求不是 MSP 请求（普通租户用户或匿名用户）。
func GetMSPContext(c *gin.Context) (*MSPContext, bool) {
	val, exists := c.Get(MSPContextKey)
	if !exists {
		return nil, false
	}
	ctx, ok := val.(*MSPContext)
	return ctx, ok
}

// mspCustomerTenantIDKey 是放在 Go context 上的 typed key，避免与其他
// 中间件传值的字符串 key 冲突。
type mspCustomerTenantIDKey struct{}

// MSPCustomerTenantIDKey 暴露给 Service 层使用，配合 Set/GetMSPCustomerTenantID。
var MSPCustomerTenantIDKey = mspCustomerTenantIDKey{}

// SetMSPCustomerTenantID 把客户租户 ID 写入 Go context，供 Service 层在
// ent 查询里加 TenantIDEQ 过滤。
func SetMSPCustomerTenantID(ctx context.Context, tenantID int) context.Context {
	return context.WithValue(ctx, MSPCustomerTenantIDKey, tenantID)
}

// GetMSPCustomerTenantID 从 Go context 取出当前请求锁定的客户租户 ID。
// 返回 (0, false) 表示非 MSP 请求或尚未限定客户。
func GetMSPCustomerTenantID(ctx context.Context) (int, bool) {
	val := ctx.Value(MSPCustomerTenantIDKey)
	if val == nil {
		return 0, false
	}
	tenantID, ok := val.(int)
	return tenantID, ok
}

// GetMSPContextFromContext 是 Service 层从 Go context 重建 MSPContext 的兼容
// 接口。Go context 只携带了 CustomerTenantID，因此 IsMSP 始终为 true，
// AllowedCustomers / Role 不可用，需要的字段应直接走 gin 上下文。
func GetMSPContextFromContext(ctx context.Context) (*MSPContext, bool) {
	tenantID, ok := GetMSPCustomerTenantID(ctx)
	if !ok || tenantID == 0 {
		return nil, false
	}
	return &MSPContext{
		IsMSP:            true,
		CustomerTenantID: &tenantID,
	}, true
}
