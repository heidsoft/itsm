package cmdb

// 回归：CI 生命周期推进到 in_service（online）时必须校验必填属性。
//
// 必填校验器只在 CreateCI/UpdateCI 两条写路径上生效，而生命周期推进是第三条
// 写路径——它直接改 lifecycle_status、从不碰 attributes。缺关键属性的 CI 因此
// 可以被推进为 online，表现为「已在服务中」却信息不全；影响分析、变更风险评估
// 与权限分配读的都是这张表。
//
// 本文件在 HTTP 边界锁住：缺必填属性 → 4xx 参数错误（而非 500），且状态不被推进。
// 服务层的细节断言见 internal/services/configuration_item_lifecycle_test.go。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/services"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type onlineGateFixture struct {
	router   *gin.Engine
	client   *ent.Client
	tenantID int
	ciID     int
	ciTypeID int
}

func newOnlineGateFixture(t *testing.T) onlineGateFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:cmdb_online_gate?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("CI 上线门禁").
		SetCode("cmdb-online-gate").
		SetDomain("cmdb-online-gate.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// draft 起点：只有 draft 能推进到 online（状态机要求）
	ci, err := client.ConfigurationItem.Create().
		SetName("gate-ci").
		SetCiTypeID(ciType.ID).
		SetCiType("Server").
		SetStatus("active").
		SetLifecycleStatus(common.CILifecycleStatusDraft).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	historySvc := service.NewCIHistoryService(client, logger)
	tagSvc := service.NewCITagService(client, logger)
	ciSvc := service.NewConfigurationItemService(client, logger, historySvc, tagSvc)
	production := NewProductionService(logger, nil, nil, ciSvc, nil, historySvc, tagSvc, nil, nil)
	h := NewHandler(NewService(nil, production, logger))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Set("user_id", 1)
		c.Next()
	})
	r.PUT("/api/v1/cmdb/cis/:id/lifecycle", h.UpdateLifecycleStatus)

	return onlineGateFixture{
		router: r, client: client, tenantID: tenant.ID, ciID: ci.ID, ciTypeID: ciType.ID,
	}
}

func (f onlineGateFixture) seedRequired(t *testing.T, name, displayName string) {
	t.Helper()
	_, err := f.client.CIAttributeDefinition.Create().
		SetName(name).
		SetDisplayName(displayName).
		SetType("string").
		SetRequired(true).
		SetCiTypeID(f.ciTypeID).
		SetTenantID(f.tenantID).
		SetIsActive(true).
		Save(context.Background())
	require.NoError(t, err)
}

func (f onlineGateFixture) putLifecycle(t *testing.T, body map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	// version 是 binding:"required" 的 int：gin 对 int 的 required 会拒绝 0，
	// 因此请求体必须显式带版本号（客户端基于当前版本提交）。
	if _, ok := body["version"]; !ok {
		body["version"] = 1
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/cmdb/cis/"+strconv.Itoa(f.ciID)+"/lifecycle", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)
	return w
}

// 缺必填属性时上线必须是 4xx 参数错误，不是 500，且状态不动。
func TestUpdateLifecycleStatus_MissingRequiredAttributeIsClientError(t *testing.T) {
	f := newOnlineGateFixture(t)
	f.seedRequired(t, "ip_address", "IP 地址")

	w := f.putLifecycle(t, map[string]interface{}{"status": common.CILifecycleStatusOnline})

	require.Equal(t, http.StatusBadRequest, w.Code,
		"缺必填属性应是 400 参数错误而非 500；body=%s", w.Body.String())

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotEqual(t, 5001, body.Code, "不得落入内部错误码（那会被前端当成服务端故障并重试）")
	assert.Contains(t, body.Message, "ip_address", "文案必须点名缺失字段，前端据此定位表单")
	assert.Contains(t, body.Message, "IP 地址", "文案同时要给展示名")

	ci, err := f.client.ConfigurationItem.Get(context.Background(), f.ciID)
	require.NoError(t, err)
	assert.Equal(t, common.CILifecycleStatusDraft, ci.LifecycleStatus, "门禁失败不得改变状态")
}

// 补齐属性后必须 200——防止门禁把正常上线也拒了。
func TestUpdateLifecycleStatus_WithRequiredAttributeIsAccepted(t *testing.T) {
	f := newOnlineGateFixture(t)
	f.seedRequired(t, "ip_address", "IP 地址")

	_, err := f.client.ConfigurationItem.UpdateOneID(f.ciID).
		SetAttributes(map[string]interface{}{"ip_address": "10.0.0.5"}).
		Save(context.Background())
	require.NoError(t, err)

	w := f.putLifecycle(t, map[string]interface{}{"status": common.CILifecycleStatusOnline})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	ci, err := f.client.ConfigurationItem.Get(context.Background(), f.ciID)
	require.NoError(t, err)
	assert.Equal(t, common.CILifecycleStatusOnline, ci.LifecycleStatus)
}

// 无必填定义的类型不受影响：绝大多数既有 CI 类型没有属性定义。
func TestUpdateLifecycleStatus_NoRequiredDefinitionIsAccepted(t *testing.T) {
	f := newOnlineGateFixture(t)

	w := f.putLifecycle(t, map[string]interface{}{"status": common.CILifecycleStatusOnline})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
