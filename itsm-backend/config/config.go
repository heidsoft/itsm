package config

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Database   DatabaseConfig   `mapstructure:"database"`
	Server     ServerConfig     `mapstructure:"server"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Log        LogConfig        `mapstructure:"log"`
	LLM        LLMConfig        `mapstructure:"llm"`
	SMS        SMSConfig        `mapstructure:"sms"`
	SMTP       SMTPConfig       `mapstructure:"smtp"`
	Ticket     TicketConfig     `mapstructure:"ticket"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Security   SecurityConfig   `mapstructure:"security"`
	Deployment DeploymentConfig `mapstructure:"deployment"`
	RLS        RLSConfig        `mapstructure:"rls"`
}

// RLSConfig 控制 PostgreSQL Row-Level Security 的启用档位。
//
// Mode:
//   - "off"     : 默认。中间件仍会向 request.Context 注入 tenant_id，
//                 但不 SET SESSION 变量，也不启用 policy。零风险。
//   - "shadow"  : 每次 request 走 rls.AcquireConn 设 SESSION 变量，
//                 但 policy 未启用 → 数据库不拦截，只观察是否有 ctx
//                 缺失情况；不影响任何业务。
//   - "enforce" : SESSION 变量 + policy 同时生效，数据库层强制隔离。
//                 需先在 shadow 模式下把所有缺失点补齐。
//
// TenantVarName: PostgreSQL 用于承载 tenant_id 的 GUC 变量名，默认
// "app.current_tenant"，与 policy 中的 current_setting() 保持一致。
type RLSConfig struct {
	Mode          string `mapstructure:"mode"`
	TenantVarName string `mapstructure:"tenant_var_name"`
}

type DeploymentConfig struct {
	Mode        string `mapstructure:"mode"`
	AutoMigrate bool   `mapstructure:"auto_migrate"`
	AutoSeed    bool   `mapstructure:"auto_seed"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	CSRFEnabled bool `mapstructure:"csrf_enabled"` // 是否启用 CSRF 保护
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type TicketConfig struct {
	DefaultResponseHours   int `mapstructure:"default_response_hours"`   // 默认响应时间（小时）
	DefaultResolutionHours int `mapstructure:"default_resolution_hours"` // 默认解决时间（小时）
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`

	// AppRoleUser / AppRolePassword: 应用请求路径使用的低权角色
	// （不带 BYPASSRLS，走 policy 过滤）。留空时降级为使用 User/Password。
	// 用于 R2 灰度：切 enforce 前先切到 itsm_app。
	AppRoleUser     string `mapstructure:"app_role_user"`
	AppRolePassword string `mapstructure:"app_role_password"`

	// AdminRoleUser / AdminRolePassword: 后台任务 / 迁移 / seed 使用的
	// BYPASSRLS 角色。留空时降级为使用 User/Password（假定 User 已具备
	// superuser，例如 dev 环境的 itsm_user）。
	AdminRoleUser     string `mapstructure:"admin_role_user"`
	AdminRolePassword string `mapstructure:"admin_role_password"`
}

// AppDSN 返回应用请求路径的连接串（低权角色，走 RLS policy）。
// 若未配置 AppRoleUser，则回落至默认 User。
func (d *DatabaseConfig) AppDSN() (user, password string) {
	if d.AppRoleUser != "" {
		return d.AppRoleUser, d.AppRolePassword
	}
	return d.User, d.Password
}

// AdminDSN 返回后台任务 / 迁移路径的连接串（BYPASSRLS 角色）。
// 若未配置 AdminRoleUser，则回落至默认 User（要求 User 是 superuser）。
func (d *DatabaseConfig) AdminDSN() (user, password string) {
	if d.AdminRoleUser != "" {
		return d.AdminRoleUser, d.AdminRolePassword
	}
	return d.User, d.Password
}

type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	Mode         string `mapstructure:"mode"`
	CookieSecure bool   `mapstructure:"cookie_secure"` // Secure flag for cookies (set true only behind HTTPS)
}

type JWTConfig struct {
	Secret            string `mapstructure:"secret"`
	ExpireTime        int    `mapstructure:"expire_time"`         // Access Token过期时间（如15分钟）
	RefreshExpireTime int    `mapstructure:"refresh_expire_time"` // Refresh Token过期时间（如7天）
}

