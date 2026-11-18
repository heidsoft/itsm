package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ciattributedefinition"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"

	cmdbHandler "itsm-backend/handlers/cmdb"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestCMDBGovernanceReportRouteContract 打在真实 SetupRoutes 上，锁治理看板
// 路由的三态契约：无 CMDB 读权的角色 403、跨租户读不到别人的治理数据、
// 归属租户 super_admin 200 且响应键为 camelCase 治理结构。
func TestCMDBGovernanceReportRouteContract(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_cmdb_governance_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:   cmdbListSecret,
		Logger:      logger,
		Client:      client,
		CMDBHandler: cmdbHandler.NewHandler(cmdbHandler.NewService(cmdbHandler.NewEntRepository(client), nil, logger)),
	})

	tenantA := seedGovernanceTenant(ctx, t, client, "gov-a")
	tenantB := seedGovernanceTenant(ctx, t, client, "gov-b")

	call := func(t *testing.T, userID int, role string, tenantID int) (*httptest.ResponseRecorder, governanceEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(userID, "user", role, tenantID, cmdbListSecret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/governance/report", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env governanceEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w, env
	}

	t.Run("归属租户 super_admin 200 且响应为 camelCase 治理结构", func(t *testing.T) {
		w, env := call(t, tenantA.adminID, "super_admin", tenantA.tenantID)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, env.Code, "body=%s", w.Body.String())
		assert.Contains(t, w.Body.String(), `"completenessPct"`, "质量指标必须是 camelCase: %s", w.Body.String())
		assert.Contains(t, w.Body.String(), `"retire_confirm"`, "差异分桶键必须是受控词表: %s", w.Body.String())
		// 空租户也是全 0 快照，不是 null data
		assert.NotEqual(t, "null", string(env.Data), "data=%s", string(env.Data))
	})

	t.Run("无 cmdb:read 的角色 403", func(t *testing.T) {
		// end_user 的默认集已无 cmdb:read（2026-10-26 自助门户收敛退役）
		w, env := call(t, tenantA.endUserID, "end_user", tenantA.tenantID)
		assert.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		assert.Equal(t, 2003, env.Code)
	})

	t.Run("跨租户读取只看到自己的空快照", func(t *testing.T) {
		// A 管理员声明自己是 B 租户：B 的治理快照与 A 的互不可见，
		// 且各租户的 data 都是各自的计数（此处双侧都空，键结构必须一致成立）。
		w, env := call(t, tenantA.adminID, "super_admin", tenantB.tenantID)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, env.Code)
		assert.Contains(t, w.Body.String(), `"cis":0`)
	})
}

// governanceEnvelope 复用全站 {code,message,data} 契约。
type governanceEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type governanceTenantFixture struct {
	tenantID  int
	adminID   int
	endUserID int
}

