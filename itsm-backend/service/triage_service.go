package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// TriageResult holds classification suggestions for incidents/tickets
type TriageResult struct {
	Category     string  `json:"category"`
	Priority     string  `json:"priority"`
	AssigneeID   int     `json:"assignee_id"`
	Confidence   float64 `json:"confidence"`
	Explanation  string  `json:"explanation"`
	SuggestedFix string  `json:"suggested_fix,omitempty"`
}

// TriageService provides LLM-powered ticket triage and classification
type TriageService struct {
	gateway          *LLMGateway
	logger           *zap.Logger
	keywordFallback  bool
	defaultAssignees map[string]int
}

// Category priority mapping
var defaultPriorities = map[string]string{
	"database":      "high",
	"network":       "high",
	"server":        "medium",
	"application":   "medium",
	"security":      "critical",
	"storage":       "high",
	"user_access":   "medium",
	"general":       "low",
}

// Default assignee IDs by category
var defaultAssignees = map[string]int{
	"database":      101,
	"network":       102,
	"server":        103,
	"application":   104,
	"security":      100,
	"storage":       105,
	"user_access":   106,
	"general":       0,
}

// NewTriageService creates a new LLM-powered triage service
func NewTriageService(gateway *LLMGateway, logger *zap.Logger) *TriageService {
	return &TriageService{
		gateway:          gateway,
		logger:           logger,
		keywordFallback:  true,
		defaultAssignees: defaultAssignees,
	}
}

// NewTriageServiceWithConfig creates a service with custom configuration
func NewTriageServiceWithConfig(gateway *LLMGateway, logger *zap.Logger, fallback bool, assignees map[string]int) *TriageService {
	svc := &TriageService{
		gateway:          gateway,
		logger:           logger,
		keywordFallback:  fallback,
		defaultAssignees: defaultAssignees,
	}
	if assignees != nil {
		svc.defaultAssignees = assignees
	}
	return svc
}

// Suggest returns classification suggestions using LLM
func (t *TriageService) Suggest(ctx context.Context, title, description string) TriageResult {
	text := strings.ToLower(title + " " + description)

	result := TriageResult{
		Category:    "general",
		Priority:    "medium",
		AssigneeID:  0,
		Confidence:  0.5,
		Explanation: "keyword heuristic",
	}

	if t.gateway == nil {
		if t.keywordFallback {
			return t.keywordBasedSuggest(text)
		}
		return result
	}

	llmResult, err := t.llmClassify(ctx, title, description)
	if err != nil {
		t.logger.Warn("TriageService: LLM classification failed, using fallback", zap.Error(err))
		if t.keywordFallback {
			return t.keywordBasedSuggest(text)
		}
		return result
	}

	return llmResult
}

// llmClassify uses LLM for intelligent ticket classification
func (t *TriageService) llmClassify(ctx context.Context, title, description string) (TriageResult, error) {
	prompt := fmt.Sprintf(`Analyze the following IT ticket and provide classification:

Title: %s

Description:
%s

Please return JSON with:
{
  "category": "database|network|server|application|security|storage|user_access|general",
  "priority": "critical|high|medium|low",
  "confidence": 0.0-1.0,
  "explanation": "reason for classification",
  "suggested_fix": "brief suggested solution"
}

Only return JSON, no other content.`, title, description)

	messages := []LLMMessage{
		{Role: "system", Content: "You are an IT service management triage assistant."},
		{Role: "user", Content: prompt},
	}

	resp, err := t.gateway.Chat(ctx, "gpt-4o-mini", messages)
	if err != nil {
		return TriageResult{}, fmt.Errorf("LLM classification failed: %w", err)
	}

	resp = strings.TrimSpace(resp)
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	var classification TriageResult
	if err := json.Unmarshal([]byte(resp), &classification); err != nil {
		start := strings.Index(resp, "{")
		end := strings.LastIndex(resp, "}")
		if start >= 0 && end >= start {
			if err := json.Unmarshal([]byte(resp[start:end+1]), &classification); err != nil {
				return TriageResult{}, fmt.Errorf("failed to parse LLM response: %w", err)
			}
		} else {
			return TriageResult{}, fmt.Errorf("no JSON found in LLM response")
		}
	}

	if classification.Category == "" {
		classification.Category = "general"
	}
	if classification.Priority == "" {
		classification.Priority = "medium"
	}
	if classification.Confidence == 0 {
		classification.Confidence = 0.6
	}

	if assigneeID, ok := t.defaultAssignees[classification.Category]; ok {
		classification.AssigneeID = assigneeID
	}

	return classification, nil
}

