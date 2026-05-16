package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"itsm-agent/internal/config"
	"itsm-agent/internal/logger"
	"itsm-agent/internal/orchestrator"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	logLevel := flag.String("log-level", "", "Override log level (debug, info, warn, error)")
	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("AI Developer Agent v%s (commit: %s, date: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Override log level if specified
	if *logLevel != "" {
		cfg.Agent.LogLevel = *logLevel
	}

	// Initialize logger
	logger.Init(cfg.Agent.LogLevel, "")
	logger.Info("Starting AI Developer Agent v%s", version)
	logger.Info("Log level: %s", cfg.Agent.LogLevel)

	// Verify required tools are available
	if err := verifyDependencies(cfg); err != nil {
		logger.Fatal("Dependency check failed: %v", err)
	}

	// Create and start agent
	agent := orchestrator.New(cfg)
	agent.Start()

	// Wait for interrupt
	logger.Info("Agent is running. Press Ctrl+C to stop.")
	select {}
}

func verifyDependencies(cfg *config.Config) error {
	// Check git
	if _, err := exec.Command("git", "--version").Output(); err != nil {
		return fmt.Errorf("git is required but not found: %w", err)
	}

	// Check gh CLI
	if _, err := exec.Command(cfg.GitHub.GhCLI, "--version").Output(); err != nil {
		return fmt.Errorf("GitHub CLI (gh) is required but not found: %w", err)
	}

	// Check claude CLI
	if _, err := exec.Command(cfg.Claude.CLIPath, "--version").Output(); err != nil {
		return fmt.Errorf("Claude CLI is required but not found: %w", err)
	}

	logger.Info("All dependencies verified")
	return nil
}
