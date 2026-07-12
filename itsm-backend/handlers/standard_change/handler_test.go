package standard_change

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
)

// newTestClient spins up an isolated in-memory SQLite-backed ent client for each test.
func newTestClient(t *testing.T) *ent.Client {
	dbName := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dbName)
	t.Cleanup(func() { client.Close() })
	return client
}

// setupTestRouter wires the handler into a gin engine with a mock auth middleware
// that injects the given user/tenant context (mirrors the real tenant middleware).
func setupTestRouter(t *testing.T, client *ent.Client, userID, tenantID int) (*gin.Engine, *Handler) {
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()
	h := NewHandler(client, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("tenant_id", tenantID)
		c.Next()
	})
	h.RegisterRoutes(r.Group("/api/v1"))
	return r, h
}

// createTemplate inserts a StandardChange template directly via ent, so tests can
// exercise the read/update/delete/instantiate paths without going through the HTTP layer.
func createTemplate(t *testing.T, client *ent.Client, tenantID, userID int, mutate func(*ent.StandardChangeCreate)) *ent.StandardChange {
	builder := client.StandardChange.Create().
		SetTitle("默认模板").
		SetImplementationPlan("实施计划步骤").
		SetRollbackPlan("回滚计划步骤").
		SetCategory("general").
		SetRiskLevel("low").
		SetImpactScope("low").
		SetCreatedBy(userID).
		SetTenantID(tenantID)
	if mutate != nil {
		mutate(builder)
	}
	sc, err := builder.Save(context.Background())
	require.NoError(t, err)
	return sc
}

// decodeResponse unmarshals the unified response envelope into a Response + its Data map.
func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) (common.Response, map[string]interface{}) {
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, _ := resp.Data.(map[string]interface{})
	return resp, data
}

func doRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		require.NoError(nil, json.NewEncoder(&buf).Encode(body))
	}
	req, _ := http.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ===================== toResponse conversion =====================

func TestToResponse_Nil(t *testing.T) {
	assert.Nil(t, toResponse(nil))
}

func TestToResponse_MapsAllFields(t *testing.T) {
	sc := createTemplate(t, newTestClient(t), 1, 7, func(b *ent.StandardChangeCreate) {
		b.SetDescription("desc").
			SetJustification("reason").
			SetExpectedDuration(45).
			SetApprovalRequired(true).
			SetAffectedCis([]string{"web", "db"}).
			SetPrerequisites([]string{"backup"}).
			SetRemarks("note").
			SetIsActive(false)
	})

	resp := toResponse(sc)
	require.NotNil(t, resp)
	assert.Equal(t, sc.ID, resp.ID)
	assert.Equal(t, "默认模板", resp.Title)
	assert.Equal(t, "desc", resp.Description)
	assert.Equal(t, "reason", resp.Justification)
	assert.Equal(t, "general", resp.Category)
	assert.Equal(t, "low", resp.RiskLevel)
	assert.Equal(t, 45, resp.ExpectedDuration)
	assert.True(t, resp.ApprovalRequired)
	assert.Equal(t, []string{"web", "db"}, resp.AffectedCis)
	assert.Equal(t, []string{"backup"}, resp.Prerequisites)
	assert.Equal(t, 7, resp.CreatedBy)
	assert.Equal(t, 1, resp.TenantID)
	assert.False(t, resp.IsActive)
}

// ===================== CreateStandardChange =====================

func TestCreateStandardChange_SuccessWithDefaults(t *testing.T) {
	r, _ := setupTestRouter(t, newTestClient(t), 1, 1)

	req := dto.CreateStandardChangeRequest{
		Title:              "日常发布模板",
		ImplementationPlan: "步骤1；步骤2",
		RollbackPlan:       "回滚步骤",
	}
	w := doRequest(r, "POST", "/api/v1/standard-changes", req)

	resp, data := decodeResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, resp.Code)

	// Defaults applied by the handler
	assert.Equal(t, "general", data["category"])
	assert.Equal(t, "low", data["riskLevel"])
	assert.Equal(t, "low", data["impactScope"])
	assert.Equal(t, float64(1), data["createdBy"])
	assert.Equal(t, float64(1), data["tenantId"])
	assert.Equal(t, true, data["isActive"])
	// Omitted expected_duration falls back to the schema default (30 minutes)
	// instead of being persisted as the JSON zero value 0.
	assert.Equal(t, float64(30), data["expectedDuration"])
}

