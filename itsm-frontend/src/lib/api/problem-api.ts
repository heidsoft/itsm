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
 * 问题实体的唯一前端声明：逐字段对齐后端 `dto.ProblemResponse`
 * （itsm-backend/dto/problem_dto.go），GET /api/v1/problems 的 items 与
 * GET /api/v1/problems/:id 都用它，不要在别处再声明 Problem 的形状。
 *
 * `severity`/`reporterId`/`affectedIncidents`/`relatedChanges` 后端从不返回，所以不声明；
 * `status`/`priority` 在 Ent 里是无枚举约束的 string 列，按契约声明为 string，
 * 取标签用 `@/constants/problem` 的 `problemStatusLabel` / `problemPriorityLabel`。
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
 * 集合键是 items 而不是 problems，页长的夹紧规则由后端 common 单点负责。
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

// ==================== 问题统计 ====================

/**
 * GET /api/v1/problems/stats 的分布条目，镜像 `dto.ProblemStatusCount` /
 * `dto.ProblemPriorityCount`。后端按词表基准顺序返回、只含真实存在的取值
 * （历史脏值按字典序追加），所以前端直接渲染，不再拉列表自己二次聚合。
 * status/priority 与实体一样是无枚举约束的 string：取标签用 `@/constants/problem`
 * 的守卫，词表外的值原样显示。
 */
export interface ProblemStatusCount {
  status: string;
  count: number;
}

export interface ProblemPriorityCount {
  priority: string;
  count: number;
}

/**
 * `dto.ProblemStatsResponse` 的唯一前端声明。
 * 注意 open/inProgress/resolved/closed 是折叠口径（investigating 与 in_progress 同进
 * inProgress），identified 不属于任何单值桶，因此单值之和会小于 total；
 * 需要与 total 对账时读 byStatus/byPriority。
 */
