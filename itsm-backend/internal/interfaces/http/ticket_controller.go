// Package interfaces provides HTTP/REST API interfaces following Clean Architecture
package interfaces

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"itsm-backend/internal/application"
	"itsm-backend/internal/domain/shared"
)

// TicketController handles HTTP requests for ticket operations
type TicketController struct {
	ticketService *application.TicketService
	logger        *zap.SugaredLogger
}

// NewTicketController creates a new ticket controller
func NewTicketController(ticketService *application.TicketService, logger *zap.SugaredLogger) *TicketController {
	return &TicketController{
		ticketService: ticketService,
		logger:        logger,
	}
}

// CreateTicketRequest represents the request to create a ticket
type CreateTicketRequest struct {
	Title       string `json:"title" binding:"required,max=200"`
	Description string `json:"description" binding:"max=2000"`
	Priority    string `json:"priority" binding:"required,oneof=low normal high urgent critical"`
	Category    string `json:"category" binding:"required"`
}

// CreateTicketResponse represents the response after creating a ticket
type CreateTicketResponse struct {
	Success bool                            `json:"success"`
	Data    *application.CreateTicketResult `json:"data,omitempty"`
	Error   *ErrorDetails                   `json:"error,omitempty"`
}

// ErrorDetails represents error information
type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// CreateTicket handles POST /api/v1/tickets
// @Summary Create a new ticket
// @Description Create a new support ticket
// @Tags tickets
// @Accept json
// @Produce json
// @Param request body CreateTicketRequest true "Ticket creation request"
// @Success 201 {object} CreateTicketResponse
// @Failure 400 {object} CreateTicketResponse
// @Failure 500 {object} CreateTicketResponse
// @Router /api/v1/tickets [post]
func (c *TicketController) CreateTicket(ctx *gin.Context) {
	var req CreateTicketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.respondWithError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err.Error())
		return
	}

	// Get user context
	userID, tenantID := c.getUserContext(ctx)

	// Create application command
	cmd := application.CreateTicketCommand{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Category:    req.Category,
		CreatedBy:   userID,
		TenantID:    tenantID,
	}

	// Execute command
	result, err := c.ticketService.CreateTicket(ctx.Request.Context(), cmd)
	if err != nil {
		c.logger.Errorw("Failed to create ticket", "error", err, "user", userID)
		c.respondWithError(ctx, http.StatusInternalServerError, "CREATION_FAILED", "Failed to create ticket", err.Error())
		return
	}

	c.respondWithSuccess(ctx, http.StatusCreated, result)
}

// AssignTicketRequest represents the request to assign a ticket
type AssignTicketRequest struct {
	AssignedTo   string  `json:"assigned_to" binding:"required"`
	TeamID       *string `json:"team_id"`
	Instructions string  `json:"instructions"`
}

// AssignTicket handles PUT /api/v1/tickets/:id/assign
// @Summary Assign a ticket
// @Description Assign a ticket to a user or team
// @Tags tickets
// @Accept json
// @Produce json
// @Param id path string true "Ticket ID"
// @Param request body AssignTicketRequest true "Assignment request"
// @Success 200 {object} BaseResponse
// @Failure 400 {object} BaseResponse
// @Failure 404 {object} BaseResponse
// @Failure 500 {object} BaseResponse
// @Router /api/v1/tickets/{id}/assign [put]
func (c *TicketController) AssignTicket(ctx *gin.Context) {
	ticketID := ctx.Param("id")
	if ticketID == "" {
		c.respondWithError(ctx, http.StatusBadRequest, "MISSING_ID", "Ticket ID is required", "")
		return
	}

	var req AssignTicketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.respondWithError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err.Error())
		return
	}

	// Get user context
	assignedBy, tenantID := c.getUserContext(ctx)

	// Create command
	cmd := application.AssignTicketCommand{
		TicketID:     ticketID,
		AssignedTo:   shared.UserID(req.AssignedTo),
		AssignedBy:   assignedBy,
		Instructions: req.Instructions,
		TenantID:     tenantID,
	}

	if req.TeamID != nil {
		teamID := shared.TeamID(*req.TeamID)
		cmd.TeamID = &teamID
	}

	// Execute command
	err := c.ticketService.AssignTicket(ctx.Request.Context(), cmd)
	if err != nil {
		c.logger.Errorw("Failed to assign ticket", "error", err, "ticket_id", ticketID)

		// Handle specific error types
		if err.Error() == "ticket not found in tenant" {
			c.respondWithError(ctx, http.StatusNotFound, "TICKET_NOT_FOUND", "Ticket not found", "")
			return
		}

		c.respondWithError(ctx, http.StatusInternalServerError, "ASSIGNMENT_FAILED", "Failed to assign ticket", err.Error())
		return
	}

	c.respondWithSuccess(ctx, http.StatusOK, gin.H{"message": "Ticket assigned successfully"})
}

// UpdateStatusRequest represents the request to update ticket status
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=new open in_progress pending resolved closed cancelled"`
	Reason string `json:"reason"`
}

