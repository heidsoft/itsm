package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/bpmnpermission"

	"go.uber.org/zap"
)

// BPMNPermissionService BPMN权限服务
type BPMNPermissionService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewBPMNPermissionService 创建BPMN权限服务
func NewBPMNPermissionService(client *ent.Client, logger *zap.SugaredLogger) *BPMNPermissionService {
	return &BPMNPermissionService{
		client: client,
		logger: logger,
	}
}

// PermissionType 权限类型
const (
	PermissionTypeRead     = "read"
	PermissionTypeWrite    = "write"
	PermissionTypeExecute = "execute"
	PermissionTypeAdmin   = "admin"
	PermissionTypeAssign  = "assign"
	PermissionTypeComplete = "complete"
	PermissionTypeDelegate = "delegate"
	PermissionTypeEscalate = "escalate"
)

// ResourceType 资源类型
const (
	ResourceTypeProcessDefinition = "process_definition"
	ResourceTypeProcessInstance  = "process_instance"
	ResourceTypeTask             = "task"
)

// PrincipalType 授权主体类型
const (
	PrincipalTypeUser      = "user"
	PrincipalTypeRole     = "role"
	PrincipalTypeGroup    = "group"
	PrincipalTypeDepartment = "department"
)

// GrantPermission 授予权限
func (s *BPMNPermissionService) GrantPermission(ctx context.Context, req *GrantPermissionRequest) (*ent.BPMNPermission, error) {
	// 检查权限是否已存在
	existing, err := s.client.BPMNPermission.Query().
		Where(bpmnpermission.ResourceType(req.ResourceType)).
		Where(bpmnpermission.ResourceID(req.ResourceID)).
		Where(bpmnpermission.PrincipalType(req.PrincipalType)).
		Where(bpmnpermission.PrincipalID(req.PrincipalID)).
		Where(bpmnpermission.PermissionType(req.PermissionType)).
		First(ctx)
	if err == nil {
		// 权限已存在，更新
		update := s.client.BPMNPermission.UpdateOne(existing).
			SetIsGranted(req.IsGranted).
			SetConditions(req.Conditions).
			SetFieldPermissions(req.FieldPermissions)
		if !req.ExpiresAt.IsZero() {
			update = update.SetExpiresAt(req.ExpiresAt)
		}
		return update.Save(ctx)
	}

	// 创建新权限
	create := s.client.BPMNPermission.Create().
		SetResourceType(req.ResourceType).
		SetResourceID(req.ResourceID).
		SetResourceKey(req.ResourceKey).
		SetPermissionType(req.PermissionType).
		SetPrincipalType(req.PrincipalType).
		SetPrincipalID(req.PrincipalID).
		SetIsGranted(req.IsGranted).
		SetConditions(req.Conditions).
		SetFieldPermissions(req.FieldPermissions).
		SetDescription(req.Description).
		SetTenantID(req.TenantID)
	if !req.ExpiresAt.IsZero() {
		create = create.SetExpiresAt(req.ExpiresAt)
	}
	return create.Save(ctx)
}

// RevokePermission 撤销权限
func (s *BPMNPermissionService) RevokePermission(ctx context.Context, req *RevokePermissionRequest) error {
	_, err := s.client.BPMNPermission.Delete().
		Where(bpmnpermission.ResourceType(req.ResourceType)).
		Where(bpmnpermission.ResourceID(req.ResourceID)).
		Where(bpmnpermission.PrincipalType(req.PrincipalType)).
		Where(bpmnpermission.PrincipalID(req.PrincipalID)).
		Where(bpmnpermission.PermissionType(req.PermissionType)).
		Exec(ctx)
	return err
}

