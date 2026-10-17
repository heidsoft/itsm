import { httpClient } from './http-client';
import type { SystemStats } from '@/types/dashboard';
import type {
  SystemConfig,
  SystemConfigListResponse,
  UpdateSystemConfigRequest,
  GetSystemConfigsParams,
} from './api-config';

export class SystemConfigAPI {
  // 编辑器默认请求最多 1000 项；显式分页参数优先。
  static async getConfigs(params?: GetSystemConfigsParams): Promise<SystemConfigListResponse> {
    return httpClient.get<SystemConfigListResponse>('/api/v1/system-configs', {
      pageSize: 1000,
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

  // 系统运行时状态。后端与 /api/v1/dashboard/stats/system 共用同一个 DTO 与
  // availability 口径，未接入的指标必须由调用方按 availability 判断。
  static async getSystemStatus(): Promise<SystemStats> {
    return httpClient.get<SystemStats>('/api/v1/system-configs/status');
  }
}