// keywordBasedSuggest provides fallback classification using keywords
func (t *TriageService) keywordBasedSuggest(text string) TriageResult {
	result := TriageResult{
		Category:    "general",
		Priority:    "medium",
		AssigneeID:  0,
		Confidence:  0.55,
		Explanation: "keyword heuristic",
	}

	// Category detection based on keywords
	if containsAny(text, "database", "db", "mysql", "postgresql", "oracle", "sql", "mongodb", "redis") {
		result.Category = "database"
		result.AssigneeID = t.defaultAssignees["database"]
		result.Confidence = 0.7
	} else if containsAny(text, "network", "wifi", "switch", "router", "firewall") {
		result.Category = "network"
		result.AssigneeID = t.defaultAssignees["network"]
		result.Confidence = 0.68
	} else if containsAny(text, "server", "cpu", "memory", "disk", "linux", "windows server") {
		result.Category = "server"
		result.AssigneeID = t.defaultAssignees["server"]
		result.Confidence = 0.65
	} else if containsAny(text, "application", "software", "app", "api", "deploy") {
		result.Category = "application"
		result.AssigneeID = t.defaultAssignees["application"]
		result.Confidence = 0.62
	} else if containsAny(text, "security", "vulnerability", "attack", "permission", "authentication") {
		result.Category = "security"
		result.AssigneeID = t.defaultAssignees["security"]
		result.Confidence = 0.75
		result.Priority = "critical"
	} else if containsAny(text, "storage", "disk", "space", "backup", "snapshot") {
		result.Category = "storage"
		result.AssigneeID = t.defaultAssignees["storage"]
		result.Confidence = 0.66
	} else if containsAny(text, "user", "account", "login", "password", "access") {
		result.Category = "user_access"
		result.AssigneeID = t.defaultAssignees["user_access"]
		result.Confidence = 0.6
	}

	// Priority escalation
	if containsAny(text, "down", "unavailable", "urgent", "critical", "outage") {
		if result.Priority != "critical" {
			result.Priority = "high"
		}
		result.Confidence += 0.1
	}

	if containsAny(text, "impact", "affect", "users", "business") {
		if result.Priority == "medium" {
			result.Priority = "high"
		}
	}

	return result
}

// containsAny checks if text contains any of the given substrings
func containsAny(text string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(text, sub) {
			return true
		}
	}
	return false
}

// BatchSuggest processes multiple tickets for triage suggestions
func (t *TriageService) BatchSuggest(ctx context.Context, tickets []struct {
	ID          int
	Title       string
	Description string
}) ([]struct {
	ID     int
	Result TriageResult
}, error) {
	results := make([]struct {
		ID     int
		Result TriageResult
	}, len(tickets))

	for i, ticket := range tickets {
		results[i] = struct {
			ID     int
			Result TriageResult
		}{
			ID:     ticket.ID,
			Result: t.Suggest(ctx, ticket.Title, ticket.Description),
		}
	}

	return results, nil
}

// TriageWithMetadata includes timing and method information
type TriageWithMetadata struct {
	Result  TriageResult   `json:"result"`
	Method  string         `json:"method"`
	Latency time.Duration  `json:"latency_ms"`
	Model   string         `json:"model,omitempty"`
}

// SuggestWithMetadata returns triage result with additional metadata
func (t *TriageService) SuggestWithMetadata(ctx context.Context, title, description string) (*TriageWithMetadata, error) {
	start := time.Now()

	text := strings.ToLower(title + " " + description)

	result := &TriageWithMetadata{
		Result: TriageResult{
			Category:    "general",
			Priority:    "medium",
			AssigneeID:  0,
			Confidence:  0.5,
			Explanation: "keyword heuristic",
		},
		Method:  "fallback",
		Latency: time.Since(start),
	}

	if t.gateway == nil {
		if t.keywordFallback {
			result.Result = t.keywordBasedSuggest(text)
			result.Method = "fallback"
		}
		return result, nil
	}

	llmResult, err := t.llmClassify(ctx, title, description)
	if err != nil {
		t.logger.Warn("TriageService: LLM classification failed", zap.Error(err))
		if t.keywordFallback {
			result.Result = t.keywordBasedSuggest(text)
			result.Method = "fallback"
		}
		return result, nil
	}

	result.Result = llmResult
	result.Method = "llm"
	result.Model = "gpt-4o-mini"
	result.Latency = time.Since(start)

	return result, nil
}

// SuggestAssignee suggests an assignee based on ticket content
func (t *TriageService) SuggestAssignee(ctx context.Context, category, title, description string) (int, string, error) {
	if t.gateway != nil {
		prompt := fmt.Sprintf(`Recommend the best handler for this IT ticket:

Category: %s
Title: %s
Description: %s

Choose from:
- Zhang San (101) - Database expert
- Li Si (102) - Network engineer
- Wang Wu (103) - Server admin
- Zhao Liu (104) - Application developer
- Zhou Qi (100) - Security expert
- Chen Ba (105) - Storage expert

Return only the ID number, or 0 if unsure.`, category, title, description)

		messages := []LLMMessage{
			{Role: "system", Content: "You are an IT service management assistant."},
			{Role: "user", Content: prompt},
		}

		resp, err := t.gateway.Chat(ctx, "gpt-4o-mini", messages)
		if err != nil {
			t.logger.Warn("TriageService: assignee suggestion failed", zap.Error(err))
		} else {
			resp = strings.TrimSpace(resp)
			for _, line := range strings.Split(resp, "\n") {
				line = strings.TrimSpace(line)
				if len(line) <= 3 {
					continue
				}
				for i := len(line) - 1; i >= 0; i-- {
					if line[i] < '0' || line[i] > '9' {
						if i < len(line)-1 {
							if id, err := parseAssigneeID(line[i+1:]); err == nil {
								return id, "", nil
							}
						}
						break
					}
				}
			}
		}
	}

	if assigneeID, ok := t.defaultAssignees[category]; ok {
		return assigneeID, "", nil
	}

	return 0, "", nil
}

func parseAssigneeID(s string) (int, error) {
	var id int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			id = id*10 + int(c-'0')
		} else {
			break
		}
	}
	if id == 0 {
		return 0, fmt.Errorf("no valid ID found")
	}
	return id, nil
}
