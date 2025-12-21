/**
 * 工单协作类型定义
 * 支持评论、@提及、通知、协作历史等功能
 */

// ==================== 评论类型 ====================

/**
 * 评论类型
 */
export enum CommentType {
  COMMENT = 'comment',           // 普通评论
  INTERNAL = 'internal',         // 内部备注
  RESOLUTION = 'resolution',     // 解决方案
  WORKAROUND = 'workaround',     // 临时方案
  ROOT_CAUSE = 'root_cause',     // 根因分析
}

/**
 * 评论
 */
export interface Comment {
  id: string;
  ticketId: number;
  content: string;                // 评论内容（支持富文本/Markdown）
  contentType: 'plain' | 'markdown' | 'html';
  type: CommentType;
  authorId: number;
  authorName: string;
  authorAvatar?: string;
  createdAt: Date;
  updatedAt?: Date;
  isEdited: boolean;
  isDeleted: boolean;
  deletedAt?: Date;
  deletedBy?: number;
  
  // 附件
  attachments?: CommentAttachment[];
  
  // @提及
  mentions?: Mention[];
  
  // 回复
  parentId?: string;              // 父评论ID（支持评论线程）
  replyToId?: string;             // 回复的评论ID
  replyToAuthor?: string;         // 回复的作者
  replyCount?: number;            // 回复数量
  
  // 互动
  likeCount: number;
  isLiked?: boolean;              // 当前用户是否点赞
  isPinned: boolean;              // 是否置顶
  
  // 元数据
  metadata?: Record<string, any>;
}

/**
 * 评论附件
 */
export interface CommentAttachment {
  id: string;
  fileName: string;
  fileSize: number;
  fileType: string;
  fileUrl: string;
  thumbnailUrl?: string;
  uploadedAt: Date;
}

/**
 * 评论统计
 */
export interface CommentStats {
  totalComments: number;
  commentsByType: Record<CommentType, number>;
  totalAttachments: number;
  totalMentions: number;
  uniqueParticipants: number;
  lastCommentAt?: Date;
  lastCommentBy?: string;
}

// ==================== @提及 ====================

/**
 * 提及类型
 */
export enum MentionType {
  USER = 'user',                 // 提及用户
  TEAM = 'team',                 // 提及团队
  ROLE = 'role',                 // 提及角色
  EVERYONE = 'everyone',         // @所有人
}

/**
 * 提及
 */
export interface Mention {
  id: string;
  type: MentionType;
  targetId: number;              // 用户/团队/角色ID
  targetName: string;
  displayText: string;           // 显示文本 (如: @张三)
  position: number;              // 在评论中的位置
  isNotified: boolean;           // 是否已通知
  notifiedAt?: Date;
}

/**
 * 提及建议
 */
export interface MentionSuggestion {
  type: MentionType;
  id: number;
  name: string;
  displayName: string;
  avatar?: string;
  email?: string;
  department?: string;
  isOnline?: boolean;
  relevanceScore?: number;       // 相关性评分
}

// ==================== 通知 ====================

/**
 * 通知类型
 */
export enum NotificationType {
  MENTION = 'mention',           // 被@提及
  COMMENT = 'comment',           // 新评论
  REPLY = 'reply',               // 评论回复
  ASSIGNMENT = 'assignment',     // 被分配
  STATUS_CHANGE = 'status_change', // 状态变更
  PRIORITY_CHANGE = 'priority_change', // 优先级变更
  DUE_DATE = 'due_date',         // 到期提醒
  SLA_WARNING = 'sla_warning',   // SLA警告
  APPROVAL_REQUEST = 'approval_request', // 审批请求
  APPROVAL_RESULT = 'approval_result',   // 审批结果
}

/**
 * 通知
 */
export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message: string;
  ticketId?: number;
  ticketNumber?: string;
  commentId?: string;
  
  // 接收者
  recipientId: number;
  recipientName: string;
  
  // 发送者
  senderId?: number;
  senderName?: string;
  senderAvatar?: string;
  
  // 状态
  isRead: boolean;
  readAt?: Date;
  createdAt: Date;
  
  // 操作
  actionUrl?: string;
  actionText?: string;
  
  // 元数据
  metadata?: Record<string, any>;
}

/**
 * 通知设置
 */
export interface NotificationSettings {
  userId: number;
  
  // 通知渠道
  channels: {
    inApp: boolean;              // 站内通知
    email: boolean;              // 邮件通知
    sms: boolean;                // 短信通知
    push: boolean;               // 推送通知
  };
  
  // 通知类型开关
  types: {
    [key in NotificationType]: boolean;
  };
  
  // 高级设置
  muteHours?: {
    start: string;               // 免打扰开始时间 (HH:mm)
    end: string;                 // 免打扰结束时间 (HH:mm)
  };
  digestMode?: 'immediate' | 'hourly' | 'daily'; // 摘要模式
  mentionsOnly?: boolean;        // 仅@提及时通知
}

// ==================== 协作历史 ====================

/**
 * 协作活动类型
 */
export enum ActivityType {
  COMMENT = 'comment',
  MENTION = 'mention',
  ASSIGNMENT = 'assignment',
  STATUS_CHANGE = 'status_change',
  PRIORITY_CHANGE = 'priority_change',
  FIELD_CHANGE = 'field_change',
  ATTACHMENT_ADDED = 'attachment_added',
  ATTACHMENT_REMOVED = 'attachment_removed',
  RELATION_ADDED = 'relation_added',
  RELATION_REMOVED = 'relation_removed',
  TAG_ADDED = 'tag_added',
  TAG_REMOVED = 'tag_removed',
  WATCHER_ADDED = 'watcher_added',
  WATCHER_REMOVED = 'watcher_removed',
}

