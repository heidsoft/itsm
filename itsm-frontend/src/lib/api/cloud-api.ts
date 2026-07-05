/**
 * Cloud API Client - 对接后端 /api/v1/cloud/* 接口
 * 后端: itsm-backend/controller/cloud_controller.go
 */

import type {
  CloudAccount,
  CreateCloudAccountRequest,
  UpdateCloudAccountRequest,
  ListCloudAccountsRequest,
  CloudAccountListResponse,
  CloudService,
  CreateCloudServiceRequest,
  UpdateCloudServiceRequest,
  ListCloudServicesRequest,
  CloudServiceListResponse,
  CloudResource,
  CreateCloudResourceRequest,
  UpdateCloudResourceRequest,
  ListCloudResourcesRequest,
  CloudResourceListResponse,
} from '@/types/cloud';
import { httpClient } from './http-client';

/**
 * Cloud Account API
 */
export const cloudAccountApi = {
  /**
   * 获取云账号列表
   */
  list: async (params?: ListCloudAccountsRequest): Promise<CloudAccountListResponse> => {
    return httpClient.get<CloudAccountListResponse>('/cloud/accounts', { params });
  },

  /**
   * 获取云账号详情
   */
  get: async (id: number): Promise<CloudAccount> => {
    return httpClient.get<CloudAccount>(`/cloud/accounts/${id}`);
  },

  /**
   * 创建云账号
   */
  create: async (data: CreateCloudAccountRequest): Promise<CloudAccount> => {
    return httpClient.post<CloudAccount>('/cloud/accounts', data);
  },

  /**
   * 更新云账号
   */
  update: async (id: number, data: UpdateCloudAccountRequest): Promise<CloudAccount> => {
    return httpClient.put<CloudAccount>(`/cloud/accounts/${id}`, data);
  },

  /**
   * 删除云账号
   */
  delete: async (id: number): Promise<void> => {
    await httpClient.delete(`/cloud/accounts/${id}`);
  },
};

/**
 * Cloud Service API
 */
export const cloudServiceApi = {
  /**
   * 获取云服务列表
   */
  list: async (params?: ListCloudServicesRequest): Promise<CloudServiceListResponse> => {
    return httpClient.get<CloudServiceListResponse>('/cloud/services', { params });
  },

  /**
   * 获取云服务详情
   */
  get: async (id: number): Promise<CloudService> => {
    return httpClient.get<CloudService>(`/cloud/services/${id}`);
  },

  /**
   * 创建云服务
   */
  create: async (data: CreateCloudServiceRequest): Promise<CloudService> => {
    return httpClient.post<CloudService>('/cloud/services', data);
  },

  /**
   * 更新云服务
   */
  update: async (id: number, data: UpdateCloudServiceRequest): Promise<CloudService> => {
    return httpClient.put<CloudService>(`/cloud/services/${id}`, data);
  },

  /**
   * 删除云服务
   */
  delete: async (id: number): Promise<void> => {
    await httpClient.delete(`/cloud/services/${id}`);
  },
};

/**
 * Cloud Resource API
 */
export const cloudResourceApi = {
  /**
   * 获取云资源列表
   */
  list: async (params?: ListCloudResourcesRequest): Promise<CloudResourceListResponse> => {
    return httpClient.get<CloudResourceListResponse>('/cloud/resources', { params });
  },

  /**
   * 获取云资源详情
   */
  get: async (id: number): Promise<CloudResource> => {
    return httpClient.get<CloudResource>(`/cloud/resources/${id}`);
  },

  /**
   * 创建云资源
   */
  create: async (data: CreateCloudResourceRequest): Promise<CloudResource> => {
    return httpClient.post<CloudResource>('/cloud/resources', data);
  },

  /**
   * 更新云资源
   */
  update: async (id: number, data: UpdateCloudResourceRequest): Promise<CloudResource> => {
    return httpClient.put<CloudResource>(`/cloud/resources/${id}`, data);
  },

  /**
   * 删除云资源
   */
  delete: async (id: number): Promise<void> => {
    await httpClient.delete(`/cloud/resources/${id}`);
  },
};
