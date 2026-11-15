import {
  ensureBpmnDI,
  hasBpmnDiagram,
  hasBpmnDiagramWithShapes,
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

  // 以下两组是 prod 事故的直接回归锁（2026-11-15，20 个流程定义实测）。
  describe('空壳 DI：存在 BPMNDiagram 但零 shape/edge', () => {
    // prod 有 3 个定义属于此类（ticket_assignment_flow / release_approval_flow /
    // incident_emergency_flow_v1.1），plane 里只有一句注释。旧判据 hasBpmnDiagram
    // 认为「已有 DI」直接跳过补全 → 画布空白且不报错。
    const emptyShellDefinitions = () =>
      makeDefinitions({
        rootElements: [
          {
            $type: 'bpmn:Process',
            id: 'Process_shell',
            flowElements: [
              createFlowNode('start', 'bpmn:StartEvent'),
              createFlowNode('end', 'bpmn:EndEvent'),
            ],
          },
        ],
        diagrams: [
          {
            $type: 'bpmndi:BPMNDiagram',
            id: 'BPMNDiagram_1',
            plane: {
              $type: 'bpmndi:BPMNPlane',
              id: 'BPMNPlane_1',
              planeElement: [],
            },
          },
        ],
      });

    it('hasBpmnDiagram 仍按「有没有 diagram」判定（保持既有契约）', () => {
      expect(hasBpmnDiagram(emptyShellDefinitions())).toBe(true);
    });

    it('hasBpmnDiagramWithShapes 必须识别出空壳 plane 并判为不可用', () => {
      expect(hasBpmnDiagramWithShapes(emptyShellDefinitions())).toBe(false);
      // 完全没有 DI 时同样不可用
      expect(hasBpmnDiagramWithShapes(makeDefinitions())).toBe(false);
      expect(hasBpmnDiagramWithShapes(null)).toBe(false);
      expect(hasBpmnDiagramWithShapes(undefined)).toBe(false);
    });

    it('plane 缺 planeElement 字段（而非空数组）时也判为不可用', () => {
      const definitions = makeDefinitions({
        diagrams: [{ $type: 'bpmndi:BPMNDiagram', id: 'd1', plane: { $type: 'bpmndi:BPMNPlane' } }],
      });
      expect(hasBpmnDiagramWithShapes(definitions)).toBe(false);
    });

    it('真的含 shape 时才判为可用，且不能被空壳逻辑误伤', () => {
      const definitions = makeDefinitions({
        diagrams: [
          {
            $type: 'bpmndi:BPMNDiagram',
            id: 'd1',
            plane: {
              $type: 'bpmndi:BPMNPlane',
              planeElement: [{ $type: 'bpmndi:BPMNShape', id: 's1' }],
            },
          },
        ],
      });
      expect(hasBpmnDiagramWithShapes(definitions)).toBe(true);
    });

    it('ensureBpmnDI 遇到空壳 plane 必须重建，而不是原样返回', () => {
      const moddle = createModdleStub();
      const definitions = emptyShellDefinitions();
      const before = definitions.diagrams as unknown as Array<Record<string, unknown>>;

      ensureBpmnDI(moddle, definitions);

      const after = definitions.diagrams as unknown as Array<Record<string, unknown>>;
      // 必须换成自动布局产出的 diagram（id 带 _auto_ 前缀）
      expect(after.length).toBe(1);
      expect(after[0].id).toBe('BPMNDiagram_auto_1');
      // 且不再是原来那个空壳对象
      expect(after[0]).not.toBe(before[0]);

      const plane = after[0].plane as Record<string, unknown>;
      const planeElements = (plane.planeElement as Array<Record<string, unknown>>) || [];
      expect(planeElements.length).toBeGreaterThan(0);
      expect(planeElements.every(el => el.$type === 'bpmndi:BPMNShape')).toBe(true);
    });
  });

  describe('waypoint 类型必须是 dc:Point', () => {
    // bpmn-moddle 的 di.json 把 di:Edge.waypoint 声明为 type=dc:Point。
    // 写 'di:waypoint' 会抛 "unknown type <di:waypoint>"，被 ensureBpmnDI 的 catch
    // 吞掉后整份 DI 一个元素都补不出来——这是"所有缺 DI 定义预览空白"的根因。
    it('创建 waypoint 时用的是 dc:Point（moddle 已注册的类型）', () => {
      const created: string[] = [];
      const moddle: ModdleLike = {
        create(type: string, props: Record<string, unknown> = {}) {
          created.push(type);
          const obj: Record<string, unknown> = { $type: type, ...props };
          obj.set = (name: string, value: unknown) => {
            obj[name] = value;
          };
          return obj;
        },
      };

      const startNode = createFlowNode('start', 'bpmn:StartEvent');
      const endNode = createFlowNode('end', 'bpmn:EndEvent');
      const nodes = new Map([
        ['start', startNode],
        ['end', endNode],
      ]);
      const flow = createSequenceFlow('flow1', 'start', 'end', nodes);
      const definitions = makeDefinitions({
        rootElements: [
          {
            $type: 'bpmn:Process',
            id: 'Process_1',
            flowElements: [startNode, endNode, flow],
          },
        ],
      });

      ensureBpmnDI(moddle, definitions);

      // 关键断言：绝不出现 di:waypoint，且必须有 dc:Point
      expect(created).not.toContain('di:waypoint');
      expect(created.filter(t => t === 'dc:Point')).toHaveLength(2);

      // 产出的 edge 上挂的两个 waypoint 都应是 dc:Point
      const diagrams = (definitions.diagrams ?? []) as unknown as Array<Record<string, unknown>>;
      const plane = diagrams[0].plane as Record<string, unknown>;
      const edges = ((plane.planeElement as Array<Record<string, unknown>>) || []).filter(
        el => el.$type === 'bpmndi:BPMNEdge'
      );
      expect(edges).toHaveLength(1);
      const waypoints = edges[0].waypoint as Array<Record<string, unknown>>;
      expect(waypoints).toHaveLength(2);
      waypoints.forEach(wp => expect(wp.$type).toBe('dc:Point'));
    });
  });
});
