package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"time"

	"go.uber.org/zap"
)

type TicketRatingService struct {
	client              *ent.Client
	logger              *zap.SugaredLogger
	notificationService *TicketNotificationService // 可选的通知服务
}

func NewTicketRatingService(client *ent.Client, logger *zap.SugaredLogger) *TicketRatingService {
	return &TicketRatingService{
		client: client,
		logger: logger,
	}
}

// SetNotificationService 设置通知服务（用于依赖注入）
func (s *TicketRatingService) SetNotificationService(notificationService *TicketNotificationService) {
	s.notificationService = notificationService
}

// SubmitRating 提交工单评分
func (s *TicketRatingService) SubmitRating(
	ctx context.Context,
	ticketID int,
	req *dto.SubmitTicketRatingRequest,
	ratedBy int,
	tenantID int,
) (*dto.TicketRatingResponse, error) {
	s.logger.Infow("Submitting ticket rating", "ticket_id", ticketID, "rating", req.Rating, "rated_by", ratedBy)

	// 验证工单是否存在且属于当前租户
	ticketEntity, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	// 验证工单状态（只有已解决或已关闭的工单才能评分）
	if ticketEntity.Status != "resolved" && ticketEntity.Status != "closed" {
		return nil, fmt.Errorf("只能对已解决或已关闭的工单进行评分")
	}

	// 验证评分人是否为申请人
	if ticketEntity.RequesterID != ratedBy {
		return nil, fmt.Errorf("只有申请人可以评分")
	}

	// 检查是否已经评分过
	if ticketEntity.Rating > 0 {
		return nil, fmt.Errorf("该工单已经评分过了")
	}

	// 更新工单评分
	now := time.Now()
	updatedTicket, err := s.client.Ticket.UpdateOneID(ticketID).
		SetRating(req.Rating).
		SetNillableRatingComment(&req.Comment).
		SetNillableRatedAt(&now).
		SetRatedBy(ratedBy).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to submit rating", "error", err)
		return nil, fmt.Errorf("failed to submit rating: %w", err)
	}

	// 获取评分人信息
	ratedByUser, _ := s.client.User.Get(ctx, ratedBy)

	// 发送通知给处理人
	if s.notificationService != nil && updatedTicket.AssigneeID > 0 {
		content := fmt.Sprintf("工单 #%s 收到了 %d 星评分", updatedTicket.TicketNumber, req.Rating)
		if req.Comment != "" {
			content += fmt.Sprintf("，评论：%s", req.Comment)
		}
		if err := s.notificationService.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
			UserIDs: []int{updatedTicket.AssigneeID},
			Type:    "rating",
			Channel: "in_app",
			Content: content,
		}, tenantID); err != nil {
			s.logger.Warnw("Failed to send rating notification", "error", err)
		}
	}

	var ratedAt *time.Time
	if !updatedTicket.RatedAt.IsZero() {
		ratedAt = &updatedTicket.RatedAt
	}
	response := &dto.TicketRatingResponse{
		Rating:  updatedTicket.Rating,
		Comment: updatedTicket.RatingComment,
		RatedAt: ratedAt,
		RatedBy: updatedTicket.RatedBy,
	}
	if ratedByUser != nil {
		response.RatedByName = ratedByUser.Name
	}

	return response, nil
}

// GetRating 获取工单评分
func (s *TicketRatingService) GetRating(
	ctx context.Context,
	ticketID int,
	tenantID int,
) (*dto.TicketRatingResponse, error) {
	ticketEntity, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	if ticketEntity.Rating == 0 {
		return nil, nil // 未评分
	}

	// 获取评分人信息
	var ratedByName string
	if ticketEntity.RatedBy > 0 {
		ratedByUser, _ := s.client.User.Get(ctx, ticketEntity.RatedBy)
		if ratedByUser != nil {
			ratedByName = ratedByUser.Name
		}
	}

	var ratedAt *time.Time
	if !ticketEntity.RatedAt.IsZero() {
		ratedAt = &ticketEntity.RatedAt
	}

	return &dto.TicketRatingResponse{
		Rating:      ticketEntity.Rating,
		Comment:     ticketEntity.RatingComment,
		RatedAt:     ratedAt,
		RatedBy:     ticketEntity.RatedBy,
		RatedByName: ratedByName,
	}, nil
}

