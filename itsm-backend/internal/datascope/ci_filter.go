package datascope

import (
	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/predicate"
)

// ApplyCIFilter 将 DataScope 应用到 CI（ConfigurationItem）查询。
//
// 与 ApplyTicketFilter 同构，但 CI 没有「分配给某用户的 user FK」
// （assigned_to 仍是自由文本），故 DataScopeOwnedOrAssigned 只按 owner_id 判定。
//
// # 未归属资产的可见性（2026-11-19 显式设计，勿改成「一律收窄」）
//
// owner_id/department_id 是 2026-11-19 新增列，存量 CI（手工建账 + 发现投影）
// 在本迁移之前全部为 NULL。若把「owner_id 为空」直接判为不可见，上线当天
// 所有非管理角色的 CMDB 列表、拓扑、影响分析会全空——这不是权限收紧，
// 是把共享资产库对一线整体关停。CI 与工单的语义差别正在此：工单天然有
// 提交人/受理人，资产是共享的、可以无主。
//
// 故定义：owner_id 与 department_id 双空的 CI 属于「租户公共资产」，
// 在 Department/OwnedOrAssigned 两档下均可见（仍是租户内可见，不跨租户）。
// 一旦该 CI 被指派了 owner 或部门，即纳入行级收窄。
// 「未归属」本身是 CMDB Health 的 ownership 缺失项，让全员可见反而推动认领。
//
// 安全收窄：userID<=0（如 service 主体）或未知 scope 一律返回空集，
// 与工单侧 IDEQ(-1) 同口径——绝不能因为「算不出范围」就放行全租户资产。
func ApplyCIFilter(query *ent.ConfigurationItemQuery, userID, departmentID int, ds DataScope) *ent.ConfigurationItemQuery {
	switch ds {
	case DataScopeAll:
		return query
	case DataScopeDepartment:
		if userID <= 0 {
			return query.Where(configurationitem.IDEQ(-1))
		}
		wheres := []predicate.ConfigurationItem{
			// 本人持有
			configurationitem.OwnerIDEQ(userID),
			// 未归属的租户公共资产（见上方设计说明）
			configurationitem.And(
				configurationitem.OwnerIDIsNil(),
				configurationitem.DepartmentIDIsNil(),
			),
		}
		// departmentID<=0 时不加部门谓词：DepartmentIDEQ(0) 会命中
		// 「department_id=0」的脏数据或零值行，等于凭空放行一批本不该可见的 CI。
		if departmentID > 0 {
			wheres = append(wheres, configurationitem.DepartmentIDEQ(departmentID))
		}
		return query.Where(configurationitem.Or(wheres...))
	case DataScopeOwnedOrAssigned:
		if userID <= 0 {
			return query.Where(configurationitem.IDEQ(-1))
		}
		return query.Where(configurationitem.Or(
			configurationitem.OwnerIDEQ(userID),
			configurationitem.And(
				configurationitem.OwnerIDIsNil(),
				configurationitem.DepartmentIDIsNil(),
			),
		))
	default:
		return query.Where(configurationitem.IDEQ(-1))
	}
}
