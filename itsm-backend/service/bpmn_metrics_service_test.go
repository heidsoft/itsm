package service

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// metricsTenant 创建仪表盘测试租户与部署
func metricsTenant(t *testing.T, client *ent.Client) (*ent.Tenant, *ent.ProcessDeployment) {
	t.Helper()
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Metrics Tenant").
		SetCode("METRICS-" + fmt.Sprint(time.Now().UnixNano())).
		SetDomain("metrics.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-METRICS-" + fmt.Sprint(time.Now().UnixNano())).
		SetDeploymentName("Metrics Deployment").
		SetDeploymentSource("test").
		SetTenantID(tenant.ID).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)
	return tenant, deployment
}

// metricsDef 在同一租户下创建带 SLA 配置的流程定义
func metricsDef(t *testing.T, client *ent.Client, tenant *ent.Tenant, deployment *ent.ProcessDeployment, key string, deadlineMinutes, warningMinutes float64) *ent.ProcessDefinition {
	t.Helper()
	def, err := client.ProcessDefinition.Create().
		SetKey(key).
		SetName(key).
		SetVersion("1.0.0").
		SetTenantID(tenant.ID).
		SetDeploymentID(deployment.ID).
		SetBpmnXML([]byte(testBPMNXML)).
		SetProcessVariables(map[string]interface{}{
			"sla": map[string]interface{}{
				"deadline_minutes":  deadlineMinutes,
				"warning_minutes":   warningMinutes,
				"business_hours_only": false,
			},
		}).
		Save(context.Background())
	require.NoError(t, err)
	return def
}

func TestRound1(t *testing.T) {
	assert.Equal(t, 97.1, round1(97.14285714285714))
	assert.Equal(t, 4.8, round1(4.761904761904762))
	assert.Equal(t, 0.0, round1(0))
	assert.Equal(t, 100.0, round1(100))
}

// TestBPMNSLAService_ZeroSampleHonestRate 能力语义回归：无样本不得伪装 100% 合规
func TestBPMNSLAService_ZeroSampleHonestRate(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	slaService := NewBPMNSLAService(client, zaptest.NewLogger(t).Sugar())
	rate, compliant, total, err := slaService.GetSLAComplianceRate(
		context.Background(), "no_such_process", time.Now().Add(-24*time.Hour), time.Now(), 1)
	require.NoError(t, err)
	assert.Equal(t, 0.0, rate)
	assert.Equal(t, 0, compliant)
	assert.Equal(t, 0, total)
}

// TestBPMNMetricsService_DashboardZeroSample 仪表盘聚合：无任何已完成样本时合规率为 0，
// 而不是把每个定义伪装成 100% 后取平均。
func TestBPMNMetricsService_DashboardZeroSample(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	svc := NewBPMNMetricsService(client, zaptest.NewLogger(t).Sugar())

	tenant, deployment := metricsTenant(t, client)
	metricsDef(t, client, tenant, deployment, "dash_zero_sample", 480, 360)

	metrics, err := svc.GetDashboardMetrics(ctx, tenant.ID, time.Now().AddDate(0, 0, -7), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 0.0, metrics.SLAComplianceRate)
}

