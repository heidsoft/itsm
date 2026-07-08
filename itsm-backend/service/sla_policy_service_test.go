package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSLAPolicyTest(t *testing.T) (*ent.Client, *SLAPolicyService, context.Context) {
	dbName := strings.NewReplacer("/", "-", " ", "-", ":", "-").Replace(t.Name())
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", dbName))
	svc := NewSLAPolicyService(client)
	return client, svc, context.Background()
}

func stringPtr(v string) *string    { return &v }
func slaPolicyIntPtr(v int) *int    { return &v }
func slaPolicyBoolPtr(v bool) *bool { return &v }

func createPolicyTenant(ctx context.Context, t *testing.T, client *ent.Client) *ent.Tenant {
	return createPolicyTenantWithCode(ctx, t, client, "Policy Tenant", "policy-tenant", "policy.example.com")
}

func createPolicyTenantWithCode(ctx context.Context, t *testing.T, client *ent.Client, name, code, domain string) *ent.Tenant {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName(name).
		SetCode(code).
		SetDomain(domain).
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tenant
}

func createOtherPolicyTenant(ctx context.Context, t *testing.T, client *ent.Client) *ent.Tenant {
	return createPolicyTenantWithCode(ctx, t, client, "Other Policy Tenant", "other-policy-tenant", "other-policy.example.com")
}

func sampleCreateRequest(tenantID int) dto.CreateSLAPolicyRequest {
	escalation := map[string]interface{}{"level_1": "team_lead", "level_2": "manager"}
	return dto.CreateSLAPolicyRequest{
		Name:                  "Standard SLA",
		Description:           "Standard response/resolution policy",
		CustomerTier:          stringPtr("gold"),
		TicketType:            stringPtr("incident"),
		Priority:              stringPtr("high"),
		ResponseTimeMinutes:   30,
		ResolutionTimeMinutes: 240,
		BusinessHours:         map[string]interface{}{"start": "09:00", "end": "18:00"},
		ExcludeWeekends:       true,
		ExcludeHolidays:       false,
		EscalationRules:       escalation,
		IsActive:              true,
		PriorityScore:         80,
		TenantID:              tenantID,
	}
}

func TestSLAPolicyService_Create_StoresAllProvidedFields(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	req := sampleCreateRequest(tenant.ID)

	policy, err := svc.CreateSLAPolicy(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, policy)

	assert.NotZero(t, policy.ID)
	assert.Equal(t, req.Name, policy.Name)
	assert.Equal(t, req.Description, policy.Description)
	assert.Equal(t, *req.CustomerTier, policy.CustomerTier)
	assert.Equal(t, *req.TicketType, policy.TicketType)
	assert.Equal(t, *req.Priority, policy.Priority)
	assert.Equal(t, req.ResponseTimeMinutes, policy.ResponseTimeMinutes)
	assert.Equal(t, req.ResolutionTimeMinutes, policy.ResolutionTimeMinutes)
	assert.Equal(t, req.ExcludeWeekends, policy.ExcludeWeekends)
	assert.Equal(t, req.ExcludeHolidays, policy.ExcludeHolidays)
	assert.Equal(t, req.IsActive, policy.IsActive)
	assert.Equal(t, req.PriorityScore, policy.PriorityScore)
	assert.Equal(t, req.TenantID, policy.TenantID)
	assert.NotNil(t, policy.EscalationRules)
}

func TestSLAPolicyService_Create_OmitsEmptyOptionalEnums(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	req := dto.CreateSLAPolicyRequest{
		Name:                  "Default SLA",
		ResponseTimeMinutes:   60,
		ResolutionTimeMinutes: 480,
		IsActive:              true,
		PriorityScore:         10,
		TenantID:              tenant.ID,
	}

	policy, err := svc.CreateSLAPolicy(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, "", policy.CustomerTier)
	assert.Equal(t, "", policy.TicketType)
	assert.Equal(t, "", policy.Priority)
}

