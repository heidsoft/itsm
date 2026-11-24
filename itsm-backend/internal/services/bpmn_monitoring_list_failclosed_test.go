package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	baseent "entgo.io/ent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ---------------------------------------------------------------------------
// P2-09 守卫：ListProcessInstancesStatus / GetProcessMetrics 的可信度三缺陷。
//
//  1. 租户 fail-closed：旧实现 `TenantID > 0` 才加租户谓词，缺租户上下文时
//     查询退化成全租户扫描（跨租户数据泄露到监控页）。
//  2. assignee 筛选必须下推到 SQL：旧实现先 Count/分页、再在内存过滤，
//     total 是过滤前的总数（分页 total 与实际条数不一致），且过滤发生在
//     分页之后导致页内条数错乱。
//  3. currentTask/progress 查询错误不得静默：旧实现 `if err == nil` 把 DB
//     故障伪装成「实例空闲 / 流程零进度」。
// ---------------------------------------------------------------------------

// firstFault 注入「第 nth 次 First 查询失败」：currentTask 查询走 OpQueryFirst。
func firstFault(nth int) baseent.Interceptor {
	seen := 0
	return baseent.InterceptFunc(func(next baseent.Querier) baseent.Querier {
		return baseent.QuerierFunc(func(ctx context.Context, q baseent.Query) (baseent.Value, error) {
			qc := baseent.QueryFromContext(ctx)
			if qc != nil && qc.Op == baseent.OpQueryFirst {
				seen++
				if seen == nth {
					return nil, fmt.Errorf("injected First failure #%d", nth)
				}
			}
			return next.Query(ctx, q)
		})
	})
}

// seedMonitorInstance 造一条 process_instances，返回实例 ID。
func seedMonitorInstance(t *testing.T, client *ent.Client, tenantID int, key string) int {
	t.Helper()
	ctx := context.Background()
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("DEP-MLIST-%d-%s", tenantID, key)).
		SetDeploymentName("monitor list seed").
		SetDeploymentTime(time.Now()).
		SetTenantID(tenantID).
		SetDeployedBy("test").
		Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey(key).SetName("monitor list seed").SetVersion("1").
		SetBpmnXML([]byte("<bpmn/>")).SetIsActive(true).SetIsLatest(true).
		SetTenantID(tenantID).SetDeploymentID(dep.ID).SetDeployedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	inst, err := client.ProcessInstance.Create().
		SetProcessInstanceID(key).SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).SetStatus("running").
		SetTenantID(tenantID).SetStartTime(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	return inst.ID
}

// seedAssignedTask 给实例挂一条 assigned 状态的活动任务。
func seedAssignedTask(t *testing.T, client *ent.Client, instanceID, tenantID int, taskKey, assignee string) {
	t.Helper()
	_, err := client.ProcessTask.Create().
		SetTaskID(taskKey).SetTaskDefinitionKey("T").SetTaskName(taskKey).
		SetProcessDefinitionKey("seed").SetProcessInstanceID(instanceID).
		SetStatus("assigned").SetAssignee(assignee).SetTenantID(tenantID).
		SetTaskVariables(map[string]interface{}{}).
		Save(context.Background())
	require.NoError(t, err)
}

