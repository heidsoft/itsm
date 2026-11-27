package change

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// GetCMDBImpactSummary 的失败关闭契约（CMDB 作为变更流程数据依据的入口）。
//
// 摘要同时被两处消费：提交门禁 validateImpactGate 与风险等级推荐
// recommendRiskLevel / recommendImpactScope。旧实现里关系计数与开放事件计数
// 两个查询写成 `if err == nil { summary.X = count }` —— 查询失败时字段保持
// Go 零值 0，等于把「算不出来」汇报成「风险为零」：
//
//   - HighRiskDependencyCount 归 0 ⇒ recommendRiskLevel 从 high 掉到 low；
//   - OpenIncidentCount 归 0 ⇒ recommendImpactScope 从 high 掉到 low；
//   - RequiresReview 随之变 false ⇒ validateImpactGate 放行，
//     绑定了高风险依赖的变更以 low 风险提交成功。
//
// 这条语义方向是错的：风险数据读不出来时必须让调用方知道（返回错误），
// 而不是静默降级成「这个变更是安全的」。
//
// 制造查询失败的手法：在真实 SQLite 上 DROP 掉目标表，ent 的 Count 会拿到
// "no such table"，从而精确地只让被锁的那条查询失败，其余查询照常工作——
// 比 mock 更接近生产里的「表缺失/迁移未完成/连接抖动」。

func newImpactSummaryFixture(t *testing.T) (*Service, context.Context, int, string) {
	t.Helper()
	// 用命名内存库 + shared cache：这样能另开一条 database/sql 连接直接执行 DDL，
	// 制造「表不存在」这类真实查询失败，而不需要 mock ent 的内部结构。
	dsn := fmt.Sprintf("file:impact_summary_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("摘要失败关闭租户").SetCode("impact-fail-closed").
		SetDomain("impact-fail-closed.example.com").SetStatus("active").SaveX(ctx)

	repo := NewEntRepository(client, nil)
	svc := newTestService(repo, client, zaptest.NewLogger(t).Sugar(), nil)
	return svc, ctx, tenant.ID, dsn
}

// dropTable 通过独立连接删表：ent.Client 不暴露任意 SQL 执行面，
// 这是在不侵入实现的前提下制造「表缺失」的最短路径。
func dropTable(t *testing.T, dsn, table string) {
	t.Helper()
	raw, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer func() { _ = raw.Close() }()
	// IF EXISTS 让「已经删过」不致命，便于用例独立重跑。
	_, err = raw.Exec("DROP TABLE IF EXISTS " + table)
	require.NoError(t, err, "测试前提：需要能 DROP %s", table)
}

// seedChangeWithHighRiskDependencies 造一个「风险只来自高等级依赖」的变更：
// CI 自身 criticality 不是 critical、没有开放事件，唯一的高风险特征是
// 4 条 strength=critical 的关系——按 recommendRiskLevel 规则应当 high。
func seedChangeWithHighRiskDependencies(t *testing.T, svc *Service, ctx context.Context, tenantID int, relCount int) (int, int) {
	t.Helper()
	client := svc.entClient

	user := client.User.Create().SetUsername("impact-sum-u").SetEmail("impact-sum@example.com").
		SetName("U").SetPasswordHash("hash").SetRole("it_admin").SetActive(true).
		SetTenantID(tenantID).SaveX(ctx)

	ciType := client.CIType.Create().SetName("应用服务").SetTenantID(tenantID).SaveX(ctx)
	ci := client.ConfigurationItem.Create().SetName("订单服务").SetCiType("application").
		SetCiTypeID(ciType.ID).SetCriticality("medium").SetStatus("in_service").
		SetEnvironment("production").SetTenantID(tenantID).SaveX(ctx)
	// 关系另一端：与被锁 CI 无关的普通 CI，避免它们自己成为高等级端点。
	for i := 0; i < relCount; i++ {
		other := client.ConfigurationItem.Create().SetName(fmt.Sprintf("依赖-%d", i)).
			SetCiType("application").SetCiTypeID(ciType.ID).SetCriticality("low").
			SetStatus("in_service").SetEnvironment("production").SetTenantID(tenantID).SaveX(ctx)
		client.CIRelationship.Create().
			SetRelationshipType("depends_on").
			SetSourceCiID(ci.ID).SetTargetCiID(other.ID).
			SetStrength("critical").SetImpactLevel("critical").
			SetTenantID(tenantID).SaveX(ctx)
	}

	change := client.Change.Create().SetTitle("发布订单服务").SetDescription("d").
		SetStatus("draft").SetChangeNumber("C-IMPACT-SUM-001").SetRiskLevel("low").
		SetCreatedBy(user.ID).SetTenantID(tenantID).SaveX(ctx)
	change.Update().AddAffectedCis(ci).SaveX(ctx)

	return change.ID, ci.ID
}

func TestGetCMDBImpactSummary_HealthyPathRecommendsHigh(t *testing.T) {
	svc, ctx, tenantID, _ := newImpactSummaryFixture(t)
	changeID, _ := seedChangeWithHighRiskDependencies(t, svc, ctx, tenantID, 4)

	summary, err := svc.GetCMDBImpactSummary(ctx, changeID, tenantID)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, 4, summary.HighRiskDependencyCount, "4 条 critical 强度关系必须被计入")
	assert.Equal(t, "high", summary.RecommendedRiskLevel,
		"高等级依赖 >=4 时必须推荐 high（这是后续失败注入用例的对照基线）")
	assert.True(t, summary.RequiresReview, "high 风险 ⇒ RequiresReview")
}

