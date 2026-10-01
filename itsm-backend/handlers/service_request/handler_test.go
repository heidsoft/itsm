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
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	entuser "itsm-backend/ent/user"
	"itsm-backend/handlers/cmdb"
	releasehandler "itsm-backend/handlers/release"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/middleware"
	"itsm-backend/service"

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

func createServiceCatalogForTest(ctx context.Context, svc *service_catalog.Service, name, category, description string, deliveryTime, tenantID int, status string, ciTypeID, cloudServiceID int) (*service_catalog.ServiceCatalog, error) {
	return svc.Create(ctx, &service_catalog.ServiceCatalog{
		Name: name, Category: category, Description: description, DeliveryTime: deliveryTime,
		TenantID: tenantID, Status: status, CITypeID: ciTypeID, CloudServiceID: cloudServiceID,
	})
}

// srAuth 注入服务请求 handler 依赖的 c.Get("tenant_id"/"user_id"/"role"/"department")
func srAuth(tid, uid int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tid})
		c.Set("tenant_id", tid)
		c.Set("user_id", uid)
		c.Set("role", "manager")
		c.Set("department", "IT")
		c.Next()
	}
}

func srDoReq(t *testing.T, r *gin.Engine, method, path string, body interface{}) *common.Response {
	t.Helper()
	_, resp := srDoHTTPReq(t, r, method, path, body)
	return resp
}

