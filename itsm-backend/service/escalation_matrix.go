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
	NotifyRoles   []string `json:"notifyRoles"`   // 通知角色码，必须取自主角色词表 domain/role.All（解析按 users.role 精确匹配）
	NotifyUserIDs []int    `json:"notifyUserIDs"` // 通知具体用户 ID
	Description   string   `json:"description"`   // 描述（用于审计/通知内容）
}

// EscalationMatrix 优先级 → 升级级别链
//
// 例如：
//
//	"critical" → [15min Manager, 30min ITAdmin, 60min Admin]
//	"high"     → [60min Manager, 240min ITAdmin]
//	"medium"   → [240min Manager]
type EscalationMatrix map[string][]EscalationLevel

// DefaultEscalationMatrix 默认升级矩阵（与 SLA 模板保持一致）
//
// 优先级约定（与 Ticket.Priority 对齐）：critical / high / medium / low
// 升级目标只使用 users.role 主角色码，因为 resolveNotifyUsers 按主角色精确查人。
var DefaultEscalationMatrix = EscalationMatrix{
	"critical": {
		{Level: 1, AfterMinutes: 15, NotifyRoles: []string{"manager"}, Description: "15分钟未响应升级至部门经理"},
		{Level: 2, AfterMinutes: 30, NotifyRoles: []string{"it_admin"}, Description: "30分钟未响应升级至IT管理员"},
		{Level: 3, AfterMinutes: 60, NotifyRoles: []string{"admin"}, Description: "60分钟未响应升级至租户管理员"},
	},
	"high": {
		{Level: 1, AfterMinutes: 60, NotifyRoles: []string{"manager"}, Description: "1小时未响应升级至部门经理"},
		{Level: 2, AfterMinutes: 240, NotifyRoles: []string{"it_admin"}, Description: "4小时未响应升级至IT管理员"},
	},
	"medium": {
		{Level: 1, AfterMinutes: 240, NotifyRoles: []string{"manager"}, Description: "4小时未响应升级至部门经理"},
	},
	"low": {
		{Level: 1, AfterMinutes: 480, NotifyRoles: []string{"manager"}, Description: "8小时未响应升级至部门经理"},
	},
}

// EscalationMatrixStore 是升级矩阵的持久化契约。
//
// 由调用方（生产用 SystemConfig-backed 实现，测试用内存实现）注入，
// 使本服务不直接依赖 ent.Client，同时可测。
type EscalationMatrixStore interface {
	// Load 返回该租户已持久化的矩阵。未配置时返回 (nil, nil) —— 调用方据此回落默认矩阵。
	Load(ctx context.Context, tenantID int) (EscalationMatrix, error)
	// Save 覆盖写入该租户的矩阵。
	Save(ctx context.Context, tenantID int, matrix EscalationMatrix) error
	// Delete 清除该租户的自定义矩阵（回落默认）。
	Delete(ctx context.Context, tenantID int) error
}

// EscalationMatrixService 升级矩阵服务
//
// 持久化语义：自定义矩阵写入 EscalationMatrixStore（生产落 system_configs 表，
// key=escalation.matrix，category=escalation），内存 map 仅作**读缓存**。
// 读路径：缓存未命中 → store.Load → 仍未配置则回落 DefaultEscalationMatrix。
// 写路径：store.Save 成功后才更新缓存；store 未注入（测试/降级）时保持纯内存行为。
type EscalationMatrixService struct {
	logger *zap.SugaredLogger
	store  EscalationMatrixStore

	mu    sync.RWMutex
	cache map[int]EscalationMatrix
}

// NewEscalationMatrixService 创建升级矩阵服务（纯内存，仅测试与降级路径使用）
func NewEscalationMatrixService(logger *zap.SugaredLogger) *EscalationMatrixService {
	return &EscalationMatrixService{
		logger: logger,
		cache:  make(map[int]EscalationMatrix),
	}
}