export interface ProblemStatsResponse {
  total: number;
  open: number;
  inProgress: number;
  resolved: number;
  closed: number;
  highPriority: number;
  byStatus: ProblemStatusCount[];
  byPriority: ProblemPriorityCount[];
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

// ==================== 问题评论（2026-10-21 GA 落地） ====================

/**
 * POST /api/v1/problems/:id/comments 请求体，镜像 `dto.CreateProblemCommentRequest`。
 * isInternal 区分「处理纪要」（团队内部）与「可对报障人可见的进展说明」。
 */
export interface CreateProblemCommentRequest {
  content: string;
  isInternal?: boolean;
  mentions?: number[];
  attachments?: number[];
}

/**
 * 问题评论条目，镜像 `dto.ProblemCommentResponse`。
 * userName 为服务端投影的显示名（可选：投影缺失时前端回退 `用户 #id`）。
 */
export interface ProblemComment {
  id: number;
  problemId: number;
  userId: number;
  userName?: string;
  content: string;
  isInternal: boolean;
  mentions?: number[];
  attachments?: number[];
  createdAt: string;
  updatedAt: string;
}

/** GET /api/v1/problems/:id/comments 信封，键名与工单评论列表一致。 */
export interface ListProblemCommentsResponse {
  items: ProblemComment[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// ==================== 问题 SLA（2026-10-21 GA 落地） ====================

/**
 * GET /api/v1/problems/:id/sla 响应，镜像 `dto.ProblemSLAResponse`。
 *
 * bound=false 表示该问题没有绑定 SLA 定义（租户未配置 problem 类型定义或
 * 优先级无匹配）——必须与 status='active'（已绑定且未超时）区分开，
 * 不能把「没配 SLA」渲染成「SLA 正常」。
 */
export interface ProblemSLA {
  problemId: number;
  problemNumber?: string;
  priority: string;
  bound: boolean;
  slaDefinitionId?: number;
  /** unbound / active / warning / breached / paused / met */
  status: string;
  responseDeadline?: string;
  resolutionDeadline?: string;
  firstResponseAt?: string;
  resolvedAt?: string;
  pausedDurationSeconds?: number;
}

/**
 * 问题沉淀为知识的请求体（2026-10-22 沉淀闭环落地）。
 * 镜像 dto.PromoteProblemKnowledgeRequest：全字段可选，
 * 空值用问题单内容兜底——沉淀降低的是记录成本，不是审核成本。
 */
export interface PromoteProblemKnowledgeRequest {
  title?: string;
  category?: string;
  tags?: string[];
}

/**
 * 沉淀产物的最小投影，镜像后端问题域 toKnowledgeArticleDTO
 * （刻意不搬知识 handler 的完整形态：freshness/权威性是知识域的展示职责）。
 */
export interface ProblemKnowledgeArticle {
  id: number;
  title: string;
  content: string;
  category: string;
  /** draft / published——沉淀产物恒为 draft，发布须过知识评审流 */
  status: string;
  tags: string[];
  sourceProblemId?: number;
  createdAt: string;
  updatedAt: string;
}

/** 反查列表信封：items 含草稿，空列表为 [] 而非 null */
export interface ListProblemKnowledgeResponse {
  items: ProblemKnowledgeArticle[];
  total: number;
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
   * 后端: GET /api/v1/problems/stats（无查询参数，租户取认证上下文）
   */
  static async getProblemStats(): Promise<ProblemStatsResponse> {
    return httpClient.get<ProblemStatsResponse>('/api/v1/problems/stats');
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

  // ==================== 评论（2026-10-21 GA 落地） ====================

  /**
   * 获取问题评论列表（时间正序，分页）
   * 后端: GET /api/v1/problems/:id/comments
   */
  static async getProblemComments(
    problemId: number,
    params?: { page?: number; pageSize?: number }
  ): Promise<ListProblemCommentsResponse> {
    return httpClient.get<ListProblemCommentsResponse>(
      `/api/v1/problems/${problemId}/comments`,
      params
    );
  }

  /**
   * 添加问题评论
   * 后端: POST /api/v1/problems/:id/comments
   */
  static async addProblemComment(
    problemId: number,
    req: CreateProblemCommentRequest
  ): Promise<ProblemComment> {
    return httpClient.post<ProblemComment>(`/api/v1/problems/${problemId}/comments`, req);
  }

  // ==================== SLA（2026-10-21 GA 落地） ====================

  /**
   * 获取问题 SLA 状态
   * 后端: GET /api/v1/problems/:id/sla（bound=false 表示未绑定，不是「SLA 正常」）
   */
  static async getProblemSLA(problemId: number): Promise<ProblemSLA> {
    return httpClient.get<ProblemSLA>(`/api/v1/problems/${problemId}/sla`);
  }

  // ==================== 知识沉淀闭环（2026-10-22 落地） ====================

  /**
   * 把问题沉淀为知识草稿
   * 后端: POST /api/v1/problems/:id/knowledge（dto.PromoteProblemKnowledgeRequest 镜像）
   *
   * 请求体全字段可选：空值用问题单内容兜底（标题=问题标题、分类=故障排查、
   * 内容=现象/根因/临时措施/解决方案模板化组装）。产物为 draft，发布须过知识评审流。
   * 错误语义：422/4220=问题既无根因也无解决方案（质量门）；404/4004=问题不存在或跨租户；
   * 403=无 knowledge:write；503/5003=知识仓储未装配。
   */
  static async promoteProblemKnowledge(
    problemId: number,
    req?: PromoteProblemKnowledgeRequest
  ): Promise<ProblemKnowledgeArticle> {
    return httpClient.post<ProblemKnowledgeArticle>(
      `/api/v1/problems/${problemId}/knowledge`,
      req ?? {}
    );
  }

  /**
   * 反查问题沉淀的知识文章
   * 后端: GET /api/v1/problems/:id/knowledge（dto.ListProblemKnowledgeResponse 镜像）
   *
   * 含草稿——沉淀当事人需要立刻看到沉淀结果；创建时间倒序；
   * 空列表 items 为 [] 而非 null。404/4004=问题不存在或跨租户。
   */
  static async listProblemKnowledge(problemId: number): Promise<ListProblemKnowledgeResponse> {
    return httpClient.get<ListProblemKnowledgeResponse>(
      `/api/v1/problems/${problemId}/knowledge`
    );
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

  // ==================== 关联管理 ====================

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