func TestSLAPolicyService_GetByID_RoundTrip(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	policy, err := svc.CreateSLAPolicy(ctx, sampleCreateRequest(tenant.ID))
	require.NoError(t, err)

	fetched, err := svc.GetSLAPolicyByID(ctx, policy.ID)
	require.NoError(t, err)
	assert.Equal(t, policy.ID, fetched.ID)
	assert.Equal(t, policy.Name, fetched.Name)
}

func TestSLAPolicyService_GetByIDForTenant_BlocksCrossTenantRead(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	otherTenant := createOtherPolicyTenant(ctx, t, client)
	policy, err := svc.CreateSLAPolicy(ctx, sampleCreateRequest(tenant.ID))
	require.NoError(t, err)

	fetched, err := svc.GetSLAPolicyByIDForTenant(ctx, policy.ID, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, policy.ID, fetched.ID)

	_, err = svc.GetSLAPolicyByIDForTenant(ctx, policy.ID, otherTenant.ID)
	assert.True(t, ent.IsNotFound(err), "cross-tenant read should look like a missing policy")
}

func TestSLAPolicyService_Query_FiltersByTenantAndSortsByScore(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenantA := createPolicyTenant(ctx, t, client)
	tenantB, err := client.Tenant.Create().
		SetName("Other Tenant").
		SetCode("other-tenant").
		SetDomain("other.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	scores := []int{50, 90, 70, 30}
	for i, score := range scores {
		req := sampleCreateRequest(tenantA.ID)
		req.Name = fmt.Sprintf("Policy %d", i)
		req.PriorityScore = score
		_, err := svc.CreateSLAPolicy(ctx, req)
		require.NoError(t, err)
	}

	otherReq := sampleCreateRequest(tenantB.ID)
	otherReq.Name = "Other Tenant Policy"
	_, err = svc.CreateSLAPolicy(ctx, otherReq)
	require.NoError(t, err)

	results, err := svc.QuerySLAPolicies(ctx, tenantA.ID)
	require.NoError(t, err)
	require.Len(t, results, len(scores))

	for i := 1; i < len(results); i++ {
		assert.GreaterOrEqual(t, results[i-1].PriorityScore, results[i].PriorityScore,
			"results must be sorted by PriorityScore descending")
		assert.Equal(t, tenantA.ID, results[i].TenantID)
	}
}

func TestSLAPolicyService_Query_EmptyTenantReturnsEmptySlice(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	results, err := svc.QuerySLAPolicies(ctx, 999)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSLAPolicyService_Update_AppliesOnlyProvidedFields(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	policy, err := svc.CreateSLAPolicy(ctx, sampleCreateRequest(tenant.ID))
	require.NoError(t, err)

	newName := "Renamed SLA"
	newScore := 99
	inactive := false
	updated, err := svc.UpdateSLAPolicy(ctx, policy.ID, dto.UpdateSLAPolicyRequest{
		Name:          stringPtr(newName),
		PriorityScore: slaPolicyIntPtr(newScore),
		IsActive:      slaPolicyBoolPtr(inactive),
	})
	require.NoError(t, err)

	assert.Equal(t, newName, updated.Name)
	assert.Equal(t, newScore, updated.PriorityScore)
	assert.False(t, updated.IsActive)
	assert.Equal(t, policy.CustomerTier, updated.CustomerTier)
	assert.Equal(t, policy.ResponseTimeMinutes, updated.ResponseTimeMinutes)
	assert.Equal(t, policy.ResolutionTimeMinutes, updated.ResolutionTimeMinutes)
}

func TestSLAPolicyService_UpdateForTenant_BlocksCrossTenantUpdate(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	otherTenant := createOtherPolicyTenant(ctx, t, client)
	policy, err := svc.CreateSLAPolicy(ctx, sampleCreateRequest(tenant.ID))
	require.NoError(t, err)

	newName := "Cross Tenant Update"
	_, err = svc.UpdateSLAPolicyForTenant(ctx, policy.ID, otherTenant.ID, dto.UpdateSLAPolicyRequest{
		Name: stringPtr(newName),
	})
	assert.True(t, ent.IsNotFound(err), "cross-tenant update should not find the policy")

	fetched, err := svc.GetSLAPolicyByID(ctx, policy.ID)
	require.NoError(t, err)
	assert.NotEqual(t, newName, fetched.Name)

	updated, err := svc.UpdateSLAPolicyForTenant(ctx, policy.ID, tenant.ID, dto.UpdateSLAPolicyRequest{
		Name: stringPtr(newName),
	})
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
}

func TestSLAPolicyService_Delete_RemovesPolicy(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	policy, err := svc.CreateSLAPolicy(ctx, sampleCreateRequest(tenant.ID))
	require.NoError(t, err)

	err = svc.DeleteSLAPolicy(ctx, policy.ID)
	require.NoError(t, err)

	_, err = svc.GetSLAPolicyByID(ctx, policy.ID)
	assert.Error(t, err, "expected error fetching deleted SLA policy")
}

func TestSLAPolicyService_DeleteForTenant_BlocksCrossTenantDelete(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	otherTenant := createOtherPolicyTenant(ctx, t, client)
	policy, err := svc.CreateSLAPolicy(ctx, sampleCreateRequest(tenant.ID))
	require.NoError(t, err)

	err = svc.DeleteSLAPolicyForTenant(ctx, policy.ID, otherTenant.ID)
	assert.True(t, ent.IsNotFound(err), "cross-tenant delete should not find the policy")

	_, err = svc.GetSLAPolicyByID(ctx, policy.ID)
	require.NoError(t, err)

	err = svc.DeleteSLAPolicyForTenant(ctx, policy.ID, tenant.ID)
	require.NoError(t, err)
	_, err = svc.GetSLAPolicyByID(ctx, policy.ID)
	assert.True(t, ent.IsNotFound(err))
}

func TestSLAPolicyService_Match_PrefersFullMatch(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)

	fullMatch := sampleCreateRequest(tenant.ID)
	fullMatch.Name = "Full Match"
	fullMatch.CustomerTier = stringPtr("platinum")
	fullMatch.TicketType = stringPtr("incident")
	fullMatch.Priority = stringPtr("critical")
	fullMatch.PriorityScore = 50
	_, err := svc.CreateSLAPolicy(ctx, fullMatch)
	require.NoError(t, err)

	typeOnly := sampleCreateRequest(tenant.ID)
	typeOnly.Name = "Incident Critical"
	typeOnly.CustomerTier = stringPtr("")
	typeOnly.TicketType = stringPtr("incident")
	typeOnly.Priority = stringPtr("critical")
	typeOnly.PriorityScore = 70
	_, err = svc.CreateSLAPolicy(ctx, typeOnly)
	require.NoError(t, err)

	priorityOnly := sampleCreateRequest(tenant.ID)
	priorityOnly.Name = "Critical Only"
	priorityOnly.CustomerTier = stringPtr("")
	priorityOnly.TicketType = stringPtr("")
	priorityOnly.Priority = stringPtr("critical")
	priorityOnly.PriorityScore = 90
	_, err = svc.CreateSLAPolicy(ctx, priorityOnly)
	require.NoError(t, err)

	matched, err := svc.MatchSLAPolicy(ctx, tenant.ID, "incident", "critical", "platinum")
	require.NoError(t, err)
	assert.Equal(t, "Full Match", matched.Name,
		"full match (customer_tier + ticket_type + priority) must beat type+priority and priority-only policies")
}

func TestSLAPolicyService_Match_FallsBackToPriorityOnly(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)

	priorityOnly := sampleCreateRequest(tenant.ID)
	priorityOnly.Name = "High Priority"
	priorityOnly.CustomerTier = stringPtr("")
	priorityOnly.TicketType = stringPtr("")
	priorityOnly.Priority = stringPtr("high")
	priorityOnly.PriorityScore = 30
	_, err := svc.CreateSLAPolicy(ctx, priorityOnly)
	require.NoError(t, err)

	unrelated := sampleCreateRequest(tenant.ID)
	unrelated.Name = "Low Priority Service Request"
	unrelated.CustomerTier = stringPtr("")
	unrelated.TicketType = stringPtr("request")
	unrelated.Priority = stringPtr("low")
	unrelated.PriorityScore = 10
	_, err = svc.CreateSLAPolicy(ctx, unrelated)
	require.NoError(t, err)

	matched, err := svc.MatchSLAPolicy(ctx, tenant.ID, "incident", "high", "bronze")
	require.NoError(t, err)
	assert.Equal(t, "High Priority", matched.Name)
}

