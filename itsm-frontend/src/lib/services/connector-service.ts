/**
 * ConnectorService - 连接器/插件/技能/IM 市场服务
 *
 * 对接后端 /api/v1/connectors/*：
 *  - list()        市场中所有可用连接器（本地 + 远端合并）
 *  - configs()     当前租户已配置的连接器实例
 *  - provision()   启用/更新一个连接器实例
 *  - revoke()      停用并移除
 *  - test()        发送测试消息
 *  - send()        通过指定连接器发消息
 *  - health()      健康检查
 */

import { httpClient } from '@/lib/api/http-client';

export interface ConnectorManifest {
  name: string;
  version: string;
  title: string;
  provider: string;
  type: string;
  description?: string;
  author?: string;
  homepage?: string;
  iconUrl?: string;
  capabilities: string[];
  tags?: string[];
  local: boolean;
  installed: boolean;
  category: string;
}

export interface ConnectorConfig {
  name: string;
  provider: string;
  type: string;
  enabled: boolean;
  healthy?: boolean;
  lifecycle?: string;
  lastCheckedAt?: string;
  lastError?: string;
  credentials?: Record<string, string>;
  settings?: Record<string, unknown>;
  labels?: Record<string, string>;
}

export interface ProvisionConnectorRequest {
  name: string;
  provider?: string;
  enabled: boolean;
  credentials?: Record<string, string>;
  settings?: Record<string, unknown>;
  labels?: Record<string, string>;
}

export interface SendConnectorMessageRequest {
  channel: string;
  type?: 'text' | 'markdown' | 'post' | 'card';
  title?: string;
  content: string;
  card?: unknown;
  mentions?: Array<{ type: string; id: string; name?: string }>;
  actions?: Array<{ type: string; text: string; url?: string; value?: string }>;
  metadata?: Record<string, unknown>;
}

export interface ConnectorHealth {
  ok: boolean;
  latencyMs?: number;
  message?: string;
  checkedAt?: string;
  extra?: Record<string, unknown>;
}

class ConnectorService {
  private readonly baseUrl = '/api/v1/connectors';

  /** 列出市场中所有可用连接器 */
  async list(): Promise<ConnectorManifest[]> {
    const r = await httpClient.get<{ items: ConnectorManifest[]; total: number }>(this.baseUrl);
    return r.items || [];
  }

  /** 当前租户已配置的实例（凭据已脱敏） */
  async configs(): Promise<ConnectorConfig[]> {
    const r = await httpClient.get<{ items: ConnectorConfig[]; total: number }>(`${this.baseUrl}/configs`);
    return r.items || [];
  }

  /** 启用 / 更新一个实例 */
  async provision(payload: ProvisionConnectorRequest): Promise<ConnectorConfig> {
    return httpClient.post<ConnectorConfig>(`${this.baseUrl}/configs`, payload);
  }

  /** 停用并移除一个实例 */
  async revoke(name: string): Promise<void> {
    await httpClient.delete(`${this.baseUrl}/configs/${encodeURIComponent(name)}`);
  }

  /** 发送测试消息（向 settings.debugChannel） */
  async test(name: string): Promise<{ sent: boolean; channel: string }> {
    return httpClient.post(`${this.baseUrl}/${encodeURIComponent(name)}/test`);
  }

  /** 通过指定连接器发送消息 */
  async send(name: string, payload: SendConnectorMessageRequest): Promise<{ sent: boolean; channel: string }> {
    return httpClient.post(`${this.baseUrl}/${encodeURIComponent(name)}/send`, payload);
  }

  /** 健康检查（所有实例） */
  async health(): Promise<Record<string, ConnectorHealth>> {
    return httpClient.get(`${this.baseUrl}/health`);
  }
}

export const connectorService = new ConnectorService();
export default connectorService;
