package sla_template

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type slaTemplateItem struct {
	Key              string          `json:"key"`
	Name             string          `json:"name"`
	Priority         string          `json:"priority"`
	Industry         string          `json:"industry"`
	Recommended      bool            `json:"recommended"`
	Installed        bool            `json:"installed"`
	SLADefinitionID  int             `json:"slaDefinitionId"`
	InstalledAt      *time.Time      `json:"installedAt"`
	BusinessHours    json.RawMessage `json:"businessHours"`
	EscalationLevels json.RawMessage `json:"escalationLevels"`
	Conditions       json.RawMessage `json:"conditions"`
}

type slaTemplateListResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []slaTemplateItem `json:"items"`
		Total int               `json:"total"`
	} `json:"data"`
}

type slaTemplateTest struct {
	router     *gin.Engine
	client     *ent.Client
	tenantA    int
	tenantB    int
	catalogKey string
}

// setupSLATemplateHandler 挂真实 handler 与真实库：安装状态要经 HTTP 验证，
// 只测 service 会漏掉租户解析和 DTO 投影这两步。
func setupSLATemplateHandler(t *testing.T) *slaTemplateTest {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	ctx := context.Background()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	seedTenant := func(name, code string) int {
		tenant, err := client.Tenant.Create().
			SetName(name).
			SetCode(code + suffix).
			SetDomain(code + suffix + ".com").
			SetStatus("active").
			Save(ctx)
		require.NoError(t, err)
		return tenant.ID
	}
	tenantA := seedTenant("SLA Template Tenant A", "sla-tpl-a")
	tenantB := seedTenant("SLA Template Tenant B", "sla-tpl-b")

	h := NewHandler(service.NewSLATemplateService(client, zaptest.NewLogger(t).Sugar()))

	r := gin.New()
	r.Use(gin.Recovery())
	// 用请求头选择租户；不带该头即无租户上下文，用于验证 fail-closed。
	r.Use(func(c *gin.Context) {
		if raw := c.GetHeader("X-Test-Tenant"); raw != "" {
			tid, err := strconv.Atoi(raw)
			require.NoError(t, err)
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tid})
		}
		c.Next()
	})
	r.GET("/api/v1/sla/templates", h.ListTemplates)
	r.GET("/api/v1/sla/templates/:key", h.GetTemplate)
	r.POST("/api/v1/sla/templates/:key/install", h.InstallTemplate)

	return &slaTemplateTest{
		router:     r,
		client:     client,
		tenantA:    tenantA,
		tenantB:    tenantB,
		catalogKey: "incident_p1_critical",
	}
}

func (tt *slaTemplateTest) do(t *testing.T, method, path string, tenant int) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)
	if tenant > 0 {
		request.Header.Set("X-Test-Tenant", strconv.Itoa(tenant))
	}
	tt.router.ServeHTTP(recorder, request)
	return recorder
}

// definitionsInTenant 供各测试直接核对库里的安装产物数量。
func (tt *slaTemplateTest) definitionsInTenant(t *testing.T, tenantID int) int {
	t.Helper()
	count, err := tt.client.SLADefinition.Query().
		Where(sladefinition.TenantIDEQ(tenantID)).
		Count(context.Background())
	require.NoError(t, err)
	return count
}

func (tt *slaTemplateTest) list(t *testing.T, tenantID int) slaTemplateListResponse {
	t.Helper()
	recorder := tt.do(t, http.MethodGet, "/api/v1/sla/templates", tenantID)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response slaTemplateListResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestListTemplatesUsesStandardListResponse(t *testing.T) {
	tt := setupSLATemplateHandler(t)

	recorder := tt.do(t, http.MethodGet, "/api/v1/sla/templates", tt.tenantA)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response slaTemplateListResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, common.SuccessCode, response.Code)
	assert.Equal(t, "success", response.Message)
	require.Len(t, response.Data.Items, 6)
	assert.Equal(t, len(response.Data.Items), response.Data.Total)
	assert.NotContains(t, recorder.Body.String(), `"templates":`)

	// 安装状态来自数据库：新租户一律未安装，也不编造定义 ID。
	for _, item := range response.Data.Items {
		assert.False(t, item.Installed, "模板 %s 未安装却报告已安装", item.Key)
		assert.Zero(t, item.SLADefinitionID)
		assert.Nil(t, item.InstalledAt)
	}
}

