package service

// test-coverage-guard: skip — 仅别名与薄委托，无逻辑。

import "itsm-backend/dto"

// BaseSkill 骨架与标识值类型的唯一实现在 dto 包；这里保留同名别名
// 与薄委托构造，既有调用点（含 SkillRegistry 的指标钩子）零改动。
type BaseSkill = dto.BaseSkill
type SkillIdentity = dto.SkillIdentity

func NewBaseSkill(code, name, version, category string, tags []string, requiredPermissions []string, capabilities []string) *BaseSkill {
	return dto.NewBaseSkill(code, name, version, category, tags, requiredPermissions, capabilities)
}
