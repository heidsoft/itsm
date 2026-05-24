package dashboard

import (
	"time"
)

// Service handles SLA dashboard business logic
type Service struct {
	// db *sql.DB // Reserved for future real data queries
}

// NewService creates a new dashboard service
func NewService( /*db *sql.DB*/ ) *Service {
	// return &Service{db: db}
	return &Service{}
}

// SLAData represents the complete SLA dashboard data
type SLAData struct {
	TotalTickets      int64   `json:"total_tickets"`
	OpenTickets       int64   `json:"open_tickets"`
	InProgressTickets int64   `json:"in_progress_tickets"`
	ResolvedTickets   int64   `json:"resolved_tickets"`
	AvgResponseTime   float64 `json:"avg_response_time_hours"`
	AvgResolutionTime float64 `json:"avg_resolution_time_hours"`
	SLAComplianceRate float64 `json:"sla_compliance_rate"`
	BreachedTickets   int64   `json:"breached_tickets"`
	LastUpdated       string  `json:"last_updated"`
}

// ChartData represents chart specific data
type ChartData struct {
	TrendData            []TrendPoint        `json:"trend_data"`
	IncidentDistribution Distribution        `json:"incident_distribution"`
	ResponseTimeBuckets  []Bucket            `json:"response_time_buckets"`
	TeamWorkload         []TeamMember        `json:"team_workload"`
	SLATargets           []SLATarget         `json:"sla_targets"`
	PeakHours            []PeakHour          `json:"peak_hours"`
	SatisfactionTrend    []SatisfactionPoint `json:"satisfaction_trend"`
}

// TrendPoint for ticket trend chart
type TrendPoint struct {
	Date       string `json:"date"`
	Open       int64  `json:"open"`
	InProgress int64  `json:"in_progress"`
	Resolved   int64  `json:"resolved"`
	Closed     int64  `json:"closed"`
}

// Distribution for pie/bar charts
type Distribution struct {
	Categories []Category `json:"categories"`
}

// Category for distribution data
type Category struct {
	Name    string  `json:"name"`
	Value   int64   `json:"value"`
	Percent float64 `json:"percentage"`
}

// Bucket for histogram data
type Bucket struct {
	Range       string  `json:"range"` // e.g., "0-1h", "1-2h"
	TicketCount int64   `json:"ticket_count"`
	Percentage  float64 `json:"percentage"`
}

// TeamMember for workload chart
type TeamMember struct {
	Name            string  `json:"name"`
	AssignedTickets int64   `json:"assigned_tickets"`
	CompletionRate  float64 `json:"completion_rate"`
}

// SLATarget for SLA monitoring
type SLATarget struct {
	Name       string  `json:"name"` // e.g., "SLA-P0-紧急"
	CurrentPct float64 `json:"current_percentage"`
	TargetPct  float64 `json:"target_percentage"`
	Status     string  `json:"status"` // "good", "warning", "critical"
}

// PeakHour for hourly distribution
type PeakHour struct {
	Hour        int   `json:"hour"` // 0-23
	TicketCount int64 `json:"ticket_count"`
}

// SatisfactionPoint for satisfaction trend
type SatisfactionPoint struct {
	Month         string  `json:"month"` // e.g., "2025-01"
	AverageRating float64 `json:"average_rating"`
	FeedbackCount int64   `json:"feedback_count"`
	MaxRating     int     `json:"max_rating"`
	MinRating     int     `json:"min_rating"`
}

// DashboardResponse is the complete API response
type DashboardResponse struct {
	Data SLAData `json:"data"`
}

// FullDashboardResponse includes charts
type FullDashboardResponse struct {
	Data   SLAData   `json:"data"`
	Charts ChartData `json:"charts"`
}

