package cmdb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockRepository 是 Repository 接口的可配置 mock，
// 用于在不打通数据库的前提下驱动 Service 层单元测试。
//
// 每个测试可以预设函数字段来改变行为；未预设的方法会返回 nil/空 slice。
type mockRepository struct {
	// Cloud services
	createCloudServiceFn   func(ctx context.Context, cs *CloudService) (*CloudService, error)
	listCloudServicesFn    func(ctx context.Context, tenantID int, provider string) ([]*CloudService, error)
	getCloudServiceFn      func(ctx context.Context, tenantID int, id int) (*CloudService, error)
	updateCloudServiceFn   func(ctx context.Context, cs *CloudService) (*CloudService, error)
	deleteCloudServiceFn   func(ctx context.Context, id int, tenantID int) error

	// Cloud accounts
	createCloudAccountFn   func(ctx context.Context, ca *CloudAccount) (*CloudAccount, error)
	listCloudAccountsFn    func(ctx context.Context, tenantID int, provider string) ([]*CloudAccount, error)
	getCloudAccountFn      func(ctx context.Context, tenantID int, id int) (*CloudAccount, error)
	updateCloudAccountFn   func(ctx context.Context, ca *CloudAccount) (*CloudAccount, error)
	deleteCloudAccountFn   func(ctx context.Context, id int, tenantID int) error

	// Cloud resources
	listCloudResourcesFn   func(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error)
	getCloudResourceFn     func(ctx context.Context, tenantID int, id int) (*CloudResource, error)
	createCloudResourceFn  func(ctx context.Context, cr *CloudResource) (*CloudResource, error)
	updateCloudResourceFn  func(ctx context.Context, cr *CloudResource) (*CloudResource, error)
	deleteCloudResourceFn  func(ctx context.Context, id int, tenantID int) error
	listCIsForReconciliationFn func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error)
	getCIByCloudResourceRefIDFn func(ctx context.Context, tenantID int, cloudResourceRefID int) (*ConfigurationItem, error)

	// Discovery
	createDiscoverySourceFn func(ctx context.Context, ds *DiscoverySource) (*DiscoverySource, error)
	listDiscoverySourcesFn  func(ctx context.Context, tenantID int) ([]*DiscoverySource, error)
	createDiscoveryJobFn    func(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error)
	listDiscoveryResultsFn  func(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error)
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

