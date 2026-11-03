package service_request

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
)

const (
	staleSRProcessKey        = "service_request_flow"
	staleSRBusinessKeyPrefix = "service_request:"
	defaultStaleSRReason     = "服务请求流程模板已升级，旧版流程实例不再兼容，请重新提交服务请求"

	staleActionCancel            = "cancel"
	staleActionTerminateWorkflow = "terminate_workflow"
	staleActionSkip              = "skip"
)

// SystemNotifier 是页面内系统通知的最小端口，便于运维命令与测试替换实现。
type SystemNotifier interface {
	CreateSystemNotification(ctx context.Context, title, message, notificationType string, userIDs []int, tenantID int) error
}

// SetSystemNotifier 注入系统通知能力。未注入时只终止与留痕，不发送通知。
func (s *Service) SetSystemNotifier(n SystemNotifier) { s.notifier = n }

// StaleInstanceResult 描述一个旧版流程实例的处置结果。
// Applied=false 表示 dry-run，或该实例按规则未做任何写入。
type StaleInstanceResult struct {
	RequestID         int    `json:"requestId"`
	RequestTitle      string `json:"requestTitle"`
	RequesterID       int    `json:"requesterId"`
	PreviousStatus    string `json:"previousStatus"`
	ProcessInstanceID string `json:"processInstanceId"`
	DefinitionVersion string `json:"definitionVersion"`
	LatestVersion     string `json:"latestVersion"`
	Action            string `json:"action"`
	Applied           bool   `json:"applied"`
	Reason            string `json:"reason"`
}

// TerminateStaleServiceRequestInstances 统一收口绑定旧版内置流程的在途服务请求：
// 终止流程实例、作废待审批步骤、将请求置为 cancelled，并通知发起人重新提交。
//
// 只处理 business_key=service_request:<id> 且 definition_key=service_request_flow 的
// running/suspended 实例，并且仅当实例绑定的定义不是该 key 的最新版本。交付中
// （provisioning）与已交付/已驳回的请求不改状态，只在结果里标记 skip 供运维判断。
// dryRun 只列出将要处置的对象，不写业务、流程、审计或通知。
func (s *Service) TerminateStaleServiceRequestInstances(ctx context.Context, tenantID, actorID int, reason string, dryRun bool) ([]StaleInstanceResult, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("tenant id is required")
	}
	if s.client == nil {
		return nil, fmt.Errorf("ent client is required")
	}
	if strings.TrimSpace(reason) == "" {
		reason = defaultStaleSRReason
	}

	latestByKey, err := s.latestDefinitionsByKey(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	instances, err := s.client.ProcessInstance.Query().
		Where(
			processinstance.TenantID(tenantID),
			processinstance.ProcessDefinitionKeyEQ(staleSRProcessKey),
			processinstance.BusinessKeyHasPrefix(staleSRBusinessKeyPrefix),
			processinstance.StatusIn("running", "suspended"),
		).
		Order(ent.Asc(processinstance.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询在途服务请求流程实例失败: %w", err)
	}

	results := make([]StaleInstanceResult, 0, len(instances))
	for _, instance := range instances {
		latest := latestByKey[instance.ProcessDefinitionKey]
		if latest == nil || latest.ID == instance.ProcessDefinitionID {
			continue
		}

		requestID, ok := parseServiceRequestBusinessKey(instance.BusinessKey)
		if !ok {
			continue
		}

		result := StaleInstanceResult{
			RequestID:         requestID,
			ProcessInstanceID: instance.ProcessInstanceID,
			LatestVersion:     latest.Version,
			Reason:            reason,
		}
		if definition, err := s.client.ProcessDefinition.Query().
			Where(processdefinition.ID(instance.ProcessDefinitionID), processdefinition.TenantID(tenantID)).
			Only(ctx); err == nil {
			result.DefinitionVersion = definition.Version
		}

		req, err := s.repo.Get(ctx, requestID, tenantID)
		if err != nil {
			result.Action = staleActionSkip
			result.Reason = "服务请求不存在或已删除"
			results = append(results, result)
			continue
		}
		result.RequestTitle = req.Title
		result.RequesterID = req.RequesterID
		result.PreviousStatus = req.Status

		switch req.Status {
		case SRStatusProvisioning:
			result.Action = staleActionSkip
			result.Reason = "请求正在交付中，需人工确认交付结果后再处理"
			results = append(results, result)
			continue
		case SRStatusDelivered, SRStatusRejected:
			result.Action = staleActionSkip
			result.Reason = "请求已进入终态，无需终止"
			results = append(results, result)
			continue
		}

		if req.Status == SRStatusCancelled {
			result.Action = staleActionTerminateWorkflow
			if dryRun {
				result.Reason = reason + "（仅终止残留流程实例）"
				results = append(results, result)
				continue
			}
			if err := s.approvalBridge.TerminateBusinessWorkflow(
				ctx, tenantID, actorID, string(dto.BusinessTypeServiceRequest), requestID, reason,
			); err != nil {
				return results, fmt.Errorf("终止服务请求 %d 的残留流程实例失败: %w", requestID, err)
			}
			result.Applied = true
			s.recordStaleTerminationAudit(ctx, tenantID, actorID, &result)
			results = append(results, result)
			continue
		}

		if !isValidServiceRequestOperationalTransition(req.Status, SRStatusCancelled) {
			result.Action = staleActionSkip
			result.Reason = "当前状态不允许取消：" + req.Status
			results = append(results, result)
			continue
		}

		result.Action = staleActionCancel
		if dryRun {
			results = append(results, result)
			continue
		}

		if err := s.repo.UpdateStatus(ctx, req, SRStatusCancelled, actorID); err != nil {
			if ent.IsNotFound(err) {
				result.Action = staleActionSkip
				result.Reason = "请求已被并发修改，请重新执行"
				results = append(results, result)
				continue
			}
			return results, fmt.Errorf("取消服务请求 %d 失败: %w", requestID, err)
		}
		s.cancelApprovalFollowUps(ctx, tenantID, actorID, requestID)
		s.recordStaleTerminationAudit(ctx, tenantID, actorID, &result)
		s.notifyStaleRebuild(ctx, tenantID, req, reason)
		result.Applied = true
		results = append(results, result)
	}

	return results, nil
}

func (s *Service) latestDefinitionsByKey(ctx context.Context, tenantID int) (map[string]*ent.ProcessDefinition, error) {
	definitions, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.TenantID(tenantID),
			processdefinition.KeyEQ(staleSRProcessKey),
			processdefinition.IsLatest(true),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询最新流程定义失败: %w", err)
	}
	latest := make(map[string]*ent.ProcessDefinition, len(definitions))
	for _, definition := range definitions {
		latest[definition.Key] = definition
	}
	return latest, nil
}

