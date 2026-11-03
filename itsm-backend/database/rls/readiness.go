package rls

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Readiness 描述「当前数据库连接角色 + 表策略姿态」能否真正让 RLS 生效。
//
// 为什么需要这个探测：RLS_MODE=enforce 只做两件事——在语句前 SET 会话变量、
// 在缺少 tenant 上下文时拒绝执行。它**不检查**数据库是否真的会按策略过滤行。
// 而 PostgreSQL 里有两类主体无条件绕过策略：
//
//  1. 超级用户（rolsuper）——FORCE 也拦不住；
//  2. BYPASSRLS 角色（rolbypassrls）；
//  3. 表 owner——对**非 FORCE** 的策略直接放行（ENABLE 只约束非 owner）。
//
// 本项目默认用 itsm 账号（superuser）连接，而策略统一是 ENABLE + 非 FORCE。
// 也就是说：只把 RLS_MODE 改成 enforce，数据库仍返回全租户数据，应用层却
// 已经开始记录「enforce_applied」——安全开关显示已开启，隔离实际不存在。
// 这比完全不启用更危险，因为它会让人停止怀疑。
//
// 因此 enforce 模式必须在启动时探测，探测不通过就拒绝启动（见
// database.InitDatabaseWithRLS）；shadow 模式探测结果只用于告警，让运维在
// 切换之前就知道还差什么。
type Readiness struct {
	Mode        Mode
	CurrentUser string
	Superuser   bool
	BypassRLS   bool
	// Checked 由 CheckReadiness 置为 true。
	//
	// 没有它时，零值 Readiness{} 的 Reasons 为空，Ready() 会返回 true——
	// 「探测从未执行」被解释成「隔离已生效」。这与 nil Readiness 是同一类
	// 缺陷，必须显式区分「检查过且通过」和「没检查」。
	Checked             bool
	RLSTables           int
	TablesWithoutPolicy int
	OwnedNonForceTables int
	// Reasons 为空表示就绪；每条是一句可执行的诊断（缺什么、怎么补）。
	Reasons []string
}

// Ready 表示 enforce 模式下 RLS 会真正过滤行。
// 未执行过检查的一律视为未就绪。
func (r *Readiness) Ready() bool { return r != nil && r.Checked && len(r.Reasons) == 0 }

// String 用于启动日志的一行摘要。
func (r *Readiness) String() string {
	state := "ready"
	if !r.Ready() {
		state = "unready"
	}
	return fmt.Sprintf(
		"rls-readiness(state=%s checked=%t user=%s superuser=%t bypassrls=%t rls_tables=%d without_policy=%d owned_non_force=%d)",
		state, r.Checked, r.CurrentUser, r.Superuser, r.BypassRLS,
		r.RLSTables, r.TablesWithoutPolicy, r.OwnedNonForceTables,
	)
}

// CheckReadiness 检查当前连接主体与 public schema 的 RLS 姿态。
//
// 只读查询，不改任何状态；可在启动、诊断、CI 冒烟中重复调用。
func CheckReadiness(ctx context.Context, db *sql.DB) (*Readiness, error) {
	if db == nil {
		return nil, fmt.Errorf("rls: readiness check requires a non-nil *sql.DB")
	}

	r := &Readiness{Checked: true}

	if err := db.QueryRowContext(ctx, `
		SELECT current_user, rolsuper, rolbypassrls
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&r.CurrentUser, &r.Superuser, &r.BypassRLS); err != nil {
		return nil, fmt.Errorf("rls: read current role attributes: %w", err)
	}

	// 三个计数一次取回，避免三次全表目录扫描。
	//
	// owned_non_force 是「当前角色拥有、已启用 RLS、但未 FORCE」的表数：
	// 这些表对当前角色等同于没开策略。
	if err := db.QueryRowContext(ctx, `
		SELECT
			count(*) FILTER (WHERE c.relrowsecurity),
			count(*) FILTER (
				WHERE c.relrowsecurity
				  AND NOT EXISTS (
					SELECT 1 FROM pg_policies p
					WHERE p.schemaname = 'public' AND p.tablename = c.relname
				  )
			),
			count(*) FILTER (
				WHERE c.relrowsecurity
				  AND NOT c.relforcerowsecurity
				  AND pg_get_userbyid(c.relowner) = current_user
			)
		FROM pg_class c
		WHERE c.relkind = 'r'
		  AND c.relnamespace = 'public'::regnamespace
	`).Scan(&r.RLSTables, &r.TablesWithoutPolicy, &r.OwnedNonForceTables); err != nil {
		return nil, fmt.Errorf("rls: read public schema RLS posture: %w", err)
	}

	var reasons []string

	if r.Superuser {
		reasons = append(reasons, fmt.Sprintf(
			"连接角色 %q 是超级用户：PostgreSQL 对超级用户无条件绕过所有 RLS 策略（FORCE 也无效）；"+
				"enforce 需要以非超级用户、非 BYPASSRLS 的应用角色连接（如 migrations/20261031_rls_roles_bootstrap.sql 创建的 itsm_app）",
			r.CurrentUser))
	}
	if r.BypassRLS {
		reasons = append(reasons, fmt.Sprintf(
			"连接角色 %q 带 BYPASSRLS 属性：策略不会生效；请用普通应用角色连接，把 BYPASSRLS 留给迁移/后台任务角色",
			r.CurrentUser))
	}
	if r.RLSTables == 0 {
		reasons = append(reasons, "public schema 中没有任何表启用 RLS：策略层为空，enforce 无从生效")
	}
	if r.TablesWithoutPolicy > 0 {
		reasons = append(reasons, fmt.Sprintf(
			"%d 张表启用了 RLS 却没有 policy：默认拒绝所有行（fail closed），业务会大面积 0 行返回",
			r.TablesWithoutPolicy))
	}
	if r.OwnedNonForceTables > 0 {
		reasons = append(reasons, fmt.Sprintf(
			"%d 张启用 RLS 的表归当前角色所有且未 FORCE：表 owner 会绕过非 FORCE 策略；"+
				"需改为 ALTER TABLE ... FORCE ROW LEVEL SECURITY，或改用非 owner 的应用角色连接",
			r.OwnedNonForceTables))
	}

	r.Reasons = reasons
	return r, nil
}

// EnforceReadinessError 把未就绪原因整理成一条启动错误。
//
// 刻意不让调用方「记个 warning 继续跑」：显式要求 enforce 却无法 enforce 时，
// 继续启动等于对外承诺一个不存在的隔离边界。
func EnforceReadinessError(r *Readiness) error {
	if r == nil || !r.Checked {
		return fmt.Errorf("rls: enforce mode readiness was not evaluated")
	}
	if r.Ready() {
		return nil
	}
	return fmt.Errorf(
		"rls: RLS_MODE=enforce but tenant isolation cannot be enforced: %s",
		strings.Join(r.Reasons, "; "),
	)
}
