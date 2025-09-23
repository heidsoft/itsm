import { httpClient } from '../http-client';

// 问题状态枚举
export enum ProblemStatus {
  OPEN = 'open',
  IN_PROGRESS = 'in_progress',
  RESOLVED = 'resolved',
  CLOSED = 'closed'
}

// 问题优先级枚举
export enum ProblemPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical'
}

// 问题接口定义
export interface Problem {
  id: number;
  title: string;
  description: string;
  status: ProblemStatus;
  priority: ProblemPriority;
  category: string;
  root_cause: string;
  impact: string;
  assignee_id?: number;
  assignee?: {
    id: number;
    name: string;
    username: string;
  };
  created_by: number;
  created_at: string;
  updated_at: string;
  tenant_id: number;
}

// 创建问题请求
export interface CreateProblemRequest {
  title: string;
  description: string;
  priority: ProblemPriority;
  category: string;
  root_cause: string;
  impact: string;
  assignee_id?: number;
}

// 更新问题请求
export interface UpdateProblemRequest {
  title?: string;
  description?: string;
  status?: ProblemStatus;
  priority?: ProblemPriority;
  category?: string;
  root_cause?: string;
  impact?: string;
  assignee_id?: number;
}

// 问题列表查询参数
export interface ListProblemsParams {
  page?: number;
  page_size?: number;
  status?: ProblemStatus;
  priority?: ProblemPriority;
  category?: string;
  keyword?: string;
  date_from?: string;
  date_to?: string;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

// 问题列表响应
export interface ListProblemsResponse {
  problems: Problem[];
  total: number;
  page: number;
  page_size: number;
}

// 问题统计响应
export interface ProblemStatsResponse {
  total: number;
  open: number;
  in_progress: number;
  resolved: number;
  closed: number;
  high_priority: number;
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
  async createProblem(data: CreateProblemRequest): Promise<{ message: string; problem_id: number }> {
    return httpClient.post<{ message: string; problem_id: number }>(this.baseUrl, data);
  }

  // 更新问题
  async updateProblem(id: number, data: UpdateProblemRequest): Promise<{ message: string; problem_id: number }> {
    return httpClient.put<{ message: string; problem_id: number }>(`${this.baseUrl}/${id}`, data);
  }

  // 删除问题
  async deleteProblem(id: number): Promise<{ message: string; problem_id: number }> {
    return httpClient.delete<{ message: string; problem_id: number }>(`${this.baseUrl}/${id}`);
  }

  // 获取问题统计
  async getProblemStats(): Promise<ProblemStatsResponse> {
    return httpClient.get<ProblemStatsResponse>(`${this.baseUrl}/stats`);
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
  getStatusLabel(status: ProblemStatus): string {
    switch (status) {
      case ProblemStatus.OPEN:
        return '待处理';
      case ProblemStatus.IN_PROGRESS:
        return '处理中';
      case ProblemStatus.RESOLVED:
        return '已解决';
      case ProblemStatus.CLOSED:
        return '已关闭';
      default:
        return status;
    }
  }

  // 获取优先级中文名称
  getPriorityLabel(priority: ProblemPriority): string {
    switch (priority) {
      case ProblemPriority.LOW:
        return '低';
      case ProblemPriority.MEDIUM:
        return '中';
      case ProblemPriority.HIGH:
        return '高';
      case ProblemPriority.CRITICAL:
        return '紧急';
      default:
        return priority;
    }
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
