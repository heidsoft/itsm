package service

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
)

// createCMDBTestTenant 创建测试租户（避免与 rag_service_test 中的同名函数冲突，使用独立名称）
func createCMDBTestTenant(ctx context.Context, client *ent.Client, name, code, domain string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName(name).
		SetCode(code).
		SetDomain(domain).
		SetStatus("active").
		Save(ctx)
}

// createTestCIType 创建测试 CI 类型
func createTestCIType(ctx context.Context, client *ent.Client, tenantID int, name string) (*ent.CIType, error) {
	return client.CIType.Create().
		SetName(name).
		SetTenantID(tenantID).
		Save(ctx)
}

// createTestCI 创建测试配置项
func createTestCI(ctx context.Context, client *ent.Client, tenantID, ciTypeID int, name string) (*ent.ConfigurationItem, error) {
	return client.ConfigurationItem.Create().
		SetName(name).
		SetCiType("server").
		SetCiTypeID(ciTypeID).
		SetStatus("active").
		SetEnvironment("production").
		SetCriticality("high").
		SetTenantID(tenantID).
		Save(ctx)
}

// TestListRelationships_TenantIsolation 测试跨租户查询返回空结果
func TestListRelationships_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_ent1?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()

	// 创建两个租户
	tenant1, err := createCMDBTestTenant(ctx, client, "Tenant 1", "t1", "t1.com")
	if err != nil {
		t.Fatalf("创建租户1失败: %v", err)
	}
	tenant2, err := createCMDBTestTenant(ctx, client, "Tenant 2", "t2", "t2.com")
	if err != nil {
		t.Fatalf("创建租户2失败: %v", err)
	}

	// 创建 CI 类型
	ciType1, err := createTestCIType(ctx, client, tenant1.ID, "server-type-1")
	if err != nil {
		t.Fatalf("创建CI类型1失败: %v", err)
	}
	ciType2, err := createTestCIType(ctx, client, tenant2.ID, "server-type-2")
	if err != nil {
		t.Fatalf("创建CI类型2失败: %v", err)
	}

	// 创建两个租户的配置项
	ci1, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "server-1")
	if err != nil {
		t.Fatalf("创建CI1失败: %v", err)
	}
	ci2, err := createTestCI(ctx, client, tenant2.ID, ciType2.ID, "server-2")
	if err != nil {
		t.Fatalf("创建CI2失败: %v", err)
	}

	// 创建租户1的关系
	client.CIRelationship.Create().
		SetRelationshipType("depends_on").
		SetSourceCiID(ci1.ID).
		SetTargetCiID(ci1.ID).
		SetTenantID(tenant1.ID).
		SaveX(ctx)

	// 创建租户2的关系
	client.CIRelationship.Create().
		SetRelationshipType("depends_on").
		SetSourceCiID(ci2.ID).
		SetTargetCiID(ci2.ID).
		SetTenantID(tenant2.ID).
		SaveX(ctx)

	svc := NewCMDBService(client)

	// 租户1查询应该只返回租户1的关系
	rels1, err := svc.ListRelationships(ctx, tenant1.ID, nil)
	if err != nil {
		t.Fatalf("ListRelationships 失败: %v", err)
	}
	for _, rel := range rels1 {
		if rel.TenantID != tenant1.ID {
			t.Errorf("跨租户数据泄露: 期望 tenant_id=%d, 实际 tenant_id=%d", tenant1.ID, rel.TenantID)
		}
	}

	// 租户2查询应该只返回租户2的关系
	rels2, err := svc.ListRelationships(ctx, tenant2.ID, nil)
	if err != nil {
		t.Fatalf("ListRelationships 失败: %v", err)
	}
	for _, rel := range rels2 {
		if rel.TenantID != tenant2.ID {
			t.Errorf("跨租户数据泄露: 期望 tenant_id=%d, 实际 tenant_id=%d", tenant2.ID, rel.TenantID)
		}
	}
}

