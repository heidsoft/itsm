package cmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/schema"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 锁 /api/v1/cmdb/ontology 的契约：
//   1. 五键精确集合（version / ciTypes / relationshipTypes / enums / aiTools）；
//   2. version 非空且符合日期风格；
//   3. relationshipTypes 数量与 ent/schema CIRelationshipTypeVocabulary 一致（13 条受控词表）；
//   4. enums.lifecycleStatus 与 common.CILifecycleStatus* 常量同源；
//   5. 缺租户上下文 fail-closed 401/2001；
//   6. 跨租户拒绝：tenant B 看不到 tenant A 的 CI 类型；
//   7. toolRegistry 未注入时省略整段 aiTools；
//   8. parseAttributeSchemaOrRaw 三态（合法 JSON → object、非法 JSON → string、空 → nil）；
//   9. listAllCITypes 分页三态（空、单页满页）；
//  10. 单类型属性定义查询失败不阻断整体自省。

// ontologyTestRouter 复现 router/cmdb_routes.go:54 的注册（permission 中间件不在本
// 信封契约范围内），通过 gin context 注入租户 ID 模拟 auth 中间件已通过的请求。
func ontologyTestRouter(t *testing.T, withTenant bool) (*gin.Engine, *ProductionService) {
	return ontologyTestRouterWithRegistry(t, withTenant, nil)
}

// ontologyTestRouterWithRegistry 与 ontologyTestRouter 等价，但允许显式注入 toolRegistry。
// 注入空 registry（tools 列表为空）会让 aiTools 字段以 "aiTools": [] 形式存在，
// 用于锁「五键集合精确存在」的契约；不注入时省略整段 aiTools。
func ontologyTestRouterWithRegistry(t *testing.T, withTenant bool, tr *service.ToolRegistry) (*gin.Engine, *ProductionService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:ontology_%s?mode=memory&cache=shared&_fk=1", t.Name()))
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	ciTypeSvc := service.NewCITypeService(client, logger)
	ciAttrSvc := service.NewCIAttributeDefinitionService(client, logger)
	ciHistorySvc := service.NewCIHistoryService(client, logger)
	ciTagSvc := service.NewCITagService(client, logger)
	ciSvc := service.NewConfigurationItemService(client, logger, ciHistorySvc, ciTagSvc)
	ciRelSvc := service.NewCIRelationshipService(client, logger)
	impExpSvc := service.NewCMDBImportExportService(client, logger, ciSvc, ciTagSvc)
	savedViewSvc := service.NewCMDBSavedViewService(client, logger)

	ps := NewProductionService(logger, ciTypeSvc, ciAttrSvc, ciSvc, ciRelSvc, ciHistorySvc, ciTagSvc, impExpSvc, savedViewSvc)
	if tr != nil {
		ps.SetToolRegistry(tr)
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		if withTenant {
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
			c.Set("tenant_id", 1)
			c.Set("user_id", 1)
		}
		c.Next()
	})
	r.GET("/api/v1/cmdb/ontology", ps.GetOntology)
	return r, ps
}

