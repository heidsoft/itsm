package dashboard

import (
	"errors"
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
	return nil, errors.New("legacy SLA dashboard handler is not wired to real data; use /api/v1/dashboard/overview or /api/v1/sla/*")
}

// GetBasicDashboard returns simplified data (without charts)
func (s *Service) GetBasicDashboard() (*DashboardResponse, error) {
	full, err := s.GetSLADashboard()
	if err != nil {
		return nil, err
	}
	return &DashboardResponse{Data: full.Data}, nil
}
