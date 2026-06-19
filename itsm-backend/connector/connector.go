// Package connector 提供可插拔的连接器/集成框架
// 设计目标：
//  1. 作为"插件市场、技能市场、连接器市场"的运行时底座
//  2. 让飞书/钉钉/企微/Webhook/数据库/邮件 等外部系统的对接变成"即插即用"
//  3. 对接 LLM/AI：通过统一的 Outbound 接口把告警/工单/审批事件 投递到任意通道
package connector

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ConnectorType 标识一个连接器的能力分类
type ConnectorType string

const (
	TypeIM        ConnectorType = "im"         // 即时通讯：飞书/钉钉/企微/Slack
	TypeWebhook   ConnectorType = "webhook"    // 通用 HTTP 出站
	TypeEmail     ConnectorType = "email"      // 邮件 SMTP
	TypeSMS       ConnectorType = "sms"        // 短信
	TypeDatabase  ConnectorType = "database"   // 数据库
	TypeAPI       ConnectorType = "api"        // 第三方 REST/SOAP
	TypeMonitor   ConnectorType = "monitoring" // 监控告警源：Zabbix/Prometheus
	TypeStorage   ConnectorType = "storage"    // 对象存储：OSS/S3/MinIO
	TypeIdentity  ConnectorType = "identity"   // 身份源：OIDC/LDAP/SAML
	TypeLLMTool   ConnectorType = "llm_tool"   // LLM Function Calling 工具
	TypeCustom    ConnectorType = "custom"
)

// Capability 描述连接器可执行的"原子能力"
type Capability string

const (
	CapSendMessage    Capability = "send_message"     // 发送消息
	CapReceiveMessage Capability = "receive_message"  // 接收消息
	CapSendCard       Capability = "send_card"        // 发送卡片
	CapReplyMessage   Capability = "reply_message"    // 回复消息
	CapCreateTicket   Capability = "create_ticket"    // 创建工单
	CapUpdateTicket   Capability = "update_ticket"    // 更新工单
	CapQueryUser      Capability = "query_user"       // 查询用户
	CapQueryCI        Capability = "query_ci"         // 查询配置项
	CapSyncAssets     Capability = "sync_assets"      // 同步资产
	CapHealthCheck    Capability = "health_check"     // 健康检查
	CapTriggerProcess   Capability = "trigger_process"    // 触发流程
	CapApproveProcess   Capability = "approve_process"    // 审批流程
	CapSyncOrganization Capability = "sync_organization"  // 同步组织架构
	CapAutoDiscoverCI   Capability = "auto_discover_ci"   // 自动发现CI配置项

)

// Message 统一的出站消息结构（IM/邮件/短信复用）
type Message struct {
	ID          string                 `json:"id,omitempty"`
	Channel     string                 `json:"channel"`           // 目标通道：user_open_id / email / phone / chat_id
	Type        string                 `json:"type"`              // text / markdown / card / image / file
	Title       string                 `json:"title,omitempty"`
	Content     string                 `json:"content"`           // 纯文本或 markdown
	Card        *Card                  `json:"card,omitempty"`    // 富文本卡片（IM 场景）
	Mentions    []Mention              `json:"mentions,omitempty"`
	Actions     []Action               `json:"actions,omitempty"` // 卡片按钮
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ReplyTo     string                 `json:"reply_to,omitempty"`
	ThreadID    string                 `json:"thread_id,omitempty"`
}

