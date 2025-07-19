import { API_BASE_URL } from './api-config';

export interface ServiceRequest {
  id: number;
  catalog_id: number;
  requester_id: number;
  status: 'pending' | 'in_progress' | 'completed' | 'rejected';
  reason: string;
  created_at: string;
  catalog?: {
    id: number;
    name: string;
    category: string;
    description: string;
    delivery_time: string;
  };
  requester?: {
    id: number;
    username: string;
    name: string;
    email: string;
    department: string;
  };
}

export interface ServiceRequestListResponse {
  requests: ServiceRequest[];
  total: number;
  page: number;
  size: number;
}

export interface CreateServiceRequestRequest {
  catalog_id: number;
  reason?: string;
}

export interface UpdateServiceRequestStatusRequest {
  status: 'pending' | 'in_progress' | 'completed' | 'rejected';
}

class ServiceRequestAPI {
  private baseURL: string;

  constructor() {
    this.baseURL = API_BASE_URL;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
    
    try {
      const response = await fetch(`${this.baseURL}${endpoint}`, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...(token && { Authorization: `Bearer ${token}` }),
          ...options.headers,
        },
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`API Error: ${response.status} ${response.statusText} - ${errorText}`);
      }

      const data = await response.json();
      
      // 检查后端返回的code字段
      if (data.code !== 0) {
        throw new Error(data.message || '请求失败');
      }
      
      return data.data; // 后端返回格式为 { code, message, data }
    } catch (error) {
      console.error('API请求失败:', error);
      throw error;
    }
  }

  // 获取当前用户的服务请求列表
  async getUserServiceRequests(params: {
    page?: number;
    size?: number;
    status?: string;
  } = {}): Promise<ServiceRequestListResponse> {
    const searchParams = new URLSearchParams();
    
    if (params.page) searchParams.append('page', params.page.toString());
    if (params.size) searchParams.append('size', params.size.toString());
    if (params.status && params.status !== 'all') searchParams.append('status', params.status);

    return this.request<ServiceRequestListResponse>(
      `/api/service-requests/me?${searchParams.toString()}`
    );
  }

  // 获取服务请求详情
  async getServiceRequestById(id: number): Promise<ServiceRequest> {
    return this.request<ServiceRequest>(`/api/service-requests/${id}`);
  }

  // 创建服务请求
  async createServiceRequest(data: CreateServiceRequestRequest): Promise<ServiceRequest> {
    return this.request<ServiceRequest>('/api/service-requests', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // 更新服务请求状态（管理员操作）
  async updateServiceRequestStatus(
    id: number, 
    data: UpdateServiceRequestStatusRequest
  ): Promise<ServiceRequest> {
    return this.request<ServiceRequest>(`/api/service-requests/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  // 健康检查
  async healthCheck(): Promise<boolean> {
    try {
      const response = await fetch(`${this.baseURL}/health`);
      return response.ok;
    } catch {
      return false;
    }
  }
}

export const serviceRequestAPI = new ServiceRequestAPI();