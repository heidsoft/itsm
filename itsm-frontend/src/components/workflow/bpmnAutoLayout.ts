// BPMN DI 自动补全工具
// BPMN Diagram Interchange (DI) Auto-Completion Utility
//
// 用途：当导入的 BPMN XML 缺少 <bpmndi:BPMNDiagram> 部分时（例如 LLM 直接吐
// 出来的纯流程结构），bpmn-js 的 `importXML` 会在渲染阶段抛
// "no diagram to display" 错误。本模块负责用 moddle API + dagre 自动布局，
// 给缺失 DI 的 Definitions 注入完整的 BPMNDiagram / BPMNPlane / BPMNShape /
// BPMNEdge，使画布可以正常显示与编辑。

import dagre from 'dagre';

/**
 * 节点类型 → 默认尺寸（像素）。bpmn-js 自带默认尺寸，这里给出常用的兜底值，
 * 对未识别的类型使用通用尺寸。开始/结束事件略小，任务与网关略大。
 */
const NODE_SIZE: Record<string, { width: number; height: number }> = {
  'bpmn:StartEvent': { width: 50, height: 50 },
  'bpmn:EndEvent': { width: 50, height: 50 },
  'bpmn:IntermediateThrowEvent': { width: 50, height: 50 },
  'bpmn:IntermediateCatchEvent': { width: 50, height: 50 },
  'bpmn:BoundaryEvent': { width: 50, height: 50 },
  'bpmn:Task': { width: 100, height: 80 },
  'bpmn:UserTask': { width: 100, height: 80 },
  'bpmn:ServiceTask': { width: 100, height: 80 },
  'bpmn:ScriptTask': { width: 100, height: 80 },
  'bpmn:ManualTask': { width: 100, height: 80 },
  'bpmn:BusinessRuleTask': { width: 100, height: 80 },
  'bpmn:SendTask': { width: 100, height: 80 },
  'bpmn:ReceiveTask': { width: 100, height: 80 },
  'bpmn:CallActivity': { width: 100, height: 80 },
  'bpmn:SubProcess': { width: 120, height: 80 },
  'bpmn:AdHocSubProcess': { width: 120, height: 80 },
  'bpmn:Transaction': { width: 120, height: 80 },
  'bpmn:Gateway': { width: 50, height: 50 },
  'bpmn:ExclusiveGateway': { width: 50, height: 50 },
  'bpmn:ParallelGateway': { width: 50, height: 50 },
  'bpmn:InclusiveGateway': { width: 50, height: 50 },
  'bpmn:EventBasedGateway': { width: 50, height: 50 },
  'bpmn:ComplexGateway': { width: 50, height: 50 },
  'bpmn:Participant': { width: 600, height: 200 },
  'bpmn:Lane': { width: 600, height: 100 },
};
const FALLBACK_NODE_SIZE = { width: 100, height: 80 };

const dagreRankSep = 80;
const dagreNodeSep = 60;
const dagreMargin = 40;

export interface ModdleLike {
  create(type: string, props?: Record<string, unknown>): unknown;
  getDefinitions?(): unknown;
}

export interface ModdleElementLike {
  $type: string;
  id?: string;
  name?: string;
  flowElements?: ModdleElementLike[];
  laneSets?: ModdleElementLike[];
  participants?: ModdleElementLike[];
  artifacts?: ModdleElementLike[];
  sourceRef?: ModdleElementLike;
  targetRef?: ModdleElementLike;
  bpmnElement?: ModdleElementLike;
  get<T = unknown>(prop: string): T;
  set?(prop: string, value: unknown): void;
}

export interface DefinitionsLike extends ModdleElementLike {
  rootElements: ModdleElementLike[];
  diagrams?: ModdleElementLike[];
}

/**
 * 返回节点的默认尺寸。优先匹配具体子类，未命中时走通用尺寸。
 */
function getNodeSize(type: string): { width: number; height: number } {
  return NODE_SIZE[type] ?? FALLBACK_NODE_SIZE;
}

/**
 * 收集 Process 内的所有 FlowNode（事件、任务、网关、子流程）。
 * SequenceFlow 不属于 FlowNode，会在另一处单独处理。
 */
