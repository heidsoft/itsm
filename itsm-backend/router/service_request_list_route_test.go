package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/servicerequest"
	cmdbHandler "itsm-backend/handlers/cmdb"
	scHandler "itsm-backend/handlers/service_catalog"
	srHandler "itsm-backend/handlers/service_request"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-03 边缘功能收口 E4-4）：GET /api/v1/service-requests 与
// GET /api/v1/service-requests/approvals/pending 这两个列表原先各有**三个**分页所有者：
//   - 请求 DTO 上写着 binding:"min=1,max=100"，越界值在绑定阶段就返回 1001 参数错误；
//   - handler 再把没越界的原值下传，自己补一份 page==0→1、pageSize==0→10；
//   - repository_impl 又按 size<1→10、size>100→100 夹一次才写 Offset/Limit。
//
// 而响应末尾用 handler 那套**未夹上限**的原值手算 totalPages 并回显 pageSize，
// 于是「SQL 实际取了几条」和「响应声明的页长/总页数」可以不是同一份真相。
//
// 同一用例还有两套集合键：生产返回 items，前端 service-request-api.ts 的
// ServiceRequestListResponse 却声明 requests+size，靠 normalizeList 的
// `raw.requests || (raw as any).items || []`、`raw.size || (raw as any).pageSize || requests.length`
// 兜住——正是 AGENTS「请求/响应多字段兼容零新增」禁止的写法，且 total 缺失时静默变成当前页长度。
//
// 本测试打在**生产注册**上（SetupRoutes → SetupServiceRequestRoutes → RequirePermission("service_request","read")），
// 解码目标按线上形状声明而不引用 dto，因此修复前也能编译、证明的是运行时行为。
// 同时锁住三条既有正确行为：行级 DataScope、租户收敛、未认证 401。

const serviceRequestListSecret = "sr-list-secret"

// serviceRequestListCreatedAt 让全部种子的 created_at 相同：排序只有 created_at 时，
// 同秒入库的请求单在页边界归属不确定，翻页会重/漏。
var serviceRequestListCreatedAt = time.Date(2026, 9, 12, 8, 0, 0, 0, time.UTC)

type serviceRequestListWire struct {
	Items      []serviceRequestItemWire `json:"items"`
	Total      int                      `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"pageSize"`
	TotalPages int                      `json:"totalPages"`
}

type serviceRequestItemWire struct {
	ID          int    `json:"id"`
	RequesterID int    `json:"requesterId"`
	Status      string `json:"status"`
	Title       string `json:"title"`
}

type serviceRequestListEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func serviceRequestItemIDs(items []serviceRequestItemWire) []int {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestServiceRequestListRouteEnvelopeAndPagination(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_sr_list_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	scRepo := scHandler.NewEntRepository(client)
	svc := srHandler.NewService(
		srHandler.NewEntRepository(client),
		scRepo,
		cmdbHandler.NewEntRepository(client),
		client,
		logger,
		nil,
	)
	handler := srHandler.NewHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:             serviceRequestListSecret,
		Logger:                logger,
		Client:                client,
		ServiceRequestHandler: handler,
	})

	tenantA, adminA, agentA := seedServiceRequestListTenant(ctx, t, client, "sr-a", 25)
	tenantB, adminB := seedServiceRequestListReadOnlyTenant(ctx, t, client, "sr-b", 25)

	allA := serviceRequestIDs(ctx, t, client, tenantA)
	require.Len(t, allA, 25)

	do := func(t *testing.T, userID int, username, role string, tenantID int, path string, query string) (int, string, serviceRequestListEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(userID, username, role, tenantID, serviceRequestListSecret, time.Hour)
		require.NoError(t, err)

		full := path
		if query != "" {
			full += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, full, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env serviceRequestListEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, w.Body.String(), env
	}

	listA := func(t *testing.T, query string) (int, string, serviceRequestListEnvelope) {
		return do(t, adminA, "sr-a-admin", "super_admin", tenantA, "/api/v1/service-requests", query)
	}
	listAs := func(t *testing.T, userID int, username, role string, tenantID int, query string) (int, string, serviceRequestListEnvelope) {
		return do(t, userID, username, role, tenantID, "/api/v1/service-requests", query)
	}
	decode := func(t *testing.T, body string, env serviceRequestListEnvelope) serviceRequestListWire {
		t.Helper()
		var res serviceRequestListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		return res
	}

	t.Run("集合键只有 items，五键齐全且默认页长为 20", func(t *testing.T) {
		status, body, env := listA(t, "")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, envelopeKeys(t, env.Data),
			"服务请求列表必须只返回标准信封五键: body=%s", body)

		res := decode(t, body, env)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 20, res.PageSize, "缺省页长统一到 common 的 20，不再是 handler 私有的 10")
		assert.Equal(t, 25, res.Total, "total 必须是全量条数，不是当前页长度")
		assert.Equal(t, 2, res.TotalPages)
		assert.Len(t, res.Items, 20)
		assert.Equal(t, allA[:20], serviceRequestItemIDs(res.Items), "默认第一页必须是全序的前 20 条")
	})

	t.Run("三页拼接不重不漏", func(t *testing.T) {
		var paged []int
		for page := 1; page <= 3; page++ {
			status, body, env := listA(t, fmt.Sprintf("page=%d&pageSize=10", page))
			require.Equal(t, http.StatusOK, status, body)
			res := decode(t, body, env)
			assert.Equal(t, page, res.Page, body)
			assert.Equal(t, 10, res.PageSize, body)
			assert.Equal(t, 25, res.Total, body)
			assert.Equal(t, 3, res.TotalPages, body)
			want := 10
			if page == 3 {
				want = 5
			}
			require.Len(t, res.Items, want, body)
			paged = append(paged, serviceRequestItemIDs(res.Items)...)
		}
		assert.Equal(t, allA, paged, "created_at 并列时必须有 ID 兜底，否则页边界归属不确定")
	})

	t.Run("pageSize=0 回到默认页长而不是整表", func(t *testing.T) {
		_, body, env := listA(t, "page=1&pageSize=0")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 20, res.PageSize, body)
		assert.Len(t, res.Items, 20, "页长为 0 不得退化成不加 LIMIT: body=%s", body)
		assert.Equal(t, 2, res.TotalPages, body)
	})

	t.Run("越界页长回落默认页长，SQL 与声明同一个真相", func(t *testing.T) {
		// 修复前：绑定阶段 min=1,max=100 先把这个请求打成 1001 参数错误，
		// 与 handler 自己那套「补默认值」是两种结局；repo 还会再夹一次。
		status, body, env := listA(t, "pageSize=5000")
		require.Equal(t, http.StatusOK, status, "越界页长应回落默认页长，不是参数错误: body=%s", body)
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 20, res.PageSize, "越界页长不得原样回显: body=%s", body)
		assert.Len(t, res.Items, 20, body)
		assert.Equal(t, 2, res.TotalPages, body)
	})

	t.Run("page 为负数时回到第一页", func(t *testing.T) {
		_, body, env := listA(t, "page=-1&pageSize=10")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 1, res.Page, "负 Offset 不得出现在查询里")
		assert.Equal(t, allA[:10], serviceRequestItemIDs(res.Items), body)
	})

	t.Run("越界页长下的翻页不丢数据", func(t *testing.T) {
		_, body, env := listA(t, "page=2&pageSize=5000")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 2, res.Page, body)
		assert.Equal(t, allA[20:], serviceRequestItemIDs(res.Items), "声明页长 20 时第 2 页必须是第 21~25 条: body=%s", body)
	})

	t.Run("status 过滤把 total 收敛到匹配条数", func(t *testing.T) {
		// 种子布局：i<20 submitted、i>=20 delivered。
		cases := []struct {
			name  string
			query string
			total int
		}{
			{"精确值 submitted", "status=submitted&pageSize=100", 20},
			{"精确值 delivered", "status=delivered&pageSize=100", 5},
			{"别名 completed→delivered", "status=completed&pageSize=100", 5},
			{"别名 in_progress→provisioning", "status=in_progress&pageSize=100", 0},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, body, env := listA(t, tc.query)
				require.Equal(t, 0, env.Code, body)
				res := decode(t, body, env)
				assert.Equal(t, tc.total, res.Total, "%s 过滤后 total 必须收敛: body=%s", tc.name, body)
				require.Len(t, res.Items, tc.total, body)
			})
		}
	})

	t.Run("空结果序列化为 []", func(t *testing.T) {
		_, body, env := listA(t, "status=cancelled&pageSize=100")
		require.Equal(t, 0, env.Code, body)
		assert.JSONEq(t, "[]", jsonOf(t, env.Data, "items"), body)
		res := decode(t, body, env)
		assert.Equal(t, 0, res.Total, body)
		assert.Equal(t, 0, res.TotalPages, body)
	})

	t.Run("size/limit/userId 等别名与自报身份都不生效", func(t *testing.T) {
		// 修复前：DTO 上有 UserID 字段（form 名 UserID），repo 认 size 别名是历史形态；
		// 现在页长只从 pageSize 读，操作者身份只从认证上下文读。
		_, body, env := listA(t, "pageSize=100&size=5&limit=5&userId="+fmt.Sprint(agentA)+"&user_id="+fmt.Sprint(agentA)+"&sortBy=id")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 25, res.Total, "自报 userId 不得收窄可见范围: body=%s", body)
		assert.Equal(t, 100, res.PageSize, "size/limit 不得作为 pageSize 的别名生效: body=%s", body)
		assert.Equal(t, allA, serviceRequestItemIDs(res.Items), "sortBy 未参与查询，不得改变返回顺序: body=%s", body)
	})

	t.Run("行级数据权限：agent 在全量列表里只看到本人创建或处理的请求单", func(t *testing.T) {
		status, body, env := listAs(t, agentA, "sr-a-agent", "agent", tenantA, "pageSize=100")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		// 种子布局：i=23、24 由 agent 创建，i=10 处理人为 agent。
		assert.Equal(t, 3, res.Total, "本人创建 2 条 + 本人处理 1 条，其余 22 条不得出现: body=%s", body)
		for _, item := range res.Items {
			assert.True(t, item.RequesterID == agentA || serviceRequestProcessorIs(ctx, t, client, item.ID, agentA),
				"越界行: id=%d requesterId=%d body=%s", item.ID, item.RequesterID, body)
		}
	})

	t.Run("me 路径按认证上下文的申请人收敛，不接受客户端身份", func(t *testing.T) {
		status, body, env := do(t, agentA, "sr-a-agent", "agent", tenantA, "/api/v1/service-requests/me", "pageSize=100")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 2, res.Total, "/me 只看本人申请的 2 条（本人处理的第 3 条不属于 me 语义）: body=%s", body)
		for _, item := range res.Items {
			assert.Equal(t, agentA, item.RequesterID, body)
		}

		// 同一路径下自报别人身份不会扩大可见范围。
		_, otherBody, otherEnv := do(t, agentA, "sr-a-agent", "agent", tenantA, "/api/v1/service-requests/me",
			fmt.Sprintf("pageSize=100&userId=%d", adminA))
		require.Equal(t, 0, otherEnv.Code, otherBody)
		assert.Equal(t, 2, decode(t, otherBody, otherEnv).Total, "查询参数里的 userId 不得替换认证身份: body=%s", otherBody)
	})

	t.Run("租户 B 只看到自己的请求单", func(t *testing.T) {
		status, body, env := listAs(t, adminB, "sr-b-admin", "super_admin", tenantB, "pageSize=100")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		allB := serviceRequestIDs(ctx, t, client, tenantB)
		assert.Equal(t, 25, res.Total, "每个租户各 25 条，跨租户不得合并")
		assert.Equal(t, allB, serviceRequestItemIDs(res.Items))
		for _, id := range serviceRequestItemIDs(res.Items) {
			assert.NotContains(t, allA, id, "租户 B 的响应里不得混入租户 A 的请求单")
		}
	})

	t.Run("已软删除的请求单不出现在列表里", func(t *testing.T) {
		target := allA[0]
		_, err := client.ServiceRequest.UpdateOneID(target).SetDeletedAt(time.Now()).Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := client.ServiceRequest.UpdateOneID(target).ClearDeletedAt().Save(ctx)
			require.NoError(t, err)
		})

		_, body, env := listA(t, "pageSize=100")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 24, res.Total, body)
		assert.NotContains(t, serviceRequestItemIDs(res.Items), target, body)
	})

	t.Run("未认证返回 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/service-requests", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	})

	t.Run("待审批收件箱同样是五键信封且页长只夹紧一次", func(t *testing.T) {
		// super_admin 走 targetLevel=0 分支：有待办审批记录、且请求单处于
		// submitted/manager_approved/it_approved 的才可见。种子给 i=0..3 各挂一条 pending 审批。
		_, body, env := do(t, adminA, "sr-a-admin", "super_admin", tenantA, "/api/v1/service-requests/approvals/pending", "pageSize=100")
		require.Equal(t, 0, env.Code, body)
		assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, envelopeKeys(t, env.Data), body)
		res := decode(t, body, env)
		assert.Equal(t, 4, res.Total, "只有挂了 pending 审批记录的 4 条请求单可见: body=%s", body)
		assert.Equal(t, 100, res.PageSize, body)

		_, clampedBody, clampedEnv := do(t, adminA, "sr-a-admin", "super_admin", tenantA, "/api/v1/service-requests/approvals/pending", "pageSize=5000")
		require.Equal(t, 0, clampedEnv.Code, clampedBody)
		clamped := decode(t, clampedBody, clampedEnv)
		assert.Equal(t, 20, clamped.PageSize, "越界页长回落默认页长: body=%s", clampedBody)
		assert.Equal(t, 1, clamped.TotalPages, clampedBody)
	})

	t.Run("待审批收件箱按租户收敛", func(t *testing.T) {
		_, body, env := do(t, adminB, "sr-b-admin", "super_admin", tenantB, "/api/v1/service-requests/approvals/pending", "pageSize=100")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 0, res.Total, "租户 B 没有 pending 审批记录，不得看到租户 A 的待办: body=%s", body)
		assert.JSONEq(t, "[]", jsonOf(t, env.Data, "items"), body)
	})
}

