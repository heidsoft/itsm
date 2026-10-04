package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	assignmentSmartHandler "itsm-backend/handlers/assignment_smart"
	automationRuleHandler "itsm-backend/handlers/automation_rule"
	ticketAttachmentHandler "itsm-backend/handlers/ticket_attachment"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-04 边缘功能收口 E4-6h）：
//   - GET /api/v1/tickets/assignment-rules 返回 {rules,total}
//   - GET /api/v1/tickets/automation-rules  返回 {rules,total}
//   - GET /api/v1/tickets/:id/attachments   返回 {attachments,total}
//
// 三个端点实测都**不分页**（service 整表取回、handler 写 Total: len(...)），所以本批
// 只把集合键改成平台规定的 items，并保留诚实的两键形状：**不补 page/pageSize/totalPages**，
// 否则等于用响应声明伪造一个不存在的分页协议（AGENTS「能力状态与失败语义」）。
//
// 测试打在**生产注册**上（SetupRoutes → router/ticket_routes.go，含
// RequirePermission("system_config","read") 与 RequirePermission("ticket","read")）。
// 解码目标是本地 wire 结构而不引用 dto，因此修复前也能编译，证明的是运行时行为。

const ticketRuleListSecret = "ticket-rule-list-secret"

type ticketRuleListTenant struct {
	tenantID int
	userID   int
	username string
	ticketID int
}

// rawEnvelope 保留 data 为原始 JSON，断言「键集合恰好是这两个」时才能发现多出来的分页键。
type rawEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type attachmentListWire struct {
	Items []struct {
		ID       int    `json:"id"`
		FileName string `json:"fileName"`
		TicketID int    `json:"ticketId"`
		Uploader struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		} `json:"uploader"`
	} `json:"items"`
	Total int `json:"total"`
}

