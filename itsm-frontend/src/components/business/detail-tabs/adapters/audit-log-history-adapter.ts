import { listAuditLogs } from '@/lib/api/auditlog-api';
import type { HistoryRecord, TargetType } from '../types';

/** 将 action + resource 映射为用户可读的操作描述 */
function toReadableAction(action: string | undefined, resource: string): string {
  if (!action) return resource;
  const a = action.toLowerCase();
  const actionMap: Record<string, string> = {
    create: '创建',
    update: '更新',
    delete: '删除',
    search: '查询',
    login_success: '登录成功',
    login_failed: '登录失败',
  };
  const resourceMap: Record<string, string> = {
    problems: '问题',
    incidents: '事件',
    changes: '变更',
    tickets: '工单',
    releases: '发布',
    users: '用户',
    roles: '角色',
    auth: '认证',
    sla: 'SLA',
    ai: 'AI',
    knowledge: '知识库',
    cmdb: 'CMDB',
    service_requests: '服务请求',
  };
  const r = resourceMap[resource] || resource.replace(/s$/, '').replace(/_/g, ' ');
  return actionMap[a] ? `${actionMap[a]}${r}` : action;
}

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
    details: toReadableAction(l.action, l.resource),
    // 后端 service 层 join user 表填充 userName（中文姓名优先，缺失回退 username）；
    // 后端兼容未升级前仍返回 `用户#<id>` 形式。
    user: { name: l.userName || `用户#${l.userId}` },
  }));
}
