package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/incidentalert"
	"itsm-backend/ent/incidentmetric"

	"go.uber.org/zap"
)

type IncidentMonitoringService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewIncidentMonitoringService(client *ent.Client, logger *zap.SugaredLogger) *IncidentMonitoringService {
	return &IncidentMonitoringService{
		client: client,
		logger: logger,
	}
}

// PrometheusMetrics 模拟Prometheus指标结构
type PrometheusMetrics struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// GrafanaDashboard 模拟Grafana仪表板结构
type GrafanaDashboard struct {
	Dashboard struct {
		Title  string `json:"title"`
		Panels []struct {
			Title   string `json:"title"`
			Type    string `json:"type"`
			Targets []struct {
				Expr string `json:"expr"`
			} `json:"targets"`
		} `json:"panels"`
	} `json:"dashboard"`
}

// CollectMetricsFromPrometheus 从Prometheus收集指标
func (s *IncidentMonitoringService) CollectMetricsFromPrometheus(ctx context.Context, queries []string, tenantID int) error {
	s.logger.Infow("Collecting metrics from Prometheus", "queries", queries, "tenant_id", tenantID)

	for _, query := range queries {
		// 模拟从Prometheus查询指标
		metrics, err := s.queryPrometheus(query)
		if err != nil {
			s.logger.Errorw("Failed to query Prometheus", "error", err, "query", query)
			continue
		}

		// 处理查询结果
		for _, result := range metrics.Data.Result {
			if len(result.Value) < 2 {
				continue
			}

			// 解析指标值
			timestamp, ok := result.Value[0].(float64)
			if !ok {
				continue
			}

			value, ok := result.Value[1].(string)
			if !ok {
				continue
			}

			// 创建事件指标记录
			err = s.createIncidentMetricFromPrometheus(ctx, result.Metric, value, timestamp, tenantID)
			if err != nil {
				s.logger.Errorw("Failed to create incident metric", "error", err)
			}
		}
	}

	return nil
}

// queryPrometheus 查询Prometheus（模拟实现）
func (s *IncidentMonitoringService) queryPrometheus(query string) (*PrometheusMetrics, error) {
	// 这里是模拟实现，实际应该调用Prometheus API
	s.logger.Infow("Querying Prometheus", "query", query)

	// 模拟返回数据
	metrics := &PrometheusMetrics{
		Status: "success",
		Data: struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		}{
			ResultType: "vector",
			Result: []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			}{
				{
					Metric: map[string]string{
						"__name__": "cpu_usage_percent",
						"instance": "server-01",
						"job":      "node-exporter",
					},
					Value: []interface{}{
						float64(time.Now().Unix()),
						"85.5",
					},
				},
			},
		},
	}

	return metrics, nil
}

// createIncidentMetricFromPrometheus 从Prometheus数据创建事件指标
func (s *IncidentMonitoringService) createIncidentMetricFromPrometheus(ctx context.Context, metric map[string]string, value string, timestamp float64, tenantID int) error {
	metricName, ok := metric["__name__"]
	if !ok {
		return fmt.Errorf("metric name not found")
	}

	// 查找相关的事件
	incidents, err := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.StatusIn("new", "in_progress"),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to get incidents: %w", err)
	}

	// 为每个事件创建指标记录
	for _, incidentEntity := range incidents {
		_, err := s.client.IncidentMetric.Create().
			SetIncidentID(incidentEntity.ID).
			SetMetricType("prometheus").
			SetMetricName(metricName).
			SetMetricValue(parseFloatValue(value)).
			SetMeasuredAt(time.Unix(int64(timestamp), 0)).
			SetTags(metric).
			SetMetadata(map[string]interface{}{
				"source": "prometheus",
				"query":  "prometheus_query",
			}).
			SetTenantID(tenantID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.logger.Errorw("Failed to create incident metric", "error", err, "incident_id", incidentEntity.ID)
		}
	}

	return nil
}

