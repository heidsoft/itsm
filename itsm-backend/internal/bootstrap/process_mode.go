package bootstrap

import (
	"fmt"
	"strings"
)

type ProcessMode string

const (
	ProcessModeAPI    ProcessMode = "api"
	ProcessModeWorker ProcessMode = "worker"
	ProcessModeAll    ProcessMode = "all"
)

func ParseProcessMode(value string) (ProcessMode, error) {
	mode := ProcessMode(strings.ToLower(strings.TrimSpace(value)))
	if mode == "" {
		return ProcessModeAll, nil
	}
	switch mode {
	case ProcessModeAPI, ProcessModeWorker, ProcessModeAll:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid ITSM_PROCESS_MODE %q: expected api, worker, or all", value)
	}
}

func ValidateProcessMode(mode ProcessMode, environment string) error {
	if mode == ProcessModeAll && strings.EqualFold(strings.TrimSpace(environment), "production") {
		return fmt.Errorf("ITSM_PROCESS_MODE=all is development-only; use api or worker in production")
	}
	return nil
}
