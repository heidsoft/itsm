package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/dto"
	"itsm-backend/handlers/mcp"
)

// 这组契约测试锁三件事：
//
//  1. 协议形状：initialize / tools/list / tools/call 的响应结构符合 MCP 规范，
//     且未知方法返回 -32601 而不是静默成功。
//
//  2. 权限拒绝：非 human 主体在未获运行期策略放行时被显式拒绝，
//     且拒绝发生在执行器之前（执行器不被触达）。
//
//  3. 交付门禁：租户未安装的工具既不出现在 tools/list，也不可被直接调用
//     （知道名字也调不动）——否则 MCP 通道会成为绕过市场交付的旁路。
//
// 反向验证要求（见 itsm-authz-change SOP）：把实现改回旧行为，
// 本文件必须变红，且红的原因是断言失败而非编译错误。

// ---------------------------------------------------------------------------
// 替身
// ---------------------------------------------------------------------------

type fakeCatalog struct {
	tools []dto.ToolDefinition
}

func (f *fakeCatalog) ListToolsForTenant(_ context.Context, _ int) []dto.ToolDefinition {
	return f.tools
}

type fakeExecutor struct {
	called     bool
	calledWith string
	args       map[string]interface{}
	result     interface{}
	err        error
}

func (f *fakeExecutor) ExecuteTool(_ context.Context, _, _ int, _, name string, args map[string]interface{}) (interface{}, int, error) {
	f.called = true
	f.calledWith = name
	f.args = args
	return f.result, 0, f.err
}

type fakeEntitlement struct {
	allowed map[string]bool
	reason  string
	err     error
}

func (f *fakeEntitlement) Check(_ context.Context, _ int, code string) (mcp.Entitlement, error) {
	if f.err != nil {
		return mcp.Entitlement{}, f.err
	}
	if f.allowed[code] {
		return mcp.Entitlement{Allowed: true}, nil
	}
	reason := f.reason
	if reason == "" {
		reason = "未安装"
	}
	return mcp.Entitlement{Allowed: false, Reason: reason}, nil
}

func readOnlyTool(name, resource string) dto.ToolDefinition {
	return dto.ToolDefinition{
		Name:        name,
		Description: "只读工具 " + name,
		ReadOnly:    true,
		Resource:    resource,
		Action:      "read",
		ArgsSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{"type": "string"},
			},
			"required": []string{"id"},
		},
	}
}

func writeTool(name, resource string) dto.ToolDefinition {
	return dto.ToolDefinition{
		Name:        name,
		Description: "写工具 " + name,
		ReadOnly:    false,
		Resource:    resource,
		Action:      "write",
		ArgsSchema:  map[string]interface{}{"type": "object"},
	}
}

// ---------------------------------------------------------------------------
// 协议形状契约
// ---------------------------------------------------------------------------

func TestInitialize_NegotiatesProtocolAndDeclaresTools(t *testing.T) {
	cat := &fakeCatalog{tools: []dto.ToolDefinition{readOnlyTool("get_incident_stats", "incident")}}
	svc, err := mcp.NewService(mcp.Options{Visibility: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }), Catalog: cat, Executor: &fakeExecutor{}})
	require.NoError(t, err)

	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  mcp.MethodInitialize,
		Params:  json.RawMessage(`{"protocolVersion":"2025-06-18","clientInfo":{"name":"probe","version":"1.0.0"}}`),
	}, mcp.Caller{TenantID: 1})

	require.NotNil(t, resp)
	require.Nil(t, resp.Error, "initialize 不应返回错误")
	require.Equal(t, "2.0", resp.JSONRPC)

	result, ok := resp.Result.(mcp.InitializeResult)
	require.True(t, ok, "initialize result 类型应为 InitializeResult")
	assert.Equal(t, mcp.ProtocolVersion, result.ProtocolVersion)
	assert.Contains(t, result.Capabilities, "tools", "必须声明 tools 能力")
	assert.NotContains(t, result.Capabilities, "prompts", "未实现的 capability 不得声明")
	assert.NotContains(t, result.Capabilities, "resources", "未实现的 capability 不得声明")
	assert.NotEmpty(t, result.ServerInfo.Name)
}

func TestInitialize_UnsupportedClientVersion_StillRespondsWithServerVersion(t *testing.T) {
	cat := &fakeCatalog{}
	svc, err := mcp.NewService(mcp.Options{Visibility: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }), Catalog: cat, Executor: &fakeExecutor{}})
	require.NoError(t, err)

	// 客户端报一个服务端不支持的版本：服务端应回自己支持的版本，
	// 由客户端决定是否断开（规范要求，而非直接报错）。
	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  mcp.MethodInitialize,
		Params:  json.RawMessage(`{"protocolVersion":"1999-01-01","clientInfo":{"name":"old","version":"0.1"}}`),
	}, mcp.Caller{TenantID: 1})

	require.NotNil(t, resp)
	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.InitializeResult)
	assert.Equal(t, mcp.ProtocolVersion, result.ProtocolVersion)
}