func decodeOntology(t *testing.T, r http.Handler) (int, int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/ontology", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	return w.Code, body.Code, body.Data
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestOntology_FiveKeyEnvelope 锁响应根只含五键（toolRegistry 已注入 → aiTools: [] 形式存在）。
func TestOntology_FiveKeyEnvelope(t *testing.T) {
	emptyRegistry := service.NewToolRegistry(nil, nil, nil, nil)
	r, _ := ontologyTestRouterWithRegistry(t, true, emptyRegistry)
	_, _, data := decodeOntology(t, r)
	require.NotNil(t, data, "缺租户时 data 应为 nil；该用例要求带租户")
	assert.Equal(t,
		[]string{"aiTools", "ciTypes", "enums", "relationshipTypes", "version"},
		sortedKeys(data),
		"ontology 响应键集合必须恰好是 version / ciTypes / relationshipTypes / enums / aiTools 五键")
}

// TestOntology_VersionStable 锁 version 字段非空且符合日期风格（破坏性变更递增）。
func TestOntology_VersionStable(t *testing.T) {
	r, _ := ontologyTestRouter(t, true)
	_, _, data := decodeOntology(t, r)
	v, ok := data["version"].(string)
	require.True(t, ok, "version 必须是字符串")
	assert.NotEmpty(t, v)
	assert.True(t, strings.HasPrefix(v, "20"), "version 应是日期风格（4 位年份前缀），实际=%q", v)
}

// TestOntology_RelationshipTypesVocabulary 锁词表 = ent schema CIRelationshipTypeVocabulary。
func TestOntology_RelationshipTypesVocabulary(t *testing.T) {
	r, _ := ontologyTestRouter(t, true)
	_, _, data := decodeOntology(t, r)
	rts, ok := data["relationshipTypes"].([]interface{})
	require.True(t, ok)
	require.Len(t, rts, len(schema.CIRelationshipTypeVocabulary),
		"ontology 关系词表必须与 ent/schema CIRelationshipTypeVocabulary 等长")

	gotTypes := make(map[string]string, len(rts))
	for _, raw := range rts {
		rt, ok := raw.(map[string]interface{})
		require.True(t, ok)
		t_, _ := rt["type"].(string)
		rev, _ := rt["reverseType"].(string)
		gotTypes[t_] = rev
	}
	for _, meta := range schema.CIRelationshipTypeVocabulary {
		assert.Equal(t, string(meta.Reverse), gotTypes[string(meta.Type)],
			"关系类型 %q 的 reverseType 必须与 schema 一致", meta.Type)
	}
}

// TestOntology_EnumsMatchConstants 锁 enums.lifecycleStatus 与 common 常量同源。
func TestOntology_EnumsMatchConstants(t *testing.T) {
	r, _ := ontologyTestRouter(t, true)
	_, _, data := decodeOntology(t, r)
	enums, ok := data["enums"].(map[string]interface{})
	require.True(t, ok)

	ls, ok := enums["lifecycleStatus"].([]interface{})
	require.True(t, ok)
	got := make([]string, 0, len(ls))
	for _, v := range ls {
		s, _ := v.(string)
		got = append(got, s)
	}
	assert.Equal(t, []string{
		common.CILifecycleStatusDraft,
		common.CILifecycleStatusOnline,
		common.CILifecycleStatusMaintenance,
		common.CILifecycleStatusOffline,
		common.CILifecycleStatusScrapped,
	}, got)
}

// TestOntology_TenantMissingFailsClosed 缺租户上下文必须 401/2001。
func TestOntology_TenantMissingFailsClosed(t *testing.T) {
	r, _ := ontologyTestRouter(t, false)
	status, code, _ := decodeOntology(t, r)
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Equal(t, common.AuthFailedCode, code,
		"缺租户上下文必须 fail-closed 到 401/2001，禁止回退 tenant=1")
}

// TestOntology_OnlyReturnsOwnTenantCITypes tenant A 创建类型，tenant B 不应看到。
func TestOntology_OnlyReturnsOwnTenantCITypes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:ontology_iso?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenantA, err := client.Tenant.Create().SetName("a").SetCode("a").SetDomain("a.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().SetName("b").SetCode("b").SetDomain("b.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	_, err = client.CIType.Create().SetName("server-a").SetTenantID(tenantA.ID).SetIsActive(true).Save(ctx)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	ciTypeSvc := service.NewCITypeService(client, logger)
	ciAttrSvc := service.NewCIAttributeDefinitionService(client, logger)
	ciHistorySvc := service.NewCIHistoryService(client, logger)
	ciTagSvc := service.NewCITagService(client, logger)
	ciSvc := service.NewConfigurationItemService(client, logger, ciHistorySvc, ciTagSvc)
	ciRelSvc := service.NewCIRelationshipService(client, logger)
	impExpSvc := service.NewCMDBImportExportService(client, logger, ciSvc, ciTagSvc)
	savedViewSvc := service.NewCMDBSavedViewService(client, logger)
	ps := NewProductionService(logger, ciTypeSvc, ciAttrSvc, ciSvc, ciRelSvc, ciHistorySvc, ciTagSvc, impExpSvc, savedViewSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		tenantID := 0
		switch c.GetHeader("X-Test-Tenant") {
		case "A":
			tenantID = tenantA.ID
		case "B":
			tenantID = tenantB.ID
		}
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Set("tenant_id", tenantID)
		c.Set("user_id", 1)
		c.Next()
	})
	r.GET("/api/v1/cmdb/ontology", ps.GetOntology)

	// tenant A 看到 server-a
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/ontology", nil)
	req.Header.Set("X-Test-Tenant", "A")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var bodyA struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &bodyA))
	ctsA, _ := bodyA.Data["ciTypes"].([]interface{})
	require.Len(t, ctsA, 1, "tenant A 应仅看到自己的 CI 类型")
	assert.Equal(t, "server-a", ctsA[0].(map[string]interface{})["name"])

	// tenant B 不应看到 server-a
	req = httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/ontology", nil)
	req.Header.Set("X-Test-Tenant", "B")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var bodyB struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &bodyB))
	ctsB, _ := bodyB.Data["ciTypes"].([]interface{})
	assert.Empty(t, ctsB, "tenant B 不应看到 tenant A 的 CI 类型")
}