// CreateGrafanaDashboard 创建Grafana仪表板
func (s *IncidentMonitoringService) CreateGrafanaDashboard(ctx context.Context, tenantID int) (*GrafanaDashboard, error) {
	s.logger.Infow("Creating Grafana dashboard", "tenant_id", tenantID)

	dashboard := &GrafanaDashboard{
		Dashboard: struct {
			Title  string `json:"title"`
			Panels []struct {
				Title   string `json:"title"`
				Type    string `json:"type"`
				Targets []struct {
					Expr string `json:"expr"`
				} `json:"targets"`
			} `json:"panels"`
		}{
			Title: "ITSM Incident Monitoring Dashboard",
			Panels: []struct {
				Title   string `json:"title"`
				Type    string `json:"type"`
				Targets []struct {
					Expr string `json:"expr"`
				} `json:"targets"`
			}{
				{
					Title: "Incident Count by Status",
					Type:  "stat",
					Targets: []struct {
						Expr string `json:"expr"`
					}{
						{Expr: "incident_count_by_status"},
					},
				},
				{
					Title: "Incident Resolution Time",
					Type:  "graph",
					Targets: []struct {
						Expr string `json:"expr"`
					}{
						{Expr: "incident_resolution_time"},
					},
				},
				{
					Title: "Critical Incidents",
					Type:  "table",
					Targets: []struct {
						Expr string `json:"expr"`
					}{
						{Expr: "critical_incidents"},
					},
				},
				{
					Title: "Incident Trends",
					Type:  "graph",
					Targets: []struct {
						Expr string `json:"expr"`
					}{
						{Expr: "incident_trends"},
					},
				},
			},
		},
	}

	return dashboard, nil
}

// AnalyzeIncidentImpact 分析事件影响
func (s *IncidentMonitoringService) AnalyzeIncidentImpact(ctx context.Context, incidentID int, tenantID int) (map[string]interface{}, error) {
	s.logger.Infow("Analyzing incident impact", "incident_id", incidentID, "tenant_id", tenantID)

	// 获取事件信息
	incidentEntity, err := s.client.Incident.Query().
		Where(
			incident.IDEQ(incidentID),
			incident.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found")
		}
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}

	// 获取事件指标
	metrics, err := s.client.IncidentMetric.Query().
		Where(
			incidentmetric.IncidentIDEQ(incidentID),
			incidentmetric.TenantIDEQ(tenantID),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get incident metrics", "error", err)
		return nil, fmt.Errorf("failed to get incident metrics: %w", err)
	}

	// 分析影响
	impactAnalysis := map[string]interface{}{
		"incident_id":     incidentID,
		"incident_number": incidentEntity.IncidentNumber,
		"title":           incidentEntity.Title,
		"severity":        incidentEntity.Severity,
		"priority":        incidentEntity.Priority,
		"status":          incidentEntity.Status,
		"created_at":      incidentEntity.CreatedAt,
		"analysis_time":   time.Now(),
	}

	// 计算影响指标
	var totalMetrics int
	var avgValue float64
	var maxValue float64
	var minValue float64

	if len(metrics) > 0 {
		totalMetrics = len(metrics)
		minValue = metrics[0].MetricValue
		maxValue = metrics[0].MetricValue

		var totalValue float64
		for _, metric := range metrics {
			totalValue += metric.MetricValue
			if metric.MetricValue > maxValue {
				maxValue = metric.MetricValue
			}
			if metric.MetricValue < minValue {
				minValue = metric.MetricValue
			}
		}
		avgValue = totalValue / float64(len(metrics))
	}

	impactAnalysis["metrics"] = map[string]interface{}{
		"total_count":   totalMetrics,
		"average_value": avgValue,
		"max_value":     maxValue,
		"min_value":     minValue,
	}

	// 计算时间影响
	timeImpact := s.calculateTimeImpact(incidentEntity)
	impactAnalysis["time_impact"] = timeImpact

	// 计算业务影响
	businessImpact := s.calculateBusinessImpact(incidentEntity, metrics)
	impactAnalysis["business_impact"] = businessImpact

	// 更新事件的影响分析
	_, err = s.client.Incident.UpdateOneID(incidentID).
		SetImpactAnalysis(impactAnalysis).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update incident impact analysis", "error", err)
	}

	return impactAnalysis, nil
}

