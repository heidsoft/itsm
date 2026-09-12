package incident

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"
)

// 本文件覆盖 PUT /api/v1/incidents/:id 的 HTTP 契约，走真实 Ent 仓储 + 真实
// production service + 真实 handler，断言 HTTP status 与业务 code 的组合。
//
// 锁住的四个缺陷（均为修复前失败、修复后通过）：
//   - 非法状态迁移必须 409/4090，不得退化成 500/5001。
//   - 客户端携带过期 version 必须 409/4090，不得静默覆盖。
//   - category/subcategory/metadata 必须真的落库，不得返回 code 0 却丢弃。
//   - 响应必须暴露 impact/urgency/version/isMajorIncident，且 NULL 的
//     resolvedAt/closedAt 不得序列化成 0001-01-01。

type updateContractFixture struct {
	client   *ent.Client
	handler  *IncidentHandler
	tenantID int
	userID   int
	stranger int
}

func newUpdateContractFixture(t *testing.T) *updateContractFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:incident_update_contract?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("UpdateContract").SetCode("update-contract").SetDomain("update.test").
		SaveX(ctx)
	owner := client.User.Create().
		SetUsername("owner").SetName("owner").SetEmail("owner@example.com").
		SetPasswordHash("hash").SetTenantID(tenant.ID).
		SaveX(ctx)
	stranger := client.User.Create().
		SetUsername("stranger").SetName("stranger").SetEmail("stranger@example.com").
		SetPasswordHash("hash").SetTenantID(tenant.ID).
		SaveX(ctx)

	production := service.NewIncidentService(client, zap.NewNop().Sugar(), nil)
	repo := NewEntRepository(client)
	handler := NewHandler(NewService(repo, production, nil, nil, nil, zap.NewNop().Sugar()))

	return &updateContractFixture{
		client:   client,
		handler:  handler,
		tenantID: tenant.ID,
		userID:   owner.ID,
		stranger: stranger.ID,
	}
}

// seedContractIncident 建一条 reporter=owner、version=1 的事件，返回 ent 行。
func (f *updateContractFixture) seedContractIncident(t *testing.T, number, status string) *ent.Incident {
	t.Helper()
	return f.client.Incident.Create().
		SetTitle("契约测试事件").
		SetDescription("d").
		SetStatus(status).
		SetPriority("medium").
		SetSeverity("medium").
		SetImpact("high").
		SetUrgency("low").
		SetIncidentNumber(number).
		SetReporterID(f.userID).
		SetTenantID(f.tenantID).
		SaveX(context.Background())
}

type updateResponse struct {
	HTTPStatus int
	Code       int
	Message    string
	Data       json.RawMessage
	Body       string
}

// doUpdate 以 actor 身份对 id 发 PUT，返回 HTTP status + 业务 code + data 原文。
func (f *updateContractFixture) doUpdate(t *testing.T, id int, actorID, tenantID int, body interface{}) updateResponse {
	t.Helper()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Set("tenant_id", tenantID)
		c.Set("user_id", actorID)
		c.Set("role", "agent")
	})
	r.PUT("/api/v1/incidents/:id", f.handler.Update)

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(
		http.MethodPut,
		"/api/v1/incidents/"+strconv.Itoa(id),
		bytes.NewReader(payload),
	))

	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if w.Body.Len() > 0 {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), w.Body.String())
	}

	return updateResponse{
		HTTPStatus: w.Code,
		Code:       envelope.Code,
		Message:    envelope.Message,
		Data:       envelope.Data,
		Body:       w.Body.String(),
	}
}

