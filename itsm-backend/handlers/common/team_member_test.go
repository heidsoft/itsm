package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/team"
	"itsm-backend/ent/user"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/crypto/bcrypt"
)

// 团队成员链路只在 HTTP 边界才有意义：租户解析、成员投影、归属冲突判定都要经真实
// handler 验证，所以这里挂真实 EntRepository + Service + 路由（路径与
// router/common_system_routes.go 注册的完全一致）。
type teamTestEnv struct {
	router  *gin.Engine
	client  *ent.Client
	tenantA int
	tenantB int
	teamA   int
	teamB   int
	userA1  int
	userA2  int
	userB1  int
}

func newTeamTestEnv(t *testing.T) *teamTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	ctx := context.Background()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)

	seedTenant := func(name string) int {
		tenant, err := client.Tenant.Create().
			SetName(name).
			SetCode(strings.ToLower(name) + "-" + suffix).
			SetDomain(strings.ToLower(name) + suffix + ".com").
			SetStatus("active").
			Save(ctx)
		require.NoError(t, err)
		return tenant.ID
	}
	seedUser := func(tenantID int, username string, active bool) int {
		hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.MinCost)
		require.NoError(t, err)
		u, err := client.User.Create().
			SetUsername(username).
			SetEmail(username + "@" + suffix + ".com").
			SetName(username).
			SetRole(user.RoleAgent).
			SetPasswordHash(string(hash)).
			SetActive(active).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		return u.ID
	}
	seedTeam := func(tenantID int, code string) int {
		tm, err := client.Team.Create().
			SetName("团队 " + code).
			SetCode(code).
			SetDescription("负责 " + code).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		return tm.ID
	}

	env := &teamTestEnv{client: client}
	env.tenantA = seedTenant("tenantA")
	env.tenantB = seedTenant("tenantB")
	env.userA1 = seedUser(env.tenantA, "a1-"+suffix, true)
	env.userA2 = seedUser(env.tenantA, "a2-"+suffix, true)
	env.userB1 = seedUser(env.tenantB, "b1-"+suffix, true)
	env.teamA = seedTeam(env.tenantA, "OPS-A")
	env.teamB = seedTeam(env.tenantB, "OPS-B")

	repo := NewEntRepository(client)
	svc := NewService(repo, "test-jwt-secret-with-32-bytes!!", zaptest.NewLogger(t).Sugar(), client)
	h := NewHandler(svc)

	r := gin.New()
	r.Use(gin.Recovery())
	// 用请求头选择租户；不带该头即无租户上下文，用于验证 fail-closed。
	r.Use(func(c *gin.Context) {
		if raw := c.GetHeader("X-Test-Tenant"); raw != "" {
			tid, err := strconv.Atoi(raw)
			require.NoError(t, err)
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tid})
		}
		c.Next()
	})

	org := r.Group("/api/v1/org")
	org.GET("/teams", h.ListTeams)
	org.GET("/teams/:id", h.GetTeam)
	org.POST("/teams", h.CreateTeam)
	org.PUT("/teams/:id", h.UpdateTeam)
	org.DELETE("/teams/:id", h.DeleteTeam)
	org.GET("/teams/:id/members", h.ListTeamMembers)
	org.POST("/teams/:id/members", h.AddTeamMember)
	org.DELETE("/teams/:id/members", h.RemoveTeamMember)
	r.GET("/api/v1/teams", h.ListTeams)

	env.router = r
	return env
}

type teamEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (e *teamTestEnv) do(method, path string, tenantID int, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if tenantID > 0 {
		request.Header.Set("X-Test-Tenant", strconv.Itoa(tenantID))
	}
	e.router.ServeHTTP(recorder, request)
	return recorder
}

func (e *teamTestEnv) decode(t *testing.T, recorder *httptest.ResponseRecorder) teamEnvelope {
	t.Helper()
	var env teamEnvelope
	require.NoErrorf(t, json.Unmarshal(recorder.Body.Bytes(), &env), "响应不是标准信封: %s", recorder.Body.String())
	return env
}

