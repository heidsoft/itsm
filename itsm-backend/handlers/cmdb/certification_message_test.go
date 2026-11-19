package cmdb

// 认证端点错误消息的语义锁。
//
// 锁的是「客户端能从 message 里读到什么」，而不是状态码：
// 四种 sentinel 都以 HTTP 400 返回，状态码完全相同，message 是唯一能让人
// 区分「该任务已完成，别重试」与「参数错/权限错，改完再试」的字段。
//
// 曾的缺陷：respondError 给所有 sentinel 共用了调用点的 msg（「完成认证失败」
// /「创建认证任务失败」），于是重复确认返回 {"code":4000,"message":"完成认证失败"}——
// 运维看到「失败」会以为是系统故障而反复重试，实际是幂等拒绝。
// 本测试把「sentinel 自有文案必须原样到达客户端」钉死。
//
// 反向验证（2026-11-19 已执行）：把 respondError 改回共用 msg 后，
// 本文件第一个用例在 assert.Equal(msg) 上立即红（实际 "完成认证失败"，
// 期望 "该认证任务已完成"），恢复后转绿——红在断言而非编译。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type certHandlerFixture struct {
	router   *gin.Engine
	tenantID int
	ciID     int
}

func newCertHandlerFixture(t *testing.T) *certHandlerFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// 每个用例独立的内存库名：用 t.Name() 生成唯一 DSN。
	// 曾共用固定名 + cache=shared，导致同一包内第二个用例看到第一个用例的
	// id=1 已确认任务，重复确认提前命中 400，断言在 require 200 处红。
	// DSN 里 t.Name() 含斜杠（子测试），替换成下划线。
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })
	ctx := t.Context()

	tenant, err := client.Tenant.Create().SetName("认证租户").SetCode("ci-cert-h").
		SetDomain("ci-cert-h.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	ci, err := client.ConfigurationItem.Create().SetName("cert-ci").SetCiTypeID(ciType.ID).
		SetTenantID(tenant.ID).SetOwnerID(7).Save(ctx)
	require.NoError(t, err)

	h := NewCICertificationHandler(NewCICertificationService(client, zap.NewNop().Sugar()), zap.NewNop().Sugar())

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Set("user_id", 9)
		c.Next()
	})
	r.POST("/certification/tasks", h.CreateTask)
	r.POST("/certification/tasks/:id/confirm", h.ConfirmTask)

	return &certHandlerFixture{router: r, tenantID: tenant.ID, ciID: ci.ID}
}

// do 发请求并解析响应体里的 code / message。
func (f *certHandlerFixture) do(t *testing.T, path, body string) (int, int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应体不是 JSON: %s", w.Body.String())
	}
	return w.Code, resp.Code, resp.Message
}

// 重复确认：message 必须是「该认证任务已完成」，不得退化为「完成认证失败」。
func TestConfirmTask_RepeatConfirmSurfacesSpecificMessage(t *testing.T) {
	f := newCertHandlerFixture(t)
	ciID := f.ciID

	status, code, _ := f.do(t, "/certification/tasks",
		`{"ciId":`+strconv.Itoa(ciID)+`,"assigneeId":7}`)
	require.Equal(t, http.StatusOK, status, "首个任务应创建成功")
	require.Equal(t, 0, code)

	status, code, _ = f.do(t, "/certification/tasks/1/confirm", `{"evidence":"核对无误"}`)
	require.Equal(t, http.StatusOK, status, "首次确认应成功")
	require.Equal(t, 0, code)

	// 第二次确认：HTTP 400 + 业务码 4000 + sentinel 自有文案。
	status, code, msg := f.do(t, "/certification/tasks/1/confirm", `{"evidence":"再来一次"}`)
	assert.Equal(t, http.StatusBadRequest, status, "重复确认必须是 400 而非 500")
	assert.Equal(t, 4000, code, "业务码应是状态类 4000，不是未分类 5001")
	assert.Equal(t, ErrCITaskAlreadyConfirmed.Error(), msg,
		"message 必须是 sentinel 自有文案，不能是调用点的笼统「完成认证失败」")
}

// 任何 sentinel 分支都不得回落到调用点传入的通用文案。
//
// 用表驱动覆盖两个调用点（confirm / create），确保将来新增 sentinel 时
// 不会有人图省事再写回共用 msg。
func TestCertificationSentinels_NeverFallBackToCallSiteMessage(t *testing.T) {
	f := newCertHandlerFixture(t)
	ciID := f.ciID

	status, _, _ := f.do(t, "/certification/tasks", `{"ciId":`+strconv.Itoa(ciID)+`,"assigneeId":7}`)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = f.do(t, "/certification/tasks/1/confirm", `{"evidence":"核对无误"}`)
	require.Equal(t, http.StatusOK, status)

	cases := []struct {
		name    string
		path    string
		generic string
	}{
		{"confirm 重复确认", "/certification/tasks/1/confirm", "完成认证失败"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, code, msg := f.do(t, tc.path, `{"evidence":"再试"}`)
			assert.Equal(t, 4000, code)
			assert.NotEqual(t, tc.generic, msg,
				"sentinel 分支不得回落到调用点通用文案 %q", tc.generic)
		})
	}
}
