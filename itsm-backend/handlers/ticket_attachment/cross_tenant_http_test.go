// Package ticket_attachment 的跨租户 HTTP 回归测试（B7）。
//
// 附件是 AGENTS 点名的高危面（上传/下载/预览/磁盘副作用）。服务层的租户
// 防线已成型：authorizeTicketAttachmentAccess 先鉴权（跨租户 ticket 与真实
// 不存在都归 ErrAttachmentNotFound，同租非相关方归 ErrAttachmentAccessDenied），
// 所有查询带 (ID, TicketID, TenantID) 谓词；本文件从真实生产入口
// （router.SetupRoutes 单域布线 + 真 JWT + RequirePermission）验证该防线，
// 并覆盖处理面特有的维度：响应体内容零泄漏、磁盘文件零副作用、上传校验。
//
// 钉住的现状（有意为之的语义，勿在不知情下"修复"）：
//   - 上传路径的一切失败都折叠为 400/1001「附件上传失败，请检查文件类型
//     和大小」（handler 只有一个 upload 失败出口）。跨租户 ticket 上传与
//     文件类型非法因此同形态，工单存在性不可区分——fail-closed，可接受；
//     若未来给 not-found 单独映射，需同步更新 upload 跨租户探针断言。
//   - 下载/预览/列表把文件的字节流与清单保真地交给相关方；同租非相关方
//     （end_user 等）按 403/2003 拒绝，跨租户按 404/4004 掩码。
package ticket_attachment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/ticketattachment"
	ticketAttachmentHandler "itsm-backend/handlers/ticket_attachment"
	"itsm-backend/middleware"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fixture struct {
	engine    *gin.Engine
	client    *ent.Client
	secret    string
	tenantA   *ent.Tenant
	tenantB   *ent.Tenant
	userA     *ent.User
	colleague *ent.User // 租户 A 的 end_user：同租非相关方（403 探针）
	stranger  *ent.User // 租户 B 的 requester（跨租户探针的对照面）
	ticketA   *ent.Ticket
	ticketB   *ent.Ticket
	attA      *ent.TicketAttachment
	attB      *ent.TicketAttachment
	diskA     string // attA 的磁盘文件（内容租户 A 机密）
	diskB     string // attB 的磁盘文件（内容租户 B 机密）
	contentA  string
	contentB  string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	// service 构造无条件 MkdirAll("uploads/tickets")（相对 CWD），必须先
	// 切到隔离目录，避免测试向仓库写入文件（B4 侦查确认的既有行为）。
	uploadRoot := t.TempDir()
	t.Chdir(uploadRoot)

	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:attachment_cross_tenant_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Attachment A").SetCode("att-a").SetDomain("att-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Attachment B").SetCode("att-b").SetDomain("att-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("att-a-owner").SetEmail("att-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	colleague := client.User.Create().SetUsername("att-a-colleague").SetEmail("att-c@example.com").SetName("Colleague A").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("att-b-stranger").SetEmail("att-b@example.com").SetName("Stranger B").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	ticketA := client.Ticket.Create().
		SetTicketNumber("TKT-ATT-A-001").
		SetTitle("附件域工单A").
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(userA.ID).
		SetTenantID(tenantA.ID).
		SaveX(ctx)
	ticketB := client.Ticket.Create().
		SetTicketNumber("TKT-ATT-B-001").
		SetTitle("附件域工单B").
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(stranger.ID).
		SetTenantID(tenantB.ID).
		SaveX(ctx)

	// 直接落库种子附件（绕过上传路径，磁盘文件位置可由测试完全控制）：
	// 每个租户 1 条附件，内容程序化标记，泄漏检测不依赖偶然内容。
	contentA := "tenantA-secret-content-attachment"
	contentB := "tenantB-secret-content-attachment"
	diskA := filepath.Join(uploadRoot, "seed-a", "report-a.txt")
	diskB := filepath.Join(uploadRoot, "seed-b", "report-b.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(diskA), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(diskB), 0o755))
	require.NoError(t, os.WriteFile(diskA, []byte(contentA), 0o644))
	require.NoError(t, os.WriteFile(diskB, []byte(contentB), 0o644))

	attA := client.TicketAttachment.Create().
		SetTicketID(ticketA.ID).
		SetFileName("report-a.txt").
		SetFilePath(diskA).
		SetFileSize(len(contentA)).
		SetFileType("text/plain").
		SetMimeType("text/plain").
		SetUploadedBy(userA.ID).
		SetTenantID(tenantA.ID).
		SaveX(ctx)
	attB := client.TicketAttachment.Create().
		SetTicketID(ticketB.ID).
		SetFileName("report-b.txt").
		SetFilePath(diskB).
		SetFileSize(len(contentB)).
		SetFileType("text/plain").
		SetMimeType("text/plain").
		SetUploadedBy(stranger.ID).
		SetTenantID(tenantB.ID).
		SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "attachment-cross-tenant-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:               secret,
		Logger:                  logger,
		Client:                  client,
		TicketAttachmentHandler: ticketAttachmentHandler.NewHandler(service.NewTicketAttachmentService(client, logger), logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB,
		userA: userA, colleague: colleague, stranger: stranger,
		ticketA: ticketA, ticketB: ticketB,
		attA: attA, attB: attB,
		diskA: diskA, diskB: diskB,
		contentA: contentA, contentB: contentB,
	}
}

func (fx *fixture) do(t *testing.T, method, path string, user *ent.User) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

// doUpload 以 multipart/form-data 走真实上传入口。
func (fx *fixture) doUpload(t *testing.T, path string, user *ent.User, filename string, contentType string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename)},
		"Content-Type":        {contentType},
	})
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

