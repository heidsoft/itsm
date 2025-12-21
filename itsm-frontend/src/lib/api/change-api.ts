/**
 * 变更管理 API 服务
 */

import { httpClient } from './http-client';

// 变更状态类型
export type ChangeStatus = 
  | 'draft' 
  | 'pending' 
  | 'approved' 
  | 'rejected' 
  | 'in_progress' 
  | 'completed' 
  | 'rolled_back' 
  | 'cancelled';

// 变更类型
export type ChangeType = 'normal' | 'standard' | 'emergency';

// 变更优先级
export type ChangePriority = 'low' | 'medium' | 'high' | 'critical';

// 变更影响范围
export type ChangeImpact = 'low' | 'medium' | 'high';

// 变更风险等级
export type ChangeRisk = 'low' | 'medium' | 'high';

// 变更请求接口
export interface ChangeRequest {
  title: string;
  description: string;
  justification: string;
  type: ChangeType;
  priority: ChangePriority;
  impact_scope: ChangeImpact;
  risk_level: ChangeRisk;
  planned_start_date?: string;
  planned_end_date?: string;
  implementation_plan: string;
  rollback_plan: string;
  affected_cis: string[];
  related_tickets: string[];
}

// 变更响应接口
export interface Change {
  id: number;
  title: string;
  description: string;
  justification: string;
  type: ChangeType;
  status: ChangeStatus;
  priority: ChangePriority;
  impact_scope: ChangeImpact;
  risk_level: ChangeRisk;
  assignee_id?: number;
  assignee_name?: string;
  created_by: number;
  created_by_name: string;
  tenant_id: number;
  planned_start_date?: string;
  planned_end_date?: string;
  actual_start_date?: string;
  actual_end_date?: string;
  implementation_plan: string;
  rollback_plan: string;
  affected_cis: string[];
  related_tickets: string[];
  created_at: string;
  updated_at: string;
}

// 变更列表响应
export interface ChangeListResponse {
  total: number;
  changes: Change[];
}

// 变更统计响应
export interface ChangeStatsResponse {
  total: number;
  pending: number;
  approved: number;
  in_progress: number;
  completed: number;
  rolled_back: number;
  rejected: number;
  cancelled: number;
}

// 变更审批请求
export interface ChangeApprovalRequest {
  status: ChangeStatus;
  comment?: string;
}

// 变更审批记录
export interface ChangeApproval {
  id: number;
  change_id: number;
  approver_id: number;
  approver_name: string;
  status: ChangeStatus;
  comment?: string;
  approved_at?: string;
  created_at: string;
}

// 变更风险评估数据
export interface RiskAssessmentData {
  risk_level: ChangeRisk;
  risk_description: string;
  impact_analysis: string;
  mitigation_measures: string;
  contingency_plan: string;
  risk_owner: string;
  risk_score?: number;
  risk_factors?: string[];
}

// 影响分析数据
export interface ImpactAnalysisData {
  business_impact: string;
  technical_impact: string;
  user_impact: string;
  affected_systems: string[];
  affected_users: number;
  estimated_downtime: number;
  data_risk_level: string;
  service_dependencies: string[];
  backup_strategy: string;
  recovery_plan: string;
  impact_score?: number;
}

// 变更API类
export class ChangeApi {
  // 获取变更列表
  static async getChanges(params?: {
    page?: number;
    page_size?: number;
    status?: ChangeStatus;
    type?: ChangeType;
    priority?: ChangePriority;
    search?: string;
  }): Promise<ChangeListResponse> {
    return httpClient.get<ChangeListResponse>('/api/v1/changes', params);
  }

  // 获取单个变更
  static async getChange(id: number): Promise<Change> {
    return httpClient.get<Change>(`/api/v1/changes/${id}`);
  }

  // 创建变更
  static async createChange(data: ChangeRequest): Promise<Change> {
    return httpClient.post<Change>('/api/v1/changes', data);
  }

  // 更新变更
  static async updateChange(id: number, data: Partial<ChangeRequest>): Promise<Change> {
    return httpClient.put<Change>(`/api/v1/changes/${id}`, data);
  }

  // 删除变更
  static async deleteChange(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/changes/${id}`);
  }

  // 获取变更统计
  static async getChangeStats(): Promise<ChangeStatsResponse> {
    return httpClient.get<ChangeStatsResponse>('/api/v1/changes/stats');
  }

  // 提交变更审批
  static async submitForApproval(id: number): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/submit`);
  }

  // 审批变更
  static async approveChange(id: number, data: ChangeApprovalRequest): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/approve`, data);
  }

  // 拒绝变更
  static async rejectChange(id: number, data: ChangeApprovalRequest): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/reject`, data);
  }

  // 开始实施变更
  static async startImplementation(id: number): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/start`);
  }

  // 完成变更实施
  static async completeImplementation(id: number): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/complete`);
  }

  // 回滚变更
  static async rollbackChange(id: number, reason?: string): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/rollback`, { reason });
  }

  // 取消变更
  static async cancelChange(id: number, reason?: string): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/cancel`, { reason });
  }

  // 分配变更
  static async assignChange(id: number, assignee_id: number): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/assign`, { assignee_id });
  }

  // 获取变更审批历史
  static async getChangeApprovals(id: number): Promise<ChangeApproval[]> {
    return httpClient.get<ChangeApproval[]>(`/api/v1/changes/${id}/approvals`);
  }

  // 获取变更模板
  static async getChangeTemplates(): Promise<any[]> {
    return httpClient.get('/api/v1/changes/templates');
  }

  // 从模板创建变更
  static async createFromTemplate(templateId: number, data: Partial<ChangeRequest>): Promise<Change> {
    return httpClient.post(`/api/v1/changes/templates/${templateId}/create`, data);
  }

  // 批量操作变更
  static async batchUpdateChanges(ids: number[], action: string, data?: any): Promise<void> {
    return httpClient.post('/api/v1/changes/batch', { ids, action, data });
  }

  // 导出变更数据
  static async exportChanges(params?: {
    format?: 'excel' | 'pdf' | 'csv';
    status?: ChangeStatus;
    start_date?: string;
    end_date?: string;
  }): Promise<Blob> {
    return httpClient.get('/api/v1/changes/export', params, {
      responseType: 'blob',
    });
  }

  // 获取变更关联工单
  static async getRelatedTickets(id: number): Promise<any[]> {
    return httpClient.get(`/api/v1/changes/${id}/tickets`);
  }

  // 获取变更影响分析
  static async getImpactAnalysis(id: number): Promise<ImpactAnalysisData> {
    return httpClient.get(`/api/v1/changes/${id}/impact`);
  }

  // 更新影响分析
  static async updateImpactAnalysis(id: number, data: ImpactAnalysisData): Promise<void> {
    return httpClient.put(`/api/v1/changes/${id}/impact`, data);
  }

  // 获取变更风险评估
  static async getRiskAssessment(id: number): Promise<RiskAssessmentData> {
    return httpClient.get(`/api/v1/changes/${id}/risk`);
  }

  // 更新风险评估
  static async updateRiskAssessment(id: number, data: RiskAssessmentData): Promise<void> {
    return httpClient.put(`/api/v1/changes/${id}/risk`, data);
  }

  // 获取变更实施日志
  static async getImplementationLogs(id: number): Promise<any[]> {
    return httpClient.get(`/api/v1/changes/${id}/logs`);
  }
}