type listData struct {
	Items      []map[string]any `json:"items"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	TotalPages int              `json:"totalPages"`
}

func (e *teamTestEnv) list(t *testing.T, tenantID int, query string) (listData, string) {
	t.Helper()
	recorder := e.do(http.MethodGet, "/api/v1/org/teams?"+query, tenantID, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	envelope := e.decode(t, recorder)
	require.Equal(t, common.SuccessCode, envelope.Code, envelope.Message)

	var data listData
	require.NoError(t, json.Unmarshal(envelope.Data, &data))
	return data, recorder.Body.String()
}

func (e *teamTestEnv) addMember(t *testing.T, teamID int, tenantID int, userID int) *httptest.ResponseRecorder {
	t.Helper()
	path := fmt.Sprintf("/api/v1/org/teams/%d/members", teamID)
	return e.do(http.MethodPost, path, tenantID, fmt.Sprintf(`{"userId":%d}`, userID))
}

func (e *teamTestEnv) memberIDsInDB(t *testing.T, teamID int) []int {
	t.Helper()
	users, err := e.client.Team.Query().Where(team.ID(teamID)).QueryUsers().IDs(context.Background())
	require.NoError(t, err)
	slices.Sort(users)
	return users
}

// TestTeamListReturnsMembersInStandardEnvelope 覆盖列表信封与成员投影：
// 团队列表必须带 camelCase 成员，且不得再透出 Ent 的 edges 形态或敏感列。
func TestTeamListReturnsMembersInStandardEnvelope(t *testing.T) {
	env := newTeamTestEnv(t)

	require.Equal(t, http.StatusOK, env.addMember(t, env.teamA, env.tenantA, env.userA1).Code)
	require.Equal(t, http.StatusOK, env.addMember(t, env.teamA, env.tenantA, env.userA2).Code)

	data, raw := env.list(t, env.tenantA, "pageSize=100")
	require.Equal(t, 1, data.Total)
	require.Len(t, data.Items, 1)

	teamItem := data.Items[0]
	members, ok := teamItem["members"].([]any)
	require.Truef(t, ok, "members 必须是数组: %v", teamItem["members"])
	require.Len(t, members, 2)

	first := members[0].(map[string]any)
	require.Equal(t, float64(env.userA1), first["id"])
	require.Contains(t, first, "username")
	require.Contains(t, first, "name")
	require.Contains(t, first, "email")

	require.Contains(t, raw, `"total"`)
	require.NotContains(t, raw, "edges")
	require.NotContains(t, raw, "password_hash")
	require.NotContains(t, raw, "passwordHash")
	require.NotContains(t, raw, "team_users")
}

// TestTeamListPagination 证明分页由服务端负责：pageSize=1 时 items 只有一条，
// 但 total 仍是全量，调用方可以据此继续翻页。
func TestTeamListPagination(t *testing.T) {
	env := newTeamTestEnv(t)
	extra, err := env.client.Team.Create().
		SetName("第二团队").SetCode("DEV-A").SetTenantID(env.tenantA).Save(context.Background())
	require.NoError(t, err)
	defer func() { require.NoError(t, env.client.Team.DeleteOneID(extra.ID).Exec(context.Background())) }()

	data, _ := env.list(t, env.tenantA, "page=1&pageSize=1")
	require.Equal(t, 2, data.Total)
	require.Len(t, data.Items, 1)
	require.Equal(t, 1, data.Page)
	require.Equal(t, 1, data.PageSize)
	require.Equal(t, 2, data.TotalPages)
}

// TestTeamListSearchIsServerSide 证明搜索在服务端完成：total 是过滤后的数量，
// 前端拿到的 items 已经是结果集，不能对半截数据二次过滤后冒充全量。
func TestTeamListSearchIsServerSide(t *testing.T) {
	env := newTeamTestEnv(t)
	ctx := context.Background()
	for _, seed := range []struct{ code, description string }{{"DEV-A", "负责研发"}, {"SRE-A", "负责稳定性"}} {
		_, err := env.client.Team.Create().
			SetName("团队 " + seed.code).
			SetCode(seed.code).
			SetDescription(seed.description).
			SetTenantID(env.tenantA).
			Save(ctx)
		require.NoError(t, err)
	}

	byCode, _ := env.list(t, env.tenantA, "search=dev&pageSize=100")
	require.Equal(t, 1, byCode.Total)
	require.Equal(t, "DEV-A", byCode.Items[0]["code"])

	byDescription, _ := env.list(t, env.tenantA, "search=%E7%A8%B3%E5%AE%9A%E6%80%A7&pageSize=100")
	require.Equal(t, 1, byDescription.Total)
	require.Equal(t, "SRE-A", byDescription.Items[0]["code"])

	all, _ := env.list(t, env.tenantA, "pageSize=100")
	require.Equal(t, 3, all.Total)

	// 搜索结果不能漏到别的租户。
	other, _ := env.list(t, env.tenantB, "search=dev&pageSize=100")
	require.Equal(t, 0, other.Total)
	require.Empty(t, other.Items)
}

// TestTeamMembersAreTenantScoped 覆盖跨租户读取：B 租户看不到 A 的团队，也不能
// 通过成员端点探测其存在。
func TestTeamMembersAreTenantScoped(t *testing.T) {
	env := newTeamTestEnv(t)
	require.Equal(t, http.StatusOK, env.addMember(t, env.teamA, env.tenantA, env.userA1).Code)

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		path := fmt.Sprintf("/api/v1/org/teams/%d/members", env.teamA)
		recorder := env.do(method, path, env.tenantB, `{"userId":`+strconv.Itoa(env.userB1)+`}`)
		require.Equalf(t, http.StatusNotFound, recorder.Code, "%s %s => %s", method, path, recorder.Body.String())
		envelope := env.decode(t, recorder)
		require.Equal(t, common.NotFoundCode, envelope.Code)
	}

	detail := env.do(http.MethodGet, fmt.Sprintf("/api/v1/org/teams/%d", env.teamA), env.tenantB, "")
	require.Equal(t, http.StatusNotFound, detail.Code, detail.Body.String())

	// A 租户的成员关系没有被 B 的尝试改动。
	require.Equal(t, []int{env.userA1}, env.memberIDsInDB(t, env.teamA))
}

// TestAddTeamMemberRules 覆盖添加成员的三条规则：同团队重复添加是幂等，
// 用户属于别的团队是冲突，跨租户与停用用户是参数拒绝。
func TestAddTeamMemberRules(t *testing.T) {
	env := newTeamTestEnv(t)

	require.Equal(t, http.StatusOK, env.addMember(t, env.teamA, env.tenantA, env.userA1).Code)
	again := env.addMember(t, env.teamA, env.tenantA, env.userA1)
	require.Equal(t, http.StatusOK, again.Code, again.Body.String())
	require.Equal(t, []int{env.userA1}, env.memberIDsInDB(t, env.teamA), "重复添加不得产生第二份成员关系")

	// userA1 已在 teamA，把它加到同租户的另一个团队必须拒绝，而不是静默搬走。
	secondTeam, err := env.client.Team.Create().
		SetName("第二个团队").SetCode("DEV-A").SetTenantID(env.tenantA).Save(context.Background())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, env.client.Team.DeleteOneID(secondTeam.ID).Exec(context.Background()))
	}()

	conflict := env.addMember(t, secondTeam.ID, env.tenantA, env.userA1)
	require.Equal(t, http.StatusConflict, conflict.Code, conflict.Body.String())
	conflictEnvelope := env.decode(t, conflict)
	require.Equal(t, common.ConflictCode, conflictEnvelope.Code)
	require.Contains(t, conflictEnvelope.Message, "团队 OPS-A")
	require.Equal(t, []int{env.userA1}, env.memberIDsInDB(t, env.teamA), "冲突拒绝不得改动原归属")
	require.Empty(t, env.memberIDsInDB(t, secondTeam.ID))

	crossTenant := env.addMember(t, env.teamA, env.tenantA, env.userB1)
	require.Equal(t, http.StatusBadRequest, crossTenant.Code, crossTenant.Body.String())
	require.Equal(t, common.BadRequestCode, env.decode(t, crossTenant).Code, crossTenant.Body.String())
	require.Equal(t, []int{env.userA1}, env.memberIDsInDB(t, env.teamA))

	inactive, err := env.client.User.Create().
		SetUsername("disabled-" + strconv.FormatInt(time.Now().UnixNano(), 10)).
		SetEmail("disabled-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "@example.com").
		SetName("停用用户").
		SetRole(user.RoleAgent).
		SetPasswordHash("x").
		SetActive(false).
		SetTenantID(env.tenantA).
		Save(context.Background())
	require.NoError(t, err)
	disabled := env.addMember(t, env.teamA, env.tenantA, inactive.ID)
	require.Equal(t, http.StatusBadRequest, disabled.Code, disabled.Body.String())
	require.Equal(t, common.BadRequestCode, env.decode(t, disabled).Code, disabled.Body.String())
	require.Equal(t, []int{env.userA1}, env.memberIDsInDB(t, env.teamA))
}

// TestAddTeamMemberResponseCarriesServerState 证明写操作返回服务端成员真相，
// 前端不需要自己拼本地状态。
func TestAddTeamMemberResponseCarriesServerState(t *testing.T) {
	env := newTeamTestEnv(t)
	recorder := env.addMember(t, env.teamA, env.tenantA, env.userA2)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var members []*TeamMember
	envelope := env.decode(t, recorder)
	require.NoError(t, json.Unmarshal(envelope.Data, &members))
	require.Len(t, members, 1)
	require.Equal(t, env.userA2, members[0].ID)

	// 列表里的成员与成员端点分页口径一致。
	paged := env.do(http.MethodGet, fmt.Sprintf("/api/v1/org/teams/%d/members?page=1&pageSize=1", env.teamA), env.tenantA, "")
	require.Equal(t, http.StatusOK, paged.Code, paged.Body.String())
	var data listData
	require.NoError(t, json.Unmarshal(env.decode(t, paged).Data, &data))
	require.Equal(t, 1, data.Total)
	require.Len(t, data.Items, 1)
}

// TestRemoveTeamMemberRules 覆盖移除：非成员是 404，真实移除后关系消失。
func TestRemoveTeamMemberRules(t *testing.T) {
	env := newTeamTestEnv(t)
	require.Equal(t, http.StatusOK, env.addMember(t, env.teamA, env.tenantA, env.userA1).Code)

	path := fmt.Sprintf("/api/v1/org/teams/%d/members", env.teamA)
	notMember := env.do(http.MethodDelete, path, env.tenantA, `{"userId":`+strconv.Itoa(env.userA2)+`}`)
	require.Equal(t, http.StatusNotFound, notMember.Code, notMember.Body.String())
	require.Equal(t, common.NotFoundCode, env.decode(t, notMember).Code, notMember.Body.String())
	require.Equal(t, []int{env.userA1}, env.memberIDsInDB(t, env.teamA))

	removed := env.do(http.MethodDelete, path, env.tenantA, `{"userId":`+strconv.Itoa(env.userA1)+`}`)
	require.Equal(t, http.StatusOK, removed.Code, removed.Body.String())
	require.Empty(t, env.memberIDsInDB(t, env.teamA))

	// 空团队的 members 仍是数组而不是缺省，前端不得为它做存在性猜测。
	data, raw := env.list(t, env.tenantA, "pageSize=100")
	require.Contains(t, raw, `"members":[]`)
	require.Len(t, data.Items[0]["members"], 0)
}

// TestTeamRoutesFailClosedWithoutTenant 证明缺租户上下文一律 401，
// 不会退化成空列表或误写别人的团队。
func TestTeamRoutesFailClosedWithoutTenant(t *testing.T) {
	env := newTeamTestEnv(t)

	cases := []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/org/teams", ""},
		{http.MethodGet, "/api/v1/teams", ""},
		{http.MethodGet, fmt.Sprintf("/api/v1/org/teams/%d/members", env.teamA), ""},
		{http.MethodPost, fmt.Sprintf("/api/v1/org/teams/%d/members", env.teamA), `{"userId":` + strconv.Itoa(env.userA1) + `}`},
		{http.MethodDelete, fmt.Sprintf("/api/v1/org/teams/%d/members", env.teamA), `{"userId":` + strconv.Itoa(env.userA1) + `}`},
		{http.MethodPost, "/api/v1/org/teams", `{"name":"无租户","code":"NO-TENANT"}`},
	}
	for _, tc := range cases {
		recorder := env.do(tc.method, tc.path, 0, tc.body)
		require.Equalf(t, http.StatusUnauthorized, recorder.Code, "%s %s => %s", tc.method, tc.path, recorder.Body.String())
		envelope := env.decode(t, recorder)
		require.Equal(t, common.AuthFailedCode, envelope.Code)
		require.NotContains(t, recorder.Body.String(), `"items"`)
	}
	require.Empty(t, env.memberIDsInDB(t, env.teamA))
	require.Equal(t, 1, countTeamsInTenant(t, env.client, env.tenantA), "无租户上下文的创建请求不得落库")
}

// TestCreateTeamRejectsForeignManager 证明负责人这种跨资源引用在写入前按租户校验。
func TestCreateTeamRejectsForeignManager(t *testing.T) {
	env := newTeamTestEnv(t)

	payload := fmt.Sprintf(`{"name":"外部负责人团队","code":"FOREIGN-MGR","managerId":%d}`, env.userB1)
	recorder := env.do(http.MethodPost, "/api/v1/org/teams", env.tenantA, payload)
	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	require.Equal(t, common.BadRequestCode, env.decode(t, recorder).Code, recorder.Body.String())
	require.Equal(t, 1, countTeamsInTenant(t, env.client, env.tenantA), "负责人校验失败不得留下团队记录")

	// 同租户负责人可创建，且响应带空 members 数组。
	payload = fmt.Sprintf(`{"name":"正常团队","code":"OK-MGR","managerId":%d}`, env.userA1)
	created := env.do(http.MethodPost, "/api/v1/org/teams", env.tenantA, payload)
	require.Equal(t, http.StatusOK, created.Code, created.Body.String())
	require.Contains(t, created.Body.String(), `"members":[]`)
	require.Equal(t, 2, countTeamsInTenant(t, env.client, env.tenantA))
}

// TestDeleteTeamUnknownIsNotFound 关掉「删除不存在的团队也返回成功」的静默路径。
func TestDeleteTeamUnknownIsNotFound(t *testing.T) {
	env := newTeamTestEnv(t)

	recorder := env.do(http.MethodDelete, "/api/v1/org/teams/999999", env.tenantA, "")
	require.Equal(t, http.StatusNotFound, recorder.Code, recorder.Body.String())
	envelope := env.decode(t, recorder)
	require.Equal(t, common.NotFoundCode, envelope.Code)
}

// TestUpdateTeamDistinguishesMissingFromCleared 证明部分更新能区分「未传」「传零值」
// 与「清空」：不传的字段保持原值，显式传空串/0 才覆盖。
func TestUpdateTeamDistinguishesMissingFromCleared(t *testing.T) {
	env := newTeamTestEnv(t)
	ctx := context.Background()

	_, err := env.client.Team.UpdateOneID(env.teamA).
		SetDescription("原始描述").
		SetManagerID(env.userA1).
		Save(ctx)
	require.NoError(t, err)

	load := func() (string, int) {
		t.Helper()
		recorder := env.do(http.MethodGet, fmt.Sprintf("/api/v1/org/teams/%d", env.teamA), env.tenantA, "")
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		var item map[string]any
		require.NoError(t, json.Unmarshal(env.decode(t, recorder).Data, &item))
		description, _ := item["description"].(string)
		managerID, _ := item["managerId"].(float64)
		return description, int(managerID)
	}

	// 只改名称：描述与负责人必须原样保留。
	renamed := env.do(http.MethodPut, fmt.Sprintf("/api/v1/org/teams/%d", env.teamA), env.tenantA, `{"name":"改名后的团队"}`)
	require.Equal(t, http.StatusOK, renamed.Code, renamed.Body.String())
	require.Contains(t, renamed.Body.String(), `"name":"改名后的团队"`)
	description, managerID := load()
	require.Equal(t, "原始描述", description)
	require.Equal(t, env.userA1, managerID)

	// 显式空串清空描述，负责人不受影响。
	cleared := env.do(http.MethodPut, fmt.Sprintf("/api/v1/org/teams/%d", env.teamA), env.tenantA, `{"description":""}`)
	require.Equal(t, http.StatusOK, cleared.Code, cleared.Body.String())
	description, managerID = load()
	require.Empty(t, description)
	require.Equal(t, env.userA1, managerID)

	// 显式 0 解除负责人，描述不受影响。
	unassigned := env.do(http.MethodPut, fmt.Sprintf("/api/v1/org/teams/%d", env.teamA), env.tenantA, `{"managerId":0}`)
	require.Equal(t, http.StatusOK, unassigned.Code, unassigned.Body.String())
	description, managerID = load()
	require.Empty(t, description)
	require.Equal(t, 0, managerID)

	// 外部租户负责人按参数拒绝，且不覆盖已有负责人。
	_, err = env.client.Team.UpdateOneID(env.teamA).SetManagerID(env.userA1).Save(ctx)
	require.NoError(t, err)
	foreign := env.do(http.MethodPut, fmt.Sprintf("/api/v1/org/teams/%d", env.teamA), env.tenantA,
		fmt.Sprintf(`{"managerId":%d}`, env.userB1))
	require.Equal(t, http.StatusBadRequest, foreign.Code, foreign.Body.String())
	require.Equal(t, common.BadRequestCode, env.decode(t, foreign).Code, foreign.Body.String())
	_, managerID = load()
	require.Equal(t, env.userA1, managerID)
}

// TestTeamProjectionCarriesManagerName 负责人姓名必须由服务端投影。
// 只回带 managerId 时前端要拿当前页用户目录比对，目录被分页截断后负责人就显示成空。
func TestTeamProjectionCarriesManagerName(t *testing.T) {
	env := newTeamTestEnv(t)
	ctx := context.Background()

	first, err := env.client.User.Get(ctx, env.userA1)
	require.NoError(t, err)
	second, err := env.client.User.Get(ctx, env.userA2)
	require.NoError(t, err)
	foreign, err := env.client.User.Get(ctx, env.userB1)
	require.NoError(t, err)

	decodeTeam := func(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		envelope := env.decode(t, recorder)
		require.Equal(t, common.SuccessCode, envelope.Code, envelope.Message)
		var data map[string]any
		require.NoError(t, json.Unmarshal(envelope.Data, &data))
		return data
	}

	created := env.do(http.MethodPost, "/api/v1/org/teams", env.tenantA,
		fmt.Sprintf(`{"name":"平台组","code":"PLAT","managerId":%d}`, env.userA1))
	require.Equal(t, http.StatusOK, created.Code, created.Body.String())
	createdTeam := decodeTeam(t, created)
	require.Equal(t, first.Name, createdTeam["managerName"], created.Body.String())
	teamID := int(createdTeam["id"].(float64))

	list, body := env.list(t, env.tenantA, "pageSize=100")
	var listed map[string]any
	for _, item := range list.Items {
		if int(item["id"].(float64)) == teamID {
			listed = item
		}
	}
	require.NotNil(t, listed, "新建团队未出现在列表里: %s", body)
	require.Equal(t, first.Name, listed["managerName"])

	detail := decodeTeam(t, env.do(http.MethodGet, fmt.Sprintf("/api/v1/org/teams/%d", teamID), env.tenantA, ""))
	require.Equal(t, first.Name, detail["managerName"])

	// 换负责人：写响应直接回带新姓名，页面不需要等下一次列表加载。
	switched := env.do(http.MethodPut, fmt.Sprintf("/api/v1/org/teams/%d", teamID), env.tenantA,
		fmt.Sprintf(`{"managerId":%d}`, env.userA2))
	require.Equal(t, http.StatusOK, switched.Code, switched.Body.String())
	require.Equal(t, second.Name, decodeTeam(t, switched)["managerName"])

	// 显式解除负责人后姓名一并清空，不能残留上一任。
	unassigned := env.do(http.MethodPut, fmt.Sprintf("/api/v1/org/teams/%d", teamID), env.tenantA, `{"managerId":0}`)
	require.Equal(t, http.StatusOK, unassigned.Code, unassigned.Body.String())
	require.Empty(t, decodeTeam(t, unassigned)["managerName"])

	// 库里残留指向其他租户用户的 manager_id 时，读取路径不回带对方姓名。
	_, err = env.client.Team.UpdateOneID(teamID).SetManagerID(env.userB1).Save(ctx)
	require.NoError(t, err)
	stale := decodeTeam(t, env.do(http.MethodGet, fmt.Sprintf("/api/v1/org/teams/%d", teamID), env.tenantA, ""))
	require.Empty(t, stale["managerName"], "跨租户用户姓名泄漏到响应里")
	require.NotEqual(t, foreign.Name, stale["managerName"])
}

func countTeamsInTenant(t *testing.T, client *ent.Client, tenantID int) int {
	t.Helper()
	count, err := client.Team.Query().
		Where(team.TenantID(tenantID), team.DeletedAtIsNil()).
		Count(context.Background())
	require.NoError(t, err)
	return count
}