// TestListRelationships_SameTenant 测试同租户查询返回正确结果
func TestListRelationships_SameTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_ent2?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()

	tenant1, err := createCMDBTestTenant(ctx, client, "Tenant 1", "t1", "t1.com")
	if err != nil {
		t.Fatalf("创建租户失败: %v", err)
	}
	ciType1, err := createTestCIType(ctx, client, tenant1.ID, "server-type")
	if err != nil {
		t.Fatalf("创建CI类型失败: %v", err)
	}

	ci1, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "app-1")
	if err != nil {
		t.Fatalf("创建CI1失败: %v", err)
	}
	ci2, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "server-1")
	if err != nil {
		t.Fatalf("创建CI2失败: %v", err)
	}

	client.CIRelationship.Create().
		SetRelationshipType("runs_on").
		SetSourceCiID(ci1.ID).
		SetTargetCiID(ci2.ID).
		SetTenantID(tenant1.ID).
		SaveX(ctx)

	svc := NewCMDBService(client)

	rels, err := svc.ListRelationships(ctx, tenant1.ID, nil)
	if err != nil {
		t.Fatalf("ListRelationships 失败: %v", err)
	}
	if len(rels) != 1 {
		t.Errorf("期望 1 条关系, 实际 %d 条", len(rels))
	}
}

// TestGetCITopology_TenantIsolation 测试拓扑查询的租户隔离
func TestGetCITopology_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_ent3?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()

	tenant1, err := createCMDBTestTenant(ctx, client, "Tenant 1", "t1", "t1.com")
	if err != nil {
		t.Fatalf("创建租户1失败: %v", err)
	}
	tenant2, err := createCMDBTestTenant(ctx, client, "Tenant 2", "t2", "t2.com")
	if err != nil {
		t.Fatalf("创建租户2失败: %v", err)
	}

	ciType1, err := createTestCIType(ctx, client, tenant1.ID, "app-type")
	if err != nil {
		t.Fatalf("创建CI类型失败: %v", err)
	}
	ciType2, err := createTestCIType(ctx, client, tenant2.ID, "server-type")
	if err != nil {
		t.Fatalf("创建CI类型2失败: %v", err)
	}

	ci1, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "app-1")
	if err != nil {
		t.Fatalf("创建CI1失败: %v", err)
	}
	ci2, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "server-1")
	if err != nil {
		t.Fatalf("创建CI2失败: %v", err)
	}
	ci3, err := createTestCI(ctx, client, tenant2.ID, ciType2.ID, "server-2")
	if err != nil {
		t.Fatalf("创建CI3失败: %v", err)
	}

	client.CIRelationship.Create().
		SetRelationshipType("runs_on").
		SetSourceCiID(ci1.ID).
		SetTargetCiID(ci2.ID).
		SetTenantID(tenant1.ID).
		SaveX(ctx)

	client.CIRelationship.Create().
		SetRelationshipType("depends_on").
		SetSourceCiID(ci3.ID).
		SetTargetCiID(ci3.ID).
		SetTenantID(tenant2.ID).
		SaveX(ctx)

	svc := NewCMDBService(client)

	topo, err := svc.GetCITopology(ctx, ci1.ID, tenant1.ID, 1)
	if err != nil {
		t.Fatalf("GetCITopology 失败: %v", err)
	}
	if topo.CI == nil {
		t.Fatal("拓扑根节点不应为空")
	}
	for _, child := range topo.Children {
		if child.CI.TenantID != tenant1.ID {
			t.Errorf("拓扑中发现跨租户CI: tenant_id=%d", child.CI.TenantID)
		}
	}
}

