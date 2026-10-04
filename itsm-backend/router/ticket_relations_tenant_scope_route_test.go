package router

// 回归（2026-10-04，E4-35）：工单关联的三个读取端点必须按调用者租户收敛。
//
// 修复前的实测事实（service/ticketassociationservice.go）：
//   - GetTicketDependencies/GetRelatedTickets/getParentChain/getChildrenTree/
//     GetConfigurationItems/getTicketForAssociation 全部只用 ticket.ID(ticketID)
//     定位工单，没有任何 tenant_id 谓词，也不校验请求者租户；
//   - 因此带 ticket:read 的乙租户用户只要拿到甲租户的工单 ID，就能读到甲租户工单的
//     标题、描述、优先级、状态与承办人 assignedTo（buildTicketResponse 逐字段透出）；
//   - getChildrenTree 更严重：Where(ticket.ParentTicketID(ticketID)) 是**全库**匹配，
//     所以甲租户的父工单会把乙租户里 parent_ticket_id 恰好指向它的子工单一起返回，
//     关联表（parent_ticket_id / related_tickets / configuration_items 边）因此
//     成为绕过租户隔离的第二条读取通道。
//
// 本文件锁定的不变量（AGENTS「身份、租户与数据范围」：tenant scope 必须覆盖
// Get/Only/Exist/List 与关联表）：
//   1. 同租户正例：只返回本租户的行，parentChain/childrenTree/relatedTickets 里
//      每个 item 的 tenantId 都等于调用者租户；
//   2. 跨租户探测：与「工单不存在」返回同一个 404/4004，不确认对方资源是否存在；
//   3. 三种失败都零泄漏：响应体不含对方工单标题，也不含本租户以外任何行；
//   4. 不存在的工单同样返回 404/4004（不得再回 500/5001，也不得把底层错误串透出），
//      且与跨租户响应**逐字节相同**——否则调用者能靠状态码差异枚举对方租户的工单 ID；
//   5. 空关联是 `[]` 而不是 `null`，与平台列表契约（E4 系列「空列表 `[]` 而非 `null`」）
//      一致；否则前端要为同一集合准备两种空值形状。
//
// 测试打在**生产注册**上（router/ticket_routes.go:88/:104/:129，含
// RequirePermission("ticket","read")），解码目标是本地 wire 结构而不引用 service
// 类型，因此修复前也能编译。

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
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	relationsTenantSecret = "relations-tenant-scope-secret"

	// 泄漏探针：标题与配置项名称都是唯一字符串，出现在任何响应体里即视为跨租户泄漏。
	probeAParentTitle = "PROBE-A-PARENT-甲租户父工单"
	probeAChildTitle  = "PROBE-A-CHILD-甲租户子工单"
	probeARelTitle    = "PROBE-A-RELATED-甲租户关联工单"
	probeBChildTitle  = "PROBE-B-CHILD-乙租户越界子工单"
	probeACIName      = "PROBE-A-CI"
	probeBCIName      = "PROBE-B-CI"

	// nonexistentID 落在任何真实主键之外，用于证明「跨租户」与「不存在」不可区分。
	nonexistentID = 999999
)

// relTicketWire 只声明本片断言用到的字段。
type relTicketWire struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	TenantID int    `json:"tenantId"`
}

type relationsWire struct {
	ParentChain    []relTicketWire `json:"parentChain"`
	ChildrenTree   []relTicketWire `json:"childrenTree"`
	RelatedTickets []relTicketWire `json:"relatedTickets"`
}

type relationsEnvelope struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    relationsWire `json:"data"`
}

