package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/common"
)

// 流程定义不可变与激活 lint 门禁（codex 审计 P1-10）：
//  1. 已发布（当前激活）或已被实例引用的定义行，BPMN XML 不可原地替换——
//     运行中实例按 definition ID 加载图，换图会让进行中的实例读到新图；
//     （deployed_at 在 schema 上 Default(time.Now)，行创建即非零，不能当
//     「首次发布」标记，锁定判定用 IsActive + 实例存在性）；
//  2. 既未激活也无实例的惰性草稿仍可原地编辑，元数据（名称/描述）不受限；
//  3. ActivateVersion / RollbackToVersion 与设计器发布走同一道 validateActivatableBPMNXML，
//     草稿可以保存未过 lint 的 XML，但激活必须被拦下且零副作用。

func TestUpdateProcessDefinition_RejectsXMLChangeAfterPublish(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "def-immutable-published")
	// 夹具即已发布状态：IsActive=true（发布=激活，StartProcess 只加载 active+latest）。
	defID := createTestProcessDefinition(t, engine.client, tenantID, "imm-pub", bizBPMNSimpleApproval)

	_, err := engine.ProcessDefinitionService().UpdateProcessDefinition(ctx, "imm-pub", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: bizBPMNSimpleApproval + "\n<!-- edited -->"})

	require.Error(t, err, "已发布定义原地换图必须被拒绝")
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr), "必须是领域 conflict，实际 %T: %v", err, err)
	assert.Equal(t, http.StatusConflict, appErr.HTTPStatus)
	assert.Contains(t, appErr.Message, "不可原地修改")

	def, getErr := engine.client.ProcessDefinition.Get(ctx, defID)
	require.NoError(t, getErr)
	assert.Equal(t, bizBPMNSimpleApproval, string(def.BpmnXML), "被拒绝的更新不得改动已发布 XML")
	assert.True(t, def.IsActive, "被拒绝的更新不得改动激活态")
}

func TestUpdateProcessDefinition_RejectsXMLChangeWithInstances(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "def-immutable-instanced")
	instance := newLifecycleInstance(t, engine.client, tenantID, "imm-inst", "running")
	def, err := engine.client.ProcessDefinition.Get(ctx, instance.ProcessDefinitionID)
	require.NoError(t, err)
	// 停用后仍被实例引用：证明锁定判定的第二条（实例存在性）独立于 IsActive。
	require.NoError(t, engine.client.ProcessDefinition.UpdateOneID(def.ID).
		SetIsActive(false).Exec(ctx))
	def, err = engine.client.ProcessDefinition.Get(ctx, instance.ProcessDefinitionID)
	require.NoError(t, err)
	require.False(t, def.IsActive, "夹具应为已停用但已有实例的定义")

	_, err = engine.ProcessDefinitionService().UpdateProcessDefinition(ctx, def.Key, def.Version,
		&UpdateProcessDefinitionRequest{BPMNXML: bizBPMNSimpleApproval + "\n<!-- edited -->"})

	require.Error(t, err, "已有实例的定义换图必须被拒绝")
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr), "必须是领域 conflict，实际 %T: %v", err, err)
	assert.Equal(t, http.StatusConflict, appErr.HTTPStatus)

	reloaded, getErr := engine.client.ProcessDefinition.Get(ctx, def.ID)
	require.NoError(t, getErr)
	assert.Equal(t, bizBPMNSimpleApproval, string(reloaded.BpmnXML), "被拒绝的更新不得改动 XML")
}

func TestUpdateProcessDefinition_AllowsDraftXMLChange(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "def-immutable-draft")
	defID := createTestProcessDefinition(t, engine.client, tenantID, "imm-draft", bizBPMNSimpleApproval)
	// 草稿：未发布（DeployedAt 为空）且无实例 → 允许原地编辑。
	require.NoError(t, engine.client.ProcessDefinition.UpdateOneID(defID).
		SetIsActive(false).Exec(ctx))

	newXML := bizBPMNSimpleApproval + "\n<!-- draft edit -->"
	updated, err := engine.ProcessDefinitionService().UpdateProcessDefinition(ctx, "imm-draft", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: newXML})
	require.NoError(t, err, "草稿定义必须允许原地编辑 XML")
	assert.Equal(t, newXML, string(updated.BpmnXML))
}

func TestUpdateProcessDefinition_MetadataUpdateAllowedOnPublished(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "def-immutable-metadata")
	defID := createTestProcessDefinition(t, engine.client, tenantID, "imm-meta", bizBPMNSimpleApproval)

	updated, err := engine.ProcessDefinitionService().UpdateProcessDefinition(ctx, "imm-meta", "1.0",
		&UpdateProcessDefinitionRequest{Name: "改名后的流程"})
	require.NoError(t, err, "不可变的是 BPMN 内容，元数据仍可更新")
	assert.Equal(t, "改名后的流程", updated.Name)
	def, getErr := engine.client.ProcessDefinition.Get(ctx, defID)
	require.NoError(t, getErr)
	assert.Equal(t, bizBPMNSimpleApproval, string(def.BpmnXML), "元数据更新不得顺带改动 XML")
}

