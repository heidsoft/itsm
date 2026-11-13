package service

import (
	"context"
	"errors"
	"testing"

	"itsm-backend/common"
	domainSR "itsm-backend/domain/servicerequest"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 本文件是 prod O1 的回归锁。
//
// 实测现象：服务目录里「IT服务台」（service_type=custom，咨询类、无云资源）
// 的服务请求，详情页显示「审批已全部通过…系统将自动创建云资源」，点「启动交付」
// 生成一条 ecs/alicloud 的交付任务，点「执行」返回 500 code 5001。
//
// 根因：建交付任务时**完全不读服务目录项**，resource_type 从表单猜、空值默认 ecs，
// provider 硬编码 alicloud。咨询类目录项因此被错当成 ECS 交付，
// 而云侧必然失败（缺地域/规格/账号），最终压成 500。
//
// 修后：资源类型由目录项 service_type 决定；非自动交付目录项在**建任务前**
// 就被拒绝，且不留任何 task/outbox 行。

// newCatalogServiceTypeFixture 建一个带指定 service_type 的目录项 + 已审批服务请求。
func newCatalogServiceTypeFixture(t *testing.T, serviceType string) (*ent.Client, context.Context, int, int, *ent.ServiceRequest) {
	t.Helper()
	dsn := "file:prov_catalog_type_" + serviceType + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, dialect.SQLite, dsn)
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	tenantID, requesterID := seedProvisioningPrerequisites(t, ctx, client)

	catalog, err := client.ServiceCatalog.Create().
		SetTenantID(tenantID).
		SetName("catalog-" + serviceType).
		SetServiceType(serviceType).
		SetStatus("enabled").
		Save(ctx)
	require.NoError(t, err)

	req, err := client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(catalog.ID).
		SetRequesterID(requesterID).
		SetTitle("Need resource").
		SetStatus(string(domainSR.StatusSecurityApproved)).
		Save(ctx)
	require.NoError(t, err)

	return client, ctx, tenantID, requesterID, req
}

// TestCreateTaskFromServiceRequestRejectsNonAutoDeliveryCatalog 是 O1 的核心回归锁：
// 咨询类目录项（custom）不得生成任何云资源交付任务。
//
// 断言必须同时覆盖三件事：报错可行动、零 task 行、零 outbox 行、
// 且服务请求状态未被改写（仍可走人工履约）。
func TestCreateTaskFromServiceRequestRejectsNonAutoDeliveryCatalog(t *testing.T) {
	client, ctx, tenantID, requesterID, req := newCatalogServiceTypeFixture(t, "custom")
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	_, err := svc.CreateTaskFromServiceRequest(ctx, req.ID, tenantID, requesterID)
	require.Error(t, err)

	// 必须是可行动的 409，不是兜底 500。
	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr), "必须是业务错误，不能被兜底成 500：%v", err)
	assert.Equal(t, common.ConflictCode, bizErr.Code)
	assert.Contains(t, bizErr.Message, "人工履约", "文案必须说明正确路径：%s", bizErr.Message)

	taskCount, err := client.ProvisioningTask.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, taskCount, "咨询类目录项不得生成云资源交付任务")

	cmdCount, err := client.OperationalCommand.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, cmdCount, "不得为被拒请求入箱执行命令")

	reloaded, err := client.ServiceRequest.Get(ctx, req.ID)
	require.NoError(t, err)
	assert.Equal(t, string(domainSR.StatusSecurityApproved), reloaded.Status,
		"被拒后服务请求状态不得改写，人工履约通道必须保持可用")
}

// TestCreateTaskFromServiceRequestUsesCatalogDeliveryType 钉住正向路径：
// vm 类目录项（云服务器 ECS）仍能正常建任务，且资源类型来自目录项而非表单。
func TestCreateTaskFromServiceRequestUsesCatalogDeliveryType(t *testing.T) {
	client, ctx, tenantID, requesterID, req := newCatalogServiceTypeFixture(t, "vm")
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	task, err := svc.CreateTaskFromServiceRequest(ctx, req.ID, tenantID, requesterID)
	require.NoError(t, err)
	require.NotZero(t, task.ID)
	assert.Equal(t, "ecs", task.ResourceType, "vm 目录项必须落 ecs 资源类型")
	assert.Equal(t, "alicloud", task.Provider)

	cmdCount, err := client.OperationalCommand.Query().
		Where(operationalcommand.AggregateTypeEQ("provisioning_task")).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, cmdCount, "自动交付目录项必须入箱执行命令")
}

// TestCatalogDeliveryResourceType 钉住映射表本身：
// 只有 vm 走自动云交付，其余目录项类型都是人工履约。
func TestCatalogDeliveryResourceType(t *testing.T) {
	cases := []struct {
		serviceType string
		wantType    string
		wantSupport bool
	}{
		{"vm", "ecs", true},
		{"VM", "ecs", true},
		{"vm ", "ecs", true},
		{"custom", "", false},
		{"rds", "", false},
		{"oss", "", false},
		{"network", "", false},
		{"storage", "", false},
		{"security", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		resourceType, supported := catalogDeliveryResourceType(&ent.ServiceCatalog{ServiceType: tc.serviceType})
		assert.Equal(t, tc.wantSupport, supported, "service_type=%q 支持判定", tc.serviceType)
		assert.Equal(t, tc.wantType, resourceType, "service_type=%q 资源类型", tc.serviceType)
	}

	// nil 目录项不得 panic，且必须判为不支持。
	resourceType, supported := catalogDeliveryResourceType(nil)
	assert.False(t, supported)
	assert.Empty(t, resourceType)
}
