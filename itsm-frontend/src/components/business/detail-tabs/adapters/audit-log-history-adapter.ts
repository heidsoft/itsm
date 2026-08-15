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
  // 审计中间件按 URL 路径段写入 resource（复数，如 'incidents'），
  // path 为完整请求路径（如 /api/v1/incidents/3/status）。查询条件必须与写入侧对齐：
  // - resource 用复数形式
  // - path 用 /api/v1/{复数}/{id} 前缀（后端 PathHasPrefix 匹配）
  const pluralResource = `${targetType}s`;
  const pathPrefix = `/api/v1/${pluralResource}/${targetId}`;
  const res = await listAuditLogs({
    resource: pluralResource,
    path: pathPrefix,
    pageSize: 100,
  });
  const logs = (res.logs || []).filter(
    (l) => l.path === pathPrefix || l.path.startsWith(`${pathPrefix}/`)
  );
  return logs.map((l) => ({
    id: l.id,
    createdAt: l.createdAt,
    action: l.action,
    details: `${l.method} ${l.path}`,
    path: l.path,
    method: l.method,
    statusCode: l.statusCode,
    ip: l.ip,
    user: { name: `用户#${l.userId}` },
  }));
}