// TestUpdateHTTPErrorContract 锁住 PUT 的错误语义：冲突 409/4090、不存在
// 404/4004、越权 403/2003、参数 400/1001，且任何情况都不得掉进 500/5001。
func TestUpdateHTTPErrorContract(t *testing.T) {
	f := newUpdateContractFixture(t)
	ctx := context.Background()

	closed := f.seedContractIncident(t, "INC-CT-CLOSED", "closed")
	active := f.seedContractIncident(t, "INC-CT-ACTIVE", "in_progress")

	newStatus := "in_progress"
	newTitle := "改个标题"

	t.Run("非法状态迁移：closed → in_progress 必须 409/4090", func(t *testing.T) {
		got := f.doUpdate(t, closed.ID, f.userID, f.tenantID, map[string]interface{}{
			"status": newStatus,
		})
		assert.Equal(t, http.StatusConflict, got.HTTPStatus,
			"P1 验收：非法状态迁移必须是 HTTP 409，不得退化成 500；body=%s", got.Body)
		assert.Equal(t, common.ConflictCode, got.Code,
			"P1 验收：业务码必须是 4090，不得是 5001；body=%s", got.Body)
		assert.NotEmpty(t, got.Message)

		row := f.client.Incident.GetX(ctx, closed.ID)
		assert.Equal(t, "closed", row.Status, "非法迁移不得改动状态")
	})

	t.Run("客户端携带过期 version 必须 409/4090", func(t *testing.T) {
		got := f.doUpdate(t, active.ID, f.userID, f.tenantID, map[string]interface{}{
			"title":   newTitle,
			"version": 999,
		})
		assert.Equal(t, http.StatusConflict, got.HTTPStatus,
			"P1 验收：过期 version 必须是 HTTP 409；body=%s", got.Body)
		assert.Equal(t, common.ConflictCode, got.Code,
			"P1 验收：过期 version 的业务码必须是 4090；body=%s", got.Body)

		row := f.client.Incident.GetX(ctx, active.ID)
		assert.Equal(t, "契约测试事件", row.Title, "冲突写入不得生效")
		assert.Equal(t, 1, row.Version, "冲突路径不得自增 version")
	})

	t.Run("version 正确：写入成功且 version 自增", func(t *testing.T) {
		got := f.doUpdate(t, active.ID, f.userID, f.tenantID, map[string]interface{}{
			"title":   newTitle,
			"version": 1,
		})
		require.Equal(t, http.StatusOK, got.HTTPStatus, got.Body)
		require.Equal(t, common.SuccessCode, got.Code, got.Body)

		row := f.client.Incident.GetX(ctx, active.ID)
		assert.Equal(t, newTitle, row.Title)
		assert.Equal(t, 2, row.Version, "P1 验收：成功写入必须自增 version，否则乐观锁形同虚设")
	})

	t.Run("未携带 version：仍受仓储条件更新保护且成功", func(t *testing.T) {
		got := f.doUpdate(t, active.ID, f.userID, f.tenantID, map[string]interface{}{
			"description": "旧客户端不带 version",
		})
		require.Equal(t, http.StatusOK, got.HTTPStatus, got.Body)
		require.Equal(t, common.SuccessCode, got.Code, got.Body)
		assert.Equal(t, "旧客户端不带 version", f.client.Incident.GetX(ctx, active.ID).Description)
	})

	t.Run("事件不存在必须 404/4004", func(t *testing.T) {
		got := f.doUpdate(t, 999999, f.userID, f.tenantID, map[string]interface{}{
			"title": newTitle,
		})
		assert.Equal(t, http.StatusNotFound, got.HTTPStatus,
			"不存在的资源必须是 404，不得 500；body=%s", got.Body)
		assert.Equal(t, common.NotFoundCode, got.Code, got.Body)
	})

	t.Run("跨租户写入 fail closed 为 404/4004", func(t *testing.T) {
		before := f.client.Incident.GetX(ctx, active.ID)

		got := f.doUpdate(t, active.ID, f.userID, f.tenantID+1, map[string]interface{}{
			"title": newTitle,
		})
		assert.Equal(t, http.StatusNotFound, got.HTTPStatus,
			"跨租户必须表现为不存在（fail closed）；body=%s", got.Body)
		assert.Equal(t, common.NotFoundCode, got.Code, got.Body)

		after := f.client.Incident.GetX(ctx, active.ID)
		assert.Equal(t, before.Title, after.Title, "跨租户写入不得改动数据")
		assert.Equal(t, before.Version, after.Version, "跨租户写入不得推进 version")
		assert.Equal(t, f.tenantID, after.TenantID, "租户归属不得被改写")
	})

	t.Run("非报告人/受理人必须 403/2003", func(t *testing.T) {
		got := f.doUpdate(t, active.ID, f.stranger, f.tenantID, map[string]interface{}{
			"title": newTitle,
		})
		assert.Equal(t, http.StatusForbidden, got.HTTPStatus,
			"行级守卫拒绝必须是 403；body=%s", got.Body)
		assert.Equal(t, common.ForbiddenCode, got.Code, got.Body)
	})

	t.Run("状态白名单内的合法迁移放行", func(t *testing.T) {
		got := f.doUpdate(t, active.ID, f.userID, f.tenantID, map[string]interface{}{
			"status": "on_hold",
		})
		require.Equal(t, http.StatusOK, got.HTTPStatus, got.Body)
		require.Equal(t, common.SuccessCode, got.Code, got.Body)
		assert.Equal(t, "on_hold", f.client.Incident.GetX(ctx, active.ID).Status)
	})
}