func TestSLAPolicyService_Match_NoMatchReturnsDefault(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)

	defaultPolicy := sampleCreateRequest(tenant.ID)
	defaultPolicy.Name = "Catch-All Default"
	defaultPolicy.CustomerTier = stringPtr("")
	defaultPolicy.TicketType = stringPtr("")
	defaultPolicy.Priority = stringPtr("")
	defaultPolicy.PriorityScore = 5
	_, err := svc.CreateSLAPolicy(ctx, defaultPolicy)
	require.NoError(t, err)

	matched, err := svc.MatchSLAPolicy(ctx, tenant.ID, "request", "low", "bronze")
	require.NoError(t, err)
	assert.Equal(t, "Catch-All Default", matched.Name)
}

func TestSLAPolicyService_Match_NoActivePoliciesErrors(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)

	inactive := sampleCreateRequest(tenant.ID)
	inactive.Name = "Inactive"
	inactive.IsActive = false
	_, err := svc.CreateSLAPolicy(ctx, inactive)
	require.NoError(t, err)

	_, err = svc.MatchSLAPolicy(ctx, tenant.ID, "incident", "high", "gold")
	assert.Error(t, err, "matching should fail when no active policy exists")
}

func TestSLAPolicyService_Match_SkipsInactivePoliciesEvenWhenMatching(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)

	active := sampleCreateRequest(tenant.ID)
	active.Name = "Active Default"
	active.CustomerTier = stringPtr("")
	active.TicketType = stringPtr("")
	active.Priority = stringPtr("")
	active.PriorityScore = 1
	_, err := svc.CreateSLAPolicy(ctx, active)
	require.NoError(t, err)

	inactiveMatch := sampleCreateRequest(tenant.ID)
	inactiveMatch.Name = "Inactive Match"
	inactiveMatch.Priority = stringPtr("critical")
	inactiveMatch.PriorityScore = 999
	inactiveMatch.IsActive = false
	_, err = svc.CreateSLAPolicy(ctx, inactiveMatch)
	require.NoError(t, err)

	matched, err := svc.MatchSLAPolicy(ctx, tenant.ID, "incident", "critical", "gold")
	require.NoError(t, err)
	assert.Equal(t, "Active Default", matched.Name,
		"inactive policies must be skipped even when their attributes match")
}

