import { httpClient } from '@/lib/api/http-client';

export interface ChangeComment {
  author: string;
  timestamp: string;
  text: string;
}

export interface ChangeApproval {
  role: string;
  status: '待审批' | '已批准' | '已拒绝';
  approver?: string;
  comment?: string;
}

export interface RelatedTicket {
  id: string | number;
  type: string;
  title: string;
  status: string;
}

export interface AffectedCI {
  id: string | number;
  name: string;
  type: string;
}

export interface Change {
  id: number;
  title: string;
  description: string;
  justification: string;
  type: 'normal' | 'standard' | 'emergency';
  status:
    | 'draft'
    | 'pending'
    | 'approved'
    | 'rejected'
    | 'implementing'
    | 'completed'
    | 'cancelled';
  priority: 'low' | 'medium' | 'high' | 'urgent';
  impactScope: 'low' | 'medium' | 'high';
  riskLevel: 'low' | 'medium' | 'high';
  assigneeId?: number;
  assigneeName?: string;
  assignee?: string | { id: number; name: string };
  createdBy: number;
  createdByName: string;
  tenantId: number;
  plannedStartDate?: string;
  plannedEndDate?: string;
  actualStartDate?: string;
  actualEndDate?: string;
  implementationPlan: string;
  rollbackPlan: string;
  affectedCIs?: string[] | AffectedCI[];
  relatedTickets?: string[] | RelatedTicket[];
  logs?: string[];
  comments?: ChangeComment[];
  approvals?: ChangeApproval[];
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
  status?:
    | 'draft'
    | 'pending'
    | 'approved'
    | 'rejected'
    | 'implementing'
    | 'completed'
    | 'cancelled';
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
  private readonly baseUrl = '/api/v1/changes';

  // 获取变更列表
  async getChanges(params: {
    page?: number;
    pageSize?: number;
    status?: string;
    search?: string;
  }): Promise<ChangeListResponse> {
    return httpClient.get<ChangeListResponse>(this.baseUrl, params);
  }

  // 获取变更详情
  async getChange(id: number): Promise<Change> {
    return httpClient.get<Change>(`${this.baseUrl}/${id}`);
  }

  // 创建变更
  async createChange(data: CreateChangeRequest): Promise<Change> {
    return httpClient.post<Change>(this.baseUrl, data);
  }

  // 更新变更
  async updateChange(id: number, data: UpdateChangeRequest): Promise<Change> {
    return httpClient.put<Change>(`${this.baseUrl}/${id}`, data);
  }

  // 删除变更
  async deleteChange(id: number): Promise<void> {
    return httpClient.delete(`${this.baseUrl}/${id}`);
  }

  // 获取变更统计
  async getChangeStats(): Promise<ChangeStats> {
    return httpClient.get<ChangeStats>(`${this.baseUrl}/stats`);
  }

  // 健康检查
  async healthCheck(): Promise<boolean> {
    try {
      await httpClient.get<{ status: string }>('/api/v1/health');
      return true;
    } catch {
      return false;
    }
  }
}

export const changeService = new ChangeService();
