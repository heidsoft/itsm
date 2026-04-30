package service

import (
	"context"
	"fmt"
	"sync"
)

// Skill 技能接口
type Skill interface {
	// Name 技能名称
	Name() string
	// Execute 执行技能
	Execute(ctx context.Context, input interface{}) (interface{}, error)
	// Validate 验证输入
	Validate(input interface{}) error
	// Tags 获取技能标签
	Tags() []string
}

// SkillRegistry 技能注册中心
type SkillRegistry struct {
	skills    map[string]Skill
	tagIndex  map[string][]string // tag -> skill names
	mu        sync.RWMutex
}

// NewSkillRegistry 创建技能注册中心
func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills:   make(map[string]Skill),
		tagIndex: make(map[string][]string),
	}
}

// Register 注册技能
func (r *SkillRegistry) Register(skill Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := skill.Name()
	if _, exists := r.skills[name]; exists {
		return fmt.Errorf("skill %s already registered", name)
	}

	r.skills[name] = skill

	// 更新标签索引
	for _, tag := range skill.Tags() {
		r.tagIndex[tag] = append(r.tagIndex[tag], name)
	}

	return nil
}

// Get 获取技能
func (r *SkillRegistry) Get(name string) (Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, exists := r.skills[name]
	if !exists {
		return nil, fmt.Errorf("skill %s not found", name)
	}

	return skill, nil
}

// FindByTag 根据标签查找技能
func (r *SkillRegistry) FindByTag(tag string) ([]Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names, exists := r.tagIndex[tag]
	if !exists {
		return nil, fmt.Errorf("no skills found for tag %s", tag)
	}

	skills := make([]Skill, 0, len(names))
	for _, name := range names {
		skills = append(skills, r.skills[name])
	}

	return skills, nil
}

// List 所有技能名称
func (r *SkillRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}

	return names
}

// Unregister 注销技能
func (r *SkillRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	skill, exists := r.skills[name]
	if !exists {
		return fmt.Errorf("skill %s not found", name)
	}

	delete(r.skills, name)

	// 从标签索引中移除
	for _, tag := range skill.Tags() {
		names := r.tagIndex[tag]
		for i, n := range names {
			if n == name {
				r.tagIndex[tag] = append(names[:i], names[i+1:]...)
				break
			}
		}
	}

	return nil
}

// GlobalSkillRegistry 全局技能注册中心
var GlobalSkillRegistry *SkillRegistry

func init() {
	GlobalSkillRegistry = NewSkillRegistry()
}

// RegisterGlobalSkill 注册全局技能
func RegisterGlobalSkill(skill Skill) error {
	return GlobalSkillRegistry.Register(skill)
}

// GetGlobalSkill 获取全局技能
func GetGlobalSkill(name string) (Skill, error) {
	return GlobalSkillRegistry.Get(name)
}

// FindGlobalSkillsByTag 根据标签查找全局技能
func FindGlobalSkillsByTag(tag string) ([]Skill, error) {
	return GlobalSkillRegistry.FindByTag(tag)
}
