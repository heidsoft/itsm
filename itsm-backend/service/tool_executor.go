package service

import (
	"context"
	"encoding/json"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/toolinvocation"

	"go.uber.org/zap"
)

// ToolExecutor 执行审批通过的 AI 写工具。
//
// 执行入口只来自 operational_commands（ai.tool.execute）：进程内 channel 在缓冲满时
// 会静默丢弃任务，进程重启后已批准的调用也永远停在 pending。执行权用 status 条件
// 更新领取，保证同一 invocation 只执行一次——工具副作用大多不可幂等（建单、改状态），
// 重放必须由操作者显式发起，不能由重试自动完成。
type ToolExecutor struct {
	client *ent.Client
	tools  *ToolRegistry
	logger *zap.SugaredLogger
}

func NewToolExecutor(client *ent.Client, tools *ToolRegistry, logger *zap.SugaredLogger) *ToolExecutor {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &ToolExecutor{client: client, tools: tools, logger: logger}
}

// HandleCommand 是 ai.tool.execute 的 outbox handler；AggregateID 即 invocation id。
func (e *ToolExecutor) HandleCommand(ctx context.Context, cmd *ent.OperationalCommand) error {
	return e.ExecuteApprovedInvocation(ctx, cmd.AggregateID, cmd.TenantID)
}

func (e *ToolExecutor) ExecuteApprovedInvocation(ctx context.Context, invocationID, tenantID int) error {
	if e.client == nil || e.tools == nil {
		return fmt.Errorf("tool executor is not configured")
	}

	inv, err := e.client.ToolInvocation.Query().
		Where(toolinvocation.ID(invocationID), toolinvocation.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("load tool invocation %d: %w", invocationID, err)
	}
	if inv.ApprovalState != "approved" {
		return fmt.Errorf("tool invocation %d is not approved (state=%s)", invocationID, inv.ApprovalState)
	}
	if inv.Status == "done" {
		// 结果已回写：命令重试/重放到此为止。
		return nil
	}

	claimed, err := e.client.ToolInvocation.Update().
		Where(
			toolinvocation.ID(invocationID),
			toolinvocation.TenantIDEQ(tenantID),
			toolinvocation.ApprovalStateEQ("approved"),
			toolinvocation.StatusIn("pending", "failed"),
		).
		SetStatus("running").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("claim tool invocation %d: %w", invocationID, err)
	}
	if claimed == 0 {
		latest, err := e.client.ToolInvocation.Query().
			Where(toolinvocation.ID(invocationID), toolinvocation.TenantIDEQ(tenantID)).
			Only(ctx)
		if err != nil {
			return fmt.Errorf("reload tool invocation %d: %w", invocationID, err)
		}
		if latest.Status == "done" {
			return nil
		}
		// running 说明上一次执行可能已产生副作用但没来得及回写结果，自动重放会重复。
		return fmt.Errorf("tool invocation %d is not claimable (status=%s)", invocationID, latest.Status)
	}

	var args map[string]interface{}
	if inv.Arguments != "" {
		if err := json.Unmarshal([]byte(inv.Arguments), &args); err != nil {
			invalid := fmt.Errorf("tool invocation %d has invalid arguments: %w", invocationID, err)
			return e.finalize(ctx, invocationID, tenantID, nil, invalid)
		}
	}

	res, execErr := e.tools.Execute(ctx, tenantID, inv.ToolName, withInvocationUser(args, inv.UserID))
	return e.finalize(ctx, invocationID, tenantID, res, execErr)
}

// finalize 把工具执行结果写回 ToolInvocation（成功/失败两态）。
func (e *ToolExecutor) finalize(ctx context.Context, invocationID, tenantID int, res interface{}, execErr error) error {
	status := "done"
	var result, errText string
	if execErr != nil {
		status = "failed"
		errText = execErr.Error()
	} else {
		out, marshalErr := json.Marshal(res)
		if marshalErr != nil {
			status = "failed"
			execErr = fmt.Errorf("marshal tool invocation %d result: %w", invocationID, marshalErr)
			errText = execErr.Error()
		} else {
			result = string(out)
		}
	}

	update := e.client.ToolInvocation.Update().
		Where(toolinvocation.ID(invocationID), toolinvocation.TenantIDEQ(tenantID)).
		SetStatus(status)
	if status == "failed" {
		update.SetError(errText)
	} else {
		update.SetResult(result)
	}
	affected, err := update.Save(ctx)
	if err != nil {
		e.logger.Errorw("Failed to persist tool invocation result",
			"invocation_id", invocationID, "tenant_id", tenantID, "error", err)
		return fmt.Errorf("persist tool invocation %d result: %w", invocationID, err)
	}
	if affected == 0 {
		// 结果没落库就不能把命令标成功，否则审计里只剩一次「执行过」。
		e.logger.Errorw("Tool invocation result not persisted: row missing or tenant mismatch",
			"invocation_id", invocationID, "tenant_id", tenantID)
		return fmt.Errorf("tool invocation %d result not persisted: row missing or tenant mismatch", invocationID)
	}
	return execErr
}

// withInvocationUser 把发起调用/审批的用户ID回填进参数，让 ToolRegistry 能正确归属
// requesterId（LLM 生成的参数里通常不含用户ID）。键名必须与工具 schema 一致。
func withInvocationUser(args map[string]interface{}, userID int) map[string]interface{} {
	if userID <= 0 {
		return args
	}
	merged := make(map[string]interface{}, len(args)+1)
	for k, v := range args {
		merged[k] = v
	}
	if _, ok := merged["requesterId"]; !ok {
		merged["requesterId"] = float64(userID)
	}
	return merged
}
