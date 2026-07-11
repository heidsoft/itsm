package middleware

// mspEnabled gates the entire /api/v1/msp/* route family. The bootstrap layer
// sets it based on cfg.Deployment.Mode so that private deployments have
// MSP endpoints return 404 even if a route is registered.
var mspEnabled = true

// SetMSPEnabled toggles the gate. main.go wires this from the deployment mode
// env var before NewApplication runs.
func SetMSPEnabled(b bool) { mspEnabled = b }

// IsMSPEnabled reports the current gate state. Useful for tests and for the
// health endpoint to surface a warning when a non-MSP deployment accidentally
// exposes the routes.
func IsMSPEnabled() bool { return mspEnabled }