type LogConfig struct {
	Level                string `mapstructure:"level"`                  // 日志级别：debug, info, warn, error
	Path                 string `mapstructure:"path"`                   // 日志文件目录
	MaxSize              int    `mapstructure:"max_size"`               // 单个日志文件最大大小 (MB)
	MaxBackups           int    `mapstructure:"max_backups"`            // 保留的旧日志文件数量
	MaxAge               int    `mapstructure:"max_age"`                // 日志文件保留天数
	Compress             bool   `mapstructure:"compress"`               // 是否压缩旧日志文件
	LocalTime            bool   `mapstructure:"local_time"`             // 日志文件名是否使用本地时间 (默认 UTC)
	Development          bool   `mapstructure:"development"`            // 开发模式 (输出到 console)
	SlowRequestThreshold int    `mapstructure:"slow_request_threshold"` // 慢请求阈值 (毫秒)，默认 1000ms
	EnableStackTrace     bool   `mapstructure:"enable_stack_trace"`     // 是否对 5xx 错误启用堆栈跟踪
	SkipHealthCheckLogs  bool   `mapstructure:"skip_health_check_logs"` // 是否跳过健康检查日志
}

type LLMConfig struct {
	Provider string `mapstructure:"provider"` // openai|azure|local
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
	Endpoint string `mapstructure:"endpoint"`
	TokenCap int    `mapstructure:"token_cap"` // per request soft cap
}

type SMSConfig struct {
	Provider string           `mapstructure:"provider"` // "aliyun" or "tencent", empty means disabled
	Aliyun   AliyunSMSConfig  `mapstructure:"aliyun"`
	Tencent  TencentSMSConfig `mapstructure:"tencent"`
}

type AliyunSMSConfig struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	SignName        string `mapstructure:"sign_name"`
	Endpoint        string `mapstructure:"endpoint"`
}

type TencentSMSConfig struct {
	SecretID  string `mapstructure:"secret_id"`
	SecretKey string `mapstructure:"secret_key"`
	AppID     string `mapstructure:"app_id"`
	SignName  string `mapstructure:"sign_name"`
	Endpoint  string `mapstructure:"endpoint"`
}

// SMTPConfig 邮件服务配置
type SMTPConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	FromEmail  string `mapstructure:"from_email"`
	FromName   string `mapstructure:"from_name"`
	Encryption string `mapstructure:"encryption"` // tls, ssl, or none
	SkipVerify bool   `mapstructure:"skip_verify"`
}

// envVarPattern matches ${VAR:default} format
var envVarPattern = regexp.MustCompile(`\$\{([^:}]+)(?::([^}]*))?\}`)

// resolveEnvVars 解析配置文件中的环境变量 ${VAR:default}
func resolveEnvVars(input string) string {
	return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		submatches := envVarPattern.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			envName := submatches[1]
			defaultValue := submatches[2]
			if value := os.Getenv(envName); value != "" {
				return value
			}
			return defaultValue
		}
		return match
	})
}

// resolveMapEnvVars 递归解析 map 中的环境变量
func resolveMapEnvVars(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			m[k] = resolveEnvVars(val)
		case map[string]interface{}:
			resolveMapEnvVars(val)
		case []interface{}:
			for i, item := range val {
				if s, ok := item.(string); ok {
					val[i] = resolveEnvVars(s)
				}
			}
		}
	}
}

