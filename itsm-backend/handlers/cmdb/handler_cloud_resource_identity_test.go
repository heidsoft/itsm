package cmdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateCloudResourceRejectsCrossTenantAccountAtHandlerBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	createCalled := false
	repo := &mockRepository{
		getCloudAccountFn: func(context.Context, int, int) (*CloudAccount, error) {
			return &CloudAccount{ID: 7, TenantID: 2, Provider: "aliyun", AccountID: "other-tenant"}, nil
		},
		getCloudServiceFn: func(context.Context, int, int) (*CloudService, error) {
			return &CloudService{ID: 11, TenantID: 1, Provider: "aliyun", ServiceCode: "ecs", ResourceTypeCode: "instance"}, nil
		},
		createCloudResourceFn: func(context.Context, *CloudResource) (*CloudResource, error) {
			createCalled = true
			return nil, nil
		},
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Next()
	})
	router.POST("/api/v1/cmdb/cloud-resources", NewHandler(NewService(repo, zap.NewNop().Sugar())).CreateCloudResource)

	body := `{"cloudAccountId":7,"serviceId":11,"resourceId":"i-cross-tenant"}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/cmdb/cloud-resources", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"code":2003`)
	assert.False(t, createCalled)
}
