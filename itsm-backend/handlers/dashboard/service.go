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
	TotalTickets      int64   `json:"totalTickets"`
	OpenTickets       int64   `json:"openTickets"`
	InProgressTickets int64   `json:"inProgressTickets"`
	ResolvedTickets   int64   `json:"resolvedTickets"`
	AvgResponseTime   float64 `json:"avgResponseTimeHours"`
	AvgResolutionTime float64 `json:"avgResolutionTimeHours"`
	SLAComplianceRate float64 `json:"slaComplianceRate"`
	BreachedTickets   int64   `json:"breachedTickets"`
	LastUpdated       string  `json:"lastUpdated"`
}

// ChartData represents chart specific data
type ChartData struct {
	TrendData            []TrendPoint        `json:"trendData"`
	IncidentDistribution Distribution        `json:"incidentDistribution"`
	ResponseTimeBuckets  []Bucket            `json:"responseTimeBuckets"`
	TeamWorkload         []TeamMember        `json:"teamWorkload"`
	SLATargets           []SLATarget         `json:"slaTargets"`
	PeakHours            []PeakHour          `json:"peakHours"`
	SatisfactionTrend    []SatisfactionPoint `json:"satisfactionTrend"`
}

// TrendPoint for ticket trend chart
type TrendPoint struct {
	Date       string `json:"date"`
	Open       int64  `json:"open"`
	InProgress int64  `json:"inProgress"`
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
	TicketCount int64   `json:"ticketCount"`
	Percentage  float64 `json:"percentage"`
}

// TeamMember for workload chart
type TeamMember struct {
	Name            string  `json:"name"`
	AssignedTickets int64   `json:"assignedTickets"`
	CompletionRate  float64 `json:"completionRate"`
}

// SLATarget for SLA monitoring
type SLATarget struct {
	Name       string  `json:"name"` // e.g., "SLA-P0-紧急"
	CurrentPct float64 `json:"currentPercentage"`
	TargetPct  float64 `json:"targetPercentage"`
	Status     string  `json:"status"` // "good", "warning", "critical"
}

// PeakHour for hourly distribution
type PeakHour struct {
	Hour        int   `json:"hour"` // 0-23
	TicketCount int64 `json:"ticketCount"`
}

// SatisfactionPoint for satisfaction trend
type SatisfactionPoint struct {
	Month         string  `json:"month"` // e.g., "2025-01"
	AverageRating float64 `json:"averageRating"`
	FeedbackCount int64   `json:"feedbackCount"`
	MaxRating     int     `json:"maxRating"`
	MinRating     int     `json:"minRating"`
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
