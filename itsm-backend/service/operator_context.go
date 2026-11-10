package service

// 操作者上下文的完整实现已下沉 dto/operator_context_dto.go。
// 这里只保留同名薄委托：WithOperator 写入与 OperatorFromContext 读取
// 必须共用同一组 context key，拆成两份实现会让 CI 变更历史静默丢操作者。

import (
	"context"

	"itsm-backend/dto"
)

// SystemOperatorID 系统操作者约定值：无法解析当前用户时（如后台任务、发现同步）使用。
const SystemOperatorID = dto.SystemOperatorID

// WithOperator 将当前操作者信息注入 context，供服务层记录变更历史时使用。
func WithOperator(ctx context.Context, operatorID int, operatorName string) context.Context {
	return dto.WithOperator(ctx, operatorID, operatorName)
}

// OperatorFromContext 解析当前操作者信息（委托 dto 单一实现）。
func OperatorFromContext(ctx context.Context) (int, string) {
	return dto.OperatorFromContext(ctx)
}
