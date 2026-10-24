package cmdb

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/internal/commandbus"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockRepository 是 Repository 接口的可配置 mock，
// 用于在不打通数据库的前提下驱动 Service 层单元测试。
//
// 每个测试可以预设函数字段来改变行为；未预设的方法会返回 nil/空 slice。
type mockRepository struct {
	// Cloud services
	createCloudServiceFn func(ctx context.Context, cs *CloudService) (*CloudService, error)
	listCloudServicesFn  func(ctx context.Context, tenantID int, provider string) ([]*CloudService, error)
	getCloudServiceFn    func(ctx context.Context, tenantID int, id int) (*CloudService, error)
	updateCloudServiceFn func(ctx context.Context, cs *CloudService) (*CloudService, error)
	deleteCloudServiceFn func(ctx context.Context, id int, tenantID int) (int, error)

	// Cloud accounts
	createCloudAccountFn func(ctx context.Context, ca *CloudAccount) (*CloudAccount, error)
	listCloudAccountsFn  func(ctx context.Context, tenantID int, provider string) ([]*CloudAccount, error)
	getCloudAccountFn    func(ctx context.Context, tenantID int, id int) (*CloudAccount, error)
	updateCloudAccountFn func(ctx context.Context, ca *CloudAccount) (*CloudAccount, error)
	deleteCloudAccountFn func(ctx context.Context, id int, tenantID int) (int, error)

	// Cloud resources
	listCloudResourcesFn        func(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error)
	listCloudResourcesPageFn    func(ctx context.Context, tenantID int, filter CloudResourceFilter, page int, pageSize int) ([]*CloudResource, int, error)
	getCloudResourceFn          func(ctx context.Context, tenantID int, id int) (*CloudResource, error)
	createCloudResourceFn       func(ctx context.Context, cr *CloudResource) (*CloudResource, error)
	updateCloudResourceFn       func(ctx context.Context, cr *CloudResource) (*CloudResource, error)
	deleteCloudResourceFn       func(ctx context.Context, id int, tenantID int) (int, error)
	listCIsForReconciliationFn  func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error)
	getCIByCloudResourceRefIDFn func(ctx context.Context, tenantID int, cloudResourceRefID int) (*ConfigurationItem, error)

	// Discovery
	createDiscoverySourceFn func(ctx context.Context, ds *DiscoverySource) (*DiscoverySource, error)
	listDiscoverySourcesFn  func(ctx context.Context, tenantID int) ([]*DiscoverySource, error)
	createDiscoveryJobFn    func(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error)
	// 调度器扫描面：默认「没有配置调度的来源」/「没有历史任务」，
	// 需要真实行为的用例走 sqlite 端到端（discovery_scheduler_test.go）。
	listScheduledDiscoverySourcesFn        func(ctx context.Context, tenantID int) ([]*DiscoverySource, error)
	latestDiscoveryJobForSourceFn          func(ctx context.Context, tenantID int, sourceID string) (*DiscoveryJob, error)
	latestScheduledDiscoveryJobForSourceFn func(ctx context.Context, tenantID int, sourceID string) (*DiscoveryJob, error)
	getDiscoveryResultFn                   func(ctx context.Context, tenantID, id int) (*DiscoveryResult, error)
	updateDiscoveryResultFn                func(ctx context.Context, res *DiscoveryResult) error
	transitionDiscoveryResultFn            func(ctx context.Context, tenantID, id int, fromStatus, toStatus, errorCode, errorMessage string) (int, error)
	listDiscoveryResultsFn                 func(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error)

	// Discovery job 执行链路
	getDiscoverySourceFn                 func(ctx context.Context, tenantID int, id string) (*DiscoverySource, error)
	createDiscoveryJobWithCommandFn      func(ctx context.Context, job *DiscoveryJob, commandType string, payload map[string]interface{}) (*DiscoveryJob, error)
	getDiscoveryJobFn                    func(ctx context.Context, tenantID, id int) (*DiscoveryJob, error)
	findDiscoveryJobByIdempotencyKeyFn   func(ctx context.Context, tenantID int, sourceID, key string) (*DiscoveryJob, error)
	listDiscoveryJobsFn                  func(ctx context.Context, tenantID int, status string, page, pageSize int) ([]*DiscoveryJob, int, error)
	updateDiscoveryJobFn                 func(ctx context.Context, job *DiscoveryJob) error
	createDiscoveryResultFn              func(ctx context.Context, res *DiscoveryResult) (*DiscoveryResult, error)
	getCloudResourceByIdentityFn         func(ctx context.Context, tenantID int, identityHash string) (*CloudResource, error)
	listCloudResourcesByAccountServiceFn func(ctx context.Context, tenantID, cloudAccountID int, serviceCode string) ([]*CloudResource, error)
	updateDiscoveredCloudResourceFn      func(ctx context.Context, cr *CloudResource) error
}

// 编译期断言：mockRepository 必须实现 Repository 接口
var _ Repository = (*mockRepository)(nil)

