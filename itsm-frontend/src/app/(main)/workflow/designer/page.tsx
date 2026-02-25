'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import {
  Layout,
  Button,
  Space,
  message,
  Typography,
  Row,
  Col,
  Tag,
  Card,
  Tabs,
  Select,
  Input,
  Form,
  Modal,
  Timeline,
  Badge,
  Alert,
  Checkbox,
} from 'antd';
import { UserApi } from '@/lib/api/user-api';
import { RoleAPI } from '@/lib/api/role-api';
import { httpClient } from '@/lib/api/http-client';
import {
  ArrowLeft,
  Save,
  PlayCircle,
  GitBranch,
  Settings,
  Eye,
  Edit3,
  CheckCircle,
  Clock,
  AlertCircle,
  FileText,
} from 'lucide-react';
import BPMNDesigner from '@/components/workflow/BPMNDesigner';
import { WorkflowAPI } from '@/lib/api/workflow-api';
import { WORKFLOW_TEMPLATES, TEMPLATE_CATEGORIES, getTemplateById } from '@/lib/workflow-templates';

const { Header, Content } = Layout;
const { Title, Text } = Typography;
const { Option } = Select;

interface WorkflowDesignerPageProps {
  params: Promise<{ id?: string }>;
}

interface WorkflowDefinition {
  id: string;
  name: string;
  description: string;
  version: string;
  category: string;
  status: 'draft' | 'active' | 'inactive' | 'archived';
  xml: string;
  created_at: string;
  updated_at: string;
  created_by: string;
  tags: string[];
  approval_config?: ApprovalConfig;
  variables?: WorkflowVariable[];
  sla_config?: SLAConfig;
}

interface ApprovalConfig {
  require_approval: boolean;
  approval_type: 'single' | 'parallel' | 'sequential' | 'conditional';
  approvers: string[];
  auto_approve_roles: string[];
  escalation_rules: EscalationRule[];
}

interface EscalationRule {
  level: number;
  timeout_hours: number;
  escalate_to: string[];
  action: 'notify' | 'auto_approve' | 'escalate';
}

interface WorkflowVariable {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'date' | 'object';
  required: boolean;
  default_value?: string | number | boolean | Date | Record<string, unknown>;
  description: string;
}

interface SLAConfig {
  response_time_hours: number;
  resolution_time_hours: number;
  business_hours_only: boolean;
  exclude_weekends: boolean;
  exclude_holidays: boolean;
}

interface WorkflowVersion {
  id: string;
  version: string;
  status: 'draft' | 'active' | 'archived';
  created_at: string;
  created_by: string;
  change_log: string;
  xml: string;
}