func TestGetCMDBImpactSummary_RelationCountErrorIsNotSilentZero(t *testing.T) {
	svc, ctx, tenantID, dsn := newImpactSummaryFixture(t)
	changeID, _ := seedChangeWithHighRiskDependencies(t, svc, ctx, tenantID, 4)

	// 只打掉关系表：CI 查询与事件查询仍然可用。
	dropTable(t, dsn, "ci_relationships")

	summary, err := svc.GetCMDBImpactSummary(ctx, changeID, tenantID)

	// 核心断言：关系计数查不出来时不得返回一个「看起来正常」的摘要。
	if assert.Error(t, err, "关系计数查询失败必须冒泡为错误，而不是把计数当 0 继续算风险") {
		assert.Nil(t, summary, "出错时不得同时返回一份内容可疑的摘要")
		return
	}

	// 若实现选择「降级但显式标注」，至少要满足这两条之一，不能两者皆无。
	assert.False(t, true, "未预期的实现：返回了 nil error 但既未标注降级也未拒绝")
}

func TestGetCMDBImpactSummary_OpenIncidentCountErrorIsNotSilentZero(t *testing.T) {
	svc, ctx, tenantID, dsn := newImpactSummaryFixture(t)

	// 这个场景的风险完全来自开放事件：2 个未关闭事件 ⇒ recommendRiskLevel=high。
	client := svc.entClient
	user := client.User.Create().SetUsername("impact-inc-u").SetEmail("impact-inc@example.com").
		SetName("U").SetPasswordHash("hash").SetRole("it_admin").SetActive(true).
		SetTenantID(tenantID).SaveX(ctx)
	ciType := client.CIType.Create().SetName("中间件").SetTenantID(tenantID).SaveX(ctx)
	ci := client.ConfigurationItem.Create().SetName("消息队列").SetCiType("middleware").
		SetCiTypeID(ciType.ID).SetCriticality("medium").SetStatus("in_service").
		SetEnvironment("production").SetTenantID(tenantID).SaveX(ctx)
	for i := 0; i < 2; i++ {
		client.Incident.Create().SetIncidentNumber(fmt.Sprintf("INC-IMPACT-SUM-%03d", i)).
			SetTitle("消息堆积").SetDescription("d").SetStatus("open").SetPriority("high").
			SetConfigurationItemID(ci.ID).SetTenantID(tenantID).SetReporterID(user.ID).SaveX(ctx)
	}
	change := client.Change.Create().SetTitle("扩容消息队列").SetDescription("d").
		SetStatus("draft").SetChangeNumber("C-IMPACT-SUM-002").SetRiskLevel("low").
		SetCreatedBy(user.ID).SetTenantID(tenantID).SaveX(ctx)
	change.Update().AddAffectedCis(ci).SaveX(ctx)

	// 对照基线：健康时开放事件确实把风险推到 high。
	healthy, err := svc.GetCMDBImpactSummary(ctx, change.ID, tenantID)
	require.NoError(t, err)
	require.Equal(t, 2, healthy.OpenIncidentCount)
	require.Equal(t, "high", healthy.RecommendedRiskLevel, "2 个开放事件 ⇒ high")

	dropTable(t, dsn, "incidents")

	summary, err := svc.GetCMDBImpactSummary(ctx, change.ID, tenantID)
	if assert.Error(t, err, "开放事件计数查询失败必须冒泡为错误，而不是把计数当 0 继续算风险") {
		assert.Nil(t, summary)
		return
	}
	assert.False(t, true, "未预期的实现：返回了 nil error 但既未标注降级也未拒绝")
}

func TestSubmitChangeImpactQueryFailureRejects(t *testing.T) {
	for _, table := range []string{"ci_relationships", "incidents"} {
		t.Run(table, func(t *testing.T) {
			svc, ctx, tenantID, dsn := newImpactSummaryFixture(t)
			changeID, _ := seedChangeWithHighRiskDependencies(t, svc, ctx, tenantID, 1)
			svc.entClient.Change.UpdateOneID(changeID).SetImplementationPlan("deploy").SetRollbackPlan("restore").ExecX(ctx)
			dropTable(t, dsn, table)
			result, err := svc.SubmitChange(ctx, changeID, tenantID, 1, &dto.SubmitChangeRequest{})
			require.Nil(t, result)
			require.Error(t, err)
			var appErr *common.AppError
			require.ErrorAs(t, err, &appErr)
			require.Equal(t, http.StatusServiceUnavailable, appErr.HTTPStatus)
			require.Contains(t, appErr.Message, "修复 CMDB")
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("user_id", 1)
				c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
				c.Next()
			})
			router.POST("/api/v1/changes/:id/submit", NewHandler(svc).SubmitChange)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/changes/%d/submit", changeID), nil))
			require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
			var response struct {
				Code    int         `json:"code"`
				Message string      `json:"message"`
				Data    interface{} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			require.Equal(t, common.ServiceUnavailableCode, response.Code)
			require.Contains(t, response.Message, "修复 CMDB")
			require.Nil(t, response.Data)
			stored, err := svc.entClient.Change.Get(ctx, changeID)
			require.NoError(t, err)
			require.EqualValues(t, "draft", stored.Status)
			require.Zero(t, svc.entClient.ApprovalRecord.Query().CountX(ctx))
		})
	}
}
