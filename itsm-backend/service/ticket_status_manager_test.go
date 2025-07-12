package service

import (
	"context"
	"testing"
)

func TestTicketStatusManager(t *testing.T) {
	// 创建状态管理器实例
	manager := NewTicketStatusManager(nil)

	// 测试合法的状态转换
	tests := []struct {
		name     string
		from     string
		to       string
		expected bool
	}{
		{"新建到处理中", "new", "in_progress", true},
		{"处理中到待审批", "in_progress", "pending", true},
		{"待审批到已解决", "pending", "resolved", true},
		{"已解决到已关闭", "resolved", "closed", true},
		{"任意状态到退回", "closed", "returned", true},
		{"退回到新建", "returned", "new", true},
		{"非法转换：新建到已关闭", "new", "closed", false},
		{"非法转换：处理中到已关闭", "in_progress", "closed", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateStatusTransition(tt.from, tt.to)
			if tt.expected && err != nil {
				t.Errorf("期望转换成功，但得到错误: %v", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("期望转换失败，但转换成功了")
			}
		})
	}
}

func TestStatusTransitionLogging(t *testing.T) {
	manager := NewTicketStatusManager(nil)
	ctx := context.Background()

	// 执行状态转换并记录日志
	err := manager.TransitionTicketStatus(ctx, 1, "new", "in_progress", 100, "开始处理工单")
	if err != nil {
		t.Errorf("状态转换失败: %v", err)
	}

	// 检查日志是否记录
	logs := manager.GetTransitionLogs(1)
	if len(logs) != 1 {
		t.Errorf("期望1条日志记录，实际得到%d条", len(logs))
	}

	log := logs[0]
	if log.FromStatus != StatusNew {
		t.Errorf("期望源状态为new，实际为%s", log.FromStatus)
	}
	if log.ToStatus != StatusInProgress {
		t.Errorf("期望目标状态为in_progress，实际为%s", log.ToStatus)
	}
	if log.UserID != 100 {
		t.Errorf("期望用户ID为100，实际为%d", log.UserID)
	}
}

func TestGetValidTransitions(t *testing.T) {
	manager := NewTicketStatusManager(nil)

	// 测试获取合法转换
	validTransitions := manager.GetValidTransitions("new")
	expected := []string{"in_progress", "returned"}

	if len(validTransitions) != len(expected) {
		t.Errorf("期望%d个合法转换，实际得到%d个", len(expected), len(validTransitions))
	}

	for _, exp := range expected {
		found := false
		for _, actual := range validTransitions {
			if actual == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("期望找到转换状态%s，但未找到", exp)
		}
	}
}
