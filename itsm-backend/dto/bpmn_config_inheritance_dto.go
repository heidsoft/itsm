package dto

import "errors"

// ErrConfigNotFound 表示全局 → 租户 → 部门 → 团队 各层级均未命中配置。
// 这是合法的空结果而非内部错误：调用方必须用 errors.Is 区分，并映射为
// data=null 的成功响应，不能返回 500/5001。
// 下沉说明：共享层保留同名别名，保证 errors.Is 包装链与空结果语义不变。
var ErrConfigNotFound = errors.New("config not found")

// ResolvedConfig represents a fully resolved configuration
type ResolvedConfig struct {
	Key         string                 `json:"key"`
	Value       map[string]interface{} `json:"value"`
	Source      string                 `json:"source"` // global, tenant, department, team
	InheritMode string                 `json:"inheritMode"`
	Version     int                    `json:"version"`
}
