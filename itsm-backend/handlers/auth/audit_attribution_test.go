package auth

import (
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent/auditlog"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 成功夹具必须真的发信或消费令牌，否则摘掉成功审计也会假绿。
func TestSuccessfulPasswordActionsAuditAttribution(t *testing.T) {
	for _, action := range []string{"forgot_password", "reset_password"} {
		t.Run(action, func(t *testing.T) {
			f := newAuthFixture(t)
			defer f.client.Close()
			// 两个租户并存，目标用户归属第二租户，不能默认为租户 1。
			f.user = f.client.User.UpdateOneID(f.user.ID).SetTenantID(f.tenant2.ID).SaveX(f.ctx)
			email := &fakeEmailService{}
			f.service.SetEmailService(email)
			h := NewHandler(f.service)
			r := gin.New()
			r.POST(pathForgotPassword, h.ForgotPassword)
			r.POST(pathResetPassword, h.ResetPassword)
			path, body := pathForgotPassword, `{"email":"alice@example.com"}`
			if action == "reset_password" {
				// 邮箱快照命中另一租户的干扰用户：按 email 查库会错误归属。
				f.client.User.Create().SetUsername("decoy").SetEmail("snapshot@example.com").SetName("Decoy").SetPasswordHash("unused").SetRole("end_user").SetTenantID(f.tenant.ID).SaveX(f.ctx)
				f.client.PasswordResetToken.Create().SetUserID(f.user.ID).SetEmail("snapshot@example.com").SetToken("valid-once").SetExpiresAt(time.Now().Add(time.Hour)).SaveX(f.ctx)
				path, body = pathResetPassword, `{"email":"snapshot@example.com","token":"valid-once","password":"NewPassword2026!","passwordConfirm":"NewPassword2026!"}`
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			require.Equal(t, 200, w.Code, w.Body.String())
			var response struct {
				Code int                    `json:"code"`
				Data map[string]interface{} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.Zero(t, response.Code)
			require.Len(t, response.Data, 1, "公开响应只能含 message，不能泄露归属")
			logs, err := f.client.AuditLog.Query().Where(auditlog.ActionEQ(action)).All(f.ctx)
			require.NoError(t, err)
			require.Len(t, logs, 1, "成功分支必须留一条审计")
			entry := logs[0]
			require.Equal(t, 200, entry.StatusCode)
			require.Equal(t, f.user.TenantID, entry.TenantID)
			require.Equal(t, f.user.ID, entry.UserID, "可信用户归属不得被 helper 清零")
			require.NotNil(t, entry.RequestBody)
			var audit map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(*entry.RequestBody), &audit))
			require.Equal(t, strconv.Itoa(f.user.ID), audit["subject"])
			require.NotContains(t, *entry.RequestBody, "valid-once")
			require.NotContains(t, *entry.RequestBody, "NewPassword2026!")
			require.NotContains(t, *entry.RequestBody, "@example.com")
			if action == "forgot_password" {
				require.Equal(t, 1, email.sent, "必须真的完成发信")
				missing := httptest.NewRecorder()
				req := httptest.NewRequest("POST", path, strings.NewReader(`{"email":"absent@example.com"}`))
				req.Header.Set("Content-Type", "application/json")
				r.ServeHTTP(missing, req)
				require.Equal(t, w.Code, missing.Code)
				require.JSONEq(t, w.Body.String(), missing.Body.String(), "存在与不存在的 HTTP 响应必须相同")
				last := f.client.AuditLog.Query().Order(auditlog.ByID()).AllX(f.ctx)
				require.Len(t, last, 2)
				require.Zero(t, last[1].TenantID)
				require.Zero(t, last[1].UserID)
				var missingAudit map[string]interface{}
				require.NoError(t, json.Unmarshal([]byte(*last[1].RequestBody), &missingAudit))
				require.Equal(t, "", missingAudit["subject"])
				require.Equal(t, 1, email.sent, "不存在的用户不得发信")
			}
		})
	}
}

// 注册已有公开 ID，直接复用可信创建结果，避免另加一份可能漂移的用户 ID。
func TestSuccessfulRegistrationAuditUser(t *testing.T) {
	f := newAuthFixture(t)
	defer f.client.Close()
	f.client.Tenant.UpdateOneID(f.tenant2.ID).SetStatus("suspended").SaveX(f.ctx)
	h := NewHandler(f.service)
	r := gin.New()
	r.POST(pathRegister, h.Register)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", pathRegister, strings.NewReader(`{"username":"newactor","email":"newactor@example.com","password":"Password2026!"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code, w.Body.String())
	var response struct {
		Code int `json:"code"`
		Data struct {
			ID int `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.NotZero(t, response.Data.ID)
	entry := f.client.AuditLog.Query().Where(auditlog.ActionEQ("register")).OnlyX(f.ctx)
	require.Equal(t, response.Data.ID, entry.UserID)
	require.Equal(t, f.tenant.ID, entry.TenantID)
	require.NotContains(t, w.Body.String(), "tenantId")
	require.NotContains(t, w.Body.String(), "TenantID")
}
