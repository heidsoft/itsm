/**
 * 问题管理 API 包装器
 * 兼容旧代码使用
 */

import { httpClient } from './http-client';

// ==================== 问题趋势相关类型 ====================

export interface ProblemTrendRequest {
  startDate: string;
  endDate: string;
  category?: string;
}

export interface CategoryCount {
  category: string;
  count: number;
}

export interface MonthlyCount {
  month: string;
  count: number;
  resolved: number;
  open: number;
}

export interface ProblemTrendData {
  period: string;
  totalProblems: number;
  resolvedProblems: number;
  openProblems: number;
  resolutionRate: number;
  avgResolutionTimeHours: number;
  categoryBreakdown: Record<string, number>;
  priorityBreakdown: Record<string, number>;
  trendDirection: string;
  topCategories: CategoryCount[];
  monthlyTrend: MonthlyCount[];
}

export interface ProblemHotspotsData {
  periodStart: string;
  periodEnd: string;
  categoryBreakdown: Record<string, number>;
  priorityBreakdown: Record<string, number>;
  hotspots: string[];
  avgPerCategory: number;
}

/**
 * 问题实体的唯一形状：逐字段镜像后端 `dto.ProblemResponse`
 * （itsm-backend/dto/problem_dto.go:59），GET /api/v1/problems 的 items 与
 * GET /api/v1/problems/:id 都用它。
 *
 * 2026-10-03（台账 E4-6d）收敛前有同一份实体的三套声明：本文件这份带后端**从不返回**
 * 的 `severity`/`reporterId`/`affectedIncidents`/`relatedChanges`，`types/biz/problem.ts`
 * 与 `lib/services/problem-service.ts` 各有一份不含这些字段、但把 `status`/`priority`
 * 定义成枚举的版本，消费点只能写 `resp.items as unknown as Problem[]`。
 * `status`/`priority` 在 Ent 里是无枚举约束的 string 列，因此这里按契约声明为 string，
 * 取标签请用 `@/constants/problem` 的 `problemStatusLabel` / `problemPriorityLabel`。
 */
export interface Problem {
  id: number;
  /** 后端 omitempty：由 handlers/incident 侧构造的 ProblemResponse 不填此字段。 */
  problemNumber?: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  category: string;
  rootCause: string;
  workaround: string;
  resolution: string;
  impact: string;
  assigneeId?: number;
  assigneeName?: string;
  createdBy: number;
  createdByName?: string;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
  associatedTickets?: AssociatedItem[];
  associatedIncidents?: AssociatedItem[];
  associatedChanges?: AssociatedItem[];
}

/**
 * GET /api/v1/problems 的列表信封：{items,total,page,pageSize,totalPages}，只有一套契约。
 * 2026-10-03 后端（E4-6c）把集合键从 problems 收敛为 items，并把三处各自夹紧的分页规则
 * 收给 common 单点。
 */
