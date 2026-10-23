package service_request

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	entuser "itsm-backend/ent/user"
	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/cmdb"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupQuorumProgress 组装一条「单级会签（2 名 manager，阈值 2）」审批链。
// 审批人通过 X-Test-Actor 头切换，模拟会签的两票来自两个不同账号。
func setupQuorumProgress(t *testing.T) (*gin.Engine, *ent.Client, int, int, int, int, int) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_quorum_progress.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("QuorumTenant").SetCode("QP" + srUID()).SetDomain("qp.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	scRepo := service_catalog.NewEntRepository(client)
	scSvc := service_catalog.NewService(scRepo, logger)
	cat, err := createServiceCatalogForTest(ctx, scSvc, "QuorumCatalog-"+srUID(), "software", "for test", 0, tenant.ID, "enabled", 0, 0)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	cmdbRepo := cmdb.NewEntRepository(client)
	acs := service.NewApprovalChainService(client, logger)
	_, err = acs.CreateApprovalChain(ctx, &dto.ApprovalChainRequest{
		Name:       "SR Quorum Chain",
		EntityType: "service_request",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "双人主管会签", IsRequired: true, ApprovalType: "parallel", Threshold: 2},
		},
	}, tenant.ID)
	require.NoError(t, err)

	svc := NewService(repo, scRepo, cmdbRepo, client, logger, acs)
	h := NewHandler(svc)

	mkUser := func(role string) int {
		u, err := client.User.Create().
			SetUsername("qp-" + role + "-" + srUID()).
			SetEmail("qp-" + role + "-" + srUID() + "@example.com").
			SetName("QP " + role).
			SetPasswordHash("h").
			SetRole(entuser.Role(role)).
			SetDepartment("IT").
			SetActive(true).
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		return u.ID
	}
	requester := mkUser("end_user")
	approver1 := mkUser("manager")
	approver2 := mkUser("manager")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		uid := approver1
		if c.Request.Method == http.MethodPost && strings.HasSuffix(c.Request.URL.Path, "/service-requests") {
			uid = requester // 创建请求的永远是请求人
		} else if v := c.GetHeader("X-Test-Actor"); v != "" {
			switch v {
			case "approver1":
				uid = approver1
			case "approver2":
				uid = approver2
			}
		}
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Set("tenant_id", tenant.ID)
		c.Set("user_id", uid)
		c.Set("role", "manager")
		c.Set("department", "IT")
		c.Next()
	})
	r.POST("/api/v1/service-requests", h.Create)
	r.POST("/api/v1/service-requests/:id/approval", h.ApplyApproval)
	return r, client, tenant.ID, cat.ID, requester, approver1, approver2
}

// 会签第一票：请求 200（票已记）但状态不动，message 必须说清「已批 1/2，还需 1 人」。
// 修复前这里只有裸 Success——运维连点审批看到的都是成功，状态却原地不动。
func TestApplyApprovalQuorumNotMetSurfacesProgressMessage(t *testing.T) {
	r, client, tenantID, catalogID, _, approver1, _ := setupQuorumProgress(t)

	resp := srDoReq(t, r, http.MethodPost, "/api/v1/service-requests", dto.CreateServiceRequestRequest{
		CatalogID:          catalogID,
		Title:              "Quorum SR-" + srUID(),
		Reason:             "need resource",
		ComplianceAck:      true,
		DataClassification: "internal",
	})
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))
	id := int(resp.Data.(map[string]interface{})["id"].(float64))

	// BPMN 待办是审批写入的前置条件（与生产链路一致）。
	srCreateBPMNFixture(t, client, tenantID, "approve", id, approver1)

	body, err := json.Marshal(dto.ServiceRequestApprovalActionRequest{Action: "approve", Comment: "第一票"})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/service-requests/"+itoaSR(id)+"/approval", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Actor", "approver1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var out common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out), "body=%s", w.Body.String())
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	require.Equal(t, common.SuccessCode, out.Code)

	assert.Contains(t, out.Message, "会签未满足", "会签未满足必须出现在 message: %q", out.Message)
	assert.Contains(t, out.Message, "1/2", "必须给出已批/阈值进度: %q", out.Message)

	data := out.Data.(map[string]interface{})
	assert.Equal(t, "submitted", data["status"], "quorum 未满足不得推进状态机")
	assert.Equal(t, float64(1), data["currentLevel"], "quorum 未满足不得推进层级")

	// 审批行记录投票痕迹（action/approver/processedAt），但状态仍 pending。
	approvals := data["approvals"].([]interface{})
	require.Len(t, approvals, 1)
	first := approvals[0].(map[string]interface{})
	assert.Equal(t, "pending", first["status"], "票已记但级未过：状态保持 pending")
	assert.Equal(t, "approve", first["action"])
}

// 纯函数单测：串行审批（无 approver_ids）、无 node 都不得产出会签文案；
// 本级已审批（已推进到下一级）也不得产出。防止 message 逻辑泛滥到普通审批响应上。
func TestQuorumProgressMessageOnlyForPendingQuorum(t *testing.T) {
	nodeWith := func(approvers, approved int, threshold float64) map[string]interface{} {
		ids := make([]interface{}, approvers)
		for i := range ids {
			ids[i] = float64(i + 1)
		}
		appr := make([]interface{}, approved)
		for i := range appr {
			appr[i] = float64(i + 1)
		}
		return map[string]interface{}{
			"approver_ids":     ids,
			"approved_ids":     appr,
			"quorum_threshold": threshold,
		}
	}

	cases := []struct {
		name string
		req  *ServiceRequest
		apps []*ServiceRequestApproval
		want string
	}{
		{
			name: "串行审批无会签配置",
			req:  &ServiceRequest{CurrentLevel: 1},
			apps: []*ServiceRequestApproval{{Level: 1, Status: ApprovalStatusPending, Node: map[string]interface{}{}}},
			want: "",
		},
		{
			name: "无 node",
			req:  &ServiceRequest{CurrentLevel: 1},
			apps: []*ServiceRequestApproval{{Level: 1, Status: ApprovalStatusPending}},
			want: "",
		},
		{
			name: "本级已审批（已推进到下一级）",
			req:  &ServiceRequest{CurrentLevel: 2},
			apps: []*ServiceRequestApproval{
				{Level: 1, Status: ApprovalStatusApproved, Node: nodeWith(2, 2, 2)},
				{Level: 2, Status: ApprovalStatusPending, Node: nodeWith(2, 0, 2)},
			},
			want: "",
		},
		{
			name: "会签未满足",
			req:  &ServiceRequest{CurrentLevel: 1},
			apps: []*ServiceRequestApproval{{Level: 1, Status: ApprovalStatusPending, Node: nodeWith(2, 1, 2)}},
			want: "已批 1/2 人），还需 1 人",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := quorumProgressMessage(tc.req, tc.apps)
			if tc.want == "" {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, tc.want)
		})
	}
}

func itoaSR(v int) string {
	return strconv.Itoa(v)
}
