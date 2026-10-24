/**
 * 变更管理 API 服务
 */

import { httpClient, ApiRequestError } from './http-client';
import type { RollbackStep, RollbackStepStatus } from '@/types/biz/change';

/**
 * 判定「资源不存在」。
 *
 * 直接写 `error?.response?.status === 404` 是在猜 axios 的内部结构：本项目
 * 的 `httpClient` 抛的是 `ApiRequestError`（status/code 在实例上），没有
 * `response` 字段——这个判断从来就没命中过，getPIR 的 404 分支是死代码。
 */
function isNotFound(error: unknown): boolean {
  return error instanceof ApiRequestError && error.status === 404;
}

// 变更状态类型：与后端 dto.ChangeStatus / ent/schema/change.go 的枚举逐值一致。
// 历史值 submitted 只在持久化层存在，后端读映射会把它转成 pending，
// 因此前端类型不得出现 submitted，也不得出现后端不存在的 implementing。
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
  | 'cancelled'
  | 'closed';

// 变更类型
export type ChangeType = 'normal' | 'standard' | 'emergency';

/**
 * 标准变更模板摘要（`GET /api/v1/changes/templates`，后端待注册）。
 *
 * 字段刻意只声明到「够用」为止：端点尚未上线，声明一个想象中的完整契约只会把
 * 猜测固化成类型。后端补齐时按 `dto.StandardChangeResponse` 扩展。
 */
export interface ChangeTemplate {
  id: number;
  name: string;
  description?: string;
  changeType?: ChangeType;
  riskLevel?: ChangeRisk;
}

/** 变更关联工单（`GET /api/v1/changes/:id/tickets`，后端待注册）。 */
export interface ChangeRelatedTicket {
  id: number;
  title?: string;
  status?: string;
  priority?: string;
}

/** 变更实施日志条目（`GET /api/v1/changes/:id/logs`，后端待注册）。 */
export interface ChangeImplementationLog {
  id: number;
  changeId?: number;
  action?: string;
  operator?: string;
  createdAt?: string;
}

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
  /** 实施结果说明：失败原因 / 回滚原因 / 完成说明（后端 outcome_note）。 */
  outcomeNote?: string;
  /** 回滚清单（含逐步执行结果）。仅详情接口下发。 */
  rollbackSteps?: RollbackStep[];
  affectedCis: string[];
  relatedTickets: string[];
  createdAt: string;
  updatedAt: string;
}