// TestListTemplatesProjectsConfigShape 模板响应与已安装定义共用投影，
// 前端不会拿到 snake_case 原始 JSON。
func TestListTemplatesProjectsConfigShape(t *testing.T) {
	tt := setupSLATemplateHandler(t)

	recorder := tt.do(t, http.MethodGet, "/api/v1/sla/templates", tt.tenantA)
	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	for _, forbidden := range []string{
		"threshold_minutes", "notify_roles", `"work_days"`, `"time_zone"`,
		"is_24_7", `"start_time"`, `"business_hours"`, `"escalation_rules"`,
	} {
		assert.NotContains(t, body, forbidden)
	}

	response := tt.list(t, tt.tenantA)
	var p1 *slaTemplateItem
	for i := range response.Data.Items {
		if response.Data.Items[i].Key == tt.catalogKey {
			p1 = &response.Data.Items[i]
		}
	}
	require.NotNil(t, p1)

	var businessHours struct {
		TimeZone  string `json:"timeZone"`
		AlwaysOn  bool   `json:"alwaysOn"`
		WorkDays  []int  `json:"workDays"`
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
	}
	require.NoError(t, json.Unmarshal(p1.BusinessHours, &businessHours))
	assert.Equal(t, "Asia/Shanghai", businessHours.TimeZone)
	assert.True(t, businessHours.AlwaysOn)
	assert.Equal(t, "00:00", businessHours.StartTime)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, businessHours.WorkDays)

	var levels []struct {
		Level            int      `json:"level"`
		ThresholdMinutes int      `json:"thresholdMinutes"`
		NotifyRoles      []string `json:"notifyRoles"`
	}
	require.NoError(t, json.Unmarshal(p1.EscalationLevels, &levels))
	require.Len(t, levels, 3)
	assert.Equal(t, 15, levels[0].ThresholdMinutes)
	assert.Equal(t, []string{"manager"}, levels[0].NotifyRoles)
}

// TestInstallThenListReportsInstalled 装完立刻在列表里可见，否则页面刷新后
// 会把已装模板显示成「未安装」，用户只能反复重装。
func TestInstallThenListReportsInstalled(t *testing.T) {
	tt := setupSLATemplateHandler(t)

	recorder := tt.do(t, http.MethodPost, "/api/v1/sla/templates/"+tt.catalogKey+"/install", tt.tenantA)
	require.Equal(t, http.StatusOK, recorder.Code)
	var install struct {
		Code int `json:"code"`
		Data struct {
			TemplateKey     string `json:"templateKey"`
			SLADefinitionID int    `json:"slaDefinitionId"`
			Created         bool   `json:"created"`
			WasAlreadyExist bool   `json:"wasAlreadyExist"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &install))
	assert.Equal(t, common.SuccessCode, install.Code)
	assert.Equal(t, tt.catalogKey, install.Data.TemplateKey)
	assert.True(t, install.Data.Created)
	assert.False(t, install.Data.WasAlreadyExist)
	require.Greater(t, install.Data.SLADefinitionID, 0)

	response := tt.list(t, tt.tenantA)
	byKey := make(map[string]slaTemplateItem, len(response.Data.Items))
	for _, item := range response.Data.Items {
		byKey[item.Key] = item
	}
	installed := byKey[tt.catalogKey]
	assert.True(t, installed.Installed)
	assert.Equal(t, install.Data.SLADefinitionID, installed.SLADefinitionID)
	assert.NotNil(t, installed.InstalledAt)
	for _, item := range response.Data.Items {
		if item.Key != tt.catalogKey {
			assert.False(t, item.Installed, "只有 %s 装过，%s 不应显示已安装", tt.catalogKey, item.Key)
		}
	}
}

// TestInstallIsIdempotentThroughHTTP 重复点击不能往租户里塞第二份相同配置。
func TestInstallIsIdempotentThroughHTTP(t *testing.T) {
	tt := setupSLATemplateHandler(t)
	path := "/api/v1/sla/templates/change_normal/install"

	require.Equal(t, http.StatusOK, tt.do(t, http.MethodPost, path, tt.tenantA).Code)

	second := tt.do(t, http.MethodPost, path, tt.tenantA)
	require.Equal(t, http.StatusOK, second.Code)
	var response struct {
		Data struct {
			Created         bool `json:"created"`
			WasAlreadyExist bool `json:"wasAlreadyExist"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &response))
	assert.False(t, response.Data.Created)
	assert.True(t, response.Data.WasAlreadyExist)
	assert.Equal(t, 1, tt.definitionsInTenant(t, tt.tenantA))
}