func newMonitorListService(t *testing.T, inters ...baseent.Interceptor) (*BPMNMonitoringService, *ent.Client) {
	t.Helper()
	dsn := fmt.Sprintf("file:bpmn_mon_list_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })
	if len(inters) > 0 {
		client.Intercept(inters...)
	}
	return NewBPMNMonitoringService(client, nil, zaptest.NewLogger(t).Sugar()), client
}

// TestListProcessInstancesStatus_TenantFailClosed 是跨租户泄露的回归锁：
// TenantID=0 时旧实现不加租户谓词，返回全租户实例。现在必须 401 拒绝。
func TestListProcessInstancesStatus_TenantFailClosed(t *testing.T) {
	svc, client := newMonitorListService(t)
	seedMonitorInstance(t, client, 1, "PI-T1-A")
	seedMonitorInstance(t, client, 2, "PI-T2-A")

	for _, tenantID := range []int{0, -1} {
		t.Run(fmt.Sprintf("tenant_%d", tenantID), func(t *testing.T) {
			statuses, total, err := svc.ListProcessInstancesStatus(context.Background(),
				&dto.ListProcessInstanceStatusQuery{TenantID: tenantID})
			require.Error(t, err, "缺租户上下文必须 fail-closed，不得返回全租户数据")
			var appErr *common.AppError
			require.True(t, errors.As(err, &appErr), "错误必须是 *AppError 以便映射 401")
			assert.Equal(t, 401, appErr.HTTPStatus)
			assert.Nil(t, statuses)
			assert.Equal(t, 0, total)
		})
	}
}

// TestGetProcessMetrics_TenantFailClosed 是 GetProcessMetrics 同款租户缺陷的回归锁。
func TestGetProcessMetrics_TenantFailClosed(t *testing.T) {
	svc, client := newMonitorListService(t)
	seedMonitorInstance(t, client, 1, "PI-T1-M")

	metrics, err := svc.GetProcessMetrics(context.Background(),
		&dto.ProcessMetricsRequest{TenantID: 0})
	require.Error(t, err, "缺租户上下文必须 fail-closed")
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 401, appErr.HTTPStatus)
	assert.Nil(t, metrics)
}

// TestListProcessInstancesStatus_AssigneePushdownTotalConsistent 是筛选下推的回归锁：
// assignee 过滤后 total 必须等于过滤后的条数（旧实现 total 是过滤前的总数），
// 且分页发生在过滤之后（旧实现过滤在分页后，页内条数错乱）。
func TestListProcessInstancesStatus_AssigneePushdownTotalConsistent(t *testing.T) {
	svc, client := newMonitorListService(t)
	instA := seedMonitorInstance(t, client, 1, "PI-A")
	instB := seedMonitorInstance(t, client, 1, "PI-B")
	instC := seedMonitorInstance(t, client, 1, "PI-C")
	seedAssignedTask(t, client, instA, 1, "TASK-A1", "alice")
	seedAssignedTask(t, client, instB, 1, "TASK-B1", "alice")
	seedAssignedTask(t, client, instC, 1, "TASK-C1", "bob")
	// 跨租户同名 assignee：不得串到本租户结果里。
	instX := seedMonitorInstance(t, client, 2, "PI-X")
	seedAssignedTask(t, client, instX, 2, "TASK-X1", "alice")

	t.Run("filter_reduces_total", func(t *testing.T) {
		statuses, total, err := svc.ListProcessInstancesStatus(context.Background(),
			&dto.ListProcessInstanceStatusQuery{TenantID: 1, Assignee: "alice"})
		require.NoError(t, err)
		assert.Equal(t, 2, total, "total 必须是 assignee 过滤后的数量")
		assert.Len(t, statuses, 2, "返回条数必须与 total 一致")
		for _, s := range statuses {
			assert.Equal(t, "alice", s.Assignee)
			assert.Equal(t, 1, s.TenantID, "不得返回其他租户的实例")
		}
	})

	t.Run("no_match_returns_zero", func(t *testing.T) {
		statuses, total, err := svc.ListProcessInstancesStatus(context.Background(),
			&dto.ListProcessInstanceStatusQuery{TenantID: 1, Assignee: "nobody"})
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, statuses)
	})

	t.Run("pagination_counts_after_filter", func(t *testing.T) {
		// PageSize=1 + assignee 过滤：第 1 页 1 条、total=2。
		// 旧实现先分页（拿到 1 条）再内存过滤，可能返回 0 条而 total=3。
		statuses, total, err := svc.ListProcessInstancesStatus(context.Background(),
			&dto.ListProcessInstanceStatusQuery{TenantID: 1, Assignee: "alice", Page: 1, PageSize: 1})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, statuses, 1, "过滤必须发生在分页之前")
	})
}

