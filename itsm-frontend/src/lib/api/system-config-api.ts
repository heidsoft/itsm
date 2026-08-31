import { httpClient } from './http-client';
import type {
  SystemConfig,
  SystemConfigListResponse,
  UpdateSystemConfigRequest,
  GetSystemConfigsParams,
} from './api-config';

export class SystemConfigAPI {
  // 获取系统配置列表。
  // 后端 ListConfigs 默认 page_size=20，系统配置全部键不到 30 条，一次拉完更安全。
  // 必须用 snake_case query 参数（后端 controller 读 page_size）；
  // http-client 不会自动转换 params 的 key。
  static async getConfigs(params?: GetSystemConfigsParams): Promise<SystemConfigListResponse> {
    return httpClient.get<SystemConfigListResponse>('/api/v1/system-configs', {
      page_size: 1000,
      ...params,
    });
  }

  // 获取单个配置项
  static async getConfig(id: number): Promise<SystemConfig> {
    return httpClient.get<SystemConfig>(`/api/v1/system-configs/${id}`);
  }

  // 根据键名获取配置项
  static async getConfigByKey(key: string): Promise<SystemConfig> {
    return httpClient.get<SystemConfig>(`/api/v1/system-configs/key/${key}`);
  }

  // 更新配置项
  static async updateConfig(id: number, data: UpdateSystemConfigRequest): Promise<SystemConfig> {
    return httpClient.put<SystemConfig>(`/api/v1/system-configs/${id}`, data);
  }

  // 批量更新配置项
  static async updateConfigs(data: UpdateSystemConfigRequest[]): Promise<SystemConfig[]> {
    return httpClient.put<SystemConfig[]>('/api/v1/system-configs/batch', data);
  }

  // 获取系统状态信息
  static async getSystemStatus(): Promise<Record<string, unknown>> {
    return httpClient.get<Record<string, unknown>>('/api/v1/system-configs/status');
  }
}
