import { httpClient } from '../http-client';

// 工单状态枚举
export enum TicketStatus {
  OPEN = 'open',
  IN_PROGRESS = 'in_progress',
  PENDING = 'pending',
  RESOLVED = 'resolved',
  CLOSED = 'closed',
  CANCELLED = 'cancelled'
}

// 工单优先级枚举
export enum TicketPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  URGENT = 'urgent'
}

// 工单类型枚举
export enum TicketType {
  INCIDENT = 'incident',
  SERVICE_REQUEST = 'service_request',
  PROBLEM = 'problem',
  CHANGE = 'change'
}

// 工单接口定义
export interface Ticket {
  id: number;
  ticket_number: string;
  title: string;
  description: string;
  status: TicketStatus;
  priority: TicketPriority;
  type: TicketType;
  category: string;
  subcategory?: string;
  assignee_id?: number;
  assignee?: {
    id: number;
    name: string;
    username: string;
    email: string;
  };
  requester_id: number;
  requester?: {
    id: number;
    name: string;
    username: string;
    email: string;
  };
  created_at: string;
  updated_at: string;
  resolved_at?: string;
  closed_at?: string;
  sla_deadline?: string;
  tags: string[];
  attachments: number;
  comments: number;
  tenant_id: number;
  source?: string;
  impact?: string;
  urgency?: string;
  business_value?: string;
  custom_fields?: Record<string, any>;
}

// 创建工单请求
export interface CreateTicketRequest {
  title: string;
  description: string;
  priority: TicketPriority;
  type: TicketType;
  category: string;
  subcategory?: string;
  assignee_id?: number;
  tags?: string[];
  source?: string;
  impact?: string;
  urgency?: string;
  business_value?: string;
  custom_fields?: Record<string, any>;
}

// 更新工单请求
export interface UpdateTicketRequest {
  title?: string;
  description?: string;
  status?: TicketStatus;
  priority?: TicketPriority;
  type?: TicketType;
  category?: string;
  subcategory?: string;
  assignee_id?: number;
  tags?: string[];
  source?: string;
  impact?: string;
  urgency?: string;
  business_value?: string;
  custom_fields?: Record<string, any>;
}

// 工单列表查询参数
export interface ListTicketsParams {
  page?: number;
  page_size?: number;
  status?: TicketStatus;
  priority?: TicketPriority;
  type?: TicketType;
  category?: string;
  assignee_id?: number;
  requester_id?: number;
  keyword?: string;
  date_from?: string;
  date_to?: string;
  tags?: string[];
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

// 工单列表响应
export interface ListTicketsResponse {
  tickets: Ticket[];
  total: number;
  page: number;
  page_size: number;
}

// 工单统计响应
export interface TicketStatsResponse {
  total: number;
  open: number;
  in_progress: number;
  pending: number;
  resolved: number;
  closed: number;
  high_priority: number;
  urgent: number;
  overdue: number;
  sla_breach: number;
}

// 工单分配请求
export interface AssignTicketRequest {
  assignee_id: number;
  reason?: string;
}

// 工单状态变更请求
export interface ChangeTicketStatusRequest {
  status: TicketStatus;
  comment?: string;
  resolution?: string;
  category?: string;
  subcategory?: string;
}

// 工单评论接口
export interface TicketComment {
  id: number;
  ticket_id: number;
  user_id: number;
  user_name: string;
  content: string;
  created_at: string;
  updated_at: string;
  is_internal: boolean;
}

// 工单附件接口
export interface TicketAttachment {
  id: number;
  ticket_id: number;
  filename: string;
  original_name: string;
  file_size: number;
  mime_type: string;
  uploaded_by: number;
  uploaded_at: string;
  url: string;
}

// 工单活动日志接口
export interface TicketActivity {
  id: number;
  ticket_id: number;
  user_id: number;
  user_name: string;
  action: string;
  details: string;
  timestamp: string;
  old_value?: string;
  new_value?: string;
}

// 工单管理API服务类
class TicketService {
  private readonly baseUrl = '/api/v1/tickets';

  // 获取工单列表
  async listTickets(params: ListTicketsParams = {}): Promise<ListTicketsResponse> {
    return httpClient.get<ListTicketsResponse>(this.baseUrl, params);
  }

  // 获取工单详情
  async getTicket(id: number): Promise<Ticket> {
    return httpClient.get<Ticket>(`${this.baseUrl}/${id}`);
  }

  // 创建工单
  async createTicket(data: CreateTicketRequest): Promise<{ message: string; ticket_id: number }> {
    return httpClient.post<{ message: string; ticket_id: number }>(this.baseUrl, data);
  }

  // 更新工单
  async updateTicket(id: number, data: UpdateTicketRequest): Promise<{ message: string; ticket_id: number }> {
    return httpClient.put<{ message: string; ticket_id: number }>(`${this.baseUrl}/${id}`, data);
  }

  // 删除工单
  async deleteTicket(id: number): Promise<{ message: string; ticket_id: number }> {
    return httpClient.delete<{ message: string; ticket_id: number }>(`${this.baseUrl}/${id}`);
  }

