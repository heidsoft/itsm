package service

import "context"

// operatorContextKey 操作者上下文键类型，避免与其他包的 context key 冲突
type operatorContextKey string

const (
	operatorIDContextKey   operatorContextKey = "operator_id"
	operatorNameContextKey operatorContextKey = "operator_name"
)

// SystemOperatorID 系统操作者约定值：无法解析当前用户时（如后台任务、发现同步）使用
const SystemOperatorID = 0

// WithOperator 将当前操作者信息注入 context，供服务层记录变更历史时使用。
// controller 层应在调用服务前通过本方法下传 gin context 中的用户信息。
func WithOperator(ctx context.Context, operatorID int, operatorName string) context.Context {
	ctx = context.WithValue(ctx, operatorIDContextKey, operatorID)
	ctx = context.WithValue(ctx, operatorNameContextKey, operatorName)
	return ctx
}

// OperatorFromContext 解析当前操作者信息。
// 优先读取 WithOperator 注入的类型化键；兼容直接传入 gin.Context 时的字符串键。
// 解析失败或非法值时返回 SystemOperatorID。
func OperatorFromContext(ctx context.Context) (int, string) {
	operatorID, ok := ctx.Value(operatorIDContextKey).(int)
	if !ok {
		// 兼容 gin.Context 及旧调用方使用的字符串键
		operatorID, _ = ctx.Value("user_id").(int) //nolint:staticcheck // 兼容旧字符串键
	}
	operatorName, ok := ctx.Value(operatorNameContextKey).(string)
	if !ok {
		operatorName, _ = ctx.Value("user_name").(string) //nolint:staticcheck // 兼容旧字符串键
	}
	if operatorID < 0 {
		operatorID = SystemOperatorID
	}
	return operatorID, operatorName
}
