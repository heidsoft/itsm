package group

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/schema"
	"itsm-backend/middleware"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 本文件验证「组引用关系只读查询视图」GET /api/v1/groups/:id/references。
//
// 它补的是前两轮遗留的可观测性缺口：变更前守卫（改名/删除 409）与发布期守卫
// （空审批组 409）能在写路径拦住问题，但管理员在写之前没有任何入口能看到
// 「这个组到底被谁引用」。视图必须与被守卫复用的同一个扫描器同源，否则会
// 出现「视图说没引用、一点保存就被 409 拒绝」的假诊断。

func setupRefViewDB(t *testing.T) *ent.Client {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(context.Background()))
	return client
}

func seedRefViewGroup(t *testing.T, client *ent.Client, name string, tenantID int) int {
	t.Helper()
	g, err := client.Group.Create().SetName(name).SetTenantID(tenantID).Save(context.Background())
	require.NoError(t, err)
	return g.ID
}

func seedRefViewProcess(t *testing.T, client *ent.Client, key string, bpmnXML []byte, tenantID int, active bool) int {
	t.Helper()
	ctx := context.Background()
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("dep-" + key).
		SetDeploymentName("部署 " + key).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey(key).
		SetName(key).
		SetVersion("1.0.0").
		SetBpmnXML(bpmnXML).
		SetIsActive(active).
		SetIsLatest(true).
		SetDeploymentID(dep.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return def.ID
}

func serveRefView(t *testing.T, client *ent.Client, tenantID int, target string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Next()
	})
	h := NewHandler(NewGroupService(client), zap.NewNop().Sugar())
	router.GET("/groups/:id/references", h.GetGroupReferences)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	return w
}

type refViewEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []struct {
			Kind   string `json:"kind"`
			RefID  int    `json:"refId"`
			RefKey string `json:"refKey"`
			NodeID string `json:"nodeId"`
		} `json:"items"`
		Total int64 `json:"total"`
	} `json:"data"`
}

// TestRefView_两类引用来源都返回 覆盖两条真实消费 groupName 的路径：
// BPMN userTask.candidateGroups 与审批链 chain[].role = "group:<name>"。
func TestRefView_两类引用来源都返回(t *testing.T) {
	client := setupRefViewDB(t)
	const name = "dual-ref-group"
	gid := seedRefViewGroup(t, client, name, 1)
	seedRefViewProcess(t, client, "dual_flow", refBPMNXML("approve_node", name), 1, true)
	_, err := client.ApprovalChain.Create().
		SetName("dual-chain").
		SetEntityType("change").
		SetTenantID(1).
		SetChain([]schema.ApprovalChainStep{
			{Level: 2, Role: "group:" + name, Name: "二级审批", IsRequired: true},
		}).
		Save(context.Background())
	require.NoError(t, err)

	refs, err := NewGroupService(client).GetGroupReferences(context.Background(), gid, 1)
	require.NoError(t, err)
	require.Len(t, refs, 2, "两条真实路径都必须出现在视图里")

	byKind := map[string]GroupReference{}
	for _, r := range refs {
		byKind[r.Kind] = r
	}
	assert.Equal(t, "dual_flow", byKind["process_definition"].RefKey)
	assert.Equal(t, "approve_node", byKind["process_definition"].NodeID)
	assert.Equal(t, "dual-chain", byKind["approval_chain"].RefKey)
	assert.Equal(t, "level-2", byKind["approval_chain"].NodeID)
}

// TestRefView_HTTP信封字段完整 走真实 gin 路由，验证契约字段名与业务码。
func TestRefView_HTTP信封字段完整(t *testing.T) {
	client := setupRefViewDB(t)
	const name = "http-group"
	gid := seedRefViewGroup(t, client, name, 1)
	defID := seedRefViewProcess(t, client, "http_flow", refBPMNXML("http_task", name), 1, true)

	w := serveRefView(t, client, 1, fmt.Sprintf("/groups/%d/references", gid))
	require.Equal(t, http.StatusOK, w.Code)

	var env refViewEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, common.SuccessCode, env.Code)
	require.Len(t, env.Data.Items, 1)
	assert.Equal(t, "process_definition", env.Data.Items[0].Kind)
	assert.Equal(t, "http_flow", env.Data.Items[0].RefKey)
	assert.Equal(t, "http_task", env.Data.Items[0].NodeID)
	assert.Equal(t, defID, env.Data.Items[0].RefID)
	assert.Equal(t, int64(1), env.Data.Total)
}