// calculateTimeImpact 计算时间影响
func (s *IncidentMonitoringService) calculateTimeImpact(incident *ent.Incident) map[string]interface{} {
	now := time.Now()
	createdAt := incident.CreatedAt

	timeImpact := map[string]interface{}{
		"hours_since_creation": now.Sub(createdAt).Hours(),
		"days_since_creation":  now.Sub(createdAt).Hours() / 24,
		"is_overdue":           false,
		"overdue_hours":        0,
	}

	// 检查是否超时
	if incident.Severity == "critical" && now.Sub(createdAt) > 4*time.Hour {
		timeImpact["is_overdue"] = true
		timeImpact["overdue_hours"] = now.Sub(createdAt.Add(4 * time.Hour)).Hours()
	} else if incident.Severity == "high" && now.Sub(createdAt) > 8*time.Hour {
		timeImpact["is_overdue"] = true
		timeImpact["overdue_hours"] = now.Sub(createdAt.Add(8 * time.Hour)).Hours()
	} else if incident.Severity == "medium" && now.Sub(createdAt) > 24*time.Hour {
		timeImpact["is_overdue"] = true
		timeImpact["overdue_hours"] = now.Sub(createdAt.Add(24 * time.Hour)).Hours()
	}

	return timeImpact
}

// calculateBusinessImpact 计算业务影响
func (s *IncidentMonitoringService) calculateBusinessImpact(incident *ent.Incident, metrics []*ent.IncidentMetric) map[string]interface{} {
	businessImpact := map[string]interface{}{
		"affected_users":       0,
		"revenue_impact":       0,
		"service_availability": 100.0,
		"performance_impact":   0,
	}

	// 根据事件严重程度估算影响
	switch incident.Severity {
	case "critical":
		businessImpact["affected_users"] = 1000
		businessImpact["revenue_impact"] = 10000
		businessImpact["service_availability"] = 50.0
	case "high":
		businessImpact["affected_users"] = 500
		businessImpact["revenue_impact"] = 5000
		businessImpact["service_availability"] = 75.0
	case "medium":
		businessImpact["affected_users"] = 100
		businessImpact["revenue_impact"] = 1000
		businessImpact["service_availability"] = 90.0
	case "low":
		businessImpact["affected_users"] = 10
		businessImpact["revenue_impact"] = 100
		businessImpact["service_availability"] = 95.0
	}

	// 根据指标调整影响
	for _, metric := range metrics {
		if metric.MetricType == "cpu_usage_percent" && metric.MetricValue > 90 {
			businessImpact["performance_impact"] = 50
		} else if metric.MetricType == "memory_usage_percent" && metric.MetricValue > 95 {
			businessImpact["performance_impact"] = 30
		}
	}

	return businessImpact
}