export interface ProblemListResponse {
  items: Problem[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// GET /api/v1/problems 的请求契约，与后端 dto.ListProblemsRequest 逐字段一致。
// 分页夹紧由后端 common.GetPaginationFromQuery 单点负责：page 缺省 1、pageSize 缺省 20，
// 越界值回落默认页长，因此前端不得再自己猜一套页长规则。
export interface ProblemListParams {
  page?: number;
  pageSize?: number;
  status?: string;
  priority?: string;
  category?: string;
  keyword?: string;
}

// ==================== 问题关联 ====================

export type RelatedType = 'ticket' | 'incident' | 'change';

/**
 * `dto.AssociatedItemResponse` 的镜像：`{id,title,status}` 必填，
 * `number`/`type` 后端是 omitempty，因此这里声明为可选。
 */
export interface AssociatedItem {
  id: number;
  title: string;
  status: string;
  number?: string;
  type?: RelatedType;
}

export interface ProblemAssociations {
  tickets: AssociatedItem[];
  incidents: AssociatedItem[];
  changes: AssociatedItem[];
}

export interface ProblemAssociationRequest {
  relatedType: RelatedType;
  relatedIds: number[];
}

export interface ProblemRemoveAssociationRequest {
  relatedType: RelatedType;
  relatedId: number;
}

export class ProblemApi {
  /**
   * 获取问题列表
   * 后端: GET /api/v1/problems（query 契约见 ProblemListParams，响应见 ProblemListResponse）
   */
  static async getProblems(params?: ProblemListParams): Promise<ProblemListResponse> {
    return httpClient.get<ProblemListResponse>('/api/v1/problems', params);
  }

  /**
   * 获取问题详情
   */
  static async getProblem(id: number): Promise<Problem> {
    return httpClient.get(`/api/v1/problems/${id}`);
  }

  /**
   * 创建问题
   */
  static async createProblem(data: Partial<Problem>): Promise<Problem> {
    return httpClient.post('/api/v1/problems', data);
  }

  /**
   * 更新问题
   */
  static async updateProblem(id: number, data: Partial<Problem>): Promise<Problem> {
    return httpClient.put(`/api/v1/problems/${id}`, data);
  }

  /**
   * 删除问题
   */
  static async deleteProblem(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/problems/${id}`);
  }

  /**
   * 获取问题统计
   */
  static async getProblemStats(params?: any): Promise<any> {
    return httpClient.get('/api/v1/problems/stats', params);
  }

  /**
   * 调查问题
   * 后端: POST /api/v1/problems/:id/investigate
   */
  static async investigateProblem(id: number, _data: unknown): Promise<Problem> {
    return httpClient.post<Problem>(`/api/v1/problems/${id}/investigate`);
  }

  /**
   * 记录根本原因
   * 后端: PUT /api/v1/problems/:id/root-cause
   */
  static async recordRootCause(id: number, rootCause: string): Promise<Problem> {
    return httpClient.put<Problem>(`/api/v1/problems/${id}/root-cause`, { rootCause });
  }

  /**
   * 提供解决方案
   * 后端: PUT /api/v1/problems/:id/solution
   */
  static async provideSolution(id: number, solution: string): Promise<Problem> {
    return httpClient.put<Problem>(`/api/v1/problems/${id}/solution`, { solution });
  }

  /**
   * 关闭问题
   * 后端: POST /api/v1/problems/:id/close
   */
  static async closeProblem(id: number, resolution: string): Promise<Problem> {
    return httpClient.post<Problem>(`/api/v1/problems/${id}/close`, { resolution });
  }

  // ==================== 趋势分析 ====================

  /**
   * 获取问题趋势分析
   */
  static async getTrends(params: ProblemTrendRequest): Promise<ProblemTrendData> {
    return httpClient.get<ProblemTrendData>('/api/v1/problems/trend', params);
  }

  /**
   * 获取问题热点分析
   */
  static async getHotspots(params: ProblemTrendRequest): Promise<ProblemHotspotsData> {
    return httpClient.get<ProblemHotspotsData>('/api/v1/problems/hotspots', params);
  }

  // ==================== 关联管理（P0 修复暴露的 TS 错误） ====================

  /**
   * 获取问题关联（工单/事件/变更）
   */
  static async getAssociations(problemId: number): Promise<ProblemAssociations> {
    return httpClient.get<ProblemAssociations>(`/api/v1/problems/${problemId}/associations`);
  }

  /**
   * 添加问题关联
   */
  static async addAssociation(problemId: number, req: ProblemAssociationRequest): Promise<void> {
    return httpClient.post(`/api/v1/problems/${problemId}/associations`, req);
  }

  /**
   * 移除问题关联
   */
  static async removeAssociation(
    problemId: number,
    req: ProblemRemoveAssociationRequest
  ): Promise<void> {
    return httpClient.request({
      method: 'DELETE',
      url: `/api/v1/problems/${problemId}/associations`,
      data: req,
    });
  }
}

export default ProblemApi;
