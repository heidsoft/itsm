// 节点属性写入前的形状规范化。
//
// bpmn-js 的 modeling.updateProperties 会原样执行 businessObject.set(key, value)，
// 不做任何类型校验。对引用型属性（default）传字符串、或对列表型属性
// （documentation / conditionExpression）传普通值，moddle-xml 在写整篇文档时抛错，
// 导致 saveXML 失败；调用方若忽略该异常，就会继续把上一次的旧 XML 当最新内容保存。

export interface ModdleCreate {
  create(type: string, attributes?: Record<string, unknown>): unknown;
}

export interface NormalizeContext {
  moddle: ModdleCreate;
  resolveElement?: (elementId: string) => { businessObject?: unknown } | undefined;
  /** 当前元素 businessObject：timer 表达式需读取现有 eventDefinitions 以保留其他事件定义 */
  currentBusinessObject?: Record<string, unknown>;
}

export type NormalizeResult =
  | { ok: true; properties: Record<string, unknown> }
  | { ok: false; error: string };

/**
 * BPMN 标准引用型属性：写入时必须传目标 businessObject，传字符串会被序列化成
 * 字面量 `="undefined"`，从而破坏网关/任务的引用关系。
 */
const REFERENCE_PROPERTIES = new Set([
  'default',
  'attachedToRef',
  'incoming',
  'outgoing',
  'operationRef',
  'sourceRef',
  'targetRef',
]);

function isNil(value: unknown): boolean {
  return value === undefined || value === null || value === '';
}

function normalizeDocumentation(value: unknown, moddle: ModdleCreate): NormalizeResult {
  if (isNil(value)) return { ok: true, properties: { documentation: [] } };
  if (Array.isArray(value)) return { ok: true, properties: { documentation: value } };
  if (typeof value !== 'string') {
    return { ok: false, error: '描述信息只能是文本' };
  }
  return {
    ok: true,
    properties: { documentation: [moddle.create('bpmn:Documentation', { text: value })] },
  };
}

function normalizeConditionExpression(value: unknown, moddle: ModdleCreate): NormalizeResult {
  if (isNil(value)) return { ok: true, properties: { conditionExpression: null } };

  if (typeof value === 'string') {
    return {
      ok: true,
      properties: { conditionExpression: moddle.create('bpmn:FormalExpression', { body: value }) },
    };
  }

  if (typeof value === 'object') {
    const candidate = value as { $type?: string; body?: unknown };
    if (candidate.$type) return { ok: true, properties: { conditionExpression: value } };
    if (typeof candidate.body === 'string') {
      return {
        ok: true,
        properties: {
          conditionExpression: moddle.create('bpmn:FormalExpression', { body: candidate.body }),
        },
      };
    }
  }

  return { ok: false, error: '条件表达式只能是文本或 bpmn:FormalExpression' };
}

function normalizeDefaultFlow(
  value: unknown,
  resolveElement: NormalizeContext['resolveElement']
): NormalizeResult {
  if (isNil(value)) return { ok: true, properties: { default: null } };
  if (typeof value !== 'string') return { ok: true, properties: { default: value } };
  if (!resolveElement) {
    return { ok: false, error: '无法解析默认分支：画布未就绪' };
  }
  const target = resolveElement(value);
  if (!target?.businessObject) {
    return { ok: false, error: `默认分支 "${value}" 不是当前流程中的连线` };
  }
  return { ok: true, properties: { default: target.businessObject } };
}

/** Timer 表达式在 TimerEventDefinition 上的三种载体 */
const TIMER_EXPRESSION_KEYS = ['timeDuration', 'timeDate', 'timeCycle'] as const;

interface TimerCarrier {
  eventDefinitions?: Array<Record<string, unknown>>;
}

interface TimerExpressionBody {
  body?: unknown;
}

function readTimerBodies(businessObject: Record<string, unknown> | undefined): Record<(typeof TIMER_EXPRESSION_KEYS)[number], string> {
  const definitions = (businessObject as TimerCarrier | undefined)?.eventDefinitions;
  const timerDef = Array.isArray(definitions)
    ? definitions.find(d => (d as { $type?: string }).$type === 'bpmn:TimerEventDefinition')
    : undefined;
  const read = (def: Record<string, unknown> | undefined, key: string): string => {
    if (!def) return '';
    const raw = def[key];
    if (typeof raw === 'string') return raw;
    if (raw && typeof raw === 'object') {
      const body = (raw as TimerExpressionBody).body;
      return typeof body === 'string' ? body : '';
    }
    return '';
  };
  return {
    timeDuration: read(timerDef, 'timeDuration'),
    timeDate: read(timerDef, 'timeDate'),
    timeCycle: read(timerDef, 'timeCycle'),
  };
}

/**
 * 把 timeDuration/timeDate/timeCycle 写入（或创建）bpmn:TimerEventDefinition：
 * - 三种表达式互斥，切换类型时清空其余两种；
 * - 全部为空时移除 TimerEventDefinition（避免产生无表达式的非法定时事件）；
 * - 保留 eventDefinitions 中的其他事件定义（消息/信号等）。
 */