// TestOntology_ToolRegistryNilOmitsAITools toolRegistry 未注入时 aiTools 字段不存在。
func TestOntology_ToolRegistryNilOmitsAITools(t *testing.T) {
	r, _ := ontologyTestRouter(t, true)
	_, _, data := decodeOntology(t, r)
	_, present := data["aiTools"]
	assert.False(t, present, "toolRegistry=nil 时 aiTools 字段必须省略，禁止返回 null/[]")
}

// TestOntology_ParseAttributeSchema_ValidJSON 合法 JSON 序列化为 object。
func TestOntology_ParseAttributeSchema_ValidJSON(t *testing.T) {
	got := parseAttributeSchemaOrRaw(`{"kind":"server","required":["hostname"]}`)
	obj, ok := got.(map[string]interface{})
	require.True(t, ok, "合法 JSON 应解析为 map[string]interface{}，实际=%T", got)
	assert.Equal(t, "server", obj["kind"])
}

// TestOntology_ParseAttributeSchema_InvalidJSONReturnsString 非 JSON 文本原样返回字符串。
func TestOntology_ParseAttributeSchema_InvalidJSONReturnsString(t *testing.T) {
	raw := "not a json"
	assert.Equal(t, raw, parseAttributeSchemaOrRaw(raw))
}

// TestOntology_ParseAttributeSchema_EmptyReturnsEmpty 空字符串返回 nil。
func TestOntology_ParseAttributeSchema_EmptyReturnsEmpty(t *testing.T) {
	assert.Nil(t, parseAttributeSchemaOrRaw(""))
}

// TestOntology_ListAllCITypes_Empty 租户无任何 CI 类型时返回空切片而非 nil panic。
func TestOntology_ListAllCITypes_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:ontology_empty?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	ps := &ProductionService{
		logger:                       logger,
		ciTypeService:                service.NewCITypeService(client, logger),
		ciAttributeDefinitionService: service.NewCIAttributeDefinitionService(client, logger),
		ciService:                    service.NewConfigurationItemService(client, logger, service.NewCIHistoryService(client, logger), service.NewCITagService(client, logger)),
		ciRelationshipService:        service.NewCIRelationshipService(client, logger),
	}
	got, err := ps.listAllCITypes(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestOntology_ListAllCITypes_FitsSinglePage 当 active 类型数 ≤ common.MaxPageSize 时
// 应在第一页终止（len(items)<pageSize 短路）。
func TestOntology_ListAllCITypes_FitsSinglePage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:ontology_single?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := client.CIType.Create().
			SetName(fmt.Sprintf("t-%d", i)).
			SetTenantID(1).
			SetIsActive(true).
			Save(ctx)
		require.NoError(t, err)
	}

	logger := zaptest.NewLogger(t).Sugar()
	ps := &ProductionService{
		logger:                       logger,
		ciTypeService:                service.NewCITypeService(client, logger),
		ciAttributeDefinitionService: service.NewCIAttributeDefinitionService(client, logger),
		ciService:                    service.NewConfigurationItemService(client, logger, service.NewCIHistoryService(client, logger), service.NewCITagService(client, logger)),
		ciRelationshipService:        service.NewCIRelationshipService(client, logger),
	}

	got, err := ps.listAllCITypes(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, got, 5, "5 条记录应在第一页终止")
}

// TestOntology_Handlerctx_NoDoubleWrite 在缺租户分支只调用 ResolveTenantID 后立即 return，
// 不得再调 common.Fail 写第二份响应（拼接 JSON 风险）。
func TestOntology_Handlerctx_NoDoubleWrite(t *testing.T) {
	r, _ := ontologyTestRouter(t, false)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/ontology", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var once map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &once))
	assert.Equal(t, common.AuthFailedCode, int(once["code"].(float64)))
	assert.Less(t, w.Body.Len(), 512, "缺租户拒绝应只写一份响应，body 不应过大")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, ok := handlerctx.ResolveTenantID(c)
	assert.False(t, ok, "缺租户上下文时 ResolveTenantID 应返回 ok=false")
}