// TestRefView_无引用返回空数组而非null 是前端可用性的关键契约：
// null 会让「无引用」和「接口失败」在 UI 上无法区分。
func TestRefView_无引用返回空数组而非null(t *testing.T) {
	client := setupRefViewDB(t)
	gid := seedRefViewGroup(t, client, "free-group", 1)

	w := serveRefView(t, client, 1, fmt.Sprintf("/groups/%d/references", gid))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"items":[]`, "空引用必须是 [] 而不是 null")

	var env refViewEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.NotNil(t, env.Data.Items)
	assert.Empty(t, env.Data.Items)
	assert.Equal(t, int64(0), env.Data.Total)
}

// TestRefView_跨租户组ID返回404 是 IDOR 防线：
// 猜测他租户的组 ID 不得读到该组的引用面。
func TestRefView_跨租户组ID返回404(t *testing.T) {
	client := setupRefViewDB(t)
	gidB := seedRefViewGroup(t, client, "tenantB-group", 2)

	w := serveRefView(t, client, 1, fmt.Sprintf("/groups/%d/references", gidB))
	assert.Equal(t, http.StatusNotFound, w.Code, "他租户组必须表现为不存在")

	var env struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, common.NotFoundCode, env.Code)
}

// TestRefView_跨租户引用隔离 验证扫描器自身的 tenantID 谓词：
// 租户 B 的流程引用了同名组，不得出现在租户 A 同名组的视图里。
func TestRefView_跨租户引用隔离(t *testing.T) {
	client := setupRefViewDB(t)
	const name = "cross-tenant-name"
	gidA := seedRefViewGroup(t, client, name, 1)
	seedRefViewProcess(t, client, "b_flow", refBPMNXML("b_task", name), 2, true)

	svc := NewGroupService(client)
	refsA, err := svc.GetGroupReferences(context.Background(), gidA, 1)
	require.NoError(t, err)
	assert.Empty(t, refsA, "他租户引用不得串进本租户视图")

	// 反向确认：同一名字在租户 B 自己的视图里确实可见，避免「两边都空」的假绿。
	gidB := seedRefViewGroup(t, client, name, 2)
	refsB, err := svc.GetGroupReferences(context.Background(), gidB, 2)
	require.NoError(t, err)
	require.Len(t, refsB, 1, "本租户引用必须可见")
}

// TestRefView_CSV整token匹配 保证视图与变更守卫用同一套 token 口径：
// candidateGroups="finance, approvers-l3, security" 不得被前缀 "approvers-l" 命中。
func TestRefView_CSV整token匹配(t *testing.T) {
	client := setupRefViewDB(t)
	const name = "approvers-l3"
	gid := seedRefViewGroup(t, client, name, 1)
	seedRefViewProcess(t, client, "csv_flow", refBPMNXML("csv_task", "finance, "+name+",security"), 1, true)

	svc := NewGroupService(client)
	refs, err := svc.GetGroupReferences(context.Background(), gid, 1)
	require.NoError(t, err)
	require.Len(t, refs, 1)

	prefixGID := seedRefViewGroup(t, client, "approvers-l", 1)
	prefixRefs, err := svc.GetGroupReferences(context.Background(), prefixGID, 1)
	require.NoError(t, err)
	assert.Empty(t, prefixRefs, "子串不得命中，必须整 token 相等")
}

// TestRefView_非活跃流程不计入 与守卫同口径：只扫 IsActive=true 的流程定义，
// 否则视图会显示已被下线的流程，诱导管理员做无谓的流程改动。
func TestRefView_非活跃流程不计入(t *testing.T) {
	client := setupRefViewDB(t)
	const name = "inactive-flow-group"
	gid := seedRefViewGroup(t, client, name, 1)
	seedRefViewProcess(t, client, "retired_flow", refBPMNXML("retired_task", name), 1, false)

	refs, err := NewGroupService(client).GetGroupReferences(context.Background(), gid, 1)
	require.NoError(t, err)
	assert.Empty(t, refs, "非生效流程不应计入引用面")
}

// TestRefView_坏XML失败关闭 是只读接口的 fail-closed 防线：
// XML 解析失败时宁可 500，也不能返回空列表伪装成「可以安全改名」。
func TestRefView_坏XML失败关闭(t *testing.T) {
	client := setupRefViewDB(t)
	gid := seedRefViewGroup(t, client, "clean-group", 1)
	seedRefViewProcess(t, client, "bad_xml", []byte("<bpmn:definitions><broken>"), 1, true)

	w := serveRefView(t, client, 1, fmt.Sprintf("/groups/%d/references", gid))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var env struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, common.InternalErrorCode, env.Code)
	assert.NotContains(t, w.Body.String(), `"items"`, "失败时不得返回空引用数据")
}

// TestRefView_无租户上下文返回401 确保视图不会绕过认证上下文兜底。
func TestRefView_无租户上下文返回401(t *testing.T) {
	client := setupRefViewDB(t)
	gid := seedRefViewGroup(t, client, "g", 1)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewHandler(NewGroupService(client), zap.NewNop().Sugar())
	router.GET("/groups/:id/references", h.GetGroupReferences)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/references", gid), nil))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestRefView_无效组ID返回400 防止非数字 id 变成「查不到」的 404 误导。
func TestRefView_无效组ID返回400(t *testing.T) {
	client := setupRefViewDB(t)
	w := serveRefView(t, client, 1, "/groups/not-a-number/references")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 保证 DTO 与 handler 的契约编译期同源（避免测试结构悄悄漂移）。
var _ = dto.GroupReferencesResponse{}
