import { httpClient } from '@/lib/api/http-client';

// 事件状态枚举
export enum IncidentStatus {
  NEW = 'new',
  IN_PROGRESS = 'in_progress',
  RESOLVED = 'resolved',
  CLOSED = 'closed',
  CANCELLED = 'cancelled'
}

// 事件优先级枚举
export enum IncidentPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical'
}

// 事件来源枚举
export enum IncidentSource {
  EMAIL = 'email',
  PHONE = 'phone',
  PORTAL = 'portal',
  MONITORING = 'monitoring',
  API = 'api'
}

// 事件接口定义
export interface Incident {
  id: number;
  title: string;
  description: string;
  status: IncidentStatus;
  priority: IncidentPriority;
  severity: string;
  category: string;
  assignee_id?: number;
  assignee?: {
    id: number;
    name: string;
    username: string;
  };
  reporter_id: number;
  reporter?: {
    id: number;
    name: string;
    username: string;
  };
  created_at: string;
  updated_at: string;
  resolved_at?: string;
  closed_at?: string;
  tenant_id: number;
  source?: IncidentSource;
  impact?: string;
  urgency?: string;
  affected_services?: string[];
  root_cause?: string;
  resolution?: string;
}

// 创建事件请求
export interface CreateIncidentRequest {
  title: string;
  description: string;
  priority: IncidentPriority;
  category: string;
  assignee_id?: number;
  source?: IncidentSource;
  impact?: string;
  urgency?: string;
  affected_services?: string[];
}

// 更新事件请求
export interface UpdateIncidentRequest {
  title?: string;
  description?: string;
  status?: IncidentStatus;
  priority?: IncidentPriority;
  severity?: string;
  category?: string;
  assignee_id?: number;
  source?: IncidentSource;
  impact?: string;
  urgency?: string;
  affected_services?: string[];
  root_cause?: string;
  resolution?: string;
}

// 事件列表查询参数
export interface ListIncidentsParams {
  page?: number;
  page_size?: number;
  status?: IncidentStatus;
  priority?: IncidentPriority;
  category?: string;
  assignee_id?: number;
  reporter_id?: number;
  keyword?: string;
  query?: string; // 为了兼容 global-search-api
  date_from?: string;
  date_to?: string;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

// 事件列表响应
export interface ListIncidentsResponse {
  incidents: Incident[];
  items: Incident[]; // 为了兼容 global-search-api
  total: number;
  page: number;
  page_size: number;
}

// 事件统计响应
export interface IncidentStatsResponse {
  total: number;
  open: number;
  in_progress: number;
  resolved: number;
  closed: number;
  high_priority: number;
  critical: number;
  mttr: number; // 平均修复时间
}

// 事件管理API服务类
class IncidentService {
  private readonly baseUrl = '/api/v1/incidents';

  // 获取事件列表
  async listIncidents(params: ListIncidentsParams = {}): Promise<ListIncidentsResponse> {
    const response = await httpClient.get<ListIncidentsResponse>(this.baseUrl, params);
    // 确保 items 存在，兼容 global-search-api
    if (!response.items && response.incidents) {
      response.items = response.incidents;
    }
    return response;
  }

  // 兼容 global-search-api 的别名方法
  async getIncidents(params: ListIncidentsParams = {}): Promise<ListIncidentsResponse> {
    return this.listIncidents(params);
  }

  // 获取事件详情
  async getIncident(id: number): Promise<Incident> {
    return httpClient.get<Incident>(`${this.baseUrl}/${id}`);
  }

  // 创建事件
  async createIncident(data: CreateIncidentRequest): Promise<{ message: string; incident_id: number }> {
    return httpClient.post<{ message: string; incident_id: number }>(this.baseUrl, data);
  }

  // 更新事件
  async updateIncident(id: number, data: UpdateIncidentRequest): Promise<{ message: string; incident_id: number }> {
    return httpClient.put<{ message: string; incident_id: number }>(`${this.baseUrl}/${id}`, data);
  }

  // 删除事件
  async deleteIncident(id: number): Promise<{ message: string; incident_id: number }> {
    return httpClient.delete<{ message: string; incident_id: number }>(`${this.baseUrl}/${id}`);
  }

  // 获取事件统计
  async getIncidentStats(): Promise<IncidentStatsResponse> {
    return httpClient.get<IncidentStatsResponse>(`${this.baseUrl}/stats`);
  }

  // 获取状态标签颜色
  getStatusColor(status: IncidentStatus): string {
    switch (status) {
      case IncidentStatus.NEW:
        return 'blue';
      case IncidentStatus.IN_PROGRESS:
        return 'processing';
      case IncidentStatus.RESOLVED:
        return 'success';
      case IncidentStatus.CLOSED:
        return 'default';
      case IncidentStatus.CANCELLED:
        return 'error';
      default:
        return 'default';
    }
  }

  // 获取优先级标签颜色
  getPriorityColor(priority: IncidentPriority): string {
    switch (priority) {
      case IncidentPriority.LOW:
        return 'green';
      case IncidentPriority.MEDIUM:
        return 'orange';
      case IncidentPriority.HIGH:
        return 'red';
      case IncidentPriority.CRITICAL:
        return 'purple';
      default:
        return 'default';
    }
  }

  // 获取状态中文名称
  getStatusLabel(status: IncidentStatus): string {
    switch (status) {
      case IncidentStatus.NEW:
        return '新建';
      case IncidentStatus.IN_PROGRESS:
        return '处理中';
      case IncidentStatus.RESOLVED:
        return '已解决';
      case IncidentStatus.CLOSED:
        return '已关闭';
      case IncidentStatus.CANCELLED:
        return '已取消';
      default:
        return status;
    }
  }

  // 获取优先级中文名称
  getPriorityLabel(priority: IncidentPriority): string {
    switch (priority) {
      case IncidentPriority.LOW:
        return '低';
      case IncidentPriority.MEDIUM:
        return '中';
      case IncidentPriority.HIGH:
        return '高';
      case IncidentPriority.CRITICAL:
        return '严重';
      default:
        return priority;
    }
  }

  getStatusText(status: IncidentStatus): string {
    switch (status) {
      case IncidentStatus.NEW:
        return 'New';
      case IncidentStatus.IN_PROGRESS:
        return 'In Progress';
      case IncidentStatus.RESOLVED:
        return 'Resolved';
      case IncidentStatus.CLOSED:
        return 'Closed';
      case IncidentStatus.CANCELLED:
        return 'Cancelled';
      default:
        return 'Unknown';
    }
  }

  getPriorityText(priority: IncidentPriority): string {
    switch (priority) {
      case IncidentPriority.LOW:
        return 'Low';
      case IncidentPriority.MEDIUM:
        return 'Medium';
      case IncidentPriority.HIGH:
        return 'High';
      case IncidentPriority.CRITICAL:
        return 'Critical';
      default:
        return 'Unknown';
    }
  }
}

export const incidentService = new IncidentService();
export default IncidentService;

