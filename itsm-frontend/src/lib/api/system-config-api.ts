import { httpClient } from './http-client';
import {
  SystemConfig,
  SystemConfigListResponse,
  UpdateSystemConfigRequest,
  GetSystemConfigsParams,
} from './api-config';

export class SystemConfigAPI {
  // 获取系统配置列表
  static async getConfigs(params?: GetSystemConfigsParams): Promise<SystemConfigListResponse> {
    return httpClient.get<SystemConfigListResponse>('/api/system-configs', params);
  }

  // 获取单个配置项
  static async getConfig(id: number): Promise<SystemConfig> {
    return httpClient.get<SystemConfig>(`/api/system-configs/${id}`);
  }

  // 根据键名获取配置项
  static async getConfigByKey(key: string): Promise<SystemConfig> {
    return httpClient.get<SystemConfig>(`/api/system-configs/key/${key}`);
  }

  // 更新配置项
  static async updateConfig(id: number, data: UpdateSystemConfigRequest): Promise<SystemConfig> {
    return httpClient.put<SystemConfig>(`/api/system-configs/${id}`, data);
  }

  // 批量更新配置项
  static async updateConfigs(data: UpdateSystemConfigRequest[]): Promise<SystemConfig[]> {
    return httpClient.put<SystemConfig[]>('/api/system-configs/batch', data);
  }

  // 获取系统状态信息
  static async getSystemStatus(): Promise<any> {
    return httpClient.get<any>('/api/system-configs/status');
  }
}