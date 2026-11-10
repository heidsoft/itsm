/**
 * ITSM 统一类型分类系统
 * 整合工单类型、服务请求类型、事件类型等到统一的分类体系
 */

// ==================== 主类型 (Major Types) ====================
export enum ITSMMainType {
  INCIDENT = 'incident',         // 事件
  SERVICE_REQUEST = 'service_request', // 服务请求
  PROBLEM = 'problem',           // 问题
  CHANGE = 'change',             // 变更
  KNOWLEDGE = 'knowledge',       // 知识库文章
}

export const ITSMMainTypeConfig: Record<ITSMMainType, {
  label: string;
  color: string;
  icon: string;
  description: string;
  badgeStatus: 'success' | 'processing' | 'warning' | 'error' | 'default';
}> = {
  [ITSMMainType.INCIDENT]: {
    label: 'Incident',
    color: 'red',
    icon: '🔴',
    description: 'Unexpected event of service interruption or service quality degradation',
    badgeStatus: 'error',
  },
  [ITSMMainType.SERVICE_REQUEST]: {
    label: 'Service Request',
    color: 'blue',
    icon: '📋',
    description: 'Service request or information consultation submitted by user',
    badgeStatus: 'processing',
  },
  [ITSMMainType.PROBLEM]: {
    label: 'Problem',
    color: 'orange',
    icon: '🔍',
    description: 'Incident root cause investigation and analysis',
    badgeStatus: 'warning',
  },
  [ITSMMainType.CHANGE]: {
    label: 'Change',
    color: 'green',
    icon: '🔄',
    description: 'Change request for IT infrastructure or services',
    badgeStatus: 'success',
  },
  [ITSMMainType.KNOWLEDGE]: {
    label: 'Knowledge',
    color: 'purple',
    icon: '📚',
    description: 'Knowledge Articles',
    badgeStatus: 'default',
  },
};

// ==================== 工单状态 (Ticket Status) ====================
// 词表唯一来源 = 后端工单状态机 itsm-backend/common/constants.go（TicketStatusNew..TicketStatusRejected）。
// pending_approval 属于 Change / ServiceRequest 领域，不是工单状态，禁止再加回本词表。
export enum TicketStatus {
  NEW = 'new',
  OPEN = 'open',
  ASSIGNED = 'assigned',
  IN_PROGRESS = 'in_progress',
  PENDING = 'pending',
  RESOLVED = 'resolved',
  CLOSED = 'closed',
  CANCELLED = 'cancelled',
  APPROVED = 'approved',
  REJECTED = 'rejected',
}

export const TicketStatusConfig: Record<TicketStatus, {
  label: string;
  text: string; // Alias for label, for backward compatibility
  color: string;
  bgColor: string;
  textColor: string;
  badgeStatus: 'success' | 'processing' | 'warning' | 'error' | 'default';
}> = {
  [TicketStatus.NEW]: {
    label: 'Create',
    text: 'Create',
    color: 'blue',
    bgColor: '#e6f7ff',
    textColor: '#1890ff',
    badgeStatus: 'processing',
  },
  [TicketStatus.OPEN]: {
    label: 'Pending',
    text: 'Pending',
    color: 'orange',
    bgColor: '#fff7e6',
    textColor: '#fa8c16',
    badgeStatus: 'warning',
  },
  [TicketStatus.ASSIGNED]: {
    label: 'Assigned',
    text: 'Assigned',
    color: 'geekblue',
    bgColor: '#f0f5ff',
    textColor: '#2f54eb',
    badgeStatus: 'processing',
  },
  [TicketStatus.IN_PROGRESS]: {
    label: 'In Progress',
    text: 'In Progress',
    color: 'cyan',
    bgColor: '#e6fffb',
    textColor: '#13c2c2',
    badgeStatus: 'processing',
  },
  [TicketStatus.PENDING]: {
    label: 'Waiting',
    text: 'Waiting',
    color: 'purple',
    bgColor: '#f9f0ff',
    textColor: '#722ed1',
    badgeStatus: 'default',
  },
  [TicketStatus.RESOLVED]: {
    label: 'Resolved',
    text: 'Resolved',
    color: 'green',
    bgColor: '#f6ffed',
    textColor: '#52c41a',
    badgeStatus: 'success',
  },
  [TicketStatus.CLOSED]: {
    label: 'Closed',
    text: 'Closed',
    color: 'default',
    bgColor: '#fafafa',
    textColor: '#8c8c8c',
    badgeStatus: 'default',
  },
  [TicketStatus.CANCELLED]: {
    label: 'Cancelled',
    text: 'Cancelled',
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    badgeStatus: 'error',
  },
  [TicketStatus.APPROVED]: {
    label: 'Approved',
    text: 'Approved',
    color: 'green',
    bgColor: '#f6ffed',
    textColor: '#389e0d',
    badgeStatus: 'success',
  },
  [TicketStatus.REJECTED]: {
    label: 'Rejected',
    text: 'Rejected',
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    badgeStatus: 'error',
  },
};