// GenerateIncidentReport 生成事件报告
func (s *IncidentMonitoringService) GenerateIncidentReport(ctx context.Context, req *dto.IncidentMonitoringRequest, tenantID int) (*dto.IncidentMonitoringResponse, error) {
	s.logger.Infow("Generating incident report", "tenant_id", tenantID)

	// 解析时间范围
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time format: %w", err)
	}

	// 构建查询
	query := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.CreatedAtGTE(startTime),
			incident.CreatedAtLTE(endTime),
		)

	// 应用过滤器
	if req.IncidentID != nil {
		query = query.Where(incident.IDEQ(*req.IncidentID))
	}
	if req.Category != nil {
		query = query.Where(incident.CategoryEQ(*req.Category))
	}
	if req.Priority != nil {
		query = query.Where(incident.PriorityEQ(*req.Priority))
	}
	if req.Status != nil {
		query = query.Where(incident.StatusEQ(*req.Status))
	}

	// 获取事件列表
	incidents, err := query.All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get incidents", "error", err)
		return nil, fmt.Errorf("failed to get incidents: %w", err)
	}

	// 计算统计数据
	stats := s.calculateIncidentStats(incidents)

	// 获取指标数据
	metrics, err := s.getIncidentMetrics(ctx, incidents, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to get incident metrics", "error", err)
	}

	// 获取告警数据
	alerts, err := s.getIncidentAlerts(ctx, incidents, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to get incident alerts", "error", err)
	}

	// 构建响应
	response := &dto.IncidentMonitoringResponse{
		TotalIncidents:        stats.TotalIncidents,
		OpenIncidents:         stats.OpenIncidents,
		ResolvedIncidents:     stats.ResolvedIncidents,
		ClosedIncidents:       stats.ClosedIncidents,
		CriticalIncidents:     stats.CriticalIncidents,
		HighPriorityIncidents: stats.HighPriorityIncidents,
		AverageResolutionTime: stats.AverageResolutionTime,
		ResolutionRate:        stats.ResolutionRate,
		EscalationRate:        stats.EscalationRate,
		Incidents:             s.convertIncidentsToResponse(incidents),
		Metrics:               metrics,
		Alerts:                alerts,
	}

	return response, nil
}

// IncidentStats 事件统计信息
type IncidentStats struct {
	TotalIncidents        int
	OpenIncidents         int
	ResolvedIncidents     int
	ClosedIncidents       int
	CriticalIncidents     int
	HighPriorityIncidents int
	AverageResolutionTime float64
	ResolutionRate        float64
	EscalationRate        float64
}

// calculateIncidentStats 计算事件统计信息
func (s *IncidentMonitoringService) calculateIncidentStats(incidents []*ent.Incident) IncidentStats {
	stats := IncidentStats{
		TotalIncidents: len(incidents),
	}

	var totalResolutionTime float64
	var resolvedCount int
	var escalatedCount int

	for _, incident := range incidents {
		switch incident.Status {
		case "new", "in_progress":
			stats.OpenIncidents++
		case "resolved":
			stats.ResolvedIncidents++
			if !incident.ResolvedAt.IsZero() {
				resolutionTime := incident.ResolvedAt.Sub(incident.CreatedAt).Hours()
				totalResolutionTime += resolutionTime
				resolvedCount++
			}
		case "closed":
			stats.ClosedIncidents++
		}

		if incident.Severity == "critical" {
			stats.CriticalIncidents++
		}
		if incident.Priority == "high" || incident.Priority == "urgent" {
			stats.HighPriorityIncidents++
		}
		if incident.EscalationLevel > 0 {
			escalatedCount++
		}
	}

	// 计算平均解决时间
	if resolvedCount > 0 {
		stats.AverageResolutionTime = totalResolutionTime / float64(resolvedCount)
	}

	// 计算解决率
	if stats.TotalIncidents > 0 {
		stats.ResolutionRate = float64(stats.ResolvedIncidents+stats.ClosedIncidents) / float64(stats.TotalIncidents) * 100
		stats.EscalationRate = float64(escalatedCount) / float64(stats.TotalIncidents) * 100
	}

	return stats
}

// getIncidentMetrics 获取事件指标
func (s *IncidentMonitoringService) getIncidentMetrics(ctx context.Context, incidents []*ent.Incident, tenantID int) ([]dto.IncidentMetricResponse, error) {
	var incidentIDs []int
	for _, incident := range incidents {
		incidentIDs = append(incidentIDs, incident.ID)
	}

	if len(incidentIDs) == 0 {
		return []dto.IncidentMetricResponse{}, nil
	}

	metrics, err := s.client.IncidentMetric.Query().
		Where(
			incidentmetric.IncidentIDIn(incidentIDs...),
			incidentmetric.TenantIDEQ(tenantID),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.IncidentMetricResponse, len(metrics))
	for i, metric := range metrics {
		responses[i] = dto.IncidentMetricResponse{
			ID:          metric.ID,
			IncidentID:  metric.IncidentID,
			MetricType:  metric.MetricType,
			MetricName:  metric.MetricName,
			MetricValue: metric.MetricValue,
			Unit:        metric.Unit,
			MeasuredAt:  metric.MeasuredAt,
			Tags:        metric.Tags,
			Metadata:    metric.Metadata,
			TenantID:    metric.TenantID,
			CreatedAt:   metric.CreatedAt,
			UpdatedAt:   metric.UpdatedAt,
		}
	}

	return responses, nil
}

