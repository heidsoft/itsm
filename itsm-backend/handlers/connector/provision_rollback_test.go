package connector

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/connector"
)

// K2：Provision 落库失败时 manager 已经把旧实例关闭并换成了新实例
// （manager.go Provision 的 previous.Close()）。旧实现只 Revoke 新实例，
// 结果是「库里有旧配置、内存里没有实例」，重启前连接器完全不可用。
// RotateSecret 早已用 Provision(previous) 恢复，本测试锁住同一模式覆盖此入口。

func provisionRequest(t *testing.T, h *Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/connectors/configs", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", 1)
	h.Provision(c)
	return w
}

func TestProvision_SaveFailureRestoresPreviousInstance(t *testing.T) {
	f := newRotateFixture(t) // 运行态与库中都是 old-token-1111
	// 关掉 DB 让下一次 store.Save 必然失败，模拟真实世界的约束错误/DB 抖动。
	require.NoError(t, f.client.Close())

	w := provisionRequest(t, f.handler,
		`{"name":"rotate-stub","provider":"stub","enabled":true,"credentials":{"token":"new-token-9999"},"settings":{}}`)
	require.NotEqual(t, http.StatusOK, w.Code, "落库失败必须显性报错，body=%s", w.Body.String())

	cfgs := f.manager.ListByTenant(1)
	require.Len(t, cfgs, 1, "旧实例必须被恢复，不能只剩一个空 manager")
	assert.Equal(t, "old-token-1111", cfgs[0].Credentials["token"],
		"恢复的必须是旧凭据，而不是未落库的新凭据")
	assert.Equal(t, "old-token-1111", (*f.lastInit)["token"],
		"恢复时要用旧配置重新 Init，运行态持有的凭据须与库一致")
	_, ok := f.manager.Get(1, "rotate-stub")
	assert.True(t, ok, "恢复后连接器必须可被调用")
}

func TestProvision_FirstTimeSaveFailureLeavesNoGhostInstance(t *testing.T) {
	f := newRotateFixture(t)
	// 造「首次配置」场景：库里没有旧实例可恢复。
	f.manager.Revoke(connector.Config{TenantID: 1, Name: "rotate-stub", Provider: "stub"})
	require.Empty(t, f.manager.ListByTenant(1))
	require.NoError(t, f.client.Close())

	w := provisionRequest(t, f.handler,
		`{"name":"rotate-stub","provider":"stub","enabled":true,"credentials":{"token":"first-token"},"settings":{}}`)
	require.NotEqual(t, http.StatusOK, w.Code)

	assert.Empty(t, f.manager.ListByTenant(1),
		"无旧实例可恢复时只需撤销新实例，不能凭空造一个未落库的运行态")
}

func TestProvision_SaveSuccessStillAppliesNewConfig(t *testing.T) {
	f := newRotateFixture(t)

	w := provisionRequest(t, f.handler,
		`{"name":"rotate-stub","provider":"stub","enabled":true,"credentials":{"token":"new-token-8888"},"settings":{}}`)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	cfgs := f.manager.ListByTenant(1)
	require.Len(t, cfgs, 1)
	assert.Equal(t, "new-token-8888", cfgs[0].Credentials["token"], "正常路径不能被回滚逻辑打断")
	assert.Equal(t, "new-token-8888", f.dbCredentials(t)["token"])
}
