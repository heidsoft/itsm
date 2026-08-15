package router

import (
	"strings"

	"itsm-backend/common"
	"itsm-backend/ent/tenant"
	bootstrapauth "itsm-backend/pkg/bootstrap"

	"github.com/gin-gonic/gin"
)

type createBootstrapAdminRequest struct {
	Token      string `json:"token" binding:"required"`
	Password   string `json:"password" binding:"required"`
	TenantCode string `json:"tenantCode"`
}

func registerBootstrapRoutes(public *gin.RouterGroup, config *RouterConfig) {
	if config == nil || config.Client == nil {
		return
	}
	manager := bootstrapauth.NewBootstrapTokenManager(config.Client, config.Logger)
	group := public.Group("/bootstrap")

	group.GET("/status", func(c *gin.Context) {
		tenantCode := strings.TrimSpace(c.DefaultQuery("tenantCode", "default"))
		rootTenant, err := config.Client.Tenant.Query().Where(tenant.CodeEQ(tenantCode)).Only(c.Request.Context())
		if err != nil {
			// Fail without revealing whether an arbitrary tenant exists.
			common.Fail(c, common.BadRequestCode, "bootstrap status unavailable")
			return
		}
		required, tokenAvailable, expiresAt, err := manager.Status(c.Request.Context(), rootTenant.ID)
		if err != nil {
			common.Fail(c, common.InternalErrorCode, "bootstrap status unavailable")
			return
		}
		state := "completed"
		if required && tokenAvailable {
			state = "ready"
		} else if required {
			state = "token_required"
		}
		common.Success(c, gin.H{
			"state":          state,
			"required":       required,
			"tokenAvailable": tokenAvailable,
			"expiresAt":      expiresAt,
		})
	})

	group.POST("/create-admin", func(c *gin.Context) {
		var req createBootstrapAdminRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			common.Fail(c, common.ParamErrorCode, "token and password are required")
			return
		}
		tenantCode := strings.TrimSpace(req.TenantCode)
		if tenantCode == "" {
			tenantCode = "default"
		}
		rootTenant, err := config.Client.Tenant.Query().Where(tenant.CodeEQ(tenantCode)).Only(c.Request.Context())
		if err != nil {
			common.Fail(c, common.AuthFailedCode, "invalid or unavailable bootstrap token")
			return
		}
		userID, err := manager.ConsumeToken(c.Request.Context(), strings.TrimSpace(req.Token), rootTenant.ID, req.Password)
		if err != nil {
			common.Fail(c, common.AuthFailedCode, "invalid or unavailable bootstrap token")
			return
		}
		common.Success(c, gin.H{"userId": userID, "state": "completed"})
	})
}
