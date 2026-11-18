package cmdb

// 回归：CI 关系的 strength / impact_level 是 ent Enum(critical/high/medium/low)。
// 词表外的值在 service 层未被拦下时，会撞 ent builder 的 EnumValidator，
// classifyError 归为未分类错误 → handler 兜底 500「创建CI关系失败」。
// 调用方（尤其 AI Agent 的 create_ci_relationship 工具）无法区分
// 「我传错了」与「后端挂了」，而后者会触发重试与告警风暴。
//
// 本文件在 HTTP 边界锁住契约：非法等级必须是 400（参数错误），不是 500。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/services"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type relationshipGradeFixture struct {
	router   *gin.Engine
	client   *ent.Client
	tenantID int
	sourceID int
	targetID int
}

func newRelationshipGradeFixture(t *testing.T) relationshipGradeFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:cmdb_rel_grade_contract?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("CMDB 关系等级契约").
		SetCode("cmdb-rel-grade").
		SetDomain("cmdb-rel-grade.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	mkCI := func(name string) *ent.ConfigurationItem {
		ci, err := client.ConfigurationItem.Create().
			SetName(name).
			SetCiTypeID(ciType.ID).
			SetCiType("Server").
			SetStatus("active").
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		return ci
	}
	source, target := mkCI("grade-source"), mkCI("grade-target")

	logger := zaptest.NewLogger(t).Sugar()
	relSvc := service.NewCIRelationshipService(client, logger)
	production := NewProductionService(logger, nil, nil, nil, relSvc, nil, nil, nil, nil)
	h := NewHandler(NewService(nil, production, logger))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Next()
	})
	r.POST("/api/v1/cmdb/relationships", h.CreateCIRelationship)
	r.PUT("/api/v1/cmdb/relationships/:id", h.UpdateCIRelationship)

	return relationshipGradeFixture{
		router: r, client: client, tenantID: tenant.ID, sourceID: source.ID, targetID: target.ID,
	}
}

func (f relationshipGradeFixture) post(t *testing.T, body map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cmdb/relationships", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)
	return w
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) (int, string) {
	t.Helper()
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), "响应不是标准包封: %s", w.Body.String())
	return body.Code, body.Message
}

// 旧工具 schema 声明的 "strong" 就是这条回归的真实入口。
func TestCreateCIRelationship_InvalidStrengthIsBadRequest(t *testing.T) {
	f := newRelationshipGradeFixture(t)

	w := f.post(t, map[string]interface{}{
		"sourceCiId":       f.sourceID,
		"targetCiId":       f.targetID,
		"relationshipType": string(dto.DependsOn),
		"strength":         "strong",
	})

	require.Equal(t, http.StatusBadRequest, w.Code, "非法 strength 必须是 400 而非 500；body=%s", w.Body.String())
	code, msg := decodeEnvelope(t, w)
	assert.Equal(t, common.ParamErrorCode, code)
	assert.NotEmpty(t, msg)

	// 关系不该被写进库：校验必须发生在落库前。
	count, err := f.client.CIRelationship.Query().Count(context.Background())
	require.NoError(t, err)
	assert.Zero(t, count, "非法等级的关系不应落库")
}

func TestCreateCIRelationship_InvalidImpactLevelIsBadRequest(t *testing.T) {
	f := newRelationshipGradeFixture(t)

	w := f.post(t, map[string]interface{}{
		"sourceCiId":       f.sourceID,
		"targetCiId":       f.targetID,
		"relationshipType": string(dto.DependsOn),
		"impactLevel":      "major",
	})

	require.Equal(t, http.StatusBadRequest, w.Code, "非法 impactLevel 必须是 400 而非 500；body=%s", w.Body.String())
	code, _ := decodeEnvelope(t, w)
	assert.Equal(t, common.ParamErrorCode, code)
}

// 合法等级必须 200，且落库的等级就是请求值——防止校验写反时静默降级成默认 medium。
func TestCreateCIRelationship_ValidGradeIsAccepted(t *testing.T) {
	f := newRelationshipGradeFixture(t)

	w := f.post(t, map[string]interface{}{
		"sourceCiId":       f.sourceID,
		"targetCiId":       f.targetID,
		"relationshipType": string(dto.DependsOn),
		"strength":         string(dto.StrengthCritical),
		"impactLevel":      string(dto.ImpactLow),
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Code int `json:"code"`
		Data struct {
			ID          int    `json:"id"`
			Strength    string `json:"strength"`
			ImpactLevel string `json:"impactLevel"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Code)
	assert.Equal(t, string(dto.StrengthCritical), body.Data.Strength, "strength 应原样落库，不得被默认值覆盖")
	assert.Equal(t, string(dto.ImpactLow), body.Data.ImpactLevel)

	rel, err := f.client.CIRelationship.Get(context.Background(), body.Data.ID)
	require.NoError(t, err)
	assert.Equal(t, "critical", rel.Strength.String())
	assert.Equal(t, "low", rel.ImpactLevel.String())
}

// 更新侧同一条契约：非法等级 400，且不落库。
func TestUpdateCIRelationship_InvalidGradeIsBadRequest(t *testing.T) {
	f := newRelationshipGradeFixture(t)

	w := f.post(t, map[string]interface{}{
		"sourceCiId":       f.sourceID,
		"targetCiId":       f.targetID,
		"relationshipType": string(dto.DependsOn),
		"strength":         string(dto.StrengthMedium),
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var created struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	raw, err := json.Marshal(map[string]interface{}{"strength": "weak"})
	require.NoError(t, err)
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/cmdb/relationships/"+strconv.Itoa(created.Data.ID), bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, "更新侧非法 strength 必须是 400；body=%s", w.Body.String())
	code, _ := decodeEnvelope(t, w)
	assert.Equal(t, common.ParamErrorCode, code)

	// 库里的原值不能被污染
	rel, err := f.client.CIRelationship.Get(context.Background(), created.Data.ID)
	require.NoError(t, err)
	assert.Equal(t, "medium", rel.Strength.String(), "非法更新不应改写已存等级")
}
