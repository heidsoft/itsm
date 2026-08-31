package common

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParseResourceID 把"读取 URL :id 参数 + Atoi + 失败时写入 400 ParamError"
// 的样板合并为一个 helper。当 Atoi 失败时，helper 已经向客户端写了
// ParamError(1001, "无效的{资源}ID") 并 abort 了请求链；调用方拿到
// id == 0 时直接 return 即可。
//
// 用法：
//
//	id, ok := common.ParseResourceID(ctx, "id", "事件")
//	if !ok {
//	    return
//	}
func ParseResourceID(c *gin.Context, paramName, resourceName string) (int, bool) {
	idStr := c.Param(paramName)
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		Fail(c, ParamErrorCode, "无效的"+resourceName+"ID")
		return 0, false
	}
	return id, true
}

// ParseOptionalQueryInt 读取查询参数中的可选整数，缺失或非法时返回
// (fallback, true)。永远不会写入错误响应，仅用于分页/筛选场景。
func ParseOptionalQueryInt(c *gin.Context, paramName string, fallback int) int {
	raw := c.Query(paramName)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}