package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"
)

func TestCITypeInheritanceReturnsMergedAttributes(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_model_inheritance?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	tenant, err := createCMDBTestTenant(ctx, client, "Model Tenant", "model-tenant", "model.example.com")
	if err != nil {
		t.Fatal(err)
	}

	typeService := NewCITypeService(client, logger)
	base, err := typeService.CreateCIType(ctx, &dto.CreateCITypeRequest{Name: "计算资源"}, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	child, err := typeService.CreateCIType(ctx, &dto.CreateCITypeRequest{
		Name:         "物理服务器",
		ParentTypeID: &base.ID,
	}, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}

	attributeService := NewCIAttributeDefinitionService(client, logger)
	_, err = attributeService.CreateCIAttributeDefinition(ctx, &dto.CreateCIAttributeDefinitionRequest{
		Name:         "owner",
		DisplayName:  "负责人",
		Type:         "string",
		CiTypeID:     base.ID,
		DisplayOrder: 10,
		GroupName:    "归属",
		IsSearchable: true,
	}, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = attributeService.CreateCIAttributeDefinition(ctx, &dto.CreateCIAttributeDefinitionRequest{
		Name:         "cpu_cores",
		DisplayName:  "CPU核数",
		Type:         "integer",
		CiTypeID:     child.ID,
		DisplayOrder: 20,
	}, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}

	attributes, err := attributeService.ListCIAttributeDefinitionsByCITypeID(ctx, child.ID, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(attributes) != 2 {
		t.Fatalf("len(attributes) = %d, want 2", len(attributes))
	}
	if attributes[0].Name != "owner" || attributes[0].GroupName != "归属" || !attributes[0].IsSearchable {
		t.Fatalf("inherited attribute not mapped completely: %#v", attributes[0])
	}
	if attributes[1].Name != "cpu_cores" {
		t.Fatalf("child attribute = %q, want cpu_cores", attributes[1].Name)
	}
}

func TestCITypeInheritanceRejectsCrossTenantAndCycle(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_model_boundaries?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	tenantA, _ := createCMDBTestTenant(ctx, client, "Tenant A", "model-a", "a.model.example.com")
	tenantB, _ := createCMDBTestTenant(ctx, client, "Tenant B", "model-b", "b.model.example.com")
	service := NewCITypeService(client, logger)

	typeA, err := service.CreateCIType(ctx, &dto.CreateCITypeRequest{Name: "A"}, tenantA.ID)
	if err != nil {
		t.Fatal(err)
	}
	typeB, err := service.CreateCIType(ctx, &dto.CreateCITypeRequest{Name: "B"}, tenantA.ID)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := service.CreateCIType(ctx, &dto.CreateCITypeRequest{Name: "Foreign"}, tenantB.ID)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = service.UpdateCIType(ctx, typeA.ID, tenantA.ID, &dto.UpdateCITypeRequest{
		ParentTypeID: &foreign.ID,
	}); err == nil {
		t.Fatal("expected cross-tenant parent to be rejected")
	}
	if _, err = service.UpdateCIType(ctx, typeB.ID, tenantA.ID, &dto.UpdateCITypeRequest{
		ParentTypeID: &typeA.ID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.UpdateCIType(ctx, typeA.ID, tenantA.ID, &dto.UpdateCITypeRequest{
		ParentTypeID: &typeB.ID,
	}); err == nil {
		t.Fatal("expected inheritance cycle to be rejected")
	}
}
