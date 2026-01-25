/**
 * User API
 * Handles all user-related API calls
 */

import type { User, UserFilters, CreateUserRequest, UpdateUserRequest, UserListResponse, UserStats } from '@/types/user';
import { apiClient } from './api-client';

export interface ListUsersParams {
  page?: number;
  pageSize?: number;
  filters?: UserFilters;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface BatchUpdateUsersRequest {
  userIds: number[];
  updates: Partial<UpdateUserRequest>;
}

export interface ResetPasswordRequest {
  userId: number;
  newPassword: string;
  sendNotification?: boolean;
}

class UserAPI {
  /**
   * List users with pagination and filters
   */
  async listUsers(params: ListUsersParams = {}): Promise<UserListResponse> {
    const { page = 1, pageSize = 20, filters = {}, sortBy, sortOrder } = params;

    const queryParams = new URLSearchParams({
      page: page.toString(),
      size: pageSize.toString(),
    });

    if (filters.role && filters.role.length > 0) {
      queryParams.append('role', filters.role.join(','));
    }
    if (filters.status && filters.status.length > 0) {
      queryParams.append('status', filters.status.join(','));
    }
    if (filters.department && filters.department.length > 0) {
      queryParams.append('department', filters.department.join(','));
    }
    if (filters.search) {
      queryParams.append('search', filters.search);
    }
    if (filters.isActive !== undefined) {
      queryParams.append('is_active', filters.isActive.toString());
    }
    if (sortBy) {
      queryParams.append('sort_by', sortBy);
    }
    if (sortOrder) {
      queryParams.append('sort_order', sortOrder);
    }

    return apiClient.get<UserListResponse>(`/users?${queryParams.toString()}`);
  }

  /**
   * Get a single user by ID
   */
  async getUser(id: number): Promise<User> {
    return apiClient.get<User>(`/users/${id}`);
  }

  /**
   * Create a new user
   */
  async createUser(data: CreateUserRequest): Promise<User> {
    return apiClient.post<User>('/users', data);
  }

  /**
   * Update an existing user
   */
  async updateUser(id: number, data: UpdateUserRequest): Promise<User> {
    return apiClient.put<User>(`/users/${id}`, data);
  }

  /**
   * Delete a user (soft delete)
   */
  async deleteUser(id: number): Promise<void> {
    await apiClient.delete(`/users/${id}`);
  }

  /**
   * Change user status (active/inactive/suspended)
   */
  async changeUserStatus(id: number, status: string): Promise<User> {
    return apiClient.put<User>(`/users/${id}/status`, { status });
  }

  /**
   * Reset user password
   */
  async resetPassword(data: ResetPasswordRequest): Promise<void> {
    await apiClient.put(`/users/${data.userId}/reset-password`, data);
  }

  /**
   * Get user statistics
   */
  async getUserStats(): Promise<UserStats> {
    return apiClient.get<UserStats>('/users/stats');
  }

  /**
   * Batch update users
   */
  async batchUpdateUsers(data: BatchUpdateUsersRequest): Promise<User[]> {
    return apiClient.put<User[]>('/users/batch', data);
  }

  /**
   * Search users by keyword
   */
  async searchUsers(keyword: string, limit = 10): Promise<User[]> {
    const queryParams = new URLSearchParams({
      search: keyword,
      size: limit.toString(),
    });
    const response = await apiClient.get<UserListResponse>(`/users?${queryParams.toString()}`);
    return response.users;
  }

  /**
   * Get users by role
   */
  async getUsersByRole(role: string): Promise<User[]> {
    const queryParams = new URLSearchParams({ role });
    const response = await apiClient.get<UserListResponse>(`/users?${queryParams.toString()}`);
    return response.users;
  }

  /**
   * Get users by department
   */
  async getUsersByDepartment(departmentId: number): Promise<User[]> {
    const queryParams = new URLSearchParams({ department_id: departmentId.toString() });
    const response = await apiClient.get<UserListResponse>(`/users?${queryParams.toString()}`);
    return response.users;
  }

  /**
   * Export users
   */
  async exportUsers(filters?: UserFilters, format: 'csv' | 'xlsx' | 'json' = 'csv'): Promise<Blob> {
    const queryParams = new URLSearchParams({ format });

    if (filters) {
      if (filters.role && filters.role.length > 0) {
        queryParams.append('role', filters.role.join(','));
      }
      if (filters.status && filters.status.length > 0) {
        queryParams.append('status', filters.status.join(','));
      }
    }

    const token = localStorage.getItem('access_token') || localStorage.getItem('token') || '';
    const response = await fetch(`${apiConfig.baseURL}/users/export?${queryParams.toString()}`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Export failed: ${response.statusText}`);
    }

    return response.blob();
  }

  /**
   * Import users from file
   */
  async importUsers(file: File, options: {
    skipHeader?: boolean;
    updateExisting?: boolean;
    sendWelcomeEmail?: boolean;
    defaultRole?: string;
  } = {}): Promise<{ success: boolean; message: string; total: number; created: number; updated: number; failed: number }> {
    const formData = new FormData();
    formData.append('file', file);

    Object.entries(options).forEach(([key, value]) => {
      if (value !== undefined) {
        formData.append(key, String(value));
      }
    });

    const token = localStorage.getItem('access_token') || localStorage.getItem('token') || '';
    const response = await fetch(`${apiConfig.baseURL}/users/import`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
      body: formData,
    });

    if (!response.ok) {
      throw new Error(`Import failed: ${response.statusText}`);
    }

    return response.json();
  }
}

export const userAPI = new UserAPI();
