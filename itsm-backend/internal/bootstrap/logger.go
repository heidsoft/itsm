package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"itsm-backend/config"
)

// initLogger 初始化日志系统
func initLogger(cfg *config.LogConfig) *zap.Logger {
	// 开发模式直接返回开发日志
	if cfg.Development {
		logger, _ := zap.NewDevelopment()
		return logger
	}

	// 生产模式：文件输出
	var level zapcore.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn", "warning":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 确保日志目录存在
	if cfg.Path != "" {
		if err := os.MkdirAll(cfg.Path, 0o755); err != nil {
			fmt.Printf("Failed to create log directory: %v\n", err)
		}
	}

	// 文件输出配置
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(cfg.Path, "itsm.log"),
		MaxSize:    cfg.MaxSize, // MB
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge, // days
		Compress:   cfg.Compress,
		LocalTime:  cfg.LocalTime, // 使用本地时间戳
	})

	// 控制台输出
	consoleWriter := zapcore.AddSync(os.Stdout)

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建编码器
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	// 创建core：同时输出到文件和控制台
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, fileWriter, level),
		zapcore.NewCore(consoleEncoder, consoleWriter, level),
	)

	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}

// GetLogFilePath 获取日志文件路径（用于外部访问）
func GetLogFilePath(logPath string) string {
	if logPath == "" {
		return ""
	}
	return filepath.Join(logPath, fmt.Sprintf("itsm-%s.log", time.Now().Format("2006-01-02")))
}
