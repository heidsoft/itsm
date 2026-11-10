package ai

import "itsm-backend/dto"

// SkillRegistrar 是内置技能注册所需的窄契约。
//
// 共享层具体注册表隐式满足本接口：Register 的入参 Skill 在两边是同一个
// dto 接口（service 侧为别名），因此组合根无需改动装配代码。
type SkillRegistrar interface {
	Register(skill dto.Skill) error
}
