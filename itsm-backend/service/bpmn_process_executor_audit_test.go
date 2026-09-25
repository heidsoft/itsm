package service

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processauditlog"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"
)

const auditBPMNStartEndOnly = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="audit_start_end" name="Audit Start End Only" isExecutable="true">
    <bpmn:startEvent id="start" name="开始"/>
    <bpmn:endEvent id="end" name="结束"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

const auditBPMNUserTaskFlow = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="audit_complete" name="Audit Complete Task" isExecutable="true">
    <bpmn:startEvent id="start" name="开始"/>
    <bpmn:userTask id="task_review" name="复核"/>
    <bpmn:endEvent id="end" name="结束"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="task_review"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="task_review" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

func newAuditTestClient(t *testing.T, name string) *ent.Client {
	t.Helper()
	dsn := "file:" + name + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, dialect.SQLite, dsn)
	t.Cleanup(func() { client.Close() })
	return client
}

func createAuditTenant(t *testing.T, client *ent.Client, suffix string) int {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName("Audit Tenant " + suffix).
		SetCode("audit-" + suffix).
		SetDomain("audit-" + suffix + ".example.com").
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)
	return tenant.ID
}

func createAuditUser(t *testing.T, client *ent.Client, tenantID int, username, displayName string) int {
	t.Helper()
	userEntity, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "@example.com").
		SetName(displayName).
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return userEntity.ID
}

func deployAuditProcess(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, key, xml string) int {
	t.Helper()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("deploy-audit-" + key).
		SetDeploymentName("Audit Deployment " + key).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey(key).
		SetName("Audit Process " + key).
		SetVersion("1.0").
		SetBpmnXML([]byte(xml)).
		SetIsActive(true).
		SetIsLatest(true).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return def.ID
}

// TestStartProcess_AuditUserContextPropagated 验证：StartProcess 通过 typed
// BPMNUserIDContextKey 注入的 actor，最终写进 process_audit_logs 的 user_id 与
// user_name 字段。修复前由于 executor 走 ctx.Value("user").(*ent.User)，这两列
// 永远是 0 与空，导致前端审计列"操作人/受理人"始终空白。
func TestStartProcess_AuditUserContextPropagated(t *testing.T) {
	client := newAuditTestClient(t, "audit_start_process")
	logger := zaptest.NewLogger(t).Sugar()

	tenantID := createAuditTenant(t, client, "sp")
	actorID := createAuditUser(t, client, tenantID, "actor-sp", "审计启动人")

	deployAuditProcess(t, context.Background(), client, tenantID, "audit_start_end", auditBPMNStartEndOnly)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, tenantID)
	ctx = context.WithValue(ctx, bpmn.BPMNUserIDContextKey, actorID)

	instance, err := engine.StartProcess(ctx, "audit_start_end", "BK-AUDIT-SP-001", nil)
	require.NoError(t, err)
	require.NotNil(t, instance)

	logs, err := client.ProcessAuditLog.Query().
		Where(
			processauditlog.ProcessInstanceID(instance.ID),
			processauditlog.ActionEQ("started"),
		).
		All(context.Background())
	require.NoError(t, err)
	require.Len(t, logs, 1, "启动流程应当只产出一条 started 审计")

	log := logs[0]
	assert.Equal(t, actorID, log.UserID, "audit log.UserID 必须取自 typed context")
	assert.Equal(t, "审计启动人", log.UserName, "audit log.UserName 必须由事务内 User.Get 解析")
	assert.NotEmpty(t, strings.TrimSpace(log.UserName), "user_name 不应为空字符串")
}

// TestCompleteTask_AuditUserContextPropagated 验证 CompleteTask 路径同样把
// typed context 里的 actor 写入 process_audit_logs 的 completed 行。
func TestCompleteTask_AuditUserContextPropagated(t *testing.T) {
	client := newAuditTestClient(t, "audit_complete_task")
	logger := zaptest.NewLogger(t).Sugar()

	tenantID := createAuditTenant(t, client, "ct")
	actorID := createAuditUser(t, client, tenantID, "actor-ct", "审计完成任务人")

	deployAuditProcess(t, context.Background(), client, tenantID, "audit_complete", auditBPMNUserTaskFlow)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, tenantID)
	ctx = context.WithValue(ctx, bpmn.BPMNUserIDContextKey, actorID)

	instance, err := engine.StartProcess(ctx, "audit_complete", "BK-AUDIT-CT-001", nil)
	require.NoError(t, err)

	tasks, err := client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		All(context.Background())
	require.NoError(t, err)
	require.Len(t, tasks, 1, "启动后应创建一个 user task")

	// 修复：让 actorID 成为 task 的 assignee，否则 authorizeTaskActor 会以
	// "当前用户不是该任务的审批人或候选人" 拦截，错误会盖过我们要验证的 bug。
	_, err = client.ProcessTask.UpdateOneID(tasks[0].ID).
		SetAssignee(strconv.Itoa(actorID)).
		Save(context.Background())
	require.NoError(t, err)

	err = engine.CompleteTask(ctx, tasks[0].TaskID, map[string]interface{}{
		"approvalAction": "approve",
		"approvalResult": "approved",
	})
	require.NoError(t, err)

	completed, err := client.ProcessAuditLog.Query().
		Where(
			processauditlog.ProcessInstanceID(instance.ID),
			processauditlog.ActionEQ("completed"),
		).
		All(context.Background())
	require.NoError(t, err)
	require.Len(t, completed, 1, "完成任务应产出一条 completed 审计")

	log := completed[0]
	assert.Equal(t, actorID, log.UserID, "audit log.UserID 必须取自 typed context")
	assert.Equal(t, "审计完成任务人", log.UserName, "audit log.UserName 必须由事务内 User.Get 解析")
	assert.NotEmpty(t, strings.TrimSpace(log.UserName), "user_name 不应为空字符串")
}