package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"itsm-agent/internal/config"
	"itsm-agent/internal/generator"
	"itsm-agent/internal/human"
	"itsm-agent/internal/logger"
	"itsm-agent/internal/pr"
	"itsm-agent/internal/validator"
	"itsm-agent/internal/watcher"
)

// Agent is the main orchestrator
type Agent struct {
	cfg        *config.Config
	watcher    *watcher.Watcher
	gen        *generator.Generator
	val        *validator.Validator
	prCreator  *pr.Creator
	reviewer   *human.Reviewer

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// In-memory deduplication (backup if Redis unavailable)
	processedIssues   map[int]bool
	processedIssuesMu sync.Mutex
}

// New creates a new agent
func New(cfg *config.Config) *Agent {
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		cfg:             cfg,
		watcher:         watcher.New(&cfg.GitHub),
		gen:             generator.New(&cfg.Claude, cfg.Agent.WorktreeBase),
		val:             validator.New(&cfg.Validation),
		prCreator:       pr.New(&cfg.PR, cfg.GitHub.Owner, cfg.GitHub.Repo, cfg.GitHub.GhCLI),
		reviewer:        human.New(&cfg.Human, cfg.GitHub.Owner, cfg.GitHub.Repo, cfg.GitHub.GhCLI),
		ctx:             ctx,
		cancel:          cancel,
		processedIssues: make(map[int]bool),
	}

	return agent
}

// Start starts the agent
func (a *Agent) Start() {
	logger.Info("Starting AI Developer Agent")
	logger.Info("Polling every %v", a.cfg.Agent.GetPollInterval())

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	a.wg.Add(1)
	go a.run(sigChan)
}

// Stop stops the agent
func (a *Agent) Stop() {
	logger.Info("Stopping AI Developer Agent...")
	a.cancel()
	a.wg.Wait()
	logger.Info("Agent stopped")
}

// run is the main loop
func (a *Agent) run(sigChan chan os.Signal) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.cfg.Agent.GetPollInterval())
	defer ticker.Stop()

	// Run immediately on start
	a.processIssues()

	for {
		select {
		case <-a.ctx.Done():
			logger.Info("Agent context cancelled, exiting loop")
			return
		case sig := <-sigChan:
			logger.Info("Received signal %v, shutting down", sig)
			return
		case <-ticker.C:
			a.processIssues()
		}
	}
}

// processIssues fetches and processes new issues
func (a *Agent) processIssues() {
	logger.Info("Checking for new issues...")

	issues, err := a.watcher.FetchIssues()
	if err != nil {
		logger.Error("Failed to fetch issues: %v", err)
		return
	}

	// Filter to new issues
	newIssues := a.watcher.FilterNewIssues(issues)

	if len(newIssues) == 0 {
		logger.Info("No new issues to process")
		return
	}

	for _, issue := range newIssues {
		if a.isProcessed(issue.Number) {
			logger.Debug("Issue #%d already processed, skipping", issue.Number)
			continue
		}

		// Process each issue
		a.processIssue(&issue)
	}
}

// isProcessed checks if issue was already processed
func (a *Agent) isProcessed(issueNumber int) bool {
	a.processedIssuesMu.Lock()
	defer a.processedIssuesMu.Unlock()
	return a.processedIssues[issueNumber]
}

// markProcessed marks issue as processed
func (a *Agent) markProcessed(issueNumber int) {
	a.processedIssuesMu.Lock()
	defer a.processedIssuesMu.Unlock()
	a.processedIssues[issueNumber] = true
}

