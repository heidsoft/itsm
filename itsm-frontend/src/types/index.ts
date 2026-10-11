/**
 * 类型统一导出入口。存在同名类型冲突的模块（例如 Comment）需要直接从源文件导入，
 * 不要经由本文件。
 */

export * from '../lib/api/types';

// 业务模块
export * from './ticket-relations';

export type {
  User,
  UserRole,
  UserStatus,
  CreateUserRequest,
  UpdateUserRequest,
} from './user';

// 只导出前端视图模型。契约类型（Ticket/TicketStatus/TicketPriority/TicketType/
// TicketSource/TicketListResponse/CreateTicketRequest/UpdateTicketRequest/
// TicketStatsResponse）由上面的 `export * from '../lib/api/types'` 再导出，
// 唯一声明处是 @/lib/api/ticket-api；在这里再列一次会让同一类型有两条导出路径。
export type {
  TicketCategory,
  TicketUser,
  TicketComment,
  TicketAttachment,
  TicketSLA,
  TicketFilters,
  TicketSortOptions,
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

// 通用工具类型

export type DateToString<T> = {
  [K in keyof T]: T[K] extends Date
    ? string
    : T[K] extends Date | undefined
      ? string | undefined
      : T[K] extends object
        ? DateToString<T[K]>
        : T[K];
};

export type StringToDate<T> = {
  [K in keyof T]: T[K] extends string
    ? Date
    : T[K] extends string | undefined
      ? Date | undefined
      : T[K] extends object
        ? StringToDate<T[K]>
        : T[K];
};

export type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

export type DeepRequired<T> = {
  [P in keyof T]-?: T[P] extends object ? DeepRequired<T[P]> : T[P];
};

export type PickByType<T, U> = {
  [P in keyof T as T[P] extends U ? P : never]: T[P];
};

export type OmitByType<T, U> = {
  [P in keyof T as T[P] extends U ? never : P]: T[P];
};