func TestCreateStandardChange_SuccessWithExplicitValues(t *testing.T) {
	r, _ := setupTestRouter(t, newTestClient(t), 3, 2)

	req := dto.CreateStandardChangeRequest{
		Title:              "网络变更模板",
		Description:        "描述",
		ImplementationPlan: "计划",
		RollbackPlan:       "回滚",
		Category:           "network",
		RiskLevel:          "high",
		ImpactScope:        "medium",
		ExpectedDuration:   120,
		ApprovalRequired:   true,
		AffectedCis:        []string{"switch"},
		Prerequisites:      []string{"审批单"},
	}
	w := doRequest(r, "POST", "/api/v1/standard-changes", req)

	resp, data := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)
	assert.Equal(t, "network", data["category"])
	assert.Equal(t, "high", data["riskLevel"])
	assert.Equal(t, "medium", data["impactScope"])
	assert.Equal(t, float64(120), data["expectedDuration"])
	assert.Equal(t, true, data["approvalRequired"])
	assert.Equal(t, []interface{}{"switch"}, data["affectedCis"])
	assert.Equal(t, float64(3), data["createdBy"])
	assert.Equal(t, float64(2), data["tenantId"])
}

// TestCreateStandardChange_ExplicitZeroDurationDefaultsTo30 locks the fix for the
// bug where an omitted/zero expected_duration was persisted as 0 instead of the
// schema default (30). A client sending 0 explicitly must also get the default.
func TestCreateStandardChange_ExplicitZeroDurationDefaultsTo30(t *testing.T) {
	r, _ := setupTestRouter(t, newTestClient(t), 1, 1)

	req := dto.CreateStandardChangeRequest{
		Title:              "零工期模板",
		ImplementationPlan: "步骤1",
		RollbackPlan:       "回滚步骤",
		ExpectedDuration:   0,
	}
	w := doRequest(r, "POST", "/api/v1/standard-changes", req)

	resp, data := decodeResponse(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, resp.Code)
	assert.Equal(t, float64(30), data["expectedDuration"],
		"explicit 0 must fall back to the schema default of 30, not persist 0")
}

func TestCreateStandardChange_MissingRequiredField(t *testing.T) {
	r, _ := setupTestRouter(t, newTestClient(t), 1, 1)

	// Missing ImplementationPlan (binding:"required")
	req := map[string]interface{}{
		"title":        "缺字段模板",
		"rollbackPlan": "回滚",
	}
	w := doRequest(r, "POST", "/api/v1/standard-changes", req)

	resp, _ := decodeResponse(t, w)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, common.ParamErrorCode, resp.Code)
}

// ===================== ListStandardChanges =====================

func TestListStandardChanges_Basic(t *testing.T) {
	client := newTestClient(t)
	createTemplate(t, client, 1, 1, nil)
	createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) {
		b.SetTitle("第二个模板").SetCategory("database")
	})
	r, _ := setupTestRouter(t, client, 1, 1)

	w := doRequest(r, "GET", "/api/v1/standard-changes", nil)
	resp, data := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)
	assert.Equal(t, float64(2), data["total"])
	templates := data["templates"].([]interface{})
	assert.Len(t, templates, 2)
}

func TestListStandardChanges_Pagination(t *testing.T) {
	client := newTestClient(t)
	for i := 0; i < 5; i++ {
		createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) {
			b.SetTitle("模板" + strconv.Itoa(i))
		})
	}
	r, _ := setupTestRouter(t, client, 1, 1)

	w := doRequest(r, "GET", "/api/v1/standard-changes?page=2&pageSize=2", nil)
	resp, data := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)
	assert.Equal(t, float64(5), data["total"])
	templates := data["templates"].([]interface{})
	assert.Len(t, templates, 2)
	assert.Equal(t, float64(2), data["page"])
	assert.Equal(t, float64(2), data["pageSize"])
}

func TestListStandardChanges_FilterByCategory(t *testing.T) {
	client := newTestClient(t)
	createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) { b.SetCategory("network") })
	createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) { b.SetCategory("database") })
	createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) { b.SetCategory("database") })
	r, _ := setupTestRouter(t, client, 1, 1)

	w := doRequest(r, "GET", "/api/v1/standard-changes?category=database", nil)
	resp, data := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)
	assert.Equal(t, float64(2), data["total"])
}

func TestListStandardChanges_Search(t *testing.T) {
	client := newTestClient(t)
	createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) { b.SetTitle("数据库扩容") })
	createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) { b.SetTitle("网络割接") })
	r, _ := setupTestRouter(t, client, 1, 1)

	w := doRequest(r, "GET", "/api/v1/standard-changes?search=数据库", nil)
	resp, data := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)
	assert.Equal(t, float64(1), data["total"])
}

func TestListStandardChanges_ActiveOnly(t *testing.T) {
	client := newTestClient(t)
	createTemplate(t, client, 1, 1, nil)
	createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) {
		b.SetTitle("已停用").SetIsActive(false)
	})
	r, _ := setupTestRouter(t, client, 1, 1)

	w := doRequest(r, "GET", "/api/v1/standard-changes?active_only=true", nil)
	resp, data := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)
	assert.Equal(t, float64(1), data["total"])
}

