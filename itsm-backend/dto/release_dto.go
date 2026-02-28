package dto

import "time"

// ReleaseStatus 发布状态
type ReleaseStatus string

const (
	ReleaseStatusDraft      ReleaseStatus = "draft"       // 草稿
	ReleaseStatusScheduled  ReleaseStatus = "scheduled"   // 已计划
	ReleaseStatusInProgress ReleaseStatus = "in-progress" // 进行中
	ReleaseStatusCompleted  ReleaseStatus = "completed"   // 已完成
	ReleaseStatusCancelled  ReleaseStatus = "cancelled"   // 已取消
	ReleaseStatusFailed     ReleaseStatus = "failed"      // 失败
	ReleaseStatusRolledBack ReleaseStatus = "rolled_back" // 已回滚
)

// ReleaseType 发布类型
type ReleaseType string

const (
	ReleaseTypeMajor  ReleaseType = "major"  // 主版本
	ReleaseTypeMinor  ReleaseType = "minor"  // 次版本
	ReleaseTypePatch  ReleaseType = "patch"  // 补丁
	ReleaseTypeHotfix ReleaseType = "hotfix" // 紧急修复
)

// ReleaseSeverity 发布严重程度
type ReleaseSeverity string

const (
	ReleaseSeverityLow      ReleaseSeverity = "low"      // 低
	ReleaseSeverityMedium   ReleaseSeverity = "medium"   // 中
	ReleaseSeverityHigh     ReleaseSeverity = "high"     // 高
	ReleaseSeverityCritical ReleaseSeverity = "critical" // 严重
)

// ReleaseEnvironment 目标环境
type ReleaseEnvironment string

const (
	ReleaseEnvironmentDev        ReleaseEnvironment = "dev"        // 开发环境
	ReleaseEnvironmentStaging    ReleaseEnvironment = "staging"    // 预发布环境
	ReleaseEnvironmentProduction ReleaseEnvironment = "production" // 生产环境
)

// CreateReleaseRequest 创建发布请求
type CreateReleaseRequest struct {
	ReleaseNumber      string     `json:"release_number" binding:"required"` // 发布编号
	Title              string     `json:"title" binding:"required"`          // 发布标题
	Description        string     `json:"description"`                       // 发布描述
	Type               string     `json:"type"`                              // 发布类型
	Environment        string     `json:"environment"`                       // 目标环境
	Severity           string     `json:"severity"`                          // 严重程度
	ChangeID           *int       `json:"change_id"`                         // 关联变更ID
	OwnerID            *int       `json:"owner_id"`                          // 负责人ID
	PlannedReleaseDate *time.Time `json:"planned_release_date"`              // 计划发布日期
	PlannedStartDate   *time.Time `json:"planned_start_date"`                // 计划开始时间
	PlannedEndDate     *time.Time `json:"planned_end_date"`                  // 计划结束时间
	ReleaseNotes       string     `json:"release_notes"`                     // 发布说明
	RollbackProcedure  string     `json:"rollback_procedure"`                // 回滚程序
	ValidationCriteria string     `json:"validation_criteria"`               // 验证标准
	AffectedSystems    []string   `json:"affected_systems"`                  // 受影响的系统
	AffectedComponents []string   `json:"affected_components"`               // 受影响的组件
	DeploymentSteps    []string   `json:"deployment_steps"`                  // 部署步骤
	Tags               []string   `json:"tags"`                              // 标签
	IsEmergency        bool       `json:"is_emergency"`                      // 是否紧急发布
	RequiresApproval   bool       `json:"requires_approval"`                 // 是否需要审批
}

