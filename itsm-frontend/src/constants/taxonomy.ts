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
    label: '事件',
    color: 'red',
    icon: '🔴',
    description: '服务中断或服务质量下降的意外事件',
    badgeStatus: 'error',
  },
  [ITSMMainType.SERVICE_REQUEST]: {
    label: '服务请求',
    color: 'blue',
    icon: '📋',
    description: '用户提交的服务申请或信息咨询',
    badgeStatus: 'processing',
  },
  [ITSMMainType.PROBLEM]: {
    label: '问题',
    color: 'orange',
    icon: '🔍',
    description: '事件根本原因的调查和分析',
    badgeStatus: 'warning',
  },
  [ITSMMainType.CHANGE]: {
    label: '变更',
    color: 'green',
    icon: '🔄',
    description: 'IT基础设施或服务的变更请求',
    badgeStatus: 'success',
  },
  [ITSMMainType.KNOWLEDGE]: {
    label: '知识',
    color: 'purple',
    icon: '📚',
    description: '知识库文章',
    badgeStatus: 'default',
  },
};

// ==================== 工单状态 (Ticket Status) ====================
export enum TicketStatus {
  NEW = 'new',
  OPEN = 'open',
  IN_PROGRESS = 'in_progress',
  PENDING_APPROVAL = 'pending_approval',
  PENDING = 'pending',
  RESOLVED = 'resolved',
  CLOSED = 'closed',
  CANCELLED = 'cancelled',
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
    label: '新建',
    text: '新建',
    color: 'blue',
    bgColor: '#e6f7ff',
    textColor: '#1890ff',
    badgeStatus: 'processing',
  },
  [TicketStatus.OPEN]: {
    label: '待处理',
    text: '待处理',
    color: 'orange',
    bgColor: '#fff7e6',
    textColor: '#fa8c16',
    badgeStatus: 'warning',
  },
  [TicketStatus.IN_PROGRESS]: {
    label: '处理中',
    text: '处理中',
    color: 'cyan',
    bgColor: '#e6fffb',
    textColor: '#13c2c2',
    badgeStatus: 'processing',
  },
  [TicketStatus.PENDING_APPROVAL]: {
    label: '待审批',
    text: '待审批',
    color: 'gold',
    bgColor: '#fffbe6',
    textColor: '#faad14',
    badgeStatus: 'warning',
  },
  [TicketStatus.PENDING]: {
    label: '等待中',
    text: '等待中',
    color: 'purple',
    bgColor: '#f9f0ff',
    textColor: '#722ed1',
    badgeStatus: 'default',
  },
  [TicketStatus.RESOLVED]: {
    label: '已解决',
    text: '已解决',
    color: 'green',
    bgColor: '#f6ffed',
    textColor: '#52c41a',
    badgeStatus: 'success',
  },
  [TicketStatus.CLOSED]: {
    label: '已关闭',
    text: '已关闭',
    color: 'default',
    bgColor: '#fafafa',
    textColor: '#8c8c8c',
    badgeStatus: 'default',
  },
  [TicketStatus.CANCELLED]: {
    label: '已取消',
    text: '已取消',
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    badgeStatus: 'error',
  },
  [TicketStatus.REJECTED]: {
    label: '已拒绝',
    text: '已拒绝',
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
    label: '低',
    color: 'blue',
    bgColor: '#e6f7ff',
    textColor: '#1890ff',
    text: '低',
    icon: '↓',
    description: '一般问题，可延后处理',
  },
  [TicketPriority.MEDIUM]: {
    label: '中',
    color: 'orange',
    bgColor: '#fff7e6',
    textColor: '#fa8c16',
    text: '中',
    icon: '→',
    description: '正常问题，按序处理',
  },
  [TicketPriority.HIGH]: {
    label: '高',
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    text: '高',
    icon: '↑',
    description: '重要问题，优先处理',
  },
  [TicketPriority.URGENT]: {
    label: '紧急',
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    text: '紧急',
    icon: '↑↑',
    description: '严重影响业务，需立即处理',
  },
  [TicketPriority.CRITICAL]: {
    label: '紧急',
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    text: '紧急',
    icon: '↑↑',
    description: '核心业务中断，需立即处理',
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
    label: '普通变更',
    color: 'blue',
    icon: '📋',
    description: '标准变更流程，需要审批',
  },
  [ChangeType.STANDARD]: {
    label: '标准变更',
    color: 'green',
    icon: '✅',
    description: '预批准的低风险变更',
  },
  [ChangeType.EMERGENCY]: {
    label: '紧急变更',
    color: 'red',
    icon: '🚨',
    description: '紧急情况下的快速变更',
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
}

