package cmdb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service"

	"github.com/stretchr/testify/require"
)

// fakeCIWriter 记录投影器传给 CI 域的请求，用来断言「发现拥有哪些字段」。
type fakeCIWriter struct {
	createReq *dto.CreateCIRequest
	updateReq *dto.UpdateCIRequest
	createErr error
	updateErr error
	nextID    int
}

func (f *fakeCIWriter) CreateCI(_ context.Context, req *dto.CreateCIRequest, _ int) (*dto.CIResponse, error) {
	f.createReq = req
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.nextID++
	return &dto.CIResponse{ID: f.nextID, Name: req.Name}, nil
}

func (f *fakeCIWriter) UpdateCI(_ context.Context, id, _ int, req *dto.UpdateCIRequest) (*dto.CIResponse, error) {
	f.updateReq = req
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return &dto.CIResponse{ID: id, Name: req.Name}, nil
}

func projectedResource() *CloudResource {
	seen := time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
	return &CloudResource{
		ID: 41, TenantID: 7, Provider: "aliyun", ServiceCode: "ecs", ResourceType: "instance",
		ResourceID: "i-abc", ResourceName: "web-01", CanonicalAccountID: "acc-1",
		Region: "cn-hangzhou", Zone: "cn-hangzhou-g", Status: "active",
		Tags:       map[string]string{"env": "prod", "Owner": " ops-team ", "cost-center": "platform"},
		SourceID:   "ds-1",
		LastSeenAt: &seen,
	}
}

func projectorWith(repo Repository, writer DiscoveryCIWriter) *DiscoveryCIProjector {
	repo.(*mockRepository).getCITypeIDByNameFn = func(_ context.Context, tenantID int, name string) (int, error) {
		if tenantID != 7 || name != "cloud_vm" {
			return 0, &ent.NotFoundError{}
		}
		return 99, nil
	}
	return NewDiscoveryCIProjector(repo, writer, nil, nopLogger())
}

func TestDiscoveryCIProjectorCreatesFromTags(t *testing.T) {
	repo := &mockRepository{}
	writer := &fakeCIWriter{}
	res := projectedResource()

	outcome := projectorWith(repo, writer).Project(context.Background(), res)

	require.Equal(t, CIProjectionCreated, outcome.Status)
	require.Equal(t, 1, outcome.CIID)
	req := writer.createReq
	require.NotNil(t, req)
	require.Equal(t, 99, req.CITypeID)
	require.Equal(t, "web-01", req.Name)
	require.Equal(t, common.CIStatusActive, req.Status)
	require.Equal(t, CIDiscoverySourceValue, req.Source)
	require.Equal(t, "ds-1", req.DiscoverySource)
	require.Equal(t, "aliyun", req.CloudProvider)
	require.Equal(t, "acc-1", req.CloudAccountID)
	require.Equal(t, "i-abc", req.CloudResourceID)
	require.Equal(t, "instance", req.CloudResourceType)
	require.Equal(t, CISyncStatusSuccess, req.CloudSyncStatus)
	require.Equal(t, res.ID, req.CloudResourceRefID)
	require.NotNil(t, req.LastDiscovered)
	require.Equal(t, res.LastSeenAt, req.LastDiscovered)
	// 标签推断：环境/负责人只在无歧义时写入，其它标签原样进动态属性。
	require.Equal(t, "production", req.Environment)
	require.Equal(t, "ops-team", req.AssignedTo)
	require.Equal(t, "platform", req.Attributes["cost-center"])
}

func TestDiscoveryCIProjectorDefersMissingRequiredAttributes(t *testing.T) {
	repo := &mockRepository{}
	writer := &fakeCIWriter{createErr: fmt.Errorf("属性 %s 为必填项 (%w)", "负责人", service.ErrCIAttributeValidation)}

	outcome := projectorWith(repo, writer).Project(context.Background(), projectedResource())

	require.Equal(t, CIProjectionDeferred, outcome.Status)
	require.Zero(t, outcome.CIID)
	require.Contains(t, outcome.Reason, "必填")
}

func TestDiscoveryCIProjectorSkipsUnmappedAndUnconfigured(t *testing.T) {
	t.Run("unmapped resource type", func(t *testing.T) {
		repo := &mockRepository{}
		writer := &fakeCIWriter{}
		res := projectedResource()
		res.ServiceCode = "oss"
		res.ResourceType = "bucket"

		outcome := projectorWith(repo, writer).Project(context.Background(), res)

		require.Equal(t, CIProjectionSkipped, outcome.Status)
		require.Contains(t, outcome.Reason, "unmapped_resource_type=oss|bucket")
		require.Nil(t, writer.createReq, "未映射的资源不能凭空建配置项")
	})

	t.Run("ci type missing in tenant", func(t *testing.T) {
		repo := &mockRepository{}
		writer := &fakeCIWriter{}
		p := NewDiscoveryCIProjector(repo, writer, nil, nopLogger())

		outcome := p.Project(context.Background(), projectedResource())

		require.Equal(t, CIProjectionSkipped, outcome.Status)
		require.Contains(t, outcome.Reason, "ci_type_missing=cloud_vm")
		require.Nil(t, writer.createReq)
	})

	t.Run("projector not wired", func(t *testing.T) {
		var p *DiscoveryCIProjector
		require.Equal(t, CIProjectionUnavailable, p.Project(context.Background(), projectedResource()).Status)
	})
}

