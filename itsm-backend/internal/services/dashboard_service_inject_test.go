package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDashboardService_InjectDB_DBReachesRepo(t *testing.T) {
	db, _, _, _ := newRepoMockDB(t)

	svc := NewDashboardServiceWithDB(nil, db, zap.NewNop().Sugar())

	require.NotNil(t, svc, "构造器必须返回非 nil service")
	assert.Same(t, db, svc.db, "service.db 必须等于注入的 *sql.DB")
	require.NotNil(t, svc.repo, "构造器必须初始化 repo")
	assert.Same(t, db, svc.repo.db, "repo.db 必须等于注入的 *sql.DB（同一指针）")
}

func TestDashboardService_InjectDB_NilDBPropagates(t *testing.T) {
	svc := NewDashboardServiceWithDB(nil, nil, zap.NewNop().Sugar())
	require.NotNil(t, svc)
	assert.Nil(t, svc.db)
	require.NotNil(t, svc.repo)
	assert.Nil(t, svc.repo.db, "repo 与 service 持有同一个 nil db，体现同源")
}

func TestDashboardService_BackwardCompat_PropagatesDB(t *testing.T) {
	db, _, _, _ := newRepoMockDB(t)

	oldSvc := NewDashboardServiceWithDB(nil, db, zap.NewNop().Sugar())
	require.NotNil(t, oldSvc)

	newSvc := NewDashboardServiceWithDB(nil, oldSvc.db, zap.NewNop().Sugar())
	require.NotNil(t, newSvc)

	assert.Same(t, oldSvc.db, newSvc.db,
		"两次 WithDB 构造使用同一 db 时，service.db 必须是同一指针")
	assert.Same(t, oldSvc.repo.db, newSvc.repo.db,
		"两次 WithDB 构造使用同一 db 时，repo.db 必须是同一指针（db 透传链稳定）")
}

func TestDashboardService_NilService_NoMethodPanicOnRepoAccess(t *testing.T) {
	var svc *DashboardService
	defer func() {
		r := recover()
		require.NotNil(t, r, "nil service 访问字段必须 panic，便于尽早发现未构造的 service")
	}()
	_ = svc.repo
}

func TestDashboardService_LoggerInjected(t *testing.T) {
	logger := zap.NewNop().Sugar()
	svc := NewDashboardServiceWithDB(nil, nil, logger)
	require.NotNil(t, svc)
	assert.Same(t, logger, svc.logger)
}

func TestDashboardService_RepoIsFreshPerConstruction(t *testing.T) {
	db, _, _, _ := newRepoMockDB(t)

	svc1 := NewDashboardServiceWithDB(nil, db, zap.NewNop().Sugar())
	svc2 := NewDashboardServiceWithDB(nil, db, zap.NewNop().Sugar())

	require.NotNil(t, svc1.repo)
	require.NotNil(t, svc2.repo)
	assert.NotSame(t, svc1.repo, svc2.repo,
		"每次构造应生成新的 repo 实例（避免实例间状态泄漏）")
	assert.Same(t, svc1.repo.db, svc2.repo.db,
		"但 repo 持有的 db 必须同源，保证 tenant 注入稳定")
}
