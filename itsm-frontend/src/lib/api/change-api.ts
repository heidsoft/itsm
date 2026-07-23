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
  | 'scheduled'
  | 'in_progress'
  | 'completed'
  | 'failed'
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
  impactScope: ChangeImpact;
  riskLevel: ChangeRisk;
  plannedStartDate?: string;
  plannedEndDate?: string;
  implementationPlan: string;
  rollbackPlan: string;
  affectedCis: string[];
  relatedTickets: string[];
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
  impactScope: ChangeImpact;
  riskLevel: ChangeRisk;
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
  affectedCis: string[];
  relatedTickets: string[];
  createdAt: string;
  updatedAt: string;
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
  inProgress: number;
  completed: number;
  rolledBack: number;
  rejected: number;
  cancelled: number;
}

// 变更审批请求
export interface ChangeApprovalRequest {
  status?: ChangeStatus;
  comment?: string;
}

// 变更审批记录
export interface ChangeApproval {
  id: number;
  changeId: number;
  approverId: number;
  approverName: string;
  status: ChangeStatus;
  comment?: string;
  approvedAt?: string;
  createdAt: string;
}

// 变更风险评估数据
export interface RiskAssessmentData {
  riskLevel: ChangeRisk;
  riskDescription: string;
  impactAnalysis: string;
  mitigationMeasures: string;
  contingencyPlan: string;
  riskOwner: string;
  riskScore?: number;
  riskFactors?: string[];
}

// 影响分析数据
export interface ImpactAnalysisData {
  businessImpact: string;
  technicalImpact: string;
  userImpact: string;
  affectedSystems: string[];
  affectedUsers: number;
  estimatedDowntime: number;
  dataRiskLevel: string;
  serviceDependencies: string[];
  backupStrategy: string;
  recoveryPlan: string;
  impactScore?: number;
}

export interface ChangeCMDBImpactSummary {
  changeId: number;
  totalAffectedCIs: number;
  criticalCICount: number;
  highRiskDependencyCount: number;
  openIncidentCount: number;
  recommendedRiskLevel: string;
  recommendedImpactScope: string;
  requiresCAB: boolean;
  requiresBackoutPlan: boolean;
  workflowHints: string[];
  itilPractices: string[];
  affectedCIs: number[];
}

// ==================== PIR (Post-Implementation Review) 类型定义 ====================
// PIR总体结果
export type PIROverallResult = 'successful' | 'partially_successful' | 'failed';


// PIR请求
export interface CreatePIRRequest {
  overallResult: PIROverallResult;
  objectivesAchieved: boolean;
  successSummary?: string;
  issuesEncountered?: string;
  lessonsLearned?: string;
  improvementRecommendations?: string;
  actualStartTime?: string;
  actualEndTime?: string;
  rollbackPerformed: boolean;
  rollbackReason?: string;
}

// PIR更新请求
export interface UpdatePIRRequest {
  overallResult?: PIROverallResult;
  objectivesAchieved?: boolean;
  successSummary?: string;
  issuesEncountered?: string;
  lessonsLearned?: string;
  improvementRecommendations?: string;
}

// PIR响应
export interface PIRResponse {
  id: number;
  changeId: number;
  changeTitle: string;
  reviewerId: number;
  reviewerName: string;
  overallResult: PIROverallResult;
  objectivesAchieved: boolean;
  successSummary?: string;
  issuesEncountered?: string;
  lessonsLearned?: string;
  improvementRecommendations?: string;
  actualStartTime?: string;
  actualEndTime?: string;
  actualDurationMinutes: number;
  rollbackPerformed: boolean;
  rollbackReason?: string;
  tenantId: number;
  reviewDate: string;
  createdAt: string;
  updatedAt: string;
}


// PIR列表响应
export interface PIRListResponse {
  total: number;
  items: PIRResponse[];
}

// 日历视图项
export interface ChangeCalendarItem {
  id: number;
  title: string;
  changeNumber: string;
  status: string;
  riskLevel: string;
  category: string;
  plannedStart: string;
  plannedEnd: string;
  assigneeName: string;
}

