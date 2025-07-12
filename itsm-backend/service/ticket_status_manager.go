package service

import (
	"context"
	"fmt"
	"time"
	"itsm-backend/ent"
)

// TicketStatus 工单状态枚举
type TicketStatus string

const (
	StatusNew        TicketStatus = "new"         // 新建
	StatusInProgress TicketStatus = "in_progress" // 处理中
	StatusPending    TicketStatus = "pending"     // 待审批
	StatusResolved   TicketStatus = "resolved"    // 已解决
	StatusClosed     TicketStatus = "closed"      // 已关闭
	StatusReturned   TicketStatus = "returned"    // 退回
)

// StatusTransition 状态转换记录
type StatusTransition struct {
	TicketID    int           `json:"ticket_id"`
	FromStatus  TicketStatus  `json:"from_status"`
	ToStatus    TicketStatus  `json:"to_status"`
	UserID      int           `json:"user_id"`
	Reason      string        `json:"reason"`
	Timestamp   time.Time     `json:"timestamp"`
}

// TicketStatusManager 工单状态管理器
type TicketStatusManager struct {
	// 合法状态转换映射表
	validTransitions map[TicketStatus][]TicketStatus
	// 状态变更日志
	transitionLogs []StatusTransition
	// 数据库客户端
	client *ent.Client
}

// NewTicketStatusManager 创建新的状态管理器
func NewTicketStatusManager(client *ent.Client) *TicketStatusManager {
	manager := &TicketStatusManager{
		client:         client,
		transitionLogs: make([]StatusTransition, 0),
	}
	
	// 初始化合法状态转换规则
	manager.initValidTransitions()
	
	return manager
}

// initValidTransitions 初始化合法状态转换规则
func (tsm *TicketStatusManager) initValidTransitions() {
	tsm.validTransitions = map[TicketStatus][]TicketStatus{
		// 新建 → 处理中
		StatusNew: {StatusInProgress, StatusReturned},
		
		// 处理中 → 待审批
		StatusInProgress: {StatusPending, StatusReturned},
		
		// 待审批 → 已解决
		StatusPending: {StatusResolved, StatusReturned},
		
		// 已解决 → 已关闭
		StatusResolved: {StatusClosed, StatusReturned},
		
		// 已关闭状态可以退回
		StatusClosed: {StatusReturned},
		
		// 退回状态可以转换到任意其他状态
		StatusReturned: {StatusNew, StatusInProgress, StatusPending, StatusResolved, StatusClosed},
	}
}

// validateStatusTransition 验证状态转换是否合法
func (tsm *TicketStatusManager) validateStatusTransition(from, to string) error {
	fromStatus := TicketStatus(from)
	toStatus := TicketStatus(to)
	
	// 检查源状态是否存在
	validToStatuses, exists := tsm.validTransitions[fromStatus]
	if !exists {
		return fmt.Errorf("无效的源状态: %s", from)
	}
	
	// 检查目标状态是否在允许的转换列表中
	for _, validStatus := range validToStatuses {
		if validStatus == toStatus {
			return nil // 转换合法
		}
	}
	
	return fmt.Errorf("不允许从状态 '%s' 转换到状态 '%s'", from, to)
}

// TransitionTicketStatus 执行工单状态转换
func (tsm *TicketStatusManager) TransitionTicketStatus(ctx context.Context, ticketID int, fromStatus, toStatus string, userID int, reason string) error {
	// 验证状态转换是否合法
	if err := tsm.validateStatusTransition(fromStatus, toStatus); err != nil {
		return err
	}
	
	// 记录状态变更日志
	transition := StatusTransition{
		TicketID:   ticketID,
		FromStatus: TicketStatus(fromStatus),
		ToStatus:   TicketStatus(toStatus),
		UserID:     userID,
		Reason:     reason,
		Timestamp:  time.Now(),
	}
	
	// 添加到内存日志
	tsm.addTransitionLog(transition)
	
	// 这里可以添加数据库持久化逻辑
	// 例如：保存到审批日志表或专门的状态变更日志表
	
	return nil
}

// addTransitionLog 添加状态转换日志
func (tsm *TicketStatusManager) addTransitionLog(transition StatusTransition) {
	tsm.transitionLogs = append(tsm.transitionLogs, transition)
}

// GetTransitionLogs 获取指定工单的状态转换日志
func (tsm *TicketStatusManager) GetTransitionLogs(ticketID int) []StatusTransition {
	var logs []StatusTransition
	for _, log := range tsm.transitionLogs {
		if log.TicketID == ticketID {
			logs = append(logs, log)
		}
	}
	return logs
}

// GetAllTransitionLogs 获取所有状态转换日志
func (tsm *TicketStatusManager) GetAllTransitionLogs() []StatusTransition {
	return tsm.transitionLogs
}

// GetValidTransitions 获取指定状态的所有合法转换目标
func (tsm *TicketStatusManager) GetValidTransitions(status string) []string {
	validStatuses, exists := tsm.validTransitions[TicketStatus(status)]
	if !exists {
		return []string{}
	}
	
	result := make([]string, len(validStatuses))
	for i, status := range validStatuses {
		result[i] = string(status)
	}
	return result
}

// IsValidStatus 检查状态是否有效
func (tsm *TicketStatusManager) IsValidStatus(status string) bool {
	_, exists := tsm.validTransitions[TicketStatus(status)]
	return exists
}