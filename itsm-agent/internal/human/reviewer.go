package human

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"itsm-agent/internal/config"
	"itsm-agent/internal/logger"
	"itsm-agent/internal/watcher"
)

// Reviewer handles human review workflow
type Reviewer struct {
	cfg       *config.HumanConfig
	owner     string
	repo      string
	ghCLI     string
	pendingPR map[int]time.Time // prNumber -> startTime
}

// New creates a new human reviewer
func New(cfg *config.HumanConfig, owner, repo, ghCLI string) *Reviewer {
	return &Reviewer{
		cfg:       cfg,
		owner:     owner,
		repo:      repo,
		ghCLI:     ghCLI,
		pendingPR: make(map[int]time.Time),
	}
}

// NotifyReview sends notification about new PR
func (r *Reviewer) NotifyReview(issue *watcher.Issue, prNumber int, prURL string) error {
	if !r.cfg.Enabled {
		logger.Debug("Human review not enabled, skipping notification")
		return nil
	}

	logger.Info("Sending review notification for PR #%d", prNumber)

	// Track pending PR
	r.pendingPR[prNumber] = time.Now()

	// Send Slack notification if configured
	if r.cfg.SlackWebhook != "" {
		if err := r.sendSlackNotification(issue, prNumber, prURL); err != nil {
			logger.Warn("Failed to send Slack notification: %v", err)
		}
	}

	// Send Discord notification if configured
	if r.cfg.DiscordWebhook != "" {
		if err := r.sendDiscordNotification(issue, prNumber, prURL); err != nil {
			logger.Warn("Failed to send Discord notification: %v", err)
		}
	}

	// Also add a comment to the PR
	if err := r.commentPR(prNumber, r.buildReviewComment(issue)); err != nil {
		logger.Warn("Failed to add PR comment: %v", err)
	}

	return nil
}

// CheckApproval checks if PR has been approved
func (r *Reviewer) CheckApproval(prNumber int) (approved bool, err error) {
	cmd := exec.Command(r.ghCLI, "pr", "view",
		"--repo", fmt.Sprintf("%s/%s", r.owner, r.repo),
		fmt.Sprintf("%d", prNumber),
		"--json", "state,mergeStateStatus,reviews")

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get PR info: %w", err)
	}

	// Parse JSON response
	var prInfo struct {
		State            string `json:"state"`
		MergeStateStatus string `json:"mergeStateStatus"`
		Reviews          []struct {
			State string `json:"state"`
		} `json:"reviews"`
	}

	if err := json.Unmarshal(output, &prInfo); err != nil {
		return false, fmt.Errorf("failed to parse PR info: %w", err)
	}

	// Check if merged
	if prInfo.MergeStateStatus == "merged" || prInfo.State == "merged" {
		return true, nil
	}

	// Check for approved reviews
	for _, review := range prInfo.Reviews {
		if review.State == "APPROVED" {
			return true, nil
		}
	}

	// Check for auto-merge
	if prInfo.MergeStateStatus == "clean" {
		return true, nil
	}

	return false, nil
}

// WaitForApproval waits for human approval with timeout
func (r *Reviewer) WaitForApproval(prNumber int) (approved bool, err error) {
	if !r.cfg.Enabled {
		logger.Info("Human review not enabled, auto-approving")
		return true, nil
	}

	timeout := time.Duration(r.cfg.ReviewTimeout) * time.Hour
	startTime := time.Now()
	pollInterval := 5 * time.Minute

	logger.Info("Waiting for approval of PR #%d (timeout: %v)", prNumber, timeout)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			approved, err := r.CheckApproval(prNumber)
			if err != nil {
				logger.Warn("Error checking approval: %v", err)
				continue
			}

			if approved {
				logger.Info("PR #%d approved!", prNumber)
				return true, nil
			}

			// Check timeout
			if time.Since(startTime) > timeout {
				logger.Warn("Approval timeout for PR #%d", prNumber)
				return false, fmt.Errorf("approval timeout after %v", timeout)
			}

			logger.Debug("Still waiting for approval of PR #%d", prNumber)
		}
	}
}

// sendSlackNotification sends notification to Slack
func (r *Reviewer) sendSlackNotification(issue *watcher.Issue, prNumber int, prURL string) error {
	payload := map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]interface{}{
					"type": "plain_text",
					"text": "🤖 New AI-Generated PR Ready for Review",
				},
			},
			{
				"type": "section",
				"fields": []map[string]interface{}{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Issue:*\n#%d", issue.Number),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*PR:*\n#%d", prNumber),
					},
				},
			},
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*%s*\n%s", issue.Title, truncate(issue.Body, 200)),
				},
			},
			{
				"type": "actions",
				"elements": []map[string]interface{}{
					{
						"type": "button",
						"text": map[string]interface{}{
							"type":  "plain_text",
							"text":  "View PR",
							"emoji": true,
						},
						"url": prURL,
					},
				},
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", r.cfg.SlackWebhook, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// sendDiscordNotification sends notification to Discord
func (r *Reviewer) sendDiscordNotification(issue *watcher.Issue, prNumber int, prURL string) error {
	embeds := []map[string]interface{}{
		{
			"title":       "🤖 New AI-Generated PR Ready for Review",
			"description": fmt.Sprintf("**Issue #%d:** %s\n\n%s", issue.Number, issue.Title, truncate(issue.Body, 300)),
			"url":         prURL,
			"color":       3066993, // Green
			"fields": []map[string]interface{}{
				{
					"name":   "PR Number",
					"value":  fmt.Sprintf("#%d", prNumber),
					"inline": true,
				},
				{
					"name":   "View PR",
					"value":  "[Link](" + prURL + ")",
					"inline": true,
				},
			},
		},
	}

	payload := map[string]interface{}{
		"embeds": embeds,
	}

	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", r.cfg.DiscordWebhook, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// commentPR adds a comment to the PR
func (r *Reviewer) commentPR(prNumber int, body string) error {
	cmd := exec.Command(r.ghCLI, "pr", "comment",
		"--repo", fmt.Sprintf("%s/%s", r.owner, r.repo),
		fmt.Sprintf("%d", prNumber),
		"--body", body)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to comment on PR: %w", err)
	}

	return nil
}

// buildReviewComment builds the review request comment
func (r *Reviewer) buildReviewComment(issue *watcher.Issue) string {
	var comment strings.Builder
	comment.WriteString("## 🔍 AI Developer Agent - Review Request\n\n")
	comment.WriteString(fmt.Sprintf("This PR was automatically generated to address Issue #%d.\n\n", issue.Number))
	comment.WriteString("### Summary\n")
	comment.WriteString(fmt.Sprintf("- **Issue:** %s\n", issue.Title))
	comment.WriteString(fmt.Sprintf("- **Description:** %s\n\n", truncate(issue.Body, 300)))

	comment.WriteString("### Review Checklist\n")
	comment.WriteString("- [ ] Code looks correct\n")
	comment.WriteString("- [ ] Tests are passing\n")
	comment.WriteString("- [ ] No security issues\n")
	comment.WriteString("- [ ] Follows project conventions\n\n")

	comment.WriteString("---\n")
	comment.WriteString("*This is an automated review request from the AI Developer Agent*")

	return comment.String()
}

// truncate truncates a string to maxLength
func truncate(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}