// TestCreateRelationship_SetsTenantID 测试创建关系时设置 tenant_id
func TestCreateRelationship_SetsTenantID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_ent4?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()

	tenant1, err := createCMDBTestTenant(ctx, client, "Tenant 1", "t1", "t1.com")
	if err != nil {
		t.Fatalf("创建租户失败: %v", err)
	}
	ciType1, err := createTestCIType(ctx, client, tenant1.ID, "server-type")
	if err != nil {
		t.Fatalf("创建CI类型失败: %v", err)
	}

	ci1, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "app-1")
	if err != nil {
		t.Fatalf("创建CI1失败: %v", err)
	}
	ci2, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "server-1")
	if err != nil {
		t.Fatalf("创建CI2失败: %v", err)
	}

	svc := NewCMDBService(client)

	desc := "应用部署在服务器上"
	rel, err := svc.CreateRelationship(ctx, &CreateRelationshipRequest{
		SourceCIID:  ci1.ID,
		TargetCIID:  ci2.ID,
		Type:        "runs_on",
		Description: &desc,
		TenantID:    tenant1.ID,
	})
	if err != nil {
		t.Fatalf("CreateRelationship 失败: %v", err)
	}
	if rel.TenantID != tenant1.ID {
		t.Errorf("关系 tenant_id 未正确设置: 期望 %d, 实际 %d", tenant1.ID, rel.TenantID)
	}
}

// TestListRelationships_CrossTenantDeny 确保跨租户查询不返回数据
func TestListRelationships_CrossTenantDeny(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_ent5?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()

	tenant1, err := createCMDBTestTenant(ctx, client, "Tenant 1", "t1", "t1.com")
	if err != nil {
		t.Fatalf("创建租户1失败: %v", err)
	}
	tenant999, err := createCMDBTestTenant(ctx, client, "Tenant 999", "t999", "t999.com")
	if err != nil {
		t.Fatalf("创建租户999失败: %v", err)
	}
	ciType1, err := createTestCIType(ctx, client, tenant1.ID, "server-type")
	if err != nil {
		t.Fatalf("创建CI类型失败: %v", err)
	}

	ci1, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "tenant1-ci")
	if err != nil {
		t.Fatalf("创建CI失败: %v", err)
	}

	client.CIRelationship.Create().
		SetRelationshipType("depends_on").
		SetSourceCiID(ci1.ID).
		SetTargetCiID(ci1.ID).
		SetTenantID(tenant1.ID).
		SaveX(ctx)

	svc := NewCMDBService(client)

	// 租户999不应该能看到租户1的关系
	rels, err := svc.ListRelationships(ctx, tenant999.ID, nil)
	if err != nil {
		t.Fatalf("ListRelationships 失败: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("跨租户数据泄露: 租户%d不应看到任何关系, 实际看到 %d 条", tenant999.ID, len(rels))
	}
}

// TestCreateRelationship_CrossTenantDenied SEC-004: 跨租户 CI 关系创建应被拒绝
func TestCreateRelationship_CrossTenantDenied(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_ent6?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()

	// 创建两个租户
	tenant1, err := createCMDBTestTenant(ctx, client, "Tenant 1", "t1", "t1.com")
	if err != nil {
		t.Fatalf("创建租户1失败: %v", err)
	}
	tenant2, err := createCMDBTestTenant(ctx, client, "Tenant 2", "t2", "t2.com")
	if err != nil {
		t.Fatalf("创建租户2失败: %v", err)
	}

	ciType1, err := createTestCIType(ctx, client, tenant1.ID, "type1")
	if err != nil {
		t.Fatalf("创建CI类型1失败: %v", err)
	}
	ciType2, err := createTestCIType(ctx, client, tenant2.ID, "type2")
	if err != nil {
		t.Fatalf("创建CI类型2失败: %v", err)
	}

	// 租户1的 CI
	ci1, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "ci-tenant1")
	if err != nil {
		t.Fatalf("创建CI1失败: %v", err)
	}
	// 租户2的 CI
	ci2, err := createTestCI(ctx, client, tenant2.ID, ciType2.ID, "ci-tenant2")
	if err != nil {
		t.Fatalf("创建CI2失败: %v", err)
	}

	svc := NewCMDBService(client)

	// 尝试以租户1身份创建 source=租户1CI, target=租户2CI 的关系 → 应失败
	_, err = svc.CreateRelationship(ctx, &CreateRelationshipRequest{
		SourceCIID: ci1.ID,
		TargetCIID: ci2.ID,
		Type:       "depends_on",
		TenantID:   tenant1.ID,
	})
	if err == nil {
		t.Error("跨租户关系创建应被拒绝，但成功创建了")
	}
}

