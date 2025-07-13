package controller

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"itsm-backend/common"
	"itsm-backend/service"
)

type WebhookController struct {
	ticketService *service.TicketService
	logger        *zap.SugaredLogger
}

// 接收外部监控告警
func (w *WebhookController) ReceiveAlert(c *gin.Context) {
	// 解析告警数据
	// 自动创建事件
	// 关联CMDB配置项
	common.Success(c, nil)
}
