/**
 * D49 回归：流程校验对非法结构给出「通过」假信号。
 * 修复后 buildConnectivityIssues 必须对以下结构产生 error/warning：
 * - 开始事件无出向连线
 * - 结束事件无入向连线
 * - 从开始事件不可达的孤儿节点
 * - 悬空连线（缺 source/target）
 * - generic bpmn:Task（未细化为 UserTask/ServiceTask 等）
 */
import { buildConnectivityIssues, type ConnectivityElement } from '../localValidation';

const t = (key: string, params?: Record<string, unknown>): string => {
  if (params && 'name' in (params as Record<string, unknown>)) {
    return `${key}::${String((params as Record<string, unknown>).name)}`;
  }
  if (params && 'id' in (params as Record<string, unknown>)) {
    return `${key}::${String((params as Record<string, unknown>).id)}`;
  }
  return key;
};

const seqFlow = (
  id: string,
  source?: { id: string } | null,
  target?: { id: string } | null
) => ({
  id,
  type: 'bpmn:SequenceFlow',
  source: source ?? undefined,
  target: target ?? undefined,
});

describe('buildConnectivityIssues (D49)', () => {
  it('flags a start event with no outgoing flow', () => {
    const elements: ConnectivityElement[] = [
      { id: 'StartEvent_1', type: 'bpmn:StartEvent', name: '开始', outgoing: [] },
      { id: 'EndEvent_1', type: 'bpmn:EndEvent', name: '结束', incoming: [] },
    ];
    const issues = buildConnectivityIssues(elements, t);
    expect(issues.some(i => i.type === 'error' && i.elementId === 'StartEvent_1' && i.message.startsWith('bpmnDesigner.messages.startEventNoOutgoing'))).toBe(true);
    expect(issues.some(i => i.type === 'error' && i.elementId === 'EndEvent_1' && i.message.startsWith('bpmnDesigner.messages.endEventNoIncoming'))).toBe(true);
  });

  it('flags nodes that are not reachable from any start event', () => {
    // 模拟 D49 实测场景：开始事件无出边，孤立 Task 只连到结束事件
    const start = { id: 'StartEvent_1', type: 'bpmn:StartEvent', name: '开始', outgoing: [] };
    const end = {
      id: 'EndEvent_1',
      type: 'bpmn:EndEvent',
      name: '结束',
      incoming: [{ id: 'Flow_1', source: { id: 'Task_1' } }],
    };
    const task: ConnectivityElement = {
      id: 'Task_1',
      type: 'bpmn:Task',
      name: undefined,
      incoming: [],
      outgoing: [{ id: 'Flow_1', target: { id: 'EndEvent_1' } }],
    };
    const flow = seqFlow('Flow_1', { id: 'Task_1' }, { id: 'EndEvent_1' });
    const issues = buildConnectivityIssues([start, task, end, flow], t);

    // 开始事件无出边 → error
    expect(issues.some(i => i.type === 'error' && i.message.startsWith('bpmnDesigner.messages.startEventNoOutgoing'))).toBe(true);
    // 孤儿 Task 从开始事件不可达 → error
    expect(issues.some(i => i.type === 'error' && i.elementId === 'Task_1' && i.message.startsWith('bpmnDesigner.messages.nodeUnreachable'))).toBe(true);
    // 结束事件从开始事件不可达 → error
    expect(issues.some(i => i.type === 'error' && i.elementId === 'EndEvent_1' && i.message.startsWith('bpmnDesigner.messages.nodeUnreachable'))).toBe(true);
    // generic Task → warning
    expect(issues.some(i => i.type === 'warning' && i.elementId === 'Task_1' && i.message.startsWith('bpmnDesigner.messages.genericTaskUnsupported'))).toBe(true);
  });

  it('passes a minimal connected start -> end flow without any issue', () => {
    const start = {
      id: 'StartEvent_1',
      type: 'bpmn:StartEvent',
      name: '开始',
      outgoing: [{ id: 'Flow_1', target: { id: 'EndEvent_1' } }],
    };
    const end = {
      id: 'EndEvent_1',
      type: 'bpmn:EndEvent',
      name: '结束',
      incoming: [{ id: 'Flow_1', source: { id: 'StartEvent_1' } }],
    };
    const flow = seqFlow('Flow_1', { id: 'StartEvent_1' }, { id: 'EndEvent_1' });
    const issues = buildConnectivityIssues([start, end, flow], t);
    expect(issues).toHaveLength(0);
  });

  it('flags a sequence flow missing an endpoint', () => {
    const start = {
      id: 'StartEvent_1',
      type: 'bpmn:StartEvent',
      outgoing: [{ id: 'Flow_1', target: { id: 'EndEvent_1' } }],
    };
    const end = {
      id: 'EndEvent_1',
      type: 'bpmn:EndEvent',
      incoming: [{ id: 'Flow_1', source: { id: 'StartEvent_1' } }],
    };
    const dangling = seqFlow('Flow_Dangling', null, { id: 'EndEvent_1' });
    const good = seqFlow('Flow_1', { id: 'StartEvent_1' }, { id: 'EndEvent_1' });
    const issues = buildConnectivityIssues([start, end, good, dangling], t);
    expect(issues.some(i => i.type === 'error' && i.elementId === 'Flow_Dangling' && i.message.startsWith('bpmnDesigner.messages.flowMissingEndpoint'))).toBe(true);
  });

  it('does not flag UserTask / ServiceTask as generic task', () => {
    const start = {
      id: 'StartEvent_1',
      type: 'bpmn:StartEvent',
      outgoing: [{ id: 'Flow_1', target: { id: 'Task_1' } }],
    };
    const task: ConnectivityElement = {
      id: 'Task_1',
      type: 'bpmn:UserTask',
      name: '审批',
      incoming: [{ id: 'Flow_1', source: { id: 'StartEvent_1' } }],
      outgoing: [{ id: 'Flow_2', target: { id: 'EndEvent_1' } }],
    };
    const end = {
      id: 'EndEvent_1',
      type: 'bpmn:EndEvent',
      incoming: [{ id: 'Flow_2', source: { id: 'Task_1' } }],
    };
    const flow1 = seqFlow('Flow_1', { id: 'StartEvent_1' }, { id: 'Task_1' });
    const flow2 = seqFlow('Flow_2', { id: 'Task_1' }, { id: 'EndEvent_1' });
    const issues = buildConnectivityIssues([start, task, end, flow1, flow2], t);
    expect(issues).toHaveLength(0);
  });
});