export const ChangeStatusConfig: Record<ChangeStatus, {
  label: string;
  color: string;
  badgeStatus: 'success' | 'processing' | 'warning' | 'error' | 'default';
}> = {
  [ChangeStatus.DRAFT]: {
    label: '草稿',
    color: 'default',
    badgeStatus: 'default',
  },
  [ChangeStatus.PENDING]: {
    label: '待审批',
    color: 'gold',
    badgeStatus: 'warning',
  },
  [ChangeStatus.APPROVED]: {
    label: '已批准',
    color: 'blue',
    badgeStatus: 'processing',
  },
  [ChangeStatus.REJECTED]: {
    label: '已拒绝',
    color: 'red',
    badgeStatus: 'error',
  },
  [ChangeStatus.SCHEDULED]: {
    label: '已排期',
    color: 'cyan',
    badgeStatus: 'processing',
  },
  [ChangeStatus.IN_PROGRESS]: {
    label: '实施中',
    color: 'cyan',
    badgeStatus: 'processing',
  },
  [ChangeStatus.COMPLETED]: {
    label: '已完成',
    color: 'green',
    badgeStatus: 'success',
  },
  [ChangeStatus.FAILED]: {
    label: '实施失败',
    color: 'red',
    badgeStatus: 'error',
  },
  [ChangeStatus.ROLLED_BACK]: {
    label: '已回滚',
    color: 'orange',
    badgeStatus: 'warning',
  },
  [ChangeStatus.CANCELLED]: {
    label: '已取消',
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
    label: 'P1 - 紧急',
    shortLabel: 'P1',
    color: 'red',
    description: '核心业务中断，影响所有用户',
    responseTime: '15分钟',
  },
  [IncidentPriority.P2]: {
    label: 'P2 - 高',
    shortLabel: 'P2',
    color: 'orange',
    description: '核心业务受损，影响大部分用户',
    responseTime: '1小时',
  },
  [IncidentPriority.P3]: {
    label: 'P3 - 中',
    shortLabel: 'P3',
    color: 'gold',
    description: '非核心业务受损，影响部分用户',
    responseTime: '4小时',
  },
  [IncidentPriority.P4]: {
    label: 'P4 - 低',
    shortLabel: 'P4',
    color: 'blue',
    description: '轻微问题，可延后处理',
    responseTime: '8小时',
  },
};

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
    label: '已提交',
    color: 'blue',
    badgeStatus: 'processing',
  },
  [ServiceRequestStatus.ACCEPTED]: {
    label: '已受理',
    color: 'cyan',
    badgeStatus: 'processing',
  },
  [ServiceRequestStatus.IN_PROGRESS]: {
    label: '处理中',
    color: 'purple',
    badgeStatus: 'processing',
  },
  [ServiceRequestStatus.PENDING_APPROVAL]: {
    label: '待审批',
    color: 'gold',
    badgeStatus: 'warning',
  },
  [ServiceRequestStatus.COMPLETED]: {
    label: '已完成',
    color: 'green',
    badgeStatus: 'success',
  },
  [ServiceRequestStatus.CANCELLED]: {
    label: '已取消',
    color: 'default',
    badgeStatus: 'default',
  },
  [ServiceRequestStatus.REJECTED]: {
    label: '已拒绝',
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
