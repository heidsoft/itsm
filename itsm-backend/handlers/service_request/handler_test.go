package service_request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/cmdb"
	"itsm-backend/handlers/service_catalog"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// srSeq 生成唯一后缀，避免跨用例命名冲突
var srSeq int64

func srUID() string {
	srSeq++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), srSeq)
}

// srAuth 注入服务请求 handler 依赖的 c.Get("tenant_id"/"user_id"/"role"/"department")
func srAuth(tid, uid int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("tenant_id", tid)
		c.Set("user_id", uid)
		c.Set("role", "manager")
		c.Set("department", "IT")
		c.Next()
	}
}

func srDoReq(t *testing.T, r *gin.Engine, method, path string, body interface{}) *common.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return &resp
}

func srStr(resp *common.Response) string {
	b, _ := json.Marshal(resp)
	return string(b)
}

// srSetup 组装服务请求 handler，并播种一个租户 + 一个服务目录（CITypeID=0，避免关联 CI 分支）
func srSetup(t *testing.T) (*gin.Engine, *ent.Client, int, int, int) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("SRTenant").
		SetCode("SR" + srUID()).
		SetDomain("sr.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 播种一个服务目录（无 CI 类型，走简单路径）
	scRepo := service_catalog.NewEntRepository(client)
	scSvc := service_catalog.NewService(scRepo, logger)
	cat, err := scSvc.Create(ctx, "SRCatalog-"+srUID(), "software", "for test", 0, tenant.ID, "enabled", 0, 0)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	cmdbRepo := cmdb.NewEntRepository(client)
	svc := NewService(repo, scRepo, cmdbRepo, logger)
	h := NewHandler(svc)

	uid := 1001
	r := gin.New()
	r.Use(srAuth(tenant.ID, uid))
	r.POST("/api/v1/service-requests", h.Create)
	r.GET("/api/v1/service-requests", h.List)
	r.GET("/api/v1/service-requests/:id", h.Get)
	r.PUT("/api/v1/service-requests/:id", h.Update)
	r.PUT("/api/v1/service-requests/:id/status", h.UpdateStatus)
	r.DELETE("/api/v1/service-requests/:id", h.Delete)
	r.POST("/api/v1/service-requests/:id/approval", h.ApplyApproval)
	return r, client, tenant.ID, uid, cat.ID
}

func srCreateOne(t *testing.T, r *gin.Engine, catalogID int) int {
	t.Helper()
	req := dto.CreateServiceRequestRequest{
		CatalogID:     catalogID,
		Title:         "Req-" + srUID(),
		Reason:        "need resource",
		ComplianceAck: true,
	}
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests", req)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	data := resp.Data.(map[string]interface{})
	return int(data["id"].(float64))
}

func TestServiceRequestHandler_Create_Success(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	req := dto.CreateServiceRequestRequest{
		CatalogID:     catID,
		Title:         "NewServer-" + srUID(),
		Reason:        "capacity",
		ComplianceAck: true,
	}
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests", req)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	data := resp.Data.(map[string]interface{})
	assert.EqualValues(t, catID, data["catalogId"])
	assert.Equal(t, "submitted", data["status"])
}