// processIssue processes a single issue
func (a *Agent) processIssue(issue *watcher.Issue) {
	logger.Info("Processing issue #%d: %s", issue.Number, issue.Title)

	// Add "in-progress" label
	if err := a.watcher.AddLabel(issue.Number, "in-progress"); err != nil {
		logger.Warn("Failed to add in-progress label: %v", err)
	}

	// Notify comment
	comment := fmt.Sprintf("🤖 AI Developer Agent is now working on this issue. Please wait while I analyze and implement a solution.")
	if err := a.watcher.Comment(issue.Number, comment); err != nil {
		logger.Warn("Failed to add start comment: %v", err)
	}

	// Generate code with retries
	var genResult *generator.GenerateResult
	var genErr error

	for attempt := 1; attempt <= a.cfg.Agent.MaxRetries; attempt++ {
		logger.Info("Generation attempt %d/%d for issue #%d", attempt, a.cfg.Agent.MaxRetries, issue.Number)

		genResult, genErr = a.gen.Generate(issue)
		if genErr == nil {
			break
		}

		logger.Warn("Generation attempt %d failed: %v", attempt, genErr)

		// Add failure comment
		comment := fmt.Sprintf("⚠️ Generation attempt %d failed: %v", attempt, genErr)
		a.watcher.Comment(issue.Number, comment)
	}

	if genErr != nil {
		logger.Error("All generation attempts failed for issue #%d: %v", issue.Number, genErr)

		// Mark as failed
		a.watcher.AddLabel(issue.Number, "ai-failed")
		a.watcher.Comment(issue.Number, fmt.Sprintf("❌ Failed to generate code after %d attempts: %v", a.cfg.Agent.MaxRetries, genErr))
		a.markProcessed(issue.Number)
		return
	}

	// Validate code
	logger.Info("Validating generated code for issue #%d", issue.Number)
	valResult, err := a.val.Validate(genResult.WorktreePath)
	if err != nil {
		logger.Error("Validation failed for issue #%d: %v", issue.Number, err)
	}

	if !valResult.Success {
		// Add validation failure comment
		a.watcher.Comment(issue.Number, fmt.Sprintf("⚠️ Validation warnings/errors:\n%s", formatValidationErrors(valResult)))
	}

	// Create PR
	logger.Info("Creating PR for issue #%d", issue.Number)
	prResult, err := a.prCreator.Create(issue, genResult)
	if err != nil {
		logger.Error("Failed to create PR for issue #%d: %v", issue.Number, err)
		a.watcher.Comment(issue.Number, fmt.Sprintf("❌ Failed to create PR: %v", err))
		a.watcher.AddLabel(issue.Number, "ai-failed")
		a.markProcessed(issue.Number)
		return
	}

	logger.Info("Created PR #%d: %s", prResult.PRNumber, prResult.PRURL)

	// Add PR URL to issue
	a.watcher.Comment(issue.Number, fmt.Sprintf("🎉 PR created: %s", prResult.PRURL))

	// Request human review
	if err := a.reviewer.NotifyReview(issue, prResult.PRNumber, prResult.PRURL); err != nil {
		logger.Warn("Failed to send review notification: %v", err)
	}

	// Wait for approval
	approved, err := a.reviewer.WaitForApproval(prResult.PRNumber)
	if err != nil {
		logger.Warn("Error waiting for approval: %v", err)
		// Don't fail - just note the error
	}

	if approved {
		// Merge the PR
		if err := a.prCreator.Merge(prResult.PRNumber); err != nil {
			logger.Error("Failed to merge PR #%d: %v", prResult.PRNumber, err)
			a.watcher.Comment(issue.Number, fmt.Sprintf("❌ Failed to merge PR: %v", err))
		} else {
			logger.Info("PR #%d merged successfully", prResult.PRNumber)
			a.watcher.Comment(issue.Number, "✅ PR merged successfully!")
			a.watcher.RemoveLabel(issue.Number, "in-progress")
			a.watcher.AddLabel(issue.Number, "ai-completed")
		}
	} else {
		logger.Warn("PR #%d was not approved within timeout", prResult.PRNumber)
		a.watcher.AddLabel(issue.Number, "pending-review")
	}

	// Mark as processed
	a.markProcessed(issue.Number)

	logger.Info("Issue #%d processing complete", issue.Number)
}

// formatValidationErrors formats validation errors for display
func formatValidationErrors(result *validator.Result) string {
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		return "No errors or warnings."
	}

	var output string

	if len(result.Errors) > 0 {
		output += "Errors:\n"
		for _, e := range result.Errors {
			output += fmt.Sprintf("- %s\n", e)
		}
	}

	if len(result.Warnings) > 0 {
		output += "Warnings:\n"
		for _, w := range result.Warnings {
			output += fmt.Sprintf("- %s\n", w)
		}
	}

	return output
}
