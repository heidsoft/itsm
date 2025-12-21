/**
 * 知识库系统类型定义
 * 支持富文本编辑、版本控制、协作和搜索
 */

// ==================== 知识库基础类型 ====================

/**
 * 文章状态
 */
export enum ArticleStatus {
  DRAFT = 'draft',               // 草稿
  UNDER_REVIEW = 'under_review', // 审核中
  PUBLISHED = 'published',       // 已发布
  ARCHIVED = 'archived',         // 已归档
}

/**
 * 文章可见性
 */
export enum ArticleVisibility {
  PUBLIC = 'public',             // 公开
  INTERNAL = 'internal',         // 内部
  PRIVATE = 'private',           // 私有
  RESTRICTED = 'restricted',     // 受限
}

/**
 * 知识库文章
 */
export interface KnowledgeArticle {
  id: string;
  title: string;
  slug: string;                  // URL友好的标识
  
  // 内容
  content: string;               // 富文本HTML内容
  plainText?: string;            // 纯文本（用于搜索）
  summary?: string;              // 摘要
  
  // 分类和标签
  categoryId?: string;
  categoryName?: string;
  tags: string[];
  keywords?: string[];           // 搜索关键词
  
  // 状态和可见性
  status: ArticleStatus;
  visibility: ArticleVisibility;
  
  // 媒体
  featuredImage?: string;
  attachments?: ArticleAttachment[];
  
  // SEO
  metaDescription?: string;
  metaKeywords?: string[];
  
  // 统计
  viewCount: number;
  likeCount: number;
  shareCount: number;
  commentCount: number;
  
  // 评分
  rating?: number;
  ratingCount?: number;
  
  // 关联
  relatedArticles?: string[];    // 相关文章ID
  relatedTickets?: number[];     // 关联工单ID
  
  // 版本控制
  version: number;
  versionHistory?: ArticleVersion[];
  
  // 协作
  authorId: number;
  authorName: string;
  contributors?: Array<{
    userId: number;
    userName: string;
    role: 'editor' | 'reviewer';
  }>;
  
  // 审批
  reviewerId?: number;
  reviewerName?: string;
  reviewComment?: string;
  reviewedAt?: Date;
  
  // 时间戳
  createdAt: Date;
  updatedAt: Date;
  publishedAt?: Date;
  
  // 有效期
  expiresAt?: Date;              // 过期时间
  isExpired?: boolean;
}

/**
 * 文章附件
 */
export interface ArticleAttachment {
  id: string;
  name: string;
  type: string;
  size: number;
  url: string;
  uploadedAt: Date;
}

/**
 * 文章版本
 */
export interface ArticleVersion {
  version: number;
  content: string;
  summary?: string;
  changeLog?: string;
  createdBy: number;
  createdByName: string;
  createdAt: Date;
}

// ==================== 分类和标签 ====================

/**
 * 知识库分类
 */
export interface KnowledgeCategory {
  id: string;
  name: string;
  slug: string;
  description?: string;
  icon?: string;
  color?: string;
  
  // 层级关系
  parentId?: string;
  parentName?: string;
  children?: KnowledgeCategory[];
  level: number;
  order: number;
  
  // 统计
  articleCount: number;
  
  // 权限
  visibility: ArticleVisibility;
  allowedRoles?: string[];
  
  createdAt: Date;
  updatedAt: Date;
}

/**
 * 标签
 */
export interface KnowledgeTag {
  id: string;
  name: string;
  slug: string;
  color?: string;
  articleCount: number;
  createdAt: Date;
}

// ==================== 评论和反馈 ====================

/**
 * 文章评论
 */
export interface ArticleComment {
  id: string;
  articleId: string;
  
  // 评论内容
  content: string;
  
  // 层级（支持回复）
  parentId?: string;
  replies?: ArticleComment[];
  
  // 反馈
  helpful: number;               // 有用计数
  notHelpful: number;            // 无用计数
  
  // 用户信息
  userId: number;
  userName: string;
  userAvatar?: string;
  
  // 状态
  isPublished: boolean;
  isPinned: boolean;             // 是否置顶
  
  createdAt: Date;
  updatedAt: Date;
}

/**
 * 文章反馈
 */
export interface ArticleFeedback {
  id: string;
  articleId: string;
  
  type: 'helpful' | 'not_helpful' | 'outdated' | 'incorrect' | 'suggestion';
  comment?: string;
  
  userId: number;
  userName: string;
  createdAt: Date;
}

// ==================== 搜索和推荐 ====================

/**
 * 搜索请求
 */
export interface KnowledgeSearchRequest {
  query: string;
  categoryId?: string;
  tags?: string[];
  visibility?: ArticleVisibility;
  status?: ArticleStatus;
  authorId?: number;
  
  // 过滤
  dateFrom?: string;
  dateTo?: string;
  minRating?: number;
  
  // 排序
  sortBy?: 'relevance' | 'date' | 'views' | 'rating';
  sortOrder?: 'asc' | 'desc';
  
  // 分页
  page?: number;
  pageSize?: number;
}

/**
 * 搜索结果
 */