func TestServiceRequestHandler_Create_MissingCatalogID(t *testing.T) {
	r, _, _, _, _ := srSetup(t)
	// CatalogID=0 → handler 直接返回 1001
	req := dto.CreateServiceRequestRequest{Title: "X", ComplianceAck: true}
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests", req)
	assert.EqualValues(t, 1001, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_Create_CatalogNotFound(t *testing.T) {
	r, _, _, _, _ := srSetup(t)
	// 不存在的 catalog → service 返回 NotFound → handler 映射 5001
	req := dto.CreateServiceRequestRequest{CatalogID: 999999, Title: "X", ComplianceAck: true}
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests", req)
	assert.EqualValues(t, 5001, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_Create_MissingComplianceAck(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	// ComplianceAck=false → service 返回 BadRequest → handler 映射 5001
	req := dto.CreateServiceRequestRequest{CatalogID: catID, Title: "X"}
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests", req)
	assert.EqualValues(t, 5001, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_Get_Success(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	id := srCreateOne(t, r, catID)
	resp := srDoReq(t, r, "GET", "/api/v1/service-requests/"+strconv.Itoa(id), nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	data := resp.Data.(map[string]interface{})
	assert.EqualValues(t, id, data["id"])
}

func TestServiceRequestHandler_Get_InvalidID(t *testing.T) {
	r, _, _, _, _ := srSetup(t)
	resp := srDoReq(t, r, "GET", "/api/v1/service-requests/abc", nil)
	assert.EqualValues(t, 1001, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_Get_NotFound(t *testing.T) {
	r, _, _, _, _ := srSetup(t)
	resp := srDoReq(t, r, "GET", "/api/v1/service-requests/999999", nil)
	assert.EqualValues(t, 404, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_List(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	srCreateOne(t, r, catID)
	resp := srDoReq(t, r, "GET", "/api/v1/service-requests?page=1&size=10", nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data, "requests")
	assert.Contains(t, data, "total")
}

func TestServiceRequestHandler_UpdateStatus(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	id := srCreateOne(t, r, catID)
	req := dto.UpdateServiceRequestStatusRequest{Status: "approved"} // 归一化为 security_approved
	resp := srDoReq(t, r, "PUT", "/api/v1/service-requests/"+strconv.Itoa(id)+"/status", req)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_UpdateStatus_MissingStatus(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	id := srCreateOne(t, r, catID)
	// status 为空 → binding required 触发 1001
	resp := srDoReq(t, r, "PUT", "/api/v1/service-requests/"+strconv.Itoa(id)+"/status", dto.UpdateServiceRequestStatusRequest{})
	assert.EqualValues(t, 1001, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_Delete(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	id := srCreateOne(t, r, catID)
	resp := srDoReq(t, r, "DELETE", "/api/v1/service-requests/"+strconv.Itoa(id), nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	// 删除后再查应 404
	resp2 := srDoReq(t, r, "GET", "/api/v1/service-requests/"+strconv.Itoa(id), nil)
	assert.EqualValues(t, 404, resp2.Code, "body=%s", srStr(resp2))
}

// --- ApplyApproval（审批动作）路径 ---

// srAuthRole 与 srAuth 类似，但允许指定角色/部门，用于覆盖审批权限分支。
func srAuthRole(tid, uid int, role, dept string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("tenant_id", tid)
		c.Set("user_id", uid)
		c.Set("role", role)
		c.Set("department", dept)
		c.Next()
	}
}

// srSetupRole 组装服务请求 handler，并指定审批人角色。
func srSetupRole(t *testing.T, role, dept string) (*gin.Engine, int, int, int) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_role_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("SRTenant").
		SetCode("SR" + srUID()).
		SetDomain("sr.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	scRepo := service_catalog.NewEntRepository(client)
	scSvc := service_catalog.NewService(scRepo, logger)
	cat, err := scSvc.Create(ctx, "SRCatalog-"+srUID(), "software", "for test", 0, tenant.ID, "enabled", 0, 0)
	require.NoError(t, err)
	repo := NewEntRepository(client)
	cmdbRepo := cmdb.NewEntRepository(client)
	svc := NewService(repo, scRepo, cmdbRepo, logger)
	h := NewHandler(svc)
	uid := 1001
	r := gin.New()
	r.Use(srAuthRole(tenant.ID, uid, role, dept))
	r.POST("/api/v1/service-requests", h.Create)
	r.POST("/api/v1/service-requests/:id/approval", h.ApplyApproval)
	return r, tenant.ID, uid, cat.ID
}

// srApprovals 提取响应里的审批步骤数组，便于按状态断言。
func srApprovals(t *testing.T, resp *common.Response) []map[string]interface{} {
	t.Helper()
	data := resp.Data.(map[string]interface{})
	raw, ok := data["approvals"].([]interface{})
	require.True(t, ok, "approvals field missing: %s", srStr(resp))
	out := make([]map[string]interface{}, 0, len(raw))
	for _, a := range raw {
		out = append(out, a.(map[string]interface{}))
	}
	return out
}

func TestServiceRequestHandler_ApplyApproval_FirstApprove(t *testing.T) {
	r, _, _, catID := srSetupRole(t, "manager", "IT")
	id := srCreateOne(t, r, catID)
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "manager_approved", data["status"])
	apps := srApprovals(t, resp)
	approved, pending := 0, 0
	for _, a := range apps {
		switch a["status"] {
		case "approved":
			approved++
		case "pending":
			pending++
		}
	}
	assert.Equal(t, 1, approved, "exactly one approval should be approved")
	assert.Equal(t, 2, pending, "two approvals should remain pending")
}

func TestServiceRequestHandler_ApplyApproval_InvalidAction(t *testing.T) {
	r, _, _, catID := srSetupRole(t, "manager", "IT")
	id := srCreateOne(t, r, catID)
	// action 不在 {approve,reject} → binding oneof 校验失败 → 1001
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "fly"})
	assert.EqualValues(t, 1001, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_ApplyApproval_RejectRequiresComment(t *testing.T) {
	r, _, _, catID := srSetupRole(t, "manager", "IT")
	id := srCreateOne(t, r, catID)
	// reject 但 comment 为空 → service 返回 BadRequest → handler 映射 5001
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "reject", Comment: ""})
	assert.EqualValues(t, 5001, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_ApplyApproval_Reject(t *testing.T) {
	r, _, _, catID := srSetupRole(t, "manager", "IT")
	id := srCreateOne(t, r, catID)
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "reject", Comment: "policy violation"})
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "rejected", data["status"])
}

func TestServiceRequestHandler_ApplyApproval_InvalidID(t *testing.T) {
	r, _, _, _ := srSetupRole(t, "manager", "IT")
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests/abc/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	assert.EqualValues(t, 1001, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_ApplyApproval_PermissionDenied(t *testing.T) {
	// 当前审批级别为 manager（level 1），但审批人是 agent → 权限不足
	r, _, _, catID := srSetupRole(t, "agent", "IT")
	id := srCreateOne(t, r, catID)
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	// 注意：handler 目前把 service 错误统一映射为 5001；
	// 权限错误理想应返回 2003(Forbidden)，此处先钉住当前行为，作为后续优化点。
	assert.EqualValues(t, 5001, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_ApplyApproval_FullProgression(t *testing.T) {
	// admin 角色可逐级审批全部 3 级
	r, _, _, catID := srSetupRole(t, "admin", "IT")
	id := srCreateOne(t, r, catID)

	// 第 1 级：manager → manager_approved
	resp1 := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	require.Equal(t, common.SuccessCode, resp1.Code, "L1 body=%s", srStr(resp1))
	assert.Equal(t, "manager_approved", resp1.Data.(map[string]interface{})["status"])

	// 第 2 级：it → it_approved
	resp2 := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	require.Equal(t, common.SuccessCode, resp2.Code, "L2 body=%s", srStr(resp2))
	assert.Equal(t, "it_approved", resp2.Data.(map[string]interface{})["status"])

	// 第 3 级：security → security_approved
	resp3 := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	require.Equal(t, common.SuccessCode, resp3.Code, "L3 body=%s", srStr(resp3))
	assert.Equal(t, "security_approved", resp3.Data.(map[string]interface{})["status"])

	// 超量审批：3 级已批完，无待审 → 冲突错误 → handler 映射 5001
	resp4 := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	assert.EqualValues(t, 5001, resp4.Code, "L4 body=%s", srStr(resp4))
}
