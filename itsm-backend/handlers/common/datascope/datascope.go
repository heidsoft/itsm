// Package datascope 提供跨域复用的行级数据权限（DataScope）原语。
//
// 该模式最初在 ticket 域落地（repository/ticket，阻断8 修复）：普通角色
// 只能看到本人创建或分配给自己的单据，管理角色可见全租户。此处将其抽离为
// 共享包，供 change/incident/problem/release/service_request 等域统一接入，
// 避免每个域各写一份枚举与角色判定。
package datascope

// DataScope 行级数据权限范围。
type DataScope int

const (
	// DataScopeAll 全租户可见（admin/manager/super_admin/sysadmin 等管理角色）。
	DataScopeAll DataScope = iota
	// DataScopeOwnedOrAssigned 仅可见本人创建或分配给本人的单据（end_user/agent）。
	DataScopeOwnedOrAssigned
)

// IsDataScopeAllRole 判断角色是否拥有全租户可见权限（DataScopeAll）。
// 管理角色（super_admin/admin/manager/sysadmin）可见全租户数据，
// 其余角色（end_user/agent 等）只能查看本人创建或分配给自己的数据。
// 未知/空角色按非全量处理（安全默认：收窄而非放宽）。
func IsDataScopeAllRole(role string) bool {
	switch role {
	case "super_admin", "admin", "manager", "sysadmin":
		return true
	default:
		return false
	}
}