// GetSLADashboard returns all SLA dashboard data
func (s *Service) GetSLADashboard() (*FullDashboardResponse, error) {
	// This is a simplified implementation with realistic mock data
	// In production, this would query the database and compute actual metrics

	now := time.Now()

	response := &FullDashboardResponse{
		Data: SLAData{
			TotalTickets:      152,
			OpenTickets:       12,
			InProgressTickets: 8,
			ResolvedTickets:   132,
			AvgResponseTime:   2.5,
			AvgResolutionTime: 4.8,
			SLAComplianceRate: 92.5,
			BreachedTickets:   3,
			LastUpdated:       now.Format("15:04:05"),
		},
		Charts: ChartData{
			TrendData: []TrendPoint{
				{Date: "2025-10-25", Open: 15, InProgress: 10, Resolved: 25, Closed: 30},
				{Date: "2025-10-26", Open: 12, InProgress: 8, Resolved: 28, Closed: 32},
				{Date: "2025-10-27", Open: 10, InProgress: 6, Resolved: 22, Closed: 28},
				{Date: "2025-10-28", Open: 8, InProgress: 5, Resolved: 20, Closed: 25},
				{Date: "2025-10-29", Open: 6, InProgress: 4, Resolved: 18, Closed: 22},
				{Date: "2025-10-30", Open: 5, InProgress: 3, Resolved: 15, Closed: 20},
				{Date: "2025-10-31", Open: 4, InProgress: 2, Resolved: 12, Closed: 18},
			},
			IncidentDistribution: Distribution{
				Categories: []Category{
					{Name: "incident", Value: 85, Percent: 55.9},
					{Name: "problem", Value: 35, Percent: 23.0},
					{Name: "change", Value: 15, Percent: 9.9},
					{Name: "service_request", Value: 17, Percent: 11.2},
				},
			},
			ResponseTimeBuckets: []Bucket{
				{Range: "0-1h", TicketCount: 45, Percentage: 29.6},
				{Range: "1-2h", TicketCount: 38, Percentage: 25.0},
				{Range: "2-4h", TicketCount: 35, Percentage: 23.0},
				{Range: "4-8h", TicketCount: 20, Percentage: 13.2},
				{Range: "8h+", TicketCount: 14, Percentage: 9.2},
			},
			TeamWorkload: []TeamMember{
				{Name: "张三", AssignedTickets: 12, CompletionRate: 88.5},
				{Name: "李四", AssignedTickets: 10, CompletionRate: 92.0},
				{Name: "王五", AssignedTickets: 15, CompletionRate: 85.2},
				{Name: "赵六", AssignedTickets: 8, CompletionRate: 95.1},
			},
			SLATargets: []SLATarget{
				{Name: "SLA-P0-紧急", CurrentPct: 100.0, TargetPct: 99.0, Status: "good"},
				{Name: "SLA-P1-高", CurrentPct: 100.0, TargetPct: 99.0, Status: "good"},
				{Name: "SLA-P2-中", CurrentPct: 100.0, TargetPct: 99.0, Status: "good"},
				{Name: "SLA-P3-低", CurrentPct: 100.0, TargetPct: 99.0, Status: "good"},
				{Name: "SLA-服务请求", CurrentPct: 100.0, TargetPct: 99.0, Status: "good"},
				{Name: "SLA-变更", CurrentPct: 100.0, TargetPct: 99.0, Status: "good"},
			},
			PeakHours: []PeakHour{
				{Hour: 9, TicketCount: 18},
				{Hour: 10, TicketCount: 22},
				{Hour: 11, TicketCount: 25},
				{Hour: 14, TicketCount: 20},
				{Hour: 15, TicketCount: 23},
				{Hour: 16, TicketCount: 19},
			},
			SatisfactionTrend: []SatisfactionPoint{
				{Month: "2025-05", AverageRating: 4.2, FeedbackCount: 28, MaxRating: 5, MinRating: 2},
				{Month: "2025-06", AverageRating: 4.3, FeedbackCount: 32, MaxRating: 5, MinRating: 3},
				{Month: "2025-07", AverageRating: 4.4, FeedbackCount: 35, MaxRating: 5, MinRating: 3},
				{Month: "2025-08", AverageRating: 4.5, FeedbackCount: 40, MaxRating: 5, MinRating: 4},
				{Month: "2025-09", AverageRating: 4.6, FeedbackCount: 45, MaxRating: 5, MinRating: 4},
				{Month: "2025-10", AverageRating: 4.7, FeedbackCount: 50, MaxRating: 5, MinRating: 4},
			},
		},
	}

	return response, nil
}

// GetBasicDashboard returns simplified data (without charts)
func (s *Service) GetBasicDashboard() (*DashboardResponse, error) {
	full, err := s.GetSLADashboard()
	if err != nil {
		return nil, err
	}
	return &DashboardResponse{Data: full.Data}, nil
}
