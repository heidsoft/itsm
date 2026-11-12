package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// R2 回归锁：CMDB 异步导出此前 completed 后把 `/tmp/xxx.csv` 原样回给前端，
// 既没有下载端点，也没有 expires_at 比较点。补端点后必须同时挡住
// 跨租户、非操作人、未完成任务、过期任务与路径穿越。
func cmdbDownloadFixture(t *testing.T) (*ent.Client, *CMDBImportExportService, context.Context) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	client := enttest.Open(t, dialect.SQLite, dsn)
	svc := NewCMDBImportExportService(client, zap.NewNop().Sugar(), nil, nil)
	return client, svc, context.Background()
}

func cmdbDownloadTask(t *testing.T, svc *CMDBImportExportService, client *ent.Client, ctx context.Context,
	taskID string, tenantID, operatorID int, mutate func(*ent.CMDBExportTaskCreate)) string {
	t.Helper()
	path, _, err := svc.generateCSVExport(nil, []string{"id", "name"}, tenantID, taskID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(path) })

	create := client.CMDBExportTask.Create().
		SetTaskID(taskID).
		SetExportType("csv").
		SetTenantID(tenantID).
		SetOperatorID(operatorID).
		SetStatus("completed").
		SetFileURL(filepath.Base(path)).
		SetTotalCount(2).
		SetExpiresAt(time.Now().Add(time.Hour))
	if mutate != nil {
		mutate(create)
	}
	_, err = create.Save(ctx)
	require.NoError(t, err)
	return path
}

func requireAppStatus(t *testing.T, err error, want int) {
	t.Helper()
	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr), "业务拒绝必须是 *common.AppError，不能兜底成 500: %v", err)
	require.Equal(t, want, appErr.HTTPStatus)
}

func TestCMDBExportDownload_ResolvesRealFileForOwner(t *testing.T) {
	client, svc, ctx := cmdbDownloadFixture(t)
	wantPath := cmdbDownloadTask(t, svc, client, ctx, "TASK-OK", 1, 42, nil)

	path, name, err := svc.ResolveExportDownload(ctx, "TASK-OK", 1, 42)
	require.NoError(t, err)
	require.Equal(t, wantPath, path)
	_, err = os.Stat(path)
	require.NoError(t, err)
	require.Contains(t, name, "TASK-OK")
	require.True(t, strings.HasSuffix(name, ".csv"))
	require.NotContains(t, name, "/")
}

func TestCMDBExportDownload_RejectsCrossTenantAndForeignOperator(t *testing.T) {
	client, svc, ctx := cmdbDownloadFixture(t)
	cmdbDownloadTask(t, svc, client, ctx, "TASK-GUARD", 1, 42, nil)

	_, _, err := svc.ResolveExportDownload(ctx, "TASK-GUARD", 2, 42)
	requireAppStatus(t, err, http.StatusNotFound)

	_, _, err = svc.ResolveExportDownload(ctx, "TASK-GUARD", 1, 999)
	requireAppStatus(t, err, http.StatusForbidden)
}

func TestCMDBExportDownload_RejectsUnfinishedAndExpiredTasks(t *testing.T) {
	client, svc, ctx := cmdbDownloadFixture(t)
	cmdbDownloadTask(t, svc, client, ctx, "TASK-PROCESSING", 1, 42, func(c *ent.CMDBExportTaskCreate) {
		c.SetStatus("processing")
	})
	cmdbDownloadTask(t, svc, client, ctx, "TASK-EXPIRED", 1, 42, func(c *ent.CMDBExportTaskCreate) {
		c.SetExpiresAt(time.Now().Add(-time.Hour))
	})

	_, _, err := svc.ResolveExportDownload(ctx, "TASK-PROCESSING", 1, 42)
	requireAppStatus(t, err, http.StatusConflict)

	_, _, err = svc.ResolveExportDownload(ctx, "TASK-EXPIRED", 1, 42)
	requireAppStatus(t, err, http.StatusForbidden)
}

func TestCMDBExportDownload_RejectsPathTraversalAndForeignFileNames(t *testing.T) {
	client, svc, ctx := cmdbDownloadFixture(t)
	cmdbDownloadTask(t, svc, client, ctx, "TASK-TRAVERSAL", 1, 42, func(c *ent.CMDBExportTaskCreate) {
		c.SetFileURL("../../etc/passwd")
	})
	cmdbDownloadTask(t, svc, client, ctx, "TASK-FOREIGN", 1, 42, func(c *ent.CMDBExportTaskCreate) {
		c.SetFileURL("cmdb-export-t2-foreign-deadbeef.csv")
	})

	_, _, err := svc.ResolveExportDownload(ctx, "TASK-TRAVERSAL", 1, 42)
	requireAppStatus(t, err, http.StatusBadRequest)

	_, _, err = svc.ResolveExportDownload(ctx, "TASK-FOREIGN", 1, 42)
	requireAppStatus(t, err, http.StatusBadRequest)
}

func TestCMDBExportDTO_NeverLeaksServerPathAndMarksTruncation(t *testing.T) {
	_, svc, _ := cmdbDownloadFixture(t)
	now := time.Now()
	completed := &ent.CMDBExportTask{
		TaskID: "TASK-DTO", Status: "completed", FileURL: "/tmp/cmdb-export-old.csv",
		TotalCount: cmdbExportMaxRecords, ExpiresAt: now,
	}

	res := svc.convertToExportDTO(completed)
	require.Equal(t, "/api/v1/cmdb/export/TASK-DTO/download", res.FileURL)
	require.NotContains(t, res.FileURL, "/tmp")
	require.Equal(t, cmdbExportMaxRecords, res.ExportLimit)
	require.True(t, res.Truncated, "达到上限必须显式告知用户，不能静默截断")

	pending := &ent.CMDBExportTask{TaskID: "TASK-PENDING", Status: "pending", ExpiresAt: now}
	require.Empty(t, svc.convertToExportDTO(pending).FileURL, "未完成任务不得给出下载 URL")
}
