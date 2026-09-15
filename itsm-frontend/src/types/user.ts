/**
 * 用户相关类型定义
 *
 * ⚠️ User / CreateUserRequest / UpdateUserRequest 的唯一来源是
 * `@/lib/api/user-api`（与后端 dto/user_dto.go 逐字段对齐）。
 * 本文件只保留角色词表再导出与少量辅助类型，禁止在此另起一套 User 定义
 * （历史上曾有两套并行定义导致字段漂移，2026-09-15 合并）。
 */
export type { User, UserRole, CreateUserRequest, UpdateUserRequest } from '@/lib/api/user-api';

export type UserStatus = 'active' | 'inactive';
