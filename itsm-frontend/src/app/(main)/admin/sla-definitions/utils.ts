/**
 * SLA 定义管理页面模型转换
 */

import { formatBusinessHours } from '@/lib/utils/sla-format';
import type { SLADefinition as APISLADefinition } from '@/lib/api/sla-api';
import type { SLADefinition, SLAStats } from './types';

/**
 * 转换 API 数据为页面模型。
 * 后端 SLADefinitionResponse 只返回响应/解决时间与 businessHours、escalationLevels 等
 * 已持久化的字段；页面不再展示可用性目标等后端未存储的指标。
 */
export const transformSLA = (item: APISLADefinition): SLADefinition => ({
  id: String(item.id),
  name: item.name ?? '',
  description: item.description ?? '',
  serviceType: item.serviceType ?? '',
  priority: item.priority ?? '',
  responseTimeMinutes: Number.isFinite(item.responseTime) ? item.responseTime : 0,
  resolutionTimeMinutes: Number.isFinite(item.resolutionTime) ? item.resolutionTime : 0,
  businessHours: formatBusinessHours(item.businessHours),
  escalationLevels: item.escalationLevels ?? [],
  status: item.isActive ? 'active' : 'inactive',
  createdAt: item.createdAt ?? '',
  updatedAt: item.updatedAt ?? '',
});

export const summarizeSLADefinitions = (rows: SLADefinition[]): SLAStats => ({
  total: rows.length,
  active: rows.filter(row => row.status === 'active').length,
  inactive: rows.filter(row => row.status === 'inactive').length,
  serviceTypeCount: new Set(rows.map(row => row.serviceType).filter(Boolean)).size,
});