// ===================== GetStandardChange =====================

func TestGetStandardChange_ByID(t *testing.T) {
	client := newTestClient(t)
	sc := createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) {
		b.SetTitle("唯一模板").SetCategory("network")
	})
	r, _ := setupTestRouter(t, client, 1, 1)

	w := doRequest(r, "GET", "/api/v1/standard-changes/"+strconv.Itoa(sc.ID), nil)
	resp, data := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)
	assert.Equal(t, "唯一模板", data["title"])
	assert.Equal(t, "network", data["category"])
}

func TestGetStandardChange_InvalidID(t *testing.T) {
	r, _ := setupTestRouter(t, newTestClient(t), 1, 1)
	for _, id := range []string{"abc", "0", "-5"} {
		w := doRequest(r, "GET", "/api/v1/standard-changes/"+id, nil)
		resp, _ := decodeResponse(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, common.ParamErrorCode, resp.Code)
	}
}

func TestGetStandardChange_NotFound(t *testing.T) {
	r, _ := setupTestRouter(t, newTestClient(t), 1, 1)
	w := doRequest(r, "GET", "/api/v1/standard-changes/99999", nil)
	resp, _ := decodeResponse(t, w)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, resp.Code)
}

// ===================== UpdateStandardChange =====================

func TestUpdateStandardChange_PartialUpdate(t *testing.T) {
	client := newTestClient(t)
	sc := createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) {
		b.SetCategory("general").SetRiskLevel("low")
	})
	r, _ := setupTestRouter(t, client, 1, 1)

	req := dto.UpdateStandardChangeRequest{
		Title:     strPtr("改名后的模板"),
		RiskLevel: strPtr("high"),
	}
	w := doRequest(r, "PUT", "/api/v1/standard-changes/"+strconv.Itoa(sc.ID), req)
	resp, data := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)
	assert.Equal(t, "改名后的模板", data["title"])
	assert.Equal(t, "high", data["riskLevel"])
	// Untouched fields are preserved
	assert.Equal(t, "general", data["category"])
}

func TestUpdateStandardChange_NotFound(t *testing.T) {
	r, _ := setupTestRouter(t, newTestClient(t), 1, 1)
	req := dto.UpdateStandardChangeRequest{Title: strPtr("x")}
	w := doRequest(r, "PUT", "/api/v1/standard-changes/99999", req)
	resp, _ := decodeResponse(t, w)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, resp.Code)
}

func TestUpdateStandardChange_InvalidID(t *testing.T) {
	r, _ := setupTestRouter(t, newTestClient(t), 1, 1)
	req := dto.UpdateStandardChangeRequest{Title: strPtr("x")}
	w := doRequest(r, "PUT", "/api/v1/standard-changes/abc", req)
	resp, _ := decodeResponse(t, w)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, common.ParamErrorCode, resp.Code)
}

// ===================== DeleteStandardChange (soft delete) =====================

func TestDeleteStandardChange_SoftDelete(t *testing.T) {
	client := newTestClient(t)
	sc := createTemplate(t, client, 1, 1, nil)
	r, _ := setupTestRouter(t, client, 1, 1)

	w := doRequest(r, "DELETE", "/api/v1/standard-changes/"+strconv.Itoa(sc.ID), nil)
	resp, _ := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)

	// Soft delete must only deactivate, not physically remove.
	refreshed, err := client.StandardChange.Get(context.Background(), sc.ID)
	require.NoError(t, err)
	assert.False(t, refreshed.IsActive, "deleted template should be deactivated, not removed")
}

func TestDeleteStandardChange_NotFound(t *testing.T) {
	r, _ := setupTestRouter(t, newTestClient(t), 1, 1)
	w := doRequest(r, "DELETE", "/api/v1/standard-changes/99999", nil)
	resp, _ := decodeResponse(t, w)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, resp.Code)
}

func TestDeleteStandardChange_InvalidID(t *testing.T) {
	r, _ := setupTestRouter(t, newTestClient(t), 1, 1)
	w := doRequest(r, "DELETE", "/api/v1/standard-changes/0", nil)
	resp, _ := decodeResponse(t, w)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, common.ParamErrorCode, resp.Code)
}

// ===================== GetCategories =====================

func TestGetCategories_Distinct(t *testing.T) {
	client := newTestClient(t)
	createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) { b.SetCategory("network") })
	createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) { b.SetCategory("network") })
	createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) { b.SetCategory("database") })
	createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) { b.SetCategory("general") })
	r, _ := setupTestRouter(t, client, 1, 1)

	w := doRequest(r, "GET", "/api/v1/standard-changes/categories", nil)
	resp, data := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)
	cats := data["categories"].([]interface{})
	assert.ElementsMatch(t, []interface{}{"network", "database", "general"}, cats)
}

