package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"itsm-backend/pkg/cache"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CacheMiddleware 缓存中间件
type CacheMiddleware struct {
	cache  *cache.CacheService
	logger *zap.Logger
}

// NewCacheMiddleware 创建缓存中间件
func NewCacheMiddleware(cache *cache.CacheService, logger *zap.Logger) *CacheMiddleware {
	return &CacheMiddleware{
		cache:  cache,
		logger: logger,
	}
}

// CacheResponse 缓存响应中间件
func (cm *CacheMiddleware) CacheResponse(ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只缓存GET请求
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// 生成缓存键
		cacheKey := cm.generateCacheKey(c)

		// 尝试从缓存获取
		var cachedResponse bytes.Buffer
		err := cm.cache.Get(c.Request.Context(), cacheKey, &cachedResponse)
		if err == nil {
			cm.logger.Debug("Cache hit", zap.String("key", cacheKey))
			c.Data(http.StatusOK, "application/json; charset=utf-8", cachedResponse.Bytes())
			c.Abort()
			return
		}

		// 缓存未命中，继续处理请求
		cm.logger.Debug("Cache miss", zap.String("key", cacheKey))

		// 包装响应写入器
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		c.Next()

		// 只缓存成功的响应
		if c.Writer.Status() == http.StatusOK && writer.body.Len() > 0 {
			if err := cm.cache.Set(c.Request.Context(), cacheKey, writer.body.Bytes(), ttl); err != nil {
				cm.logger.Warn("Failed to cache response", zap.Error(err))
			} else {
				cm.logger.Debug("Response cached", zap.String("key", cacheKey))
			}
		}
	}
}

// InvalidateCache 失效缓存中间件
func (cm *CacheMiddleware) InvalidateCache(patterns ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 只在成功的POST/PUT/DELETE请求后失效缓存
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 && 
			(c.Request.Method == http.MethodPost || 
			 c.Request.Method == http.MethodPut || 
			 c.Request.Method == http.MethodDelete) {
			
			for _, pattern := range patterns {
				if err := cm.cache.DeleteByPattern(c.Request.Context(), pattern); err != nil {
					cm.logger.Warn("Failed to invalidate cache", 
						zap.Error(err), 
						zap.String("pattern", pattern))
				} else {
					cm.logger.Debug("Cache invalidated", zap.String("pattern", pattern))
				}
			}
		}
	}
}

// generateCacheKey 生成缓存键
func (cm *CacheMiddleware) generateCacheKey(c *gin.Context) string {
	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	path := c.Request.URL.Path
	query := c.Request.URL.RawQuery

	return fmt.Sprintf("api:tenant:%d:user:%d:path:%s:query:%s", 
		tenantID, userID, path, query)
}

// responseWriter 响应写入器包装
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

// CacheStats 缓存统计中间件
func (cm *CacheMiddleware) CacheStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		
		// 记录慢查询
		if duration > 1*time.Second {
			cm.logger.Warn("Slow request detected",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Duration("duration", duration),
				zap.Int("status", c.Writer.Status()),
			)
		}
	}
}

// Cacheable 缓存标记（用于控制器方法）
type Cacheable struct {
	Key     string
	TTL     time.Duration
	InvalidateOn []string
}

// CacheHelper 缓存辅助函数
type CacheHelper struct {
	cache *cache.CacheService
	logger *zap.Logger
}

func NewCacheHelper(cache *cache.CacheService, logger *zap.Logger) *CacheHelper {
	return &CacheHelper{cache: cache, logger: logger}
}

// CacheTicketList 缓存工单列表
func (h *CacheHelper) CacheTicketList(ctx *gin.Context, tenantID int, page, pageSize int, fn func() (interface{}, error)) (interface{}, error) {
	key := fmt.Sprintf("tickets:tenant:%d:page:%d:size:%d", tenantID, page, pageSize)
	
	var result interface{}
	err := h.cache.GetOrSet(ctx.Request.Context(), key, &result, 5*time.Minute, fn)
	return result, err
}

// InvalidateTicketCache 失效工单缓存
func (h *CacheHelper) InvalidateTicketCache(ctx *gin.Context, tenantID int) error {
	pattern := fmt.Sprintf("tickets:tenant:%d:*", tenantID)
	return h.cache.DeleteByPattern(ctx.Request.Context(), pattern)
}

// CacheCMDBData 缓存CMDB数据
func (h *CacheHelper) CacheCMDBData(ctx *gin.Context, tenantID, ciID int, fn func() (interface{}, error)) (interface{}, error) {
	key := fmt.Sprintf("cmdb:tenant:%d:ci:%d", tenantID, ciID)
	
	var result interface{}
	err := h.cache.GetOrSet(ctx.Request.Context(), key, &result, 10*time.Minute, fn)
	return result, err
}

// InvalidateCMDBCache 失效CMDB缓存
func (h *CacheHelper) InvalidateCMDBCache(ctx *gin.Context, tenantID int) error {
	pattern := fmt.Sprintf("cmdb:tenant:%d:*", tenantID)
	return h.cache.DeleteByPattern(ctx.Request.Context(), pattern)
}

// ParsePagination 解析分页参数
func ParsePagination(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return
}
