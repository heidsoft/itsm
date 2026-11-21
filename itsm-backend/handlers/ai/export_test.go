package ai

import "context"

// 本文件仅参与测试编译（文件名后缀 _test.go），生产构建不含这些符号。
//
// 用途：控制面聚合读面的 fail-closed 判据需要「让队列查询真的失败」，
// 而 Service 的 repo 字段不对外暴露。提供这个薄钩子比给 Service 加一个
// 生产可用的 SetRepository 安全——后者会让生产路径用哪个仓储变成
// 可被外部替换，而仓储是判定与执行的真源。

// ControlPlaneWithRepoForTest 用指定仓储跑一次聚合读面。
//
// 只在测试里用来注入带毒仓储；正常调用走 Service.GetControlPlaneOverview。
func ControlPlaneWithRepoForTest(ctx context.Context, s *Service, repo Repository, tenantID int, role string) (*ControlPlaneOverview, error) {
	return s.getControlPlaneOverview(ctx, repo, tenantID, role)
}