func srDoHTTPReq(t *testing.T, r *gin.Engine, method, path string, body interface{}) (int, *common.Response) {
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
	return w.Code, &resp
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
	cat, err := createServiceCatalogForTest(ctx, scSvc, "SRCatalog-"+srUID(), "software", "for test", 0, tenant.ID, "enabled", 0, 0)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	cmdbRepo := cmdb.NewEntRepository(client)
	svc := NewService(repo, scRepo, cmdbRepo, client, logger, nil)
	h := NewHandler(svc)

	user, err := client.User.Create().
		SetUsername("sr-user-" + srUID()).
		SetEmail("sr-" + srUID() + "@example.com").
		SetName("SR User").
		SetPasswordHash("hash").
		SetRole("manager").
		SetDepartment("IT").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	uid := user.ID
	r := gin.New()
	r.Use(srAuth(tenant.ID, uid))
	r.POST("/api/v1/service-requests", h.Create)
	r.GET("/api/v1/service-requests", h.List)
	r.GET("/api/v1/service-requests/:id", h.Get)
	r.GET("/api/v1/service-requests/:id/approvals", h.ListApprovals)
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
	assert.EqualValues(t, common.NotFoundErrorCode, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_Create_MissingComplianceAck(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	// ComplianceAck=false → service 返回 BadRequest → handler 映射 5001
	req := dto.CreateServiceRequestRequest{CatalogID: catID, Title: "X"}
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests", req)
	assert.EqualValues(t, common.ParamErrorCode, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestCreateDefersNewCIUntilProvisioning(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_deferred_ci.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	tenant, err := client.Tenant.Create().
		SetName("Deferred CI Tenant").SetCode("DEFER-" + srUID()).SetDomain("defer.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("defer-" + srUID()).SetEmail("defer-" + srUID() + "@example.com").SetName("Requester").
		SetPasswordHash("hash").SetRole("agent").SetDepartment("IT").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Virtual Machine").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	scRepo := service_catalog.NewEntRepository(client)
	catalog, err := createServiceCatalogForTest(ctx, service_catalog.NewService(scRepo, logger),
		"VM Request", "infrastructure", "Provision VM", 24, tenant.ID, "enabled", ciType.ID, 0)
	require.NoError(t, err)
	service := NewService(NewEntRepository(client), scRepo, cmdb.NewEntRepository(client), client, logger, nil)
	expireAt := time.Now().Add(30 * 24 * time.Hour)

	created, err := service.Create(ctx, tenant.ID, user.ID, catalog.ID, &ServiceRequest{
		Title: "Production VM", ComplianceAck: true, DataClassification: "internal", ExpireAt: &expireAt,
	})
	require.NoError(t, err)
	assert.Zero(t, created.CiID)
	ciCount, err := client.ConfigurationItem.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, ciCount, "request submission must not create an active CI before approval")
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

func TestServiceRequestHandler_ListApprovals(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	id := srCreateOne(t, r, catID)

	resp := srDoReq(t, r, "GET", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approvals", nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	approvals, ok := resp.Data.([]interface{})
	require.True(t, ok, "body=%s", srStr(resp))
	require.Len(t, approvals, 3)
	first := approvals[0].(map[string]interface{})
	assert.EqualValues(t, id, first["serviceRequestId"])
	assert.Equal(t, "manager", first["step"])
}

func TestServiceRequestHandler_ListApprovals_RejectsCrossTenant(t *testing.T) {
	r, client, _, _, catID := srSetup(t)
	id := srCreateOne(t, r, catID)

	foreignTenant, err := client.Tenant.Create().
		SetName("Foreign Tenant").
		SetCode("FOREIGN-" + srUID()).
		SetDomain("foreign-" + srUID() + ".test").
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)

	foreignRouter := gin.New()
	foreignRouter.Use(srAuth(foreignTenant.ID, 999999))
	scRepo := service_catalog.NewEntRepository(client)
	svc := NewService(NewEntRepository(client), scRepo, cmdb.NewEntRepository(client), client, zaptest.NewLogger(t).Sugar(), nil)
	foreignRouter.GET("/api/v1/service-requests/:id/approvals", NewHandler(svc).ListApprovals)

	resp := srDoReq(t, foreignRouter, "GET", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approvals", nil)
	assert.EqualValues(t, common.NotFoundErrorCode, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_List(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	srCreateOne(t, r, catID)
	resp := srDoReq(t, r, "GET", "/api/v1/service-requests?page=1&pageSize=10", nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data, "items")
	assert.Contains(t, data, "total")
}

func TestServiceRequestHandler_PartialUpdatePreservesBooleanFields(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	create := srDoReq(t, r, "POST", "/api/v1/service-requests", dto.CreateServiceRequestRequest{
		CatalogID: catID, Title: "Public endpoint", ComplianceAck: true,
		NeedsPublicIP: true, SourceIPWhitelist: []string{"10.0.0.1"},
	})
	require.Equal(t, common.SuccessCode, create.Code, "body=%s", srStr(create))
	id := int(create.Data.(map[string]interface{})["id"].(float64))

	update := srDoReq(t, r, "PUT", "/api/v1/service-requests/"+strconv.Itoa(id),
		dto.UpdateServiceRequestRequest{Title: "Renamed endpoint"})
	require.Equal(t, common.SuccessCode, update.Code, "body=%s", srStr(update))
	data := update.Data.(map[string]interface{})
	assert.Equal(t, true, data["needsPublicIp"])
	assert.Equal(t, true, data["complianceAck"])
}

func TestServiceRequestHandler_UpdateStatusCannotBypassApproval(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	id := srCreateOne(t, r, catID)
	req := dto.UpdateServiceRequestStatusRequest{Status: "approved"} // 归一化为 security_approved
	resp := srDoReq(t, r, "PUT", "/api/v1/service-requests/"+strconv.Itoa(id)+"/status", req)
	assert.EqualValues(t, common.ConflictCode, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_UpdateStatus_MissingStatus(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	id := srCreateOne(t, r, catID)
	// status 为空 → binding required 触发 1001
	resp := srDoReq(t, r, "PUT", "/api/v1/service-requests/"+strconv.Itoa(id)+"/status", dto.UpdateServiceRequestStatusRequest{})
	assert.EqualValues(t, 1001, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_Delete(t *testing.T) {
	r, client, _, _, catID := srSetup(t)
	id := srCreateOne(t, r, catID)
	resp := srDoReq(t, r, "DELETE", "/api/v1/service-requests/"+strconv.Itoa(id), nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	// 删除后再查应 404
	resp2 := srDoReq(t, r, "GET", "/api/v1/service-requests/"+strconv.Itoa(id), nil)
	assert.EqualValues(t, 404, resp2.Code, "body=%s", srStr(resp2))
	stored, err := client.ServiceRequest.Get(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, stored.DeletedAt)
	approvalCount, err := client.ServiceRequestApproval.Query().Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, approvalCount, "soft deletion must preserve approval audit records")
}

// --- ApplyApproval（审批动作）路径 ---

// Snapshot all approval-related rows in this isolated SQLite fixture, including
// both tenants. Rejections must not change timestamps, votes, tasks or audit rows.
func approvalWriteSnapshot(t *testing.T, client *ent.Client) []byte {
	t.Helper()
	ctx := context.Background()
	requests, err := client.ServiceRequest.Query().Order(ent.Asc("id")).All(ctx)
	require.NoError(t, err)
	approvals, err := client.ServiceRequestApproval.Query().Order(ent.Asc("id")).All(ctx)
	require.NoError(t, err)
	releases, err := client.Release.Query().Order(ent.Asc("id")).All(ctx)
	require.NoError(t, err)
	instances, err := client.ProcessInstance.Query().Order(ent.Asc("id")).All(ctx)
	require.NoError(t, err)
	tasks, err := client.ProcessTask.Query().Order(ent.Asc("id")).All(ctx)
	require.NoError(t, err)
	decisions, err := client.ProcessApprovalDecision.Query().Order(ent.Asc("id")).All(ctx)
	require.NoError(t, err)
	audits, err := client.ProcessAuditLog.Query().Order(ent.Asc("id")).All(ctx)
	require.NoError(t, err)
	history, err := client.ProcessExecutionHistory.Query().Order(ent.Asc("id")).All(ctx)
	require.NoError(t, err)
	legacy, err := client.ApprovalRecord.Query().Order(ent.Asc("id")).All(ctx)
	require.NoError(t, err)
	commands, err := client.OperationalCommand.Query().Order(ent.Asc("id")).All(ctx)
	require.NoError(t, err)
	snapshot, err := json.Marshal([]any{requests, approvals, releases, instances, tasks, decisions, audits, history, legacy, commands})
	require.NoError(t, err)
	return snapshot
}

// bpmnAbsentApprovalMessage 三态下两种 fail-closed 成因的文案不同，但业务码同为 conflict：
// 本租户查不到流程实例（noInstance/foreignInstance）由调用方判定为「没有可处理的待办」；
// 已绑定流程却拿不到可操作待办（completedInstance/noPendingTask/foreignTask）由 bridge
// 判定为「流程已绑定但无待办」，后者必须能在响应里被区分出来，否则运维无法判断该补绑定还是补待办。
func bpmnAbsentApprovalMessage(state string) string {
	switch state {
	case "noInstance", "foreignInstance":
		return "没有可处理的 BPMN 审批任务，请先确认流程绑定与待办状态"
	default:
		return "流程已绑定但当前没有可处理的审批待办，请勿直接业务审批"
	}
}

func TestServiceRequestHandler_ApplyApproval_NoBPMNDoesNotWrite(t *testing.T) {
	for _, state := range []string{"noInstance", "completedInstance", "noPendingTask", "foreignInstance", "foreignTask"} {
		for _, action := range []string{"approve", "reject"} {
			t.Run(state+"/"+action, func(t *testing.T) {
				r, client, tenantID, actorID, catalogID := srSetupBridge(t)
				id := srCreateOne(t, r, catalogID)
				ctx := context.Background()
				if state != "noInstance" {
					fixtureTenant := tenantID
					if state == "foreignInstance" {
						foreign, err := client.Tenant.Create().SetName("Foreign").SetCode("foreign").SetStatus("active").Save(ctx)
						require.NoError(t, err)
						fixtureTenant = foreign.ID
					}
					taskID := srCreateBPMNFixture(t, client, fixtureTenant, "absent", id, actorID)
					task, err := client.ProcessTask.Get(ctx, taskID)
					require.NoError(t, err)
					switch state {
					case "completedInstance":
						_, err = client.ProcessInstance.UpdateOneID(task.ProcessInstanceID).SetStatus("completed").Save(ctx)
					case "noPendingTask":
						_, err = client.ProcessTask.UpdateOneID(taskID).SetStatus("completed").Save(ctx)
					case "foreignTask":
						foreign, createErr := client.Tenant.Create().SetName("Foreign").SetCode("foreign").SetStatus("active").Save(ctx)
						require.NoError(t, createErr)
						_, err = client.ProcessTask.UpdateOneID(taskID).SetTenantID(foreign.ID).Save(ctx)
					}
					require.NoError(t, err)
				}
				before := approvalWriteSnapshot(t, client)
				for attempt := 0; attempt < 2; attempt++ {
					status, resp := srDoHTTPReq(t, r, http.MethodPost, fmt.Sprintf("/api/v1/service-requests/%d/approval", id),
						dto.ServiceRequestApprovalActionRequest{Action: action, Comment: "decision"})
					require.Equal(t, http.StatusConflict, status, "body=%s", srStr(resp))
					require.Equal(t, common.ConflictCode, resp.Code)
					require.Equal(t, bpmnAbsentApprovalMessage(state), resp.Message)
					require.Nil(t, resp.Data)
					require.Equal(t, before, approvalWriteSnapshot(t, client), "attempt %d must not write", attempt)
				}
			})
		}
	}
}

// Keep cross-domain HTTP regressions in this existing handler test file: the
// service-package release tests cannot import release handlers without a cycle.
func newReleaseApprovalHTTPFixture(t *testing.T) (*releasehandler.ReleaseHandler, *ent.Client, int, int, int) {
	t.Helper()
	_, client, tenantID, actorID, _ := srSetupBridge(t)
	requesterID := mkChainUser(t, client, tenantID, "end_user", "IT")
	release, err := client.Release.Create().SetReleaseNumber("REL-HTTP").SetTitle("HTTP approval").
		SetStatus("draft").SetTenantID(tenantID).SetCreatedBy(requesterID).Save(context.Background())
	require.NoError(t, err)
	logger := zaptest.NewLogger(t).Sugar()
	return releasehandler.NewHandler(logger, service.NewReleaseService(client, logger)), client, tenantID, actorID, release.ID
}

func releaseCreateBPMNFixture(t *testing.T, client *ent.Client, tenantID, releaseID, actorID int) int {
	t.Helper()
	taskID := srCreateBPMNFixture(t, client, tenantID, "release", releaseID, actorID)
	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	businessKey := fmt.Sprintf("release:%d", releaseID)
	// These are persisted BPMN engine context keys, not HTTP request/response keys.
	_, err = client.ProcessInstance.UpdateOneID(task.ProcessInstanceID).SetBusinessKey(businessKey).
		SetVariables(map[string]interface{}{
			"business_type": "release", "business_id": strconv.Itoa(releaseID), "business_key": businessKey,
		}).Save(context.Background())
	require.NoError(t, err)
	return taskID
}

func releaseApprovalHTTPBody(action string) map[string]string {
	if action == "reject" {
		return map[string]string{"reason": "decision"}
	}
	return map[string]string{"comment": "decision"}
}

func TestReleaseApprovalHTTP_NoBPMNDoesNotWrite(t *testing.T) {
	for _, state := range []string{"noInstance", "noPendingTask", "foreignInstance"} {
		for _, action := range []string{"approve", "reject"} {
			t.Run(state+"/"+action, func(t *testing.T) {
				h, client, tenantID, actorID, releaseID := newReleaseApprovalHTTPFixture(t)
				ctx := context.Background()
				if state != "noInstance" {
					fixtureTenant := tenantID
					if state == "foreignInstance" {
						foreign, err := client.Tenant.Create().SetName("Foreign").SetCode("foreign").SetStatus("active").Save(ctx)
						require.NoError(t, err)
						fixtureTenant = foreign.ID
					}
					taskID := releaseCreateBPMNFixture(t, client, fixtureTenant, releaseID, actorID)
					if state == "noPendingTask" {
						_, err := client.ProcessTask.UpdateOneID(taskID).SetStatus("completed").Save(ctx)
						require.NoError(t, err)
					}
				}
				r := gin.New()
				r.Use(_srAuthRole(tenantID, actorID, "manager", "IT"))
				r.POST("/api/v1/releases/:id/approve", h.ApproveRelease)
				r.POST("/api/v1/releases/:id/reject", h.RejectRelease)
				before := approvalWriteSnapshot(t, client)
				for attempt := 0; attempt < 2; attempt++ {
					status, resp := srDoHTTPReq(t, r, http.MethodPost, fmt.Sprintf("/api/v1/releases/%d/%s", releaseID, action),
						releaseApprovalHTTPBody(action))
					require.Equal(t, http.StatusConflict, status, "body=%s", srStr(resp))
					require.Equal(t, common.ConflictCode, resp.Code)
					require.Equal(t, bpmnAbsentApprovalMessage(state), resp.Message)
					require.Nil(t, resp.Data)
					require.Equal(t, before, approvalWriteSnapshot(t, client))
				}
			})
		}
	}
}

func TestReleaseApprovalHTTP_BPMNSuccess(t *testing.T) {
	for _, action := range []string{"approve", "reject"} {
		t.Run(action, func(t *testing.T) {
			h, client, tenantID, actorID, releaseID := newReleaseApprovalHTTPFixture(t)
			taskID := releaseCreateBPMNFixture(t, client, tenantID, releaseID, actorID)
			r := gin.New()
			r.Use(_srAuthRole(tenantID, actorID, "manager", "IT"))
			r.POST("/api/v1/releases/:id/approve", h.ApproveRelease)
			r.POST("/api/v1/releases/:id/reject", h.RejectRelease)
			status, resp := srDoHTTPReq(t, r, http.MethodPost, fmt.Sprintf("/api/v1/releases/%d/%s", releaseID, action),
				releaseApprovalHTTPBody(action))
			require.Equal(t, http.StatusOK, status, "body=%s", srStr(resp))
			require.Equal(t, common.SuccessCode, resp.Code)
			data := resp.Data.(map[string]interface{})
			expected := "scheduled"
			if action == "reject" {
				expected = "cancelled"
			}
			require.Equal(t, expected, data["status"])
			require.EqualValues(t, releaseID, data["id"])
			require.Contains(t, data, "releaseNumber")
			require.NotContains(t, data, "release_number")
			require.NotContains(t, data, "passwordHash")
			task, err := client.ProcessTask.Get(context.Background(), taskID)
			require.NoError(t, err)
			require.Equal(t, "completed", task.Status)
			decision, err := client.ProcessApprovalDecision.Query().Only(context.Background())
			require.NoError(t, err)
			require.Equal(t, actorID, decision.ActorID)
		})
	}
}

func TestBusinessApprovalHTTP_TenantAndIdentityIsolation(t *testing.T) {
	for _, domain := range []string{"serviceRequest", "release"} {
		for _, identity := range []string{"foreignTenant", "foreignActor", "missingTenant", "missingActor"} {
			t.Run(domain+"/"+identity, func(t *testing.T) {
				r, client, tenantID, actorID, catalogID := srSetupBridge(t)
				id := srCreateOne(t, r, catalogID)
				srCreateBPMNFixture(t, client, tenantID, "isolation", id, actorID)
				ctx := context.Background()
				logger := zaptest.NewLogger(t).Sugar()
				svc := NewService(NewEntRepository(client), service_catalog.NewEntRepository(client), cmdb.NewEntRepository(client), client, logger, nil)
				handler := NewHandler(svc).ApplyApproval
				path := fmt.Sprintf("/api/v1/service-requests/%d/approval", id)
				if domain == "release" {
					requesterID := mkChainUser(t, client, tenantID, "end_user", "IT")
					rel, err := client.Release.Create().SetReleaseNumber("REL-ISOLATION").SetTitle("Protected release").
						SetStatus("draft").SetTenantID(tenantID).SetCreatedBy(requesterID).Save(ctx)
					require.NoError(t, err)
					releaseCreateBPMNFixture(t, client, tenantID, rel.ID, actorID)
					handler = releasehandler.NewHandler(logger, service.NewReleaseService(client, logger)).ApproveRelease
					path = fmt.Sprintf("/api/v1/releases/%d/approve", rel.ID)
				}
				foreign, err := client.Tenant.Create().SetName("Foreign").SetCode("foreign").SetStatus("active").Save(ctx)
				require.NoError(t, err)
				foreignActor := mkChainUser(t, client, foreign.ID, "manager", "IT")
				authTenant, authActor := tenantID, actorID
				expectedStatus, expectedCode := http.StatusUnauthorized, common.UnauthorizedCode
				switch identity {
				case "foreignTenant":
					authTenant, authActor = foreign.ID, foreignActor
					expectedStatus, expectedCode = http.StatusNotFound, common.NotFoundCode
				case "foreignActor":
					authActor = foreignActor
					expectedStatus, expectedCode = http.StatusForbidden, common.ForbiddenCode
				case "missingTenant":
					authTenant = 0
				case "missingActor":
					authActor = 0
				}
				// Register the same route pattern as the production router.
				pattern := "/api/v1/service-requests/:id/approval"
				if domain == "release" {
					pattern = "/api/v1/releases/:id/approve"
				}
				r = gin.New()
				r.Use(_srAuthRole(authTenant, authActor, "manager", "IT"))
				r.POST(pattern, handler)
				before := approvalWriteSnapshot(t, client)
				for attempt := 0; attempt < 2; attempt++ {
					// Body/query/header identity claims must not replace authenticated context.
					body := fmt.Sprintf(`{"action":"approve","comment":"decision","tenantId":%d,"userId":%d}`, tenantID, actorID)
					req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s?tenantId=%d&userId=%d", path, tenantID, actorID), strings.NewReader(body))
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("X-Tenant-ID", strconv.Itoa(tenantID))
					req.Header.Set("X-User-ID", strconv.Itoa(actorID))
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)
					var resp common.Response
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
					require.Equal(t, expectedStatus, w.Code, "body=%s", w.Body.String())
					require.Equal(t, expectedCode, resp.Code)
					require.Nil(t, resp.Data)
					require.Equal(t, before, approvalWriteSnapshot(t, client))
				}
			})
		}
	}
}

// _srAuthRole 与 srAuth 类似，但允许指定角色/部门，用于覆盖审批权限分支。
func _srAuthRole(tid, uid int, role, dept string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tid})
		c.Set("tenant_id", tid)
		c.Set("user_id", uid)
		c.Set("role", role)
		c.Set("department", dept)
		c.Next()
	}
}

func srAuthRoleActors(tid, requesterID, actorID int, role, dept string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := actorID
		if c.Request.Method == http.MethodPost &&
			strings.HasSuffix(c.Request.URL.Path, "/service-requests") {
			uid = requesterID
		}
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tid})
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
	r, _, tenantID, actorID, catalogID := srSetupRoleClient(t, role, dept)
	return r, tenantID, actorID, catalogID
}

func srSetupRoleClient(t *testing.T, role, dept string) (*gin.Engine, *ent.Client, int, int, int) {
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
	cat, err := createServiceCatalogForTest(ctx, scSvc, "SRCatalog-"+srUID(), "software", "for test", 0, tenant.ID, "enabled", 0, 0)
	require.NoError(t, err)
	repo := NewEntRepository(client)
	cmdbRepo := cmdb.NewEntRepository(client)
	svc := NewService(repo, scRepo, cmdbRepo, client, logger, nil)
	h := NewHandler(svc)
	requester, err := client.User.Create().
		SetUsername("sr-requester-" + srUID()).
		SetEmail("sr-requester-" + srUID() + "@example.com").
		SetName("SR Requester").
		SetPasswordHash("hash").
		SetRole("agent").
		SetDepartment(dept).
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("sr-role-user-" + srUID()).
		SetEmail("sr-role-" + srUID() + "@example.com").
		SetName("SR Role User").
		SetPasswordHash("hash").
		SetRole(entuser.Role(role)).
		SetDepartment(dept).
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	uid := user.ID
	r := gin.New()
	r.Use(srAuthRoleActors(tenant.ID, requester.ID, uid, role, dept))
	r.POST("/api/v1/service-requests", h.Create)
	r.POST("/api/v1/service-requests/:id/approval", h.ApplyApproval)
	r.PUT("/api/v1/service-requests/:id", h.Update)
	r.PUT("/api/v1/service-requests/:id/status", h.UpdateStatus)
	r.DELETE("/api/v1/service-requests/:id", h.Delete)
	return r, client, tenant.ID, uid, cat.ID
}

// srCreateSerialBPMNFixture seeds the deployed graph and its first pending task.
// Only the real engine may create subsequent tasks when the HTTP action completes
// the current one; no business approval fallback or mock success is involved.
func srCreateSerialBPMNFixture(t *testing.T, client *ent.Client, tenantID, requestID int, assignees ...int) int {
	t.Helper()
	require.NotEmpty(t, assignees)
	taskID := srCreateBPMNFixture(t, client, tenantID, "serial", requestID, assignees[0])
	ctx := context.Background()
	task, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	instance, err := client.ProcessInstance.Get(ctx, task.ProcessInstanceID)
	require.NoError(t, err)
	var nodes strings.Builder
	for i, assignee := range assignees {
		fmt.Fprintf(&nodes, `<bpmn:userTask id="Approval_%d" name="Approval %d" itsm:taskPurpose="approval" itsm:approvalMode="single" itsm:assignee="%d"/>`, i+1, i+1, assignee)
		if i > 0 {
			fmt.Fprintf(&nodes, `<bpmn:sequenceFlow id="Next_%d" sourceRef="Approval_%d" targetRef="Approval_%d"/>`, i, i, i+1)
		}
	}
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:itsm="https://github.com/heidsoft/itsm/schema/bpmn" targetNamespace="https://github.com/heidsoft/itsm">
<bpmn:process id="%s" isExecutable="true"><bpmn:startEvent id="StartEvent_1"/>
%s<bpmn:endEvent id="EndEvent_1"/>
<bpmn:sequenceFlow id="Start" sourceRef="StartEvent_1" targetRef="Approval_1"/>
<bpmn:sequenceFlow id="End" sourceRef="Approval_%d" targetRef="EndEvent_1"/>
</bpmn:process></bpmn:definitions>`, instance.ProcessDefinitionKey, nodes.String(), len(assignees))
	_, err = client.ProcessDefinition.UpdateOneID(instance.ProcessDefinitionID).SetBpmnXML([]byte(xml)).Save(ctx)
	require.NoError(t, err)
	return instance.ID
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
	r, client, tenantID, actorID, catID := srSetupBridge(t)
	id := srCreateOne(t, r, catID)
	taskID := srCreateBPMNFixture(t, client, tenantID, "approve", id, actorID)
	status, resp := srDoHTTPReq(t, r, http.MethodPost, fmt.Sprintf("/api/v1/service-requests/%d/approval", id),
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	require.Equal(t, "completed", task.Status)
	require.Equal(t, "approved", task.TaskVariables["approvalResult"])
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
	assert.EqualValues(t, common.ParamErrorCode, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_ApplyApproval_Reject(t *testing.T) {
	r, client, tenantID, actorID, catID := srSetupBridge(t)
	id := srCreateOne(t, r, catID)
	taskID := srCreateBPMNFixture(t, client, tenantID, "reject", id, actorID)
	status, resp := srDoHTTPReq(t, r, http.MethodPost, fmt.Sprintf("/api/v1/service-requests/%d/approval", id),
		dto.ServiceRequestApprovalActionRequest{Action: "reject", Comment: "policy violation"})
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "rejected", data["status"])
	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	require.Equal(t, "completed", task.Status)
	require.Equal(t, "rejected", task.TaskVariables["approvalResult"])
}

func TestServiceRequestHandler_ApplyApproval_InvalidID(t *testing.T) {
	r, _, _, _ := srSetupRole(t, "manager", "IT")
	for _, id := range []string{"abc", "0", "-1"} {
		status, resp := srDoHTTPReq(t, r, http.MethodPost, fmt.Sprintf("/api/v1/service-requests/%s/approval", id),
			dto.ServiceRequestApprovalActionRequest{Action: "approve"})
		require.Equal(t, http.StatusBadRequest, status)
		require.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", srStr(resp))
	}
}

func TestServiceRequestHandler_ApplyApproval_PermissionDenied(t *testing.T) {
	// 当前审批级别为 manager（level 1），但审批人是 agent → 权限不足
	r, _, _, catID := srSetupRole(t, "agent", "IT")
	id := srCreateOne(t, r, catID)
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	// 注意：handler 目前把 service 错误统一映射为 5001；
	// 权限错误理想应返回 2003(Forbidden)，此处先钉住当前行为，作为后续优化点。
	assert.EqualValues(t, common.ForbiddenErrorCode, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestHandler_RequesterCannotSelfApprove(t *testing.T) {
	r, _, _, _, catID := srSetup(t)
	id := srCreateOne(t, r, catID)
	resp := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	assert.EqualValues(t, common.ForbiddenErrorCode, resp.Code, "body=%s", srStr(resp))
}

func TestServiceRequestPendingApprovalsAreDepartmentScopedAndUnknownRoleDenied(t *testing.T) {
	_, client, tenantID, managerID, catID := srSetup(t)
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	scRepo := service_catalog.NewEntRepository(client)
	service := NewService(NewEntRepository(client), scRepo, cmdb.NewEntRepository(client), client, logger, nil)
	itRequester, err := client.User.Create().
		SetUsername("it-requester-" + srUID()).SetEmail("it-" + srUID() + "@example.com").SetName("IT Requester").
		SetPasswordHash("hash").SetRole("agent").SetDepartment("IT").SetActive(true).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	hrRequester, err := client.User.Create().
		SetUsername("hr-requester-" + srUID()).SetEmail("hr-" + srUID() + "@example.com").SetName("HR Requester").
		SetPasswordHash("hash").SetRole("agent").SetDepartment("HR").SetActive(true).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	expireAt := time.Now().Add(30 * 24 * time.Hour)
	for _, requesterID := range []int{itRequester.ID, hrRequester.ID} {
		_, err = service.Create(ctx, tenantID, requesterID, catID, &ServiceRequest{
			Title: "Department scoped request", ComplianceAck: true,
			DataClassification: "internal", ExpireAt: &expireAt,
		})
		require.NoError(t, err)
	}

	pending, total, err := service.ListPendingApprovals(ctx, tenantID, managerID, "manager", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, pending, 1)
	assert.Equal(t, itRequester.ID, pending[0].RequesterID)

	_, _, err = service.ListPendingApprovals(ctx, tenantID, managerID, "viewer", 1, 10)
	require.Error(t, err)
}

func TestServiceRequestHandler_ApplyApproval_FullProgression(t *testing.T) {
	// Admin is eligible for all levels, but each action still needs its own BPMN task.
	r, client, tenantID, actorID, catID := srSetupRoleClient(t, "admin", "IT")
	id := srCreateOne(t, r, catID)
	instanceID := srCreateSerialBPMNFixture(t, client, tenantID, id, actorID, actorID, actorID)
	path := fmt.Sprintf("/api/v1/service-requests/%d/approval", id)
	for level, expected := range []string{"manager_approved", "it_approved", "security_approved"} {
		status, resp := srDoHTTPReq(t, r, http.MethodPost, path, dto.ServiceRequestApprovalActionRequest{Action: "approve"})
		require.Equal(t, http.StatusOK, status, "level %d: %s", level+1, srStr(resp))
		require.Equal(t, common.SuccessCode, resp.Code)
		data := resp.Data.(map[string]interface{})
		require.Equal(t, expected, data["status"])
		nextLevel := level + 2
		if nextLevel > 3 {
			nextLevel = 3
		}
		require.EqualValues(t, nextLevel, data["currentLevel"])
		require.NotContains(t, data, "current_level")
		require.NotContains(t, data, "tenantId")
	}
	instance, err := client.ProcessInstance.Get(context.Background(), instanceID)
	require.NoError(t, err)
	require.Equal(t, "completed", instance.Status)
	tasks, err := client.ProcessTask.Query().All(context.Background())
	require.NoError(t, err)
	require.Len(t, tasks, 3, "engine must create the next two tasks")
	for _, task := range tasks {
		require.Equal(t, "completed", task.Status)
	}
	decisions, err := client.ProcessApprovalDecision.Query().Count(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3, decisions)

	before := approvalWriteSnapshot(t, client)
	status, resp := srDoHTTPReq(t, r, http.MethodPost, path, dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, common.ConflictCode, resp.Code)
	require.Equal(t, before, approvalWriteSnapshot(t, client))
}

func TestGoldenJourney_ServiceRequestApprovedProvisionedAndDelivered(t *testing.T) {
	r, client, tenantID, actorID, catID := srSetupRoleClient(t, "admin", "IT")
	id := srCreateOne(t, r, catID)
	srCreateSerialBPMNFixture(t, client, tenantID, id, actorID, actorID, actorID)
	for i := 0; i < 3; i++ {
		resp := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
			dto.ServiceRequestApprovalActionRequest{Action: "approve"})
		require.Equal(t, common.SuccessCode, resp.Code, "approval %d: %s", i+1, srStr(resp))
	}

	provisioning := srDoReq(t, r, "PUT", "/api/v1/service-requests/"+strconv.Itoa(id)+"/status",
		dto.UpdateServiceRequestStatusRequest{Status: "in_progress"})
	require.Equal(t, common.SuccessCode, provisioning.Code, "body=%s", srStr(provisioning))
	assert.Equal(t, "provisioning", provisioning.Data.(map[string]interface{})["status"])
	assert.NotNil(t, provisioning.Data.(map[string]interface{})["processorId"])
	assert.NotNil(t, provisioning.Data.(map[string]interface{})["startedAt"])

	delivered := srDoReq(t, r, "PUT", "/api/v1/service-requests/"+strconv.Itoa(id)+"/status",
		dto.UpdateServiceRequestStatusRequest{Status: "completed"})
	require.Equal(t, common.SuccessCode, delivered.Code, "body=%s", srStr(delivered))
	assert.Equal(t, "delivered", delivered.Data.(map[string]interface{})["status"])
	assert.NotNil(t, delivered.Data.(map[string]interface{})["completedAt"])

	edit := srDoReq(t, r, "PUT", "/api/v1/service-requests/"+strconv.Itoa(id),
		dto.UpdateServiceRequestRequest{Title: "must not change"})
	assert.EqualValues(t, common.ConflictCode, edit.Code)
	deleteResp := srDoReq(t, r, "DELETE", "/api/v1/service-requests/"+strconv.Itoa(id), nil)
	assert.EqualValues(t, common.ConflictCode, deleteResp.Code)
}