// 后端标准列表信封：集合只在 items 下
export interface ChangeListResponse {
  items: Change[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// 变更类型分布中的一项，来自后端真实 GROUP BY type。
// type 声明为 string 而不是 ChangeType：Ent 的 type 是无约束字符串列，
// 后端会原样返回枚举外的历史值（按字典序追加在 canonical 类型之后），
// 丢弃它们会让分布总数与 total 对不上账。展示标签见 constants/change.ts。
export interface ChangeTypeCount {
  type: string;
  count: number;
}

// 变更统计响应：逐字段镜像后端 dto.ChangeStatsResponse。
// 本接口是变更统计的唯一契约来源。types/biz/change.ts 只做 re-export；
// lib/services/change-service.ts 里那份互不一致的副本（含后端从不返回的
// implementing）已随该重复模块一并删除。
export interface ChangeStatsResponse {
  total: number;
  draft: number;
  pending: number;
  approved: number;
  scheduled: number;
  inProgress: number;
  completed: number;
  failed: number;
  rolledBack: number;
  rejected: number;
  cancelled: number;
  closed: number;
  byType: ChangeTypeCount[];
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
  requiresReview: boolean;
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
//
// 后端 service/pir_service.go 用 Count + Offset/Limit 真实分页，因此信封带全套
// page/pageSize/totalPages；调用方必须按这五个键读取，不得再对 items 做二次分页。
export interface PIRListResponse {
  items: PIRResponse[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
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
  // 后端 query 契约：page/pageSize/status/search/riskLevel（camelCase，只有一套名字）。
  // type/priority 目前不被后端识别，发出去会被静默忽略，不得当成已实现的筛选能力。
  static async getChanges(params?: {
    page?: number;
    pageSize?: number;
    status?: ChangeStatus;
    type?: ChangeType;
    priority?: ChangePriority;
    riskLevel?: string;
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
    // 后端需要 JSON body（空 body 会触发 ShouldBindJSON EOF → 400），显式传空对象
    return httpClient.post(`/api/v1/changes/${id}/submit`, {});
  }

  // 审批变更。
  // 返回变更的当前快照：会签/阈值/多级链未集齐时后端返回 200 但状态仍是 pending
  // （本次票已记录）。调用方必须看 status，不能把"调用成功"当成"已批准"。
  static async approveChange(id: number, data: ChangeApprovalRequest): Promise<Change> {
    return httpClient.post<Change>(`/api/v1/changes/${id}/approve`, data);
  }

  // 拒绝变更
  static async rejectChange(id: number, data: ChangeApprovalRequest): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/reject`, data);
  }

  // 排期变更（普通变更 approved → scheduled 的必经步骤）
  static async scheduleChange(id: number): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/schedule`, {});
  }

  // 开始实施变更
  static async startImplementation(id: number): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/start`, {});
  }

  // 完成变更实施
  static async completeImplementation(id: number): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/complete`, {});
  }

  // 关闭变更（已完成 → 已关闭）
  static async closeChange(id: number): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/close`, {});
  }

  // 回滚变更（reason 必填：后端会拒收没有原因的 rolled_back）
  static async rollbackChange(id: number, reason: string): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/rollback`, { reason });
  }

  // 标记实施失败（reason 必填：失败原因写入 outcome_note，进 PIR 与审计）
  static async failChange(id: number, reason: string): Promise<void> {
    return httpClient.post(`/api/v1/changes/${id}/fail`, { reason });
  }

  // ---- 回滚清单（失败可回退的可验证形态） ----

  // 读取回滚清单（按执行顺序，含每步确认状态/备注/确认人/时间）
  static async getRollbackSteps(id: number): Promise<RollbackStep[]> {
    return httpClient.get(`/api/v1/changes/${id}/rollback-steps`);
  }

  /**
   * 覆盖式编排回滚清单（按执行顺序）。
   * 仅在 draft/pending/approved/scheduled 允许——实施开始后清单即执行依据，
   * 后端会以 409 拒绝改写（要说明偏差请把步骤标记为 skipped 并写依据）。
   */
  static async replaceRollbackSteps(id: number, steps: string[]): Promise<RollbackStep[]> {
    return httpClient.put(`/api/v1/changes/${id}/rollback-steps`, { steps });
  }

  /**
   * 确认某一步的回滚执行结果。
   * skipped / failed 必须带 note（不适用依据 / 失败现象）；
   * 传 'pending' 表示撤销误点。
   */
  static async resolveRollbackStep(
    id: number,
    stepId: number,
    status: RollbackStepStatus,
    note?: string
  ): Promise<RollbackStep> {
    return httpClient.post(`/api/v1/changes/${id}/rollback-steps/${stepId}`, {
      status,
      note: note ?? '',
    });
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
  //
  // ⚠️ 后端 `router/change_routes.go` 目前**没有**注册 `/changes/templates`，
  // 该缺口由 `@/config/product-capabilities` 的 `changeClassification` 禁用契约
  // 跟踪（标准变更模板实际落在 `/api/v1/standard-changes`）。本方法当前无调用方，
  // 保留是为了后端补齐后能直接对接；不要在新代码里依赖它。
  static async getChangeTemplates(): Promise<ChangeTemplate[]> {
    return httpClient.get<ChangeTemplate[]>('/api/v1/changes/templates');
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

  // 获取变更关联工单（后端 /changes/:id/tickets 尚未注册，见 getChangeTemplates 说明）
  static async getRelatedTickets(id: number): Promise<ChangeRelatedTicket[]> {
    return httpClient.get<ChangeRelatedTicket[]>(`/api/v1/changes/${id}/tickets`);
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

  // 获取变更实施日志（后端 /changes/:id/logs 尚未注册，见 getChangeTemplates 说明）
  static async getImplementationLogs(id: number): Promise<ChangeImplementationLog[]> {
    return httpClient.get<ChangeImplementationLog[]>(`/api/v1/changes/${id}/logs`);
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
    } catch (error) {
      // 变更未建 PIR 是正常业务状态（404），不是异常；其余错误照常上抛。
      // 这里只按「是不是 404」判定，不假设错误一定有 response 字段。
      if (isNotFound(error)) {
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
