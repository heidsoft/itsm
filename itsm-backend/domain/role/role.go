// Package role 提供全后端统一的用户角色词表。
//
// 背景（2026-09-07 服务请求深测 P1-B）：角色字符串此前散落在 7+ 处硬编码
// （users.role 单字段、roles.code 种子表、各 handler 的 switch/case、
// 审批资格 fallback 词表），三套词汇已实际漂移——service_request 审批角色
// manager/it_admin/security_admin 在 roles 表缺失、审批 403。
//
// 规则：
//   - 判断/比较角色时一律引用本包常量，禁止裸写字符串字面量；
//   - 新增角色必须同时更新 SeedRoles（见 internal/bootstrap 种子）；
//   - users.role 单字段与 roles.code 种子表共用本词表，两处不同步视为 Bug。
package role

import "sort"

// 内置角色 code（users.role 单字段值 = roles.code 种子值，单一事实源）。
const (
	SuperAdmin    = "super_admin"
	Admin         = "admin"
	Manager       = "manager"
	ITAdmin       = "it_admin"
	SecurityAdmin = "security_admin"
	Agent         = "agent"
	Technician    = "technician"
	SysAdmin      = "sysadmin"
	EndUser       = "end_user"
)

// 全量合法角色清单（播种与校验用）。顺序即层级语义参考，不作权限依据。
var All = []string{
	SuperAdmin,
	Admin,
	Manager,
	ITAdmin,
	SecurityAdmin,
	Agent,
	Technician,
	SysAdmin,
	EndUser,
}

// IsAdminLike 判断角色是否为管理类（可见全租户数据/绕过行级 scope）。
// 单一源对齐 handlers/common/datascope.IsDataScopeAllRole 与各处
// role == "admin" || role == "super_admin" 式硬编码。
func IsAdminLike(r string) bool {
	switch r {
	case SuperAdmin, Admin, Manager, SysAdmin:
		return true
	default:
		return false
	}
}

// IsSuperAdmin 判断是否超级管理员。
func IsSuperAdmin(r string) bool { return r == SuperAdmin }

// IsServiceRequestApprover 判断角色是否可参与服务请求审批
// （L1 manager / L2 it_admin / L3 security_admin 及管理类兜底）。
// 与 handlers/service_request.checkEligibility 的 fallback 词表对齐。
func IsServiceRequestApprover(r string) bool {
	switch r {
	case Manager, ITAdmin, SecurityAdmin, Agent, Technician, Admin, SuperAdmin:
		return true
	default:
		return false
	}
}

// builtinSet 是 All 的 O(1) 查找表，IsBuiltinCode 使用。
var builtinSet map[string]struct{}

func init() {
	builtinSet = make(map[string]struct{}, len(All))
	for _, c := range All {
		builtinSet[c] = struct{}{}
	}
}

// IsBuiltinCode 判断角色 code 是否属于平台内置系统角色。
// 内置角色受 schema.is_system + service 层双重保护，禁止删除。
// 这是 service 层的第二道防线——即使 seed 时 IsSystem 字段漏设，
// 只要 code 命中内置词表，仍会被阻止删除。
func IsBuiltinCode(code string) bool {
	_, ok := builtinSet[code]
	return ok
}

// SortCodes 对角色 code 列表排序（稳定顺序便于测试断言与 diff）。
func SortCodes(codes []string) []string {
	out := make([]string, len(codes))
	copy(out, codes)
	sort.Strings(out)
	return out
}
