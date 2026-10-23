// Package authz is the single source of truth for permission vocabulary and
// role bindings.
//
// 权限码与角色绑定涉及四处：
//  1. router/ + handlers/ 中的 RequirePermission 家族声明（路径 → (resource, action)）
//  2. middleware/rbac_precheck_gen.go 的预检映射
//  3. 本包的权限码定义与角色绑定
//  4. DB 表 permissions / role_permissions
//
// (1)⇒(2) 由 cmd/authz-gen 单向生成，不手写。本包承载 (3)：seeder 启动时把
// cmd/authz-codegen 生成的 YAML 落库到 (4)。改权限码 / 角色绑定只改本包，
// go generate 同步其余各处。
package authz

// PermissionDef 权限码定义条目（码空间唯一源）。
//
// 字段语义：
//   - Code: 完整权限码 "{resource}:{action}"，是路由 RequirePermission 与
//     checkPermissionMatch 匹配的唯一键值；
//   - Name / Description: 用于 UI 展示与 API 文档；
//   - Resource / Action: Code 的拆分，用于 codegen / 守卫验证一致性。
type PermissionDef struct {
	Code        string
	Name        string
	Resource    string
	Action      string
	Description string
}

// Definitions 返回权限码权威清单（Permission 表的码空间唯一源）。
//
// 增删权限码的唯一入口。新增码必须满足：
//  1. 独立的资源面（避免把已有码的不同 action 重复定义）；
//  2. 至少一个路由 RequirePermission 引用（否则永远没人能用到，等于死码）；
//  3. 至少一个角色通过 builtinRoleBindings 授权（否则同上）。
//
// 守卫：
//   - pkg/seeder/role_permission_guard_test.go：码不重复 + 角色引用必须 ⊆ 本清单；
//   - router/permission_code_catalog_guard_test.go：路由声明码 ⊆ 本清单。
func Definitions() []PermissionDef {
	return []PermissionDef{
		// 工单权限
		{"ticket:read", "查看工单", "ticket", "read", "查看工单列表和详情"},
		{"ticket:write", "管理工单", "ticket", "write", "创建、编辑工单"},
		{"ticket:create", "创建工单", "ticket", "create", "创建工单"},
		{"ticket:update", "更新工单", "ticket", "update", "更新工单"},
		{"ticket:assign", "分派工单", "ticket", "assign", "分派工单"},
		{"ticket:escalate", "升级工单", "ticket", "escalate", "升级工单"},
		{"ticket:resolve", "解决工单", "ticket", "resolve", "解决工单"},
		{"ticket:approve", "审批工单", "ticket", "approve", "批准或拒绝工单审批"},
		{"ticket:close", "关闭工单", "ticket", "close", "关闭工单"},
		{"ticket:export", "导出工单", "ticket", "export", "导出工单"},
		{"ticket:import", "导入工单", "ticket", "import", "导入工单"},
		{"ticket:admin", "工单管理配置", "ticket", "admin", "管理工单自动化和配置"},
		{"ticket:delete", "删除工单", "ticket", "delete", "删除工单"},
		{"ticket_type:read", "查看工单类型", "ticket_type", "read", "查看工单类型和预设"},
		{"ticket_type:manage", "管理工单类型", "ticket_type", "manage", "创建、编辑、启停、克隆和恢复工单类型"},
		{"ticket_type:install_preset", "安装工单类型预设", "ticket_type", "install_preset", "从预设库安装工单类型"},
		{"ticket_type:archive", "归档工单类型", "ticket_type", "archive", "归档租户工单类型"},
		{"ticket_category:read", "查看工单分类", "ticket_category", "read", "查看工单分类"},
		{"ticket_category:create", "创建工单分类", "ticket_category", "create", "创建工单分类"},
		{"ticket_category:update", "更新工单分类", "ticket_category", "update", "更新工单分类"},
		{"ticket_category:delete", "删除工单分类", "ticket_category", "delete", "删除工单分类"},
		{"ticket_tag:read", "查看工单标签", "ticket_tag", "read", "查看工单标签"},
		{"ticket_tag:create", "创建工单标签", "ticket_tag", "create", "创建工单标签"},
		{"ticket_tag:update", "更新工单标签", "ticket_tag", "update", "更新工单标签"},
		{"ticket_tag:delete", "删除工单标签", "ticket_tag", "delete", "删除工单标签"},
		{"ticket_template:read", "查看工单模板", "ticket_template", "read", "查看工单模板"},
		{"ticket_template:create", "创建工单模板", "ticket_template", "create", "创建工单模板"},
		{"ticket_template:update", "更新工单模板", "ticket_template", "update", "更新工单模板"},
		{"ticket_template:delete", "删除工单模板", "ticket_template", "delete", "删除工单模板"},
		// 事件权限
		{"incident:read", "查看事件", "incident", "read", "查看事件列表和详情"},
		{"incident:write", "管理事件", "incident", "write", "创建、编辑事件"},
		{"incident:delete", "删除事件", "incident", "delete", "删除事件"},
		// 问题权限
		{"problem:read", "查看问题", "problem", "read", "查看问题列表和详情"},
		{"problem:write", "管理问题", "problem", "write", "创建、编辑问题"},
		{"problem:delete", "删除问题", "problem", "delete", "删除问题"},
		// 变更权限
		{"change:read", "查看变更", "change", "read", "查看变更列表和详情"},
		{"change:write", "管理变更", "change", "write", "创建、编辑变更"},
		{"change:delete", "删除变更", "change", "delete", "删除变更"},
		{"change:approve", "审批变更", "change", "approve", "变更评审审批/驳回/回滚"},
		{"change:rollback", "回滚变更", "change", "rollback", "变更实施后回滚"},
		// 发布权限
		{"release:read", "查看发布", "release", "read", "查看发布列表和详情"},
		{"release:write", "管理发布", "release", "write", "创建、编辑发布"},
		{"release:delete", "删除发布", "release", "delete", "删除发布"},
		{"release:approve", "审批发布", "release", "approve", "发布审批/驳回"},
		{"release:rollback", "回滚发布", "release", "rollback", "发布回滚"},
		// 资产权限
		{"asset:read", "查看资产", "asset", "read", "查看资产列表和详情"},
		{"asset:write", "管理资产", "asset", "write", "创建、编辑资产"},
		{"asset:delete", "删除资产", "asset", "delete", "删除资产"},
		// CMDB 权限
		{"cmdb:read", "查看CMDB", "cmdb", "read", "查看配置项"},
		{"cmdb:write", "管理CMDB", "cmdb", "write", "管理配置项"},
		{"cmdb:delete", "删除CMDB", "cmdb", "delete", "删除配置项和关系"},
		// 报表权限
		{"report:read", "查看报表", "report", "read", "查看报表"},
		{"report:write", "管理报表", "report", "write", "创建、编辑报表"},
		// 许可证权限
		{"license:read", "查看许可证", "license", "read", "查看许可证列表和详情"},
		{"license:write", "管理许可证", "license", "write", "创建、编辑许可证"},
		{"license:delete", "删除许可证", "license", "delete", "删除许可证"},
		// 服务目录权限
		{"service:read", "查看服务", "service", "read", "查看服务目录"},
		{"service:write", "管理服务", "service", "write", "管理服务目录"},
		{"service_catalog:read", "查看服务目录", "service_catalog", "read", "查看服务目录"},
		{"service_catalog:write", "管理服务目录", "service_catalog", "write", "创建、编辑服务目录"},
		{"service_catalog:delete", "删除服务目录", "service_catalog", "delete", "删除服务目录"},
		{"service_request:read", "查看服务请求", "service_request", "read", "查看服务请求"},
		{"service_request:write", "处理服务请求", "service_request", "write", "创建、处理服务请求"},
		{"service_request:delete", "删除服务请求", "service_request", "delete", "删除服务请求"},
		// 审批动作独立权限
		{"service_request:approve", "审批服务请求", "service_request", "approve", "审批/驳回服务请求（L1/L2/L3 审批人）"},
		// 通知权限（路由 RequirePermission("notification", "read"/"create") 依赖）
		{"notification:read", "查看通知", "notification", "read", "查看通知和偏好设置"},
		{"notification:create", "发送通知", "notification", "create", "创建/发送通知"},
		// SLA权限
		{"sla:read", "查看SLA", "sla", "read", "查看SLA定义"},
		{"sla:write", "管理SLA", "sla", "write", "管理SLA定义"},
		{"sla:delete", "删除SLA", "sla", "delete", "删除SLA定义"},
		// 用户权限
		{"user:read", "查看用户", "user", "read", "查看用户列表"},
		{"user:write", "管理用户", "user", "write", "创建、编辑用户"},
		{"user:delete", "删除用户", "user", "delete", "删除用户"},
		// 组权限
		{"group:read", "查看组", "group", "read", "查看组列表和详情"},
		{"group:write", "管理组", "group", "write", "创建、编辑、删除组"},
		// 角色权限
		{"role:read", "查看角色", "role", "read", "查看角色列表"},
		{"role:write", "管理角色", "role", "write", "创建、编辑角色"},
		// 部门权限
		{"department:read", "查看部门", "department", "read", "查看部门列表"},
		{"department:write", "管理部门", "department", "write", "创建、编辑部门"},
		// 团队权限
		{"team:read", "查看团队", "team", "read", "查看团队列表"},
		{"team:write", "管理团队", "team", "write", "创建、编辑团队"},
		// 审批权限
		{"approval:read", "查看审批", "approval", "read", "查看审批记录"},
		{"approval:write", "管理审批", "approval", "write", "审批操作"},
		// 工作流权限
		{"workflow:read", "查看工作流", "workflow", "read", "查看工作流"},
		{"workflow:write", "管理工作流", "workflow", "write", "创建、编辑工作流"},
		// 知识库权限
		{"knowledge:read", "查看知识库", "knowledge", "read", "查看知识库文章"},
		{"knowledge:write", "管理知识库", "knowledge", "write", "创建、编辑知识库"},
		{"knowledge:delete", "删除知识库", "knowledge", "delete", "删除知识库文章"},
		// 系统权限
		{"system:read", "查看系统", "system", "read", "查看系统配置"},
		{"system:write", "系统管理", "system", "write", "管理系统配置"},
		{"org:read", "查看组织", "org", "read", "查看组织、部门和团队"},
		{"org:write", "管理组织", "org", "write", "管理组织、部门和团队"},
		{"project:read", "查看项目", "project", "read", "查看项目"},
		{"project:write", "管理项目", "project", "write", "管理项目"},
		{"application:read", "查看应用", "application", "read", "查看应用"},
		{"application:write", "管理应用", "application", "write", "管理应用"},
		{"audit:read", "查看审计", "audit", "read", "查看审计日志"},
		{"ai:read", "查看AI能力", "ai", "read", "查看AI能力和审计"},
		{"ai:write", "管理AI能力", "ai", "write", "调用和管理AI能力"},
		{"connector:write", "管理连接器", "connector", "write", "管理连接器配置"},
		{"connector:read", "查看连接器", "connector", "read", "查看连接器配置与健康状态"},
		{"email_intake:read", "查看邮件报障", "email_intake", "read", "查看邮件报障会话与分析"},
		{"email_intake:review", "复核邮件报障", "email_intake", "review", "修正、确认或拒绝邮件报障"},
		{"email_intake:retry", "重试邮件报障", "email_intake", "retry", "重试邮件解析和核验"},
		{"email_intake:override", "强制邮件开单", "email_intake", "override", "带原因绕过合同状态限制创建事件"},
		{"customer_master:read", "查看服务客户", "customer_master", "read", "查看服务客户和分部主数据"},
		{"customer_master:write", "管理服务客户", "customer_master", "write", "维护服务客户、分部和来源组织"},
		{"support_contract:read", "查看支持合同", "support_contract", "read", "查看支持合同和外部合同映射"},
		{"support_contract:write", "管理支持合同", "support_contract", "write", "维护支持合同和外部合同映射"},
		{"on_call:read", "查看值班", "on_call", "read", "查看支持组排班"},
		{"on_call:write", "管理值班", "on_call", "write", "维护支持组排班"},
		{"vendor:read", "查看供应商", "vendor", "read", "查看供应商"},
		{"vendor:write", "管理供应商", "vendor", "write", "创建、编辑供应商"},
		{"vendor:delete", "删除供应商", "vendor", "delete", "删除供应商"},
		{"survey:write", "管理调研", "survey", "write", "创建、编辑满意度调研"},
		// MSP 权限
		{"msp:read", "查看MSP", "msp", "read", "查看MSP状态和上下文"},
		{"msp:write", "管理MSP", "msp", "write", "管理MSP配置"},
		{"msp_customer:read", "查看客户", "msp_customer", "read", "查看MSP客户列表和详情"},
		{"msp_customer:write", "管理客户", "msp_customer", "write", "创建、编辑MSP客户"},
		{"msp_ticket:read", "查看客户工单", "msp_ticket", "read", "查看客户工单"},
		{"msp_ticket:write", "处理客户工单", "msp_ticket", "write", "处理客户工单"},
		{"msp_allocation:read", "查看分配", "msp_allocation", "read", "查看MSP分配"},
		{"msp_allocation:write", "管理分配", "msp_allocation", "write", "创建、编辑MSP分配"},
		{"msp_report:read", "查看报表", "msp_report", "read", "查看MSP报表"},
		{"msp_report:write", "管理报表", "msp_report", "write", "生成和管理MSP报表"},
		// middleware.RolePermissions 硬编码表引用但 DB 码空间缺失的 16 个码。
		// 缺失后果：role_permissions 无法授予这些 (resource,action)，admin/technician 权限集不完整。
		{"notification:write", "更新通知", "notification", "write", "更新/标记已读通知"},
		{"ticket_tag:write", "管理工单标签", "ticket_tag", "write", "创建、编辑工单标签"},
		{"ticket_template:write", "管理工单模板", "ticket_template", "write", "创建、编辑工单模板"},
		{"dashboard:admin", "仪表盘管理", "dashboard", "admin", "管理仪表盘配置"},
		{"knowledge:admin", "知识库管理配置", "knowledge", "admin", "管理知识库配置和分类"},
		{"incident:force-update", "强制更新事件", "incident", "force-update", "绕过状态机限制强制更新事件"},
		{"incident:admin", "事件管理配置", "incident", "admin", "管理事件规则和配置"},
		{"alert:read", "查看告警", "alert", "read", "查看告警列表"},
		{"alert:write", "管理告警", "alert", "write", "处理/确认告警"},
		{"alerts:read", "查看告警中心", "alerts", "read", "查看告警中心列表"},
		{"alerts:write", "管理告警中心", "alerts", "write", "管理告警中心规则"},
		{"permission:read", "查看权限", "permission", "read", "查看权限定义列表"},
		{"department:create", "创建部门", "department", "create", "创建部门"},
		{"department:update", "更新部门", "department", "update", "更新部门信息"},
		{"department:delete", "删除部门", "department", "delete", "删除部门"},
		{"project:delete", "删除项目", "project", "delete", "删除项目"},
		// BPMN 工作流
		{"bpmn:read", "查看BPMN流程", "bpmn", "read", "查看BPMN流程定义和实例"},
		{"bpmn:write", "管理BPMN流程", "bpmn", "write", "设计、部署BPMN流程"},
		{"bpmn:delete", "删除BPMN流程", "bpmn", "delete", "删除BPMN流程定义"},
		// 以下为 DB 已有但种子清单缺失、硬编码兜底引用的码
		{"ticket_category:write", "管理工单分类", "ticket_category", "write", "创建、编辑工单分类"},
		{"ticket_type:write", "管理工单类型", "ticket_type", "write", "创建、编辑工单类型"},
		{"ticket_type:create", "创建工单类型", "ticket_type", "create", "创建工单类型"},
		{"ticket_type:update", "更新工单类型", "ticket_type", "update", "更新工单类型"},
		{"ticket_type:delete", "删除工单类型", "ticket_type", "delete", "删除工单类型"},
		{"dashboard:read", "查看仪表盘", "dashboard", "read", "查看仪表盘"},
		{"role:delete", "删除角色", "role", "delete", "删除角色"},
		{"system_config:read", "查看系统配置", "system_config", "read", "查看系统配置"},
		{"system_config:write", "管理系统配置", "system_config", "write", "管理系统配置"},
		// 任务面：任务与流程分权。
		// 路由 /bpmn/tasks*、/workflow/tasks* 早已引用这些码，但码空间从未定义
		// → DBOnly configured 态下除 super_admin 外全部 403（审批中心不可用）；
		// 同时 bpmn:write 连带授予了「对任意任务下决策」的越权能力。
		{"task:read", "查看任务", "task", "read", "查看本人待办与候选任务"},
		{"task:update", "处理任务", "task", "update", "认领、完成、加签、投票与提交决策（归属由 handler 层二次校验）"},
		{"task:admin", "管理任务", "task", "admin", "查看租户内全量任务与流程任务统计"},
		// 以下码为「路由已引用但码空间缺失」的补齐：
		// 仅登记码定义，不授予任何角色；放开需连同授权与预检映射一并评估。
		{"survey:read", "查看满意度调查", "survey", "read", "查看问卷定义与响应"},
		{"marketplace:read", "查看扩展市场", "marketplace", "read", "查看扩展市场条目与安装记录"},
		{"marketplace:write", "管理扩展市场", "marketplace", "write", "安装、卸载与配置扩展"},
		// 目录（商品）管理面：与「安装」分成两个权限码，因为两者的作用域不同。
		//
		// marketplace_items 是全局表（无 tenant_id），所有租户共享同一份目录；
		// 而 marketplace:write 已授予租户 admin。若目录写复用该码，任一租户管理员
		// 都能改写其他租户看到、安装的条目——上架一个 name 指向未注册 provider 的
		// 连接器商品，就能让全平台安装按钮全部失败。目录写必须是平台级动作，
		// 因此独立成面并只授予 sysadmin（allPermissionCodes 自动覆盖）。
		//
		// 读码同样独立：租户侧只应看到 published 条目（ListItems 已强制），
		// 草稿/待审条目只对平台运营可见。
		{"marketplace_catalog:read", "查看商品目录", "marketplace_catalog", "read", "查看含草稿/下架状态的全局商品目录"},
		{"marketplace_catalog:write", "管理商品目录", "marketplace_catalog", "write", "上架/下架商品、维护版本与元数据（平台级）"},
		// 租户管理：码空间无语义归宿的独立管理面，
		// 收敛进 system/system_config 会丢失租户边界，故新增；授予 admin/sysadmin。
		{"tenant:read", "查看租户", "tenant", "read", "查看租户列表与详情"},
		{"tenant:write", "管理租户", "tenant", "write", "创建、更新、停用租户"},
		// SSO（OIDC）管理面：提供方配置可改登录入口与 JIT 提权策略，
		// 与 tenant 同级独立成面；bind/unbind 是用户自助动作，不走权限码。
		{"sso:read", "查看SSO配置", "sso", "read", "查看 SSO 提供方配置"},
		{"sso:write", "管理SSO配置", "sso", "write", "创建、编辑、启停 SSO 提供方"},
		// 工单规则配置面（自动化规则 / 分配规则）。
		// 不能复用 workflow:read：该码同时挂着最终用户的工单流转进度与抄送读接口
		// （/tickets/cc/my、/tickets/:id/workflow/state*），租规则列表会连带返回
		// 规则创建人的邮箱与角色，等于把租户运营配置和人员目录开放给报障人。
		// 也不能留在 system_config:*：那份码集是租户级安全配置（SMTP/密码策略/2FA），
		// 持有者只有三员与租管，实测 it_admin 看得到规则菜单却恒 403。
		{"automation_rule:read", "查看自动化规则", "automation_rule", "read", "查看工单自动化规则清单"},
		{"automation_rule:write", "管理自动化规则", "automation_rule", "write", "创建、编辑、启停、删除与试跑工单自动化规则"},
		{"assignment_rule:read", "查看分配规则", "assignment_rule", "read", "查看工单智能分配规则清单"},
		{"assignment_rule:write", "管理分配规则", "assignment_rule", "write", "创建、编辑、启停、删除与试跑工单分配规则"},
	}
}

// AllCodes 返回权限码权威清单（Definitions）中的全部码。
//
// 供 CI 守卫校验「路由声明的 (resource, action) ⊆ 码空间」：
// checkPermissionMatch 只做精确匹配，路由引用了码空间不存在的码时，
// 该路由对除 super_admin 外的所有角色永久 403。
func AllCodes() []string {
	defs := Definitions()
	out := make([]string, 0, len(defs))
	for _, d := range defs {
		out = append(out, d.Code)
	}
	return out
}