// UpdateReleaseRequest 更新发布请求
type UpdateReleaseRequest struct {
	Title              *string    `json:"title"`                // 发布标题
	Description        *string    `json:"description"`          // 发布描述
	Type               *string    `json:"type"`                 // 发布类型
	Environment        *string    `json:"environment"`          // 目标环境
	Severity           *string    `json:"severity"`             // 严重程度
	ChangeID           *int       `json:"change_id"`            // 关联变更ID
	OwnerID            *int       `json:"owner_id"`             // 负责人ID
	PlannedReleaseDate *time.Time `json:"planned_release_date"` // 计划发布日期
	PlannedStartDate   *time.Time `json:"planned_start_date"`   // 计划开始时间
	PlannedEndDate     *time.Time `json:"planned_end_date"`     // 计划结束时间
	ActualReleaseDate  *time.Time `json:"actual_release_date"`  // 实际发布日期
	ReleaseNotes       *string    `json:"release_notes"`        // 发布说明
	RollbackProcedure  *string    `json:"rollback_procedure"`   // 回滚程序
	ValidationCriteria *string    `json:"validation_criteria"`  // 验证标准
	AffectedSystems    []string   `json:"affected_systems"`     // 受影响的系统
	AffectedComponents []string   `json:"affected_components"`  // 受影响的组件
	DeploymentSteps    []string   `json:"deployment_steps"`     // 部署步骤
	Tags               []string   `json:"tags"`                 // 标签
	IsEmergency        *bool      `json:"is_emergency"`         // 是否紧急发布
	RequiresApproval   *bool      `json:"requires_approval"`    // 是否需要审批
}

// ReleaseResponse 发布响应
type ReleaseResponse struct {
	ID                 int        `json:"id"`                   // 发布ID
	ReleaseNumber      string     `json:"release_number"`       // 发布编号
	Title              string     `json:"title"`                // 发布标题
	Description        string     `json:"description"`          // 发布描述
	Type               string     `json:"type"`                 // 发布类型
	Status             string     `json:"status"`               // 状态
	Severity           string     `json:"severity"`             // 严重程度
	Environment        string     `json:"environment"`          // 目标环境
	ChangeID           *int       `json:"change_id"`            // 关联变更ID
	OwnerID            *int       `json:"owner_id"`             // 负责人ID
	OwnerName          *string    `json:"owner_name"`           // 负责人姓名
	CreatedBy          int        `json:"created_by"`           // 创建人ID
	CreatedByName      string     `json:"created_by_name"`      // 创建人姓名
	TenantID           int        `json:"tenant_id"`            // 租户ID
	PlannedReleaseDate *time.Time `json:"planned_release_date"` // 计划发布日期
	ActualReleaseDate  *time.Time `json:"actual_release_date"`  // 实际发布日期
	PlannedStartDate   *time.Time `json:"planned_start_date"`   // 计划开始时间
	PlannedEndDate     *time.Time `json:"planned_end_date"`     // 计划结束时间
	ReleaseNotes       string     `json:"release_notes"`        // 发布说明
	RollbackProcedure  string     `json:"rollback_procedure"`   // 回滚程序
	ValidationCriteria string     `json:"validation_criteria"`  // 验证标准
	AffectedSystems    []string   `json:"affected_systems"`     // 受影响的系统
	AffectedComponents []string   `json:"affected_components"`  // 受影响的组件
	DeploymentSteps    []string   `json:"deployment_steps"`     // 部署步骤
	Tags               []string   `json:"tags"`                 // 标签
	IsEmergency        bool       `json:"is_emergency"`         // 是否紧急发布
	RequiresApproval   bool       `json:"requires_approval"`    // 是否需要审批
	CreatedAt          time.Time  `json:"created_at"`           // 创建时间
	UpdatedAt          time.Time  `json:"updated_at"`           // 更新时间
}

// ReleaseListResponse 发布列表响应
type ReleaseListResponse struct {
	Total    int               `json:"total"`    // 总数
	Releases []ReleaseResponse `json:"releases"` // 发布列表
}

// ReleaseStatsResponse 发布统计响应
type ReleaseStatsResponse struct {
	Total      int `json:"total"`       // 总数
	Draft      int `json:"draft"`       // 草稿
	Scheduled  int `json:"scheduled"`   // 已计划
	InProgress int `json:"in_progress"` // 进行中
	Completed  int `json:"completed"`   // 已完成
	Cancelled  int `json:"cancelled"`   // 已取消
	Failed     int `json:"failed"`      // 失败
	RolledBack int `json:"rolled_back"` // 已回滚
}

// ReleaseStatusUpdateRequest 发布状态更新请求
type ReleaseStatusUpdateRequest struct {
	Status  ReleaseStatus `json:"status" binding:"required"` // 新状态
	Comment *string       `json:"comment"`                   // 状态变更说明
}

// ReleaseApprovalRequest 发布审批请求
type ReleaseApprovalRequest struct {
	Status  string  `json:"status" binding:"required"` // 审批状态: approved, rejected
	Comment *string `json:"comment"`                   // 审批意见
}
