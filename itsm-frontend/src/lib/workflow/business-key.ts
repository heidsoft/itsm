/**
 * 流程实例业务键（businessKey）解析与业务单据回跳。
 *
 * 后端写入格式固定为 `<businessType>:<businessID>`，见
 *   internal/services/bpmn_approval_bridge_service.go:283
 *   internal/services/ticket_workflow_service.go:1354
 *   internal/services/incident_service.go:2183（incident:%d）
 *   internal/services/workflow_start_command_handler.go:40（%s:%d）
 * 即 fmt.Sprintf("%s:%d", strings.ToLower(businessType), businessID)。
 *
 * 这个键是实例与业务单据之间唯一的关联锚点：用户在「我的单子走到哪了」这个问题上，
 * 需要的不是 `ticket:1` 这样的字符串，而是能点回工单详情的一跳。此前实例详情只把
 * businessKey 当文本渲染，用户看完流程还得回列表页自己找单。
 *
 * 刻意保持纯函数、不依赖任何 React / API，便于单测锁定解析口径。
 */

/** 业务类型词表，与后端 dto.BusinessType 枚举 1:1 对齐（bpmn_process_trigger_dto.go:9-14）。 */
export const BUSINESS_TYPES = [
  'ticket',
  'change',
  'incident',
  'service_request',
  'problem',
  'release',
] as const;

export type BusinessType = (typeof BUSINESS_TYPES)[number];

/** 业务类型的展示名。未登记类型回退原始词，不伪造中文名。 */
const BUSINESS_TYPE_LABELS: Record<BusinessType, string> = {
  ticket: '工单',
  change: '变更',
  incident: '事件',
  service_request: '服务请求',
  problem: '问题',
  release: '发布',
};

// 各业务单据详情页路径前缀（与前端 src/app/(main) 下各域 [id] 动态路由实测一致）。
const BUSINESS_DETAIL_PATHS: Record<BusinessType, string> = {
  ticket: '/tickets',
  change: '/changes',
  incident: '/incidents',
  service_request: '/service-requests',
  problem: '/problems',
  release: '/releases',
};

export interface ParsedBusinessKey {
  /** 归一化后的业务类型（小写）；无法识别时为 undefined */
  type?: BusinessType;
  /** 后端原始类型词（已小写）；未识别时保留原词，便于运维定位脏数据 */
  rawType: string;
  /** 业务单据数字 ID；非正整数时为 undefined（不外泄 NaN 给路由） */
  id?: number;
  /** 详情页路径；type 与 id 都有效时才产出 */
  href?: string;
  /** 展示名，如「工单 #1」；无法解析时回退原始键 */
  label: string;
}

const isBusinessType = (value: string): value is BusinessType =>
  (BUSINESS_TYPES as readonly string[]).includes(value);

/**
 * 解析业务键。
 *
 * 容错口径（都有 prod 现实依据）：
 * - 空串 / 非字符串 → 全部字段为空，label 为 '-'，让调用方直接显示占位符；
 * - 冒号取**第一个**（业务类型里不可能含冒号，但 ID 段可能带脏字符，
 *   如定时器的 `timer:<timerID>:<unix>`，取其头部才能定位到 timer 这一类）；
 * - ID 段必须是纯数字正整数，否则不产出 href —— 避免把 `NaN` 拼进路由；
 * - 未登记的业务类型保留 rawType 但不产出 href，前端显示原文而不是假装能跳。
 */
export function parseBusinessKey(businessKey?: string | null): ParsedBusinessKey {
  const raw = typeof businessKey === 'string' ? businessKey.trim() : '';
  if (!raw) {
    return { rawType: '', label: '-' };
  }

  const separator = raw.indexOf(':');
  if (separator <= 0) {
    // 没有 `type:id` 结构：可能是手工写入的租户自定义键，原样展示，
    // 绝不猜测 ID —— 猜错会把用户导到别人的单子上。
    return { rawType: '', label: raw };
  }

  const rawType = raw.slice(0, separator).toLowerCase();
  const idText = raw.slice(separator + 1);

  // 只接受纯数字：`12abc`、`1.5`、`-3`、` ` 一律视为无效 ID。
  const id = /^\d+$/.test(idText) ? Number(idText) : undefined;
  const validId = id !== undefined && Number.isSafeInteger(id) && id > 0 ? id : undefined;

  const type = isBusinessType(rawType) ? rawType : undefined;

  if (!type || validId === undefined) {
    // 类型未登记或 ID 非法：保留原文可读，但不提供跳转。
    return type
      ? { type, rawType, id: validId, label: `${BUSINESS_TYPE_LABELS[type]} #${idText}` }
      : { rawType, id: validId, label: raw };
  }

  return {
    type,
    rawType,
    id: validId,
    href: `${BUSINESS_DETAIL_PATHS[type]}/${validId}`,
    label: `${BUSINESS_TYPE_LABELS[type]} #${validId}`,
  };
}

/**
 * 业务类型的展示名。
 *
 * 单列一个函数而不是让调用方读 map：未登记类型必须先判空再取，
 * 直接下标访问在 TS 里返回 undefined，模板串会渲染成 "undefined"。
 */
export function businessTypeLabel(type?: string | null): string | undefined {
  const normalized = (type || '').toLowerCase();
  return isBusinessType(normalized) ? BUSINESS_TYPE_LABELS[normalized] : undefined;
}
