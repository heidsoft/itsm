package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initiateStatus 驱动真实 SSOInitiate handler，返回 HTTP 状态码与应用业务码。
func initiateStatus(t *testing.T, f *ssoFixture, body string) (int, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler := NewHandler(nil)
	handler.SetSSOService(f.service)
	router := gin.New()
	router.POST("/initiate", handler.SSOInitiate)

	req := httptest.NewRequest(http.MethodPost, "/initiate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var envelope struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	return rec.Code, envelope.Code
}

// 关键回归（预览能力收口）：发起阶段的「依赖不可用」必须归类为 503，而不是和真正的
// 认证失败一样兜底成 401。此前 SSOInitiate 对所有错误一律 AuthFailedWithErr(401)，
// 运维无法从状态码区分「IdP 不可达/没配好」与「凭证不对」。
//
// 这里走真实可达路径：提供方只配 issuer、端点不全，触发 OIDC discovery；
// 用本地 httptest 服务返回 404 模拟 IdP 发现端点不可用（不触外网）。
func TestSSOInitiate_DiscoveryUnavailableReturns503(t *testing.T) {
	f := newSSOFixture(t)
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer idp.Close()

	provider := f.createProvider(t, "discovery-down", true, "end_user")
	_, err := f.client.SSOProvider.UpdateOneID(provider.ID).
		SetIssuer(idp.URL).
		SetAuthorizationEndpoint("").
		SetTokenEndpoint("").
		SetJwksURI("").
		Save(f.ctx)
	require.NoError(t, err)

	httpStatus, code := initiateStatus(t, f, `{"provider":"discovery-down"}`)
	assert.Equal(t, http.StatusServiceUnavailable, httpStatus)
	assert.Equal(t, common.ServiceUnavailableCode, code)
}

func TestSSOInitiate_UnknownProviderReturns404(t *testing.T) {
	f := newSSOFixture(t)
	httpStatus, code := initiateStatus(t, f, `{"provider":"missing-idp"}`)
	assert.Equal(t, http.StatusNotFound, httpStatus)
	assert.Equal(t, common.NotFoundCode, code)
}

func TestSSOInitiate_DisabledProviderReturns404(t *testing.T) {
	f := newSSOFixture(t)
	provider := f.createProvider(t, "disabled-idp", true, "end_user")
	_, err := f.client.SSOProvider.UpdateOneID(provider.ID).SetEnabled(false).Save(f.ctx)
	require.NoError(t, err)

	httpStatus, code := initiateStatus(t, f, `{"provider":"disabled-idp"}`)
	assert.Equal(t, http.StatusNotFound, httpStatus)
	assert.Equal(t, common.NotFoundCode, code)
}

func TestSSOInitiate_AmbiguousCodeReturns400(t *testing.T) {
	f := newSSOFixture(t)
	tenantB, err := f.client.Tenant.Create().
		SetName("Tenant B").SetCode("tenant-b").SetDomain("b.example.com").SetStatus("active").Save(f.ctx)
	require.NoError(t, err)
	_, err = f.client.SSOProvider.Create().
		SetTenantID(tenantB.ID).SetCode("shared").SetName("Shared B").SetProtocol("oidc").
		SetClientID("client-b").SetIssuer("https://idp.example.com/b").
		SetAuthorizationEndpoint("https://idp.example.com/b/authorize").
		SetTokenEndpoint("https://idp.example.com/b/token").
		SetJwksURI("https://idp.example.com/b/jwks").
		SetRedirectURI("https://itsm.example.com/sso/callback").
		SetAutoProvision(true).SetDefaultRole("end_user").SetEnabled(true).Save(f.ctx)
	require.NoError(t, err)
	f.createProvider(t, "shared", true, "end_user")

	httpStatus, code := initiateStatus(t, f, `{"provider":"shared"}`)
	assert.Equal(t, http.StatusBadRequest, httpStatus)
	assert.Equal(t, common.BadRequestCode, code)
}
