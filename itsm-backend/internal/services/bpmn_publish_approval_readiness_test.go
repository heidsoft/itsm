package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/service/bpmn"
)

// 本文件验证「发布期审批人就绪守卫」。
//
// 真实缺陷：组名是审批人解析键。组成员可被单独移除（不触发 PR #147 的组级
// 改名/删除守卫），此时流程照常发布、任务照常创建，ExpandGroupsToUsers 展开
// 结果恒空 → 永远无人可批且无报错。守卫必须在写库前阻断并给出可定位诊断。
//
// 判定口径严格对齐运行时 service/bpmn.GroupResolver.ExpandGroupsToUsers：
// 组存在只看成员（即使空也不回退角色）；组不存在才尝试同名 role.code 回退且
// 只认活跃用户。所有用例走真实 ent 查询（内存 SQLite），不手抄判定逻辑副本。

const readinessTaskID = "Activity_Approve"

func readinessBPMN(candidateGroups string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="readiness_process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="%s" name="审批节点" candidateGroups="%s"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="%s"/>
    <bpmn:sequenceFlow id="f2" sourceRef="%s" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`, readinessTaskID, candidateGroups, readinessTaskID, readinessTaskID)
}

// setupReadinessDef 建一条未发布的草稿定义（首次发布才允许写入新 XML）。
func setupReadinessDef(t *testing.T, key string) (*CustomProcessEngine, context.Context, int, int) {
	t.Helper()
	engine, ctx, tenantID := lifecycleTestSetup(t, "readiness-"+key)
	defID := createTestProcessDefinition(t, engine.client, tenantID, key, bizBPMNSimpleApproval)
	require.NoError(t, engine.client.ProcessDefinition.UpdateOneID(defID).SetIsActive(false).Exec(ctx))
	return engine, ctx, tenantID, defID
}

func readinessUser(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, name string) *ent.User {
	t.Helper()
	u, err := client.User.Create().
		SetUsername("readiness-" + name).
		SetEmail(name + "@readiness.example.com").
		SetName(name).
		SetPasswordHash("h").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return u
}

func readinessGroup(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, name string, memberIDs ...int) {
	t.Helper()
	builder := client.Group.Create().SetName(name).SetTenantID(tenantID)
	if len(memberIDs) > 0 {
		builder.AddMemberIDs(memberIDs...)
	}
	_, err := builder.Save(ctx)
	require.NoError(t, err)
}

func readinessRole(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, code string) *ent.Role {
	t.Helper()
	r, err := client.Role.Create().SetCode(code).SetName(code).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	return r
}

func requireReadinessConflict(t *testing.T, err error, wantFragments ...string) {
	t.Helper()
	require.Error(t, err, "空审批组引用必须阻断发布")
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr), "必须是结构化 AppError，实际 %T: %v", err, err)
	assert.Equal(t, http.StatusConflict, appErr.HTTPStatus)
	for _, fragment := range wantFragments {
		assert.Contains(t, appErr.Message, fragment)
	}
}

// 1. 空组（存在但零成员，且无同名角色）→ 阻断发布，文案含节点/组名/原因。
func TestPublishApprovalReadiness_EmptyGroupBlocksPublish(t *testing.T) {
	engine, ctx, tenantID, defID := setupReadinessDef(t, "empty-group")
	readinessGroup(t, ctx, engine.client, tenantID, "approvers-l1")

	_, err := engine.ProcessDefinitionService().PublishProcessDefinition(ctx, "empty-group", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: readinessBPMN("approvers-l1")})

	requireReadinessConflict(t, err, readinessTaskID, "approvers-l1", "解析不到任何审批人")
	// 拒绝必须零写入：定义仍在草稿态、XML 未变。
	def, getErr := engine.client.ProcessDefinition.Get(ctx, defID)
	require.NoError(t, getErr)
	assert.False(t, def.IsActive, "被拒绝的发布不得翻激活态")
	assert.Equal(t, bizBPMNSimpleApproval, string(def.BpmnXML), "被拒绝的发布不得写入 XML")
}

// 2. 组有成员 → 放行。证明守卫不误伤正常审批组。
func TestPublishApprovalReadiness_GroupWithMembersAllowsPublish(t *testing.T) {
	engine, ctx, tenantID, _ := setupReadinessDef(t, "with-members")
	member := readinessUser(t, ctx, engine.client, tenantID, "member")
	readinessGroup(t, ctx, engine.client, tenantID, "approvers-l2", member.ID)

	published, err := engine.ProcessDefinitionService().PublishProcessDefinition(ctx, "with-members", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: readinessBPMN("approvers-l2")})

	require.NoError(t, err, "有成员的审批组必须允许发布")
	assert.True(t, published.IsActive)
	assert.Contains(t, string(published.BpmnXML), readinessTaskID)
}

// 3. 组不存在但同名角色有活跃用户 → 放行（运行时角色回退口径）。
// 这条是反证目标之一：中和角色回退分支后必须变红。
func TestPublishApprovalReadiness_RoleFallbackAllowsPublish(t *testing.T) {
	engine, ctx, tenantID, _ := setupReadinessDef(t, "role-fallback")
	role := readinessRole(t, ctx, engine.client, tenantID, "it_director")
	u := readinessUser(t, ctx, engine.client, tenantID, "director")
	require.NoError(t, u.Update().AddRoleIDs(role.ID).Exec(ctx))

	published, err := engine.ProcessDefinitionService().PublishProcessDefinition(ctx, "role-fallback", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: readinessBPMN("it_director")})

	require.NoError(t, err, "同名角色有活跃用户时必须按运行时回退口径放行")
	assert.True(t, published.IsActive)
}

// 4. 关键口径：组存在但零成员时，即使有同名角色活跃用户也不回退 → 仍阻断。
// 运行时 found[name]=true 会跳过角色分支，诊断不得比运行时更宽松。
func TestPublishApprovalReadiness_EmptyGroupDoesNotFallBackToRole(t *testing.T) {
	engine, ctx, tenantID, _ := setupReadinessDef(t, "empty-no-fallback")
	readinessGroup(t, ctx, engine.client, tenantID, "approvers-l3")
	role := readinessRole(t, ctx, engine.client, tenantID, "approvers-l3")
	u := readinessUser(t, ctx, engine.client, tenantID, "l3-director")
	require.NoError(t, u.Update().AddRoleIDs(role.ID).Exec(ctx))

	_, err := engine.ProcessDefinitionService().PublishProcessDefinition(ctx, "empty-no-fallback", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: readinessBPMN("approvers-l3")})

	requireReadinessConflict(t, err, "approvers-l3", "组存在但无成员")
}

// 5. 跨租户隔离：A 租户空组不阻断 B 租户同名有成员组的发布。
func TestPublishApprovalReadiness_CrossTenantIsolation(t *testing.T) {
	engine, ctxA, tenantA, _ := setupReadinessDef(t, "cross-tenant-a")
	readinessGroup(t, ctxA, engine.client, tenantA, "approvers-cross")

	tenantB := createTimerTenant(t, engine.client, "readiness-b")
	ctxB := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, tenantB)
	// B 租户独立建草稿定义 + 有成员的同名组。
	defB := createTestProcessDefinition(t, engine.client, tenantB, "cross-tenant-b", bizBPMNSimpleApproval)
	require.NoError(t, engine.client.ProcessDefinition.UpdateOneID(defB).SetIsActive(false).Exec(ctxB))
	memberB := readinessUser(t, ctxB, engine.client, tenantB, "member-b")
	readinessGroup(t, ctxB, engine.client, tenantB, "approvers-cross", memberB.ID)

	published, err := engine.ProcessDefinitionService().PublishProcessDefinition(ctxB, "cross-tenant-b", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: readinessBPMN("approvers-cross")})

	require.NoError(t, err, "他租户同名空组不得阻断本租户发布")
	assert.True(t, published.IsActive)

	// 反向：A 租户自己仍被空组阻断。
	_, errA := engine.ProcessDefinitionService().PublishProcessDefinition(ctxA, "cross-tenant-a", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: readinessBPMN("approvers-cross")})
	requireReadinessConflict(t, errA, "approvers-cross")
}

// 6. 坏 XML 失败关闭：无法确认审批人时拒绝发布，不能因解析失败而放行。
func TestPublishApprovalReadiness_MalformedXMLFailsClosed(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "readiness-bad-xml")
	badXML := `<?xml version="1.0"?><bpmn:definitions><bpmn:process`

	issues, err := inspectPublishApprovalReadiness(ctx, engine.client, tenantID, badXML)
	require.Error(t, err, "XML 解析失败必须返回 error")
	assert.Nil(t, issues)

	err = validatePublishApprovalReadiness(ctx, engine.client, tenantID, badXML)
	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, http.StatusConflict, appErr.HTTPStatus)
	assert.Contains(t, appErr.Message, "无法解析")
}

// 7. 重新激活路径共用守卫：停用后重新激活也不得放过空审批组引用。
func TestPublishApprovalReadiness_ReactivateBlocked(t *testing.T) {
	engine, ctx, tenantID, defID := setupReadinessDef(t, "reactivate")
	readinessGroup(t, ctx, engine.client, tenantID, "approvers-reactivate")
	// 让 store 中的 XML 就是带空组引用的版本（模拟已发布后被移除成员）。
	require.NoError(t, engine.client.ProcessDefinition.UpdateOneID(defID).
		SetBpmnXML([]byte(readinessBPMN("approvers-reactivate"))).Exec(ctx))

	err := engine.ProcessDefinitionService().SetProcessDefinitionActive(ctx, "reactivate", "1.0", true)

	requireReadinessConflict(t, err, "approvers-reactivate")
	def, getErr := engine.client.ProcessDefinition.Get(ctx, defID)
	require.NoError(t, getErr)
	assert.False(t, def.IsActive, "被拒绝的激活不得翻转激活态")
}

// 8. CSV 整 token 相等匹配：引用 approvers-l 不得被 approvers-l1 的成员满足。
func TestPublishApprovalReadiness_CSVTokenExactMatch(t *testing.T) {
	engine, ctx, tenantID, _ := setupReadinessDef(t, "csv-token")
	member := readinessUser(t, ctx, engine.client, tenantID, "l1-member")
	readinessGroup(t, ctx, engine.client, tenantID, "approvers-l1", member.ID)

	_, err := engine.ProcessDefinitionService().PublishProcessDefinition(ctx, "csv-token", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: readinessBPMN("approvers-l")})

	requireReadinessConflict(t, err, "approvers-l", "解析不到任何审批人")
}

// 9. 显式 assignee 节点跳过静态判定：已有明确审批人，不得误伤。
func TestPublishApprovalReadiness_ExplicitAssigneeSkipped(t *testing.T) {
	engine, ctx, _, _ := setupReadinessDef(t, "explicit-assignee")
	xmlWithAssignee := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="readiness_process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="Activity_Approve" name="审批节点" candidateGroups="approvers-missing" assignee="alice"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="Activity_Approve"/>
    <bpmn:sequenceFlow id="f2" sourceRef="Activity_Approve" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`)

	published, err := engine.ProcessDefinitionService().PublishProcessDefinition(ctx, "explicit-assignee", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: xmlWithAssignee})

	require.NoError(t, err, "显式 assignee 已有审批人，不得因空候选组误伤")
	assert.True(t, published.IsActive)
}
