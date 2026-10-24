package incident

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestCreateIncidentPersistsImpactFields 锁住 ITIL 影响度三字段的落库与回显。
//
// 2026-10-24 本地生产栈实测发现的静默缺陷：POST /incidents 传
// severity/impact/urgency = high，接口返回 200，但库里三列都是 schema 默认的
// medium，创建响应还回显空串——原因有两处：Create 的 ent 建库语句完全没有
// SetSeverity/SetImpact/SetUrgency，转换回领域对象时也没带上这三个字段。
// 影响是「影响度矩阵」从创建那一刻就是错的，而且没有任何报错可循。
func TestCreateIncidentPersistsImpactFields(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:inc_impact_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { _ = client.Close() })

	tenant, err := client.Tenant.Create().
		SetName("影响度测试租户").SetCode("impact").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	reporter, err := client.User.Create().
		SetUsername("impact-reporter").SetEmail("impact@example.com").
		SetName("Reporter").SetPasswordHash("hash").SetRole("end_user").
		SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// repo 只提供编号（PG 序列在 sqlite 下不可用），写路径走真实 ent client，
	// 这样断言的是 Create 的建库语句本身，而不是 mock 的回声。
	svc := NewService(newMockRepository(), client, nil, nil, nil, nil, nil, zap.NewNop().Sugar())

	created, err := svc.Create(ctx, tenant.ID, &Incident{
		Title:       "核心交换机宕机",
		Description: "整层楼网络中断",
		Priority:    "critical",
		Severity:    "high",
		Impact:      "High", // 大小写归一
		Urgency:     "critical",
		ReporterID:  reporter.ID,
	})
	require.NoError(t, err)

	// 1) 落库必须是调用方给的值，不能被 schema 默认 medium 覆盖
	row, err := client.Incident.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "high", row.Severity)
	assert.Equal(t, "high", row.Impact, "impact 应大小写归一后落库")
	assert.Equal(t, "critical", row.Urgency)

	// 2) 创建响应必须与库一致（此前回显空串）
	assert.Equal(t, "high", created.Severity)
	assert.Equal(t, "high", created.Impact)
	assert.Equal(t, "critical", created.Urgency)

	// 3) 未传时沿用 schema 默认 medium，且实体里就是 medium（不是空串）。
	//    换一个租户建：mock 仓储的编号是常量，唯一索引是 (tenant_id, incident_number)。
	fallbackTenant, err := client.Tenant.Create().
		SetName("影响度测试租户2").SetCode("impact2").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	fallbackReporter, err := client.User.Create().
		SetUsername("impact-reporter-2").SetEmail("impact2@example.com").
		SetName("Reporter").SetPasswordHash("hash").SetRole("end_user").
		SetActive(true).SetTenantID(fallbackTenant.ID).Save(ctx)
	require.NoError(t, err)

	fallback, err := svc.Create(ctx, fallbackTenant.ID, &Incident{
		Title: "未标注影响度", Description: "d", Priority: "low", ReporterID: fallbackReporter.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, "medium", fallback.Severity)
	assert.Equal(t, "medium", fallback.Impact)
	assert.Equal(t, "medium", fallback.Urgency)
}
