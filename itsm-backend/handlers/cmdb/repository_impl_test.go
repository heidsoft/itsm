package cmdb

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
)

func TestEntRepository_CreateCI_ValidatesAttributes(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_repo_create?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Tenant A").
		SetCode("tenant-a").
		SetDomain("tenant-a.example.com").
		SetStatus("active").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建租户失败: %v", err)
	}

	ciType, err := client.CIType.Create().
		SetName("application").
		SetTenantID(tenant.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建CI类型失败: %v", err)
	}

	_, err = client.CIAttributeDefinition.Create().
		SetName("owner_email").
		SetDisplayName("Owner Email").
		SetType("string").
		SetRequired(true).
		SetCiTypeID(ciType.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建属性定义失败: %v", err)
	}

	_, err = client.CIAttributeDefinition.Create().
		SetName("cpu_cores").
		SetDisplayName("CPU Cores").
		SetType("integer").
		SetDefaultValue("2").
		SetCiTypeID(ciType.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建属性定义失败: %v", err)
	}

	repo := NewEntRepository(client)

	_, err = repo.CreateCI(ctx, &ConfigurationItem{
		Name:        "app-without-owner",
		CITypeID:    ciType.ID,
		Status:      "active",
		Environment: "production",
		Criticality: "high",
		TenantID:    tenant.ID,
		Attributes: map[string]interface{}{
			"cpu_cores": "4",
		},
	})
	if err == nil {
		t.Fatal("缺少必填属性时应返回错误")
	}

	created, err := repo.CreateCI(ctx, &ConfigurationItem{
		Name:        "app-with-owner",
		CITypeID:    ciType.ID,
		Status:      "active",
		Environment: "production",
		Criticality: "high",
		TenantID:    tenant.ID,
		Attributes: map[string]interface{}{
			"owner_email": "ops@example.com",
		},
	})
	if err != nil {
		t.Fatalf("创建CI失败: %v", err)
	}

	if created.Attributes["cpu_cores"] != 2 {
		t.Fatalf("默认值未生效, got=%v", created.Attributes["cpu_cores"])
	}
}

func TestEntRepository_UpdateCI_EnforcesUniqueAttributes(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_repo_update?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Tenant B").
		SetCode("tenant-b").
		SetDomain("tenant-b.example.com").
		SetStatus("active").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建租户失败: %v", err)
	}

	ciType, err := client.CIType.Create().
		SetName("server").
		SetTenantID(tenant.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建CI类型失败: %v", err)
	}

	_, err = client.CIAttributeDefinition.Create().
		SetName("hostname").
		SetDisplayName("Hostname").
		SetType("string").
		SetUnique(true).
		SetCiTypeID(ciType.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建属性定义失败: %v", err)
	}

	repo := NewEntRepository(client)
	first, err := repo.CreateCI(ctx, &ConfigurationItem{
		Name:         "server-01",
		CITypeID:     ciType.ID,
		Status:       "active",
		Environment:  "production",
		Criticality:  "high",
		TenantID:     tenant.ID,
		SerialNumber: "sn-01",
		Attributes: map[string]interface{}{
			"hostname": "prod-app-01",
		},
	})
	if err != nil {
		t.Fatalf("创建首个CI失败: %v", err)
	}

	second, err := repo.CreateCI(ctx, &ConfigurationItem{
		Name:         "server-02",
		CITypeID:     ciType.ID,
		Status:       "active",
		Environment:  "production",
		Criticality:  "high",
		TenantID:     tenant.ID,
		SerialNumber: "sn-02",
		Attributes: map[string]interface{}{
			"hostname": "prod-app-02",
		},
	})
	if err != nil {
		t.Fatalf("创建第二个CI失败: %v", err)
	}

	second.Attributes["hostname"] = first.Attributes["hostname"]
	_, err = repo.UpdateCI(ctx, second)
	if err == nil {
		t.Fatal("更新为重复唯一属性时应返回错误")
	}
}
