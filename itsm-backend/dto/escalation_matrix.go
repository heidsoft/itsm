package dto

import "fmt"

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
//
// 值类型下沉到 dto，使 handlers/escalation_matrix 的窄契约无需引用共享层；
// service 侧以类型别名保持公开 API 与类型身份不变。
type EscalationMatrix map[string][]EscalationLevel

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
