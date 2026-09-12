/**
 * itsm 流程扩展属性的真实往返测试。
 *
 * 覆盖此前的盲区：WorkflowNodeInspector 写入的属性只有静态字段名断言，
 * 从未验证「bpmn-moddle 能不能把它写进 XML、重开流程后能不能读回来」。
 * 因此 followUpDate 漏声明、cc* 系列被塞进 $attrs（面板重开即显示为空）、
 * documentation/conditionExpression 写错形状导致整篇 saveXML 抛错，
 * 全都没有被测到。
 *
 * 真实 moddle 只能在 Node 侧运行（bpmn-moddle 及其依赖是纯 ESM，
 * next/jest 注入的 transformIgnorePatterns 会跳过 node_modules），
 * 所以这里由 Jest 传入 descriptor，交给 scripts/bpmn-moddle-roundtrip.mjs 执行。
 */
import { execFileSync } from 'node:child_process';
import path from 'node:path';

import itsmModdleDescriptor from '../itsm-moddle-descriptor';

type Check = Record<string, unknown>;
type DeclaredEntry = {
  target: string;
  name: string;
  fieldType: string;
  result: {
    emitted: boolean;
    namespaced: boolean;
    readBack: unknown;
    parkedInAttrs: boolean;
    failure: string | null;
  };
};

interface Report {
  declared: DeclaredEntry[];
  checks: Record<string, Check>;
}

const SCRIPT = path.resolve(__dirname, '../../../../scripts/bpmn-moddle-roundtrip.mjs');

function runRoundTrip(): Report {
  const stdout = execFileSync(
    process.execPath,
    [SCRIPT],
    {
      encoding: 'utf8',
      input: JSON.stringify({ descriptor: itsmModdleDescriptor }),
      maxBuffer: 16 * 1024 * 1024,
    }
  );
  return JSON.parse(stdout) as Report;
}

let report: Report;
let runFailure: string | null = null;

beforeAll(() => {
  try {
    report = runRoundTrip();
  } catch (error) {
    runFailure = error instanceof Error ? error.message : String(error);
  }
});

describe('声明属性必须写入 XML 并在重开后可读回', () => {
  it('脚本可用且覆盖了 descriptor 声明的全部属性', () => {
    expect(runFailure).toBeNull();
    const expected = itsmModdleDescriptor.types.reduce(
      (total, type) => total + type.properties.length,
      0
    );
    expect(report.declared).toHaveLength(expected);
  });

  it.each(
    itsmModdleDescriptor.types.flatMap(type =>
      type.properties.map(property => ({
        label: `${type.extends[0]}#${property.name}`,
        name: property.name,
      }))
    )
  )('$label', ({ name }) => {
    const entry = report.declared.find(declared => declared.name === name);
    if (!entry) throw new Error(`${name} 未出现在往返结果中`);
    expect(entry.result.failure).toBeNull();
    // 必须带 itsm 前缀落到 XML 上，而不是作为无限定名属性逃逸出去
    expect(entry.result.namespaced).toBe(true);
    expect(entry.result.parkedInAttrs).toBe(false);
    if (entry.fieldType === 'Boolean') {
      expect(entry.result.readBack).toBe(false);
    } else if (entry.fieldType === 'Integer') {
      expect(entry.result.readBack).toBe(7);
    } else {
      expect(entry.result.readBack).toBe('sample-value');
    }
  });
});

describe('后端契约对齐', () => {
  it('抄送/通知属性已声明（后端 BPMNServiceTask 会解析这些局部名）', () => {
    const declared = report.declared
      .filter(entry => entry.target === 'bpmn:ServiceTask')
      .map(entry => entry.name);

    expect(declared).toEqual(
      expect.arrayContaining([
        'ccType', 'ccUserIds', 'ccGroupIds', 'ccRoleIds', 'ccVariable', 'ccNotify',
        'notifyChannels',
      ])
    );
  });

  it('用户任务保留后端 BPMNUserTask 用到的全部语义属性', () => {
    const declared = report.declared
      .filter(entry => entry.target === 'bpmn:UserTask')
      .map(entry => entry.name);

    expect(declared).toEqual(
      expect.arrayContaining([
        'assignee', 'candidateUsers', 'candidateGroups', 'formKey', 'priority', 'dueDate',
        'taskPurpose', 'approvalMode', 'approvalThreshold', 'rejectStrategy', 'timeoutAction',
        'allowDelegate', 'allowAddApprover', 'commentRequiredOnReject',
      ])
    );
  });
});

describe('未声明属性不能作为可编辑控件暴露', () => {
  it('followUpDate 写出后读不回来（后端也没有对应字段，故面板保持禁用）', () => {
    const check = report.checks.followUpDate as {
      emitted: boolean;
      readBack: unknown;
      parkedInAttrs: boolean;
    };

    expect(check.emitted).toBe(true);
    expect(check.readBack).toBeNull();
    expect(check.parkedInAttrs).toBe(true);
  });
});

describe('写入形状：错误形状破坏整篇序列化，正确形状可往返', () => {
  it('字符串 documentation 会让 saveXML 抛错（这就是编辑静默丢失的根因）', () => {
    expect(report.checks.documentationAsString).toMatchObject({
      serialized: false,
      failure: expect.any(String),
    });
  });

  it('对象形态的 documentation 不报错但会被静默丢弃（更危险的失败方式）', () => {
    const check = report.checks.documentationAsObject as {
      serialized: boolean;
      xmlFragment: string;
      readBack: unknown;
    };
    expect(check.serialized).toBe(true);
    expect(check.xmlFragment).not.toContain('documentation');
    expect(check.readBack).toBeNull();
  });

  it('普通对象 conditionExpression 破坏序列化', () => {
    expect(report.checks.conditionAsPlainObject).toMatchObject({ serialized: false });
  });

  it('moddle.create 出来的 bpmn:Documentation 可往返并读回文本', () => {
    expect(report.checks.correctDocumentation).toMatchObject({
      xmlHasElement: true,
      readBackText: '节点说明',
    });
  });

  it('moddle.create 出来的 bpmn:FormalExpression 可往返且无解析告警', () => {
    const check = report.checks.correctCondition as { readBackBody: string; warnings: string[] };
    expect(check.readBackBody).toBe('${variables["amount"] > 10000}');
    expect(check.warnings).toEqual([]);
  });

  it('default 传字符串会写成字面量 "undefined"，破坏网关默认分支', () => {
    const check = report.checks.defaultAsString as { serialized: boolean; xmlFragment: string };
    expect(check.serialized).toBe(true);
    expect(check.xmlFragment).toContain('default="undefined"');
  });

  it('default 传连线 businessObject 时按 ID 序列化并可解析回引用', () => {
    expect(report.checks.correctDefault).toMatchObject({ xmlHasRef: true, readBackId: 'F2' });
  });

  it('operationRef 传字符串同样写成 "undefined"', () => {
    const check = report.checks.operationRefAsString as { xmlFragment: string };
    expect(check.xmlFragment).toContain('operationRef="undefined"');
  });

  it('在 descriptor 中重复声明标准属性名会让整个 serviceTask 解析失败', () => {
    const check = report.checks.standardNameCollision as {
      serviceTaskParsed: boolean;
      warnings: string[];
    };
    expect(check.serviceTaskParsed).toBe(false);
    expect(check.warnings.join(' ')).toMatch(/already|unparsable/i);
  });
});
