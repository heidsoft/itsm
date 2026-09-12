// Package schema 提供 schema 治理门禁：单一源记录"哪些表被刻意设计为不含 tenant_id"。
//
// 2026-09-08（外部审计修复）：原 auditsm 是「表是否含 tenant_id」人工/经验判断，
// 导致：
//  1. 真「漏加」与「有意豁免」无法区分——reviews 只看到表名+列，无理由可循；
//  2. 新建表时无任何机制阻止「忘加 tenant_id」；
//  3. 改写建表 SQL 时无人复核豁免理由是否仍成立。
//
// 本包落地"豁免清单即真相 + 启动自检"治理：
//   - TenantExemptTables 是单一源：每个豁免表必须写明理由（reason）+ 治理负责人（owner）
//   - 最近一次复核日期（reviewed_at）+ 影响范围（scope：platform/global/per-tenant 拼接等）；
//   - ApplyGuard 在生产启动时扫 information_schema，对比豁免清单，
//     发现「未登记但缺 tenant_id 的表」按策略（Fatal / Warn / Silent）拒绝启动；
//   - 单元测试自描述：任何新增豁免必须同时新增覆盖测试，否则 CI fail。
//
// 治理流程：
//  1. 新增豁免 → 在 TenantExemptTables 添条目 + 更新 reviewed_at + 写测试；
//  2. 复核周期（建议季度）：owner 验证豁免理由仍成立 → 更新 reviewed_at；
//  3. 表 schema 变更触发 entc generate 后，CI 跑 TestTenantGuard_ExemptConsistency
//     自动识别新增缺租户表，推动补办豁免或加列。
package schema

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ExemptTable 描述「刻意不含 tenant_id 的表」的元数据：单一治理入口。
//
// 字段语义：
//   - TableName: 必须与 information_schema.tables.table_name 一致（区分大小写视 collation 而定）；
//   - Reason:   简短中文理由，要求指向具体业务/架构决策，例如「网关级可观测性，与租户无关」；
//   - Owner:    治理负责人（与 ITSM GitHub 团队对齐），负责季度复核；
//   - Scope:    豁免粒度，可选 platform(全平台共享) / global(跨租户共享) / derived(从父表继承，如 *_tags)；
//   - ReviewedAt: 上次复核日期，超过 180 天启动告警需复核（不阻断启动）。
type ExemptTable struct {
	TableName  string
	Reason     string
	Owner      string
	Scope      string
	ReviewedAt time.Time
}

