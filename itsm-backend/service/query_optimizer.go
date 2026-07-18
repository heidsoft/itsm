package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// QueryCache 实现了一个简单的查询缓存层，用于优化频繁查询的性能
type QueryCache struct {
	cache      map[string]cacheEntry
	mu         sync.RWMutex
	maxSize    int
	defaultTTL time.Duration
	logger     *zap.SugaredLogger
}

type cacheEntry struct {
	value      interface{}
	expireTime time.Time
}

// NewQueryCache 创建一个新的查询缓存
func NewQueryCache(maxSize int, defaultTTL time.Duration, logger *zap.SugaredLogger) *QueryCache {
	return &QueryCache{
		cache:      make(map[string]cacheEntry),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		logger:     logger,
	}
}

// Get 从缓存中获取值
func (qc *QueryCache) Get(key string) (interface{}, bool) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	entry, exists := qc.cache[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expireTime) {
		delete(qc.cache, key)
		return nil, false
	}

	return entry.value, true
}

// Set 将值存入缓存
func (qc *QueryCache) Set(key string, value interface{}, ttl ...time.Duration) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	// 如果缓存已满，删除过期的条目
	if len(qc.cache) >= qc.maxSize {
		qc.cleanExpired()
	}

	// 如果缓存仍然满，删除最早的条目
	if len(qc.cache) >= qc.maxSize {
		for k := range qc.cache {
			delete(qc.cache, k)
			break
		}
	}

	expireTTL := qc.defaultTTL
	if len(ttl) > 0 {
		expireTTL = ttl[0]
	}

	qc.cache[key] = cacheEntry{
		value:      value,
		expireTime: time.Now().Add(expireTTL),
	}
}

// Delete 从缓存中删除值
func (qc *QueryCache) Delete(key string) {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	delete(qc.cache, key)
}

// DeleteByPrefix 删除具有指定前缀的所有缓存条目
func (qc *QueryCache) DeleteByPrefix(prefix string) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	for key := range qc.cache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(qc.cache, key)
		}
	}
}

// Clear 清空所有缓存
func (qc *QueryCache) Clear() {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.cache = make(map[string]cacheEntry)
}

// cleanExpired 删除所有过期的缓存条目
func (qc *QueryCache) cleanExpired() {
	now := time.Now()
	for key, entry := range qc.cache {
		if now.After(entry.expireTime) {
			delete(qc.cache, key)
		}
	}
}

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(prefix string, params ...interface{}) string {
	key := prefix
	for _, param := range params {
		key += fmt.Sprintf(":%v", param)
	}
	return key
}

// QueryOptimizer 提供查询优化建议和辅助方法
type QueryOptimizer struct {
	logger *zap.SugaredLogger
}

func NewQueryOptimizer(logger *zap.SugaredLogger) *QueryOptimizer {
	return &QueryOptimizer{logger: logger}
}

// OptimizeListQuery 优化列表查询，添加分页和限制
func (qo *QueryOptimizer) OptimizeListQuery(page, pageSize int, maxPageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// CalculateOffset 计算分页偏移量
func (qo *QueryOptimizer) CalculateOffset(page, pageSize int) int {
	return (page - 1) * pageSize
}

// ShouldCache 判断查询结果是否应该被缓存
func (qo *QueryOptimizer) ShouldCache(queryType string, tenantID int) bool {
	cacheableQueries := []string{
		"ticket_list",
		"ticket_stats",
		"knowledge_list",
		"service_catalog",
		"user_list",
		"tenant_config",
	}

	for _, qt := range cacheableQueries {
		if qt == queryType {
			return true
		}
	}
	return false
}

// CachedQueryExecutor 执行可缓存的查询
func (qc *QueryCache) CachedQueryExecutor(
	ctx context.Context,
	cacheKey string,
	queryFunc func() (interface{}, error),
	ttl ...time.Duration,
) (interface{}, error) {

	// 尝试从缓存获取
	if cachedValue, ok := qc.Get(cacheKey); ok {
		qc.logger.Debugw("Cache hit", "key", cacheKey)
		return cachedValue, nil
	}

	qc.logger.Debugw("Cache miss", "key", cacheKey)

	// 执行查询
	value, err := queryFunc()
	if err != nil {
		return nil, err
	}

	// 存入缓存
	qc.Set(cacheKey, value, ttl...)

	return value, nil
}

// GlobalQueryCache 全局查询缓存实例
var GlobalQueryCache *QueryCache

// InitQueryCache 初始化全局查询缓存
func InitQueryCache(maxSize int, defaultTTL time.Duration, logger *zap.SugaredLogger) {
	GlobalQueryCache = NewQueryCache(maxSize, defaultTTL, logger)
	logger.Infow("Query cache initialized", "maxSize", maxSize, "defaultTTL", defaultTTL)
}
