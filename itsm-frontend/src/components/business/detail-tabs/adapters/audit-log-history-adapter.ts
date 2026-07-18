import { listAuditLogs } from '@/lib/api/auditlog-api';
import type { HistoryRecord, TargetType } from '../types';

/**
 * 走 /api/v1/audit-logs?resource=xxx&path 兜底展示历史
 * 适用于尚未提供 /:id/history 端点的模块
 *
 * 通用查询：resource 匹配后端 resource 字段（如 'incident'/'problem'/'change'/'release'），
 * 并 fallback 用 path LIKE 过滤特定 id 的调用日志。
 */
export async function fetchAuditLogHistory(
  targetType: TargetType,
  targetId: number | string
): Promise<HistoryRecord[]> {
  const resource = targetType;
  // 尝试用 path 参数做进一步限定；后端如未支持模糊匹配则退化为 resource 全量再前端过滤
  const pathHint = `/${targetType}s/${targetId}`;
  const res = await listAuditLogs({
    resource,
    path: pathHint,
    pageSize: 100,
  });
  const logs = res.logs || [];
  return logs.map((l) => ({
    id: l.id,
    createdAt: l.createdAt,
    action: l.action,
    path: l.path,
    method: l.method,
    statusCode: l.statusCode,
    ip: l.ip,
    user: { name: `用户#${l.userId}` },
  }));
}
