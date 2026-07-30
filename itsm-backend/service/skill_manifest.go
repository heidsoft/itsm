package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

// SkillManifest 技能自描述信息，用于市场展示和自动装配
type SkillManifest struct {
	Name                string      `json:"name"`                          // 唯一标识名称
	Version             string      `json:"version"`                       // 版本号
	Title               string      `json:"title"`                         // 显示名称
	Provider            string      `json:"provider"`                      // 提供商
	Description         string      `json:"description"`                   // 描述
	LongDescription     string      `json:"longDescription,omitempty"`     // 详细描述
	IconURL             string      `json:"iconUrl,omitempty"`             // 图标URL
	Screenshots         []string    `json:"screenshots,omitempty"`         // 截图列表
	Tags                []string    `json:"tags,omitempty"`                // 标签列表
	Category            string      `json:"category,omitempty"`            // 分类
	Capabilities        []string    `json:"capabilities,omitempty"`        // 支持的能力列表
	InputSchema         interface{} `json:"inputSchema,omitempty"`         // 输入参数JSON Schema
	OutputSchema        interface{} `json:"outputSchema,omitempty"`        // 输出结果JSON Schema
	RequiredPermissions []string    `json:"requiredPermissions,omitempty"` // 需要的系统权限
	MinSystemVersion    string      `json:"minSystemVersion,omitempty"`    // 最低支持的系统版本
	Author              string      `json:"author,omitempty"`              // 作者
	Rating              float64     `json:"rating,omitempty"`              // 评分
	InstallCount        int         `json:"installCount,omitempty"`        // 安装次数
	IsOfficial          bool        `json:"isOfficial,omitempty"`          // 是否是官方技能
	Checksum            string      `json:"checksum,omitempty"`            // manifest 完整性校验和（注册时自动计算，sha256:...）
}

// ComputeChecksum 基于身份与安全关键字段计算确定性校验和，
// 与 connector.Manifest.ComputeChecksum 保持同样的覆盖范围约定。
func (m SkillManifest) ComputeChecksum() string {
	caps := append([]string(nil), m.Capabilities...)
	sort.Strings(caps)
	perms := append([]string(nil), m.RequiredPermissions...)
	sort.Strings(perms)
	payload := strings.Join([]string{
		m.Name,
		m.Version,
		m.Provider,
		strings.Join(caps, ","),
		strings.Join(perms, ","),
		m.MinSystemVersion,
	}, "|")
	sum := sha256.Sum256([]byte(payload))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ValidateForRegistration 注册前的 manifest 完整性校验（fail closed）：
// 所有技能必须声明 name、version 和所需系统权限，否则拒绝注册。
func (m SkillManifest) ValidateForRegistration() error {
	if m.Name == "" {
		return errors.New("skill manifest: name must not be empty")
	}
	if m.Version == "" {
		return errors.New("skill manifest: version must not be empty for " + m.Name)
	}
	if len(m.RequiredPermissions) == 0 {
		return errors.New("skill manifest: requiredPermissions must be declared for " + m.Name)
	}
	return nil
}

// SkillMetrics 技能运行指标
type SkillMetrics struct {
	TotalCalls   int64   `json:"totalCalls"`   // 总调用次数
	SuccessRate  float64 `json:"successRate"`  // 成功率
	AvgLatencyMs int64   `json:"avgLatencyMs"` // 平均延迟(ms)
	ErrorCount   int64   `json:"errorCount"`   // 错误次数
	LastUsedAt   string  `json:"lastUsedAt"`   // 最后使用时间
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
