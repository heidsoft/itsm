import { httpClient } from './http-client';

// 审计日志项类型（与后端 DTO 对齐）
export interface AuditLog {
  id: number;
  created_at: string; // ISO 时间字符串
  tenant_id: number;
  user_id: number;
  request_id: string;
  ip: string;
  resource: string;
  action: string;
  path: string;
  method: string;
  status_code: number;
  request_body: string;
}

// 查询参数（对应后端 ListAuditLogsRequest）
export interface ListAuditLogsParams {
  page?: number;
  page_size?: number;
  user_id?: number;
  resource?: string;
  action?: string;
  method?: string;
  status_code?: number;
  path?: string;
  request_id?: string;
  from?: string; // RFC3339
  to?: string;   // RFC3339
}

// 响应结构（对应后端 ListAuditLogsResponse）
export interface ListAuditLogsResponse {
  logs: AuditLog[];
  total: number;
  page: number;
  page_size: number;
}

// 查询审计日志（自动携带 Authorization、X-Tenant-ID / X-Tenant-Code）
export async function listAuditLogs(params: ListAuditLogsParams): Promise<ListAuditLogsResponse> {
  const query = new URLSearchParams();
  if (params.page) query.set('page', String(params.page));
  if (params.page_size) query.set('page_size', String(params.page_size));
  if (params.user_id !== undefined) query.set('user_id', String(params.user_id));
  if (params.resource) query.set('resource', params.resource);
  if (params.action) query.set('action', params.action);
  if (params.method) query.set('method', params.method);
  if (params.status_code !== undefined) query.set('status_code', String(params.status_code));
  if (params.path) query.set('path', params.path);
  if (params.request_id) query.set('request_id', params.request_id);
  if (params.from) query.set('from', params.from);
  if (params.to) query.set('to', params.to);

  const endpoint = `/api/v1/audit-logs${query.toString() ? `?${query.toString()}` : ''}`;
  return await httpClient.get<ListAuditLogsResponse>(endpoint);
}