func seedGovernanceTenant(ctx context.Context, t *testing.T, client *ent.Client, code string) governanceTenantFixture {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName("Governance " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	admin, err := client.User.Create().
		SetUsername(code + "-admin").
		SetEmail(code + "-admin@example.com").
		SetName(code + "-admin").
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	endUser, err := client.User.Create().
		SetUsername(code + "-enduser").
		SetEmail(code + "-enduser@example.com").
		SetName(code + "-enduser").
		SetPasswordHash("hash").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return governanceTenantFixture{tenantID: tenant.ID, adminID: admin.ID, endUserID: endUser.ID}
}

// TestCMDBGovernanceReportRouteQualityContract 锁属性级完整率与重复率的对外契约。
//
// 为什么单独一条：上面那条只验证「空租户返回全 0 且键是 camelCase」，
// 空租户场景下 assessable/duplicateGroups 恒为 0/空——即使实现完全没做属性级
// 完整率（沿用旧 owner 口径），那条测试也照样绿。必须造出「有 CI、有必填定义、
// 有重复」的真实数据，断言才能区分新旧实现。
//
// 数据面刻意造**无云引用**的 CI：锁住「手工 CI 必须进治理基数」——
// 旧实现共用对账基数（只含带云引用的 CI）时它们根本不会出现在看板里。
func TestCMDBGovernanceReportRouteQualityContract(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_cmdb_gov_quality_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 真实 CIAttributeValidator（包内不可导出），用同语义的窄实现注入：
	// 路由测试不经过 bootstrap，Service 的 ciRequirements 缺省为 nil，
	// 不注入就永远测不出「属性级完整率」。
	reqReader := &govReqReader{client: client}
	cmdbSvc := cmdbHandler.NewService(cmdbHandler.NewEntRepository(client), nil, logger).
		WithCIRequirementReader(reqReader)
	SetupRoutes(r, &RouterConfig{
		JWTSecret:   cmdbListSecret,
		Logger:      logger,
		Client:      client,
		CMDBHandler: cmdbHandler.NewHandler(cmdbSvc),
	})

	tenant := seedGovernanceTenant(ctx, t, client, "gov-quality")

	// 类型 A：配必填属性定义 serial_number + vendor（走真实 ent 表，不用 stub，
	// 这样顺带验证 CIRequirementReader 的真实链路）。
	tA, err := client.CIType.Create().
		SetName("cmdb_ci_server").SetDescription("服务器").SetIcon("server").
		SetColor("#28a745").SetAttributeSchema("").SetTenantID(tenant.tenantID).
		Save(ctx)
	require.NoError(t, err)
	for _, def := range []struct{ name, display string }{
		{"serial_number", "序列号"}, {"vendor", "厂商"},
	} {
		_, err := client.CIAttributeDefinition.Create().
			SetCiTypeID(tA.ID).SetName(def.name).SetDisplayName(def.display).
			SetType("string").SetRequired(true).SetTenantID(tenant.tenantID).
			Save(ctx)
		require.NoError(t, err, "建属性定义 %s", def.name)
	}
	// 类型 B：刻意不配任何必填定义 → 锁「无可评估 ≠ 满分」。
	tB, err := client.CIType.Create().
		SetName("cmdb_ci_free").SetDescription("无定义类型").SetIcon("default").
		SetColor("#000000").SetAttributeSchema("").SetTenantID(tenant.tenantID).
		Save(ctx)
	require.NoError(t, err)

	// 四条 CI，全部无 cloud_resource_ref_id / 无 cloud_resource_id：
	//   CI-1 类型 A 属性齐
	//   CI-2 类型 A 缺 vendor            → incomplete
	//   CI-3 类型 A 与 CI-1 同云身份      → 重复组（见下）
	//   CI-4 类型 B                      → 不进完整率分母
	seed := func(number, name string, typeID int, typeName string, attrs map[string]interface{}, provider, cloudID string) *ent.ConfigurationItem {
		create := client.ConfigurationItem.Create().
			SetCiNumber(number).SetName(name).SetCiTypeID(typeID).SetCiType(typeName).
			SetStatus("active").SetLifecycleStatus("online").SetTenantID(tenant.tenantID)
		if attrs != nil {
			create.SetAttributes(attrs)
		}
		if provider != "" {
			create.SetCloudProvider(provider).SetCloudResourceID(cloudID)
		}
		e, err := create.Save(ctx)
		require.NoError(t, err)
		return e
	}
	ci1 := seed("CI-GOV-1", "gov-host-1", tA.ID, tA.Name,
		map[string]interface{}{"serial_number": "SN-GOV-1", "vendor": "Dell"}, "aliyun", "i-gov-1")
	_ = seed("CI-GOV-2", "gov-host-2", tA.ID, tA.Name,
		map[string]interface{}{"serial_number": "SN-GOV-2"}, "aliyun", "i-gov-2")
	// CI-3 与 CI-1 同 (provider, cloud_resource_id)：真实的重复建账场景
	ci3 := seed("CI-GOV-3", "gov-host-3", tA.ID, tA.Name,
		map[string]interface{}{"serial_number": "SN-GOV-3", "vendor": "HP"}, "aliyun", "i-gov-1")
	seed("CI-GOV-4", "gov-free-4", tB.ID, tB.Name, nil, "", "")

	token, err := middleware.GenerateAccessToken(tenant.adminID, "user", "super_admin", tenant.tenantID, cmdbListSecret, time.Hour)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/governance/report", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var env governanceEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.Equal(t, 0, env.Code, "body=%s", w.Body.String())
	body := w.Body.String()

	// 🔴 治理基数含无云引用的 CI：4 条全计入。旧实现用对账基数时这里会是 0。
	assert.Contains(t, body, `"cis":4`, "治理基数应含无云引用的 CI: %s", body)
	assert.Contains(t, body, `"total":4`)
	// 属性级完整率：类型 A 的 3 条进分母，CI-2 缺 vendor → 66.67%
	assert.Contains(t, body, `"assessable":3`, "只有配了必填定义的类型 A 进分母: %s", body)
	assert.Contains(t, body, `"incomplete":1`)
	assert.Contains(t, body, `"completenessPct":66.67`)
	assert.Contains(t, body, `"attributeDefinedTypes":1`)
	// 重复组：CI-1 与 CI-3 同云身份
	assert.Contains(t, body, `"duplicateCiCount":2`)
	assert.Contains(t, body, `"reason":"cloud_identity"`)
	assert.NotContains(t, body, `"duplicateGroups":null`, "空/非空都必须给数组，不能是 null")
	// 缺失统计指向 vendor
	assert.Contains(t, body, `"attribute":"vendor"`)
	// 重复组必须下发成员而非只有计数
	// 只有重复组成员应出现在 duplicateGroups 里；CI-2 不重复，不该出现
	for _, ci := range []*ent.ConfigurationItem{ci1, ci3} {
		require.NotNil(t, ci.CiNumber, "种子数据必须带 ci_number")
		assert.Contains(t, body, *ci.CiNumber, "重复组成员应下发 CI number")
	}
}

// govReqReader 是 cmdb.CIRequirementReader 的测试实现：从真实
// ci_attribute_definitions 表按 ciTypeID 读必填属性名。
//
// 为什么不在 cmdb 包内导出一个构造函数：真实实现是
// *service.ConfigurationItemService（转发 CIAttributeValidator），
// 它需要整个 services 依赖树，路由测试不该背这个。这里直接查表，
// 语义与 CIAttributeValidator.RequiredAttributes 的「按类型取必填名」一致；
// 继承链展开不在本用例范围内（那是 validator 自己的单测职责）。
type govReqReader struct {
	client *ent.Client
}

func (r *govReqReader) ListRequiredCIAttributes(ctx context.Context, tenantID, ciTypeID int) ([]dto.CIAttributeRequirement, error) {
	defs, err := r.client.CIAttributeDefinition.Query().
		Where(
			ciattributedefinition.CiTypeID(ciTypeID),
			ciattributedefinition.TenantID(tenantID),
			ciattributedefinition.IsActive(true),
			ciattributedefinition.Required(true),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.CIAttributeRequirement, 0, len(defs))
	for _, d := range defs {
		out = append(out, dto.CIAttributeRequirement{Name: d.Name, DisplayName: d.DisplayName})
	}
	return out, nil
}
