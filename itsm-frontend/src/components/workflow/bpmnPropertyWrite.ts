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

export function normalizeNodeProperties(
  patch: Record<string, unknown>,
  context: NormalizeContext
): NormalizeResult {
  const { moddle, resolveElement } = context;
  const properties: Record<string, unknown> = {};

  for (const [key, value] of Object.entries(patch)) {
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