// UpdateTicketStatus handles PUT /api/v1/tickets/:id/status
// @Summary Update ticket status
// @Description Update the status of a ticket
// @Tags tickets
// @Accept json
// @Produce json
// @Param id path string true "Ticket ID"
// @Param request body UpdateStatusRequest true "Status update request"
// @Success 200 {object} BaseResponse
// @Failure 400 {object} BaseResponse
// @Failure 404 {object} BaseResponse
// @Failure 500 {object} BaseResponse
// @Router /api/v1/tickets/{id}/status [put]
func (c *TicketController) UpdateTicketStatus(ctx *gin.Context) {
	ticketID := ctx.Param("id")
	if ticketID == "" {
		c.respondWithError(ctx, http.StatusBadRequest, "MISSING_ID", "Ticket ID is required", "")
		return
	}

	var req UpdateStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.respondWithError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err.Error())
		return
	}

	// Get user context
	changedBy, tenantID := c.getUserContext(ctx)

	// Create command
	cmd := application.UpdateTicketStatusCommand{
		TicketID:  ticketID,
		NewStatus: req.Status,
		Reason:    req.Reason,
		ChangedBy: changedBy,
		TenantID:  tenantID,
	}

	// Execute command
	err := c.ticketService.UpdateTicketStatus(ctx.Request.Context(), cmd)
	if err != nil {
		c.logger.Errorw("Failed to update ticket status", "error", err, "ticket_id", ticketID)

		if err.Error() == "ticket not found in tenant" {
			c.respondWithError(ctx, http.StatusNotFound, "TICKET_NOT_FOUND", "Ticket not found", "")
			return
		}

		if err.Error() == "invalid status transition" {
			c.respondWithError(ctx, http.StatusBadRequest, "INVALID_TRANSITION", "Invalid status transition", err.Error())
			return
		}

		c.respondWithError(ctx, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update ticket status", err.Error())
		return
	}

	c.respondWithSuccess(ctx, http.StatusOK, gin.H{"message": "Ticket status updated successfully"})
}

// AddCommentRequest represents the request to add a comment
type AddCommentRequest struct {
	Content   string `json:"content" binding:"required,max=2000"`
	IsPrivate bool   `json:"is_private"`
}

// AddComment handles POST /api/v1/tickets/:id/comments
// @Summary Add a comment to a ticket
// @Description Add a comment to a ticket
// @Tags tickets
// @Accept json
// @Produce json
// @Param id path string true "Ticket ID"
// @Param request body AddCommentRequest true "Comment request"
// @Success 201 {object} BaseResponse
// @Failure 400 {object} BaseResponse
// @Failure 404 {object} BaseResponse
// @Failure 500 {object} BaseResponse
// @Router /api/v1/tickets/{id}/comments [post]
func (c *TicketController) AddComment(ctx *gin.Context) {
	ticketID := ctx.Param("id")
	if ticketID == "" {
		c.respondWithError(ctx, http.StatusBadRequest, "MISSING_ID", "Ticket ID is required", "")
		return
	}

	var req AddCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.respondWithError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err.Error())
		return
	}

	// Get user context
	authorID, tenantID := c.getUserContext(ctx)

	// Create command
	cmd := application.AddCommentCommand{
		TicketID:  ticketID,
		Content:   req.Content,
		AuthorID:  authorID,
		IsPrivate: req.IsPrivate,
		TenantID:  tenantID,
	}

	// Execute command
	err := c.ticketService.AddComment(ctx.Request.Context(), cmd)
	if err != nil {
		c.logger.Errorw("Failed to add comment", "error", err, "ticket_id", ticketID)

		if err.Error() == "ticket not found in tenant" {
			c.respondWithError(ctx, http.StatusNotFound, "TICKET_NOT_FOUND", "Ticket not found", "")
			return
		}

		c.respondWithError(ctx, http.StatusInternalServerError, "COMMENT_FAILED", "Failed to add comment", err.Error())
		return
	}

	c.respondWithSuccess(ctx, http.StatusCreated, gin.H{"message": "Comment added successfully"})
}

// GetTicket handles GET /api/v1/tickets/:id
// @Summary Get a ticket by ID
// @Description Retrieve a ticket with all its details
// @Tags tickets
// @Produce json
// @Param id path string true "Ticket ID"
// @Success 200 {object} TicketDetailsResponse
// @Failure 404 {object} BaseResponse
// @Failure 500 {object} BaseResponse
// @Router /api/v1/tickets/{id} [get]
func (c *TicketController) GetTicket(ctx *gin.Context) {
	ticketID := ctx.Param("id")
	if ticketID == "" {
		c.respondWithError(ctx, http.StatusBadRequest, "MISSING_ID", "Ticket ID is required", "")
		return
	}

	// Get user context
	_, tenantID := c.getUserContext(ctx)

	// Create query
	query := application.GetTicketQuery{
		TicketID: ticketID,
		TenantID: tenantID,
	}

	// Execute query
	result, err := c.ticketService.GetTicket(ctx.Request.Context(), query)
	if err != nil {
		c.logger.Errorw("Failed to get ticket", "error", err, "ticket_id", ticketID)

		if err.Error() == "ticket not found" {
			c.respondWithError(ctx, http.StatusNotFound, "TICKET_NOT_FOUND", "Ticket not found", "")
			return
		}

		c.respondWithError(ctx, http.StatusInternalServerError, "RETRIEVAL_FAILED", "Failed to retrieve ticket", err.Error())
		return
	}

	c.respondWithSuccess(ctx, http.StatusOK, result)
}

