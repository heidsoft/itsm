package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/ent/processinstance"
	"itsm-backend/service/bpmn"
)

// P2-05 错误类型化守卫。
//
// 审计发现的问题：BPMN 引擎的业务拒绝（终态冲突、版本冲突、校验失败、
// 候选人不符）大量使用裸 fmt.Errorf，RespondError 无法分类 → 一律兜底
// 5001/500。用户看到的是「服务器内部错误」，前端也无法按 4090/4000/2003
// 做差异化处理。
//
// 本文件的断言口径按审计要求：不按中文关键词猜错误类型，而是走生产同款
// common.RespondError 分流，断言真实 HTTP status、业务码与安全文案；
// 同时用 errors.As 断言错误类型本身（防止「文案对了但类型仍是裸错误」）。

// assertRespondError 走生产同款 RespondError 分流，断言真实 HTTP 响应。
func assertRespondError(t *testing.T, err error, wantStatus, wantCode int, wantMsgContains string) {
	t.Helper()
	require.Error(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/bpmn/test", nil)
	common.RespondError(c, err, "fallback-must-not-be-used")

	assert.Equal(t, wantStatus, w.Code, "HTTP status 必须来自错误类型，body=%s", w.Body.String())
	var resp struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, wantCode, resp.Code, "业务码必须分类正确，message=%q", resp.Message)
	assert.Contains(t, resp.Message, wantMsgContains)
	assert.Nil(t, resp.Data, "冲突/校验类错误不得携带 data 载荷（仅 VersionConflictError 有载荷）")
}

// assertAppError 断言错误本身是类型化 AppError 且 HTTP 语义正确。
func assertAppError(t *testing.T, err error, wantStatus int, wantMsgContains string) *common.AppError {
	t.Helper()
	require.Error(t, err)
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr, "错误必须是类型化 AppError，实际 %T: %v", err, err)
	assert.Equal(t, wantStatus, appErr.HTTPStatus)
	assert.Contains(t, appErr.Message, wantMsgContains)
	return appErr
}

// 认领拒绝：非候选人 → 403/2003；已被认领 → 409/4090。
// 旧实现返回裸 fmt.Errorf("you are not a candidate...") / ("task already
// assigned")，会被 classifyError 兜底成 5001/500。
func TestBPMNErrorTyping_ClaimRejections(t *testing.T) {
	engine, baseCtx := newApprovalDecisionTestEngine(t)
	tenantID, actorID := setupApprovalDecisionFixture(t, engine)
	ctx := context.WithValue(baseCtx, bpmn.BPMNTenantIDContextKey, tenantID)

	_, rejectedTaskID := createProcessFixture(t, engine, tenantID, "errtype-rejected")
	_, assignedTaskID := createProcessFixture(t, engine, tenantID, "errtype-assigned")
	_, err := engine.client.ProcessTask.UpdateOneID(rejectedTaskID).
		SetCandidateUsers("another-user").Save(context.Background())
	require.NoError(t, err)
	_, err = engine.client.ProcessTask.UpdateOneID(assignedTaskID).
		SetAssignee("999999").Save(context.Background())
	require.NoError(t, err)

	taskService := engine.TaskService()

	rejectedErr := taskService.ClaimTaskByID(ctx, rejectedTaskID, actorID)
	assertAppError(t, rejectedErr, http.StatusForbidden, "您不是该任务的候选人")
	assertRespondError(t, rejectedErr, http.StatusForbidden, common.ForbiddenCode, "您不是该任务的候选人")

	assignedErr := taskService.ClaimTaskByID(ctx, assignedTaskID, actorID)
	assertAppError(t, assignedErr, http.StatusConflict, "任务已被他人认领")
	assertRespondError(t, assignedErr, http.StatusConflict, common.ConflictCode, "任务已被他人认领")
}

