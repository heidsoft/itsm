package workbench

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/internal/datascope"
	"itsm-backend/middleware"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkbenchDomainPermissionFilter 验证跨域工作台的域级权限过滤：
// 工作台聚合工单/事件/问题/变更四个域，路由层只挂 dashboard:read，
// 仪表盘读权限不等于各业务域读权限。此前 service.go 里那句
// "TODO: Add permission checks to filter domains" 就是这条缺口——
// 没有 incident:read 的用户仍能在工作台看到事件条目。
//
// 反向证明：把 filterDomainsByPermission 退化成恒等放行，本用例必须转红。
func TestWorkbenchDomainPermissionFilter(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(ctx))

	tenantID := 1
	// User/Role/Permission 都带 tenant_id 外键，必须先有租户行。
	_, err = client.Tenant.Create().
		SetName("WB Domain").
		SetCode("wb-domain").
		Save(ctx)
	require.NoError(t, err)
	seedDomainRecords(t, ctx, client, tenantID)

	// 建角色：ticket_reader 只持 ticket:read；event_reader 持 incident:read。
	ticketReader := seedRoleWithPermissions(t, ctx, client, tenantID, "wb-ticket-reader",
		[]string{"ticket:read"})
	eventReader := seedRoleWithPermissions(t, ctx, client, tenantID, "wb-event-reader",
		[]string{"incident:read"})

	svc := NewServiceWithClient(NewRepository(db), client)

	t.Run("只持 ticket:read 只能看到工单域", func(t *testing.T) {
		resp, err := svc.Query(ctx, WorkbenchQuery{
			TenantID: tenantID, Page: 1, PageSize: 50,
			DataScope: datascope.DataScopeAll,
			Roles:     []string{ticketReader},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Items, "有 ticket:read 至少应看到工单")
		for _, item := range resp.Items {
			assert.Equal(t, "ticket", item.RecordType,
				"只持 ticket:read 的角色不应看到非工单域条目（recordType=%s）", item.RecordType)
		}
	})

	t.Run("只持 incident:read 只能看到事件域", func(t *testing.T) {
		resp, err := svc.Query(ctx, WorkbenchQuery{
			TenantID: tenantID, Page: 1, PageSize: 50,
			DataScope: datascope.DataScopeAll,
			Roles:     []string{eventReader},
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Items, "有 incident:read 至少应看到事件")
		for _, item := range resp.Items {
			assert.Equal(t, "incident", item.RecordType,
				"只持 incident:read 的角色不应看到非事件域条目（recordType=%s）", item.RecordType)
		}
	})

	t.Run("显式指定 recordType 也不能绕过域权限", func(t *testing.T) {
		// 后门形态：?recordType=incident 若不经裁剪，只持 ticket:read 的角色
		// 就能点名看事件域。
		resp, err := svc.Query(ctx, WorkbenchQuery{
			TenantID: tenantID, Page: 1, PageSize: 50,
			DataScope:  datascope.DataScopeAll,
			Roles:      []string{ticketReader},
			RecordType: []string{"incident"},
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Items, "显式指定无权域的 recordType 必须被裁剪为空")
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("无任何域读权限返回空页且不落库查询", func(t *testing.T) {
		noPerm := seedRoleWithPermissions(t, ctx, client, tenantID, "wb-noperm", nil)
		resp, err := svc.Query(ctx, WorkbenchQuery{
			TenantID: tenantID, Page: 1, PageSize: 50,
			DataScope: datascope.DataScopeAll,
			Roles:     []string{noPerm},
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Items)
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("多角色并集：ticket+incident 可见两域", func(t *testing.T) {
		resp, err := svc.Query(ctx, WorkbenchQuery{
			TenantID: tenantID, Page: 1, PageSize: 50,
			DataScope: datascope.DataScopeAll,
			Roles:     []string{ticketReader, eventReader},
		})
		require.NoError(t, err)
		seen := map[string]bool{}
		for _, item := range resp.Items {
			seen[item.RecordType] = true
		}
		assert.True(t, seen["ticket"], "应能看到工单域")
		assert.True(t, seen["incident"], "应能看到事件域")
		assert.False(t, seen["problem"], "两角色都不持 problem:read，不应看到问题域")
		assert.False(t, seen["change"], "两角色都不持 change:read，不应看到变更域")
	})

	t.Run("super_admin 全域放行", func(t *testing.T) {
		resp, err := svc.Query(ctx, WorkbenchQuery{
			TenantID: tenantID, Page: 1, PageSize: 50,
			DataScope: datascope.DataScopeAll,
			Roles:     []string{"super_admin"},
		})
		require.NoError(t, err)
		seen := map[string]bool{}
		for _, item := range resp.Items {
			seen[item.RecordType] = true
		}
		assert.True(t, seen["ticket"])
		assert.True(t, seen["incident"])
		assert.True(t, seen["problem"])
		assert.True(t, seen["change"])
	})

	t.Run("未知 recordType 不放行", func(t *testing.T) {
		resp, err := svc.Query(ctx, WorkbenchQuery{
			TenantID: tenantID, Page: 1, PageSize: 50,
			DataScope:  datascope.DataScopeAll,
			Roles:      []string{ticketReader},
			RecordType: []string{"not_a_domain"},
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Items, "未知域必须被丢弃")
	})
}

// TestWorkbenchDomainPermissionOverHTTP 走真实 HTTP 入口，锁生产装配路径：
// handler 把 gin 上下文的角色（含附加角色）传给 service，service 按域权限码
// 裁剪。此前 service 层没有任何权限判定（只有一句 TODO），因此这条链路
// 完全没有被任何测试覆盖过。
func TestWorkbenchDomainPermissionOverHTTP(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(ctx))

	_, err = client.Tenant.Create().SetName("WB HTTP").SetCode("wb-http").Save(ctx)
	require.NoError(t, err)
	const tenantID = 1
	seedDomainRecords(t, ctx, client, tenantID)
	ticketReader := seedRoleWithPermissions(t, ctx, client, tenantID, "wb-http-ticket-reader",
		[]string{"ticket:read"})

	// 生产装配形态：NewServiceWithClient
	h := NewHandler(NewServiceWithClient(NewRepository(db), client))

	// gin 上下文里只放主角色（无附加角色）
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("role", ticketReader)
		c.Set("user_id", 1)
		c.Set("tenant_id", tenantID)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Next()
	})
	router.GET("/api/v1/workbench", h.Query)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/workbench", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var env workbenchEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), rec.Body.String())
	require.Equal(t, 0, env.Code, rec.Body.String())

	var resp struct {
		Items []WorkbenchItem `json:"items"`
		Total int             `json:"total"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &resp), rec.Body.String())
	require.NotEmpty(t, resp.Items, "持 ticket:read 应能看到工单")
	for _, item := range resp.Items {
		assert.Equal(t, "ticket", item.RecordType,
			"HTTP 路径同样必须裁剪域（recordType=%s）", item.RecordType)
	}
}

// seedDomainRecords 为四个域各建一条记录，供域过滤断言使用。
func seedDomainRecords(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
	t.Helper()
	actor, err := client.User.Create().
		SetUsername("wb-domain-actor").
		SetEmail("wb-domain-actor@example.com").
		SetName("wb-domain-actor").
		SetPasswordHash("not-a-real-hash").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.Ticket.Create().
		SetTitle("工单").SetTicketNumber("TK-DOM-1").
		SetRequesterID(actor.ID).SetAssigneeID(actor.ID).SetTenantID(tenantID).
		SetStatus("open").SetPriority("high").Save(ctx)
	require.NoError(t, err)

	_, err = client.Incident.Create().
		SetTitle("事件").SetIncidentNumber("IN-DOM-1").
		SetReporterID(actor.ID).SetAssigneeID(actor.ID).SetTenantID(tenantID).
		SetStatus("new").SetPriority("critical").Save(ctx)
	require.NoError(t, err)

	_, err = client.Problem.Create().
		SetTitle("问题").SetCreatedBy(actor.ID).SetAssigneeID(actor.ID).SetTenantID(tenantID).
		SetStatus("in_progress").SetPriority("high").Save(ctx)
	require.NoError(t, err)

	_, err = client.Change.Create().
		SetTitle("变更").SetCreatedBy(actor.ID).SetAssigneeID(actor.ID).SetTenantID(tenantID).
		SetStatus("draft").SetPriority("medium").Save(ctx)
	require.NoError(t, err)
}

// seedRoleWithPermissions 建角色并授予指定权限码（"resource:action"）。
// 返回角色 code（与 domainrole 的内置 code 不同，用独立 code 避免与播种集混淆）。
func seedRoleWithPermissions(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, roleCode string, permCodes []string) string {
	t.Helper()
	role, err := client.Role.Create().
		SetName(roleCode).
		SetCode(roleCode).
		SetIsSystem(false).
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	for _, code := range permCodes {
		resource, action := splitPermCode(t, code)
		perm, err := client.Permission.Create().
			SetCode(code).
			SetName(code).
			SetResource(resource).
			SetAction(action).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		_, err = client.RolePermission.Create().
			SetRoleID(role.ID).
			SetPermissionID(perm.ID).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
	}
	return roleCode
}

func splitPermCode(t *testing.T, code string) (string, string) {
	t.Helper()
	parts := strings.SplitN(code, ":", 2)
	require.Len(t, parts, 2, "权限码格式应为 resource:action，实际 %q", code)
	return parts[0], parts[1]
}
