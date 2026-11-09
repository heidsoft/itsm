package incident

import (
	"context"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// 本包的五个能力面由组合根注入实现。方法集 = handlers/incident 下所有生产文件
// 对相应字段的真实调用并集（handler.go 与 service.go 各贡献一部分），因此新增
// 调用点时必须在契约里同步声明，编译期即暴露漂移。
//
// 依赖方向：handlers/incident -> dto/repository/ent，不再反向引用共享层 service 包。

type IncidentProductionService interface {
	AcknowledgeIncident(ctx context.Context, id, userID, tenantID int) error
	AssignIncident(ctx context.Context, id, assigneeID, tenantID int) (*dto.IncidentResponse, error)
	CloseIncident(ctx context.Context, id, userID, tenantID int, closeNotes string) error
	CreateIncidentAlert(ctx context.Context, req *dto.CreateIncidentAlertRequest, tenantID int) (*dto.IncidentAlertResponse, error)
	CreateIncidentEvent(ctx context.Context, req *dto.CreateIncidentEventRequest, tenantID int) (*dto.IncidentEventResponse, error)
	DeleteIncident(ctx context.Context, id, tenantID int) error
	DeleteIncidentComment(ctx context.Context, incidentID, commentID, tenantID int) error
	EscalateToMajorIncident(ctx context.Context, id, userID, tenantID int, req *dto.EscalateMajorIncidentRequest) error
	GetIncidentAlerts(ctx context.Context, incidentID, tenantID int) ([]dto.IncidentAlertResponse, error)
	GetIncidentComments(ctx context.Context, incidentID, tenantID int) ([]dto.IncidentEventResponse, error)
	GetIncidentEvents(ctx context.Context, incidentID, tenantID int) ([]dto.IncidentEventResponse, error)
	GetIncidentMetrics(ctx context.Context, incidentID, tenantID int) ([]dto.IncidentMetricResponse, error)
	GetIncidentMonitoring(ctx context.Context, req *dto.IncidentMonitoringRequest, tenantID int) (*dto.IncidentMonitoringResponse, error)
	ReopenIncident(ctx context.Context, id, userID, tenantID int) error
	ResolveIncident(ctx context.Context, id, userID, tenantID int, resolution, rootCause string) error
	ResolveIncidentAlert(ctx context.Context, alertID, userID, tenantID int) error
	ValidateAssignee(ctx context.Context, assigneeID, tenantID int) error
}

type IncidentMonitoringProvider interface {
	AnalyzeIncidentImpact(ctx context.Context, incidentID, tenantID int) (map[string]interface{}, error)
}

type IncidentAlertingProvider interface {
	AcknowledgeAlert(ctx context.Context, alertID, userID, tenantID int) error
	GetActiveAlerts(ctx context.Context, tenantID, page, size int) ([]*dto.IncidentAlertResponse, int, error)
	GetAlertStatistics(ctx context.Context, tenantID int, startTime, endTime time.Time) (map[string]interface{}, error)
}

type IncidentRootCauseService interface {
	CreateProblemFromIncident(ctx context.Context, incidentID, createdBy, tenantID int, req *dto.ConvertIncidentToProblemRequest) (*ent.Problem, error)
}

type IncidentSLAMonitor interface {
	BatchGetSLAStates(ctx context.Context, tenantID int, aggregateType string, aggregateIDs []int) (map[int]*ent.SLAState, error)
	GetSLAState(ctx context.Context, tenantID int, aggregateType string, aggregateID int) (*ent.SLAState, error)
	PauseSLA(ctx context.Context, tenantID int, entityType string, entityID int, reason string) error
	ResumeSLA(ctx context.Context, tenantID int, entityType string, entityID int) error
}
