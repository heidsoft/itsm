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

	BPMNPendingInstances = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "itsm_bpmn_pending_instances",
			Help: "Number of pending BPMN instances waiting for execution",
		},
		[]string{"process_definition", "tenant_id"},
	)

	// Timer metrics - BPMN 定时器指标（PRD ITSM-PRD-2026-002 §8.1）
	// status 标签区分触发结果：success = 回调推进成功；failed = 回调报错（进入重试）。
	TimerFiredTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_timer_fired_total",
			Help: "Total number of BPMN timer firings by outcome",
		},
		[]string{"type", "status", "tenant_id"},
	)

	TimerFireLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "itsm_timer_fire_latency_seconds",
			Help:    "Latency between timer fire_at and actual firing",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 15, 60, 300, 900},
		},
		[]string{"type", "tenant_id"},
	)

	TimerRecoveryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_timer_recovery_total",
			Help: "Total number of timers rescheduled by recovery scans",
		},
		[]string{"tenant_id"},
	)

	TimerRetryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_timer_retry_total",
			Help: "Total number of timer fire retries after callback failure",
		},
		[]string{"type", "tenant_id"},
	)

	// TimerPausedTotal 记录流程实例挂起时被取消的活跃定时器数量
	// （恢复时由 reregisterInstanceTimers 重建，重建不计数——
	// 该指标用于发现"挂起后 timer 仍在触发"的泄漏回归）。
	TimerPausedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_timer_paused_total",
			Help: "Total number of active timers cancelled due to process instance suspension",
		},
		[]string{"tenant_id"},
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

	// AI metrics - AI 服务指标
	// AIPersistErrors 记录 AI 对话/工具审计的持久化失败次数。
	// 历史背景：L8 安全审计发现 handlers/ai/service.go 中 5 处 CreateMessage / CreateToolInvocation
	// 调用以 _, _ = 形式丢弃错误，导致 DB 抖动时对话历史无声丢失。
	// operation 标签区分写入路径（create_message | create_tool_invocation），
	// role 标签区分消息角色（user | assistant | empty）；tenant_id 用于多租户归因。
	AIPersistErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itsm_ai_persist_errors_total",
			Help: "Total number of AI persistence failures (conversation messages, tool audit)",
		},
		[]string{"operation", "role", "tenant_id"},
	)
)