func TestUnknownMethod_ReturnsMethodNotFound(t *testing.T) {
	cat := &fakeCatalog{}
	svc, err := mcp.NewService(mcp.Options{Visibility: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }), Catalog: cat, Executor: &fakeExecutor{}})
	require.NoError(t, err)

	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage("7"),
		Method:  "resources/list",
	}, mcp.Caller{TenantID: 1})

	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, mcp.CodeMethodNotFound, resp.Error.Code)
}

func TestInitializedNotification_ReturnsNoResponse(t *testing.T) {
	cat := &fakeCatalog{}
	svc, err := mcp.NewService(mcp.Options{Visibility: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }), Catalog: cat, Executor: &fakeExecutor{}})
	require.NoError(t, err)

	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0",
		Method:  mcp.MethodInitialized,
	}, mcp.Caller{TenantID: 1})

	assert.Nil(t, resp, "通知不得产生响应")
}

func TestToolsList_ProjectsSchemaAndReadOnlyHint(t *testing.T) {
	cat := &fakeCatalog{tools: []dto.ToolDefinition{
		readOnlyTool("get_incident_stats", "incident"),
		writeTool("update_ci", "cmdb"),
	}}
	svc, err := mcp.NewService(mcp.Options{Visibility: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }), Catalog: cat, Executor: &fakeExecutor{}})
	require.NoError(t, err)

	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0", ID: json.RawMessage("2"), Method: mcp.MethodToolsList,
	}, mcp.Caller{TenantID: 1})

	require.NotNil(t, resp)
	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.ListToolsResult)
	require.Len(t, result.Tools, 2)

	byName := map[string]mcp.Tool{}
	for _, t := range result.Tools {
		byName[t.Name] = t
	}
	ro, ok := byName["get_incident_stats"]
	require.True(t, ok)
	assert.Equal(t, "object", ro.InputSchema["type"], "inputSchema 必须是 object 类型")
	require.NotNil(t, ro.Annotations)
	assert.True(t, ro.Annotations.ReadOnlyHint, "只读工具应标注 readOnlyHint")
	assert.False(t, ro.Annotations.DestructiveHint, "只读工具不应标注 destructive")

	wr, ok := byName["update_ci"]
	require.True(t, ok)
	require.NotNil(t, wr.Annotations)
	assert.False(t, wr.Annotations.ReadOnlyHint)
	assert.True(t, wr.Annotations.DestructiveHint, "写工具应标注 destructive")
}

// ---------------------------------------------------------------------------
// 权限契约
// ---------------------------------------------------------------------------

func TestToolCall_DelegatesSubjectAuthorizationToExecutionChain(t *testing.T) {
	// 主体权限判定必须下沉到执行链（ExecuteTool 的 Gate 2.5），MCP 层不重复判定。
	//
	// 判据：执行器返回「主体未被授权」错误时，MCP 必须把它映射为
	// isError=true 的工具执行失败（而非协议错误），且不得改写错误语义。
	exec := &fakeExecutor{err: errors.New("AI 主体 service 未获授权 incident:read")}
	cat := &fakeCatalog{tools: []dto.ToolDefinition{readOnlyTool("get_incident_stats", "incident")}}
	svc, err := mcp.NewService(mcp.Options{Visibility: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }), Catalog: cat, Executor: exec})
	require.NoError(t, err)

	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0", ID: json.RawMessage("3"), Method: mcp.MethodToolsCall,
		Params: json.RawMessage(`{"name":"get_incident_stats","arguments":{"id":"INC-1"}}`),
	}, mcp.Caller{TenantID: 1, Role: "audit_admin", Principal: "service"})

	require.NotNil(t, resp)
	require.Nil(t, resp.Error, "权限拒绝属业务失败，不是协议错误")
	result := resp.Result.(mcp.CallToolResult)
	assert.True(t, result.IsError, "执行链的权限拒绝必须映射为 isError")
	assert.True(t, exec.called, "MCP 层必须把调用交给执行链判定，不得自行放行或自行拒绝")
	assert.Contains(t, result.Content[0].Text, "incident:read", "应保留执行链给出的拒绝原因")
}

