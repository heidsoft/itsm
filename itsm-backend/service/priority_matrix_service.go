package service

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// PriorityMatrix ITIL标准优先级矩阵
// 横轴：urgency(紧急程度)，纵轴：impact(影响范围)
// 值：priority(优先级)
type PriorityMatrix map[string]map[string]string

// DefaultPriorityMatrix 默认ITIL标准优先级矩阵
var DefaultPriorityMatrix = PriorityMatrix{
	"low": {
		"low":      "low",
		"medium":   "low",
		"high":     "medium",
		"critical": "medium",
	},
	"medium": {
		"low":      "low",
		"medium":   "medium",
		"high":     "high",
		"critical": "high",
	},
	"high": {
		"low":      "medium",
		"medium":   "high",
		"high":     "high",
		"critical": "critical",
	},
	"critical": {
		"low":      "medium",
		"medium":   "high",
		"high":     "critical",
		"critical": "critical",
	},
}

// PriorityMatrixService 优先级矩阵服务
// 支持租户自定义优先级矩阵，内存缓存
type PriorityMatrixService struct {
	logger *zap.SugaredLogger

	mu    sync.RWMutex
	cache map[int]PriorityMatrix
}

// NewPriorityMatrixService 创建优先级矩阵服务
func NewPriorityMatrixService(logger *zap.SugaredLogger) *PriorityMatrixService {
	return &PriorityMatrixService{
		logger: logger,
		cache:  make(map[int]PriorityMatrix),
	}
}

// GetMatrix 获取指定租户的优先级矩阵
// 优先使用租户自定义矩阵，不存在则返回默认矩阵
func (s *PriorityMatrixService) GetMatrix(tenantID int) PriorityMatrix {
	s.mu.RLock()
	if m, ok := s.cache[tenantID]; ok {
		s.mu.RUnlock()
		return m
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查
	if m, ok := s.cache[tenantID]; ok {
		return m
	}

	// 复制默认矩阵
	matrix := make(PriorityMatrix, len(DefaultPriorityMatrix))
	for impact, urgencies := range DefaultPriorityMatrix {
		matrix[impact] = make(map[string]string, len(urgencies))
		for urgency, priority := range urgencies {
			matrix[impact][urgency] = priority
		}
	}

	s.cache[tenantID] = matrix
	return matrix
}

// SetMatrix 设置租户自定义优先级矩阵
func (s *PriorityMatrixService) SetMatrix(tenantID int, matrix PriorityMatrix) error {
	// 验证矩阵格式是否正确
	validValues := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	
	for impact, urgencies := range matrix {
		if !validValues[impact] {
			return fmt.Errorf("invalid impact value: %s", impact)
		}
		for urgency, priority := range urgencies {
			if !validValues[urgency] {
				return fmt.Errorf("invalid urgency value: %s", urgency)
			}
			if !validValues[priority] {
				return fmt.Errorf("invalid priority value: %s", priority)
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[tenantID] = matrix
	return nil
}

// InvalidateCache 清除租户矩阵缓存
func (s *PriorityMatrixService) InvalidateCache(tenantID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, tenantID)
}

// CalculatePriority 根据影响范围和紧急程度计算优先级
func (s *PriorityMatrixService) CalculatePriority(tenantID int, impact, urgency string) (string, error) {
	matrix := s.GetMatrix(tenantID)
	
	urgencyMap, ok := matrix[impact]
	if !ok {
		return "", fmt.Errorf("impact value %s not found in matrix", impact)
	}
	
	priority, ok := urgencyMap[urgency]
	if !ok {
		return "", fmt.Errorf("urgency value %s not found in matrix for impact %s", urgency, impact)
	}
	
	return priority, nil
}

// ValidatePriority 验证给定的优先级是否符合impact和urgency的计算结果
func (s *PriorityMatrixService) ValidatePriority(tenantID int, impact, urgency, priority string) (bool, error) {
	calculated, err := s.CalculatePriority(tenantID, impact, urgency)
	if err != nil {
		return false, err
	}
	return calculated == priority, nil
}
