package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	ticketAttachmentHandler "itsm-backend/handlers/ticket_attachment"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-02 边缘功能收口 E1-2）：工单附件的下载与预览 handler
// （handlers/ticket_attachment/handler.go DownloadAttachment/PreviewAttachment）此前
// 从未在 router/ticket_routes.go 注册，前端两个入口点了就 404；同时 DTO 的 fileUrl
// 透传了库里指向从未注册路由的存量死链。必须从真实路由入口证明：
//   - GET /api/v1/tickets/:id/attachments/:attachment_id 返回真实字节与 attachment 头；
//   - GET .../preview 返回 inline 头；
//   - 列表返回的 fileUrl 就是可点击下载的真实路由（自证一致性）；
//   - 跨租户失败是 404（不确认对方资源存在），同租户非相关方是 403；
//   - 磁盘读取失败仍是 500，不与权限语义混淆。

// attachmentContent 是写入临时盘的附件字节，长度断言由它派生。
const attachmentContent = "%PDF-1.4 attachment bytes"

type attachmentRouteFixture struct {
	engine      *gin.Engine
	secret      string
	ticketA     *ent.Ticket
	attachmentA *ent.TicketAttachment // 磁盘上有真实文件
	attachmentB *ent.TicketAttachment // 记录在，磁盘文件已丢失
	userA       *ent.User             // tenant A 工单申请人，super_admin
	stranger    *ent.User             // tenant B super_admin，用于跨租户探测
	outsider    *ent.User             // tenant A 同租户、有 ticket:read、但非工单相关方
}

func setupAttachmentRouteTest(t *testing.T) attachmentRouteFixture {
	t.Helper()

	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:router_ticket_attach_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Attach A").SetCode("attach-a").SetDomain("attach-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Attach B").SetCode("attach-b").SetDomain("attach-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("attach-a-owner").SetEmail("attach-a@example.com").SetName("A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("attach-b-stranger").SetEmail("attach-b@example.com").SetName("B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)
	// outsider 有 ticket:read，能穿过 RequirePermission 进到 handler，
	// 但不是工单申请人/处理人，必须停在服务层的相关方检查上。
	outsider := client.User.Create().SetUsername("attach-a-outsider").SetEmail("outsider@example.com").SetName("C").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)

	ticketA := client.Ticket.Create().
		SetTicketNumber("TKT-ATTACH-A-001").
		SetTitle("附件下载回归").
		SetDescription("验证附件下载/预览路由真实接线").
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(userA.ID).
		SetTenantID(tenantA.ID).
		SaveX(ctx)

	// 附件落盘用独立临时目录，避免污染仓库 uploads/。
	dir := t.TempDir()
	filePath := filepath.Join(dir, "report.pdf")
	require.NoError(t, os.WriteFile(filePath, []byte(attachmentContent), 0o600))

	newAttachment := func(name, path string) *ent.TicketAttachment {
		return client.TicketAttachment.Create().
			SetTicketID(ticketA.ID).
			SetFileName(name).
			SetFilePath(path).
			// 存量行里 file_url 是历史上写进去的死链：下载入口必须以注册路由为准。
			SetFileURL("/api/v1/tickets/attachments/" + name + "/download").
			SetFileSize(len(attachmentContent)).
			SetFileType("application/pdf").
			SetMimeType("application/pdf").
			SetUploadedBy(userA.ID).
			SetTenantID(tenantA.ID).
			SaveX(ctx)
	}
	attachmentA := newAttachment("report.pdf", filePath)
	attachmentB := newAttachment("gone.pdf", filepath.Join(dir, "gone.pdf"))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	const secret = "ticket-attachment-secret"
	SetupRoutes(r, &RouterConfig{
		JWTSecret:               secret,
		Logger:                  logger,
		Client:                  client,
		TicketAttachmentHandler: ticketAttachmentHandler.NewHandler(service.NewTicketAttachmentService(client, logger), logger),
	})

	return attachmentRouteFixture{
		engine:      r,
		secret:      secret,
		ticketA:     ticketA,
		attachmentA: attachmentA,
		attachmentB: attachmentB,
		userA:       userA,
		stranger:    stranger,
		outsider:    outsider,
	}
}

