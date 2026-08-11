// Package capability exposes the product capability control plane.
package capability

import (
	"os"
	"strings"
	"time"

	"itsm-backend/common"

	"github.com/gin-gonic/gin"
)

const acceptanceVersion = "2026.08-private-preview"

type Maturity string

const (
	MaturityGA       Maturity = "ga"
	MaturityPilot    Maturity = "pilot"
	MaturityDisabled Maturity = "disabled"
)

// Capability is the backend-owned contract used by every product surface.
type Capability struct {
	Key               string     `json:"key"`
	Maturity          Maturity   `json:"maturity"`
	BuildAvailable    bool       `json:"buildAvailable"`
	DeploymentReady   bool       `json:"deploymentReady"`
	TenantReady       bool       `json:"tenantReady"`
	AllowedActions    []string   `json:"allowedActions"`
	Dependencies      []string   `json:"dependencies"`
	DegradedReason    string     `json:"degradedReason,omitempty"`
	LastHealthCheckAt *time.Time `json:"lastHealthCheckAt,omitempty"`
	AcceptanceVersion string     `json:"acceptanceVersion"`
}

type definition struct {
	key          string
	maturity     Maturity
	built        bool
	dependencies []string
	readinessEnv string
	readOnly     bool
}

var registry = []definition{
	{key: "ticket", maturity: MaturityGA, built: true},
	{key: "incident", maturity: MaturityGA, built: true, dependencies: []string{"workflow", "sla", "notification"}},
	{key: "problem", maturity: MaturityPilot, built: true},
	{key: "change", maturity: MaturityGA, built: true, dependencies: []string{"workflow"}},
	{key: "serviceRequest", maturity: MaturityPilot, built: true, dependencies: []string{"workflow", "notification"}},
	{key: "cmdb", maturity: MaturityGA, built: true},
	{key: "cmdb.configurationItems", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.ciTypes", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.relationships", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.topology", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdbDiscovery", maturity: MaturityPilot, built: true, dependencies: []string{"cmdb"}, readinessEnv: "ITSM_CMDB_DISCOVERY_READY"},
	{key: "cmdbReconciliation", maturity: MaturityPilot, built: true, dependencies: []string{"cmdb", "cmdbDiscovery"}, readinessEnv: "ITSM_CMDB_DISCOVERY_READY"},
	{key: "workflow", maturity: MaturityGA, built: true},
	{key: "sla", maturity: MaturityGA, built: true},
	{key: "knowledge", maturity: MaturityPilot, built: true},
	{key: "ai", maturity: MaturityPilot, built: true, readinessEnv: "ITSM_AI_READY"},
	{key: "notification", maturity: MaturityPilot, built: true},
	{key: "connector.feishu", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_FEISHU_READY"},
	{key: "marketplace", maturity: MaturityPilot, built: true, readOnly: true},
	{key: "identity.oidc", maturity: MaturityDisabled, built: false},
	{key: "connector.wecom", maturity: MaturityDisabled, built: false},
	{key: "connector.dingtalk", maturity: MaturityDisabled, built: false},
}

// Handler returns tenant- and role-aware product capabilities. Runtime secrets
// are never returned; readiness is represented only as a boolean and reason.
func Handler(c *gin.Context) {
	now := time.Now().UTC()
	role := c.GetString("role")
	if role == "" {
		if value, ok := c.Get("user_role"); ok {
			role, _ = value.(string)
		}
	}
	tenantReady := c.GetInt("tenant_id") > 0
	admin := role == "admin" || role == "super_admin"

	items := make([]Capability, 0, len(registry))
	for _, item := range registry {
		dependencies := item.dependencies
		if dependencies == nil {
			dependencies = []string{}
		}
		deploymentReady := item.built
		degradedReason := ""
		if item.maturity == MaturityDisabled {
			deploymentReady = false
			degradedReason = "capability is not included in the current commercial release"
		} else if item.readinessEnv != "" && !envEnabled(item.readinessEnv) {
			deploymentReady = false
			degradedReason = "deployment dependency is not ready"
		}

		actions := []string{}
		if item.built && item.maturity != MaturityDisabled && tenantReady {
			actions = append(actions, "read")
			if admin && !item.readOnly && deploymentReady {
				actions = append(actions, "manage")
			}
		}

		items = append(items, Capability{
			Key: item.key, Maturity: item.maturity, BuildAvailable: item.built,
			DeploymentReady: deploymentReady, TenantReady: tenantReady,
			AllowedActions: actions, Dependencies: dependencies,
			DegradedReason: degradedReason, LastHealthCheckAt: &now,
			AcceptanceVersion: acceptanceVersion,
		})
	}

	common.Success(c, gin.H{"items": items, "acceptanceVersion": acceptanceVersion})
}

func envEnabled(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
