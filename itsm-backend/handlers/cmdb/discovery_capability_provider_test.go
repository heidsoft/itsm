package cmdb

import (
	"context"
	"testing"

	"itsm-backend/service/cloud"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 能力探测的厂商维度回归锁。
//
// 背景：适配器从阿里云一家扩到五家后，GetDiscoveryCapability/GetDiscoveryHealth
// 仍在问 HasAdapter("aliyun","ecs") 且只列 aliyun 账号——只配了腾讯云的租户会被判成
// disabled/unconfigured，而它的发现链路其实完全可用。能力状态问的是「装了哪个厂商」，
// 不是「装了阿里云」。

// scopedInspector 按 provider→serviceCode 声明装配情况，清单仍按 KnownProviders
// 全量输出：少一个厂商就多一条 NotImplemented，与真实注册表口径一致。
type scopedInspector struct{ codesByProvider map[string][]string }

func (i scopedInspector) HasAdapter(provider, serviceCode string) bool {
	for _, code := range i.codesByProvider[provider] {
		if code == serviceCode {
			return true
		}
	}
	return false
}

func (i scopedInspector) SupportedProviders() []cloud.ProviderSupport {
	out := make([]cloud.ProviderSupport, 0, len(cloud.KnownProviders))
	for _, p := range cloud.KnownProviders {
		codes := i.codesByProvider[p]
		if codes == nil {
			codes = []string{}
		}
		out = append(out, cloud.ProviderSupport{
			Provider:       p,
			DisplayName:    cloud.ProviderDisplayName(p),
			ServiceCodes:   codes,
			NotImplemented: len(codes) == 0,
		})
	}
	return out
}

func newCapabilityService(repo Repository, codesByProvider map[string][]string) *Service {
	return NewServiceWithDiscoveryRuntime(repo, nil, zap.NewNop().Sugar(), DiscoveryRuntime{
		Adapters:                scopedInspector{codesByProvider: codesByProvider},
		CredentialResolverReady: true,
		WorkerReady:             true,
	})
}

func TestDiscoveryCapabilityAcceptsAnySupportedProvider(t *testing.T) {
	var queriedProvider string
	repo := &mockRepository{listCloudAccountsFn: func(_ context.Context, tenantID int, provider string) ([]*CloudAccount, error) {
		queriedProvider = provider
		return []*CloudAccount{{
			TenantID: tenantID, Provider: "tencent", AccountID: "acc-tencent",
			IsActive: true, CredentialRef: "secret://tenant-7/tencent-prod",
		}}, nil
	}}

	status, err := newCapabilityService(repo, map[string][]string{
		"aliyun": {"ecs"}, "tencent": {"cvm"},
	}).GetDiscoveryCapability(context.Background(), 7)

	require.NoError(t, err)
	require.Equal(t, "ready", status.State, "只配了腾讯云的租户不该被判成不可用")
	require.True(t, status.TenantReadiness)
	require.Empty(t, status.MissingRequirements)
	// 账号清单必须跨厂商取，按单一厂商过滤就是本次缺陷的另一半。
	require.Equal(t, "", queriedProvider)
}

func TestDiscoveryCapabilityRejectsAccountForUnsupportedProvider(t *testing.T) {
	repo := &mockRepository{listCloudAccountsFn: func(_ context.Context, tenantID int, _ string) ([]*CloudAccount, error) {
		return []*CloudAccount{{
			TenantID: tenantID, Provider: "huawei", IsActive: true, CredentialRef: "secret://tenant-7/huawei",
		}}, nil
	}}

	status, err := newCapabilityService(repo, map[string][]string{"aliyun": {"ecs"}}).
		GetDiscoveryCapability(context.Background(), 7)

	require.NoError(t, err)
	// 华为云在 KnownProviders 里但没装适配器：配了账号也不会自动发现，
	// 不能因为「租户有账号」就报 ready。
	require.Equal(t, "unconfigured", status.State)
	require.False(t, status.TenantReadiness)
	require.Equal(t, []string{"cloudAccountWithDiscoveryAdapter"}, status.MissingRequirements)
}

func TestDiscoveryCapabilityDisabledWhenNoAdapterInstalled(t *testing.T) {
	repo := &mockRepository{listCloudAccountsFn: func(_ context.Context, tenantID int, _ string) ([]*CloudAccount, error) {
		return []*CloudAccount{{TenantID: tenantID, Provider: "aliyun", IsActive: true, CredentialRef: "secret://t/a"}}, nil
	}}

	status, err := newCapabilityService(repo, nil).GetDiscoveryCapability(context.Background(), 7)

	require.NoError(t, err)
	require.Equal(t, "disabled", status.State)
	require.False(t, status.DeploymentReadiness)
	require.Equal(t, []string{"discoveryAdapter", "cloudAccountWithDiscoveryAdapter"}, status.MissingRequirements)
}

func TestDiscoveryHealthNeedsNoTenantAccount(t *testing.T) {
	// 健康端点是部署级事实，不查租户账号；只看注册表装了谁。
	status := newCapabilityService(&mockRepository{}, map[string][]string{"aws": {"ec2"}}).GetDiscoveryHealth()

	require.Equal(t, "ready", status.State)
	require.True(t, status.DeploymentReadiness)
	require.Empty(t, status.MissingRequirements)

	unready := newCapabilityService(&mockRepository{}, nil).GetDiscoveryHealth()
	require.Equal(t, "unready", unready.State)
	require.Equal(t, []string{"discoveryAdapter"}, unready.MissingRequirements)
}
