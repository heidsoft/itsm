package service

import (
	"context"
	"strings"
)

// TriageResult holds classification suggestions for incidents/tickets
type TriageResult struct {
	Category    string  `json:"category"`
	Priority    string  `json:"priority"`
	AssigneeID  int     `json:"assignee_id"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

// TriageService is a heuristic stub; replace with LLM or classifier later
type TriageService struct{}

func NewTriageService() *TriageService { return &TriageService{} }

// Suggest returns naive suggestions based on keywords
func (t *TriageService) Suggest(ctx context.Context, title, description string) TriageResult {
	text := strings.ToLower(title + " " + description)
	result := TriageResult{Category: "general", Priority: "medium", AssigneeID: 0, Confidence: 0.55, Explanation: "keyword heuristic"}
	if strings.Contains(text, "数据库") || strings.Contains(text, "db") {
		result.Category = "database"
		result.AssigneeID = 101
		result.Confidence = 0.7
	}
	if strings.Contains(text, "网络") || strings.Contains(text, "network") {
		result.Category = "network"
		result.AssigneeID = 102
		result.Confidence = 0.68
	}
	if strings.Contains(text, "故障") || strings.Contains(text, "down") || strings.Contains(text, "不可用") {
		result.Priority = "urgent"
		result.Confidence += 0.1
	}
	return result
}