// TestInstallStatesAreTenantScoped 租户 A 的安装事实不能出现在租户 B 的视图里。
func TestInstallStatesAreTenantScoped(t *testing.T) {
	tt := setupSLATemplateHandler(t)

	require.Equal(t, http.StatusOK,
		tt.do(t, http.MethodPost, "/api/v1/sla/templates/"+tt.catalogKey+"/install", tt.tenantA).Code)

	for _, item := range tt.list(t, tt.tenantB).Data.Items {
		assert.False(t, item.Installed, "租户 B 不应看到租户 A 的安装状态")
	}
	assert.Zero(t, tt.definitionsInTenant(t, tt.tenantB))
}

func TestGetTemplateReturnsProjectedDTO(t *testing.T) {
	tt := setupSLATemplateHandler(t)

	recorder := tt.do(t, http.MethodGet, "/api/v1/sla/templates/service_request_standard", tt.tenantA)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Code int `json:"code"`
		Data struct {
			Key        string                 `json:"key"`
			Industry   string                 `json:"industry"`
			Installed  bool                   `json:"installed"`
			Conditions map[string]interface{} `json:"conditions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, common.SuccessCode, response.Code)
	assert.Equal(t, "service_request_standard", response.Data.Key)
	assert.Equal(t, "service_request", response.Data.Industry)
	assert.False(t, response.Data.Installed)
	require.NotNil(t, response.Data.Conditions)
	assert.NotContains(t, response.Data.Conditions, "templateKey")
}

// TestUnknownTemplateIsNotFound 未知模板是 not-found，不是「请求参数错误」，
// 也不能伪装成一条空的 200 响应。
func TestUnknownTemplateIsNotFound(t *testing.T) {
	tt := setupSLATemplateHandler(t)

	recorder := tt.do(t, http.MethodGet, "/api/v1/sla/templates/does_not_exist", tt.tenantA)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), strconv.Itoa(common.NotFoundCode))

	recorder = tt.do(t, http.MethodPost, "/api/v1/sla/templates/does_not_exist/install", tt.tenantA)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), strconv.Itoa(common.NotFoundCode))

	assert.Zero(t, tt.definitionsInTenant(t, tt.tenantA), "失败的安装不得留下定义")
}

// TestHandlersFailClosedWithoutTenant 缺租户上下文时不能返回一份看起来正常的列表：
// 安装状态会全显示未安装，用户会重复安装出脏数据。
func TestHandlersFailClosedWithoutTenant(t *testing.T) {
	tt := setupSLATemplateHandler(t)

	recorder := tt.do(t, http.MethodGet, "/api/v1/sla/templates", 0)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), `"items"`)

	recorder = tt.do(t, http.MethodGet, "/api/v1/sla/templates/change_normal", 0)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	recorder = tt.do(t, http.MethodPost, "/api/v1/sla/templates/change_normal/install", 0)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	assert.Zero(t, tt.definitionsInTenant(t, tt.tenantA))
}
