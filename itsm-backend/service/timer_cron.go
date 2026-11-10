package service

// 定时器表达式的纯函数实现已下沉 dto/timer_expression_dto.go。
// 本文件只保留依赖 ent.Client 的租户时区读取，以及 service 包内既有的
// 重复判定/循环次数辅助函数；三处公开入口改为薄委托，行为逐字不变。

import (
	"context"
	"strconv"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

const DefaultTimerTimezone = dto.DefaultTimerTimezone

// ParseTimerExpression 识别 BPMN 定时表达式类型（委托 dto 单一实现）。
func ParseTimerExpression(expr string) (ExpressionType, error) {
	return dto.ParseTimerExpression(expr)
}

// ResolveLocation 解析 IANA 时区名；空值或非法名称回退到 DefaultTimerTimezone。
func ResolveLocation(tz string) *time.Location {
	return dto.ResolveLocation(tz)
}

func LoadTenantLocation(ctx context.Context, client *ent.Client, tenantID int) *time.Location {
	if client == nil || tenantID <= 0 {
		return ResolveLocation("")
	}
	tn, err := client.Tenant.Get(ctx, tenantID)
	if err != nil || tn == nil {
		return ResolveLocation("")
	}
	return ResolveLocation(tn.Timezone)
}

// NextFireAt 计算下一次触发时间，结果统一转为 UTC 存储（委托 dto 单一实现）。
func NextFireAt(expr string, exprType ExpressionType, loc *time.Location, after time.Time) (time.Time, error) {
	return dto.NextFireAt(expr, exprType, loc, after)
}

func IsRecurring(expr string, exprType ExpressionType) bool {
	switch exprType {
	case ExprTypeCron:
		return true
	case ExprTypeCycle:
		return strings.HasPrefix(strings.TrimSpace(expr), "R")
	default:
		return false
	}
}

func CycleRemaining(expr string) (remaining int, bounded bool, ok bool) {
	parts := dto.SplitCycleExpression(strings.TrimSpace(expr))
	if len(parts) != 2 {
		return 0, false, false
	}
	spec := strings.TrimSpace(parts[0])
	if spec == "" {
		// splitCycleExpression 对无前缀形式返回 {"", expr}
		return 1, true, true
	}
	if !strings.HasPrefix(spec, "R") {
		return 0, false, false
	}
	countStr := strings.TrimPrefix(spec, "R")
	if countStr == "" {
		return -1, false, true
	}
	n, err := strconv.Atoi(countStr)
	if err != nil || n < 0 {
		return 0, false, false
	}
	return n, true, true
}