type ciItemWire struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestTicketRelationsRoutesTenantScope(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_relations_tenant_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	assoc := service.NewTicketAssociationService(client)

	r := gin.New()
	gin.SetMode(gin.TestMode)
	SetupRoutes(r, &RouterConfig{
		JWTSecret:                relationsTenantSecret,
		Logger:                   logger,
		Client:                   client,
		TicketAssociationService: assoc,
	})

	// 租户 A：父工单 + 子工单 + 关联工单 + 归属本租户的配置项。
	tenantA, adminA := seedRelationsTenant(ctx, t, client, "rel-a")
	ticketAParent := createRelationsTicket(ctx, t, client, tenantA.ID, adminA.ID, "TKT-REL-A-PARENT", probeAParentTitle)
	ticketAChild := createRelationsTicket(ctx, t, client, tenantA.ID, adminA.ID, "TKT-REL-A-CHILD", probeAChildTitle)
	ticketARelated := createRelationsTicket(ctx, t, client, tenantA.ID, adminA.ID, "TKT-REL-A-RELATED", probeARelTitle)
	// 三条路径都为空的工单，用来锁定空集合的序列化形状。
	ticketALone := createRelationsTicket(ctx, t, client, tenantA.ID, adminA.ID, "TKT-REL-A-LONE", "PROBE-A-LONE-无关联工单")

	if _, err := client.Ticket.UpdateOne(ticketAChild).SetParentTicketID(ticketAParent.ID).Save(ctx); err != nil {
		t.Fatalf("挂子工单失败: %v", err)
	}
	if err := ticketAParent.Update().AddRelatedTickets(ticketARelated).Exec(ctx); err != nil {
		t.Fatalf("建立关联工单边失败: %v", err)
	}
	ciA := createRelationsCI(ctx, t, client, tenantA.ID, probeACIName, ticketAParent)

	// 租户 B：自己的工单与配置项，外加一处**跨租户关系注入**（正是缺少校验的写入路径会
	// 造出来的存量形态）：B 的子工单把 parent_ticket_id 指向 A 的父工单。
	// 配置项侧 B 挂在自家工单上（CI→tickets 是 O2M，一张工单只属于一个配置项，
	// 所以「B 的配置项挂到 A 的工单」这条注入不成立，改用自家工单验证过滤）。
	tenantB, adminB := seedRelationsTenant(ctx, t, client, "rel-b")
	ticketBChild := createRelationsTicket(ctx, t, client, tenantB.ID, adminB.ID, "TKT-REL-B-CHILD", probeBChildTitle)
	if _, err := client.Ticket.UpdateOne(ticketBChild).SetParentTicketID(ticketAParent.ID).Save(ctx); err != nil {
		t.Fatalf("注入跨租户口子工单失败: %v", err)
	}
	ciB := createRelationsCI(ctx, t, client, tenantB.ID, probeBCIName, ticketBChild)

	doAs := func(t *testing.T, tenantID, userID int, username, path string) (int, []byte) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(userID, username, "super_admin", tenantID, relationsTenantSecret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code, w.Body.Bytes()
	}

	relationsPath := fmt.Sprintf("/api/v1/tickets/%d/relations", ticketAParent.ID)
	statsPath := fmt.Sprintf("/api/v1/tickets/%d/relations/stats", ticketAParent.ID)
	ciPath := fmt.Sprintf("/api/v1/tickets/%d/configuration-items", ticketAParent.ID)

	// shadowPath 把三个真实路径的 ID 段换成不存在的工单 ID，静态段保持不变，
	// 用于把「跨租户」响应和「不存在」响应做逐字节比较。
	shadowPath := func(path string) string {
		return strings.Replace(path,
			fmt.Sprintf("/tickets/%d/", ticketAParent.ID),
			fmt.Sprintf("/tickets/%d/", nonexistentID), 1)
	}

	// 1. 同租户正例：三条读取路径都只返回租户 A 的行。
	t.Run("同租户只拿到本租户的关联行", func(t *testing.T) {
		status, body := doAs(t, tenantA.ID, adminA.ID, "rel-a-admin", relationsPath)
		require.Equal(t, http.StatusOK, status, "body=%s", body)

		var env relationsEnvelope
		require.NoError(t, json.Unmarshal(body, &env), "body=%s", body)
		require.Equal(t, 0, env.Code, "body=%s", body)

		// 父链：请求的工单本身没有父工单。
		assert.Empty(t, env.Data.ParentChain, "body=%s", body)

		// 子工单树：必须只有 A 的子工单；B 注入的那条不能出现。
		require.Len(t, env.Data.ChildrenTree, 1, "childrenTree 必须按租户收敛: body=%s", body)
		assert.Equal(t, probeAChildTitle, env.Data.ChildrenTree[0].Title)
		assert.Equal(t, tenantA.ID, env.Data.ChildrenTree[0].TenantID)

		// 关联工单：只有 A 的那条。
		require.Len(t, env.Data.RelatedTickets, 1, "relatedTickets 必须按租户收敛: body=%s", body)
		assert.Equal(t, probeARelTitle, env.Data.RelatedTickets[0].Title)
		assert.Equal(t, tenantA.ID, env.Data.RelatedTickets[0].TenantID)

		assert.NotContains(t, string(body), probeBChildTitle, "不得返回其他租户的子工单")

		status, body = doAs(t, tenantA.ID, adminA.ID, "rel-a-admin", ciPath)
		require.Equal(t, http.StatusOK, status, "body=%s", body)
		var ciEnv struct {
			Code int          `json:"code"`
			Data []ciItemWire `json:"data"`
		}
		require.NoError(t, json.Unmarshal(body, &ciEnv), "body=%s", body)
		require.Equal(t, 0, ciEnv.Code, "body=%s", body)
		names := make([]string, 0, len(ciEnv.Data))
		for _, item := range ciEnv.Data {
			names = append(names, item.Name)
			assert.Equal(t, ciA.ID, item.ID, "配置项必须是本租户挂在工单上的那一个")
		}
		assert.Equal(t, []string{probeACIName}, names, "配置项列表必须按租户收敛: body=%s", body)
		assert.NotContains(t, string(body), probeBCIName, "不得返回其他租户的配置项")
	})

	// 2. 跨租户探测：三个端点都必须与「工单不存在」同码，且零泄漏。
	t.Run("跨租户读取三个端点都是 404 且零泄漏", func(t *testing.T) {
		for _, path := range []string{relationsPath, statsPath, ciPath} {
			status, body := doAs(t, tenantB.ID, adminB.ID, "rel-b-admin", path)
			assert.Equal(t, http.StatusNotFound, status, "%s 跨租户必须与不存在同码: body=%s", path, body)

			var env struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(body, &env), "body=%s", body)
			assert.Equal(t, 4004, env.Code, "%s body=%s", path, body)

			assert.NotContains(t, string(body), probeAParentTitle, "%s 不得返回对方工单标题", path)
			assert.NotContains(t, string(body), probeAChildTitle, "%s 不得返回对方子工单", path)
			assert.NotContains(t, string(body), probeARelTitle, "%s 不得返回对方关联工单", path)
			assert.NotContains(t, string(body), probeACIName, "%s 不得返回对方配置项", path)

			// 关键不变量：跨租户与「这张工单根本不存在」必须逐字节不可区分，
			// 否则调用者能用 404 与 500/5001 的差异枚举对方租户的工单 ID。
			nfStatus, nfBody := doAs(t, tenantB.ID, adminB.ID, "rel-b-admin", shadowPath(path))
			assert.Equal(t, http.StatusNotFound, nfStatus, "%s 不存在的工单必须是 404: body=%s", path, nfBody)
			assert.Equal(t, string(nfBody), string(body), "%s 跨租户与不存在不得可区分", path)
		}
	})

	// 3. 不存在的工单返回 404/4004 而不是 500/5001，也不透出底层错误串。
	t.Run("不存在的工单返回 404 而不是 500", func(t *testing.T) {
		for _, path := range []string{relationsPath, statsPath, ciPath} {
			status, body := doAs(t, tenantA.ID, adminA.ID, "rel-a-admin", shadowPath(path))
			assert.Equal(t, http.StatusNotFound, status, "%s body=%s", path, body)
			var env struct {
				Code int `json:"code"`
			}
			require.NoError(t, json.Unmarshal(body, &env), "body=%s", body)
			assert.Equal(t, 4004, env.Code, "body=%s", body)
			assert.NotContains(t, string(body), "获取工单失败", "底层错误串不得透出: body=%s", body)
			assert.NotContains(t, string(body), "ent:", "Ent 错误串不得透出: body=%s", body)
		}
	})

	// 4. 非法 ID 仍是参数错误，不与「不存在」混淆。
	t.Run("非法工单 ID 返回 400", func(t *testing.T) {
		for _, path := range []string{
			"/api/v1/tickets/abc/relations",
			"/api/v1/tickets/abc/relations/stats",
			"/api/v1/tickets/abc/configuration-items",
		} {
			status, body := doAs(t, tenantA.ID, adminA.ID, "rel-a-admin", path)
			assert.Equal(t, http.StatusBadRequest, status, "%s body=%s", path, body)
			var env struct {
				Code int `json:"code"`
			}
			require.NoError(t, json.Unmarshal(body, &env), "body=%s", body)
			assert.Equal(t, 1001, env.Code, "%s body=%s", path, body)
		}
	})

	// 5. 没有任何关联的工单：三条集合与配置项列表都序列化为 `[]`。
	t.Run("空关联返回空数组而不是 null", func(t *testing.T) {
		status, body := doAs(t, tenantA.ID, adminA.ID, "rel-a-admin",
			fmt.Sprintf("/api/v1/tickets/%d/relations", ticketALone.ID))
		require.Equal(t, http.StatusOK, status, "body=%s", body)
		for _, key := range []string{`"parentChain":[]`, `"childrenTree":[]`, `"relatedTickets":[]`} {
			assert.Contains(t, string(body), key, "空集合必须是 []: body=%s", body)
		}

		status, body = doAs(t, tenantA.ID, adminA.ID, "rel-a-admin",
			fmt.Sprintf("/api/v1/tickets/%d/configuration-items", ticketALone.ID))
		require.Equal(t, http.StatusOK, status, "body=%s", body)
		assert.Contains(t, string(body), `"data":[]`, "配置项空集合必须是 []: body=%s", body)
	})

	// 编译期使用，确保 fixture 变量未被编译器优化掉（B 侧行只用于注入关系）。
	_ = ciB
	_ = tenantB
}

func seedRelationsTenant(ctx context.Context, t *testing.T, client *ent.Client, code string) (*ent.Tenant, *ent.User) {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("relations tenant " + code).
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

func createRelationsTicket(ctx context.Context, t *testing.T, client *ent.Client, tenantID, requesterID int, number, title string) *ent.Ticket {
	t.Helper()

	ticket, err := client.Ticket.Create().
		SetTicketNumber(number).
		SetTitle(title).
		SetDescription("验证工单关联读取路径的租户收敛").
		SetType("incident").
		SetPriority("medium").
		SetStatus("open").
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return ticket
}

func createRelationsCI(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, name string, ticket *ent.Ticket) *ent.ConfigurationItem {
	t.Helper()

	ciType, err := client.CIType.Create().
		SetName("server-" + name).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	ci, err := client.ConfigurationItem.Create().
		SetName(name).
		SetCiTypeID(ciType.ID).
		SetCiType("server").
		SetStatus("active").
		SetSerialNumber("SN-" + name).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, ci.Update().AddTickets(ticket).Exec(ctx))
	return ci
}
