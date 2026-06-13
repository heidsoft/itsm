package dto

import "time"

// ConnectorManifestDTO 连接器清单（用于插件/技能/连接器市场）
type ConnectorManifestDTO struct {
	Name          string     `json:"name"`
	Version       string     `json:"version"`
	Title         string     `json:"title"`
	Provider      string     `json:"provider"`
	Type          string     `json:"type"`
	Description   string     `json:"description"`
	Author        string     `json:"author,omitempty"`
	Homepage      string     `json:"homepage,omitempty"`
	IconURL       string     `json:"icon_url,omitempty"`
	Capabilities  []string   `json:"capabilities"`
	Tags          []string   `json:"tags,omitempty"`
	MinITSMVer    string     `json:"min_itsm_ver,omitempty"`
	Local         bool       `json:"local"`     // 是否本地内置
	Installed     bool       `json:"installed"` // 当前租户是否已安装
	Enabled       bool       `json:"enabled"`
	Healthy       bool       `json:"healthy"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	Lifecycle     string     `json:"lifecycle"` // available / installed / enabled / healthy / unhealthy
	Category      string     `json:"category"`
}

// ProvisionConnectorRequest 配置/启用一个连接器实例
type ProvisionConnectorRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Provider    string                 `json:"provider"`
	Enabled     bool                   `json:"enabled"`
	Credentials map[string]string      `json:"credentials,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty"`
}

// SendConnectorMessageRequest 通过指定连接器发消息
type SendConnectorMessageRequest struct {
	Name     string                 `json:"name" binding:"required"` // 连接器名
	Channel  string                 `json:"channel" binding:"required"`
	Type     string                 `json:"type"` // text / markdown / post / card
	Title    string                 `json:"title,omitempty"`
	Content  string                 `json:"content"`
	Card     *CardPayloadDTO        `json:"card,omitempty"`
	Mentions []MentionDTO           `json:"mentions,omitempty"`
	Actions  []ActionDTO            `json:"actions,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type CardPayloadDTO struct {
	Header    *CardHeaderDTO         `json:"header,omitempty"`
	Sections  []CardSectionDTO       `json:"sections,omitempty"`
	Elements  []CardElementDTO       `json:"elements,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type CardHeaderDTO struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Color    string `json:"color,omitempty"`
}

type CardSectionDTO struct {
	Title   string           `json:"title,omitempty"`
	Content []CardElementDTO `json:"content,omitempty"`
}

type CardElementDTO struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	Fields   []KVDTO                `json:"fields,omitempty"`
	Action   *ActionDTO             `json:"action,omitempty"`
	ImageURL string                 `json:"image_url,omitempty"`
	Extras   map[string]interface{} `json:"extras,omitempty"`
}

type KVDTO struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

type MentionDTO struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type ActionDTO struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	URL   string `json:"url,omitempty"`
	Value string `json:"value,omitempty"`
}

// ConnectorHealthDTO 健康检查结果
type ConnectorHealthDTO struct {
	OK        bool                   `json:"ok"`
	LatencyMs int64                  `json:"latency_ms"`
	Message   string                 `json:"message,omitempty"`
	CheckedAt time.Time              `json:"checked_at"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// ConnectorConfigDTO 当前生效的配置（凭据脱敏）
type ConnectorConfigDTO struct {
	Name          string                 `json:"name"`
	Provider      string                 `json:"provider"`
	Type          string                 `json:"type"`
	Enabled       bool                   `json:"enabled"`
	Healthy       bool                   `json:"healthy"`
	Lifecycle     string                 `json:"lifecycle"`
	LastCheckedAt *time.Time             `json:"last_checked_at,omitempty"`
	LastError     string                 `json:"last_error,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Credentials   map[string]string      `json:"credentials,omitempty"` // 脱敏：仅返回键名
	Settings      map[string]interface{} `json:"settings,omitempty"`
	Labels        map[string]string      `json:"labels,omitempty"`
}

// ConnectorLifecycleDTO is the GA-facing lifecycle view used by the connector market.
type ConnectorLifecycleDTO struct {
	Name          string     `json:"name"`
	Provider      string     `json:"provider"`
	Type          string     `json:"type"`
	Installed     bool       `json:"installed"`
	Enabled       bool       `json:"enabled"`
	Healthy       bool       `json:"healthy"`
	Lifecycle     string     `json:"lifecycle"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	Capabilities  []string   `json:"capabilities,omitempty"`
}
