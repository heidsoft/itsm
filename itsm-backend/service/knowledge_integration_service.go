package service

import (
	"context"
	"fmt"
	"itsm-backend/ent"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/tickettag"
	"sort"
	"strings"
	"time"
)

// KnowledgeIntegrationService 知识库集成服务
type KnowledgeIntegrationService struct {
	client *ent.Client
}

// NewKnowledgeIntegrationService 创建知识库集成服务实例
func NewKnowledgeIntegrationService(client *ent.Client) *KnowledgeIntegrationService {
	return &KnowledgeIntegrationService{
		client: client,
	}
}

// SolutionRecommendation 解决方案推荐
type SolutionRecommendation struct {
	ArticleID      int       `json:"article_id"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	RelevanceScore float64   `json:"relevance_score"`
	Category       string    `json:"category"`
	Tags           []string  `json:"tags"`
	LastUpdated    time.Time `json:"last_updated"`
}

// KnowledgeAssociation 知识库关联
type KnowledgeAssociation struct {
	TicketID        int       `json:"ticket_id"`
	ArticleID       int       `json:"article_id"`
	AssociationType string    `json:"association_type"` // auto, manual, suggested
	RelevanceScore  float64   `json:"relevance_score"`
	CreatedAt       time.Time `json:"created_at"`
}

// AIRecommendation AI辅助建议
type AIRecommendation struct {
	TicketID   int      `json:"ticket_id"`
	Category   string   `json:"suggested_category"`
	Priority   string   `json:"suggested_priority"`
	Tags       []string `json:"suggested_tags"`
	Assignee   *int     `json:"suggested_assignee"`
	Confidence float64  `json:"confidence"`
	Reason     string   `json:"reason"`
}

// RecommendSolutions 推荐解决方案
func (s *KnowledgeIntegrationService) RecommendSolutions(ctx context.Context, ticketID int, limit int) ([]SolutionRecommendation, error) {
	// 1. 获取工单信息
	ticketEntity, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("获取工单失败: %w", err)
	}

	// 2. 提取工单关键词
	keywords := s.extractKeywords(ticketEntity.Title, ticketEntity.Description)

	// 3. 基于关键词搜索知识库文章
	articles, err := s.searchKnowledgeArticles(ctx, keywords, ticketEntity.TenantID, limit*2)
	if err != nil {
		return nil, err
	}

	// 4. 计算相关性评分
	var recommendations []SolutionRecommendation
	for _, article := range articles {
		score := s.calculateRelevanceScore(ticketEntity, article, keywords)
		if score > 0.3 { // 只推荐相关性大于30%的解决方案
			recommendations = append(recommendations, SolutionRecommendation{
				ArticleID:      article.ID,
				Title:          article.Title,
				Content:        article.Content,
				RelevanceScore: score,
				Category:       article.Category,
				Tags:           s.parseTags(article.Tags),
				LastUpdated:    article.UpdatedAt,
			})
		}
	}

	// 5. 按相关性评分排序
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].RelevanceScore > recommendations[j].RelevanceScore
	})

	// 6. 返回前N个推荐
	if len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	return recommendations, nil
}

// extractKeywords 提取关键词
func (s *KnowledgeIntegrationService) extractKeywords(title, description string) []string {
	var keywords []string
	
	// 合并标题和描述
	text := strings.ToLower(title + " " + description)
	
	// 简单的关键词提取（实际项目中可以使用NLP库）
	words := strings.Fields(text)
	wordCount := make(map[string]int)
	
	for _, word := range words {
		// 过滤掉常见的停用词
		if len(word) > 2 && !s.isStopWord(word) {
			wordCount[word]++
		}
	}
	
	// 选择出现频率最高的词作为关键词
	for word, count := range wordCount {
		if count >= 2 { // 至少出现2次
			keywords = append(keywords, word)
		}
	}
	
	// 限制关键词数量
	if len(keywords) > 10 {
		keywords = keywords[:10]
	}
	
	return keywords
}

// isStopWord 判断是否为停用词
func (s *KnowledgeIntegrationService) isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "may": true, "might": true, "can": true, "this": true, "that": true,
		"these": true, "those": true, "i": true, "you": true, "he": true, "she": true,
		"it": true, "we": true, "they": true, "me": true, "him": true, "her": true,
		"us": true, "them": true, "my": true, "your": true, "his": true, "hers": true,
		"its": true, "our": true, "their": true, "mine": true, "yours": true, "ours": true, "theirs": true,
	}
	
	return stopWords[word]
}

// searchKnowledgeArticles 搜索知识库文章
func (s *KnowledgeIntegrationService) searchKnowledgeArticles(ctx context.Context, keywords []string, tenantID int, limit int) ([]*ent.KnowledgeArticle, error) {
	if len(keywords) == 0 {
		return []*ent.KnowledgeArticle{}, nil
	}

	// 构建搜索查询
	query := s.client.KnowledgeArticle.Query().
		Where(knowledgearticle.TenantIDEQ(tenantID)).
		Where(knowledgearticle.IsPublishedEQ(true))

	// 添加关键词搜索条件
	var conditions []predicate.KnowledgeArticle
	for _, keyword := range keywords {
		conditions = append(conditions, 
			knowledgearticle.Or(
				knowledgearticle.TitleContains(keyword),
				knowledgearticle.ContentContains(keyword),
			),
		)
	}
	
	if len(conditions) > 0 {
		query = query.Where(knowledgearticle.And(conditions...))
	}

	// 执行查询
	articles, err := query.
		Order(ent.Desc(knowledgearticle.FieldUpdatedAt)).
		Limit(limit).
		All(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("搜索知识库文章失败: %w", err)
	}

	return articles, nil
}

// calculateRelevanceScore 计算相关性评分
func (s *KnowledgeIntegrationService) calculateRelevanceScore(ticket *ent.Ticket, article *ent.KnowledgeArticle, keywords []string) float64 {
	var score float64

	// 1. 标题匹配度 (50%)
	titleScore := s.calculateTextMatchScore(ticket.Title, article.Title, keywords)
	score += titleScore * 0.5

	// 2. 描述匹配度 (30%)
	descScore := s.calculateTextMatchScore(ticket.Description, article.Content, keywords)
	score += descScore * 0.3

	// 3. 分类匹配度 (20%)
	if ticket.CategoryID != 0 && article.Category != "" {
		categoryScore := s.calculateCategoryMatchScore(ticket.CategoryID, article.Category)
		score += categoryScore * 0.2
	}

	return score
}

// calculateTextMatchScore 计算文本匹配度
func (s *KnowledgeIntegrationService) calculateTextMatchScore(text1, text2 string, keywords []string) float64 {
	if len(keywords) == 0 {
		return 0.5
	}

	text1Lower := strings.ToLower(text1)
	text2Lower := strings.ToLower(text2)
	
	matched := 0
	for _, keyword := range keywords {
		if strings.Contains(text1Lower, keyword) && strings.Contains(text2Lower, keyword) {
			matched++
		}
	}

	return float64(matched) / float64(len(keywords))
}

// calculateCategoryMatchScore 计算分类匹配度
func (s *KnowledgeIntegrationService) calculateCategoryMatchScore(ticketCategoryID int, articleCategory string) float64 {
	// 这里简化处理，实际应该查询分类的层级关系
	// 暂时返回默认值
	return 0.7
}

// parseTags 解析标签
func (s *KnowledgeIntegrationService) parseTags(tags string) []string {
	// 这里简化处理，实际应该解析JSON格式的标签
	// 暂时返回空数组
	return []string{}
}

// AssociateWithKnowledge 关联知识库文章
func (s *KnowledgeIntegrationService) AssociateWithKnowledge(ctx context.Context, ticketID int, articleID int, associationType string) (*KnowledgeAssociation, error) {
	// 1. 验证工单和文章是否存在
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	article, err := s.client.KnowledgeArticle.Get(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("知识库文章不存在: %w", err)
	}

	// 2. 计算相关性评分
	keywords := s.extractKeywords(ticket.Title, ticket.Description)
	relevanceScore := s.calculateRelevanceScore(ticket, article, keywords)

	// 3. 创建关联记录（这里简化处理，实际应该创建关联表）
	association := &KnowledgeAssociation{
		TicketID:        ticketID,
		ArticleID:       articleID,
		AssociationType: associationType,
		RelevanceScore:  relevanceScore,
		CreatedAt:       time.Now(),
	}

	// 4. 更新工单标签（添加知识库相关标签）
	err = s.addKnowledgeTagToTicket(ctx, ticketID, "knowledge_linked")
	if err != nil {
		// 记录错误但不影响主要流程
		fmt.Printf("添加知识库标签失败: %v\n", err)
	}

	return association, nil
}

// addKnowledgeTagToTicket 为工单添加知识库标签
func (s *KnowledgeIntegrationService) addKnowledgeTagToTicket(ctx context.Context, ticketID int, tagName string) error {
	// 查找或创建标签
	_, err := s.client.TicketTag.Query().
		Where(tickettag.NameEQ(tagName)).
		First(ctx)
	
	if err != nil {
		// 创建新标签
		_, err = s.client.TicketTag.Create().
			SetName(tagName).
			SetColor("#52c41a").
			SetDescription("知识库关联标签").
			SetTenantID(1). // 这里应该从工单获取租户ID
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建标签失败: %w", err)
		}
	}

	// 这里应该创建工单和标签的关联关系
	// 由于没有many-to-many关系表，暂时跳过
	return nil
}

// GetAIRecommendations 获取AI辅助建议
func (s *KnowledgeIntegrationService) GetAIRecommendations(ctx context.Context, ticketID int) (*AIRecommendation, error) {
	// 1. 获取工单信息
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("获取工单失败: %w", err)
	}

	// 2. 分析工单内容
	keywords := s.extractKeywords(ticket.Title, ticket.Description)
	
	// 3. 基于内容分析生成建议
	recommendation := &AIRecommendation{
		TicketID:   ticketID,
		Confidence: 0.8,
		Reason:     "基于工单内容分析",
	}

	// 4. 建议分类
	if ticket.CategoryID == 0 {
		recommendation.Category = s.suggestCategory(ctx, keywords, ticket.TenantID)
		recommendation.Confidence *= 0.9
	}

	// 5. 建议优先级
	if ticket.Priority == "" {
		recommendation.Priority = s.suggestPriority(ctx, keywords, ticket.TenantID)
		recommendation.Confidence *= 0.9
	}

	// 6. 建议标签
	if len(keywords) > 0 {
		recommendation.Tags = s.suggestTags(ctx, keywords, ticket.TenantID)
		recommendation.Confidence *= 0.95
	}

	// 7. 建议处理人
	if ticket.AssigneeID == 0 {
		recommendation.Assignee = s.suggestAssignee(ctx, ticket, keywords)
		recommendation.Confidence *= 0.85
	}

	return recommendation, nil
}

// suggestCategory 建议分类
func (s *KnowledgeIntegrationService) suggestCategory(ctx context.Context, keywords []string, tenantID int) string {
	// 这里简化处理，实际应该使用机器学习模型
	// 基于关键词匹配历史数据来建议分类
	
	// 简单的关键词到分类映射
	keywordToCategory := map[string]string{
		"network":   "网络问题",
		"server":    "服务器问题",
		"database":  "数据库问题",
		"application": "应用问题",
		"hardware":  "硬件问题",
		"software":  "软件问题",
		"user":      "用户问题",
		"access":    "权限问题",
	}

	for _, keyword := range keywords {
		if category, exists := keywordToCategory[keyword]; exists {
			return category
		}
	}

	return "其他问题"
}

// suggestPriority 建议优先级
func (s *KnowledgeIntegrationService) suggestPriority(ctx context.Context, keywords []string, tenantID int) string {
	// 基于关键词和内容分析建议优先级
	
	// 高优先级关键词
	highPriorityWords := []string{"urgent", "critical", "error", "failed", "down", "broken", "emergency"}
	
	// 中优先级关键词
	mediumPriorityWords := []string{"issue", "problem", "slow", "performance", "bug"}
	
	// 低优先级关键词
	lowPriorityWords := []string{"question", "request", "enhancement", "feature"}

	for _, keyword := range keywords {
		for _, word := range highPriorityWords {
			if keyword == word {
				return "high"
			}
		}
		for _, word := range mediumPriorityWords {
			if keyword == word {
				return "medium"
			}
		}
		for _, word := range lowPriorityWords {
			if keyword == word {
				return "low"
			}
		}
	}

	return "medium" // 默认中等优先级
}

// suggestTags 建议标签
func (s *KnowledgeIntegrationService) suggestTags(ctx context.Context, keywords []string, tenantID int) []string {
	// 基于关键词建议标签
	var suggestedTags []string
	
	// 简单的关键词到标签映射
	keywordToTags := map[string][]string{
		"network":     {"网络", "连接", "通信"},
		"server":      {"服务器", "系统", "性能"},
		"database":    {"数据库", "数据", "存储"},
		"application": {"应用", "软件", "功能"},
		"hardware":    {"硬件", "设备", "物理"},
		"software":    {"软件", "程序", "代码"},
		"user":        {"用户", "账户", "权限"},
		"access":      {"访问", "权限", "安全"},
	}

	for _, keyword := range keywords {
		if tags, exists := keywordToTags[keyword]; exists {
			suggestedTags = append(suggestedTags, tags...)
		}
	}

	// 去重
	seen := make(map[string]bool)
	var uniqueTags []string
	for _, tag := range suggestedTags {
		if !seen[tag] {
			seen[tag] = true
			uniqueTags = append(uniqueTags, tag)
		}
	}

	return uniqueTags
}

// suggestAssignee 建议处理人
func (s *KnowledgeIntegrationService) suggestAssignee(ctx context.Context, ticket *ent.Ticket, keywords []string) *int {
	// 这里简化处理，实际应该基于技能匹配和工作负载来建议
	// 暂时返回nil，表示不提供建议
	return nil
}

// GetRelatedArticles 获取相关文章
func (s *KnowledgeIntegrationService) GetRelatedArticles(ctx context.Context, ticketID int, limit int) ([]*ent.KnowledgeArticle, error) {
	// 获取工单信息
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("获取工单失败: %w", err)
	}

	// 提取关键词
	keywords := s.extractKeywords(ticket.Title, ticket.Description)
	
	// 搜索相关文章
	articles, err := s.searchKnowledgeArticles(ctx, keywords, ticket.TenantID, limit)
	if err != nil {
		return nil, err
	}

	return articles, nil
}

// UpdateArticleViewCount 更新文章查看次数
func (s *KnowledgeIntegrationService) UpdateArticleViewCount(ctx context.Context, articleID int) error {
	// 由于KnowledgeArticle没有ViewCount字段，这个方法暂时不实现
	// 实际项目中需要先添加该字段到schema中
	return fmt.Errorf("文章查看次数字段未实现")
}
