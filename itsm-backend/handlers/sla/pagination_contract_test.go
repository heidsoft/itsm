package sla

// 台账 E4-47①–② 回归锁：SLA 域四个分页端点（definitions / violations /
// performance / alert-history）的页长唯一所有者是 common.GetPaginationFromQuery
// （缺省 1/20，只采纳 [1,100]，越界/非数字/非正回落缺省，信封回显=实际下传值）。
//
// 修复前的四套私有 owner：
//   - ListSLADefinitions：缺省 10、接受 size 别名、无上界（pageSize=5000 原样进 LIMIT）；
//   - GetSLAViolations：缺省 20、同样带 size 别名、无上界；
//   - GetAlertHistory：strconv.Atoi 失败得 0，LIMIT 子句整体消失（整表读取）；
//   - GetSLAPerformance：入口把非法值判 400，service 又私有夹紧 >200→200
//     （200 不是平台值，平台 MaxPageSize=100）。
//
// 跑测命令：cd itsm-backend && go test ./handlers/sla/ -run Pagination

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const slaPageSeedCount = 25

func seedSLADefinitionsForPagination(t *testing.T, client *ent.Client, tenantID, n int) {
	t.Helper()
	ctx := context.Background()
	uid := slaUniqueID()
	for i := 0; i < n; i++ {
		_, err := client.SLADefinition.Create().
			SetName(fmt.Sprintf("pg-sla-%s-%02d", uid, i)).
			SetResponseTime(30).
			SetResolutionTime(240).
			SetIsActive(true).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
	}
}

func slaListData(t *testing.T, r *gin.Engine, path string) map[string]interface{} {
	t.Helper()
	resp := doSLAReq(t, r, http.MethodGet, path, nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "body=%s", slaStr(resp))
	return data
}

// assertSLAStandardEnvelope 锁定平台五键信封与「回显即采纳」口径。
func assertSLAStandardEnvelope(t *testing.T, data map[string]interface{}, wantTotal, wantPage, wantPageSize, wantTotalPages, wantItems int, ctx string) {
	t.Helper()
	assert.Len(t, data, 5, "%s: 信封必须恰为 {items,total,page,pageSize,totalPages}，实际 %v", ctx, data)
	for _, key := range []string{"items", "total", "page", "pageSize", "totalPages"} {
		assert.Contains(t, data, key, ctx)
	}
	assert.Equal(t, float64(wantTotal), slaNum(t, data, "total"), ctx)
	assert.Equal(t, float64(wantPage), slaNum(t, data, "page"), ctx)
	assert.Equal(t, float64(wantPageSize), slaNum(t, data, "pageSize"), ctx)
	assert.Equal(t, float64(wantTotalPages), slaNum(t, data, "totalPages"), ctx)
	items, ok := data["items"].([]interface{})
	require.True(t, ok, "%s: items 必须是数组，实际 %T", ctx, data["items"])
	assert.Len(t, items, wantItems, ctx)
}

func TestListSLADefinitions_MissingPagingUsesPlatformDefault(t *testing.T) {
	r, client, tenantID := setupSLAHandler(t)
	seedSLADefinitionsForPagination(t, client, tenantID, slaPageSeedCount)

	data := slaListData(t, r, "/api/v1/sla/definitions")
	assertSLAStandardEnvelope(t, data, slaPageSeedCount, 1, 20, 2, 20, "缺省 1/20（修复前是 1/10）")
}

