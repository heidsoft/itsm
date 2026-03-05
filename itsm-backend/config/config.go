package config

import (
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Server   ServerConfig   `mapstructure:"server"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	LLM      LLMConfig      `mapstructure:"llm"`
	SMS      SMSConfig      `mapstructure:"sms"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
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

	// 重新绑定到 Config 结构
	if err := viper.Unmarshal(&Config{}); err != nil {
		return nil, err
	}

	// 使用 viper 再次解析
	viper.Set("database", rawConfig["database"])
	viper.Set("server", rawConfig["server"])
	viper.Set("jwt", rawConfig["jwt"])
	viper.Set("log", rawConfig["log"])
	viper.Set("llm", rawConfig["llm"])
	viper.Set("sms", rawConfig["sms"])
	viper.Set("smtp", rawConfig["smtp"])

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// 后备: 如果环境变量直接设置了 ITSM_XXX，则使用它
	config.JWT.Secret = getEnvWithDefault("JWT_SECRET", config.JWT.Secret)
	config.Database.Password = getEnvWithDefault("DB_PASSWORD", config.Database.Password)
	config.Server.Mode = getEnvWithDefault("SERVER_MODE", config.Server.Mode)
	config.Log.Level = getEnvWithDefault("LOG_LEVEL", config.Log.Level)
	config.Log.Path = getEnvWithDefault("LOG_PATH", config.Log.Path)
	config.Log.Development = os.Getenv("LOG_DEVELOPMENT") == "true"
	config.LLM.APIKey = getEnvWithDefault("OPENAI_API_KEY", config.LLM.APIKey)

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
