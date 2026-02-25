/**
 * 工作流模板定义
 * 提供常用企业流程模板
 */

export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  icon: string;
  bpmn_xml: string;
  approval_config: {
    require_approval: boolean;
    approval_type: 'single' | 'parallel' | 'sequential';
    approvers: string[];
  };
}

/**
 * 预设工作流模板
 */
export const WORKFLOW_TEMPLATES: WorkflowTemplate[] = [
  {
    id: 'leave_request',
    name: '请假审批流程',
    description: '员工请假申请审批，支持年假、病假、事假等类型',
    category: 'hr',
    icon: 'Calendar',
    bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_LeaveRequest" name="请假审批流程" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="开始">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="Task_Submit" name="提交请假申请">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Task_ManagerApprove" name="部门经理审批">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:exclusiveGateway id="Gateway_Approval" name="是否通过">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_Approved</bpmn:outgoing>
      <bpmn:outgoing>Flow_Rejected</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:endEvent id="EndEvent_Approved" name="审批通过">
      <bpmn:incoming>Flow_Approved</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:endEvent id="EndEvent_Rejected" name="审批拒绝">
      <bpmn:incoming>Flow_Rejected</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_Submit" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_Submit" targetRef="Task_ManagerApprove" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Task_ManagerApprove" targetRef="Gateway_Approval" />
    <bpmn:sequenceFlow id="Flow_Approved" name="通过" sourceRef="Gateway_Approval" targetRef="EndEvent_Approved">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">approved == true</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_Rejected" name="拒绝" sourceRef="Gateway_Approval" targetRef="EndEvent_Rejected">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">approved == false</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_LeaveRequest">
      <bpmndi:BPMNShape id="StartEvent_1_di" bpmnElement="StartEvent_1">
        <dc:Bounds x="180" y="160" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Task_Submit_di" bpmnElement="Task_Submit">
        <dc:Bounds x="270" y="138" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Task_ManagerApprove_di" bpmnElement="Task_ManagerApprove">
        <dc:Bounds x="420" y="138" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Gateway_Approval_di" bpmnElement="Gateway_Approval" isMarkerVisible="true">
        <dc:Bounds x="575" y="155" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_Approved_di" bpmnElement="EndEvent_Approved">
        <dc:Bounds x="692" y="100" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_Rejected_di" bpmnElement="EndEvent_Rejected">
        <dc:Bounds x="692" y="220" width="36" height="36" />
      </bpmndi:BPMNShape>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`,
    approval_config: {
      require_approval: true,
      approval_type: 'sequential',
      approvers: [],
    },
  },
  {
    id: 'expense_approval',
    name: '费用报销流程',
    description: '员工费用报销审批，支持差旅、招待、采购等费用类型',
    category: 'finance',
    icon: 'DollarSign',
    bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ExpenseApproval" name="费用报销流程" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="开始">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="Task_Submit" name="提交报销单">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Task_ManagerApprove" name="部门负责人审批">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Task_FinanceApprove" name="财务审批">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:exclusiveGateway id="Gateway_Approval" name="是否通过">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_Approved</bpmn:outgoing>
      <bpmn:outgoing>Flow_Rejected</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:endEvent id="EndEvent_Approved" name="审批通过">
      <bpmn:incoming>Flow_Approved</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:endEvent id="EndEvent_Rejected" name="审批拒绝">
      <bpmn:incoming>Flow_Rejected</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_Submit" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_Submit" targetRef="Task_ManagerApprove" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Task_ManagerApprove" targetRef="Task_FinanceApprove" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="Task_FinanceApprove" targetRef="Gateway_Approval" />
    <bpmn:sequenceFlow id="Flow_Approved" name="通过" sourceRef="Gateway_Approval" targetRef="EndEvent_Approved">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">approved == true</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_Rejected" name="拒绝" sourceRef="Gateway_Approval" targetRef="EndEvent_Rejected">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">approved == false</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
  </bpmn:process>
</bpmn:definitions>`,
    approval_config: {
      require_approval: true,
      approval_type: 'sequential',
      approvers: [],
    },
  },
  {
    id: 'procurement',
    name: '采购审批流程',
    description: '办公用品、设备采购审批流程，支持金额阈值自动路由',
    category: 'procurement',
    icon: 'ShoppingCart',
    bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Procurement" name="采购审批流程" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="开始"/>
    <bpmn:userTask id="Task_Submit" name="提交采购申请"/>
    <bpmn:userTask id="Task_ManagerApprove" name="部门经理审批"/>
    <bpmn:userTask id="Task_DirectorApprove" name="总监审批"/>
    <bpmn:userTask id="Task_CFOApprove" name="CFO审批"/>
    <bpmn:endEvent id="EndEvent_Approved" name="审批通过"/>
  </bpmn:process>
</bpmn:definitions>`,
    approval_config: {
      require_approval: true,
      approval_type: 'sequential',
      approvers: [],
    },
  },
  {
    id: 'change_request',
    name: '变更管理流程',
    description: 'IT系统变更、配置变更审批流程，包含评估和测试环节',
    category: 'it',
    icon: 'GitBranch',
    bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ChangeRequest" name="变更管理流程" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="开始">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="Task_Submit" name="提交变更申请">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Task_ImpactAnalysis" name="影响分析">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Task_ITApproval" name="IT负责人审批">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Task_Implementation" name="实施变更">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Task_Verification" name="验证确认">
      <bpmn:incoming>Flow_5</bpmn:incoming>
      <bpmn:outgoing>Flow_6</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_Completed" name="变更完成">
      <bpmn:incoming>Flow_6</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_Submit" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_Submit" targetRef="Task_ImpactAnalysis" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Task_ImpactAnalysis" targetRef="Task_ITApproval" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="Task_ITApproval" targetRef="Task_Implementation" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="Task_Implementation" targetRef="Task_Verification" />
    <bpmn:sequenceFlow id="Flow_6" sourceRef="Task_Verification" targetRef="EndEvent_Completed" />
  </bpmn:process>
</bpmn:definitions>`,
    approval_config: {
      require_approval: true,
      approval_type: 'sequential',
      approvers: [],
    },
  },
  {
    id: 'contract_approval',
    name: '合同审批流程',
    description: '合同签订审批流程，包含法务审核和会签环节',
    category: 'legal',
    icon: 'FileText',
    bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ContractApproval" name="合同审批流程" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="开始">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="Task_Submit" name="提交合同">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Task_LegalReview" name="法务会签">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Task_FinanceReview" name="财务会签">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:parallelGateway id="Gateway_Parallel" name="并行会签">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:parallelGateway>
    <bpmn:userTask id="Task_GMApproval" name="总经理审批">
      <bpmn:incoming>Flow_5</bpmn:incoming>
      <bpmn:outgoing>Flow_6</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_Approved" name="审批完成">
      <bpmn:incoming>Flow_6</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_Submit" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_Submit" targetRef="Task_LegalReview" />
    <bpmn:sequenceFlow id="Flow_2b" sourceRef="Task_Submit" targetRef="Task_FinanceReview" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Task_LegalReview" targetRef="Gateway_Parallel" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="Task_FinanceReview" targetRef="Gateway_Parallel" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="Gateway_Parallel" targetRef="Task_GMApproval" />
    <bpmn:sequenceFlow id="Flow_6" sourceRef="Task_GMApproval" targetRef="EndEvent_Approved" />
  </bpmn:process>
</bpmn:definitions>`,
    approval_config: {
      require_approval: true,
      approval_type: 'parallel',
      approvers: [],
    },
  },
  {
    id: 'empty',
    name: '空白流程',
    description: '从零开始创建自定义流程',
    category: 'custom',
    icon: 'Plus',
    bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Custom" name="自定义流程" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="开始"/>
    <bpmn:endEvent id="EndEvent_1" name="结束"/>
  </bpmn:process>
</bpmn:definitions>`,
    approval_config: {
      require_approval: false,
      approval_type: 'single',
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
  { key: 'custom', name: '自定义', icon: 'Plus' },
];

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