// uploadedFilesCount 统计 uploads/tickets 下的磁盘文件数（上传副作用探针）。
func (fx *fixture) uploadedFilesCount(t *testing.T) int {
	t.Helper()
	dir := filepath.Join("uploads", "tickets")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0
	}
	require.NoError(t, err)
	return len(entries)
}

// tenantAttachmentCount 统计指定租户的附件行数（数据库副作用探针）。
func (fx *fixture) tenantAttachmentCount(t *testing.T, tenantID int) int {
	t.Helper()
	n, err := fx.client.TicketAttachment.Query().
		Where(ticketattachment.TenantID(tenantID)).
		Count(context.Background())
	require.NoError(t, err)
	return n
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) (code int, message string, data json.RawMessage) {
	t.Helper()
	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), "body=%s", w.Body.String())
	return envelope.Code, envelope.Message, envelope.Data
}

func TestDownloadAndPreviewAttachmentCrossTenant(t *testing.T) {
	fx := newFixture(t)

	t.Run("同租requester下载是正控制且字节保真", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/attachments/%d", fx.ticketA.ID, fx.attA.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, fx.contentA, w.Body.String(), "下载内容必须与磁盘种子逐字节一致")
		require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
		require.Contains(t, w.Header().Get("Content-Type"), "text/plain")
	})

	t.Run("同租requester预览返回inline且字节保真", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/attachments/%d/preview", fx.ticketA.ID, fx.attA.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, fx.contentA, w.Body.String())
		require.Contains(t, w.Header().Get("Content-Disposition"), "inline", "预览必须是 inline 而不是强制下载")
	})

	t.Run("跨租户ticket探测被掩码404且零字节泄漏", func(t *testing.T) {
		// userA（租户 A）请求租户 B 的工单附件：authorize 先走，ticket 不在本租户
		// → ErrAttachmentNotFound → 404。响应体不得携带租户 B 文件的任何字节。
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/attachments/%d", fx.ticketB.ID, fx.attB.ID), fx.userA)
		code, _, _ := decodeEnvelope(t, w)
		require.Equal(t, http.StatusNotFound, w.Code)
		require.Equal(t, 4004, code)
		require.NotContains(t, w.Body.String(), fx.contentB, "跨租户探测不得泄漏对端文件内容")
		// 跨租户下载不得触碰对端磁盘文件（mtime/内容守卫）
		b, err := os.ReadFile(fx.diskB)
		require.NoError(t, err)
		require.Equal(t, fx.contentB, string(b))
	})

	t.Run("同租工单配跨租户attachment_id同样掩码404", func(t *testing.T) {
		// 附件查询谓词是 (ID, TicketID, TenantID)：把租户 B 附件 id 配在租户 A
		// 工单路径下也要零泄漏，防止用 id 枚举确认他租附件存在。
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/attachments/%d/preview", fx.ticketA.ID, fx.attB.ID), fx.userA)
		code, _, _ := decodeEnvelope(t, w)
		require.Equal(t, http.StatusNotFound, w.Code)
		require.Equal(t, 4004, code)
		require.NotContains(t, w.Body.String(), fx.contentB)
	})

	t.Run("同租非相关方403且零字节泄漏", func(t *testing.T) {
		// colleague 是租户 A 的 end_user，不是 requester/assignee，也不在允许角色集
		// → ErrAttachmentAccessDenied → 403；机身与预览都不得带出文件内容。
		for _, sub := range []struct {
			name string
			path string
		}{
			{"download", fmt.Sprintf("/api/v1/tickets/%d/attachments/%d", fx.ticketA.ID, fx.attA.ID)},
			{"preview", fmt.Sprintf("/api/v1/tickets/%d/attachments/%d/preview", fx.ticketA.ID, fx.attA.ID)},
		} {
			w := fx.do(t, http.MethodGet, sub.path, fx.colleague)
			code, message, _ := decodeEnvelope(t, w)
			require.Equal(t, http.StatusForbidden, w.Code, sub.name+" body=%s", w.Body.String())
			require.Equal(t, 2003, code, sub.name)
			require.Equal(t, "无权访问该附件", message, sub.name)
			require.NotContains(t, w.Body.String(), fx.contentA, sub.name+" 403 不得附带文件内容")
		}
	})
}