func ticketRuleListKeys(t *testing.T, body []byte) []string {
	t.Helper()
	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &data), "body=%s", body)
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestTicketRuleAndAttachmentListRoutesEnvelope(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_ticket_rule_list_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	r := setupTicketRuleListRouter(t, client, logger)

	tenantA := seedTicketRuleTenant(ctx, t, client, "rule-a", 3)
	tenantB := seedTicketRuleTenant(ctx, t, client, "rule-b", 1)

	doAs := func(t *testing.T, tenant ticketRuleListTenant, path string) (*httptest.ResponseRecorder, []byte) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(tenant.userID, tenant.username, "super_admin", tenant.tenantID, ticketRuleListSecret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w, w.Body.Bytes()
	}

	// 1. 三个端点都只返回诚实的两键信封，旧集合键永不存在。
	for _, tc := range []struct {
		name      string
		path      string
		legacyKey string
	}{
		{"分配规则列表", "/api/v1/tickets/assignment-rules", "rules"},
		{"自动化规则列表", "/api/v1/tickets/automation-rules", "rules"},
		{"工单附件列表", fmt.Sprintf("/api/v1/tickets/%d/attachments", tenantA.ticketID), "attachments"},
	} {
		t.Run(tc.name+"只返回 items+total 两键", func(t *testing.T) {
			w, body := doAs(t, tenantA, tc.path)
			require.Equal(t, http.StatusOK, w.Code, "body=%s", body)

			var envelope rawEnvelope
			require.NoError(t, json.Unmarshal(body, &envelope), "body=%s", body)
			assert.Equal(t, 0, envelope.Code)
			assert.Equal(t, []string{"items", "total"}, ticketRuleListKeys(t, envelope.Data), "body=%s", body)
			assert.NotContains(t, string(envelope.Data), `"`+tc.legacyKey+`":`)
			// 不分页的端点不得出现分页键。
			assert.NotContains(t, string(envelope.Data), `"page"`)
			assert.NotContains(t, string(envelope.Data), `"pageSize"`)
			assert.NotContains(t, string(envelope.Data), `"totalPages"`)
		})
	}

	// 2. total 是真实条数、items 长度与之一致（不是「当前页长度冒充总数」）。
	t.Run("total 等于 items 长度", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			path string
			want int
		}{
			{"分配规则", "/api/v1/tickets/assignment-rules", 3},
			{"自动化规则", "/api/v1/tickets/automation-rules", 3},
		} {
			w, body := doAs(t, tenantA, tc.path)
			require.Equal(t, http.StatusOK, w.Code, "body=%s", body)
			var data struct {
				Items []json.RawMessage `json:"items"`
				Total int               `json:"total"`
			}
			dataJSON := rawData(t, body)
			require.NoError(t, json.Unmarshal(dataJSON, &data), "body=%s", body)
			assert.Equal(t, tc.want, data.Total, tc.name)
			assert.Len(t, data.Items, tc.want, tc.name)
		}
	})

	// 3. 空结果序列化为 []，不是 null（前端按数组渲染）。
	t.Run("空列表序列化为 []", func(t *testing.T) {
		empty := seedTicketRuleTenant(ctx, t, client, "rule-empty", 0)
		for _, path := range []string{"/api/v1/tickets/assignment-rules", "/api/v1/tickets/automation-rules"} {
			w, body := doAs(t, empty, path)
			require.Equal(t, http.StatusOK, w.Code, "body=%s", body)
			assert.Contains(t, string(body), `"items":[]`)
			assert.NotContains(t, string(body), `"items":null`)
		}
	})

	// 4. 租户收敛：B 的规则列表不含 A 的规则名。
	t.Run("租户 B 只看到自己的规则", func(t *testing.T) {
		for _, path := range []string{"/api/v1/tickets/assignment-rules", "/api/v1/tickets/automation-rules"} {
			w, body := doAs(t, tenantB, path)
			require.Equal(t, http.StatusOK, w.Code, "body=%s", body)
			var data struct {
				Items []struct {
					Name string `json:"name"`
				} `json:"items"`
				Total int `json:"total"`
			}
			require.NoError(t, json.Unmarshal(rawData(t, body), &data), "body=%s", body)
			assert.Equal(t, 1, data.Total, path)
			require.Len(t, data.Items, 1, path)
			assert.Contains(t, data.Items[0].Name, "rule-b", path)
		}
	})

	// 5. 跨租户工单附件：fail closed 到 404/4004，且不确认对方工单存在。
	t.Run("跨租户工单附件不泄漏", func(t *testing.T) {
		w, body := doAs(t, tenantB, fmt.Sprintf("/api/v1/tickets/%d/attachments", tenantA.ticketID))
		assert.Equal(t, http.StatusNotFound, w.Code, "body=%s", body)
		var envelope rawEnvelope
		require.NoError(t, json.Unmarshal(body, &envelope), "body=%s", body)
		assert.Equal(t, 4004, envelope.Code, "body=%s", body)
		assert.NotContains(t, string(body), "secret-file-rule-a", "跨租户不得列出对方附件")
	})

	// 6. 本租户工单附件按租户谓词返回，items 携带文件名与上传人。
	t.Run("本租户工单附件返回 items", func(t *testing.T) {
		w, body := doAs(t, tenantA, fmt.Sprintf("/api/v1/tickets/%d/attachments", tenantA.ticketID))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", body)
		var envelope rawEnvelope
		require.NoError(t, json.Unmarshal(body, &envelope), "body=%s", body)
		assert.Equal(t, []string{"items", "total"}, ticketRuleListKeys(t, envelope.Data))

		var data attachmentListWire
		require.NoError(t, json.Unmarshal(envelope.Data, &data), "body=%s", body)
		assert.Equal(t, 2, data.Total)
		require.Len(t, data.Items, 2)
		for _, item := range data.Items {
			assert.Equal(t, tenantA.ticketID, item.TicketID)
			assert.Contains(t, item.FileName, "secret-file-rule-a")
			assert.Equal(t, tenantA.username, item.Uploader.Username)
		}
	})

	// 7. 未认证不得返回列表载荷。
	t.Run("未认证不得返回列表载荷", func(t *testing.T) {
		for _, path := range []string{
			"/api/v1/tickets/assignment-rules",
			"/api/v1/tickets/automation-rules",
			fmt.Sprintf("/api/v1/tickets/%d/attachments", tenantA.ticketID),
		} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code, path)
			assert.NotContains(t, w.Body.String(), `"items"`)
		}
	})
}