func TestPublishProcessDefinition_RejectsXMLChangeOnPublishedRow(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "def-publish-immutable")
	defID := createTestProcessDefinition(t, engine.client, tenantID, "pub-imm", bizBPMNSimpleApproval)

	_, err := engine.ProcessDefinitionService().PublishProcessDefinition(ctx, "pub-imm", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: bizBPMNSimpleApproval + "\n<!-- edited -->"})

	require.Error(t, err, "已发布行重发布时换图必须被拒绝")
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr), "必须是领域 conflict，实际 %T: %v", err, err)
	assert.Equal(t, http.StatusConflict, appErr.HTTPStatus)

	def, getErr := engine.client.ProcessDefinition.Get(ctx, defID)
	require.NoError(t, getErr)
	assert.Equal(t, bizBPMNSimpleApproval, string(def.BpmnXML), "被拒绝的发布不得改动已发布 XML")
	assert.True(t, def.IsActive, "拒绝必须发生在停用旧版本之前——激活态零变化")
}

func TestPublishProcessDefinition_AllowsRepublishIdenticalXML(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "def-publish-idempotent")
	defID := createTestProcessDefinition(t, engine.client, tenantID, "pub-same", bizBPMNSimpleApproval)

	// 幂等重发布（XML 完全一致）必须放行：设计器「重新发布」与时间表重建依赖此路径。
	published, err := engine.ProcessDefinitionService().PublishProcessDefinition(ctx, "pub-same", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: bizBPMNSimpleApproval})
	require.NoError(t, err, "同 XML 重发布必须是幂等放行")
	assert.Equal(t, bizBPMNSimpleApproval, string(published.BpmnXML))
	assert.True(t, published.IsActive)
	def, getErr := engine.client.ProcessDefinition.Get(ctx, defID)
	require.NoError(t, getErr)
	assert.Equal(t, bizBPMNSimpleApproval, string(def.BpmnXML), "幂等重发布不得改动 XML")
}

func TestPublishProcessDefinition_FirstPublishAllowsNewXML(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "def-publish-first")
	defID := createTestProcessDefinition(t, engine.client, tenantID, "pub-first", bizBPMNSimpleApproval)
	// 草稿行：未发布、无实例 → 首次发布允许落新 XML（草稿→发布的正常路径）。
	require.NoError(t, engine.client.ProcessDefinition.UpdateOneID(defID).
		SetIsActive(false).Exec(ctx))

	newXML := bizBPMNSimpleApproval + "\n<!-- final -->"
	published, err := engine.ProcessDefinitionService().PublishProcessDefinition(ctx, "pub-first", "1.0",
		&UpdateProcessDefinitionRequest{BPMNXML: newXML})
	require.NoError(t, err, "首次发布必须允许写入新 XML")
	assert.Equal(t, newXML, string(published.BpmnXML))
	assert.True(t, published.IsActive)
}

// 草稿允许保存未过 lint 的 XML（CreateVersion 不设门禁），但激活必须被
// validateActivatableBPMNXML 拦下，且不得翻转任何 active/latest 标志。
func TestActivateVersion_RejectsLintErrorXMLBeforeAnyWrite(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	seedDefinition(t, f, "lint_gate", "1") // v1 激活

	badXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="bad_proc" isExecutable="true">
    <bpmn:startEvent id="start"/>
  </bpmn:process>
</bpmn:definitions>` // 缺少结束事件 → lint error

	_, err := f.svc.CreateVersion(ctx, &CreateVersionRequest{
		ProcessDefinitionKey: "lint_gate",
		Name:                 "bad v2",
		BPMNXML:              badXML,
		TenantID:             f.tenantID,
		CreatedBy:            "1",
	})
	require.NoError(t, err, "草稿版本允许保存未过 lint 的 XML")

	err = f.svc.ActivateVersion(ctx, "lint_gate", 2, f.tenantID)
	require.Error(t, err, "未过 lint 的版本不得被激活")
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr), "激活拒绝必须是 400 语义的校验错误，实际 %T: %v", err, err)
	assert.Equal(t, http.StatusBadRequest, appErr.HTTPStatus)
	assert.Contains(t, err.Error(), "BPMN XML 校验失败")

	v1, getErr := f.svc.GetVersion(ctx, "lint_gate", 1, f.tenantID)
	require.NoError(t, getErr)
	assert.True(t, v1.IsActive, "拒绝激活不得停用现有激活版本")
	v2, getErr := f.svc.GetVersion(ctx, "lint_gate", 2, f.tenantID)
	require.NoError(t, getErr)
	assert.False(t, v2.IsActive, "被拒绝的版本不得被激活")
}

// 回滚链路（CreateVersion + ActivateVersion）继承同一道门禁：目标版本 XML 不过
// lint 时回滚必须失败且原激活版本保持激活。
func TestRollbackToVersion_RejectsLintErrorTarget(t *testing.T) {
	f := setupVersionFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	// v1 激活但 XML 缺 BPMN 命名空间 → lint 必失败。
	seedDefinitionWithXML(t, f, "rb_lint", "1", `<definitions><process id="p1"></process></definitions>`)

	err := f.svc.RollbackToVersion(ctx, "rb_lint", 1, f.tenantID, "bad target")
	require.Error(t, err, "回滚到不过 lint 的目标必须失败")
	assert.Contains(t, err.Error(), "激活回滚版本失败")
	assert.Contains(t, err.Error(), "BPMN XML 校验失败")

	v1, getErr := f.svc.GetVersion(ctx, "rb_lint", 1, f.tenantID)
	require.NoError(t, getErr)
	assert.True(t, v1.IsActive, "回滚被拒后原激活版本必须保持激活")
}
