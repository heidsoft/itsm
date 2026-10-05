import { httpClient } from './http-client';

// 审计日志项类型（与后端 DTO 对齐）
export interface AuditLog {
  id: number;
  createdAt: string; // ISO 时间字符串
  tenantId: number;
  userId: number;
  // 后端 service 层 join user 表填充，name 优先、缺失回退 username
  userName?: string;
  requestId: string;
  ip: string;
  resource: string;
  action: string;
  path: string;
  method: string;
  statusCode: number;
  requestBody: string;
}

// 查询参数（对应后端 ListAuditLogsRequest）
export interface ListAuditLogsParams {
  page?: number;
  pageSize?: number;
  userId?: number;
  resource?: string;
  action?: string;
  method?: string;
  statusCode?: number;
  path?: string;
  requestId?: string;
  from?: string; // RFC3339
  to?: string; // RFC3339
}

// 响应结构（对应后端 ListAuditLogsResponse）
export interface ListAuditLogsResponse {
  logs: AuditLog[];
  total: number;
  page: number;
  pageSize: number;
}

// httpClient 已解包 `{ code, message, data }` 并做 camelCase 转换，
// 因此这里的 raw 就是后端 dto.ListAuditLogsResponse 本体：{ items, total, page, pageSize, totalPages }。
// 注意：后端字段名是 items，而不是本文件早期假设的 logs——正是这个不一致让审计日志页
// 「total 正常增长、表格永远为空」。在此做一次归一化，避免每个调用方各自踩坑。
interface RawAuditLogListResponse {
  items?: AuditLog[];
  logs?: AuditLog[];
  total?: number;
  page?: number;
  pageSize?: number;
  totalPages?: number;
}

// 查询审计日志（自动携带 Authorization、X-Tenant-ID / X-Tenant-Code）
export async function listAuditLogs(params: ListAuditLogsParams): Promise<ListAuditLogsResponse> {
  const query = new URLSearchParams();
  if (params.page) query.set('page', String(params.page));
  if (params.pageSize) query.set('pageSize', String(params.pageSize));
  if (params.userId !== undefined) query.set('userId', String(params.userId));
  if (params.resource) query.set('resource', params.resource);
  if (params.action) query.set('action', params.action);
  if (params.method) query.set('method', params.method);
  if (params.statusCode !== undefined) query.set('statusCode', String(params.statusCode));
  if (params.path) query.set('path', params.path);
  if (params.requestId) query.set('requestId', params.requestId);
  if (params.from) query.set('from', params.from);
  if (params.to) query.set('to', params.to);

  const endpoint = `/api/v1/audit-logs${query.toString() ? `?${query.toString()}` : ''}`;
  const raw = await httpClient.get<RawAuditLogListResponse>(endpoint);
  const logs = raw?.items ?? raw?.logs ?? [];
  return {
    logs,
    total: raw?.total ?? logs.length,
    page: raw?.page ?? params.page ?? 1,
    pageSize: raw?.pageSize ?? params.pageSize ?? 20,
  };
}
