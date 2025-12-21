package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"itsm-backend/common"
)

// API版本常量
const (
	APIVersionV1 = "/api/v1"
)

// API端点配置
type Endpoints struct {
	Auth      string `json:"auth"`
	Users     string `json:"users"`
	Incidents string `json:"incidents"`
	Changes   string `json:"changes"`
	Services  string `json:"services"`
	Dashboard string `json:"dashboard"`
	SLA       string `json:"sla"`
	Reports   string `json:"reports"`
	Knowledge string `json:"knowledge"`
}

// 获取所有端点配置
func GetEndpoints() map[string]string {
	return map[string]string{
		"auth":      APIVersionV1 + "/auth",
		"users":     APIVersionV1 + "/users",
		"incidents": APIVersionV1 + "/incidents",
		"changes":   APIVersionV1 + "/changes",
		"services":  APIVersionV1 + "/services",
		"dashboard": APIVersionV1 + "/dashboard",
		"sla":       APIVersionV1 + "/sla",
		"reports":   APIVersionV1 + "/reports",
		"knowledge": APIVersionV1 + "/knowledge",
	}
}

// API路由组设置
func SetupRoutes(router *gin.Engine) {
	// API版本组
	v1 := router.Group(APIVersionV1)
	{
		// 添加API版本中间件
		v1.Use(APIMiddleware())
		
		// 健康检查
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "healthy",
				"version": "v1",
			})
		})
	}
}

// API中间件
func APIMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置API版本头
		c.Header("API-Version", "v1")
		c.Header("Content-Type", "application/json")
		
		// 处理CORS
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
		
		// 请求ID追踪
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = common.GenerateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}

// 标准化响应
type StandardResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
	TraceID   string      `json:"trace_id,omitempty"`
}

// 标准化分页响应
type PaginatedResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      struct {
		Items      interface{} `json:"items"`
		Total      int         `json:"total"`
		Page       int         `json:"page"`
		PageSize   int         `json:"page_size"`
		TotalPages int         `json:"total_pages"`
	} `json:"data"`
	Timestamp string `json:"timestamp,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

// 批量操作响应
type BatchResponse struct {
	Code      int `json:"code"`
	Message   string `json:"message"`
	Success   int `json:"success"`
	Failed    int `json:"failed"`
	Errors    []struct {
		ID    int    `json:"id"`
		Error string `json:"error"`
	} `json:"errors,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
	TraceID   string      `json:"trace_id,omitempty"`
}

// 标准化错误响应
type ErrorResponse struct {
	Code      int                    `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Field     string                 `json:"field,omitempty"`
	Timestamp string                 `json:"timestamp,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
}

// 成功响应辅助函数
func Success(c *gin.Context, data interface{}) {
	traceID, _ := c.Get("request_id")
	c.JSON(http.StatusOK, StandardResponse{
		Code:      common.SuccessCode,
		Message:   "success",
		Data:      data,
		Timestamp: common.GetTimestamp(),
		TraceID:   traceID.(string),
	})
}

// 分页响应辅助函数
func SuccessWithPagination(c *gin.Context, items interface{}, total, page, pageSize int) {
	traceID, _ := c.Get("request_id")
	totalPages := (total + pageSize - 1) / pageSize
	
	responseData := struct {
		Items      interface{} `json:"items"`
		Total      int         `json:"total"`
		Page       int         `json:"page"`
		PageSize   int         `json:"page_size"`
		TotalPages int         `json:"total_pages"`
	}{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
	
	c.JSON(http.StatusOK, PaginatedResponse{
		Code:    common.SuccessCode,
		Message: "success",
		Data:    responseData,
		TraceID: traceID.(string),
	})
}

// 错误响应辅助函数
func Error(c *gin.Context, code int, message string, details ...map[string]interface{}) {
	traceID, _ := c.Get("request_id")
	
	response := ErrorResponse{
		Code:      code,
		Message:   message,
		Timestamp: common.GetTimestamp(),
		TraceID:   traceID.(string),
	}
	
	if len(details) > 0 {
		response.Details = details[0]
	}
	
	c.JSON(getHTTPStatus(code), response)
}

// 批量操作响应辅助函数
func BatchOperationResponse(c *gin.Context, success, failed int, errors []struct {
	ID    int    `json:"id"`
	Error string `json:"error"`
}, data interface{}) {
	traceID, _ := c.Get("request_id")
	
	response := BatchResponse{
		Code:      common.SuccessCode,
		Message:   "batch operation completed",
		Success:   success,
		Failed:    failed,
		Data:      data,
		Timestamp: common.GetTimestamp(),
		TraceID:   traceID.(string),
	}
	
	if len(errors) > 0 {
		response.Errors = errors
	}
	
	c.JSON(http.StatusOK, response)
}

// 获取HTTP状态码
func getHTTPStatus(code int) int {
	switch {
	case code >= 1000 && code < 2000:
		return http.StatusBadRequest
	case code >= 2000 && code < 3000:
		return http.StatusUnauthorized
	case code >= 3000 && code < 4000:
		return http.StatusNotFound
	case code >= 4000 && code < 5000:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}