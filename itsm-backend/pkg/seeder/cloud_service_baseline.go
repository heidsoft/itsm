package seeder

import (
	"context"
	"fmt"

	"itsm-backend/ent/cloudservice"
)

// cloudServiceTypeSeed 是「云服务类型」基线条目，即发现执行器把
// provider+serviceCode 落到哪个资源类型定义上的那份事实。
//
// 为什么必须有基线：DiscoveryJobExecutor.runScope 对每个 scope 都要先按
// tenant+provider+serviceCode 找到一条 cloud_services，找不到就整批
// 「load cloud service <code>」失败（discovery_job_executor.go:293）。此前全仓
// 没有任何一处生产代码写入这张表——seeder 零引用、迁移零 INSERT，只有单测在造行。
// 也就是说五家厂商适配器都装配好了，但任何新租户第一次点「开始发现」必然失败，
// 除非运维先手工建出这张表。
type cloudServiceTypeSeed struct {
	Provider         string
	Category         string
	ServiceCode      string
	ServiceName      string
	ResourceTypeCode string
	ResourceTypeName string
	APIVersion       string
}

// cloudServiceTypeDefinitions 只为「发现适配器已装配」的厂商建条目。
//
// 与 service/cloud/adapters.RegisterAll 的同源性由守卫测试双向锁定
// （pkg/seeder/cloud_service_baseline_guard_test.go）：多写会造出「能选但不能自动
// 发现」的假能力，正是本仓库反复踩过的枚举漂移；漏写则新接入的厂商在真实租户下
// 依旧跑不起来。华为云/Azure/onprem 不在列——它们只在 KnownProviders 里被识别。
//
// 这份清单刻意不接 JSON 种子配置覆盖：条目内容取决于运行时装配了谁，
// 让配置能凭空增删厂商就会与注册表分叉。
func cloudServiceTypeDefinitions() []cloudServiceTypeSeed {
	return []cloudServiceTypeSeed{
		{Provider: "aliyun", Category: "compute", ServiceCode: "ecs", ServiceName: "云服务器 ECS", ResourceTypeCode: "instance", ResourceTypeName: "实例", APIVersion: "2014-05-26"},
		{Provider: "tencent", Category: "compute", ServiceCode: "cvm", ServiceName: "云服务器 CVM", ResourceTypeCode: "instance", ResourceTypeName: "实例", APIVersion: "2017-03-12"},
		{Provider: "aws", Category: "compute", ServiceCode: "ec2", ServiceName: "Amazon EC2 实例", ResourceTypeCode: "instance", ResourceTypeName: "实例", APIVersion: "2016-11-15"},
		{Provider: "volcengine", Category: "compute", ServiceCode: "ecs", ServiceName: "云服务器 ECS（火山引擎）", ResourceTypeCode: "instance", ResourceTypeName: "实例", APIVersion: "2020-04-01"},
		{Provider: "k8s", Category: "container", ServiceCode: "pod", ServiceName: "Kubernetes Pod", ResourceTypeCode: "pod", ResourceTypeName: "容器组", APIVersion: "v1"},
	}
}

// seedCloudServiceTypes 幂等补齐基线租户的云服务类型。
//
// 存在性判定按 (tenant, provider, service_code, resource_type_code)，即表上那条
// 复合唯一索引；已存在的行一律不动，租户对预置类型的改名/停用属于租户所有。
func (s *Seeder) seedCloudServiceTypes(ctx context.Context) error {
	t, err := s.baselineTenant(ctx)
	if err != nil {
		return fmt.Errorf("cloud service types tenant: %w", err)
	}

	defs := cloudServiceTypeDefinitions()
	created := 0
	for _, def := range defs {
		exists, err := s.client.CloudService.Query().Where(
			cloudservice.TenantIDEQ(t.ID),
			cloudservice.ProviderEQ(def.Provider),
			cloudservice.ServiceCodeEQ(def.ServiceCode),
			cloudservice.ResourceTypeCodeEQ(def.ResourceTypeCode),
		).Exist(ctx)
		if err != nil {
			return fmt.Errorf("check cloud service type %s/%s: %w", def.Provider, def.ServiceCode, err)
		}
		if exists {
			continue
		}
		_, err = s.client.CloudService.Create().
			SetProvider(def.Provider).
			SetCategory(def.Category).
			SetServiceCode(def.ServiceCode).
			SetServiceName(def.ServiceName).
			SetResourceTypeCode(def.ResourceTypeCode).
			SetResourceTypeName(def.ResourceTypeName).
			SetAPIVersion(def.APIVersion).
			SetIsSystem(true).
			SetIsActive(true).
			SetTenantID(t.ID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("seed cloud service type %s/%s: %w", def.Provider, def.ServiceCode, err)
		}
		created++
	}
	s.sugar.Infow("cloud service types reconciled",
		"created", created, "total", len(defs), "tenant_id", t.ID)
	return nil
}

// verifyCloudServiceTypes 校验基线已落地；cmdb-core 的 verify 调用它。
//
// 只查存在性，不查名称：名称归租户所有，预置行被租户改名不是基线破坏。
func (s *Seeder) verifyCloudServiceTypes(ctx context.Context) error {
	tenantID, err := s.resolveBaselineTenantID(ctx)
	if err != nil {
		return err
	}
	for _, def := range cloudServiceTypeDefinitions() {
		exists, err := s.client.CloudService.Query().Where(
			cloudservice.TenantIDEQ(tenantID),
			cloudservice.ProviderEQ(def.Provider),
			cloudservice.ServiceCodeEQ(def.ServiceCode),
			cloudservice.ResourceTypeCodeEQ(def.ResourceTypeCode),
		).Exist(ctx)
		if err != nil {
			return fmt.Errorf("verify cloud service type %s/%s: %w", def.Provider, def.ServiceCode, err)
		}
		if !exists {
			return fmt.Errorf("verify cloud service types: missing %s/%s(%s)",
				def.Provider, def.ServiceCode, def.ResourceTypeCode)
		}
	}
	return nil
}
