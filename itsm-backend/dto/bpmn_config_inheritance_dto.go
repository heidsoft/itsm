package dto

import "errors"

// ErrConfigNotFound 表示全局 → 租户 → 部门 → 团队 各层级均未命中配置。
// 这是合法的空结果而非内部错误：调用方必须用 errors.Is 区分，并映射为
// data=null 的成功响应，不能返回 500/5001。
// 下沉说明：共享层保留同名别名，保证 errors.Is 包装链与空结果语义不变。
var ErrConfigNotFound = errors.New("config not found")

// 域配置继承模式是闭合枚举。写入口必须校验，解析入口遇到脏数据只能记日志跳过，
// 不能静默当成 inherit（历史上会让「新增配置」看起来保存成功却永远不生效）。
const (
	InheritModeInherit  = "inherit"
	InheritModeOverride = "override"
	InheritModeExtend   = "extend"
)

// IsValidInheritMode 报告 mode 是否为受支持的继承模式。
func IsValidInheritMode(mode string) bool {
	switch mode {
	case InheritModeInherit, InheritModeOverride, InheritModeExtend:
		return true
	default:
		return false
	}
}

// ResolvedConfig represents a fully resolved configuration
type ResolvedConfig struct {
	Key         string                 `json:"key"`
	Value       map[string]interface{} `json:"value"`
	Source      string                 `json:"source"` // global, tenant, department, team
	InheritMode string                 `json:"inheritMode"`
	Version     int                    `json:"version"`
}