// ==================== 工单优先级 (Ticket Priority) ====================
export enum TicketPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  URGENT = 'urgent',
  CRITICAL = 'critical',
}

export const TicketPriorityConfig: Record<TicketPriority, {
  label: string;
  color: string;
  bgColor: string;
  textColor: string;
  text: string; // For legacy compatibility
  icon: string; // For legacy compatibility (arrow symbols)
  description: string;
}> = {
  [TicketPriority.LOW]: {
    label: 'Low',
    color: 'blue',
    bgColor: '#e6f7ff',
    textColor: '#1890ff',
    text: 'Low',
    icon: '↓',
    description: 'General problem, can be postponed',
  },
  [TicketPriority.MEDIUM]: {
    label: 'Medium',
    color: 'orange',
    bgColor: '#fff7e6',
    textColor: '#fa8c16',
    text: 'Medium',
    icon: '→',
    description: 'Normal problem, handle in order',
  },
  [TicketPriority.HIGH]: {
    label: 'High',
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    text: 'High',
    icon: '↑',
    description: 'Important issues, priority processing',
  },
  [TicketPriority.URGENT]: {
    label: 'Urgent',
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    text: 'Urgent',
    icon: '↑↑',
    description: 'Serious impact on business, needs to be dealt with immediately',
  },
  [TicketPriority.CRITICAL]: {
    label: 'Urgent',
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    text: 'Urgent',
    icon: '↑↑',
    description: 'Core business is interrupted and needs to be handled immediately',
  },
};

// ==================== 变更类型 (Change Types) ====================
export enum ChangeType {
  NORMAL = 'normal',
  STANDARD = 'standard',
  EMERGENCY = 'emergency',
}

export const ChangeTypeConfig: Record<ChangeType, {
  label: string;
  color: string;
  icon: string;
  description: string;
}> = {
  [ChangeType.NORMAL]: {
    label: 'Normal Change',
    color: 'blue',
    icon: '📋',
    description: 'Standard change process, approval required',
  },
  [ChangeType.STANDARD]: {
    label: 'Standard Change',
    color: 'green',
    icon: '✅',
    description: 'Pre-approved low-risk change',
  },
  [ChangeType.EMERGENCY]: {
    label: 'Emergency Change',
    color: 'red',
    icon: '🚨',
    description: 'Rapid changes in emergency situations',
  },
};

// ==================== 变更状态 (Change Status) ====================
export enum ChangeStatus {
  DRAFT = 'draft',
  PENDING = 'pending',
  APPROVED = 'approved',
  REJECTED = 'rejected',
  SCHEDULED = 'scheduled',
  IN_PROGRESS = 'in_progress',
  COMPLETED = 'completed',
  FAILED = 'failed',
  ROLLED_BACK = 'rolled_back',
  CANCELLED = 'cancelled',
  CLOSED = 'closed',
}