// CheckPermission 检查权限
func (s *BPMNPermissionService) CheckPermission(ctx context.Context, resourceType string, resourceID int, resourceKey string, principalType string, principalID int, permissionType string, tenantID int) (bool, error) {
	// 首先检查是否有直接权限
	permission, err := s.client.BPMNPermission.Query().
		Where(bpmnpermission.ResourceType(resourceType)).
		Where(bpmnpermission.ResourceID(resourceID)).
		Where(bpmnpermission.PrincipalType(principalType)).
		Where(bpmnpermission.PrincipalID(principalID)).
		Where(bpmnpermission.PermissionType(permissionType)).
		Where(bpmnpermission.TenantID(tenantID)).
		Where(bpmnpermission.IsGranted(true)).
		First(ctx)
	if err == nil {
		// 检查是否过期
		if !permission.ExpiresAt.IsZero() && permission.ExpiresAt.Before(time.Now()) {
			return false, nil
		}
		return true, nil
	}

	// 检查基于resourceKey的权限
	if resourceKey != "" {
		permission, err = s.client.BPMNPermission.Query().
			Where(bpmnpermission.ResourceType(resourceType)).
			Where(bpmnpermission.ResourceKey(resourceKey)).
			Where(bpmnpermission.PrincipalType(principalType)).
			Where(bpmnpermission.PrincipalID(principalID)).
			Where(bpmnpermission.PermissionType(permissionType)).
			Where(bpmnpermission.TenantID(tenantID)).
			Where(bpmnpermission.IsGranted(true)).
			First(ctx)
		if err == nil {
			if !permission.ExpiresAt.IsZero() && permission.ExpiresAt.Before(time.Now()) {
				return false, nil
			}
			return true, nil
		}
	}

	return false, nil
}

// CheckProcessDefinitionPermission 检查流程定义权限
func (s *BPMNPermissionService) CheckProcessDefinitionPermission(ctx context.Context, definitionID int, definitionKey string, userID int, roleIDs []int, permissionType string, tenantID int) (bool, error) {
	// 1. 检查用户直接权限
	hasPermission, err := s.CheckPermission(ctx, ResourceTypeProcessDefinition, definitionID, definitionKey, PrincipalTypeUser, userID, permissionType, tenantID)
	if err != nil {
		return false, err
	}
	if hasPermission {
		return true, nil
	}

	// 2. 检查用户角色权限
	for _, roleID := range roleIDs {
		hasPermission, err = s.CheckPermission(ctx, ResourceTypeProcessDefinition, definitionID, definitionKey, PrincipalTypeRole, roleID, permissionType, tenantID)
		if err != nil {
			return false, err
		}
		if hasPermission {
			return true, nil
		}
	}

	// 3. 检查是否具有admin权限（admin拥有所有权限）
	hasAdmin, err := s.CheckPermission(ctx, ResourceTypeProcessDefinition, definitionID, definitionKey, PrincipalTypeUser, userID, PermissionTypeAdmin, tenantID)
	if err != nil {
		return false, err
	}
	if hasAdmin {
		return true, nil
	}
	for _, roleID := range roleIDs {
		hasAdmin, err = s.CheckPermission(ctx, ResourceTypeProcessDefinition, definitionID, definitionKey, PrincipalTypeRole, roleID, PermissionTypeAdmin, tenantID)
		if err != nil {
			return false, err
		}
		if hasAdmin {
			return true, nil
		}
	}

	return false, nil
}

// CheckProcessInstancePermission 检查流程实例权限
func (s *BPMNPermissionService) CheckProcessInstancePermission(ctx context.Context, instanceID int, instance *ent.ProcessInstance, userID int, roleIDs []int, permissionType string, tenantID int) (bool, error) {
	// 1. 发起人总是有权限查看自己的流程实例
	if permissionType == PermissionTypeRead && instance.Initiator != "" {
		if fmt.Sprintf("%d", userID) == instance.Initiator {
			return true, nil
		}
	}

	// 2. 检查直接权限
	hasPermission, err := s.CheckPermission(ctx, ResourceTypeProcessInstance, instanceID, instance.ProcessDefinitionKey, PrincipalTypeUser, userID, permissionType, tenantID)
	if err != nil {
		return false, err
	}
	if hasPermission {
		return true, nil
	}

	// 3. 检查角色权限
	for _, roleID := range roleIDs {
		hasPermission, err = s.CheckPermission(ctx, ResourceTypeProcessInstance, instanceID, instance.ProcessDefinitionKey, PrincipalTypeRole, roleID, permissionType, tenantID)
		if err != nil {
			return false, err
		}
		if hasPermission {
			return true, nil
		}
	}

	// 4. 检查是否具有admin权限
	hasAdmin, _ := s.CheckPermission(ctx, ResourceTypeProcessInstance, instanceID, instance.ProcessDefinitionKey, PrincipalTypeUser, userID, PermissionTypeAdmin, tenantID)
	if hasAdmin {
		return true, nil
	}
	for _, roleID := range roleIDs {
		hasAdmin, err = s.CheckPermission(ctx, ResourceTypeProcessInstance, instanceID, instance.ProcessDefinitionKey, PrincipalTypeRole, roleID, PermissionTypeAdmin, tenantID)
		if err != nil {
			return false, err
		}
		if hasAdmin {
			return true, nil
		}
	}

	return false, nil
}

