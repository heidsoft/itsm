package datascope

// 工具执行链的可信 actor，通过 context 传递。
//
// 为什么必须是 context 而不是工具参数：模型可以构造任意 args（包括 userId /
// role / scopeAll），若工具从 args 读身份就等于让提示词注入自封管理员。
// 只有服务器在鉴权后写入的 actor 才可信；缺失时业务读面 fail-closed。
//
// 本文件只承载「身份传递」；范围判定仍由 ForTicketReader / ApplyTicketFilter /
// ForCIReader / ApplyCIFilter 单一源完成。

import "context"

type Actor struct {
	UserID       int
	Role         string
	DepartmentID int
	TenantID     int
}

type actorContextKey struct{}

// WithActor 把鉴权后的可信 actor 写入请求 context。
func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

// ActorFromContext 取可信 actor。第二个返回值为 false 表示缺失或租户不匹配，
// 调用方必须 fail-closed，不得回退「租户全量」。
func ActorFromContext(ctx context.Context, tenantID int) (Actor, bool) {
	if ctx == nil {
		return Actor{}, false
	}
	a, ok := ctx.Value(actorContextKey{}).(Actor)
	if !ok || a.TenantID != tenantID {
		return Actor{}, false
	}
	return a, true
}