// SearchTicketsRequest represents the request to search tickets
type SearchTicketsRequest struct {
	Status     []string `form:"status"`
	Priority   []string `form:"priority"`
	AssignedTo *string  `form:"assigned_to"`
	CreatedBy  *string  `form:"created_by"`
	Category   string   `form:"category"`
	Keywords   string   `form:"keywords"`
	DateFrom   *string  `form:"date_from"`
	DateTo     *string  `form:"date_to"`
	Page       int      `form:"page,default=1" binding:"min=1"`
	PageSize   int      `form:"page_size,default=20" binding:"min=1,max=100"`
	SortBy     string   `form:"sort_by"`
	SortOrder  string   `form:"sort_order,default=desc"`
}

// SearchTickets handles GET /api/v1/tickets
// @Summary Search tickets
// @Description Search tickets with various filters
// @Tags tickets
// @Produce json
// @Param status query []string false "Status filter"
// @Param priority query []string false "Priority filter"
// @Param assigned_to query string false "Assigned to filter"
// @Param created_by query string false "Created by filter"
// @Param category query string false "Category filter"
// @Param keywords query string false "Keywords search"
// @Param date_from query string false "Date from filter"
// @Param date_to query string false "Date to filter"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param sort_by query string false "Sort by field"
// @Param sort_order query string false "Sort order" default(desc)
// @Success 200 {object} SearchTicketsResponse
// @Failure 400 {object} BaseResponse
// @Failure 500 {object} BaseResponse
// @Router /api/v1/tickets [get]
func (c *TicketController) SearchTickets(ctx *gin.Context) {
	var req SearchTicketsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		c.respondWithError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request parameters", err.Error())
		return
	}

	// Get user context
	_, tenantID := c.getUserContext(ctx)

	// Create query
	query := application.SearchTicketsQuery{
		TenantID:   tenantID,
		Status:     req.Status,
		Priority:   req.Priority,
		AssignedTo: req.AssignedTo,
		CreatedBy:  req.CreatedBy,
		Category:   req.Category,
		Keywords:   req.Keywords,
		DateFrom:   req.DateFrom,
		DateTo:     req.DateTo,
		Page:       req.Page,
		PageSize:   req.PageSize,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
	}

	// Execute query
	result, err := c.ticketService.SearchTickets(ctx.Request.Context(), query)
	if err != nil {
		c.logger.Errorw("Failed to search tickets", "error", err)
		c.respondWithError(ctx, http.StatusInternalServerError, "SEARCH_FAILED", "Failed to search tickets", err.Error())
		return
	}

	c.respondWithSuccess(ctx, http.StatusOK, result)
}

// Response types
type BaseResponse struct {
	Success bool          `json:"success"`
	Data    interface{}   `json:"data,omitempty"`
	Error   *ErrorDetails `json:"error,omitempty"`
}

type TicketDetailsResponse struct {
	Success bool                       `json:"success"`
	Data    *application.TicketDetails `json:"data,omitempty"`
	Error   *ErrorDetails              `json:"error,omitempty"`
}

type SearchTicketsResponse struct {
	Success bool                             `json:"success"`
	Data    *application.SearchTicketsResult `json:"data,omitempty"`
	Error   *ErrorDetails                    `json:"error,omitempty"`
}

// Helper methods
func (c *TicketController) getUserContext(ctx *gin.Context) (shared.UserID, shared.TenantID) {
	// In a real implementation, these would be extracted from JWT token or session
	userID := ctx.GetHeader("X-User-ID")
	tenantID := ctx.GetHeader("X-Tenant-ID")

	if userID == "" {
		userID = "anonymous"
	}
	if tenantID == "" {
		tenantID = "1" // Default tenant
	}

	return shared.UserID(userID), shared.TenantID(tenantID)
}

func (c *TicketController) respondWithSuccess(ctx *gin.Context, statusCode int, data interface{}) {
	ctx.JSON(statusCode, BaseResponse{
		Success: true,
		Data:    data,
	})
}

func (c *TicketController) respondWithError(ctx *gin.Context, statusCode int, code, message, details string) {
	ctx.JSON(statusCode, BaseResponse{
		Success: false,
		Error: &ErrorDetails{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// RegisterRoutes registers all ticket routes
func (c *TicketController) RegisterRoutes(router *gin.RouterGroup) {
	tickets := router.Group("/tickets")
	{
		tickets.POST("", c.CreateTicket)
		tickets.GET("", c.SearchTickets)
		tickets.GET("/:id", c.GetTicket)
		tickets.PUT("/:id/assign", c.AssignTicket)
		tickets.PUT("/:id/status", c.UpdateTicketStatus)
		tickets.POST("/:id/comments", c.AddComment)
	}
}
