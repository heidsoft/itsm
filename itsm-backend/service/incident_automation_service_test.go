package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incidentalert"
	"itsm-backend/ent/incidentevent"
	"itsm-backend/ent/incidentmetric"
	"itsm-backend/ent/incidentruleexecution"
	entuser "itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func createAutomationIncident(
	t *testing.T,
	ctx context.Context,
	client *ent.Client,
	tenantID, reporterID int,
	number string,
) *ent.Incident {
	t.Helper()
	entity, err := client.Incident.Create().
		SetTitle("Automation incident").
		SetStatus("new").
		SetPriority("high").
		SetSeverity("high").
		SetIncidentNumber(number).
		SetReporterID(reporterID).
		SetTenantID(tenantID).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	return entity
}

func TestIncidentAlertingLifecycleIsTenantScopedAndAudited(t *testing.T) {
	client, _, ctx := setupIncidentTest(t)
	defer client.Close()
	tenant, err := createIncidentTestTenant(ctx, client, "alert-lifecycle")
	require.NoError(t, err)
	otherTenant, err := createIncidentTestTenant(ctx, client, "alert-lifecycle-other")
	require.NoError(t, err)
	actor, err := createIncidentTestUser(ctx, client, tenant.ID, "alert-actor")
	require.NoError(t, err)
	_, err = actor.Update().SetRole(entuser.RoleAdmin).Save(ctx)
	require.NoError(t, err)
	foreignActor, err := createIncidentTestUser(ctx, client, otherTenant.ID, "alert-foreign")
	require.NoError(t, err)
	incidentEntity := createAutomationIncident(t, ctx, client, tenant.ID, actor.ID, "INC-ALERT-LIFECYCLE")
	alerting := NewIncidentAlertingService(client, zaptest.NewLogger(t).Sugar())
	triggeredAt := time.Now().Add(-5 * time.Minute).Truncate(time.Second)

	alert, err := alerting.CreateIncidentAlert(ctx, &dto.CreateIncidentAlertRequest{
		IncidentID: incidentEntity.ID, AlertType: "monitoring", AlertName: "CPU high",
		Message: "CPU exceeded threshold", Severity: "high", TriggeredAt: &triggeredAt,
	}, tenant.ID)
	require.NoError(t, err)
	assert.WithinDuration(t, triggeredAt, alert.TriggeredAt, time.Second)

	err = alerting.AcknowledgeAlert(ctx, alert.ID, foreignActor.ID, tenant.ID)
	require.ErrorContains(t, err, "actor not found")
	require.NoError(t, alerting.AcknowledgeAlert(ctx, alert.ID, actor.ID, tenant.ID))
	err = alerting.AcknowledgeAlert(ctx, alert.ID, actor.ID, tenant.ID)
	require.ErrorContains(t, err, "alert not found")
	require.NoError(t, alerting.ResolveAlert(ctx, alert.ID, actor.ID, tenant.ID))
	err = alerting.ResolveAlert(ctx, alert.ID, actor.ID, tenant.ID)
	require.ErrorContains(t, err, "alert not found")

	stored, err := client.IncidentAlert.Query().Where(incidentalert.IDEQ(alert.ID)).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "resolved", stored.Status)
	assert.False(t, stored.AcknowledgedAt.IsZero())
	assert.False(t, stored.ResolvedAt.IsZero())
	events, err := client.IncidentEvent.Query().
		Where(incidentevent.IncidentIDEQ(incidentEntity.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, actor.ID, events[0].UserID)
	assert.Equal(t, actor.ID, events[1].UserID)
}

func TestIncidentAlertCreationRejectsCrossTenantAndThresholdRequiresIncident(t *testing.T) {
	client, _, ctx := setupIncidentTest(t)
	defer client.Close()
	tenantA, err := createIncidentTestTenant(ctx, client, "alert-boundary-a")
	require.NoError(t, err)
	tenantB, err := createIncidentTestTenant(ctx, client, "alert-boundary-b")
	require.NoError(t, err)
	userA, err := createIncidentTestUser(ctx, client, tenantA.ID, "alert-boundary-a")
	require.NoError(t, err)
	foreignIncident := createAutomationIncident(t, ctx, client, tenantB.ID, userA.ID, "INC-ALERT-FOREIGN")
	alerting := NewIncidentAlertingService(client, zaptest.NewLogger(t).Sugar())

	_, err = alerting.CreateIncidentAlert(ctx, &dto.CreateIncidentAlertRequest{
		IncidentID: foreignIncident.ID, AlertType: "monitoring", AlertName: "Foreign", Message: "Foreign",
	}, tenantA.ID)
	require.ErrorContains(t, err, "incident not found")
	err = alerting.ProcessThresholdAlerts(ctx, 0, "cpu", 95, 90, tenantA.ID)
	require.ErrorContains(t, err, "incident id is required")
	count, err := client.IncidentAlert.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count)
}

