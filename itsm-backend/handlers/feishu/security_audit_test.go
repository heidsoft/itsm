package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/enttest"
	domainCommon "itsm-backend/handlers/common"
	connectorDomain "itsm-backend/handlers/connector"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// 真实 handler 配真实 recorder，防止只有日志、没有持久审计的假闭环。
func TestFeishuSecurityRejectionsPersistAudit(t *testing.T) {
	for _, tc := range []struct{ name, path, action, method string }{
		{"signature", "/api/v1/feishu/webhook/inst-1", "feishu.webhook.verify_signature", "POST"},
		{"state", "/api/v1/feishu/oauth/callback/inst-1?code=private-code&state=forged-state", "feishu.oauth.state_rejected", "GET"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:feishu_audit_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
			defer client.Close()
			h, _ := newOAuthFixture(t)
			// 接口断言让尚未实现的注入点也能以行为测试 RED，而非编译失败。
			setter, ok := interface{}(h).(interface {
				SetAuditRecorder(connectorDomain.AuditRecorder)
			})
			require.True(t, ok, "飞书必须提供审计 recorder 注入点")
			setter.SetAuditRecorder(domainCommon.NewEntRepository(client))
			r := gin.New()
			r.POST("/api/v1/feishu/webhook/:instance_id", h.Webhook)
			r.GET("/api/v1/feishu/oauth/callback/:instance_id", h.OAuthCallback)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
			req.Header.Set("X-Lark-Signature", "private-signature")
			req.Header.Set("Authorization", "Bearer private-token")
			req.Header.Set("X-Api-Token", "private-token")
			r.ServeHTTP(w, req)
			require.Equal(t, 403, w.Code)
			var response struct {
				Code int `json:"code"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.Equal(t, common.ForbiddenCode, response.Code)
			rows, err := client.AuditLog.Query().Where(auditlog.ActionEQ(tc.action)).All(context.Background())
			require.NoError(t, err)
			require.Len(t, rows, 1, "安全拒绝必须真实落库")
			require.Equal(t, 1, rows[0].TenantID)
			require.Equal(t, "connector_inbound", rows[0].Resource)
			require.NotNil(t, rows[0].RequestBody)
			require.NotContains(t, *rows[0].RequestBody, "private-signature")
			require.NotContains(t, *rows[0].RequestBody, "private-token")
			require.NotContains(t, *rows[0].RequestBody, "private-code")
			require.NotContains(t, *rows[0].RequestBody, "forged-state")
			if tc.name == "signature" {
				require.Equal(t, "/feishu/webhook/1", rows[0].Path)
				var body struct {
					Headers map[string]string `json:"headers"`
				}
				require.NoError(t, json.Unmarshal([]byte(*rows[0].RequestBody), &body))
				for _, name := range []string{"X-Lark-Signature", "Authorization", "X-Api-Token"} {
					require.Equal(t, "***", body.Headers[name], name+" 必须脱敏且保留字段证据")
				}
			}
		})
	}
}
