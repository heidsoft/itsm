package cmdb

// 回归：CI 负责人（owner_id）是行级权限的唯一判定载体，写路径必须真正落地它。
//
// 三条不变式：
//  1. 给了 ownerId 就必须落 owner_id，且 owned_by 文本同步成该用户显示名
//     （否则界面显示一个名字、权限按另一个 ID 判定，排障时无从对齐）；
//  2. 不存在的 / 跨租户的 ownerId 必须被拒（400），不能写入——写入的后果是
//     该 CI 永久落入「谁都无权查看、也无人能认领」的黑洞；
//  3. UpdateCI 的 ownerId 用指针语义区分「不改」与「清空」：人离职后资产
//     必须能释放回未归属公共池，否则会永久占着权限。

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

type ownerFixture struct {
	router   *gin.Engine
	client   *ent.Client
	tenantID int
	ciTypeID int
	ownerID  int
}

func newOwnerFixture(t *testing.T) ownerFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:ci_owner_write?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("CI 负责人写入").SetCode("ci-owner-write").
		SetDomain("ci-owner-write.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	owner, err := client.User.Create().
		SetUsername("alice").SetName("Alice Chen").SetEmail("alice@ci-owner-write.test").
		SetPasswordHash("hashedpassword").
		SetTenantID(tenant.ID).SetRole("agent").Save(ctx)
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
	r.POST("/api/v1/cmdb/cis", h.CreateCI)
	r.PUT("/api/v1/cmdb/cis/:id", h.UpdateCI)

	return ownerFixture{
		router: r, client: client, tenantID: tenant.ID, ciTypeID: ciType.ID, ownerID: owner.ID,
	}
}

func (f ownerFixture) do(t *testing.T, method, path string, payload map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)
	return w
}

// 创建时给 ownerId：必须落 owner_id，且 owned_by 同步成用户显示名。
func TestCreateCI_PersistsOwnerIDAndSyncsDisplayName(t *testing.T) {
	f := newOwnerFixture(t)

	w := f.do(t, http.MethodPost, "/api/v1/cmdb/cis", map[string]interface{}{
		"name": "alice-server", "ciTypeId": f.ciTypeID, "status": common.CIStatusActive,
		"ownerId": f.ownerID,
	})
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Data struct {
			ID    int    `json:"id"`
			Owner string `json:"ownedBy"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	ci, err := f.client.ConfigurationItem.Get(context.Background(), resp.Data.ID)
	require.NoError(t, err)
	// ent 对 Optional+非 Nillable 的 int 在 Go 侧读成值类型，0 即「未归属」。
	assert.NotZero(t, ci.OwnerID, "ownerId 必须落库，否则行级权限无从判定")
	assert.Equal(t, f.ownerID, ci.OwnerID)
	assert.Equal(t, "Alice Chen", ci.OwnedBy, "owned_by 应同步成显示名，避免与 owner_id 自相矛盾")
}

// 不存在的 ownerId：400 拒绝，绝不写入「无人可认领」的资产。
func TestCreateCI_RejectsUnknownOwner(t *testing.T) {
	f := newOwnerFixture(t)

	w := f.do(t, http.MethodPost, "/api/v1/cmdb/cis", map[string]interface{}{
		"name": "ghost-owner-ci", "ciTypeId": f.ciTypeID, "status": common.CIStatusActive,
		"ownerId": 99999,
	})
	require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.BadRequestCode, resp.Code)

	// 确认确实没落库：拒绝必须是全有或全无，不能留一条半成品 CI。
	n, err := f.client.ConfigurationItem.Query().Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n, "被拒绝的创建不得留下任何 CI 记录")
}

// 跨租户 ownerId：同样 400，避免把资产挂到别的租户用户上绕过租户隔离。
func TestCreateCI_RejectsCrossTenantOwner(t *testing.T) {
	f := newOwnerFixture(t)

	otherTenant, err := f.client.Tenant.Create().SetName("别的租户").SetCode("ci-owner-other").
		SetDomain("ci-owner-other.test").SetStatus("active").Save(context.Background())
	require.NoError(t, err)
	foreign, err := f.client.User.Create().
		SetUsername("bob").SetName("Bob").SetEmail("bob@ci-owner-other.test").
		SetPasswordHash("hashedpassword").
		SetTenantID(otherTenant.ID).SetRole("agent").Save(context.Background())
	require.NoError(t, err)

	w := f.do(t, http.MethodPost, "/api/v1/cmdb/cis", map[string]interface{}{
		"name": "cross-tenant-owner", "ciTypeId": f.ciTypeID, "status": common.CIStatusActive,
		"ownerId": foreign.ID,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
}

// UpdateCI 的 ownerId 指针语义：这是最容易出错的一处。
//
// 若「没传字段」和「传 0」被当成同一件事，编辑页每次保存都会把负责人清空
// （前端表单回填 ownerId 后提交，字段总是在 payload 里）。反过来，若「传 0」
// 被当成「不改」，人离职后资产永远卸不掉负责人，会一直占着行级权限。
// 两种情况都是真实事故，故两侧都要锁。
func TestUpdateCI_OwnerIDPointerSemantics(t *testing.T) {
	f := newOwnerFixture(t)
	ctx := context.Background()

	// 先建一条带负责人的 CI。
	w := f.do(t, http.MethodPost, "/api/v1/cmdb/cis", map[string]interface{}{
		"name": "update-target", "ciTypeId": f.ciTypeID, "status": common.CIStatusActive,
		"ownerId": f.ownerID,
	})
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var created struct {
		Data struct {
			ID      int `json:"id"`
			Version int `json:"version"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	ciID := created.Data.ID

	// ① 不传 ownerId：负责人必须保持原值（「不改」）。
	w = f.do(t, http.MethodPut, "/api/v1/cmdb/cis/"+strconv.Itoa(ciID), map[string]interface{}{
		"name": "update-target-renamed",
	})
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	ci, err := f.client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	assert.Equal(t, f.ownerID, ci.OwnerID, "未传 ownerId 时不得改动负责人")
	assert.Equal(t, "Alice Chen", ci.OwnedBy)

	// ② 显式传 0：清空归属，回落为未归属公共资产。
	w = f.do(t, http.MethodPut, "/api/v1/cmdb/cis/"+strconv.Itoa(ciID), map[string]interface{}{
		"ownerId": 0,
	})
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	ci, err = f.client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	assert.Equal(t, 0, ci.OwnerID, "显式传 0 必须清空 owner_id")
	assert.Equal(t, "", ci.OwnedBy, "清空归属时展示文本也要清空，否则界面显示旧负责人")

	// ③ 再指派回原负责人：能恢复。
	w = f.do(t, http.MethodPut, "/api/v1/cmdb/cis/"+strconv.Itoa(ciID), map[string]interface{}{
		"ownerId": f.ownerID,
	})
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	ci, err = f.client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	assert.Equal(t, f.ownerID, ci.OwnerID)
}
