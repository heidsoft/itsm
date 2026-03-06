package dto

import (
	"time"
)

// SLAPolicyDTO SLA策略DTO
type CreateSLAPolicyRequest struct {
	Name                  string                 `json:"name" binding:"required"`
	Description           string                 `json:"description"`
	CustomerTier          *string                `json:"customer_tier"` // platinum/gold/silver/bronze
	TicketType            *string                `json:"ticket_type"`   // incident/problem/change/request
	Priority              *string                `json:"priority"`      // critical/high/medium/low
	ResponseTimeMinutes   int                   `json:"response_time_minutes"`
	ResolutionTimeMinutes int                   `json:"resolution_time_minutes"`
	BusinessHours         map[string]interface{} `json:"business_hours"`
	ExcludeWeekends       bool                  `json:"exclude_weekends"`
	ExcludeHolidays       bool                  `json:"exclude_holidays"`
	EscalationRules       map[string]interface{} `json:"escalation_rules"`
	IsActive              bool                  `json:"is_active"`
	PriorityScore         int                   `json:"priority_score"`
	TenantID              int                   `json:"tenant_id"`
}

type UpdateSLAPolicyRequest struct {
	Name                  *string                `json:"name,omitempty"`
	Description           *string                `json:"description,omitempty"`
	CustomerTier          *string                `json:"customer_tier,omitempty"`
	TicketType            *string                `json:"ticket_type,omitempty"`
	Priority              *string                `json:"priority,omitempty"`
	ResponseTimeMinutes   *int                   `json:"response_time_minutes,omitempty"`
	ResolutionTimeMinutes *int                   `json:"resolution_time_minutes,omitempty"`
	BusinessHours         *map[string]interface{} `json:"business_hours,omitempty"`
	ExcludeWeekends       *bool                  `json:"exclude_weekends,omitempty"`
	ExcludeHolidays       *bool                  `json:"exclude_holidays,omitempty"`
	EscalationRules       *map[string]interface{} `json:"escalation_rules,omitempty"`
	IsActive              *bool                  `json:"is_active,omitempty"`
	PriorityScore         *int                   `json:"priority_score,omitempty"`
}

type SLAPolicyResponse struct {
	ID                    int                    `json:"id"`
	Name                  string                 `json:"name"`
	Description           string                 `json:"description"`
	CustomerTier          string                 `json:"customer_tier"`
	TicketType            string                 `json:"ticket_type"`
	Priority              string                 `json:"priority"`
	ResponseTimeMinutes   int                   `json:"response_time_minutes"`
	ResolutionTimeMinutes int                   `json:"resolution_time_minutes"`
	BusinessHours         map[string]interface{} `json:"business_hours"`
	ExcludeWeekends       bool                  `json:"exclude_weekends"`
	ExcludeHolidays       bool                  `json:"exclude_holidays"`
	IsActive              bool                  `json:"is_active"`
	PriorityScore         int                   `json:"priority_score"`
	TenantID              int                    `json:"tenant_id"`
	CreatedAt             time.Time             `json:"created_at"`
	UpdatedAt             time.Time             `json:"updated_at"`
}

// EngineerSkillDTO 工程师技能DTO
type CreateEngineerSkillRequest struct {
	UserID           int                   `json:"user_id"`
	Category         string                `json:"category"`
	SkillName        string                `json:"skill_name"`
	ProficiencyLevel int                   `json:"proficiency_level"`
	ExperienceYears  int                   `json:"experience_years"`
	Certifications   []string              `json:"certifications"`
	IsAvailable      bool                  `json:"is_available"`
	CurrentLoad      int                   `json:"current_load"`
	MaxLoad          int                   `json:"max_load"`
	PreferredShift   *string               `json:"preferred_shift"`
	WorkingHours     map[string]interface{} `json:"working_hours"`
	TenantID         int                   `json:"tenant_id"`
}

type UpdateEngineerSkillRequest struct {
	Category         *string                `json:"category,omitempty"`
	SkillName        *string                `json:"skill_name,omitempty"`
	ProficiencyLevel *int                   `json:"proficiency_level,omitempty"`
	ExperienceYears  *int                   `json:"experience_years,omitempty"`
	Certifications   *[]string              `json:"certifications,omitempty"`
	IsAvailable      *bool                  `json:"is_available,omitempty"`
	CurrentLoad      *int                   `json:"current_load,omitempty"`
	MaxLoad          *int                   `json:"max_load,omitempty"`
	PreferredShift   *string                `json:"preferred_shift,omitempty"`
	WorkingHours     *map[string]interface{} `json:"working_hours,omitempty"`
}

