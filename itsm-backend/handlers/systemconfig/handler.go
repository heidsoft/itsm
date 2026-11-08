// Package systemconfig 是系统配置域的 HTTP handler 层（域切片架构）。
// 业务逻辑仍由 SystemConfigService 承载，本包只做参数解析与响应封装。
package systemconfig

import (
	"go.uber.org/zap"
)

// Handler 系统配置 HTTP handler
type Handler struct {
	configService *SystemConfigService
	logger        *zap.SugaredLogger
}

// NewHandler 创建系统配置 handler 实例
func NewHandler(configService *SystemConfigService, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		configService: configService,
		logger:        logger,
	}
}
