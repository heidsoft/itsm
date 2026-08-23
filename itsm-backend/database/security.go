package database

// security.go：全局数据安全拦截器
//
// 目标：从根上消除“写路径漏点”类缺陷，提供三层防护：
//
//  1. 软删除读透明（SoftDelete）：所有带 deleted_at 字段的实体，读路径
//     默认附加 DeletedAtIsNil()，避免软删除记录出现在列表/详情/关联查询/
//     计数中。范围从 ticket/incident/problem/servicerequest 扩到全部 9 个
//     带 deleted_at 的实体。
//
//  2. 写路径租户守卫（TenantWriteGuard）：每次 Create 变更，若实体带
//     tenant_id 字段则校验并强制与上下文租户一致——缺失时自动注入，不一致
//     时 fail-closed 拒绝。覆盖“跨租户插入”这一最高风险写路径漏点。
//     Update/DeleteOne 的租户校验由业务层 TenantIDEQ 兜底（见 Batch P1-infra）。
//
//  3. RLS enforce 全路径接线见 database/rls/driver.go：Exec/Query 在 enforce
//     模式下包裹进带 SET LOCAL app.current_tenant 的事务，使非事务（autocommit）
//     查询也能被 RLS 策略正确隔离（此前仅事务路径生效）。
//
// 设计约束：
//   - 守卫仅对“上下文带租户且非系统绕过”的请求生效；SystemBypass / 无租户
//     上下文（迁移、种子、跨租户 MSP 作业）一律放行，避免误伤基础设施代码。
//   - 读路径不强制附加应用层租户过滤：当 RLS 处于 enforce 模式时由数据库
//     策略负责读隔离；当前默认 off，业务层手工 TenantIDEQ 仍是主机制。

import (
	"context"
	"fmt"

	"itsm-backend/common/tenantctx"
	"itsm-backend/ent"

	// 各实体谓词包，用于软删过滤
	"itsm-backend/ent/application"
	"itsm-backend/ent/department"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/servicerequest"
	"itsm-backend/ent/systemconfig"
	"itsm-backend/ent/team"
	"itsm-backend/ent/ticket"
)

// RegisterSecurityInterceptors 注册全局软删除读拦截器与写路径租户守卫。
// rlsMode 当前保留用于未来在读路径叠加应用层租户过滤；现在读隔离在 RLS
// enforce 模式下由数据库策略负责，非 enforce 模式由业务层手工 TenantIDEQ 负责。
//
// 调用方：database.go 的 InitDatabase / InitDatabaseWithRLS，替换原
// RegisterSoftDeleteInterceptors。
func RegisterSecurityInterceptors(client *ent.Client, rlsMode string) {
	registerSoftDeleteInterceptor(client)
	registerTenantWriteGuard(client)
	// 读路径租户过滤不在此处强制——避免在无租户上下文（如 admin 跨租户看板）
	// 时静默丢数据。RLS enforce 模式下由 DB 策略负责读隔离。
	_ = rlsMode
}

// registerSoftDeleteInterceptor 为全部带 deleted_at 字段的实体附加读透明过滤。
func registerSoftDeleteInterceptor(client *ent.Client) {
	client.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) {
			switch q := query.(type) {
			case *ent.TicketQuery:
				q.Where(ticket.DeletedAtIsNil())
			case *ent.IncidentQuery:
				q.Where(incident.DeletedAtIsNil())
			case *ent.ProblemQuery:
				q.Where(problem.DeletedAtIsNil())
			case *ent.ServiceRequestQuery:
				q.Where(servicerequest.DeletedAtIsNil())
			case *ent.ApplicationQuery:
				q.Where(application.DeletedAtIsNil())
			case *ent.DepartmentQuery:
				q.Where(department.DeletedAtIsNil())
			case *ent.KnowledgeArticleQuery:
				q.Where(knowledgearticle.DeletedAtIsNil())
			case *ent.SystemConfigQuery:
				q.Where(systemconfig.DeletedAtIsNil())
			case *ent.TeamQuery:
				q.Where(team.DeletedAtIsNil())
			}
			return next.Query(ctx, query)
		})
	}))
}

// registerTenantWriteGuard 注册写路径租户守卫（fail-closed）。
//
// 规则（仅当上下文携带租户且非系统绕过时生效）：
//   - Create 变更：实体含 tenant_id 字段时，
//     * 未设置（0）→ 自动注入上下文租户，杜绝 NULL tenant_id 插入；
//     * 已设置但与上下文租户不符 → 拒绝，杜绝跨租户写入。
//   - 系统绕过 / 无租户上下文 → 放行（迁移、种子、MSP 跨租户作业）。
func registerTenantWriteGuard(client *ent.Client) {
	client.Use(ent.Hook(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if tenantctx.IsSystemBypass(ctx) {
				return next.Mutate(ctx, m)
			}
			tid, ok := tenantctx.TenantID(ctx)
			if !ok {
				// 无租户上下文：依赖调用方自行保证（如基础设施作业）。
				return next.Mutate(ctx, m)
			}

			if m.Op().Is(ent.OpCreate) {
				v, hasField := m.Field("tenant_id")
				if !hasField {
					// 该实体无 tenant_id 字段，跳过。
					return next.Mutate(ctx, m)
				}
				switch tv := v.(type) {
				case int:
					if tv != 0 && tv != tid {
						return nil, fmt.Errorf(
							"tenant guard: create %s with tenant_id=%d but request tenant=%d (cross-tenant insert blocked)",
							m.Type(), tv, tid,
						)
					}
					if tv == 0 {
						if err := m.SetField("tenant_id", tid); err != nil {
							return nil, fmt.Errorf("tenant guard: inject tenant_id: %w", err)
						}
					}
				case int64:
					if tv != 0 && int(tv) != tid {
						return nil, fmt.Errorf(
							"tenant guard: create %s with tenant_id=%d but request tenant=%d (cross-tenant insert blocked)",
							m.Type(), tv, tid,
						)
					}
					if tv == 0 {
						if err := m.SetField("tenant_id", tid); err != nil {
							return nil, fmt.Errorf("tenant guard: inject tenant_id: %w", err)
						}
					}
				}
			}
			return next.Mutate(ctx, m)
		})
	}))
}
