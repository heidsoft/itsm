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
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
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
	risks := make([]DefaultCredentialRisk, 0)
	isProd := isProductionEnvironment(environment)

	// 检测 1: admin 默认密码
	if isProd {
		if hasDefaultAdminPassword() {
			risks = append(risks, DefaultCredentialRisk{
				Severity: "fatal",
				Code:     "DEFAULT_ADMIN_PASSWORD",
				Message:  "检测到生产环境使用 admin 默认密码。请在首次部署前修改 ADMIN_PASSWORD 环境变量，或通过 seeder 重新初始化。",
			})
		}
	}

	// 检测 2: JWT_SECRET 默认值
	if isProd {
		if isDefaultJWTSecret() {
			risks = append(risks, DefaultCredentialRisk{
				Severity: "fatal",
				Code:     "DEFAULT_JWT_SECRET",
				Message:  "检测到生产环境 JWT_SECRET 为默认值。请设置强随机字符串（至少 32 字符）。",
			})
		}
	}

	// 检测 3: 数据库弱密码
	if isProd {
		if isWeakDBPassword() {
			risks = append(risks, DefaultCredentialRisk{
				Severity: "warning",
				Code:     "WEAK_DB_PASSWORD",
				Message:  "检测到生产环境 DB_PASSWORD 为常见弱密码。建议使用密码管理器生成 16+ 字符强密码。",
			})
		}
	}

	// 报告
	for _, r := range risks {
		if r.Severity == "fatal" {
			fmt.Fprintf(os.Stderr, "\n[FATAL %s] %s\n\n", r.Code, r.Message)
		} else {
			fmt.Fprintf(os.Stderr, "\n[WARN %s] %s\n\n", r.Code, r.Message)
		}
	}

	return risks
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

// isProductionEnvironment 判定是否为生产部署
func isProductionEnvironment(env string) bool {
	if env == "" {
		env = os.Getenv("ENV")
		if env == "" {
			env = os.Getenv("DEPLOYMENT_MODE")
		}
	}
	env = strings.ToLower(env)
	return env == "production" || env == "prod" ||
		env == "saas" || env == "saas_msp"
}

func hasDefaultAdminPassword() bool {
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		// 未设置：依赖 seeder 的默认值。在生产环境应视为默认。
		// 这里保守判定：有 ENV=prod + 没设 ADMIN_PASSWORD = 默认值
		return true
	}
	// 检查是否为已知的弱默认值
	weakDefaults := []string{
		"admin", "admin123", "password", "123456", "itsm123", "changeme",
	}
	lower := strings.ToLower(adminPass)
	for _, w := range weakDefaults {
		if lower == w {
			return true
		}
	}
	return false
}

func isDefaultJWTSecret() bool {
	secret := os.Getenv("JWT_SECRET")
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

func isWeakDBPassword() bool {
	pass := os.Getenv("DB_PASSWORD")
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