// 变更API类
export class ChangeApi {
  // 获取变更列表
  static async getChanges(params?: {
    page?: number;
    pageSize?: number;
    status?: ChangeStatus;
    type?: ChangeType;
    priority?: ChangePriority;
    risk?: string;
    search?: string;
  }): Promise<ChangeListResponse> {
    return httpClient.get<ChangeListResponse>('/api/v1/changes', params && {
      ...params,
      riskLevel: params.risk,
      risk: undefined,
    });
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
  static async assignChange(id: number, assigneeId: number): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/assign`, { assigneeId });
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
  static async createFromTemplate(
    templateId: number,
    data: Partial<ChangeRequest>
  ): Promise<Change> {
    return httpClient.post(`/api/v1/changes/templates/${templateId}/create`, data);
  }

  // 批量操作变更
  static async batchUpdateChanges(ids: number[], action: string, data?: unknown): Promise<void> {
    return httpClient.post('/api/v1/changes/batch', { ids, action, data });
  }

  // 导出变更数据
  static async exportChanges(params?: {
    format?: 'excel' | 'pdf' | 'csv';
    status?: ChangeStatus;
    startDate?: string;
    endDate?: string;
  }): Promise<Blob> {
    return httpClient.request<Blob>({
      url: '/api/v1/changes/export',
      method: 'GET',
      params,
      responseType: 'blob',
    });
  }

  // 获取变更关联工单
  static async getRelatedTickets(id: number): Promise<any[]> {
    return httpClient.get(`/api/v1/changes/${id}/tickets`);
  }

  // 获取日历视图数据
  static async getCalendar(params?: {
    startDate?: string;
    endDate?: string;
    status?: string;
  }): Promise<{ items: ChangeCalendarItem[]; total: number }> {
    return httpClient.get('/api/v1/changes/calendar', params);
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

  // 获取基于 CMDB 的变更影响摘要
  static async getCMDBImpactSummary(id: number): Promise<ChangeCMDBImpactSummary> {
    return httpClient.get(`/api/v1/changes/${id}/cmdb-impact`);
  }

  // 更新风险评估
  static async updateRiskAssessment(id: number, data: RiskAssessmentData): Promise<void> {
    return this.updateRisk(id, data);
  }

  static async updateRisk(id: number, data: RiskAssessmentData): Promise<void> {
    return httpClient.put(`/api/v1/changes/${id}/risk`, data);
  }

  // 获取变更实施日志
  static async getImplementationLogs(id: number): Promise<any[]> {
    return httpClient.get(`/api/v1/changes/${id}/logs`);
  }

  // ==================== PIR (Post-Implementation Review) ====================


  // PIR总体结果类型
  static async getPIRs(params?: {
    page?: number;
    pageSize?: number;
    result?: 'successful' | 'partially_successful' | 'failed' | '全部';
  }): Promise<PIRListResponse> {
    return httpClient.get<PIRListResponse>('/api/v1/changes/pirs', params as Record<string, unknown>);
  }

  // 获取变更关联的PIR
  static async getPIR(changeId: number): Promise<PIRResponse | null> {
    try {
      return await httpClient.get<PIRResponse>(`/api/v1/changes/${changeId}/pir`);
    } catch (error: any) {
      if (error?.response?.status === 404) {
        return null;
      }
      throw error;
    }
  }

  // 创建PIR
  static async createPIR(changeId: number, data: CreatePIRRequest): Promise<PIRResponse> {
    return httpClient.post<PIRResponse>(`/api/v1/changes/${changeId}/pir`, data);
  }

  // 更新PIR
  static async updatePIR(pirId: number, data: UpdatePIRRequest): Promise<PIRResponse> {
    return httpClient.put<PIRResponse>(`/api/v1/changes/pir/${pirId}`, data);
  }

  // 删除PIR
  static async deletePIR(pirId: number): Promise<void> {
    return httpClient.delete(`/api/v1/changes/pir/${pirId}`);
  }

  // ==================== 兼容别名（旧代码使用） ====================

  /** @deprecated 使用 getChangeApprovals */
  static async getApprovalSummary(id: number) {
    return this.getChangeApprovals(id);
  }
}
