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

// setupLicenseController 复用本包既有 helper：withTestAuth / doReq / seedTenantUser / uniqueTestID / mustString
func setupLicenseController(t *testing.T) (*gin.Engine, *ent.Client, int, int) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "license_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)

	logger := zaptest.NewLogger(t).Sugar()
	tenantID, userID := seedTenantUser(t, client)
	svc := service.NewAssetLicenseService(client, logger)
	ctrl := NewAssetLicenseController(logger, svc)

	r := gin.New()
	r.Use(withTestAuth(tenantID, userID))
	r.GET("/api/v1/licenses", ctrl.ListLicenses)
	r.POST("/api/v1/licenses", ctrl.CreateLicense)
	r.GET("/api/v1/licenses/stats", ctrl.GetLicenseStats)
	r.GET("/api/v1/licenses/:id", ctrl.GetLicense)
	r.PUT("/api/v1/licenses/:id", ctrl.UpdateLicense)
	r.PUT("/api/v1/licenses/:id/assign", ctrl.AssignUsers)
	r.DELETE("/api/v1/licenses/:id", ctrl.DeleteLicense)
	return r, client, tenantID, userID
}

// createTestLicense 播种一条许可证，返回其 ID
func createTestLicense(t *testing.T, r *gin.Engine, name string) int {
	t.Helper()
	req := dto.CreateLicenseRequest{
		Name:          name,
		Vendor:        "ACME",
		LicenseType:   string(dto.LicenseTypeSubscription),
		TotalQuantity: 10,
	}
	resp := doReq(t, r, "POST", "/api/v1/licenses", req, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	return int(data["id"].(float64))
}

func TestAssetLicenseController_CreateLicense_Success(t *testing.T) {
	r, _, _, _ := setupLicenseController(t)
	name := "Office-" + uniqueTestID()
	req := dto.CreateLicenseRequest{
		Name:          name,
		Vendor:        "Microsoft",
		LicenseType:   string(dto.LicenseTypePerUser),
		TotalQuantity: 50,
	}
	resp := doReq(t, r, "POST", "/api/v1/licenses", req, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, name, data["name"])
	assert.EqualValues(t, 50, data["totalQuantity"])
}

func TestAssetLicenseController_CreateLicense_MissingName(t *testing.T) {
	r, _, _, _ := setupLicenseController(t)
	// 缺少 name（binding:"required"）→ BadRequest
	resp := doReq(t, r, "POST", "/api/v1/licenses", dto.CreateLicenseRequest{Vendor: "X"}, false)
	assert.Equal(t, common.BadRequestCode, resp.Code, "body=%s", mustString(resp))
}

func TestAssetLicenseController_CreateLicense_Unauthorized(t *testing.T) {
	r, _, _, _ := setupLicenseController(t)
	// 跳过鉴权注入 → 无租户上下文 → Unauthorized
	resp := doReq(t, r, "POST", "/api/v1/licenses", dto.CreateLicenseRequest{Name: "X"}, true)
	assert.Equal(t, common.UnauthorizedCode, resp.Code, "body=%s", mustString(resp))
}

func TestAssetLicenseController_GetLicense_Success(t *testing.T) {
	r, _, _, _ := setupLicenseController(t)
	id := createTestLicense(t, r, "GetLic-"+uniqueTestID())
	resp := doReq(t, r, "GET", "/api/v1/licenses/"+strconv.Itoa(id), nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.EqualValues(t, id, data["id"])
}

func TestAssetLicenseController_GetLicense_InvalidID(t *testing.T) {
	r, _, _, _ := setupLicenseController(t)
	resp := doReq(t, r, "GET", "/api/v1/licenses/abc", nil, false)
	assert.Equal(t, common.BadRequestCode, resp.Code, "body=%s", mustString(resp))
}

func TestAssetLicenseController_GetLicense_NotFound(t *testing.T) {
	r, _, _, _ := setupLicenseController(t)
	resp := doReq(t, r, "GET", "/api/v1/licenses/999999", nil, false)
	assert.Equal(t, common.NotFoundCode, resp.Code, "body=%s", mustString(resp))
}

func TestAssetLicenseController_ListLicenses(t *testing.T) {
	r, _, _, _ := setupLicenseController(t)
	createTestLicense(t, r, "ListLic-"+uniqueTestID())
	resp := doReq(t, r, "GET", "/api/v1/licenses?page=1&pageSize=10", nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data, "total")
	assert.Contains(t, data, "licenses")
}

func TestAssetLicenseController_UpdateLicense(t *testing.T) {
	r, _, _, _ := setupLicenseController(t)
	id := createTestLicense(t, r, "UpdLic-"+uniqueTestID())
	newName := "Renamed-" + uniqueTestID()
	req := dto.UpdateLicenseRequest{Name: &newName}
	resp := doReq(t, r, "PUT", "/api/v1/licenses/"+strconv.Itoa(id), req, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, newName, data["name"])
}

func TestAssetLicenseController_AssignUsers_MissingUserIDs(t *testing.T) {
	r, _, _, _ := setupLicenseController(t)
	id := createTestLicense(t, r, "AssignLic-"+uniqueTestID())
	// UserIDs 为空（binding:"required"）→ BadRequest
	resp := doReq(t, r, "PUT", "/api/v1/licenses/"+strconv.Itoa(id)+"/assign", dto.LicenseAssignRequest{}, false)
	assert.Equal(t, common.BadRequestCode, resp.Code, "body=%s", mustString(resp))
}

func TestAssetLicenseController_GetLicenseStats(t *testing.T) {
	r, _, _, _ := setupLicenseController(t)
	createTestLicense(t, r, "StatLic-"+uniqueTestID())
	resp := doReq(t, r, "GET", "/api/v1/licenses/stats", nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data, "total")
}

func TestAssetLicenseController_DeleteLicense(t *testing.T) {
	r, _, _, _ := setupLicenseController(t)
	id := createTestLicense(t, r, "DelLic-"+uniqueTestID())
	resp := doReq(t, r, "DELETE", "/api/v1/licenses/"+strconv.Itoa(id), nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	// 删除后再查应 404
	resp2 := doReq(t, r, "GET", "/api/v1/licenses/"+strconv.Itoa(id), nil, false)
	assert.Equal(t, common.NotFoundCode, resp2.Code, "body=%s", mustString(resp2))
}
