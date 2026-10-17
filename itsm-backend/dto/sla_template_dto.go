package dto

import "time"

// SLATemplateResponse 是 SLA 模板目录在 API 边界的契约。
//
// businessHours / escalationLevels 与已安装 SLA 定义共用同一套投影函数，
// 模板预览的形状和安装后定义的形状因此是一致的。
type SLATemplateResponse struct {
	Key              string                 `json:"key"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	ServiceType      string                 `json:"serviceType"`
	Priority         string                 `json:"priority"`
	ResponseTime     int                    `json:"responseTime"`
	ResolutionTime   int                    `json:"resolutionTime"`
	BusinessHours    *SLABusinessHours      `json:"businessHours,omitempty"`
	EscalationLevels []SLAEscalationLevel   `json:"escalationLevels,omitempty"`
	Conditions       map[string]interface{} `json:"conditions"`
	Industry         string                 `json:"industry"`
	Recommended      bool                   `json:"recommended"`

	// Installed / SLADefinitionID / InstalledAt 描述当前租户的安装事实。
	Installed       bool       `json:"installed"`
	SLADefinitionID int        `json:"slaDefinitionId,omitempty"`
	InstalledAt     *time.Time `json:"installedAt,omitempty"`
}
