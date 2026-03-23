/**
 * 工单状态和优先级统一配置
 * @deprecated 请使用 @/constants/taxonomy 中的统一分类系统
 */

import {
  TicketStatus,
  TicketPriority,
  ITSMMainType,
  TicketStatusConfig,
  TicketPriorityConfig,
  ITSMMainTypeConfig,
} from '@/constants/taxonomy';

// 保持向后兼容的导出
export { TicketStatus, TicketPriority, ITSMMainType };

// 工单状态配置 (legacy)
export const TICKET_STATUS_CONFIG = {
  new: TicketStatusConfig[TicketStatus.NEW],
  open: TicketStatusConfig[TicketStatus.OPEN],
  in_progress: TicketStatusConfig[TicketStatus.IN_PROGRESS],
  pending_approval: TicketStatusConfig[TicketStatus.PENDING_APPROVAL],
  resolved: TicketStatusConfig[TicketStatus.RESOLVED],
  closed: TicketStatusConfig[TicketStatus.CLOSED],
  cancelled: TicketStatusConfig[TicketStatus.CANCELLED],
} as const;

// 工单优先级配置 (legacy)
export const TICKET_PRIORITY_CONFIG = {
  low: TicketPriorityConfig[TicketPriority.LOW],
  medium: TicketPriorityConfig[TicketPriority.MEDIUM],
  high: TicketPriorityConfig[TicketPriority.HIGH],
  urgent: TicketPriorityConfig[TicketPriority.URGENT],
  critical: TicketPriorityConfig[TicketPriority.CRITICAL],
} as const;

// 工单类型配置 (legacy - 转换 label -> text)
export const TICKET_TYPE_CONFIG = {
  incident: { ...ITSMMainTypeConfig[ITSMMainType.INCIDENT], text: '事件' },
  service_request: { ...ITSMMainTypeConfig[ITSMMainType.SERVICE_REQUEST], text: '服务请求' },
  problem: { ...ITSMMainTypeConfig[ITSMMainType.PROBLEM], text: '问题' },
  change: { ...ITSMMainTypeConfig[ITSMMainType.CHANGE], text: '变更' },
} as const;

// 获取状态配置 (legacy)
export const getStatusConfig = (status: string) => {
  return (
    TICKET_STATUS_CONFIG[status as keyof typeof TICKET_STATUS_CONFIG] ||
    TICKET_STATUS_CONFIG.open
  );
};

// 获取优先级配置 (legacy)
export const getPriorityConfig = (priority: string) => {
  return (
    TICKET_PRIORITY_CONFIG[priority as keyof typeof TICKET_PRIORITY_CONFIG] ||
    TICKET_PRIORITY_CONFIG.medium
  );
};

// 获取类型配置 (legacy)
export const getTypeConfig = (type: string) => {
  return (
    TICKET_TYPE_CONFIG[type as keyof typeof TICKET_TYPE_CONFIG] ||
    TICKET_TYPE_CONFIG.service_request
  );
};