func parseServiceRequestBusinessKey(businessKey string) (int, bool) {
	if !strings.HasPrefix(businessKey, staleSRBusinessKeyPrefix) {
		return 0, false
	}
	id, err := strconv.Atoi(strings.TrimPrefix(businessKey, staleSRBusinessKeyPrefix))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func (s *Service) recordStaleTerminationAudit(ctx context.Context, tenantID, actorID int, result *StaleInstanceResult) {
	body, err := json.Marshal(map[string]interface{}{
		"reason":            result.Reason,
		"processInstanceId": result.ProcessInstanceID,
		"definitionVersion": result.DefinitionVersion,
		"latestVersion":     result.LatestVersion,
		"previousStatus":    result.PreviousStatus,
		"requestId":         result.RequestID,
	})
	if err != nil {
		s.logger.Warnw("序列化旧版流程实例终止审计失败", "request_id", result.RequestID, "error", err)
		return
	}
	if _, err := s.client.AuditLog.Create().
		SetTenantID(tenantID).
		SetUserID(actorID).
		SetResource("service_request").
		SetAction("cancel_stale_process_instance").
		SetMethod("SYSTEM").
		SetPath(fmt.Sprintf("/api/v1/service-requests/%d", result.RequestID)).
		SetRequestBody(string(body)).
		Save(ctx); err != nil {
		s.logger.Warnw("写入旧版流程实例终止审计失败", "request_id", result.RequestID, "error", err)
	}
}

func (s *Service) notifyStaleRebuild(ctx context.Context, tenantID int, req *ServiceRequest, reason string) {
	if s.notifier == nil || req.RequesterID <= 0 {
		return
	}
	message := fmt.Sprintf("服务请求 #%d「%s」因流程模板升级已终止，请按新流程重新提交。原因：%s", req.ID, req.Title, reason)
	if err := s.notifier.CreateSystemNotification(
		ctx, "服务请求需要重新提交", message, "warning", []int{req.RequesterID}, tenantID,
	); err != nil {
		s.logger.Warnw("通知服务请求发起人重新提交失败", "request_id", req.ID, "tenant_id", tenantID, "error", err)
	}
}
