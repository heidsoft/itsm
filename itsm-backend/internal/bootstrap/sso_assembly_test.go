package bootstrap

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSSOServiceIsInjectedIntoAuthHandler 锁定 SSO（OIDC）子系统的装配链。
//
// 为什么需要这条守卫：router.SetupSSORoutes 整组以 AuthHandler.SSOEnabled()
// 为总开关（h.sso != nil），而路由级测试自己注入了 SSO 服务——NewApplication
// 一旦漏掉 SetSSOService，生产上整套 SSO（登录发起/回调/绑定/提供方管理）
// 静默不注册：管理接口 404、SSO 登录不可用，却没有任何测试会红。
// 这就是「测试注入 ≠ 生产装配」的盲区，与 SLA 的 SetMatrixService 同类。
//
// 同时锁定两个配套条件：
//   - client_secret 加密与 connector_configs 同口径（connectorEncryptionKey）：
//     换密钥来源会让已落库的密文全部无法解密，属破坏性变更；
//   - state store 必须经 authHandler.NewSSOStateStore 构造（Redis 跨副本共享 +
//     进程内降级），绕过它直连内存实现会让多副本回调静默失败。
func TestSSOServiceIsInjectedIntoAuthHandler(t *testing.T) {
	source, err := os.ReadFile("app.go")
	require.NoError(t, err)
	assembly := functionSource(t, source, "NewApplication")

	for _, required := range []string{
		"authHTTPHandler.SetSSOService(",
		"authHandler.NewSSOService(",
		"middleware.NewEncryptionService(connectorEncryptionKey)",
		"authHandler.NewSSOStateStore(",
	} {
		require.Truef(t, strings.Contains(assembly, required),
			"NewApplication 缺少 SSO 装配项 %q；缺任一即整套 SSO 在生产静默失效"+
				"（路由不注册且测试无感）或加密密钥口径漂移", required)
	}
}
