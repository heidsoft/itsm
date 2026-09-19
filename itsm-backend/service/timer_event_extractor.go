package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TimerEventInfo 定时器事件信息（从 BPMN 提取）
type TimerEventInfo struct {
	TimerID              string
	TimerType            string // start, intermediate, boundary
	ProcessDefinitionKey string
	ActivityID           string
	Expression           string // ISO 8601 duration/date/cycle
	ExpressionType       string // duration, date, cycle
	EventName            string
	AttachedToRef        string // for boundary events
	CancelActivity       bool   // for boundary events
}

// ExtractTimerEvents 从 BPMN 定义中提取所有定时器事件
func ExtractTimerEvents(definitions *BPMNDefinitions, processDefinitionKey string) ([]*TimerEventInfo, error) {
	if definitions == nil {
		return nil, fmt.Errorf("BPMN definitions is nil")
	}

	var timers []*TimerEventInfo

	for _, process := range definitions.Processes {
		// 提取开始定时器事件
		for _, startEvent := range process.StartEvents {
			if startEvent.TimerEventDefinition != nil {
				timer := extractTimerFromDefinition(
					"start",
					processDefinitionKey,
					startEvent.ID,
					startEvent.Name,
					startEvent.TimerEventDefinition,
				)
				if timer != nil {
					timers = append(timers, timer)
				}
			}
		}

		// 提取中间定时器事件
		for _, intermediateEvent := range process.IntermediateEvents {
			if intermediateEvent.TimerEventDefinition != nil {
				timer := extractTimerFromDefinition(
					"intermediate",
					processDefinitionKey,
					intermediateEvent.ID,
					intermediateEvent.Name,
					intermediateEvent.TimerEventDefinition,
				)
				if timer != nil {
					timers = append(timers, timer)
				}
			}
		}

		// 提取边界定时器事件
		for _, boundaryEvent := range process.BoundaryEvents {
			if boundaryEvent.TimerEventDefinition != nil {
				timer := extractTimerFromDefinition(
					"boundary",
					processDefinitionKey,
					boundaryEvent.ID,
					boundaryEvent.Name,
					boundaryEvent.TimerEventDefinition,
				)
				if timer != nil {
					timer.AttachedToRef = boundaryEvent.AttachedToRef
					timer.CancelActivity = boundaryEvent.CancelActivity
					timers = append(timers, timer)
				}
			}
		}
	}

	return timers, nil
}

// extractTimerFromDefinition 从定时器事件定义中提取定时器信息
func extractTimerFromDefinition(timerType, processKey, activityID, eventName string, timerDef *BPMNTimerEventDefinition) *TimerEventInfo {
	if timerDef == nil {
		return nil
	}

	var expression string
	var expressionType string

	if timerDef.TimeDuration != "" {
		expression = timerDef.TimeDuration
		expressionType = string(ExprTypeDuration)
	} else if timerDef.TimeDate != "" {
		expression = timerDef.TimeDate
		expressionType = string(ExprTypeDate)
	} else if timerDef.TimeCycle != "" {
		expression = timerDef.TimeCycle
		// timeCycle 既可能是 ISO 8601 循环（R5/PT10M），也可能是 cron（"0 9 * * *"），
		// 按实际内容分类，供调度器选择正确的触发时间算法（Phase 5）。
		if detected, err := ParseTimerExpression(timerDef.TimeCycle); err == nil {
			expressionType = string(detected)
		} else {
			expressionType = string(ExprTypeCycle)
		}
	} else {
		return nil
	}

	return &TimerEventInfo{
		TimerID:              generateTimerID(),
		TimerType:            timerType,
		ProcessDefinitionKey: processKey,
		ActivityID:           activityID,
		Expression:           expression,
		ExpressionType:       expressionType,
		EventName:            eventName,
	}
}

// generateTimerID 生成定时器 ID
func generateTimerID() string {
	return "timer-" + uuid.New().String()
}

// CalculateFireAt 根据表达式计算触发时间（默认使用服务器本地时区）。
// 租户时区感知的场景请直接调用 NextFireAt（cron 表达式语义依赖时区）。
func CalculateFireAt(expression, expressionType string, now time.Time) (time.Time, error) {
	exprType := ExpressionType(expressionType)

	// 兼容 duration 的 Go 原生格式（如 "30m"），ISO 8601 由 NextFireAt 处理。
	if exprType == ExprTypeDuration {
		if d, err := time.ParseDuration(expression); err == nil {
			return now.Add(d).UTC(), nil
		}
	}

	return NextFireAt(expression, exprType, time.Local, now)
}

// parseISO8601Duration 解析 ISO 8601 持续时间（如 PT30M, P1D）
func parseISO8601Duration(duration string) (time.Duration, error) {
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

// hasISO8601Component 报告字符串中是否至少存在一个 "<数字><单位>" 片段（单位 H/M/S/D）。
// 允许显式的零值（如 PT0S、P0D），只拒绝完全没有可解析片段的输入。
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

// findChar 查找字符位置
func findChar(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// extractNumber 提取数字部分
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

// splitCycleExpression 分割循环表达式
func splitCycleExpression(expression string) []string {
	// 支持 R5/PT10M 或 PT10M 格式
	if idx := findChar(expression, '/'); idx >= 0 {
		return []string{expression[:idx], expression[idx+1:]}
	}
	return []string{"", expression}
}