// getIncidentAlerts 获取事件告警
func (s *IncidentMonitoringService) getIncidentAlerts(ctx context.Context, incidents []*ent.Incident, tenantID int) ([]dto.IncidentAlertResponse, error) {
	var incidentIDs []int
	for _, incident := range incidents {
		incidentIDs = append(incidentIDs, incident.ID)
	}

	if len(incidentIDs) == 0 {
		return []dto.IncidentAlertResponse{}, nil
	}

	alerts, err := s.client.IncidentAlert.Query().
		Where(
			incidentalert.IncidentIDIn(incidentIDs...),
			incidentalert.TenantIDEQ(tenantID),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.IncidentAlertResponse, len(alerts))
	for i, alert := range alerts {
		responses[i] = dto.IncidentAlertResponse{
			ID:             alert.ID,
			IncidentID:     alert.IncidentID,
			AlertType:      alert.AlertType,
			AlertName:      alert.AlertName,
			Message:        alert.Message,
			Severity:       alert.Severity,
			Status:         alert.Status,
			Channels:       alert.Channels,
			Recipients:     alert.Recipients,
			TriggeredAt:    alert.TriggeredAt,
			AcknowledgedAt: &alert.AcknowledgedAt,
			ResolvedAt:     &alert.ResolvedAt,
			AcknowledgedBy: &alert.AcknowledgedBy,
			Metadata:       alert.Metadata,
			TenantID:       alert.TenantID,
			CreatedAt:      alert.CreatedAt,
			UpdatedAt:      alert.UpdatedAt,
		}
	}

	return responses, nil
}

// convertIncidentsToResponse 转换事件为响应格式
func (s *IncidentMonitoringService) convertIncidentsToResponse(incidents []*ent.Incident) []dto.IncidentResponse {
	responses := make([]dto.IncidentResponse, len(incidents))
	for i, incident := range incidents {
		responses[i] = dto.IncidentResponse{
			ID:                  incident.ID,
			Title:               incident.Title,
			Description:         incident.Description,
			Status:              incident.Status,
			Priority:            incident.Priority,
			Severity:            incident.Severity,
			IncidentNumber:      incident.IncidentNumber,
			ReporterID:          incident.ReporterID,
			AssigneeID:          &incident.AssigneeID,
			ConfigurationItemID: &incident.ConfigurationItemID,
			Category:            incident.Category,
			Subcategory:         incident.Subcategory,
			ImpactAnalysis:      incident.ImpactAnalysis,
			RootCause:           incident.RootCause,
			ResolutionSteps:     incident.ResolutionSteps,
			DetectedAt:          incident.DetectedAt,
			ResolvedAt:          &incident.ResolvedAt,
			ClosedAt:            &incident.ClosedAt,
			EscalatedAt:         &incident.EscalatedAt,
			EscalationLevel:     incident.EscalationLevel,
			IsAutomated:         incident.IsAutomated,
			Source:              incident.Source,
			Metadata:            incident.Metadata,
			TenantID:            incident.TenantID,
			CreatedAt:           incident.CreatedAt,
			UpdatedAt:           incident.UpdatedAt,
		}
	}
	return responses
}

// parseFloatValue 解析浮点数值
func parseFloatValue(value string) float64 {
	var result float64
	fmt.Sscanf(value, "%f", &result)
	return result
}
