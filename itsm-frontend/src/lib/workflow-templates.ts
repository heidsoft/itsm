/**
 * 工作流模板定义
 * 提供常用企业流程模板。
 *
 * BPMN XML 由 buildTemplateXml 统一生成，保证每个模板都携带完整的
 * BPMNDiagram DI（节点 shape + 连线 edge）。此前部分模板缺少 DI，
 * bpmn-js 导入时报 "no diagram to display"；请假模板缺少 BPMNEdge，
 * 渲染后节点之间没有连线。
 */

export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  icon: string;
  bpmnXml: string;
  // 关联的工单类型code（如 k8s_scale, account_apply 等）
  ticketTypeCode?: string;
  approvalConfig: {
    requireApproval: boolean;
    approvalType: 'single' | 'parallel' | 'sequential';
    approvers: string[];
  };
}

type NodeKind =
  | 'startEvent'
  | 'endEvent'
  | 'userTask'
  | 'exclusiveGateway'
  | 'parallelGateway';

interface TplNode {
  id: string;
  kind: NodeKind;
  name: string;
  x: number;
  y: number;
}

interface TplFlow {
  id: string;
  from: string;
  to: string;
  name?: string;
  /** 条件表达式（如网关分支的 approved == true） */
  condition?: string;
  /** 连线拐点坐标，首尾分别贴源/目标节点 */
  waypoints: Array<[number, number]>;
}

const NODE_SIZE: Record<NodeKind, [number, number]> = {
  startEvent: [36, 36],
  endEvent: [36, 36],
  userTask: [100, 80],
  exclusiveGateway: [50, 50],
  parallelGateway: [50, 50],
};

function escapeXml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

/** 生成带完整 DI 的 BPMN 2.0 XML */
function buildTemplateXml(
  processId: string,
  processName: string,
  nodes: TplNode[],
  flows: TplFlow[]
): string {
  const incoming = new Map<string, string[]>();
  const outgoing = new Map<string, string[]>();
  flows.forEach(flow => {
    incoming.set(flow.to, [...(incoming.get(flow.to) ?? []), flow.id]);
    outgoing.set(flow.from, [...(outgoing.get(flow.from) ?? []), flow.id]);
  });

  const processElements = nodes
    .map(node => {
      const ins = incoming.get(node.id) ?? [];
      const outs = outgoing.get(node.id) ?? [];
      const children = [
        ...ins.map(id => `      <bpmn:incoming>${id}</bpmn:incoming>`),
        ...outs.map(id => `      <bpmn:outgoing>${id}</bpmn:outgoing>`),
      ].join('\n');
      return `    <bpmn:${node.kind} id="${node.id}" name="${escapeXml(node.name)}">${
        children ? `\n${children}\n    </bpmn:${node.kind}>` : '/>'
      }`;
    })
    .join('\n');

  const flowElements = flows
    .map(flow => {
      const attrs = `id="${flow.id}" sourceRef="${flow.from}" targetRef="${flow.to}"${
        flow.name ? ` name="${escapeXml(flow.name)}"` : ''
      }`;
      if (!flow.condition) {
        return `    <bpmn:sequenceFlow ${attrs} />`;
      }
      return `    <bpmn:sequenceFlow ${attrs}>
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${escapeXml(
        flow.condition
      )}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>`;
    })
    .join('\n');

  const shapes = nodes
    .map(node => {
      const [width, height] = NODE_SIZE[node.kind];
      const marker = node.kind === 'exclusiveGateway' ? ' isMarkerVisible="true"' : '';
      return `      <bpmndi:BPMNShape id="${node.id}_di" bpmnElement="${node.id}"${marker}>
        <dc:Bounds x="${node.x}" y="${node.y}" width="${width}" height="${height}" />
      </bpmndi:BPMNShape>`;
    })
    .join('\n');

  const edges = flows
    .map(
      flow => `      <bpmndi:BPMNEdge id="${flow.id}_di" bpmnElement="${flow.id}">
${flow.waypoints
  .map(([x, y]) => `        <di:waypoint x="${x}" y="${y}" />`)
  .join('\n')}
      </bpmndi:BPMNEdge>`
    )
    .join('\n');

  return `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" id="Definitions_${processId}" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="${processId}" name="${escapeXml(processName)}" isExecutable="true">
${processElements}
${flowElements}
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="${processId}">
${shapes}
${edges}
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`;
}

