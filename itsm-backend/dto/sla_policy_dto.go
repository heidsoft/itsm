package dto

import (
	"time"
)

// SLAPolicyDTO SLA策略DTO
type CreateSLAPolicyRequest struct {
	Name                  string                 `json:"name" binding:"required"`
	Description           string                 `json:"description"`
	CustomerTier          *string                `json:"customerTier"` // platinum/gold/silver/bronze
	TicketType            *string                `json:"ticketType"`   // incident/problem/change/request
	Priority              *string                `json:"priority"`     // critical/high/medium/low
	ResponseTimeMinutes   int                    `json:"responseTimeMinutes"`
	ResolutionTimeMinutes int                    `json:"resolutionTimeMinutes"`
	BusinessHours         map[string]interface{} `json:"businessHours"`
	ExcludeWeekends       bool                   `json:"excludeWeekends"`
	ExcludeHolidays       bool                   `json:"excludeHolidays"`
	EscalationRules       map[string]interface{} `json:"escalationRules"`
	IsActive              bool                   `json:"isActive"`
	PriorityScore         int                    `json:"priorityScore"`
	TenantID              int                    `json:"tenantId"`
}

type UpdateSLAPolicyRequest struct {
	Name                  *string                 `json:"name,omitempty"`
	Description           *string                 `json:"description,omitempty"`
	CustomerTier          *string                 `json:"customerTier,omitempty"`
	TicketType            *string                 `json:"ticketType,omitempty"`
	Priority              *string                 `json:"priority,omitempty"`
	ResponseTimeMinutes   *int                    `json:"responseTimeMinutes,omitempty"`
	ResolutionTimeMinutes *int                    `json:"resolutionTimeMinutes,omitempty"`
	BusinessHours         *map[string]interface{} `json:"businessHours,omitempty"`
	ExcludeWeekends       *bool                   `json:"excludeWeekends,omitempty"`
	ExcludeHolidays       *bool                   `json:"excludeHolidays,omitempty"`
	EscalationRules       *map[string]interface{} `json:"escalationRules,omitempty"`
	IsActive              *bool                   `json:"isActive,omitempty"`
	PriorityScore         *int                    `json:"priorityScore,omitempty"`
}

type SLAPolicyResponse struct {
	ID                    int                    `json:"id"`
	Name                  string                 `json:"name"`
	Description           string                 `json:"description"`
	CustomerTier          string                 `json:"customerTier"`
	TicketType            string                 `json:"ticketType"`
	Priority              string                 `json:"priority"`
	ResponseTimeMinutes   int                    `json:"responseTimeMinutes"`
	ResolutionTimeMinutes int                    `json:"resolutionTimeMinutes"`
	BusinessHours         map[string]interface{} `json:"businessHours"`
	ExcludeWeekends       bool                   `json:"excludeWeekends"`
	ExcludeHolidays       bool                   `json:"excludeHolidays"`
	IsActive              bool                   `json:"isActive"`
	PriorityScore         int                    `json:"priorityScore"`
	TenantID              int                    `json:"tenantId"`
	CreatedAt             time.Time              `json:"createdAt"`
	UpdatedAt             time.Time              `json:"updatedAt"`
}

// EngineerSkillDTO 工程师技能DTO
type CreateEngineerSkillRequest struct {
	UserID           int                    `json:"userId"`
	Category         string                 `json:"category"`
	SkillName        string                 `json:"skillName"`
	ProficiencyLevel int                    `json:"proficiencyLevel"`
	ExperienceYears  int                    `json:"experienceYears"`
	Certifications   []string               `json:"certifications"`
	IsAvailable      bool                   `json:"isAvailable"`
	CurrentLoad      int                    `json:"currentLoad"`
	MaxLoad          int                    `json:"maxLoad"`
	PreferredShift   *string                `json:"preferredShift"`
	WorkingHours     map[string]interface{} `json:"workingHours"`
	TenantID         int                    `json:"tenantId"`
}

type UpdateEngineerSkillRequest struct {
	Category         *string                 `json:"category,omitempty"`
	SkillName        *string                 `json:"skillName,omitempty"`
	ProficiencyLevel *int                    `json:"proficiencyLevel,omitempty"`
	ExperienceYears  *int                    `json:"experienceYears,omitempty"`
	Certifications   *[]string               `json:"certifications,omitempty"`
	IsAvailable      *bool                   `json:"isAvailable,omitempty"`
	CurrentLoad      *int                    `json:"currentLoad,omitempty"`
	MaxLoad          *int                    `json:"maxLoad,omitempty"`
	PreferredShift   *string                 `json:"preferredShift,omitempty"`
	WorkingHours     *map[string]interface{} `json:"workingHours,omitempty"`
}

