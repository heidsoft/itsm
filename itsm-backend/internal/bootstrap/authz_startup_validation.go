package bootstrap

// 启动期权限目录自检（codex 审计 P2-6）。
//
// registry.ValidateBootstrap 此前全仓无生产调用点，注释却承诺「供生产启动自检」。
// 结果：DB 缺码 / 码三元组不一致这类部署缺陷只能在运行期被 fail-closed 拒绝时
// 从日志反查。这里把唯一校验实现接进启动链。
//
// 产品决策（用户拍板）：全部立即 strict —— 目录损坏的已初始化租户拒绝启动。
// 兼容性边界：
//   - 没有租户的空库（全新安装、迁移尚未跑）跳过：此时无目录可校验，
//     且 InitializeStorage 先跑迁移再 AutoSeed，空库校验必然假红（MCP 迁移曾因此中断首装）。
//   - 校验前对该租户 Invalidate，避免进程内旧缓存造成假绿。
//   - 只校验权限目录完整性与三元组，不校验 role_permissions 关联
//     （那是迁移 B5 的职责，ValidateBootstrap 查不出来）。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/internal/authz"
)

// AuthzStartupValidationMode 决定目录缺陷是否阻断启动。
type AuthzStartupValidationMode string

const (
	AuthzStartupStrict AuthzStartupValidationMode = "strict"
	AuthzStartupReport AuthzStartupValidationMode = "report"
)

// ResolveAuthzStartupValidationMode 从环境变量解析模式。
// 默认 strict（用户拍板：全部立即 strict）；只有显式设置 report 才降级为诊断。
func ResolveAuthzStartupValidationMode(raw string) AuthzStartupValidationMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "report":
		return AuthzStartupReport
	default:
		return AuthzStartupStrict
	}
}

// ValidateAuthzCatalogAtStartup 对所有租户执行目录自检。
// 返回 nil 表示可继续启动；strict 下任一已初始化租户目录损坏即返回错误。
func ValidateAuthzCatalogAtStartup(ctx context.Context, client *ent.Client, registry *authz.Registry, mode AuthzStartupValidationMode) error {
	if client == nil || registry == nil {
		return fmt.Errorf("authz startup validation: client or registry not wired")
	}
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tenants, err := client.Tenant.Query().All(checkCtx)
	if err != nil {
		// DB 读不出来是运行期 fail-closed 已覆盖的情况；启动期不额外
		// 阻断，避免把暂时性 DB 抖动放大成整个实例无法启动。
		return fmt.Errorf("authz startup validation: tenant query failed: %w", err)
	}
	if len(tenants) == 0 {
		return nil // 空库：seeder/迁移尚未运行，无目录可校验
	}

	var failures []string
	for _, t := range tenants {
		registry.Invalidate(t.ID) // 强制新鲜读取，避免旧缓存假绿
		if err := registry.ValidateBootstrap(checkCtx, t.ID); err != nil {
			failures = append(failures, fmt.Sprintf("tenant %d (%s): %v", t.ID, t.Name, err))
		}
	}
	if len(failures) == 0 {
		return nil
	}
	msg := "authz bootstrap catalog validation failed: " + strings.Join(failures, "; ")
	if mode == AuthzStartupStrict {
		return fmt.Errorf("%s", msg)
	}
	// report 模式只由调用方记日志，不阻断。
	return &AuthzStartupDiagnostic{Messages: failures}
}

// AuthzStartupDiagnostic 是 report 模式下的非阻断诊断结果。
type AuthzStartupDiagnostic struct {
	Messages []string
}

func (d *AuthzStartupDiagnostic) Error() string {
	return "authz bootstrap catalog diagnostic: " + strings.Join(d.Messages, "; ")
}
