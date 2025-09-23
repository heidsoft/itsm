import { httpClient } from './http-client';

// Service catalog related interface definitions
export interface ServiceCatalog {
  id: number;
  name: string;
  category: string;
  description: string;
  delivery_time: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface ServiceCatalogListResponse {
  catalogs: ServiceCatalog[];
  total: number;
  page: number;
  size: number;
}

export interface GetServiceCatalogsParams {
  page?: number;
  size?: number;
  category?: string;
  status?: string;
  [key: string]: unknown;
}

export interface CreateServiceRequestRequest {
  catalog_id: number;
  reason?: string;
}

export interface ServiceRequest {
  id: number;
  catalog_id: number;
  requester_id: number;
  status: string;
  reason: string;
  created_at: string;
  catalog?: ServiceCatalog;
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

export interface GetServiceRequestsParams {
  page?: number;
  size?: number;
  status?: string;
  [key: string]: unknown;
}

export interface UpdateServiceRequestStatusRequest {
  status: string;
}

// Create service catalog request interface
export interface CreateServiceCatalogRequest {
  name: string;
  category: string;
  description?: string;
  delivery_time: string;
  status?: 'enabled' | 'disabled';
}

// Update service catalog request interface
export interface UpdateServiceCatalogRequest {
  name?: string;
  category?: string;
  description?: string;
  delivery_time?: string;
  status?: 'enabled' | 'disabled';
}

export class ServiceCatalogApi {
  // Get service catalog list
  static async getServiceCatalogs(params?: GetServiceCatalogsParams): Promise<ServiceCatalogListResponse> {
    return httpClient.get<ServiceCatalogListResponse>('/api/v1/services', params);
  }

  // Create service request
  static async createServiceRequest(request: CreateServiceRequestRequest): Promise<ServiceRequest> {
    return httpClient.post<ServiceRequest>('/api/v1/services/requests', request);
  }

  // Get current user's service request list
  static async getUserServiceRequests(params?: GetServiceRequestsParams): Promise<ServiceRequestListResponse> {
    return httpClient.get<ServiceRequestListResponse>('/api/v1/services/requests', params);
  }

  // Get service request by ID
  static async getServiceRequestById(id: number): Promise<ServiceRequest> {
    return httpClient.get<ServiceRequest>(`/api/v1/services/requests/${id}`);
  }

  // Update service request status
  static async updateServiceRequestStatus(id: number, request: UpdateServiceRequestStatusRequest): Promise<ServiceRequest> {
    return httpClient.put<ServiceRequest>(`/api/v1/services/requests/${id}/status`, request);
  }

  // Create service catalog
  static async createServiceCatalog(request: CreateServiceCatalogRequest): Promise<ServiceCatalog> {
    return httpClient.post<ServiceCatalog>('/api/v1/services/catalogs', request);
  }

  // Get service catalog by ID
  static async getServiceCatalogById(id: number): Promise<ServiceCatalog> {
    return httpClient.get<ServiceCatalog>(`/api/v1/services/catalogs/${id}`);
  }

  // Update service catalog
  static async updateServiceCatalog(id: number, request: UpdateServiceCatalogRequest): Promise<ServiceCatalog> {
    return httpClient.put<ServiceCatalog>(`/api/v1/services/catalogs/${id}`, request);
  }

  // Delete service catalog
  static async deleteServiceCatalog(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/services/catalogs/${id}`);
  }
}