/**
 * 评论相关类型定义
 * 统一的评论类型，避免多处定义导致冲突
 */

// ==================== 评论类型 ====================

/**
 * 评论类型枚举
 */
export enum CommentType {
  INTERNAL = 'internal',     // 内部评论
  PUBLIC = 'public',         // 公开评论
  SYSTEM = 'system',         // 系统评论
}

/**
 * 评论实体
 * 统一的评论接口定义
 */
export interface Comment {
  id: number;
  ticket_id: number;
  user_id: number;
  user?: {
    id: number;
    username: string;
    name: string;
    email: string;
    avatar?: string;
  };
  content: string;
  is_internal: boolean;
  type?: CommentType;
  mentions?: number[];
  attachments?: number[];
  created_at: string;
  updated_at: string;
}

/**
 * 创建评论请求
 */
export interface CommentCreateRequest {
  content: string;
  is_internal?: boolean;
  mentions?: number[];
  attachments?: number[];
}

/**
 * 更新评论请求
 */
export interface CommentUpdateRequest {
  content?: string;
  is_internal?: boolean;
}

/**
 * 评论列表查询参数
 */
export interface CommentListQuery {
  ticket_id: number;
  type?: CommentType;
  is_internal?: boolean;
  user_id?: number;
  page?: number;
  page_size?: number;
}

/**
 * 评论列表响应
 */
export interface CommentListResponse {
  comments: Comment[];
  total: number;
  page: number;
  page_size: number;
}

/**
 * 评论统计
 */
export interface CommentStats {
  total_comments: number;
  internal_comments: number;
  public_comments: number;
  system_comments: number;
  comments_by_user: Record<number, number>;
}

/**
 * 评论附件
 */
export interface CommentAttachment {
  id: number;
  comment_id: number;
  file_name: string;
  file_path: string;
  file_url: string;
  file_size: number;
  mime_type: string;
  created_at: string;
}
