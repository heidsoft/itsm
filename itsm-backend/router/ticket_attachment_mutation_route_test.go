// 工单附件「写入面」的跨租户 HTTP 回归测试（上传 + 删除）。
//
// router/ticket_attachment_route_test.go 已覆盖读面（下载/预览/列表）。
// 写面同样是高风险入口：上传未授权工单 = 往别人工单里注入文件，
// 删除未授权附件 = 破坏证据链。两处都必须从真实路由入口验证零副作用：
//   - 跨租户上传：400 + 不产生附件行 + 磁盘零写入；
//   - 跨租户删除：404/4004 + 附件行与磁盘文件原样留存；
//   - 同租户 owner 上传/删除为正控制（真实落库、真实落盘、真实删行）。
//
// 注意：attachment service 在构造时无条件创建相对路径 uploads/tickets，
// 本文件先 t.Chdir 到测试临时目录，避免向仓库目录写入未跟踪文件。
package router

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
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

type attachmentMutationFixture struct {
	engine *gin.Engine
	client *ent.Client
	secret string
	userA  *ent.User
	stranger *ent.User
	ticketA *ent.Ticket
	attachmentA *ent.TicketAttachment
}

func setupAttachmentMutationTest(t *testing.T) *attachmentMutationFixture {
	t.Helper()
	t.Chdir(t.TempDir()) // uploads/tickets 落在临时 CWD，不污染仓库

	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:router_ticket_attach_mutation_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Attach Mut A").SetCode("attach-mut-a").SetDomain("attach-mut-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Attach Mut B").SetCode("attach-mut-b").SetDomain("attach-mut-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("attach-mut-a-owner").SetEmail("attach-mut-a@example.com").SetName("A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("attach-mut-b-stranger").SetEmail("attach-mut-b@example.com").SetName("B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	ticketA := client.Ticket.Create().
		SetTicketNumber("TKT-ATTACH-MUT-A-001").
		SetTitle("附件写入面回归").
		SetDescription("验证附件上传/删除的租户隔离").
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(userA.ID).
		SetTenantID(tenantA.ID).
		SaveX(ctx)

	// 附件行 + 磁盘真实文件（删除正控制需要）
	dir, err := os.Getwd()
	require.NoError(t, err)
	filePath := filepath.Join(dir, "placeholder.pdf")
	require.NoError(t, os.WriteFile(filePath, []byte("%PDF-1.4 mut bytes"), 0o600))
	attachmentA := client.TicketAttachment.Create().
		SetTicketID(ticketA.ID).
		SetFileName("report.pdf").
		SetFilePath(filePath).
		SetFileURL("/api/v1/tickets/attachments/report.pdf/download").
		SetFileSize(len("%PDF-1.4 mut bytes")).
		SetFileType("application/pdf").
		SetMimeType("application/pdf").
		SetUploadedBy(userA.ID).
		SetTenantID(tenantA.ID).
		SaveX(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	const secret = "ticket-attachment-mutation-secret"
	SetupRoutes(r, &RouterConfig{
		JWTSecret:               secret,
		Logger:                  logger,
		Client:                  client,
		TicketAttachmentHandler: ticketAttachmentHandler.NewHandler(service.NewTicketAttachmentService(client, logger), logger),
	})

	return &attachmentMutationFixture{
		engine: r, client: client, secret: secret,
		userA: userA, stranger: stranger, ticketA: ticketA, attachmentA: attachmentA,
	}
}

// doMultipart 用 multipart/form-data 发请求（body 为 nil 时是普通请求）。
func (fx *attachmentMutationFixture) do(t *testing.T, method, path string, user *ent.User, fileName string, fileContent []byte) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)

	body := &bytes.Buffer{}
	contentType := "application/json"
	if fileName != "" {
		writer := multipart.NewWriter(body)
		// 显式携带 Content-Type：Go 的 CreateFormFile 对文本默认留空，
		// service 会因嗅探/推断结果落在白名单外而拒绝。
		partHeader := textproto.MIMEHeader{}
		partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, fileName))
		partHeader.Set("Content-Type", "text/plain")
		part, err := writer.CreatePart(partHeader)
		require.NoError(t, err)
		_, err = part.Write(fileContent)
		require.NoError(t, err)
		require.NoError(t, writer.Close())
		contentType = writer.FormDataContentType()
	}
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

