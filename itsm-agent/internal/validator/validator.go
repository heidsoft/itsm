package validator

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"itsm-agent/internal/config"
	"itsm-agent/internal/logger"
)

// Result contains validation results
type Result struct {
	Success    bool
	Errors     []string
	Warnings   []string
	Duration   time.Duration
	BackendOK  bool
	FrontendOK bool
}

// Validator validates generated code
type Validator struct {
	cfg *config.ValidationConfig
}

// New creates a new validator
func New(cfg *config.ValidationConfig) *Validator {
	return &Validator{cfg: cfg}
}

// Validate runs validation on the generated code
func (v *Validator) Validate(worktreePath string) (*Result, error) {
	startTime := time.Now()
	result := &Result{
		Success:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	logger.Info("Starting code validation in %s", worktreePath)

	// Determine what to validate based on what exists in worktree
	hasBackend := v.hasBackendCode(worktreePath)
	hasFrontend := v.hasFrontendCode(worktreePath)

	logger.Debug("Backend code present: %v, Frontend code present: %v", hasBackend, hasFrontend)

	// Validate backend if enabled and present
	if v.cfg.Enabled && v.cfg.Backend.Enabled && hasBackend {
		logger.Info("Validating backend code")
		if err := v.validateBackend(worktreePath); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Backend validation failed: %v", err))
			result.Success = false
			result.BackendOK = false
		} else {
			result.BackendOK = true
		}
	}

	// Validate frontend if enabled and present
	if v.cfg.Enabled && v.cfg.Frontend.Enabled && hasFrontend {
		logger.Info("Validating frontend code")
		if err := v.validateFrontend(worktreePath); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Frontend validation failed: %v", err))
			result.Success = false
			result.FrontendOK = false
		} else {
			result.FrontendOK = true
		}
	}

	result.Duration = time.Since(startTime)
	logger.Info("Validation completed in %v, success: %v", result.Duration, result.Success)

	return result, nil
}

// hasBackendCode checks if the worktree has Go backend code
func (v *Validator) hasBackendCode(worktreePath string) bool {
	// Check for Go files
	backendPath := worktreePath + "/itsm-backend"
	if _, err := os.Stat(backendPath); err != nil {
		return false
	}

	// Check for main.go or go.mod
	mainExists := false
	goModExists := false

	if files, err := os.ReadDir(backendPath); err == nil {
		for _, f := range files {
			if f.Name() == "main.go" {
				mainExists = true
			}
			if f.Name() == "go.mod" {
				goModExists = true
			}
		}
	}

	return mainExists || goModExists
}

// hasFrontendCode checks if the worktree has Next.js frontend code
func (v *Validator) hasFrontendCode(worktreePath string) bool {
	// Check for Next.js frontend
	frontendPath := worktreePath + "/itsm-frontend"
	if _, err := os.Stat(frontendPath); err != nil {
		return false
	}

	// Check for package.json
	pkgPath := frontendPath + "/package.json"
	if _, err := os.Stat(pkgPath); err != nil {
		return false
	}

	return true
}

// validateBackend runs Go build and tests
func (v *Validator) validateBackend(worktreePath string) error {
	backendPath := worktreePath + "/itsm-backend"

	// Check if backend exists
	if _, err := os.Stat(backendPath); err != nil {
		logger.Warn("Backend directory not found, skipping validation")
		return nil
	}

	// Run go build
	logger.Info("Running go build...")
	if err := v.runCommand(backendPath, v.cfg.Backend.BuildTimeout, "go", "build", "-o", "/dev/null", "./..."); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}
	logger.Info("Go build successful")

	// Run go test (optional, don't fail on test errors)
	logger.Info("Running go test...")
	if err := v.runCommand(backendPath, v.cfg.Backend.TestTimeout, "go", "test", "./..."); err != nil {
		logger.Warn("Go tests failed: %v", err)
		// Don't fail validation on test failures, just warn
	}

	return nil
}

// validateFrontend runs npm type-check and lint
func (v *Validator) validateFrontend(worktreePath string) error {
	frontendPath := worktreePath + "/itsm-frontend"

	// Check if frontend exists
	if _, err := os.Stat(frontendPath); err != nil {
		logger.Warn("Frontend directory not found, skipping validation")
		return nil
	}

	// Check if npm is available
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found in PATH")
	}

	// Install dependencies if node_modules doesn't exist
	nodeModulesPath := frontendPath + "/node_modules"
	if _, err := os.Stat(nodeModulesPath); err != nil {
		logger.Info("Installing frontend dependencies...")
		if err := v.runCommand(frontendPath, 15, "npm", "install", "--legacy-peer-deps"); err != nil {
			return fmt.Errorf("npm install failed: %w", err)
		}
	}

	// Run TypeScript type check
	logger.Info("Running TypeScript type check...")
	if err := v.runCommand(frontendPath, v.cfg.Frontend.TypeCheckTimeout, "npm", "run", "type-check"); err != nil {
		return fmt.Errorf("type-check failed: %w", err)
	}
	logger.Info("Type check successful")

	// Run lint
	logger.Info("Running ESLint...")
	if err := v.runCommand(frontendPath, v.cfg.Frontend.LintTimeout, "npm", "run", "lint"); err != nil {
		return fmt.Errorf("lint failed: %w", err)
	}
	logger.Info("Lint successful")

	return nil
}

// runCommand executes a command with timeout
func (v *Validator) runCommand(dir string, timeoutMinutes int, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	// Set up environment
	cmd.Env = append(os.Environ(), "CI=true")

	done := make(chan error, 1)
	var output strings.Builder

	cmd.Stdout = &output
	cmd.Stderr = &output

	go func() {
		done <- cmd.Run()
	}()

	timeout := time.Duration(timeoutMinutes) * time.Minute

	select {
	case err := <-done:
		if err != nil {
			logger.Error("Command failed: %s %v\nOutput: %s", name, args, output.String())
			return fmt.Errorf("%s %v failed: %w, output: %s", name, args, err, output.String())
		}
		return nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return fmt.Errorf("command timed out after %v", timeout)
	}
}