// seedServiceRequestListTenant 建租户 + super_admin/agent + 25 条同 created_at 的请求单。
// 布局固定：i<20 submitted / i>=20 delivered；i=23、24 申请人是 agent；i=10 处理人是 agent。
// i=0..3 各挂一条 pending 审批记录，用于待办收件箱。
func seedServiceRequestListTenant(ctx context.Context, t *testing.T, client *ent.Client, code string, count int) (tenantID, adminID, agentID int) {
	t.Helper()

	tenantID, adminID = seedServiceRequestListReadOnlyTenant(ctx, t, client, code, count)
	agent, err := client.User.Create().
		SetUsername(code + "-agent").
		SetEmail(code + "-agent@example.com").
		SetName("Service Request Agent").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	ids, err := client.ServiceRequest.Query().
		Where(servicerequest.TenantID(tenantID)).
		Order(ent.Asc(servicerequest.FieldID)).
		IDs(ctx)
	require.NoError(t, err)
	require.Len(t, ids, count)

	// i=23、24 改由 agent 申请，i=10 处理人改成 agent。
	for _, i := range []int{23, 24} {
		_, err := client.ServiceRequest.UpdateOneID(ids[i]).SetRequesterID(agent.ID).Save(ctx)
		require.NoError(t, err)
	}
	_, err = client.ServiceRequest.UpdateOneID(ids[10]).SetProcessorID(agent.ID).Save(ctx)
	require.NoError(t, err)

	// 前 4 条各挂一条 pending 审批记录（level 1），状态仍是 submitted，
	// 因此 super_admin 的待办收件箱恰好看到 4 条。
	for _, id := range ids[:4] {
		_, err := client.ServiceRequestApproval.Create().
			SetTenantID(tenantID).
			SetServiceRequestID(id).
			SetLevel(1).
			SetStep("manager").
			SetStatus("pending").
			SetCreatedAt(serviceRequestListCreatedAt).
			Save(ctx)
		require.NoError(t, err)
	}

	return tenantID, adminID, agent.ID
}

