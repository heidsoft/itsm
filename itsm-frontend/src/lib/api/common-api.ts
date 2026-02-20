/**
 * 通用 API 包装器
 * 兼容旧代码使用
 */

import { httpClient } from './http-client';

// 使用 any 类型避免导入错误
type User = any;
type Department = any;
type Team = any;
type Tag = any;
type AuditLog = any;

export class CommonApi {
  /**
   * 获取用户列表
   */
  static async getUsers(params?: any): Promise<User[]> {
    return httpClient.get('/api/v1/users', params);
  }

  /**
   * 获取用户详情
   */
  static async getUser(id: number): Promise<User> {
    return httpClient.get(`/api/v1/users/${id}`);
  }

  /**
   * 列出用户
   */
  static async listUsers(params?: any): Promise<User[]> {
    return this.getUsers(params);
  }

  /**
   * 获取部门列表
   */
  static async getDepartments(): Promise<Department[]> {
    return httpClient.get('/api/v1/departments');
  }

  /**
   * 获取部门树
   */
  static async getDepartmentTree(): Promise<any[]> {
    return httpClient.get('/api/v1/departments/tree');
  }

  /**
   * 获取团队列表
   */
  static async getTeams(): Promise<Team[]> {
    return httpClient.get('/api/v1/teams');
  }

  /**
   * 获取标签列表
   */
  static async getTags(): Promise<Tag[]> {
    return httpClient.get('/api/v1/tags');
  }

  /**
   * 获取审计日志
   */
  static async getAuditLogs(params?: any): Promise<{ items: AuditLog[]; total: number }> {
    return httpClient.get('/api/v1/audit-logs', params);
  }
}

export default CommonApi;
