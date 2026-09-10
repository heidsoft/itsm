package dingtalk

import "context"

// callerContext returns a detached background context for audit write —
// audit must outlive a cancelled request so signature failures are still recorded.
func callerContext() context.Context { return context.Background() }