func (m *mockRepository) DeleteCloudService(ctx context.Context, id int, tenantID int) error {
	if m.deleteCloudServiceFn != nil {
		return m.deleteCloudServiceFn(ctx, id, tenantID)
	}
	return nil
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

func (m *mockRepository) DeleteCloudAccount(ctx context.Context, id int, tenantID int) error {
	if m.deleteCloudAccountFn != nil {
		return m.deleteCloudAccountFn(ctx, id, tenantID)
	}
	return nil
}

func (m *mockRepository) ListCloudResources(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
	if m.listCloudResourcesFn != nil {
		return m.listCloudResourcesFn(ctx, tenantID, provider, serviceID, region)
	}
	return nil, nil
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

func (m *mockRepository) DeleteCloudResource(ctx context.Context, id int, tenantID int) error {
	if m.deleteCloudResourceFn != nil {
		return m.deleteCloudResourceFn(ctx, id, tenantID)
	}
	return nil
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

func (m *mockRepository) CreateDiscoveryJob(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error) {
	if m.createDiscoveryJobFn != nil {
		return m.createDiscoveryJobFn(ctx, job)
	}
	return job, nil
}

func (m *mockRepository) ListDiscoveryResults(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error) {
	if m.listDiscoveryResultsFn != nil {
		return m.listDiscoveryResultsFn(ctx, tenantID, jobID)
	}
	return nil, nil
}

// newTestService 构造一个挂载了 mock 仓库的 Service。
// 使用 zap.NewNop() 避免日志输出，避免污染测试输出。
func newTestService(repo Repository) *Service {
	return NewService(repo, zap.NewNop().Sugar())
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
			deleteCloudServiceFn: func(ctx context.Context, id int, tenantID int) error {
				gotID = id
				gotTenantID = tenantID
				return nil
			},
		}
		svc := newTestService(repo)
		err := svc.DeleteCloudService(context.Background(), 100, 1)
		require.NoError(t, err)
		require.Equal(t, 100, gotID)
		require.Equal(t, 1, gotTenantID)
	})

	t.Run("repo 错误透传", func(t *testing.T) {
		repoErr := errors.New("fk violation")
		repo := &mockRepository{
			deleteCloudServiceFn: func(ctx context.Context, id int, tenantID int) error {
				return repoErr
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
			deleteCloudAccountFn: func(ctx context.Context, id int, tenantID int) error {
				gotTenantID = tenantID
				return nil
			},
		}
		svc := newTestService(repo)
		require.NoError(t, svc.DeleteCloudAccount(context.Background(), 1, 7))
		require.Equal(t, 7, gotTenantID)
	})
}

func TestService_CloudResource_CRUD(t *testing.T) {
	t.Run("List 透传过滤参数", func(t *testing.T) {
		vars := struct {
			tenantID, providerLen, serviceID, regionLen int
		}{}
		repo := &mockRepository{
			listCloudResourcesFn: func(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
				vars.tenantID = tenantID
				vars.providerLen = len(provider)
				vars.serviceID = serviceID
				vars.regionLen = len(region)
				return []*CloudResource{{ID: 1}}, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.ListCloudResources(context.Background(), 5, "alibaba", 11, "cn-hangzhou")
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, 5, vars.tenantID)
		require.Equal(t, len("alibaba"), vars.providerLen)
		require.Equal(t, 11, vars.serviceID)
		require.Equal(t, len("cn-hangzhou"), vars.regionLen)
	})

	t.Run("Create 透传", func(t *testing.T) {
		repo := &mockRepository{
			createCloudResourceFn: func(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
				cr.ID = 99
				return cr, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.CreateCloudResource(context.Background(), &CloudResource{ResourceID: "i-abc"})
		require.NoError(t, err)
		require.Equal(t, 99, got.ID)
	})

	t.Run("Update 透传", func(t *testing.T) {
		var gotID int
		repo := &mockRepository{
			updateCloudResourceFn: func(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
				gotID = cr.ID
				return cr, nil
			},
		}
		svc := newTestService(repo)
		_, err := svc.UpdateCloudResource(context.Background(), &CloudResource{ID: 42})
		require.NoError(t, err)
		require.Equal(t, 42, gotID)
	})

	t.Run("Delete 错误透传", func(t *testing.T) {
		repoErr := errors.New("delete failed")
		repo := &mockRepository{
			deleteCloudResourceFn: func(ctx context.Context, id int, tenantID int) error {
				return repoErr
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
			listCloudResourcesFn: func(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
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
			listCloudResourcesFn: func(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
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
			listCloudResourcesFn: func(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
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
			listCloudResourcesFn: func(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
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
			listCloudResourcesFn: func(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
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
			listCloudResourcesFn: func(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
				return []*CloudResource{
					{ID: 500, TenantID: 1, FirstSeenAt: &now},
					{ID: 501, TenantID: 1, FirstSeenAt: &now},
				}, nil
			},
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return []*ConfigurationItem{
					{ID: 1, TenantID: 1, CloudResourceRefID: 500},                            // 绑定
					{ID: 2, TenantID: 1, CloudResourceRefID: 9999},                           // 孤儿
					{ID: 3, TenantID: 1, CloudResourceID: "i-lost"},                          // 未关联
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
			listCloudResourcesFn: func(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
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
		got, err := svc.CreateDiscoverySource(context.Background(), &DiscoverySource{SourceType: "alibaba-cloud"})
		require.NoError(t, err)
		require.Equal(t, "alibaba-cloud", got.SourceType)
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

	t.Run("CreateDiscoveryJob 透传", func(t *testing.T) {
		repo := &mockRepository{
			createDiscoveryJobFn: func(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error) {
				job.ID = 7
				return job, nil
			},
		}
		svc := newTestService(repo)
		got, err := svc.CreateDiscoveryJob(context.Background(), &DiscoveryJob{SourceID: "ds-1"})
		require.NoError(t, err)
		require.Equal(t, 7, got.ID)
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