// TestListSLADefinitions_PageSizeCannotEscapePlatformBound 锁住越界大值、非数字、
// 零/负值与 size 别名：全部回落 20，且回显值就是真正下传给 Offset/Limit 的值。
// 修复前 pageSize=5000 会原样进 LIMIT，size=10 会被私有别名接受。
func TestListSLADefinitions_PageSizeCannotEscapePlatformBound(t *testing.T) {
	r, client, tenantID := setupSLAHandler(t)
	seedSLADefinitionsForPagination(t, client, tenantID, slaPageSeedCount)

	for _, query := range []string{"pageSize=5000", "pageSize=150", "pageSize=abc", "pageSize=0", "pageSize=-5", "size=10"} {
		data := slaListData(t, r, "/api/v1/sla/definitions?"+query)
		assertSLAStandardEnvelope(t, data, slaPageSeedCount, 1, 20, 2, 20, query)
	}

	// 非数字页码回落 1，不能算出负 OFFSET。
	data := slaListData(t, r, "/api/v1/sla/definitions?page=abc&pageSize=10")
	assertSLAStandardEnvelope(t, data, slaPageSeedCount, 1, 10, 3, 10, "page=abc")

	data = slaListData(t, r, "/api/v1/sla/definitions?page=3&pageSize=10")
	assertSLAStandardEnvelope(t, data, slaPageSeedCount, 3, 10, 3, 5, "末页余数")
}

func TestGetSLAViolations_StandardEnvelopeAdoptsPlatformBound(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	for _, query := range []string{"pageSize=5000", "pageSize=abc", "size=10"} {
		data := slaListData(t, r, "/api/v1/sla/violations?"+query)
		assertSLAStandardEnvelope(t, data, 0, 1, 20, 0, 0, query)
	}

	// [1,100] 内的值照旧采纳。
	data := slaListData(t, r, "/api/v1/sla/violations?page=1&pageSize=5")
	assertSLAStandardEnvelope(t, data, 0, 1, 5, 0, 0, "pageSize=5 采纳")
}

// TestGetAlertHistory_BadPageSizeFallsBackInsteadOfWholeTableRead 锁修复前最危险的
// 形态：Atoi 失败得 0，Ent 的 LIMIT 子句整体消失，一次写错分页参数读穿该租户
// 全部告警历史。现在一律回落 20。
func TestGetAlertHistory_BadPageSizeFallsBackInsteadOfWholeTableRead(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	for _, query := range []string{"pageSize=abc", "pageSize=-1", "pageSize=0", "pageSize=5000"} {
		data := slaListData(t, r, "/api/v1/sla/alert-history?"+query)
		assertSLAStandardEnvelope(t, data, 0, 1, 20, 0, 0, query)
	}
}

// TestGetSLAPerformance_PageLengthPlatformOwner 锁 E4-47① 的另一半：入口此前把
// 非数字/零值判 400，越界大值则原样下传、由 service 私有上限夹到 200（第三套界）。
// 现在越界/非数字回落 20，且信封回显即采纳值。
func TestGetSLAPerformance_PageLengthPlatformOwner(t *testing.T) {
	_, client, tenantID := setupSLAHandler(t)
	f := seedSLAMonitoringFixture(t, client, tenantID)
	engine := slaEngineForTenant(t, client, tenantID)
	window := "startDate=" + f.start.Format(time.RFC3339) + "&endDate=" + f.end.Format(time.RFC3339)

	for _, query := range []string{"pageSize=5000", "pageSize=150", "pageSize=abc", "pageSize=0", "page=-2"} {
		resp, status := doSLAReqRaw(t, engine, http.MethodGet, "/api/v1/sla/performance?"+window+"&"+query, nil)
		require.Equal(t, http.StatusOK, status, "%s: 非法页长不再是 400，回落缺省（body=%s）", query, slaStr(resp))
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		data, ok := resp.Data.(map[string]interface{})
		require.True(t, ok, "body=%s", slaStr(resp))
		assert.Equal(t, float64(1), slaNum(t, data, "page"), query)
		assert.Equal(t, float64(20), slaNum(t, data, "pageSize"), query)
	}

	// 窗口内取值照常采纳。
	resp, status := doSLAReqRaw(t, engine, http.MethodGet, "/api/v1/sla/performance?"+window+"&page=2&pageSize=2", nil)
	require.Equal(t, http.StatusOK, status, "body=%s", slaStr(resp))
	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "body=%s", slaStr(resp))
	assert.Equal(t, float64(2), slaNum(t, data, "page"))
	assert.Equal(t, float64(2), slaNum(t, data, "pageSize"))
}
