import {
  ensureBpmnDI,
  hasBpmnDiagram,
  type DefinitionsLike,
  type ModdleLike,
} from '@/components/workflow/bpmnAutoLayout';

/**
 * 构造一个最小可用的 moddle stub，校验 bpmnAutoLayout 的边界行为。
 *
 * 这里不直接使用 bpmn-moddle，因为：
 *  - 该模块在 jest jsdom 环境加载有 canvas 副作用；
 *  - 我们只关心 ensureBpmnDI 在不同 Definitions 上的分支行为。
 */
function createModdleStub(): ModdleLike {
  return {
    create(type: string, props: Record<string, unknown> = {}) {
      const obj: Record<string, unknown> = { $type: type, ...props };
      obj.set = (name: string, value: unknown) => {
        obj[name] = value;
      };
      return obj;
    },
  };
}

function createFlowNode(id: string, type: string) {
  return {
    $type: type,
    id,
    name: id,
    outgoing: [] as unknown[],
    incoming: [] as unknown[],
  };
}

function createSequenceFlow(id: string, sourceId: string, targetId: string, nodes: Map<string, ReturnType<typeof createFlowNode>>) {
  const source = nodes.get(sourceId);
  const target = nodes.get(targetId);
  const flow = {
    $type: 'bpmn:SequenceFlow',
    id,
    sourceRef: source,
    targetRef: target,
  };
  if (source) source.outgoing.push(flow);
  if (target) target.incoming.push(flow);
  return flow;
}

/**
 * 构造一个测试用的 Definitions 桩：仅暴露 ensureBpmnDI 关心的接口。
 * 类型上通过 `unknown` 中转，避开 DefinitionsLike 强制要求的 get/$type。
 */
function makeDefinitions(overrides: Record<string, unknown> = {}): DefinitionsLike {
  const obj: Record<string, unknown> = {
    $type: 'bpmn:Definitions',
    rootElements: [],
    ...overrides,
  };
  obj.set = (name: string, value: unknown) => {
    obj[name] = value;
  };
  obj.get = <T = unknown>(prop: string): T => obj[prop] as T;
  return obj as unknown as DefinitionsLike;
}

describe('bpmnAutoLayout', () => {
  describe('hasBpmnDiagram', () => {
    it('returns false when definitions is null/undefined', () => {
      expect(hasBpmnDiagram(null)).toBe(false);
      expect(hasBpmnDiagram(undefined)).toBe(false);
    });

    it('returns false when diagrams is empty or missing', () => {
      expect(hasBpmnDiagram(makeDefinitions())).toBe(false);
      expect(hasBpmnDiagram(makeDefinitions({ diagrams: [] }))).toBe(false);
    });

    it('returns true when at least one diagram is present', () => {
      expect(hasBpmnDiagram(makeDefinitions({ diagrams: [{ id: 'd1' }] }))).toBe(true);
    });
  });

  describe('ensureBpmnDI', () => {
    it('returns the original definitions when BPMNDI already exists', () => {
      const moddle = createModdleStub();
      const existingDiagram = { id: 'existing' };
      const definitions = makeDefinitions({ diagrams: [existingDiagram] });
      ensureBpmnDI(moddle, definitions);
      const diagrams = (definitions.diagrams ?? []) as unknown[];
      expect(diagrams.length).toBe(1);
      // 已有图时不修改 set，避免意外清空用户已经手工维护的 DI。
      expect(diagrams[0]).toBe(existingDiagram);
    });

    it('injects BPMNDiagram + BPMNPlane + BPMNShape + BPMNEdge when missing', () => {
      const moddle = createModdleStub();
      const startNode = createFlowNode('start', 'bpmn:StartEvent');
      const taskNode = createFlowNode('task', 'bpmn:UserTask');
      const endNode = createFlowNode('end', 'bpmn:EndEvent');
      const nodes = new Map([
        ['start', startNode],
        ['task', taskNode],
        ['end', endNode],
      ]);
      const flow1 = createSequenceFlow('flow1', 'start', 'task', nodes);
      const flow2 = createSequenceFlow('flow2', 'task', 'end', nodes);

      const process = {
        $type: 'bpmn:Process',
        id: 'Process_1',
        flowElements: [startNode, taskNode, endNode, flow1, flow2],
      };
      const definitions = makeDefinitions({ rootElements: [process] });

      ensureBpmnDI(moddle, definitions);

      const diagrams = (definitions.diagrams ?? []) as unknown as Array<Record<string, unknown>>;
      expect(diagrams.length).toBe(1);

      const diagram = diagrams[0] as unknown as Record<string, unknown>;
      expect(diagram.$type).toBe('bpmndi:BPMNDiagram');
      expect(diagram.id).toBe('BPMNDiagram_auto_1');

      const plane = diagram.plane as Record<string, unknown>;
      expect(plane.$type).toBe('bpmndi:BPMNPlane');
      expect(plane.bpmnElement).toBe(process);

      const planeElements = (plane.planeElement as Array<Record<string, unknown>>) || [];
      // 3 个 FlowNode + 2 个 SequenceFlow = 5 个 DI 元素
      expect(planeElements.length).toBe(5);

      const shapes = planeElements.filter(el => el.$type === 'bpmndi:BPMNShape');
      const edges = planeElements.filter(el => el.$type === 'bpmndi:BPMNEdge');
      expect(shapes.length).toBe(3);
      expect(edges.length).toBe(2);

      // 每个 Shape 必须有 bounds + bpmnElement
      shapes.forEach(shape => {
        const s = shape as Record<string, unknown>;
        expect(s.bpmnElement).toBeDefined();
        const bounds = s.bounds as Record<string, unknown>;
        expect(bounds).toBeDefined();
        expect(typeof bounds.x).toBe('number');
        expect(typeof bounds.y).toBe('number');
        expect(typeof bounds.width).toBe('number');
        expect(typeof bounds.height).toBe('number');
      });

      // 每个 Edge 必须有 sourceElement/targetElement/waypoint
      edges.forEach(edge => {
        const e = edge as Record<string, unknown>;
        expect(e.sourceElement).toBeDefined();
        expect(e.targetElement).toBeDefined();
        const waypoints = e.waypoint as Array<Record<string, unknown>>;
        expect(Array.isArray(waypoints)).toBe(true);
        expect(waypoints.length).toBeGreaterThanOrEqual(2);
      });
    });

    it('skips Process elements without any FlowNode', () => {
      const moddle = createModdleStub();
      const emptyProcess = { $type: 'bpmn:Process', id: 'Empty', flowElements: [] };
      const definitions = makeDefinitions({ rootElements: [emptyProcess] });

      ensureBpmnDI(moddle, definitions);
      // 空 Process 不产出 DI，避免画出一个空画布让用户困惑
      const diagrams = (definitions.diagrams ?? []) as unknown[];
      expect(diagrams.length).toBe(0);
    });

    it('handles non-Process root elements without crashing', () => {
      const moddle = createModdleStub();
      const definitions = makeDefinitions({ rootElements: [{ $type: 'bpmn:Message', id: 'msg1' }] });
      expect(() => ensureBpmnDI(moddle, definitions)).not.toThrow();
      const diagrams = (definitions.diagrams ?? []) as unknown[];
      expect(diagrams.length).toBe(0);
    });

    it('does not throw on malformed definitions (defensive)', () => {
      const moddle = createModdleStub();
      const definitions = {
        $type: 'bpmn:Definitions',
        rootElements: null,
        set: () => undefined,
        get: () => undefined,
      } as unknown as DefinitionsLike;
      expect(() => ensureBpmnDI(moddle, definitions)).not.toThrow();
    });
  });
});