type EngineerSkillResponse struct {
	ID               int                    `json:"id"`
	UserID           int                   `json:"user_id"`
	Category         string                 `json:"category"`
	SkillName        string                 `json:"skill_name"`
	ProficiencyLevel int                   `json:"proficiency_level"`
	ExperienceYears  int                   `json:"experience_years"`
	Certifications   []string               `json:"certifications"`
	IsAvailable      bool                   `json:"is_available"`
	CurrentLoad      int                    `json:"current_load"`
	MaxLoad          int                    `json:"max_load"`
	PreferredShift   string                 `json:"preferred_shift"`
	WorkingHours     map[string]interface{} `json:"working_hours"`
	TenantID         int                    `json:"tenant_id"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// IncidentEscalationRuleDTO 事件升级规则DTO
type CreateIncidentEscalationRuleRequest struct {
	Name                  string                 `json:"name" binding:"required"`
	Description           string                 `json:"description"`
	TriggerType           string                 `json:"trigger_type"` // time_based/sla_breach/manual
	EscalationLevel       int                   `json:"escalation_level"` // 1/2/3
	TriggerMinutes       int                   `json:"trigger_minutes"`
	FromStatus           *string                `json:"from_status"`
	ToStatus             *string                `json:"to_status"`
	TargetAssigneeType   string                 `json:"target_assignee_type"` // group/role/user
	TargetAssigneeID     *int                   `json:"target_assignee_id"`
	TargetGroup          *string                `json:"target_group"`
	AutoEscalate         bool                  `json:"auto_escalate"`
	NotificationConfig   map[string]interface{} `json:"notification_config"`
	IsActive             bool                  `json:"is_active"`
	PriorityMatch        *string                `json:"priority_match"`
	CategoryMatch        *string                `json:"category_match"`
	TenantID             int                   `json:"tenant_id"`
}

type UpdateIncidentEscalationRuleRequest struct {
	Name                *string                `json:"name,omitempty"`
	Description         *string                `json:"description,omitempty"`
	TriggerType         *string                `json:"trigger_type,omitempty"`
	EscalationLevel     *int                   `json:"escalation_level,omitempty"`
	TriggerMinutes      *int                   `json:"trigger_minutes,omitempty"`
	FromStatus          *string                `json:"from_status,omitempty"`
	ToStatus            *string                `json:"to_status,omitempty"`
	TargetAssigneeType  *string                `json:"target_assignee_type,omitempty"`
	TargetAssigneeID    *int                   `json:"target_assignee_id,omitempty"`
	TargetGroup         *string                `json:"target_group,omitempty"`
	AutoEscalate        *bool                  `json:"auto_escalate,omitempty"`
	NotificationConfig  *map[string]interface{} `json:"notification_config,omitempty"`
	IsActive           *bool                  `json:"is_active,omitempty"`
	PriorityMatch      *string                `json:"priority_match,omitempty"`
	CategoryMatch      *string                `json:"category_match,omitempty"`
}

type IncidentEscalationRuleResponse struct {
	ID                  int                    `json:"id"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	TriggerType         string                 `json:"trigger_type"`
	EscalationLevel     int                   `json:"escalation_level"`
	TriggerMinutes      int                   `json:"trigger_minutes"`
	FromStatus         string                 `json:"from_status"`
	ToStatus           string                 `json:"to_status"`
	TargetAssigneeType  string                 `json:"target_assignee_type"`
	TargetAssigneeID    int                    `json:"target_assignee_id"`
	TargetGroup         string                 `json:"target_group"`
	AutoEscalate        bool                   `json:"auto_escalate"`
	NotificationConfig map[string]interface{} `json:"notification_config"`
	IsActive           bool                   `json:"is_active"`
	PriorityMatch      string                 `json:"priority_match"`
	CategoryMatch      string                 `json:"category_match"`
	TenantID           int                    `json:"tenant_id"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// KnownErrorDTO 已知错误DTO
type CreateKnownErrorRequest struct {
	Title            string                 `json:"title" binding:"required"`
	Description     string                 `json:"description"`
	Symptoms         string                 `json:"symptoms"`
	RootCause       string                 `json:"root_cause"`
	Workaround      string                 `json:"workaround"`
	Resolution      string                 `json:"resolution"`
	Status          string                `json:"status"` // draft/active/resolved/deprecated
	Category        string                 `json:"category"`
	Severity        string                `json:"severity"` // critical/high/medium/low
	AffectedProducts []string              `json:"affected_products"`
	AffectedCIs     []string              `json:"affected_cis"`
	Keywords        []string              `json:"keywords"`
	CreatedBy       int                   `json:"created_by"`
	ApprovedBy      *int                   `json:"approved_by"`
	ApprovedAt      *time.Time            `json:"approved_at"`
	TenantID        int                   `json:"tenant_id"`
}

type UpdateKnownErrorRequest struct {
	Title            *string                `json:"title,omitempty"`
	Description     *string                `json:"description,omitempty"`
	Symptoms         *string                `json:"symptoms,omitempty"`
	RootCause       *string                `json:"root_cause,omitempty"`
	Workaround      *string                `json:"workaround,omitempty"`
	Resolution      *string                `json:"resolution,omitempty"`
	Status          *string                `json:"status,omitempty"`
	Category        *string                `json:"category,omitempty"`
	Severity        *string                `json:"severity,omitempty"`
	AffectedProducts *[]string              `json:"affected_products,omitempty"`
	AffectedCIs     *[]string              `json:"affected_cis,omitempty"`
	Keywords        *[]string              `json:"keywords,omitempty"`
	ApprovedBy      *int                   `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time            `json:"approved_at,omitempty"`
}

type KnownErrorResponse struct {
	ID               int        `json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Symptoms        string     `json:"symptoms"`
	RootCause       string     `json:"root_cause"`
	Workaround      string     `json:"workaround"`
	Resolution     string     `json:"resolution"`
	Status         string     `json:"status"`
	Category       string     `json:"category"`
	Severity       string     `json:"severity"`
	AffectedProducts []string  `json:"affected_products"`
	AffectedCIs    []string   `json:"affected_cis"`
	Keywords       []string   `json:"keywords"`
	OccurrenceCount int       `json:"occurrence_count"`
	CreatedBy      int        `json:"created_by"`
	ApprovedBy     int        `json:"approved_by"`
	TenantID       int        `json:"tenant_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
