package cloudsignal

import (
	"encoding/json"
	"time"
)

// SignalKind 区分两种独立同步的信号。它与表里的 signal_kind 取值一一对应，
// 不要用字符串字面量在调用方散落拼接。
type SignalKind string

const (
	SignalKindAlarm         SignalKind = "alarm"
	SignalKindVulnerability SignalKind = "vulnerability"
)

func (k SignalKind) Valid() bool {
	return k == SignalKindAlarm || k == SignalKindVulnerability
}

// 同步状态口径与其他能力状态一致：未配置、不可用、就绪。
// 空结果不等于未运行——没有结果但同步成功仍然记 ready。
const (
	SyncStateUnconfigured = "unconfigured"
	SyncStateUnavailable  = "unavailable"
	SyncStateReady        = "ready"
)

// 漏洞严重度。SAS 只给 CVSS 分数，分档口径固定在这里，报表才能跨账号比较。
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
	SeverityUnknown  = "unknown"
)

// 漏洞状态。open/fixing 是云上未修复与修复中；fixed/ignored 是本地终态。
const (
	FindingStatusOpen    = "open"
	FindingStatusFixing  = "fixing"
	FindingStatusFixed   = "fixed"
	FindingStatusIgnored = "ignored"
)

func IsActiveFindingStatus(status string) bool {
	return status == FindingStatusOpen || status == FindingStatusFixing
}

// 告警状态：轮询只写 firing，acknowledged/resolved 由本地操作员推进。
const (
	AlertStatusFiring       = "firing"
	AlertStatusAcknowledged = "acknowledged"
	AlertStatusResolved     = "resolved"
)

// SyncState 是 cloud_signal_sync_states 的一行，附带账号展示字段。
type SyncState struct {
	ID                  int64
	TenantID            int
	CloudAccountID      int64
	AccountName         string
	Provider            string
	AccountActive       bool
	SignalKind          SignalKind
	State               string
	LastAttemptAt       *time.Time
	LastSuccessAt       *time.Time
	Watermark           *time.Time
	LastErrorCode       string
	LastErrorMessage    string
	ItemsSynced         int64
	ConsecutiveFailures int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// SecurityFinding 是 security_findings 的一行，附带已关联 CI 的展示字段。
type SecurityFinding struct {
	ID             int64
	TenantID       int
	CloudAccountID int64
	Provider       string
	VulnType       string
	FindingKey     string
	Name           string
	AliasName      string
	Severity       string
	Status         string
	Necessity      string
	InstanceID     string
	InstanceName   string
	UUID           string
	InternetIP     string
	IntranetIP     string
	Region         string
	CIID           *int64
	CINumber       string
	CIName         string
	CVSSScore      *float64
	RawPayload     json.RawMessage
	FirstSeenAt    *time.Time
	LastSeenAt     *time.Time
	LastSyncedAt   time.Time
	ResolvedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AlertRecord 是 alerts 表在云监控口径下的一行。
// 建单桥的状态放在 Annotations 里（bridgeStatus / incidentId），不另建表。
type AlertRecord struct {
	ID             int64
	TenantID       int
	AlertID        string
	Name           string
	Description    string
	Severity       string
	Status         string
	Labels         map[string]string
	Annotations    map[string]string
	SourceIP       string
	Service        string
	FiredAt        time.Time
	AcknowledgedAt *time.Time
	ResolvedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// BridgeCandidate 是一个已被本次 worker 认领、可以建单的告警。
// Annotations 返回的是认领后的值，调用方据此判断重试次数。
type BridgeCandidate struct {
	AlertRecord
}

// FindingFilter 是列表查询条件。空字段表示不过滤；分页由仓储兜底收敛。
type FindingFilter struct {
	Statuses       []string
	Severities     []string
	VulnType       string
	InstanceID     string
	CloudAccountID int64
	Search         string
	Page           int
	PageSize       int
}

// AlertFilter 是云监控告警的列表条件。
type AlertFilter struct {
	Statuses       []string
	Severities     []string
	CloudAccountID int64
	Search         string
	Page           int
	PageSize       int
}

// FindingSummary 只统计未修复的漏洞（open/fixing），
// 已修复的存量会无限增长，混进来会让看板数字失去意义。
type FindingSummary struct {
	Total      int            `json:"total"`
	BySeverity map[string]int `json:"bySeverity"`
	ByStatus   map[string]int `json:"byStatus"`
}

const (
	defaultPageSize = 20
	maxPageSize     = 200
)
