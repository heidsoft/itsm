package generator

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"itsm-agent/internal/config"
	"itsm-agent/internal/logger"
	"itsm-agent/internal/watcher"
)

// Generator uses Claude Code to generate code
type Generator struct {
	cfg         *config.ClaudeConfig
	worktreeDir string
}

// New creates a new generator
func New(cfg *config.ClaudeConfig, worktreeBase string) *Generator {
	// Ensure worktree directory exists
	if err := os.MkdirAll(worktreeBase, 0o755); err != nil {
		logger.Fatal("Failed to create worktree directory: %v", err)
	}

	return &Generator{
		cfg:         cfg,
		worktreeDir: worktreeBase,
	}
}

// GenerateResult contains the result of code generation
type GenerateResult struct {
	Success      bool
	WorktreePath string
	BranchName   string
	Error        error
}

// Generate generates code for the given issue
func (g *Generator) Generate(issue *watcher.Issue) (*GenerateResult, error) {
	logger.Info("Starting code generation for issue #%d: %s", issue.Number, issue.Title)

	// Determine issue type (bug fix or feature)
	issueType := g.detectIssueType(issue)

	// Create worktree for this task
	branchName := fmt.Sprintf("ai/issue-%d-%s", issue.Number, g.sanitizeBranchName(issue.Title))
	worktreePath := filepath.Join(g.worktreeDir, fmt.Sprintf("issue-%d", issue.Number))

	if err := g.createWorktree(branchName, worktreePath); err != nil {
		return &GenerateResult{Success: false, Error: err}, err
	}

	logger.Info("Created worktree at %s on branch %s", worktreePath, branchName)

	// Build the prompt
	prompt := g.buildPrompt(issue, issueType)

	// Call Claude Code CLI
	if err := g.callClaude(prompt, worktreePath); err != nil {
		// Cleanup worktree on failure
		g.cleanupWorktree(worktreePath, branchName)
		return &GenerateResult{Success: false, Error: err}, err
	}

	logger.Info("Code generation completed for issue #%d", issue.Number)

	return &GenerateResult{
		Success:      true,
		WorktreePath: worktreePath,
		BranchName:   branchName,
	}, nil
}

// detectIssueType determines if this is a bug fix or feature request
func (g *Generator) detectIssueType(issue *watcher.Issue) string {
	body := strings.ToLower(issue.Title + " " + issue.Body)

	if strings.Contains(body, "bug") ||
		strings.Contains(body, "fix") ||
		strings.Contains(body, "error") ||
		strings.Contains(body, "crash") ||
		strings.Contains(body, "fail") {
		return "bug-fix"
	}

	return "feature"
}

// sanitizeBranchName creates a safe branch name
func (g *Generator) sanitizeBranchName(title string) string {
	// Replace spaces with hyphens and remove special characters
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, s)

	// Truncate if too long
	if len(s) > 50 {
		s = s[:50]
	}

	// Remove trailing hyphens
	s = strings.Trim(s, "-")

	return s
}

// createWorktree creates a new git worktree for development
func (g *Generator) createWorktree(branchName, worktreePath string) error {
	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		logger.Info("Worktree already exists, removing and recreating")
		if err := g.cleanupWorktree(worktreePath, branchName); err != nil {
			logger.Warn("Failed to cleanup existing worktree: %v", err)
		}
	}

	// Create worktree
	cmd := exec.Command("git", "worktree", "add", worktreePath, "-B", branchName)
	cmd.Dir = filepath.Dir(g.worktreeDir) // Parent of worktree dir (repo root)
	if _, err := cmd.CombinedOutput(); err != nil {
		// Try with -c instead of -B for existing branches
		cmd = exec.Command("git", "worktree", "add", worktreePath, branchName)
		cmd.Dir = filepath.Dir(g.worktreeDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create worktree: %w, output: %s", err, string(output))
		}
	}

	return nil
}

// cleanupWorktree removes a worktree
func (g *Generator) cleanupWorktree(worktreePath, branchName string) error {
	logger.Info("Cleaning up worktree at %s", worktreePath)

	// Remove worktree
	cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Warn("Failed to remove worktree: %v, output: %s", err, string(output))
	}

	// Delete branch if it exists
	cmd = exec.Command("git", "branch", "-D", branchName)
	cmd.Dir = worktreePath
	cmd.Run()

	return nil
}

// buildPrompt creates a detailed prompt for Claude
func (g *Generator) buildPrompt(issue *watcher.Issue, issueType string) string {
	var promptBuilder strings.Builder

	promptBuilder.WriteString(fmt.Sprintf("## GitHub Issue #%d\n\n", issue.Number))
	promptBuilder.WriteString(fmt.Sprintf("**Title:** %s\n\n", issue.Title))
	promptBuilder.WriteString(fmt.Sprintf("**Description:**\n%s\n\n", issue.Body))

	if issueType == "bug-fix" {
		promptBuilder.WriteString(`
## Task: Bug Fix

Please analyze this bug report and implement a fix. Follow these steps:

1. First, explore the codebase to understand the relevant code
2. Identify the root cause of the bug
3. Implement the fix
4. If applicable, add or update tests

`)
	} else {
		promptBuilder.WriteString(`
## Task: Feature Implementation

Please analyze this feature request and implement it. Follow these steps:

1. First, explore the codebase to understand the existing architecture
2. Plan the implementation approach
3. Implement the feature
4. Add appropriate tests

`)
	}

	promptBuilder.WriteString(`

## Important Guidelines

- Use the project's existing code style and patterns
- Follow the architecture documented in CLAUDE.md
- Keep changes focused and minimal
- Write clean, maintainable code
- Add tests for new functionality
- Do NOT create unnecessary documentation files unless explicitly requested

## Output Format

When you're done, provide a summary of:
1. Files changed
2. Brief explanation of the changes
3. Any tests added or updated

Now please start working on this issue. Use "exit" when you have completed the implementation.
`)

	return promptBuilder.String()
}

// callClaude invokes the Claude CLI with the generated prompt
func (g *Generator) callClaude(prompt, worktreePath string) error {
	logger.Info("Calling Claude Code CLI in %s", worktreePath)

	// Build claude command
	args := []string{
		g.cfg.CLIPath,
		"--print",
		"--no-pretty",
	}

	if g.cfg.Model != "" {
		args = append(args, "--model", g.cfg.Model)
	}

	// Add project context
	args = append(args, "--project", worktreePath)

	// Add prompt via stdin
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo %q | %s", prompt, strings.Join(args, " ")))

	// Set working directory to the worktree
	cmd.Dir = worktreePath

	// Set timeout
	done := make(chan error, 1)
	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	go func() {
		done <- cmd.Run()
	}()

	// Wait with timeout
	timeout := 30 * time.Minute
	select {
	case err := <-done:
		if err != nil {
			logger.Error("Claude CLI failed: %v, stderr: %s", err, stderr.String())
			return fmt.Errorf("claude cli failed: %w, output: %s", err, stderr.String())
		}
		logger.Info("Claude CLI completed successfully")
		logger.Debug("Claude output: %s", stdout.String())
		return nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return fmt.Errorf("claude cli timed out after %v", timeout)
	}
}

// GetChanges returns the git diff for the worktree
func (g *Generator) GetChanges(worktreePath string) (string, error) {
	cmd := exec.Command("git", "diff", "--stat")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get changes: %w", err)
	}

	return string(output), nil
}
