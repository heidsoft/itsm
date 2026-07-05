import { API_BASE_URL } from '@/lib/api/api-config';
import { getTenantCode } from '@/lib/auth/token-storage';

export interface ServiceRequest {
  id: number;
  catalogId: number;
  requesterId: number;
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
  formData?: Record<string, unknown>;
  costCenter?: string;
  dataClassification?: 'public' | 'internal' | 'confidential' | 'restricted';
  needsPublicIp?: boolean;
  sourceIpWhitelist?: string[];
  expireAt?: string | null;
  complianceAck?: boolean;
  currentLevel?: number;
  totalLevels?: number;
  createdAt: string;
  catalog?: {
    id: number;
    name: string;
    category: string;
    description: string;
    deliveryTime: string;
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
  catalogId: number;
  title?: string;
  reason?: string;
  formData?: Record<string, unknown>;
  costCenter?: string;
  dataClassification?: 'public' | 'internal' | 'confidential' | 'restricted';
  needsPublicIp?: boolean;
  sourceIpWhitelist?: string[];
  expireAt?: string;
  complianceAck: boolean;
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
    const tenantCode = getTenantCode();

    try {
      const response = await fetch(`${this.baseURL}${endpoint}`, {
        ...options,
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
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

  private normalizeRequest(raw: any): ServiceRequest {
    const catalogId = raw?.catalogId;
    const requesterId = raw?.requesterId;
    const createdAt = raw?.createdAt;
    const updatedAt = raw?.updatedAt;
    return {
      ...raw,
      catalogId,
      requesterId,
      ciId: raw?.ciId,
      formData: raw?.formData ?? {},
      costCenter: raw?.costCenter,
      dataClassification: raw?.dataClassification,
      needsPublicIp: raw?.needsPublicIp ?? raw?.needsPublicIP,
      sourceIpWhitelist: raw?.sourceIpWhitelist ?? raw?.sourceIPWhitelist,
      expireAt: raw?.expireAt,
      complianceAck: raw?.complianceAck,
      currentLevel: raw?.currentLevel,
      totalLevels: raw?.totalLevels,
      createdAt,
      updatedAt,
      catalog: raw?.catalog || {
        id: catalogId,
        name: raw?.serviceName || (catalogId ? `服务 #${catalogId}` : '未知服务'),
        category: raw?.category || '',
        description: raw?.reason || '',
        deliveryTime: raw?.deliveryTime || raw?.deliveryTime || '',
      },
      requester: raw?.requester || {
        id: requesterId,
        username: raw?.requesterName || raw?.requestedByName || '',
        name: raw?.requesterName || raw?.requestedByName || (requesterId ? `用户 #${requesterId}` : '-'),
        email: raw?.requestedByEmail || '',
        department: '',
      },
    } as ServiceRequest;
  }

  private normalizeList(raw: ServiceRequestListResponse): ServiceRequestListResponse {
    const requests = (raw.requests || (raw as any).items || []).map(item => this.normalizeRequest(item));
    return {
      requests,
      total: raw.total || 0,
      page: raw.page || 1,
      size: raw.size || (raw as any).pageSize || requests.length,
    };
  }

  // Get current user's service request list
  async getUserServiceRequests(
    params: {
      page?: number;
      size?: number;
      status?: string;
    } = {}
  ): Promise<ServiceRequestListResponse> {
    const searchParams = new URLSearchParams();

    if (params.page) searchParams.append('page', params.page.toString());
    if (params.size) searchParams.append('size', params.size.toString());
    if (params.status && params.status !== 'all') searchParams.append('status', params.status);

    const resp = await this.request<ServiceRequestListResponse>(
      `/api/v1/service-requests/me?${searchParams.toString()}`
    );
    return this.normalizeList(resp);
  }

  // Get pending approvals (approver inbox)
  async getPendingApprovals(
    params: { page?: number; size?: number } = {}
  ): Promise<ServiceRequestListResponse> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.append('page', params.page.toString());
    if (params.size) searchParams.append('size', params.size.toString());
    const qs = searchParams.toString();
    const resp = await this.request<ServiceRequestListResponse>(
      `/api/v1/service-requests/approvals/pending${qs ? `?${qs}` : ''}`
    );
    return this.normalizeList(resp);
  }

  // Get service request details
  async getServiceRequestDetails(id: number): Promise<ServiceRequest> {
    const resp = await this.request<ServiceRequest>(`/api/v1/service-requests/${id}`);
    return this.normalizeRequest(resp);
  }

  // Create service request
  async createServiceRequest(data: CreateServiceRequestRequest): Promise<ServiceRequest> {
    const resp = await this.request<ServiceRequest>('/api/v1/service-requests', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    return this.normalizeRequest(resp);
  }

  // Update service request status (admin operation)
  async updateServiceRequestStatus(
    id: number,
    status: string,
    comment?: string
  ): Promise<ServiceRequest> {
    const resp = await this.request<ServiceRequest>(`/api/v1/service-requests/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status, comment }),
    });
    return this.normalizeRequest(resp);
  }

  // Apply approval action on current step
  async applyApprovalAction(
    id: number,
    payload: ServiceRequestApprovalActionRequest
  ): Promise<ServiceRequest> {
    const resp = await this.request<ServiceRequest>(`/api/v1/service-requests/${id}/approvals`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    return this.normalizeRequest(resp);
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
    return this.request<ProvisioningTask>(`/api/v1/provisioning-tasks/${taskId}/execute`, {
      method: 'POST',
    });
  }

  // ==================== 兼容别名 ====================

  /** @deprecated 使用 getUserServiceRequests */
  async getServiceRequests(params?: any): Promise<ServiceRequestListResponse> {
    return this.getUserServiceRequests(params);
  }

  /** @deprecated 使用 getUserServiceRequests */
  async getServiceRequest(id: number): Promise<ServiceRequest> {
    return this.getServiceRequestDetails(id);
  }

  /** @deprecated 使用 applyApprovalAction */
  async applyApproval(id: number, action: string, comment?: string): Promise<ServiceRequest> {
    return this.applyApprovalAction(id, { action: action as 'approve' | 'reject', comment });
  }
}

export interface ProvisioningTask {
  id: number;
  serviceRequestId: number;
  provider: string;
  resourceType: string;
  status: 'pending' | 'running' | 'succeeded' | 'failed';
  payload?: Record<string, unknown>;
  result?: Record<string, unknown>;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
}

export const serviceRequestAPI = new ServiceRequestAPI();
