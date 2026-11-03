package marketplace

import (
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service/marketplace"
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

	// 以下字段来自安装记录的商品边：安装列表的展示与过滤靠它们，
	// 前端不应再拉一次目录来「补齐名称/描述」——那是第二次真相来源。
	// 边未加载时留空（不是「没有新版本」）：界面必须把「不知道」和
	// 「已最新」区分开，否则升级按钮会在数据缺失时消失。
	ItemName         string `json:"itemName,omitempty"`
	ItemTitle        string `json:"itemTitle,omitempty"`
	ItemDescription  string `json:"itemDescription,omitempty"`
	ItemIconURL      string `json:"itemIconUrl,omitempty"`
	ItemType         string `json:"itemType,omitempty"`
	Deliverable      bool   `json:"deliverable"`
	LatestVersion    string `json:"latestVersion,omitempty"`
	UpgradeAvailable bool   `json:"upgradeAvailable"`
}

func ToInstallationResponse(in *ent.TenantInstallation) *InstallationResponse {
	if in == nil {
		return nil
	}
	resp := &InstallationResponse{
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
	if item := in.Edges.Item; item != nil {
		resp.ItemName = item.Name
		resp.ItemTitle = item.Title
		resp.ItemDescription = item.Description
		resp.ItemIconURL = item.IconURL
		resp.ItemType = string(item.Type)
		resp.Deliverable = marketplace.IsDeliverableType(item.Type)
		resp.LatestVersion = item.LatestVersion
		// 可升级 = 类型可交付 + 目录里的 latest 与实际装的不是一个版本。
		// 版本号大小不在此比较：回滚也是「升到另一个已发布版本」，
		// 由服务端 UpgradeInstallation 判定目标是否可用。
		resp.UpgradeAvailable = resp.Deliverable &&
			strings.TrimSpace(item.LatestVersion) != "" &&
			strings.TrimSpace(item.LatestVersion) != strings.TrimSpace(in.InstalledVersion)
	}
	return resp
}

// ToInstallationResponseList 批量转换出站安装记录。
func ToInstallationResponseList(in []*ent.TenantInstallation) []*InstallationResponse {
	out := make([]*InstallationResponse, 0, len(in))
	for _, item := range in {
		out = append(out, ToInstallationResponse(item))
	}
	return out
}

// ToDependencyReportResponse 把依赖对齐的领域结果转成出站 DTO。
func ToDependencyReportResponse(report *marketplace.DependencyReport) *dto.MarketplaceDependencyReportResponse {
	if report == nil {
		return nil
	}
	out := &dto.MarketplaceDependencyReportResponse{
		TenantID:           report.TenantID,
		CheckedAt:          report.CheckedAt,
		Total:              report.Total,
		Satisfied:          report.Satisfied,
		Unsatisfied:        report.Unsatisfied,
		DeclarationMissing: report.DeclarationMissing,
		Findings:           make([]*dto.MarketplaceDependencyResponse, 0, len(report.Findings)),
	}
	for _, finding := range report.Findings {
		out.Findings = append(out.Findings, &dto.MarketplaceDependencyResponse{
			ItemID:             finding.ItemID,
			ItemName:           finding.ItemName,
			ItemTitle:          finding.ItemTitle,
			ItemType:           finding.ItemType,
			Status:             finding.Status,
			InstalledVersion:   finding.InstalledVersion,
			InstallationStatus: finding.InstallStatus,
			Dependencies:       finding.Dependencies,
			Problems:           finding.Problems,
		})
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
