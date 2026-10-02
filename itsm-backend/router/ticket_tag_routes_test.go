package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	ticketTagHandler "itsm-backend/handlers/ticket_tag"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-02 边缘功能收口 E1-3）：全局标签页原先打的是只读别名
// /api/v1/tags，真 CRUD 所在的 /api/v1/ticket-tags 没有任何 UI 调用，且这条写链
// 本身是坏的：
//   - CreateTagRequest.TenantID 带 binding:"required"，前端不来自报 tenantId，
//     「新建标签」必 400；租户本应只来自认证上下文。
//   - IsActive 是非指针 bool，请求体不带该字段时按 false 建，UI 建出来的标签
//     全部是停用态。
//   - 空颜色被写成空串（Ent 字段默认值对显式 SetColor 不生效），列表渲染成无色。
//   - 重名、在用删除、跨租户读写全部回 500/5001「操作失败」，调用方无法区分
//     冲突、不存在和后端故障。
// 必须从真实 Router 入口证明修复后的契约，并覆盖跨租户拒绝。

const ticketTagsPath = "/api/v1/ticket-tags"

type tagRouteFixture struct {
	engine   *gin.Engine
	secret   string
	client   *ent.Client
	userA    *ent.User // tenant A super_admin
	stranger *ent.User // tenant B super_admin，用于跨租户探测
	ticketA  *ent.Ticket
	tagA     *ent.TicketTag // tenant A 已存在的标签
}

