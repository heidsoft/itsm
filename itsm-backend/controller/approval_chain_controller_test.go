package controller

import (
	"path/filepath"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupApprovalChainController(t *testing.T) (*gin.Engine, *ent.Client, int, int) {
	gin.SetMode(gin.TestMode)
	dsn := "file:" + filepath.Join(t.TempDir(), "approval_chain_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()

	tenantID, userID := seedTenantUser(t, client)

	svc := service.NewApprovalChainService(client, logger)
	ctrl := NewApprovalChainController(svc, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(withTestAuth(tenantID, userID))
	r.POST("/api/v1/approval-chains", ctrl.CreateChain)
	r.GET("/api/v1/approval-chains", ctrl.ListChains)
	r.GET("/api/v1/approval-chains/stats", ctrl.GetStats)
	r.GET("/api/v1/approval-chains/:id", ctrl.GetChain)
	r.PUT("/api/v1/approval-chains/:id", ctrl.UpdateChain)
	r.DELETE("/api/v1/approval-chains/:id", ctrl.DeleteChain)
	return r, client, tenantID, userID
}

func TestApprovalChainController_CreateChain(t *testing.T) {
	r, _, _, _ := setupApprovalChainController(t)

	t.Run("成功创建审批链", func(t *testing.T) {
		body := dto.ApprovalChainRequest{
			Name:       "变更审批链",
			EntityType: "change",
			Chain: []dto.ApprovalChainStepDTO{
				{Level: 1, Role: "manager", Name: "经理审批", IsRequired: true},
				{Level: 2, Role: "director", Name: "总监审批", IsRequired: true},
			},
		}
		resp := doReq(t, r, "POST", "/api/v1/approval-chains", body, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "变更审批链", data["name"])
		assert.Equal(t, float64(2), float64(len(data["chain"].([]interface{}))))
	})

	t.Run("缺少名称应返回参数错误", func(t *testing.T) {
		body := dto.ApprovalChainRequest{EntityType: "change", Chain: []dto.ApprovalChainStepDTO{{Level: 1, Role: "x", Name: "n"}}}
		resp := doReq(t, r, "POST", "/api/v1/approval-chains", body, false)
		assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("缺少租户上下文应返回未授权", func(t *testing.T) {
		body := dto.ApprovalChainRequest{Name: "x", EntityType: "change", Chain: []dto.ApprovalChainStepDTO{{Level: 1, Role: "x", Name: "n"}}}
		resp := doReq(t, r, "POST", "/api/v1/approval-chains", body, true)
		assert.Equal(t, common.UnauthorizedCode, resp.Code, "body=%s", mustString(resp))
	})
}

func TestApprovalChainController_ListAndStats(t *testing.T) {
	r, _, _, _ := setupApprovalChainController(t)

	t.Run("列表返回成功", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/approval-chains", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("统计返回成功", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/approval-chains/stats", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	})
}

func TestApprovalChainController_GetChain(t *testing.T) {
	r, _, _, _ := setupApprovalChainController(t)

	created := doReq(t, r, "POST", "/api/v1/approval-chains", dto.ApprovalChainRequest{
		Name:       "查询测试链",
		EntityType: "change",
		Chain:      []dto.ApprovalChainStepDTO{{Level: 1, Role: "x", Name: "n"}},
	}, false)
	require.Equal(t, common.SuccessCode, created.Code)
	id := int(created.Data.(map[string]interface{})["id"].(float64))

	t.Run("按ID获取成功", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/approval-chains/"+strconv.Itoa(id), nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("非法ID应返回参数错误", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/approval-chains/abc", nil, false)
		assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("不存在的ID应返回未找到", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/approval-chains/999999", nil, false)
		assert.Equal(t, common.NotFoundCode, resp.Code, "body=%s", mustString(resp))
	})
}