function normalizeTimerExpressions(
  timerPatch: Record<string, unknown>,
  currentBusinessObject: Record<string, unknown> | undefined,
  moddle: ModdleCreate
): NormalizeResult {
  const bodies = readTimerBodies(currentBusinessObject);
  for (const [key, value] of Object.entries(timerPatch)) {
    if (!TIMER_EXPRESSION_KEYS.includes(key as (typeof TIMER_EXPRESSION_KEYS)[number])) continue;
    bodies[key as (typeof TIMER_EXPRESSION_KEYS)[number]] = typeof value === 'string' ? value : '';
  }

  const attributes: Record<string, unknown> = {};
  for (const key of TIMER_EXPRESSION_KEYS) {
    if (bodies[key] !== '') {
      attributes[key] = moddle.create('bpmn:FormalExpression', { body: bodies[key] });
    }
  }

  const definitions = (currentBusinessObject as TimerCarrier | undefined)?.eventDefinitions;
  const others = Array.isArray(definitions)
    ? definitions.filter(d => (d as { $type?: string }).$type !== 'bpmn:TimerEventDefinition')
    : [];

  const eventDefinitions =
    Object.keys(attributes).length > 0
      ? [...others, moddle.create('bpmn:TimerEventDefinition', attributes)]
      : others;

  return { ok: true, properties: { eventDefinitions } };
}

export function normalizeNodeProperties(
  patch: Record<string, unknown>,
  context: NormalizeContext
): NormalizeResult {
  const { moddle, resolveElement, currentBusinessObject } = context;
  const properties: Record<string, unknown> = {};

  // timer 表达式（timeDuration/timeDate/timeCycle）必须写进 bpmn:TimerEventDefinition
  // 子元素，直接 set 到 businessObject 顶层会产生非法 XML（后端 extractor 读不到）。
  const timerPatch: Record<string, unknown> = {};
  for (const key of TIMER_EXPRESSION_KEYS) {
    if (key in patch) timerPatch[key] = patch[key];
  }
  if (Object.keys(timerPatch).length > 0) {
    const result = normalizeTimerExpressions(timerPatch, currentBusinessObject, moddle);
    if (!result.ok) return result;
    Object.assign(properties, result.properties);
  }

  for (const [key, value] of Object.entries(patch)) {
    if (key in timerPatch) continue;
    if (key === 'documentation') {
      const result = normalizeDocumentation(value, moddle);
      if (!result.ok) return result;
      Object.assign(properties, result.properties);
      continue;
    }

    if (key === 'conditionExpression') {
      const result = normalizeConditionExpression(value, moddle);
      if (!result.ok) return result;
      Object.assign(properties, result.properties);
      continue;
    }

    if (key === 'default') {
      const result = normalizeDefaultFlow(value, resolveElement);
      if (!result.ok) return result;
      Object.assign(properties, result.properties);
      continue;
    }

    if (REFERENCE_PROPERTIES.has(key)) {
      return {
        ok: false,
        error: `属性 ${key} 是 BPMN 引用，不能按文本写入`,
      };
    }

    properties[key] = value;
  }

  return { ok: true, properties };
}

/** 把 moddle 中列表/引用型的值还原成面板可编辑的文本 */
export function readDocumentationText(businessObject: Record<string, unknown> | undefined): string {
  const documentation = businessObject?.documentation;
  if (typeof documentation === 'string') return documentation;
  if (Array.isArray(documentation)) {
    const first = documentation[0] as { text?: unknown } | undefined;
    return typeof first?.text === 'string' ? first.text : '';
  }
  return '';
}

export function readReferenceId(businessObject: Record<string, unknown> | undefined, key: string): string {
  const value = businessObject?.[key];
  if (typeof value === 'string') return value;
  if (value && typeof value === 'object') {
    const id = (value as { id?: unknown }).id;
    return typeof id === 'string' ? id : '';
  }
  return '';
}

export function readConditionExpressionText(
  businessObject: Record<string, unknown> | undefined
): string {
  const expression = businessObject?.conditionExpression;
  if (typeof expression === 'string') return expression;
  if (expression && typeof expression === 'object') {
    const body = (expression as { body?: unknown }).body;
    return typeof body === 'string' ? body : '';
  }
  return '';
}

/** 读取元素 TimerEventDefinition 上的时间表达式（按 timeDuration/timeDate/timeCycle） */
export function readTimerExpressionText(
  businessObject: Record<string, unknown> | undefined,
  key: 'timeDuration' | 'timeDate' | 'timeCycle'
): string {
  return readTimerBodies(businessObject)[key];
}

/** 元素是否挂有 TimerEventDefinition */
export function hasTimerDefinition(businessObject: Record<string, unknown> | undefined): boolean {
  const bodies = readTimerBodies(businessObject);
  return bodies.timeDuration !== '' || bodies.timeDate !== '' || bodies.timeCycle !== '';
}
