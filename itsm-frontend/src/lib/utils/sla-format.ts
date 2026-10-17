/**
 * SLA 展示层格式化：优先级词表、时限、服务窗口、升级级别。
 * 时间戳格式化统一用 `@/lib/formatters` 的 formatDateTime，此处不重复实现。
 *
 * 口径来自后端 dto.SLADefinitionResponse：优先级取值是 critical/high/medium/low，
 * 时限单位统一为分钟，businessHours 与 escalationLevels 由后端 DTO 投影为 camelCase。
 */

import type { SLABusinessHours, SLAEscalationLevel } from '@/lib/api/sla-api';

const PRIORITY_META: Record<string, { label: string; color: string }> = {
  critical: { label: '紧急', color: 'red' },
  high: { label: '高', color: 'orange' },
  medium: { label: '中', color: 'gold' },
  low: { label: '低', color: 'green' },
};

// urgent 是存量数据里的别名；后端 SLA 匹配器会把它归一成 critical，前端口径保持一致。
const PRIORITY_ALIASES: Record<string, string> = { urgent: 'critical' };

export const SLA_PRIORITY_VALUES = Object.keys(PRIORITY_META);

const canonicalPriority = (priority: string): string => {
  const value = (priority || '').toLowerCase();
  return PRIORITY_ALIASES[value] ?? value;
};

export const getPriorityLabel = (priority: string): string =>
  PRIORITY_META[canonicalPriority(priority)]?.label ?? priority;

export const getPriorityColor = (priority: string): string =>
  PRIORITY_META[canonicalPriority(priority)]?.color ?? 'default';

export const matchesPriority = (rowPriority: string, filterValue: string): boolean =>
  filterValue === 'all' || canonicalPriority(rowPriority) === canonicalPriority(filterValue);

const WEEKDAY_LABELS = ['一', '二', '三', '四', '五', '六', '日'];

const formatWorkDays = (days?: number[]): string => {
  if (!days || days.length === 0) return '';
  const sorted = [...days].sort((a, b) => a - b);
  if (sorted.length === 7) return '每天';
  return sorted.map(day => WEEKDAY_LABELS[day - 1] ?? String(day)).join('、');
};

export const formatBusinessHours = (config?: SLABusinessHours): string => {
  if (!config) return '未配置';
  if (config.alwaysOn) return '7×24';

  const days = formatWorkDays(config.workDays);
  const window = config.startTime && config.endTime ? `${config.startTime}-${config.endTime}` : '';
  const parts = [days, window].filter(Boolean);
  if (parts.length === 0) return '未配置';

  return config.timeZone ? `${parts.join(' ')} (${config.timeZone})` : parts.join(' ');
};

export const formatDuration = (minutes: number): string => {
  if (!Number.isFinite(minutes) || minutes <= 0) return '未设置';
  if (minutes < 60) return `${minutes} 分钟`;

  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (hours < 24) return rest === 0 ? `${hours} 小时` : `${hours} 小时 ${rest} 分钟`;

  const days = Math.floor(hours / 24);
  const restHours = hours % 24;
  return restHours === 0 ? `${days} 天` : `${days} 天 ${restHours} 小时`;
};

export const formatEscalationLevel = (level: SLAEscalationLevel): string => {
  const roles = level.notifyRoles?.length ? ` → ${level.notifyRoles.join('、')}` : '';
  return `L${level.level} · ${formatDuration(level.thresholdMinutes)}后升级${roles}`;
};

// serviceType 词表来自后端（Ent 的 service_type 列与 SLA 模板目录），
// 模板页与定义页共用，避免同一取值在两个页面显示成不同名称。
export const SLA_SERVICE_TYPE_LABELS: Record<string, string> = {
  incident: '事件管理',
  service_request: '服务请求',
  change: '变更管理',
  problem: '问题管理',
  ticket: '工单',
};

export const getServiceTypeLabel = (serviceType: string): string =>
  SLA_SERVICE_TYPE_LABELS[serviceType] ?? serviceType;

// 模板目录的 industry 取值与 serviceType 同一套词表（下划线形式）。
const INDUSTRY_LABELS: Record<string, string> = {
  incident: '事件',
  change: '变更',
  service_request: '服务请求',
};

export const getIndustryLabel = (industry: string): string =>
  INDUSTRY_LABELS[industry] ?? industry;
