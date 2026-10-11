package service_request

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
)

// 条件更新 0 行命中时 Ent 统一报 NotFoundError，无法区分「记录已不存在」与
// 「版本被并发推进」。service 层经 classifyWriteFailure 复查存在性后分别映射
// 404 与 409；冲突若被吞成 InternalError(5001)，前端无法区分「刷新重试」与
// 「系统故障」。冲突窗口只存在于入口 Get 与条件更新之间的 TOCTOU 间隙，
// 串行测试无法从公开入口稳定注入，故对收口 helper 直接单测三分类。

func casConflictSetup(t *testing.T) (*Service, *EntRepository, *ent.Client, int) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_cas_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName(fmt.Sprintf("SRCAS-%d", time.Now().UnixNano())).
		SetCode(fmt.Sprintf("SRCAS-%d", time.Now().UnixNano())).
		SetDomain("sr-cas.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	svc := NewService(repo, nil, nil, client, zaptest.NewLogger(t).Sugar(), nil)
	return svc, repo, client, tenant.ID
}

func TestClassifyWriteFailure_ConcurrentVersionAdvance_ReturnsVersionConflict(t *testing.T) {
	svc, repo, client, tenantID := casConflictSetup(t)
	ctx := context.Background()

	created, err := client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(1).
		SetRequesterID(1).
		SetStatus(SRStatusSecurityApproved).
		SetTitle("CAS conflict").
		SetReason("testing").
		SetCurrentLevel(1).
		SetTotalLevels(3).
		Save(ctx)
	require.NoError(t, err)

	stale := &ServiceRequest{ID: created.ID, TenantID: tenantID, Version: created.Version}

	// 并发推进：条件更新成功后版本 +1，stale 快照的后续写必然 0 行命中。
	require.NoError(t, repo.UpdateStatus(ctx, &ServiceRequest{ID: created.ID, TenantID: tenantID, Version: created.Version}, SRStatusProvisioning, 1))

	err = svc.classifyWriteFailure(ctx, stale, &ent.NotFoundError{}, "Failed to update service request status")
	require.Error(t, err)
	require.True(t, common.IsVersionConflictError(err), "expected VersionConflictError, got %T: %v", err, err)

	conflict := err.(*common.VersionConflictError)
	require.Equal(t, stale.Version, conflict.CurrentVersion)
	require.Equal(t, created.Version+1, conflict.ServerVersion)
}

func TestClassifyWriteFailure_RecordGone_ReturnsNotFound(t *testing.T) {
	svc, _, _, tenantID := casConflictSetup(t)
	ctx := context.Background()

	gone := &ServiceRequest{ID: 999999, TenantID: tenantID, Version: 1}
	err := svc.classifyWriteFailure(ctx, gone, &ent.NotFoundError{}, "Failed to delete service request")
	require.Error(t, err)

	appErr, ok := err.(*common.AppError)
	require.True(t, ok, "expected AppError, got %T", err)
	require.Equal(t, common.ErrCodeNotFound, appErr.Code)
}

func TestClassifyWriteFailure_OtherError_StaysInternal(t *testing.T) {
	svc, _, _, tenantID := casConflictSetup(t)
	ctx := context.Background()

	req := &ServiceRequest{ID: 1, TenantID: tenantID, Version: 1}
	err := svc.classifyWriteFailure(ctx, req, errors.New("db connection lost"), "Failed to update service request")
	require.Error(t, err)

	appErr, ok := err.(*common.AppError)
	require.True(t, ok, "expected AppError, got %T", err)
	require.Equal(t, common.ErrCodeInternal, appErr.Code)
}
