// Package project 的 CRUD 写面 HTTP 契约回归（B10.1）。
//
// 权限码：/projects 组挂 project:{read,write,delete}（router/common_system_routes.go）。
//
// 锁定语义（2026-10-18 B10 收敛 + 既有正确面防回归）：
//   - [PJ1 已修复] DELETE /projects/{id} 条件删除验 affected rows，
//     不存在/跨租户/重放 → 404/4004（此前裸「项目不存在: id=..」兜底 500 且
//     把资源 id 泄漏进消息）。
//   - [PJ2 已修复] POST /projects 的 managerId/departmentId 引用必须属于本租户：
//     他租引用 → 400/1001（此前静默落库跨租户引用——L3 类引用注入）。
//   - [PJ3 既有正确面] GET/PUT 跨租户/不存在 → ent NotFound 经 classifyError
//     → 404/4004（存在性掩码），响应字段 camelCase。
package project_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/project"
	commonhandler "itsm-backend/handlers/common"
	projectHandler "itsm-backend/handlers/project"
	"itsm-backend/middleware"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fixture struct {
	engine  *gin.Engine
	client  *ent.Client
	secret  string
	tenantA *ent.Tenant
	tenantB *ent.Tenant
	userA   *ent.User
	endUser *ent.User
	userB   *ent.User
	deptB   *ent.Department
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:project_http_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Proj A").SetCode("proj-a").SetDomain("proj-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Proj B").SetCode("proj-b").SetDomain("proj-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("proj-a-owner").SetEmail("proj-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	endUser := client.User.Create().SetUsername("proj-a-enduser").SetEmail("proj-a-end@example.com").SetName("EndUser A").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	userB := client.User.Create().SetUsername("proj-b-owner").SetEmail("proj-b@example.com").SetName("Owner B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)
	deptB := client.Department.Create().SetName("部门B").SetCode("dept-b").SetTenantID(tenantB.ID).SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "project-contract-secret"
	// projects 路由挂在 SetupCommonSystemRoutes 内，外层 gate 要求
	// CommonHandler != nil（与生产装配一致）；projects 组自身只消费
	// ProjectHandler != nil。CommonHandler 构造为空 Service 即可满足外层 gate。
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:      secret,
		Logger:         logger,
		Client:         client,
		CommonHandler:  commonhandler.NewHandler(nil),
		ProjectHandler: projectHandler.NewHandler(service.NewProjectService(client)),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB,
		userA: userA, endUser: endUser, userB: userB, deptB: deptB,
	}
}

func (fx *fixture) do(t *testing.T, method, path string, user *ent.User, payload ...string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)
	var reader io.Reader
	if len(payload) > 0 {
		reader = strings.NewReader(payload[0])
	}
	req := httptest.NewRequest(method, path, reader)
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
	return env
}

func TestProjectHTTPContract(t *testing.T) {
	fx := newFixture(t)
	base := "/api/v1/projects"
	ctx := context.Background()

	t.Run("create正控制与camelCase响应", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, base, fx.userA,
			`{"name":"平台项目","code":"plat","managerId":0,"departmentId":0}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, decodeEnvelope(t, w).Code)
		require.Contains(t, w.Body.String(), `"startDate"`, "字段必须是 camelCase")
		require.NotContains(t, w.Body.String(), `"start_date"`, "不得出现 snake_case 字段")
	})

	t.Run("create引用他租manager是400掩码", func(t *testing.T) {
		payload := `{"name":"注入项目","code":"inj-mgr","managerId":` + fmt.Sprint(fx.userB.ID) + `,"departmentId":0}`
		w := fx.do(t, http.MethodPost, base, fx.userA, payload)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 1001, decodeEnvelope(t, w).Code)
		// 零副作用守卫：跨租户引用不得落库
		n, err := fx.client.Project.Query().Where(project.TenantIDEQ(fx.tenantA.ID), project.CodeEQ("inj-mgr")).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, n)
	})

	t.Run("create引用他租department是400掩码", func(t *testing.T) {
		payload := `{"name":"注入项目","code":"inj-dept","managerId":0,"departmentId":` + fmt.Sprint(fx.deptB.ID) + `}`
		w := fx.do(t, http.MethodPost, base, fx.userA, payload)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 1001, decodeEnvelope(t, w).Code)
		n, err := fx.client.Project.Query().Where(project.TenantIDEQ(fx.tenantA.ID), project.CodeEQ("inj-dept")).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, n)
	})

	t.Run("get/update正控制与跨租户404掩码", func(t *testing.T) {
		projB := fx.client.Project.Create().SetName("他租项目").SetCode("proj-b-1").
			SetTenantID(fx.tenantB.ID).SaveX(ctx)

		projA := fx.client.Project.Create().SetName("本租项目").SetCode("proj-a-1").
			SetTenantID(fx.tenantA.ID).SaveX(ctx)
		w := fx.do(t, http.MethodGet, fmt.Sprintf("%s/%d", base, projA.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// GET 跨租户：ent First → classifyError 404
		w = fx.do(t, http.MethodGet, fmt.Sprintf("%s/%d", base, projB.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)

		// PUT 跨租户：同掩码且对端不受影响
		w = fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, projB.ID), fx.userA, `{"name":"越权改名"}`)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		b, err := fx.client.Project.Get(ctx, projB.ID)
		require.NoError(t, err)
		require.Equal(t, "他租项目", b.Name, "跨租户更新不得触碰对端")

		// PUT 404：不存在的 id 同掩码
		w = fx.do(t, http.MethodPut, base+"/999999", fx.userA, `{"name":"幽灵改名"}`)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("delete重放与跨租户是404掩码", func(t *testing.T) {
		projA := fx.client.Project.Create().SetName("待删项目").SetCode("proj-a-del").
			SetTenantID(fx.tenantA.ID).SaveX(ctx)
		projB := fx.client.Project.Create().SetName("他租待删").SetCode("proj-b-del").
			SetTenantID(fx.tenantB.ID).SaveX(ctx)

		// 冒烟：本租删除成功一次
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, projA.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// 重放删除 → 404，不再假装成功
		w = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, projA.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)

		// 跨租户删除 → 404 掩码且对端保留
		w = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, projB.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		_, err := fx.client.Project.Get(ctx, projB.ID)
		require.NoError(t, err, "跨租户删除不得注销对端行")
	})

	t.Run("list租户隔离", func(t *testing.T) {
		fx.client.Project.Create().SetName("B项目").SetCode("proj-b-2").
			SetTenantID(fx.tenantB.ID).SaveX(ctx)
		w := fx.do(t, http.MethodGet, base, fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		body := w.Body.String()
		require.NotContains(t, body, `"name":"B项目"`, "列表不得混入他租项目")
	})

	t.Run("list权限码403禁拒", func(t *testing.T) {
		// project:read；end_user 不持有。
		w := fx.do(t, http.MethodGet, base, fx.endUser)
		require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 2003, decodeEnvelope(t, w).Code)
	})
}
