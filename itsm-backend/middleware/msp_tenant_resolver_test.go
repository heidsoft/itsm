package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newRouterWithTenant(tenantID int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(TenantContextKey, &TenantContext{TenantID: tenantID})
		c.Next()
	})
	return r
}

func TestResolveRequestTenantID(t *testing.T) {
	t.Run("missing tenant context returns ErrNoTenantContext", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		var got error
		r.GET("/x", func(c *gin.Context) {
			_, got = ResolveRequestTenantID(c)
			c.Status(http.StatusOK)
		})
		req := httptest.NewRequest("GET", "/x", nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
		assert.True(t, errors.Is(got, ErrNoTenantContext))
	})

	t.Run("regular user returns home tenant", func(t *testing.T) {
		r := newRouterWithTenant(100)
		var got int
		var err error
		r.GET("/x", func(c *gin.Context) {
			got, err = ResolveRequestTenantID(c)
			c.Status(http.StatusOK)
		})
		req := httptest.NewRequest("GET", "/x", nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
		assert.NoError(t, err)
		assert.Equal(t, 100, got)
	})

	t.Run("MSP user without customer lock returns home tenant", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(TenantContextKey, &TenantContext{TenantID: 500})
			c.Set(MSPContextKey, &MSPContext{
				IsMSP:            true,
				MSPUserID:        7,
				AllowedCustomers: []int{200, 300},
			})
			c.Next()
		})
		var got int
		var err error
		r.GET("/x", func(c *gin.Context) {
			got, err = ResolveRequestTenantID(c)
			c.Status(http.StatusOK)
		})
		req := httptest.NewRequest("GET", "/x", nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
		assert.NoError(t, err)
		assert.Equal(t, 500, got, "未锁定客户时应保留 MSP 自己的 home tenant")
	})

	t.Run("MSP user with valid customer lock returns customer tenant", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		cust := 200
		r.Use(func(c *gin.Context) {
			c.Set(TenantContextKey, &TenantContext{TenantID: 500})
			c.Set(MSPContextKey, &MSPContext{
				IsMSP:            true,
				MSPUserID:        7,
				CustomerTenantID: &cust,
				AllowedCustomers: []int{200, 300},
			})
			c.Next()
		})
		var got int
		var err error
		r.GET("/x", func(c *gin.Context) {
			got, err = ResolveRequestTenantID(c)
			c.Status(http.StatusOK)
		})
		req := httptest.NewRequest("GET", "/x", nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
		assert.NoError(t, err)
		assert.Equal(t, 200, got, "锁定到有效分配客户时应返回客户租户")
	})

	t.Run("MSP user with unauthorized customer lock is denied", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		denied := 999
		r.Use(func(c *gin.Context) {
			c.Set(TenantContextKey, &TenantContext{TenantID: 500})
			c.Set(MSPContextKey, &MSPContext{
				IsMSP:            true,
				MSPUserID:        7,
				CustomerTenantID: &denied,
				AllowedCustomers: []int{200, 300},
			})
			c.Next()
		})
		var got int
		var err error
		r.GET("/x", func(c *gin.Context) {
			got, err = ResolveRequestTenantID(c)
			c.Status(http.StatusOK)
		})
		req := httptest.NewRequest("GET", "/x", nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
		assert.Equal(t, 0, got)
		assert.True(t, errors.Is(err, ErrMSPCustomerDenied),
			"未被分配的客户必须返回 ErrMSPCustomerDenied,防止越权读")
	})

	t.Run("non-MSP user with stray customer lock still returns home tenant", func(t *testing.T) {
		// 防御性场景:如果有人在非 MSP 路由上误设置 MSPContext 且 IsMSP=false,
		// 解析器应忽略 CustomerTenantID,继续用 home tenant。
		gin.SetMode(gin.TestMode)
		r := gin.New()
		bad := 999
		r.Use(func(c *gin.Context) {
			c.Set(TenantContextKey, &TenantContext{TenantID: 100})
			c.Set(MSPContextKey, &MSPContext{
				IsMSP:            false,
				CustomerTenantID: &bad,
				AllowedCustomers: nil,
			})
			c.Next()
		})
		var got int
		var err error
		r.GET("/x", func(c *gin.Context) {
			got, err = ResolveRequestTenantID(c)
			c.Status(http.StatusOK)
		})
		req := httptest.NewRequest("GET", "/x", nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
		assert.NoError(t, err)
		assert.Equal(t, 100, got)
	})
}

func TestAbortIfTenantError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string
	}{
		{"no tenant context", ErrNoTenantContext, 401, "租户上下文缺失"},
		{"msp customer denied", ErrMSPCustomerDenied, 403, "MSP员工无权访问该客户租户"},
		{"unknown error", errors.New("boom"), 500, "租户解析失败"},
		{"nil error is no-op", nil, 200, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			var aborted bool
			r.GET("/x", func(c *gin.Context) {
				aborted = AbortIfTenantError(c, tc.err)
			})
			req := httptest.NewRequest("GET", "/x", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if tc.err == nil {
				assert.False(t, aborted)
				assert.Equal(t, 200, w.Code)
				return
			}
			assert.True(t, aborted)
			assert.Equal(t, tc.wantCode, w.Code)
			assert.Contains(t, w.Body.String(), tc.wantMsg)
		})
	}
}
