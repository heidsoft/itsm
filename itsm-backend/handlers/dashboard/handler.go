package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for dashboard
type Handler struct {
	service *Service
}

// NewHandler creates a new dashboard handler
func NewHandler(service *Service) *Handler {
	if service == nil {
		service = NewService()
	}
	return &Handler{service: service}
}

// GetDashboard returns the full SLA dashboard data (with charts)
func (h *Handler) GetDashboard(c *gin.Context) {
	data, err := h.service.GetSLADashboard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to retrieve dashboard data",
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": data,
	})
}

// GetBasicDashboard returns simplified dashboard data (KPI only)
func (h *Handler) GetBasicDashboard(c *gin.Context) {
	data, err := h.service.GetBasicDashboard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to retrieve dashboard data",
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": data,
	})
}