func TestDiscoveryCIProjectorUpdatesLinkedCIWithoutHumanFields(t *testing.T) {
	repo := &mockRepository{
		getCIByCloudResourceRefIDFn: func(_ context.Context, tenantID, refID int) (*ConfigurationItem, error) {
			require.Equal(t, 7, tenantID)
			require.Equal(t, 41, refID)
			return &ConfigurationItem{ID: 300, CloudSyncStatus: CISyncStatusStale}, nil
		},
	}
	writer := &fakeCIWriter{}

	outcome := projectorWith(repo, writer).Project(context.Background(), projectedResource())

	require.Equal(t, CIProjectionUpdated, outcome.Status)
	require.Equal(t, 300, outcome.CIID)
	req := writer.updateReq
	require.NotNil(t, req)
	// 人工/导入填的动态属性、环境、负责人不归发现管。
	require.Nil(t, req.Attributes)
	require.Empty(t, req.Environment)
	require.Empty(t, req.AssignedTo)
	require.Empty(t, req.CITypeID)
	// 同步状态从 stale 回到 success，否则资源回来了账还是滞后。
	require.Equal(t, CISyncStatusSuccess, req.CloudSyncStatus)
}

func TestDiscoveryCIProjectorLinksManualCIByCloudIdentity(t *testing.T) {
	// 没有 ref 链接（人工登记的账），按 (provider, 云上资源ID) 命中后要补上 ref，
	// 而不是再建一条重复配置项。
	repo := &mockRepository{
		findCIByCloudResourceIdentityFn: func(_ context.Context, tenantID int, provider, cloudResourceID string) (*ConfigurationItem, error) {
			require.Equal(t, 7, tenantID)
			require.Equal(t, "aliyun", provider)
			require.Equal(t, "i-abc", cloudResourceID)
			return &ConfigurationItem{ID: 500}, nil
		},
	}
	writer := &fakeCIWriter{}

	outcome := projectorWith(repo, writer).Project(context.Background(), projectedResource())

	require.Equal(t, CIProjectionUpdated, outcome.Status)
	require.Equal(t, 500, outcome.CIID)
	require.Equal(t, 41, writer.updateReq.CloudResourceRefID)
	require.Nil(t, writer.createReq)
}

func TestDiscoveryCIProjectorMarkStale(t *testing.T) {
	staleRepo := func(ci *ConfigurationItem) *mockRepository {
		return &mockRepository{
			getCIByCloudResourceRefIDFn: func(_ context.Context, _ int, _ int) (*ConfigurationItem, error) {
				return ci, nil
			},
		}
	}

	t.Run("linked ci gets stale sync status", func(t *testing.T) {
		writer := &fakeCIWriter{}
		outcome := NewDiscoveryCIProjector(staleRepo(&ConfigurationItem{ID: 300}), writer, nil, nopLogger()).
			MarkStale(context.Background(), projectedResource())

		require.Equal(t, CIProjectionUpdated, outcome.Status)
		require.Equal(t, CIProjectionReasonMarkedStale, outcome.Reason)
		require.Equal(t, 300, outcome.CIID)
		require.Equal(t, CISyncStatusStale, writer.updateReq.CloudSyncStatus)
		require.Empty(t, writer.updateReq.LifecycleStatus, "退役是人的决定，回写只标同步滞后")
		require.Zero(t, writer.updateReq.CloudResourceRefID)
	})

	t.Run("already stale writes nothing", func(t *testing.T) {
		writer := &fakeCIWriter{}
		outcome := NewDiscoveryCIProjector(staleRepo(&ConfigurationItem{ID: 300, CloudSyncStatus: CISyncStatusStale}), writer, nil, nopLogger()).
			MarkStale(context.Background(), projectedResource())

		require.Equal(t, CIProjectionSkipped, outcome.Status)
		require.Equal(t, CIProjectionReasonAlreadyStale, outcome.Reason)
		require.Nil(t, writer.updateReq)
	})

	t.Run("without a linked ci there is nothing to mark", func(t *testing.T) {
		writer := &fakeCIWriter{}
		repo := &mockRepository{}
		outcome := NewDiscoveryCIProjector(repo, writer, nil, nopLogger()).
			MarkStale(context.Background(), projectedResource())

		require.Equal(t, CIProjectionSkipped, outcome.Status)
		require.Equal(t, CIProjectionReasonNoLinkedCI, outcome.Reason)
		require.Nil(t, writer.updateReq)
	})
}

func TestCiStatusFromCloudStatus(t *testing.T) {
	// pending（创建中）不能算成在提供服务。
	require.Equal(t, "active", ciStatusFromCloudStatus("active"))
	require.Equal(t, "inactive", ciStatusFromCloudStatus("pending"))
	require.Equal(t, "inactive", ciStatusFromCloudStatus("inactive"))
	require.Equal(t, "retired", ciStatusFromCloudStatus("retired"))
}

func TestCiEnvironmentFromTagsOnlyUnambiguous(t *testing.T) {
	require.Equal(t, "staging", mustEnv(t, map[string]string{"ENV": "Stage"}))
	require.Equal(t, "development", mustEnv(t, map[string]string{"environment": "Development"}))
	if _, ok := ciEnvironmentFromTags(map[string]string{"env": "pre-prod"}); ok {
		t.Fatal("歧义环境值不能猜测")
	}
	if _, ok := ciEnvironmentFromTags(map[string]string{"env": ""}); ok {
		t.Fatal("空标签值不能算环境")
	}
}

func mustEnv(t *testing.T, tags map[string]string) string {
	t.Helper()
	value, ok := ciEnvironmentFromTags(tags)
	require.True(t, ok)
	return value
}