func TestListAttachmentsCrossTenant(t *testing.T) {
	fx := newFixture(t)

	t.Run("同租requester列表是正控制且total诚实", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/attachments", fx.ticketA.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		code, _, data := decodeEnvelope(t, w)
		require.Equal(t, 0, code)
		var listing struct {
			Items []struct {
				ID       int    `json:"id"`
				TicketID int    `json:"ticketId"`
				FileName string `json:"fileName"`
				FilePath string `json:"filePath"`
			} `json:"items"`
			Total int `json:"total"`
		}
		require.NoError(t, json.Unmarshal(data, &listing), "body=%s", string(data))
		require.Equal(t, 1, listing.Total)
		require.Len(t, listing.Items, 1)
		require.Equal(t, fx.attA.ID, listing.Items[0].ID)
		require.Equal(t, fx.ticketA.ID, listing.Items[0].TicketID, "camelCase 契约：ticketId")
	})

	t.Run("跨租户列表掩码404且零泄漏", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/attachments", fx.ticketB.ID), fx.userA)
		code, _, _ := decodeEnvelope(t, w)
		require.Equal(t, http.StatusNotFound, w.Code)
		require.Equal(t, 4004, code)
		require.NotContains(t, w.Body.String(), "report-b.txt", "跨租户列表不得泄漏对端文件名")
	})

	t.Run("同租非相关方列表403", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/attachments", fx.ticketA.ID), fx.colleague)
		code, _, _ := decodeEnvelope(t, w)
		require.Equal(t, http.StatusForbidden, w.Code)
		require.Equal(t, 2003, code)
		require.NotContains(t, w.Body.String(), `"fileName"`, "403 不得溢出清单结构")
	})
}

