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
		expressionType = "duration"
	} else if timerDef.TimeDate != "" {
		expression = timerDef.TimeDate
		expressionType = "date"
	} else if timerDef.TimeCycle != "" {
		expression = timerDef.TimeCycle
		expressionType = "cycle"
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

// CalculateFireAt 根据表达式计算触发时间
func CalculateFireAt(expression, expressionType string, now time.Time) (time.Time, error) {
	switch expressionType {
	case "duration":
		duration, err := time.ParseDuration(expression)
		if err != nil {
			// 尝试解析 ISO 8601 格式
			duration, err = parseISO8601Duration(expression)
			if err != nil {
				return time.Time{}, fmt.Errorf("failed to parse duration expression '%s': %w", expression, err)
			}
		}
		return now.Add(duration), nil

	case "date":
		// 尝试解析 ISO 8601 日期时间
		fireAt, err := time.Parse(time.RFC3339, expression)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse date expression '%s': %w", expression, err)
		}
		return fireAt, nil

	case "cycle":
		// 循环表达式（如 R5/PT10M）- 只取第一次触发时间
		// 简化处理：解析为 duration
		return parseCycleExpression(expression, now)

	default:
		return time.Time{}, fmt.Errorf("unknown expression type: %s", expressionType)
	}
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

	return total, nil
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

// parseCycleExpression 解析循环表达式（如 R5/PT10M）
func parseCycleExpression(expression string, now time.Time) (time.Time, error) {
	// 简化实现：取第一次触发的 duration
	// 完整实现需要支持重复调度
	parts := splitCycleExpression(expression)
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid cycle expression: %s", expression)
	}

	// 解析 duration 部分
	duration, err := parseISO8601Duration(parts[1])
	if err != nil {
		return time.Time{}, err
	}

	return now.Add(duration), nil
}

// splitCycleExpression 分割循环表达式
func splitCycleExpression(expression string) []string {
	// 支持 R5/PT10M 或 PT10M 格式
	if idx := findChar(expression, '/'); idx >= 0 {
		return []string{expression[:idx], expression[idx+1:]}
	}
	return []string{"", expression}
}
