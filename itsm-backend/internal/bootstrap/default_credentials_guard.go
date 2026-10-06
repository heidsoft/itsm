package bootstrap

// 默认账号安全检测 · v1.0 GA 准入
//
// 在服务启动时检测：
//  1. 生产环境（ENV=production / saas / saas_msp）+ admin/admin123 → 拒绝启动
//  2. 生产环境 + JWT_SECRET 为默认值 → 警告
//  3. 生产环境 + DB_PASSWORD 为弱密码 → 警告
//
// 调用方式：在 main.go 启动早期调用 GuardDefaultCredentials(environment)

import (
	"os"
	"strings"

	"go.uber.org/zap"

	"itsm-backend/pkg/credential"
)

// DefaultCredentialRisk 检测到的风险项
type DefaultCredentialRisk struct {
	Severity string // "fatal" | "warning"
	Code     string
	Message  string
}

// GuardDefaultCredentials 在启动时执行安全检查
// environment: 部署模式 ("private" | "saas" | "saas_msp" | "development")
// 返回 fatal 级风险时进程应 panic
func GuardDefaultCredentials(environment string) []DefaultCredentialRisk {
	return guardCredentials(
		environment,
		os.Getenv("ADMIN_PASSWORD"),
		os.Getenv("JWT_SECRET"),
		os.Getenv("DB_PASSWORD"),
		true,
	)
}

func guardCredentials(
	environment string,
	adminPassword string,
	jwtSecret string,
	dbPassword string,
	includeAdmin bool,
) []DefaultCredentialRisk {
	risks := make([]DefaultCredentialRisk, 0)
	isProd := isProductionEnvironment(environment)

	// 检测 1: admin 默认密码
	if isProd && includeAdmin {
		if hasDefaultAdminPassword(adminPassword) {
			risks = append(risks, DefaultCredentialRisk{
				Severity: "fatal",
				Code:     "DEFAULT_ADMIN_PASSWORD",
				Message:  "检测到生产环境使用 admin 默认密码。请在首次部署前修改 ADMIN_PASSWORD 环境变量，或通过 seeder 重新初始化。",
			})
		}
	}

	// 检测 2: JWT_SECRET 默认值
	if isProd {
		if isDefaultJWTSecret(jwtSecret) {
			risks = append(risks, DefaultCredentialRisk{
				Severity: "fatal",
				Code:     "DEFAULT_JWT_SECRET",
				Message:  "检测到生产环境 JWT_SECRET 为默认值。请设置强随机字符串（至少 32 字符）。",
			})
		}
	}

	// 检测 3: 数据库弱密码
	if isProd {
		if isWeakDBPassword(dbPassword) {
			risks = append(risks, DefaultCredentialRisk{
				Severity: "warning",
				Code:     "WEAK_DB_PASSWORD",
				Message:  "检测到生产环境 DB_PASSWORD 为常见弱密码。建议使用密码管理器生成 16+ 字符强密码。",
			})
		}
	}

	return risks
}

// GuardRuntimeCredentials validates credentials required for every process
// start. The one-time bootstrap admin secret is intentionally excluded and is
// checked only when initialization is about to create the first administrator.
func GuardRuntimeCredentials(environment, jwtSecret, dbPassword string) []DefaultCredentialRisk {
	return guardCredentials(environment, "", jwtSecret, dbPassword, false)
}

// GuardBootstrapAdminCredentials validates the one-time administrator secret.
func GuardBootstrapAdminCredentials(environment, adminPassword string) []DefaultCredentialRisk {
	risks := guardCredentials(environment, adminPassword, "", "", true)
	filtered := make([]DefaultCredentialRisk, 0, 1)
	for _, risk := range risks {
		if risk.Code == "DEFAULT_ADMIN_PASSWORD" {
			filtered = append(filtered, risk)
		}
	}
	return filtered
}

// LogDefaultCredentialRisks 用 zap 记录风险
func LogDefaultCredentialRisks(risks []DefaultCredentialRisk, logger *zap.SugaredLogger) {
	for _, r := range risks {
		if r.Severity == "fatal" {
			logger.Fatalw(
				"default credential risk detected",
				"code", r.Code,
				"message", r.Message,
			)
		} else {
			logger.Warnw(
				"default credential risk detected",
				"code", r.Code,
				"message", r.Message,
			)
		}
	}
}

// isProductionEnvironment 判定是否为生产部署。
// 优先使用 resolveEnvironment()（SERVER_ENV > ENV），其次使用传入的 env 参数（通常来自 cfg.Deployment.Mode）。
func isProductionEnvironment(env string) bool {
	stage := resolveEnvironment()
	switch stage {
	case "development", "dev", "test", "testing", "local":
		return false
	case "production", "prod":
		return true
	}

	if env == "" {
		env = os.Getenv("DEPLOYMENT_MODE")
	}
	env = strings.ToLower(strings.TrimSpace(env))
	return env == "production" || env == "prod" ||
		env == "private" || env == "saas" || env == "saas_msp"
}

func hasDefaultAdminPassword(adminPass string) bool {
	// Delegate to pkg/credential (single source of weak-defaults list).
	// Empty string short-circuits inside IsWeakAdminPassword.
	return credential.IsWeakAdminPassword(adminPass)
}

func isDefaultJWTSecret(secret string) bool {
	if secret == "" {
		return true
	}
	// 已知的占位符/弱默认
	weakSecrets := []string{
		"your-jwt-secret",
		"change-me",
		"secret",
		"jwt-secret",
		"itsm-secret",
		"dev-secret",
		"please-change-in-production",
	}
	lower := strings.ToLower(secret)
	for _, w := range weakSecrets {
		if lower == w || strings.Contains(lower, w) {
			return true
		}
	}
	// 长度检查：< 32 字符视为弱
	if len(secret) < 32 {
		return true
	}
	return false
}

func isWeakDBPassword(pass string) bool {
	if pass == "" {
		return false // 未设置时不强制判定
	}
	weak := []string{
		"itsm_password_2026", "dev123", "password", "admin", "123456",
		"postgres", "root", "test",
	}
	lower := strings.ToLower(pass)
	for _, w := range weak {
		if lower == w {
			return true
		}
	}
	return false
}