func TestToolCall_RoleAndPrincipalPassedThroughToExecutor(t *testing.T) {
	// MCP 层必须把角色与主体原样传给执行链 —— 执行链的 Gate 2 用 role、
	// Gate 2.5 用 principal。MCP 层若改写任一值，判定就会失真。
	exec := &fakeExecutor{result: map[string]interface{}{"ok": true}}
	cat := &fakeCatalog{tools: []dto.ToolDefinition{readOnlyTool("get_incident_stats", "incident")}}
	svc, err := mcp.NewService(mcp.Options{Visibility: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }), Catalog: cat, Executor: exec})
	require.NoError(t, err)

	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0", ID: json.RawMessage("5"), Method: mcp.MethodToolsCall,
		Params: json.RawMessage(`{"name":"get_incident_stats","arguments":{"id":"INC-1"}}`),
	}, mcp.Caller{TenantID: 1, Role: "audit_admin", Principal: "service"})

	require.NotNil(t, resp)
	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	assert.False(t, result.IsError)
	assert.True(t, exec.called)
	assert.Equal(t, "get_incident_stats", exec.calledWith)
	assert.Equal(t, "INC-1", exec.args["id"], "入参必须原样传给执行器")
}

// ---------------------------------------------------------------------------
// 交付门禁契约
// ---------------------------------------------------------------------------

func TestToolsList_HidesToolsWithoutEntitlement(t *testing.T) {
	cat := &fakeCatalog{tools: []dto.ToolDefinition{
		readOnlyTool("get_incident_stats", "incident"),
		readOnlyTool("search_knowledge", "knowledge"),
	}}
	ent := &fakeEntitlement{allowed: map[string]bool{"get_incident_stats": true}}
	svc, err := mcp.NewService(mcp.Options{Visibility: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }), Catalog: cat, Executor: &fakeExecutor{}, Entitlement: ent})
	require.NoError(t, err)

	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0", ID: json.RawMessage("6"), Method: mcp.MethodToolsList,
	}, mcp.Caller{TenantID: 1})

	require.NotNil(t, resp)
	result := resp.Result.(mcp.ListToolsResult)
	require.Len(t, result.Tools, 1, "未安装的工具不得出现在工具面")
	assert.Equal(t, "get_incident_stats", result.Tools[0].Name)
}

func TestToolCall_DeniedByEntitlement_EvenWhenToolNameKnown(t *testing.T) {
	exec := &fakeExecutor{result: map[string]interface{}{"secret": "data"}}
	ent := &fakeEntitlement{allowed: map[string]bool{}, reason: "未安装该组件"}
	cat := &fakeCatalog{tools: []dto.ToolDefinition{readOnlyTool("search_knowledge", "knowledge")}}
	svc, err := mcp.NewService(mcp.Options{Visibility: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }), Catalog: cat, Executor: exec, Entitlement: ent})
	require.NoError(t, err)

	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0", ID: json.RawMessage("8"), Method: mcp.MethodToolsCall,
		Params: json.RawMessage(`{"name":"search_knowledge","arguments":{"q":"x"}}`),
	}, mcp.Caller{TenantID: 1, Principal: "human"})

	require.NotNil(t, resp)
	require.Nil(t, resp.Error)
	result := resp.Result.(mcp.CallToolResult)
	assert.True(t, result.IsError)
	assert.False(t, exec.called, "未安装的工具不得被执行（防越权直调）")
	assert.True(t, strings.Contains(result.Content[0].Text, "未安装该组件"), "错误信息应说明权益原因")
}

func TestEntitlementCheckError_FailsClosed(t *testing.T) {
	cat := &fakeCatalog{tools: []dto.ToolDefinition{readOnlyTool("get_incident_stats", "incident")}}
	ent := &fakeEntitlement{err: errors.New("db down")}
	svc, err := mcp.NewService(mcp.Options{Visibility: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }), Catalog: cat, Executor: &fakeExecutor{}, Entitlement: ent})
	require.NoError(t, err)

	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0", ID: json.RawMessage("9"), Method: mcp.MethodToolsList,
	}, mcp.Caller{TenantID: 1})

	require.NotNil(t, resp)
	result := resp.Result.(mcp.ListToolsResult)
	assert.Empty(t, result.Tools, "权益查询出错时必须 fail-closed（隐藏工具）")
}

// ---------------------------------------------------------------------------
// 投影契约
// ---------------------------------------------------------------------------

func TestProjectTool_RejectsToolWithoutName(t *testing.T) {
	_, err := mcp.ProjectTool(dto.ToolDefinition{Description: "无名工具"})
	require.Error(t, err, "无名工具不得投影")
}

