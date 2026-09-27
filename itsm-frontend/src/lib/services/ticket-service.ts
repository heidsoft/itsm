import { httpClient } from '@/lib/api/http-client';
import { TicketStatus, TicketPriority } from '@/constants/taxonomy';
import type {
  Ticket,
  TicketListResponse,
  TicketType,
  TicketSource,
  CreateTicketRequest,
  UpdateTicketRequest,
  AssignTicketRequest,
  TicketExportRequest,
  GetTicketsParams,
  TicketStatsResponse,
} from '@/lib/api/ticket-api';

/**
 * 工单契约的唯一声明处是 `@/lib/api/ticket-api`（逐字段对齐后端 dto/ticket_dto.go，
 * 状态词表对齐 itsm-backend/common/constants.go 的工单状态机）。
 * 本模块只负责 HTTP 调用封装，禁止再自行声明状态/优先级/类型词表或请求 DTO。
 */
export { TicketStatus, TicketPriority };
export type {
  Ticket,
  TicketListResponse,
  TicketType,
  TicketSource,
  CreateTicketRequest,
  UpdateTicketRequest,
  TicketStatsResponse,
  // 筛选/分页参数的唯一名字：与后端 dto.ListTicketsRequest 一致，
  // 不再保留 size / search / created_after 等第二套别名。
  GetTicketsParams,
};

// 工单评论接口
export interface TicketComment {
  id: number;
  ticketId: number;
  userId: number;
  userName: string;
  content: string;
  createdAt: string;
  updatedAt: string;
  isInternal: boolean;
}

// 工单附件接口
export interface TicketAttachment {
  id: number;
  ticketId: number;
  filename: string;
  originalName: string;
  fileSize: number;
  mimeType: string;
  uploadedBy: number;
  uploadedAt: string;
  url: string;
}

// 工单管理API服务类
class TicketService {
  private readonly baseUrl = '/api/v1/tickets';

  // 获取工单列表
  async listTickets(params: GetTicketsParams = {}): Promise<TicketListResponse> {
    return httpClient.get<TicketListResponse>(this.baseUrl, params);
  }

  // 获取工单详情
  async getTicket(id: number): Promise<Ticket> {
    return httpClient.get<Ticket>(`${this.baseUrl}/${id}`);
  }

  // 创建工单
  async createTicket(data: CreateTicketRequest): Promise<Ticket> {
    return httpClient.post<Ticket>(this.baseUrl, data);
  }

  // 更新工单
  async updateTicket(
    id: number,
    data: UpdateTicketRequest
  ): Promise<{ message: string; ticketId: number }> {
    return httpClient.put<{ message: string; ticketId: number }>(`${this.baseUrl}/${id}`, data);
  }

  // 删除工单
  async deleteTicket(id: number): Promise<{ message: string; ticketId: number }> {
    return httpClient.delete<{ message: string; ticketId: number }>(`${this.baseUrl}/${id}`);
  }

  // 获取工单统计
  async getTicketStats(): Promise<TicketStatsResponse> {
    return httpClient.get<TicketStatsResponse>(`${this.baseUrl}/stats`);
  }

  // 分配工单
  async assignTicket(
    id: number,
    data: AssignTicketRequest
  ): Promise<{ message: string; ticketId: number }> {
    return httpClient.post<{ message: string; ticketId: number }>(
      `${this.baseUrl}/${id}/assign`,
      data
    );
  }

  // 获取工单评论
  async getTicketComments(id: number): Promise<TicketComment[]> {
    return httpClient.get<TicketComment[]>(`${this.baseUrl}/${id}/comments`);
  }

  // 添加工单评论
  async addTicketComment(
    id: number,
    content: string,
    isInternal: boolean = false
  ): Promise<{ message: string; commentId: number }> {
    return httpClient.post<{ message: string; commentId: number }>(
      `${this.baseUrl}/${id}/comments`,
      {
        content,
        isInternal: isInternal,
      }
    );
  }

  // 获取工单附件
  async getTicketAttachments(id: number): Promise<TicketAttachment[]> {
    return httpClient.get<TicketAttachment[]>(`${this.baseUrl}/${id}/attachments`);
  }

  // 上传工单附件
  async uploadTicketAttachment(
    id: number,
    file: File
  ): Promise<{ message: string; attachmentId: number }> {
    const formData = new FormData();
    formData.append('file', file);
    return httpClient.post<{ message: string; attachmentId: number }>(
      `${this.baseUrl}/${id}/attachments`,
      formData
    );
  }

  // 删除工单附件
  async deleteTicketAttachment(
    id: number,
    attachmentId: number
  ): Promise<{ message: string; attachmentId: number }> {
    return httpClient.delete<{ message: string; attachmentId: number }>(
      `${this.baseUrl}/${id}/attachments/${attachmentId}`
    );
  }

  // 导出工单（后端 POST /tickets/export，请求体为 dto.TicketExportRequest）
  async exportTickets(data: TicketExportRequest): Promise<Blob> {
    return httpClient.request<Blob>({
      method: 'POST',
      url: `${this.baseUrl}/export`,
      data,
      responseType: 'blob',
    });
  }
}

export const ticketService = new TicketService();
export default TicketService;
