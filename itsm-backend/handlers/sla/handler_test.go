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
	r.POST("/api/v1/sla/monitor", h.GetSLAMonitoring)
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
}

func TestSLAHandler_GetMonitoringUsesCamelCaseContract(t *testing.T) {
	r, _, _ := setupSLAHandler(t)
	resp := doSLAReq(t, r, "POST", "/api/v1/sla/monitor", dto.SLAMonitoringRequest{
		StartTime: "30d",
		EndTime:   "now",
	}, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data, "totalViolations")
	assert.Contains(t, data, "resolvedViolations")
	assert.Contains(t, data, "activeViolations")
	assert.Contains(t, data, "complianceRate")
	assert.Equal(t, float64(1), data["complianceRate"])
	assert.NotContains(t, data, "total_violations")
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

func slaUniqueID() string { return strconv.FormatInt(time.Now().UnixNano(), 10) }

func slaStr(resp *common.Response) string {
	b, _ := json.Marshal(resp)
	return string(b)
}
