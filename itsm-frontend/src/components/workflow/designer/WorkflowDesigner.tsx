// 工作流设计器主组件
// Workflow Designer Main Component

'use client';

import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { Layout, Tabs, Form, Modal, Tag, Button, Space, Typography, Switch, App } from 'antd';
import { useRouter, useSearchParams } from 'next/navigation';
import { 
  CheckCircleOutlined, 
  CloseCircleOutlined, 
  WarningOutlined, 
  BugOutlined,
  HistoryOutlined,
  DiffOutlined
} from '@ant-design/icons';
import { WorkflowAPI } from '@/lib/api/workflow-api';
import { UserApi } from '@/lib/api/user-api';
import { RoleAPI } from '@/lib/api/role-api';
import { GroupAPI } from '@/lib/api/group-api';
import { httpClient } from '@/lib/api/http-client';

import { WorkflowDesignerContext } from './WorkflowContext';
import WorkflowToolbar from './WorkflowToolbar';
import WorkflowCanvas, { getBpmnDesignerApi } from './WorkflowCanvas';
import WorkflowNodeInspector from './WorkflowNodeInspector';
import WorkflowProperties from './WorkflowProperties';
import WorkflowNewModal from './WorkflowNewModal';
import WorkflowVersionModal from './WorkflowVersionModal';
import WorkflowSettingsModal from './WorkflowSettingsModal';
import WorkflowMetadataModal from './WorkflowMetadataModal';
import WorkflowAIModal from './WorkflowAIModal';

import type { BpmnNodeSelection } from '../BPMNDesigner';
import type { WorkflowDefinition, WorkflowVersion, ApprovalConfig } from './WorkflowTypes';

const { Content } = Layout;
const { Text, Title } = Typography;

interface WorkflowDesignerProps {
  workflowId?: string;
  initialXML?: string;
}

// 校验问题类型
interface ValidationIssue {
  type: 'error' | 'warning' | 'info';
  message: string;
  elementId?: string;
  elementType?: string;
  elementName?: string;
}