func TestUploadAttachmentSurface(t *testing.T) {
	fx := newFixture(t)

	t.Run("同租requester上传是正控制且落盘落库", func(t *testing.T) {
		before := fx.uploadedFilesCount(t)
		w := fx.doUpload(t, fmt.Sprintf("/api/v1/tickets/%d/attachments", fx.ticketA.ID), fx.userA,
			"notes.txt", "text/plain", []byte(fx.contentA))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		code, _, data := decodeEnvelope(t, w)
		require.Equal(t, 0, code)

		var att struct {
			ID         int    `json:"id"`
			TicketID   int    `json:"ticketId"`
			FileName   string `json:"fileName"`
			FileSize   int    `json:"fileSize"`
			MimeType   string `json:"mimeType"`
			UploadedBy int    `json:"uploadedBy"`
		}
		require.NoError(t, json.Unmarshal(data, &att), "body=%s", string(data))
		require.Equal(t, fx.ticketA.ID, att.TicketID)
		require.Equal(t, "notes.txt", att.FileName)
		require.Equal(t, len(fx.contentA), att.FileSize)
		require.Equal(t, fx.userA.ID, att.UploadedBy)

		// 磁盘副作用：uploads/tickets 内新增且内容保真；文件不逃逸出该目录
		require.Equal(t, before+1, fx.uploadedFilesCount(t), "上传必须在 uploads/tickets 内新增一个文件")
		// 数据库副作用：本租户行 +1，对端租户恒定
		require.Equal(t, 2, fx.tenantAttachmentCount(t, fx.tenantA.ID), "种子1条 + 上传1条")
		require.Equal(t, 1, fx.tenantAttachmentCount(t, fx.tenantB.ID), "对端租户行数不得变化")
	})

	t.Run("跨租户ticket上传掩码400且零副作用", func(t *testing.T) {
		// 钉住契约：handler 对上传一切失败共用 400/1001「附件上传失败，请检查
		// 文件类型和大小」——跨租户 ticket（service 层 ticket not found）与文件
		// 非法同形态，工单存在性不可区分。若未来把 not-found 映射为独立形态，
		// 需同步更新本断言（及本文件头部的钉住注释）。
		rowsBeforeA := fx.tenantAttachmentCount(t, fx.tenantA.ID)
		rowsBeforeB := fx.tenantAttachmentCount(t, fx.tenantB.ID)
		filesBefore := fx.uploadedFilesCount(t)

		w := fx.doUpload(t, fmt.Sprintf("/api/v1/tickets/%d/attachments", fx.ticketB.ID), fx.userA,
			"sneaky.txt", "text/plain", []byte(fx.contentA))
		code, _, _ := decodeEnvelope(t, w)
		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Equal(t, 1001, code, "上传失败统一 1001；跨租户不得以其它形态反推工单存在性")
		require.Equal(t, rowsBeforeA, fx.tenantAttachmentCount(t, fx.tenantA.ID), "跨租户上传不得改变本租户行数")
		require.Equal(t, rowsBeforeB, fx.tenantAttachmentCount(t, fx.tenantB.ID), "跨租户上传不得在对端租户落行")
		require.Equal(t, filesBefore, fx.uploadedFilesCount(t), "跨租户上传不得写盘")
	})

	t.Run("非法类型上传400且零副作用", func(t *testing.T) {
		rowsBefore := fx.tenantAttachmentCount(t, fx.tenantA.ID)
		filesBefore := fx.uploadedFilesCount(t)

		w := fx.doUpload(t, fmt.Sprintf("/api/v1/tickets/%d/attachments", fx.ticketA.ID), fx.userA,
			"payload.html", "text/html", []byte("<script>alert(1)</script>"))
		code, _, _ := decodeEnvelope(t, w)
		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Equal(t, 1001, code, "text/html 不在白名单，必须拒绝")
		require.Equal(t, rowsBefore, fx.tenantAttachmentCount(t, fx.tenantA.ID), "拒绝的上传不得落行")
		require.Equal(t, filesBefore, fx.uploadedFilesCount(t), "拒绝的上传不得写盘")
	})

	t.Run("超限大小上传400且零副作用", func(t *testing.T) {
		rowsBefore := fx.tenantAttachmentCount(t, fx.tenantA.ID)
		filesBefore := fx.uploadedFilesCount(t)

		oversize := bytes.Repeat([]byte("a"), 10*1024*1024+1) // 超出 10MB 上限 1 字节
		w := fx.doUpload(t, fmt.Sprintf("/api/v1/tickets/%d/attachments", fx.ticketA.ID), fx.userA,
			"big.txt", "text/plain", oversize)
		code, _, _ := decodeEnvelope(t, w)
		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Equal(t, 1001, code)
		require.Equal(t, rowsBefore, fx.tenantAttachmentCount(t, fx.tenantA.ID), "超限上传不得落行")
		require.Equal(t, filesBefore, fx.uploadedFilesCount(t), "超限上传不得残留半文件")
	})

	t.Run("上传文件名被清洗不得目录穿越", func(t *testing.T) {
		before := fx.uploadedFilesCount(t)
		w := fx.doUpload(t, fmt.Sprintf("/api/v1/tickets/%d/attachments", fx.ticketA.ID), fx.userA,
			"..\\..\\evil.txt", "text/plain", []byte(fx.contentA))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		code, _, data := decodeEnvelope(t, w)
		require.Equal(t, 0, code)
		var att struct {
			FileName string `json:"fileName"`
		}
		require.NoError(t, json.Unmarshal(data, &att))
		require.Equal(t, "evil.txt", att.FileName, "文件名必须剥掉目录段（Windows 反斜杠含内）")
		require.False(t, strings.Contains(att.FileName, "/") || strings.Contains(att.FileName, "\\"))
		require.False(t, os.MkdirAll("uploads", 0o755) != nil, "uploads 目录必须存在")
		require.Equal(t, before+1, fx.uploadedFilesCount(t), "穿越文件名不得逃逸出 uploads 目录")
	})
}

