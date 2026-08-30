// Package datascope 提供跨域复用的行级数据权限（DataScope）原语。
//
// 该模式最初在 ticket 域落地（repository/ticket，阻断8 修复）：普通角色
// 只能看到本人创建或分配给自己的单据，管理角色可见全租户。此处将其抽离为
// 共享包，供 change/incident/problem/release/service_request 等域统一接入，
// 避免每个域各写一份枚举与角色判定。
package datascope

import (
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
)

// DataScope 行级数据权限范围。
type DataScope int

const (
	// DataScopeAll 全租户可见（admin/manager/super_admin/sysadmin 等管理角色）。
	DataScopeAll DataScope = iota
	// DataScopeDepartment 仅可见本部门数据（需部门树查询支持）。
	DataScopeDepartment
	// DataScopeOwnedOrAssigned 仅可见本人创建或分配给本人的单据（end_user/agent）。
	DataScopeOwnedOrAssigned
)

// ParseDataScope 从 ent Role 的 data_scope 字段解析 DataScope 常量。
// 未知值默认返回 DataScopeOwnedOrAssigned（安全收窄）。
func ParseDataScope(dataScope string) DataScope {
	switch dataScope {
	case "all":
		return DataScopeAll
	case "department":
		return DataScopeDepartment
	case "owner":
		return DataScopeOwnedOrAssigned
	default:
		// 未知值安全收窄
		return DataScopeOwnedOrAssigned
	}
}

// FromRoleEntity 从 ent Role 实体获取 DataScope。
func FromRoleEntity(r *ent.Role) DataScope {
	if r == nil {
		return DataScopeOwnedOrAssigned
	}
	// role.DataScope 是 role.DataScope 类型，转换为 string 再解析
	return ParseDataScope(string(r.DataScope))
}

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

// IsDataScopeDepartmentRole 判断角色是否拥有本部门数据权限。
// 某些角色（如 department_manager）可配置为 department 档。
func IsDataScopeDepartmentRole(role string) bool {
	switch role {
	case "department_manager", "l1_support", "l2_support":
		return true
	default:
		return false
	}
}

// DataScopeFromRole 根据角色代码推断默认 DataScope。
// 注意：此函数用于 Role.data_scope 字段为空时的兜底推断，
// 正常应优先使用 Role 实体的 data_scope 字段。
func DataScopeFromRole(roleCode string) DataScope {
	if IsDataScopeAllRole(roleCode) {
		return DataScopeAll
	}
	if IsDataScopeDepartmentRole(roleCode) {
		return DataScopeDepartment
	}
	return DataScopeOwnedOrAssigned
}

// ApplyTicketFilter 将 DataScope 过滤器应用到 ticket 查询。
// query 为 ent Ticket 查询，userID 为当前用户 ID，departmentID 为用户所属部门。
//
// 示例:
//
//	query := client.Ticket.Query()
//	query = datascope.ApplyTicketFilter(query, 123, 456, DataScopeOwnedOrAssigned)
func ApplyTicketFilter(query *ent.TicketQuery, userID, departmentID int, ds DataScope) *ent.TicketQuery {
	switch ds {
	case DataScopeAll:
		// 无过滤，全量
		return query
	case DataScopeDepartment:
		// 本人创建 或 本部门
		if userID <= 0 {
			return query.Where(ticket.IDEQ(-1)) // 安全收窄
		}
		return query.Where(ticket.Or(
			ticket.RequesterID(userID),
			ticket.DepartmentID(departmentID),
		))
	case DataScopeOwnedOrAssigned:
		// 本人创建 或 分配给本人
		if userID <= 0 {
			return query.Where(ticket.IDEQ(-1)) // 安全收窄
		}
		return query.Where(ticket.Or(
			ticket.RequesterID(userID),
			ticket.AssigneeID(userID),
		))
	default:
		// 未知 scope 安全收窄
		return query.Where(ticket.IDEQ(-1))
	}
}
