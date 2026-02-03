import { API_BASE_URL } from '@/app/lib/api-config';
import { getAccessToken, getTenantCode } from '@/lib/auth/token-storage';

export interface ServiceRequest {
  id: number;
  catalog_id: number;
  requester_id: number;
  status:
    | 'submitted'
    | 'manager_approved'
    | 'it_approved'
    | 'security_approved'
    | 'provisioning'
    | 'delivered'
    | 'failed'
    | 'rejected'
    | 'cancelled';
  title?: string;
  reason?: string;
  form_data?: Record<string, unknown>;
  cost_center?: string;
  data_classification?: 'public' | 'internal' | 'confidential';
  needs_public_ip?: boolean;
  source_ip_whitelist?: string[];
  expire_at?: string | null;
  compliance_ack?: boolean;
  current_level?: number;
  total_levels?: number;
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
  title?: string;
  reason?: string;
  form_data?: Record<string, unknown>;
  cost_center?: string;
  data_classification?: 'public' | 'internal' | 'confidential';
  needs_public_ip?: boolean;
  source_ip_whitelist?: string[];
  expire_at?: string;
  compliance_ack: boolean;
}

export interface UpdateServiceRequestStatusRequest {
  status:
    | 'submitted'
    | 'manager_approved'
    | 'it_approved'
    | 'security_approved'
    | 'provisioning'
    | 'delivered'
    | 'failed'
    | 'rejected'
    | 'cancelled';
}

export interface ServiceRequestApprovalActionRequest {
  action: 'approve' | 'reject';
  comment?: string;
}

class ServiceRequestAPI {
  private baseURL: string;

  constructor() {
    this.baseURL = API_BASE_URL;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const token = getAccessToken();
    const tenantCode = getTenantCode();
    
    try {
      const response = await fetch(`${this.baseURL}${endpoint}`, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...(token && { Authorization: `Bearer ${token}` }),
          ...(tenantCode && { 'X-Tenant-Code': tenantCode }),
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
      // console.error('API request failed:', error);
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
      `/api/v1/service-requests/me?${searchParams.toString()}`
    );
  }

  // Get pending approvals (approver inbox)
  async getPendingApprovals(params: { page?: number; size?: number } = {}): Promise<ServiceRequestListResponse> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.append('page', params.page.toString());
    if (params.size) searchParams.append('size', params.size.toString());
    const qs = searchParams.toString();
    return this.request<ServiceRequestListResponse>(
      `/api/v1/service-requests/approvals/pending${qs ? `?${qs}` : ''}`
    );
  }

  // Get service request details
  async getServiceRequestDetails(id: number): Promise<ServiceRequest> {
    return this.request<ServiceRequest>(`/api/v1/service-requests/${id}`);
  }

  // Create service request
  async createServiceRequest(data: CreateServiceRequestRequest): Promise<ServiceRequest> {
    return this.request<ServiceRequest>('/api/v1/service-requests', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Update service request status (admin operation)
  async updateServiceRequestStatus(id: number, status: string, comment?: string): Promise<ServiceRequest> {
    return this.request<ServiceRequest>(`/api/v1/service-requests/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status, comment }),
    });
  }

  // Apply approval action on current step
  async applyApprovalAction(id: number, payload: ServiceRequestApprovalActionRequest): Promise<ServiceRequest> {
    return this.request<ServiceRequest>(`/api/v1/service-requests/${id}/approvals`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  // Health check
  async healthCheck(): Promise<{ status: string }> {
    // backend exposes public health endpoint under /api/v1/health
    return this.request<{ status: string }>('/api/v1/health');
  }

  // ==================== Provisioning Tasks ====================
  
  // Start provisioning for a service request
  async startProvisioning(serviceRequestId: number): Promise<{ task: ProvisioningTask }> {
    return this.request<{ task: ProvisioningTask }>(
      `/api/v1/service-requests/${serviceRequestId}/provision`,
      {
        method: 'POST',
      }
    );
  }

  // List provisioning tasks for a service request
  async listProvisioningTasks(serviceRequestId: number): Promise<ProvisioningTask[]> {
    return this.request<ProvisioningTask[]>(
      `/api/v1/service-requests/${serviceRequestId}/provisioning-tasks`
    );
  }

  // Execute a provisioning task
  async executeProvisioningTask(taskId: number): Promise<ProvisioningTask> {
    return this.request<ProvisioningTask>(
      `/api/v1/provisioning-tasks/${taskId}/execute`,
      {
        method: 'POST',
      }
    );
  }
}

export interface ProvisioningTask {
  id: number;
  service_request_id: number;
  provider: string;
  resource_type: string;
  status: 'pending' | 'running' | 'succeeded' | 'failed';
  payload?: Record<string, unknown>;
  result?: Record<string, unknown>;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export const serviceRequestAPI = new ServiceRequestAPI();