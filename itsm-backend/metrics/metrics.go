package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Ticket metrics - 工单指标
	TicketCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_ticket_created_total",
			Help: "Total number of tickets created",
		},
		[]string{"type", "priority", "tenant_id"},
	)

	TicketResolvedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_ticket_resolved_total",
			Help: "Total number of tickets resolved",
		},
		[]string{"type", "tenant_id"},
	)

	TicketClosedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_ticket_closed_total",
			Help: "Total number of tickets closed",
		},
		[]string{"type", "tenant_id"},
	)

	TicketResolutionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "itsm_ticket_resolution_duration_seconds",
			Help:    "Time taken to resolve tickets",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type", "priority"},
	)

	TicketActiveGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "itsm_ticket_active",
			Help: "Number of currently active tickets",
		},
		[]string{"type", "status", "tenant_id"},
	)

	// SLA metrics - SLA 指标
	SLABreachedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_sla_breached_total",
			Help: "Total number of SLA breaches",
		},
		[]string{"sla_name", "tenant_id"},
	)

	SLAWarningTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_sla_warning_total",
			Help: "Total number of SLA warnings",
		},
		[]string{"sla_name", "tenant_id"},
	)

	SLAComplianceRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "itsm_sla_compliance_rate",
			Help: "Current SLA compliance rate",
		},
		[]string{"tenant_id"},
	)

	// BPMN workflow metrics - BPMN 工作流指标
	BPMNWorkflowStartedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_bpmn_workflow_started_total",
			Help: "Total number of BPMN workflows started",
		},
		[]string{"process_definition", "tenant_id"},
	)

	BPMNWorkflowCompletedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_bpmn_workflow_completed_total",
			Help: "Total number of BPMN workflows completed",
		},
		[]string{"process_definition", "tenant_id"},
	)

	BPMNWorkflowFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_bpmn_workflow_failed_total",
			Help: "Total number of BPMN workflows failed",
		},
		[]string{"process_definition", "error_type", "tenant_id"},
	)

	BPMNWorkflowDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "itsm_bpmn_workflow_duration_seconds",
			Help:    "Time taken to complete BPMN workflows",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"process_definition"},
	)

	BPMNActiveInstances = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "itsm_bpmn_active_instances",
			Help: "Number of currently active BPMN instances",
		},
		[]string{"process_definition", "tenant_id"},
	)

	// API metrics - API 指标
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "itsm_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
)