// TestCreateRelationship_SourceCINotBelongToTenant SEC-004: source CI 不属于当前租户时拒绝
func TestCreateRelationship_SourceCINotBelongToTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_ent7?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()

	tenant1, err := createCMDBTestTenant(ctx, client, "Tenant 1", "t1", "t1.com")
	if err != nil {
		t.Fatalf("创建租户1失败: %v", err)
	}
	tenant2, err := createCMDBTestTenant(ctx, client, "Tenant 2", "t2", "t2.com")
	if err != nil {
		t.Fatalf("创建租户2失败: %v", err)
	}

	ciType1, err := createTestCIType(ctx, client, tenant1.ID, "type1")
	if err != nil {
		t.Fatalf("创建CI类型1失败: %v", err)
	}
	ciType2, err := createTestCIType(ctx, client, tenant2.ID, "type2")
	if err != nil {
		t.Fatalf("创建CI类型2失败: %v", err)
	}

	// 租户1的 CI（将作为不匹配的 source）
	ci1, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "ci-tenant1")
	if err != nil {
		t.Fatalf("创建CI1失败: %v", err)
	}
	// 租户2的两个 CI
	ci2, err := createTestCI(ctx, client, tenant2.ID, ciType2.ID, "ci-tenant2-a")
	if err != nil {
		t.Fatalf("创建CI2失败: %v", err)
	}

	svc := NewCMDBService(client)

	// 尝试以租户2身份创建 source=租户1CI 的关系 → 应失败（source CI 不属于租户2）
	_, err = svc.CreateRelationship(ctx, &CreateRelationshipRequest{
		SourceCIID: ci1.ID,
		TargetCIID: ci2.ID,
		Type:       "depends_on",
		TenantID:   tenant2.ID,
	})
	if err == nil {
		t.Error("source CI 不属于当前租户时应被拒绝")
	}
}

// TestCreateRelationship_SameTenantSuccess SEC-004: 同租户 CI 关系创建成功
func TestCreateRelationship_SameTenantSuccess(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_ent8?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()

	tenant1, err := createCMDBTestTenant(ctx, client, "Tenant 1", "t1", "t1.com")
	if err != nil {
		t.Fatalf("创建租户失败: %v", err)
	}
	ciType1, err := createTestCIType(ctx, client, tenant1.ID, "type1")
	if err != nil {
		t.Fatalf("创建CI类型失败: %v", err)
	}

	ci1, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "ci-a")
	if err != nil {
		t.Fatalf("创建CI1失败: %v", err)
	}
	ci2, err := createTestCI(ctx, client, tenant1.ID, ciType1.ID, "ci-b")
	if err != nil {
		t.Fatalf("创建CI2失败: %v", err)
	}

	svc := NewCMDBService(client)

	desc := "同租户关系"
	rel, err := svc.CreateRelationship(ctx, &CreateRelationshipRequest{
		SourceCIID:  ci1.ID,
		TargetCIID:  ci2.ID,
		Type:        "runs_on",
		Description: &desc,
		TenantID:    tenant1.ID,
	})
	if err != nil {
		t.Fatalf("同租户关系创建应成功: %v", err)
	}
	if rel.TenantID != tenant1.ID {
		t.Errorf("关系 tenant_id 未正确设置: 期望 %d, 实际 %d", tenant1.ID, rel.TenantID)
	}
}