// seedServiceRequestListReadOnlyTenant 只建租户 + super_admin + count 条请求单，
// 全部同 created_at、同申请人，用于租户收敛与逐页拼接断言。
func seedServiceRequestListReadOnlyTenant(ctx context.Context, t *testing.T, client *ent.Client, code string, count int) (tenantID, adminID int) {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("Service request list " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	admin, err := client.User.Create().
		SetUsername(code + "-admin").
		SetEmail(code + "-admin@example.com").
		SetName("Service Request Admin").
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	for i := 0; i < count; i++ {
		status := "submitted"
		if i >= 20 {
			status = "delivered"
		}
		_, err := client.ServiceRequest.Create().
			SetTenantID(tenant.ID).
			SetCatalogID(1).
			SetRequesterID(admin.ID).
			SetStatus(status).
			SetTitle(fmt.Sprintf("SR-%s-%02d", code, i)).
			SetReason("验证服务请求列表分页夹紧与信封收敛").
			SetVersion(1).
			SetCreatedAt(serviceRequestListCreatedAt).
			SetUpdatedAt(serviceRequestListCreatedAt).
			Save(ctx)
		require.NoError(t, err)
	}

	return tenant.ID, admin.ID
}

func serviceRequestIDs(ctx context.Context, t *testing.T, client *ent.Client, tenantID int) []int {
	t.Helper()
	ids, err := client.ServiceRequest.Query().Where(servicerequest.TenantID(tenantID)).IDs(ctx)
	require.NoError(t, err)
	sort.Ints(ids)
	return ids
}

func serviceRequestProcessorIs(ctx context.Context, t *testing.T, client *ent.Client, requestID, processorID int) bool {
	t.Helper()
	row, err := client.ServiceRequest.Get(ctx, requestID)
	require.NoError(t, err)
	return row.ProcessorID == processorID
}