type EngineerSkillResponse struct {
	ID               int                    `json:"id"`
	UserID           int                    `json:"userId"`
	Category         string                 `json:"category"`
	SkillName        string                 `json:"skillName"`
	ProficiencyLevel int                    `json:"proficiencyLevel"`
	ExperienceYears  int                    `json:"experienceYears"`
	Certifications   []string               `json:"certifications"`
	IsAvailable      bool                   `json:"isAvailable"`
	CurrentLoad      int                    `json:"currentLoad"`
	MaxLoad          int                    `json:"maxLoad"`
	PreferredShift   string                 `json:"preferredShift"`
	WorkingHours     map[string]interface{} `json:"workingHours"`
	TenantID         int                    `json:"tenantId"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
}

// IncidentEscalationRuleDTO 事件升级规则DTO
type CreateIncidentEscalationRuleRequest struct {
	Name               string                 `json:"name" binding:"required"`
	Description        string                 `json:"description"`
	TriggerType        string                 `json:"triggerType"`     // time_based/sla_breach/manual
	EscalationLevel    int                    `json:"escalationLevel"` // 1/2/3
	TriggerMinutes     int                    `json:"triggerMinutes"`
	FromStatus         *string                `json:"fromStatus"`
	ToStatus           *string                `json:"toStatus"`
	TargetAssigneeType string                 `json:"targetAssigneeType"` // group/role/user
	TargetAssigneeID   *int                   `json:"targetAssigneeId"`
	TargetGroup        *string                `json:"targetGroup"`
	AutoEscalate       bool                   `json:"autoEscalate"`
	NotificationConfig map[string]interface{} `json:"notificationConfig"`
	IsActive           bool                   `json:"isActive"`
	PriorityMatch      *string                `json:"priorityMatch"`
	CategoryMatch      *string                `json:"categoryMatch"`
	TenantID           int                    `json:"tenantId"`
}

type UpdateIncidentEscalationRuleRequest struct {
	Name               *string                 `json:"name,omitempty"`
	Description        *string                 `json:"description,omitempty"`
	TriggerType        *string                 `json:"triggerType,omitempty"`
	EscalationLevel    *int                    `json:"escalationLevel,omitempty"`
	TriggerMinutes     *int                    `json:"triggerMinutes,omitempty"`
	FromStatus         *string                 `json:"fromStatus,omitempty"`
	ToStatus           *string                 `json:"toStatus,omitempty"`
	TargetAssigneeType *string                 `json:"targetAssigneeType,omitempty"`
	TargetAssigneeID   *int                    `json:"targetAssigneeId,omitempty"`
	TargetGroup        *string                 `json:"targetGroup,omitempty"`
	AutoEscalate       *bool                   `json:"autoEscalate,omitempty"`
	NotificationConfig *map[string]interface{} `json:"notificationConfig,omitempty"`
	IsActive           *bool                   `json:"isActive,omitempty"`
	PriorityMatch      *string                 `json:"priorityMatch,omitempty"`
	CategoryMatch      *string                 `json:"categoryMatch,omitempty"`
}

type IncidentEscalationRuleResponse struct {
	ID                 int                    `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	TriggerType        string                 `json:"triggerType"`
	EscalationLevel    int                    `json:"escalationLevel"`
	TriggerMinutes     int                    `json:"triggerMinutes"`
	FromStatus         string                 `json:"fromStatus"`
	ToStatus           string                 `json:"toStatus"`
	TargetAssigneeType string                 `json:"targetAssigneeType"`
	TargetAssigneeID   int                    `json:"targetAssigneeId"`
	TargetGroup        string                 `json:"targetGroup"`
	AutoEscalate       bool                   `json:"autoEscalate"`
	NotificationConfig map[string]interface{} `json:"notificationConfig"`
	IsActive           bool                   `json:"isActive"`
	PriorityMatch      string                 `json:"priorityMatch"`
	CategoryMatch      string                 `json:"categoryMatch"`
	TenantID           int                    `json:"tenantId"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
}

type UpdateKnownErrorRequest struct {
	Title            *string    `json:"title,omitempty"`
	Description      *string    `json:"description,omitempty"`
	Symptoms         *string    `json:"symptoms,omitempty"`
	RootCause        *string    `json:"rootCause,omitempty"`
	Workaround       *string    `json:"workaround,omitempty"`
	Resolution       *string    `json:"resolution,omitempty"`
	Status           *string    `json:"status,omitempty"`
	Category         *string    `json:"category,omitempty"`
	Severity         *string    `json:"severity,omitempty"`
	AffectedProducts *[]string  `json:"affectedProducts,omitempty"`
	AffectedCIs      *[]string  `json:"affectedCis,omitempty"`
	Keywords         *[]string  `json:"keywords,omitempty"`
	ApprovedBy       *int       `json:"approvedBy,omitempty"`
	ApprovedAt       *time.Time `json:"approvedAt,omitempty"`
}

type KnownErrorResponse struct {
	ID               int       `json:"id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Symptoms         string    `json:"symptoms"`
	RootCause        string    `json:"rootCause"`
	Workaround       string    `json:"workaround"`
	Resolution       string    `json:"resolution"`
	Status           string    `json:"status"`
	Category         string    `json:"category"`
	Severity         string    `json:"severity"`
	AffectedProducts []string  `json:"affectedProducts"`
	AffectedCIs      []string  `json:"affectedCis"`
	Keywords         []string  `json:"keywords"`
	OccurrenceCount  int       `json:"occurrenceCount"`
	CreatedBy        int       `json:"createdBy"`
	ApprovedBy       int       `json:"approvedBy"`
	TenantID         int       `json:"tenantId"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