// 默认 BPMN XML
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
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="UserTask_2_di" bpmnElement="UserTask_2">
        <dc:Bounds x="400" y="80" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_1_di" bpmnElement="EndEvent_1">
        <dc:Bounds x="552" y="102" width="36" height="36" />
      </bpmndi:BPMNShape>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`;
};

// 内部组件
function WorkflowDesignerInner({ workflowId }: { workflowId?: string }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { message } = App.useApp();

  const [form] = Form.useForm();
  const [metadataForm] = Form.useForm();

  // 状态
  const [workflow, setWorkflow] = useState<WorkflowDefinition | null>(null);
  const [currentXML, setCurrentXML] = useState('');
  const [hasChanges, setHasChanges] = useState(false);
  const [activeTab, setActiveTab] = useState('designer');
  const [saving, setSaving] = useState(false);
  const [deploying, setDeploying] = useState(false);
  const [approvalConfig, setApprovalConfig] = useState<ApprovalConfig>({
    require_approval: true,
    approval_type: 'sequential',
    approvers: [],
    // 审批组是节点级。详见 WorkflowNodeInspector 节点面板的「候选组」字段。
    auto_approve_roles: [],
    escalation_rules: [],
  });
  const [workflowVersions, setWorkflowVersions] = useState<WorkflowVersion[]>([]);
  const [userList, setUserList] = useState<{ id: number; name: string; username: string }[]>([]);
  const [roleList, setRoleList] = useState<{ id: number; name: string; code: string }[]>([]);
  const [groupList, setGroupList] = useState<{ id: number; name: string; description?: string; memberCount?: number }[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [loadingRoles, setLoadingRoles] = useState(false);
  const [loadingGroups, setLoadingGroups] = useState(false);

  // 画布选中状态 - 驱动 WorkflowNodeInspector
  const [selectedNode, setSelectedNode] = useState<BpmnNodeSelection | null>(null);
  // 保留 ref 以便在 onChange/命令栈中读取最新值
  const selectedNodeRef = useRef<BpmnNodeSelection | null>(null);
  useEffect(() => {
    selectedNodeRef.current = selectedNode;
  }, [selectedNode]);

  // 校验相关状态
  const [validationIssues, setValidationIssues] = useState<ValidationIssue[]>([]);
  const [showValidationPanel, setShowValidationPanel] = useState(false);
  const [validating, setValidating] = useState(false);
  const [autoValidate, setAutoValidate] = useState(true); // 保存前自动校验

  // 版本对比相关状态
  const [showVersionCompare, setShowVersionCompare] = useState(false);
  const [compareVersions, setCompareVersions] = useState<{ version1?: WorkflowVersion; version2?: WorkflowVersion }>({});

  // AI模态框状态
  const [showAIModal, setShowAIModal] = useState(false);

  const handleSelectionChange = useCallback((selection: BpmnNodeSelection | null) => {
    setSelectedNode(selection);
  }, []);

  // 通过模块级 _apiRef 拿到 BPMNDesigner 的命令式 API，修改节点属性
  const handleUpdateNodeProperties = useCallback(
    (elementId: string, properties: Record<string, unknown>) => {
      const api = getBpmnDesignerApi();
      if (!api) {
        message.warning('流程设计器未就绪，请稍后重试');
        return false;
      }
      const ok = api.updateElementProperties(elementId, properties);
      if (ok) {
        // 同步本地选中节点的快照，避免面板一直显示旧值
        setSelectedNode(prev =>
          prev && prev.id === elementId
            ? { ...prev, businessObject: { ...(prev.businessObject || {}), ...properties } }
            : prev
        );
      }
      return ok;
    },
    []
  );

  const handleRefreshSelection = useCallback(() => {
    // 重读：从当前画布 XML 重新解析选中节点的 businessObject
    // 这里简单地清空再触发 onChange 重新通知 BPMNDesigner 重新触发选择事件
    const cur = selectedNodeRef.current;
    if (cur) {
      setSelectedNode(null);
      // 下一帧恢复，触发面板重新拉取 businessObject
      requestAnimationFrame(() => setSelectedNode(cur));
    }
  }, []);

  // 弹窗状态
  const [showNewWorkflowModal, setShowNewWorkflowModal] = useState(false);
  const [showVersionModal, setShowVersionModal] = useState(false);
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [showMetadataModal, setShowMetadataModal] = useState(false);

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
      const response = (await RoleAPI.getRoles()) as any;
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

  // 加载审批组列表
  const loadGroupList = async () => {
    setLoadingGroups(true);
    try {
      const tenantId = httpClient.getTenantId() || 1;
      const response = await GroupAPI.getGroups({ page: 1, page_size: 100, tenant_id: tenantId });
      const groups = (response.groups || []).map((g: any) => ({
        id: g.id,
        name: g.name || '未命名组',
        description: g.description,
        memberCount: Array.isArray(g.members) ? g.members.length : undefined,
      }));
      setGroupList(groups);
    } catch (error) {
      console.error('加载审批组列表失败:', error);
    } finally {
      setLoadingGroups(false);
    }
  };

  // 加载工作流
  const loadWorkflow = async (id: string) => {
    if (id === 'new' || !id) return;

    try {
      const response = (await WorkflowAPI.getProcessDefinition(id)) as any;

      if (!response || (!response.key && !response.id && !response.name)) {
        message.error('加载工作流失败，该工作流可能不存在');
        router.push('/workflow');
        return;
      }

      let xmlContent = '';
      if (response.bpmnXml) {
        try {
          if (
            response.bpmnXml.trim().startsWith('<?xml') ||
            response.bpmnXml.trim().startsWith('<bpmn:definitions')
          ) {
            xmlContent = response.bpmnXml;
          } else {
            xmlContent = atob(response.bpmnXml);
          }
        } catch (e) {
          console.warn('XML Base64 decode failed, using raw content', e);
          xmlContent = response.bpmnXml;
        }
      }

      const workflowData: WorkflowDefinition = {
        id: response.key || response.id,
        name: response.name || '未命名工作流',
        description: response.description || '',
        version: (response.version || '1').toString(),
        category: response.category || response.type || 'general',
        status: response.isActive ? 'active' : 'inactive',
        xml: xmlContent,
        created_at: response.createdAt || response.created_at || new Date().toISOString(),
        updated_at: response.updatedAt || response.updated_at || new Date().toISOString(),
        created_by: '系统',
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
      };

      setWorkflow(workflowData);
      setCurrentXML(xmlContent || getDefaultBPMNXML());
    } catch (error) {
      console.error('加载工作流失败:', error);
      message.error('加载工作流失败');
    }
  };

  // 加载工作流版本
  const loadWorkflowVersions = async (id: string) => {
    if (id === 'new') return;

    try {
      const versions = await WorkflowAPI.getProcessVersions(id);
      const normalized = (versions as any[]).map((version, index) => ({
        id: version.id || version.key || `version-${index}`,
        version: String(version.version ?? '1.0.0'),
        status: version.status || (version.isActive ? 'active' : 'draft'),
        created_at: version.createdAt || version.created_at || new Date().toISOString(),
        created_by: version.createdBy || version.created_by || '系统',
        change_log: version.changeLog || version.change_log || '',
        xml: version.bpmnXml || '',
      }));
      setWorkflowVersions(normalized);
    } catch (error) {
      console.error('加载工作流版本失败:', error);
    }
  };

  // 加载工作流配置
  const loadWorkflowConfig = async (key: string) => {
    try {
      const response = (await WorkflowAPI.getProcessDefinition(key)) as any;
      if (response) {
        setApprovalConfig({
          require_approval: response.requireApproval ?? response.require_approval ?? true,
          approval_type: response.approvalType || response.approval_type || 'sequential',
          approvers: response.approvers || [],
          auto_approve_roles: response.autoApproveRoles || response.auto_approve_roles || [],
          escalation_rules: response.escalationRules || response.escalation_rules || [],
        });

        const slaConfig = response.slaConfig || response.sla_config;
        if (slaConfig && workflow) {
          setWorkflow({
            ...workflow,
            sla_config: slaConfig,
          });
        }
      }
    } catch (error) {
      console.error('加载工作流配置失败:', error);
    }
  };

  // 初始化加载
  useEffect(() => {
    loadUserList();
    loadRoleList();
    loadGroupList();
  }, []);

  // 根据 ID 加载工作流
  useEffect(() => {
    const id = workflowId || searchParams?.get('id');
    if (id && id !== 'new') {
      loadWorkflow(id);
      loadWorkflowVersions(id);
      loadWorkflowConfig(id);
    } else if (!id) {
      setShowNewWorkflowModal(true);
    }
  }, [workflowId, searchParams]);

  // 更新工作流
  const updateWorkflow = (updates: Partial<WorkflowDefinition>) => {
    setWorkflow(prev => (prev ? { ...prev, ...updates } : null));
  };

  // 更新 SLA 配置
  const updateSLAConfig = (config: Partial<NonNullable<WorkflowDefinition['sla_config']>>) => {
    setWorkflow(prev =>
      prev
        ? {
            ...prev,
            sla_config: prev.sla_config
              ? { ...prev.sla_config, ...config }
              : {
                  response_time_hours: 24,
                  resolution_time_hours: 72,
                  business_hours_only: true,
                  exclude_weekends: true,
                  exclude_holidays: true,
                  ...config,
                },
          }
        : null
    );
  };

  // 流程校验
  const validateWorkflow = useCallback(async (showSuccessMessage = false) => {
    const api = getBpmnDesignerApi();
    if (!api) {
      message.warning('流程设计器未就绪，请稍后重试');
      return [];
    }

    setValidating(true);
    try {
      const issues = (await api.validate()) as ValidationIssue[];
      setValidationIssues(issues);
      
      if (issues.length > 0) {
        const errorCount = issues.filter((i: ValidationIssue) => i.type === 'error').length;
        const warningCount = issues.filter((i: ValidationIssue) => i.type === 'warning').length;
        
        if (showSuccessMessage) {
          if (errorCount > 0) {
            message.error(`校验发现 ${errorCount} 个错误，${warningCount} 个警告`);
          } else if (warningCount > 0) {
            message.warning(`校验发现 ${warningCount} 个警告`);
          } else {
            message.success('流程校验通过');
          }
        }
        
        // 如果有错误，自动显示校验面板
        if (errorCount > 0) {
          setShowValidationPanel(true);
        }
        
        return issues;
      } else {
        if (showSuccessMessage) {
          message.success('流程校验通过，未发现问题');
        }
        setShowValidationPanel(false);
        return [];
      }
    } catch (error) {
      console.error('校验失败:', error);
      message.error('流程校验失败');
      return [];
    } finally {
      setValidating(false);
    }
  }, []);

  // 保存工作流
  const handleSave = async (xml: string) => {
    if (!workflow) return;

    // 自动校验
    if (autoValidate) {
      const issues = await validateWorkflow();
      const hasErrors = issues.some((i: ValidationIssue) => i.type === 'error');
      if (hasErrors) {
        Modal.confirm({
          title: '流程存在错误',
          content: '当前流程存在校验错误，建议修复后再保存。是否仍要继续保存？',
          okText: '继续保存',
          cancelText: '取消',
          onOk: async () => {
            await doSave(xml);
          }
        });
        return;
      }
    }

    await doSave(xml);
  };

  // 执行保存
  const doSave = async (xml: string) => {
    if (!workflow) return;

    setSaving(true);
    try {
      if (workflow.id === 'new') {
        const tenantId = httpClient.getTenantId() || 1;
        const response = (await WorkflowAPI.createProcessDefinition({
          name: workflow.name,
          description: workflow.description,
          category: workflow.category,
          bpmnXml: xml,
          approval_config: approvalConfig,
          sla_config: workflow.sla_config,
          tenantId,
        })) as any;

        updateWorkflow({
          id: response.key || response.id,
          version: response.version || '1.0.0',
          status: 'draft',
        });

        message.success('工作流创建成功');
      } else {
        const response = (await WorkflowAPI.updateProcessDefinition(
          workflow.id,
          {
            name: workflow.name,
            description: workflow.description,
            category: workflow.category,
            bpmnXml: xml,
            approval_config: approvalConfig,
            sla_config: workflow.sla_config,
          },
          workflow.version
        )) as any;

        updateWorkflow({
          version: response.version || workflow.version,
        });

        message.success('工作流更新成功');
      }

      setCurrentXML(xml);
      setHasChanges(false);
      // 重新加载版本列表
      if (workflow.id !== 'new') {
        loadWorkflowVersions(workflow.id);
      }
    } catch (error) {
      console.error('保存工作流失败:', error);
      message.error('保存工作流失败: ' + (error as Error).message);
    } finally {
      setSaving(false);
    }
  };

  // 部署工作流
  const handleDeploy = async () => {
    if (!workflow || !currentXML) return;

    // 部署前必须校验
    const issues = await validateWorkflow(true);
    const hasErrors = issues.some((i: ValidationIssue) => i.type === 'error');
    if (hasErrors) {
      message.error('流程存在错误，请修复后再部署');
      setShowValidationPanel(true);
      return;
    }

    setDeploying(true);
    try {
      await WorkflowAPI.deployProcessDefinition(workflow.id, workflow.version);
      updateWorkflow({ status: 'active' });
      message.success('工作流部署成功');
    } catch (error) {
      console.error('部署工作流失败:', error);
      message.error('部署工作流失败');
    } finally {
      setDeploying(false);
    }
  };

  // 保存并部署
  const handleSaveAndDeploy = async (xml: string) => {
    if (!workflow) return;

    // 先校验
    const issues = await validateWorkflow(true);
    const hasErrors = issues.some((i: ValidationIssue) => i.type === 'error');
    if (hasErrors) {
      message.error('流程存在错误，请修复后再部署');
      setShowValidationPanel(true);
      return;
    }

    setSaving(true);
    setDeploying(true);
    try {
      const currentVersion = workflow.version || '1.0.0';

      if (workflow.id === 'new') {
        const createData = {
          name: workflow.name,
          description: workflow.description || '',
          category: workflow.category || 'general',
          bpmnXml: xml,
          approval_config: approvalConfig,
          sla_config: workflow.sla_config,
          tenantId: httpClient.getTenantId() || 1,
        };

        const response = (await WorkflowAPI.createProcessDefinition(createData)) as any;

        if (!response) {
          throw new Error('创建工作流失败：服务器返回空响应');
        }

        const newVersion = response.version || '1.0.0';
        const newKey = response.key || response.id;

        if (!newKey) {
          throw new Error('创建工作流失败：未获取到工作流ID');
        }

        await new Promise(resolve => setTimeout(resolve, 500));
        await WorkflowAPI.deployProcessDefinition(newKey, newVersion);

        updateWorkflow({
          id: newKey,
          version: newVersion,
          status: 'active',
        });

        message.success('工作流创建并部署成功');
      } else {
        const updateData = {
          name: workflow.name,
          description: workflow.description || '',
          category: workflow.category || 'general',
          bpmnXml: xml,
          approval_config: approvalConfig,
          sla_config: workflow.sla_config,
        };

        await WorkflowAPI.updateProcessDefinition(workflow.id, updateData, currentVersion);
        await WorkflowAPI.deployProcessDefinition(workflow.id, currentVersion);

        updateWorkflow({ status: 'active' });
        message.success('工作流保存并部署成功');
      }

      setCurrentXML(xml);
      setHasChanges(false);
      // 重新加载版本列表
      if (workflow.id !== 'new') {
        loadWorkflowVersions(workflow.id);
      }
    } catch (error) {
      console.error('保存并部署失败:', error);
      const errorMsg = error instanceof Error ? error.message : String(error);
      message.error('保存并部署失败: ' + errorMsg);
    } finally {
      setSaving(false);
      setDeploying(false);
    }
  };

  // 切换版本
  const handleSwitchVersion = async (versionId: string) => {
    try {
      const version = workflowVersions.find(v => v.id === versionId);
      if (version) {
        setCurrentXML(version.xml);
        updateWorkflow({ version: version.version });
        message.success(`已切换到版本 ${version.version}`);
      }
    } catch (error) {
      console.error('切换版本失败:', error);
      message.error('切换版本失败');
    }
  };

  // 创建版本
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
      updateWorkflow({ version: newVersion.version });

      message.success('新版本创建成功');
      setShowVersionModal(false);
    } catch (error) {
      console.error('创建版本失败:', error);
      message.error('创建版本失败');
    }
  };

  // 对比版本
  const handleCompareVersions = (version1: WorkflowVersion, version2: WorkflowVersion) => {
    setCompareVersions({ version1, version2 });
    setShowVersionCompare(true);
  };

  // 跳转到问题元素
  const jumpToIssue = (issue: ValidationIssue) => {
    if (!issue.elementId) return;
    
    const api = getBpmnDesignerApi();
    if (api) {
      api.selectElement(issue.elementId);
      // 切换到设计器标签
      setActiveTab('designer');
      // 关闭校验面板（可选）
      // setShowValidationPanel(false);
      message.info(`已定位到元素 "${issue.elementName || issue.elementId}"`);
    }
  };

  // Tab 切换
  const handleTabChange = (key: string) => {
    setActiveTab(key);
  };

  // 提供给子组件的值
  const contextValue = useMemo(
    () => ({
      workflow,
      setWorkflow,
      currentXML,
      setCurrentXML,
      hasChanges,
      setHasChanges,
      activeTab,
      setActiveTab,
      saving,
      setSaving,
      deploying,
      setDeploying,
      approvalConfig,
      setApprovalConfig,
      workflowVersions,
      setWorkflowVersions,
      userList,
      setUserList,
      roleList,
      setRoleList,
      groupList,
      setGroupList,
      loadingUsers,
      setLoadingUsers,
      loadingRoles,
      setLoadingRoles,
      loadingGroups,
      setLoadingGroups,
      updateWorkflow,
      updateSLAConfig,
      addWorkflowVersion: () => {},
      // 弹窗状态
      showNewWorkflowModal,
      setShowNewWorkflowModal,
      showVersionModal,
      setShowVersionModal,
      showSettingsModal,
      setShowSettingsModal,
      showMetadataModal,
      setShowMetadataModal,
      metadataForm,
      // 操作
      handleSwitchVersion,
    }),
    [
      workflow,
      currentXML,
      hasChanges,
      activeTab,
      saving,
      deploying,
      approvalConfig,
      workflowVersions,
      userList,
      roleList,
      groupList,
      loadingUsers,
      loadingRoles,
      loadingGroups,
      showNewWorkflowModal,
      showVersionModal,
      showSettingsModal,
      showMetadataModal,
      metadataForm,
    ]
  );

  return (
    <WorkflowDesignerContext.Provider value={contextValue}>
      <Layout className="h-screen">
        {/* 工具栏 */}
        <WorkflowToolbar
          workflow={workflow}
          saving={saving}
          deploying={deploying}
          onSave={handleSave}
          onSaveAndDeploy={handleSaveAndDeploy}
          onDeploy={handleDeploy}
          currentXML={currentXML}
          onValidate={validateWorkflow}
          validationIssues={validationIssues} onAIClick={() => setShowAIModal(true)}
        />

        <Content className="p-4 md:p-6 bg-gray-50 overflow-hidden">
          <Tabs
            activeKey={activeTab}
            onChange={handleTabChange}
            size="small"
            items={[
              {
                key: 'designer',
                label: '流程设计',
                children: (
                  <div className="flex flex-col md:flex-row gap-2 md:gap-4 h-[calc(100vh-220px)] md:h-[calc(100vh-200px)]">
                    <div className="flex-1 min-w-0 min-h-[300px] md:min-h-0">
                      <WorkflowCanvas
                        currentXML={currentXML}
                        onSave={handleSave}
                        onChange={xml => {
                          setCurrentXML(xml);
                          setHasChanges(true);
                        }}
                        onSelectionChange={handleSelectionChange}
                      />
                    </div>
                    <div className="w-full md:w-80 shrink-0 overflow-y-auto bg-white rounded-lg shadow-sm border border-gray-200">
                      <WorkflowNodeInspector
                        selection={selectedNode}
                        onUpdateProperties={handleUpdateNodeProperties}
                        onRefresh={handleRefreshSelection}
                      />
                    </div>
                  </div>
                ),
              },
              {
                key: 'versions',
                label: '版本历史',
                children: (
                  <WorkflowProperties
                    workflow={workflow}
                    approvalConfig={approvalConfig}
                    setApprovalConfig={setApprovalConfig}
                    workflowVersions={workflowVersions}
                    userList={userList}
                    roleList={roleList}
                    groupList={groupList}
                    loadingUsers={loadingUsers}
                    loadingRoles={loadingRoles}
                    loadingGroups={loadingGroups}
                    onSwitchVersion={handleSwitchVersion}
                    onShowVersionModal={() => setShowVersionModal(true)}
                  />
                ),
              },
              {
                key: 'config',
                label: '流程配置',
                children: (
                  <WorkflowProperties
                    workflow={workflow}
                    approvalConfig={approvalConfig}
                    setApprovalConfig={setApprovalConfig}
                    workflowVersions={workflowVersions}
                    userList={userList}
                    roleList={roleList}
                    groupList={groupList}
                    loadingUsers={loadingUsers}
                    loadingRoles={loadingRoles}
                    loadingGroups={loadingGroups}
                    onUpdateSLA={updateSLAConfig}
                  />
                ),
              },
              {
                key: 'validation',
                label: (
                  <span>
                    校验结果
                    {validationIssues.length > 0 && (
                      <Tag color={validationIssues.some(i => i.type === 'error') ? 'error' : 'warning'} className="ml-1">
                        {validationIssues.length}
                      </Tag>
                    )}
                  </span>
                ),
                children: (
                  <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6 h-[calc(100vh-200px)] overflow-y-auto">
                    <div className="mb-4">
                      <Space>
                        <Button 
                          type="primary" 
                          icon={<BugOutlined />} 
                          onClick={() => validateWorkflow(true)}
                          loading={validating}
                        >
                          重新校验
                        </Button>
                        <Switch 
                          checked={autoValidate} 
                          onChange={setAutoValidate} 
                          checkedChildren="自动校验开启" 
                          unCheckedChildren="自动校验关闭"
                        />
                      </Space>
                    </div>

                    {validationIssues.length === 0 ? (
                      <div className="text-center py-12">
                        <CheckCircleOutlined className="text-4xl text-green-500 mb-2" />
                        <Title level={4}>流程校验通过</Title>
                        <Text type="secondary">未发现任何问题，可以正常部署</Text>
                      </div>
                    ) : (
                      <div className="divide-y divide-gray-100">
                        {validationIssues.map(item => (
                          <div
                            key={`${item.elementId || 'process'}-${item.message}`}
                            className="cursor-pointer hover:bg-gray-50 transition-colors"
                            onClick={() => item.elementId && jumpToIssue(item)}
                          >
                            <div className="flex gap-3 px-4 py-3">
                              <div className="pt-1">
                                {
                                item.type === 'error' ? (
                                  <CloseCircleOutlined className="text-red-500 text-xl" />
                                ) : item.type === 'warning' ? (
                                  <WarningOutlined className="text-yellow-500 text-xl" />
                                ) : (
                                  <CheckCircleOutlined className="text-blue-500 text-xl" />
                                )
                                }
                              </div>
                              <div className="min-w-0 flex-1">
                                <Space wrap>
                                  <Text>{item.message}</Text>
                                  {item.elementId && (
                                    <Tag color="blue">
                                      {item.elementType?.replace('bpmn:', '') || '元素'}: {item.elementName || item.elementId}
                                    </Tag>
                                  )}
                                </Space>
                                {item.elementId && (
                                  <div>
                                    <Text type="secondary" className="text-xs">
                                      点击定位到该元素
                                    </Text>
                                  </div>
                                )}
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                ),
              },
            ]}
          />
        </Content>

        {/* 弹窗组件 */}
        <WorkflowNewModal
          visible={showNewWorkflowModal}
          onClose={() => {
            setShowNewWorkflowModal(false);
            if (!workflow) {
              router.push('/workflow');
            }
          }}
          onSelectTemplate={templateWorkflow => {
            setWorkflow(templateWorkflow);
            setCurrentXML(templateWorkflow.xml);
            setShowNewWorkflowModal(false);
          }}
          onCreateCustom={values => {
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
        />

        <WorkflowVersionModal
          visible={showVersionModal}
          onClose={() => setShowVersionModal(false)}
          onCreate={handleCreateVersion}
          workflow={workflow}
        />

        <WorkflowSettingsModal
          visible={showSettingsModal}
          onClose={() => setShowSettingsModal(false)}
          onSave={async () => {
            try {
              const values = await form.validateFields();
              updateWorkflow({
                approval_config: values.approvalConfig,
                sla_config: values.slaConfig,
              });
              message.success('设置保存成功');
              setShowSettingsModal(false);
            } catch (error) {
              console.error('保存设置失败:', error);
            }
          }}
          form={form}
        />

        <WorkflowMetadataModal
          visible={showMetadataModal}
          onClose={() => setShowMetadataModal(false)}
          onSave={values => {
            updateWorkflow({
              name: values.name,
              description: values.description,
              category: values.category,
            });
            setShowMetadataModal(false);
          }}
          form={metadataForm}
        />

        {/* 版本对比弹窗 */}
        <Modal
          title="版本对比"
          open={showVersionCompare}
          onCancel={() => setShowVersionCompare(false)}
          width={900}
          footer={null}
        >
          {compareVersions.version1 && compareVersions.version2 ? (
            <div>
              <div className="mb-4 flex justify-between">
                <Tag color="blue">版本 {compareVersions.version1.version}</Tag>
                <span className="mx-2">VS</span>
                <Tag color="green">版本 {compareVersions.version2.version}</Tag>
              </div>
              <div className="grid grid-cols-2 gap-4 h-[600px]">
                <div className="border border-gray-200 rounded-lg p-4 overflow-y-auto bg-gray-50 font-mono text-xs whitespace-pre-wrap">
                  {compareVersions.version1.xml}
                </div>
                <div className="border border-gray-200 rounded-lg p-4 overflow-y-auto bg-gray-50 font-mono text-xs whitespace-pre-wrap">
                  {compareVersions.version2.xml}
                </div>
              </div>
            </div>
          ) : (
            <div className="text-center py-12">
              <DiffOutlined className="text-4xl text-gray-400 mb-2" />
              <Text type="secondary">请选择两个版本进行对比</Text>
            </div>
          )}
        </Modal>

        {/* AI辅助模态框 */}
        <WorkflowAIModal
          visible={showAIModal}
          onClose={() => setShowAIModal(false)}
          currentXML={currentXML}
          workflowName={workflow?.name}
          onApplyGeneratedProcess={(xml) => {
            setCurrentXML(xml);
            setHasChanges(true);
          }}
        />
      </Layout>
    </WorkflowDesignerContext.Provider>
  );
}

// 主入口组件
export default function WorkflowDesigner({ workflowId }: WorkflowDesignerProps) {
  return <WorkflowDesignerInner workflowId={workflowId} />;
}