/**
 * 预设工作流模板
 */
export const WORKFLOW_TEMPLATES: WorkflowTemplate[] = [
  {
    id: 'generic_request',
    name: '通用申请审批流程',
    description: '各类申请类工单的通用审批流：提交申请 → 主管审批 → 通过/拒绝',
    category: 'ticket',
    icon: 'Ticket',
    bpmnXml: buildTemplateXml(
      'Process_GenericRequest',
      '通用申请审批流程',
      [
        { id: 'StartEvent_1', kind: 'startEvent', name: '开始', x: 180, y: 160 },
        { id: 'Task_Submit', kind: 'userTask', name: '提交申请', x: 270, y: 138 },
        { id: 'Task_ManagerApprove', kind: 'userTask', name: '主管审批', x: 420, y: 138 },
        { id: 'Gateway_Approval', kind: 'exclusiveGateway', name: '是否通过', x: 575, y: 155 },
        { id: 'EndEvent_Approved', kind: 'endEvent', name: '审批通过', x: 692, y: 100 },
        { id: 'EndEvent_Rejected', kind: 'endEvent', name: '审批拒绝', x: 692, y: 220 },
      ],
      [
        { id: 'Flow_1', from: 'StartEvent_1', to: 'Task_Submit', waypoints: [[216, 178], [270, 178]] },
        { id: 'Flow_2', from: 'Task_Submit', to: 'Task_ManagerApprove', waypoints: [[370, 178], [420, 178]] },
        { id: 'Flow_3', from: 'Task_ManagerApprove', to: 'Gateway_Approval', waypoints: [[520, 178], [575, 180]] },
        {
          id: 'Flow_Approved',
          from: 'Gateway_Approval',
          to: 'EndEvent_Approved',
          name: '通过',
          condition: '${approved == true}',
          waypoints: [[600, 155], [600, 118], [692, 118]],
        },
        {
          id: 'Flow_Rejected',
          from: 'Gateway_Approval',
          to: 'EndEvent_Rejected',
          name: '拒绝',
          condition: '${approved == false}',
          waypoints: [[600, 205], [600, 238], [692, 238]],
        },
      ]
    ),
    approvalConfig: {
      requireApproval: true,
      approvalType: 'single',
      approvers: [],
    },
  },
  {
    id: 'leave_request',
    name: '请假审批流程',
    description: '员工请假申请审批，支持年假、病假、事假等类型',
    category: 'hr',
    icon: 'Calendar',
    bpmnXml: buildTemplateXml(
      'Process_LeaveRequest',
      '请假审批流程',
      [
        { id: 'StartEvent_1', kind: 'startEvent', name: '开始', x: 180, y: 160 },
        { id: 'Task_Submit', kind: 'userTask', name: '提交请假申请', x: 270, y: 138 },
        { id: 'Task_ManagerApprove', kind: 'userTask', name: '部门经理审批', x: 420, y: 138 },
        { id: 'Gateway_Approval', kind: 'exclusiveGateway', name: '是否通过', x: 575, y: 155 },
        { id: 'EndEvent_Approved', kind: 'endEvent', name: '审批通过', x: 692, y: 100 },
        { id: 'EndEvent_Rejected', kind: 'endEvent', name: '审批拒绝', x: 692, y: 220 },
      ],
      [
        { id: 'Flow_1', from: 'StartEvent_1', to: 'Task_Submit', waypoints: [[216, 178], [270, 178]] },
        { id: 'Flow_2', from: 'Task_Submit', to: 'Task_ManagerApprove', waypoints: [[370, 178], [420, 178]] },
        { id: 'Flow_3', from: 'Task_ManagerApprove', to: 'Gateway_Approval', waypoints: [[520, 178], [575, 180]] },
        {
          id: 'Flow_Approved',
          from: 'Gateway_Approval',
          to: 'EndEvent_Approved',
          name: '通过',
          condition: '${approved == true}',
          waypoints: [[600, 155], [600, 118], [692, 118]],
        },
        {
          id: 'Flow_Rejected',
          from: 'Gateway_Approval',
          to: 'EndEvent_Rejected',
          name: '拒绝',
          condition: '${approved == false}',
          waypoints: [[600, 205], [600, 238], [692, 238]],
        },
      ]
    ),
    approvalConfig: {
      requireApproval: true,
      approvalType: 'sequential',
      approvers: [],
    },
  },
  {
    id: 'expense_approval',
    name: '费用报销流程',
    description: '员工费用报销审批，支持差旅、招待、采购等费用类型',
    category: 'finance',
    icon: 'DollarSign',
    bpmnXml: buildTemplateXml(
      'Process_ExpenseApproval',
      '费用报销流程',
      [
        { id: 'StartEvent_1', kind: 'startEvent', name: '开始', x: 150, y: 200 },
        { id: 'Task_Submit', kind: 'userTask', name: '提交报销单', x: 250, y: 178 },
        { id: 'Task_ManagerApprove', kind: 'userTask', name: '部门负责人审批', x: 420, y: 178 },
        { id: 'Task_FinanceApprove', kind: 'userTask', name: '财务审批', x: 590, y: 178 },
        { id: 'Gateway_Approval', kind: 'exclusiveGateway', name: '是否通过', x: 760, y: 195 },
        { id: 'EndEvent_Approved', kind: 'endEvent', name: '审批通过', x: 880, y: 140 },
        { id: 'EndEvent_Rejected', kind: 'endEvent', name: '审批拒绝', x: 880, y: 260 },
      ],
      [
        { id: 'Flow_1', from: 'StartEvent_1', to: 'Task_Submit', waypoints: [[186, 218], [250, 218]] },
        { id: 'Flow_2', from: 'Task_Submit', to: 'Task_ManagerApprove', waypoints: [[350, 218], [420, 218]] },
        { id: 'Flow_3', from: 'Task_ManagerApprove', to: 'Task_FinanceApprove', waypoints: [[520, 218], [590, 218]] },
        { id: 'Flow_4', from: 'Task_FinanceApprove', to: 'Gateway_Approval', waypoints: [[690, 218], [760, 220]] },
        {
          id: 'Flow_Approved',
          from: 'Gateway_Approval',
          to: 'EndEvent_Approved',
          name: '通过',
          condition: '${approved == true}',
          waypoints: [[785, 195], [785, 158], [880, 158]],
        },
        {
          id: 'Flow_Rejected',
          from: 'Gateway_Approval',
          to: 'EndEvent_Rejected',
          name: '拒绝',
          condition: '${approved == false}',
          waypoints: [[810, 220], [845, 278], [880, 278]],
        },
      ]
    ),
    approvalConfig: {
      requireApproval: true,
      approvalType: 'sequential',
      approvers: [],
    },
  },
  {
    id: 'procurement',
    name: '采购审批流程',
    description: '办公用品、设备采购审批流程，逐级经理审批',
    category: 'procurement',
    icon: 'ShoppingCart',
    bpmnXml: buildTemplateXml(
      'Process_Procurement',
      '采购审批流程',
      [
        { id: 'StartEvent_1', kind: 'startEvent', name: '开始', x: 150, y: 200 },
        { id: 'Task_Submit', kind: 'userTask', name: '提交采购申请', x: 250, y: 178 },
        { id: 'Task_ManagerApprove', kind: 'userTask', name: '部门经理审批', x: 420, y: 178 },
        { id: 'Task_DirectorApprove', kind: 'userTask', name: '总监审批', x: 590, y: 178 },
        { id: 'Task_CFOApprove', kind: 'userTask', name: 'CFO审批', x: 760, y: 178 },
        { id: 'EndEvent_Approved', kind: 'endEvent', name: '审批通过', x: 930, y: 200 },
      ],
      [
        { id: 'Flow_1', from: 'StartEvent_1', to: 'Task_Submit', waypoints: [[186, 218], [250, 218]] },
        { id: 'Flow_2', from: 'Task_Submit', to: 'Task_ManagerApprove', waypoints: [[350, 218], [420, 218]] },
        { id: 'Flow_3', from: 'Task_ManagerApprove', to: 'Task_DirectorApprove', waypoints: [[520, 218], [590, 218]] },
        { id: 'Flow_4', from: 'Task_DirectorApprove', to: 'Task_CFOApprove', waypoints: [[690, 218], [760, 218]] },
        { id: 'Flow_5', from: 'Task_CFOApprove', to: 'EndEvent_Approved', waypoints: [[860, 218], [930, 218]] },
      ]
    ),
    approvalConfig: {
      requireApproval: true,
      approvalType: 'sequential',
      approvers: [],
    },
  },
  {
    id: 'change_request',
    name: '变更管理流程',
    description: 'IT系统变更、配置变更审批流程，包含评估和测试环节',
    category: 'it',
    icon: 'GitBranch',
    bpmnXml: buildTemplateXml(
      'Process_ChangeRequest',
      '变更管理流程',
      [
        { id: 'StartEvent_1', kind: 'startEvent', name: '开始', x: 150, y: 200 },
        { id: 'Task_Submit', kind: 'userTask', name: '提交变更申请', x: 250, y: 178 },
        { id: 'Task_ImpactAnalysis', kind: 'userTask', name: '影响分析', x: 420, y: 178 },
        { id: 'Task_ITApproval', kind: 'userTask', name: 'IT负责人审批', x: 590, y: 178 },
        { id: 'Task_Implementation', kind: 'userTask', name: '实施变更', x: 760, y: 178 },
        { id: 'Task_Verification', kind: 'userTask', name: '验证确认', x: 930, y: 178 },
        { id: 'EndEvent_Completed', kind: 'endEvent', name: '变更完成', x: 1100, y: 200 },
      ],
      [
        { id: 'Flow_1', from: 'StartEvent_1', to: 'Task_Submit', waypoints: [[186, 218], [250, 218]] },
        { id: 'Flow_2', from: 'Task_Submit', to: 'Task_ImpactAnalysis', waypoints: [[350, 218], [420, 218]] },
        { id: 'Flow_3', from: 'Task_ImpactAnalysis', to: 'Task_ITApproval', waypoints: [[520, 218], [590, 218]] },
        { id: 'Flow_4', from: 'Task_ITApproval', to: 'Task_Implementation', waypoints: [[690, 218], [760, 218]] },
        { id: 'Flow_5', from: 'Task_Implementation', to: 'Task_Verification', waypoints: [[860, 218], [930, 218]] },
        { id: 'Flow_6', from: 'Task_Verification', to: 'EndEvent_Completed', waypoints: [[1030, 218], [1100, 218]] },
      ]
    ),
    approvalConfig: {
      requireApproval: true,
      approvalType: 'sequential',
      approvers: [],
    },
  },
  {
    id: 'contract_approval',
    name: '合同审批流程',
    description: '合同签订审批流程，包含法务审核和会签环节',
    category: 'legal',
    icon: 'FileText',
    bpmnXml: buildTemplateXml(
      'Process_ContractApproval',
      '合同审批流程',
      [
        { id: 'StartEvent_1', kind: 'startEvent', name: '开始', x: 150, y: 240 },
        { id: 'Task_Submit', kind: 'userTask', name: '提交合同', x: 250, y: 218 },
        { id: 'Task_LegalReview', kind: 'userTask', name: '法务会签', x: 420, y: 120 },
        { id: 'Task_FinanceReview', kind: 'userTask', name: '财务会签', x: 420, y: 300 },
        { id: 'Gateway_Parallel', kind: 'parallelGateway', name: '并行会签', x: 590, y: 233 },
        { id: 'Task_GMApproval', kind: 'userTask', name: '总经理审批', x: 700, y: 218 },
        { id: 'EndEvent_Approved', kind: 'endEvent', name: '审批完成', x: 860, y: 238 },
      ],
      [
        { id: 'Flow_1', from: 'StartEvent_1', to: 'Task_Submit', waypoints: [[186, 258], [250, 258]] },
        {
          id: 'Flow_2',
          from: 'Task_Submit',
          to: 'Task_LegalReview',
          waypoints: [[350, 258], [385, 258], [385, 160], [420, 160]],
        },
        {
          id: 'Flow_2b',
          from: 'Task_Submit',
          to: 'Task_FinanceReview',
          waypoints: [[350, 258], [385, 258], [385, 340], [420, 340]],
        },
        {
          id: 'Flow_3',
          from: 'Task_LegalReview',
          to: 'Gateway_Parallel',
          waypoints: [[520, 160], [555, 160], [555, 258], [590, 258]],
        },
        {
          id: 'Flow_4',
          from: 'Task_FinanceReview',
          to: 'Gateway_Parallel',
          waypoints: [[520, 340], [555, 340], [555, 258], [590, 258]],
        },
        { id: 'Flow_5', from: 'Gateway_Parallel', to: 'Task_GMApproval', waypoints: [[640, 258], [700, 258]] },
        { id: 'Flow_6', from: 'Task_GMApproval', to: 'EndEvent_Approved', waypoints: [[800, 258], [860, 258]] },
      ]
    ),
    approvalConfig: {
      requireApproval: true,
      approvalType: 'parallel',
      approvers: [],
    },
  },
  {
    id: 'empty',
    name: '空白流程',
    description: '从零开始创建自定义流程',
    category: 'custom',
    icon: 'Plus',
    bpmnXml: buildTemplateXml(
      'Process_Custom',
      '自定义流程',
      [
        { id: 'StartEvent_1', kind: 'startEvent', name: '开始', x: 180, y: 160 },
        { id: 'EndEvent_1', kind: 'endEvent', name: '结束', x: 350, y: 160 },
      ],
      []
    ),
    approvalConfig: {
      requireApproval: false,
      approvalType: 'single',
      approvers: [],
    },
  },
];