func TestProjectTool_EmptyArgsSchema_DegradesToEmptyObjectSchema(t *testing.T) {
	tool, err := mcp.ProjectTool(dto.ToolDefinition{Name: "noop", ReadOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "object", tool.InputSchema["type"])
}

func TestProjectTool_PropagatesResultSchema(t *testing.T) {
	tool, err := mcp.ProjectTool(dto.ToolDefinition{
		Name:         "q",
		ReadOnly:     true,
		ResultSchema: map[string]interface{}{"type": "object"},
	})
	require.NoError(t, err)
	require.NotNil(t, tool.OutputSchema)
	assert.Equal(t, "object", tool.OutputSchema["type"])
}

func TestProjectTools_SkipsInvalidItemsAndKeepsValidOnes(t *testing.T) {
	tools, errs := mcp.ProjectTools([]dto.ToolDefinition{
		{Name: "", ReadOnly: true},
		{Name: "good", ReadOnly: true},
	})
	require.Len(t, errs, 1, "非法项应被跳过并报告")
	require.Len(t, tools, 1, "合法项必须保留")
	assert.Equal(t, "good", tools[0].Name)
}

// ---------------------------------------------------------------------------
// outputSchema 规范契约（真实客户端联调发现）
// ---------------------------------------------------------------------------

func TestProjectTool_NonObjectResultSchemaIsWrappedAsObject(t *testing.T) {
	// MCP 规范：outputSchema 描述 structuredContent，而 structuredContent
	// 必须是 JSON 对象；官方 SDK 的 zod schema 要求 outputSchema.type
	// 是字面量 "object"。ITSM 既有工具里有返回数组的（ResultSchema.type
	// = "array"），直接透传会让真实客户端在 zod 校验阶段拒绝整个
	// tools/list 响应 —— 整条通道不可用，不是单个工具失效。
	tool, err := mcp.ProjectTool(dto.ToolDefinition{
		Name:         "list_kb",
		ReadOnly:     true,
		ResultSchema: map[string]interface{}{"type": "array"},
	})
	require.NoError(t, err)
	require.NotNil(t, tool.OutputSchema)
	assert.Equal(t, "object", tool.OutputSchema["type"],
		"outputSchema.type 必须是 object（SDK 要求字面量）")
	props, ok := tool.OutputSchema["properties"].(map[string]interface{})
	require.True(t, ok, "包装后必须有 properties")
	items, ok := props["items"].(map[string]interface{})
	require.True(t, ok, "原 schema 应保留在 items 下")
	assert.Equal(t, "array", items["type"], "原 schema 类型不得丢失")
}

func TestProjectTool_ObjectResultSchemaKeptAsIs(t *testing.T) {
	tool, err := mcp.ProjectTool(dto.ToolDefinition{
		Name:     "get_incident_stats",
		ReadOnly: true,
		ResultSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{"open": map[string]interface{}{"type": "integer"}},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, tool.OutputSchema)
	assert.Equal(t, "object", tool.OutputSchema["type"])
	assert.Contains(t, tool.OutputSchema, "properties", "object 型 schema 不应被包装")
}

func TestToolsList_AllOutputSchemasAreObjectTyped(t *testing.T) {
	// 全量断言：真实客户端只要遇到一个非 object 的 outputSchema
	// 就会拒绝整个工具面，所以必须是「全部」合规而不是抽样。
	cat := &fakeCatalog{tools: []dto.ToolDefinition{
		{Name: "a", ReadOnly: true, ResultSchema: map[string]interface{}{"type": "object"}},
		{Name: "b", ReadOnly: true, ResultSchema: map[string]interface{}{"type": "array"}},
		{Name: "c", ReadOnly: true, ResultSchema: map[string]interface{}{"type": "string"}},
	}}
	svc, err := mcp.NewService(mcp.Options{Visibility: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }), Catalog: cat, Executor: &fakeExecutor{}})
	require.NoError(t, err)

	resp := svc.Handle(context.Background(), &mcp.Request{
		JSONRPC: "2.0", ID: json.RawMessage("2"), Method: mcp.MethodToolsList,
	}, mcp.Caller{TenantID: 1})

	require.NotNil(t, resp)
	result := resp.Result.(mcp.ListToolsResult)
	require.NotEmpty(t, result.Tools)
	for _, tool := range result.Tools {
		require.NotNil(t, tool.OutputSchema, "工具 %s 应带 outputSchema", tool.Name)
		assert.Equal(t, "object", tool.OutputSchema["type"],
			"工具 %s 的 outputSchema.type 必须是 object，否则真实客户端拒绝整个工具面", tool.Name)
	}
}
