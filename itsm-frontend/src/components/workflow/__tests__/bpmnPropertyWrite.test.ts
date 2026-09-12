/**
 * bpmnPropertyWrite 回归测试。
 *
 * 这些用例对应真实缺陷：WorkflowNodeInspector 曾把 documentation 当字符串、
 * conditionExpression 当普通对象、default/operationRef 当字符串 ID 直接写入
 * businessObject。bpmn-js 的 modeling.updateProperties 不做校验，导致
 * - documentation / conditionExpression：整篇 saveXML 抛错，画布停止同步，
 *   保存回退到旧 XML 并提示成功（编辑静默丢失）；
 * - default / operationRef：序列化成字面量 `="undefined"`，引用关系被破坏。
 */
import {
  normalizeNodeProperties,
  readConditionExpressionText,
  readDocumentationText,
  readReferenceId,
  type ModdleCreate,
} from '../bpmnPropertyWrite';

interface CreatedElement {
  type: string;
  attributes: Record<string, unknown> | undefined;
}

function makeModdle() {
  const created: CreatedElement[] = [];
  const moddle: ModdleCreate = {
    create: (type, attributes) => {
      created.push({ type, attributes });
      return { $type: type, ...attributes };
    },
  };
  return { moddle, created };
}

const FLOW_BO = { $type: 'bpmn:SequenceFlow', id: 'Flow_2' };

function makeContext(moddle: ModdleCreate) {
  return {
    moddle,
    resolveElement: (id: string) => (id === 'Flow_2' ? { businessObject: FLOW_BO } : undefined),
  };
}

describe('normalizeNodeProperties — documentation', () => {
  it('wraps text into a bpmn:Documentation list', () => {
    const { moddle, created } = makeModdle();
    const result = normalizeNodeProperties({ documentation: '节点说明' }, makeContext(moddle));

    expect(result.ok).toBe(true);
    expect(created).toEqual([{ type: 'bpmn:Documentation', attributes: { text: '节点说明' } }]);
    expect(result.ok && result.properties.documentation).toEqual([
      { $type: 'bpmn:Documentation', text: '节点说明' },
    ]);
  });

  it('clears the element with an empty list instead of an empty string', () => {
    const { moddle } = makeModdle();
    for (const value of ['', undefined, null]) {
      const result = normalizeNodeProperties({ documentation: value }, makeContext(moddle));
      expect(result.ok && result.properties.documentation).toEqual([]);
    }
  });

  it('passes an already built Documentation list through', () => {
    const { moddle } = makeModdle();
    const list = [{ $type: 'bpmn:Documentation', text: 'x' }];
    const result = normalizeNodeProperties({ documentation: list }, makeContext(moddle));
    expect(result.ok && result.properties.documentation).toBe(list);
  });

  it('rejects a non-text value instead of corrupting the document', () => {
    const { moddle } = makeModdle();
    const result = normalizeNodeProperties({ documentation: 123 }, makeContext(moddle));
    expect(result).toEqual({ ok: false, error: '描述信息只能是文本' });
  });
});

describe('normalizeNodeProperties — conditionExpression', () => {
  it('builds a real bpmn:FormalExpression from text', () => {
    const { moddle, created } = makeModdle();
    const result = normalizeNodeProperties(
      { conditionExpression: '${variables["amount"] > 10000}' },
      makeContext(moddle)
    );

    expect(created).toEqual([
      { type: 'bpmn:FormalExpression', attributes: { body: '${variables["amount"] > 10000}' } },
    ]);
    expect(result.ok && (result.properties.conditionExpression as { $type: string }).$type).toBe(
      'bpmn:FormalExpression'
    );
  });

  it('converts the legacy hand-built object shape (type/body, no $type)', () => {
    const { moddle } = makeModdle();
    const result = normalizeNodeProperties(
      {
        conditionExpression: {
          type: 'bpmn:FormalExpression',
          body: '${a > 1}',
          language: 'javascript',
        },
      },
      makeContext(moddle)
    );

    // 旧形状会让 saveXML 整篇抛错，这里必须转换而不是原样透传
    expect(result.ok && result.properties.conditionExpression).toEqual({
      $type: 'bpmn:FormalExpression',
      body: '${a > 1}',
    });
  });

  it('keeps an existing moddle element untouched', () => {
    const { moddle } = makeModdle();
    const existing = { $type: 'bpmn:FormalExpression', body: '${ok}' };
    const result = normalizeNodeProperties({ conditionExpression: existing }, makeContext(moddle));
    expect(result.ok && result.properties.conditionExpression).toBe(existing);
  });

  it('clears with null', () => {
    const { moddle } = makeModdle();
    const result = normalizeNodeProperties({ conditionExpression: '' }, makeContext(moddle));
    expect(result.ok && result.properties.conditionExpression).toBeNull();
  });
});