// TestUpdatePersistsClassificationFields 锁住「静默成功」型数据丢失：
// 修复前 handler 把 category/subcategory/metadata 映射到 updates，service 却
// 只留了 "// ... other fields" 占位注释，PUT 返回 code 0 而 DB 毫无变化。
// 线上 IncidentDetail 的「保存事件分类」与事件编辑页的分类字段因此完全失效。
func TestUpdatePersistsClassificationFields(t *testing.T) {
	f := newUpdateContractFixture(t)
	ctx := context.Background()

	inc := f.seedContractIncident(t, "INC-CT-CATEGORY", "in_progress")

	got := f.doUpdate(t, inc.ID, f.userID, f.tenantID, map[string]interface{}{
		"category":    "network",
		"subcategory": "firewall",
		"metadata":    map[string]interface{}{"probeKey": "probeVal"},
	})
	require.Equal(t, http.StatusOK, got.HTTPStatus, got.Body)
	require.Equal(t, common.SuccessCode, got.Code, got.Body)

	row := f.client.Incident.GetX(ctx, inc.ID)
	assert.Equal(t, "network", row.Category,
		"P1 验收：category 必须真的落库，不得返回成功却丢弃")
	assert.Equal(t, "firewall", row.Subcategory,
		"P1 验收：subcategory 必须真的落库")
	require.NotNil(t, row.Metadata, "P1 验收：metadata 必须真的落库")
	assert.Equal(t, "probeVal", row.Metadata["probeKey"])

	// 出口同样要能看到，否则前端刷新后仍是旧值。
	assert.Contains(t, string(got.Data), `"category":"network"`, got.Body)
	assert.Contains(t, string(got.Data), `"subcategory":"firewall"`, got.Body)
}

