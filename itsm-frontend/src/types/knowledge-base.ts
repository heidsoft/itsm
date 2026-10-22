/**
 * 知识库系统类型定义
 * 支持富文本编辑、版本控制、协作和搜索
 */

// ==================== 知识库基础类型 ====================

/**
 * 文章状态
 */
export enum ArticleStatus {
  DRAFT = 'draft', // 草稿
  UNDER_REVIEW = 'under_review', // 审核中
  PUBLISHED = 'published', // 已发布
  ARCHIVED = 'archived', // 已归档
}

/**
 * 评审状态 —— 后端 ent schema KnowledgeArticle.review_status 的枚举。
 *
 * 与 ArticleStatus 的区别：后者是前端自造的展示态，后端没有对应字段；
 * 本枚举是真实落库值，判断"这篇知识批没批过"只能用它。
 */
export enum KnowledgeReviewStatus {
  DRAFT = 'draft', // 草稿：作者可编辑，未提交
  UNDER_REVIEW = 'under_review', // 评审中：等待评审人决策
  APPROVED = 'approved', // 已通过：评审人批准，对外可见
  REJECTED = 'rejected', // 已驳回：作者可修改后重新提交
}

/** 评审动作 */
export type KnowledgeReviewAction =
  | 'submit'
  | 'approve'
  | 'reject'
  | 'withdraw'
  | 'republish';

/** 一条评审审计流水 */
export interface KnowledgeReviewRecord {
  id: number;
  articleId: number;
  tenantId: number;
  fromStatus: string;
  toStatus: string;
  action: KnowledgeReviewAction | string;
  operatorId: number;
  comment?: string;
  createdAt: string;
}

/**
 * 文章可见性
 */
export enum ArticleVisibility {
  PUBLIC = 'public', // 公开
  INTERNAL = 'internal', // 内部
  PRIVATE = 'private', // 私有
  RESTRICTED = 'restricted', // 受限
}

/**
 * 知识库文章
 */
export interface KnowledgeArticle {
  id: string;
  title: string;
  slug: string; // URL友好的标识

  // 内容
  content: string; // 富文本HTML内容
  plainText?: string; // 纯文本（用于搜索）
  summary?: string; // 摘要

  // 分类和标签
  categoryId?: string;
  categoryName?: string;
  tags: string[];
  keywords?: string[]; // 搜索关键词

  // 状态和可见性
  status: ArticleStatus;
  visibility: ArticleVisibility;

  // ---- 评审状态机（后端 ent schema KnowledgeArticle.review_status）----
  //
  // 权威来源是 reviewStatus，取值 draft/under_review/approved/rejected。
  //
  // 为什么与上面的 status 并存：status 是前端自造的枚举，后端文章实体上
  // **并不存在** status 字段（只有 is_published 布尔）。历史代码把
  // status 当成"待审核/已发布"用，导致 knowledge/reviews 页面永远筛不出数据。
  // reviewStatus 是真实落库的字段，新代码请一律用它判断评审阶段；
  // status 仅保留给仍依赖它的旧组件。
  reviewStatus?: KnowledgeReviewStatus;
  // reviewerId/reviewedAt/reviewComment 只在本文（KnowledgeArticle）声明一次。
  // 相邻的 ReviewArticleRequest 面向旧的 review/decision 断链端点，
  // 字段同名但语义不同（那边是请求入参、类型也不同），不要合并。

  // isPublished 是「对外可见」的布尔投影，由 reviewStatus 派生但语义不同：
  // 评审通过才为 true，检索面（RAG / 搜索）只看它。
  isPublished?: boolean;

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
  relatedArticles?: string[]; // 相关文章ID
  relatedTickets?: number[]; // 关联工单ID

  // 来源追溯（2026-10-22 沉淀闭环）：文章由哪个问题沉淀而来；undefined=手工创建。
  // 后端 knowledge_articles.source_problem_id，裸 ID 引用而非外键——知识是独立资产，
  // 问题删除不级联删知识。问题详情页「知识沉淀」Tab 与文章详情页「来源问题」共用。
  sourceProblemId?: number;

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
  expiresAt?: Date; // 过期时间
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
  helpful: number; // 有用计数
  notHelpful: number; // 无用计数

  // 用户信息
  userId: number;
  userName: string;
  userAvatar?: string;

  // 状态
  isPublished: boolean;
  isPinned: boolean; // 是否置顶

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
    score: number; // 相关度分数
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
  maxImageSize?: number; // MB
  allowedImageTypes?: string[];

  // 自动保存
  autoSave: boolean;
  autoSaveInterval?: number; // 秒
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
 * 知识库统计 - 与后端 KnowledgeStatsResponse 对齐
 */
export interface KnowledgeBaseStats {
  total: number;           // 总文章数 (后端字段)
  published: number;      // 已发布文章数 (后端字段)
  draft: number;          // 草稿数 (后端字段)
  views: number;          // 总浏览次数 (后端字段)
  rating: number;         // 平均评分 (后端字段)
  categories: Array<{     // 分类统计 (后端字段)
    name: string;
    count: number;
  }>;
  
  // 兼容旧字段 (可选)
  totalArticles?: number;
  publishedArticles?: number;
  draftArticles?: number;
  archivedArticles?: number;
  totalViews?: number;
  totalLikes?: number;
  totalComments?: number;
  articlesByCategory?: Record<string, number>;
  articlesByStatus?: Record<ArticleStatus, number>;
  articlesByVisibility?: Record<ArticleVisibility, number>;
}

// 知识库分析仪表盘
export interface KnowledgeAnalyticsDashboard {
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

  trends: Array<{
    date: string;
    created: number;
    published: number;
    views: number;
  }>;
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
    avgReadTime: number; // 秒
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
  category?: string;
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
  scheduleAt?: Date; // 定时发布
}

/**
 * 审核文章请求
 */
export interface ReviewArticleRequest {
  action: 'approve' | 'reject';
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