// TestListProcessInstancesStatus_AssigneeIgnoresCompletedTasks 只有已完成任务的
// 实例不得被 assignee 命中——下推谓词必须限定 assigned/in_progress 活动态，
// 否则「历史经办人」会把已结束实例混进待办视角。
func TestListProcessInstancesStatus_AssigneeIgnoresCompletedTasks(t *testing.T) {
	svc, client := newMonitorListService(t)
	inst := seedMonitorInstance(t, client, 1, "PI-DONE")
	_, err := client.ProcessTask.Create().
		SetTaskID("TASK-DONE").SetTaskDefinitionKey("T").SetTaskName("done").
		SetProcessDefinitionKey("seed").SetProcessInstanceID(inst).
		SetStatus("completed").SetAssignee("alice").SetTenantID(1).
		SetTaskVariables(map[string]interface{}{}).
		Save(context.Background())
	require.NoError(t, err)

	statuses, total, err := svc.ListProcessInstancesStatus(context.Background(),
		&dto.ListProcessInstanceStatusQuery{TenantID: 1, Assignee: "alice"})
	require.NoError(t, err)
	assert.Equal(t, 0, total, "已完成任务不算活动 assignee")
	assert.Empty(t, statuses)
}

// TestListProcessInstancesStatus_CurrentTaskErrorPropagates 是 currentTask 吞错的
// 回归锁：旧实现 `if err == nil` 把 DB 故障静默成「实例空闲」，监控页显示
// 假的无当前任务。注入 First 故障必须向上返回错误。
func TestListProcessInstancesStatus_CurrentTaskErrorPropagates(t *testing.T) {
	svc, client := newMonitorListService(t, firstFault(1))
	seedMonitorInstance(t, client, 1, "PI-FT")

	statuses, total, err := svc.ListProcessInstancesStatus(context.Background(),
		&dto.ListProcessInstanceStatusQuery{TenantID: 1})
	require.Error(t, err, "currentTask 查询故障必须向上返回，不得伪装成实例空闲")
	assert.Nil(t, statuses)
	assert.Equal(t, 0, total)
}

// TestListProcessInstancesStatus_ProgressErrorPropagates 是 progress 吞错的回归锁：
// 旧实现 `if err == nil` 把 DB 故障静默成 0% 进度。注入 progress 的 Count 故障
// 必须向上返回错误。
func TestListProcessInstancesStatus_ProgressErrorPropagates(t *testing.T) {
	// 查询顺序：total Count(1) → All → currentTask First → progress totalTasks Count(2)
	// → progress completed Count(3)。
	svc, client := newMonitorListService(t, countFault(2))
	seedMonitorInstance(t, client, 1, "PI-PROG-FT")

	statuses, _, err := svc.ListProcessInstancesStatus(context.Background(),
		&dto.ListProcessInstanceStatusQuery{TenantID: 1})
	require.Error(t, err, "progress 计数故障必须向上返回，不得伪装成 0% 进度")
	assert.Nil(t, statuses)
}

// TestListProcessInstancesStatus_NoActiveTaskIsNormal 是 fail-closed 的成功侧：
// 实例没有活动任务时 currentTask 返回 NotFound 属正常态（不是故障），
// 不得被 fail-closed 误伤成错误。
func TestListProcessInstancesStatus_NoActiveTaskIsNormal(t *testing.T) {
	svc, client := newMonitorListService(t)
	seedMonitorInstance(t, client, 1, "PI-IDLE")

	statuses, total, err := svc.ListProcessInstancesStatus(context.Background(),
		&dto.ListProcessInstanceStatusQuery{TenantID: 1})
	require.NoError(t, err, "无活动任务是正常态，不得当故障上抛")
	assert.Equal(t, 1, total)
	require.Len(t, statuses, 1)
	assert.Empty(t, statuses[0].CurrentTask, "无活动任务时 CurrentTask 为空")
	assert.Equal(t, 0.0, statuses[0].Progress, "无任务时进度 0")
}