  // 获取工单统计
  async getTicketStats(): Promise<TicketStatsResponse> {
    return httpClient.get<TicketStatsResponse>(`${this.baseUrl}/stats`);
  }

  // 分配工单
  async assignTicket(id: number, data: AssignTicketRequest): Promise<{ message: string; ticket_id: number }> {
    return httpClient.post<{ message: string; ticket_id: number }>(`${this.baseUrl}/${id}/assign`, data);
  }

  // 变更工单状态
  async changeTicketStatus(id: number, data: ChangeTicketStatusRequest): Promise<{ message: string; ticket_id: number }> {
    return httpClient.post<{ message: string; ticket_id: number }>(`${this.baseUrl}/${id}/status`, data);
  }

  // 获取工单评论
  async getTicketComments(id: number): Promise<TicketComment[]> {
    return httpClient.get<TicketComment[]>(`${this.baseUrl}/${id}/comments`);
  }

  // 添加工单评论
  async addTicketComment(id: number, content: string, isInternal: boolean = false): Promise<{ message: string; comment_id: number }> {
    return httpClient.post<{ message: string; comment_id: number }>(`${this.baseUrl}/${id}/comments`, {
      content,
      is_internal: isInternal
    });
  }

  // 获取工单附件
  async getTicketAttachments(id: number): Promise<TicketAttachment[]> {
    return httpClient.get<TicketAttachment[]>(`${this.baseUrl}/${id}/attachments`);
  }

  // 上传工单附件
  async uploadTicketAttachment(id: number, file: File): Promise<{ message: string; attachment_id: number }> {
    const formData = new FormData();
    formData.append('file', file);
    return httpClient.post<{ message: string; attachment_id: number }>(`${this.baseUrl}/${id}/attachments`, formData);
  }

  // 删除工单附件
  async deleteTicketAttachment(id: number, attachmentId: number): Promise<{ message: string; attachment_id: number }> {
    return httpClient.delete<{ message: string; attachment_id: number }>(`${this.baseUrl}/${id}/attachments/${attachmentId}`);
  }

  // 获取工单活动日志
  async getTicketActivities(id: number): Promise<TicketActivity[]> {
    return httpClient.get<TicketActivity[]>(`${this.baseUrl}/${id}/activities`);
  }

  // 获取状态标签颜色
  getStatusColor(status: TicketStatus): string {
    switch (status) {
      case TicketStatus.OPEN:
        return 'processing';
      case TicketStatus.IN_PROGRESS:
        return 'processing';
      case TicketStatus.PENDING:
        return 'warning';
      case TicketStatus.RESOLVED:
        return 'success';
      case TicketStatus.CLOSED:
        return 'default';
      case TicketStatus.CANCELLED:
        return 'error';
      default:
        return 'default';
    }
  }

  // 获取优先级标签颜色
  getPriorityColor(priority: TicketPriority): string {
    switch (priority) {
      case TicketPriority.LOW:
        return 'green';
      case TicketPriority.MEDIUM:
        return 'orange';
      case TicketPriority.HIGH:
        return 'red';
      case TicketPriority.URGENT:
        return 'red';
      default:
        return 'default';
    }
  }

  // 获取类型标签颜色
  getTypeColor(type: TicketType): string {
    switch (type) {
      case TicketType.INCIDENT:
        return 'red';
      case TicketType.SERVICE_REQUEST:
        return 'blue';
      case TicketType.PROBLEM:
        return 'orange';
      case TicketType.CHANGE:
        return 'purple';
      default:
        return 'default';
    }
  }

  // 获取状态中文名称
  getStatusLabel(status: TicketStatus): string {
    switch (status) {
      case TicketStatus.OPEN:
        return '待处理';
      case TicketStatus.IN_PROGRESS:
        return '处理中';
      case TicketStatus.PENDING:
        return '等待中';
      case TicketStatus.RESOLVED:
        return '已解决';
      case TicketStatus.CLOSED:
        return '已关闭';
      case TicketStatus.CANCELLED:
        return '已取消';
      default:
        return status;
    }
  }

  // 获取优先级中文名称
  getPriorityLabel(priority: TicketPriority): string {
    switch (priority) {
      case TicketPriority.LOW:
        return '低';
      case TicketPriority.MEDIUM:
        return '中';
      case TicketPriority.HIGH:
        return '高';
      case TicketPriority.URGENT:
        return '紧急';
      default:
        return priority;
    }
  }

  // 获取类型中文名称
  getTypeLabel(type: TicketType): string {
    switch (type) {
      case TicketType.INCIDENT:
        return '事件';
      case TicketType.SERVICE_REQUEST:
        return '服务请求';
      case TicketType.PROBLEM:
        return '问题';
      case TicketType.CHANGE:
        return '变更';
      default:
        return type;
    }
  }

  // 健康检查
  async healthCheck(): Promise<boolean> {
    try {
      await httpClient.get(`${this.baseUrl}/health`);
      return true;
    } catch (error) {
      return false;
    }
  }
}

export const ticketService = new TicketService();
export default TicketService;