func TestPrometheusMetricCorrelationDoesNotFanOut(t *testing.T) {
	client, _, ctx := setupIncidentTest(t)
	defer client.Close()
	tenant, err := createIncidentTestTenant(ctx, client, "metric-correlation")
	require.NoError(t, err)
	reporter, err := createIncidentTestUser(ctx, client, tenant.ID, "metric-correlation")
	require.NoError(t, err)
	first := createAutomationIncident(t, ctx, client, tenant.ID, reporter.ID, "INC-METRIC-001")
	second := createAutomationIncident(t, ctx, client, tenant.ID, reporter.ID, "INC-METRIC-002")
	monitoring := NewIncidentMonitoringService(client, zaptest.NewLogger(t).Sugar())

	err = monitoring.createIncidentMetricFromPrometheus(ctx, map[string]string{
		"__name__": "cpu_usage_percent", "incident_id": fmt.Sprint(first.ID),
	}, "91.5", float64(time.Now().Unix()), tenant.ID)
	require.NoError(t, err)
	err = monitoring.createIncidentMetricFromPrometheus(ctx, map[string]string{
		"__name__": "memory_usage_percent",
	}, "88", float64(time.Now().Unix()), tenant.ID)
	require.ErrorContains(t, err, "must include")

	firstCount, err := client.IncidentMetric.Query().Where(incidentmetric.IncidentIDEQ(first.ID)).Count(ctx)
	require.NoError(t, err)
	secondCount, err := client.IncidentMetric.Query().Where(incidentmetric.IncidentIDEQ(second.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, firstCount)
	assert.Zero(t, secondCount)
}

func TestIncidentRuleActionFailureMarksExecutionFailed(t *testing.T) {
	client, _, ctx := setupIncidentTest(t)
	defer client.Close()
	tenant, err := createIncidentTestTenant(ctx, client, "rule-failure")
	require.NoError(t, err)
	reporter, err := createIncidentTestUser(ctx, client, tenant.ID, "rule-failure")
	require.NoError(t, err)
	incidentEntity := createAutomationIncident(t, ctx, client, tenant.ID, reporter.ID, "INC-RULE-FAILURE")
	rule, err := client.IncidentRule.Create().
		SetName("Invalid assignment").
		SetRuleType("assignment").
		SetConditions(map[string]interface{}{"priority": []string{"high"}}).
		SetActions([]map[string]interface{}{{"type": "assign", "assignee_id": 999999}}).
		SetIsActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	rule, err = client.IncidentRule.Get(ctx, rule.ID)
	require.NoError(t, err)
	incidentEntity, err = client.Incident.Get(ctx, incidentEntity.ID)
	require.NoError(t, err)
	engine := NewIncidentRuleEngine(client, zaptest.NewLogger(t).Sugar())

	err = engine.ExecuteRule(ctx, rule, incidentEntity, tenant.ID)
	require.ErrorContains(t, err, "rule action failed")
	execution, err := client.IncidentRuleExecution.Query().
		Where(incidentruleexecution.RuleIDEQ(rule.ID)).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "failed", execution.Status)
	updatedRule, err := client.IncidentRule.Get(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, updatedRule.ExecutionCount)
}

func TestIncidentCreationUsesFormalRuleEngine(t *testing.T) {
	client, incidentService, ctx := setupIncidentTest(t)
	defer client.Close()
	tenant, err := createIncidentTestTenant(ctx, client, "formal-rule-engine")
	require.NoError(t, err)
	reporter, err := createIncidentTestUser(ctx, client, tenant.ID, "formal-rule-engine")
	require.NoError(t, err)
	_, err = client.IncidentRule.Create().
		SetName("Collect creation metric").
		SetRuleType("metric").
		SetConditions(map[string]interface{}{"priority": []string{"high"}}).
		SetActions([]map[string]interface{}{{
			"type": "collect_metric", "metric_type": "automation", "metric_name": "rule_applied",
			"metric_value": 1.0, "unit": "count",
		}}).
		SetIsActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	incidentService.SetRuleEngine(NewIncidentRuleEngine(client, zaptest.NewLogger(t).Sugar()))

	created, err := incidentService.CreateIncident(ctx, &dto.CreateIncidentRequest{
		Title: "Formal engine incident", Priority: "high", Severity: "high",
	}, tenant.ID, reporter.ID)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		count, countErr := client.IncidentMetric.Query().
			Where(incidentmetric.IncidentIDEQ(created.ID)).
			Count(ctx)
		return countErr == nil && count == 1
	}, 2*time.Second, 20*time.Millisecond)
	execution, err := client.IncidentRuleExecution.Query().
		Where(incidentruleexecution.IncidentIDEQ(created.ID)).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "completed", execution.Status)
}