func setupTicketTagRouteTest(t *testing.T) tagRouteFixture {
	t.Helper()

	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:router_ticket_tag_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Tag A").SetCode("tag-a").SetDomain("tag-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Tag B").SetCode("tag-b").SetDomain("tag-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("tag-a-admin").SetEmail("tag-a@example.com").SetName("A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("tag-b-admin").SetEmail("tag-b@example.com").SetName("B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	ticketA := client.Ticket.Create().
		SetTicketNumber("TKT-TAG-A-001").
		SetTitle("标签绑定回归").
		SetDescription("验证工单标签写链与租户隔离").
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(userA.ID).
		SetTenantID(tenantA.ID).
		SaveX(ctx)

	tagA := client.TicketTag.Create().SetName("network").SetColor("#1890ff").SetIsActive(true).
		SetTenantID(tenantA.ID).SaveX(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	const secret = "ticket-tag-secret"
	SetupRoutes(r, &RouterConfig{
		JWTSecret:        secret,
		Logger:           logger,
		Client:           client,
		TicketTagHandler: ticketTagHandler.NewHandler(service.NewTicketTagService(client), logger),
	})

	return tagRouteFixture{engine: r, secret: secret, client: client, userA: userA, stranger: stranger, ticketA: ticketA, tagA: tagA}
}

// tagEnvelope 是 { code, message, data } 的解析目标；data 保持 RawMessage，
// 由各子用例按自己的契约形状解码。
type tagEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (f tagRouteFixture) do(t *testing.T, method, path string, body *string, user *ent.User) (int, tagEnvelope) {
	t.Helper()

	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(*body)
	}
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, f.secret, time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	f.engine.ServeHTTP(w, req)

	var env tagEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
	return w.Code, env
}

// tagDTO 解码单个标签响应。
func (f tagRouteFixture) tagDTO(t *testing.T, env tagEnvelope) map[string]any {
	t.Helper()
	var tag map[string]any
	require.NoError(t, json.Unmarshal(env.Data, &tag), "data=%s", env.Data)
	return tag
}

func jsonBody(s string) *string { return &s }

func TestTicketTagWriteRoutes(t *testing.T) {
	fx := setupTicketTagRouteTest(t)

	t.Run("新建标签不要求客户端自报租户，默认启用并补默认颜色", func(t *testing.T) {
		// 名称带首尾空格：必须与工单绑定路径 ResolveTagIDsByNames 的口径一致，
		// 否则同一个显示名称会存在两条标签。
		w, env := fx.do(t, http.MethodPost, ticketTagsPath, jsonBody(`{"name":"  vpn  ","description":"VPN 访问"}`), fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		require.Equal(t, 0, env.Code)

		tag := fx.tagDTO(t, env)
		assert.Equal(t, "vpn", tag["name"], "名称必须归一化")
		assert.Equal(t, true, tag["isActive"], "请求体未带 isActive 必须建为启用")
		assert.Equal(t, "#1890ff", tag["color"], "空颜色必须补默认值，否则标签渲染成无色")
		assert.Equal(t, fx.tagA.TenantID, int(tag["tenantId"].(float64)), "租户必须取自认证上下文，而非请求体")
	})

	t.Run("重名是 409 冲突而不是后端故障", func(t *testing.T) {
		w, env := fx.do(t, http.MethodPost, ticketTagsPath, jsonBody(`{"name":"network"}`), fx.userA)
		assert.Equal(t, http.StatusConflict, w, "body=%s", string(env.Data))
		assert.Equal(t, 4090, env.Code)
		assert.Contains(t, env.Message, "标签名称已存在")
	})

	t.Run("空白名称是 400 参数错误", func(t *testing.T) {
		w, env := fx.do(t, http.MethodPost, ticketTagsPath, jsonBody(`{"name":"   "}`), fx.userA)
		assert.Equal(t, http.StatusBadRequest, w, "body=%s", string(env.Data))
		assert.Equal(t, 1001, env.Code)
	})

	t.Run("列表信封使用 items 且带租户范围", func(t *testing.T) {
		w, env := fx.do(t, http.MethodGet, ticketTagsPath, nil, fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))

		var list struct {
			Items []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
			Total int `json:"total"`
			Tags  []struct {
				ID int `json:"id"`
			} `json:"tags"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &list))
		assert.Equal(t, 2, list.Total)
		assert.Len(t, list.Items, 2, "列表键必须是 items，与前端契约一致")

		// 信封里不得同时存在第二套列表键。
		assert.Empty(t, list.Tags, "data 不得同时返回 items 和 tags")

		// tenant B 只能看到自己的范围。
		w, env = fx.do(t, http.MethodGet, ticketTagsPath, nil, fx.stranger)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		require.NoError(t, json.Unmarshal(env.Data, &list))
		assert.Zero(t, list.Total, "跨租户不得看到 tenant A 的标签")
		assert.Empty(t, list.Items)
	})

	t.Run("更新可以改颜色并保留启用位", func(t *testing.T) {
		w, env := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", ticketTagsPath, fx.tagA.ID),
			jsonBody(`{"color":"#ff4d4f"}`), fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))

		tag := fx.tagDTO(t, env)
		assert.Equal(t, "#ff4d4f", tag["color"])
		assert.Equal(t, true, tag["isActive"], "未传 isActive 不得把标签改成停用")
	})

	t.Run("改名撞名是 409", func(t *testing.T) {
		vpnID := fx.mustFindTagID(t, "vpn", fx.userA)

		w, env := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", ticketTagsPath, vpnID), jsonBody(`{"name":"network"}`), fx.userA)
		assert.Equal(t, http.StatusConflict, w, "body=%s", string(env.Data))
		assert.Equal(t, 4090, env.Code)
		assert.Contains(t, env.Message, "标签名称已存在")
	})

	t.Run("跨租户读写删除别人的标签一律 404", func(t *testing.T) {
		path := fmt.Sprintf("%s/%d", ticketTagsPath, fx.tagA.ID)

		w, env := fx.do(t, http.MethodGet, path, nil, fx.stranger)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)

		w, env = fx.do(t, http.MethodPut, path, jsonBody(`{"color":"#000000"}`), fx.stranger)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)

		w, env = fx.do(t, http.MethodDelete, path, nil, fx.stranger)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)

		// 未被跨租户改掉。
		w, env = fx.do(t, http.MethodGet, path, nil, fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		assert.NotEqual(t, "#000000", fx.tagDTO(t, env)["color"])
	})

	t.Run("在用标签删除是 409，解绑后可以删除", func(t *testing.T) {
		w, env := fx.do(t, http.MethodPost, ticketTagsPath, jsonBody(`{"name":"temp"}`), fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		tempID := int(fx.tagDTO(t, env)["id"].(float64))

		bindBody := jsonBody(fmt.Sprintf(`{"tagIds":[%d]}`, tempID))
		w, env = fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/tags", fx.ticketA.ID), bindBody, fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		assert.Equal(t, 0, env.Code)

		w, env = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", ticketTagsPath, tempID), nil, fx.userA)
		assert.Equal(t, http.StatusConflict, w, "body=%s", string(env.Data))
		assert.Equal(t, 4090, env.Code)
		assert.Contains(t, env.Message, "标签仍被工单使用")

		// 真实删除路径仍然要按租户 fail closed：tenant B 不能替 tenant A 解绑。
		unbindBody := jsonBody(fmt.Sprintf(`{"tagIds":[%d]}`, tempID))
		w, env = fx.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/tickets/%d/tags", fx.ticketA.ID), unbindBody, fx.stranger)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)

		w, env = fx.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/tickets/%d/tags", fx.ticketA.ID), unbindBody, fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))

		w, env = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", ticketTagsPath, tempID), nil, fx.userA)
		assert.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		assert.Equal(t, 0, env.Code)
	})

	t.Run("绑定不存在的标签是 404，不返回 500", func(t *testing.T) {
		w, env := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/tags", fx.ticketA.ID),
			jsonBody(`{"tagIds":[987654]}`), fx.userA)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)
	})

	t.Run("按名称绑定的标签落在调用者租户", func(t *testing.T) {
		w, env := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/tags", fx.ticketA.ID),
			jsonBody(`{"tags":["存储"]}`), fx.stranger)
		// stranger 不是 tenant A 工单的相关方，跨租户工单必须先 fail closed。
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))

		w, env = fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/tags", fx.ticketA.ID),
			jsonBody(`{"tags":["存储"]}`), fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))

		// 自动创建的标签归属 tenant A，不得写进别的租户。
		id := fx.mustFindTagID(t, "存储", fx.userA)
		tag, err := fx.client.TicketTag.Get(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, fx.tagA.TenantID, tag.TenantID)
		assert.True(t, tag.IsActive, "自动创建的标签必须启用")
	})
}

func (f tagRouteFixture) mustFindTagID(t *testing.T, name string, user *ent.User) int {
	t.Helper()
	w, env := f.do(t, http.MethodGet, ticketTagsPath, nil, user)
	require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))

	var list struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &list))
	for _, item := range list.Items {
		if item.Name == name {
			return item.ID
		}
	}
	t.Fatalf("未找到名为 %q 的标签", name)
	return 0
}