export const ChangeStatusConfig: Record<ChangeStatus, {
  label: string;
  color: string;
  badgeStatus: 'success' | 'processing' | 'warning' | 'error' | 'default';
}> = {
  [ChangeStatus.DRAFT]: {
    label: 'Draft',
    color: 'default',
    badgeStatus: 'default',
  },
  [ChangeStatus.PENDING]: {
    label: 'Pending Approval',
    color: 'gold',
    badgeStatus: 'warning',
  },
  [ChangeStatus.APPROVED]: {
    label: 'Approved',
    color: 'blue',
    badgeStatus: 'processing',
  },
  [ChangeStatus.REJECTED]: {
    label: 'Rejected',
    color: 'red',
    badgeStatus: 'error',
  },
  [ChangeStatus.SCHEDULED]: {
    label: 'Scheduled',
    color: 'cyan',
    badgeStatus: 'processing',
  },
  [ChangeStatus.IN_PROGRESS]: {
    label: 'In Progress',
    color: 'cyan',
    badgeStatus: 'processing',
  },
  [ChangeStatus.COMPLETED]: {
    label: 'Completed',
    color: 'green',
    badgeStatus: 'success',
  },
  [ChangeStatus.FAILED]: {
    label: 'Implementation failed',
    color: 'red',
    badgeStatus: 'error',
  },
  [ChangeStatus.ROLLED_BACK]: {
    label: 'Rolled Back',
    color: 'orange',
    badgeStatus: 'warning',
  },
  [ChangeStatus.CANCELLED]: {
    label: 'Cancelled',
    color: 'default',
    badgeStatus: 'default',
  },
  [ChangeStatus.CLOSED]: {
    label: 'Closed',
    color: 'default',
    badgeStatus: 'default',
  },
};

// ==================== 事件优先级 (Incident Priority) ====================
export enum IncidentPriority {
  P1 = 'p1',
  P2 = 'p2',
  P3 = 'p3',
  P4 = 'p4',
}

export const IncidentPriorityConfig: Record<IncidentPriority, {
  label: string;
  shortLabel: string;
  color: string;
  description: string;
  responseTime: string; // SLA响应时间
}> = {
  [IncidentPriority.P1]: {
    label: 'P1 - Urgent',
    shortLabel: 'P1',
    color: 'red',
    description: 'Core business is interrupted, affecting all users',
    responseTime: '15 minutes',
  },
  [IncidentPriority.P2]: {
    label: 'P2 - High',
    shortLabel: 'P2',
    color: 'orange',
    description: 'Core business is damaged, affecting most users',
    responseTime: '1 hour',
  },
  [IncidentPriority.P3]: {
    label: 'P3 - Medium',
    shortLabel: 'P3',
    color: 'gold',
    description: 'Non-core services are damaged, affecting some users',
    responseTime: '4 hours',
  },
  [IncidentPriority.P4]: {
    label: 'P4 - Low',
    shortLabel: 'P4',
    color: 'blue',
    description: 'Minor problem, can be postponed',
    responseTime: '8 hours',
  },
};

// ==================== 事件分类 (Incident Category) ====================
// 创建/编辑/展示三处共用同一份值域，防止回显空白
export const IncidentCategoryOptions: { value: string; label: string }[] = [
  { value: 'hardware', label: 'Hardware failure' },
  { value: 'software', label: 'Software failure' },
  { value: 'network', label: 'Network problem' },
  { value: 'security', label: 'Security issue' },
  { value: 'performance', label: 'Performance issue' },
  { value: 'connectivity', label: 'Connection problem' },
  { value: 'storage', label: 'Storage problem' },
  { value: 'application', label: 'Application failure' },
  { value: 'database', label: 'Database failure' },
  { value: 'other', label: 'Other' },
];

export const getIncidentCategoryLabel = (value?: string): string =>
  IncidentCategoryOptions.find(option => option.value === value)?.label || value || '-';

// ==================== 服务请求状态 (Service Request Status) ====================
export enum ServiceRequestStatus {
  SUBMITTED = 'submitted',
  ACCEPTED = 'accepted',
  IN_PROGRESS = 'in_progress',
  PENDING_APPROVAL = 'pending_approval',
  COMPLETED = 'completed',
  CANCELLED = 'cancelled',
  REJECTED = 'rejected',
}

