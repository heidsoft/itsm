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

// setupAssetController 复用本包既有 helper：withTestAuth / doReq / seedTenantUser / uniqueTestID
func setupAssetController(t *testing.T) (*gin.Engine, *ent.Client, int, int) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "asset_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)

	logger := zaptest.NewLogger(t).Sugar()
	tenantID, userID := seedTenantUser(t, client)
	svc := service.NewAssetService(client, logger)
	ctrl := NewAssetController(logger, svc)

	r := gin.New()
	r.Use(withTestAuth(tenantID, userID))
	r.GET("/api/v1/assets", ctrl.ListAssets)
	r.POST("/api/v1/assets", ctrl.CreateAsset)
	r.GET("/api/v1/assets/stats", ctrl.GetAssetStats)
	r.GET("/api/v1/assets/:id", ctrl.GetAsset)
	r.PUT("/api/v1/assets/:id", ctrl.UpdateAsset)
	r.PUT("/api/v1/assets/:id/status", ctrl.UpdateAssetStatus)
	r.PUT("/api/v1/assets/:id/assign", ctrl.AssignAsset)
	r.PUT("/api/v1/assets/:id/retire", ctrl.RetireAsset)
	r.DELETE("/api/v1/assets/:id", ctrl.DeleteAsset)
	return r, client, tenantID, userID
}

func assetUID() string { return uniqueTestID() }

func TestAssetController_CreateAsset_Success(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	assetNo := "AST-" + assetUID()
	req := dto.CreateAssetRequest{
		AssetNumber: assetNo,
		Name:        "Server-01",
		Type:        "hardware",
		Category:    "server",
	}
	resp := doReq(t, r, "POST", "/api/v1/assets", req, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "Server-01", data["name"])
	assert.Equal(t, assetNo, data["assetNumber"])
}

func TestAssetController_CreateAsset_MissingRequired(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	// 缺少 assetNumber 与 name（均 binding:"required"）→ 4000
	resp := doReq(t, r, "POST", "/api/v1/assets", dto.CreateAssetRequest{Type: "hardware"}, false)
	assert.Equal(t, common.BadRequestCode, resp.Code, "body=%s", mustString(resp))
}

func TestAssetController_GetAsset_Success(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	create := doReq(t, r, "POST", "/api/v1/assets", dto.CreateAssetRequest{
		AssetNumber: "AST-" + assetUID(),
		Name:        "Switch-A",
	}, false)
	require.Equal(t, common.SuccessCode, create.Code)
	id := int(create.Data.(map[string]interface{})["id"].(float64))

	resp := doReq(t, r, "GET", "/api/v1/assets/"+strconv.Itoa(id), nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	assert.Equal(t, "Switch-A", resp.Data.(map[string]interface{})["name"])
}

func TestAssetController_GetAsset_InvalidID(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	resp := doReq(t, r, "GET", "/api/v1/assets/notanint", nil, false)
	assert.Equal(t, common.BadRequestCode, resp.Code, "body=%s", mustString(resp))
}

func TestAssetController_GetAsset_NotFound(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	resp := doReq(t, r, "GET", "/api/v1/assets/999999", nil, false)
	assert.Equal(t, common.NotFoundCode, resp.Code, "body=%s", mustString(resp))
}

func TestAssetController_ListAssets(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	doReq(t, r, "POST", "/api/v1/assets", dto.CreateAssetRequest{
		AssetNumber: "AST-" + assetUID(), Name: "Laptop-1",
	}, false)
	resp := doReq(t, r, "GET", "/api/v1/assets", nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data, "total")
	assert.Contains(t, data, "assets")
}

func TestAssetController_UpdateAsset_Success(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	create := doReq(t, r, "POST", "/api/v1/assets", dto.CreateAssetRequest{
		AssetNumber: "AST-" + assetUID(), Name: "Router-Old",
	}, false)
	require.Equal(t, common.SuccessCode, create.Code)
	id := int(create.Data.(map[string]interface{})["id"].(float64))

	newName := "Router-New"
	resp := doReq(t, r, "PUT", "/api/v1/assets/"+strconv.Itoa(id), dto.UpdateAssetRequest{
		Name: &newName,
	}, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	assert.Equal(t, "Router-New", resp.Data.(map[string]interface{})["name"])
}

func TestAssetController_UpdateAssetStatus_Success(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	create := doReq(t, r, "POST", "/api/v1/assets", dto.CreateAssetRequest{
		AssetNumber: "AST-" + assetUID(), Name: "Printer-1",
	}, false)
	require.Equal(t, common.SuccessCode, create.Code)
	id := int(create.Data.(map[string]interface{})["id"].(float64))

	resp := doReq(t, r, "PUT", "/api/v1/assets/"+strconv.Itoa(id)+"/status",
		dto.AssetStatusUpdateRequest{Status: "in-use"}, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	assert.Equal(t, "in-use", resp.Data.(map[string]interface{})["status"])
}

func TestAssetController_AssignAsset_Success(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	create := doReq(t, r, "POST", "/api/v1/assets", dto.CreateAssetRequest{
		AssetNumber: "AST-" + assetUID(), Name: "Monitor-1",
	}, false)
	require.Equal(t, common.SuccessCode, create.Code)
	id := int(create.Data.(map[string]interface{})["id"].(float64))

	resp := doReq(t, r, "PUT", "/api/v1/assets/"+strconv.Itoa(id)+"/assign",
		dto.AssetAssignRequest{AssignedTo: 42}, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
}

func TestAssetController_RetireAsset_Success(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	create := doReq(t, r, "POST", "/api/v1/assets", dto.CreateAssetRequest{
		AssetNumber: "AST-" + assetUID(), Name: "Old-PC",
	}, false)
	require.Equal(t, common.SuccessCode, create.Code)
	id := int(create.Data.(map[string]interface{})["id"].(float64))

	resp := doReq(t, r, "PUT", "/api/v1/assets/"+strconv.Itoa(id)+"/retire",
		dto.AssetRetireRequest{RetireDate: "2026-07-12", RetireReason: "eol"}, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
}

func TestAssetController_DeleteAsset_Success(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	create := doReq(t, r, "POST", "/api/v1/assets", dto.CreateAssetRequest{
		AssetNumber: "AST-" + assetUID(), Name: "Temp-Dev",
	}, false)
	require.Equal(t, common.SuccessCode, create.Code)
	id := int(create.Data.(map[string]interface{})["id"].(float64))

	del := doReq(t, r, "DELETE", "/api/v1/assets/"+strconv.Itoa(id), nil, false)
	require.Equal(t, common.SuccessCode, del.Code, "body=%s", mustString(del))
}

func TestAssetController_GetAssetStats(t *testing.T) {
	r, _, _, _ := setupAssetController(t)
	doReq(t, r, "POST", "/api/v1/assets", dto.CreateAssetRequest{
		AssetNumber: "AST-" + assetUID(), Name: "Stat-Asset",
	}, false)
	resp := doReq(t, r, "GET", "/api/v1/assets/stats", nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data, "total")
}