/**
 * 协作活动
 */
export interface Activity {
  id: string;
  ticketId: number;
  type: ActivityType;
  description: string;
  
  // 操作者
  actorId: number;
  actorName: string;
  actorAvatar?: string;
  
  // 时间
  occurredAt: Date;
  
  // 变更详情
  changes?: ActivityChange[];
  
  // 关联对象
  relatedComment?: Comment;
  relatedUser?: {
    id: number;
    name: string;
  };
  
  // 元数据
  metadata?: Record<string, any>;
}

/**
 * 活动变更
 */
export interface ActivityChange {
  field: string;
  fieldLabel: string;
  oldValue?: any;
  newValue?: any;
  oldValueDisplay?: string;
  newValueDisplay?: string;
}

/**
 * 活动流
 */
export interface ActivityFeed {
  activities: Activity[];
  total: number;
  hasMore: boolean;
  nextCursor?: string;
}

// ==================== 观察者 ====================

/**
 * 观察者
 */
export interface Watcher {
  id: string;
  ticketId: number;
  userId: number;
  userName: string;
  userAvatar?: string;
  addedAt: Date;
  addedBy: number;
  addedByName: string;
  isAutoAdded: boolean;          // 是否自动添加（如被@提及）
}

// ==================== API请求/响应 ====================

/**
 * 创建评论请求
 */
export interface CreateCommentRequest {
  ticketId: number;
  content: string;
  contentType?: 'plain' | 'markdown' | 'html';
  type?: CommentType;
  parentId?: string;
  replyToId?: string;
  mentions?: Array<{
    type: MentionType;
    targetId: number;
    position: number;
  }>;
  attachments?: Array<{
    fileName: string;
    fileUrl: string;
    fileSize: number;
    fileType: string;
  }>;
}

/**
 * 更新评论请求
 */
export interface UpdateCommentRequest {
  content: string;
  contentType?: 'plain' | 'markdown' | 'html';
}

/**
 * 评论列表查询
 */
export interface CommentListQuery {
  ticketId: number;
  type?: CommentType;
  includeDeleted?: boolean;
  includeInternal?: boolean;
  sortBy?: 'created_at' | 'updated_at' | 'like_count';
  sortOrder?: 'asc' | 'desc';
  page?: number;
  pageSize?: number;
}

/**
 * 评论列表响应
 */
export interface CommentListResponse {
  comments: Comment[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

/**
 * 通知列表查询
 */
export interface NotificationListQuery {
  types?: NotificationType[];
  isRead?: boolean;
  startDate?: string;
  endDate?: string;
  page?: number;
  pageSize?: number;
}

/**
 * 通知列表响应
 */
export interface NotificationListResponse {
  notifications: Notification[];
  total: number;
  unreadCount: number;
  page: number;
  pageSize: number;
}

/**
 * 活动查询
 */
export interface ActivityQuery {
  ticketId: number;
  types?: ActivityType[];
  actorId?: number;
  startDate?: string;
  endDate?: string;
  cursor?: string;
  limit?: number;
}

// ==================== 富文本编辑器 ====================

/**
 * 编辑器配置
 */
export interface EditorConfig {
  mode: 'plain' | 'markdown' | 'wysiwyg';
  placeholder?: string;
  maxLength?: number;
  minHeight?: number;
  maxHeight?: number;
  
  // 工具栏
  toolbar?: {
    bold?: boolean;
    italic?: boolean;
    underline?: boolean;
    strikethrough?: boolean;
    heading?: boolean;
    bulletList?: boolean;
    orderedList?: boolean;
    link?: boolean;
    image?: boolean;
    code?: boolean;
    codeBlock?: boolean;
    quote?: boolean;
    mention?: boolean;
    emoji?: boolean;
  };
  
  // @提及
  mentionConfig?: {
    trigger: string;              // 触发字符 (默认 @)
    maxSuggestions: number;
    debounceMs: number;
  };
  
  // 附件
  attachmentConfig?: {
    enabled: boolean;
    maxSize: number;              // 最大文件大小 (bytes)
    maxFiles: number;
    allowedTypes: string[];
  };
}

// ==================== 实时协作 ====================

/**
 * 在线状态
 */
export interface OnlinePresence {
  userId: number;
  userName: string;
  userAvatar?: string;
  isOnline: boolean;
  lastSeenAt?: Date;
  currentActivity?: string;      // 当前正在做什么
}

/**
 * 正在输入状态
 */
export interface TypingIndicator {
  ticketId: number;
  userId: number;
  userName: string;
  startedAt: Date;
}

// ==================== 协作统计 ====================

/**
 * 协作统计
 */
export interface CollaborationStats {
  ticketId: number;
  
  // 评论统计
  commentStats: CommentStats;
  
  // 参与者
  participants: Array<{
    userId: number;
    userName: string;
    userAvatar?: string;
    commentCount: number;
    lastActivityAt: Date;
  }>;
  
  // 活动统计
  activityStats: {
    totalActivities: number;
    activitiesByType: Record<ActivityType, number>;
    activitiesByDay: Array<{
      date: string;
      count: number;
    }>;
  };
  
  // 观察者
  watcherCount: number;
  watchers: Watcher[];
}

// ==================== 导出所有类型 ====================

export default Comment;