func (m *mockRepository) CreateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error) {
	if m.createCloudServiceFn != nil {
		return m.createCloudServiceFn(ctx, cs)
	}
	return cs, nil
}

func (m *mockRepository) ListCloudServices(ctx context.Context, tenantID int, provider string) ([]*CloudService, error) {
	if m.listCloudServicesFn != nil {
		return m.listCloudServicesFn(ctx, tenantID, provider)
	}
	return nil, nil
}

func (m *mockRepository) GetCloudService(ctx context.Context, tenantID int, id int) (*CloudService, error) {
	if m.getCloudServiceFn != nil {
		return m.getCloudServiceFn(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *mockRepository) UpdateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error) {
	if m.updateCloudServiceFn != nil {
		return m.updateCloudServiceFn(ctx, cs)
	}
	return cs, nil
}

func (m *mockRepository) DeleteCloudService(ctx context.Context, id int, tenantID int) (int, error) {
	if m.deleteCloudServiceFn != nil {
		return m.deleteCloudServiceFn(ctx, id, tenantID)
	}
	// 默认视为命中一行，否则「删除成功」路径会被 service 当成 not found。
	return 1, nil
}

func (m *mockRepository) CreateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error) {
	if m.createCloudAccountFn != nil {
		return m.createCloudAccountFn(ctx, ca)
	}
	return ca, nil
}

func (m *mockRepository) ListCloudAccounts(ctx context.Context, tenantID int, provider string) ([]*CloudAccount, error) {
	if m.listCloudAccountsFn != nil {
		return m.listCloudAccountsFn(ctx, tenantID, provider)
	}
	return nil, nil
}

