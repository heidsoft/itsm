package service

import (
	"context"
	"fmt"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/user"
	"sort"
	"time"
)

// TicketAssignmentService 工单智能分配和路由服务
type TicketAssignmentService struct {
	client *ent.Client
	logger interface{} // 可选：如果需要日志
}

// NewTicketAssignmentService 创建工单分配服务实例
func NewTicketAssignmentService(client *ent.Client) *TicketAssignmentService {
	return &TicketAssignmentService{
		client: client,
	}
}

// AssignmentRequest 分配请求
type AssignmentRequest struct {
	TicketID       int      `json:"ticket_id"`
	CategoryID     *int     `json:"category_id,omitempty"`
	Priority       string   `json:"priority"`
	RequiredSkills []string `json:"required_skills,omitempty"`
	TenantID       int      `json:"tenant_id"`
	AutoAssign     bool     `json:"auto_assign"`
	PreferredUser  *int     `json:"preferred_user,omitempty"`
}

// AssignmentResponse 分配响应
type AssignmentResponse struct {
	TicketID       int     `json:"ticket_id"`
	AssignedTo     *int    `json:"assigned_to,omitempty"`
	AssignmentType string  `json:"assignment_type"` // auto, manual, routing
	Reason         string  `json:"reason"`
	Score          float64 `json:"score,omitempty"`
	Alternatives   []int   `json:"alternatives,omitempty"`
}

// UserWorkload 用户工作负载信息
type UserWorkload struct {
	UserID        int           `json:"user_id"`
	Username      string        `json:"username"`
	ActiveTickets int           `json:"active_tickets"`
	TotalTickets  int           `json:"total_tickets"`
	AvgResolution time.Duration `json:"avg_resolution"`
	Skills        []string      `json:"skills"`
	Categories    []int         `json:"categories"`
	Score         float64       `json:"score"`
}

// RoutingRule 路由规则
type RoutingRule struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Priority   int      `json:"priority"`
	Conditions []string `json:"conditions"`
	Actions    []string `json:"actions"`
	IsActive   bool     `json:"is_active"`
	TenantID   int      `json:"tenant_id"`
}

// AssignTicket 智能分配工单
func (s *TicketAssignmentService) AssignTicket(ctx context.Context, req *AssignmentRequest) (*AssignmentResponse, error) {
	// 1. 验证工单是否存在
	_, err := s.client.Ticket.Get(ctx, req.TicketID)
	if err != nil {
		return nil, fmt.Errorf("获取工单失败: %w", err)
	}

	// 2. 如果指定了首选用户，直接分配
	if req.PreferredUser != nil {
		return s.assignToSpecificUser(ctx, req, *req.PreferredUser)
	}

	// 3. 自动分配
	if req.AutoAssign {
		return s.autoAssignTicket(ctx, req)
	}

	// 4. 基于路由规则分配
	return s.routeTicket(ctx, req)
}

