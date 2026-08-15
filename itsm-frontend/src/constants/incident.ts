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
  [IncidentStatus.NEW]: '新建',
  [IncidentStatus.ACKNOWLEDGED]: '已确认',
  [IncidentStatus.ASSIGNED]: '已分配',
  [IncidentStatus.IN_PROGRESS]: '处理中',
  [IncidentStatus.TRIAGED]: '已分类',
  [IncidentStatus.ESCALATED]: '已升级',
  [IncidentStatus.ON_HOLD]: '已暂停',
  [IncidentStatus.RESOLVED]: '已解决',
  [IncidentStatus.CLOSED]: '已关闭',
  [IncidentStatus.CANCELLED]: '已取消',
};

// 优先级
export enum IncidentPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  URGENT = 'urgent',
}

export const IncidentPriorityLabels: Record<IncidentPriority, string> = {
  [IncidentPriority.LOW]: '低',
  [IncidentPriority.MEDIUM]: '中',
  [IncidentPriority.HIGH]: '高',
  [IncidentPriority.URGENT]: '紧急',
};

// 严重程度
export enum IncidentSeverity {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical',
}

export const IncidentSeverityLabels: Record<IncidentSeverity, string> = {
  [IncidentSeverity.LOW]: '低',
  [IncidentSeverity.MEDIUM]: '中',
  [IncidentSeverity.HIGH]: '高',
  [IncidentSeverity.CRITICAL]: '严重',
};
