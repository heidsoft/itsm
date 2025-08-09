import { BaseService, type PaginatedResponse, type PaginationParams, type BaseEntity } from "./api-service";

// 工单相关类型定义
export interface Ticket extends BaseEntity {
  ticket_number: string;
  title: string;
  description: string;
  status: TicketStatus;
  priority: TicketPriority;
  category: string;
  assignee_id?: string;
  assignee_name?: string;
  requester_id: string;
  requester_name: string;
  sla_deadline?: string;
  resolution_time?: number;
  tags?: string[];
  attachments?: Attachment[];
  comments?: Comment[];
}

export type TicketStatus = 
  | 'open'
  | 'in_progress'
  | 'pending'
  | 'resolved'
  | 'closed'
  | 'cancelled';

export type TicketPriority = 
  | 'low'
  | 'medium'
  | 'high'
  | 'critical';

export interface CreateTicketRequest {
  title: string;
  description: string;
  priority: TicketPriority;
  category: string;
  assignee_id?: string;
  tags?: string[];
}

export interface UpdateTicketRequest {
  title?: string;
  description?: string;
  status?: TicketStatus;
  priority?: TicketPriority;
  category?: string;
  assignee_id?: string;
  tags?: string[];
}

export interface TicketFilterParams extends PaginationParams {
  status?: TicketStatus;
  priority?: TicketPriority;
  category?: string;
  assignee_id?: string;
  requester_id?: string;
  created_after?: string;
  created_before?: string;
  tags?: string[];
  search?: string;
}

export interface Attachment {
  id: string;
  filename: string;
  size: number;
  mime_type: string;
  url: string;
  uploaded_at: string;
}

export interface Comment {
  id: string;
  content: string;
  author_id: string;
  author_name: string;
  created_at: string;
  updated_at: string;
}

// 工单服务类
export class TicketService extends BaseService {
  
  /**
   * 获取工单列表
   * 对应后端: GET /api/v1/tickets
   */
  async getTickets(params: TicketFilterParams): Promise<PaginatedResponse<Ticket>> {
    try {
      const queryParams = new URLSearchParams();
      
      // 分页参数
      if (params.page) queryParams.append('page', params.page.toString());
      if (params.size) queryParams.append('size', params.size.toString());
      if (params.sort) queryParams.append('sort', params.sort);
      if (params.order) queryParams.append('order', params.order);
      
      // 过滤参数
      if (params.status) queryParams.append('status', params.status);
      if (params.priority) queryParams.append('priority', params.priority);
      if (params.category) queryParams.append('category', params.category);
      if (params.assignee_id) queryParams.append('assignee_id', params.assignee_id);
      if (params.requester_id) queryParams.append('requester_id', params.requester_id);
      if (params.created_after) queryParams.append('created_after', params.created_after);
      if (params.created_before) queryParams.append('created_before', params.created_before);
      if (params.tags) params.tags.forEach(tag => queryParams.append('tags', tag));
      if (params.search) queryParams.append('search', params.search);
      
      const url = `${this.getUrl('/tickets')}?${queryParams.toString()}`;
      return await this.client.get<PaginatedResponse<Ticket>>(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 获取单个工单详情
   * 对应后端: GET /api/v1/tickets/{id}
   */
  async getTicket(id: string): Promise<Ticket> {
    try {
      const url = this.getUrl(`/tickets/${id}`);
      return await this.client.get<Ticket>(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 创建工单
   * 对应后端: POST /api/v1/tickets
   */
  async createTicket(data: CreateTicketRequest): Promise<Ticket> {
    try {
      const url = this.getUrl('/tickets');
      return await this.client.post<Ticket>(url, data);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 更新工单
   * 对应后端: PUT /api/v1/tickets/{id}
   */
  async updateTicket(id: string, data: UpdateTicketRequest): Promise<Ticket> {
    try {
      const url = this.getUrl(`/tickets/${id}`);
      return await this.client.put<Ticket>(url, data);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 删除工单
   * 对应后端: DELETE /api/v1/tickets/{id}
   */
  async deleteTicket(id: string): Promise<void> {
    try {
      const url = this.getUrl(`/tickets/${id}`);
      await this.client.delete(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 批量更新工单状态
   * 对应后端: PATCH /api/v1/tickets/batch
   */
  async batchUpdateStatus(ticketIds: string[], status: TicketStatus): Promise<void> {
    try {
      const url = this.getUrl('/tickets/batch');
      await this.client.patch(url, { ticket_ids: ticketIds, status });
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 分配工单
   * 对应后端: POST /api/v1/tickets/{id}/assign
   */
  async assignTicket(id: string, assigneeId: string): Promise<Ticket> {
    try {
      const url = this.getUrl(`/tickets/${id}/assign`);
      return await this.client.post<Ticket>(url, { assignee_id: assigneeId });
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 添加工单评论
   * 对应后端: POST /api/v1/tickets/{id}/comments
   */
  async addComment(id: string, content: string): Promise<Comment> {
    try {
      const url = this.getUrl(`/tickets/${id}/comments`);
      return await this.client.post<Comment>(url, { content });
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 获取工单评论列表
   * 对应后端: GET /api/v1/tickets/{id}/comments
   */
  async getComments(id: string, params?: PaginationParams): Promise<PaginatedResponse<Comment>> {
    try {
      const queryParams = new URLSearchParams();
      if (params?.page) queryParams.append('page', params.page.toString());
      if (params?.size) queryParams.append('size', params.size.toString());
      
      const url = `${this.getUrl(`/tickets/${id}/comments`)}?${queryParams.toString()}`;
      return await this.client.get<PaginatedResponse<Comment>>(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 上传工单附件
   * 对应后端: POST /api/v1/tickets/{id}/attachments
   */
  async uploadAttachment(id: string, file: File): Promise<Attachment> {
    try {
      const url = this.getUrl(`/tickets/${id}/attachments`);
      const formData = new FormData();
      formData.append('file', file);
      
      return await this.client.post<Attachment>(url, formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 获取工单统计信息
   * 对应后端: GET /api/v1/tickets/stats
   */
  async getTicketStats(params?: {
    tenant_id?: string;
    date_from?: string;
    date_to?: string;
  }): Promise<{
    total: number;
    open: number;
    in_progress: number;
    resolved: number;
    closed: number;
    avg_resolution_time: number;
    sla_compliance_rate: number;
  }> {
    try {
      const queryParams = new URLSearchParams();
      if (params?.tenant_id) queryParams.append('tenant_id', params.tenant_id);
      if (params?.date_from) queryParams.append('date_from', params.date_from);
      if (params?.date_to) queryParams.append('date_to', params.date_to);
      
      const url = `${this.getUrl('/tickets/stats')}?${queryParams.toString()}`;
      return await this.client.get(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 导出工单数据
   * 对应后端: GET /api/v1/tickets/export
   */
  async exportTickets(params: TicketFilterParams, format: 'csv' | 'excel' = 'csv'): Promise<Blob> {
    try {
      const queryParams = new URLSearchParams();
      
      // 添加过滤参数
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          if (Array.isArray(value)) {
            value.forEach(v => queryParams.append(key, v));
          } else {
            queryParams.append(key, value.toString());
          }
        }
      });
      
      queryParams.append('format', format);
      
      const url = `${this.getUrl('/tickets/export')}?${queryParams.toString()}`;
      const response = await this.client.get(url, {
        responseType: 'blob',
      });
      
      return response as Blob;
    } catch (error) {
      this.handleError(error);
    }
  }
}

// 创建服务实例
export const ticketService = new TicketService(); 