// autoAssignTicket 自动分配工单
func (s *TicketAssignmentService) autoAssignTicket(ctx context.Context, req *AssignmentRequest) (*AssignmentResponse, error) {
	// 1. 获取可用的处理人
	availableUsers, err := s.getAvailableUsers(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(availableUsers) == 0 {
		return &AssignmentResponse{
			TicketID:       req.TicketID,
			AssignmentType: "auto",
			Reason:         "没有可用的处理人",
		}, nil
	}

	// 2. 计算每个用户的评分
	for i := range availableUsers {
		availableUsers[i].Score = s.calculateUserScore(ctx, &availableUsers[i], req)
	}

	// 3. 按评分排序，选择最高分的用户
	sort.Slice(availableUsers, func(i, j int) bool {
		return availableUsers[i].Score > availableUsers[j].Score
	})

	bestUser := availableUsers[0]

	// 4. 执行分配
	err = s.client.Ticket.UpdateOneID(req.TicketID).
		SetAssigneeID(bestUser.UserID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("分配工单失败: %w", err)
	}

	return &AssignmentResponse{
		TicketID:       req.TicketID,
		AssignedTo:     &bestUser.UserID,
		AssignmentType: "auto",
		Reason:         fmt.Sprintf("基于技能匹配和工作负载自动分配，评分: %.2f", bestUser.Score),
		Score:          bestUser.Score,
		Alternatives:   s.getAlternativeUserIDs(availableUsers[1:5]), // 前5个备选
	}, nil
}

// getAvailableUsers 获取可用的处理人
func (s *TicketAssignmentService) getAvailableUsers(ctx context.Context, req *AssignmentRequest) ([]UserWorkload, error) {
	// 1. 获取所有活跃用户
	users, err := s.client.User.Query().
		Where(user.TenantIDEQ(req.TenantID)).
		Where(user.ActiveEQ(true)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	var availableUsers []UserWorkload

	for _, u := range users {
		// 2. 检查用户技能匹配
		if !s.checkUserSkills(ctx, u, req.RequiredSkills) {
			continue
		}

		// 3. 检查用户分类权限
		if req.CategoryID != nil && !s.checkUserCategoryAccess(ctx, u.ID, *req.CategoryID) {
			continue
		}

		// 4. 获取用户工作负载
		workload, err := s.getUserWorkload(ctx, u.ID)
		if err != nil {
			continue
		}

		// 5. 检查工作负载是否超限
		if workload.ActiveTickets >= s.getMaxActiveTickets(u.ID, req.Priority) {
			continue
		}

		availableUsers = append(availableUsers, *workload)
	}

	return availableUsers, nil
}

// CalculateUserScore 计算用户评分（公开方法）
func (s *TicketAssignmentService) CalculateUserScore(ctx context.Context, user *UserWorkload, req *AssignmentRequest) float64 {
	return s.calculateUserScore(ctx, user, req)
}

// calculateUserScore 计算用户评分
func (s *TicketAssignmentService) calculateUserScore(ctx context.Context, user *UserWorkload, req *AssignmentRequest) float64 {
	var score float64

	// 1. 技能匹配度 (40%)
	skillScore := s.calculateSkillScore(user.Skills, req.RequiredSkills)
	score += skillScore * 0.4

	// 2. 工作负载评分 (30%)
	workloadScore := s.calculateWorkloadScore(user.ActiveTickets, user.AvgResolution)
	score += workloadScore * 0.3

	// 3. 分类经验评分 (20%)
	categoryScore := s.calculateCategoryExperienceScore(ctx, user.UserID, req.CategoryID)
	score += categoryScore * 0.2

	// 4. 历史表现评分 (10%)
	performanceScore := s.calculatePerformanceScore(ctx, user.UserID)
	score += performanceScore * 0.1

	return score
}

// calculateSkillScore 计算技能匹配度
func (s *TicketAssignmentService) calculateSkillScore(userSkills, requiredSkills []string) float64 {
	if len(requiredSkills) == 0 {
		return 1.0
	}

	matched := 0
	for _, required := range requiredSkills {
		for _, userSkill := range userSkills {
			if required == userSkill {
				matched++
				break
			}
		}
	}

	return float64(matched) / float64(len(requiredSkills))
}

// calculateWorkloadScore 计算工作负载评分
func (s *TicketAssignmentService) calculateWorkloadScore(activeTickets int, avgResolution time.Duration) float64 {
	// 工作负载越少，评分越高
	workloadScore := 1.0 - float64(activeTickets)/10.0 // 假设最大10个活跃工单
	if workloadScore < 0.1 {
		workloadScore = 0.1
	}

	// 平均解决时间越短，评分越高
	resolutionScore := 1.0 - avgResolution.Hours()/24.0 // 假设24小时为基准
	if resolutionScore < 0.1 {
		resolutionScore = 0.1
	}

	return (workloadScore + resolutionScore) / 2.0
}

// calculateCategoryExperienceScore 计算分类经验评分
func (s *TicketAssignmentService) calculateCategoryExperienceScore(ctx context.Context, userID int, categoryID *int) float64 {
	if categoryID == nil {
		return 0.5 // 默认中等评分
	}

	// 统计用户在该分类下处理的工单数量
	count, err := s.client.Ticket.Query().
		Where(ticket.AssigneeIDEQ(userID)).
		Where(ticket.CategoryIDEQ(*categoryID)).
		Where(ticket.StatusIn("resolved", "closed")).
		Count(ctx)
	if err != nil {
		return 0.5
	}

	// 根据处理数量计算评分
	if count >= 10 {
		return 1.0
	} else if count >= 5 {
		return 0.8
	} else if count >= 2 {
		return 0.6
	} else {
		return 0.4
	}
}

// calculatePerformanceScore 计算历史表现评分
func (s *TicketAssignmentService) calculatePerformanceScore(ctx context.Context, userID int) float64 {
	// 计算最近30天的SLA达成率
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	totalTickets, err := s.client.Ticket.Query().
		Where(ticket.AssigneeIDEQ(userID)).
		Where(ticket.CreatedAtGTE(thirtyDaysAgo)).
		Where(ticket.StatusIn("resolved", "closed")).
		Count(ctx)
	if err != nil || totalTickets == 0 {
		return 0.5
	}

	// 这里可以添加SLA检查逻辑，暂时返回默认值
	return 0.8
}

// getUserWorkload 获取用户工作负载
func (s *TicketAssignmentService) getUserWorkload(ctx context.Context, userID int) (*UserWorkload, error) {
	// 获取活跃工单数量
	activeTickets, err := s.client.Ticket.Query().
		Where(ticket.AssigneeIDEQ(userID)).
		Where(ticket.StatusIn("open", "in_progress", "pending")).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 获取总工单数量
	totalTickets, err := s.client.Ticket.Query().
		Where(ticket.AssigneeIDEQ(userID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 计算平均解决时间
	avgResolution, err := s.calculateAverageResolutionTime(ctx, userID)
	if err != nil {
		avgResolution = 24 * time.Hour // 默认24小时
	}

	// 获取用户技能（这里简化处理，实际应该从用户配置中获取）
	skills := []string{"general"} // 默认通用技能

	// 获取用户擅长的分类
	categories, err := s.getUserCategories(ctx, userID)
	if err != nil {
		categories = []int{}
	}

	return &UserWorkload{
		UserID:        userID,
		ActiveTickets: activeTickets,
		TotalTickets:  totalTickets,
		AvgResolution: avgResolution,
		Skills:        skills,
		Categories:    categories,
	}, nil
}

// calculateAverageResolutionTime 计算平均解决时间
func (s *TicketAssignmentService) calculateAverageResolutionTime(ctx context.Context, userID int) (time.Duration, error) {
	// 获取已解决的工单
	resolvedTickets, err := s.client.Ticket.Query().
		Where(ticket.AssigneeIDEQ(userID)).
		Where(ticket.StatusIn("resolved", "closed")).
		All(ctx)
	if err != nil {
		return 0, err
	}

	if len(resolvedTickets) == 0 {
		return 24 * time.Hour, nil
	}

	var totalDuration time.Duration
	count := 0

	for _, t := range resolvedTickets {
		duration := t.UpdatedAt.Sub(t.CreatedAt)
		totalDuration += duration
		count++
	}

	if count == 0 {
		return 24 * time.Hour, nil
	}

	return totalDuration / time.Duration(count), nil
}

// getUserCategories 获取用户擅长的分类
func (s *TicketAssignmentService) getUserCategories(ctx context.Context, userID int) ([]int, error) {
	// 统计用户处理过的分类
	// 这里简化处理，实际应该从用户配置中获取
	return []int{}, nil
}

// getMaxActiveTickets 获取最大活跃工单数
func (s *TicketAssignmentService) getMaxActiveTickets(userID int, priority string) int {
	// 根据优先级设置不同的最大活跃工单数
	switch priority {
	case "critical":
		return 3
	case "high":
		return 5
	case "medium":
		return 8
	case "low":
		return 12
	default:
		return 8
	}
}

// checkUserSkills 检查用户技能
func (s *TicketAssignmentService) checkUserSkills(ctx context.Context, user *ent.User, requiredSkills []string) bool {
	if len(requiredSkills) == 0 {
		return true
	}

	// 这里简化处理，实际应该从用户配置中获取技能
	// 暂时返回true，表示所有用户都有通用技能
	return true
}

// checkUserCategoryAccess 检查用户分类访问权限
func (s *TicketAssignmentService) checkUserCategoryAccess(ctx context.Context, userID, categoryID int) bool {
	// 这里简化处理，实际应该检查用户权限配置
	// 暂时返回true，表示所有用户都有访问权限
	return true
}

// assignToSpecificUser 分配给指定用户
func (s *TicketAssignmentService) assignToSpecificUser(ctx context.Context, req *AssignmentRequest, userID int) (*AssignmentResponse, error) {
	// 检查用户是否存在且可用
	userEntity, err := s.client.User.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	if !userEntity.Active {
		return nil, fmt.Errorf("用户状态不可用: 非活跃状态")
	}

	// 执行分配
	err = s.client.Ticket.UpdateOneID(req.TicketID).
		SetAssigneeID(userID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("分配工单失败: %w", err)
	}

	return &AssignmentResponse{
		TicketID:       req.TicketID,
		AssignedTo:     &userID,
		AssignmentType: "manual",
		Reason:         "手动指定分配",
	}, nil
}

// routeTicket 基于路由规则分配工单
func (s *TicketAssignmentService) routeTicket(ctx context.Context, req *AssignmentRequest) (*AssignmentResponse, error) {
	// 这里实现基于规则的路由逻辑
	// 暂时返回自动分配结果
	return s.autoAssignTicket(ctx, req)
}

// getAlternativeUserIDs 获取备选用户ID列表
func (s *TicketAssignmentService) getAlternativeUserIDs(users []UserWorkload) []int {
	var ids []int
	for _, u := range users {
		ids = append(ids, u.UserID)
	}
	return ids
}

// GetUserWorkload 获取用户工作负载信息
func (s *TicketAssignmentService) GetUserWorkload(ctx context.Context, userID int) (*UserWorkload, error) {
	return s.getUserWorkload(ctx, userID)
}

// GetTeamWorkload 获取团队工作负载信息
func (s *TicketAssignmentService) GetTeamWorkload(ctx context.Context, tenantID int) ([]UserWorkload, error) {
	users, err := s.client.User.Query().
		Where(user.TenantIDEQ(tenantID)).
		Where(user.ActiveEQ(true)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var workloads []UserWorkload
	for _, u := range users {
		workload, err := s.getUserWorkload(ctx, u.ID)
		if err != nil {
			continue
		}
		workload.Username = u.Username
		workloads = append(workloads, *workload)
	}

	return workloads, nil
}

// ReassignTicket 重新分配工单
func (s *TicketAssignmentService) ReassignTicket(ctx context.Context, ticketID int, newAssigneeID int, reason string) error {
	// 更新工单分配人
	err := s.client.Ticket.UpdateOneID(ticketID).
		SetAssigneeID(newAssigneeID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("重新分配失败: %w", err)
	}

	// 这里可以添加分配历史记录和通知逻辑
	return nil
}

// LoadBalance 负载均衡
func (s *TicketAssignmentService) LoadBalance(ctx context.Context, tenantID int) error {
	// 获取所有活跃工单
	_, err := s.client.Ticket.Query().
		Where(ticket.StatusIn("open", "in_progress", "pending")).
		Where(ticket.TenantIDEQ(tenantID)).
		All(ctx)
	if err != nil {
		return err
	}

	// 获取团队工作负载
	workloads, err := s.GetTeamWorkload(ctx, tenantID)
	if err != nil {
		return err
	}

	// 计算平均负载
	var totalLoad int
	for _, w := range workloads {
		totalLoad += w.ActiveTickets
	}
	avgLoad := totalLoad / len(workloads)

	// 重新分配超载用户的工单
	for _, w := range workloads {
		if w.ActiveTickets > avgLoad+2 { // 超过平均负载2个工单
			// 找到负载较低的用户
			var targetUser *UserWorkload
			for _, candidate := range workloads {
				if candidate.UserID != w.UserID && candidate.ActiveTickets < avgLoad {
					if targetUser == nil || candidate.ActiveTickets < targetUser.ActiveTickets {
						targetUser = &candidate
					}
				}
			}

			if targetUser != nil {
				// 重新分配一个工单
				err := s.ReassignTicket(ctx, 0, targetUser.UserID, "负载均衡自动重新分配")
				if err != nil {
					continue
				}
			}
		}
	}

	return nil
}