export const ServiceRequestStatusConfig: Record<ServiceRequestStatus, {
  label: string;
  color: string;
  badgeStatus: 'success' | 'processing' | 'warning' | 'error' | 'default';
}> = {
  [ServiceRequestStatus.SUBMITTED]: {
    label: 'Submitted',
    color: 'blue',
    badgeStatus: 'processing',
  },
  [ServiceRequestStatus.ACCEPTED]: {
    label: 'Accepted',
    color: 'cyan',
    badgeStatus: 'processing',
  },
  [ServiceRequestStatus.IN_PROGRESS]: {
    label: 'In Progress',
    color: 'purple',
    badgeStatus: 'processing',
  },
  [ServiceRequestStatus.PENDING_APPROVAL]: {
    label: 'Pending Approval',
    color: 'gold',
    badgeStatus: 'warning',
  },
  [ServiceRequestStatus.COMPLETED]: {
    label: 'Completed',
    color: 'green',
    badgeStatus: 'success',
  },
  [ServiceRequestStatus.CANCELLED]: {
    label: 'Cancelled',
    color: 'default',
    badgeStatus: 'default',
  },
  [ServiceRequestStatus.REJECTED]: {
    label: 'Rejected',
    color: 'red',
    badgeStatus: 'error',
  },
};

// ==================== 辅助函数 ====================

/**
 * 根据主类型获取配置
 */
export const getMainTypeConfig = (type: string) => {
  return ITSMMainTypeConfig[type as ITSMMainType] || ITSMMainTypeConfig[ITSMMainType.SERVICE_REQUEST];
};

/**
 * 根据状态获取配置
 */
export const getStatusConfig = (status: string, mainType?: string) => {
  // 如果指定了主类型，根据类型返回对应的状态配置
  if (mainType === ITSMMainType.CHANGE) {
    return ChangeStatusConfig[status as ChangeStatus] || ChangeStatusConfig[ChangeStatus.DRAFT];
  }
  if (mainType === ITSMMainType.INCIDENT) {
    // 事件使用工单状态
    return TicketStatusConfig[status as TicketStatus] || TicketStatusConfig[TicketStatus.NEW];
  }
  // 默认返回工单状态配置
  return TicketStatusConfig[status as TicketStatus] || TicketStatusConfig[TicketStatus.OPEN];
};

/**
 * 根据优先级获取配置
 */
export const getPriorityConfig = (priority: string, mainType?: string) => {
  if (mainType === ITSMMainType.INCIDENT) {
    return IncidentPriorityConfig[priority as IncidentPriority] || IncidentPriorityConfig[IncidentPriority.P3];
  }
  return TicketPriorityConfig[priority as TicketPriority] || TicketPriorityConfig[TicketPriority.MEDIUM];
};

/**
 * 获取主类型的颜色
 */
export const getMainTypeColor = (type: string): string => {
  const config = getMainTypeConfig(type);
  const colorMap: Record<string, string> = {
    red: '#ff4d4f',
    blue: '#1890ff',
    orange: '#fa8c16',
    green: '#52c41a',
    purple: '#722ed1',
    default: '#8c8c8c',
  };
  return colorMap[config.color] || colorMap.default;
};

/**
 * 获取所有主类型选项（用于下拉选择）
 */
export const getMainTypeOptions = () => {
  return Object.entries(ITSMMainTypeConfig).map(([value, config]) => ({
    value,
    label: config.label,
    color: config.color,
    icon: config.icon,
  }));
};

/**
 * 判断是否有效的状态
 */
export const isValidStatus = (status: string, mainType?: string): boolean => {
  if (mainType === ITSMMainType.CHANGE) {
    return status in ChangeStatus;
  }
  return status in TicketStatus;
};

/**
 * 判断是否有效的优先级
 */
export const isValidPriority = (priority: string, mainType?: string): boolean => {
  if (mainType === ITSMMainType.INCIDENT) {
    return priority in IncidentPriority;
  }
  return priority in TicketPriority;
};
