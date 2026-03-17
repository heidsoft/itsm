package service

import (
	"fmt"

	"itsm-backend/dto"
)

// MapProcessStatus 映射流程状态为标准化状态
// 这个函数在多个服务中重复，现在提取为公共方法
func MapProcessStatus(status string) dto.ProcessStatus {
	switch status {
	case "running", "active":
		return dto.ProcessStatusRunning
	case "completed":
		return dto.ProcessStatusCompleted
	case "suspended":
		return dto.ProcessStatusSuspended
	case "terminated", "cancelled":
		return dto.ProcessStatusTerminated
	default:
		return dto.ProcessStatusPending
	}
}

// MapProcessStatusToTicketStatus 将流程状态映射为工单状态
// 用于 TicketLifecycleService
func MapProcessStatusToTicketStatus(status string) string {
	switch status {
	case "running", "active":
		return "in_progress"
	case "completed":
		return "closed"
	case "suspended":
		return "pending"
	case "terminated", "cancelled":
		return "cancelled"
	default:
		return "open"
	}
}

// BuildBusinessKey 构建业务键
// 用于在不同服务中构建统一的业务键格式
// 例如: ticket:123, incident:456
func BuildBusinessKey(entityType string, entityID int) string {
	return fmt.Sprintf("%s:%d", entityType, entityID)
}
