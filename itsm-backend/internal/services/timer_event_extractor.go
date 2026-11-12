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
		// 按实际内容分类，供调度器选择正确的触发时间算法。
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
