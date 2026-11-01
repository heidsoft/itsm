/**
 * 云信号 API：云监控告警与安全漏洞。
 * 路由在 /api/v1/cloud-signals 下，权限复用 cmdb:read / cmdb:write（与云账号同域）。
 */

import { httpClient } from './http-client';
import type { PaginationResponse } from './types';
import type {
  CloudAlert,
  CloudSignalSyncState,
  FindingSummary,
  SecurityFinding,
} from '@/types/biz/cloudsignal';

const CLOUD_SIGNALS_BASE = '/api/v1/cloud-signals';

// status/severity 是重复参数（?status=firing&status=acknowledged），
// httpClient.get 会把数组展开成重复键。
export interface AlertListQuery {
  status?: string[];
  severity?: string[];
  cloudAccountId?: number;
  search?: string;
  page?: number;
  pageSize?: number;
}

export interface FindingListQuery {
  status?: string[];
  severity?: string[];
  vulnType?: string;
  instanceId?: string;
  cloudAccountId?: number;
  search?: string;
  page?: number;
  pageSize?: number;
}

export class CloudSignalApi {
  static async listAlerts(query: AlertListQuery = {}): Promise<PaginationResponse<CloudAlert>> {
    return httpClient.get(`${CLOUD_SIGNALS_BASE}/alerts`, query);
  }

  static async acknowledgeAlert(id: number): Promise<CloudAlert> {
    return httpClient.post(`${CLOUD_SIGNALS_BASE}/alerts/${id}/acknowledge`);
  }

  static async resolveAlert(id: number): Promise<CloudAlert> {
    return httpClient.post(`${CLOUD_SIGNALS_BASE}/alerts/${id}/resolve`);
  }

  static async listFindings(query: FindingListQuery = {}): Promise<PaginationResponse<SecurityFinding>> {
    return httpClient.get(`${CLOUD_SIGNALS_BASE}/vulnerabilities`, query);
  }

  static async getFindingSummary(): Promise<FindingSummary> {
    return httpClient.get(`${CLOUD_SIGNALS_BASE}/vulnerabilities/summary`);
  }

  static async ignoreFinding(id: number): Promise<SecurityFinding> {
    return httpClient.post(`${CLOUD_SIGNALS_BASE}/vulnerabilities/${id}/ignore`);
  }

  static async reopenFinding(id: number): Promise<SecurityFinding> {
    return httpClient.post(`${CLOUD_SIGNALS_BASE}/vulnerabilities/${id}/reopen`);
  }

  static async getSyncStatus(): Promise<PaginationResponse<CloudSignalSyncState>> {
    return httpClient.get(`${CLOUD_SIGNALS_BASE}/sync-status`);
  }
}

export default CloudSignalApi;
