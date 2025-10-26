import { API_BASE_URL } from '@/app/lib/api-config';

export interface Change {
  id: number;
  title: string;
  description: string;
  justification: string;
  type: 'normal' | 'standard' | 'emergency';
  status: 'draft' | 'pending' | 'approved' | 'rejected' | 'implementing' | 'completed' | 'cancelled';
  priority: 'low' | 'medium' | 'high' | 'urgent';
  impactScope: 'low' | 'medium' | 'high';
  riskLevel: 'low' | 'medium' | 'high';
  assigneeId?: number;
  assigneeName?: string;
  createdBy: number;
  createdByName: string;
  tenantId: number;
  plannedStartDate?: string;
  plannedEndDate?: string;
  actualStartDate?: string;
  actualEndDate?: string;
  implementationPlan: string;
  rollbackPlan: string;
  affectedCIs?: string[];
  relatedTickets?: string[];
  createdAt: string;
  updatedAt: string;
}

export interface CreateChangeRequest {
  title: string;
  description: string;
  justification: string;
  type: 'normal' | 'standard' | 'emergency';
  priority: 'low' | 'medium' | 'high' | 'urgent';
  impactScope: 'low' | 'medium' | 'high';
  riskLevel: 'low' | 'medium' | 'high';
  plannedStartDate?: string;
  plannedEndDate?: string;
  implementationPlan: string;
  rollbackPlan: string;
  affectedCIs?: string[];
  relatedTickets?: string[];
}

export interface UpdateChangeRequest extends Partial<CreateChangeRequest> {
  status?: 'draft' | 'pending' | 'approved' | 'rejected' | 'implementing' | 'completed' | 'cancelled';
  assigneeId?: number;
  actualStartDate?: string;
  actualEndDate?: string;
}

export interface ChangeListResponse {
  changes: Change[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface ChangeStats {
  total: number;
  draft: number;
  pending: number;
  approved: number;
  implementing: number;
  completed: number;
  cancelled: number;
}

class ChangeService {
  private baseUrl = `${API_BASE_URL}/changes`;

  // 获取变更列表
  async getChanges(params: {
    page?: number;
    pageSize?: number;
    status?: string;
    search?: string;
  }): Promise<ChangeListResponse> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.append('page', params.page.toString());
    if (params.pageSize) searchParams.append('pageSize', params.pageSize.toString());
    if (params.status && params.status !== '全部') searchParams.append('status', params.status);
    if (params.search) searchParams.append('search', params.search);

    const response = await fetch(`${this.baseUrl}?${searchParams}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getToken()}`,
        'X-Tenant-Code': this.getTenantCode(),
      },
    });

    if (!response.ok) {
      throw new Error(`获取变更列表失败: ${response.statusText}`);
    }

    const result = await response.json();
    return result.data;
  }

  // 获取变更详情
  async getChange(id: number): Promise<Change> {
    const response = await fetch(`${this.baseUrl}/${id}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getToken()}`,
        'X-Tenant-Code': this.getTenantCode(),
      },
    });

    if (!response.ok) {
      throw new Error(`获取变更详情失败: ${response.statusText}`);
    }

    const result = await response.json();
    return result.data;
  }

  // 创建变更
  async createChange(data: CreateChangeRequest): Promise<Change> {
    const response = await fetch(this.baseUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getToken()}`,
        'X-Tenant-Code': this.getTenantCode(),
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      throw new Error(`创建变更失败: ${response.statusText}`);
    }

    const result = await response.json();
    return result.data;
  }

  // 更新变更
  async updateChange(id: number, data: UpdateChangeRequest): Promise<Change> {
    const response = await fetch(`${this.baseUrl}/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getToken()}`,
        'X-Tenant-Code': this.getTenantCode(),
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      throw new Error(`更新变更失败: ${response.statusText}`);
    }

    const result = await response.json();
    return result.data;
  }

  // 删除变更
  async deleteChange(id: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/${id}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getToken()}`,
        'X-Tenant-Code': this.getTenantCode(),
      },
    });

    if (!response.ok) {
      throw new Error(`删除变更失败: ${response.statusText}`);
    }
  }

  // 获取变更统计
  async getChangeStats(): Promise<ChangeStats> {
    const response = await fetch(`${this.baseUrl}/stats`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.getToken()}`,
        'X-Tenant-Code': this.getTenantCode(),
      },
    });

    if (!response.ok) {
      throw new Error(`获取变更统计失败: ${response.statusText}`);
    }

    const result = await response.json();
    return result.data;
  }

  // 健康检查
  async healthCheck(): Promise<boolean> {
    try {
      const response = await fetch(`${apiConfig.baseUrl}/healthz`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      return response.ok;
    } catch {
      return false;
    }
  }

  private getToken(): string {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('token') || '';
    }
    return '';
  }

  private getTenantCode(): string {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('tenantCode') || '';
    }
    return '';
  }
}

export const changeService = new ChangeService();