// ===================== InstantiateStandardChange =====================

func TestInstantiateStandardChange_Defaults(t *testing.T) {
	client := newTestClient(t)
	tmpl := createTemplate(t, client, 1, 5, func(b *ent.StandardChangeCreate) {
		b.SetTitle("发布模板").SetRiskLevel("medium").SetImpactScope("high").
			SetAffectedCis([]string{"web"}).
			SetImplementationPlan("实施计划步骤").SetRollbackPlan("回滚计划步骤")
	})
	r, _ := setupTestRouter(t, client, 9, 1)

	w := doRequest(r, "POST", "/api/v1/standard-changes/"+strconv.Itoa(tmpl.ID)+"/instantiate", dto.InstantiateStandardChangeRequest{})
	resp, data := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)
	require.Contains(t, data, "change_id")
	changeID := int(data["change_id"].(float64))

	// Verify the Change was created from the template with the expected mapping.
	created, err := client.Change.Query().
		Where(change.ID(changeID), change.TenantID(1)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "发布模板", created.Title)
	assert.Equal(t, "standard", created.Type)
	assert.Equal(t, "draft", created.Status)
	assert.Equal(t, "medium", created.Priority)
	assert.Equal(t, "high", created.ImpactScope)
	assert.Equal(t, "medium", created.RiskLevel)
	assert.Equal(t, 9, created.CreatedBy)
	assert.Equal(t, []string{"web"}, created.AffectedCis)
	assert.Equal(t, "实施计划步骤", created.ImplementationPlan)
	assert.Equal(t, "回滚计划步骤", created.RollbackPlan)
}

func TestInstantiateStandardChange_Overrides(t *testing.T) {
	client := newTestClient(t)
	tmpl := createTemplate(t, client, 1, 5, func(b *ent.StandardChangeCreate) {
		b.SetTitle("原模板").SetAffectedCis([]string{"old"})
	})
	r, _ := setupTestRouter(t, client, 9, 1)

	req := dto.InstantiateStandardChangeRequest{
		Title:       "覆盖标题",
		AffectedCis: []string{"new1", "new2"},
	}
	w := doRequest(r, "POST", "/api/v1/standard-changes/"+strconv.Itoa(tmpl.ID)+"/instantiate", req)
	resp, data := decodeResponse(t, w)
	assert.Equal(t, common.SuccessCode, resp.Code)
	require.Contains(t, data, "change_id")
	changeID := int(data["change_id"].(float64))

	created, err := client.Change.Query().
		Where(change.ID(changeID), change.TenantID(1)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "覆盖标题", created.Title)
	assert.Equal(t, []string{"new1", "new2"}, created.AffectedCis)
	// Untouched fields fall back to the template's values (implementation plan is unchanged).
	assert.Equal(t, "实施计划步骤", created.ImplementationPlan)
}

func TestInstantiateStandardChange_InactiveTemplateNotFound(t *testing.T) {
	client := newTestClient(t)
	tmpl := createTemplate(t, client, 1, 5, func(b *ent.StandardChangeCreate) {
		b.SetIsActive(false)
	})
	r, _ := setupTestRouter(t, client, 9, 1)

	w := doRequest(r, "POST", "/api/v1/standard-changes/"+strconv.Itoa(tmpl.ID)+"/instantiate", dto.InstantiateStandardChangeRequest{})
	resp, _ := decodeResponse(t, w)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, resp.Code)
}

func TestInstantiateStandardChange_NotFound(t *testing.T) {
	r, _ := setupTestRouter(t, newTestClient(t), 9, 1)
	w := doRequest(r, "POST", "/api/v1/standard-changes/99999/instantiate", dto.InstantiateStandardChangeRequest{})
	resp, _ := decodeResponse(t, w)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, resp.Code)
}

// ===================== Tenant isolation (multi-tenant core logic) =====================

func TestStandardChange_TenantIsolation(t *testing.T) {
	client := newTestClient(t)
	// Template belongs to tenant 1
	sc := createTemplate(t, client, 1, 1, func(b *ent.StandardChangeCreate) {
		b.SetTitle("租户1模板")
	})

	// Router scoped to tenant 2 must not see tenant 1's template
	r2, _ := setupTestRouter(t, client, 1, 2)

	w := doRequest(r2, "GET", "/api/v1/standard-changes/"+strconv.Itoa(sc.ID), nil)
	resp, _ := decodeResponse(t, w)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, resp.Code)

	// And it must not appear in tenant 2's list
	wList := doRequest(r2, "GET", "/api/v1/standard-changes", nil)
	_, data := decodeResponse(t, wList)
	assert.Equal(t, float64(0), data["total"])
}

// strPtr is a small helper for building optional-string update requests.
func strPtr(s string) *string { return &s }
