package cloud

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent"
)

// mockAdapter implements CloudDiscoveryAdapter for testing
type mockAdapter struct {
	provider    string
	serviceCode string
	regionsErr  error
	initErr     error
}

func (m *mockAdapter) Provider() string    { return m.provider }
func (m *mockAdapter) ServiceCode() string { return m.serviceCode }
func (m *mockAdapter) InitClients(ctx context.Context, account *ent.CloudAccount, regions []string) (map[string]Client, error) {
	if m.initErr != nil {
		return nil, m.initErr
	}
	result := make(map[string]Client)
	for _, r := range regions {
		result[r] = &mockClient{}
	}
	return result, nil
}

func (m *mockAdapter) ListRegions(ctx context.Context, account *ent.CloudAccount) ([]string, error) {
	if m.regionsErr != nil {
		return nil, m.regionsErr
	}
	return []string{"region-a", "region-b"}, nil
}

func (m *mockAdapter) DiscoverRegion(ctx context.Context, account *ent.CloudAccount, region string, client Client, nextToken string) (*PageResult, error) {
	return &PageResult{
		Resources: []DiscoveredResource{
			{
				BaseResource: BaseResource{
					ResourceID:   "res-1",
					ResourceName: "TestResource",
					Region:       region,
					Status:       "running",
				},
				CloudServiceCode: m.serviceCode,
			},
		},
	}, nil
}
func (m *mockAdapter) Close() {}
func (m *mockAdapter) ValidateCredential(ctx context.Context, account *ent.CloudAccount) error {
	return nil
}

type mockClient struct{}

func (c *mockClient) Close() error { return nil }

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := &Registry{adapters: make(map[string]map[string]CloudDiscoveryAdapter)}

	mockA := &mockAdapter{provider: "aliyun", serviceCode: "ecs"}
	mockB := &mockAdapter{provider: "tencent", serviceCode: "cvm"}

	reg.Register(mockA)
	reg.Register(mockB)

	t.Run("get registered adapter", func(t *testing.T) {
		adapter, ok := reg.Get("aliyun", "ecs")
		require.True(t, ok)
		assert.Equal(t, "aliyun", adapter.Provider())
		assert.Equal(t, "ecs", adapter.ServiceCode())
	})

	t.Run("get second adapter", func(t *testing.T) {
		adapter, ok := reg.Get("tencent", "cvm")
		require.True(t, ok)
		assert.Equal(t, "tencent", adapter.Provider())
		assert.Equal(t, "cvm", adapter.ServiceCode())
	})

	t.Run("unknown provider returns false", func(t *testing.T) {
		_, ok := reg.Get("huawei", "ecs")
		assert.False(t, ok)
	})

	t.Run("unknown service returns false", func(t *testing.T) {
		_, ok := reg.Get("aliyun", "rds")
		assert.False(t, ok)
	})

	t.Run("override same key replaces adapter", func(t *testing.T) {
		mockA2 := &mockAdapter{provider: "aliyun", serviceCode: "ecs"}
		reg.Register(mockA2)
		adapter, _ := reg.Get("aliyun", "ecs")
		assert.Equal(t, "aliyun", adapter.Provider())
	})
}

func TestRegistry_GetByAccount(t *testing.T) {
	reg := &Registry{adapters: make(map[string]map[string]CloudDiscoveryAdapter)}
	reg.Register(&mockAdapter{provider: "aliyun", serviceCode: "ecs"})
	reg.Register(&mockAdapter{provider: "aliyun", serviceCode: "rds"})
	reg.Register(&mockAdapter{provider: "tencent", serviceCode: "cvm"})

	t.Run("aliyun account returns all aliyun adapters", func(t *testing.T) {
		adapters := reg.GetByAccount(&ent.CloudAccount{Provider: "aliyun"})
		assert.Len(t, adapters, 2)
		for _, a := range adapters {
			assert.Equal(t, "aliyun", a.Provider())
		}
	})

	t.Run("tencent account returns tencent adapters", func(t *testing.T) {
		adapters := reg.GetByAccount(&ent.CloudAccount{Provider: "tencent"})
		assert.Len(t, adapters, 1)
		assert.Equal(t, "tencent", adapters[0].Provider())
	})

	t.Run("normalized alias works", func(t *testing.T) {
		adapters := reg.GetByAccount(&ent.CloudAccount{Provider: "alibaba"})
		assert.Len(t, adapters, 2)
	})

	t.Run("unknown provider returns nil", func(t *testing.T) {
		adapters := reg.GetByAccount(&ent.CloudAccount{Provider: "oracle"})
		assert.Nil(t, adapters)
	})
}

