/**
 * 通用系统 API 客户端
 */

import { httpClient } from '@/lib/api/http-client';
import type { User, Department, Team, Tag, AuditLog, AuthResult } from './types';

export class CommonApi {
    /**
     * 用户登录
     */
    static async login(data: any): Promise<AuthResult> {
        return httpClient.post<AuthResult>('/api/v1/auth/login', data);
    }

    /**
     * 获取当前用户信息
     */
    static async getMe(): Promise<User> {
        return httpClient.get<User>('/api/v1/users/me');
    }

    /**
     * 获取用户列表
     */
    static async listUsers(): Promise<User[]> {
        return httpClient.get<User[]>('/api/v1/users');
    }

    /**
     * 获取部门树
     */
    static async getDepartmentTree(): Promise<Department[]> {
        return httpClient.get<Department[]>('/api/v1/org/departments/tree');
    }

    /**
     * 获取团队列表
     */
    static async listTeams(): Promise<Team[]> {
        return httpClient.get<Team[]>('/api/v1/org/teams');
    }

    /**
     * 获取标签列表
     */
    static async listTags(): Promise<Tag[]> {
        return httpClient.get<Tag[]>('/api/v1/system/tags');
    }

    /**
     * 获取审计日志
     */
    static async getAuditLogs(params?: { user_id?: number }): Promise<AuditLog[]> {
        return httpClient.get<AuditLog[]>('/api/v1/system/audit-logs', { params });
    }
}
