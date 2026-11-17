package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/toolinvocation"
	"itsm-backend/handlers/ai"
	service "itsm-backend/internal/services"
	"itsm-backend/middleware"
)

// Phase B6（2026-11-17）：ToolInvocation 业务外键强关联。
// inputRef 是字符串弱关联（格式漂移即断链）；ticket_id/change_id 是可反查的强关联列。
// 本测试锁住两件事：
//  1. 字段可持久化往返（ent schema + 生成物 + 仓储三层接通）；
//  2. 工具参数里的 ticketId/changeId 经 ExecuteTool 审计路径真实落库，
//     且兼容 JSON 反序列化后的 float64 形态。
func TestToolInvocationBusinessRefsPersist(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:toolinv_bizref?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	tenant := client.Tenant.Create().
		SetName("BizRef").SetCode("BIZ_REF").SetDomain("bizref.test").SetStatus("active").
		SaveX(ctx)
	user := client.User.Create().
		SetUsername("bizref_user").SetEmail("bizref@example.com").SetName("BizRef User").
		SetPasswordHash("hash").SetActive(true).SetTenantID(tenant.ID).
		SaveX(ctx)

	repo := ai.NewEntRepository(client)

	// 直接往返：验证新列写入并可反查。
	created, err := repo.CreateToolInvocation(ctx, &ai.ToolInvocation{
		TenantID: tenant.ID, UserID: user.ID, ToolName: "analyze_ddl",
		Arguments: `{"ticketId":101}`, Status: "success",
		ApprovalState: "auto", TicketID: 101, ChangeID: 202,
	})
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	stored, err := repo.GetToolInvocation(ctx, created.ID, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 101, stored.TicketID, "ticket_id 必须持久化往返")
	assert.Equal(t, 202, stored.ChangeID, "change_id 必须持久化往返")

	// 经 ExecuteTool 审计路径：float64 形态的 ticketId 必须被解析写入新列。
	prevMode := middleware.PermissionConfig.Mode
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeHardcodeOnly
	middleware.InvalidateAllPermissionCaches()
	ai.ResetRBACFlagForTest()
	t.Cleanup(func() {
		middleware.PermissionConfig.Mode = prevMode
		middleware.InvalidateAllPermissionCaches()
		ai.ResetRBACFlagForTest()
	})
	t.Setenv("AI_TOOL_RBAC_ENABLED", "true")
	t.Setenv("AI_TOOL_RBAC_ENFORCE", "true")
	ai.ResetRBACFlagForTest()

	svc := ai.NewService(repo, zap.NewNop().Sugar(), nil, service.NewToolRegistry(nil, nil, nil, nil), nil, nil, nil, nil, nil, nil)
	svc.SetEntClient(client)
	_, _, err = svc.ExecuteTool(ctx, user.ID, tenant.ID, "super_admin", "analyze_ddl", map[string]interface{}{
		"sqlText":  "ALTER TABLE t ADD COLUMN c INT;",
		"ticketId": float64(303), // JSON 反序列化的真实形态
		"changeId": float64(404),
	})
	require.NoError(t, err)

	// 按 ticket_id 反查：验证强关联可检索（这是 inputRef 字符串做不到的）。
	invocations, err := client.ToolInvocation.Query().
		Where(toolinvocation.TicketID(303)).All(ctx)
	require.NoError(t, err)
	require.Len(t, invocations, 1, "必须能按 ticket_id 反查到审计记录")
	assert.Equal(t, 404, invocations[0].ChangeID)
}

// bizRefRepo 覆盖两个查询方法，让 handler 测试拿到带业务外键的记录。
type bizRefRepo struct{ rbacMockRepo }

func (r *bizRefRepo) GetToolInvocation(_ context.Context, id, _ int) (*ai.ToolInvocation, error) {
	return &ai.ToolInvocation{ID: id, TicketID: 501, ChangeID: 601, ToolName: "analyze_ddl", Status: "success"}, nil
}

func (r *bizRefRepo) ListToolInvocations(_ context.Context, _ int, _ string) ([]*ai.ToolInvocation, error) {
	return []*ai.ToolInvocation{{ID: 1, TicketID: 501, ChangeID: 601, ToolName: "analyze_ddl", Status: "success"}}, nil
}

// 锁定 GET invocations 详情与列表响应都暴露 B6 业务外键，防止文档与实现漂移。
func TestToolInvocationBizRefsInHTTPResponse(t *testing.T) {
	repo := &bizRefRepo{}
	svc := ai.NewService(repo, zap.NewNop().Sugar(), nil, service.NewToolRegistry(nil, nil, nil, nil), nil, nil, nil, nil, nil, nil)
	h := ai.NewHandler(svc)

	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name    string
		route   string
		handler gin.HandlerFunc
	}{
		{"detail", "/detail/:id", h.GetToolInvocation},
		{"list", "/list", h.ListToolInvocations},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.GET(tc.route, func(c *gin.Context) {
				c.Set("tenant_id", 10)
				c.Set("user_id", 1)
				tc.handler(c)
			})
			path := tc.route
			if tc.name == "detail" {
				path = "/detail/1"
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())

			var env struct {
				Data json.RawMessage `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))

			var body map[string]interface{}
			if tc.name == "detail" {
				require.NoError(t, json.Unmarshal(env.Data, &body))
				assert.Equal(t, float64(501), body["ticketId"])
				assert.Equal(t, float64(601), body["changeId"])
			} else {
				var list struct {
					Items []map[string]interface{} `json:"items"`
				}
				require.NoError(t, json.Unmarshal(env.Data, &list))
				require.Len(t, list.Items, 1)
				assert.Equal(t, float64(501), list.Items[0]["ticketId"])
				assert.Equal(t, float64(601), list.Items[0]["changeId"])
			}
		})
	}
}
