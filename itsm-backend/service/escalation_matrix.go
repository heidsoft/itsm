package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EscalationLevel 单个升级级别配置
//
// 表示"超过 AfterMinutes 分钟后，通知 NotifyRoles / NotifyUserIDs"。
// AfterMinutes 为 0 表示立即升级（与创建时间同时）。
type EscalationLevel struct {
	Level         int      `json:"level"`         // 升级级别编号（1, 2, 3...）
	AfterMinutes  int      `json:"afterMinutes"`  // 距离告警开始多少分钟后触发
	NotifyRoles   []string `json:"notifyRoles"`   // 通知角色名（manager/team_lead/director）
	NotifyUserIDs []int    `json:"notifyUserIDs"` // 通知具体用户 ID
	Description   string   `json:"description"`   // 描述（用于审计/通知内容）
}

// EscalationMatrix 优先级 → 升级级别链
//
// 例如：
//   "critical" → [15min TeamLead, 30min Manager, 60min Director]
//   "high"     → [60min TeamLead, 240min Manager]
//   "medium"   → [240min TeamLead]
type EscalationMatrix map[string][]EscalationLevel

// DefaultEscalationMatrix 默认升级矩阵（与 SLA 模板保持一致）
//
// 优先级约定（与 Ticket.Priority 对齐）：critical / high / medium / low
var DefaultEscalationMatrix = EscalationMatrix{
	"critical": {
		{Level: 1, AfterMinutes: 15, NotifyRoles: []string{"team_lead"}, Description: "15分钟未响应升级至 Team Lead"},
		{Level: 2, AfterMinutes: 30, NotifyRoles: []string{"manager"}, Description: "30分钟未响应升级至 Manager"},
		{Level: 3, AfterMinutes: 60, NotifyRoles: []string{"director"}, Description: "60分钟未响应升级至 Director"},
	},
	"high": {
		{Level: 1, AfterMinutes: 60, NotifyRoles: []string{"team_lead"}, Description: "1小时未响应升级至 Team Lead"},
		{Level: 2, AfterMinutes: 240, NotifyRoles: []string{"manager"}, Description: "4小时未响应升级至 Manager"},
	},
	"medium": {
		{Level: 1, AfterMinutes: 240, NotifyRoles: []string{"team_lead"}, Description: "4小时未响应升级至 Team Lead"},
	},
	"low": {
		{Level: 1, AfterMinutes: 480, NotifyRoles: []string{"team_lead"}, Description: "8小时未响应升级至 Team Lead"},
	},
}

// EscalationMatrixService 升级矩阵服务
//
// 内存缓存每个租户的升级矩阵，缺省使用 DefaultEscalationMatrix。
// 未来可扩展：每个租户自定义矩阵（通过 DB 存储）。
type EscalationMatrixService struct {
	logger *zap.SugaredLogger

	mu      sync.RWMutex
	cache   map[int]EscalationMatrix
}

// NewEscalationMatrixService 创建升级矩阵服务
func NewEscalationMatrixService(logger *zap.SugaredLogger) *EscalationMatrixService {
	return &EscalationMatrixService{
		logger: logger,
		cache:  make(map[int]EscalationMatrix),
	}
}

// GetMatrix 获取指定租户的升级矩阵
//
// 实现：
//  1. 命中缓存 → 直接返回
//  2. 未命中 → 返回 DefaultEscalationMatrix 的拷贝并写入缓存
func (s *EscalationMatrixService) GetMatrix(tenantID int) EscalationMatrix {
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
	matrix := make(EscalationMatrix, len(DefaultEscalationMatrix))
	for priority, levels := range DefaultEscalationMatrix {
		matrix[priority] = append([]EscalationLevel(nil), levels...)
	}
	s.cache[tenantID] = matrix
	return matrix
}

// SetMatrix 设置租户的自定义升级矩阵
func (s *EscalationMatrixService) SetMatrix(tenantID int, matrix EscalationMatrix) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[tenantID] = matrix
}

// InvalidateCache 清除租户的矩阵缓存（用于配置变更时刷新）
func (s *EscalationMatrixService) InvalidateCache(tenantID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, tenantID)
}

// FindNextEscalationLevel 查找下一个应触发的升级级别
//
// 输入：priority + 当前已升级到的最大级别
// 输出：下一个级别（若所有级别均已升级则返回 nil）
//
// 触发条件：levels[i].AfterMinutes <= elapsedMinutes
func (s *EscalationMatrixService) FindNextEscalationLevel(tenantID int, priority string, elapsedMinutes int, currentMaxLevel int) *EscalationLevel {
	matrix := s.GetMatrix(tenantID)
	levels, ok := matrix[priority]
	if !ok {
		// 未知 priority 降级到 medium
		levels, ok = matrix["medium"]
		if !ok {
			return nil
		}
	}
	for i := range levels {
		lvl := levels[i]
		if lvl.Level <= currentMaxLevel {
			continue // 已升级过
		}
		if elapsedMinutes >= lvl.AfterMinutes {
			out := lvl
			return &out
		}
	}
	return nil
}

// CachedMatrixSnapshot 返回当前缓存中所有租户的矩阵快照（用于调试）
func (s *EscalationMatrixService) CachedMatrixSnapshot() map[int]EscalationMatrix {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[int]EscalationMatrix, len(s.cache))
	for k, v := range s.cache {
		out[k] = v
	}
	return out
}

// EscalationHistoryEntry 升级历史记录（用于审计）
type EscalationHistoryEntry struct {
	TicketID       int       `json:"ticketId"`
	TicketNumber   string    `json:"ticketNumber"`
	Priority       string    `json:"priority"`
	Level          int       `json:"level"`
	NotifyRoles    []string  `json:"notifyRoles"`
	NotifyUserIDs  []int     `json:"notifyUserIds"`
	TriggeredAt    time.Time `json:"triggeredAt"`
	ElapsedMinutes int       `json:"elapsedMinutes"`
	Reason         string    `json:"reason"`
}

// EscalationHistoryRecorder 升级历史记录接口
//
// 用于解耦：EscalationMatrixService 不直接依赖 ent.Client。
// 调用方实现该接口后可传入（生产环境用 SLAAlertHistory，测试用 mock）。
type EscalationHistoryRecorder interface {
	RecordEscalation(ctx context.Context, entry EscalationHistoryEntry) error
}

// InMemoryEscalationHistoryRecorder 内存版升级历史记录器（用于测试 & 开发）
type InMemoryEscalationHistoryRecorder struct {
	mu      sync.RWMutex
	entries []EscalationHistoryEntry
}

// NewInMemoryEscalationHistoryRecorder 创建内存版记录器
func NewInMemoryEscalationHistoryRecorder() *InMemoryEscalationHistoryRecorder {
	return &InMemoryEscalationHistoryRecorder{}
}

// RecordEscalation 记录升级历史
func (r *InMemoryEscalationHistoryRecorder) RecordEscalation(ctx context.Context, entry EscalationHistoryEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, entry)
	return nil
}

// Entries 返回所有历史条目
func (r *InMemoryEscalationHistoryRecorder) Entries() []EscalationHistoryEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]EscalationHistoryEntry, len(r.entries))
	copy(out, r.entries)
	return out
}

// String 格式化打印（用于日志）
func (m EscalationMatrix) String(priority string) string {
	levels, ok := m[priority]
	if !ok {
		return fmt.Sprintf("priority=%s: <undefined>", priority)
	}
	out := fmt.Sprintf("priority=%s: [", priority)
	for i, l := range levels {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("L%d(%dmin→%v)", l.Level, l.AfterMinutes, l.NotifyRoles)
	}
	out += "]"
	return out
}