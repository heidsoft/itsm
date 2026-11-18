package cmdb

// 回归：CreateCI 的属性校验失败必须是 4xx（400/4000），不是 5001。
//
// 历史缺陷：CIAttributeValidator 用裸 fmt.Errorf("%w", sentinel) 包装校验错误，
// 该错误既不是 *AppError 也不是 *BusinessError，落到 common.classifyError 的兜底
// 分支 → 5001 / HTTP 500。前端把 500 当服务端故障反复重试，用户却只看到"创建失败"。
// 正确的对照是 UpdateLifecycleStatus 的必填门禁：用 common.NewValidationError 包成
// *AppError，classifyError 命中 HTTP 400 → 业务码 4000。
//
// 本文件在 HTTP 边界锁住状态码与业务码；服务层 sentinel 身份由
// internal/services/configuration_item_lifecycle_test.go 与 internal/layering 锁定。
// 不要在 common.classifyError 里加 errors.Is(sentinel) 分支——common 不能依赖 dto，
// 否则 common ↔ dto 循环依赖（dto 已 import common）。修复点必须在服务层包装处。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
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

type createCICodeFixture struct {
	router   *gin.Engine
	client   *ent.Client
	tenantID int
	ciTypeID int
}

func newCreateCICodeFixture(t *testing.T) createCICodeFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:create_ci_code?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("CI 创建校验码").SetCode("create-ci-code").
		SetDomain("create-ci-code.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	_, err = client.CIAttributeDefinition.Create().
		SetName("ip_address").SetDisplayName("IP 地址").SetType("string").
		SetRequired(true).SetCiTypeID(ciType.ID).SetTenantID(tenant.ID).SetIsActive(true).Save(ctx)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	historySvc := service.NewCIHistoryService(client, logger)
	tagSvc := service.NewCITagService(client, logger)
	ciSvc := service.NewConfigurationItemService(client, logger, historySvc, tagSvc)
	production := NewProductionService(logger, nil, nil, ciSvc, nil, historySvc, tagSvc, nil, nil)
	h := NewHandler(NewService(nil, production, logger))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Set("user_id", 1)
		c.Next()
	})
	r.POST("/api/v1/cmdb/cis", h.CreateCI)

	return createCICodeFixture{router: r, client: client, tenantID: tenant.ID, ciTypeID: ciType.ID}
}

func (f createCICodeFixture) postCreate(t *testing.T, attrs map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]interface{}{
		"name": "missing-ip-ci", "ciTypeId": f.ciTypeID,
		"status": common.CIStatusActive, "attributes": attrs,
	})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cmdb/cis", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)
	return w
}

// 缺必填属性：HTTP 400 + 业务码 4000（不是 500/5001），且文案点名缺失字段。
func TestCreateCI_MissingRequiredAttributeIsValidationError(t *testing.T) {
	f := newCreateCICodeFixture(t)

	w := f.postCreate(t, map[string]interface{}{})

	require.Equal(t, http.StatusBadRequest, w.Code,
		"属性校验失败应是 400 而非 500；body=%s", w.Body.String())
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.BadRequestCode, resp.Code, "业务码必须是 4000，不是 5001 内部错误")
	assert.NotEqual(t, common.InternalErrorCode, resp.Code)
	assert.Contains(t, resp.Message, "IP 地址", "文案要点名缺失字段的展示名")

	count, err := f.client.ConfigurationItem.Query().Count(context.Background())
	require.NoError(t, err)
	assert.Zero(t, count, "校验失败不得留下半条 CI")
}

// 补齐必填属性后必须创建成功——防止分类修复把正常创建也拒了。
func TestCreateCI_WithRequiredAttributeSucceeds(t *testing.T) {
	f := newCreateCICodeFixture(t)

	w := f.postCreate(t, map[string]interface{}{"ip_address": "10.0.0.1"})

	require.Equal(t, http.StatusOK, w.Code, "补齐属性后应创建成功；body=%s", w.Body.String())
	count, err := f.client.ConfigurationItem.Query().Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
