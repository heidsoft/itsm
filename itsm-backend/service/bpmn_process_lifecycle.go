package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
)

func (e *CustomProcessEngine) SuspendProcess(ctx context.Context, processInstanceID string, reason string) error {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}

	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID), processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取流程实例失败: %w", err)
	}

	_, err = e.client.ProcessInstance.UpdateOne(instance).
		SetStatus("suspended").
		SetSuspendedTime(time.Now()).
		SetSuspendedReason(reason).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("暂停流程实例失败: %w", err)
	}

	userID, userName := extractUserFromContext(ctx)
	if err := e.auditService.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           instance.CurrentActivityID,
		ActivityName:         instance.CurrentActivityName,
		ActivityType:         ActivityTypeUserTask,
		Action:               AuditActionProcessSuspended,
		UserID:               userID,
		UserName:             userName,
		Comment:              reason,
		TenantID:             instance.TenantID,
	}); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}

	return nil
}

func (e *CustomProcessEngine) ResumeProcess(ctx context.Context, processInstanceID string) error {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}

	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID), processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取流程实例失败: %w", err)
	}

	_, err = e.client.ProcessInstance.UpdateOne(instance).
		SetStatus("running").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("恢复流程实例失败: %w", err)
	}

	userID, userName := extractUserFromContext(ctx)
	if err := e.auditService.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           instance.CurrentActivityID,
		ActivityName:         instance.CurrentActivityName,
		ActivityType:         ActivityTypeUserTask,
		Action:               AuditActionProcessResumed,
		UserID:               userID,
		UserName:             userName,
		TenantID:             instance.TenantID,
	}); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}

	return nil
}

func (e *CustomProcessEngine) TerminateProcess(ctx context.Context, processInstanceID string, reason string) error {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}

	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID), processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取流程实例失败: %w", err)
	}

	_, err = e.client.ProcessInstance.UpdateOne(instance).
		SetStatus("terminated").
		SetEndTime(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("终止流程实例失败: %w", err)
	}

	_, err = e.client.ProcessTask.Update().
		Where(processtask.ProcessInstanceID(instance.ID)).
		Where(processtask.StatusNEQ("completed")).
		Where(processtask.StatusNEQ("cancelled")).
		SetStatus("cancelled").
		SetCompletedTime(time.Now()).
		Save(ctx)
	if err != nil {
		e.logger.Warnw("取消流程任务失败", "error", err)
	}

	userID, userName := extractUserFromContext(ctx)
	if err := e.auditService.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           instance.CurrentActivityID,
		ActivityName:         instance.CurrentActivityName,
		ActivityType:         ActivityTypeEndEvent,
		Action:               AuditActionProcessTerminated,
		UserID:               userID,
		UserName:             userName,
		Comment:              reason,
		TenantID:             instance.TenantID,
	}); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}

	return nil
}

func extractUserFromContext(ctx context.Context) (int, string) {
	if u, ok := ctx.Value("user").(*ent.User); ok {
		return u.ID, u.Name
	}
	return 0, ""
}