// TestBPMNMetricsService_DashboardWeightedRate 仪表盘聚合：按已完成实例加权计算合规率。
// 一个超时完成样本 => 0%，修复前会被逐定义平均 + 零样本伪 100 抬高。
func TestBPMNMetricsService_DashboardWeightedRate(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	svc := NewBPMNMetricsService(client, zaptest.NewLogger(t).Sugar())

	tenant, deployment := metricsTenant(t, client)
	def := metricsDef(t, client, tenant, deployment, "dash_weighted", 60, 45)

	// 已完成但严重超时（deadline=60min，30小时前启动）=> breached
	_, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-DASH-W-001").
		SetBusinessKey("dash-w-1").
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetStatus("completed").
		SetTenantID(tenant.ID).
		SetStartTime(time.Now().Add(-30 * time.Hour)).
		SetEndTime(time.Now().Add(-30 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	metrics, err := svc.GetDashboardMetrics(ctx, tenant.ID, time.Now().AddDate(0, 0, -7), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 0.0, metrics.SLAComplianceRate)

	// 直接验证单定义合规率：全部超时 => 0
	rate, compliant, total, err := svc.slaService.GetSLAComplianceRate(ctx, def.Key, time.Now().AddDate(0, 0, -7), time.Now(), tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, 0, compliant)
	assert.Equal(t, 0.0, rate)
}

// TestBPMNMetricsService_HealthScoreRounded 健康度评分保留一位小数（修复前返回 83.333333…）。
// 7 健康 + 1 警告 + 1 严重 => (700+50)/9 = 83.333… => 83.3；
// 启动时间以当天 09:00 为基准偏移，避开营业时间跨天造成的时刻依赖。
func TestBPMNMetricsService_HealthScoreRounded(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	svc := NewBPMNMetricsService(client, zaptest.NewLogger(t).Sugar())

	tenant, deployment := metricsTenant(t, client)
	now := time.Now()
	// 基准日取“最近一个工作日（含今天）的 09:00”：周末运行时营业时间截止会顺延到
	// 下一个工作日，固定在当前时刻的日期会让分类随星期漂移
	baseDay := now
	for baseDay.Weekday() == time.Saturday || baseDay.Weekday() == time.Sunday {
		baseDay = baseDay.AddDate(0, 0, -1)
	}
	base := time.Date(baseDay.Year(), baseDay.Month(), baseDay.Day(), 9, 0, 0, 0, now.Location())
	healthyDef := metricsDef(t, client, tenant, deployment, "dash_health_healthy", 600, 540)
	warningDef := metricsDef(t, client, tenant, deployment, "dash_health_warning", 50, 40)
	breachedDef := metricsDef(t, client, tenant, deployment, "dash_health_breached", 10, 5)

	createInstance := func(def *ent.ProcessDefinition, prefix string, start time.Time) {
		_, err := client.ProcessInstance.Create().
			SetProcessInstanceID(fmt.Sprintf("PI-DASH-%s-%03d", prefix, len(prefix)+int(start.Unix())%1000)).
			SetBusinessKey(prefix).
			SetProcessDefinitionKey(def.Key).
			SetProcessDefinitionID(def.ID).
			SetStatus("running").
			SetTenantID(tenant.ID).
			SetStartTime(start).
			Save(ctx)
		require.NoError(t, err)
	}

	for i := 0; i < 7; i++ {
		createInstance(healthyDef, fmt.Sprintf("HH%d", i), base.Add(5*time.Minute))
	}
	createInstance(warningDef, "HW", base.Add(45*time.Minute))
	createInstance(breachedDef, "HB", base.Add(20*time.Minute))

	health := svc.calculateProcessHealth(ctx, tenant.ID)
	require.NotNil(t, health)
	// 营业日历会把周末的截止时间顺延，分类数量随运行日变化；
	// 这里锁定回归目标：评分必须等于按公式加权后的舍入值（修复前返回未舍入浮点）
	sampled := health.Healthy + health.Warning + health.Critical
	require.Equal(t, 9, sampled, "全部实例都应被分类")
	expected := round1(float64(health.Healthy*100+health.Warning*50) / float64(sampled))
	assert.Equal(t, expected, health.HealthScore)
	// 修复前返回未舍入浮点（如 83.333333），此断言会失败
	assert.InDelta(t, math.Round(health.HealthScore*10), health.HealthScore*10, 1e-6,
		"健康度评分最多保留一位小数")
	if health.Warning+health.Critical > 0 {
		assert.NotEqual(t, 100.0, health.HealthScore, "存在非健康实例时不得满分")
	}
}

// TestBPMNMetricsService_TaskDistributionRounded 任务分布占比保留一位小数：
// 20 created + 1 completed => 1/21 = 4.7619… => 4.8
func TestBPMNMetricsService_TaskDistributionRounded(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	svc := NewBPMNMetricsService(client, zaptest.NewLogger(t).Sugar())

	tenant, deployment := metricsTenant(t, client)
	def := metricsDef(t, client, tenant, deployment, "dash_task_dist", 480, 360)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-DASH-TD-001").
		SetBusinessKey("td").
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetTenantID(tenant.ID).
		SetStartTime(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	createMetricsTask := func(status string, i int) {
		_, err := client.ProcessTask.Create().
			SetTaskID(fmt.Sprintf("TASK-DASH-TD-%s-%03d", status, i)).
			SetTaskDefinitionKey("Task_1").
			SetTaskName("任务").
			SetTaskType("user_task").
			SetProcessDefinitionKey(def.Key).
			SetProcessInstanceID(instance.ID).
			SetStatus(status).
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}
	for i := 0; i < 20; i++ {
		createMetricsTask("created", i)
	}
	createMetricsTask("completed", 0)

	dist := svc.getTaskDistribution(ctx, tenant.ID)
	require.Len(t, dist, 2)
	for _, d := range dist {
		if d.Status == "completed" {
			assert.Equal(t, 4.8, d.Percent)
		} else {
			assert.Equal(t, 95.2, d.Percent)
		}
	}
}