// TestUpdateResponseExposesOptimisticLockFields 锁住响应契约：
// 修复前 toDTO 不映射 impact/urgency/version/isMajorIncident，前端拿不到
// version 就无法回传乐观锁，影响度/紧急度列在列表页永远是空白。
func TestUpdateResponseExposesOptimisticLockFields(t *testing.T) {
	f := newUpdateContractFixture(t)

	inc := f.seedContractIncident(t, "INC-CT-DTO", "in_progress")
	_, err := f.client.Incident.UpdateOneID(inc.ID).SetIsMajorIncident(true).Save(context.Background())
	require.NoError(t, err)

	got := f.doUpdate(t, inc.ID, f.userID, f.tenantID, map[string]interface{}{
		"title": "DTO 契约",
	})
	require.Equal(t, http.StatusOK, got.HTTPStatus, got.Body)

	var data struct {
		Impact          string `json:"impact"`
		Urgency         string `json:"urgency"`
		Version         int    `json:"version"`
		IsMajorIncident bool   `json:"isMajorIncident"`
	}
	require.NoError(t, json.Unmarshal(got.Data, &data), got.Body)

	assert.Equal(t, "high", data.Impact, "P2 验收：响应必须暴露 impact")
	assert.Equal(t, "low", data.Urgency, "P2 验收：响应必须暴露 urgency")
	assert.Equal(t, 2, data.Version, "P2 验收：响应必须暴露自增后的 version，前端才能回传乐观锁")
	assert.True(t, data.IsMajorIncident, "P2 验收：响应必须暴露 isMajorIncident")

	// camelCase 契约：不得出现 snake_case 同义字段。
	assert.NotContains(t, got.Body, `"is_major_incident"`)
	assert.NotContains(t, got.Body, `"incident_number"`)
}

// TestUpdateDoesNotCorruptNullTimesViaHTTP 是 P1 数据污染的端到端回归：
// 一次只改标题的 PUT 不得把 NULL 的 resolved_at/closed_at 写成 0001-01-01，
// 也不得把公元 1 年的时间戳泄漏进 API 响应。
func TestUpdateDoesNotCorruptNullTimesViaHTTP(t *testing.T) {
	f := newUpdateContractFixture(t)
	ctx := context.Background()

	inc := f.seedContractIncident(t, "INC-CT-NULLTIME", "in_progress")

	got := f.doUpdate(t, inc.ID, f.userID, f.tenantID, map[string]interface{}{
		"title": "只改标题",
	})
	require.Equal(t, http.StatusOK, got.HTTPStatus, got.Body)

	row := f.client.Incident.GetX(ctx, inc.ID)
	assert.Equal(t, "只改标题", row.Title)
	assert.True(t, row.ResolvedAt.IsZero(),
		"P1 验收：PUT 不得把 resolved_at 从 NULL 污染成 0001-01-01")
	assert.True(t, row.ClosedAt.IsZero(),
		"P1 验收：PUT 不得把 closed_at 从 NULL 污染成 0001-01-01")

	// DTO 的 resolvedAt/closedAt 是 *time.Time + omitempty，NULL 应当整体缺席。
	assert.NotContains(t, string(got.Data), `"resolvedAt"`,
		"未解决的事件不得在响应里出现 resolvedAt，更不能是 0001-01-01；body=%s", got.Body)
	assert.NotContains(t, string(got.Data), `"closedAt"`, got.Body)
	assert.NotContains(t, got.Body, "0001-01-01", "任何字段都不得泄漏公元 1 年时间戳")
}

// TestUpdateKeepsRealResolvedTimeViaHTTP 防止上面的修复矫枉过正：
// 已解决事件的真实 resolved_at 必须保留并出现在响应中。
func TestUpdateKeepsRealResolvedTimeViaHTTP(t *testing.T) {
	f := newUpdateContractFixture(t)
	ctx := context.Background()

	inc := f.seedContractIncident(t, "INC-CT-RESOLVED", "in_progress")
	resolvedAt := time.Date(2026, 9, 12, 8, 30, 0, 0, time.UTC)
	_, err := f.client.Incident.UpdateOneID(inc.ID).
		SetStatus("resolved").SetResolvedAt(resolvedAt).
		Save(ctx)
	require.NoError(t, err)

	got := f.doUpdate(t, inc.ID, f.userID, f.tenantID, map[string]interface{}{
		"title": "改标题但保留解决时间",
	})
	require.Equal(t, http.StatusOK, got.HTTPStatus, got.Body)

	row := f.client.Incident.GetX(ctx, inc.ID)
	assert.True(t, row.ResolvedAt.Equal(resolvedAt), "PUT 不得丢失已有的 resolved_at")
	assert.Contains(t, string(got.Data), `"resolvedAt"`, "已解决事件必须回传 resolvedAt")
}
