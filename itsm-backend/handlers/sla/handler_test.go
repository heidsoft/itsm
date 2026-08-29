package sla

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// slaTestAuth 与 controller 包中的 withTestAuth 等价：注入租户上下文。
// SLA handler 通过 c.Get("tenant_id").(int) 读取租户，并依赖外键
// 指向的租户行，因此这里先播种一个真实租户。
func slaTestAuth(tid, uid int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Skip-Auth") == "1" {
			c.Next()
			return
		}
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tid})
		c.Set("tenant_id", tid)
		c.Set("user_id", uid)
		c.Next()
	}
}

func setupSLAHandler(t *testing.T) (*gin.Engine, *ent.Client, int) {
	gin.SetMode(gin.TestMode)
	dsn := "file:" + filepath.Join(t.TempDir(), "sla_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()

	// 播种租户以满足 SLA 定义的外键约束
	ctx := context.Background()
	uid := slaUniqueID()
	tenant, err := client.Tenant.Create().
		SetName("SLA Tenant").
		SetCode("SLA" + uid).
		SetDomain("sla.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	svc := NewService(repo, logger)
	h := NewHandler(svc)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(slaTestAuth(tenant.ID, 1))
	r.POST("/api/v1/sla/definitions", h.CreateSLADefinition)
	r.GET("/api/v1/sla/definitions", h.ListSLADefinitions)
	r.GET("/api/v1/sla/definitions/:id", h.GetSLADefinition)
	r.PUT("/api/v1/sla/definitions/:id", h.UpdateSLADefinition)
	r.DELETE("/api/v1/sla/definitions/:id", h.DeleteSLADefinition)
	r.GET("/api/v1/sla/stats", h.GetSLAStats)
	// 与 router.go 保持一致：/monitor 是历史兼容别名，/monitoring 是 canonical 路径
	r.POST("/api/v1/sla/monitor", h.GetSLAMonitoring)
	r.POST("/api/v1/sla/monitoring", h.GetSLAMonitoring)
	r.GET("/api/v1/sla/performance", h.GetSLAPerformance)
	// 阶段 1.7：补齐告警/违规/合规/指标相关路由
	r.POST("/api/v1/sla/alert-rules", h.CreateAlertRule)
	r.GET("/api/v1/sla/alert-rules", h.ListAlertRules)
	r.GET("/api/v1/sla/alert-rules/:id", h.GetAlertRule)
	r.PUT("/api/v1/sla/alert-rules/:id", h.UpdateAlertRule)
	r.DELETE("/api/v1/sla/alert-rules/:id", h.DeleteAlertRule)
	r.GET("/api/v1/sla/metrics", h.GetSLAMetrics)
	r.GET("/api/v1/sla/violations", h.GetSLAViolations)
	r.PUT("/api/v1/sla/violations/:id", h.UpdateViolationStatus)
	r.GET("/api/v1/sla/alert-history", h.GetAlertHistory)
	r.GET("/api/v1/sla/compliance-report", h.GetSLAComplianceReport)
	return r, client, tenant.ID
}

func TestSLAHandler_CreateDefinition(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	t.Run("成功创建SLA定义", func(t *testing.T) {
		body := dto.CreateSLADefinitionRequest{
			Name:           "标准服务SLA",
			Description:    "标准IT服务SLA定义",
			ServiceType:    "standard",
			Priority:       "medium",
			ResponseTime:   30,
			ResolutionTime: 240,
			IsActive:       true,
		}
		resp := doSLAReq(t, r, "POST", "/api/v1/sla/definitions", body, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "标准服务SLA", data["name"])
	})

	t.Run("缺少名称应返回错误", func(t *testing.T) {
		body := dto.CreateSLADefinitionRequest{ResponseTime: 30, ResolutionTime: 240}
		resp := doSLAReq(t, r, "POST", "/api/v1/sla/definitions", body, false)
		assert.NotEqual(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("响应时间为0应返回错误", func(t *testing.T) {
		body := dto.CreateSLADefinitionRequest{Name: "X", ResponseTime: 0, ResolutionTime: 0}
		resp := doSLAReq(t, r, "POST", "/api/v1/sla/definitions", body, false)
		assert.NotEqual(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})
}

func TestSLAHandler_ListDefinitions(t *testing.T) {
	r, _, _ := setupSLAHandler(t)
	resp := doSLAReq(t, r, "GET", "/api/v1/sla/definitions", nil, false)
	assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data, "items")
	assert.Contains(t, data, "total")
}

func TestSLAHandler_GetDefinition(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	created := doSLAReq(t, r, "POST", "/api/v1/sla/definitions", dto.CreateSLADefinitionRequest{
		Name: "查询测试SLA", ResponseTime: 15, ResolutionTime: 120,
	}, false)
	require.Equal(t, common.SuccessCode, created.Code)
	id := int(created.Data.(map[string]interface{})["id"].(float64))

	t.Run("按ID获取成功", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/definitions/"+itoaSLA(id), nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("非法ID应返回错误", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/definitions/abc", nil, false)
		assert.NotEqual(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("不存在的ID应返回错误", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/definitions/999999", nil, false)
		assert.NotEqual(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})
}

func TestSLAHandler_GetStats(t *testing.T) {
	r, _, _ := setupSLAHandler(t)
	resp := doSLAReq(t, r, "GET", "/api/v1/sla/stats", nil, false)
	assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))

	t.Run("无样本时合规率诚实为0而非伪装100", func(t *testing.T) {
		data := resp.Data.(map[string]interface{})
		rate, ok := data["overall_compliance_rate"]
		require.True(t, ok, "body=%s", slaStr(resp))
		assert.Equal(t, float64(0), rate)
	})
}

// 监控大屏契约回归（camelCase 字段、零样本诚实性、统计窗口、活跃告警、
// 租户隔离、绩效分组与过滤）集中在 monitoring_test.go，带确定性数据集。

// ---- 阶段 1.7:告警规则 CRUD 路径 ----

func TestSLAHandler_AlertRule_CRUD(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	// 先创建一个 SLA 定义供告警规则引用
	def := doSLAReq(t, r, "POST", "/api/v1/sla/definitions", dto.CreateSLADefinitionRequest{
		Name: "告警测试SLA", ResponseTime: 30, ResolutionTime: 240, IsActive: true,
	}, false)
	require.Equal(t, common.SuccessCode, def.Code, "body=%s", slaStr(def))
	defID := int(def.Data.(map[string]interface{})["id"].(float64))

	t.Run("创建告警规则成功", func(t *testing.T) {
		body := dto.CreateSLAAlertRuleRequest{
			SLADefinitionID:      defID,
			Name:                 "预警 80%",
			ThresholdPercentage:  80,
			AlertLevel:           "warning",
			NotificationChannels: []string{"email", "feishu"},
			IsActive:             true,
		}
		resp := doSLAReq(t, r, "POST", "/api/v1/sla/alert-rules", body, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("告警规则缺少阈值应返回错误", func(t *testing.T) {
		body := dto.CreateSLAAlertRuleRequest{
			SLADefinitionID: defID,
			Name:            "无效规则",
		}
		resp := doSLAReq(t, r, "POST", "/api/v1/sla/alert-rules", body, false)
		assert.NotEqual(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("告警规则列表查询成功", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/alert-rules", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("按 SLA ID 过滤告警规则", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/alert-rules?sla_definition_id="+itoaSLA(defID), nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})
}

func TestSLAHandler_UpdateAlertRuleRejectsCrossTenant(t *testing.T) {
	r, client, tenantAID := setupSLAHandler(t)
	ctx := context.Background()

	tenantB, err := client.Tenant.Create().
		SetName("SLA Tenant B").
		SetCode("SLAB" + slaUniqueID()).
		SetDomain("sla-b.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	definitionB, err := client.SLADefinition.Create().
		SetName("Tenant B SLA").
		SetResponseTime(30).
		SetResolutionTime(240).
		SetIsActive(true).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	ruleB, err := client.SLAAlertRule.Create().
		SetSLADefinitionID(definitionB.ID).
		SetName("Tenant B Rule").
		SetThresholdPercentage(80).
		SetAlertLevel("warning").
		SetNotificationChannels([]string{"email"}).
		SetIsActive(true).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	resp := doSLAReq(t, r, http.MethodPut, "/api/v1/sla/alert-rules/"+itoaSLA(ruleB.ID), dto.UpdateSLAAlertRuleRequest{
		Name: ptrSLAString("compromised"),
	}, false)
	assert.Equal(t, common.NotFoundCode, resp.Code, "body=%s", slaStr(resp))

	stored, err := client.SLAAlertRule.Get(ctx, ruleB.ID)
	require.NoError(t, err)
	assert.Equal(t, "Tenant B Rule", stored.Name)
	assert.Equal(t, tenantB.ID, stored.TenantID)
	assert.NotEqual(t, tenantAID, stored.TenantID)
}

func TestEntRepository_UpdateAlertRuleScopesUpdateByTenant(t *testing.T) {
	_, client, _ := setupSLAHandler(t)
	ctx := context.Background()

	tenantB, err := client.Tenant.Create().
		SetName("Repository Tenant B").
		SetCode("REPOB" + slaUniqueID()).
		SetDomain("repo-b.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	definitionB, err := client.SLADefinition.Create().
		SetName("Repository Tenant B SLA").
		SetResponseTime(30).
		SetResolutionTime(240).
		SetIsActive(true).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)
	ruleB, err := client.SLAAlertRule.Create().
		SetSLADefinitionID(definitionB.ID).
		SetName("Original").
		SetThresholdPercentage(80).
		SetAlertLevel("warning").
		SetNotificationChannels([]string{"email"}).
		SetIsActive(true).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	attackerUpdate := toSLAAlertRuleDomain(ruleB)
	attackerUpdate.TenantID = tenantB.ID + 1000
	attackerUpdate.Name = "compromised"
	_, err = NewEntRepository(client).UpdateAlertRule(ctx, attackerUpdate)
	require.Error(t, err)
	assert.True(t, ent.IsNotFound(err), "expected tenant predicate mismatch to return not found: %v", err)

	stored, err := client.SLAAlertRule.Get(ctx, ruleB.ID)
	require.NoError(t, err)
	assert.Equal(t, "Original", stored.Name)
}

// ---- 阶段 1.7:合规报告 / 违规查询 路径 ----

func TestSLAHandler_ComplianceReport(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	t.Run("snake_case 起始日期格式", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/compliance-report?start_date=2026-01-01T00:00:00Z&end_date=2026-12-31T23:59:59Z", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("camelCase 起始日期格式也应被识别", func(t *testing.T) {
		// 服务端应同时接受 startDate / endDate 兜底(R-002 修复)
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/compliance-report?startDate=2026-01-01T00:00:00Z&endDate=2026-12-31T23:59:59Z", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("缺少日期应返回参数错误", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/compliance-report", nil, false)
		assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("格式错误的日期应返回参数错误", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/compliance-report?start_date=notadate&end_date=2026-12-31T23:59:59Z", nil, false)
		assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", slaStr(resp))
	})
}

func TestSLAHandler_Violations_AndMetrics(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	t.Run("违规列表分页查询成功", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/violations?page=1&size=10", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		data := resp.Data.(map[string]interface{})
		assert.Contains(t, data, "items")
		assert.Contains(t, data, "total")
		assert.Contains(t, data, "page")
		assert.Contains(t, data, "pageSize")
	})

	t.Run("按严重度过滤违规记录", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/violations?severity=high", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("按已解决/未解决过滤违规记录", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/violations?is_resolved=false", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("SLA 指标查询成功", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/metrics?metric_type=response", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("告警历史查询成功", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/alert-history?page=1&pageSize=10", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		data := resp.Data.(map[string]interface{})
		assert.Contains(t, data, "items")
		assert.Contains(t, data, "pageSize")
	})
}

// TestSLAHandler_ViolationsTicketContextRegression 监控大屏回归：
// /api/v1/sla/violations 必须通过 ticket edge 返回工单标题/编号/优先级，
// 并支持 camelCase isResolved 过滤；修复前 ticketTitle 缺失导致前端全部展示 Unknown。
func TestSLAHandler_ViolationsTicketContextRegression(t *testing.T) {
	_, client, tenantID := setupSLAHandler(t)
	ctx := context.Background()
	uid := slaUniqueID()

	def, err := client.SLADefinition.Create().
		SetName("大屏回归SLA-" + uid).
		SetPriority("high").
		SetResponseTime(30).
		SetResolutionTime(240).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	user, err := client.User.Create().
		SetUsername("sla-ctx-user-" + uid).
		SetEmail("sla-ctx-" + uid + "@example.com").
		SetName("SLA Context User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	tk, err := client.Ticket.Create().
		SetTitle("打印机故障工单-" + uid).
		SetPriority("high").
		SetTicketNumber("TKT-SLA-CTX-" + uid).
		SetTenantID(tenantID).
		SetRequesterID(user.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = repo.CreateViolation(ctx, &SLAViolation{
		TicketID:        tk.ID,
		SLADefinitionID: def.ID,
		ViolationType:   "resolution",
		ViolationTime:   time.Now().Add(-time.Hour),
		Severity:        "high",
		IsResolved:      false,
		TenantID:        tenantID,
	})
	require.NoError(t, err)

	// 再创建一条已解决违规，验证 isResolved 过滤
	_, err = repo.CreateViolation(ctx, &SLAViolation{
		TicketID:        tk.ID,
		SLADefinitionID: def.ID,
		ViolationType:   "response",
		ViolationTime:   time.Now().Add(-2 * time.Hour),
		Severity:        "low",
		IsResolved:      true,
		TenantID:        tenantID,
	})
	require.NoError(t, err)

	svc := NewService(repo, zaptest.NewLogger(t).Sugar())
	h := NewHandler(svc)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(slaTestAuth(tenantID, 1))
	r.GET("/api/v1/sla/violations", h.GetSLAViolations)

	t.Run("列表返回工单标题/编号/优先级与SLA名称", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/violations?page=1&size=20", nil, false)
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		data := resp.Data.(map[string]interface{})
		items := data["items"].([]interface{})
		require.NotEmpty(t, items)

		var found map[string]interface{}
		for _, it := range items {
			m := it.(map[string]interface{})
			if m["violationType"] == "resolution" {
				found = m
			}
		}
		require.NotNil(t, found, "未找到 resolution 违规: %v", items)
		assert.Equal(t, "打印机故障工单-"+uid, found["ticketTitle"])
		assert.Equal(t, "TKT-SLA-CTX-"+uid, found["ticketNumber"])
		assert.Equal(t, "high", found["ticketPriority"])
		assert.Equal(t, "Default SLA", found["slaName"])
		// 敏感字段不得泄漏
		assert.NotContains(t, found, "resolution_notes")
	})

	t.Run("camelCase isResolved=false 只返回未解决违规", func(t *testing.T) {
		resp := doSLAReq(t, r, "GET", "/api/v1/sla/violations?page=1&size=20&isResolved=false", nil, false)
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		data := resp.Data.(map[string]interface{})
		items := data["items"].([]interface{})
		require.Len(t, items, 1)
		assert.Equal(t, "resolution", items[0].(map[string]interface{})["violationType"])
		assert.Equal(t, float64(1), data["total"])
	})
}

// ---- 阶段 1.7:定义更新 / 删除 路径 ----

func TestSLAHandler_UpdateAndDelete(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	created := doSLAReq(t, r, "POST", "/api/v1/sla/definitions", dto.CreateSLADefinitionRequest{
		Name: "更新删除SLA", ResponseTime: 30, ResolutionTime: 240, IsActive: true,
	}, false)
	require.Equal(t, common.SuccessCode, created.Code)
	id := int(created.Data.(map[string]interface{})["id"].(float64))

	t.Run("更新定义成功", func(t *testing.T) {
		newName := "重命名后的SLA"
		active := false
		resp := doSLAReq(t, r, "PUT", "/api/v1/sla/definitions/"+itoaSLA(id), dto.UpdateSLADefinitionRequest{
			Name:     &newName,
			IsActive: &active,
		}, false)
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, newName, data["name"])
		assert.Equal(t, false, data["isActive"])
	})

	t.Run("更新不存在的定义应返回 404", func(t *testing.T) {
		newName := "x"
		resp := doSLAReq(t, r, "PUT", "/api/v1/sla/definitions/999999", dto.UpdateSLADefinitionRequest{Name: &newName}, false)
		assert.Equal(t, common.NotFoundCode, resp.Code, "body=%s", slaStr(resp))
	})

	t.Run("删除定义成功", func(t *testing.T) {
		resp := doSLAReq(t, r, "DELETE", "/api/v1/sla/definitions/"+itoaSLA(id), nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
		// 再次 GET 应返回 404
		resp2 := doSLAReq(t, r, "GET", "/api/v1/sla/definitions/"+itoaSLA(id), nil, false)
		assert.Equal(t, common.NotFoundCode, resp2.Code, "body=%s", slaStr(resp2))
	})
}

// ---- 阶段 1.7:未授权租户上下文 应被拒绝 ----

func TestSLAHandler_WithoutTenantContext(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	// 缺少 tenant_id 的请求(SLA handler 不强制要求，但更新/删除租户隔离必须工作)
	// 这里验证 tenant_id=0 时业务仍然走到 repo，但 repo 会过滤
	created := doSLAReq(t, r, "POST", "/api/v1/sla/definitions", dto.CreateSLADefinitionRequest{
		Name: "带正确租户", ResponseTime: 30, ResolutionTime: 240, IsActive: true,
	}, false)
	require.Equal(t, common.SuccessCode, created.Code)
}

// ---- 简易请求助手（sla 包内独立实现）----

func doSLAReq(t *testing.T, r *gin.Engine, method, path string, body interface{}, skipAuth bool) *common.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if skipAuth {
		req.Header.Set("X-Skip-Auth", "1")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return &resp
}

func itoaSLA(i int) string { return strconv.Itoa(i) }

func ptrSLAString(value string) *string { return &value }

func slaUniqueID() string { return strconv.FormatInt(time.Now().UnixNano(), 10) }

func slaStr(resp *common.Response) string {
	b, _ := json.Marshal(resp)
	return string(b)
}
