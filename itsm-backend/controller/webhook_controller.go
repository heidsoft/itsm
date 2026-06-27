package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"itsm-backend/common"
)

// WebhookController webhook controller (stubs are reserved for future use)
type WebhookController struct{}

// 接收外部监控告警
// @Summary 接收监控告警
// @Description 接收外部监控系统发送的告警，自动创建事件工单
// @Tags Webhook
// @Accept json
// @Produce json
// @Param request body object true "告警信息"
// @Success 200 {object} common.Response
// @Router /api/v1/webhooks/alert [post]
func (w *WebhookController) ReceiveAlert(c *gin.Context) {
	// 解析告警数据
	// 自动创建事件
	// 关联CMDB配置项
	c.JSON(http.StatusNotImplemented, gin.H{
		"code":    common.InternalErrorCode,
		"message": "功能尚未实现",
	})
}