// 取消已结束任务 → 409/4090，而不是 5001。
func TestBPMNErrorTyping_CancelCompletedTask(t *testing.T) {
	client := bizTestClient(t, "errtype_cancel")
	ctx, tenantID := bizTenantCtx(t, client, "etc1")

	bizDeploy(t, ctx, client, tenantID, "errtype_cancel_proc", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	instance, err := engine.StartProcess(ctx, "errtype_cancel_proc", "BK-ETC1", nil)
	require.NoError(t, err)
	task := bizFindTask(t, ctx, client, instance.ID, "manager_approval")
	require.NoError(t, engine.CompleteTask(ctx, task.TaskID, map[string]interface{}{"approvalAction": "approve"}))

	cancelErr := engine.TaskService().CancelTask(ctx, task.TaskID, "迟到的取消")
	assertAppError(t, cancelErr, http.StatusConflict, "任务已结束，不能取消")
	assertRespondError(t, cancelErr, http.StatusConflict, common.ConflictCode, "任务已结束，不能取消")
}

// 重复完成已结束任务 → 409/4090（实例仍在运行、任务已被置 completed 的
// 并发窗口），而不是 5001。
func TestBPMNErrorTyping_CompleteCompletedTask(t *testing.T) {
	client := bizTestClient(t, "errtype_complete")
	ctx, tenantID := bizTenantCtx(t, client, "etc2")

	bizDeploy(t, ctx, client, tenantID, "errtype_complete_proc", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	instance, err := engine.StartProcess(ctx, "errtype_complete_proc", "BK-ETC2", nil)
	require.NoError(t, err)
	task := bizFindTask(t, ctx, client, instance.ID, "manager_approval")

	// 直接把任务置为 completed（实例保持 running），复现「重复完成」窗口。
	_, err = client.ProcessTask.UpdateOneID(task.ID).
		SetStatus("completed").Save(ctx)
	require.NoError(t, err)

	completeErr := engine.CompleteTask(ctx, task.TaskID, nil)
	assertAppError(t, completeErr, http.StatusConflict, "任务已结束，不能重复完成")
	assertRespondError(t, completeErr, http.StatusConflict, common.ConflictCode, "任务已结束，不能重复完成")
}

// 发布路径的三类校验失败必须是 400/4000：
//   - 空 BPMN XML；
//   - 候选策略缺字段（具体缺失项必须进响应文案）；
//   - lint 拦截的非法 XML。
func TestBPMNErrorTyping_PublishValidation(t *testing.T) {
	client := bizTestClient(t, "errtype_publish")
	ctx, _ := bizTenantCtx(t, client, "etp1")
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	defSvc := engine.ProcessDefinitionService()

	// 1. 空 XML。
	emptyErr := func() error {
		_, err := defSvc.PublishProcessDefinition(ctx, "etp1", "1.0", &UpdateProcessDefinitionRequest{})
		return err
	}()
	assertAppError(t, emptyErr, http.StatusBadRequest, "发布流程必须包含 BPMN XML")
	assertRespondError(t, emptyErr, http.StatusBadRequest, common.BadRequestCode, "发布流程必须包含 BPMN XML")

	// 2. 候选策略缺字段：具体缺失项必须进响应文案（classifyError 只回显
	//    最外层 AppError.Message）。
	badCandidateErr := func() error {
		_, err := defSvc.PublishProcessDefinition(ctx, "etp1", "1.0", &UpdateProcessDefinitionRequest{
			BPMNXML:            bizBPMNSimpleApproval,
			CandidateDefinition: map[string]interface{}{"domain": "expense"},
		})
		return err
	}()
	assertAppError(t, badCandidateErr, http.StatusBadRequest, "候选流程策略校验失败")
	assert.Contains(t, badCandidateErr.Error(), "缺少", "具体缺失项必须保留在错误链中")
	assertRespondError(t, badCandidateErr, http.StatusBadRequest, common.BadRequestCode, "候选流程策略校验失败")

	// 3. lint 拦截的非法 XML（helper 直调，覆盖 Update/SetActive/Publish
	//    共用的类型化出口）。
	lintErr := validateActivatableBPMNXML(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="bad_proc" isExecutable="true">
    <bpmn:startEvent id="start"/>
  </bpmn:process>
</bpmn:definitions>`)
	assertAppError(t, lintErr, http.StatusBadRequest, "BPMN 发布校验失败")
	assertRespondError(t, lintErr, http.StatusBadRequest, common.BadRequestCode, "BPMN 发布校验失败")
}

// 实例变量 CAS 重试预算耗尽 → 409/4090 的类型化版本冲突，
// 而不是裸 fmt.Errorf 兜底的 5001。
func TestBPMNErrorTyping_VersionConflict(t *testing.T) {
	client := bizTestClient(t, "errtype_verconf")
	ctx, tenantID := bizTenantCtx(t, client, "evc1")
	seedOneProcessInstance(t, client, tenantID, "PI-ERRTYPE-EVC")

	inst, err := client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Only(ctx)
	require.NoError(t, err)

	// mutate 在「读取快照」与「CAS 写入」之间被调用：每次尝试都推进版本，
	// 模拟持续并发写者，让 4 次重试预算必然耗尽。
	_, err = updateProcessInstanceVariablesCAS(ctx, client, inst.ID, tenantID,
		func(_ map[string]interface{}) error {
			return client.ProcessInstance.Update().
				Where(processinstance.ID(inst.ID)).
				AddVersion(1).
				Exec(ctx)
		})
	assertAppError(t, err, http.StatusConflict, "流程实例版本冲突")
	assertRespondError(t, err, http.StatusConflict, common.ConflictCode, "流程实例版本冲突")
}

// 缺租户上下文：requireBPMNTenantContext 的拒绝也必须类型化
// （P2-05 同类——裸错误曾被兜底成 5001，让 fail-closed 表现为 5xx 毛刺）。
func TestBPMNErrorTyping_MissingTenantContext(t *testing.T) {
	client := bizTestClient(t, "errtype_noctx")
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)

	_, err := engine.StartProcess(context.Background(), "whatever", "BK-NOCTX", nil)
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr, "缺租户上下文必须是类型化错误，实际 %T: %v", err, err)
	assert.Equal(t, common.AuthFailedCode, bizErr.Code)
	assertRespondError(t, err, http.StatusUnauthorized, common.AuthFailedCode, "未授权访问：缺少有效租户上下文")
}
