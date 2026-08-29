package cmdb

import (
	"errors"
	"fmt"
	"time"
)

const (
	DiscoveryJobQueued        = "queued"
	DiscoveryJobDiscovering   = "discovering"
	DiscoveryJobDiscovered    = "discovered"
	DiscoveryJobReconciling   = "reconciling"
	DiscoveryJobSucceeded     = "succeeded"
	DiscoveryJobPartialFailed = "partial_failed"
	DiscoveryJobFailed        = "failed"
	DiscoveryJobCancelled     = "cancelled"
)

var (
	ErrInvalidJobTransition = errors.New("invalid discovery job transition")
	ErrJobLeaseConflict     = errors.New("discovery job lease conflict")
	ErrJobAttemptsExhausted = errors.New("discovery job attempts exhausted")
)

var discoveryJobTransitions = map[string]map[string]struct{}{
	DiscoveryJobQueued: {
		DiscoveryJobDiscovering: {}, DiscoveryJobCancelled: {},
	},
	DiscoveryJobDiscovering: {
		DiscoveryJobDiscovered: {}, DiscoveryJobPartialFailed: {}, DiscoveryJobFailed: {}, DiscoveryJobCancelled: {},
	},
	DiscoveryJobDiscovered: {
		DiscoveryJobReconciling: {}, DiscoveryJobSucceeded: {}, DiscoveryJobPartialFailed: {}, DiscoveryJobFailed: {}, DiscoveryJobCancelled: {},
	},
	DiscoveryJobReconciling: {
		DiscoveryJobSucceeded: {}, DiscoveryJobPartialFailed: {}, DiscoveryJobFailed: {}, DiscoveryJobCancelled: {},
	},
}

func TransitionDiscoveryJob(job *DiscoveryJob, target string, now time.Time) error {
	if job == nil {
		return fmt.Errorf("%w: nil job", ErrInvalidJobTransition)
	}
	if _, ok := discoveryJobTransitions[job.Status][target]; !ok {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidJobTransition, job.Status, target)
	}
	job.Status = target
	if target == DiscoveryJobDiscovering && job.StartedAt == nil {
		job.StartedAt = timePointer(now)
	}
	if isTerminalDiscoveryJobStatus(target) {
		job.FinishedAt = timePointer(now)
		job.LeaseOwner = ""
		job.LeaseExpiresAt = nil
	}
	return nil
}

func ClaimDiscoveryJob(job *DiscoveryJob, workerID string, now time.Time, ttl time.Duration) (int64, error) {
	if job == nil || workerID == "" || ttl <= 0 {
		return 0, fmt.Errorf("%w: invalid claim", ErrJobLeaseConflict)
	}
	if job.Attempt >= job.MaxAttempts {
		return 0, ErrJobAttemptsExhausted
	}
	claimable := job.Status == DiscoveryJobQueued ||
		(job.Status == DiscoveryJobDiscovering && job.LeaseExpiresAt != nil && !job.LeaseExpiresAt.After(now))
	if !claimable {
		return 0, ErrJobLeaseConflict
	}
	if job.Status == DiscoveryJobQueued {
		if err := TransitionDiscoveryJob(job, DiscoveryJobDiscovering, now); err != nil {
			return 0, err
		}
	}
	job.Attempt++
	job.FencingToken++
	job.LeaseOwner = workerID
	job.HeartbeatAt = timePointer(now)
	job.LeaseExpiresAt = timePointer(now.Add(ttl))
	return job.FencingToken, nil
}

func HeartbeatDiscoveryJob(job *DiscoveryJob, workerID string, fencingToken int64, now time.Time, ttl time.Duration) error {
	if job == nil || job.Status != DiscoveryJobDiscovering || ttl <= 0 ||
		job.LeaseOwner != workerID || job.FencingToken != fencingToken ||
		job.LeaseExpiresAt == nil || !job.LeaseExpiresAt.After(now) {
		return ErrJobLeaseConflict
	}
	job.HeartbeatAt = timePointer(now)
	job.LeaseExpiresAt = timePointer(now.Add(ttl))
	return nil
}

func RequestDiscoveryJobCancellation(job *DiscoveryJob, now time.Time) error {
	if job == nil || isTerminalDiscoveryJobStatus(job.Status) {
		return fmt.Errorf("%w: cannot cancel %v", ErrInvalidJobTransition, discoveryJobStatus(job))
	}
	job.CancelRequestedAt = timePointer(now)
	if job.Status == DiscoveryJobQueued {
		return TransitionDiscoveryJob(job, DiscoveryJobCancelled, now)
	}
	return nil
}

func NewDiscoveryJobRetry(previous *DiscoveryJob, now time.Time) (*DiscoveryJob, error) {
	if previous == nil || (previous.Status != DiscoveryJobFailed && previous.Status != DiscoveryJobPartialFailed) {
		return nil, fmt.Errorf("%w: retry requires failed job", ErrInvalidJobTransition)
	}
	if previous.Attempt >= previous.MaxAttempts {
		return nil, ErrJobAttemptsExhausted
	}
	return &DiscoveryJob{
		SourceID: previous.SourceID, Status: DiscoveryJobQueued, Operation: previous.Operation,
		RequestFingerprint: previous.RequestFingerprint, SourceSnapshot: previous.SourceSnapshot,
		ScopeSnapshot: previous.ScopeSnapshot, SnapshotGeneration: previous.SnapshotGeneration,
		RequestedBy: previous.RequestedBy, QueuedAt: timePointer(now), ParentJobID: previous.ID,
		MaxAttempts: previous.MaxAttempts, TenantID: previous.TenantID,
	}, nil
}

func isTerminalDiscoveryJobStatus(status string) bool {
	return status == DiscoveryJobSucceeded || status == DiscoveryJobPartialFailed ||
		status == DiscoveryJobFailed || status == DiscoveryJobCancelled
}

func discoveryJobStatus(job *DiscoveryJob) string {
	if job == nil {
		return "<nil>"
	}
	return job.Status
}

func timePointer(value time.Time) *time.Time { return &value }
