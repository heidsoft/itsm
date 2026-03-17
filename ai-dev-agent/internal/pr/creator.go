package pr

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"ai-dev-agent/internal/config"
	"ai-dev-agent/internal/generator"
	"ai-dev-agent/internal/logger"
	"ai-dev-agent/internal/watcher"
)

// Creator creates GitHub pull requests
type Creator struct {
	cfg   *config.PRConfig
	owner string
	repo  string
	ghCLI string
}

// New creates a new PR creator
func New(cfg *config.PRConfig, owner, repo, ghCLI string) *Creator {
	return &Creator{
		cfg:   cfg,
		owner: owner,
		repo:  repo,
		ghCLI: ghCLI,
	}
}

// CreateResult contains the result of PR creation
type CreateResult struct {
	Success    bool
	PRURL      string
	PRNumber   int
	BranchName string
	Error      error
}

// Create creates a pull request for the generated code
func (c *Creator) Create(issue *watcher.Issue, genResult *generator.GenerateResult) (*CreateResult, error) {
	logger.Info("Creating pull request for issue #%d", issue.Number)

	branchName := genResult.BranchName

	// Stage and commit changes
	if err := c.commitChanges(genResult.WorktreePath, issue); err != nil {
		return &CreateResult{Success: false, Error: err}, err
	}

	// Push the branch
	if err := c.pushBranch(genResult.WorktreePath, branchName); err != nil {
		return &CreateResult{Success: false, Error: err}, err
	}

	// Create the PR
	prURL, prNumber, err := c.createPR(issue, branchName)
	if err != nil {
		return &CreateResult{Success: false, Error: err}, err
	}

	logger.Info("Successfully created PR #%d: %s", prNumber, prURL)

	return &CreateResult{
		Success:    true,
		PRURL:      prURL,
		PRNumber:   prNumber,
		BranchName: branchName,
	}, nil
}

// commitChanges stages and commits the changes
func (c *Creator) commitChanges(worktreePath string, issue *watcher.Issue) error {
	logger.Info("Committing changes in %s", worktreePath)

	// Check for changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if len(bytes.TrimSpace(output)) == 0 {
		logger.Warn("No changes to commit")
		return nil
	}

	// Stage all changes
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Create commit message
	commitMsg := fmt.Sprintf("fix: Resolve issue #%d - %s", issue.Number, issue.Title)

	// Commit
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = worktreePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit: %w, output: %s", err, string(output))
	}

	logger.Info("Changes committed successfully")
	return nil
}

// pushBranch pushes the branch to remote
func (c *Creator) pushBranch(worktreePath, branchName string) error {
	logger.Info("Pushing branch %s", branchName)

	cmd := exec.Command("git", "push", "-u", "origin", branchName)
	cmd.Dir = worktreePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push branch: %w, output: %s", err, string(output))
	}

	logger.Info("Branch pushed successfully")
	return nil
}

// createPR creates the pull request
func (c *Creator) createPR(issue *watcher.Issue, branchName string) (string, int, error) {
	logger.Info("Creating pull request from %s to %s", branchName, c.cfg.BaseBranch)

	// Build PR title
	title := c.cfg.TitlePrefix + issue.Title

	// Build PR body
	body := c.buildPRBody(issue)

	// Create PR using gh cli
	args := []string{
		"pr", "create",
		"--repo", fmt.Sprintf("%s/%s", c.owner, c.repo),
		"--title", title,
		"--body", body,
		"--base", c.cfg.BaseBranch,
		"--head", branchName,
	}

	if c.cfg.AllowEdits {
		args = append(args, "--allow-maintainer-edits")
	}

	// Add reviewers if configured
	for _, reviewer := range c.cfg.Reviewers {
		args = append(args, "--reviewer", reviewer)
	}

	cmd := exec.Command(c.ghCLI, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("failed to create PR: %w, output: %s", err, string(output))
	}

	// Extract PR URL and number from output
	outputStr := string(output)
	prURL := strings.TrimSpace(outputStr)

	// Get PR number from URL - extract last number from URL
	parts := strings.Split(prURL, "/pull/")
	var prNumber int
	if len(parts) > 1 {
		fmt.Sscanf(parts[len(parts)-1], "%d", &prNumber)
	}

	// Alternative: use gh pr view to get number
	if prNumber == 0 {
		cmd = exec.Command(c.ghCLI, "pr", "view", "--repo", fmt.Sprintf("%s/%s", c.owner, c.repo),
			"--json", "number,url")
		output, _ := cmd.Output()
		// Parse JSON to get number
		prNumber = c.parsePRNumber(string(output))
	}

	return prURL, prNumber, nil
}

// buildPRBody builds the PR description body
func (c *Creator) buildPRBody(issue *watcher.Issue) string {
	var body strings.Builder

	body.WriteString(fmt.Sprintf("## Summary\n\n"))
	body.WriteString(fmt.Sprintf("- Resolves #%d\n\n", issue.Number))

	body.WriteString("## Changes\n\n")
	body.WriteString("This PR implements the solution for the issue.\n\n")

	body.WriteString("## Test Plan\n\n")
	body.WriteString("- [ ] Code builds successfully\n")
	body.WriteString("- [ ] All tests pass\n")
	body.WriteString("- [ ] Manual testing completed\n\n")

	body.WriteString("---\n")
	body.WriteString("*Generated by AI Developer Agent*")

	return body.String()
}

// parsePRNumber extracts PR number from JSON output
func (c *Creator) parsePRNumber(jsonStr string) int {
	// Simple extraction - look for "number": X
	parts := strings.Split(jsonStr, `"number":`)
	if len(parts) > 1 {
		var n int
		fmt.Sscanf(parts[1], "%d", &n)
		return n
	}
	return 0
}

// Close closes a PR (after human approval)
func (c *Creator) Close(prNumber int) error {
	logger.Info("Closing PR #%d", prNumber)

	cmd := exec.Command(c.ghCLI, "pr", "close",
		"--repo", fmt.Sprintf("%s/%s", c.owner, c.repo),
		fmt.Sprintf("%d", prNumber))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to close PR: %w", err)
	}

	return nil
}

// Merge merges a PR (after human approval)
func (c *Creator) Merge(prNumber int) error {
	logger.Info("Merging PR #%d", prNumber)

	cmd := exec.Command(c.ghCLI, "pr", "merge",
		"--repo", fmt.Sprintf("%s/%s", c.owner, c.repo),
		"--admin", "--auto",
		fmt.Sprintf("%d", prNumber))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to merge PR: %w", err)
	}

	logger.Info("PR #%d merged successfully", prNumber)
	return nil
}
