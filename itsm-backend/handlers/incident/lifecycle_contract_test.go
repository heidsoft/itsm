package incident

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"
)

func TestLifecycleHTTPErrorContract(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:incident_lifecycle_contract?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("Lifecycle").SetCode("lifecycle").SetDomain("lifecycle.test").SaveX(ctx)
	user := client.User.Create().SetUsername("lifecycle").SetName("test").SetEmail("lifecycle@example.com").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	inc := client.Incident.Create().SetTitle("closed incident").SetIncidentNumber("INC-review").SetStatus("closed").SetReporterID(user.ID).SetTenantID(tenant.ID).SaveX(ctx)
	production := service.NewIncidentService(client, zap.NewNop().Sugar(), nil)
	// P1-DataScope：生命周期方法现在先经 repo.Get 行级守卫（CanWriteResource），
	// repo 不能为 nil。接真实 Ent repo：reporter 本人操作守卫放行，跨租户由
	// repo.Get 返回 ent NotFound（404），既有契约断言不变。
	repo := NewEntRepository(client)
	h := NewHandler(NewService(repo, production, nil, nil, nil, zap.NewNop().Sugar()))
	for _, tc := range []struct {
		name, path           string
		tenant, status, code int
	}{
		{"transition", "acknowledge", tenant.ID, 409, 4090},
		{"foreign", "acknowledge", tenant.ID + 1, 404, 4004},
		{"missing identity", "acknowledge", 0, 401, 2001},
		{"sla unavailable", "sla/pause", tenant.ID, 503, 5003},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) { c.Set("tenant_id", tc.tenant); c.Set("user_id", user.ID) })
			r.POST("/api/v1/incidents/:id/acknowledge", h.Acknowledge)
			r.PUT("/api/v1/incidents/:id/sla/pause", h.PauseSLA)
			method := http.MethodPost
			if tc.path == "sla/pause" {
				method = http.MethodPut
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(method, "/api/v1/incidents/"+strconv.Itoa(inc.ID)+"/"+tc.path, nil))
			require.Equal(t, tc.status, w.Code, w.Body.String())
			var response struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.Equal(t, tc.code, response.Code)
			require.NotEmpty(t, response.Message)
		})
	}
	require.Equal(t, "closed", client.Incident.GetX(ctx, inc.ID).Status)
}