/**
 * 按类别分组模板
 */
export const TEMPLATE_CATEGORIES = [
  { key: 'hr', name: '人力资源', icon: 'Users' },
  { key: 'finance', name: '财务费用', icon: 'DollarSign' },
  { key: 'procurement', name: '采购管理', icon: 'ShoppingCart' },
  { key: 'it', name: 'IT管理', icon: 'Server' },
  { key: 'legal', name: '法务合同', icon: 'FileText' },
  { key: 'ticket', name: '工单流程', icon: 'Ticket' },
  { key: 'custom', name: '自定义', icon: 'Plus' },
];

/**
 * 工单类型码 → 工作流模板 id 映射。
 *
 * 背景：后端 ticket_types.code（k8s_scale / ddl_execute / account_apply 等）
 * 与通用审批模板并非一一对应——大多数工单类型共享同一「申请-审批」形态。
 * 这里用显式映射代替给每个模板硬塞 ticketTypeCode 属性：
 * 未列出的工单类型回退到 generic_request（通用申请审批流）。
 * 新增专用流程时，在此登记映射即可让 getTemplateByTicketType 生效。
 */
const TICKET_TYPE_TEMPLATE_MAP: Record<string, string> = {
  account_apply: 'generic_request',
  vm_apply: 'generic_request',
  db_account_apply: 'generic_request',
  app_apply: 'generic_request',
  project_apply: 'generic_request',
  domain_apply: 'generic_request',
  gitlab_repo_apply: 'generic_request',
  firewall_apply: 'generic_request',
  data_export: 'generic_request',
  ddl_execute: 'change_request',
  k8s_scale: 'change_request',
  general: 'generic_request',
};

/**
 * 根据工单类型获取对应的模板
 * 用于工单创建时关联工作流；未知类型回退到通用申请审批流
 */
export const getTemplateByTicketType = (ticketTypeCode: string): WorkflowTemplate | undefined => {
  const templateId = TICKET_TYPE_TEMPLATE_MAP[ticketTypeCode] ?? 'generic_request';
  return WORKFLOW_TEMPLATES.find(t => t.id === templateId);
};

/**
 * 获取所有工单相关的模板（被映射引用的模板）
 */
export const getTicketWorkflowTemplates = (): WorkflowTemplate[] => {
  const usedIds = new Set(Object.values(TICKET_TYPE_TEMPLATE_MAP));
  return WORKFLOW_TEMPLATES.filter(t => usedIds.has(t.id));
};

/**
 * 根据类别获取模板
 */
export const getTemplatesByCategory = (category: string): WorkflowTemplate[] => {
  if (category === 'all') {
    return WORKFLOW_TEMPLATES;
  }
  return WORKFLOW_TEMPLATES.filter(t => t.category === category);
};

/**
 * 根据ID获取模板
 */
export const getTemplateById = (id: string): WorkflowTemplate | undefined => {
  return WORKFLOW_TEMPLATES.find(t => t.id === id);
};
