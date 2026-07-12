package service

import (
	"context"
	"encoding/json"
	"testing"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"
)

func TestListAllCIRelationships_StandardPaginationContract(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ci_relationship_pagination?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := createCMDBTestTenant(ctx, client, "Tenant Pagination", "tenant-pagination", "pagination.example.com")
	if err != nil {
		t.Fatalf("创建租户失败: %v", err)
	}
	ciType, err := createTestCIType(ctx, client, tenant.ID, "server-pagination")
	if err != nil {
		t.Fatalf("创建CI类型失败: %v", err)
	}
	ci1, err := createTestCI(ctx, client, tenant.ID, ciType.ID, "server-pagination-1")
	if err != nil {
		t.Fatalf("创建CI1失败: %v", err)
	}
	ci2, err := createTestCI(ctx, client, tenant.ID, ciType.ID, "server-pagination-2")
	if err != nil {
		t.Fatalf("创建CI2失败: %v", err)
	}
	ci3, err := createTestCI(ctx, client, tenant.ID, ciType.ID, "server-pagination-3")
	if err != nil {
		t.Fatalf("创建CI3失败: %v", err)
	}

	client.CIRelationship.Create().
		SetRelationshipType("depends_on").
		SetSourceCiID(ci1.ID).
		SetTargetCiID(ci2.ID).
		SetTenantID(tenant.ID).
		SaveX(ctx)
	client.CIRelationship.Create().
		SetRelationshipType("hosts").
		SetSourceCiID(ci2.ID).
		SetTargetCiID(ci3.ID).
		SetTenantID(tenant.ID).
		SaveX(ctx)
	client.CIRelationship.Create().
		SetRelationshipType("connects_to").
		SetSourceCiID(ci3.ID).
		SetTargetCiID(ci1.ID).
		SetTenantID(tenant.ID).
		SaveX(ctx)

	svc := NewCIRelationshipService(client, logger)
	result, err := svc.ListAllCIRelationships(ctx, tenant.ID, 1, 2, "")
	if err != nil {
		t.Fatalf("ListAllCIRelationships 失败: %v", err)
	}

	if result.Total != 3 {
		t.Fatalf("Total = %d, want 3", result.Total)
	}
	if result.Page != 1 {
		t.Fatalf("Page = %d, want 1", result.Page)
	}
	if result.PageSize != 2 {
		t.Fatalf("PageSize = %d, want 2", result.PageSize)
	}
	if result.TotalPages != 2 {
		t.Fatalf("TotalPages = %d, want 2", result.TotalPages)
	}
	if len(result.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(result.Items))
	}

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("序列化响应失败: %v", err)
	}
	if !json.Valid(payload) {
		t.Fatalf("响应不是有效JSON: %s", string(payload))
	}
	var responseFields map[string]interface{}
	if err := json.Unmarshal(payload, &responseFields); err != nil {
		t.Fatalf("解析响应JSON失败: %v", err)
	}
	if _, ok := responseFields["pageSize"]; !ok {
		t.Fatalf("响应缺少 pageSize 字段: %s", string(payload))
	}
	if _, ok := responseFields["totalPages"]; !ok {
		t.Fatalf("响应缺少 totalPages 字段: %s", string(payload))
	}
	if _, ok := responseFields["size"]; ok {
		t.Fatalf("响应不应再包含 size 字段: %s", string(payload))
	}
}

func TestListAllCIRelationships_NormalizesPagination(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ci_relationship_pagination_normalize?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewCIRelationshipService(client, logger)

	tenant, err := createCMDBTestTenant(ctx, client, "Tenant Normalize", "tenant-normalize", "normalize.example.com")
	if err != nil {
		t.Fatalf("创建租户失败: %v", err)
	}

	result, err := svc.ListAllCIRelationships(ctx, tenant.ID, 0, 500, "")
	if err != nil {
		t.Fatalf("ListAllCIRelationships 失败: %v", err)
	}
	if result.Page != 1 {
		t.Fatalf("Page = %d, want 1", result.Page)
	}
	if result.PageSize != 100 {
		t.Fatalf("PageSize = %d, want 100", result.PageSize)
	}
	if result.TotalPages != 0 {
		t.Fatalf("TotalPages = %d, want 0", result.TotalPages)
	}
}