describe('normalizeNodeProperties — reference properties', () => {
  it('resolves gateway default flow text to the target business object', () => {
    const { moddle } = makeModdle();
    const result = normalizeNodeProperties({ default: 'Flow_2' }, makeContext(moddle));
    expect(result.ok && result.properties.default).toBe(FLOW_BO);
  });

  it('rejects an unknown default flow id', () => {
    const { moddle } = makeModdle();
    const result = normalizeNodeProperties({ default: 'Flow_missing' }, makeContext(moddle));
    expect(result).toEqual({ ok: false, error: '默认分支 "Flow_missing" 不是当前流程中的连线' });
  });

  it('rejects writing operationRef as text (serializes as "undefined")', () => {
    const { moddle } = makeModdle();
    const result = normalizeNodeProperties({ operationRef: 'cc_handler' }, makeContext(moddle));
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain('operationRef');
  });

  it('rejects sourceRef/targetRef text writes as well', () => {
    const { moddle } = makeModdle();
    expect(normalizeNodeProperties({ sourceRef: 'A' }, makeContext(moddle)).ok).toBe(false);
    expect(normalizeNodeProperties({ targetRef: 'B' }, makeContext(moddle)).ok).toBe(false);
  });
});

describe('normalizeNodeProperties — pass-through properties', () => {
  it('leaves declared itsm attributes as plain values', () => {
    const { moddle } = makeModdle();
    const patch = {
      assignee: '42',
      candidateUsers: '1,2',
      candidateGroups: '运维组',
      formKey: 'approve_form_v1',
      taskPurpose: 'approval',
      approvalThreshold: 3,
      commentRequiredOnReject: false,
      ccType: 'role',
      name: '审批',
    };
    const result = normalizeNodeProperties(patch, makeContext(moddle));
    expect(result).toEqual({ ok: true, properties: patch });
  });
});

describe('panel readers', () => {
  it('reads documentation text out of the moddle list', () => {
    expect(readDocumentationText({ documentation: [{ text: '说明' }] })).toBe('说明');
    expect(readDocumentationText({ documentation: [] })).toBe('');
    expect(readDocumentationText({})).toBe('');
    expect(readDocumentationText(undefined)).toBe('');
    // 列表项不是对象时不能把 [object Object] 显示给用户
    expect(readDocumentationText({ documentation: [null] })).toBe('');
  });

  it('reads a reference id from either a string or a resolved object', () => {
    expect(readReferenceId({ default: FLOW_BO }, 'default')).toBe('Flow_2');
    expect(readReferenceId({ default: 'Flow_2' }, 'default')).toBe('Flow_2');
    expect(readReferenceId({ default: null }, 'default')).toBe('');
    expect(readReferenceId({}, 'default')).toBe('');
  });

  it('reads condition expression body', () => {
    expect(readConditionExpressionText({ conditionExpression: { body: '${a>1}' } })).toBe('${a>1}');
    expect(readConditionExpressionText({ conditionExpression: '${a>1}' })).toBe('${a>1}');
    expect(readConditionExpressionText({})).toBe('');
  });
});
