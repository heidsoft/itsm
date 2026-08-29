package cmdb

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDiscoveryJobStateMachineRejectsIllegalTransition(t *testing.T) {
	now := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	job := &DiscoveryJob{Status: DiscoveryJobQueued}

	err := TransitionDiscoveryJob(job, DiscoveryJobSucceeded, now)

	require.ErrorIs(t, err, ErrInvalidJobTransition)
	require.Equal(t, DiscoveryJobQueued, job.Status)
}

func TestDiscoveryJobLeaseUsesFencingAndRejectsStaleWorker(t *testing.T) {
	now := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	job := &DiscoveryJob{Status: DiscoveryJobQueued, MaxAttempts: 3}

	firstToken, err := ClaimDiscoveryJob(job, "worker-a", now, time.Minute)
	require.NoError(t, err)
	require.EqualValues(t, 1, firstToken)

	secondToken, err := ClaimDiscoveryJob(job, "worker-b", now.Add(2*time.Minute), time.Minute)
	require.NoError(t, err)
	require.EqualValues(t, 2, secondToken)
	require.Equal(t, "worker-b", job.LeaseOwner)

	err = HeartbeatDiscoveryJob(job, "worker-a", firstToken, now.Add(2*time.Minute), time.Minute)
	require.ErrorIs(t, err, ErrJobLeaseConflict)
}

func TestDiscoveryJobCancellationAndRetryRules(t *testing.T) {
	now := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	queued := &DiscoveryJob{Status: DiscoveryJobQueued}
	require.NoError(t, RequestDiscoveryJobCancellation(queued, now))
	require.Equal(t, DiscoveryJobCancelled, queued.Status)
	require.NotNil(t, queued.FinishedAt)

	failed := &DiscoveryJob{ID: 9, TenantID: 42, SourceID: "source-a", Status: DiscoveryJobFailed, Attempt: 1, MaxAttempts: 3}
	retry, err := NewDiscoveryJobRetry(failed, now)
	require.NoError(t, err)
	require.Equal(t, 9, retry.ParentJobID)
	require.Equal(t, 42, retry.TenantID)
	require.Equal(t, DiscoveryJobQueued, retry.Status)

	failed.Attempt = failed.MaxAttempts
	_, err = NewDiscoveryJobRetry(failed, now)
	require.True(t, errors.Is(err, ErrJobAttemptsExhausted))
}