func (f attachmentRouteFixture) downloadPath(attachment *ent.TicketAttachment) string {
	return fmt.Sprintf("/api/v1/tickets/%d/attachments/%d", f.ticketA.ID, attachment.ID)
}

func (f attachmentRouteFixture) do(t *testing.T, method, path string, user *ent.User) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, f.secret, time.Hour)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	f.engine.ServeHTTP(w, req)
	return w
}

func TestTicketAttachmentDownloadRoute(t *testing.T) {
	fx := setupAttachmentRouteTest(t)
	downloadPath := fx.downloadPath(fx.attachmentA)
	previewPath := downloadPath + "/preview"

	t.Run("download 路由存在且返回真实字节", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, downloadPath, fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		assert.Equal(t, attachmentContent, w.Body.String(), "必须回吐磁盘上的原始字节")
		assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
		cd := w.Header().Get("Content-Disposition")
		assert.True(t, strings.HasPrefix(cd, "attachment"), "下载必须是 attachment，got=%q", cd)
		assert.Contains(t, cd, "report.pdf")
		assert.Equal(t, fmt.Sprint(len(attachmentContent)), w.Header().Get("Content-Length"))
	})

	t.Run("preview 路由返回 inline", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, previewPath, fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		cd := w.Header().Get("Content-Disposition")
		assert.True(t, strings.HasPrefix(cd, "inline"), "预览必须是 inline，got=%q", cd)
		assert.Contains(t, cd, "report.pdf")
		assert.Equal(t, attachmentContent, w.Body.String())
	})

	t.Run("列表返回的 fileUrl 就是可下载的真实路由", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/attachments", fx.ticketA.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var body struct {
			Code int `json:"code"`
			Data struct {
				Attachments []struct {
					ID       int    `json:"id"`
					TicketID int    `json:"ticketId"`
					FileURL  string `json:"fileUrl"`
				} `json:"attachments"`
				Total int `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Len(t, body.Data.Attachments, 2)

		var got string
		for _, a := range body.Data.Attachments {
			if a.ID == fx.attachmentA.ID {
				got = a.FileURL
			}
		}
		require.NotEmpty(t, got)
		assert.Equal(t, downloadPath, got,
			"fileUrl 必须由 (ticketId, id) 推导并等于注册路由，不得透传库里的死链")

		// 自证：前端照 fileUrl 点击必须拿到 200，而不是又一轮 404。
		click := fx.do(t, http.MethodGet, got, fx.userA)
		assert.Equal(t, http.StatusOK, click.Code, "fileUrl 指向的路由必须真实可达")
	})

	t.Run("跨租户探测返回 404 而不是确认资源存在", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, downloadPath, fx.stranger)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), `"code":4004`)
		assert.NotContains(t, w.Body.String(), "%PDF", "跨租户不得泄漏任何附件内容")

		w = fx.do(t, http.MethodGet, previewPath, fx.stranger)
		assert.Equal(t, http.StatusNotFound, w.Code)

		w = fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/attachments", fx.ticketA.ID), fx.stranger)
		assert.Equal(t, http.StatusNotFound, w.Code, "列表同样必须按租户 fail closed")
	})

	t.Run("同租户非相关方是 403", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, downloadPath, fx.outsider)
		assert.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		assert.Contains(t, w.Body.String(), `"code":2003`)
		assert.NotContains(t, w.Body.String(), "%PDF")
	})

	t.Run("附件不存在是 404", func(t *testing.T) {
		w := fx.do(t, http.MethodGet,
			fmt.Sprintf("/api/v1/tickets/%d/attachments/999999", fx.ticketA.ID), fx.userA)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), `"code":4004`)
	})

	t.Run("磁盘文件缺失是 500，不伪装成权限问题", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fx.downloadPath(fx.attachmentB), fx.userA)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
		assert.Contains(t, w.Body.String(), `"code":5001`)
		// 响应体不得泄漏服务端绝对路径。
		assert.NotContains(t, w.Body.String(), filepath.Dir(fx.attachmentB.FilePath))
	})
}
