package dto

import "time"

// 创建知识库文章请求
type CreateKnowledgeArticleRequest struct {
	Title    string   `json:"title" binding:"required"`
	Content  string   `json:"content" binding:"required,min=1,max=10000"`
	Category string   `json:"category" binding:"required,oneof=faq troubleshooting documentation announcement guide other"`
	Tags     []string `json:"tags"`
}

// 更新知识库文章请求
type UpdateKnowledgeArticleRequest struct {
	Title    *string  `json:"title"`
	Content  *string  `json:"content"`
	Category *string  `json:"category"`
	Status   *string  `json:"status"`
	Tags     []string `json:"tags"`
}

// 知识库文章响应
type KnowledgeArticleResponse struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`
	Status    string    `json:"status"`
	Author    string    `json:"author"`
	Views     int       `json:"views"`
	Tags      []string  `json:"tags"`
	TenantID  int       `json:"tenantId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// 知识库文章列表请求
type ListKnowledgeArticlesRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"pageSize" binding:"min=1,max=100"`
	Category string `form:"category"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}

// 知识库文章列表响应
type KnowledgeArticleListResponse struct {
	Articles []KnowledgeArticleResponse `json:"articles"`
	Total    int                        `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"pageSize"`
}

// KnowledgeStatsResponse 知识库统计响应
type KnowledgeStatsResponse struct {
	Total      int             `json:"total"`      // 总文章数
	Published  int             `json:"published"`  // 已发布文章数
	Draft      int             `json:"draft"`      // 草稿数
	Views      int64           `json:"views"`      // 总浏览次数
	Rating     float64         `json:"rating"`     // 平均评分 (基于点赞数)
	Categories []CategoryStats `json:"categories"` // 按分类统计
}

// CategoryStats 分类统计
type CategoryStats struct {
	Name  string `json:"name"`  // 分类名称
	Count int    `json:"count"` // 文章数量
}

// KnowledgeArticleVersionResponse 文章版本历史响应
type KnowledgeArticleVersionResponse struct {
	ID            int       `json:"id"`
	ArticleID     int       `json:"articleId"`
	Version       int       `json:"version"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	Category      string    `json:"category"`
	Tags          []string  `json:"tags"`
	AuthorID      int       `json:"authorId"`
	AuthorName    string    `json:"authorName"`
	ChangeSummary string    `json:"changeSummary"`
	CreatedAt     time.Time `json:"createdAt"`
}

// ListArticleVersionsRequest 列出版本请求
type ListArticleVersionsRequest struct {
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"pageSize" binding:"min=1,max=50"`
}

// KnowledgeArticleVersionListResponse 文章版本列表响应
type KnowledgeArticleVersionListResponse struct {
	Versions []KnowledgeArticleVersionResponse `json:"versions"`
	Total    int                               `json:"total"`
	Page     int                               `json:"page"`
	PageSize int                               `json:"pageSize"`
}

// RestoreArticleVersionRequest 恢复版本请求
type RestoreArticleVersionRequest struct {
	Version int `json:"version" binding:"required,min=1"`
}

// ArticleSessionResponse 协作会话响应
type ArticleSessionResponse struct {
	SessionID     int       `json:"sessionId"`
	ArticleID     int       `json:"articleId"`
	UserID        int       `json:"userId"`
	UserName      string    `json:"userName"`
	SessionToken  string    `json:"sessionToken"`
	Status        string    `json:"status"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	CreatedAt     time.Time `json:"createdAt"`
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	ArticleID int `json:"articleId" binding:"required,min=1"`
}

// SessionHeartbeatRequest 心跳请求
type SessionHeartbeatRequest struct {
	SessionToken string `json:"sessionToken" binding:"required"`
	CursorPos    *int   `json:"cursorPosition"`
}

// ArticleParticipantResponse 参与者响应
type ArticleParticipantResponse struct {
	UserID       int       `json:"userId"`
	UserName     string    `json:"userName"`
	Avatar       string    `json:"avatar"`
	CursorPos    int       `json:"cursorPosition"`
	IsActive     bool      `json:"isActive"`
	JoinedAt     time.Time `json:"joinedAt"`
	LastActivity time.Time `json:"lastActivity"`
}

// ListParticipantsResponse 参与者列表响应
type ListParticipantsResponse struct {
	Participants []ArticleParticipantResponse `json:"participants"`
	Total        int                          `json:"total"`
}