// assignRecommendationListWire 只按后端 dto.AssignmentRecommendation 的真实字段声明。
// 前端旧类型曾发明 userName/userEmail/userAvatar/factors，这里刻意不放这些键，
// 好让断言能证明服务端从未发出它们。
type assignRecommendationListWire struct {
	Items []struct {
		UserID     int      `json:"userId"`
		Username   string   `json:"username"`
		Name       string   `json:"name"`
		Email      string   `json:"email"`
		Score      float64  `json:"score"`
		Reason     string   `json:"reason"`
		Workload   int      `json:"workload"`
		Skills     []string `json:"skills"`
		Categories []int    `json:"categories"`
	} `json:"items"`
	Total int `json:"total"`
}

// 回归（2026-10-04 边缘功能收口 E4-27a）：
// GET /api/v1/tickets/assign-recommendations/:id 旧响应把集合放在 recommendations 键下，
// 结构体名又不含 List，因此静默逃逸过 tests/contract 的信封棘轮扫描。本批把它收敛为
// 平台唯一的 {items,total}（实测不分页，service 整表取回、handler 写 Total=len(items)），
// 并改名 AssignRecommendationListResponse 让它落进守卫范围。
//
// 测试打在**生产注册**上（SetupRoutes → router/ticket_routes.go:247，含
// RequirePermission("system_config","read")）。
func TestTicketAssignRecommendationsRouteEnvelope(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_assign_reco_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	r := setupTicketRuleListRouter(t, client, logger)

	tenantA := seedTicketRuleTenant(ctx, t, client, "reco-a", 1)
	tenantB := seedTicketRuleTenant(ctx, t, client, "reco-b", 1)
	// 生产路由是 /tickets/assign-recommendations/:id，id 即工单 ID。
	path := fmt.Sprintf("/api/v1/tickets/assign-recommendations/%d", tenantA.ticketID)

	doAs := func(t *testing.T, tenant ticketRuleListTenant, target string) (*httptest.ResponseRecorder, []byte) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(tenant.userID, tenant.username, "super_admin", tenant.tenantID, ticketRuleListSecret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w, w.Body.Bytes()
	}

	// 再给租户 A 加一名活跃可分配用户，让 total==len(items) 不是一页数据自证。
	extra, err := client.User.Create().
		SetUsername("reco-a-agent").
		SetEmail("reco-a-agent@example.com").
		SetName("reco-a-agent").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantA.tenantID).
		Save(ctx)
	require.NoError(t, err)

	// 1. 信封恰好是诚实的两键：领域名集合键永不存在，也不许出现伪造的分页键。
	t.Run("只返回 items+total 两键", func(t *testing.T) {
		w, body := doAs(t, tenantA, path)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", body)

		var envelope rawEnvelope
		require.NoError(t, json.Unmarshal(body, &envelope), "body=%s", body)
		assert.Equal(t, 0, envelope.Code)
		assert.Equal(t, []string{"items", "total"}, ticketRuleListKeys(t, envelope.Data), "body=%s", body)
		assert.NotContains(t, string(envelope.Data), `"recommendations":`)
		assert.NotContains(t, string(envelope.Data), `"page"`)
		assert.NotContains(t, string(envelope.Data), `"pageSize"`)
		assert.NotContains(t, string(envelope.Data), `"totalPages"`)
	})

	// 2. total 等于 items 真实条数，item 字段用后端 DTO 的 camelCase；
	//    前端旧类型发明的 userName/userEmail/userAvatar/factors 永不存在。
	t.Run("total 等于 items 长度且字段与后端 DTO 一致", func(t *testing.T) {
		w, body := doAs(t, tenantA, path)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", body)

		var data assignRecommendationListWire
		require.NoError(t, json.Unmarshal(rawData(t, body), &data), "body=%s", body)
		assert.Equal(t, 2, data.Total)
		require.Len(t, data.Items, data.Total, "total 必须等于 items 真实长度")

		names := []string{data.Items[0].Username, data.Items[1].Username}
		sort.Strings(names)
		assert.Equal(t, []string{"reco-a-admin", "reco-a-agent"}, names)
		assert.Contains(t, []int{data.Items[0].UserID, data.Items[1].UserID}, extra.ID)
		for _, item := range data.Items {
			assert.NotZero(t, item.UserID)
			assert.Equal(t, item.Username+"@example.com", item.Email)
			assert.Equal(t, item.Username, item.Name)
			assert.Zero(t, item.Workload, "fixture 里没有指派给任何人的工单")
			assert.NotEmpty(t, item.Reason)
		}
		assert.NotContains(t, string(body), `"userName"`)
		assert.NotContains(t, string(body), `"userEmail"`)
		assert.NotContains(t, string(body), `"userAvatar"`)
		assert.NotContains(t, string(body), `"factors"`)
	})

	// 3. 候选用户按租户谓词取（user.TenantIDEQ），不得出现对方租户用户。
	t.Run("租户 A 的推荐不含租户 B 的用户", func(t *testing.T) {
		w, body := doAs(t, tenantA, path)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", body)
		assert.NotContains(t, string(body), "reco-b-admin", "跨租户用户不得进入推荐列表")
	})

	// 4. 无候选用户时序列化成 []，不是 null（前端按数组渲染）。
	t.Run("空推荐序列化为 []", func(t *testing.T) {
		emptyTenantID, emptyUserID := seedInactiveOnlyTenant(ctx, t, client, "reco-empty")
		emptyTicketID, err := createRuleListTicket(ctx, t, client, emptyTenantID, emptyUserID, "reco-empty")
		require.NoError(t, err)

		empty := ticketRuleListTenant{
			tenantID: emptyTenantID,
			userID:   emptyUserID,
			username: "reco-empty-admin",
			ticketID: emptyTicketID,
		}
		w, body := doAs(t, empty, fmt.Sprintf("/api/v1/tickets/assign-recommendations/%d", emptyTicketID))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", body)
		assert.Contains(t, string(body), `"items":[]`)
		assert.NotContains(t, string(body), `"items":null`)
		assert.Contains(t, string(body), `"total":0`)
	})

	// 5. 跨租户工单 fail closed，且不确认对方工单存在、不返回任何推荐载荷。
	//    实测映射是 500/5001（handler 走 common.FailWithErr，见 common/response.go:183），
	//    语义上应为 404/4004；该错配属 E4-5 错误分类收敛范围，E4-5 落地时需同步把这两行
	//    改成 StatusNotFound / 4004。这里先锁定「绝不泄漏」这一半。
	t.Run("跨租户工单不泄漏推荐载荷", func(t *testing.T) {
		w, body := doAs(t, tenantB, path)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", body)
		var envelope rawEnvelope
		require.NoError(t, json.Unmarshal(body, &envelope), "body=%s", body)
		assert.Equal(t, 5001, envelope.Code)
		assert.NotContains(t, string(body), `"items"`, "失败响应不得带列表载荷")
		assert.NotContains(t, string(body), "reco-a-admin")
		assert.NotContains(t, string(body), "reco-a-agent")
	})

	// 6. 非法 ID 参数 → 400/1001，且不回落到某个默认工单。
	t.Run("非法工单ID返回参数错误", func(t *testing.T) {
		w, body := doAs(t, tenantA, "/api/v1/tickets/assign-recommendations/not-a-number")
		assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", body)
		var envelope rawEnvelope
		require.NoError(t, json.Unmarshal(body, &envelope), "body=%s", body)
		assert.Equal(t, 1001, envelope.Code)
		assert.NotContains(t, string(body), `"items"`)
	})

	// 7. 未认证不得返回推荐载荷。
	t.Run("未认证不得返回推荐载荷", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body)
		assert.NotContains(t, w.Body.String(), `"items"`)
	})
}

