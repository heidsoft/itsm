package dto

import "time"

// CreatePermissionRequest 创建权限请求
type CreatePermissionRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Resource    string `json:"resource" binding:"required"`
	Action      string `json:"action" binding:"required"`
	// Policy 可选：不传时由后端按 action 语义派生安全初值（AI 零权限）。
	// 传了则走 ValidatePolicy 闭集校验，非法配置直接拒绝入库。
	Policy *PermissionPolicyPayload `json:"policy,omitempty"`
}

// PermissionPolicyPayload 是管理员在运行期可改的策略字段。
//
// 刻意不包含 code/resource/action/isSystem：权限码一旦创建就不允许在运行期
// 改写身份（会让 role_permissions 的历史引用与路由声明同时失真），
// isSystem 由引导清单决定，不接受客户端提交。
//
// 这是一次「整体替换」：前端请把当前策略原样回传再改要改的字段，
// 避免用零值把未提交的字段静默清空。
type PermissionPolicyPayload struct {
	RiskLevel         string   `json:"riskLevel" binding:"required"`
	AllowedPrincipals []string `json:"allowedPrincipals" binding:"required"`
	RequiresApproval  bool     `json:"requiresApproval"`
	MinApprovals      int      `json:"minApprovals"`
	Category          string   `json:"category" binding:"required"`
	DisplayName       string   `json:"displayName,omitempty"`
	Icon              string   `json:"icon,omitempty"`
	Color             string   `json:"color,omitempty"`
	I18nKey           string   `json:"i18nKey,omitempty"`
	SortOrder         int      `json:"sortOrder"`
	IsActive          bool     `json:"isActive"`
}

// UpdatePermissionRequest 更新权限的名称/描述/策略（不改码身份）。
type UpdatePermissionRequest struct {
	Name        string                   `json:"name,omitempty"`
	Description string                   `json:"description,omitempty"`
	Policy      *PermissionPolicyPayload `json:"policy" binding:"required"`
}

// PermissionListResponse 权限字典响应（不分页，只含 items/total）
type PermissionListResponse struct {
	Items []*PermissionResponse `json:"items"`
	Total int                   `json:"total"`
}

// PermissionResponse 权限响应。
//
// 字段全部由 DB 的 permissions 行 + 版本化策略信封派生；
// 前端不得再内置模块/动作字典，展示所需的分类、风险、主体、展示元数据都在这里。
type PermissionResponse struct {
	ID          int       `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	TenantID    int       `json:"tenantId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`

	// 运行期策略
	RiskLevel         string   `json:"riskLevel"`
	AllowedPrincipals []string `json:"allowedPrincipals"`
	RequiresApproval  bool     `json:"requiresApproval"`
	MinApprovals      int      `json:"minApprovals"`
	IsActive          bool     `json:"isActive"`
	IsSystem          bool     `json:"isSystem"`
	// Configured=false 表示这行还是历史纯文本描述、策略是派生初值，
	// 管理页据此提示管理员显式确认一次（派生初值恒为 AI 零权限）。
	Configured bool `json:"configured"`

	// 展示元数据（前端按这些字段渲染，不再硬编码）
	Category      string `json:"category"`
	CategoryLabel string `json:"categoryLabel"`
	DisplayName   string `json:"displayName,omitempty"`
	Icon          string `json:"icon,omitempty"`
	Color         string `json:"color,omitempty"`
	I18nKey       string `json:"i18nKey,omitempty"`
	SortOrder     int    `json:"sortOrder"`
}

// PermissionCatalogMetaResponse 给前端的展示口径词表（闭集来自后端，前端零硬编码）。
type PermissionCatalogMetaResponse struct {
	Principals      []string          `json:"principals"`
	RiskLevels      []string          `json:"riskLevels"`
	Categories      []string          `json:"categories"`
	CategoryLabels  map[string]string `json:"categoryLabels"`
	MinApprovalsMax int               `json:"minApprovalsMax"`
}
