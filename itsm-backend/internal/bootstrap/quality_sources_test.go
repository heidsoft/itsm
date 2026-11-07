package bootstrap

import (
	"os"
	"strings"
	"testing"
)

// TestQualityDomainStatsWired 统一服务质量视图的问题/变更统计块必须在生产
// 装配里真实注入——装配缺失时响应块静默缺省（2026-11-07 之前的真实状态：
// 后端有聚合器、有两域统计，但从没接到一起）。用源码守卫锁定调用存在。
func TestQualityDomainStatsWired(t *testing.T) {
	raw, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatalf("read app.go: %v", err)
	}
	src := string(raw)

	callIdx := strings.Index(src, "wireQualityDomainStats(qualityReportService,")
	if callIdx < 0 {
		t.Fatal("app.go 必须调用 wireQualityDomainStats 注入问题/变更统计块")
	}
	changeIdx := strings.Index(src, "changeServiceDomain := change.NewService(")
	if changeIdx < 0 || callIdx < changeIdx {
		t.Fatal("wireQualityDomainStats 必须在 changeServiceDomain 构造之后调用")
	}
	problemIdx := strings.Index(src, "problemServiceDomain := problem.NewService(")
	if problemIdx < 0 || callIdx < problemIdx {
		t.Fatal("wireQualityDomainStats 必须在 problemServiceDomain 构造之后调用")
	}
}
