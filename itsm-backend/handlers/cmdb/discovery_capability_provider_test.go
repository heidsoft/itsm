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

// enableDiscoveryReadiness 打开部署级开关。
//
// 域内能力面与能力矩阵共用同一个 fail-closed 开关（ITSM_CMDB_DISCOVERY_READY）。
// 本文件聚焦「厂商维度」与「租户配置」判定，先把开关打开，让用例断言不被
// 部署级默认值干扰；开关自身的行为由 TestDiscoveryCapabilityHonoursReadinessEnv
// 与 TestDiscoveryHealthNeedsNoTenantAccount 的未启用分支分别锁定。
func enableDiscoveryReadiness(t *testing.T) {
	t.Helper()
	t.Setenv(discoveryReadinessEnv, "true")
}

func TestDiscoveryCapabilityAcceptsAnySupportedProvider(t *testing.T) {
	enableDiscoveryReadiness(t)

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
	enableDiscoveryReadiness(t)

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
	enableDiscoveryReadiness(t)

	repo := &mockRepository{listCloudAccountsFn: func(_ context.Context, tenantID int, _ string) ([]*CloudAccount, error) {
		return []*CloudAccount{{TenantID: tenantID, Provider: "aliyun", IsActive: true, CredentialRef: "secret://t/a"}}, nil
	}}

	status, err := newCapabilityService(repo, nil).GetDiscoveryCapability(context.Background(), 7)

	require.NoError(t, err)
	require.Equal(t, "disabled", status.State)
	require.False(t, status.DeploymentReadiness)
	require.Equal(t, []string{"discoveryAdapter", "cloudAccountWithDiscoveryAdapter"}, status.MissingRequirements)
}

// 域内能力面与能力矩阵必须对同一个开关得出同一结论。
//
// 背景：矩阵读 readinessEnv 报 deploymentReady=false，域内面此前完全不读，
// 于是 prod 上出现「矩阵说没就绪、注册页说只缺云账号」。这里把三种情形钉死：
// 未设置 / 显式关闭 / 显式开启，并要求缺失项里点名那个开关，
// 让运维能直接看出「不是账号没配，是开关没开」。
func TestDiscoveryCapabilityHonoursReadinessEnv(t *testing.T) {
	deploymentAssemblyComplete := func() Repository {
		return &mockRepository{listCloudAccountsFn: func(_ context.Context, tenantID int, _ string) ([]*CloudAccount, error) {
			return []*CloudAccount{{
				TenantID: tenantID, Provider: "aliyun", IsActive: true, CredentialRef: "secret://t/a",
			}}, nil
		}}
	}

	for _, tc := range []struct {
		name        string
		envValue    string
		wantReady   bool
		wantState   string
		wantMissing []string
	}{
		// fail-closed：不显式申报就是没就绪
		{"未设置开关", "", false, "unready", []string{discoveryReadinessEnv}},
		{"显式关闭", "false", false, "unready", []string{discoveryReadinessEnv}},
		{"显式开启", "true", true, "ready", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// 空串等价于「未设置」：EnvEnabled 对空值同样判 false
			t.Setenv(discoveryReadinessEnv, tc.envValue)

			status, err := newCapabilityService(deploymentAssemblyComplete(), map[string][]string{"aliyun": {"ecs"}}).
				GetDiscoveryCapability(context.Background(), 7)

			require.NoError(t, err)
			require.Equal(t, tc.wantReady, status.DeploymentReadiness, "部署就绪必须跟随开关")
			require.Equal(t, tc.wantState, status.State)
			if len(tc.wantMissing) > 0 {
				require.Equal(t, tc.wantMissing, status.MissingRequirements,
					"未开启时必须在缺失项里点名该开关，否则运维只会去配云账号")
			} else {
				require.Empty(t, status.MissingRequirements)
			}
			// 开关不影响租户维度：租户有合规账号就应报就绪
			require.True(t, status.TenantReadiness)
		})
	}
}

func TestDiscoveryHealthNeedsNoTenantAccount(t *testing.T) {
	// 健康端点是部署级事实，不查租户账号；只看注册表装了谁 + 部署级开关。
	enableDiscoveryReadiness(t)
	status := newCapabilityService(&mockRepository{}, map[string][]string{"aws": {"ec2"}}).GetDiscoveryHealth()

	require.Equal(t, "ready", status.State)
	require.True(t, status.DeploymentReadiness)
	require.Empty(t, status.MissingRequirements)

	// 开关关闭时即便适配器装齐也不得报就绪——健康端点与 /capabilities 同口径
	t.Setenv(discoveryReadinessEnv, "false")
	unready := newCapabilityService(&mockRepository{}, map[string][]string{"aws": {"ec2"}}).GetDiscoveryHealth()
	require.Equal(t, "unready", unready.State)
	require.False(t, unready.DeploymentReadiness)
	require.Equal(t, []string{discoveryReadinessEnv}, unready.MissingRequirements)

	// 开关打开但没装任何适配器：仍是不就绪，缺失项要能同时反映两件事
	t.Setenv(discoveryReadinessEnv, "true")
	noAdapter := newCapabilityService(&mockRepository{}, nil).GetDiscoveryHealth()
	require.Equal(t, "unready", noAdapter.State)
	require.Equal(t, []string{"discoveryAdapter"}, noAdapter.MissingRequirements)
}