const WorkflowDesignerPage: React.FC<WorkflowDesignerPageProps> = ({ params }) => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [resolvedParams, setResolvedParams] = useState<{ id?: string }>({ id: undefined });

  const [form] = Form.useForm();
  const [metadataForm] = Form.useForm();

  const [workflow, setWorkflow] = useState<WorkflowDefinition | null>(null);
  const [showMetadataModal, setShowMetadataModal] = useState(false);
  const [showNewWorkflowModal, setShowNewWorkflowModal] = useState(false);
  const [newWorkflowForm] = Form.useForm();
  const [saving, setSaving] = useState(false);
  const [deploying, setDeploying] = useState(false);
  const [currentXML, setCurrentXML] = useState('');
  const [hasChanges, setHasChanges] = useState(false);
  const [activeTab, setActiveTab] = useState('designer');
  const [showVersionModal, setShowVersionModal] = useState(false);
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [workflowVersions, setWorkflowVersions] = useState<WorkflowVersion[]>([]);
  const [approvalConfig, setApprovalConfig] = useState<ApprovalConfig>({
    require_approval: true,
    approval_type: 'sequential',
    approvers: [],
    auto_approve_roles: [],
    escalation_rules: [],
  });

  // 用户和角色列表
  const [userList, setUserList] = useState<{ id: number; name: string; username: string }[]>([]);
  const [roleList, setRoleList] = useState<{ id: number; name: string; code: string }[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [loadingRoles, setLoadingRoles] = useState(false);

  // 加载用户列表
  const loadUserList = async () => {
    setLoadingUsers(true);
    try {
      const response = await UserApi.getUsers({ page: 1, page_size: 100 });
      const users = (response.users || []).map((u: any) => ({
        id: u.id,
        name: u.name || u.username || '未知用户',
        username: u.username,
      }));
      setUserList(users);
    } catch (error) {
      console.error('加载用户列表失败:', error);
    } finally {
      setLoadingUsers(false);
    }
  };

  // 加载角色列表
  const loadRoleList = async () => {
    setLoadingRoles(true);
    try {
      const response = await RoleAPI.getRoles() as any;
      const roles = (response.roles || response.data || []).map((r: any) => ({
        id: r.id,
        name: r.name || r.code || '未知角色',
        code: r.code,
      }));
      setRoleList(roles);
    } catch (error) {
      console.error('加载角色列表失败:', error);
    } finally {
      setLoadingRoles(false);
    }
  };

  // 解析 Promise params
  useEffect(() => {
    params.then(setResolvedParams).catch(console.error);
  }, [params]);

  // 从URL参数获取工作流ID
  const workflowId = resolvedParams?.id || searchParams.get('id');

  // 加载用户和角色列表
  useEffect(() => {
    loadUserList();
    loadRoleList();
  }, []);

  // 加载工作流配置（审批配置+SLA）
  const loadWorkflowConfig = async (key: string) => {
    try {
      // 从后端获取工作流配置
      const response = (await WorkflowAPI.getProcessDefinition(key)) as any;
      if (response) {
        setApprovalConfig({
          require_approval: response.require_approval ?? true,
          approval_type: response.approval_type || 'sequential',
          approvers: response.approvers || [],
          auto_approve_roles: response.auto_approve_roles || [],
          escalation_rules: response.escalation_rules || [],
        });
        setWorkflow(prev =>
          prev
            ? {
                ...prev,
                sla_config: response.sla_config || {
                  response_time_hours: 24,
                  resolution_time_hours: 72,
                  business_hours_only: true,
                  exclude_weekends: true,
                  exclude_holidays: true,
                },
              }
            : null
        );
      }
    } catch (error) {
      console.error('加载工作流配置失败:', error);
    }
  };

  useEffect(() => {
    if (workflowId && workflowId !== 'new') {
      loadWorkflow(workflowId);
      loadWorkflowVersions(workflowId);
      loadWorkflowConfig(workflowId);
    } else if (!workflowId) {
      // 显示新工作流创建向导
      setShowNewWorkflowModal(true);
    }
  }, [workflowId]);

  const loadWorkflow = async (id: string) => {
    if (id === 'new' || !id) return;

    try {
      // 使用新的BPMN API
      const response = (await WorkflowAPI.getProcessDefinition(id)) as any;

      // 如果返回空或无效，提示错误并跳转到列表
      if (!response || (!response.key && !response.id && !response.name)) {
        message.error('加载工作流失败，该工作流可能不存在');
        router.push('/workflow');
        return;
      }

      let xmlContent = '';
      if (response.bpmn_xml) {
        try {
          // 尝试Base64解码，如果失败则假设是原始XML
          // 简单的启发式检查：如果包含 <?xml 或 <bpmn:definitions 则可能是原始文本
          if (
            response.bpmn_xml.trim().startsWith('<?xml') ||
            response.bpmn_xml.trim().startsWith('<bpmn:definitions')
          ) {
            xmlContent = response.bpmn_xml;
          } else {
            xmlContent = atob(response.bpmn_xml);
          }
        } catch (e) {
          console.warn('XML Base64 decode failed, using raw content', e);
          xmlContent = response.bpmn_xml;
        }
      }

      setWorkflow({
        id: response.key || response.id,
        name: response.name || '未命名工作流',
        description: response.description || '',
        version: (response.version || '1').toString(),
        category: response.category || response.type || 'general',
        status: response.is_active ? 'active' : 'inactive',
        xml: xmlContent,
        created_at: response.created_at || new Date().toISOString(),
        updated_at: response.updated_at || new Date().toISOString(),
        created_by: '系统', // 从用户信息中获取
        tags: [],
        approval_config: approvalConfig,
        variables: [],
        sla_config: {
          response_time_hours: 24,
          resolution_time_hours: 72,
          business_hours_only: true,
          exclude_weekends: true,
          exclude_holidays: true,
        },
      });

      setCurrentXML(xmlContent || getDefaultBPMNXML());
    } catch (error) {
      console.error('加载工作流失败:', error);
      message.error('加载工作流失败');
    }
  };

  const loadWorkflowVersions = async (id: string) => {
    if (id === 'new') return;

    try {
      const versions = await WorkflowAPI.getProcessVersions(id);
      const normalized = (versions as any[]).map((version, index) => ({
        id: version.id || version.key || `version-${index}`,
        version: String(version.version ?? '1.0.0'),
        status: version.status || (version.is_active ? 'active' : 'draft'),
        created_at: version.created_at || new Date().toISOString(),
        created_by: version.created_by || '系统',
        change_log: version.change_log || '',
        xml: version.bpmn_xml || '',
      }));
      setWorkflowVersions(normalized);
    } catch (error) {
      console.error('加载工作流版本失败:', error);
    }
  };

  const getDefaultBPMNXML = () => {
    return `<?xml version="1.0" encoding="UTF-8"?>
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
    <bpmn:userTask id="UserTask_1" name="提交申请">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_2" name="审核">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_1" name="结束">
      <bpmn:incoming>Flow_3</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="UserTask_1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="UserTask_1" targetRef="UserTask_2" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="UserTask_2" targetRef="EndEvent_1" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_1">
      <bpmndi:BPMNShape id="StartEvent_1_di" bpmnElement="StartEvent_1">
        <dc:Bounds x="152" y="102" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="158" y="145" width="24" height="14" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="UserTask_1_di" bpmnElement="UserTask_1">
        <dc:Bounds x="240" y="80" width="100" height="80" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="255" y="105" width="70" height="30" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="UserTask_2_di" bpmnElement="UserTask_2">
        <dc:Bounds x="400" y="80" width="100" height="80" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="425" y="105" width="50" height="30" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_1_di" bpmnElement="EndEvent_1">
        <dc:Bounds x="552" y="102" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="558" y="145" width="24" height="14" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_1_di" bpmnElement="Flow_1">
        <di:waypoint x="188" y="120" />
        <di:waypoint x="240" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_2_di" bpmnElement="Flow_2">
        <di:waypoint x="340" y="120" />
        <di:waypoint x="400" y="120" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_3_di" bpmnElement="Flow_3">
        <di:waypoint x="500" y="120" />
        <di:waypoint x="552" y="120" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`;
  };

  // 工单流程模板
  const getTicketWorkflowBPMN = (name: string) => {
    return `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
                  id="Definitions_Ticket" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Ticket" name="${name}" isExecutable="true">
    <bpmn:startEvent id="StartEvent_Ticket" name="新建工单">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="Activity_Assign" name="分配处理人">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Activity_Handle" name="处理工单">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Activity_Confirm" name="确认完成">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_Ticket" name="工单关闭">
      <bpmn:incoming>Flow_4</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_Ticket" targetRef="Activity_Assign" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Activity_Assign" targetRef="Activity_Handle" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Activity_Handle" targetRef="Activity_Confirm" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="Activity_Confirm" targetRef="EndEvent_Ticket" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_Ticket">
      <bpmndi:BPMNShape id="StartEvent_Ticket_di" bpmnElement="StartEvent_Ticket">
        <dc:Bounds x="152" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_Assign_di" bpmnElement="Activity_Assign">
        <dc:Bounds x="240" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_Handle_di" bpmnElement="Activity_Handle">
        <dc:Bounds x="380" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_Confirm_di" bpmnElement="Activity_Confirm">
        <dc:Bounds x="520" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_Ticket_di" bpmnElement="EndEvent_Ticket">
        <dc:Bounds x="672" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`;
  };

  // 事件流程模板
  const getIncidentWorkflowBPMN = (name: string) => {
    return `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
                  id="Definitions_Incident" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Incident" name="${name}" isExecutable="true">
    <bpmn:startEvent id="StartEvent_Incident" name="故障发生">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="Activity_Triage" name="事件定级">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Activity_Resolve" name="故障处理">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Activity_Verify" name="验证恢复">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Activity_Review" name="事件回顾">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_Incident" name="事件关闭">
      <bpmn:incoming>Flow_5</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_Incident" targetRef="Activity_Triage" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Activity_Triage" targetRef="Activity_Resolve" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Activity_Resolve" targetRef="Activity_Verify" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="Activity_Verify" targetRef="Activity_Review" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="Activity_Review" targetRef="EndEvent_Incident" />
  </bpmn:process>
</bpmn:definitions>`;
  };

  // 变更流程模板
  const getChangeWorkflowBPMN = (name: string) => {
    return `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
                  id="Definitions_Change" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Change" name="${name}" isExecutable="true">
    <bpmn:startEvent id="StartEvent_Change" name="变更申请">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="Activity_Review" name="变更评审">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Activity_Approval" name="经理审批">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Activity_Implement" name="实施变更">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Activity_Verify" name="验证确认">
      <bpmn:incoming>Flow_4</bpmn:incoming>
      <bpmn:outgoing>Flow_5</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_Change" name="变更完成">
      <bpmn:incoming>Flow_5</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_Change" targetRef="Activity_Review" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Activity_Review" targetRef="Activity_Approval" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Activity_Approval" targetRef="Activity_Implement" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="Activity_Implement" targetRef="Activity_Verify" />
    <bpmn:sequenceFlow id="Flow_5" sourceRef="Activity_Verify" targetRef="EndEvent_Change" />
  </bpmn:process>
</bpmn:definitions>`;
  };

  // 审批流程模板
  const getApprovalWorkflowBPMN = (name: string) => {
    return `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
                  id="Definitions_Approval" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_Approval" name="${name}" isExecutable="true">
    <bpmn:startEvent id="StartEvent_Approval" name="提交申请">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="Activity_Approve1" name="一级审批">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="Activity_Approve2" name="二级审批">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_Approval" name="审批完成">
      <bpmn:incoming>Flow_3</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_Approval" targetRef="Activity_Approve1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Activity_Approve1" targetRef="Activity_Approve2" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Activity_Approve2" targetRef="EndEvent_Approval" />
  </bpmn:process>
</bpmn:definitions>`;
  };

  const handleSave = async (xml: string) => {
    if (!workflow) return;

    setSaving(true);
    try {
      if (workflow.id === 'new') {
        // 创建新工作流 - 使用新的BPMN API
        const tenantId = httpClient.getTenantId() || 1;
        const response = (await WorkflowAPI.createProcessDefinition({
          name: workflow.name,
          description: workflow.description,
          category: workflow.category,
          bpmn_xml: xml,
          tenant_id: tenantId,
        })) as any;

        // 更新工作流ID（使用key）、版本和状态
        setWorkflow(prev =>
          prev
            ? {
                ...prev,
                id: response.key || response.id, // 使用 key 作为 id
                version: response.version || '1.0.0',
                status: 'draft',
              }
            : null
        );

        message.success('工作流创建成功');
      } else {
        // 更新现有工作流（包括BPMN XML + 配置）
        const response = (await WorkflowAPI.updateProcessDefinition(workflow.id, {
          name: workflow.name,
          description: workflow.description,
          category: workflow.category,
          bpmn_xml: xml,
          approval_config: approvalConfig,
          sla_config: workflow.sla_config,
        }, workflow.version)) as any;

        // 更新版本号
        setWorkflow(prev =>
          prev
            ? {
                ...prev,
                version: response.version || prev.version,
              }
            : null
        );

        message.success('工作流更新成功');
      }

      setCurrentXML(xml);
      setHasChanges(false);
    } catch (error) {
      console.error('保存工作流失败:', error);
      message.error('保存工作流失败: ' + (error as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleDeploy = async () => {
    if (!workflow || !currentXML) return;

    setDeploying(true);
    try {
      // 部署工作流，传递版本号
      await WorkflowAPI.deployProcessDefinition(workflow.id, workflow.version);

      // 更新状态
      setWorkflow(prev =>
        prev
          ? {
              ...prev,
              status: 'active',
            }
          : null
      );

      message.success('工作流部署成功');
    } catch (error) {
      console.error('部署工作流失败:', error);
      message.error('部署工作流失败');
    } finally {
      setDeploying(false);
    }
  };

  // 保存并部署 - 一键完成
  const handleSaveAndDeploy = async (xml: string) => {
    if (!workflow) return;

    setSaving(true);
    setDeploying(true);
    try {
      // 获取当前版本号
      const currentVersion = workflow.version || '1.0.0';
      console.log('handleSaveAndDeploy:', { id: workflow.id, version: currentVersion, name: workflow.name });

      if (workflow.id === 'new') {
        // 创建并部署新工作流
        const createData = {
          name: workflow.name,
          description: workflow.description || '',
          category: workflow.category || 'general',
          bpmn_xml: xml,
          tenant_id: httpClient.getTenantId() || 1,
        };
        console.log('Creating workflow:', createData);

        const response = await WorkflowAPI.createProcessDefinition(createData) as any;
        console.log('Create response:', response);

        if (!response) {
          throw new Error('创建工作流失败：服务器返回空响应');
        }

        const newVersion = response.version || '1.0.0';
        const newKey = response.key || response.id;

        if (!newKey) {
          throw new Error('创建工作流失败：未获取到工作流ID');
        }

        console.log('Deploying workflow:', { key: newKey, version: newVersion });
        // 等待一下确保创建完成
        await new Promise(resolve => setTimeout(resolve, 500));

        // 立即部署
        await WorkflowAPI.deployProcessDefinition(newKey, newVersion);

        // 更新工作流状态
        setWorkflow(prev =>
          prev
            ? {
                ...prev,
                id: newKey,
                version: newVersion,
                status: 'active',
              }
            : null
        );

        message.success('工作流创建并部署成功');
      } else {
        // 先保存 - 只传递简单字段，避免序列化问题
        const updateData = {
          name: workflow.name,
          description: workflow.description || '',
          category: workflow.category || 'general',
          bpmn_xml: xml,
        };
        console.log('Updating workflow:', { id: workflow.id, version: currentVersion, data: updateData });

        const updateResponse = await WorkflowAPI.updateProcessDefinition(workflow.id, updateData, currentVersion) as any;
        console.log('Update response:', updateResponse);

        // 再部署
        console.log('Deploying workflow:', { id: workflow.id, version: currentVersion });
        await WorkflowAPI.deployProcessDefinition(workflow.id, currentVersion);

        // 更新状态
        setWorkflow(prev =>
          prev
            ? {
                ...prev,
                status: 'active',
              }
            : null
        );

        message.success('工作流保存并部署成功');
      }

      setCurrentXML(xml);
      setHasChanges(false);
    } catch (error) {
      console.error('保存并部署失败:', error);
      const errorMsg = error instanceof Error ? error.message : String(error);
      message.error('保存并部署失败: ' + errorMsg);
    } finally {
      setSaving(false);
      setDeploying(false);
    }
  };

  const handleCreateVersion = async () => {
    if (!workflow) return;

    try {
      const newVersion: WorkflowVersion = {
        id: `version-${Date.now()}`,
        version: `${parseFloat(workflow.version) + 0.1}`.slice(0, 3),
        status: 'draft',
        created_at: new Date().toISOString(),
        created_by: '当前用户',
        change_log: '创建新版本',
        xml: currentXML,
      };

      setWorkflowVersions([...workflowVersions, newVersion]);
      setWorkflow(prev =>
        prev
          ? {
              ...prev,
              version: newVersion.version,
            }
          : null
      );

      message.success('新版本创建成功');
      setShowVersionModal(false);
    } catch (error) {
      console.error('创建版本失败:', error);
      message.error('创建版本失败');
    }
  };

  const handleSwitchVersion = async (versionId: string) => {
    try {
      const version = workflowVersions.find(v => v.id === versionId);
      if (version) {
        setCurrentXML(version.xml);
        setWorkflow(prev =>
          prev
            ? {
                ...prev,
                version: version.version,
              }
            : null
        );
        message.success(`已切换到版本 ${version.version}`);
      }
    } catch (error) {
      console.error('切换版本失败:', error);
      message.error('切换版本失败');
    }
  };

  const handleSaveSettings = async () => {
    try {
      const values = await form.validateFields();

      setWorkflow(prev =>
        prev
          ? {
              ...prev,
              approval_config: values.approval_config,
              sla_config: values.sla_config,
            }
          : null
      );

      message.success('设置保存成功');
      setShowSettingsModal(false);
    } catch (error) {
      console.error('保存设置失败:', error);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'success';
      case 'draft':
        return 'processing';
      case 'inactive':
        return 'default';
      case 'archived':
        return 'warning';
      default:
        return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return <CheckCircle className='w-4 h-4' />;
      case 'draft':
        return <Edit3 className='w-4 h-4' />;
      case 'inactive':
        return <Clock className='w-4 h-4' />;
      case 'archived':
        return <AlertCircle className='w-4 h-4' />;
      default:
        return <Clock className='w-4 h-4' />;
    }
  };

  return (
    <Layout className='h-screen'>
      <Header className='bg-white px-6 border-b border-gray-100 flex items-center justify-between h-16 leading-none'>
        <div className='flex items-center justify-between w-full h-full'>
          <div className='flex items-center'>
            <Button
              type='text'
              icon={<ArrowLeft className='w-4 h-4' />}
              onClick={() => router.back()}
              className='mr-4'
            >
              返回
            </Button>
            <div>
              <Title level={4} className='!mb-0 !text-base flex items-center gap-2'>
                {workflow?.name || '工作流设计器'}
                <Button
                  type="text"
                  size="small"
                  icon={<Edit3 className="w-3 h-3" />}
                  onClick={() => {
                    metadataForm.setFieldsValue({
                      name: workflow?.name,
                      description: workflow?.description,
                      category: workflow?.category,
                    });
                    setShowMetadataModal(true);
                  }}
                />
              </Title>
              <div className='flex items-center gap-2 mt-1'>
                <Tag color={getStatusColor(workflow?.status || 'draft')} className='mr-0'>
                  <span className='flex items-center gap-1'>
                    {getStatusIcon(workflow?.status || 'draft')}
                    {workflow?.status === 'active'
                      ? '已激活'
                      : workflow?.status === 'draft'
                        ? '草稿'
                        : workflow?.status === 'inactive'
                          ? '未激活'
                          : '已归档'}
                  </span>
                </Tag>
                <Text type='secondary' className='text-xs'>
                  版本 {workflow?.version}
                </Text>
                {workflow?.category && (
                  <Tag color='blue' className='ml-2'>
                    {workflow.category}
                  </Tag>
                )}
              </div>
            </div>
          </div>

          <Space>
            <Button
              icon={<GitBranch className='w-4 h-4' />}
              onClick={() => setShowVersionModal(true)}
            >
              版本管理
            </Button>
            <Button
              icon={<Settings className='w-4 h-4' />}
              onClick={() => setShowSettingsModal(true)}
            >
              流程设置
            </Button>
            <Button
              icon={<Save className='w-4 h-4' />}
              loading={saving}
              onClick={() => handleSave(currentXML)}
            >
              保存
            </Button>
            <Button
              type='primary'
              icon={<PlayCircle className='w-4 h-4' />}
              loading={saving || deploying}
              onClick={() => handleSaveAndDeploy(currentXML)}
              disabled={!workflow}
            >
              保存并部署
            </Button>
            <Button
              icon={<PlayCircle className='w-4 h-4' />}
              loading={deploying}
              onClick={handleDeploy}
              disabled={!workflow || workflow.status === 'active'}
            >
              部署
            </Button>
          </Space>
        </div>
      </Header>

      <Layout>
        <Content className='p-6 bg-gray-50'>
          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            items={[
              {
                key: 'designer',
                label: '流程设计',
                children: (
                  <div className='h-[calc(100vh-200px)] bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden'>
                    <BPMNDesigner
                      xml={currentXML}
                      onSave={handleSave}
                      onChange={xml => {
                        setCurrentXML(xml);
                        setHasChanges(true);
                      }}
                    />
                  </div>
                ),
              },
              {
                key: 'versions',
                label: '版本历史',
                children: (
                  <Card className='rounded-lg shadow-sm border border-gray-200' variant="borderless">
                    <div className='flex justify-between items-center mb-6'>
                      <Title level={5} className='!mb-0'>
                        版本历史
                      </Title>
                      <Button
                        type='primary'
                        icon={<GitBranch className='w-4 h-4' />}
                        onClick={() => setShowVersionModal(true)}
                      >
                        创建新版本
                      </Button>
                    </div>

                    <Timeline className='mt-4'>
                      {workflowVersions.map(version => (
                        <Timeline.Item
                          key={version.id}
                          dot={
                            <Badge
                              status={version.status === 'active' ? 'success' : 'default'}
                              text={version.status === 'active' ? '当前' : ''}
                            />
                          }
                        >
                          <div className='flex justify-between items-center ml-2'>
                            <div>
                              <Text strong>版本 {version.version}</Text>
                              <div className='text-sm text-gray-500 mt-1'>{version.change_log}</div>
                              <div className='text-xs text-gray-400 mt-1'>
                                {new Date(version.created_at).toLocaleString()} -{' '}
                                {version.created_by}
                              </div>
                            </div>
                            <Space>
                              <Button
                                size='small'
                                icon={<Eye className='w-3 h-3' />}
                                onClick={() => handleSwitchVersion(version.id)}
                              >
                                查看
                              </Button>
                              {version.status !== 'active' && (
                                <Button
                                  size='small'
                                  type='primary'
                                  onClick={() => handleSwitchVersion(version.id)}
                                >
                                  切换到此版本
                                </Button>
                              )}
                            </Space>
                          </div>
                        </Timeline.Item>
                      ))}
                    </Timeline>
                  </Card>
                ),
              },
              {
                key: 'config',
                label: '流程配置',
                children: (
                  <Row gutter={[24, 24]}>
                    <Col span={12}>
                      <Card
                        title='审批配置'
                        className='h-full rounded-lg shadow-sm border border-gray-200'
                        variant="borderless"
                      >
                        <div className='space-y-6'>
                          <div>
                            <Text strong className='block mb-2'>
                              审批类型
                            </Text>
                            <Select
                              value={approvalConfig.approval_type}
                              onChange={value =>
                                setApprovalConfig(prev => ({
                                  ...prev,
                                  approval_type: value,
                                }))
                              }
                              className='w-full'
                            >
                              <Option value='single'>单人审批</Option>
                              <Option value='parallel'>并行审批</Option>
                              <Option value='sequential'>串行审批</Option>
                              <Option value='conditional'>条件审批</Option>
                            </Select>
                          </div>

                          <div>
                            <Text strong className='block mb-2'>
                              审批人
                            </Text>
                            <Select
                              mode='multiple'
                              placeholder='选择审批人'
                              value={approvalConfig.approvers}
                              onChange={value =>
                                setApprovalConfig(prev => ({
                                  ...prev,
                                  approvers: value,
                                }))
                              }
                              className='w-full'
                              loading={loadingUsers}
                            >
                              {userList.map(user => (
                                <Option key={user.id} value={String(user.id)}>
                                  {user.name}
                                </Option>
                              ))}
                            </Select>
                          </div>

                          <div>
                            <Text strong className='block mb-2'>
                              自动审批角色
                            </Text>
                            <Select
                              mode='multiple'
                              placeholder='选择角色'
                              value={approvalConfig.auto_approve_roles}
                              onChange={value =>
                                setApprovalConfig(prev => ({
                                  ...prev,
                                  auto_approve_roles: value,
                                }))
                              }
                              className='w-full'
                              loading={loadingRoles}
                            >
                              {roleList.map(role => (
                                <Option key={role.code} value={role.code}>
                                  {role.name}
                                </Option>
                              ))}
                            </Select>
                          </div>
                        </div>
                      </Card>
                    </Col>

                    <Col span={12}>
                      <Card
                        title='SLA配置'
                        className='h-full rounded-lg shadow-sm border border-gray-200'
                        variant="borderless"
                      >
                        <div className='space-y-6'>
                          <div>
                            <Text strong className='block mb-2'>
                              响应时间
                            </Text>
                            <Input
                              type='number'
                              suffix='小时'
                              value={workflow?.sla_config?.response_time_hours}
                              onChange={e =>
                                setWorkflow(prev =>
                                  prev
                                    ? {
                                        ...prev,
                                        sla_config: {
                                          ...prev.sla_config!,
                                          response_time_hours: parseInt(e.target.value) || 24,
                                        },
                                      }
                                    : null
                                )
                              }
                            />
                          </div>

                          <div>
                            <Text strong className='block mb-2'>
                              解决时间
                            </Text>
                            <Input
                              type='number'
                              suffix='小时'
                              value={workflow?.sla_config?.resolution_time_hours}
                              onChange={e =>
                                setWorkflow(prev =>
                                  prev
                                    ? {
                                        ...prev,
                                        sla_config: {
                                          ...prev.sla_config!,
                                          resolution_time_hours: parseInt(e.target.value) || 72,
                                        },
                                      }
                                    : null
                                )
                              }
                            />
                          </div>

                          <div>
                            <Text strong className='block mb-2'>
                              工作时间设置
                            </Text>
                            <div className='space-y-3'>
                              <div className='flex items-center'>
                                <Checkbox
                                  checked={workflow?.sla_config?.business_hours_only}
                                  onChange={e =>
                                    setWorkflow(prev =>
                                      prev
                                        ? {
                                            ...prev,
                                            sla_config: {
                                              ...prev.sla_config!,
                                              business_hours_only: e.target.checked,
                                            },
                                          }
                                        : null
                                    )
                                  }
                                >
                                  仅工作时间
                                </Checkbox>
                              </div>
                              <div className='flex items-center'>
                                <Checkbox
                                  checked={workflow?.sla_config?.exclude_weekends}
                                  onChange={e =>
                                    setWorkflow(prev =>
                                      prev
                                        ? {
                                            ...prev,
                                            sla_config: {
                                              ...prev.sla_config!,
                                              exclude_weekends: e.target.checked,
                                            },
                                          }
                                        : null
                                    )
                                  }
                                >
                                  排除周末
                                </Checkbox>
                              </div>
                            </div>
                          </div>
                        </div>
                      </Card>
                    </Col>
                  </Row>
                ),
              },
            ]}
          />
        </Content>
      </Layout>

      {/* 版本管理模态框 */}
      <Modal
        title='创建新版本'
        open={showVersionModal}
        onOk={handleCreateVersion}
        onCancel={() => setShowVersionModal(false)}
        okText='创建'
        cancelText='取消'
      >
        <div className='space-y-4'>
          <Alert
            message='版本管理'
            description='创建新版本将保存当前的设计状态，不会影响已部署的版本。'
            type='info'
            showIcon
          />
          <div>
            <Text strong>当前版本</Text>
            <div className='mt-1'>
              <Tag color='blue'>{workflow?.version}</Tag>
            </div>
          </div>
          <div>
            <Text strong>新版本号</Text>
            <div className='mt-1'>
              <Tag color='green'>
                {workflow ? `${parseFloat(workflow.version) + 0.1}`.slice(0, 3) : '1.1'}
              </Tag>
            </div>
          </div>
        </div>
      </Modal>

      {/* 流程设置模态框 */}
      <Modal
        title='流程设置'
        open={showSettingsModal}
        onOk={handleSaveSettings}
        onCancel={() => setShowSettingsModal(false)}
        width={800}
        okText='保存'
        cancelText='取消'
      >
        <Form form={form} layout='vertical' initialValues={workflow || {}}>
          <Tabs
            items={[
              {
                key: 'approval',
                label: '审批配置',
                children: (
                  <>
                    <Form.Item
                      label='审批类型'
                      name={['approval_config', 'approval_type']}
                      rules={[{ required: true, message: '请选择审批类型' }]}
                    >
                      <Select>
                        <Option value='single'>单人审批</Option>
                        <Option value='parallel'>并行审批</Option>
                        <Option value='sequential'>串行审批</Option>
                        <Option value='conditional'>条件审批</Option>
                      </Select>
                    </Form.Item>

                    <Form.Item label='审批人' name={['approval_config', 'approvers']}>
                      <Select mode='multiple' placeholder='选择审批人' loading={loadingUsers}>
                        {userList.map(user => (
                          <Option key={user.id} value={String(user.id)}>
                            {user.name}
                          </Option>
                        ))}
                      </Select>
                    </Form.Item>
                  </>
                )
              },
              {
                key: 'sla',
                label: 'SLA配置',
                children: (
                  <>
                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item label='响应时间(小时)' name={['sla_config', 'response_time_hours']}>
                          <Input type='number' />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item label='解决时间(小时)' name={['sla_config', 'resolution_time_hours']}>
                          <Input type='number' />
                        </Form.Item>
                      </Col>
                    </Row>

                    <Form.Item
                      label='工作时间设置'
                      name={['sla_config', 'business_hours_only']}
                      valuePropName='checked'
                    >
                      <Checkbox>仅工作时间</Checkbox>
                    </Form.Item>
                  </>
                )
              }
            ]}
          />
        </Form>
      </Modal>

      {/* 工作流元数据编辑模态框 */}
      <Modal
        title='编辑工作流信息'
        open={showMetadataModal}
        onOk={() => {
          metadataForm.validateFields().then(values => {
            setWorkflow(prev =>
              prev
                ? {
                    ...prev,
                    name: values.name,
                    description: values.description,
                    category: values.category,
                  }
                : null
            );
            setShowMetadataModal(false);
          });
        }}
        onCancel={() => setShowMetadataModal(false)}
        okText='保存'
        cancelText='取消'
      >
        <Form form={metadataForm} layout='vertical'>
          <Form.Item
            label='工作流名称'
            name='name'
            rules={[{ required: true, message: '请输入工作流名称' }]}
          >
            <Input placeholder='请输入工作流名称' />
          </Form.Item>
          <Form.Item label='描述' name='description'>
            <Input.TextArea rows={3} placeholder='请输入工作流描述' />
          </Form.Item>
          <Form.Item label='分类' name='category'>
            <Select placeholder='请选择分类'>
              <Option value='general'>通用</Option>
              <Option value='approval'>审批流程</Option>
              <Option value='ticket'>工单流程</Option>
              <Option value='incident'>事件流程</Option>
              <Option value='change'>变更流程</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 新建工作流向导弹窗 - 模板选择 */}
      <Modal
        title='选择工作流模板'
        open={showNewWorkflowModal}
        onCancel={() => {
          setShowNewWorkflowModal(false);
          router.push('/workflow');
        }}
        footer={null}
        width={900}
        destroyOnClose
      >
        <div className='mb-4'>
          <Input.Search
            placeholder='搜索模板...'
            style={{ width: 300 }}
            onSearch={(value) => {
              // 可以添加搜索功能
            }}
          />
        </div>

        <div className='grid grid-cols-2 md:grid-cols-3 gap-4' style={{ gridTemplateColumns: 'repeat(3, 1fr)' }}>
          {WORKFLOW_TEMPLATES.map((template) => (
            <div
              key={template.id}
              className='border rounded-lg p-4 cursor-pointer hover:border-blue-500 hover:bg-blue-50 transition-all'
              style={{
                border: '1px solid #d9d9d9',
                borderRadius: '8px',
                padding: '16px',
                cursor: 'pointer',
                transition: 'all 0.2s',
              }}
              onClick={() => {
                // 选择模板后创建工作流
                const newWorkflow: WorkflowDefinition = {
                  id: 'new',
                  name: template.name,
                  description: template.description,
                  version: '1.0.0',
                  category: template.category,
                  status: 'draft',
                  xml: template.bpmn_xml,
                  created_at: new Date().toISOString(),
                  updated_at: new Date().toISOString(),
                  created_by: '当前用户',
                  tags: [],
                  approval_config: {
                    require_approval: template.approval_config.require_approval,
                    approval_type: template.approval_config.approval_type,
                    approvers: template.approval_config.approvers,
                    auto_approve_roles: [],
                    escalation_rules: [],
                  },
                  variables: [],
                  sla_config: {
                    response_time_hours: 24,
                    resolution_time_hours: 72,
                    business_hours_only: true,
                    exclude_weekends: true,
                    exclude_holidays: true,
                  },
                };
                setWorkflow(newWorkflow);
                setCurrentXML(newWorkflow.xml);
                setShowNewWorkflowModal(false);
              }}
            >
              <div className='flex items-center mb-2'>
                <div className='w-10 h-10 rounded-lg bg-blue-100 flex items-center justify-center mr-3'>
                  <FileText className='w-5 h-5 text-blue-600' />
                </div>
                <div>
                  <div className='font-medium'>{template.name}</div>
                  <div className='text-xs text-gray-500'>{TEMPLATE_CATEGORIES.find(c => c.key === template.category)?.name || template.category}</div>
                </div>
              </div>
              <div className='text-xs text-gray-500 mt-2'>{template.description}</div>
            </div>
          ))}
        </div>

        <div className='mt-6 border-t pt-4'>
          <div className='text-sm text-gray-500 mb-3'>或者自定义创建</div>
          <Form
            form={newWorkflowForm}
            layout='vertical'
            onFinish={(values) => {
              const newWorkflow: WorkflowDefinition = {
                id: 'new',
                name: values.name,
                description: values.description || '',
                version: '1.0.0',
                category: 'custom',
                status: 'draft',
                xml: getDefaultBPMNXML(),
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                created_by: '当前用户',
                tags: [],
                approval_config: approvalConfig,
                variables: [],
                sla_config: {
                  response_time_hours: values.sla_response || 24,
                  resolution_time_hours: values.sla_resolution || 72,
                  business_hours_only: true,
                  exclude_weekends: true,
                  exclude_holidays: true,
                },
              };
              setWorkflow(newWorkflow);
              setCurrentXML(newWorkflow.xml);
              setShowNewWorkflowModal(false);
            }}
          >
            <div className='flex gap-4'>
              <Form.Item
                label='工作流名称'
                name='name'
                rules={[{ required: true, message: '请输入工作流名称' }]}
                style={{ flex: 1 }}
              >
                <Input placeholder='自定义流程名称' />
              </Form.Item>
              <Form.Item style={{ marginTop: '32px' }}>
                <Button type='primary' htmlType='submit'>
                  创建空白流程
                </Button>
              </Form.Item>
            </div>
          </Form>
        </div>
      </Modal>
    </Layout>
  );
};

export default WorkflowDesignerPage;