func TestRegistry_RequireAdapter(t *testing.T) {
	reg := &Registry{adapters: make(map[string]map[string]CloudDiscoveryAdapter)}
	reg.Register(&mockAdapter{provider: "aliyun", serviceCode: "ecs"})

	t.Run("existing returns adapter", func(t *testing.T) {
		adapter, err := reg.RequireAdapter("aliyun", "ecs")
		require.NoError(t, err)
		assert.Equal(t, "aliyun", adapter.Provider())
	})

	t.Run("missing returns error", func(t *testing.T) {
		_, err := reg.RequireAdapter("tencent", "cvm")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no adapter registered")
	})
}

func TestNormalizeProvider(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"aliyun", "aliyun"},
		{"alibaba", "aliyun"},
		{"alicloud", "aliyun"},
		{"tencent", "tencent"},
		{"qcloud", "tencent"},
		{"tencentcloud", "tencent"},
		{"aws", "aws"},
		{"amazon", "aws"},
		{"azure", "azure"},
		{"huawei", "huawei"},
		{"onprem", "onprem"},
		{"private", "onprem"},
		{"private_cloud", "onprem"},
		{"PRIVATE_CLOUD", "onprem"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeProvider(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestProviderDisplayName(t *testing.T) {
	assert.Equal(t, "阿里云", ProviderDisplayName("aliyun"))
	assert.Equal(t, "腾讯云", ProviderDisplayName("tencent"))
	assert.Equal(t, "AWS", ProviderDisplayName("aws"))
	assert.Equal(t, "Azure", ProviderDisplayName("azure"))
	assert.Equal(t, "华为云", ProviderDisplayName("huawei"))
	assert.Equal(t, "私有云", ProviderDisplayName("onprem"))
}

func TestGlobalRegistry(t *testing.T) {
	reg := GlobalRegistry()
	require.NotNil(t, reg)
	assert.NotNil(t, reg)
}

func TestRegistryHasAdapterNormalizesProviderAndService(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockAdapter{provider: "aliyun", serviceCode: "ecs"})

	assert.True(t, registry.HasAdapter("alicloud", " ECS "))
	assert.False(t, registry.HasAdapter("aliyun", "rds"))
	var nilRegistry *Registry
	assert.False(t, nilRegistry.HasAdapter("aliyun", "ecs"))
}

func TestMockAdapter(t *testing.T) {
	adapter := &mockAdapter{provider: "aws", serviceCode: "ec2"}
	ctx := context.Background()

	t.Run("Provider and ServiceCode", func(t *testing.T) {
		assert.Equal(t, "aws", adapter.Provider())
		assert.Equal(t, "ec2", adapter.ServiceCode())
	})

	t.Run("InitClients returns clients per region", func(t *testing.T) {
		clients, err := adapter.InitClients(ctx, nil, []string{"us-east-1", "us-west-2"})
		require.NoError(t, err)
		assert.Len(t, clients, 2)
	})

	t.Run("ListRegions returns configured regions", func(t *testing.T) {
		regions, err := adapter.ListRegions(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"region-a", "region-b"}, regions)
	})

	t.Run("DiscoverRegion returns resources", func(t *testing.T) {
		result, err := adapter.DiscoverRegion(ctx, nil, "region-a", &mockClient{}, "")
		require.NoError(t, err)
		require.Len(t, result.Resources, 1)
		assert.Equal(t, "res-1", result.Resources[0].ResourceID)
	})

	t.Run("ListRegions error propagation", func(t *testing.T) {
		adapter.regionsErr = errors.New("region list failed")
		_, err := adapter.ListRegions(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "region list failed")
		adapter.regionsErr = nil
	})

	t.Run("InitClients error propagation", func(t *testing.T) {
		adapter.initErr = errors.New("init failed")
		_, err := adapter.InitClients(ctx, nil, []string{"us-east-1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "init failed")
		adapter.initErr = nil
	})
}
