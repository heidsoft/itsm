package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"openclaw-deploy/internal/models"
	"openclaw-deploy/pkg/aliyun"
	"openclaw-deploy/pkg/database"
)

type DeploymentHandler struct {
	db *database.Database
}

func NewDeploymentHandler(db *database.Database) *DeploymentHandler {
	return &DeploymentHandler{db: db}
}

// GetDeployments 获取部署列表
func (h *DeploymentHandler) GetDeployments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	deployments, total, err := models.GetDeployments(h.db.GetDB(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      deployments,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// CreateDeployment 创建部署
func (h *DeploymentHandler) CreateDeployment(c *gin.Context) {
	var req CreateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 生成实例名称
	instanceName := "openclaw-" + req.UserID + "-" + req.Plan

	// 创建 ECS 实例
	instance, err := aliyun.CreateInstance(aliyun.CreateInstanceArgs{
		InstanceType:    getInstanceType(req.Plan),
		ImageID:         "ubuntu_20_04_x64_20G_alibase_20210120.vhd",
		SecurityGroupID: aliyun.GetSecurityGroupID(),
		InstanceName:    instanceName,
		Bandwidth:       getBandwidth(req.Plan),
		DiskSize:        getDiskSize(req.Plan),
		Password:        generatePassword(),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ECS instance: " + err.Error()})
		return
	}

	// 创建数据库记录
	deployment := &models.Deployment{
		ID:           uuid.New().String(),
		UserID:       req.UserID,
		Plan:         req.Plan,
		InstanceID:   instance.InstanceId,
		InstanceName: instanceName,
		Domain:       req.Domain,
		Status:       "deploying",
		Region:       aliyun.GetRegionID(),
		Username:     req.Username,
		Password:     generatePassword(),
	}

	if err := models.CreateDeployment(h.db.GetDB(), deployment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    deployment,
		"message": "Deployment created successfully",
	})
}

// GetDeployment 获取部署详情
func (h *DeploymentHandler) GetDeployment(c *gin.Context) {
	id := c.Param("id")

	deployment, err := models.GetDeployment(h.db.GetDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": deployment})
}

// StartDeployment 启动部署
func (h *DeploymentHandler) StartDeployment(c *gin.Context) {
	id := c.Param("id")

	deployment, err := models.GetDeployment(h.db.GetDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	if err := aliyun.StartInstance(deployment.InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	deployment.Status = "running"
	models.UpdateDeployment(h.db.GetDB(), deployment)

	c.JSON(http.StatusOK, gin.H{"message": "Deployment started"})
}

// StopDeployment 停止部署
func (h *DeploymentHandler) StopDeployment(c *gin.Context) {
	id := c.Param("id")

	deployment, err := models.GetDeployment(h.db.GetDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	if err := aliyun.StopInstance(deployment.InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	deployment.Status = "stopped"
	models.UpdateDeployment(h.db.GetDB(), deployment)

	c.JSON(http.StatusOK, gin.H{"message": "Deployment stopped"})
}

// GetMetrics 获取监控数据
func (h *DeploymentHandler) GetMetrics(c *gin.Context) {
	id := c.Param("id")

	deployment, err := models.GetDeployment(h.db.GetDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	// 获取真实监控数据（这里简化，实际应该从云监控获取）
	metrics := &models.DeploymentMetrics{
		CPUUsage:     45.5,
		MemoryUsage:  2.1,
		DiskUsage:    15.3,
		NetworkIn:    1024,
		NetworkOut:   512,
		QPS:          1234,
		ResponseTime: 45,
		Timestamp:    time.Now().Unix(),
	}

	c.JSON(http.StatusOK, gin.H{"data": metrics})
}

// DeleteDeployment 删除部署
func (h *DeploymentHandler) DeleteDeployment(c *gin.Context) {
	id := c.Param("id")

	deployment, err := models.GetDeployment(h.db.GetDB(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	// 删除 ECS 实例
	if err := aliyun.DeleteInstance(deployment.InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 删除数据库记录
	models.DeleteDeployment(h.db.GetDB(), id)

	c.JSON(http.StatusOK, gin.H{"message": "Deployment deleted"})
}

// 辅助函数
func getInstanceType(plan string) string {
	switch plan {
	case "community":
		return "ecs.t5-lc1m1.small" // 1 核 1GB
	case "pro":
		return "ecs.n4.small" // 2 核 4GB
	case "enterprise":
		return "ecs.n4.large" // 4 核 8GB
	default:
		return "ecs.n4.small"
	}
}

func getBandwidth(plan string) int {
	switch plan {
	case "community":
		return 1
	case "pro":
		return 3
	case "enterprise":
		return 5
	default:
		return 3
	}
}

func getDiskSize(plan string) int {
	switch plan {
	case "community":
		return 20
	case "pro":
		return 40
	case "enterprise":
		return 100
	default:
		return 40
	}
}

func generatePassword() string {
	// 生成 16 位随机密码
	return uuid.New().String()[:16]
}

type CreateDeploymentRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Plan     string `json:"plan" binding:"required"`
	Domain   string `json:"domain"`
	Username string `json:"username"`
}
