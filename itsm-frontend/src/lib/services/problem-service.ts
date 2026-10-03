import { httpClient } from '@/lib/api/http-client';
import {
  ProblemPriority,
  ProblemStatus,
  problemPriorityLabel,
  problemStatusLabel,
} from '@/constants/problem';

// 问题实体只有一份形状：@/lib/api/problem-api 的 Problem（逐字段镜像后端
// dto.ProblemResponse）。本文件此前另有一份，含后端从不返回的 `assignee` 对象
// —— 报表页因此把处理人显示成「未分配」，即使问题已指派（台账 E4-6d）。
import type { Problem } from '@/lib/api/problem-api';
export type { Problem };

// 状态/优先级枚举同样只有 @/constants/problem 一份。本文件此前自有一份只列
// open/in_progress/resolved/closed 的副本，缺 investigating/identified，
// 用它做参数类型会把合法的后端值挡在类型之外。消费方请直接从常量层取用。

// 创建问题请求
export interface CreateProblemRequest {
  title: string;
  description: string;
  priority: ProblemPriority;
  category: string;
  rootCause: string;
  impact: string;
  assigneeId?: number;
}

// 更新问题请求
export interface UpdateProblemRequest {
  title?: string;
  description?: string;
  status?: ProblemStatus;
  priority?: ProblemPriority;
  category?: string;
  rootCause?: string;
  impact?: string;
  assigneeId?: number;
}

// 问题列表查询参数
// 与 dto.ListProblemsRequest 逐字段一致：后端在 2026-10-03（E4-6c）删掉了 sortBy/
// sortOrder/dateFrom/dateTo —— 它们既不在 handler 的 filters 里，也不在 repository 的
// 查询条件里，传与不传结果完全相同，属假契约。
export interface ListProblemsParams {
  page?: number;
  pageSize?: number;
  status?: ProblemStatus;
  priority?: ProblemPriority;
  category?: string;
  keyword?: string;
}

// 问题列表响应：GET /api/v1/problems 的 {items,total,page,pageSize,totalPages} 信封，
// 与 ProblemApi.ProblemListResponse 是同一套契约，不再有 problems 别名键。
export interface ListProblemsResponse {
  items: Problem[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// 问题统计响应
export interface ProblemStatsResponse {
  total: number;
  open: number;
  inProgress: number;
  resolved: number;
  closed: number;
  highPriority: number;
}

// 问题管理API服务类
class ProblemService {
  private readonly baseUrl = '/api/v1/problems';

  // 获取问题列表
  async listProblems(params: ListProblemsParams = {}): Promise<ListProblemsResponse> {
    return httpClient.get<ListProblemsResponse>(this.baseUrl, params);
  }

  // 获取问题详情
  async getProblem(id: number): Promise<Problem> {
    return httpClient.get<Problem>(`${this.baseUrl}/${id}`);
  }

  // 创建问题
  async createProblem(
    data: CreateProblemRequest
  ): Promise<{ message: string; problemId: number }> {
    return httpClient.post<{ message: string; problemId: number }>(this.baseUrl, data);
  }

  // 更新问题
  async updateProblem(
    id: number,
    data: UpdateProblemRequest
  ): Promise<{ message: string; problemId: number }> {
    return httpClient.put<{ message: string; problemId: number }>(`${this.baseUrl}/${id}`, data);
  }

  // 删除问题
  async deleteProblem(id: number): Promise<{ message: string; problemId: number }> {
    return httpClient.delete<{ message: string; problemId: number }>(`${this.baseUrl}/${id}`);
  }

  // 获取问题统计
  async getProblemStats(): Promise<ProblemStatsResponse> {
    return httpClient.get<ProblemStatsResponse>(`${this.baseUrl}/stats`);
  }

  // 添加问题评论
  async addProblemComment(
    problemId: number,
    content: string
  ): Promise<{ message: string; commentId: number }> {
    return httpClient.post<{ message: string; commentId: number }>(
      `${this.baseUrl}/${problemId}/comments`,
      { content }
    );
  }

  // 获取问题评论列表
  async getProblemComments(problemId: number): Promise<{
    comments: Array<{
      id: number;
      problemId: number;
      userId: number;
      content: string;
      createdAt: string;
      user?: {
        id: number;
        name: string;
        username: string;
      };
    }>;
    total: number;
  }> {
    return httpClient.get(`${this.baseUrl}/${problemId}/comments`);
  }

  // 获取状态标签颜色
  getStatusColor(status: ProblemStatus): string {
    switch (status) {
      case ProblemStatus.OPEN:
        return 'processing';
      case ProblemStatus.IN_PROGRESS:
        return 'processing';
      case ProblemStatus.RESOLVED:
        return 'success';
      case ProblemStatus.CLOSED:
        return 'default';
      default:
        return 'default';
    }
  }

  // 获取优先级标签颜色
  getPriorityColor(priority: ProblemPriority): string {
    switch (priority) {
      case ProblemPriority.LOW:
        return 'green';
      case ProblemPriority.MEDIUM:
        return 'orange';
      case ProblemPriority.HIGH:
        return 'red';
      case ProblemPriority.CRITICAL:
        return 'red';
      default:
        return 'default';
    }
  }

  // 获取状态中文名称
  getStatusLabel(status: string): string {
    return problemStatusLabel(status);
  }

  // 获取优先级中文名称
  getPriorityLabel(priority: string): string {
    return problemPriorityLabel(priority);
  }

  getStatusText(status: ProblemStatus): string {
    switch (status) {
      case ProblemStatus.OPEN:
        return 'Open';
      case ProblemStatus.IN_PROGRESS:
        return 'In Progress';
      case ProblemStatus.RESOLVED:
        return 'Resolved';
      case ProblemStatus.CLOSED:
        return 'Closed';
      default:
        return 'Unknown';
    }
  }

  getPriorityText(priority: ProblemPriority): string {
    switch (priority) {
      case ProblemPriority.LOW:
        return 'Low';
      case ProblemPriority.MEDIUM:
        return 'Medium';
      case ProblemPriority.HIGH:
        return 'High';
      case ProblemPriority.CRITICAL:
        return 'Critical';
      default:
        return 'Unknown';
    }
  }
}

export const problemService = new ProblemService();
export default ProblemService;
