package cmdb

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
)

// 租户隔离回归：发现源查询必须严格按租户过滤，不再回退到空租户记录
func TestListDiscoverySources_CrossTenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_source_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	repo := NewEntRepository(client)

	tenantA, err := client.Tenant.Create().SetName("Tenant A").SetCode("ds-tenant-a").
		SetDomain("ds-a.example.com").SetStatus("active").Save(ctx)
	if err != nil {
		t.Fatalf("create tenant A: %v", err)
	}
	tenantB, err := client.Tenant.Create().SetName("Tenant B").SetCode("ds-tenant-b").
		SetDomain("ds-b.example.com").SetStatus("active").Save(ctx)
	if err != nil {
		t.Fatalf("create tenant B: %v", err)
	}

	if _, err := repo.CreateDiscoverySource(ctx, &DiscoverySource{
		ID:         "ds-a-1",
		Name:       "阿里云发现源A",
		SourceType: "cloud_api",
		Provider:   "alibaba",
		IsActive:   true,
		TenantID:   tenantA.ID,
	}); err != nil {
		t.Fatalf("create discovery source for tenant A: %v", err)
	}

	// tenant_id 现在为必填字段，缺失时创建必须失败
	if _, err := client.DiscoverySource.Create().
		SetID("ds-no-tenant").
		SetName("无租户发现源").
		SetSourceType("cloud_api").
		Save(ctx); err == nil {
		t.Fatal("creating discovery source without tenant_id should fail")
	}

	listA, err := repo.ListDiscoverySources(ctx, tenantA.ID)
	if err != nil {
		t.Fatalf("list tenant A discovery sources: %v", err)
	}
	if len(listA) != 1 {
		t.Fatalf("tenant A should see 1 discovery source, got %d", len(listA))
	}

	listB, err := repo.ListDiscoverySources(ctx, tenantB.ID)
	if err != nil {
		t.Fatalf("list tenant B discovery sources: %v", err)
	}
	if len(listB) != 0 {
		t.Fatalf("tenant B should see 0 discovery sources, got %d", len(listB))
	}
}