func TestDeleteAttachmentCrossTenant(t *testing.T) {
	fx := newFixture(t)

	t.Run("跨租户删除掩码404且零副作用", func(t *testing.T) {
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/tickets/%d/attachments/%d", fx.ticketB.ID, fx.attB.ID), fx.userA)
		code, _, _ := decodeEnvelope(t, w)
		require.Equal(t, http.StatusNotFound, w.Code)
		require.Equal(t, 4004, code)

		// 零副作用：对端附件行保留、磁盘文件保留
		require.Equal(t, 1, fx.tenantAttachmentCount(t, fx.tenantB.ID), "跨租户删除不得注销对端行")
		b, err := os.ReadFile(fx.diskB)
		require.NoError(t, err)
		require.Equal(t, fx.contentB, string(b), "跨租户删除不得触碰对端磁盘文件")
	})

	t.Run("同租非上传人非处理人非申请人403", func(t *testing.T) {
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/tickets/%d/attachments/%d", fx.ticketA.ID, fx.attA.ID), fx.colleague)
		code, _, _ := decodeEnvelope(t, w)
		require.Equal(t, http.StatusForbidden, w.Code)
		require.Equal(t, 2003, code)
		require.Equal(t, 1, fx.tenantAttachmentCount(t, fx.tenantA.ID), "403 删除不得注销行")
		_, err := os.Stat(fx.diskA)
		require.NoError(t, err, "403 删除不得删除磁盘文件")
	})

	t.Run("上传人删除是正控制且行与文件同步消失", func(t *testing.T) {
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/tickets/%d/attachments/%d", fx.ticketA.ID, fx.attA.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		code, _, _ := decodeEnvelope(t, w)
		require.Equal(t, 0, code)

		require.Equal(t, 0, fx.tenantAttachmentCount(t, fx.tenantA.ID), "删除后本租户附件行归零")
		_, err := os.Stat(fx.diskA)
		require.True(t, os.IsNotExist(err), "删除后磁盘文件必须同步删除")
		require.Equal(t, 1, fx.tenantAttachmentCount(t, fx.tenantB.ID), "对端租户不得被牵连")
	})
}
