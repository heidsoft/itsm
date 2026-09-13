package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processexecutionhistory"
	"itsm-backend/ent/processinstance"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// bpmn_process_instance_service.go — 流程实例查询与变量管理
//
// 职责：流程实例的查询（单个/列表）、变量读写、执行历史、统计。
// 不包含实例的生命周期流转（启动/暂停/恢复/终止在 bpmn_process_engine.go）。
//
// 社区贡献者只需理解此文件即可掌握实例的读取面。
// ---------------------------------------------------------------------------

// Request/Response DTOs

type ListProcessInstancesRequest struct {
	ProcessDefinitionKey string `json:"processDefinitionKey"`
	Status               string `json:"status"`
	BusinessKey          string `json:"businessKey"`
	TenantID             int    `json:"tenantId"`
	Page                 int    `json:"page"`
	PageSize             int    `json:"pageSize"`
}

// InstanceStatisticsRequest 实例统计请求
type InstanceStatisticsRequest struct {
	ProcessDefinitionKey string     `json:"processDefinitionKey"`
	Status               string     `json:"status"`
	TenantID             int        `json:"tenantId"`
	StartDate            *time.Time `json:"startDate"`
	EndDate              *time.Time `json:"endDate"`
}

// InstanceStatistics 实例统计
type InstanceStatistics struct {
	Total      int `json:"total"`
	Running    int `json:"running"`
	Completed  int `json:"completed"`
	Suspended  int `json:"suspended"`
	Terminated int `json:"terminated"`
}

// Service struct

type bpmnProcessInstanceService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// Queries

func (s *bpmnProcessInstanceService) GetProcessInstance(ctx context.Context, processInstanceID string) (*ent.ProcessInstance, error) {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := strconv.Atoi(processInstanceID)
	if err != nil {
		return nil, fmt.Errorf("无效的流程实例ID: %w", err)
	}
	instance, err := s.client.ProcessInstance.Query().
		Where(processinstance.ID(id), processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程实例失败: %w", err)
	}

	return instance, nil
}

func (s *bpmnProcessInstanceService) ListProcessInstances(ctx context.Context, req *ListProcessInstancesRequest) ([]*ent.ProcessInstance, int, error) {
	ctxTenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	tenantID := ctxTenantID
	if req.TenantID > 0 && req.TenantID != ctxTenantID {
		return nil, 0, fmt.Errorf("请求租户 %d 与上下文租户 %d 不一致，已拒绝", req.TenantID, ctxTenantID)
	}

	query := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID))

	if req.ProcessDefinitionKey != "" {
		query = query.Where(processinstance.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.Status != "" {
		query = query.Where(processinstance.Status(req.Status))
	}
	if req.BusinessKey != "" {
		query = query.Where(processinstance.BusinessKey(req.BusinessKey))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程实例总数失败: %w", err)
	}

	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	instances, err := query.Order(ent.Desc(processinstance.FieldStartTime)).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程实例列表失败: %w", err)
	}

	return instances, total, nil
}

// Variables

func (s *bpmnProcessInstanceService) GetProcessInstanceVariables(ctx context.Context, processInstanceID string) (map[string]interface{}, error) {
	instance, err := s.GetProcessInstance(ctx, processInstanceID)
	if err != nil {
		return nil, err
	}

	return instance.Variables, nil
}

func (s *bpmnProcessInstanceService) SetProcessInstanceVariables(ctx context.Context, processInstanceID string, variables map[string]interface{}) error {
	instance, err := s.GetProcessInstance(ctx, processInstanceID)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessInstance.UpdateOne(instance).
		SetVariables(variables).
		Save(ctx)

	return err
}

// History

func (s *bpmnProcessInstanceService) GetProcessInstanceHistory(ctx context.Context, processInstanceID string) ([]*ent.ProcessExecutionHistory, error) {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := strconv.Atoi(processInstanceID)
	if err != nil {
		return nil, fmt.Errorf("无效的流程实例ID: %w", err)
	}

	query := s.client.ProcessExecutionHistory.Query().
		Where(
			processexecutionhistory.ProcessInstanceID(id),
			processexecutionhistory.TenantID(tenantID),
		)

	history, err := query.Order(ent.Asc(processexecutionhistory.FieldTimestamp)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程实例历史失败: %w", err)
	}

	return history, nil
}

// Statistics

// GetInstanceStatistics 获取实例统计
func (s *bpmnProcessInstanceService) GetInstanceStatistics(ctx context.Context, req *InstanceStatisticsRequest) (*InstanceStatistics, error) {
	ctxTenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	tenantID := ctxTenantID
	if req.TenantID > 0 && req.TenantID != ctxTenantID {
		return nil, fmt.Errorf("请求租户 %d 与上下文租户 %d 不一致，已拒绝", req.TenantID, ctxTenantID)
	}

	query := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID))

	if req.ProcessDefinitionKey != "" {
		query = query.Where(processinstance.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.StartDate != nil {
		query = query.Where(processinstance.StartTimeGTE(*req.StartDate))
	}
	if req.EndDate != nil {
		query = query.Where(processinstance.StartTimeLTE(*req.EndDate))
	}

	instances, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取实例统计失败: %w", err)
	}

	stats := &InstanceStatistics{
		Total:      len(instances),
		Running:    0,
		Completed:  0,
		Suspended:  0,
		Terminated: 0,
	}

	for _, inst := range instances {
		switch inst.Status {
		case "running":
			stats.Running++
		case "completed":
			stats.Completed++
		case "suspended":
			stats.Suspended++
		case "terminated":
			stats.Terminated++
		}
	}

	if req.Status != "" {
		stats.Total = 0
		switch req.Status {
		case "running":
			stats.Total = stats.Running
		case "completed":
			stats.Total = stats.Completed
		case "suspended":
			stats.Total = stats.Suspended
		case "terminated":
			stats.Total = stats.Terminated
		}
	}

	return stats, nil
}
