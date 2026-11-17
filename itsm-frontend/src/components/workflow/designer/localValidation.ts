import type { ValidationIssue } from './workflowValidation';

/**
 * D49：画布结构连通性校验。
 * 从 bpmn-js elementRegistry 提取的轻量视图；图形字段允许 undefined
 * （bpmn-js 对未连线节点不会注入 incoming/outgoing 数组）。
 */
export interface ConnectivityElement {
  id: string;
  type: string;
  name?: string;
  incoming?: ReadonlyArray<{ id: string; source?: { id: string } | null }>;
  outgoing?: ReadonlyArray<{ id: string; target?: { id: string } | null }>;
}

type TranslateFn = (key: string, params?: Record<string, string | number>) => string;

/** 仅参与连通性遍历的 FlowNode 类型；SequenceFlow / Lane / DataObject 等不算节点 */
function isFlowNode(type: string): boolean {
  return (
    type.endsWith('Event') ||
    type.endsWith('Task') ||
    type.endsWith('Gateway') ||
    type === 'bpmn:SubProcess' ||
    type === 'bpmn:CallActivity' ||
    type === 'bpmn:AdHocSubProcess' ||
    type === 'bpmn:Transaction'
  );
}

function isSequenceFlow(type: string): boolean {
  return type === 'bpmn:SequenceFlow';
}

/**
 * 生成连通性 / generic Task 相关的本地校验问题。
 *
 * 规则：
 * 1. 开始事件必须有出向连线（否则流程无法启动）。
 * 2. 结束事件必须有入向连线（否则永远不可达）。
 * 3. 所有 FlowNode 必须能从至少一个开始事件沿出向连线到达；不可达即 error。
 * 4. SequenceFlow 两端必须都有图形端点；缺失视为悬空连线 error。
 * 5. generic `bpmn:Task`（未细化为 UserTask/ServiceTask 等）引擎不可执行，warning。
 *
 * 该函数只依赖输入 elements 数组，可在单测中脱离 bpmn-js modeler 直接调用。
 */
export function buildConnectivityIssues(
  elements: ReadonlyArray<ConnectivityElement>,
  t: TranslateFn
): ValidationIssue[] {
  const issues: ValidationIssue[] = [];
  if (elements.length === 0) return issues;

  const nodes = elements.filter(el => isFlowNode(el.type));
  const flows = elements.filter(el => isSequenceFlow(el.type));
  const nodeIds = new Set(nodes.map(n => n.id));

  // 1) 起止事件连线完备性
  for (const node of nodes) {
    const name = node.name || node.id;
    if (node.type === 'bpmn:StartEvent') {
      const outgoing = node.outgoing ?? [];
      if (outgoing.length === 0) {
        issues.push({
          type: 'error',
          message: t('bpmnDesigner.messages.startEventNoOutgoing', { name }),
          elementId: node.id,
          elementType: node.type,
          elementName: name,
        });
      }
    }
    if (node.type === 'bpmn:EndEvent') {
      const incoming = node.incoming ?? [];
      if (incoming.length === 0) {
        issues.push({
          type: 'error',
          message: t('bpmnDesigner.messages.endEventNoIncoming', { name }),
          elementId: node.id,
          elementType: node.type,
          elementName: name,
        });
      }
    }
  }

  // 2) SequenceFlow 悬空检查
  for (const flow of flows) {
    const srcId = (flow as { source?: { id: string } | null }).source?.id;
    const tgtId = (flow as { target?: { id: string } | null }).target?.id;
    if (!srcId || !tgtId) {
      issues.push({
        type: 'error',
        message: t('bpmnDesigner.messages.flowMissingEndpoint', { id: flow.id }),
        elementId: flow.id,
        elementType: flow.type,
        elementName: flow.name || flow.id,
      });
    }
  }

  // 3) 从开始事件 BFS，所有 FlowNode 必须可达
  const adjacency = new Map<string, string[]>();
  for (const flow of flows) {
    const srcId = (flow as { source?: { id: string } | null }).source?.id;
    const tgtId = (flow as { target?: { id: string } | null }).target?.id;
    if (!srcId || !tgtId) continue;
    if (!nodeIds.has(srcId) || !nodeIds.has(tgtId)) continue;
    adjacency.set(srcId, [...(adjacency.get(srcId) ?? []), tgtId]);
  }

  const reachable = new Set<string>();
  const queue: string[] = [];
  for (const node of nodes) {
    if (node.type === 'bpmn:StartEvent') {
      reachable.add(node.id);
      queue.push(node.id);
    }
  }
  while (queue.length > 0) {
    const current = queue.shift()!;
    for (const next of adjacency.get(current) ?? []) {
      if (reachable.has(next)) continue;
      reachable.add(next);
      queue.push(next);
    }
  }

  if (reachable.size > 0) {
    for (const node of nodes) {
      if (reachable.has(node.id)) continue;
      const name = node.name || node.id;
      issues.push({
        type: 'error',
        message: t('bpmnDesigner.messages.nodeUnreachable', { name }),
        elementId: node.id,
        elementType: node.type,
        elementName: name,
      });
    }
  }

  // 4) generic bpmn:Task 引擎不可执行，至少给 warning
  for (const node of nodes) {
    if (node.type !== 'bpmn:Task') continue;
    const name = node.name || node.id;
    issues.push({
      type: 'warning',
      message: t('bpmnDesigner.messages.genericTaskUnsupported', { name }),
      elementId: node.id,
      elementType: node.type,
      elementName: name,
    });
  }

  return issues;
}
