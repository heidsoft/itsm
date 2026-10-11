/**
 * 用户相关类型定义。
 *
 * User / CreateUserRequest / UpdateUserRequest 的唯一来源是
 * `@/lib/api/user-api`（与后端 dto/user_dto.go 逐字段对齐），本文件只再导出
 * 角色词表和少量辅助类型，不要在这里另起一套 User 定义。
 */
export type { User, UserRole, CreateUserRequest, UpdateUserRequest } from '@/lib/api/user-api';

export type UserStatus = 'active' | 'inactive';
