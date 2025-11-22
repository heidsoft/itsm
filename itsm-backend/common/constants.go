package common

import "time"

// ===================================
// Role Constants
// ===================================
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleManager    = "manager"
	RoleAgent      = "agent"
	RoleTechnician = "technician"
	RoleEndUser    = "end_user"
)

// ===================================
// Resource Constants
// ===================================
const (
	ResourceAll       = "*"
	ResourceTicket    = "ticket"
	ResourceUser      = "user"
	ResourceDashboard = "dashboard"
	ResourceKnowledge = "knowledge"
	ResourceCMDB      = "cmdb"
	ResourceIncident  = "incident"
	ResourceProblem   = "problem"
	ResourceChange    = "change"
	ResourceSLA       = "sla"
	ResourceAuditLog  = "audit_logs"
)

// ===================================
// Action Constants
// ===================================
const (
	ActionAll    = "*"
	ActionRead   = "read"
	ActionWrite  = "write"
	ActionDelete = "delete"
	ActionAdmin  = "admin"
)

// ===================================
// Ticket Status Constants
// ===================================
const (
	TicketStatusNew        = "new"
	TicketStatusOpen       = "open"
	TicketStatusInProgress = "in_progress"
	TicketStatusPending    = "pending"
	TicketStatusResolved   = "resolved"
	TicketStatusClosed     = "closed"
	TicketStatusCancelled  = "cancelled"
)

// ===================================
// Ticket Priority Constants
// ===================================
const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

// ===================================
// Incident Status Constants
// ===================================
const (
	IncidentStatusNew        = "new"
	IncidentStatusInProgress = "in_progress"
	IncidentStatusResolved   = "resolved"
	IncidentStatusClosed     = "closed"
)

// ===================================
// Incident Severity Constants
// ===================================
const (
	SeverityP1 = "P1" // Critical
	SeverityP2 = "P2" // High
	SeverityP3 = "P3" // Medium
	SeverityP4 = "P4" // Low
)

// ===================================
// SLA Status Constants
// ===================================
const (
	SLAStatusActive    = "active"
	SLAStatusBreached  = "breached"
	SLAStatusCompleted = "completed"
	SLAStatusPaused    = "paused"
)

// ===================================
// Change Status Constants
// ===================================
const (
	ChangeStatusDraft      = "draft"
	ChangeStatusSubmitted  = "submitted"
	ChangeStatusApproved   = "approved"
	ChangeStatusRejected   = "rejected"
	ChangeStatusScheduled  = "scheduled"
	ChangeStatusInProgress = "in_progress"
	ChangeStatusCompleted  = "completed"
	ChangeStatusFailed     = "failed"
	ChangeStatusCancelled  = "cancelled"
)

// ===================================
// Change Risk Level Constants
// ===================================
const (
	RiskLevelLow      = "low"
	RiskLevelMedium   = "medium"
	RiskLevelHigh     = "high"
	RiskLevelCritical = "critical"
)

// ===================================
// Notification Type Constants
// ===================================
const (
	NotificationTypeEmail  = "email"
	NotificationTypeSMS    = "sms"
	NotificationTypeWebhook = "webhook"
	NotificationTypeInApp  = "in_app"
)

// ===================================
// CMDB CI Status Constants
// ===================================
const (
	CIStatusActive     = "active"
	CIStatusInactive   = "inactive"
	CIStatusRetired    = "retired"
	CIStatusMaintenance = "maintenance"
)

// ===================================
// HTTP Status Codes (Custom)
// ===================================
const (
	HTTPStatusSuccess       = 200
	HTTPStatusCreated       = 201
	HTTPStatusNoContent     = 204
	HTTPStatusBadRequest    = 400
	HTTPStatusUnauthorized  = 401
	HTTPStatusForbidden     = 403
	HTTPStatusNotFound      = 404
	HTTPStatusConflict      = 409
	HTTPStatusTooManyReqs   = 429
	HTTPStatusInternalError = 500
)

// ===================================
// Rate Limiting Constants
// ===================================
const (
	DefaultRateLimit       = 100
	DefaultRateLimitWindow = time.Minute
	AdminRateLimit         = 1000
	PublicRateLimit        = 50
)