func TestSLAPolicyService_CalculateExpire_WithoutBusinessHoursUses24h(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	req := sampleCreateRequest(tenant.ID)
	req.BusinessHours = nil
	req.ExcludeWeekends = false
	policy, err := svc.CreateSLAPolicy(ctx, req)
	require.NoError(t, err)

	start := time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC)
	got := svc.CalculateSLAExpireTime(ctx, policy, start)

	want := start.Add(time.Duration(policy.ResolutionTimeMinutes) * time.Minute)
	assert.Equal(t, want, got)
}

func TestSLAPolicyService_CalculateExpire_UsesBusinessHoursWithinSameDay(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	req := sampleCreateRequest(tenant.ID)
	req.ResolutionTimeMinutes = 120
	req.BusinessHours = map[string]interface{}{"start": "09:00", "end": "18:00"}
	req.ExcludeWeekends = true
	policy, err := svc.CreateSLAPolicy(ctx, req)
	require.NoError(t, err)

	start := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC) // Monday
	got := svc.CalculateSLAExpireTime(ctx, policy, start)

	assert.Equal(t, time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC), got)
}

func TestSLAPolicyService_CalculateExpire_CrossesBusinessDayBoundary(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	req := sampleCreateRequest(tenant.ID)
	req.ResolutionTimeMinutes = 120
	req.BusinessHours = map[string]interface{}{"start": "09:00", "end": "18:00"}
	req.ExcludeWeekends = true
	policy, err := svc.CreateSLAPolicy(ctx, req)
	require.NoError(t, err)

	start := time.Date(2026, 6, 22, 17, 0, 0, 0, time.UTC) // Monday
	got := svc.CalculateSLAExpireTime(ctx, policy, start)

	assert.Equal(t, time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC), got)
}

