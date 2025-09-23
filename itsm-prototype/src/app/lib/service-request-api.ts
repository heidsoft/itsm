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
      
      // Check backend response code field
      if (data.code !== 0) {
        throw new Error(data.message || 'Request failed');
      }
      
      return data.data; // Backend response format: { code, message, data }
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  }

  // Get current user's service request list
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

  // Get service request details
  async getServiceRequestDetails(id: number): Promise<ServiceRequest> {
    return this.request<ServiceRequest>(`/api/service-requests/${id}`);
  }

  // Create service request
  async createServiceRequest(data: CreateServiceRequestRequest): Promise<ServiceRequest> {
    return this.request<ServiceRequest>('/api/service-requests', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Update service request status (admin operation)
  async updateServiceRequestStatus(id: number, status: string, comment?: string): Promise<ServiceRequest> {
    return this.request<ServiceRequest>(`/api/service-requests/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status, comment }),
    });
  }

  // Health check
  async healthCheck(): Promise<{ status: string }> {
    return this.request<{ status: string }>('/api/health');
  }
}

export const serviceRequestAPI = new ServiceRequestAPI();