// CheckTaskPermission 检查任务权限
func (s *BPMNPermissionService) CheckTaskPermission(ctx context.Context, taskID int, task *ent.ProcessTask, userID int, roleIDs []int, permissionType string, tenantID int) (bool, error) {
	// 1. 任务是分配给用户的，用户总是可以完成自己的任务
	if permissionType == PermissionTypeComplete && task.Assignee != "" {
		if fmt.Sprintf("%d", userID) == task.Assignee {
			return true, nil
		}
	}

	// 2. 检查直接权限
	hasPermission, err := s.CheckPermission(ctx, ResourceTypeTask, taskID, task.ProcessDefinitionKey, PrincipalTypeUser, userID, permissionType, tenantID)
	if err != nil {
		return false, err
	}
	if hasPermission {
		return true, nil
	}

	// 3. 检查角色权限
	for _, roleID := range roleIDs {
		hasPermission, err = s.CheckPermission(ctx, ResourceTypeTask, taskID, task.ProcessDefinitionKey, PrincipalTypeRole, roleID, permissionType, tenantID)
		if err != nil {
			return false, err
		}
		if hasPermission {
			return true, nil
		}
	}

	// 4. 检查是否具有admin或assign权限
	hasAdmin, _ := s.CheckPermission(ctx, ResourceTypeTask, taskID, task.ProcessDefinitionKey, PrincipalTypeUser, userID, PermissionTypeAdmin, tenantID)
	if hasAdmin {
		return true, nil
	}
	if permissionType == PermissionTypeAssign {
		hasAssign, _ := s.CheckPermission(ctx, ResourceTypeTask, taskID, task.ProcessDefinitionKey, PrincipalTypeUser, userID, PermissionTypeAssign, tenantID)
		if hasAssign {
			return true, nil
		}
	}

	return false, nil
}

// GetUserPermissions 获取用户权限列表
func (s *BPMNPermissionService) GetUserPermissions(ctx context.Context, userID int, tenantID int) ([]*ent.BPMNPermission, error) {
	return s.client.BPMNPermission.Query().
		Where(bpmnpermission.PrincipalType(PrincipalTypeUser)).
		Where(bpmnpermission.PrincipalID(userID)).
		Where(bpmnpermission.TenantID(tenantID)).
		Where(bpmnpermission.IsGranted(true)).
		Order(ent.Desc(bpmnpermission.FieldCreatedAt)).
		All(ctx)
}

// GetRolePermissions 获取角色权限列表
func (s *BPMNPermissionService) GetRolePermissions(ctx context.Context, roleID int, tenantID int) ([]*ent.BPMNPermission, error) {
	return s.client.BPMNPermission.Query().
		Where(bpmnpermission.PrincipalType(PrincipalTypeRole)).
		Where(bpmnpermission.PrincipalID(roleID)).
		Where(bpmnpermission.TenantID(tenantID)).
		Order(ent.Desc(bpmnpermission.FieldCreatedAt)).
		All(ctx)
}

// GetResourcePermissions 获取资源权限列表
func (s *BPMNPermissionService) GetResourcePermissions(ctx context.Context, resourceType string, resourceID int) ([]*ent.BPMNPermission, error) {
	return s.client.BPMNPermission.Query().
		Where(bpmnpermission.ResourceType(resourceType)).
		Where(bpmnpermission.ResourceID(resourceID)).
		Order(ent.Desc(bpmnpermission.FieldCreatedAt)).
		All(ctx)
}

// CleanExpiredPermissions 清理过期权限
func (s *BPMNPermissionService) CleanExpiredPermissions(ctx context.Context) (int, error) {
	affectedRows, err := s.client.BPMNPermission.Delete().
		Where(bpmnpermission.ExpiresAtLT(time.Now())).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return affectedRows, nil
}

// GrantPermissionRequest 授予权限请求
type GrantPermissionRequest struct {
	ResourceType    string
	ResourceID     int
	ResourceKey    string
	PermissionType string
	PrincipalType  string
	PrincipalID    int
	IsGranted      bool
	Conditions     string
	FieldPermissions string
	Description    string
	TenantID       int
	ExpiresAt      time.Time
}

// RevokePermissionRequest 撤销权限请求
type RevokePermissionRequest struct {
	ResourceType   string
	ResourceID    int
	PrincipalType string
	PrincipalID   int
	PermissionType string
}