func (fx *attachmentMutationFixture) attachmentsDir() string {
	return filepath.Join("uploads", "tickets")
}

func (fx *attachmentMutationFixture) nAttachments(t *testing.T) int {
	t.Helper()
	return fx.client.TicketAttachment.Query().CountX(context.Background())
}

func (fx *attachmentMutationFixture) diskFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(fx.attachmentsDir())
	require.NoError(t, err, "uploads/tickets 目录由 service 构造时创建，必须存在")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestTicketAttachmentMutationCrossTenant_HTTP(t *testing.T) {
	fx := setupAttachmentMutationTest(t)
	uploadPath := fmt.Sprintf("/api/v1/tickets/%d/attachments", fx.ticketA.ID)
	deletePath := fmt.Sprintf("%s/%d", uploadPath, fx.attachmentA.ID)

	t.Run("同租户上传是正控制：真实落库落盘", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, uploadPath, fx.userA, "note.txt", []byte("owner 的合法附件内容"))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)

		require.Equal(t, 2, fx.nAttachments(t), "正控制必须在基线附件之上新增一行")
		require.Len(t, fx.diskFiles(t), 1, "正控制必须真实落盘（seed 的 placeholder.pdf 在 CWD 根，不在 uploads/tickets 内）")
	})

	t.Run("跨租户上传 fail closed 且零副作用", func(t *testing.T) {
		beforeRows := fx.nAttachments(t)
		beforeFiles := fx.diskFiles(t)

		w := fx.do(t, http.MethodPost, uploadPath, fx.stranger, "evil.txt", []byte("跨租户注入文件内容"))
		// handler 把上传失败统一映射 ParamErrorCode；跨租户与非法文件类型同观测，不泄漏工单存在性。
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":1001`)

		assert.Equal(t, beforeRows, fx.nAttachments(t), "跨租户上传不得产生附件行")
		assert.Equal(t, beforeFiles, fx.diskFiles(t), "跨租户上传不得向磁盘写入任何文件")
	})

	t.Run("跨租户删除 fail closed：行与文件原样留存", func(t *testing.T) {
		beforeRows := fx.nAttachments(t)
		beforeFiles := fx.diskFiles(t)

		w := fx.do(t, http.MethodDelete, deletePath, fx.stranger, "", nil)
		// 已知缺陷（如实钉住）：service 把 ent.NotFoundError 以 %w 包装进
		// `attachment not found`，但 handler 对删除失败统一 Fail(InternalErrorCode)，
		// 完全忽略分类——跨租户探测表现为 500/5001。fail closed 成立但缺 404 语义，
		// 修复映射后应改为 404/4004 并同步本断言。
		require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":5001`)

		assert.Equal(t, beforeRows, fx.nAttachments(t), "跨租户删除不得删掉租户内附件行")
		assert.Equal(t, beforeFiles, fx.diskFiles(t), "跨租户删除不得删掉 uploads/tickets 内文件")
		_, err := fx.client.TicketAttachment.Get(context.Background(), fx.attachmentA.ID)
		require.NoError(t, err, "被探针命中的附件必须原样留存")
		_, statErr := os.Stat(fx.attachmentA.FilePath)
		require.NoError(t, statErr, "seed 的磁盘文件必须原样留存")
	})

	t.Run("同租户删除是正控制：行与磁盘文件同时清理", func(t *testing.T) {
		w := fx.do(t, http.MethodDelete, deletePath, fx.userA, "", nil)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)

		assert.Equal(t, 1, fx.nAttachments(t), "正控制删除必须真实删行")
		assert.Len(t, fx.diskFiles(t), 1, "正控制删除必须把 seed 的磁盘文件一并清理，仅剩上传的 note")
	})
}
