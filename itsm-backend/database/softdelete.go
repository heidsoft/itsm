package database

import (
	"context"

	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/servicerequest"
	"itsm-backend/ent/ticket"
)

// RegisterSoftDeleteInterceptors 为带 deleted_at 字段的业务实体（ticket / incident /
// problem / servicerequest）注册全局查询拦截器，确保：
//
//   - 所有 Query（List / Get / First / Only）默认附加 DeletedAtIsNil() 条件，
//     避免软删除记录出现在列表、详情、带边（WithXxx）的关联查询结果中。
//   - 所有 Count 查询也自动附加 DeletedAtIsNil()，防止仪表盘统计、分页总数
//     等接口把已软删除记录算入。
//
// 说明：
//
//   - 本拦截器只处理 Query 端的"读透明"，不把 Delete 语义从物理删改成软删。
//     Service 层应当继续使用显式 UpdateOne/Update().SetDeletedAt(now) 进行软
//     删除，避免与现有 Delete 逻辑产生双写行为或破坏 Delete 的幂等性。
//   - 如果某个场景需要读取软删除记录（例如回收站、管理员数据巡检），调用方
//     应使用 ent client 的底层 API 或加 DeletedAtNotNil() 覆盖过滤——在
//     当前 R2 阶段我们默认读路径都应该屏蔽软删除数据。
func RegisterSoftDeleteInterceptors(client *ent.Client) {
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
			}
			return next.Query(ctx, query)
		})
	}))
}
