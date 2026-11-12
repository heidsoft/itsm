package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service/bpmn"
)

// 实例生命周期状态机：suspend/resume/terminate 只接受合法 source 状态，
// 非法迁移必须是 conflict（409），且不得改动实例状态；not-found 必须是 404。

func lifecycleTestSetup(t *testing.T, name string) (*CustomProcessEngine, context.Context, int) {
	t.Helper()
	client := enttest.Open(t, dialect.SQLite, "file:lifecycle_"+name+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	tenantID := createTimerTenant(t, client, name)
	ctx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, tenantID)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	return engine, ctx, tenantID
}

func newLifecycleInstance(t *testing.T, client *ent.Client, tenantID int, key, status string) *ent.ProcessInstance {
	t.Helper()
	defID := createTestProcessDefinition(t, client, tenantID, key, bizBPMNSimpleApproval)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-LIFECYCLE-" + key).
		SetProcessDefinitionKey(key).
		SetProcessDefinitionID(defID).
		SetStatus(status).
		SetCurrentActivityID("manager_approval").
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return instance
}

func requireConflict(t *testing.T, err error, action, from string) {
	t.Helper()
	require.Error(t, err, "%s 从 %s 迁移应被拒绝", action, from)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr), "%s 非法迁移必须是领域 conflict，实际 %T: %v", action, err, err)
	assert.Equal(t, http.StatusConflict, appErr.HTTPStatus, "body=%v", err)
}

func TestSuspendProcessRejectsIllegalSourceState(t *testing.T) {
	for _, status := range []string{"suspended", "completed", "terminated"} {
		t.Run(status, func(t *testing.T) {
			engine, ctx, tenantID := lifecycleTestSetup(t, "suspend-"+status)
			instance := newLifecycleInstance(t, engine.client, tenantID, "ok-suspend-"+status, status)

			err := engine.SuspendProcess(ctx, instance.ProcessInstanceID, "manual hold")

			requireConflict(t, err, "suspend", status)
			stored := reloadInstance(t, engine.client, instance.ID)
			assert.Equal(t, status, stored.Status, "被拒绝的迁移不得改动实例状态")
		})
	}
}

func TestResumeProcessRejectsIllegalSourceState(t *testing.T) {
	for _, status := range []string{"running", "completed", "terminated"} {
		t.Run(status, func(t *testing.T) {
			engine, ctx, tenantID := lifecycleTestSetup(t, "resume-"+status)
			instance := newLifecycleInstance(t, engine.client, tenantID, "resume-"+status, status)

			err := engine.ResumeProcess(ctx, instance.ProcessInstanceID)

			requireConflict(t, err, "resume", status)
			stored := reloadInstance(t, engine.client, instance.ID)
			assert.Equal(t, status, stored.Status, "被拒绝的迁移不得改动实例状态")
		})
	}
}

func TestTerminateProcessRejectsIllegalSourceState(t *testing.T) {
	for _, status := range []string{"completed", "terminated"} {
		t.Run(status, func(t *testing.T) {
			engine, ctx, tenantID := lifecycleTestSetup(t, "terminate-"+status)
			instance := newLifecycleInstance(t, engine.client, tenantID, "term-"+status, status)

			err := engine.TerminateProcess(ctx, instance.ProcessInstanceID, "no longer needed")

			requireConflict(t, err, "terminate", status)
			stored := reloadInstance(t, engine.client, instance.ID)
			assert.Equal(t, status, stored.Status, "被拒绝的迁移不得改动实例状态")
		})
	}
}

func TestProcessLifecycleAllowsLegalTransitions(t *testing.T) {
	t.Run("suspend_running", func(t *testing.T) {
		engine, ctx, tenantID := lifecycleTestSetup(t, "ok-suspend")
		instance := newLifecycleInstance(t, engine.client, tenantID, "ok-suspend", "running")
		require.NoError(t, engine.SuspendProcess(ctx, instance.ProcessInstanceID, "manual hold"))
		assert.Equal(t, "suspended", reloadInstance(t, engine.client, instance.ID).Status)
	})

	t.Run("resume_suspended", func(t *testing.T) {
		engine, ctx, tenantID := lifecycleTestSetup(t, "ok-resume")
		instance := newLifecycleInstance(t, engine.client, tenantID, "ok-resume", "suspended")
		require.NoError(t, engine.ResumeProcess(ctx, instance.ProcessInstanceID))
		assert.Equal(t, "running", reloadInstance(t, engine.client, instance.ID).Status)
	})

	t.Run("terminate_running", func(t *testing.T) {
		engine, ctx, tenantID := lifecycleTestSetup(t, "ok-term-running")
		instance := newLifecycleInstance(t, engine.client, tenantID, "ok-term-running", "running")
		require.NoError(t, engine.TerminateProcess(ctx, instance.ProcessInstanceID, "cancelled by owner"))
		assert.Equal(t, "terminated", reloadInstance(t, engine.client, instance.ID).Status)
	})

	t.Run("terminate_suspended", func(t *testing.T) {
		engine, ctx, tenantID := lifecycleTestSetup(t, "ok-term-suspended")
		instance := newLifecycleInstance(t, engine.client, tenantID, "ok-term-suspended", "suspended")
		require.NoError(t, engine.TerminateProcess(ctx, instance.ProcessInstanceID, "cancelled while paused"))
		assert.Equal(t, "terminated", reloadInstance(t, engine.client, instance.ID).Status)
	})
}

// 实例不存在时必须是 404 语义，而不是被兜底成 500。
func TestProcessLifecycleMissingInstanceIsNotFound(t *testing.T) {
	engine, ctx, _ := lifecycleTestSetup(t, "missing")

	err := engine.TerminateProcess(ctx, "PI-NOT-EXISTENT", "gone")
	require.Error(t, err)
	var notFound *ent.NotFoundError
	assert.True(t, errors.As(err, &notFound), "缺失实例必须是 not-found，实际 %T: %v", err, err)
}

func reloadInstance(t *testing.T, client *ent.Client, id int) *ent.ProcessInstance {
	t.Helper()
	stored, err := client.ProcessInstance.Get(context.Background(), id)
	require.NoError(t, err)
	return stored
}
