package watcher

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"ai-dev-agent/internal/config"
	"ai-dev-agent/internal/logger"
)

// Issue represents a GitHub issue
type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Labels    []string  `json:"labels"`
	State     string    `json:"state"`
	URL       string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Watcher monitors GitHub issues
type Watcher struct {
	cfg      *config.GitHubConfig
	ghCLI    string
	lastSeen map[int]time.Time
}

// New creates a new watcher
func New(cfg *config.GitHubConfig) *Watcher {
	return &Watcher{
		cfg:      cfg,
		ghCLI:    cfg.GhCLI,
		lastSeen: make(map[int]time.Time),
	}
}

// FetchIssues fetches issues matching the configured labels
func (w *Watcher) FetchIssues() ([]Issue, error) {
	logger.Debug("Fetching issues from %s/%s with labels: %v", w.cfg.Owner, w.cfg.Repo, w.cfg.Labels)

	// Build label filter query
	labelQuery := strings.Join(w.cfg.Labels, ",")

	// Use gh cli to search issues
	cmd := exec.Command(w.ghCLI, "issue", "list",
		"--repo", fmt.Sprintf("%s/%s", w.cfg.Owner, w.cfg.Repo),
		"--state", w.cfg.IssueState,
		"--label", labelQuery,
		"--json", "number,title,body,labels,state,html_url,createdAt,updatedAt",
		"--limit", "30")

	output, err := cmd.Output()
	if err != nil {
		// Try alternative approach using GitHub API directly
		return w.fetchIssuesAPI()
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	logger.Info("Found %d issues matching criteria", len(issues))
	return issues, nil
}

// fetchIssuesAPI fetches issues using GitHub API directly (fallback)
func (w *Watcher) fetchIssuesAPI() ([]Issue, error) {
	logger.Debug("Using GitHub API to fetch issues")

	labelQuery := strings.Join(w.cfg.Labels, ",")
	cmd := exec.Command(w.ghCLI, "api", "repos/"+w.cfg.Owner+"/"+w.cfg.Repo+"/issues",
		"--method", "GET",
		"--field", "labels="+labelQuery,
		"--field", "state="+w.cfg.IssueState,
		"--jq", ".[] | {number, title, body, labels: [.labels[].name], state, html_url, created_at, updated_at}")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issues: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return issues, nil
}

// FilterNewIssues returns only new issues not seen before
func (w *Watcher) FilterNewIssues(issues []Issue) []Issue {
	var newIssues []Issue
	now := time.Now()

	for _, issue := range issues {
		// Check if we've seen this issue before
		if lastTime, seen := w.lastSeen[issue.Number]; seen {
			// Only process if updated since last seen
			if issue.UpdatedAt.After(lastTime) {
				newIssues = append(newIssues, issue)
				w.lastSeen[issue.Number] = now
			}
		} else {
			// New issue we've never seen
			newIssues = append(newIssues, issue)
			w.lastSeen[issue.Number] = now
		}
	}

	if len(newIssues) > 0 {
		logger.Info("Found %d new/updated issues", len(newIssues))
	}

	return newIssues
}

// AddLabel adds a label to an issue
func (w *Watcher) AddLabel(issueNumber int, label string) error {
	logger.Info("Adding label '%s' to issue #%d", label, issueNumber)

	cmd := exec.Command(w.ghCLI, "issue", "add-label",
		"--repo", fmt.Sprintf("%s/%s", w.cfg.Owner, w.cfg.Repo),
		fmt.Sprintf("%d", issueNumber),
		label)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add label: %w", err)
	}

	return nil
}

// RemoveLabel removes a label from an issue
func (w *Watcher) RemoveLabel(issueNumber int, label string) error {
	logger.Info("Removing label '%s' from issue #%d", label, issueNumber)

	cmd := exec.Command(w.ghCLI, "issue", "remove-label",
		"--repo", fmt.Sprintf("%s/%s", w.cfg.Owner, w.cfg.Repo),
		fmt.Sprintf("%d", issueNumber),
		label)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove label: %w", err)
	}

	return nil
}

// Comment adds a comment to an issue
func (w *Watcher) Comment(issueNumber int, body string) error {
	logger.Info("Adding comment to issue #%d", issueNumber)

	cmd := exec.Command(w.ghCLI, "issue", "comment",
		"--repo", fmt.Sprintf("%s/%s", w.cfg.Owner, w.cfg.Repo),
		fmt.Sprintf("%d", issueNumber),
		"--body", body)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	return nil
}

// GetIssue fetches a single issue by number
func (w *Watcher) GetIssue(issueNumber int) (*Issue, error) {
	cmd := exec.Command(w.ghCLI, "issue", "view",
		"--repo", fmt.Sprintf("%s/%s", w.cfg.Owner, w.cfg.Repo),
		fmt.Sprintf("%d", issueNumber),
		"--json", "number,title,body,labels,state,html_url,createdAt,updatedAt")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse issue: %w", err)
	}

	return &issue, nil
}
