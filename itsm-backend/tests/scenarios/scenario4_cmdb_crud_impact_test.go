package scenarios

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/enttest"
)

// Scenario 4: CMDB CI 类型/实例/关系 CRUD + 影响分析 + 阿里云发现
//
// 覆盖：
//   - CI 类型创建
//   - CI 实例 CRUD
//   - CI 关系创建（depends_on, runs_on）
//   - 影响分析：通过关系遍历确定影响范围
//   - 云发现模拟：标记 discovery_source
//   - 跨租户隔离

func TestScenario4_CMDBCrudImpactAndDiscovery(t *testing.T) {
	client := enttest.Open(t, "sqlite3", scenarioDSN("cmdb_crud"))
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	_ = logger

	tenantA := mustCreateTenant(ctx, t, client, "TenantA", "cmdb-a", "cmdb-a.test.com")
	tenantB := mustCreateTenant(ctx, t, client, "TenantB", "cmdb-b", "cmdb-b.test.com")

	t.Run("CI type and instance CRUD", func(t *testing.T) {
		ciType := mustCreateCIType(ctx, t, client, tenantA.ID, "应用服务器")

		web1 := mustCreateCI(ctx, t, client, tenantA.ID, ciType.ID, "web-prod-01", "server", "high")
		web2 := mustCreateCI(ctx, t, client, tenantA.ID, ciType.ID, "web-prod-02", "server", "high")
		db := mustCreateCI(ctx, t, client, tenantA.ID, ciType.ID, "db-prod-01", "database", "critical")
		app := mustCreateCI(ctx, t, client, tenantA.ID, ciType.ID, "app-service", "application", "high")

		t.Logf("CI 创建完成: web1=%d, web2=%d, db=%d, app=%d", web1.ID, web2.ID, db.ID, app.ID)
	})

	t.Run("CI relationships", func(t *testing.T) {
		appCI := findCIByName(ctx, t, client, tenantA.ID, "app-service")
		dbCI := findCIByName(ctx, t, client, tenantA.ID, "db-prod-01")
		web1 := findCIByName(ctx, t, client, tenantA.ID, "web-prod-01")
		web2 := findCIByName(ctx, t, client, tenantA.ID, "web-prod-02")

		mustCreateCIRel(ctx, t, client, tenantA.ID, appCI.ID, dbCI.ID, "depends_on")
		mustCreateCIRel(ctx, t, client, tenantA.ID, appCI.ID, web1.ID, "runs_on")
		mustCreateCIRel(ctx, t, client, tenantA.ID, appCI.ID, web2.ID, "runs_on")

		rels, err := client.CIRelationship.Query().
			Where(cirelationship.SourceCiID(appCI.ID), cirelationship.TenantID(tenantA.ID)).
			All(ctx)
		if err != nil {
			t.Fatalf("查询关系失败: %v", err)
		}
		if len(rels) != 3 {
			t.Fatalf("期望 3 条关系, 实际: %d", len(rels))
		}
		t.Logf("关系验证: app-service 有 %d 条出向关系", len(rels))
	})

	t.Run("impact analysis - db failure impacts app", func(t *testing.T) {
		dbCI := findCIByName(ctx, t, client, tenantA.ID, "db-prod-01")

		dependents, err := client.CIRelationship.Query().
			Where(
				cirelationship.TargetCiID(dbCI.ID),
				cirelationship.RelationshipType("depends_on"),
				cirelationship.TenantID(tenantA.ID),
			).
			All(ctx)
		if err != nil {
			t.Fatalf("影响分析查询失败: %v", err)
		}
		if len(dependents) != 1 {
			t.Fatalf("db-prod-01 应有 1 个 depends_on 依赖方, 实际: %d", len(dependents))
		}

		affectedCI, err := client.ConfigurationItem.Get(ctx, dependents[0].SourceCiID)
		if err != nil {
			t.Fatalf("获取受影响 CI 失败: %v", err)
		}
		if affectedCI.Name != "app-service" {
			t.Fatalf("受影响 CI 应为 app-service, 实际: %s", affectedCI.Name)
		}
		t.Logf("影响分析: db-prod-01 故障 → 影响 %s", affectedCI.Name)
	})

	t.Run("cloud discovery - mark discovery source as aliyun", func(t *testing.T) {
		ciType := mustCreateCIType(ctx, t, client, tenantA.ID, "云资源")

		discovered, err := client.ConfigurationItem.Create().
			SetName("ecs-instance-001").
			SetCiType("cloud_server").
			SetCiTypeID(ciType.ID).
			SetStatus("active").
			SetEnvironment("production").
			SetCriticality("medium").
			SetDiscoverySource("aliyun").
			SetTenantID(tenantA.ID).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建云发现 CI 失败: %v", err)
		}
		if discovered.DiscoverySource != "aliyun" {
			t.Fatalf("发现源应为 aliyun, 实际: %s", discovered.DiscoverySource)
		}

		aliyunCIs, err := client.ConfigurationItem.Query().
			Where(
				configurationitem.DiscoverySource("aliyun"),
				configurationitem.TenantID(tenantA.ID),
			).
			All(ctx)
		if err != nil {
			t.Fatalf("查询阿里云 CI 失败: %v", err)
		}
		if len(aliyunCIs) != 1 {
			t.Fatalf("期望 1 个阿里云发现 CI, 实际: %d", len(aliyunCIs))
		}
		t.Logf("云发现验证: %d 个 aliyun 来源 CI", len(aliyunCIs))
	})

	t.Run("cross-tenant CI isolation", func(t *testing.T) {
		ciTypeB := mustCreateCIType(ctx, t, client, tenantB.ID, "TenantB Type")
		mustCreateCI(ctx, t, client, tenantB.ID, ciTypeB.ID, "tenant-b-server", "server", "low")

		tenantACIs, err := client.ConfigurationItem.Query().
			Where(configurationitem.TenantID(tenantA.ID)).
			All(ctx)
		if err != nil {
			t.Fatalf("查询 tenant A CI 失败: %v", err)
		}
		for _, ci := range tenantACIs {
			if ci.TenantID == tenantB.ID {
				t.Fatalf("tenant A 查询中不应包含 tenant B 的 CI: %s", ci.Name)
			}
		}

		tenantBCIs, err := client.ConfigurationItem.Query().
			Where(configurationitem.TenantID(tenantB.ID)).
			All(ctx)
		if err != nil {
			t.Fatalf("查询 tenant B CI 失败: %v", err)
		}
		if len(tenantBCIs) != 1 {
			t.Fatalf("tenant B 应只有 1 个 CI, 实际: %d", len(tenantBCIs))
		}
		t.Logf("租户隔离: A=%d CIs, B=%d CIs", len(tenantACIs), len(tenantBCIs))
	})
}

