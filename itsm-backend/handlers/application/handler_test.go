package application

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockApplicationService struct{ mock.Mock }

func (m *mockApplicationService) CreateApplication(ctx context.Context, name, code, appType string, projectID, tenantID int) (*ent.Application, error) {
	a := m.Called(ctx, name, code, appType, projectID, tenantID)
	if app, ok := a.Get(0).(*ent.Application); ok {
		return app, a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockApplicationService) ListApplications(ctx context.Context, tenantID int) ([]*ent.Application, error) {
	a := m.Called(ctx, tenantID)
	if l, ok := a.Get(0).([]*ent.Application); ok {
		return l, a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockApplicationService) UpdateApplication(ctx context.Context, id int, name, code, appType *string, projectID *int, tenantID int) (*ent.Application, error) {
	a := m.Called(ctx, id, name, code, appType, projectID, tenantID)
	if app, ok := a.Get(0).(*ent.Application); ok {
		return app, a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockApplicationService) DeleteApplication(ctx context.Context, id int, tenantID int) error {
	return m.Called(ctx, id, tenantID).Error(0)
}

func (m *mockApplicationService) CreateMicroservice(ctx context.Context, name, code, language, framework string, appID, tenantID int) (*ent.Microservice, error) {
	a := m.Called(ctx, name, code, language, framework, appID, tenantID)
	if svc, ok := a.Get(0).(*ent.Microservice); ok {
		return svc, a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockApplicationService) ListMicroservices(ctx context.Context, tenantID int) ([]*ent.Microservice, error) {
	a := m.Called(ctx, tenantID)
	if l, ok := a.Get(0).([]*ent.Microservice); ok {
		return l, a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockApplicationService) UpdateMicroservice(ctx context.Context, id int, name, code, language, framework *string, appID *int, tenantID int) (*ent.Microservice, error) {
	a := m.Called(ctx, id, name, code, language, framework, appID, tenantID)
	if svc, ok := a.Get(0).(*ent.Microservice); ok {
		return svc, a.Error(1)
	}
	return nil, a.Error(1)
}

func (m *mockApplicationService) DeleteMicroservice(ctx context.Context, id int, tenantID int) error {
	return m.Called(ctx, id, tenantID).Error(0)
}

func appCtx(method, path, body string, tenantID int) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if body != "" {
		c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
	} else {
		c.Request = httptest.NewRequest(method, path, nil)
	}
	if tenantID > 0 {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
	}
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	return w, c
}

func TestCreateApplication_Success(t *testing.T) {
	m := &mockApplicationService{}
	h := NewHandler(m)
	m.On("CreateApplication", mock.Anything, "shop", "shop-app", "web", 0, 3).Return(&ent.Application{ID: 1, Name: "shop", Code: "shop-app"}, nil)

	w, c := appCtx(http.MethodPost, "/api/v1/applications", `{"name":"shop","code":"shop-app","type":"web"}`, 3)
	h.CreateApplication(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.SuccessCode, resp.Code)
	m.AssertExpectations(t)
}

func TestCreateApplication_ValidationFailure(t *testing.T) {
	m := &mockApplicationService{}
	h := NewHandler(m)

	// 缺 name（binding required）→ 400
	w, c := appCtx(http.MethodPost, "/api/v1/applications", `{"code":"shop-app"}`, 3)
	h.CreateApplication(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	m.AssertNotCalled(t, "CreateApplication")
}

func TestListApplications_Success(t *testing.T) {
	m := &mockApplicationService{}
	h := NewHandler(m)
	m.On("ListApplications", mock.Anything, 3).Return([]*ent.Application{{ID: 1, Code: "shop-app"}}, nil)

	w, c := appCtx(http.MethodGet, "/api/v1/applications", "", 3)
	h.ListApplications(c)

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}

func TestCreateMicroservice_Success(t *testing.T) {
	m := &mockApplicationService{}
	h := NewHandler(m)
	m.On("CreateMicroservice", mock.Anything, "order", "order-svc", "go", "gin", 5, 3).Return(&ent.Microservice{ID: 9, Code: "order-svc"}, nil)

	w, c := appCtx(http.MethodPost, "/api/v1/applications/microservices", `{"name":"order","code":"order-svc","language":"go","framework":"gin","applicationId":5}`, 3)
	h.CreateMicroservice(c)

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}