func TestGetCIImpactAnalysis_TraversesByDepthAndStopsCycles(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ci_impact_depth?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, err := createCMDBTestTenant(ctx, client, "Impact Tenant", "impact-tenant", "impact.example.com")
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	ciType, err := createTestCIType(ctx, client, tenant.ID, "impact-server")
	if err != nil {
		t.Fatalf("create CI type: %v", err)
	}
	root, err := createTestCI(ctx, client, tenant.ID, ciType.ID, "root")
	if err != nil {
		t.Fatalf("create root CI: %v", err)
	}
	level1, err := createTestCI(ctx, client, tenant.ID, ciType.ID, "level-1")
	if err != nil {
		t.Fatalf("create level-1 CI: %v", err)
	}
	level2, err := createTestCI(ctx, client, tenant.ID, ciType.ID, "level-2")
	if err != nil {
		t.Fatalf("create level-2 CI: %v", err)
	}

	for _, edge := range [][2]int{{root.ID, level1.ID}, {level1.ID, level2.ID}, {level2.ID, root.ID}} {
		client.CIRelationship.Create().
			SetRelationshipType("impacts").
			SetSourceCiID(edge[0]).
			SetTargetCiID(edge[1]).
			SetTenantID(tenant.ID).
			SaveX(ctx)
	}

	svc := NewCIRelationshipService(client, zaptest.NewLogger(t).Sugar())
	result, err := svc.GetCIImpactAnalysis(ctx, root.ID, tenant.ID, 2)
	if err != nil {
		t.Fatalf("GetCIImpactAnalysis failed: %v", err)
	}
	if result.TotalImpacted != 2 {
		t.Fatalf("TotalImpacted = %d, want 2", result.TotalImpacted)
	}
	level2Distance := 0
	for _, impacted := range result.DownstreamImpact {
		if impacted.CIID == level2.ID {
			level2Distance = impacted.Distance
		}
	}
	if level2Distance != 2 {
		t.Fatalf("level-2 distance = %d, want 2", level2Distance)
	}
	if result.Graph == nil || result.Graph.TotalNodes != 3 {
		t.Fatalf("impact graph = %#v, want 3 nodes", result.Graph)
	}
}

func TestGetCITopology_UsesUnifiedGraphContract(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ci_topology_graph?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant, _ := createCMDBTestTenant(ctx, client, "Topology Tenant", "topology-tenant", "topology.example.com")
	ciType, _ := createTestCIType(ctx, client, tenant.ID, "server")
	root, _ := createTestCI(ctx, client, tenant.ID, ciType.ID, "root")
	child, _ := createTestCI(ctx, client, tenant.ID, ciType.ID, "child")
	client.CIRelationship.Create().SetRelationshipType("depends_on").SetSourceCiID(root.ID).
		SetTargetCiID(child.ID).SetTenantID(tenant.ID).SaveX(ctx)

	result, err := NewCIRelationshipService(client, zaptest.NewLogger(t).Sugar()).GetCITopology(ctx, root.ID, tenant.ID, 1)
	if err != nil {
		t.Fatalf("GetCITopology failed: %v", err)
	}
	if result.RootCIID != root.ID || result.TotalNodes != 2 || result.TotalEdges != 1 {
		t.Fatalf("unexpected topology graph: %#v", result)
	}
}

func TestGetCIImpactAnalysis_RequiresTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ci_impact_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewCIRelationshipService(client, zaptest.NewLogger(t).Sugar())
	if _, err := svc.GetCIImpactAnalysis(context.Background(), 1, 0, 3); err == nil {
		t.Fatal("expected missing tenant ID to be rejected")
	}
}
