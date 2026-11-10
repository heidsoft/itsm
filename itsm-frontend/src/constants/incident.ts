/**
 * 事件管理相关常量定义
 */

// 事件状态（与后端 common/constants.go IncidentStatus* 保持一致）
export enum IncidentStatus {
  NEW = 'new',
  ACKNOWLEDGED = 'acknowledged',
  ASSIGNED = 'assigned',
  IN_PROGRESS = 'in_progress',
  TRIAGED = 'triaged',
  ESCALATED = 'escalated',
  ON_HOLD = 'on_hold',
  RESOLVED = 'resolved',
  CLOSED = 'closed',
  CANCELLED = 'cancelled',
}

// 状态描述映射
export const IncidentStatusLabels: Record<IncidentStatus, string> = {
  [IncidentStatus.NEW]: 'Create',
  [IncidentStatus.ACKNOWLEDGED]: 'Acknowledged',
  [IncidentStatus.ASSIGNED]: 'Assigned',
  [IncidentStatus.IN_PROGRESS]: 'In Progress',
  [IncidentStatus.TRIAGED]: 'Triaged',
  [IncidentStatus.ESCALATED]: 'Escalated',
  [IncidentStatus.ON_HOLD]: 'Paused',
  [IncidentStatus.RESOLVED]: 'Resolved',
  [IncidentStatus.CLOSED]: 'Closed',
  [IncidentStatus.CANCELLED]: 'Cancelled',
};

// 优先级（与后端 ent/schema/incident.go 的 priority 校验取值一致：
// low/medium/high/critical。这里曾写着 urgent —— 那是工单词表，事件域从不接受它，
// 于是 critical 事件在详情页标签映射为空、编辑页提交被后端拒绝。）
export enum IncidentPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical',
}

export const IncidentPriorityLabels: Record<IncidentPriority, string> = {
  [IncidentPriority.LOW]: 'Low',
  [IncidentPriority.MEDIUM]: 'Medium',
  [IncidentPriority.HIGH]: 'High',
  [IncidentPriority.CRITICAL]: 'Urgent',
};

// 严重程度
export enum IncidentSeverity {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical',
}

export const IncidentSeverityLabels: Record<IncidentSeverity, string> = {
  [IncidentSeverity.LOW]: 'Low',
  [IncidentSeverity.MEDIUM]: 'Medium',
  [IncidentSeverity.HIGH]: 'High',
  [IncidentSeverity.CRITICAL]: 'Critical',
};

// status 与 priority 在持久层是无约束/半约束字符串列，词表外的历史取值必须
// 原样显示而不是并进已知桶，因此判定与取值分离暴露给调用方。
export function isKnownIncidentStatus(value: unknown): value is IncidentStatus {
  return Object.values(IncidentStatus).includes(value as IncidentStatus);
}

export function isKnownIncidentPriority(value: unknown): value is IncidentPriority {
  return Object.values(IncidentPriority).includes(value as IncidentPriority);
}

export function incidentStatusLabel(value: string): string {
  return isKnownIncidentStatus(value) ? IncidentStatusLabels[value] : value;
}

export function incidentPriorityLabel(value: string): string {
  return isKnownIncidentPriority(value) ? IncidentPriorityLabels[value] : value;
}