// TestLifecycleGuard_SQLite_AllowedByRelation 用真实 Ent + 真实 production
// service 覆盖行级守卫的放行分支：reporter 本人、受理人本人、管理角色均可
// 对事件单执行生命周期操作（守卫放行后由状态机决定业务结果）。
func TestLifecycleGuard_SQLite_AllowedByRelation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:incident_guard_allow?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("GuardAllow").SetCode("guard-allow").SetDomain("guard.test").SaveX(ctx)

	mkUser := func(name string) *ent.User {
		return client.User.Create().SetUsername(name).SetName(name).SetEmail(name + "@guard.test").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	}
	reporter := mkUser("reporter")
	assignee := mkUser("assignee")
	stranger := mkUser("stranger")

	production := service.NewIncidentService(client, zap.NewNop().Sugar(), nil)
	repo := NewEntRepository(client)
	h := NewHandler(NewService(repo, production, nil, nil, nil, zap.NewNop().Sugar()))

	// 每个事件单单号唯一（incidents.incident_number 全局唯一约束），
	// actorID 取真实租户用户（createIncidentEventTx 校验 actor 必须是
	// 同租户活跃用户）。admin 放行场景用 stranger 用户身份（真实存在、
	// 仅凭 admin 角色越权放行）。
	var seq int
	mkIncident := func(status string, assigneeID int) int {
		seq++
		b := client.Incident.Create().
			SetTitle("guard allow " + status).
			SetIncidentNumber(fmt.Sprintf("INC-GA-%s-%d", status, seq)).
			SetStatus(status).
			SetReporterID(reporter.ID).
			SetTenantID(tenant.ID)
		if assigneeID > 0 {
			b = b.SetAssigneeID(assigneeID)
		}
		return b.SaveX(ctx).ID
	}

	serve := func(role string, userID int) *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("tenant_id", tenant.ID)
			c.Set("user_id", userID)
			c.Set("role", role)
			c.Next()
		})
		r.POST("/api/v1/incidents/:id/acknowledge", h.Acknowledge)
		r.POST("/api/v1/incidents/:id/resolve", h.Resolve)
		r.POST("/api/v1/incidents/:id/close", h.Close)
		r.POST("/api/v1/incidents/:id/reopen", h.Reopen)
		return r
	}

	t.Run("reporter acknowledges own incident", func(t *testing.T) {
		id := mkIncident("new", 0)
		r := serve("agent", reporter.ID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/incidents/%d/acknowledge", id), nil))
		require.Equal(t, 200, w.Code, w.Body.String())
		require.Equal(t, "acknowledged", client.Incident.GetX(ctx, id).Status)
	})

	t.Run("assignee resolves assigned incident", func(t *testing.T) {
		id := mkIncident("in_progress", assignee.ID)
		r := serve("agent", assignee.ID)
		body := `{"resolution":"fixed","rootCause":"bug"}`
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/incidents/%d/resolve", id), strings.NewReader(body)))
		require.Equal(t, 200, w.Code, w.Body.String())
		require.Equal(t, "resolved", client.Incident.GetX(ctx, id).Status)
	})

	t.Run("admin closes incident as stranger", func(t *testing.T) {
		id := mkIncident("resolved", 0)
		// stranger 是租户内真实用户（createIncidentEventTx 校验 actor），
		// 与单据无归属关系——仅凭 admin 角色放行，证明管理角色全租户可写。
		r := serve("admin", stranger.ID)
		body := `{"closeNotes":"closed by admin"}`
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/incidents/%d/close", id), strings.NewReader(body)))
		require.Equal(t, 200, w.Code, w.Body.String())
		require.Equal(t, "closed", client.Incident.GetX(ctx, id).Status)
	})

	t.Run("reporter reopens resolved incident", func(t *testing.T) {
		id := mkIncident("resolved", 0)
		r := serve("agent", reporter.ID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/incidents/%d/reopen", id), nil))
		require.Equal(t, 200, w.Code, w.Body.String())
		require.Equal(t, "in_progress", client.Incident.GetX(ctx, id).Status)
	})
}

// TestAssignGuard_SQLite_NotRowLevelEnforced 固化 Assign 的设计边界：
// 指派是"产生受理关系"的授权动作（受理人正是通过 assign 产生，对其做
// owner/assignee 校验会循环依赖），不做行级校验——路由层仅有
// incident:assign RBAC 门禁。无关人（stranger）指派活跃同租户用户应成功。
func TestAssignGuard_SQLite_NotRowLevelEnforced(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:incident_guard_assign?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("GuardAssign").SetCode("guard-assign").SetDomain("assign.test").SaveX(ctx)

	reporter := client.User.Create().SetUsername("rep").SetName("rep").SetEmail("rep@assign.test").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("str").SetName("str").SetEmail("str@assign.test").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	target := client.User.Create().SetUsername("tgt").SetName("tgt").SetEmail("tgt@assign.test").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)

	production := service.NewIncidentService(client, zap.NewNop().Sugar(), nil)
	repo := NewEntRepository(client)
	h := NewHandler(NewService(repo, production, nil, nil, nil, zap.NewNop().Sugar()))

	inc := client.Incident.Create().
		SetTitle("assign boundary").SetIncidentNumber("INC-GA-ASSIGN").
		SetStatus("new").SetReporterID(reporter.ID).SetTenantID(tenant.ID).SaveX(ctx)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenant.ID)
		c.Set("user_id", stranger.ID) // 无关人执行指派
		c.Set("role", "agent")
		c.Next()
	})
	r.POST("/api/v1/incidents/:id/assign", h.Assign)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/incidents/%d/assign", inc.ID),
		strings.NewReader(fmt.Sprintf(`{"assigneeId":%d}`, target.ID))))
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, target.ID, client.Incident.GetX(ctx, inc.ID).AssigneeID)
}