// ===================================
// Request Size Limits
// ===================================
const (
	MaxRequestSize       = 10 * 1024 * 1024 // 10MB
	MaxUploadSize        = 50 * 1024 * 1024 // 50MB
	MaxJSONPayloadSize   = 5 * 1024 * 1024  // 5MB
)

// ===================================
// Pagination Defaults
// ===================================
const (
	DefaultPageSize = 10
	MaxPageSize     = 100
	MinPageSize     = 1
	DefaultPage     = 1
)

// ===================================
// Cache TTL Constants
// ===================================
const (
	CacheTTLShort  = 5 * time.Minute
	CacheTTLMedium = 30 * time.Minute
	CacheTTLLong   = 2 * time.Hour
	CacheTTLDay    = 24 * time.Hour
)

// ===================================
// Timeout Constants
// ===================================
const (
	DefaultContextTimeout = 30 * time.Second
	LongRunningTimeout    = 5 * time.Minute
	DatabaseTimeout       = 10 * time.Second
	HTTPClientTimeout     = 30 * time.Second
)

// ===================================
// Audit Log Actions
// ===================================
const (
	AuditActionCreate = "create"
	AuditActionRead   = "read"
	AuditActionUpdate = "update"
	AuditActionDelete = "delete"
	AuditActionLogin  = "login"
	AuditActionLogout = "logout"
	AuditActionExport = "export"
	AuditActionImport = "import"
)

// ===================================
// Validation Constants
// ===================================
const (
	MinUsernameLength = 3
	MaxUsernameLength = 50
	MinPasswordLength = 6
	MaxPasswordLength = 128
	MaxNameLength     = 100
	MaxEmailLength    = 255
	MinTitleLength    = 1
	MaxTitleLength    = 200
	MaxDescriptionLength = 10000
)

// ===================================
// JWT Constants
// ===================================
const (
	JWTTokenTypeAccess  = "access"
	JWTTokenTypeRefresh = "refresh"
	MinJWTSecretLength  = 32
)

// ===================================
// Feature Flags
// ===================================
const (
	FeatureAI              = "ai_enabled"
	FeatureAdvancedWorkflow = "advanced_workflow"
	FeatureVectorSearch    = "vector_search"
	FeatureNotifications   = "notifications"
)

// ===================================
// Content Types
// ===================================
const (
	ContentTypeJSON = "application/json"
	ContentTypeXML  = "application/xml"
	ContentTypeForm = "application/x-www-form-urlencoded"
	ContentTypeText = "text/plain"
)

// ===================================
// Header Keys
// ===================================
const (
	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"
	HeaderAccept        = "Accept"
	HeaderUserAgent     = "User-Agent"
	HeaderRequestID     = "X-Request-ID"
	HeaderTenantID      = "X-Tenant-ID"
	HeaderTenantCode    = "X-Tenant-Code"
	HeaderAPIVersion    = "X-API-Version"
)

// ===================================
// File Extensions
// ===================================
const (
	ExtJSON = ".json"
	ExtYAML = ".yaml"
	ExtYML  = ".yml"
	ExtXML  = ".xml"
	ExtCSV  = ".csv"
	ExtPDF  = ".pdf"
)

// ===================================
// Regex Patterns
// ===================================
const (
	EmailRegex    = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	UsernameRegex = `^[a-zA-Z0-9_-]{3,50}$`
	PhoneRegex    = `^[0-9+\-\s()]{7,20}$`
)

// ===================================
// Message Templates
// ===================================
const (
	MsgSuccess          = "操作成功"
	MsgCreated          = "创建成功"
	MsgUpdated          = "更新成功"
	MsgDeleted          = "删除成功"
	MsgNotFound         = "资源不存在"
	MsgUnauthorized     = "未授权"
	MsgForbidden        = "权限不足"
	MsgBadRequest       = "请求参数错误"
	MsgInternalError    = "服务器内部错误"
	MsgValidationFailed = "数据验证失败"
)

// ===================================
// System Constants
// ===================================
const (
	SystemUser     = "system"
	DefaultTenant  = "default"
	SuperAdminUser = "admin"
)