// seedInactiveOnlyTenant 建一个只有停用用户的租户，用来证明推荐列表的空形状。
func seedInactiveOnlyTenant(ctx context.Context, t *testing.T, client *ent.Client, code string) (int, int) {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("Recommendation empty " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	admin, err := client.User.Create().
		SetUsername(code + "-admin").
		SetEmail(code + "-admin@example.com").
		SetName(code + "-admin").
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(false).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID, admin.ID
}

func rawData(t *testing.T, body []byte) []byte {
	t.Helper()
	var envelope rawEnvelope
	require.NoError(t, json.Unmarshal(body, &envelope), "body=%s", body)
	require.Equal(t, 0, envelope.Code, "body=%s", body)
	return envelope.Data
}

func setupTicketRuleListRouter(t *testing.T, client *ent.Client, logger *zap.SugaredLogger) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	assignmentRuleService := service.NewTicketAssignmentRuleService(client, logger)
	SetupRoutes(r, &RouterConfig{
		JWTSecret: ticketRuleListSecret,
		Logger:    logger,
		Client:    client,
		TicketAssignmentSmartHandler: assignmentSmartHandler.NewHandler(
			service.NewTicketAssignmentSmartService(
				client,
				logger,
				service.NewTicketAssignmentService(client, logger),
				assignmentRuleService,
			),
			assignmentRuleService,
			logger,
		),
		TicketAutomationRuleHandler: automationRuleHandler.NewHandler(service.NewTicketAutomationRuleService(client, logger), logger),
		TicketAttachmentHandler:     ticketAttachmentHandler.NewHandler(service.NewTicketAttachmentService(client, logger), logger),
	})
	return r
}

