package skill

import (
	"context"

	"itsm-backend/dto"
)

// SkillRegistryCatalog 是技能管理 API 对共享层注册表的真实消费面（窄契约）。
//
// 方法集 = handlers/skill 全部生产文件的调用并集：市场/管理端列表读
// ListEntriesScoped / ListEntriesByTagScoped / GetEntryScoped，注册与
// 更新用 RegisterScoped + GetScoped + UnregisterScoped，统一调用入口用
// InvokeWithMetrics(ctx, tenantID, code, input)。tenantID 必须进执行签名：
// 注册表进程级共享，执行入口若不按租户收口，handler 的 scoped 查找一旦失败，
// 仍能用全局 code 命中并跑掉别的租户的技能。共享层注册表隐式满足本接口
// （Skill/SkillEntry/SkillInvokeResult 在两侧是同一 dto 类型）。
type SkillRegistryCatalog interface {
	ListEntriesScoped(tenantID int) []dto.SkillEntry
	ListEntriesByTagScoped(tenantID int, tag string) []dto.SkillEntry
	GetEntryScoped(tenantID int, code string) (dto.SkillEntry, error)
	GetScoped(tenantID int, code string) (dto.Skill, error)
	RegisterScoped(tenantID int, skill dto.Skill) error
	UnregisterScoped(tenantID int, code string) error
	InvokeWithMetrics(ctx context.Context, tenantID int, code string, input interface{}) (dto.SkillInvokeResult, error)
}
