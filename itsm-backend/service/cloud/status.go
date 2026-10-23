package cloud

// NormalizeStatus 统一状态映射（发现执行器与旧 reconcile 共用，避免第二份映射漂移）。
func NormalizeStatus(srcStatus string) string { return mapStatus(srcStatus) }

// mapStatus 统一状态映射（私有，经 NormalizeStatus 导出）。
func mapStatus(srcStatus string) string {
	switch srcStatus {
	case "pending", "creating", "Creating":
		return "pending"
	case "active", "running", "Running":
		return "active"
	case "inactive", "stopped", "Stopped", "stopping", "Stopping":
		return "inactive"
	case "retired", "released", "expired", "deleted", "Released", "Expired", "Deleted":
		return "retired"
	default:
		return "inactive"
	}
}