package mcp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/dto"
	"itsm-backend/handlers/mcp"
)

// 这组测试锁「写工具经审批流时的协议回执语义」。
//
// 缺陷背景（2026-11-21 审计发现）：handlers/mcp handleToolsCall 此前对
// `ExecuteTool` 返回的 (res=nil, invocationID>0, err=nil) 只把 invocationId
// 塞进 _meta，正文是 formatResult(nil) 的空串，且 isError=false。
// 客户端（Claude Desktop / 第三方 Agent）拿到「空内容 + 无错误」只能判定
// 「调用成功、无返回」，于是会宣告操作已完成 —— 而真实副作用尚未发生。
//
// 判据（三条缺一不可）：
//  1. 正文必须显式声明 pending_approval，而不是空串；
//  2. 不得把 pending 当成功结果填进 structuredContent（工具 outputSchema
//     描述真实产出，pending 不是产出，填充会让客户端按 schema 校验失败
//     或被误读成工具结果）；
//  3. 不用 isError=true（受理不是错误，标错误会触发客户端重试/告警）。

// pendingExecutor 模拟执行链在写工具受理时的返回：
// 无结果、有 invocationID、无错误。
type pendingExecutor struct {
	called       bool
	invocationID int
	toolName     string
}

func (p *pendingExecutor) ExecuteTool(_ context.Context, _, _ int, _, name string, _ map[string]interface{}) (interface{}, int, error) {
	p.called = true
	p.toolName = name
	return nil, p.invocationID, nil
}

// executedExecutor 模拟只读/已执行工具：有结果、无 invocationID。
type executedExecutor struct {
	result interface{}
}

func (e *executedExecutor) ExecuteTool(_ context.Context, _, _ int, _, _ string, _ map[string]interface{}) (interface{}, int, error) {
	return e.result, 0, nil
}

func callTool(t *testing.T, svc *mcp.Service, toolName string, caller mcp.Caller) mcp.CallToolResult {
	t.Helper()
	params, err := json.Marshal(mcp.CallToolParams{Name: toolName, Arguments: map[string]interface{}{}})
	require.NoError(t, err)
	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0", ID: json.RawMessage("9"), Method: mcp.MethodToolsCall, Params: params,
	}, caller)
	require.NotNil(t, resp)
	require.Nil(t, resp.Error, "受理/执行都属业务结果，不是协议错误")
	result, ok := resp.Result.(mcp.CallToolResult)
	require.True(t, ok, "tools/call result 类型应为 CallToolResult")
	return result
}

func TestToolCall_WriteToolPendingApprovalIsExplicit(t *testing.T) {
	exec := &pendingExecutor{invocationID: 42}
	cat := &fakeCatalog{tools: []dto.ToolDefinition{writeTool("create_ticket", "ticket")}}
	svc, err := mcp.NewService(mcp.Options{Catalog: cat, Executor: exec})
	require.NoError(t, err)

	result := callTool(t, svc, "create_ticket", mcp.Caller{TenantID: 1, UserID: 7, Role: "agent", Principal: "agent"})

	require.True(t, exec.called, "受理必须经过执行链，MCP 层不得自行判定")
	require.False(t, result.IsError, "受理不是错误：标成 isError 会触发客户端重试/告警")

	require.Len(t, result.Content, 1)
	text := result.Content[0].Text
	require.NotEmpty(t, text, "正文不得为空（空正文会让客户端把受理读成「成功但无返回」）")
	assert.Contains(t, text, "pending_approval", "必须显式声明待审批状态")
	assert.Contains(t, text, "invocation_id=42", "必须回传 invocation id 供轮询")
	assert.Contains(t, text, "尚未执行", "必须写明副作用尚未发生")
	assert.Contains(t, text, "请勿声称操作已完成", "必须显式约束客户端不要宣告完成")

	require.NotNil(t, result.Meta)
	assert.Equal(t, 42, result.Meta["invocationId"])
	assert.Equal(t, "pending_approval", result.Meta["status"])
	assert.Contains(t, result.Meta["pollHint"], "42", "pollHint 应指向该 invocation 的查询路径")
}

// pending 不是工具产出：不得填 structuredContent，否则客户端按 outputSchema 校验
// 会失败，或被误读成工具真实结果。
func TestToolCall_PendingApprovalOmitsStructuredContent(t *testing.T) {
	exec := &pendingExecutor{invocationID: 7}
	cat := &fakeCatalog{tools: []dto.ToolDefinition{writeTool("update_ticket", "ticket")}}
	svc, err := mcp.NewService(mcp.Options{Catalog: cat, Executor: exec})
	require.NoError(t, err)

	result := callTool(t, svc, "update_ticket", mcp.Caller{TenantID: 1, UserID: 7, Role: "agent", Principal: "agent"})
	assert.Nil(t, result.StructuredContent,
		"待审批回执不得填充 structuredContent（pending 不是工具产出）")
}

// 已执行/只读路径不能被 pending 分支误伤：有结果时必须照常返回内容与结构化结果。
func TestToolCall_ExecutedResultStillReturnsContentAndStructured(t *testing.T) {
	exec := &executedExecutor{result: map[string]interface{}{"total": 3}}
	cat := &fakeCatalog{tools: []dto.ToolDefinition{readOnlyTool("get_incident_stats", "incident")}}
	svc, err := mcp.NewService(mcp.Options{Catalog: cat, Executor: exec})
	require.NoError(t, err)

	result := callTool(t, svc, "get_incident_stats", mcp.Caller{TenantID: 1, UserID: 7, Role: "agent", Principal: "human"})

	require.False(t, result.IsError)
	text := result.Content[0].Text
	require.NotEmpty(t, text)
	assert.False(t, strings.Contains(text, "pending_approval"), "已执行路径不得出现待审批文案")
	require.NotNil(t, result.StructuredContent, "有结果时应返回 structuredContent")
	assert.Nil(t, result.Meta, "只读/已执行且无 invocation 时不回传 invocation meta")
}
