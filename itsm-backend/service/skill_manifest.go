package service

import "context"

// SkillManifest 技能自描述信息，用于市场展示和自动装配
type SkillManifest struct {
	Name                string      `json:"name"`                           // 唯一标识名称
	Version             string      `json:"version"`                        // 版本号
	Title               string      `json:"title"`                          // 显示名称
	Provider            string      `json:"provider"`                       // 提供商
	Description         string      `json:"description"`                    // 描述
	LongDescription     string      `json:"long_description,omitempty"`     // 详细描述
	IconURL             string      `json:"icon_url,omitempty"`             // 图标URL
	Screenshots         []string    `json:"screenshots,omitempty"`          // 截图列表
	Tags                []string    `json:"tags,omitempty"`                 // 标签列表
	Category            string      `json:"category,omitempty"`             // 分类
	Capabilities        []string    `json:"capabilities,omitempty"`         // 支持的能力列表
	InputSchema         interface{} `json:"input_schema,omitempty"`         // 输入参数JSON Schema
	OutputSchema        interface{} `json:"output_schema,omitempty"`        // 输出结果JSON Schema
	RequiredPermissions []string    `json:"required_permissions,omitempty"` // 需要的系统权限
	MinSystemVersion    string      `json:"min_system_version,omitempty"`   // 最低支持的系统版本
	Author              string      `json:"author,omitempty"`               // 作者
	Rating              float64     `json:"rating,omitempty"`               // 评分
	InstallCount        int         `json:"install_count,omitempty"`        // 安装次数
	IsOfficial          bool        `json:"is_official,omitempty"`          // 是否是官方技能
}

// SkillMetrics 技能运行指标
type SkillMetrics struct {
	TotalCalls   int64   `json:"total_calls"`    // 总调用次数
	SuccessRate  float64 `json:"success_rate"`   // 成功率
	AvgLatencyMs int64   `json:"avg_latency_ms"` // 平均延迟(ms)
	ErrorCount   int64   `json:"error_count"`    // 错误次数
	LastUsedAt   string  `json:"last_used_at"`   // 最后使用时间
}

// ExtendedSkill 扩展的Skill接口，包含市场和训练相关能力
// 所有新开发的技能都应该实现这个接口
type ExtendedSkill interface {
	Skill
	// Manifest 返回技能的自描述信息
	Manifest() SkillManifest
	// Train 使用提供的数据集训练/微调技能
	Train(ctx context.Context, dataset interface{}) error
	// GetMetrics 获取技能的运行指标
	GetMetrics() SkillMetrics
}