// NewEscalationMatrixServiceWithStore 创建带持久化的升级矩阵服务（生产装配使用）
func NewEscalationMatrixServiceWithStore(logger *zap.SugaredLogger, store EscalationMatrixStore) *EscalationMatrixService {
	return &EscalationMatrixService{
		logger: logger,
		store:  store,
		cache:  make(map[int]EscalationMatrix),
	}
}

// cloneMatrix 深拷贝一份矩阵，避免调用方改动缓存内部切片。
//
// 通知目标为空的级别在这里归一成空切片：handler 直接把矩阵序列化进响应，
// nil 会变成 JSON null，而 `/api/v1/escalation-matrices` 的契约是数组，
// 前端每处取值都得再判一次 null。
func cloneMatrix(src EscalationMatrix) EscalationMatrix {
	out := make(EscalationMatrix, len(src))
	for priority, levels := range src {
		out[priority] = make([]EscalationLevel, len(levels))
		for i, l := range levels {
			if l.NotifyRoles == nil {
				l.NotifyRoles = []string{}
			}
			if l.NotifyUserIDs == nil {
				l.NotifyUserIDs = []int{}
			}
			out[priority][i] = l
		}
	}
	return out
}

// defaultMatrixCopy 返回默认矩阵的拷贝。
func defaultMatrixCopy() EscalationMatrix {
	return cloneMatrix(DefaultEscalationMatrix)
}

// GetMatrix 获取指定租户的升级矩阵。
//
// 读路径优先级：内存缓存 → 持久化 store → DefaultEscalationMatrix。
// 三级都拿不到时返回默认矩阵的拷贝（保证调用方永远拿到非 nil）。
//
// 一律返回**深拷贝**：调用方（handler 序列化、升级判定等）不应能通过改返回值
// 污染服务内部缓存——否则一次误改会让该租户后续所有升级判定读到脏配置。
func (s *EscalationMatrixService) GetMatrix(tenantID int) EscalationMatrix {
	s.mu.RLock()
	if m, ok := s.cache[tenantID]; ok {
		s.mu.RUnlock()
		return cloneMatrix(m)
	}
	s.mu.RUnlock()

	var matrix EscalationMatrix

	// store 为 nil 时直接回落默认（保持旧构造函数的纯内存语义）
	if s.store != nil {
		loaded, err := s.store.Load(context.Background(), tenantID)
		switch {
		case err != nil:
			// 持久化读取失败**不阻断**升级判定：降级到默认矩阵并记日志。
			// 升级是 SLA 兜底路径，因 DB 抖动而停摆比用默认矩阵更糟。
			s.logger.Warnf("读取升级矩阵失败，降级到默认矩阵: tenant=%d err=%v", tenantID, err)
			matrix = defaultMatrixCopy()
		case loaded != nil:
			matrix = loaded
		default:
			matrix = defaultMatrixCopy()
		}
	} else {
		matrix = defaultMatrixCopy()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// 双重检查：期间可能已被别的 goroutine 写入
	if m, ok := s.cache[tenantID]; ok {
		return cloneMatrix(m)
	}
	s.cache[tenantID] = matrix
	return cloneMatrix(matrix)
}

// SetMatrix 设置租户的自定义升级矩阵。
//
// 语义：**先落库成功，再更新缓存**。落库失败时缓存保持原值并返回错误，
// 避免出现"接口返回成功但重启后配置丢失"的假成功。
// store 为 nil（测试/降级）时保持纯内存写入。
func (s *EscalationMatrixService) SetMatrix(tenantID int, matrix EscalationMatrix) error {
	if s.store != nil {
		if err := s.store.Save(context.Background(), tenantID, matrix); err != nil {
			s.logger.Errorf("持久化升级矩阵失败: tenant=%d err=%v", tenantID, err)
			return err
		}
	}

	stored := cloneMatrix(matrix)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[tenantID] = stored
	return nil
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
// 返回深拷贝，避免调试代码误改服务内部缓存。
func (s *EscalationMatrixService) CachedMatrixSnapshot() map[int]EscalationMatrix {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[int]EscalationMatrix, len(s.cache))
	for k, v := range s.cache {
		out[k] = cloneMatrix(v)
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
