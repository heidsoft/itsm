package service

import (
	"context"
	"sort"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// SkillMatchingService 技能匹配服务
type SkillMatchingService struct {
	client *ent.Client
}

// NewSkillMatchingService 创建技能匹配服务
func NewSkillMatchingService(client *ent.Client) *SkillMatchingService {
	return &SkillMatchingService{client: client}
}

// CreateEngineerSkill 创建工程师技能
func (s *SkillMatchingService) CreateEngineerSkill(ctx context.Context, input dto.CreateEngineerSkillRequest) (*ent.EngineerSkill, error) {
	build := s.client.EngineerSkill.Create().
		SetUserID(input.UserID).
		SetCategory(input.Category).
		SetSkillName(input.SkillName).
		SetProficiencyLevel(input.ProficiencyLevel).
		SetExperienceYears(input.ExperienceYears).
		SetCertifications(input.Certifications).
		SetIsAvailable(input.IsAvailable).
		SetCurrentLoad(input.CurrentLoad).
		SetMaxLoad(input.MaxLoad).
		SetWorkingHours(input.WorkingHours).
		SetTenantID(input.TenantID)

	if input.PreferredShift != nil && *input.PreferredShift != "" {
		build.SetPreferredShift(*input.PreferredShift)
	}

	return build.Save(ctx)
}

// GetEngineerSkillByID 获取工程师技能
func (s *SkillMatchingService) GetEngineerSkillByID(ctx context.Context, id int) (*ent.EngineerSkill, error) {
	return s.client.EngineerSkill.Get(ctx, id)
}

// QueryEngineerSkills 查询工程师技能列表
func (s *SkillMatchingService) QueryEngineerSkills(ctx context.Context, tenantID int) ([]*ent.EngineerSkill, error) {
	all, err := s.client.EngineerSkill.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*ent.EngineerSkill
	for _, e := range all {
		if e.TenantID == tenantID {
			result = append(result, e)
		}
	}
	return result, nil
}

// GetEngineerSkillsByUserID 根据用户ID获取技能列表
func (s *SkillMatchingService) GetEngineerSkillsByUserID(ctx context.Context, userID, tenantID int) ([]*ent.EngineerSkill, error) {
	all, err := s.client.EngineerSkill.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*ent.EngineerSkill
	for _, e := range all {
		if e.UserID == userID && e.TenantID == tenantID {
			result = append(result, e)
		}
	}
	return result, nil
}

// UpdateEngineerSkill 更新工程师技能
func (s *SkillMatchingService) UpdateEngineerSkill(ctx context.Context, id int, input dto.UpdateEngineerSkillRequest) (*ent.EngineerSkill, error) {
	update := s.client.EngineerSkill.UpdateOneID(id)
	if input.Category != nil {
		update.SetCategory(*input.Category)
	}
	if input.SkillName != nil {
		update.SetSkillName(*input.SkillName)
	}
	if input.ProficiencyLevel != nil {
		update.SetProficiencyLevel(*input.ProficiencyLevel)
	}
	if input.ExperienceYears != nil {
		update.SetExperienceYears(*input.ExperienceYears)
	}
	if input.Certifications != nil {
		update.SetCertifications(*input.Certifications)
	}
	if input.IsAvailable != nil {
		update.SetIsAvailable(*input.IsAvailable)
	}
	if input.CurrentLoad != nil {
		update.SetCurrentLoad(*input.CurrentLoad)
	}
	if input.MaxLoad != nil {
		update.SetMaxLoad(*input.MaxLoad)
	}
	if input.PreferredShift != nil {
		update.SetPreferredShift(*input.PreferredShift)
	}
	if input.WorkingHours != nil {
		update.SetWorkingHours(*input.WorkingHours)
	}
	return update.Save(ctx)
}

// DeleteEngineerSkill 删除工程师技能
func (s *SkillMatchingService) DeleteEngineerSkill(ctx context.Context, id int) error {
	return s.client.EngineerSkill.DeleteOneID(id).Exec(ctx)
}

// MatchBestEngineer 根据工单属性匹配最佳工程师
// 匹配因素: 技能匹配度 > 负载情况 > 经验年限
func (s *SkillMatchingService) MatchBestEngineer(ctx context.Context, tenantID int, category, skillName string, priority string) (*ent.EngineerSkill, error) {
	// 查询所有可用工程师
	all, err := s.client.EngineerSkill.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var engineers []*ent.EngineerSkill
	for _, e := range all {
		if e.TenantID == tenantID && e.IsAvailable {
			engineers = append(engineers, e)
		}
	}

	if len(engineers) == 0 {
		return nil, nil
	}

	// 过滤有匹配技能的工程师
	var matchedEngineers []*EngineerScore
	for _, e := range engineers {
		if s.matchSkill(e, category, skillName) {
			score := s.calculateScore(e, priority)
			matchedEngineers = append(matchedEngineers, &EngineerScore{
				Engineer: e,
				Score:    score,
			})
		}
	}

	if len(matchedEngineers) == 0 {
		// 没有精确匹配，返回负载最低的工程师
		return s.getLeastLoadedEngineer(engineers), nil
	}

	// 按分数排序
	sort.Slice(matchedEngineers, func(i, j int) bool {
		return matchedEngineers[i].Score > matchedEngineers[j].Score
	})

	return matchedEngineers[0].Engineer, nil
}

// EngineerScore 工程师得分
type EngineerScore struct {
	Engineer *ent.EngineerSkill
	Score    int
}

// matchSkill 检查技能是否匹配
func (s *SkillMatchingService) matchSkill(e *ent.EngineerSkill, category, skillName string) bool {
	if e.Category == category {
		if skillName == "" || e.SkillName == skillName {
			return true
		}
	}
	return false
}

// calculateScore 计算工程师得分
// 分数 = 技能熟练度 * 30 + (最大负载 - 当前负载) * 20 + 经验年限 * 10
func (s *SkillMatchingService) calculateScore(e *ent.EngineerSkill, priority string) int {
	skillScore := e.ProficiencyLevel * 30
	loadScore := (e.MaxLoad - e.CurrentLoad) * 20
	experienceScore := e.ExperienceYears * 10

	// 优先级加权
	priorityBonus := 0
	switch priority {
	case "critical":
		priorityBonus = 50
	case "high":
		priorityBonus = 30
	case "medium":
		priorityBonus = 10
	}

	return skillScore + loadScore + experienceScore + priorityBonus
}

// getLeastLoadedEngineer 获取负载最低的工程师
func (s *SkillMatchingService) getLeastLoadedEngineer(engineers []*ent.EngineerSkill) *ent.EngineerSkill {
	if len(engineers) == 0 {
		return nil
	}

	sort.Slice(engineers, func(i, j int) bool {
		return engineers[i].CurrentLoad < engineers[j].CurrentLoad
	})

	return engineers[0]
}

// UpdateEngineerLoad 更新工程师负载
func (s *SkillMatchingService) UpdateEngineerLoad(ctx context.Context, userID, tenantID int, delta int) error {
	skills, err := s.GetEngineerSkillsByUserID(ctx, userID, tenantID)
	if err != nil {
		return err
	}

	for _, skill := range skills {
		newLoad := skill.CurrentLoad + delta
		if newLoad < 0 {
			newLoad = 0
		}
		_, err := s.client.EngineerSkill.UpdateOneID(skill.ID).
			SetCurrentLoad(newLoad).
			Save(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetAvailableEngineers 获取可用工程师列表(按技能分类)
func (s *SkillMatchingService) GetAvailableEngineers(ctx context.Context, tenantID int, category string) ([]*ent.EngineerSkill, error) {
	all, err := s.client.EngineerSkill.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*ent.EngineerSkill
	for _, e := range all {
		if e.TenantID == tenantID && e.IsAvailable {
			if category == "" || e.Category == category {
				result = append(result, e)
			}
		}
	}
	return result, nil
}

// CheckEngineerAvailability 检查工程师是否可接单
func (s *SkillMatchingService) CheckEngineerAvailability(ctx context.Context, userID, tenantID int) (bool, error) {
	skills, err := s.GetEngineerSkillsByUserID(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}

	if len(skills) == 0 {
		return false, nil
	}

	// 只要有一个技能可用且未达到最大负载就可以接单
	for _, skill := range skills {
		if skill.IsAvailable && skill.CurrentLoad < skill.MaxLoad {
			return true, nil
		}
	}

	return false, nil
}
