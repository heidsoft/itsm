"use client";

import React, { useEffect, useRef, useState, useCallback } from "react";
import {
  Layout,
  Card,
  Button,
  Space,
  message,
  Upload,
  Modal,
  Form,
  Input,
  Select,
  Divider,
  Row,
  Col,
  Tabs,
  Tag,
} from "antd";
import {
  SaveOutlined,
  PlayCircleOutlined,
  UploadOutlined,
  DownloadOutlined,
  UndoOutlined,
  RedoOutlined,
  ZoomInOutlined,
  ZoomOutOutlined,
  FullscreenOutlined,
  SettingOutlined,
  CodeOutlined,
  FileTextOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import dynamic from "next/dynamic";
import BPMNPropertiesPanel from "./BPMNPropertiesPanel";

const { Sider, Content } = Layout;
const { Option } = Select;
const { TextArea } = Input;
const { TabPane } = Tabs;

// 动态导入BPMN.js以避免SSR问题
const BPMNModeler = dynamic(
  () => import("bpmn-js").then((mod) => ({ default: mod.BPMNModeler })),
  {
    ssr: false,
    loading: () => <div>Loading BPMN Designer...</div>,
  }
);

interface EnhancedBPMNDesignerProps {
  initialXML?: string;
  onSave?: (xml: string) => void;
  onDeploy?: (xml: string) => void;
  readOnly?: boolean;
  height?: number;
  showPropertiesPanel?: boolean;
}

interface WorkflowMetadata {
  name: string;
  description: string;
  version: string;
  category: string;
  tags: string[];
  author: string;
  createdDate: string;
  lastModified: string;
}

interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  xml: string;
  thumbnail?: string;
}