func LoadConfig() (*Config, error) {
	// 加载 .env 文件（如果存在）
	_ = godotenv.Load()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("配置文件读取失败: %v", err)
		return nil, err
	}

	// 解析环境变量
	var rawConfig map[string]interface{}
	if err := viper.Unmarshal(&rawConfig); err != nil {
		return nil, err
	}
	resolveMapEnvVars(rawConfig)

	// 重新设置解析后的值到 viper
	viper.Set("database", rawConfig["database"])
	viper.Set("server", rawConfig["server"])
	viper.Set("jwt", rawConfig["jwt"])
	viper.Set("log", rawConfig["log"])
	viper.Set("llm", rawConfig["llm"])
	viper.Set("sms", rawConfig["sms"])
	viper.Set("smtp", rawConfig["smtp"])
	viper.Set("redis", rawConfig["redis"])
	viper.Set("ticket", rawConfig["ticket"])
	viper.Set("embedding", rawConfig["embedding"])
	viper.Set("security", rawConfig["security"])
	viper.Set("admin", rawConfig["admin"])
	viper.Set("deployment", rawConfig["deployment"])

	// 重新绑定到 Config 结构
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// 后备: 如果环境变量直接设置了 ITSM_XXX，则使用它
	config.JWT.Secret = getEnvWithDefault("JWT_SECRET", config.JWT.Secret)
	config.Database.Password = getEnvWithDefault("DB_PASSWORD", config.Database.Password)
	// RLS 双角色 DSN（可选）：留空则回落至默认 DB_USER/DB_PASSWORD
	config.Database.AppRoleUser = getEnvWithDefault("DB_APP_ROLE_USER", config.Database.AppRoleUser)
	config.Database.AppRolePassword = getEnvWithDefault("DB_APP_ROLE_PASSWORD", config.Database.AppRolePassword)
	config.Database.AdminRoleUser = getEnvWithDefault("DB_ADMIN_ROLE_USER", config.Database.AdminRoleUser)
	config.Database.AdminRolePassword = getEnvWithDefault("DB_ADMIN_ROLE_PASSWORD", config.Database.AdminRolePassword)
	config.Server.Mode = getEnvWithDefault("SERVER_MODE", config.Server.Mode)
	config.Log.Level = getEnvWithDefault("LOG_LEVEL", config.Log.Level)
	config.Log.Path = getEnvWithDefault("LOG_PATH", config.Log.Path)
	config.Log.Development = os.Getenv("LOG_DEVELOPMENT") == "true"
	// Support both LLM_API_KEY (preferred) and OPENAI_API_KEY (legacy fallback) env vars
	config.LLM.APIKey = getEnvWithDefault("OPENAI_API_KEY", config.LLM.APIKey)
	config.LLM.APIKey = getEnvWithDefault("LLM_API_KEY", config.LLM.APIKey)
	config.Deployment.Mode = getEnvWithDefault("DEPLOYMENT_MODE", config.Deployment.Mode)
	config.Deployment.AutoMigrate = getEnvBoolWithDefault("ITSM_AUTO_MIGRATE", config.Deployment.AutoMigrate)
	config.Deployment.AutoSeed = getEnvBoolWithDefault("ITSM_AUTO_SEED", config.Deployment.AutoSeed)

	// RLS 三档开关，默认 off（零风险）。
	config.RLS.Mode = getEnvWithDefault("RLS_MODE", config.RLS.Mode)
	if config.RLS.Mode == "" {
		config.RLS.Mode = "off"
	}
	config.RLS.TenantVarName = getEnvWithDefault("RLS_TENANT_VAR_NAME", config.RLS.TenantVarName)
	if config.RLS.TenantVarName == "" {
		config.RLS.TenantVarName = "app.current_tenant"
	}

	if config.JWT.Secret == "" {
		secretBytes := make([]byte, 32)
		if _, err := rand.Read(secretBytes); err != nil {
			return nil, err
		}
		config.JWT.Secret = base64.RawURLEncoding.EncodeToString(secretBytes)
	}

	// SMS 环境变量
	if config.SMS.Provider == "" {
		// 检查环境变量决定 provider
		if os.Getenv("ALIYUN_ACCESS_KEY_ID") != "" {
			config.SMS.Provider = "aliyun"
		} else if os.Getenv("TENCENT_SECRET_ID") != "" {
			config.SMS.Provider = "tencent"
		}
	}

	// SMTP 环境变量支持
	config.SMTP.Enabled = config.SMTP.Enabled || os.Getenv("SMTP_ENABLED") == "true"
	config.SMTP.Host = getEnvWithDefault("SMTP_HOST", config.SMTP.Host)
	config.SMTP.Username = getEnvWithDefault("SMTP_USERNAME", config.SMTP.Username)
	config.SMTP.Password = getEnvWithDefault("SMTP_PASSWORD", config.SMTP.Password)
	config.SMTP.FromEmail = getEnvWithDefault("SMTP_FROM_EMAIL", config.SMTP.FromEmail)
	config.SMTP.FromName = getEnvWithDefault("SMTP_FROM_NAME", config.SMTP.FromName)

	return &config, nil
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	// 也检查 ITSM_ 前缀
	if value := os.Getenv("ITSM_" + strings.ToUpper(key)); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBoolWithDefault(key string, defaultValue bool) bool {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		switch strings.ToLower(value) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}

	if value := strings.TrimSpace(os.Getenv("ITSM_" + strings.ToUpper(key))); value != "" {
		switch strings.ToLower(value) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}

	return defaultValue
}
