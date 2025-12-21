package common

import (
	"net/http"
	"strconv"

	"itsm-backend/common"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Auth

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		TenantID int    `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.svc.Login(c.Request.Context(), req.Username, req.Password, req.TenantID)
	if err != nil {
		common.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}

	common.Success(c, res)
}

// Users

func (h *Handler) GetMe(c *gin.Context) {
	userID := c.GetInt("user_id")
	u, err := h.svc.GetUser(c.Request.Context(), userID)
	if err != nil {
		common.Fail(c, http.StatusNotFound, "User not found")
		return
	}
	common.Success(c, u)
}

func (h *Handler) ListUsers(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	users, err := h.svc.ListUsers(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, users)
}

// Departments

func (h *Handler) GetDepartmentTree(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	tree, err := h.svc.GetDepartmentTree(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, tree)
}

// Teams

func (h *Handler) ListTeams(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	teams, err := h.svc.ListTeams(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, teams)
}

// Tags

func (h *Handler) ListTags(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	tags, err := h.svc.ListTags(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, tags)
}

// Audit Logs

func (h *Handler) GetAuditLogs(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	userID, _ := strconv.Atoi(c.Query("user_id"))
	logs, err := h.svc.GetAuditLogs(c.Request.Context(), tenantID, userID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, logs)
}