export interface KnowledgeSearchResult {
  articles: Array<{
    article: KnowledgeArticle;
    score: number;              // 相关度分数
    highlights?: {
      title?: string;
      content?: string;
      summary?: string;
    };
  }>;
  total: number;
  facets?: {
    categories: Record<string, number>;
    tags: Record<string, number>;
    authors: Record<string, number>;
  };
}

/**
 * 推荐文章
 */
export interface ArticleRecommendation {
  article: KnowledgeArticle;
  reason: 'popular' | 'related' | 'recent' | 'personalized';
  score: number;
}

// ==================== 富文本编辑器 ====================

/**
 * 编辑器配置
 */
export interface EditorConfig {
  // 工具栏配置
  toolbar: Array<
    | 'heading'
    | 'bold'
    | 'italic'
    | 'underline'
    | 'strikethrough'
    | 'link'
    | 'image'
    | 'video'
    | 'code'
    | 'codeblock'
    | 'quote'
    | 'list'
    | 'orderedlist'
    | 'table'
    | 'horizontalrule'
    | 'undo'
    | 'redo'
  >;
  
  // 功能开关
  enableImageUpload: boolean;
  enableVideoEmbed: boolean;
  enableCodeHighlight: boolean;
  enableMarkdown: boolean;
  enableMentions: boolean;
  
  // 限制
  maxLength?: number;
  maxImageSize?: number;         // MB
  allowedImageTypes?: string[];
  
  // 自动保存
  autoSave: boolean;
  autoSaveInterval?: number;     // 秒
}

/**
 * 编辑器内容变更
 */
export interface EditorChange {
  content: string;
  plainText: string;
  wordCount: number;
  characterCount: number;
  timestamp: Date;
}

/**
 * 图片上传结果
 */
export interface ImageUploadResult {
  url: string;
  name: string;
  size: number;
  width?: number;
  height?: number;
}

// ==================== 协作和通知 ====================

/**
 * 协作会话
 */
export interface CollaborationSession {
  articleId: string;
  participants: Array<{
    userId: number;
    userName: string;
    color: string;
    cursor?: {
      line: number;
      column: number;
    };
  }>;
  startedAt: Date;
}

/**
 * 文章通知
 */
export interface ArticleNotification {
  id: string;
  type: 'comment' | 'mention' | 'review_request' | 'published' | 'updated';
  articleId: string;
  articleTitle: string;
  
  message: string;
  
  // 发送者
  fromUserId?: number;
  fromUserName?: string;
  
  // 接收者
  toUserId: number;
  
  isRead: boolean;
  createdAt: Date;
}

// ==================== 统计和分析 ====================

/**
 * 知识库统计
 */
export interface KnowledgeBaseStats {
  totalArticles: number;
  publishedArticles: number;
  draftArticles: number;
  archivedArticles: number;
  
  totalViews: number;
  totalLikes: number;
  totalComments: number;
  
  articlesByCategory: Record<string, number>;
  articlesByStatus: Record<ArticleStatus, number>;
  articlesByVisibility: Record<ArticleVisibility, number>;
  
  topArticles: Array<{
    article: KnowledgeArticle;
    views: number;
    rating: number;
  }>;
  
  topAuthors: Array<{
    userId: number;
    userName: string;
    articleCount: number;
    avgRating: number;
  }>;
  
  trends: {
    date: string;
    created: number;
    published: number;
    views: number;
  }[];
}

/**
 * 文章分析
 */
export interface ArticleAnalytics {
  articleId: string;
  period: {
    start: Date;
    end: Date;
  };
  
  metrics: {
    views: number;
    uniqueVisitors: number;
    avgReadTime: number;        // 秒
    likes: number;
    shares: number;
    comments: number;
    rating: number;
  };
  
  viewTrend: {
    date: string;
    views: number;
  }[];
  
  referrers: {
    source: string;
    count: number;
  }[];
  
  searchKeywords: {
    keyword: string;
    count: number;
  }[];
}

// ==================== API请求/响应 ====================

/**
 * 创建文章请求
 */
export interface CreateArticleRequest {
  title: string;
  content: string;
  summary?: string;
  categoryId?: string;
  tags?: string[];
  visibility?: ArticleVisibility;
  featuredImage?: string;
  metaDescription?: string;
  expiresAt?: Date;
}

/**
 * 更新文章请求
 */
export type UpdateArticleRequest = Partial<CreateArticleRequest> & {
  changeLog?: string;
};

/**
 * 发布文章请求
 */
export interface PublishArticleRequest {
  scheduleAt?: Date;             // 定时发布
}

/**
 * 审核文章请求
 */
export interface ReviewArticleRequest {
  approved: boolean;
  comment?: string;
}

/**
 * 文章查询
 */
export interface ArticleQuery {
  categoryId?: string;
  status?: ArticleStatus;
  visibility?: ArticleVisibility;
  authorId?: number;
  tags?: string[];
  search?: string;
  isExpired?: boolean;
  page?: number;
  pageSize?: number;
  sortBy?: 'created' | 'updated' | 'views' | 'rating';
  sortOrder?: 'asc' | 'desc';
}

/**
 * 批量操作请求
 */
export interface BatchArticleOperation {
  articleIds: string[];
  operation: 'publish' | 'archive' | 'delete' | 'change_category' | 'add_tags';
  params?: {
    categoryId?: string;
    tags?: string[];
  };
}

export default KnowledgeArticle;

