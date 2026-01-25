package config

import (
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Server   ServerConfig   `mapstructure:"server"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	LLM      LLMConfig      `mapstructure:"llm"`
	SMS      SMSConfig      `mapstructure:"sms"`
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
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

type LLMConfig struct {
	Provider string `mapstructure:"provider"` // openai|azure|local
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
	Endpoint string `mapstructure:"endpoint"`
	TokenCap int    `mapstructure:"token_cap"` // per request soft cap
}

type SMSConfig struct {
	Provider string          `mapstructure:"provider"` // "aliyun" or "tencent", empty means disabled
	Aliyun   AliyunSMSConfig `mapstructure:"aliyun"`
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

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// 后备: 如果环境变量直接设置了 ITSM_XXX，则使用它
	config.JWT.Secret = getEnvWithDefault("JWT_SECRET", config.JWT.Secret)
	config.Database.Password = getEnvWithDefault("DB_PASSWORD", config.Database.Password)
	config.Server.Mode = getEnvWithDefault("SERVER_MODE", config.Server.Mode)
	config.Log.Level = getEnvWithDefault("LOG_LEVEL", config.Log.Level)
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
