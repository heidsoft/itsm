package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/sladefinition"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupSLATemplateTest(t *testing.T) (*ent.Client, *SLATemplateService, context.Context, *ent.Tenant) {
	t.Helper()
	dsn := fmt.Sprintf("file:sla_tpl_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("SLA Template Tenant").
		SetCode("sla-tpl-" + fmt.Sprint(time.Now().UnixNano())).
		SetDomain("sla-tpl.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	return client, NewSLATemplateService(client, logger), ctx, tenant
}

func countTemplateDefinitions(t *testing.T, client *ent.Client, tenantID int) int {
	t.Helper()
	count, err := client.SLADefinition.Query().
		Where(sladefinition.TenantIDEQ(tenantID)).
		Count(context.Background())
	require.NoError(t, err)
	return count
}

// TestSLATemplateInstallStatesComeFromDatabase 安装事实必须来自数据库：
// 进程重启（换一个 service 实例）后仍然知道装过，否则页面会把已装模板显示成「未安装」。
func TestSLATemplateInstallStatesComeFromDatabase(t *testing.T) {
	client, svc, ctx, tenant := setupSLATemplateTest(t)
	defer client.Close()

	states, err := svc.InstallStates(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Empty(t, states)

	result, err := svc.InstallTemplate(ctx, "incident_p1_critical", tenant.ID)
	require.NoError(t, err)
	require.True(t, result.Created)

	states, err = svc.InstallStates(ctx, tenant.ID)
	require.NoError(t, err)
	require.Contains(t, states, "incident_p1_critical")
	assert.Equal(t, result.SLADefinitionID, states["incident_p1_critical"].SLADefinitionID)
	assert.False(t, states["incident_p1_critical"].InstalledAt.IsZero())

	// 新实例 = 进程重启，旧实现的内存在这里会清空
	restarted := NewSLATemplateService(client, zaptest.NewLogger(t).Sugar())
	states, err = restarted.InstallStates(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, result.SLADefinitionID, states["incident_p1_critical"].SLADefinitionID)
}

// TestSLATemplateInstallIsIdempotent 同租户重复安装只产生一条定义。
func TestSLATemplateInstallIsIdempotent(t *testing.T) {
	client, svc, ctx, tenant := setupSLATemplateTest(t)
	defer client.Close()

	first, err := svc.InstallTemplate(ctx, "change_normal", tenant.ID)
	require.NoError(t, err)
	assert.True(t, first.Created)

	second, err := svc.InstallTemplate(ctx, "change_normal", tenant.ID)
	require.NoError(t, err)
	assert.False(t, second.Created)
	assert.True(t, second.WasAlreadyExist)
	assert.Equal(t, first.SLADefinitionID, second.SLADefinitionID)

	assert.Equal(t, 1, countTemplateDefinitions(t, client, tenant.ID))
}

// TestSLATemplateInstallSurvivesRename 幂等身份来自 conditions 里的 templateKey，
// 不是名称前缀；管理员改名后重装不会产生第二份相同配置。
func TestSLATemplateInstallSurvivesRename(t *testing.T) {
	client, svc, ctx, tenant := setupSLATemplateTest(t)
	defer client.Close()

	installed, err := svc.InstallTemplate(ctx, "incident_p2_high", tenant.ID)
	require.NoError(t, err)

	_, err = client.SLADefinition.UpdateOneID(installed.SLADefinitionID).
		SetName("客服部自定 SLA").
		SetDescription("").
		Save(ctx)
	require.NoError(t, err)

	again, err := svc.InstallTemplate(ctx, "incident_p2_high", tenant.ID)
	require.NoError(t, err)
	assert.True(t, again.WasAlreadyExist, "改名后应识别为同一模板，而不是再装一份")
	assert.Equal(t, installed.SLADefinitionID, again.SLADefinitionID)

	defs, err := client.SLADefinition.Query().
		Where(sladefinition.TenantIDEQ(tenant.ID)).All(ctx)
	require.NoError(t, err)
	assert.Len(t, defs, 1)
}

// TestSLATemplateInstallStatesReadsLegacyMarker 标记引入之前装的历史数据只能靠
// description 的「[Template] <key> - 」约定识别。
func TestSLATemplateInstallStatesReadsLegacyMarker(t *testing.T) {
	client, svc, ctx, tenant := setupSLATemplateTest(t)
	defer client.Close()

	legacy, err := client.SLADefinition.Create().
		SetTenantID(tenant.ID).
		SetName(slaTemplateNamePrefix + "事件 P3 中").
		SetDescription(fmt.Sprintf("%s%s - 老数据", slaTemplateNamePrefix, "incident_p3_medium")).
		SetPriority("medium").
		SetResponseTime(240).
		SetResolutionTime(1440).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)

	states, err := svc.InstallStates(ctx, tenant.ID)
	require.NoError(t, err)
	require.Contains(t, states, "incident_p3_medium")
	assert.Equal(t, legacy.ID, states["incident_p3_medium"].SLADefinitionID)
}

// TestSLATemplateInstallDoesNotMutateCatalog 标记写进副本而不是目录：目录 map 被所有
// 租户共享，污染后第二个租户会读到第一个租户的痕迹。
func TestSLATemplateInstallDoesNotMutateCatalog(t *testing.T) {
	client, svc, ctx, tenantA := setupSLATemplateTest(t)
	defer client.Close()

	tenantB, err := client.Tenant.Create().
		SetName("Tenant B").
		SetCode("sla-tpl-catalog-b").
		SetDomain("b2.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.InstallTemplate(ctx, "service_request_standard", tenantA.ID)
	require.NoError(t, err)
	_, err = svc.InstallTemplate(ctx, "service_request_standard", tenantB.ID)
	require.NoError(t, err)

	for _, tmpl := range svc.ListTemplates() {
		assert.NotContains(t, tmpl.Conditions, templateKeyMarker, "目录数据被安装路径污染: %s", tmpl.Key)
	}
}

// TestSLATemplateInstallStatesAreTenantScoped 租户 A 的安装不能出现在租户 B 的状态里。
func TestSLATemplateInstallStatesAreTenantScoped(t *testing.T) {
	client, svc, ctx, tenantA := setupSLATemplateTest(t)
	defer client.Close()

	tenantB, err := client.Tenant.Create().
		SetName("Tenant B").
		SetCode("sla-tpl-b").
		SetDomain("b.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	installed, err := svc.InstallTemplate(ctx, "change_emergency", tenantA.ID)
	require.NoError(t, err)

	statesB, err := svc.InstallStates(ctx, tenantB.ID)
	require.NoError(t, err)
	assert.NotContains(t, statesB, "change_emergency")

	second, err := svc.InstallTemplate(ctx, "change_emergency", tenantB.ID)
	require.NoError(t, err)
	assert.True(t, second.Created, "跨租户不得复用对方的定义")
	assert.NotEqual(t, installed.SLADefinitionID, second.SLADefinitionID)
}

// TestSLATemplateConcurrentInstallsCreateOneDefinition 重复点击/并发请求不能产出两份定义。
func TestSLATemplateConcurrentInstallsCreateOneDefinition(t *testing.T) {
	client, svc, ctx, tenant := setupSLATemplateTest(t)
	defer client.Close()

	var wg sync.WaitGroup
	created := make([]bool, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			result, err := svc.InstallTemplate(ctx, "incident_p1_critical", tenant.ID)
			if assert.NoError(t, err) {
				created[i] = result.Created
			}
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 1, countTemplateDefinitions(t, client, tenant.ID))
	assert.Equal(t, 1, countTrue(created), "并发安装只允许一个请求声称新建")
}

func countTrue(values []bool) int {
	n := 0
	for _, v := range values {
		if v {
			n++
		}
	}
	return n
}

// TestSLATemplateUnknownKeyIsNotFound 未知模板必须是可识别的 not-found，
// handler 才能映射成 404 而不是 400。
func TestSLATemplateUnknownKeyIsNotFound(t *testing.T) {
	client, svc, ctx, tenant := setupSLATemplateTest(t)
	defer client.Close()

	_, err := svc.InstallTemplate(ctx, "does_not_exist", tenant.ID)
	assert.True(t, errors.Is(err, ErrSLATemplateNotFound), "got %v", err)

	_, err = svc.GetTemplate("does_not_exist")
	assert.True(t, errors.Is(err, ErrSLATemplateNotFound), "got %v", err)
}

// TestSLATemplateInstallStatesIgnoreTenantAuthoredDefinitions 租户自建的 SLA 定义
// 不能因为名称里带 template 字样就被认成模板安装产物。
func TestSLATemplateInstallStatesIgnoreTenantAuthoredDefinitions(t *testing.T) {
	client, svc, ctx, tenant := setupSLATemplateTest(t)
	defer client.Close()

	_, err := client.SLADefinition.Create().
		SetTenantID(tenant.ID).
		SetName("template 内部采购流程 SLA").
		SetDescription("template 字样出现在描述里也不算").
		SetPriority("medium").
		SetResponseTime(120).
		SetResolutionTime(960).
		SetConditions(map[string]interface{}{"ticket_type": []string{"incident"}}).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)

	states, err := svc.InstallStates(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Empty(t, states)

	result, err := svc.InstallTemplate(ctx, "incident_p3_medium", tenant.ID)
	require.NoError(t, err)
	assert.True(t, result.Created)
}
