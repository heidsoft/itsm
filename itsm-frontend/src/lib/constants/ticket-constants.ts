/**
 * 工单状态和优先级统一配置
 * 用于确保整个应用中状态和优先级的颜色、文本显示一致
 */

// 工单状态配置
export const TICKET_STATUS_CONFIG = {
  new: {
    color: 'blue',
    bgColor: '#e6f7ff',
    textColor: '#1890ff',
    text: '新建',
    badgeStatus: 'processing' as const,
  },
  open: {
    color: 'orange',
    bgColor: '#fff7e6',
    textColor: '#fa8c16',
    text: '待处理',
    badgeStatus: 'warning' as const,
  },
  in_progress: {
    color: 'cyan',
    bgColor: '#e6fffb',
    textColor: '#13c2c2',
    text: '处理中',
    badgeStatus: 'processing' as const,
  },
  pending_approval: {
    color: 'gold',
    bgColor: '#fffbe6',
    textColor: '#faad14',
    text: '待审批',
    badgeStatus: 'warning' as const,
  },
  resolved: {
    color: 'green',
    bgColor: '#f6ffed',
    textColor: '#52c41a',
    text: '已解决',
    badgeStatus: 'success' as const,
  },
  closed: {
    color: 'default',
    bgColor: '#fafafa',
    textColor: '#8c8c8c',
    text: '已关闭',
    badgeStatus: 'default' as const,
  },
  cancelled: {
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    text: '已取消',
    badgeStatus: 'error' as const,
  },
} as const;

// 工单优先级配置
export const TICKET_PRIORITY_CONFIG = {
  low: {
    color: 'blue',
    bgColor: '#e6f7ff',
    textColor: '#1890ff',
    text: '低',
    icon: '↓',
    description: '一般问题，可延后处理',
  },
  medium: {
    color: 'orange',
    bgColor: '#fff7e6',
    textColor: '#fa8c16',
    text: '中',
    icon: '→',
    description: '正常问题，按序处理',
  },
  high: {
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    text: '高',
    icon: '↑',
    description: '重要问题，优先处理',
  },
  urgent: {
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    text: '紧急',
    icon: '↑↑',
    description: '严重影响业务，立即处理',
  },
  critical: {
    color: 'red',
    bgColor: '#fff1f0',
    textColor: '#ff4d4f',
    text: '紧急',
    icon: '↑↑',
    description: '严重影响业务，立即处理',
  },
} as const;

// 工单类型配置
export const TICKET_TYPE_CONFIG = {
  incident: {
    color: 'red',
    text: '事件',
    icon: '⚠️',
  },
  service_request: {
    color: 'blue',
    text: '服务请求',
    icon: '📋',
  },
  problem: {
    color: 'orange',
    text: '问题',
    icon: '🔍',
  },
  change: {
    color: 'green',
    text: '变更',
    icon: '🔄',
  },
} as const;

// 获取状态配置
export const getStatusConfig = (status: string) => {
  return TICKET_STATUS_CONFIG[status as keyof typeof TICKET_STATUS_CONFIG] || TICKET_STATUS_CONFIG.open;
};

// 获取优先级配置
export const getPriorityConfig = (priority: string) => {
  return TICKET_PRIORITY_CONFIG[priority as keyof typeof TICKET_PRIORITY_CONFIG] || TICKET_PRIORITY_CONFIG.medium;
};

// 获取类型配置
export const getTypeConfig = (type: string) => {
  return TICKET_TYPE_CONFIG[type as keyof typeof TICKET_TYPE_CONFIG] || TICKET_TYPE_CONFIG.service_request;
};

