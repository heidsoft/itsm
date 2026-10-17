package common

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParsePositiveID 从 URL 参数解析正整数 ID
// 非法或非正数统一返回 ParamError (code: 1001, status: 400)
//
// 参数:
//   - c: Gin 上下文
//   - paramName: URL 参数名称
//
// 返回:
//   - int: 解析到的正整数 ID
//   - bool: 是否解析成功
//
// 示例:
//
//	id, ok := ParsePositiveID(c, "id")
//	if !ok {
//	    return // 参数校验失败，错误已写入响应
//	}
//	// 使用 id 进行后续操作
func ParsePositiveID(c *gin.Context, paramName string) (int, bool) {
	idStr := c.Param(paramName)
	if idStr == "" {
		ParamError(c, paramName+" is required")
		return 0, false
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		ParamError(c, paramName+" must be a positive integer")
		return 0, false
	}

	return id, true
}

// ParsePositiveIDFromQuery 从查询参数解析正整数
// 如果参数为空则返回 (0, false)，不报错
// 与 ParsePositiveID 的区别：
//   - ParsePositiveID: 参数为空时返回 ParamError
//   - ParsePositiveIDFromQuery: 参数为空时静默返回 (0, false)
//
// 适用场景:
//   - 可选参数：?page=1&pageSize=20
//   - 必需参数：/tickets/:id
//
// 示例:
//
//	// 获取可选的分页参数
//	page, ok := ParsePositiveIDFromQuery(c, "page")
//	if !ok {
//	    page = 1 // 默认第一页
//	}
func ParsePositiveIDFromQuery(c *gin.Context, paramName string) (int, bool) {
	idStr := c.Query(paramName)
	if idStr == "" {
		return 0, false
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		ParamError(c, paramName+" must be a positive integer")
		return 0, false
	}

	return id, true
}

// ParseRequiredPositiveIDFromQuery 从查询参数解析**必填**正整数。
//
// 与 ParsePositiveIDFromQuery 的区别：参数缺失时写 ParamError（code 1001 / 400），
// 而不是静默返回 (0, false)。与 ParsePositiveID 的区别：读的是 query 而非 path。
//
// 存在理由（2026-10-18 缺陷 P0-1）：
// knowledge 的版本对比曾用 ParsePositiveID 取 `from`/`to`，而该函数读 c.Param——
// 路由 `GET /articles/:id/versions/compare` 只有 `:id`，于是 `from` 恒为空，
// 端点永久返回 `400 from is required`。参数在 query 里就必须用本函数，
// 用 ParsePositiveID 会在编译期与运行期都静默通过，只在真实请求时暴露。
func ParseRequiredPositiveIDFromQuery(c *gin.Context, paramName string) (int, bool) {
	idStr := c.Query(paramName)
	if idStr == "" {
		ParamError(c, paramName+" is required")
		return 0, false
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		ParamError(c, paramName+" must be a positive integer")
		return 0, false
	}

	return id, true
}

// MustParsePositiveID 从 URL 参数解析正整数 ID，失败时终止
// 注意：这是便捷函数，仅适用于确定参数存在的情况
//
// 与 ParsePositiveID 的区别:
//   - ParsePositiveID: 返回 (0, false) 并写入错误响应
//   - MustParsePositiveID: 返回 0，不写入响应（调用方需自行处理）
//
// 警告: 仅在确信参数必定存在时使用！
//
// 示例:
//
//	// URL: /tickets/:id 已被路由中间件验证必填
//	id := MustParsePositiveID(c, "id")
//	if id == 0 {
//	    c.JSON(400, gin.H{"error": "id is required"})
//	    return
//	}
func MustParsePositiveID(c *gin.Context, paramName string) int {
	id, ok := ParsePositiveID(c, paramName)
	if !ok {
		return 0
	}
	return id
}
