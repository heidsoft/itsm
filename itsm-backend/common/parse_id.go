package common

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParsePositiveID 从 URL 参数解析正整数 ID
// 非法或非正数统一返回 ParamError (code: 1001, status: 400)
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

// MustParsePositiveID 从 URL 参数解析正整数 ID，失败时终止
// 注意：这是便捷函数，仅适用于确定参数存在的情况
func MustParsePositiveID(c *gin.Context, paramName string) int {
	id, ok := ParsePositiveID(c, paramName)
	if !ok {
		return 0
	}
	return id
}
