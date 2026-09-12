/**
 * Cloud API Client - 对接后端 /api/v1/cloud/* 接口
 * 后端: itsm-backend/handlers/cloud + router/cloud_routes.go
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

// 必须是字符串字面量：派生模板常量无法被 api-contract 扫描器静态解析。
const CLOUD_ACCOUNTS_BASE = '/api/v1/cloud/accounts';
const CLOUD_SERVICES_BASE = '/api/v1/cloud/services';
const CLOUD_RESOURCES_BASE = '/api/v1/cloud/resources';

/**
 * Cloud Account API
 */
export const cloudAccountApi = {
  /**
   * 获取云账号列表
   */
  list: async (params?: ListCloudAccountsRequest): Promise<CloudAccountListResponse> => {
    return httpClient.get<CloudAccountListResponse>(CLOUD_ACCOUNTS_BASE, params);
  },

  /**
   * 获取云账号详情
   */
  get: async (id: number): Promise<CloudAccount> => {
    return httpClient.get<CloudAccount>(`${CLOUD_ACCOUNTS_BASE}/${id}`);
  },

  /**
   * 创建云账号
   */
  create: async (data: CreateCloudAccountRequest): Promise<CloudAccount> => {
    return httpClient.post<CloudAccount>(CLOUD_ACCOUNTS_BASE, data);
  },

  /**
   * 更新云账号
   */
  update: async (id: number, data: UpdateCloudAccountRequest): Promise<CloudAccount> => {
    return httpClient.put<CloudAccount>(`${CLOUD_ACCOUNTS_BASE}/${id}`, data);
  },

  /**
   * 删除云账号
   */
  delete: async (id: number): Promise<void> => {
    await httpClient.delete(`${CLOUD_ACCOUNTS_BASE}/${id}`);
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
    return httpClient.get<CloudServiceListResponse>(CLOUD_SERVICES_BASE, params);
  },

  /**
   * 获取云服务详情
   */
  get: async (id: number): Promise<CloudService> => {
    return httpClient.get<CloudService>(`${CLOUD_SERVICES_BASE}/${id}`);
  },

  /**
   * 创建云服务
   */
  create: async (data: CreateCloudServiceRequest): Promise<CloudService> => {
    return httpClient.post<CloudService>(CLOUD_SERVICES_BASE, data);
  },

  /**
   * 更新云服务
   */
  update: async (id: number, data: UpdateCloudServiceRequest): Promise<CloudService> => {
    return httpClient.put<CloudService>(`${CLOUD_SERVICES_BASE}/${id}`, data);
  },

  /**
   * 删除云服务
   */
  delete: async (id: number): Promise<void> => {
    await httpClient.delete(`${CLOUD_SERVICES_BASE}/${id}`);
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
    return httpClient.get<CloudResourceListResponse>(CLOUD_RESOURCES_BASE, params);
  },

  /**
   * 获取云资源详情
   */
  get: async (id: number): Promise<CloudResource> => {
    return httpClient.get<CloudResource>(`${CLOUD_RESOURCES_BASE}/${id}`);
  },

  /**
   * 创建云资源
   */
  create: async (data: CreateCloudResourceRequest): Promise<CloudResource> => {
    return httpClient.post<CloudResource>(CLOUD_RESOURCES_BASE, data);
  },

  /**
   * 更新云资源
   */
  update: async (id: number, data: UpdateCloudResourceRequest): Promise<CloudResource> => {
    return httpClient.put<CloudResource>(`${CLOUD_RESOURCES_BASE}/${id}`, data);
  },

  /**
   * 删除云资源
   */
  delete: async (id: number): Promise<void> => {
    await httpClient.delete(`${CLOUD_RESOURCES_BASE}/${id}`);
  },
};