const EnhancedBPMNDesigner: React.FC<EnhancedBPMNDesignerProps> = ({
  initialXML = "",
  onSave,
  onDeploy,

  height = 600,
  showPropertiesPanel = true,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const modelerRef = useRef<any>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [selectedElement, setSelectedElement] = useState<any>(null);
  const [metadata, setMetadata] = useState<WorkflowMetadata>({
    name: "新工作流",
    description: "",
    version: "1.0.0",
    category: "general",
    tags: [],
    author: "",
    createdDate: new Date().toISOString(),
    lastModified: new Date().toISOString(),
  });
  const [showMetadataModal, setShowMetadataModal] = useState(false);
  const [showTemplatesModal, setShowTemplatesModal] = useState(false);
  const [canUndo, setCanUndo] = useState(false);
  const [canRedo, setCanRedo] = useState(false);
  const [activeTab, setActiveTab] = useState("designer");
  const [xmlPreview, setXmlPreview] = useState("");

  // 预定义的工作流模板
  const workflowTemplates: WorkflowTemplate[] = [
    {
      id: "simple-approval",
      name: "简单审批流程",
      description: "包含开始、审批、结束三个节点的简单审批流程",
      category: "approval",
      xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_1" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="开始">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="UserTask_1" name="审批">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_1" name="结束">
      <bpmn:incoming>Flow_2</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="UserTask_1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="UserTask_1" targetRef="EndEvent_1" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_1">
      <bpmndi:BPMNShape id="StartEvent_1_di" bpmnElement="StartEvent_1">
        <dc:Bounds x="152" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="UserTask_1_di" bpmnElement="UserTask_1">
        <dc:Bounds x="240" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_1_di" bpmnElement="EndEvent_1">
        <dc:Bounds x="392" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_1_di" bpmnElement="Flow_1">
        <di:waypoint x="188" y="120" />
        <di:waypoint x="240" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_2_di" bpmnElement="Flow_2">
        <di:waypoint x="340" y="120" />
        <di:waypoint x="392" y="120" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`,
    },
    {
      id: "it-incident-process",
      name: "IT事件处理流程",
      description: "完整的IT事件处理工作流，包含分类、分配、处理、验证等步骤",
      category: "it_service",
      xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_1" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ITIncidentProcess" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="事件报告">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:userTask id="ClassifyTask" name="事件分类">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:exclusiveGateway id="PriorityGateway" name="优先级判断">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    
    <bpmn:userTask id="HighPriorityTask" name="高优先级处理">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="NormalPriorityTask" name="标准处理">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_6</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:parallelGateway id="MergeGateway" name="合并网关">
      <bpmn:incoming>Flow_5</bpmn:incoming>
      <bpmn:incoming>Flow_6</bpmn:incoming>
      <bpmn:outgoing>Flow_7</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <bpmn:userTask id="VerificationTask" name="解决方案验证">
      <bpmn:incoming>Flow_7</bpmn:incoming>
      <bpmn:outgoing>Flow_8</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:exclusiveGateway id="ResolutionGateway" name="解决确认">
      <bpmn:incoming>Flow_8</bpmn:incoming>
      <bpmn:outgoing>Flow_9</bpmn:outgoing>
      <bpmn:outgoing>Flow_10</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    
    <bpmn:userTask id="ReopenTask" name="重新打开">
      <bpmn:incoming>Flow_10</bpmn:incoming>
      <bpmn:outgoing>Flow_11</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:endEvent id="EndEvent_1" name="事件关闭">
      <bpmn:incoming>Flow_9</bpmn:incoming>
    </bpmn:endEvent>
    
    <bpmn:endEvent id="EndEvent_2" name="重新处理">
      <bpmn:incoming>Flow_11</bpmn:incoming>
    </bpmn:endEvent>
    
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ClassifyTask" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ClassifyTask" targetRef="PriorityGateway" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="PriorityGateway" targetRef="HighPriorityTask" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="PriorityGateway" targetRef="NormalPriorityTask" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="HighPriorityTask" targetRef="MergeGateway" />
    <bpmn:sequenceFlow id="Flow_6" sourceRef="NormalPriorityTask" targetRef="MergeGateway" />
    <bpmn:sequenceFlow id="Flow_7" sourceRef="MergeGateway" targetRef="VerificationTask" />
    <bpmn:sequenceFlow id="Flow_8" sourceRef="VerificationTask" targetRef="ResolutionGateway" />
    <bpmn:sequenceFlow id="Flow_9" sourceRef="ResolutionGateway" targetRef="EndEvent_1" />
    <bpmn:sequenceFlow id="Flow_10" sourceRef="ResolutionGateway" targetRef="ReopenTask" />
    <bpmn:sequenceFlow id="Flow_11" sourceRef="ReopenTask" targetRef="EndEvent_2" />
  </bpmn:process>
  
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="ITIncidentProcess">
      <bpmndi:BPMNShape id="StartEvent_1_di" bpmnElement="StartEvent_1">
        <dc:Bounds x="152" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="ClassifyTask_di" bpmnElement="ClassifyTask">
        <dc:Bounds x="240" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="PriorityGateway_di" bpmnElement="PriorityGateway">
        <dc:Bounds x="385" y="95" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="HighPriorityTask_di" bpmnElement="HighPriorityTask">
        <dc:Bounds x="480" y="40" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="NormalPriorityTask_di" bpmnElement="NormalPriorityTask">
        <dc:Bounds x="480" y="160" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="MergeGateway_di" bpmnElement="MergeGateway">
        <dc:Bounds x="625" y="95" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="VerificationTask_di" bpmnElement="VerificationTask">
        <dc:Bounds x="720" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="ResolutionGateway_di" bpmnElement="ResolutionGateway">
        <dc:Bounds x="865" y="95" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="ReopenTask_di" bpmnElement="ReopenTask">
        <dc:Bounds x="960" y="160" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_1_di" bpmnElement="EndEvent_1">
        <dc:Bounds x="960" y="40" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_2_di" bpmnElement="EndEvent_2">
        <dc:Bounds x="1100" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
      
      <bpmndi:BPMNEdge id="Flow_1_di" bpmnElement="Flow_1">
        <di:waypoint x="188" y="120" />
        <di:waypoint x="240" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_2_di" bpmnElement="Flow_2">
        <di:waypoint x="340" y="120" />
        <di:waypoint x="385" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_3_di" bpmnElement="Flow_3">
        <di:waypoint x="410" y="95" />
        <di:waypoint x="480" y="80" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_4_di" bpmnElement="Flow_4">
        <di:waypoint x="410" y="145" />
        <di:waypoint x="480" y="200" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_5_di" bpmnElement="Flow_5">
        <di:waypoint x="580" y="80" />
        <di:waypoint x="625" y="95" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_6_di" bpmnElement="Flow_6">
        <di:waypoint x="580" y="200" />
        <di:waypoint x="625" y="145" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_7_di" bpmnElement="Flow_7">
        <di:waypoint x="675" y="120" />
        <di:waypoint x="720" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_8_di" bpmnElement="Flow_8">
        <di:waypoint x="820" y="120" />
        <di:waypoint x="865" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_9_di" bpmnElement="Flow_9">
        <di:waypoint x="890" y="95" />
        <di:waypoint x="960" y="58" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_10_di" bpmnElement="Flow_10">
        <di:waypoint x="890" y="145" />
        <di:waypoint x="960" y="200" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_11_di" bpmnElement="Flow_11">
        <di:waypoint x="1060" y="200" />
        <di:waypoint x="1100" y="120" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`,
    },
    {
      id: "parallel-tasks",
      name: "并行任务流程",
      description: "包含并行网关的复杂任务流程",
      category: "parallel",
      xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_1" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="开始">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:parallelGateway id="Gateway_1" name="并行网关">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:parallelGateway>
    <bpmn:userTask id="UserTask_1" name="任务1">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_2" name="任务2">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:parallelGateway id="Gateway_2" name="合并网关">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:incoming>Flow_5</bpmn:incoming>
      <bpmn:outgoing>Flow_6</bpmn:outgoing>
    </bpmn:parallelGateway>
    <bpmn:endEvent id="EndEvent_1" name="结束">
      <bpmn:incoming>Flow_6</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Gateway_1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Gateway_1" targetRef="UserTask_1" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Gateway_1" targetRef="UserTask_2" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="UserTask_1" targetRef="Gateway_2" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="UserTask_2" targetRef="Gateway_2" />
    <bpmn:sequenceFlow id="Flow_6" sourceRef="Gateway_2" targetRef="EndEvent_1" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_1">
      <bpmndi:BPMNShape id="StartEvent_1_di" bpmnElement="StartEvent_1">
        <dc:Bounds x="152" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Gateway_1_di" bpmnElement="Gateway_1">
        <dc:Bounds x="225" y="95" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="UserTask_1_di" bpmnElement="UserTask_1">
        <dc:Bounds x="320" y="60" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="UserTask_2_di" bpmnElement="UserTask_2">
        <dc:Bounds x="320" y="160" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Gateway_2_di" bpmnElement="Gateway_2">
        <dc:Bounds x="465" y="95" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_1_di" bpmnElement="EndEvent_1">
        <dc:Bounds x="560" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_1_di" bpmnElement="Flow_1">
        <di:waypoint x="188" y="120" />
        <di:waypoint x="225" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_2_di" bpmnElement="Flow_2">
        <di:waypoint x="250" y="95" />
        <di:waypoint x="320" y="100" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_3_di" bpmnElement="Flow_3">
        <di:waypoint x="250" y="145" />
        <di:waypoint x="320" y="200" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_4_di" bpmnElement="Flow_4">
        <di:waypoint x="420" y="100" />
        <di:waypoint x="465" y="95" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_5_di" bpmnElement="Flow_5">
        <di:waypoint x="420" y="200" />
        <di:waypoint x="465" y="145" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_6_di" bpmnElement="Flow_6">
        <di:waypoint x="490" y="120" />
        <di:waypoint x="560" y="120" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`,
    },
    {
      id: "change-management",
      name: "变更管理流程",
      description:
        "IT变更管理标准流程，包含变更请求、评估、审批、实施、验证等步骤",
      category: "change_management",
      xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_1" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ChangeManagementProcess" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="变更请求">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:userTask id="ChangeRequestTask" name="变更请求提交">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="ImpactAssessmentTask" name="影响评估">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:exclusiveGateway id="RiskAssessmentGateway" name="风险评估">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    
    <bpmn:userTask id="LowRiskApprovalTask" name="低风险审批">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_6</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="HighRiskApprovalTask" name="高风险审批">
      <bpmn:incoming>Flow_5</bpmn:incoming>
      <bpmn:outgoing>Flow_7</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:parallelGateway id="ApprovalMergeGateway" name="审批合并">
      <bpmn:incoming>Flow_6</bpmn:incoming>
      <bpmn:incoming>Flow_7</bpmn:incoming>
      <bpmn:outgoing>Flow_8</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <bpmn:userTask id="ImplementationTask" name="变更实施">
      <bpmn:incoming>Flow_8</bpmn:incoming>
      <bpmn:outgoing>Flow_9</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="PostImplementationTask" name="实施后验证">
      <bpmn:incoming>Flow_9</bpmn:incoming>
      <bpmn:outgoing>Flow_10</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:exclusiveGateway id="SuccessGateway" name="成功确认">
      <bpmn:incoming>Flow_10</bpmn:incoming>
      <bpmn:outgoing>Flow_11</bpmn:outgoing>
      <bpmn:outgoing>Flow_12</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    
    <bpmn:userTask id="RollbackTask" name="回滚变更">
      <bpmn:incoming>Flow_12</bpmn:incoming>
      <bpmn:outgoing>Flow_13</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:endEvent id="EndEvent_1" name="变更完成">
      <bpmn:incoming>Flow_11</bpmn:incoming>
    </bpmn:endEvent>
    
    <bpmn:endEvent id="EndEvent_2" name="变更失败">
      <bpmn:incoming>Flow_13</bpmn:incoming>
    </bpmn:endEvent>
    
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ChangeRequestTask" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ChangeRequestTask" targetRef="ImpactAssessmentTask" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="ImpactAssessmentTask" targetRef="RiskAssessmentGateway" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="RiskAssessmentGateway" targetRef="LowRiskApprovalTask" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="RiskAssessmentGateway" targetRef="HighRiskApprovalTask" />
    <bpmn:sequenceFlow id="Flow_6" sourceRef="LowRiskApprovalTask" targetRef="ApprovalMergeGateway" />
    <bpmn:sequenceFlow id="Flow_7" sourceRef="HighRiskApprovalTask" targetRef="ApprovalMergeGateway" />
    <bpmn:sequenceFlow id="Flow_8" sourceRef="ApprovalMergeGateway" targetRef="ImplementationTask" />
    <bpmn:sequenceFlow id="Flow_9" sourceRef="ImplementationTask" targetRef="PostImplementationTask" />
    <bpmn:sequenceFlow id="Flow_10" sourceRef="PostImplementationTask" targetRef="SuccessGateway" />
    <bpmn:sequenceFlow id="Flow_11" sourceRef="SuccessGateway" targetRef="EndEvent_1" />
    <bpmn:sequenceFlow id="Flow_12" sourceRef="SuccessGateway" targetRef="RollbackTask" />
    <bpmn:sequenceFlow id="Flow_13" sourceRef="RollbackTask" targetRef="EndEvent_2" />
  </bpmn:process>
  
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="ChangeManagementProcess">
      <bpmndi:BPMNShape id="StartEvent_1_di" bpmnElement="StartEvent_1">
        <dc:Bounds x="152" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="ChangeRequestTask_di" bpmnElement="ChangeRequestTask">
        <dc:Bounds x="240" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="ImpactAssessmentTask_di" bpmnElement="ImpactAssessmentTask">
        <dc:Bounds x="385" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="RiskAssessmentGateway_di" bpmnElement="RiskAssessmentGateway">
        <dc:Bounds x="530" y="95" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="LowRiskApprovalTask_di" bpmnElement="LowRiskApprovalTask">
        <dc:Bounds x="625" y="40" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="HighRiskApprovalTask_di" bpmnElement="HighRiskApprovalTask">
        <dc:Bounds x="625" y="160" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="ApprovalMergeGateway_di" bpmnElement="ApprovalMergeGateway">
        <dc:Bounds x="770" y="95" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="ImplementationTask_di" bpmnElement="ImplementationTask">
        <dc:Bounds x="865" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="PostImplementationTask_di" bpmnElement="PostImplementationTask">
        <dc:Bounds x="1010" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="SuccessGateway_di" bpmnElement="SuccessGateway">
        <dc:Bounds x="1155" y="95" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="RollbackTask_di" bpmnElement="RollbackTask">
        <dc:Bounds x="1250" y="160" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_1_di" bpmnElement="EndEvent_1">
        <dc:Bounds x="1250" y="40" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_2_di" bpmnElement="EndEvent_2">
        <dc:Bounds x="1390" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
      
      <bpmndi:BPMNEdge id="Flow_1_di" bpmnElement="Flow_1">
        <di:waypoint x="188" y="120" />
        <di:waypoint x="240" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_2_di" bpmnElement="Flow_2">
        <di:waypoint x="340" y="120" />
        <di:waypoint x="385" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_3_di" bpmnElement="Flow_3">
        <di:waypoint x="485" y="120" />
        <di:waypoint x="530" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_4_di" bpmnElement="Flow_4">
        <di:waypoint x="555" y="95" />
        <di:waypoint x="625" y="80" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_5_di" bpmnElement="Flow_5">
        <di:waypoint x="555" y="145" />
        <di:waypoint x="625" y="200" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_6_di" bpmnElement="Flow_6">
        <di:waypoint x="725" y="80" />
        <di:waypoint x="770" y="95" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_7_di" bpmnElement="Flow_7">
        <di:waypoint x="725" y="200" />
        <di:waypoint x="770" y="145" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_8_di" bpmnElement="Flow_8">
        <di:waypoint x="820" y="120" />
        <di:waypoint x="865" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_9_di" bpmnElement="Flow_9">
        <di:waypoint x="965" y="120" />
        <di:waypoint x="1010" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_10_di" bpmnElement="Flow_10">
        <di:waypoint x="1110" y="120" />
        <di:waypoint x="1155" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_11_di" bpmnElement="Flow_11">
        <di:waypoint x="1180" y="95" />
        <di:waypoint x="1250" y="58" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_12_di" bpmnElement="Flow_12">
        <di:waypoint x="1180" y="145" />
        <di:waypoint x="1250" y="200" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_13_di" bpmnElement="Flow_13">
        <di:waypoint x="1350" y="200" />
        <di:waypoint x="1390" y="120" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`,
    },
    {
      id: "problem-management-process",
      name: "问题管理流程",
      description: "IT问题管理流程，包含问题识别、分析、解决、验证等步骤",
      category: "problem_management",
      xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_1" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ProblemManagementProcess" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="问题识别">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:userTask id="ProblemAnalysisTask" name="问题分析">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="RootCauseAnalysisTask" name="根本原因分析">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="SolutionDesignTask" name="解决方案设计">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="SolutionImplementationTask" name="解决方案实施">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="VerificationTask" name="解决方案验证">
      <bpmn:incoming>Flow_5</bpmn:incoming>
      <bpmn:outgoing>Flow_6</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:endEvent id="EndEvent_1" name="问题解决">
      <bpmn:incoming>Flow_6</bpmn:incoming>
    </bpmn:endEvent>
    
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ProblemAnalysisTask" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ProblemAnalysisTask" targetRef="RootCauseAnalysisTask" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="RootCauseAnalysisTask" targetRef="SolutionDesignTask" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="SolutionDesignTask" targetRef="SolutionImplementationTask" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="SolutionImplementationTask" targetRef="VerificationTask" />
    <bpmn:sequenceFlow id="Flow_6" sourceRef="VerificationTask" targetRef="EndEvent_1" />
  </bpmn:process>
</bpmn:definitions>`,
    },
    {
      id: "service-request-process",
      name: "服务请求流程",
      description: "IT服务请求处理流程，包含请求接收、分类、处理、交付等步骤",
      category: "service_request",
      xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_1" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ServiceRequestProcess" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="服务请求">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:userTask id="RequestValidationTask" name="请求验证">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:exclusiveGateway id="RequestTypeGateway" name="请求类型判断">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    
    <bpmn:userTask id="StandardServiceTask" name="标准服务处理">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="CustomServiceTask" name="定制服务处理">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_6</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:parallelGateway id="MergeGateway" name="合并网关">
      <bpmn:incoming>Flow_5</bpmn:incoming>
      <bpmn:incoming>Flow_6</bpmn:incoming>
      <bpmn:outgoing>Flow_7</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <bpmn:userTask id="ServiceDeliveryTask" name="服务交付">
      <bpmn:incoming>Flow_7</bpmn:incoming>
      <bpmn:outgoing>Flow_8</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="CustomerSatisfactionTask" name="客户满意度调查">
      <bpmn:incoming>Flow_8</bpmn:incoming>
      <bpmn:outgoing>Flow_9</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:endEvent id="EndEvent_1" name="服务完成">
      <bpmn:incoming>Flow_9</bpmn:incoming>
    </bpmn:endEvent>
    
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="RequestValidationTask" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="RequestValidationTask" targetRef="RequestTypeGateway" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="RequestTypeGateway" targetRef="StandardServiceTask" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="RequestTypeGateway" targetRef="CustomServiceTask" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="StandardServiceTask" targetRef="MergeGateway" />
    <bpmn:sequenceFlow id="Flow_6" sourceRef="CustomServiceTask" targetRef="MergeGateway" />
    <bpmn:sequenceFlow id="Flow_7" sourceRef="MergeGateway" targetRef="ServiceDeliveryTask" />
    <bpmn:sequenceFlow id="Flow_8" sourceRef="ServiceDeliveryTask" targetRef="CustomerSatisfactionTask" />
    <bpmn:sequenceFlow id="Flow_9" sourceRef="CustomerSatisfactionTask" targetRef="EndEvent_1" />
  </bpmn:process>
</bpmn:definitions>`,
    },
    {
      id: "asset-lifecycle-process",
      name: "资产生命周期管理",
      description: "IT资产从采购到报废的完整生命周期管理流程",
      category: "asset_management",
      xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_1" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="AssetLifecycleProcess" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="资产需求">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    
    <bpmn:userTask id="ProcurementTask" name="采购申请">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="ApprovalTask" name="采购审批">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="PurchaseTask" name="资产采购">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="AssetRegistrationTask" name="资产登记">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="AssetDeploymentTask" name="资产部署">
      <bpmn:incoming>Flow_5</bpmn:incoming>
      <bpmn:outgoing>Flow_6</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="MaintenanceTask" name="定期维护">
      <bpmn:incoming>Flow_6</bpmn:incoming>
      <bpmn:outgoing>Flow_7</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="RetirementTask" name="资产报废">
      <bpmn:incoming>Flow_7</bpmn:incoming>
      <bpmn:outgoing>Flow_8</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:endEvent id="EndEvent_1" name="生命周期结束">
      <bpmn:incoming>Flow_8</bpmn:incoming>
    </bpmn:endEvent>
    
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ProcurementTask" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ProcurementTask" targetRef="ApprovalTask" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="ApprovalTask" targetRef="PurchaseTask" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="PurchaseTask" targetRef="AssetRegistrationTask" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="AssetRegistrationTask" targetRef="AssetDeploymentTask" />
    <bpmn:sequenceFlow id="Flow_6" sourceRef="AssetDeploymentTask" targetRef="MaintenanceTask" />
    <bpmn:sequenceFlow id="Flow_7" sourceRef="MaintenanceTask" targetRef="RetirementTask" />
    <bpmn:sequenceFlow id="Flow_8" sourceRef="RetirementTask" targetRef="EndEvent_1" />
  </bpmn:process>
</bpmn:definitions>`,
    },
  ];

  // 模板分类
  const templateCategories = [
    { value: "all", label: "全部模板", count: workflowTemplates.length },
    {
      value: "approval",
      label: "审批流程",
      count: workflowTemplates.filter((t) => t.category === "approval").length,
    },
    {
      value: "it_service",
      label: "IT服务",
      count: workflowTemplates.filter((t) => t.category === "it_service")
        .length,
    },
    {
      value: "change_management",
      label: "变更管理",
      count: workflowTemplates.filter((t) => t.category === "change_management")
        .length,
    },
    {
      value: "problem_management",
      label: "问题管理",
      count: workflowTemplates.filter(
        (t) => t.category === "problem_management"
      ).length,
    },
    {
      value: "service_request",
      label: "服务请求",
      count: workflowTemplates.filter((t) => t.category === "service_request")
        .length,
    },
    {
      value: "asset_management",
      label: "资产管理",
      count: workflowTemplates.filter((t) => t.category === "asset_management")
        .length,
    },
  ];

  const [selectedCategory, setSelectedCategory] = useState("all");
  const [searchKeyword, setSearchKeyword] = useState("");
  const [isMobile, setIsMobile] = useState(false);

  // 初始化BPMN设计器
  useEffect(() => {
    if (!containerRef.current || modelerRef.current) return;

    try {
      // 创建BPMN模型器
      const modeler = new BPMNModeler({
        container: containerRef.current,
        keyboard: {
          bindTo: window,
        },
        additionalModules: [
          // 可以在这里添加自定义模块
        ],
      });

      modelerRef.current = modeler;

      // 导入XML
      const xmlToImport = initialXML || workflowTemplates[0].xml;
      modeler
        .importXML(xmlToImport)
        .then(() => {
          // 自动调整视图
          const canvas = modeler.get("canvas");
          canvas.zoom("fit-viewport");

          // 设置事件监听器
          const eventBus = modeler.get("eventBus");
          const commandStack = modeler.get("commandStack");

          // 监听命令栈变化
          eventBus.on("commandStack.changed", () => {
            setCanUndo(commandStack.canUndo());
            setCanRedo(commandStack.canRedo());
          });

          // 监听元素选择
          eventBus.on("element.click", (event: any) => {
            setSelectedElement(event.element);
          });

          // 监听元素取消选择
          eventBus.on("element.out", () => {
            setSelectedElement(null);
          });

          // 监听画布点击
          eventBus.on("canvas.click", () => {
            setSelectedElement(null);
          });

          message.success("BPMN设计器初始化成功");
        })
        .catch((err: Error) => {
          console.error("导入XML失败:", err);
          message.error("导入XML失败: " + err.message);
        });

      // 清理函数
      return () => {
        if (modelerRef.current) {
          modelerRef.current.destroy();
          modelerRef.current = null;
        }
      };
    } catch (error) {
      console.error("初始化BPMN设计器失败:", error);
      message.error("初始化BPMN设计器失败");
    }
  }, [initialXML]);

  // 保存工作流
  const handleSave = useCallback(async () => {
    if (!modelerRef.current) {
      message.error("设计器未初始化");
      return;
    }

    try {
      const { xml } = await modelerRef.current.saveXML({ format: true });
      if (onSave) {
        onSave(xml);
      }
      message.success("工作流保存成功");
    } catch (error) {
      console.error("保存失败:", error);
      message.error("保存失败: " + error);
    }
  }, [onSave]);

  // 部署工作流
  const handleDeploy = useCallback(async () => {
    if (!modelerRef.current) {
      message.error("设计器未初始化");
      return;
    }

    try {
      const { xml } = await modelerRef.current.saveXML({ format: true });
      if (onDeploy) {
        onDeploy(xml);
      }
      message.success("工作流部署成功");
    } catch (error) {
      console.error("部署失败:", error);
      message.error("部署失败: " + error);
    }
  }, [onDeploy]);

  // 导入XML文件
  const handleImport = useCallback(async (file: File) => {
    if (!modelerRef.current) {
      message.error("设计器未初始化");
      return;
    }

    try {
      const text = await file.text();
      await modelerRef.current.importXML(text);
      message.success("XML文件导入成功");
    } catch (error) {
      console.error("导入失败:", error);
      message.error("导入失败: " + error);
    }
  }, []);

  // 导出XML文件
  const handleExport = useCallback(async () => {
    if (!modelerRef.current) {
      message.error("设计器未初始化");
      return;
    }

    try {
      const { xml } = await modelerRef.current.saveXML({ format: true });
      const blob = new Blob([xml], { type: "application/xml" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${metadata.name || "workflow"}.bpmn`;
      a.click();
      URL.revokeObjectURL(url);
      message.success("XML文件导出成功");
    } catch (error) {
      console.error("导出失败:", error);
      message.error("导出失败: " + error);
    }
  }, [metadata.name]);

  // 应用模板
  const applyTemplate = useCallback(async (template: WorkflowTemplate) => {
    if (!modelerRef.current) {
      message.error("设计器未初始化");
      return;
    }

    try {
      await modelerRef.current.importXML(template.xml);
      setMetadata((prev) => ({
        ...prev,
        name: template.name,
        description: template.description,
        category: template.category,
      }));
      setShowTemplatesModal(false);
      message.success(`模板"${template.name}"应用成功`);
    } catch (error) {
      console.error("应用模板失败:", error);
      message.error("应用模板失败: " + error);
    }
  }, []);

  // 预览XML
  const previewXML = useCallback(async () => {
    if (!modelerRef.current) {
      message.error("设计器未初始化");
      return;
    }

    try {
      const { xml } = await modelerRef.current.saveXML({ format: true });
      setXmlPreview(xml);
      setActiveTab("preview");
    } catch (error) {
      console.error("预览失败:", error);
      message.error("预览失败: " + error);
    }
  }, []);

  // 撤销操作
  const handleUndo = useCallback(() => {
    if (modelerRef.current && canUndo) {
      const commandStack = modelerRef.current.get("commandStack");
      commandStack.undo();
    }
  }, [canUndo]);

  // 重做操作
  const handleRedo = useCallback(() => {
    if (modelerRef.current && canRedo) {
      const commandStack = modelerRef.current.get("commandStack");
      commandStack.redo();
    }
  }, [canRedo]);

  // 缩放操作
  const handleZoom = useCallback((direction: "in" | "out" | "fit") => {
    if (!modelerRef.current) return;

    const canvas = modelerRef.current.get("canvas");
    const currentZoom = canvas.getZoom();

    switch (direction) {
      case "in":
        canvas.zoom(Math.min(currentZoom * 1.2, 3));
        break;
      case "out":
        canvas.zoom(Math.max(currentZoom / 1.2, 0.2));
        break;
      case "fit":
        canvas.zoom("fit-viewport");
        break;
    }
  }, []);

  // 切换全屏
  const toggleFullscreen = useCallback(() => {
    setIsFullscreen(!isFullscreen);
  }, [isFullscreen]);

  // 更新元数据
  const handleMetadataUpdate = useCallback((values: WorkflowMetadata) => {
    setMetadata(values);
    setShowMetadataModal(false);
    message.success("元数据更新成功");
  }, []);

  // 属性变更处理
  const handlePropertyChange = useCallback(
    (elementId: string, properties: Record<string, unknown>) => {
      console.log("Element property changed:", elementId, properties);
      // 这里可以添加额外的处理逻辑
    },
    []
  );

  return (
    <div
      className={`enhanced-bpmn-designer ${isFullscreen ? "fullscreen" : ""}`}
    >
      <Layout style={{ height: isFullscreen ? "100vh" : height }}>
        <Content>
          <Card
            title={
              <Space>
                <span>BPMN工作流设计器</span>
                {metadata.name && (
                  <span className="text-gray-500">- {metadata.name}</span>
                )}
              </Space>
            }
            extra={
              <Space>
                <Button
                  icon={<FileTextOutlined />}
                  onClick={() => setShowTemplatesModal(true)}
                  size="small"
                >
                  模板
                </Button>
                <Button
                  icon={<SettingOutlined />}
                  onClick={() => setShowMetadataModal(true)}
                  size="small"
                >
                  元数据
                </Button>
                <Button
                  icon={<UndoOutlined />}
                  disabled={!canUndo}
                  onClick={handleUndo}
                  size="small"
                >
                  撤销
                </Button>
                <Button
                  icon={<RedoOutlined />}
                  disabled={!canRedo}
                  onClick={handleRedo}
                  size="small"
                >
                  重做
                </Button>
                <Divider type="vertical" />
                <Button
                  icon={<ZoomOutOutlined />}
                  onClick={() => handleZoom("out")}
                  size="small"
                />
                <Button
                  icon={<ZoomInOutlined />}
                  onClick={() => handleZoom("in")}
                  size="small"
                />
                <Button
                  icon={<FullscreenOutlined />}
                  onClick={() => handleZoom("fit")}
                  size="small"
                >
                  适应
                </Button>
                <Button
                  icon={<FullscreenOutlined />}
                  onClick={toggleFullscreen}
                  size="small"
                >
                  {isFullscreen ? "退出全屏" : "全屏"}
                </Button>
              </Space>
            }
            bodyStyle={{ padding: 0 }}
          >
            {/* 工具栏 */}
            <div
              className="bpmn-toolbar"
              style={{ padding: "8px 16px", borderBottom: "1px solid #f0f0f0" }}
            >
              <Row gutter={16} align="middle">
                <Col>
                  <Space>
                    <Upload
                      accept=".bpmn,.xml"
                      showUploadList={false}
                      beforeUpload={(file) => {
                        handleImport(file);
                        return false;
                      }}
                    >
                      <Button icon={<UploadOutlined />} size="small">
                        导入XML
                      </Button>
                    </Upload>
                    <Button
                      icon={<DownloadOutlined />}
                      onClick={handleExport}
                      size="small"
                    >
                      导出XML
                    </Button>
                    <Button
                      icon={<CodeOutlined />}
                      onClick={previewXML}
                      size="small"
                    >
                      预览XML
                    </Button>
                  </Space>
                </Col>
                <Col>
                  <Space>
                    <Button
                      icon={<SaveOutlined />}
                      onClick={handleSave}
                      type="primary"
                      size="small"
                    >
                      保存
                    </Button>
                    <Button
                      icon={<PlayCircleOutlined />}
                      onClick={handleDeploy}
                      type="primary"
                      size="small"
                    >
                      部署
                    </Button>
                  </Space>
                </Col>
              </Row>
            </div>

            {/* 主内容区域 */}
            <Tabs
              activeKey={activeTab}
              onChange={setActiveTab}
              style={{ height: "calc(100% - 60px)" }}
            >
              <TabPane tab="设计器" key="designer">
                <div
                  ref={containerRef}
                  style={{
                    height: "calc(100vh - 200px)",
                    width: "100%",
                    position: "relative",
                  }}
                />
              </TabPane>
              <TabPane tab="XML预览" key="preview">
                <div
                  style={{
                    padding: "16px",
                    height: "calc(100vh - 200px)",
                    overflow: "auto",
                  }}
                >
                  <pre
                    style={{
                      background: "#f5f5f5",
                      padding: "16px",
                      borderRadius: "4px",
                    }}
                  >
                    {xmlPreview}
                  </pre>
                </div>
              </TabPane>
            </Tabs>
          </Card>
        </Content>

        {/* 右侧属性面板 */}
        {showPropertiesPanel && (
          <Sider
            width={350}
            theme="light"
            style={{ borderLeft: "1px solid #f0f0f0" }}
          >
            <BPMNPropertiesPanel
              selectedElement={selectedElement}
              modeler={modelerRef.current}
              onPropertyChange={handlePropertyChange}
            />
          </Sider>
        )}
      </Layout>

      {/* 元数据编辑模态框 */}
      <Modal
        title="工作流元数据"
        open={showMetadataModal}
        onCancel={() => setShowMetadataModal(false)}
        footer={null}
        width={600}
      >
        <Form
          layout="vertical"
          initialValues={metadata}
          onFinish={handleMetadataUpdate}
        >
          <Form.Item
            label="工作流名称"
            name="name"
            rules={[{ required: true, message: "请输入工作流名称" }]}
          >
            <Input placeholder="请输入工作流名称" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <TextArea rows={3} placeholder="请输入工作流描述" />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="版本"
                name="version"
                rules={[{ required: true, message: "请输入版本号" }]}
              >
                <Input placeholder="1.0.0" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="分类" name="category">
                <Select placeholder="选择分类">
                  <Option value="general">通用</Option>
                  <Option value="it">IT服务</Option>
                  <Option value="hr">人力资源</Option>
                  <Option value="finance">财务</Option>
                  <Option value="sales">销售</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Form.Item label="作者" name="author">
            <Input placeholder="请输入作者" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                保存
              </Button>
              <Button onClick={() => setShowMetadataModal(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 模板选择模态框 */}
      <Modal
        title="选择工作流模板"
        open={showTemplatesModal}
        onCancel={() => setShowTemplatesModal(false)}
        footer={null}
        width={900}
      >
        {/* 搜索和分类筛选 */}
        <div style={{ marginBottom: "16px" }}>
          <Row gutter={16}>
            <Col span={12}>
              <Input
                placeholder="搜索模板..."
                prefix={<SearchOutlined />}
                value={searchKeyword}
                onChange={(e) => setSearchKeyword(e.target.value)}
                allowClear
              />
            </Col>
            <Col span={12}>
              <Select
                placeholder="选择分类"
                value={selectedCategory}
                onChange={setSelectedCategory}
                style={{ width: "100%" }}
              >
                {templateCategories.map((category) => (
                  <Option key={category.value} value={category.value}>
                    {category.label} ({category.count})
                  </Option>
                ))}
              </Select>
            </Col>
          </Row>
        </div>

        {/* 模板列表 */}
        <Row gutter={[16, 16]}>
          {workflowTemplates
            .filter((template) => {
              const matchesCategory =
                selectedCategory === "all" ||
                template.category === selectedCategory;
              const matchesSearch =
                searchKeyword === "" ||
                template.name
                  .toLowerCase()
                  .includes(searchKeyword.toLowerCase()) ||
                template.description
                  .toLowerCase()
                  .includes(searchKeyword.toLowerCase());
              return matchesCategory && matchesSearch;
            })
            .map((template) => (
              <Col span={12} key={template.id}>
                <Card
                  hoverable
                  onClick={() => applyTemplate(template)}
                  style={{ cursor: "pointer" }}
                >
                  <Card.Meta
                    title={template.name}
                    description={template.description}
                  />
                  <div style={{ marginTop: "8px" }}>
                    <Tag color="blue">{template.category}</Tag>
                    <div
                      style={{
                        marginTop: "4px",
                        fontSize: "12px",
                        color: "#666",
                      }}
                    >
                      点击应用此模板
                    </div>
                  </div>
                </Card>
              </Col>
            ))}
        </Row>

        {workflowTemplates.filter((template) => {
          const matchesCategory =
            selectedCategory === "all" ||
            template.category === selectedCategory;
          const matchesSearch =
            searchKeyword === "" ||
            template.name.toLowerCase().includes(searchKeyword.toLowerCase()) ||
            template.description
              .toLowerCase()
              .includes(searchKeyword.toLowerCase());
          return matchesCategory && matchesSearch;
        }).length === 0 && (
          <div style={{ textAlign: "center", padding: "40px", color: "#999" }}>
            没有找到匹配的模板
          </div>
        )}
      </Modal>

      <style jsx>{`
        .enhanced-bpmn-designer {
          width: 100%;
        }

        .enhanced-bpmn-designer.fullscreen {
          position: fixed;
          top: 0;
          left: 0;
          width: 100vw;
          height: 100vh;
          z-index: 1000;
          background: white;
        }

        .bpmn-toolbar {
          background: #fafafa;
        }

        .enhanced-bpmn-designer :global(.djs-palette) {
          border: 1px solid #ccc;
        }

        .enhanced-bpmn-designer :global(.djs-context-pad) {
          border: 1px solid #ccc;
          border-radius: 4px;
        }
      `}</style>
    </div>
  );
};

export default EnhancedBPMNDesigner;