function collectFlowNodes(process: ModdleElementLike): ModdleElementLike[] {
  const nodes: ModdleElementLike[] = [];
  const visit = (el: ModdleElementLike) => {
    if (!el || typeof el.$type !== 'string') return;
    const t = el.$type;
    if (t === 'bpmn:SequenceFlow') return;
    if (
      t === 'bpmn:FlowNode' ||
      t === 'bpmn:Task' ||
      t === 'bpmn:UserTask' ||
      t === 'bpmn:ServiceTask' ||
      t === 'bpmn:ScriptTask' ||
      t === 'bpmn:ManualTask' ||
      t === 'bpmn:BusinessRuleTask' ||
      t === 'bpmn:SendTask' ||
      t === 'bpmn:ReceiveTask' ||
      t === 'bpmn:Gateway' ||
      t === 'bpmn:ExclusiveGateway' ||
      t === 'bpmn:ParallelGateway' ||
      t === 'bpmn:InclusiveGateway' ||
      t === 'bpmn:EventBasedGateway' ||
      t === 'bpmn:ComplexGateway' ||
      t === 'bpmn:SubProcess' ||
      t === 'bpmn:AdHocSubProcess' ||
      t === 'bpmn:Transaction' ||
      t === 'bpmn:CallActivity' ||
      t.endsWith('Event')
    ) {
      nodes.push(el);
    }
    // 递归子流程 / 事务 / 临时子流程 / 泳道
    if (el.flowElements && Array.isArray(el.flowElements)) {
      el.flowElements.forEach(visit);
    }
    if (el.laneSets && Array.isArray(el.laneSets)) {
      el.laneSets.forEach((ls: ModdleElementLike) => {
        const lanes = (ls as ModdleElementLike & { lanes?: ModdleElementLike[] }).lanes;
        if (lanes && Array.isArray(lanes)) {
          lanes.forEach((lane: ModdleElementLike) => {
            const laneFlow = (lane as ModdleElementLike & { flowNodeRef?: ModdleElementLike[] }).flowNodeRef;
            if (laneFlow) {
              laneFlow.forEach(visit);
            }
          });
        }
      });
    }
  };
  if (process.flowElements && Array.isArray(process.flowElements)) {
    process.flowElements.forEach(visit);
  }
  return nodes;
}

/**
 * 收集 Process 内的所有 SequenceFlow。
 */
function collectSequenceFlows(process: ModdleElementLike): ModdleElementLike[] {
  const flows: ModdleElementLike[] = [];
  const visit = (el: ModdleElementLike) => {
    if (el.$type === 'bpmn:SequenceFlow') {
      flows.push(el);
      return;
    }
    if (el.flowElements && Array.isArray(el.flowElements)) {
      el.flowElements.forEach(visit);
    }
  };
  if (process.flowElements && Array.isArray(process.flowElements)) {
    process.flowElements.forEach(visit);
  }
  return flows;
}

/**
 * 使用 dagre 计算 FlowNode 的中心点坐标。
 * 返回 Map<elementId, { x, y }>（dagre 返回中心点）。
 */
function layoutNodes(nodes: ModdleElementLike[]): Map<string, { x: number; y: number }> {
  const positions = new Map<string, { x: number; y: number }>();
  if (nodes.length === 0) return positions;

  const graph = new dagre.graphlib.Graph();
  graph.setDefaultEdgeLabel(() => ({}));
  graph.setGraph({
    rankdir: 'TB',
    nodesep: dagreNodeSep,
    ranksep: dagreRankSep,
    marginx: dagreMargin,
    marginy: dagreMargin,
  });

  nodes.forEach(node => {
    const size = getNodeSize(node.$type);
    graph.setNode(node.id || node.name || `node-${positions.size}`, {
      width: size.width,
      height: size.height,
    });
  });

  // 用所有 sequenceFlow 的 source/target 连边
  nodes.forEach(node => {
    const out = (node as ModdleElementLike & { outgoing?: ModdleElementLike[] }).outgoing;
    if (out && Array.isArray(out)) {
      out.forEach(sf => {
        const target = (sf as ModdleElementLike & { targetRef?: ModdleElementLike }).targetRef;
        if (target?.id) {
          graph.setEdge(node.id!, target.id);
        }
      });
    }
  });

  dagre.layout(graph);

  nodes.forEach(node => {
    const id = node.id;
    if (!id) return;
    const pos = graph.node(id);
    if (pos && typeof pos.x === 'number' && typeof pos.y === 'number') {
      positions.set(id, { x: pos.x, y: pos.y });
    }
  });

  return positions;
}

/**
 * 为单个 Process 创建 BPMNDiagram + BPMNPlane，并附带所有 FlowNode 与 SequenceFlow 的 DI。
 */
