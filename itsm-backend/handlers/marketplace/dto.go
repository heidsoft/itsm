package marketplace

import (
	"strings"
	"time"

	"itsm-backend/ent"
)

// InstallationResponse 是安装记录的出站 DTO：camelCase 契约，
// Config 中敏感值已脱敏——连接器密钥绝不能到达前端。
type InstallationResponse struct {
	ID               int                    `json:"id"`
	TenantID         int                    `json:"tenantId"`
	ItemID           int                    `json:"itemId"`
	InstalledVersion string                 `json:"installedVersion,omitempty"`
	Status           string                 `json:"status"`
	Config           map[string]interface{} `json:"config,omitempty"`
	AutoUpgrade      bool                   `json:"autoUpgrade"`
	InstalledBy      string                 `json:"installedBy,omitempty"`
	InstalledAt      time.Time              `json:"installedAt,omitempty"`
	LastUpdatedAt    time.Time              `json:"lastUpdatedAt,omitempty"`
	LastUsedAt       time.Time              `json:"lastUsedAt,omitempty"`
	ErrorMessage     string                 `json:"errorMessage,omitempty"`
}

func ToInstallationResponse(in *ent.TenantInstallation) *InstallationResponse {
	if in == nil {
		return nil
	}
	return &InstallationResponse{
		ID:               in.ID,
		TenantID:         in.TenantID,
		ItemID:           in.ItemID,
		InstalledVersion: in.InstalledVersion,
		Status:           string(in.Status),
		Config:           MaskSensitiveConfig(in.Config),
		AutoUpgrade:      in.AutoUpgrade,
		InstalledBy:      in.InstalledBy,
		InstalledAt:      in.InstalledAt,
		LastUpdatedAt:    in.LastUpdatedAt,
		LastUsedAt:       in.LastUsedAt,
		ErrorMessage:     in.ErrorMessage,
	}
}

func ToInstallationResponseList(in []*ent.TenantInstallation) []*InstallationResponse {
	out := make([]*InstallationResponse, 0, len(in))
	for _, item := range in {
		out = append(out, ToInstallationResponse(item))
	}
	return out
}

// MaskSensitiveConfig 递归脱敏配置 map：键名命中敏感片段的叶子值替换为掩码，
// 保留键结构与非敏感值，前端仍可见配置了哪些字段。
// 配置更新入口的掩码还原见 service/marketplace.RestoreMaskedConfig。
func MaskSensitiveConfig(cfg map[string]interface{}) map[string]interface{} {
	if cfg == nil {
		return nil
	}
	out := make(map[string]interface{}, len(cfg))
	for k, v := range cfg {
		if nested, ok := v.(map[string]interface{}); ok {
			out[k] = MaskSensitiveConfig(nested)
			continue
		}
		if isSensitiveConfigKey(k) {
			out[k] = maskPlaceholder
			continue
		}
		out[k] = v
	}
	return out
}

const maskPlaceholder = "***"

func isSensitiveConfigKey(key string) bool {
	lk := strings.ToLower(key)
	for _, fragment := range []string{"token", "secret", "password", "passwd", "apikey", "api_key", "credential", "encrypt"} {
		if strings.Contains(lk, fragment) {
			return true
		}
	}
	return false
}
