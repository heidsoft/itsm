package dto

// 定时器表达式解析与触发时点计算的单一实现。
// 2026-11-10 由 service/timer_cron.go 与 service/timer_event_extractor.go 下沉：
// handlers/cmdb 的发现调度器与回显端点需要同一份纯函数，但不得反向 import 共享层。
// service 侧保留同名薄委托，包内既有调用点与测试无需改动。

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type ExpressionType string

const (
	ExprTypeDuration ExpressionType = "duration"
	ExprTypeCron     ExpressionType = "cron"
	ExprTypeDate     ExpressionType = "date"
	// ExprTypeCycle ISO 8601 循环表达式（R5/PT10M）或重复 cron 的通用别名。
	ExprTypeCycle ExpressionType = "cycle"
)

const DefaultTimerTimezone = "Asia/Shanghai"

var cronParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func ParseTimerExpression(expr string) (ExpressionType, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return "", fmt.Errorf("定时表达式为空")
	}

	// 1. ISO 8601 duration
	if strings.HasPrefix(trimmed, "P") {
		if _, err := ParseISO8601Duration(trimmed); err == nil {
			return ExprTypeDuration, nil
		}
	}

	// 2. RFC3339 绝对时间
	if _, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return ExprTypeDate, nil
	}

	// 3. ISO 8601 cycle（R[n]/<duration>）
	if strings.HasPrefix(trimmed, "R") && strings.Contains(trimmed, "/") {
		parts := SplitCycleExpression(trimmed)
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			if _, err := ParseISO8601Duration(parts[1]); err == nil {
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

func NextFireAt(expr string, exprType ExpressionType, loc *time.Location, after time.Time) (time.Time, error) {
	if loc == nil {
		loc = ResolveLocation("")
	}
	trimmed := strings.TrimSpace(expr)

	switch exprType {
	case ExprTypeDuration:
		d, err := ParseISO8601Duration(trimmed)
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
		parts := SplitCycleExpression(trimmed)
		if len(parts) != 2 {
			return time.Time{}, fmt.Errorf("非法 cycle 表达式: %q", expr)
		}
		d, err := ParseISO8601Duration(parts[1])
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

func ParseISO8601Duration(duration string) (time.Duration, error) {
	// 简化实现：支持 PT##M, PT##H, PT##S, P##D 格式
	// 完整实现需要更复杂的解析器
	if len(duration) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	if duration[0] != 'P' {
		return 0, fmt.Errorf("duration must start with 'P'")
	}

	var total time.Duration
	remaining := duration[1:]

	// 处理时间部分（T 后面）
	if idx := findChar(remaining, 'T'); idx >= 0 {
		timePart := remaining[idx+1:]
		remaining = remaining[:idx]

		// 解析小时
		if h := extractNumber(timePart, 'H'); h > 0 {
			total += time.Duration(h) * time.Hour
		}
		// 解析分钟
		if m := extractNumber(timePart, 'M'); m > 0 {
			total += time.Duration(m) * time.Minute
		}
		// 解析秒
		if s := extractNumber(timePart, 'S'); s > 0 {
			total += time.Duration(s) * time.Second
		}
	}

	// 处理日期部分（T 前面）
	// 解析天
	if d := extractNumber(remaining, 'D'); d > 0 {
		total += time.Duration(d) * 24 * time.Hour
	}

	// 静默失败治理：extractNumber 对无法识别的片段返回 0，若不作校验，
	// "PT"、"PTxxX" 这类输入会被解析成 0 时长 → 定时器"立即触发"，
	// 且调用方拿不到任何错误。要求至少存在一个"<数字><单位>"片段。
	if !hasISO8601Component(duration) {
		return 0, fmt.Errorf("invalid ISO 8601 duration: %q", duration)
	}

	return total, nil
}

func hasISO8601Component(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			continue
		}
		j := i
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j < len(s) {
			switch s[j] {
			case 'H', 'M', 'S', 'D':
				return true
			}
		}
		i = j
	}
	return false
}

func findChar(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func extractNumber(s string, unit byte) int {
	idx := findChar(s, unit)
	if idx < 0 {
		return 0
	}

	numStr := s[:idx]
	var num int
	for _, c := range numStr {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		}
	}
	return num
}

func SplitCycleExpression(expression string) []string {
	// 支持 R5/PT10M 或 PT10M 格式
	if idx := findChar(expression, '/'); idx >= 0 {
		return []string{expression[:idx], expression[idx+1:]}
	}
	return []string{"", expression}
}
