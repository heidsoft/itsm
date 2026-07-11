package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrNoTenantContext 是 ResolveRequestTenantID 在租户中间件未挂载时返回的
// 哨兵错误，控制器可以选择把它翻成 401 或者直接当成错误返回。
var ErrNoTenantContext = errNoTenantContext{}

type errNoTenantContext struct{}

func (errNoTenantContext) Error() string { return "租户上下文不存在" }

// ResolveRequestTenantID 返回当前请求应当使用的有效租户 ID。
//
// 决策表：
//
//   - 租户中间件未挂载：返回 ErrNoTenantContext（fail-closed）
//   - 普通用户：返回用户自己的 home tenant
//   - MSP 用户已锁定 CustomerTenantID：先校验该客户在 AllowedCustomers 中，
//     然后返回客户租户 ID；否则返回 ErrMSPCustomerDenied
//
// 调用方通常无需关心是否是 MSP；只需要把返回值当作"业务查询要过滤的
// tenant_id"传入 Service。Service 层已经按 tenantID 做了强过滤，所以
// 一旦解析器返回错误的租户，跨租户自然就拦住了。
func ResolveRequestTenantID(c *gin.Context) (int, error) {
	tenantCtx, exists := GetTenantContext(c)
	if !exists {
		return 0, ErrNoTenantContext
	}

	mspCtx, hasMSP := GetMSPContext(c)
	if !hasMSP || !mspCtx.IsMSP {
		return tenantCtx.TenantID, nil
	}

	// MSP 用户：如果没有锁定客户，保持原来的 home tenant（MSP 自家租户）。
	if mspCtx.CustomerTenantID == nil {
		return tenantCtx.TenantID, nil
	}

	target := *mspCtx.CustomerTenantID
	allowed := false
	for _, id := range mspCtx.AllowedCustomers {
		if id == target {
			allowed = true
			break
		}
	}
	if !allowed {
		return 0, ErrMSPCustomerDenied
	}
	return target, nil
}

// ErrMSPCustomerDenied 表示 MSP 用户访问了未被分配的客户租户。
var ErrMSPCustomerDenied = errMSPCustomerDenied{}

type errMSPCustomerDenied struct{}

func (errMSPCustomerDenied) Error() string { return "MSP员工无权访问该客户租户" }

// AbortIfTenantError 是控制器层常用的样板：如果解析租户失败，统一返回
// 401 / 403 + 标准响应体，避免每个控制器重复写。
func AbortIfTenantError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch err {
	case ErrNoTenantContext:
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    2001,
			"message": "租户上下文缺失",
		})
	case ErrMSPCustomerDenied:
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code":    2003,
			"message": "MSP员工无权访问该客户租户",
		})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"code":    5001,
			"message": "租户解析失败",
			"data":    gin.H{"error": err.Error()},
		})
	}
	return true
}
