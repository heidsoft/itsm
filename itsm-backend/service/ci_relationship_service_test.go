package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/cirelationship"
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

func TestGetCIImpactAnalysis_UpstreamCriticalAndAffectedItems(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ci_impact_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, _ := createCMDBTestTenant(ctx, client, "Impact Fields Tenant", "impact-fields", "impact-fields.example.com")
	ciType, _ := createTestCIType(ctx, client, tenant.ID, "impact-fields-server")
	root, _ := createTestCI(ctx, client, tenant.ID, ciType.ID, "if-root")
	down, _ := createTestCI(ctx, client, tenant.ID, ciType.ID, "if-down")
	up, _ := createTestCI(ctx, client, tenant.ID, ciType.ID, "if-up")

	// root --impacts(critical)--> down； up --depends_on--> root
	client.CIRelationship.Create().SetRelationshipType("impacts").
		SetSourceCiID(root.ID).SetTargetCiID(down.ID).
		SetStrength(cirelationship.StrengthCritical).
		SetTenantID(tenant.ID).SaveX(ctx)
	client.CIRelationship.Create().SetRelationshipType("depends_on").
		SetSourceCiID(up.ID).SetTargetCiID(root.ID).
		SetTenantID(tenant.ID).SaveX(ctx)

	// root 关联工单与事件
	user := client.User.Create().SetUsername("impact-user").SetEmail("impact-user@example.com").
		SetName("Impact User").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	tk := client.Ticket.Create().SetTitle("impact ticket").SetTicketNumber("TCK-IMPACT-1").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).SaveX(ctx)
	in := client.Incident.Create().SetTitle("impact incident").SetIncidentNumber("INC-IMPACT-1").
		SetReporterID(user.ID).SetTenantID(tenant.ID).SaveX(ctx)
	client.ConfigurationItem.UpdateOneID(root.ID).AddTickets(tk).AddIncidents(in).ExecX(ctx)

	svc := NewCIRelationshipService(client, zaptest.NewLogger(t).Sugar())
	result, err := svc.GetCIImpactAnalysis(ctx, root.ID, tenant.ID, 3)
	if err != nil {
		t.Fatalf("GetCIImpactAnalysis failed: %v", err)
	}

	if len(result.DownstreamImpact) != 1 || result.DownstreamImpact[0].CIID != down.ID {
		t.Fatalf("DownstreamImpact = %#v, want only CI %d", result.DownstreamImpact, down.ID)
	}
	if len(result.UpstreamImpact) != 1 || result.UpstreamImpact[0].CIID != up.ID {
		t.Fatalf("UpstreamImpact = %#v, want only CI %d", result.UpstreamImpact, up.ID)
	}
	if len(result.CriticalDependencies) != 1 || result.CriticalDependencies[0].CIID != down.ID {
		t.Fatalf("CriticalDependencies = %#v, want only critical CI %d", result.CriticalDependencies, down.ID)
	}
	if len(result.AffectedTickets) != 1 || result.AffectedTickets[0].ID != tk.ID {
		t.Fatalf("AffectedTickets = %#v, want ticket %d", result.AffectedTickets, tk.ID)
	}
	if len(result.AffectedIncidents) != 1 || result.AffectedIncidents[0].ID != in.ID {
		t.Fatalf("AffectedIncidents = %#v, want incident %d", result.AffectedIncidents, in.ID)
	}
	if result.TotalImpacted != 1 {
		t.Fatalf("TotalImpacted = %d, want 1 (downstream only)", result.TotalImpacted)
	}
	if result.Graph == nil || result.Graph.TotalNodes != 3 || result.Graph.TotalEdges != 2 {
		t.Fatalf("impact graph = %#v, want 3 nodes / 2 edges", result.Graph)
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

func TestCIRelationship_CrossTenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ci_rel_cross_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	svc := NewCIRelationshipService(client, zaptest.NewLogger(t).Sugar())

	tenantA, _ := createCMDBTestTenant(ctx, client, "Tenant A", "tenant-a-iso", "a-iso.example.com")
	tenantB, _ := createCMDBTestTenant(ctx, client, "Tenant B", "tenant-b-iso", "b-iso.example.com")
	ciType, _ := createTestCIType(ctx, client, tenantA.ID, "server-iso")
	ciA1, _ := createTestCI(ctx, client, tenantA.ID, ciType.ID, "iso-a1")
	ciA2, _ := createTestCI(ctx, client, tenantA.ID, ciType.ID, "iso-a2")

	// 租户A创建关系成功
	rel, err := svc.CreateCIRelationship(ctx, &dto.CreateCIRelationshipRequest{
		SourceCIID:       ciA1.ID,
		TargetCIID:       ciA2.ID,
		RelationshipType: "depends_on",
	}, tenantA.ID)
	if err != nil {
		t.Fatalf("tenant A create relationship failed: %v", err)
	}

	// 租户B不能用租户A的CI创建关系（失败闭合）
	if _, err := svc.CreateCIRelationship(ctx, &dto.CreateCIRelationshipRequest{
		SourceCIID:       ciA1.ID,
		TargetCIID:       ciA2.ID,
		RelationshipType: "hosts",
	}, tenantB.ID); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("cross-tenant create should fail with not found, got: %v", err)
	}

	// 租户B查不到租户A的关系（未命中语义为返回 nil, nil）
	if got, err := svc.GetCIRelationshipByID(ctx, rel.ID, tenantB.ID); err != nil || got != nil {
		t.Fatalf("cross-tenant GetCIRelationshipByID should return nil, got=%v err=%v", got, err)
	}
	listB, err := svc.ListAllCIRelationships(ctx, tenantB.ID, 1, 10, "")
	if err != nil {
		t.Fatalf("tenant B list relationships failed: %v", err)
	}
	if listB.Total != 0 {
		t.Fatalf("tenant B should see 0 relationships, got %d", listB.Total)
	}
	byCI, err := svc.ListCIRelationshipsByCIID(ctx, ciA1.ID, tenantB.ID, "all")
	if err != nil {
		t.Fatalf("tenant B list by CI failed: %v", err)
	}
	if len(byCI) != 0 {
		t.Fatalf("tenant B should see 0 relationships for tenant A CI, got %d", len(byCI))
	}

	// 租户B不能删除租户A的关系
	if err := svc.DeleteCIRelationship(ctx, rel.ID, tenantB.ID); err == nil {
		t.Fatal("cross-tenant delete should fail")
	}
}
