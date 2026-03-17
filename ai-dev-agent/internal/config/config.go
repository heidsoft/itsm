package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Agent      AgentConfig      `yaml:"agent"`
	GitHub     GitHubConfig     `yaml:"github"`
	Claude     ClaudeConfig     `yaml:"claude"`
	Validation ValidationConfig `yaml:"validation"`
	PR         PRConfig         `yaml:"pr"`
	Human      HumanConfig      `yaml:"human"`
	Redis      RedisConfig      `yaml:"redis"`
}

type AgentConfig struct {
	PollInterval  string `yaml:"poll_interval"`
	MaxRetries    int    `yaml:"max_retries"`
	Timeout       string `yaml:"timeout"`
	WorktreeBase  string `yaml:"worktree_base"`
	LogLevel      string `yaml:"log_level"`
}

func (c AgentConfig) GetPollInterval() time.Duration {
	d, _ := time.ParseDuration(c.PollInterval)
	if d == 0 {
		d = 5 * time.Minute
	}
	return d
}

func (c AgentConfig) GetTimeout() time.Duration {
	d, _ := time.ParseDuration(c.Timeout)
	if d == 0 {
		d = 30 * time.Minute
	}
	return d
}

type GitHubConfig struct {
	Owner      string   `yaml:"owner"`
	Repo       string   `yaml:"repo"`
	Labels     []string `yaml:"labels"`
	IssueState string   `yaml:"issue_state"`
	GhCLI      string   `yaml:"gh_cli"`
}

type ClaudeConfig struct {
	CLIPath   string   `yaml:"cli_path"`
	Model     string   `yaml:"model"`
	ExtraArgs []string `yaml:"extra_args"`
}

type ValidationConfig struct {
	Enabled  bool            `yaml:"enabled"`
	Backend  BackendConfig  `yaml:"backend"`
	Frontend FrontendConfig  `yaml:"frontend"`
}

type BackendConfig struct {
	Enabled      bool `yaml:"enabled"`
	BuildTimeout int  `yaml:"build_timeout"`
	TestTimeout  int  `yaml:"test_timeout"`
}

type FrontendConfig struct {
	Enabled          bool `yaml:"enabled"`
	TypeCheckTimeout int  `yaml:"type_check_timeout"`
	LintTimeout      int  `yaml:"lint_timeout"`
}

type PRConfig struct {
	Reviewers   []string `yaml:"reviewers"`
	TitlePrefix string   `yaml:"title_prefix"`
	BaseBranch  string   `yaml:"base_branch"`
	AllowEdits  bool     `yaml:"allow_edits"`
}

type HumanConfig struct {
	Enabled         bool   `yaml:"enabled"`
	ReviewTimeout   int    `yaml:"review_timeout"`
	SlackWebhook    string `yaml:"slack_webhook"`
	DiscordWebhook  string `yaml:"discord_webhook"`
}

type RedisConfig struct {
	Address    string `yaml:"address"`
	Password   string `yaml:"password"`
	DB         int    `yaml:"db"`
	KeyPrefix  string `yaml:"key_prefix"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if cfg.Agent.PollInterval == "" {
		cfg.Agent.PollInterval = "5m"
	}
	if cfg.Agent.MaxRetries == 0 {
		cfg.Agent.MaxRetries = 3
	}
	if cfg.Agent.Timeout == "" {
		cfg.Agent.Timeout = "30m"
	}
	if cfg.Agent.WorktreeBase == "" {
		cfg.Agent.WorktreeBase = "/tmp/ai-worktrees"
	}
	if cfg.Agent.LogLevel == "" {
		cfg.Agent.LogLevel = "info"
	}
	if cfg.GitHub.GhCLI == "" {
		cfg.GitHub.GhCLI = "gh"
	}
	if cfg.Claude.CLIPath == "" {
		cfg.Claude.CLIPath = "claude"
	}
	if cfg.Claude.Model == "" {
		cfg.Claude.Model = "sonnet"
	}

	return &cfg, nil
}