function buildDiagramForProcess(
  moddle: ModdleLike,
  process: ModdleElementLike,
  diagramId: string
): ModdleElementLike | null {
  const flowNodes = collectFlowNodes(process);
  const sequenceFlows = collectSequenceFlows(process);
  if (flowNodes.length === 0) return null;

  const positions = layoutNodes(flowNodes);

  const diagram = moddle.create('bpmndi:BPMNDiagram', { id: diagramId }) as ModdleElementLike;
  const plane = moddle.create('bpmndi:BPMNPlane', {
    id: `${diagramId}_plane`,
    bpmnElement: process,
  }) as ModdleElementLike & { planeElement?: ModdleElementLike[] };
  diagram.set?.('plane', plane);

  // 收集 planeElement（既是 shape 也是 edge）。直接挂到 plane 上即可。
  const planeElements: ModdleElementLike[] = [];

  // FlowNode → BPMNShape
  flowNodes.forEach(node => {
    if (!node.id) return;
    const pos = positions.get(node.id);
    if (!pos) return;
    const size = getNodeSize(node.$type);

    const shape = moddle.create('bpmndi:BPMNShape', {
      id: `${node.id}_di`,
      bpmnElement: node,
    }) as ModdleElementLike & { bounds?: ModdleElementLike };

    const bounds = moddle.create('dc:Bounds', {
      x: pos.x - size.width / 2,
      y: pos.y - size.height / 2,
      width: size.width,
      height: size.height,
    }) as ModdleElementLike;

    shape.set?.('bounds', bounds);
    planeElements.push(shape);
  });

  // SequenceFlow → BPMNEdge
  const shapeByElementId = new Map<string, ModdleElementLike>();
  planeElements.forEach(el => {
    const be = (el as ModdleElementLike & { bpmnElement?: ModdleElementLike }).bpmnElement;
    if (be?.id) shapeByElementId.set(be.id, el);
  });

  sequenceFlows.forEach(flow => {
    if (!flow.id) return;
    const source = (flow as ModdleElementLike & { sourceRef?: ModdleElementLike }).sourceRef;
    const target = (flow as ModdleElementLike & { targetRef?: ModdleElementLike }).targetRef;
    if (!source?.id || !target?.id) return;

    const sourceShape = shapeByElementId.get(source.id);
    const targetShape = shapeByElementId.get(target.id);
    if (!sourceShape || !targetShape) return;

    const sourcePos = positions.get(source.id);
    const targetPos = positions.get(target.id);
    if (!sourcePos || !targetPos) return;

    const edge = moddle.create('bpmndi:BPMNEdge', {
      id: `${flow.id}_di`,
      bpmnElement: flow,
    }) as ModdleElementLike & { waypoint?: ModdleElementLike[] };

    const wp1 = moddle.create('di:waypoint', { x: sourcePos.x, y: sourcePos.y }) as ModdleElementLike;
    const wp2 = moddle.create('di:waypoint', { x: targetPos.x, y: targetPos.y }) as ModdleElementLike;
    edge.set?.('waypoint', [wp1, wp2]);

    edge.set?.('sourceElement', sourceShape);
    edge.set?.('targetElement', targetShape);

    planeElements.push(edge);
  });

  plane.set?.('planeElement', planeElements);
  return diagram;
}

/**
 * 给定的 Definitions 是否已经具备任何 BPMNDiagram。
 */
export function hasBpmnDiagram(definitions: DefinitionsLike | undefined | null): boolean {
  if (!definitions) return false;
  const diagrams = definitions.diagrams;
  return Array.isArray(diagrams) && diagrams.length > 0;
}

/**
 * 为缺失 BPMNDI 的 Definitions 注入自动布局的 BPMNDiagram。
 *
 * 仅处理根级 Process（最常见的 AI 生成结构）；忽略 Collaboration 中的子流程。
 * 若已存在 BPMNDiagram 则直接返回原对象，不做任何修改。
 *
 * @returns 修改后的 Definitions；任何异常均回退为原始对象（保持兼容性）。
 */
export function ensureBpmnDI(
  moddle: ModdleLike,
  definitions: DefinitionsLike
): DefinitionsLike {
  try {
    if (hasBpmnDiagram(definitions)) {
      return definitions;
    }
    const rootElements = definitions.rootElements || [];
    let diagramCounter = 0;
    const diagrams: ModdleElementLike[] = [];
    rootElements.forEach(root => {
      if (root?.$type === 'bpmn:Process') {
        diagramCounter += 1;
        const diagram = buildDiagramForProcess(
          moddle,
          root,
          `BPMNDiagram_auto_${diagramCounter}`
        );
        if (diagram) diagrams.push(diagram);
      }
    });
    if (diagrams.length > 0) {
      definitions.set?.('diagrams', diagrams);
    }
  } catch (err) {
    // 防御性捕获：任何 moddle / dagre 异常都不应破坏主流程，让上层 import 抛错更清晰。
    console.warn('[bpmnAutoLayout] failed to inject BPMNDI:', err);
  }
  return definitions;
}