// Card 富文本卡片（飞书/Lark/钉钉通用结构）
type Card struct {
	TemplateID string                 `json:"template_id,omitempty"`
	Header     *CardHeader            `json:"header,omitempty"`
	Sections   []CardSection          `json:"sections,omitempty"`
	Elements   []CardElement          `json:"elements,omitempty"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
}

type CardHeader struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Color    string `json:"color,omitempty"` // blue/red/orange/green
}

type CardSection struct {
	Title   string        `json:"title,omitempty"`
	Content []CardElement `json:"content,omitempty"`
}

type CardElement struct {
	Type    string                 `json:"type"`              // text / markdown / divider / image / button / select / input
	Text    string                 `json:"text,omitempty"`
	Fields  []KV                   `json:"fields,omitempty"`
	Action  *Action                `json:"action,omitempty"`
	ImageURL string                `json:"image_url,omitempty"`
	Options []CardElement          `json:"options,omitempty"`
	Extras  map[string]interface{} `json:"extras,omitempty"`
}

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

type Mention struct {
	Type string `json:"type"`  // user / all
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type Action struct {
	Type  string `json:"type"`           // link / button / primary / danger
	Text  string `json:"text"`           // 按钮文案
	URL   string `json:"url,omitempty"`  // 跳转链接
	Value string `json:"value,omitempty"` // 回传数据
}

// HealthStatus 健康检查结果
type HealthStatus struct {
	OK        bool                   `json:"ok"`
	LatencyMs int64                  `json:"latency_ms"`
	Message   string                 `json:"message,omitempty"`
	CheckedAt time.Time              `json:"checked_at"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// Config 连接器实例配置（来自 DB / 环境变量）
type Config struct {
	// 基础
	TenantID  int                    `json:"tenant_id"`
	Name      string                 `json:"name"`
	Type      ConnectorType          `json:"type"`
	Provider  string                 `json:"provider"`  // feishu / dingtalk / wecom / slack / generic
	Enabled   bool                   `json:"enabled"`

	// 凭据与端点（敏感字段建议在 DB 中加密存储）
	Credentials map[string]string     `json:"credentials,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`

	// 元数据
	Labels    map[string]string      `json:"labels,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Manifest 连接器"自描述"，用于插件市场展示与自动装配
type Manifest struct {
	Name         string       `json:"name"`         // 唯一 key
	Version      string       `json:"version"`
	Title        string       `json:"title"`        // 中文显示名
	Provider     string       `json:"provider"`
	Type         ConnectorType `json:"type"`
	Description  string       `json:"description"`
	Author       string       `json:"author,omitempty"`
	Homepage     string       `json:"homepage,omitempty"`
	IconURL      string       `json:"icon_url,omitempty"`
	Capabilities []Capability `json:"capabilities"`
	ConfigSchema string       `json:"config_schema,omitempty"` // JSON Schema
	Tags         []string     `json:"tags,omitempty"`
	MinITSMVer   string       `json:"min_itsm_ver,omitempty"`
	Screenshots  []string     `json:"screenshots,omitempty"`  // 截图URL列表
	Changelog    string       `json:"changelog,omitempty"`    // 版本更新日志
	InstallCount int          `json:"install_count,omitempty"`// 安装次数
	Rating       float64      `json:"rating,omitempty"`       // 评分，0-5
	IsOfficial   bool         `json:"is_official,omitempty"`  // 是否是官方组件
	Category     string       `json:"category,omitempty"`     // 分类
	RequiredPermissions []string `json:"required_permissions,omitempty"` // 需要的系统权限列表

}

// InboundMessage 入站消息（来自 IM 回调 / Webhook）
type InboundMessage struct {
	ConnectorName string                 `json:"connector_name"`
	ConnectorType ConnectorType          `json:"connector_type"`
	Channel       string                 `json:"channel"`
	UserID        string                 `json:"user_id,omitempty"`
	UserName      string                 `json:"user_name,omitempty"`
	ChatID        string                 `json:"chat_id,omitempty"`
	ChatType      string                 `json:"chat_type,omitempty"` // direct / group
	MessageID     string                 `json:"message_id,omitempty"`
	Content       string                 `json:"content"`
	Type          string                 `json:"type"`
	Mentions      []Mention              `json:"mentions,omitempty"`
	Raw           json.RawMessage        `json:"raw,omitempty"`
	ReceivedAt    time.Time              `json:"received_at"`
	Extras        map[string]interface{} `json:"extras,omitempty"`
}

// Connector 任何"可插拔"的连接器都必须实现此接口
// 设计原则：Send/Receive 分离，HealthCheck 必选，Manifest 用于市场
type Connector interface {
	// Manifest 返回自描述
	Manifest() Manifest

	// Init 注入配置与依赖（一次性）
	Init(ctx context.Context, cfg Config) error

	// Send 发送出站消息
	Send(ctx context.Context, msg *Message) error

	// HealthCheck 健康检查（用于监控与告警）
	HealthCheck(ctx context.Context) HealthStatus

	// Close 释放资源
	Close() error
}

// Receiver 可选：支持入站消息的连接器（IM 回调）
type Receiver interface {
	Connector
	// VerifySignature 校验入站签名
	VerifySignature(headers map[string]string, body []byte) error
	// ParseInbound 解析入站 payload 为统一结构
	ParseInbound(body []byte) (*InboundMessage, error)
}

// ErrNotSupported 连接器不支持的能力
var ErrNotSupported = errors.New("connector: capability not supported")
