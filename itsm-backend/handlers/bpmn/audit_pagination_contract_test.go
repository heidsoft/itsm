package bpmn

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 台账 E4-47⑤：/bpmn/dashboard/audit-logs 用 c.Query("pageSize") + strconv.Atoi，
// 只要求「能解析」就原样下传（连 >0 都不判）；/bpmn/monitoring/instances/status 用
// DefaultQuery 且把 Atoi 的错误丢掉（monitoring/audit-logs 第二套只读表面已在
// E4-48 收敛删除）。
// 两者都把值交给 service 里 `if Page > 0 && PageSize > 0 { Offset/Limit }` 这一分支，
// 于是 pageSize=abc（→0）或 pageSize=-5 会让 LIMIT/OFFSET 子句整体消失：一次写错的
// 分页参数就把该租户的审计日志读穿，而信封回显的仍是客户端原值。
// 现在 HTTP 入口的唯一所有者是 common.GetPaginationFromQuery（缺省 1/20，只采纳 [1,100]）。

const bpmnAuditPageSize = 25

func seedAuditLogsRange(t *testing.T, client *ent.Client, tenantID, n int, prefix string, base time.Time) {
	t.Helper()
	for i := 0; i < n; i++ {
		seedAuditLog(t, client, tenantID, 1000+i, fmt.Sprintf("%s-%02d", prefix, i), base.Add(time.Duration(i)*time.Minute))
	}
}

func TestDashboardAuditLogs_MissingPagingUsesPlatformDefault(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_audit_pg_default?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	seedAuditLogsRange(t, client, 1, bpmnAuditPageSize, "pg", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC))
	r := newEnvelopeTestRouter(t, client)

	data := decodeEnvelope(t, r, "/api/v1/bpmn/dashboard/audit-logs", "1")
	assertListEnvelope(t, data, bpmnAuditPageSize, 1, 20, 2, 20)
}

// TestDashboardAuditLogs_PageSizeCannotEscapePlatformBound 锁住三类越界取值：
// 越界大值、非数字、以及 HEAD 上真实读穿整表的负值。
func TestDashboardAuditLogs_PageSizeCannotEscapePlatformBound(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_audit_pg_bound?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	seedAuditLogsRange(t, client, 1, bpmnAuditPageSize, "pg", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC))
	r := newEnvelopeTestRouter(t, client)

	for _, query := range []string{"pageSize=5000", "pageSize=150", "pageSize=abc", "pageSize=0", "pageSize=-5"} {
		data := decodeEnvelope(t, r, "/api/v1/bpmn/dashboard/audit-logs?"+query, "1")
		assertListEnvelope(t, data, bpmnAuditPageSize, 1, 20, 2, 20, query)
	}

	// 非法页码同样只能落回 1，不能算出负 OFFSET。
	data := decodeEnvelope(t, r, "/api/v1/bpmn/dashboard/audit-logs?page=-3&pageSize=10", "1")
	assertListEnvelope(t, data, bpmnAuditPageSize, 1, 10, 3, 10)
}

func TestDashboardAuditLogs_SecondPageIsTheRemainder(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_audit_pg_remainder?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	seedAuditLogsRange(t, client, 1, bpmnAuditPageSize, "pg", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC))
	r := newEnvelopeTestRouter(t, client)

	data := decodeEnvelope(t, r, "/api/v1/bpmn/dashboard/audit-logs?page=2&pageSize=20", "1")
	assertListEnvelope(t, data, bpmnAuditPageSize, 2, 20, 2, 5)
}

// TestMonitoringInstancesStatus_EchoMatchesAdoptedLimit 锁另一半缺陷：
// handler 回显客户端原值，service 自己另夹一次，两者在 pageSize=abc 时分家。
func TestMonitoringInstancesStatus_EchoMatchesAdoptedLimit(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_mon_instances_pg?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	definition := seedProcessDefinition(t, client, 1)
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := 0; i < bpmnAuditPageSize; i++ {
		seedProcessInstance(t, client, definition, 1, fmt.Sprintf("pi-%02d", i), base.Add(time.Duration(i)*time.Minute))
	}
	r := newEnvelopeTestRouter(t, client)

	for _, query := range []string{"pageSize=abc", "pageSize=5000", "pageSize=-1"} {
		data := decodeEnvelope(t, r, "/api/v1/bpmn/monitoring/instances/status?"+query, "1")
		assertListEnvelope(t, data, bpmnAuditPageSize, 1, 20, 2, 20, query)
	}

	data := decodeEnvelope(t, r, "/api/v1/bpmn/monitoring/instances/status?page=2&pageSize=10", "1")
	assertListEnvelope(t, data, bpmnAuditPageSize, 2, 10, 3, 10)
}

// TestDashboardAuditLogs_DefaultPageSizeStillTenantScoped 证明「页长归一」没有把
// 租户收敛放宽：25 条属于租户 1，租户 2 只有自己那 3 条。
//
// 这里按 processInstanceKey 前缀而不是 tenant_id 判断：响应已是 camelCase DTO，
// tenantId 不再出现在契约里（台账 E4-48 已收敛）。
func TestDashboardAuditLogs_DefaultPageSizeStillTenantScoped(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_audit_pg_tenant?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	seedAuditLogsRange(t, client, 1, bpmnAuditPageSize, "t1", base)
	seedAuditLogsRange(t, client, 2, 3, "t2", base)
	r := newEnvelopeTestRouter(t, client)

	data := decodeEnvelope(t, r, "/api/v1/bpmn/dashboard/audit-logs?pageSize=5000", "2")
	require.Equal(t, float64(3), data["total"])
	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 3)
	for _, raw := range items {
		row, ok := raw.(map[string]interface{})
		require.True(t, ok)
		assert.True(t, strings.HasPrefix(fmt.Sprint(row["processInstanceKey"]), "t2-"),
			"响应不得混入其他租户的行，实际 %v", row["processInstanceKey"])
	}
}
