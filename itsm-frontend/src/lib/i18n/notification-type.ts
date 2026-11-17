/**
 * 通知类型词表 —— 后端枚举到 i18n 键的唯一映射源。
 *
 * 背景（缺陷 O6-A）：后端 `deliverInApp` / `ticket_notification_service` 把
 * event_type 短枚举直接写进 `notifications.title` 列（`created` / `status_changed` /
 * `sla_warning` …），列表页标题又优先渲染 `title`，于是英文枚举原样上屏，
 * `getNotificationTypeLabel` 的字典沦为几乎不生效的兜底分支。
 *
 * 词表同时收纳三套历史写法，避免各页各自维护不完整的局部字典：
 *   1. 短枚举（ticket_notification_dto `Type` 值域）
 *   2. 偏好长词（`normalizeNotificationEventType` 归一化后的 EVENT_TYPES 词表）
 *   3. 通知抽屉既有 case 分支（NotificationDrawer）
 */

/** 单个通知类型的 i18n 键（`notifications.type.*` 命名空间）。 */
export type NotificationTypeKey =
  | 'created'
  | 'assigned'
  | 'statusChanged'
  | 'commented'
  | 'slaWarning'
  | 'slaViolated'
  | 'resolved'
  | 'closed'
  | 'approvalRequired'
  | 'approvalCompleted'
  | 'info';

/**
 * 后端枚举（短词 + 偏好长词）→ 词表键。
 *
 * 未登记的值不走 `default` 兜底静默透传，由调用方决定回退到 `info` 还是原始文案
 * （见 `notificationTypeKey` 的返回值语义），避免把未知枚举直接上屏。
 */
const NOTIFICATION_TYPE_KEYS: Readonly<Record<string, NotificationTypeKey>> = {
  // 1. 短枚举（落库/推送实际使用的值）
  created: 'created',
  assigned: 'assigned',
  updated: 'statusChanged',
  status_changed: 'statusChanged',
  commented: 'commented',
  sla_warning: 'slaWarning',
  sla_alert: 'slaWarning',
  sla_breached: 'slaViolated',
  resolved: 'resolved',
  closed: 'closed',
  change_approval_required: 'approvalRequired',
  change_approval_decided: 'approvalCompleted',
  info: 'info',

  // 2. 偏好长词（EVENT_TYPES 词表 / normalizeNotificationEventType 输出）
  ticket_created: 'created',
  ticket_assigned: 'assigned',
  ticket_updated: 'statusChanged',
  ticket_commented: 'commented',
  comment_added: 'commented',
  ticket_resolved: 'resolved',
  ticket_closed: 'closed',
  sla_violated: 'slaViolated',
  approval_required: 'approvalRequired',
  approval_completed: 'approvalCompleted',
};

/**
 * 解析通知类型对应的词表键。
 * @returns 命中返回词表键；未登记返回 `null`（调用方须自行决定回退，不要直接透传枚举）
 */
export function notificationTypeKey(type: string | undefined | null): NotificationTypeKey | null {
  if (!type) return null;
  const normalized = type.trim();
  if (!normalized) return null;
  return NOTIFICATION_TYPE_KEYS[normalized] ?? null;
}

/**
 * 判断一段文本本身是否为「裸枚举」——即后端误把 event_type 当标题写进来的情形。
 *
 * 标题列既可能是人工文案（如 BPMN 抄送通知的真实标题），也可能是枚举原文；
 * 只有后者才应被词表替换，否则会把正常标题吞掉。
 */
export function isNotificationTypeToken(value: string | undefined | null): boolean {
  return notificationTypeKey(value) !== null;
}

/**
 * 通知卡片标题的最终展示文本 —— 把「优先级 + 裸枚举替换」这条规则收在一处，
 * 便于回归锁直接断言（页面组件本身只负责调用）。
 *
 * 优先级：
 *   1. `title` 是人工文案 → 原样展示（BPMN 抄送等真实标题不能被吞）
 *   2. `title` 是 event_type 裸枚举 → 用词表翻译（O6-A 的核心修复）
 *   3. `title` 为空 → 退回 `content`
 *   4. 都为空 → 用 `type` 翻译兜底
 *
 * @param label 把词表键翻译成当前语言的文案（由调用方传入 `t`）
 */
export function resolveNotificationTitle(
  notification: {
    title?: string;
    content?: string;
    type?: string;
  },
  label: (key: NotificationTypeKey) => string
): string {
  const rawTitle = (notification.title || '').trim();
  const key = notificationTypeKey(rawTitle);
  if (key) return label(key);
  if (rawTitle) return rawTitle;

  const content = (notification.content || '').trim();
  if (content) return content;

  const typeKey = notificationTypeKey(notification.type);
  return typeKey ? label(typeKey) : label('info');
}
