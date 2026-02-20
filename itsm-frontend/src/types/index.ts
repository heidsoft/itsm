/**
 * 统一类型定义导出
 * 所有类型定义的单一入口点
 *
 * 注意：由于类型冲突，部分模块需要直接从源文件导入
 * 例如：import { Comment } from '@/types/comment'
 */

// ==================== 核心实体类型 ====================

// 导出API类型（包含UserBasicInfo等基础类型）
export * from '../lib/api/types';

// ==================== 业务模块类型 ====================
export * from './ticket-relations';

// ==================== 功能模块类型 ====================

// 用户相关（从types/user导入）
export type {
  UserRole,
  UserStatus,
  UserGroup,
  UserPreferences,
  CreateUserRequest,
  UpdateUserRequest,
  UserFilters,
  UserListResponse,
  UserStats,
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  ChangePasswordRequest,
  ResetPasswordRequest,
  SetPasswordRequest,
  UserActivity,
  UserSession,
  Permission,
  RolePermission,
  UserImportRequest,
  UserImportResult,
  UserExportRequest,
} from './user';

// 工单相关（从types/ticket导入）
export type {
  TicketPriority,
  TicketStatus,
  TicketSource,
  TicketType,
  TicketCategory,
  TicketUser,
  TicketComment,
  TicketAttachment,
  TicketSLA,
  Ticket,
  CreateTicketRequest,
  UpdateTicketRequest,
  TicketFilters,
  TicketSortOptions,
  TicketListResponse,
  TicketStats,
  TicketActivity,
  TicketTemplate,
  TicketBatchOperation,
  TicketBatchResult,
} from './ticket';

// 评论相关
export type {
  Comment,
  CommentType,
  CommentCreateRequest,
  CommentUpdateRequest,
  CommentListQuery,
  CommentListResponse,
  CommentStats,
  CommentAttachment,
} from './comment';

// ==================== 工具类型 ====================

/**
 * 将Date类型转换为string类型
 * 用于API响应数据的类型转换
 */
export type DateToString<T> = {
  [K in keyof T]: T[K] extends Date
    ? string
    : T[K] extends Date | undefined
    ? string | undefined
    : T[K] extends object
    ? DateToString<T[K]>
    : T[K];
};

/**
 * 将string类型转换为Date类型
 * 用于前端数据处理的类型转换
 */
export type StringToDate<T> = {
  [K in keyof T]: T[K] extends string
    ? Date
    : T[K] extends string | undefined
    ? Date | undefined
    : T[K] extends object
    ? StringToDate<T[K]>
    : T[K];
};

/**
 * 使所有属性可选
 */
export type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

/**
 * 使所有属性必需
 */
export type DeepRequired<T> = {
  [P in keyof T]-?: T[P] extends object ? DeepRequired<T[P]> : T[P];
};

/**
 * 提取对象中的某些属性
 */
export type PickByType<T, U> = {
  [P in keyof T as T[P] extends U ? P : never]: T[P];
};

/**
 * 排除对象中的某些属性
 */
export type OmitByType<T, U> = {
  [P in keyof T as T[P] extends U ? never : P]: T[P];
};
