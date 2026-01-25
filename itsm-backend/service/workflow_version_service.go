package service

import (
	"context"
	"fmt"
	"itsm-backend/ent"
	"itsm-backend/ent/workflowversion"

	"go.uber.org/zap"
)

type WorkflowVersionService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewWorkflowVersionService(client *ent.Client, logger *zap.SugaredLogger) *WorkflowVersionService {
	return &WorkflowVersionService{
		client: client,
		logger: logger,
	}
}

// CreateVersion 创建工作流版本
func (wvs *WorkflowVersionService) CreateVersion(ctx context.Context, workflowID int, version string, bpmnXML string, changeLog string, tenantID int) (*ent.WorkflowVersion, error) {
	wvs.logger.Infow("Creating workflow version", "workflow_id", workflowID, "version", version, "tenant_id", tenantID)

	// 检查版本是否已存在
	_, err := wvs.client.WorkflowVersion.Query().
		Where(workflowversion.WorkflowID(workflowID)).
		Where(workflowversion.Version(version)).
		Only(ctx)

	if err == nil {
		return nil, fmt.Errorf("version %s already exists", version)
	}

	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to check existing version: %w", err)
	}

	// 获取当前版本并取消当前状态
	_, err = wvs.client.WorkflowVersion.Query().
		Where(workflowversion.WorkflowID(workflowID)).
		Where(workflowversion.IsCurrent(true)).
		Only(ctx)

	if err == nil {
		// 取消当前版本的 is_current 状态
		wvs.client.WorkflowVersion.Update().
			Where(workflowversion.WorkflowID(workflowID), workflowversion.IsCurrent(true)).
			SetIsCurrent(false).
			Exec(ctx)
	} else if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	// 创建新版本
	newVersion, err := wvs.client.WorkflowVersion.Create().
		SetTenantID(tenantID).
		SetWorkflowID(workflowID).
		SetVersion(version).
		SetBpmnXML(bpmnXML).
		SetStatus("draft").
		SetCreatedBy("system"). // 用户ID需要从调用方传入或从上下文获取
		SetChangeLog(changeLog).
		SetIsCurrent(true).
		Save(ctx)

	if err != nil {
		wvs.logger.Errorw("Failed to create workflow version", "error", err)
		return nil, fmt.Errorf("failed to create workflow version: %w", err)
	}

	wvs.logAuditEvent(ctx, "workflow_version_created", newVersion.ID, tenantID, map[string]interface{}{
		"workflow_id": workflowID,
		"version":     version,
		"change_log":  changeLog,
	})

	return newVersion, nil
}

// DeployVersion 部署工作流版本
func (wvs *WorkflowVersionService) DeployVersion(ctx context.Context, workflowID int, version string, tenantID int) error {
	wvs.logger.Infow("Deploying workflow version", "workflow_id", workflowID, "version", version, "tenant_id", tenantID)

	// 获取要部署的版本
	versionToDeploy, err := wvs.client.WorkflowVersion.Query().
		Where(workflowversion.WorkflowID(workflowID)).
		Where(workflowversion.Version(version)).
		Only(ctx)

	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	// 取消当前版本的 is_current 状态
	wvs.client.WorkflowVersion.Update().
		Where(workflowversion.WorkflowID(workflowID), workflowversion.IsCurrent(true)).
		SetIsCurrent(false).
		Exec(ctx)

	// 部署新版本
	_, err = versionToDeploy.Update().
		SetStatus("active").
		SetIsCurrent(true).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to deploy version: %w", err)
	}

	wvs.logAuditEvent(ctx, "workflow_version_deployed", versionToDeploy.ID, tenantID, map[string]interface{}{
		"workflow_id": workflowID,
		"version":     version,
	})

	return nil
}

// GetVersion 获取指定版本
func (wvs *WorkflowVersionService) GetVersion(ctx context.Context, workflowID int, version string, tenantID int) (*ent.WorkflowVersion, error) {
	ver, err := wvs.client.WorkflowVersion.Query().
		Where(workflowversion.WorkflowID(workflowID)).
		Where(workflowversion.Version(version)).
		Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("version not found: %w", err)
	}

	return ver, nil
}

// GetCurrentVersion 获取当前版本
func (wvs *WorkflowVersionService) GetCurrentVersion(ctx context.Context, workflowID int, tenantID int) (*ent.WorkflowVersion, error) {
	ver, err := wvs.client.WorkflowVersion.Query().
		Where(workflowversion.WorkflowID(workflowID)).
		Where(workflowversion.IsCurrent(true)).
		Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("current version not found: %w", err)
	}

	return ver, nil
}

// ListVersions 获取版本列表
func (wvs *WorkflowVersionService) ListVersions(ctx context.Context, workflowID int, tenantID int) ([]*ent.WorkflowVersion, error) {
	versions, err := wvs.client.WorkflowVersion.Query().
		Where(workflowversion.WorkflowID(workflowID)).
		Where(workflowversion.TenantID(tenantID)).
		Order(ent.Desc(workflowversion.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	return versions, nil
}

// DeleteVersion 删除版本
func (wvs *WorkflowVersionService) DeleteVersion(ctx context.Context, workflowID int, version string, tenantID int) error {
	ver, err := wvs.client.WorkflowVersion.Query().
		Where(workflowversion.WorkflowID(workflowID)).
		Where(workflowversion.Version(version)).
		Only(ctx)

	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	if ver.IsCurrent {
		return fmt.Errorf("cannot delete current version")
	}

	err = wvs.client.WorkflowVersion.DeleteOne(ver).Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}

	wvs.logAuditEvent(ctx, "workflow_version_deleted", ver.ID, tenantID, map[string]interface{}{
		"workflow_id": workflowID,
		"version":     version,
	})

	return nil
}

func (wvs *WorkflowVersionService) logAuditEvent(ctx context.Context, eventType string, resourceID int, tenantID int, details map[string]interface{}) {
	wvs.logger.Infow("Audit event", "event_type", eventType, "resource_id", resourceID, "tenant_id", tenantID, "details", details)
}
