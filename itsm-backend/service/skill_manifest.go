package service

import (
	"context"

	"itsm-backend/dto"
)

// Skill 值类型/接口的唯一定义在 dto 包；这里保留同名别名，
// 使共享层既有调用点与类型身份零改动。
type SkillManifest = dto.SkillManifest
type SkillMetrics = dto.SkillMetrics
type Skill = dto.Skill

// ExtendedSkill 扩展的Skill接口，包含市场和训练相关能力
// 所有新开发的技能都应该实现这个接口
type ExtendedSkill interface {
	Skill
	// Train 使用提供的数据集训练/微调技能
	Train(ctx context.Context, dataset interface{}) error
}