func TestSLAPolicyService_CalculateExpire_SkipsWeekend(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	req := sampleCreateRequest(tenant.ID)
	req.ResolutionTimeMinutes = 120
	req.BusinessHours = map[string]interface{}{"start": "09:00", "end": "18:00"}
	req.ExcludeWeekends = true
	policy, err := svc.CreateSLAPolicy(ctx, req)
	require.NoError(t, err)

	start := time.Date(2026, 6, 26, 17, 0, 0, 0, time.UTC) // Friday
	got := svc.CalculateSLAExpireTime(ctx, policy, start)

	assert.Equal(t, time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC), got)
}

func TestSLAPolicyService_CalculateExpire_ExcludeWeekendsWithoutBusinessHours(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	req := sampleCreateRequest(tenant.ID)
	req.ResolutionTimeMinutes = 180
	req.BusinessHours = nil
	req.ExcludeWeekends = true
	policy, err := svc.CreateSLAPolicy(ctx, req)
	require.NoError(t, err)

	start := time.Date(2026, 6, 26, 23, 0, 0, 0, time.UTC) // Friday
	got := svc.CalculateSLAExpireTime(ctx, policy, start)

	assert.Equal(t, time.Date(2026, 6, 29, 2, 0, 0, 0, time.UTC), got)
}

func TestSLAPolicyService_ComplianceRate_NoTicketsReturns100(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	rate, err := svc.GetSLAComplianceRate(ctx, 999, time.Now().Add(-24*time.Hour), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 100.0, rate)
}

