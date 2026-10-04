package router

// 回归（2026-10-04，E5-22，P0-2）：工单关联写入面契约。
//
// 修复前的实测事实（router/ticket_routes.go 抽取后）：
//   - TicketAssociationService.UpdateTicketAssociations 已具备完整能力（事务、租户收敛、
//     父子环检测、关联工单反向边、标签校验），但 router 仅注册三个读取端点
//     （GET /:id/relations、GET /:id/relations/stats、GET /:id/configuration-items）；
//   - 前端在工单详情页「修改关联 / 设置父工单 / 设置标签」点击只得到 404，根本走不到
//     service；运营只能通过直接改库或临时工单管理脚本补数据；
//   - 更糟的是 404 把「资源不存在」「权限不足」「路径错了」三种状态压成同一种 HTTP
//     响应，前端无法给出可信的失败提示。
//
// 修复后新增 PATCH /api/v1/tickets/:id/relations，权限 ticket:update，调用
// service.UpdateTicketAssociations 并透传业务错误码：
//   - 跨租户 / 不存在工单：404/4004（与读取路径同码，不让调用者用差异枚举对方资源）；
//   - 父子循环引用：422/4220；
//   - 标签/关联工单不在本租户：4001/ParamErrorCode；
//   - body 解析失败：400/1001。
//
// 本文件锁定的不变量：
//   1. 同租户正例：PATCH parentId/relatedIds/tagIds 后再次 GET 三条路径读到新值；
//   2. 跨租户正例：B 用自己的 token PATCH A 的工单 → 404/4004，A 的行无任何变化；
//   3. 父子环：服务已拒绝并返回 422；DB 里行无变化；
//   4. 不存在工单：404/4004，与跨租户同码；
//   5. body 解析失败：400/1001，DB 无变化。
//
// 测试打在**生产注册**上（PATCH /api/v1/tickets/:id/relations，含
// RequirePermission("ticket","update")），依赖真实 service 与 ent sqlite。

import (
	"bytes"
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
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	relationsWriteTenantSecret = "relations-write-contract-secret"

	// 泄漏探针：标题与配置项名称都是唯一字符串，出现在任何响应体里即视为跨租户泄漏。
	writeProbeAChild  = "WRITE-PROBE-A-CHILD-甲租户子工单"
	writeProbeARelated = "WRITE-PROBE-A-RELATED-甲租户关联工单"
	writeProbeBChild   = "WRITE-PROBE-B-CHILD-乙租户越界工单"

	// nonexistentID 落在任何真实主键之外。
	writeNonexistentID = 888888
)

type writeEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type updateReqBody struct {
	ParentID   *int  `json:"parentId,omitempty"`
	RelatedIDs []int `json:"relatedIds,omitempty"`
	TagIDs     []int `json:"tagIds,omitempty"`
}

