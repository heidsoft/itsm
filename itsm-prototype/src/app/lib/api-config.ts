// API基础配置
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// 统一响应接口
export interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
}

// 工单相关接口
export interface Ticket {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  form_fields?: Record<string, any>;
  ticket_number: string;
  requester_id: number;
  assignee_id?: number;
  created_at: string;
  updated_at: string;
  requester?: User;
  assignee?: User;
}

export interface User {
  id: number;
  username: string;
  email: string;
  name: string;
}

export interface TicketListResponse {
  tickets: Ticket[];
  total: number;
  page: number;
  size: number;
}

export interface CreateTicketRequest {
  title: string;
  description: string;
  priority: string;
  form_fields?: Record<string, any>;
  assignee_id?: number;
}

export interface UpdateStatusRequest {
  status: string;
}

export interface GetTicketsParams {
  page?: number;
  size?: number;
  status?: string;
  priority?: string;
}