func TestSLAPolicyService_ComplianceRate_FiltersTenantAndDateRange(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	otherTenant, err := client.Tenant.Create().
		SetName("Policy Tenant Other").
		SetCode("policy-tenant-other").
		SetDomain("policy-other.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("sla-policy-user").
		SetEmail("sla-policy-user@example.com").
		SetName("SLA Policy User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	otherUser, err := client.User.Create().
		SetUsername("sla-policy-other-user").
		SetEmail("sla-policy-other-user@example.com").
		SetName("SLA Policy Other User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(otherTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	slaDef, err := client.SLADefinition.Create().
		SetName("Compliance SLA").
		SetDescription("Compliance calculation test SLA").
		SetPriority("high").
		SetResponseTime(30).
		SetResolutionTime(240).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	otherSlaDef, err := client.SLADefinition.Create().
		SetName("Other Compliance SLA").
		SetDescription("Other tenant SLA").
		SetPriority("high").
		SetResponseTime(30).
		SetResolutionTime(240).
		SetTenantID(otherTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	inRange := start.Add(2 * time.Hour)
	outOfRange := start.Add(-2 * time.Hour)

	var targetTickets []*ent.Ticket
	for i := 0; i < 4; i++ {
		ticket, err := client.Ticket.Create().
			SetTitle("Tenant SLA ticket").
			SetTicketNumber("SLA-POLICY-" + string(rune('A'+i))).
			SetRequesterID(user.ID).
			SetTenantID(tenant.ID).
			SetSLADefinitionID(slaDef.ID).
			SetCreatedAt(inRange).
			Save(ctx)
		require.NoError(t, err)
		targetTickets = append(targetTickets, ticket)
	}

	otherTicket, err := client.Ticket.Create().
		SetTitle("Other tenant SLA ticket").
		SetTicketNumber("SLA-POLICY-OTHER").
		SetRequesterID(otherUser.ID).
		SetTenantID(otherTenant.ID).
		SetSLADefinitionID(otherSlaDef.ID).
		SetCreatedAt(inRange).
		Save(ctx)
	require.NoError(t, err)

	oldTicket, err := client.Ticket.Create().
		SetTitle("Old tenant SLA ticket").
		SetTicketNumber("SLA-POLICY-OLD").
		SetRequesterID(user.ID).
		SetTenantID(tenant.ID).
		SetSLADefinitionID(slaDef.ID).
		SetCreatedAt(outOfRange).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SLAViolation.Create().
		SetSLADefinitionID(slaDef.ID).
		SetTicketID(targetTickets[0].ID).
		SetViolationType("resolution").
		SetTenantID(tenant.ID).
		SetCreatedAt(inRange).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SLAViolation.Create().
		SetSLADefinitionID(slaDef.ID).
		SetTicketID(targetTickets[0].ID).
		SetViolationType("response").
		SetTenantID(tenant.ID).
		SetCreatedAt(inRange).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SLAViolation.Create().
		SetSLADefinitionID(otherSlaDef.ID).
		SetTicketID(otherTicket.ID).
		SetViolationType("resolution").
		SetTenantID(otherTenant.ID).
		SetCreatedAt(inRange).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SLAViolation.Create().
		SetSLADefinitionID(slaDef.ID).
		SetTicketID(oldTicket.ID).
		SetViolationType("resolution").
		SetTenantID(tenant.ID).
		SetCreatedAt(outOfRange).
		Save(ctx)
	require.NoError(t, err)

	rate, err := svc.GetSLAComplianceRate(ctx, tenant.ID, start, end)
	require.NoError(t, err)
	assert.Equal(t, 75.0, rate)
}

func TestSLAPolicyService_ComplianceRate_IgnoresMismatchedViolationTenant(t *testing.T) {
	client, svc, ctx := setupSLAPolicyTest(t)
	defer client.Close()

	tenant := createPolicyTenant(ctx, t, client)
	otherTenant := createOtherPolicyTenant(ctx, t, client)
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	inRange := start.Add(2 * time.Hour)

	user, err := client.User.Create().
		SetUsername("sla-policy-mismatch-user").
		SetEmail("sla-policy-mismatch-user@example.com").
		SetName("SLA Policy Mismatch User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	otherUser, err := client.User.Create().
		SetUsername("sla-policy-mismatch-other-user").
		SetEmail("sla-policy-mismatch-other-user@example.com").
		SetName("SLA Policy Mismatch Other User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(otherTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	slaDef, err := client.SLADefinition.Create().
		SetName("Mismatch Compliance SLA").
		SetDescription("Compliance mismatch test SLA").
		SetPriority("high").
		SetResponseTime(30).
		SetResolutionTime(240).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	var targetTickets []*ent.Ticket
	for i := 0; i < 2; i++ {
		ticket, err := client.Ticket.Create().
			SetTitle("Tenant mismatch SLA ticket").
			SetTicketNumber("SLA-POLICY-MISMATCH-" + string(rune('A'+i))).
			SetRequesterID(user.ID).
			SetTenantID(tenant.ID).
			SetSLADefinitionID(slaDef.ID).
			SetCreatedAt(inRange).
			Save(ctx)
		require.NoError(t, err)
		targetTickets = append(targetTickets, ticket)
	}

	otherTicket, err := client.Ticket.Create().
		SetTitle("Other tenant mismatch SLA ticket").
		SetTicketNumber("SLA-POLICY-MISMATCH-OTHER").
		SetRequesterID(otherUser.ID).
		SetTenantID(otherTenant.ID).
		SetSLADefinitionID(slaDef.ID).
		SetCreatedAt(inRange).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SLAViolation.Create().
		SetSLADefinitionID(slaDef.ID).
		SetTicketID(targetTickets[0].ID).
		SetViolationType("resolution").
		SetTenantID(tenant.ID).
		SetCreatedAt(inRange).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SLAViolation.Create().
		SetSLADefinitionID(slaDef.ID).
		SetTicketID(otherTicket.ID).
		SetViolationType("resolution").
		SetTenantID(tenant.ID).
		SetCreatedAt(inRange).
		Save(ctx)
	require.NoError(t, err)

	rate, err := svc.GetSLAComplianceRate(ctx, tenant.ID, start, end)
	require.NoError(t, err)
	assert.Equal(t, 50.0, rate)
}
