/**
 * 工单状态/优先级/类型配置的兼容转发层。
 *
 * @deprecated 请直接使用 @/constants/taxonomy；本文件只做名称转发，不再维护第二套映射。
 *
 * 这里曾经用 inProgress / pendingApproval / serviceRequest 等 camelCase 键自建映射表，
 * 而后端返回的 ticket.status/priority/type 永远是 snake_case，
 * 因此 getStatusConfig('in_progress') 一直落到 open 兜底，徽标颜色与文案长期错显示；
 * pending_approval 也不属于工单状态词表（见 common/constants.go 的工单状态机）。
 * 现在键名由 TicketStatus/TicketPriority 枚举值直接给出，与后端词表逐字一致。
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

export const TICKET_STATUS_CONFIG = TicketStatusConfig;
export const TICKET_PRIORITY_CONFIG = TicketPriorityConfig;

// 工单类型配置（legacy - 转换 label -> text），键为后端 type 值
export const TICKET_TYPE_CONFIG = {
  [ITSMMainType.INCIDENT]: { ...ITSMMainTypeConfig[ITSMMainType.INCIDENT], text: '事件' },
  [ITSMMainType.SERVICE_REQUEST]: {
    ...ITSMMainTypeConfig[ITSMMainType.SERVICE_REQUEST],
    text: '服务请求',
  },
  [ITSMMainType.PROBLEM]: { ...ITSMMainTypeConfig[ITSMMainType.PROBLEM], text: '问题' },
  [ITSMMainType.CHANGE]: { ...ITSMMainTypeConfig[ITSMMainType.CHANGE], text: '变更' },
} as const;

/** 未知/未来新增的状态值仍需要可渲染的兜底，因此保留显式 fallback。 */
export const getStatusConfig = (status: string) =>
  TICKET_STATUS_CONFIG[status as TicketStatus] ?? TICKET_STATUS_CONFIG[TicketStatus.OPEN];

export const getPriorityConfig = (priority: string) =>
  TICKET_PRIORITY_CONFIG[priority as TicketPriority] ??
  TICKET_PRIORITY_CONFIG[TicketPriority.MEDIUM];

export const getTypeConfig = (type: string) => {
  const config = TICKET_TYPE_CONFIG[type as keyof typeof TICKET_TYPE_CONFIG];
  return config ?? TICKET_TYPE_CONFIG[ITSMMainType.SERVICE_REQUEST];
};
