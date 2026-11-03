package cmdb

import (
	"context"
	"fmt"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/service"

	"github.com/stretchr/testify/require"
)

func provisioningProjectionRequest() service.ProvisioningCIProjectionRequest {
	return service.ProvisioningCIProjectionRequest{
		TenantID:           7,
		ServiceRequestID:   321,
		ProvisioningTaskID: 55,
		ResourceType:       "ecs",
		Provider:           "aliyun",
		Resources: map[string]string{
			"provider":     "aliyun",
			"account_id":   "acc-1",
			"region_id":    "cn-hangzhou",
			"instance_id":  "i-abc",
			"instance_ids": "i-abc",
			"request_id":   "req-1",
			"trade_price":  "0.12",
		},
		Payload: map[string]any{
			"instanceName": "srv-web-01",
			"instanceType": "ecs.e-c1m1.large",
			"imageId":      "img-1",
		},
	}
}

func provisioningProjectorWith(repo Repository, writer DiscoveryCIWriter) *ProvisioningCIProjector {
	repo.(*mockRepository).getCITypeIDByNameFn = func(_ context.Context, tenantID int, name string) (int, error) {
		if tenantID != 7 || name != "cloud_vm" {
			return 0, &ent.NotFoundError{}
		}
		return 99, nil
	}
	return NewProvisioningCIProjector(repo, writer, nil)
}

func TestProvisioningCIProjectorCreatesAndMarksProvenance(t *testing.T) {
	repo := &mockRepository{}
	writer := &fakeCIWriter{}

	result := provisioningProjectorWith(repo, writer).ProjectProvisionedResource(context.Background(), provisioningProjectionRequest())

	require.Equal(t, service.ProvisioningCIProjectionCreated, result.Status)
	require.Equal(t, 1, result.CIID)
	req := writer.createReq
	require.NotNil(t, req)
	require.Equal(t, 99, req.CITypeID)
	require.Equal(t, "srv-web-01", req.Name)
	require.Equal(t, common.CIStatusActive, req.Status)
	require.Equal(t, CIProvisioningSourceValue, req.Source)
	require.Equal(t, "provisioning_task:55", req.DiscoverySource)
	require.Equal(t, "aliyun", req.CloudProvider)
	require.Equal(t, "acc-1", req.CloudAccountID)
	require.Equal(t, "cn-hangzhou", req.CloudRegion)
	require.Equal(t, "i-abc", req.CloudResourceID)
	require.Equal(t, "instance", req.CloudResourceType)
	require.Equal(t, CISyncStatusSuccess, req.CloudSyncStatus)
	require.Equal(t, "ecs.e-c1m1.large", req.Attributes["instanceType"])
	require.Equal(t, "req-1", req.CloudMetadata["requestId"])
	require.Zero(t, req.CloudResourceRefID, "交付链路没有 cloud_resources 行，不能挂不存在的 ref")
}

func TestProvisioningCIProjectorUpdatesLinkedCIInsteadOfDuplicating(t *testing.T) {
	repo := &mockRepository{}
	repo.findCIByCloudResourceIdentityFn = func(_ context.Context, tenantID int, provider, cloudResourceID string) (*ConfigurationItem, error) {
		require.Equal(t, 7, tenantID)
		require.Equal(t, "aliyun", provider)
		require.Equal(t, "i-abc", cloudResourceID)
		return &ConfigurationItem{ID: 22}, nil
	}
	writer := &fakeCIWriter{}

	result := provisioningProjectorWith(repo, writer).ProjectProvisionedResource(context.Background(), provisioningProjectionRequest())

	require.Equal(t, service.ProvisioningCIProjectionUpdated, result.Status)
	require.Equal(t, 22, result.CIID)
	require.Nil(t, writer.createReq, "已存在同一云实例时必须更新而不是重复建账")
	require.NotNil(t, writer.updateReq)
	require.Equal(t, CISyncStatusSuccess, writer.updateReq.CloudSyncStatus)
}

func TestProvisioningCIProjectorSkipsWhenProviderResultIsIncomplete(t *testing.T) {
	req := provisioningProjectionRequest()
	req.Resources = map[string]string{}

	result := provisioningProjectorWith(&mockRepository{}, &fakeCIWriter{}).
		ProjectProvisionedResource(context.Background(), req)

	require.Equal(t, service.ProvisioningCIProjectionSkipped, result.Status)
	require.Equal(t, "provider_result_missing_instance_id", result.Reason)
}

func TestProvisioningCIProjectorDefersWhenRequiredAttributesMissing(t *testing.T) {
	writer := &fakeCIWriter{createErr: fmt.Errorf("属性 %s 为必填项 (%w)", "负责人", service.ErrCIAttributeValidation)}

	result := provisioningProjectorWith(&mockRepository{}, writer).
		ProjectProvisionedResource(context.Background(), provisioningProjectionRequest())

	require.Equal(t, service.ProvisioningCIProjectionDeferred, result.Status)
	require.Zero(t, result.CIID)
	require.Contains(t, result.Reason, "必填")
}
