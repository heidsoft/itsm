// Package role 提供全后端统一的用户角色词表。
//
// 背景（2026-10-07 角色模型重设）：角色字符串此前散落在 7+ 处硬编码
// （users.role 单字段、roles.code 种子表、各 handler 的 switch/case、
// 审批资格 fallback 词表），三套词汇已实际漂移——service_request 审批角色
// manager/it_admin/security_admin 在 roles 表缺失、审批 403。
//
// 本产品的角色分层（角色=权限包，不是岗位/组织架构；岗位差异由部门、
// 团队与租户自建角色承载）：
//
//	Tier 1 内置主角色（All）：users.role 枚举值，一人一主角色，
//	        同时是 roles 表的 is_system 种子，封闭词表。
//	Tier 2 ITIL 实践角色（Practice）：roles 表种子，通过 user_roles 边叠加。
//	Tier 3 MSP 协作角色（MSP）：同上，MSP/代维场景使用。
//	Retired：历史值，禁止再出现在 schema/DTO/authz/种子/前端任何一层。
//
// 规则：
//   - 判断/比较角色时一律引用本包常量，禁止裸写字符串字面量；
//   - Tier 1 变更必须同步：ent/schema/user.go 枚举、dto/user_dto.go oneof、
//     pkg/seeder.BuiltinRoles、internal/authz 权限条目、前端 PRIMARY_ROLE_OPTIONS
//     （由 domain/role/contract_test.go 锁死）；
//   - Tier 2/3 变更必须同步 internal/authz 权限条目与种子清单；
//   - 退役码只增不减：IsRetired 命中即代表该 code 不得被重新启用。
package role

import "sort"

// Tier 1 内置主角色 code（users.role 单字段值 = roles.code 种子值，单一事实源）。
const (
	SuperAdmin    = "super_admin"
	Admin         = "admin"
	SysAdmin      = "sysadmin"
	SecurityAdmin = "security_admin"
	AuditAdmin    = "audit_admin"
	ITAdmin       = "it_admin"
	Manager       = "manager"
	Agent         = "agent"
	Technician    = "technician"
	EndUser       = "end_user"
)

// All 是封闭的主角色词表；顺序即审批/管理层级参考，不作权限依据。
var All = []string{
	SuperAdmin,
	Admin,
	SysAdmin,
	SecurityAdmin,
	AuditAdmin,
	ITAdmin,
	Manager,
	Agent,
	Technician,
	EndUser,
}

// Tier 2 ITIL 实践角色 code（roles 表种子，可叠加；不进入 users.role 枚举）。
const (
	ChangeManager       = "change_manager"
	ProblemManager      = "problem_manager"
	KnowledgeManager    = "knowledge_manager"
	CmdbAdmin           = "cmdb_admin"
	ServiceCatalogAdmin = "service_catalog_admin"
)

var Practice = []string{
	ChangeManager,
	ProblemManager,
	KnowledgeManager,
	CmdbAdmin,
	ServiceCatalogAdmin,
}

// Tier 3 MSP 协作角色 code（middleware.GetMSPRBACRole 的映射目标，
// 同时也是 roles 表种子，让管理员能在角色页直接分配 MSP 协作身份）。
const (
	MspViewer     = "msp_viewer"
	MspTech       = "msp_tech"
	MspSpecialist = "msp_specialist"
	MspManager    = "msp_manager"
	MspAdmin      = "msp_admin"
)

var MSP = []string{
	MspViewer,
	MspTech,
	MspSpecialist,
	MspManager,
	MspAdmin,
}

// Retired 是已退役角色 code 的封闭清单（2026-10-07 角色模型重设）。
// security 是 users.role 的 legacy 值，语义并入 security_admin；
// 其余是岗位/团队型角色，属于组织架构而非权限包。
var Retired = []string{
	"security",
	"it_director",
	"ops_director",
	"ops_manager",
	"ops_engineer",
	"dba",
	"network_eng",
	"sd_manager",
	"l1_support",
	"l2_support",
	"l3_expert",
	"rd_manager",
	"developer",
	"qa_engineer",
	"dept_manager",
	"team_lead",
	"guest",
}

// IsRetired 判断角色 code 是否已退役（禁止再播种、再分配、再出现在词表里）。
func IsRetired(code string) bool {
	for _, r := range Retired {
		if r == code {
			return true
		}
	}
	return false
}

// IsAdminLike 判断角色是否为管理类（可见全租户数据/绕过行级 scope）。
// audit_admin 在此列出只为读视野覆盖全租户：审计员没有任何写权限码，
// 行级写仍由 RBAC 权限码（ticket:write 等）二次拦截。
func IsAdminLike(r string) bool {
	switch r {
	case SuperAdmin, Admin, Manager, SysAdmin, AuditAdmin:
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
