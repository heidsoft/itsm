package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/ent"

	"github.com/robfig/cron/v3"
)

// DefaultTimerTimezone 租户未配置 timezone 时的默认时区（PRD §12 决策 Q1）。
const DefaultTimerTimezone = "Asia/Shanghai"

// cronParser 标准 5 字段解析器（支持 * / 范围 / 列表 / 步长，以及 @every 等描述符）。
// 不接受 6 字段（带秒）——BPMN timer 场景秒级调度无业务意义，收窄以防误配。
var cronParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// ParseTimerExpression 识别 BPMN 定时表达式类型。
//
// 判定顺序（互斥，先命中先返回）：
//  1. ISO 8601 duration：以 P 开头且能被 parseISO8601Duration 解析（如 PT1H、P1D）
//  2. RFC3339 日期时间：如 2026-12-31T23:59:59Z
//  3. ISO 8601 cycle：以 R 开头且含 /（如 R5/PT10M、R/PT1H）
//  4. cron 表达式：5 字段标准格式（如 "0 9 * * 1-5"）或 @every 描述符
//
// 无法归类时返回错误，由调用方按"配置错误"处理（lint 会提前拦截）。
func ParseTimerExpression(expr string) (ExpressionType, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return "", fmt.Errorf("定时表达式为空")
	}

	// 1. ISO 8601 duration
	if strings.HasPrefix(trimmed, "P") {
		if _, err := parseISO8601Duration(trimmed); err == nil {
			return ExprTypeDuration, nil
		}
	}

	// 2. RFC3339 绝对时间
	if _, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return ExprTypeDate, nil
	}

	// 3. ISO 8601 cycle（R[n]/<duration>）
	if strings.HasPrefix(trimmed, "R") && strings.Contains(trimmed, "/") {
		parts := splitCycleExpression(trimmed)
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			if _, err := parseISO8601Duration(parts[1]); err == nil {
				return ExprTypeCycle, nil
			}
		}
	}

	// 4. cron
	if _, err := cronParser.Parse(trimmed); err == nil {
		return ExprTypeCron, nil
	}

	return "", fmt.Errorf("无法识别的定时表达式: %q（支持 ISO 8601 duration/date/cycle 或 cron）", expr)
}

// ResolveLocation 解析 IANA 时区名；空值或非法名称回退到 DefaultTimerTimezone。
// 返回的 *time.Location 始终非 nil。
func ResolveLocation(tz string) *time.Location {
	name := strings.TrimSpace(tz)
	if name != "" {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	loc, err := time.LoadLocation(DefaultTimerTimezone)
	if err != nil {
		// 容器内缺 tzdata 的兜底：固定偏移 +08:00，语义等价于 Asia/Shanghai。
		return time.FixedZone(DefaultTimerTimezone, 8*3600)
	}
	return loc
}

// LoadTenantLocation 读取租户时区（tenants.timezone）。
// 租户不存在、字段为空或名称非法时回退 DefaultTimerTimezone（不阻断调度注册）。
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

// NextFireAt 计算下一次触发时间（结果统一转为 UTC 存储，PRD §12 决策 Q4）。
//
//   - duration：after + duration（一次性）
//   - date：解析后的绝对时间（一次性；若已过则原样返回，由调用方判定跳过）
//   - cycle：after + duration（重复性由 IsRecurring/CycleRemaining 判定）
//   - cron：按 loc 时区计算 after 之后的下一个匹配时刻
func NextFireAt(expr string, exprType ExpressionType, loc *time.Location, after time.Time) (time.Time, error) {
	if loc == nil {
		loc = ResolveLocation("")
	}
	trimmed := strings.TrimSpace(expr)

	switch exprType {
	case ExprTypeDuration:
		d, err := parseISO8601Duration(trimmed)
		if err != nil {
			return time.Time{}, fmt.Errorf("解析 duration %q 失败: %w", expr, err)
		}
		return after.Add(d).UTC(), nil

	case ExprTypeDate:
		t, err := time.Parse(time.RFC3339, trimmed)
		if err != nil {
			return time.Time{}, fmt.Errorf("解析 date %q 失败: %w", expr, err)
		}
		return t.UTC(), nil

	case ExprTypeCycle:
		parts := splitCycleExpression(trimmed)
		if len(parts) != 2 {
			return time.Time{}, fmt.Errorf("非法 cycle 表达式: %q", expr)
		}
		d, err := parseISO8601Duration(parts[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("解析 cycle 间隔 %q 失败: %w", parts[1], err)
		}
		return after.Add(d).UTC(), nil

	case ExprTypeCron:
		sched, err := cronParser.Parse(trimmed)
		if err != nil {
			return time.Time{}, fmt.Errorf("解析 cron %q 失败: %w", expr, err)
		}
		// cron 语义依赖时区：在租户本地时间轴上求下一个匹配点，再转 UTC 存储。
		next := sched.Next(after.In(loc))
		if next.IsZero() {
			return time.Time{}, fmt.Errorf("cron %q 在当前时区 %s 无下一次触发时间", expr, loc.String())
		}
		return next.UTC(), nil

	default:
		return time.Time{}, fmt.Errorf("unknown expression type: %s", exprType)
	}
}

// IsRecurring 判断表达式是否为重复触发（cron 恒为重复；cycle 带 R 前缀为重复）。
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

// CycleRemaining 解析 cycle 表达式的剩余触发次数。
//
//	R5/PT10M → (5, true)  有限次数
//	R/PT10M  → (-1, false) 无限重复
//	PT10M    → (1, true)  无 R 前缀按一次性处理
//
// ok=false 表示表达式不是合法 cycle，调用方应回退为一次性。
func CycleRemaining(expr string) (remaining int, bounded bool, ok bool) {
	parts := splitCycleExpression(strings.TrimSpace(expr))
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
