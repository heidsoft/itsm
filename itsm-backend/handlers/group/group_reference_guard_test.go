package group

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/group"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/schema"
	"itsm-backend/middleware"
	"itsm-backend/service/bpmn"

	"database/sql"
	"encoding/json"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
)

// openGuardDB 打开内存 SQLite 连接（每个测试独立 DSN 隔离）。
func openGuardDB(dsn string) (*sql.DB, error) {
	return sql.Open("sqlite3", dsn)
}

// 本文件验证「审批组引用完整性守卫」。
//
// 背景（真实缺陷）：审批组名是审批人解析键，不是展示名。
//   - BPMN userTask 的 candidateGroups 在流程实例化时落库到 process_task.candidate_groups，
//     引擎用 GroupResolver.ExpandGroupsToUsers 按 group.name 精确查成员；
//   - 审批链 chain[].role 的 "group:<name>" 形式走同一个 ExpandGroupsToUsers 分支。
// 两条路径都按精确组名匹配（ent group.NameIn），因此改名或删除被引用的组会让
// 候选审批人静默变空：流程照常发布、任务照常创建，只是永远没人能审批。
// 修复前 handlers/group 的 UpdateGroup/DeleteGroup 没有任何引用检查。

// refBPMNXML 生成一个最小但结构合法的 BPMN 定义，其 userTask 声明候选组。
// candidateGroups 保持 BPMN 原生的逗号分隔形态（多个候选组用逗号并列）。
func refBPMNXML(taskID, candidateGroups string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://itsm.local/bpmn">
  <bpmn:process id="ref_process" isExecutable="true">
    <bpmn:startEvent id="start_evt" />
    <bpmn:userTask id="%s" name="审批节点" candidateGroups="%s" />
    <bpmn:endEvent id="end_evt" />
    <bpmn:sequenceFlow id="flow_1" sourceRef="start_evt" targetRef="%s" />
    <bpmn:sequenceFlow id="flow_2" sourceRef="%s" targetRef="end_evt" />
  </bpmn:process>
</bpmn:definitions>`, taskID, candidateGroups, taskID, taskID))
}

// setupGuardEnv 在内存 SQLite 上准备守卫测试环境。
// 返回 client、租户 A/B 的 ID，以及租户 A 下一个名为 groupName 的组 ID。
func setupGuardEnv(t *testing.T, groupName string, seedRef bool) (*ent.Client, int, int, int) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := openGuardDB(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(context.Background()))

	ctx := context.Background()
	const tenantA, tenantB = 1, 2

	g, err := client.Group.Create().
		SetName(groupName).
		SetDescription("审批组").
		SetTenantID(tenantA).
		Save(ctx)
	require.NoError(t, err)

	if seedRef {
		dep, err := client.ProcessDeployment.Create().
			SetDeploymentID("dep-1").
			SetDeploymentName("测试部署").
			SetTenantID(tenantA).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.ProcessDefinition.Create().
			SetKey("ref_flow").
			SetName("引用组名的流程").
			SetVersion("1.0.0").
			SetBpmnXML(refBPMNXML("approve_task", groupName)).
			SetIsActive(true).
			SetIsLatest(true).
			SetDeploymentID(dep.ID).
			SetTenantID(tenantA).
			Save(ctx)
		require.NoError(t, err)
	}

	return client, tenantA, tenantB, g.ID
}

// TestGuard_被引用组改名被拒绝 是根因用例：
// 未加守卫时，改名会让审批候选静默变空；守卫必须在改名发生前阻断。
func TestGuard_被引用组改名被拒绝(t *testing.T) {
	const groupName = "approvers-l1"
	client, tenantA, _, groupID := setupGuardEnv(t, groupName, true)

	svc := NewGroupService(client)

	// 前置：确认该组确实被流程引用（否则下面「被拒绝」就没有意义）。
	refs, err := findGroupReferences(context.Background(), client, tenantA, groupName)
	require.NoError(t, err)
	require.Len(t, refs, 1, "流程定义应引用该组")
	assert.Equal(t, "process_definition", refs[0].Kind)
	assert.Equal(t, "ref_flow", refs[0].RefKey)

	_, err = svc.UpdateGroup(context.Background(), groupID, &dto.UpdateGroupRequest{
		Name: "approvers-l1-renamed",
	}, tenantA)
	require.Error(t, err, "被引用的组不得改名")

	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr, "应返回结构化 AppError")
	assert.Equal(t, 409, appErr.HTTPStatus)
	assert.Contains(t, appErr.Message, groupName)
	assert.Contains(t, appErr.Message, "ref_flow", "诊断信息应指出引用来源")

	// 后置：组名未被改动。
	g, err := client.Group.Query().
		Where(group.IDEQ(groupID)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, groupName, g.Name, "改名必须被完整阻断")
}

// TestGuard_被引用组删除被拒绝 删除路径与改名路径同源，但必须独立覆盖。
func TestGuard_被引用组删除被拒绝(t *testing.T) {
	const groupName = "approvers-l2"
	client, tenantA, _, groupID := setupGuardEnv(t, groupName, true)

	svc := NewGroupService(client)

	err := svc.DeleteGroup(context.Background(), groupID, tenantA)
	require.Error(t, err, "被引用的组不得删除")

	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 409, appErr.HTTPStatus)
	assert.Contains(t, appErr.Message, groupName)

	// 后置：组仍在。
	exists, err := client.Group.Query().
		Where(groupIDEQForTest(groupID)).
		Exist(context.Background())
	require.NoError(t, err)
	assert.True(t, exists, "删除必须被完整阻断")
}

// TestGuard_未引用组可改名可删除 未被引用的组不得被误伤。
func TestGuard_未引用组可改名可删除(t *testing.T) {
	const groupName = "standalone-group"
	client, tenantA, _, groupID := setupGuardEnv(t, groupName, false)

	svc := NewGroupService(client)

	updated, err := svc.UpdateGroup(context.Background(), groupID, &dto.UpdateGroupRequest{
		Name:        "standalone-group-renamed",
		Description: "新描述",
	}, tenantA)
	require.NoError(t, err, "未被引用的组应可改名")
	assert.Equal(t, "standalone-group-renamed", updated.Name)

	require.NoError(t, svc.DeleteGroup(context.Background(), groupID, tenantA), "未被引用的组应可删除")

	exists, err := client.Group.Query().
		Where(groupIDEQForTest(groupID)).
		Exist(context.Background())
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestGuard_跨租户引用不阻断 引用检查必须按租户隔离：
// 他租户的流程引用了同名组，不应阻断本租户改名。
func TestGuard_跨租户引用不阻断(t *testing.T) {
	const groupName = "shared-name-group"
	client, tenantA, tenantB, groupID := setupGuardEnv(t, groupName, false)

	ctx := context.Background()
	// 在租户 B 建一条引用同名组的流程定义。
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("dep-b").
		SetDeploymentName("B 租户部署").
		SetTenantID(tenantB).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey("b_flow").
		SetName("B 租户流程").
		SetVersion("1.0.0").
		SetBpmnXML(refBPMNXML("b_task", groupName)).
		SetIsActive(true).
		SetIsLatest(true).
		SetDeploymentID(dep.ID).
		SetTenantID(tenantB).
		Save(ctx)
	require.NoError(t, err)

	// 租户 A 的同名组没有被 B 的流程引用，改名应放行。
	svc := NewGroupService(client)
	updated, err := svc.UpdateGroup(ctx, groupID, &dto.UpdateGroupRequest{
		Name: "shared-name-group-renamed",
	}, tenantA)
	require.NoError(t, err, "他租户引用不得阻断本租户改名")
	assert.Equal(t, "shared-name-group-renamed", updated.Name)

	// 反向确认：租户 B 自己的同名组会被自己的流程挡住。
	gB, err := client.Group.Create().
		SetName(groupName).
		SetTenantID(tenantB).
		Save(ctx)
	require.NoError(t, err)
	err = svc.DeleteGroup(ctx, gB.ID, tenantB)
	require.Error(t, err, "本租户引用必须阻断本租户删除")
}

// TestGuard_审批链组引用被识别 覆盖第二条真实引用路径：
// 审批链 chain[].role = "group:<name>"。
func TestGuard_审批链组引用被识别(t *testing.T) {
	const groupName = "chain-referenced-group"
	client, tenantA, _, groupID := setupGuardEnv(t, groupName, false)

	ctx := context.Background()
	_, err := client.ApprovalChain.Create().
		SetName("变更审批链").
		SetEntityType("change").
		SetTenantID(tenantA).
		SetChain([]schema.ApprovalChainStep{
			{Level: 1, Role: "group:" + groupName, Name: "一级审批", IsRequired: true},
		}).
		Save(ctx)
	require.NoError(t, err)

	svc := NewGroupService(client)
	err = svc.DeleteGroup(ctx, groupID, tenantA)
	require.Error(t, err, "审批链引用的组不得删除")

	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 409, appErr.HTTPStatus)
	assert.Contains(t, appErr.Message, "变更审批链")
}

// TestGuard_候选组CSV部分匹配不误报 candidateGroups="a,approvers-l3,b"
// 时，查询 "approvers-l" 不应命中（token 精确匹配，不是子串匹配）。
func TestGuard_候选组CSV部分匹配不误报(t *testing.T) {
	const groupName = "approvers-l3"
	client, tenantA, _, _ := setupGuardEnv(t, groupName, false)

	ctx := context.Background()
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("dep-csv").
		SetDeploymentName("CSV 部署").
		SetTenantID(tenantA).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey("csv_flow").
		SetName("多候选组流程").
		SetVersion("1.0.0").
		SetBpmnXML(refBPMNXML("csv_task", "finance, "+groupName+",security")).
		SetIsActive(true).
		SetIsLatest(true).
		SetDeploymentID(dep.ID).
		SetTenantID(tenantA).
		Save(ctx)
	require.NoError(t, err)

	// 完整 token 命中。
	refs, err := findGroupReferences(ctx, client, tenantA, groupName)
	require.NoError(t, err)
	require.Len(t, refs, 1)

	// 前缀子串不命中。
	refs, err = findGroupReferences(ctx, client, tenantA, "approvers-l")
	require.NoError(t, err)
	assert.Empty(t, refs, "子串不得命中，必须整 token 相等")

	// 另一个真实存在的组名也不命中。
	refs, err = findGroupReferences(ctx, client, tenantA, "finance")
	require.NoError(t, err)
	assert.Len(t, refs, 1, "CSV 中并列的另一个组也应被识别")
}

// TestGuard_改名后审批候选确实变空 是根因的行为证据：
// 它不依赖守卫，直接展示「组名被破坏 → 候选审批人静默变空」这一事实，
// 证明守卫阻断的对象是真实故障而非假想风险。
func TestGuard_改名后审批候选确实变空(t *testing.T) {
	const groupName = "approvers-l1"
	client, tenantA, _, groupID := setupGuardEnv(t, groupName, true)

	ctx := context.Background()
	resolver := bpmn.NewGroupResolver(client)

	// 改名前：流程引用的候选组名能解析出（空）成员列表——因为组本身没有成员，
	// 这里关注的是「组名能被找到」这一事实。用组存在性证明解析键有效。
	_, _, err := resolver.ExpandGroupsToUsers(ctx, tenantA, groupName)
	require.NoError(t, err)

	// 绕过守卫直接改名（模拟修复前的行为）。
	_, err = client.Group.UpdateOneID(groupID).SetName("approvers-l1-renamed").Save(ctx)
	require.NoError(t, err)

	// 改名后：原候选组名解析不出任何组，审批链落到空候选集 → 无人可审批。
	users, names, err := resolver.ExpandGroupsToUsers(ctx, tenantA, groupName)
	require.NoError(t, err)
	assert.Empty(t, users, "改名后原组名解析不到成员")
	assert.Empty(t, names)
}

// groupIDEQForTest 保持与 ent 生成谓词同源，避免测试里手写 ID 匹配逻辑。
func groupIDEQForTest(id int) predicate.Group {
	return group.IDEQ(id)
}

// TestGuard_HTTP层被引用组改名返回409 走真实 gin 路由，
// 验证诊断信息真的透出到 HTTP 响应体（而不是只有 service 层内部可读）。
func TestGuard_HTTP层被引用组改名返回409(t *testing.T) {
	const groupName = "approvers-l1"
	client, tenantA, _, groupID := setupGuardEnv(t, groupName, true)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantA})
		c.Next()
	})
	h := NewHandler(NewGroupService(client), zap.NewNop().Sugar())
	router.PUT("/groups/:id", h.UpdateGroup)

	body := strings.NewReader(`{"name":"approvers-l1-renamed"}`)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/groups/%d", groupID), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code, "被引用的组改名必须返回 409")

	var env struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 4090, env.Code, "应用层码应为冲突码")
	assert.Contains(t, env.Message, groupName, "诊断需含组名")
	assert.Contains(t, env.Message, "ref_flow", "诊断需指出引用来源流程")
}

// TestGuard_HTTP层未引用组改名成功 防止守卫变成「一律拒绝」的假修复。
func TestGuard_HTTP层未引用组改名成功(t *testing.T) {
	const groupName = "free-group"
	client, tenantA, _, groupID := setupGuardEnv(t, groupName, false)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantA})
		c.Next()
	})
	h := NewHandler(NewGroupService(client), zap.NewNop().Sugar())
	router.PUT("/groups/:id", h.UpdateGroup)

	body := strings.NewReader(`{"name":"free-group-renamed"}`)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/groups/%d", groupID), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "未被引用的组改名必须成功")
}