// GetRatingStats 获取评分统计
func (s *TicketRatingService) GetRatingStats(
	ctx context.Context,
	req *dto.GetRatingStatsRequest,
) (*dto.RatingStatsResponse, error) {
	query := s.client.Ticket.Query().
		Where(
			ticket.TenantID(req.TenantID),
			ticket.RatingGT(0), // 只统计已评分的工单
		)

	// 添加筛选条件
	if req.AssigneeID != nil {
		query = query.Where(ticket.AssigneeID(*req.AssigneeID))
	}
	if req.CategoryID != nil {
		query = query.Where(ticket.CategoryID(*req.CategoryID))
	}
	if req.StartDate != nil {
		startDate, err := time.Parse("2006-01-02", *req.StartDate)
		if err == nil {
			query = query.Where(ticket.RatedAtGTE(startDate))
		}
	}
	if req.EndDate != nil {
		endDate, err := time.Parse("2006-01-02", *req.EndDate)
		if err == nil {
			// 结束日期应该是当天的23:59:59
			endDate = endDate.Add(24*time.Hour - time.Second)
			query = query.Where(ticket.RatedAtLTE(endDate))
		}
	}

	// 获取所有已评分的工单
	ratedTickets, err := query.All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get rated tickets", "error", err)
		return nil, fmt.Errorf("failed to get rated tickets: %w", err)
	}

	// 计算统计信息
	totalRatings := len(ratedTickets)
	if totalRatings == 0 {
		return &dto.RatingStatsResponse{
			TotalRatings:      0,
			AverageRating:     0,
			RatingDistribution: make(map[int]int),
		}, nil
	}

	// 计算总分和平均分
	totalScore := 0
	ratingDistribution := make(map[int]int)
	for _, t := range ratedTickets {
		totalScore += t.Rating
		ratingDistribution[t.Rating]++
	}
	averageRating := float64(totalScore) / float64(totalRatings)

	// 按处理人统计
	assigneeStats := make(map[int]*dto.AssigneeRatingStats)
	assigneeScores := make(map[int][]int) // assigneeID -> []ratings

	for _, t := range ratedTickets {
		if t.AssigneeID > 0 {
			assigneeScores[t.AssigneeID] = append(assigneeScores[t.AssigneeID], t.Rating)
		}
	}

	for assigneeID, ratings := range assigneeScores {
		total := 0
		for _, r := range ratings {
			total += r
		}
		avg := float64(total) / float64(len(ratings))

		// 获取处理人姓名
		assignee, _ := s.client.User.Get(ctx, assigneeID)
		assigneeName := ""
		if assignee != nil {
			assigneeName = assignee.Name
		}

		assigneeStats[assigneeID] = &dto.AssigneeRatingStats{
			AssigneeID:    assigneeID,
			AssigneeName:   assigneeName,
			TotalRatings:  len(ratings),
			AverageRating: avg,
		}
	}

	// 按分类统计
	categoryStats := make(map[int]*dto.CategoryRatingStats)
	categoryScores := make(map[int][]int) // categoryID -> []ratings

	for _, t := range ratedTickets {
		if t.CategoryID > 0 {
			categoryScores[t.CategoryID] = append(categoryScores[t.CategoryID], t.Rating)
		}
	}

	for categoryID, ratings := range categoryScores {
		total := 0
		for _, r := range ratings {
			total += r
		}
		avg := float64(total) / float64(len(ratings))

		// 获取分类名称
		category, _ := s.client.TicketCategory.Get(ctx, categoryID)
		categoryName := ""
		if category != nil {
			categoryName = category.Name
		}

		categoryStats[categoryID] = &dto.CategoryRatingStats{
			CategoryID:    categoryID,
			CategoryName:  categoryName,
			TotalRatings:  len(ratings),
			AverageRating: avg,
		}
	}

	return &dto.RatingStatsResponse{
		TotalRatings:      totalRatings,
		AverageRating:     averageRating,
		RatingDistribution: ratingDistribution,
		ByAssignee:       assigneeStats,
		ByCategory:       categoryStats,
	}, nil
}