// seedTicketRuleTenant 建租户 + 一名 super_admin + 该租户自己的工单与附件，并按 count
// 建同名的分配规则与自动化规则（规则名带租户码，便于断言归属）。
func seedTicketRuleTenant(ctx context.Context, t *testing.T, client *ent.Client, code string, count int) ticketRuleListTenant {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("Rule list " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	username := code + "-admin"
	admin, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "@example.com").
		SetName(username).
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	result := ticketRuleListTenant{tenantID: tenant.ID, userID: admin.ID, username: username}

	result.ticketID, err = createRuleListTicket(ctx, t, client, tenant.ID, admin.ID, code)
	require.NoError(t, err)

	for i := 1; i <= count; i++ {
		_, err := client.TicketAssignmentRule.Create().
			SetName(fmt.Sprintf("分配规则 %s %02d", code, i)).
			SetPriority(i).
			SetConditions([]map[string]interface{}{{"field": "category", "operator": "equals", "value": "network"}}).
			SetActions(map[string]interface{}{"assign_to": "team"}).
			SetIsActive(true).
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.TicketAutomationRule.Create().
			SetName(fmt.Sprintf("自动化规则 %s %02d", code, i)).
			SetPriority(i).
			SetConditions([]map[string]interface{}{{"field": "priority", "operator": "equals", "value": "critical"}}).
			SetActions([]map[string]interface{}{{"type": "notify", "target": "owner"}}).
			SetIsActive(true).
			SetCreatedBy(admin.ID).
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	// 每个租户在自己的工单上挂两个附件，文件名带租户码。
	for i := 1; i <= 2; i++ {
		_, err := client.TicketAttachment.Create().
			SetTicketID(result.ticketID).
			SetFileName(fmt.Sprintf("secret-file-%s-%d.txt", code, i)).
			SetFilePath("/tmp/attachments").
			SetFileSize(1024).
			SetFileType("text/plain").
			SetUploadedBy(admin.ID).
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	return result
}

// createRuleListTicket 建一张归属于该租户 super_admin 的工单：附件列表的服务层要求
// 调用者是工单相关方或具备管理角色，把申请人设成 token 用户本身才能走通本租户路径，
// 而跨租户路径必须在租户谓词上 fail closed。
func createRuleListTicket(ctx context.Context, t *testing.T, client *ent.Client, tenantID, userID int, code string) (int, error) {
	t.Helper()

	created, err := client.Ticket.Create().
		SetTicketNumber(fmt.Sprintf("TKT-RULE-LIST-%s", strings.ToUpper(code))).
		SetTitle("规则与附件信封回归").
		SetDescription("验证工单域三个不分页信封的集合键").
		SetType("incident").
		SetPriority("medium").
		SetStatus("open").
		SetRequesterID(userID).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return created.ID, nil
}