func mustCreateCIType(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, name string) *ent.CIType {
	t.Helper()
	ct, err := client.CIType.Create().SetName(name).SetTenantID(tenantID).Save(ctx)
	if err != nil {
		t.Fatalf("创建 CI 类型失败: %v", err)
	}
	return ct
}

func mustCreateCI(ctx context.Context, t *testing.T, client *ent.Client, tenantID, ciTypeID int, name, ciType, criticality string) *ent.ConfigurationItem {
	t.Helper()
	ci, err := client.ConfigurationItem.Create().
		SetName(name).
		SetCiType(ciType).
		SetCiTypeID(ciTypeID).
		SetStatus("active").
		SetEnvironment("production").
		SetCriticality(criticality).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 CI 实例失败: %v", err)
	}
	return ci
}

func mustCreateCIRel(ctx context.Context, t *testing.T, client *ent.Client, tenantID, sourceID, targetID int, relType string) *ent.CIRelationship {
	t.Helper()
	rel, err := client.CIRelationship.Create().
		SetSourceCiID(sourceID).
		SetTargetCiID(targetID).
		SetRelationshipType(relType).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 CI 关系失败: %v", err)
	}
	return rel
}

func findCIByName(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, name string) *ent.ConfigurationItem {
	t.Helper()
	ci, err := client.ConfigurationItem.Query().
		Where(configurationitem.Name(name), configurationitem.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("查找 CI '%s' 失败: %v", name, err)
	}
	return ci
}
