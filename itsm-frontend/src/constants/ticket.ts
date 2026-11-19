/**
 * 工单（Ticket）枚举词表 —— 全产品单一源。
 *
 * 背景：工单状态/优先级此前没有权威词表，`PRIORITY_LABELS` 在 4 个不同文件
 * 各自定义、口径不一，展示层频繁出现 `new` / `in_progress` / `high` 这类
 * 后端原始值直接透给用户的情况（报表中心「工单统计」表格即一例）。
 *
 * 取值与后端 repository/ticket/model.go 的 Status 常量、优先级语义保持一致：
 *   StatusNew "new" / StatusOpen "open" / StatusInProgress "in_progress"
 *   StatusPending "pending" / StatusResolved "resolved" / StatusClosed "closed"
 *   StatusCancelled "cancelled"
 *
 * 约定：未知值一律原样返回，不做枚举断言——新增状态时宁可透出原始值，
 * 也不要被吞成空白或误导性的默认文案。
 */

export const TICKET_STATUS_LABELS: Record<string, string> = {
  new: '新建',
  open: '待处理',
  in_progress: '处理中',
  pending: '挂起',
  resolved: '已解决',
  closed: '已关闭',
  cancelled: '已取消',
};

/** 工单状态色（antd Tag/Status 语义色） */
export const TICKET_STATUS_COLORS: Record<string, string> = {
  new: 'blue',
  open: 'cyan',
  in_progress: 'processing',
  pending: 'orange',
  resolved: 'green',
  closed: 'default',
  cancelled: 'red',
};

export const TICKET_PRIORITY_LABELS: Record<string, string> = {
  low: '低',
  medium: '中',
  high: '高',
  urgent: '紧急',
  critical: '严重',
};

export const TICKET_PRIORITY_COLORS: Record<string, string> = {
  low: 'default',
  medium: 'blue',
  high: 'orange',
  urgent: 'red',
  critical: 'magenta',
};

export const ticketStatusLabel = (status?: string | null, fallback = '-'): string =>
  status ? TICKET_STATUS_LABELS[status] || status : fallback;

export const ticketPriorityLabel = (priority?: string | null, fallback = '-'): string =>
  priority ? TICKET_PRIORITY_LABELS[priority] || priority : fallback;

export const ticketStatusColor = (status?: string | null): string =>
  (status ? TICKET_STATUS_COLORS[status] : undefined) || 'default';

export const ticketPriorityColor = (priority?: string | null): string =>
  (priority ? TICKET_PRIORITY_COLORS[priority] : undefined) || 'default';
