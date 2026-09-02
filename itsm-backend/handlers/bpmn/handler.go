package bpmn

import (
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// Handler owns every BPMN HTTP entry point while the workflow engine and
// domain services remain in the service package.
type Handler struct {
	workflow       *WorkflowHandler
	processTrigger *ProcessTriggerHandler
	dashboard      *DashboardHandler
	monitoring     *MonitoringHandler
	aiGenerator    *AIGeneratorHandler
	lint           *LintHandler
}

func NewHandler(
	workflow *WorkflowHandler,
	processTrigger *ProcessTriggerHandler,
	dashboard *DashboardHandler,
	monitoring *MonitoringHandler,
	aiGenerator *AIGeneratorHandler,
	lint *LintHandler,
) *Handler {
	return &Handler{
		workflow:       workflow,
		processTrigger: processTrigger,
		dashboard:      dashboard,
		monitoring:     monitoring,
		aiGenerator:    aiGenerator,
		lint:           lint,
	}
}

// RegisterRoutes registers both the canonical /bpmn routes and the retained
// /workflow compatibility surface from one production handler owner.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	if h == nil {
		return
	}
	if h.workflow != nil {
		h.workflow.RegisterRoutes(r)
		h.registerWorkflowCompatibilityRoutes(r)
	}
	if h.processTrigger != nil {
		h.processTrigger.RegisterRoutes(r)
	}
	if h.dashboard != nil {
		h.dashboard.RegisterRoutes(r)
	}
	if h.aiGenerator != nil {
		h.aiGenerator.RegisterRoutes(r)
	}
	if h.lint != nil {
		h.lint.RegisterRoutes(r)
	}
	if h.monitoring != nil {
		h.monitoring.RegisterRoutes(r)
	}
}

func (h *Handler) registerWorkflowCompatibilityRoutes(r *gin.RouterGroup) {
	workflow := r.Group("/workflow")
	workflow.GET("/instances", middleware.RequirePermission("process_instance", "read"), h.workflow.ListProcessInstances)
	workflow.GET("/instances/:id", middleware.RequirePermission("process_instance", "read"), h.workflow.GetProcessInstance)
	workflow.POST("/instances", middleware.RequirePermission("process_instance", "create"), h.workflow.StartProcess)
	workflow.PUT("/instances/:id/terminate", middleware.RequirePermission("process_instance", "update"), h.workflow.TerminateProcess)
	workflow.PUT("/instances/:id/suspend", middleware.RequirePermission("process_instance", "update"), h.workflow.SuspendProcess)
	workflow.PUT("/instances/:id/resume", middleware.RequirePermission("process_instance", "update"), h.workflow.ResumeProcess)
	workflow.GET("/tasks", middleware.RequirePermission("task", "read"), h.workflow.ListUserTasks)
	workflow.GET("/tasks/all", middleware.RequirePermission("task", "admin"), h.workflow.ListAllTasks)
	workflow.PUT("/tasks/:id/complete", middleware.RequirePermission("task", "update"), h.workflow.CompleteTask)
	workflow.POST("/tasks/:id/claim", middleware.RequirePermission("task", "update"), h.workflow.ClaimTask)
	workflow.PUT("/tasks/:id/reassign", middleware.RequirePermission("task", "update"), h.workflow.ReassignTask)
	workflow.PUT("/tasks/:id/terminate", middleware.RequirePermission("process_instance", "update"), h.workflow.TerminateTask)
}
