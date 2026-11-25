package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// CMDB 异步导出落盘回归锁（2026-11-05 扩展白盒审查 R3）。
//
// 旧文件名只有秒级时间戳，两个同秒任务（可不同租户/不同过滤条件）会互相
// os.Create 截断，先完成任务的 file_url 指向后一个任务的文件内容。补上下载
// 端点后这就是一条跨租户文件串读路径。

func TestCMDBExportFilePath_IsUniqueTenantScopedAndPathSafe(t *testing.T) {
	p1, err := cmdbExportFilePath(1, "TASK-1", "csv")
	require.NoError(t, err)
	p2, err := cmdbExportFilePath(1, "TASK-1", "csv")
	require.NoError(t, err)
	p3, err := cmdbExportFilePath(2, "TASK-1", "csv")
	require.NoError(t, err)

	require.NotEqual(t, p1, p2, "同一任务同一秒两次生成不得同名")
	require.Contains(t, filepath.Base(p1), "-t1-")
	require.Contains(t, filepath.Base(p3), "-t2-", "文件名必须带租户维度")
	require.True(t, strings.HasSuffix(p1, ".csv"))

	// 任务号里的路径分隔符不得逃出租户隔离目录。
	evil, err := cmdbExportFilePath(1, "../../etc/passwd", "csv")
	require.NoError(t, err)
	tenantDir, err := cmdbExportDir(1)
	require.NoError(t, err)
	require.Equal(t, tenantDir, filepath.Dir(evil), "导出文件必须留在当前租户的私有目录")
	require.Contains(t, tenantDir, "t1")
	require.NotContains(t, filepath.Base(evil), "/")
	require.Contains(t, filepath.Base(evil), "______etc_passwd")
}

func TestCMDBExportCSV_DoesNotOverwriteConcurrentTask(t *testing.T) {
	svc := &CMDBImportExportService{}

	p1, size1, err := svc.generateCSVExport(nil, []string{"id", "name"}, 1, "TASK-SAME-SECOND")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(p1) })
	p2, size2, err := svc.generateCSVExport(nil, []string{"id"}, 1, "TASK-SAME-SECOND")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(p2) })

	require.NotEqual(t, p1, p2, "同秒并发导出必须落在不同文件")

	b1, err := os.ReadFile(p1)
	require.NoError(t, err)
	b2, err := os.ReadFile(p2)
	require.NoError(t, err)
	require.NotEqual(t, string(b1), string(b2), "两次导出的表头不同，说明没有互相覆盖")
	require.Greater(t, size1, int64(0))
	require.Greater(t, size2, int64(0))

	info, err := os.Stat(p1)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "CI 资产清单不应同机其他用户可读")
}

func TestCMDBExportExcel_DoesNotOverwriteConcurrentTask(t *testing.T) {
	svc := &CMDBImportExportService{}

	p1, _, err := svc.generateExcelExport(nil, []string{"id", "name"}, 1, "TASK-XLSX")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(p1) })
	p2, _, err := svc.generateExcelExport(nil, []string{"id"}, 1, "TASK-XLSX")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(p2) })

	require.NotEqual(t, p1, p2)
	require.True(t, strings.HasSuffix(p1, ".xlsx"))
	_, err = os.Stat(p1)
	require.NoError(t, err, "后一个任务不得覆盖前一个任务的 xlsx")
	_, err = os.Stat(p2)
	require.NoError(t, err)
}
