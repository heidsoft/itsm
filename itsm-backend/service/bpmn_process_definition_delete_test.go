package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/service/bpmn"
)

// 流程定义删除的错误语义：无引用可删、活跃实例与历史实例分别返回 conflict，
// 其他租户不可见（not-found），全部不得退化成 500/5001 或驱动层约束错误。

func TestDeleteProcessDefinitionRejectsLiveInstances(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "def-delete-live")
	instance := newLifecycleInstance(t, engine.client, tenantID, "def-live", "running")
	def, err := engine.client.ProcessDefinition.Get(ctx, instance.ProcessDefinitionID)
	require.NoError(t, err)

	err = engine.ProcessDefinitionService().DeleteProcessDefinition(ctx, def.Key, def.Version)

	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr), "活跃实例阻塞必须是 conflict，实际 %T: %v", err, err)
	assert.Equal(t, http.StatusConflict, appErr.HTTPStatus)
	assert.Contains(t, appErr.Message, "活跃实例")
	assert.True(t, definitionExists(t, engine.client, ctx, def.ID), "被拒绝的删除不得移除定义行")
}

// 只有 completed/terminated 历史时物理删除仍会破坏 process_instances 的必填外键，
// 必须给可解释的 conflict，而不是让数据库约束报错变成 500。
func TestDeleteProcessDefinitionRejectsHistoricalInstances(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "def-delete-history")
	instance := newLifecycleInstance(t, engine.client, tenantID, "def-history", "completed")
	def, err := engine.client.ProcessDefinition.Get(ctx, instance.ProcessDefinitionID)
	require.NoError(t, err)

	err = engine.ProcessDefinitionService().DeleteProcessDefinition(ctx, def.Key, def.Version)

	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr), "历史实例阻塞必须是 conflict，实际 %T: %v", err, err)
	assert.Equal(t, http.StatusConflict, appErr.HTTPStatus)
	assert.True(t, definitionExists(t, engine.client, ctx, def.ID))
}

func TestDeleteProcessDefinitionRemovesUnusedVersion(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "def-delete-unused")
	defID := createTestProcessDefinition(t, engine.client, tenantID, "def-unused", bizBPMNSimpleApproval)

	require.NoError(t, engine.ProcessDefinitionService().DeleteProcessDefinition(ctx, "def-unused", "1.0"))
	assert.False(t, definitionExists(t, engine.client, ctx, defID))
}

func TestDeleteProcessDefinitionIsTenantScoped(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "def-delete-tenant")
	createTestProcessDefinition(t, engine.client, tenantID, "def-other-tenant", bizBPMNSimpleApproval)

	otherTenantID := createTimerTenant(t, engine.client, "def-delete-other")
	otherCtx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, otherTenantID)

	err := engine.ProcessDefinitionService().DeleteProcessDefinition(otherCtx, "def-other-tenant", "1.0")
	require.Error(t, err, "跨租户不得删除他租户的流程定义")
	var notFound *ent.NotFoundError
	assert.True(t, errors.As(err, &notFound), "他租户定义按 not-found 处理，实际 %v", err)
	assert.Equal(t, 1, countDefinitions(t, engine.client, ctx, "def-other-tenant"))
}

func definitionExists(t *testing.T, client *ent.Client, ctx context.Context, id int) bool {
	t.Helper()
	exists, err := client.ProcessDefinition.Query().Where(processdefinition.ID(id)).Exist(ctx)
	require.NoError(t, err)
	return exists
}

func countDefinitions(t *testing.T, client *ent.Client, ctx context.Context, key string) int {
	t.Helper()
	count, err := client.ProcessDefinition.Query().Where(processdefinition.Key(key)).Count(ctx)
	require.NoError(t, err)
	return count
}
