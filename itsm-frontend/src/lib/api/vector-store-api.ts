'use client';

import { httpClient } from './http-client';

const VECTOR_STORE_PATH = '/api/v1/system/vector-store';

/** 向量存储状态（camelCase 契约，与后端 VectorStoreStatusResponse 逐字段对应） */
export interface VectorStoreStatus {
  configured: boolean;
  source: string;
  backend: string;
  collection: string;
  fallbackEnabled: boolean;
  /** ready | degraded | unconfigured | error */
  capability: string;
  settings: Record<string, unknown>;
  vectorCount: number;
  checkedAt: string;
  message?: string;
}

/** 连通性测试结果 */
export interface VectorStoreTestResult {
  ok: boolean;
  backend: string;
  latencyMs: number;
  fallback: boolean;
  message: string;
  checkedAt: string;
}

export const VectorStoreApi = {
  async getStatus(): Promise<VectorStoreStatus> {
    return httpClient.get<VectorStoreStatus>(VECTOR_STORE_PATH);
  },

  async testConnection(): Promise<VectorStoreTestResult> {
    return httpClient.post<VectorStoreTestResult>(`${VECTOR_STORE_PATH}/test`);
  },
};