func (m *mockRepository) GetCloudAccount(ctx context.Context, tenantID int, id int) (*CloudAccount, error) {
	if m.getCloudAccountFn != nil {
		return m.getCloudAccountFn(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *mockRepository) UpdateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error) {
	if m.updateCloudAccountFn != nil {
		return m.updateCloudAccountFn(ctx, ca)
	}
	return ca, nil
}

func (m *mockRepository) DeleteCloudAccount(ctx context.Context, id int, tenantID int) (int, error) {
	if m.deleteCloudAccountFn != nil {
		return m.deleteCloudAccountFn(ctx, id, tenantID)
	}
	return 1, nil
}

func (m *mockRepository) ListCloudResources(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
	if m.listCloudResourcesFn != nil {
		return m.listCloudResourcesFn(ctx, tenantID, filter)
	}
	return nil, nil
}

func (m *mockRepository) ListCloudResourcesPage(ctx context.Context, tenantID int, filter CloudResourceFilter, page int, pageSize int) ([]*CloudResource, int, error) {
	if m.listCloudResourcesPageFn != nil {
		return m.listCloudResourcesPageFn(ctx, tenantID, filter, page, pageSize)
	}
	return nil, 0, nil
}

func (m *mockRepository) GetCloudResource(ctx context.Context, tenantID int, id int) (*CloudResource, error) {
	if m.getCloudResourceFn != nil {
		return m.getCloudResourceFn(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *mockRepository) CreateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
	if m.createCloudResourceFn != nil {
		return m.createCloudResourceFn(ctx, cr)
	}
	return cr, nil
}

func (m *mockRepository) UpdateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
	if m.updateCloudResourceFn != nil {
		return m.updateCloudResourceFn(ctx, cr)
	}
	return cr, nil
}

func (m *mockRepository) DeleteCloudResource(ctx context.Context, id int, tenantID int) (int, error) {
	if m.deleteCloudResourceFn != nil {
		return m.deleteCloudResourceFn(ctx, id, tenantID)
	}
	return 1, nil
}

func (m *mockRepository) ListCIsForReconciliation(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
	if m.listCIsForReconciliationFn != nil {
		return m.listCIsForReconciliationFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockRepository) GetCIByCloudResourceRefID(ctx context.Context, tenantID int, cloudResourceRefID int) (*ConfigurationItem, error) {
	if m.getCIByCloudResourceRefIDFn != nil {
		return m.getCIByCloudResourceRefIDFn(ctx, tenantID, cloudResourceRefID)
	}
	return nil, nil
}

func (m *mockRepository) CreateDiscoverySource(ctx context.Context, ds *DiscoverySource) (*DiscoverySource, error) {
	if m.createDiscoverySourceFn != nil {
		return m.createDiscoverySourceFn(ctx, ds)
	}
	return ds, nil
}

func (m *mockRepository) ListDiscoverySources(ctx context.Context, tenantID int) ([]*DiscoverySource, error) {
	if m.listDiscoverySourcesFn != nil {
		return m.listDiscoverySourcesFn(ctx, tenantID)
	}
	return nil, nil
}

// 调度器扫描入口：默认返回空集（未配置 schedule），需要时由用例注入。
func (m *mockRepository) ListScheduledDiscoverySources(ctx context.Context, tenantID int) ([]*DiscoverySource, error) {
	if m.listScheduledDiscoverySourcesFn != nil {
		return m.listScheduledDiscoverySourcesFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockRepository) LatestDiscoveryJobForSource(ctx context.Context, tenantID int, sourceID string) (*DiscoveryJob, error) {
	if m.latestDiscoveryJobForSourceFn != nil {
		return m.latestDiscoveryJobForSourceFn(ctx, tenantID, sourceID)
	}
	return nil, &ent.NotFoundError{}
}

func (m *mockRepository) LatestScheduledDiscoveryJobForSource(ctx context.Context, tenantID int, sourceID string) (*DiscoveryJob, error) {
	if m.latestScheduledDiscoveryJobForSourceFn != nil {
		return m.latestScheduledDiscoveryJobForSourceFn(ctx, tenantID, sourceID)
	}
	return nil, &ent.NotFoundError{}
}

func (m *mockRepository) CreateDiscoveryJob(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error) {
	if m.createDiscoveryJobFn != nil {
		return m.createDiscoveryJobFn(ctx, job)
	}
	return job, nil
}

// 差异决策读写面：默认「没有这条差异」，需要真实行为的用例走 sqlite 端到端。
func (m *mockRepository) GetDiscoveryResult(ctx context.Context, tenantID, id int) (*DiscoveryResult, error) {
	if m.getDiscoveryResultFn != nil {
		return m.getDiscoveryResultFn(ctx, tenantID, id)
	}
	return nil, &ent.NotFoundError{}
}

func (m *mockRepository) UpdateDiscoveryResult(ctx context.Context, res *DiscoveryResult) error {
	if m.updateDiscoveryResultFn != nil {
		return m.updateDiscoveryResultFn(ctx, res)
	}
	return nil
}

func (m *mockRepository) TransitionDiscoveryResult(ctx context.Context, tenantID, id int, fromStatus, toStatus, errorCode, errorMessage string) (int, error) {
	if m.transitionDiscoveryResultFn != nil {
		return m.transitionDiscoveryResultFn(ctx, tenantID, id, fromStatus, toStatus, errorCode, errorMessage)
	}
	return 1, nil
}

func (m *mockRepository) ListDiscoveryResults(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error) {
	if m.listDiscoveryResultsFn != nil {
		return m.listDiscoveryResultsFn(ctx, tenantID, jobID)
	}
	return nil, nil
}

// ---- Discovery job 执行链路新增的接口方法（默认返回零值/NotFound，测试按需注入 fn）----

func (m *mockRepository) GetDiscoverySource(ctx context.Context, tenantID int, id string) (*DiscoverySource, error) {
	if m.getDiscoverySourceFn != nil {
		return m.getDiscoverySourceFn(ctx, tenantID, id)
	}
	return nil, &ent.NotFoundError{}
}

func (m *mockRepository) TouchDiscoverySourceSuccess(ctx context.Context, tenantID int, id string, at time.Time) error {
	return nil
}

func (m *mockRepository) CreateDiscoveryJobWithCommand(ctx context.Context, job *DiscoveryJob, commandType string, payload map[string]interface{}) (*DiscoveryJob, error) {
	if m.createDiscoveryJobWithCommandFn != nil {
		return m.createDiscoveryJobWithCommandFn(ctx, job, commandType, payload)
	}
	job.ID = 1
	return job, nil
}

func (m *mockRepository) GetDiscoveryJob(ctx context.Context, tenantID, id int) (*DiscoveryJob, error) {
	if m.getDiscoveryJobFn != nil {
		return m.getDiscoveryJobFn(ctx, tenantID, id)
	}
	return nil, &ent.NotFoundError{}
}

func (m *mockRepository) FindDiscoveryJobByIdempotencyKey(ctx context.Context, tenantID int, sourceID, key string) (*DiscoveryJob, error) {
	if m.findDiscoveryJobByIdempotencyKeyFn != nil {
		return m.findDiscoveryJobByIdempotencyKeyFn(ctx, tenantID, sourceID, key)
	}
	return nil, &ent.NotFoundError{}
}

func (m *mockRepository) ListDiscoveryJobs(ctx context.Context, tenantID int, status string, page, pageSize int) ([]*DiscoveryJob, int, error) {
	if m.listDiscoveryJobsFn != nil {
		return m.listDiscoveryJobsFn(ctx, tenantID, status, page, pageSize)
	}
	return nil, 0, nil
}

func (m *mockRepository) UpdateDiscoveryJob(ctx context.Context, job *DiscoveryJob) error {
	if m.updateDiscoveryJobFn != nil {
		return m.updateDiscoveryJobFn(ctx, job)
	}
	return nil
}

func (m *mockRepository) CreateDiscoveryResult(ctx context.Context, res *DiscoveryResult) (*DiscoveryResult, error) {
	if m.createDiscoveryResultFn != nil {
		return m.createDiscoveryResultFn(ctx, res)
	}
	res.ID = 1
	return res, nil
}

func (m *mockRepository) GetCloudResourceByIdentity(ctx context.Context, tenantID int, identityHash string) (*CloudResource, error) {
	if m.getCloudResourceByIdentityFn != nil {
		return m.getCloudResourceByIdentityFn(ctx, tenantID, identityHash)
	}
	return nil, &ent.NotFoundError{}
}

func (m *mockRepository) ListCloudResourcesByAccountService(ctx context.Context, tenantID, cloudAccountID int, serviceCode string) ([]*CloudResource, error) {
	if m.listCloudResourcesByAccountServiceFn != nil {
		return m.listCloudResourcesByAccountServiceFn(ctx, tenantID, cloudAccountID, serviceCode)
	}
	return nil, nil
}

func (m *mockRepository) UpdateDiscoveredCloudResource(ctx context.Context, cr *CloudResource) error {
	if m.updateDiscoveredCloudResourceFn != nil {
		return m.updateDiscoveredCloudResourceFn(ctx, cr)
	}
	return nil
}

// newTestService 构造一个挂载了 mock 仓库的 Service。
// 使用 zap.NewNop() 避免日志输出，避免污染测试输出。
func newTestService(repo Repository) *Service {
	return NewService(repo, nil, zap.NewNop().Sugar())
}

func TestService_CreateCloudService(t *testing.T) {
	t.Run("成功路径透传 repo 返回", func(t *testing.T) {
		want := &CloudService{ID: 7, Provider: "alibaba", ServiceCode: "ecs", TenantID: 1}
		repo := &mockRepository{
			createCloudServiceFn: func(ctx context.Context, cs *CloudService) (*CloudService, error) {
				require.Equal(t, "alibaba", cs.Provider)
				return want, nil
			},
		}
		svc := newTestService(repo)

		got, err := svc.CreateCloudService(context.Background(), &CloudService{Provider: "alibaba", ServiceCode: "ecs"})
		require.NoError(t, err)
		require.Same(t, want, got)
	})

	t.Run("repo 错误透传", func(t *testing.T) {
		repoErr := errors.New("db down")
		repo := &mockRepository{
			createCloudServiceFn: func(ctx context.Context, cs *CloudService) (*CloudService, error) {
				return nil, repoErr
			},
		}
		svc := newTestService(repo)

		got, err := svc.CreateCloudService(context.Background(), &CloudService{Provider: "alibaba"})
		require.ErrorIs(t, err, repoErr)
		require.Nil(t, got)
	})
}

func TestService_ListCloudServices(t *testing.T) {
	t.Run("返回命中数量用于日志", func(t *testing.T) {
		repo := &mockRepository{
			listCloudServicesFn: func(ctx context.Context, tenantID int, provider string) ([]*CloudService, error) {
				return []*CloudService{{ID: 1}, {ID: 2}, {ID: 3}}, nil
			},
		}
		svc := newTestService(repo)

		got, err := svc.ListCloudServices(context.Background(), 1, "alibaba")
		require.NoError(t, err)
		require.Len(t, got, 3)
	})

	t.Run("按 provider 过滤透传", func(t *testing.T) {
		var capturedProvider string
		repo := &mockRepository{
			listCloudServicesFn: func(ctx context.Context, tenantID int, provider string) ([]*CloudService, error) {
				capturedProvider = provider
				return nil, nil
			},
		}
		svc := newTestService(repo)
		_, err := svc.ListCloudServices(context.Background(), 1, "tencent")
		require.NoError(t, err)
		require.Equal(t, "tencent", capturedProvider)
	})

	t.Run("repo 错误透传", func(t *testing.T) {
		repoErr := errors.New("list failed")
		repo := &mockRepository{
			listCloudServicesFn: func(ctx context.Context, tenantID int, provider string) ([]*CloudService, error) {
				return nil, repoErr
			},
		}
		svc := newTestService(repo)
		_, err := svc.ListCloudServices(context.Background(), 1, "")
		require.ErrorIs(t, err, repoErr)
	})
}

func TestService_GetCloudService(t *testing.T) {
	t.Run("成功按 tenantID + id 透传", func(t *testing.T) {
		want := &CloudService{ID: 11, Provider: "huaweicloud", TenantID: 5}
		var gotTenantID, gotID int
		repo := &mockRepository{
			getCloudServiceFn: func(ctx context.Context, tenantID int, id int) (*CloudService, error) {
				gotTenantID = tenantID
				gotID = id
				return want, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.GetCloudService(context.Background(), 5, 11)
		require.NoError(t, err)
		require.Same(t, want, got)
		require.Equal(t, 5, gotTenantID)
		require.Equal(t, 11, gotID)
	})

	t.Run("repo 错误透传", func(t *testing.T) {
		repoErr := errors.New("not found")
		repo := &mockRepository{
			getCloudServiceFn: func(ctx context.Context, tenantID int, id int) (*CloudService, error) {
				return nil, repoErr
			},
		}
		svc := newTestService(repo)
		_, err := svc.GetCloudService(context.Background(), 1, 42)
		require.ErrorIs(t, err, repoErr)
	})
}

func TestService_UpdateCloudService(t *testing.T) {
	t.Run("更新透传", func(t *testing.T) {
		in := &CloudService{ID: 1, Provider: "alibaba", TenantID: 2}
		repo := &mockRepository{
			updateCloudServiceFn: func(ctx context.Context, cs *CloudService) (*CloudService, error) {
				require.Equal(t, 1, cs.ID)
				return cs, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.UpdateCloudService(context.Background(), in)
		require.NoError(t, err)
		require.Equal(t, "alibaba", got.Provider)
	})

	t.Run("repo 错误透传", func(t *testing.T) {
		repoErr := errors.New("update failed")
		repo := &mockRepository{
			updateCloudServiceFn: func(ctx context.Context, cs *CloudService) (*CloudService, error) {
				return nil, repoErr
			},
		}
		svc := newTestService(repo)
		_, err := svc.UpdateCloudService(context.Background(), &CloudService{ID: 1})
		require.ErrorIs(t, err, repoErr)
	})
}

func TestService_DeleteCloudService(t *testing.T) {
	t.Run("成功按 id+tenantID 删除", func(t *testing.T) {
		var gotID, gotTenantID int
		repo := &mockRepository{
			deleteCloudServiceFn: func(ctx context.Context, id int, tenantID int) (int, error) {
				gotID = id
				gotTenantID = tenantID
				return 1, nil
			},
		}
		svc := newTestService(repo)
		err := svc.DeleteCloudService(context.Background(), 100, 1)
		require.NoError(t, err)
		require.Equal(t, 100, gotID)
		require.Equal(t, 1, gotTenantID)
	})

	t.Run("0 行受影响必须是 not found 而不是删除成功", func(t *testing.T) {
		// DELETE 必须看受影响行数：租户 B 删租户 A 的 ID 时匹配 0 行，
		// 若把 0 行当成成功，调用方以为对象已消失，实际它还在租户 A 名下。
		repo := &mockRepository{
			deleteCloudServiceFn: func(ctx context.Context, id int, tenantID int) (int, error) {
				return 0, nil
			},
		}
		svc := newTestService(repo)
		err := svc.DeleteCloudService(context.Background(), 999, 2)
		require.Error(t, err)
		var appErr *common.AppError
		require.True(t, errors.As(err, &appErr), "必须映射成业务错误而不是 500: %v", err)
		require.Equal(t, common.ErrCodeNotFound, appErr.Code)
	})

	t.Run("repo 错误透传", func(t *testing.T) {
		repoErr := errors.New("fk violation")
		repo := &mockRepository{
			deleteCloudServiceFn: func(ctx context.Context, id int, tenantID int) (int, error) {
				return 0, repoErr
			},
		}
		svc := newTestService(repo)
		err := svc.DeleteCloudService(context.Background(), 1, 1)
		require.ErrorIs(t, err, repoErr)
	})
}

func TestService_CloudAccount_CRUD(t *testing.T) {
	t.Run("Create 透传", func(t *testing.T) {
		want := &CloudAccount{ID: 5, Provider: "alibaba", AccountID: "act-x", TenantID: 1}
		repo := &mockRepository{
			createCloudAccountFn: func(ctx context.Context, ca *CloudAccount) (*CloudAccount, error) {
				return want, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.CreateCloudAccount(context.Background(), &CloudAccount{Provider: "alibaba"})
		require.NoError(t, err)
		require.Same(t, want, got)
	})

	t.Run("List 透传", func(t *testing.T) {
		repo := &mockRepository{
			listCloudAccountsFn: func(ctx context.Context, tenantID int, provider string) ([]*CloudAccount, error) {
				return []*CloudAccount{{ID: 1}, {ID: 2}}, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.ListCloudAccounts(context.Background(), 1, "tencent")
		require.NoError(t, err)
		require.Len(t, got, 2)
	})

	t.Run("Get 透传", func(t *testing.T) {
		want := &CloudAccount{ID: 9, Provider: "huaweicloud", AccountID: "act-9"}
		repo := &mockRepository{
			getCloudAccountFn: func(ctx context.Context, tenantID int, id int) (*CloudAccount, error) {
				return want, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.GetCloudAccount(context.Background(), 1, 9)
		require.NoError(t, err)
		require.Equal(t, "act-9", got.AccountID)
	})

	t.Run("Update 透传+错误", func(t *testing.T) {
		repoErr := errors.New("update failed")
		repo := &mockRepository{
			updateCloudAccountFn: func(ctx context.Context, ca *CloudAccount) (*CloudAccount, error) {
				return nil, repoErr
			},
		}
		svc := newTestService(repo)
		_, err := svc.UpdateCloudAccount(context.Background(), &CloudAccount{ID: 1})
		require.ErrorIs(t, err, repoErr)
	})

	t.Run("Delete 按 tenantID 隔离", func(t *testing.T) {
		var gotTenantID int
		repo := &mockRepository{
			deleteCloudAccountFn: func(ctx context.Context, id int, tenantID int) (int, error) {
				gotTenantID = tenantID
				return 1, nil
			},
		}
		svc := newTestService(repo)
		require.NoError(t, svc.DeleteCloudAccount(context.Background(), 1, 7))
		require.Equal(t, 7, gotTenantID)
	})
}

func TestService_CloudResource_CRUD(t *testing.T) {
	t.Run("List 透传过滤参数与页长", func(t *testing.T) {
		vars := struct {
			tenantID, serviceID, page, pageSize int
			provider, region, status, search    string
			total                               int
		}{}
		repo := &mockRepository{
			listCloudResourcesPageFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter, page int, pageSize int) ([]*CloudResource, int, error) {
				vars.tenantID = tenantID
				vars.provider = filter.Provider
				vars.serviceID = filter.ServiceID
				vars.region = filter.Region
				vars.status = filter.Status
				vars.search = filter.Search
				vars.page = page
				vars.pageSize = pageSize
				return []*CloudResource{{ID: 1}}, 7, nil
			},
		}
		svc := newTestService(repo)
		got, total, err := svc.ListCloudResources(context.Background(), 5, CloudResourceFilter{
			Provider: "aliyun", ServiceID: 11, Region: "cn-hangzhou", Status: "running", Search: "i-",
		}, 2, 20)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, 7, total)
		require.Equal(t, 5, vars.tenantID)
		require.Equal(t, "aliyun", vars.provider)
		require.Equal(t, 11, vars.serviceID)
		require.Equal(t, "cn-hangzhou", vars.region)
		require.Equal(t, "running", vars.status)
		require.Equal(t, "i-", vars.search)
		require.Equal(t, 2, vars.page)
		require.Equal(t, 20, vars.pageSize)
	})

	t.Run("非 HTTP 调用方的越界页长被兜住", func(t *testing.T) {
		var gotPage, gotPageSize int
		repo := &mockRepository{
			listCloudResourcesPageFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter, page int, pageSize int) ([]*CloudResource, int, error) {
				gotPage, gotPageSize = page, pageSize
				return nil, 0, nil
			},
		}
		svc := newTestService(repo)
		_, _, err := svc.ListCloudResources(context.Background(), 5, CloudResourceFilter{}, 0, 5000)
		require.NoError(t, err)
		require.Equal(t, 1, gotPage)
		require.Equal(t, 100, gotPageSize)
	})

	t.Run("Create 生成稳定身份", func(t *testing.T) {
		repo := &mockRepository{
			getCloudAccountFn: func(context.Context, int, int) (*CloudAccount, error) {
				return &CloudAccount{ID: 7, TenantID: 5, Provider: "Aliyun", AccountID: "123456"}, nil
			},
			getCloudServiceFn: func(context.Context, int, int) (*CloudService, error) {
				return &CloudService{ID: 11, TenantID: 5, Provider: "aliyun", ServiceCode: "ECS", ResourceTypeCode: "Instance"}, nil
			},
			createCloudResourceFn: func(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
				cr.ID = 99
				return cr, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.CreateCloudResource(context.Background(), &CloudResource{
			TenantID: 5, CloudAccountID: 7, ServiceID: 11, ResourceID: " i-abc ", Region: "CN-Hangzhou",
		})
		require.NoError(t, err)
		require.Equal(t, 99, got.ID)
		require.Equal(t, "123456", got.CanonicalAccountID)
		require.Equal(t, "cn-hangzhou", got.Region)
		require.Equal(t, "regional", got.ResourceScope)
		require.Len(t, got.IdentityHash, 64)
	})

	t.Run("Update 重新计算稳定身份", func(t *testing.T) {
		var gotID int
		repo := &mockRepository{
			getCloudAccountFn: func(context.Context, int, int) (*CloudAccount, error) {
				return &CloudAccount{ID: 7, TenantID: 5, Provider: "aliyun", AccountID: "123456"}, nil
			},
			getCloudServiceFn: func(context.Context, int, int) (*CloudService, error) {
				return &CloudService{ID: 11, TenantID: 5, Provider: "aliyun", ServiceCode: "ecs", ResourceTypeCode: "instance"}, nil
			},
			updateCloudResourceFn: func(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
				gotID = cr.ID
				return cr, nil
			},
		}
		svc := newTestService(repo)
		_, err := svc.UpdateCloudResource(context.Background(), &CloudResource{
			ID: 42, TenantID: 5, CloudAccountID: 7, ServiceID: 11, ResourceID: "i-abc",
		})
		require.NoError(t, err)
		require.Equal(t, 42, gotID)
	})

	t.Run("Create 拒绝跨租户引用", func(t *testing.T) {
		repo := &mockRepository{
			getCloudAccountFn: func(context.Context, int, int) (*CloudAccount, error) {
				return &CloudAccount{ID: 7, TenantID: 6, Provider: "aliyun", AccountID: "other"}, nil
			},
			getCloudServiceFn: func(context.Context, int, int) (*CloudService, error) {
				return &CloudService{ID: 11, TenantID: 5, Provider: "aliyun"}, nil
			},
		}
		_, err := newTestService(repo).CreateCloudResource(context.Background(), &CloudResource{
			TenantID: 5, CloudAccountID: 7, ServiceID: 11, ResourceID: "i-cross-tenant",
		})
		require.ErrorContains(t, err, "authenticated tenant")
	})

	t.Run("Delete 错误透传", func(t *testing.T) {
		repoErr := errors.New("delete failed")
		repo := &mockRepository{
			deleteCloudResourceFn: func(ctx context.Context, id int, tenantID int) (int, error) {
				return 0, repoErr
			},
		}
		svc := newTestService(repo)
		err := svc.DeleteCloudResource(context.Background(), 1, 1)
		require.ErrorIs(t, err, repoErr)
	})
}

func TestService_GetReconciliation(t *testing.T) {
	t.Run("空数据", func(t *testing.T) {
		repo := &mockRepository{
			listCloudResourcesFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
				return nil, nil
			},
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return nil, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.GetReconciliation(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, 0, got.Summary.ResourceTotal)
		require.Equal(t, 0, got.Summary.BoundResourceCount)
		require.Equal(t, 0, got.Summary.UnboundResourceCount)
		require.Equal(t, 0, got.Summary.OrphanCICount)
		require.Equal(t, 0, got.Summary.UnlinkedCICount)
		require.Empty(t, got.UnboundResources)
		require.Empty(t, got.OrphanCIs)
		require.Empty(t, got.UnlinkedCIs)
	})

	t.Run("全部已绑定", func(t *testing.T) {
		repo := &mockRepository{
			listCloudResourcesFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
				return []*CloudResource{
					{ID: 100, TenantID: 1},
					{ID: 101, TenantID: 1},
				}, nil
			},
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return []*ConfigurationItem{
					{ID: 1, TenantID: 1, CloudResourceRefID: 100},
					{ID: 2, TenantID: 1, CloudResourceRefID: 101},
				}, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.GetReconciliation(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, 2, got.Summary.ResourceTotal)
		require.Equal(t, 2, got.Summary.BoundResourceCount)
		require.Equal(t, 0, got.Summary.UnboundResourceCount)
		require.Equal(t, 0, got.Summary.OrphanCICount)
		require.Equal(t, 0, got.Summary.UnlinkedCICount)
	})

	t.Run("unboundResources: CI 未引用的资源", func(t *testing.T) {
		repo := &mockRepository{
			listCloudResourcesFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
				return []*CloudResource{
					{ID: 200, TenantID: 1},
					{ID: 201, TenantID: 1},
				}, nil
			},
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return []*ConfigurationItem{
					{ID: 1, TenantID: 1, CloudResourceRefID: 200},
				}, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.GetReconciliation(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, 2, got.Summary.ResourceTotal)
		require.Equal(t, 1, got.Summary.BoundResourceCount)
		require.Equal(t, 1, got.Summary.UnboundResourceCount)
		require.Len(t, got.UnboundResources, 1)
		require.Equal(t, 201, got.UnboundResources[0].ID)
	})

	t.Run("orphanCIs: CI 引用了不存在的资源", func(t *testing.T) {
		repo := &mockRepository{
			listCloudResourcesFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
				return []*CloudResource{{ID: 300, TenantID: 1}}, nil
			},
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return []*ConfigurationItem{
					{ID: 1, TenantID: 1, CloudResourceRefID: 300},
					{ID: 2, TenantID: 1, CloudResourceRefID: 999}, // 引用不存在
				}, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.GetReconciliation(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, 1, got.Summary.BoundResourceCount)
		require.Equal(t, 1, got.Summary.OrphanCICount)
		require.Len(t, got.OrphanCIs, 1)
		require.Equal(t, 2, got.OrphanCIs[0].ID)
	})

	t.Run("unlinkedCIs: CI 有 CloudResourceID 但未关联 CloudResourceRefID", func(t *testing.T) {
		repo := &mockRepository{
			listCloudResourcesFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
				return nil, nil
			},
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return []*ConfigurationItem{
					{ID: 1, TenantID: 1, CloudResourceID: "i-orphan"},
				}, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.GetReconciliation(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, 1, got.Summary.UnlinkedCICount)
		require.Len(t, got.UnlinkedCIs, 1)
		require.Equal(t, "i-orphan", got.UnlinkedCIs[0].CloudResourceID)
	})

	t.Run("混合场景", func(t *testing.T) {
		now := time.Now()
		repo := &mockRepository{
			listCloudResourcesFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
				return []*CloudResource{
					{ID: 500, TenantID: 1, FirstSeenAt: &now},
					{ID: 501, TenantID: 1, FirstSeenAt: &now},
				}, nil
			},
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return []*ConfigurationItem{
					{ID: 1, TenantID: 1, CloudResourceRefID: 500},   // 绑定
					{ID: 2, TenantID: 1, CloudResourceRefID: 9999},  // 孤儿
					{ID: 3, TenantID: 1, CloudResourceID: "i-lost"}, // 未关联
				}, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.GetReconciliation(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, 2, got.Summary.ResourceTotal)
		require.Equal(t, 1, got.Summary.BoundResourceCount)
		require.Equal(t, 1, got.Summary.UnboundResourceCount)
		require.Equal(t, 1, got.Summary.OrphanCICount)
		require.Equal(t, 1, got.Summary.UnlinkedCICount)
		// 501 没被任何 CI 引用 → unbound
		require.Len(t, got.UnboundResources, 1)
		require.Equal(t, 501, got.UnboundResources[0].ID)
	})

	t.Run("ListCloudResources 错误透传", func(t *testing.T) {
		repoErr := errors.New("resources query failed")
		repo := &mockRepository{
			listCloudResourcesFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
				return nil, repoErr
			},
		}
		svc := newTestService(repo)
		_, err := svc.GetReconciliation(context.Background(), 1)
		require.ErrorIs(t, err, repoErr)
	})

	t.Run("ListCIsForReconciliation 错误透传", func(t *testing.T) {
		repoErr := errors.New("cis query failed")
		repo := &mockRepository{
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return nil, repoErr
			},
		}
		svc := newTestService(repo)
		_, err := svc.GetReconciliation(context.Background(), 1)
		require.ErrorIs(t, err, repoErr)
	})
}

func TestService_Discovery(t *testing.T) {
	t.Run("CreateDiscoverySource 透传", func(t *testing.T) {
		repo := &mockRepository{
			createDiscoverySourceFn: func(ctx context.Context, ds *DiscoverySource) (*DiscoverySource, error) {
				require.Equal(t, "alibaba-cloud", ds.SourceType)
				return ds, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.CreateDiscoverySource(context.Background(), &DiscoverySource{SourceType: "alibaba-cloud", TenantID: 1})
		require.NoError(t, err)
		require.Equal(t, "alibaba-cloud", got.SourceType)
		require.Equal(t, "manual", got.ReconcilePolicy)
		require.Equal(t, 3, got.StaleThreshold)
	})

	t.Run("ListDiscoverySources 透传", func(t *testing.T) {
		repo := &mockRepository{
			listDiscoverySourcesFn: func(ctx context.Context, tenantID int) ([]*DiscoverySource, error) {
				return []*DiscoverySource{{ID: "ds-1"}, {ID: "ds-2"}}, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.ListDiscoverySources(context.Background(), 1)
		require.NoError(t, err)
		require.Len(t, got, 2)
	})

	t.Run("CreateDiscoveryJob 校验来源与入队", func(t *testing.T) {
		t.Run("来源不存在返回 404", func(t *testing.T) {
			svc := newTestService(&mockRepository{})
			_, err := svc.CreateDiscoveryJob(context.Background(), 1, 9, &dto.DiscoveryJobRequest{SourceID: "ds-x"})
			require.Error(t, err)
			appErr, ok := err.(*common.AppError)
			require.True(t, ok)
			require.Equal(t, common.ErrCodeNotFound, appErr.Code)
			require.Equal(t, http.StatusNotFound, appErr.HTTPStatus)
		})

		t.Run("有效来源落库并入队命令", func(t *testing.T) {
			var capturedType string
			var capturedJob *DiscoveryJob
			repo := &mockRepository{
				getDiscoverySourceFn: func(ctx context.Context, tenantID int, id string) (*DiscoverySource, error) {
					return &DiscoverySource{ID: "ds-1", IsActive: true, Provider: "aliyun", CloudAccountID: 5, ServiceCodes: []string{"ecs"}, Regions: []string{"cn-hangzhou"}}, nil
				},
				getCloudAccountFn: func(ctx context.Context, tenantID, id int) (*CloudAccount, error) {
					return &CloudAccount{ID: 5, IsActive: true, CredentialRef: "env://X"}, nil
				},
				createDiscoveryJobWithCommandFn: func(ctx context.Context, job *DiscoveryJob, commandType string, payload map[string]interface{}) (*DiscoveryJob, error) {
					capturedType = commandType
					capturedJob = job
					job.ID = 7
					return job, nil
				},
			}
			svc := newTestService(repo)
			got, err := svc.CreateDiscoveryJob(context.Background(), 1, 9, &dto.DiscoveryJobRequest{SourceID: "ds-1"})
			require.NoError(t, err)
			require.Equal(t, 7, got.ID)
			require.Equal(t, DiscoveryJobQueued, got.Status)
			require.Equal(t, commandbus.CommandRunCMDBCloudDiscovery, capturedType)
			require.Equal(t, DiscoveryJobQueued, capturedJob.Status)
			require.Equal(t, "ds-1", capturedJob.SourceID)
			require.NotNil(t, capturedJob.QueuedAt)
		})
	})

	t.Run("ListDiscoveryResults 透传", func(t *testing.T) {
		var capturedJobID int
		repo := &mockRepository{
			listDiscoveryResultsFn: func(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error) {
				capturedJobID = jobID
				return []*DiscoveryResult{{ID: 1, JobID: jobID}}, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.ListDiscoveryResults(context.Background(), 1, 99)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, 99, capturedJobID)
	})
}
