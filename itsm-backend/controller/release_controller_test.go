package controller

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

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// withTestAuth injects tenant/user context exactly like the real auth
// middleware, so controllers that call middleware.GetTenantID / c.Get are
// exercised. Sending header X-Skip-Auth: 1 skips injection so we can
// verify the unauthorized (missing-tenant) guard path.
func withTestAuth(tid, uid int) gin.HandlerFunc {
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

// doReq is a small helper that performs an HTTP request against the test
// router and decodes the standard {code,message,data} envelope.
func doReq(t *testing.T, r *gin.Engine, method, path string, body interface{}, skipAuth bool) *common.Response {
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

func seedTenantUser(t *testing.T, client *ent.Client) (int, int) {
	t.Helper()
	ctx := context.Background()
	uid := uniqueTestID()
	tenant, err := client.Tenant.Create().
		SetName("TestTenant").
		SetCode("TEST" + uid).
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("testuser" + uid).
		SetEmail("test" + uid + "@example.com").
		SetPasswordHash("hashed").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetRole("admin").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID, user.ID
}

func setupReleaseController(t *testing.T) (*gin.Engine, *ent.Client, int, int) {
	gin.SetMode(gin.TestMode)
	dsn := "file:" + filepath.Join(t.TempDir(), "release_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()

	tenantID, userID := seedTenantUser(t, client)

	svc := service.NewReleaseService(client, logger)
	ctrl := NewReleaseController(logger, svc)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(withTestAuth(tenantID, userID))
	r.POST("/api/v1/releases", ctrl.CreateRelease)
	r.GET("/api/v1/releases", ctrl.ListReleases)
	r.GET("/api/v1/releases/stats", ctrl.GetReleaseStats)
	r.GET("/api/v1/releases/:id", ctrl.GetRelease)
	r.PUT("/api/v1/releases/:id", ctrl.UpdateRelease)
	r.PUT("/api/v1/releases/:id/status", ctrl.UpdateReleaseStatus)
	r.DELETE("/api/v1/releases/:id", ctrl.DeleteRelease)
	return r, client, tenantID, userID
}

func TestReleaseController_CreateRelease(t *testing.T) {
	r, client, _, _ := setupReleaseController(t)

	t.Run("成功创建发布", func(t *testing.T) {
		body := dto.CreateReleaseRequest{
			ReleaseNumber: "REL-2026-001",
			Title:         "Q3 生产环境发布",
			Description:   "例行版本发布",
			Type:          "standard",
			Environment:   "production",
		}
		resp := doReq(t, r, "POST", "/api/v1/releases", body, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "REL-2026-001", data["releaseNumber"])
		assert.Equal(t, "Q3 生产环境发布", data["title"])
	})

	t.Run("缺少必填标题应返回参数错误", func(t *testing.T) {
		body := dto.CreateReleaseRequest{ReleaseNumber: "REL-X"}
		resp := doReq(t, r, "POST", "/api/v1/releases", body, false)
		assert.Equal(t, common.BadRequestCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("缺少租户上下文应返回未授权", func(t *testing.T) {
		body := dto.CreateReleaseRequest{ReleaseNumber: "REL-Y", Title: "T"}
		resp := doReq(t, r, "POST", "/api/v1/releases", body, true)
		assert.Equal(t, common.UnauthorizedCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("缺少用户上下文应返回未授权", func(t *testing.T) {
		// 仅注入租户、不注入用户：验证 user_id 校验。
		r2 := gin.New()
		r2.Use(gin.Recovery())
		r2.Use(func(c *gin.Context) {
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
			c.Set("tenant_id", 1)
			c.Next()
		})
		r2.POST("/api/v1/releases", NewReleaseController(zaptest.NewLogger(t).Sugar(), service.NewReleaseService(client, zaptest.NewLogger(t).Sugar())).CreateRelease)
		req, _ := http.NewRequest("POST", "/api/v1/releases", bytesFor(t, dto.CreateReleaseRequest{ReleaseNumber: "REL-Z", Title: "T"}))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r2.ServeHTTP(w, req)
		var resp common.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, common.UnauthorizedCode, resp.Code, "body=%s", mustString(&resp))
	})
}

func TestReleaseController_ListAndStats(t *testing.T) {
	r, _, _, _ := setupReleaseController(t)

	t.Run("列表返回成功", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/releases", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("统计返回成功", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/releases/stats", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	})
}

func TestReleaseController_GetRelease(t *testing.T) {
	r, _, _, _ := setupReleaseController(t)

	created := doReq(t, r, "POST", "/api/v1/releases", dto.CreateReleaseRequest{
		ReleaseNumber: "REL-GET-1", Title: "获取测试",
	}, false)
	require.Equal(t, common.SuccessCode, created.Code)
	id := int(created.Data.(map[string]interface{})["id"].(float64))

	t.Run("按ID获取成功", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/releases/"+strconv.Itoa(id), nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("非法ID应返回参数错误", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/releases/abc", nil, false)
		assert.Equal(t, common.BadRequestCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("不存在的ID应返回未找到", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/releases/999999", nil, false)
		assert.Equal(t, common.NotFoundCode, resp.Code, "body=%s", mustString(resp))
	})
}

func TestReleaseController_DeleteRelease(t *testing.T) {
	r, _, _, _ := setupReleaseController(t)

	created := doReq(t, r, "POST", "/api/v1/releases", dto.CreateReleaseRequest{
		ReleaseNumber: "REL-DEL-1", Title: "删除测试",
	}, false)
	require.Equal(t, common.SuccessCode, created.Code)
	id := int(created.Data.(map[string]interface{})["id"].(float64))

	resp := doReq(t, r, "DELETE", "/api/v1/releases/"+strconv.Itoa(id), nil, false)
	assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
}

func bytesFor(t *testing.T, v interface{}) *bytes.Buffer {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func mustString(resp *common.Response) string {
	b, _ := json.Marshal(resp)
	return string(b)
}