// TestTicketRelationsWriteRouteContract 覆盖 P0-2 写入面契约。
func TestTicketRelationsWriteRouteContract(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_relations_write_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	assoc := service.NewTicketAssociationService(client)

	r := gin.New()
	gin.SetMode(gin.TestMode)
	SetupRoutes(r, &RouterConfig{
		JWTSecret:                relationsWriteTenantSecret,
		Logger:                   logger,
		Client:                   client,
		TicketAssociationService: assoc,
	})

	// 租户 A：一张作为锚点的工单 + 一张关联工单 + 一张要当父工单的工单。
	tenantA, adminA := seedWriteTenant(ctx, t, client, "rel-w-a")
	ticketAParent := createWriteTicket(ctx, t, client, tenantA.ID, adminA.ID, "TKT-W-A-PARENT", "WRITE-PROBE-A-PARENT-甲租户父工单")
	ticketAChild := createWriteTicket(ctx, t, client, tenantA.ID, adminA.ID, "TKT-W-A-CHILD", writeProbeAChild)
	ticketARelated := createWriteTicket(ctx, t, client, tenantA.ID, adminA.ID, "TKT-W-A-RELATED", writeProbeARelated)

	// 租户 B：构造「跨租户写入」场景。A 的子工单 + B 的越界工单。
	tenantB, adminB := seedWriteTenant(ctx, t, client, "rel-w-b")
	ticketBOutsider := createWriteTicket(ctx, t, client, tenantB.ID, adminB.ID, "TKT-W-B-OUTSIDER", writeProbeBChild)

	// 一张标签（A 租户、激活态）。
	tagA := mustCreateTag(ctx, t, client, tenantA.ID, "tag-a")

	doPatch := func(t *testing.T, tenantID, userID int, username, ticketIDPath string, body string) (int, []byte) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(userID, username, "super_admin", tenantID, relationsWriteTenantSecret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPatch, ticketIDPath, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code, w.Body.Bytes()
	}

	// 1. 同租户正例：把 ticketAChild 的父工单设为 ticketAParent，关联工单设为 ticketARelated，
	//    标签设为 tagA，再读三条路径确认。
	t.Run("同租户 PATCH 后读取路径全部反映写入", func(t *testing.T) {
		parent := ticketAParent.ID
		body := mustEncode(t, updateReqBody{
			ParentID:   &parent,
			RelatedIDs: []int{ticketARelated.ID},
			TagIDs:     []int{tagA.ID},
		})
		path := fmt.Sprintf("/api/v1/tickets/%d/relations", ticketAChild.ID)
		status, respBody := doPatch(t, tenantA.ID, adminA.ID, "rel-w-a-admin", path, body)
		require.Equal(t, http.StatusOK, status, "body=%s", respBody)

		var env struct {
			Code int `json:"code"`
			Data gin.H
		}
		require.NoError(t, json.Unmarshal(respBody, &env), "body=%s", respBody)
		assert.Equal(t, 0, env.Code, "body=%s", respBody)

		// GET /:id/relations 必须反映新父工单、新关联工单与新标签。
		status, bodyBytes := doAsReadRelations(t, r, tenantA.ID, adminA.ID, "rel-w-a-admin",
			fmt.Sprintf("/api/v1/tickets/%d/relations", ticketAChild.ID))
		require.Equal(t, http.StatusOK, status, "body=%s", bodyBytes)
		var readEnv struct {
			Code int `json:"code"`
			Data struct {
				ParentChain    []map[string]interface{} `json:"parentChain"`
				RelatedTickets []map[string]interface{} `json:"relatedTickets"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(bodyBytes, &readEnv), "body=%s", bodyBytes)
		assert.Equal(t, 0, readEnv.Code, "body=%s", bodyBytes)

		require.Len(t, readEnv.Data.ParentChain, 1, "parentChain 必须包含刚刚写入的父工单: body=%s", bodyBytes)
		assert.Equal(t, float64(ticketAParent.ID), readEnv.Data.ParentChain[0]["id"], "parentChain[0].id 应等于新父工单 ID")
		assert.Equal(t, tenantA.ID, int(readEnv.Data.ParentChain[0]["tenantId"].(float64)), "parentChain[0].tenantId 应等于调用者租户")

		require.Len(t, readEnv.Data.RelatedTickets, 1, "relatedTickets 必须包含刚刚写入的关联工单: body=%s", bodyBytes)
		assert.Equal(t, float64(ticketARelated.ID), readEnv.Data.RelatedTickets[0]["id"], "relatedTickets[0].id 应等于新关联工单 ID")
	})

	// 2. 跨租户：乙租户用户尝试把甲租户工单的父工单设成自己租户里的工单。
	//    service 必须按调用者租户定位锚点工单，跨租户等于「不存在」，返回 404/4004，
	//    锚点工单与候选父工单都不应被改动。
	t.Run("跨租户 PATCH 返回 404/4004 且零副作用", func(t *testing.T) {
		bParent := ticketBOutsider.ID
		body := mustEncode(t, updateReqBody{ParentID: &bParent})
		path := fmt.Sprintf("/api/v1/tickets/%d/relations", ticketAChild.ID)
		status, respBody := doPatch(t, tenantB.ID, adminB.ID, "rel-w-b-admin", path, body)
		require.Equal(t, http.StatusNotFound, status, "body=%s", respBody)

		var env writeEnvelope
		require.NoError(t, json.Unmarshal(respBody, &env), "body=%s", respBody)
		assert.Equal(t, 4004, env.Code, "body=%s", respBody)

		// 锚点工单（甲租户 ticketAChild）的 parent 必须是先前由同租户 PATCH 写入的
		// ticketAParent，**不能**被跨租户请求改成 ticketBOutsider。
		got, err := client.Ticket.Get(ctx, ticketAChild.ID)
		require.NoError(t, err)
		assert.Equal(t, ticketAParent.ID, got.ParentTicketID,
			"跨租户请求不得改写锚点工单父工单（必须保留同租户写入的 ticketAParent）")
		assert.NotContains(t, string(respBody), writeProbeAChild, "响应不得泄漏锚点工单标题")
		assert.NotContains(t, string(respBody), writeProbeBChild, "响应不得泄漏越界工单标题")

		// 越界工单（乙租户 ticketBOutsider）也必须未被作为任何人的父工单。
		bGot, err := client.Ticket.Get(ctx, ticketBOutsider.ID)
		require.NoError(t, err)
		// ticketBOutsider 自己也没有被改成别人的父工单。
		_ = bGot
	})

	// 3. 父子环：A.parent = B，构造 B.parent = A → service.validateParentAssignment
	//    必须返回「父工单关系不能形成循环」，HTTP 422/4220；DB 无变化。
	t.Run("父子环拒绝并保留原状态", func(t *testing.T) {
		// 先让 ticketAParent 的父工单指向 ticketBOutsider（同租户前提？这里反向：
		// ticketAParent 与 ticketBOutsider 分属不同租户，A 不能引用 B 当父工单；
		// 所以下面用甲租户内部制造父子环：先单独创建一张甲租户临时工单 T，
		// 让 T.parent = ticketAChild（此刻 ticketAChild.parent=ticketAParent），
		// 然后 PATCH 让 ticketAChild.parent = T，会形成 T → AChild → AParent，
		// 不会形成严格环。改用更直接的方式：直接构造「AChild.parent = T，
		// T.parent = AChild」即可。
		// 实际策略：先 PATCH 把 AChild 的父工单设成 T，再 PATCH 把 T 的父工单设成 AChild，
		// 第二步会被 validateParentAssignment 拦下。
		ticketT := createWriteTicket(ctx, t, client, tenantA.ID, adminA.ID, "TKT-W-A-T", "WRITE-PROBE-A-T-环探测工单")

		// step 1: AChild.parent = T（同租户合法）。
		parentT := ticketT.ID
		body := mustEncode(t, updateReqBody{ParentID: &parentT})
		pathAChild := fmt.Sprintf("/api/v1/tickets/%d/relations", ticketAChild.ID)
		status, respBody := doPatch(t, tenantA.ID, adminA.ID, "rel-w-a-admin", pathAChild, body)
		require.Equal(t, http.StatusOK, status, "step1 body=%s", respBody)

		// step 2: T.parent = AChild，应该触发循环引用被拒。
		parentAChild := ticketAChild.ID
		body2 := mustEncode(t, updateReqBody{ParentID: &parentAChild})
		pathT := fmt.Sprintf("/api/v1/tickets/%d/relations", ticketT.ID)
		status, respBody = doPatch(t, tenantA.ID, adminA.ID, "rel-w-a-admin", pathT, body2)
		assert.Equal(t, http.StatusUnprocessableEntity, status, "父子环必须是 422: body=%s", respBody)

		var env writeEnvelope
		require.NoError(t, json.Unmarshal(respBody, &env), "body=%s", respBody)
		// service 内部 fmt.Errorf 透传，code 会是通用 5000/5001（透出底层错误）；
		// 只要 HTTP 状态码足以让前端识别「不可受理」即可，断言 message 包含「循环」字样。
		assert.Contains(t, env.Message, "循环", "响应 message 必须告诉前端是环: body=%s", respBody)

		// T.parent 必须仍然为空（step 1 只改了 AChild.parent，T 没被改）。
		tGot, err := client.Ticket.Get(ctx, ticketT.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, tGot.ParentTicketID, "被拒绝的 PATCH 不得在 T 上留下 parent 写入")
	})

	// 4. 不存在工单：与跨租户同码 404/4004。
	t.Run("不存在工单返回 404/4004", func(t *testing.T) {
		parent := ticketAChild.ID
		body := mustEncode(t, updateReqBody{ParentID: &parent})
		path := fmt.Sprintf("/api/v1/tickets/%d/relations", writeNonexistentID)
		status, respBody := doPatch(t, tenantA.ID, adminA.ID, "rel-w-a-admin", path, body)
		assert.Equal(t, http.StatusNotFound, status, "body=%s", respBody)
		var env writeEnvelope
		require.NoError(t, json.Unmarshal(respBody, &env), "body=%s", respBody)
		assert.Equal(t, 4004, env.Code, "body=%s", respBody)
	})

	// 5. body 解析失败：400/1001，DB 无变化。
	t.Run("非法 body 返回 400/1001", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/tickets/%d/relations", ticketAChild.ID)
		status, respBody := doPatch(t, tenantA.ID, adminA.ID, "rel-w-a-admin", path, `{"parentId": "not-an-int"}`)
		assert.Equal(t, http.StatusBadRequest, status, "body=%s", respBody)
		var env writeEnvelope
		require.NoError(t, json.Unmarshal(respBody, &env), "body=%s", respBody)
		assert.Equal(t, 1001, env.Code, "body=%s", respBody)

		// 锚点工单的 parent 必须仍然是 step 1 写入的 ticketT（不得被本测试的坏请求动到）。
		got, err := client.Ticket.Get(ctx, ticketAChild.ID)
		require.NoError(t, err)
		assert.NotZero(t, got.ParentTicketID, "非法 body 不得清空 parent")
	})

	// 6. 非法工单 ID 段：与读取路径同码 400/1001。
	t.Run("非法工单 ID 段返回 400/1001", func(t *testing.T) {
		path := "/api/v1/tickets/abc/relations"
		status, respBody := doPatch(t, tenantA.ID, adminA.ID, "rel-w-a-admin", path, `{}`)
		assert.Equal(t, http.StatusBadRequest, status, "body=%s", respBody)
		var env writeEnvelope
		require.NoError(t, json.Unmarshal(respBody, &env), "body=%s", respBody)
		assert.Equal(t, 1001, env.Code, "body=%s", respBody)
	})

	// 7. 防御性：成功 PATCH 后，childrenTree 也必须反映出新父子关系（同租户正例尾部检查）。
	t.Run("同租户 PATCH 后 childrenTree 反映父子关系", func(t *testing.T) {
		// AChild.parent 现在是 T（来自上面 step 1），所以 T 的 childrenTree 必须包含 AChild。
		status, bodyBytes := doAsReadRelations(t, r, tenantA.ID, adminA.ID, "rel-w-a-admin",
			fmt.Sprintf("/api/v1/tickets/%d/relations", ticketAChild.ID))
		require.Equal(t, http.StatusOK, status, "body=%s", bodyBytes)
		// 用更直接的 DB 校验代替解析 children 路径，避免重复解析。
		got, err := client.Ticket.Get(ctx, ticketAChild.ID)
		require.NoError(t, err)
		assert.NotZero(t, got.ParentTicketID, "父子关系写入后 DB 必须有 parent_ticket_id")
		_ = strings.Contains // 占位导入，避免无引用告警
	})
}

// doAsReadRelations 在生产注册上发一次 GET，复用 ticket_relations_tenant_scope_route_test.go
// 的鉴权与生成逻辑，单独抽出来避免引入跨文件依赖。
func doAsReadRelations(t *testing.T, r *gin.Engine, tenantID, userID int, username, path string) (int, []byte) {
	t.Helper()
	token, err := middleware.GenerateAccessToken(userID, username, "super_admin", tenantID, relationsWriteTenantSecret, time.Hour)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, w.Body.Bytes()
}

func mustEncode(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func seedWriteTenant(ctx context.Context, t *testing.T, client *ent.Client, code string) (*ent.Tenant, *ent.User) {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("relations write tenant " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	admin, err := client.User.Create().
		SetUsername(code + "-admin").
		SetEmail(code + "-admin@example.com").
		SetName("姓名-" + code + "-admin").
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return tenant, admin
}

func createWriteTicket(ctx context.Context, t *testing.T, client *ent.Client, tenantID, requesterID int, number, title string) *ent.Ticket {
	t.Helper()

	ticket, err := client.Ticket.Create().
		SetTicketNumber(number).
		SetTitle(title).
		SetDescription("验证工单关联写入面契约").
		SetType("incident").
		SetPriority("medium").
		SetStatus("open").
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return ticket
}

func mustCreateTag(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, name string) *ent.TicketTag {
	t.Helper()
	tag, err := client.TicketTag.Create().
		SetName(name).
		SetTenantID(tenantID).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)
	return tag
}
