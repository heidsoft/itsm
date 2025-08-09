import { httpClient } from './http-client';

// 服务目录相关接口定义
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
}

export interface UpdateServiceRequestStatusRequest {
  status: string;
}

// 创建服务目录请求接口
export interface CreateServiceCatalogRequest {
  name: string;
  category: string;
  description?: string;
  delivery_time: string;
  status?: 'enabled' | 'disabled';
}

// 更新服务目录请求接口
export interface UpdateServiceCatalogRequest {
  name?: string;
  category?: string;
  description?: string;
  delivery_time?: string;
  status?: 'enabled' | 'disabled';
}

export class ServiceCatalogApi {
  // 获取服务目录列表
  static async getServiceCatalogs(params?: GetServiceCatalogsParams): Promise<ServiceCatalogListResponse> {
    return httpClient.get<ServiceCatalogListResponse>('/api/v1/services', params);
  }

  // 创建服务请求
  static async createServiceRequest(request: CreateServiceRequestRequest): Promise<ServiceRequest> {
    return httpClient.post<ServiceRequest>('/api/v1/services/requests', request);
  }

  // 获取当前用户的服务请求列表
  static async getUserServiceRequests(params?: GetServiceRequestsParams): Promise<ServiceRequestListResponse> {
    return httpClient.get<ServiceRequestListResponse>('/api/v1/services/requests', params);
  }

  // 获取服务请求详情
  static async getServiceRequestById(id: number): Promise<ServiceRequest> {
    return httpClient.get<ServiceRequest>(`/api/v1/services/requests/${id}`);
  }

  // 更新服务请求状态
  static async updateServiceRequestStatus(id: number, request: UpdateServiceRequestStatusRequest): Promise<ServiceRequest> {
    return httpClient.put<ServiceRequest>(`/api/v1/services/requests/${id}`,(request));
  }

  // 创建服务目录
  static async createServiceCatalog(request: CreateServiceCatalogRequest): Promise<ServiceCatalog> {
    return httpClient.post<ServiceCatalog>('/api/v1/services', request);
  }

  // 根据ID获取服务目录
  static async getServiceCatalogById(id: number): Promise<ServiceCatalog> {
    return httpClient.get<ServiceCatalog>(`/api/v1/services/${id}`);
  }

  // 更新服务目录
  static async updateServiceCatalog(id: number, request: UpdateServiceCatalogRequest): Promise<ServiceCatalog> {
    return httpClient.put<ServiceCatalog>(`/api/v1/services/${id}`, request);
  }

  // 删除服务目录
  static async deleteServiceCatalog(id: number): Promise<void> {
    return httpClient.delete<void>(`/api/v1/services/${id}`);
  }
}