// TenantExemptTables 是单一真相源。任何新增条目必须同步：
//  1. 至少一条 TableGuardTest 单元测试（断言表确实不存在于 production DB，或豁免理由引用准确）；
//  2. docs/architecture/tenant-isolation.md 的豁免章节；
//  3. CHANGELOG.md（Unreleased）。
//
// 历史豁免依据汇总（避免误删）：
//   - tags 系列（application_tags/department_tags/...）：用户可自定义，颜色/标签云跨租户共享是合理设计
//   - tenants：自身承载所有租户元数据，递归依赖无法含 tenant_id
//   - schema_migrations：迁移历史表，全局唯一
//   - user_roles / role_permissions / role_permissions_join：RBAC 关系表，
//     平台级 RBAC 不应被租户过滤拦截（详见 docs/adr/0001）
//   - ai_llm_calls：LLM 网关级可观测性，与租户无关（service/ai_telemetry.go 注释）
//   - knowledge_article_participants/sessions/version：协作快照，session id 是天然唯一键
//   - msp_allocations：MSP 模式下的租户-服务分配，是租户维度之上的概念
var TenantExemptTables = []ExemptTable{
	{
		TableName: "tenants", Reason: "租户自身元数据，递归依赖无法含 tenant_id",
		Owner: "platform", Scope: "platform",
		ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC),
	},
	{
		TableName: "schema_migrations", Reason: "迁移历史表，全局唯一",
		Owner: "platform", Scope: "platform",
		ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC),
	},
	// tags 系列：跨租户共享的颜色/标签云
	{TableName: "application_tags", Reason: "标签云跨租户共享", Owner: "platform", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "department_tags", Reason: "标签云跨租户共享", Owner: "platform", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "microservice_tags", Reason: "标签云跨租户共享", Owner: "platform", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "project_tags", Reason: "标签云跨租户共享", Owner: "platform", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "team_tags", Reason: "标签云跨租户共享", Owner: "platform", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	// RBAC 平台级关系表
	{TableName: "user_roles", Reason: "平台级 RBAC 关系，不应被租户过滤拦截（ADR 0001）", Owner: "rbac", Scope: "platform", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	// AI 网关级观测
	{TableName: "ai_llm_calls", Reason: "LLM 网关级可观测性，与租户无关（service/ai_telemetry.go）", Owner: "ai", Scope: "platform", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	// 协作快照/版本
	{TableName: "knowledge_article_participants", Reason: "协作会话快照，session_id 天然唯一", Owner: "knowledge", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "knowledge_article_session_participants", Reason: "协作会话快照", Owner: "knowledge", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "user_article_participations", Reason: "用户-会话参与记录", Owner: "knowledge", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "user_article_sessions", Reason: "用户会话记录", Owner: "knowledge", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	// MSP 顶层分配
	{TableName: "msp_allocations", Reason: "MSP 模式下的租户-服务分配，是租户维度之上概念", Owner: "platform", Scope: "platform", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	// 初始化/版本管理（无 tenant_id 由 initialization ledger 自行管理 scope）
	{TableName: "initialization_component_attempts", Reason: "初始化任务尝试记录，scope_type/scope_id 自带租户维度", Owner: "platform", Scope: "platform", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "initialization_installations", Reason: "初始化组件安装记录，scope_type/scope_id 自带租户维度", Owner: "platform", Scope: "platform", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "initialization_managed_records", Reason: "初始化托管记录，scope_type/scope_id 自带租户维度", Owner: "platform", Scope: "platform", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "initialization_runs", Reason: "初始化运行记录，scope_type/scope_id 自带租户维度", Owner: "platform", Scope: "platform", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	// 关联表/纯关系表
	{TableName: "configuration_item_incidents", Reason: "CI↔事件纯关联表", Owner: "cmdb", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "configuration_item_tags", Reason: "CI↔标签关联表", Owner: "cmdb", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "incident_related_incidents", Reason: "事件↔事件纯关联表", Owner: "incident", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "problem_changes", Reason: "问题↔变更纯关联表", Owner: "problem", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "problem_incidents", Reason: "问题↔事件纯关联表", Owner: "problem", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "ticket_category_tickets", Reason: "工单分类关联表", Owner: "ticket", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "ticket_related_tickets", Reason: "工单↔工单纯关联表", Owner: "ticket", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	// 知识库会话/版本
	{TableName: "knowledge_article_sessions", Reason: "知识库协作会话，session_id 天然唯一", Owner: "knowledge", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "knowledge_article_versions", Reason: "知识库版本快照", Owner: "knowledge", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "item_versions", Reason: "通用版本快照", Owner: "platform", Scope: "derived", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	// marketplace / messages / prompt_templates / password_reset_tokens：需评估，暂列豁免并标注 owner
	{TableName: "marketplace_items", Reason: "市场项目模板，跨租户共享", Owner: "marketplace", Scope: "global", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "messages", Reason: "消息模板/通知字典，跨租户共享（待评估：是否需要按租户隔离）", Owner: "notification", Scope: "global", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "prompt_templates", Reason: "AI prompt 模板库，跨租户共享（待评估）", Owner: "ai", Scope: "global", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
	{TableName: "password_reset_tokens", Reason: "密码重置令牌，按 token 唯一，无租户归属语义", Owner: "auth", Scope: "platform", ReviewedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
}

// Policy 控制 ApplyGuard 的严格度，由环境变量 ITSM_TENANT_GUARD_POLICY 决定。
//   - "fatal": 任何未豁免且缺 tenant_id 的表 → 拒绝启动（生产默认）
//   - "warn":  记 ERROR 日志但不阻止（dev/legacy 阶段使用）
//   - "silent": 不做任何检查（仅测试）
type Policy string

const (
	PolicyFatal  Policy = "fatal"
	PolicyWarn   Policy = "warn"
	PolicySilent Policy = "silent"
)

// ApplyGuard 在生产启动流程内调用：扫真实 DB，识别未豁免缺 tenant_id 的表，按策略处置。
//
// 返回值约定：
//   - 未授权的违规表清单（即便策略为 warn 也返回，方便调用方记指标/上报）；
//   - 错误：仅当策略为 fatal 且存在违规时返回 non-nil error，调用方应拒绝启动。
//
// 设计权衡：把"治理门禁"做成可降级的库函数，而非硬编码 bootstrap 调用，
// 方便将来在部署前 lint、CI 数据库烟测等场景复用同一逻辑。
func ApplyGuard(ctx context.Context, db *sql.DB, logger *zap.SugaredLogger, policy Policy) ([]string, error) {
	if policy == PolicySilent {
		return nil, nil
	}

	// 1. 取真实 DB 中所有 public schema 表
	rows, err := db.QueryContext(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'`)
	if err != nil {
		return nil, fmt.Errorf("scan information_schema: %w", err)
	}
	defer rows.Close()

	allTables := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		allTables[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 2. 缺 tenant_id 的表
	rows2, err := db.QueryContext(ctx, `
		SELECT t.table_name FROM information_schema.tables t
		WHERE t.table_schema = 'public' AND t.table_type = 'BASE TABLE'
		AND NOT EXISTS (SELECT 1 FROM information_schema.columns c
		                WHERE c.table_schema = 'public' AND c.table_name = t.table_name
		                AND c.column_name = 'tenant_id')`)
	if err != nil {
		return nil, fmt.Errorf("scan missing-tenant tables: %w", err)
	}
	defer rows2.Close()

	exemptSet := buildExemptSet()
	var violations []string
	for rows2.Next() {
		var name string
		if err := rows2.Scan(&name); err != nil {
			return nil, err
		}
		if exemptSet[name] {
			continue
		}
		violations = append(violations, name)
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}
	sort.Strings(violations)

	// 3. 过期豁免告警（reviewed_at > 180 天）
	staleExempts := []string{}
	cutoff := time.Now().UTC().AddDate(0, -6, 0)
	for _, e := range TenantExemptTables {
		if e.ReviewedAt.Before(cutoff) {
			staleExempts = append(staleExempts, e.TableName)
		}
	}

	switch {
	case len(staleExempts) > 0 && logger != nil:
		logger.Warnw("tenant_guard: exempt entries need review (>180d)",
			"stale_tables", staleExempts,
			"owner_action", "update ReviewedAt in internal/schema/tenant_guard.go after quarterly review")
	case logger != nil:
		logger.Infow("tenant_guard: exempt review status healthy")
	}

	if len(violations) == 0 {
		if logger != nil {
			logger.Infow("tenant_guard: pass",
				"scanned_tables", len(allTables),
				"exempt_entries", len(TenantExemptTables))
		}
		return nil, nil
	}

	msg := fmt.Sprintf("tenant_guard: %d table(s) without tenant_id and not in exempt list: %s",
		len(violations), strings.Join(violations, ", "))

	switch policy {
	case PolicyFatal:
		if logger != nil {
			logger.Errorw(msg,
				"violations", violations,
				"action", "add to TenantExemptTables (with reviewed_at + owner) or add tenant_id column")
		}
		return violations, fmt.Errorf("%s; refusing to start (policy=fatal)", msg)
	case PolicyWarn:
		if logger != nil {
			logger.Warnw(msg,
				"violations", violations,
				"action", "set ITSM_TENANT_GUARD_POLICY=fatal or register exemption")
		}
		return violations, nil
	default:
		return nil, nil
	}
}

// buildExemptSet 把 TenantExemptTables 转成快速查找集合。导出以方便测试覆盖。
func buildExemptSet() map[string]bool {
	out := make(map[string]bool, len(TenantExemptTables))
	for _, e := range TenantExemptTables {
		out[e.TableName] = true
	}
	return out
}

// ResolvePolicy 从环境变量解析策略，缺省根据 deployment mode 推断：
// production/private/saas/saas_msp → fatal；dev → warn。
func ResolvePolicy() Policy {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("ITSM_TENANT_GUARD_POLICY"))); v != "" {
		switch v {
		case "fatal":
			return PolicyFatal
		case "warn":
			return PolicyWarn
		case "silent", "off", "none":
			return PolicySilent
		}
	}
	env := strings.ToLower(os.Getenv("ENV"))
	switch env {
	case "production", "private", "saas", "saas_msp":
		return PolicyFatal
	default:
		return PolicyWarn
	}
}
