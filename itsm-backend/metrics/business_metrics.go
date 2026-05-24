package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// BusinessMetrics 业务指标收集器
type BusinessMetrics struct {
	logger *zap.Logger

	// 工单指标
	TicketsCreated       *prometheus.CounterVec
	TicketsResolved      *prometheus.CounterVec
	TicketResolutionTime *prometheus.HistogramVec
	TicketsByStatus      *prometheus.GaugeVec
	TicketsByPriority    *prometheus.GaugeVec

	// 事件指标
	IncidentsCreated    *prometheus.CounterVec
	IncidentsResolved   *prometheus.CounterVec
	IncidentMTTR        *prometheus.HistogramVec
	IncidentsByCategory *prometheus.GaugeVec

	// SLA指标
	SLACompliance   *prometheus.GaugeVec
	SLAViolations   *prometheus.CounterVec
	SLAResponseTime *prometheus.HistogramVec

	// 变更指标
	ChangesCreated     *prometheus.CounterVec
	ChangesSuccess     *prometheus.CounterVec
	ChangesFailed      *prometheus.CounterVec
	ChangeApprovalTime *prometheus.HistogramVec

	// 知识库指标
	KnowledgeArticlesCreated *prometheus.CounterVec
	KnowledgeSearches        *prometheus.CounterVec
	KnowledgeArticleViews    *prometheus.CounterVec

	// 用户指标
	ActiveUsers  *prometheus.GaugeVec
	UserSessions *prometheus.CounterVec
	UserActions  *prometheus.CounterVec
}

// NewBusinessMetrics 创建业务指标收集器
func NewBusinessMetrics(logger *zap.Logger) *BusinessMetrics {
	return &BusinessMetrics{
		logger: logger,

		// 工单指标
		TicketsCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "itsm_tickets_created_total",
				Help: "Total number of tickets created",
			},
			[]string{"tenant_id", "category", "priority"},
		),

		TicketsResolved: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "itsm_tickets_resolved_total",
				Help: "Total number of tickets resolved",
			},
			[]string{"tenant_id", "category"},
		),

		TicketResolutionTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "itsm_ticket_resolution_time_seconds",
				Help:    "Time taken to resolve tickets",
				Buckets: []float64{60, 300, 900, 1800, 3600, 7200, 14400, 28800, 86400},
			},
			[]string{"tenant_id", "priority"},
		),

		TicketsByStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "itsm_tickets_by_status",
				Help: "Current number of tickets by status",
			},
			[]string{"tenant_id", "status"},
		),

		// 事件指标
		IncidentsCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "itsm_incidents_created_total",
				Help: "Total number of incidents created",
			},
			[]string{"tenant_id", "category", "priority"},
		),

		IncidentsResolved: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "itsm_incidents_resolved_total",
				Help: "Total number of incidents resolved",
			},
			[]string{"tenant_id", "category"},
		),

		IncidentMTTR: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "itsm_incident_mttr_seconds",
				Help:    "Mean time to resolve incidents",
				Buckets: []float64{60, 300, 900, 1800, 3600, 7200, 14400, 28800, 86400},
			},
			[]string{"tenant_id", "priority"},
		),

		// SLA指标
		SLACompliance: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "itsm_sla_compliance_ratio",
				Help: "SLA compliance ratio",
			},
			[]string{"tenant_id", "sla_type"},
		),

		SLAViolations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "itsm_sla_violations_total",
				Help: "Total number of SLA violations",
			},
			[]string{"tenant_id", "sla_type"},
		),

		// 变更指标
		ChangesCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "itsm_changes_created_total",
				Help: "Total number of changes created",
			},
			[]string{"tenant_id", "change_type"},
		),

		ChangesSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "itsm_changes_success_total",
				Help: "Total number of successful changes",
			},
			[]string{"tenant_id"},
		),

		ChangesFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "itsm_changes_failed_total",
				Help: "Total number of failed changes",
			},
			[]string{"tenant_id"},
		),

		// 知识库指标
		KnowledgeArticlesCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "itsm_knowledge_articles_created_total",
				Help: "Total number of knowledge articles created",
			},
			[]string{"tenant_id", "category"},
		),

		KnowledgeSearches: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "itsm_knowledge_searches_total",
				Help: "Total number of knowledge base searches",
			},
			[]string{"tenant_id"},
		),

		// 用户指标
		ActiveUsers: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "itsm_active_users",
				Help: "Current number of active users",
			},
			[]string{"tenant_id"},
		),
	}
}
