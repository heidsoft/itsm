// 工作流设计器主组件
// Workflow Designer Main Component

'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { Layout, Tabs, Form, message } from 'antd';
import { useRouter, useSearchParams } from 'next/navigation';
import { WorkflowAPI } from '@/lib/api/workflow-api';
import { UserApi } from '@/lib/api/user-api';
import { RoleAPI } from '@/lib/api/role-api';
import { httpClient } from '@/lib/api/http-client';

import { WorkflowDesignerProvider } from './WorkflowContext';
import WorkflowToolbar from './WorkflowToolbar';
import WorkflowCanvas from './WorkflowCanvas';
import WorkflowProperties from './WorkflowProperties';
import WorkflowNewModal from './WorkflowNewModal';
import WorkflowVersionModal from './WorkflowVersionModal';
import WorkflowSettingsModal from './WorkflowSettingsModal';
import WorkflowMetadataModal from './WorkflowMetadataModal';

import type { WorkflowDefinition, WorkflowVersion, ApprovalConfig } from './WorkflowTypes';

const { Content } = Layout;

interface WorkflowDesignerProps {
  workflowId?: string;
  initialXML?: string;
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
    auto_approve_roles: [],
    escalation_rules: [],
  });
  const [workflowVersions, setWorkflowVersions] = useState<WorkflowVersion[]>([]);
  const [userList, setUserList] = useState<{ id: number; name: string; username: string }[]>([]);
  const [roleList, setRoleList] = useState<{ id: number; name: string; code: string }[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [loadingRoles, setLoadingRoles] = useState(false);

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
      if (response.bpmn_xml) {
        try {
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

      const workflowData: WorkflowDefinition = {
        id: response.key || response.id,
        name: response.name || '未命名工作流',
        description: response.description || '',
        version: (response.version || '1').toString(),
        category: response.category || response.type || 'general',
        status: response.is_active ? 'active' : 'inactive',
        xml: xmlContent,
        created_at: response.created_at || new Date().toISOString(),
        updated_at: response.updated_at || new Date().toISOString(),
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

  // 加载工作流配置
  const loadWorkflowConfig = async (key: string) => {
    try {
      const response = (await WorkflowAPI.getProcessDefinition(key)) as any;
      if (response) {
        setApprovalConfig({
          require_approval: response.require_approval ?? true,
          approval_type: response.approval_type || 'sequential',
          approvers: response.approvers || [],
          auto_approve_roles: response.auto_approve_roles || [],
          escalation_rules: response.escalation_rules || [],
        });

        if (response.sla_config && workflow) {
          setWorkflow({
            ...workflow,
            sla_config: response.sla_config,
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
    setWorkflow(prev => prev ? { ...prev, ...updates } : null);
  };

  // 更新 SLA 配置
  const updateSLAConfig = (config: Partial<WorkflowDefinition['sla_config']>) => {
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

  // 保存工作流
  const handleSave = async (xml: string) => {
    if (!workflow) return;

    setSaving(true);
    try {
      if (workflow.id === 'new') {
        const tenantId = httpClient.getTenantId() || 1;
        const response = (await WorkflowAPI.createProcessDefinition({
          name: workflow.name,
          description: workflow.description,
          category: workflow.category,
          bpmn_xml: xml,
          tenant_id: tenantId,
        })) as any;

        updateWorkflow({
          id: response.key || response.id,
          version: response.version || '1.0.0',
          status: 'draft',
        });

        message.success('工作流创建成功');
      } else {
        const response = (await WorkflowAPI.updateProcessDefinition(workflow.id, {
          name: workflow.name,
          description: workflow.description,
          category: workflow.category,
          bpmn_xml: xml,
          approval_config: approvalConfig,
          sla_config: workflow.sla_config,
        }, workflow.version)) as any;

        updateWorkflow({
          version: response.version || workflow.version,
        });

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

  // 部署工作流
  const handleDeploy = async () => {
    if (!workflow || !currentXML) return;

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

    setSaving(true);
    setDeploying(true);
    try {
      const currentVersion = workflow.version || '1.0.0';

      if (workflow.id === 'new') {
        const createData = {
          name: workflow.name,
          description: workflow.description || '',
          category: workflow.category || 'general',
          bpmn_xml: xml,
          tenant_id: httpClient.getTenantId() || 1,
        };

        const response = await WorkflowAPI.createProcessDefinition(createData) as any;

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
          bpmn_xml: xml,
        };

        await WorkflowAPI.updateProcessDefinition(workflow.id, updateData, currentVersion);
        await WorkflowAPI.deployProcessDefinition(workflow.id, currentVersion);

        updateWorkflow({ status: 'active' });
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

  // Tab 切换
  const handleTabChange = (key: string) => {
    setActiveTab(key);
  };

  // 提供给子组件的值
  const contextValue = useMemo(() => ({
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
    loadingUsers,
    setLoadingUsers,
    loadingRoles,
    setLoadingRoles,
    updateWorkflow,
    updateSLAConfig,
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
  }), [
    workflow, currentXML, hasChanges, activeTab, saving, deploying,
    approvalConfig, workflowVersions, userList, roleList, loadingUsers, loadingRoles,
    showNewWorkflowModal, showVersionModal, showSettingsModal, showMetadataModal,
    metadataForm
  ]);

  return (
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
      />

      <Content className="p-6 bg-gray-50">
        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          items={[
            {
              key: 'designer',
              label: '流程设计',
              children: (
                <WorkflowCanvas
                  currentXML={currentXML}
                  onSave={handleSave}
                  onChange={(xml) => {
                    setCurrentXML(xml);
                    setHasChanges(true);
                  }}
                />
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
                  loadingUsers={loadingUsers}
                  loadingRoles={loadingRoles}
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
                  loadingUsers={loadingUsers}
                  loadingRoles={loadingRoles}
                  onUpdateSLA={updateSLAConfig}
                />
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
        onSelectTemplate={(templateWorkflow) => {
          setWorkflow(templateWorkflow);
          setCurrentXML(templateWorkflow.xml);
          setShowNewWorkflowModal(false);
        }}
        onCreateCustom={(values) => {
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
              approval_config: values.approval_config,
              sla_config: values.sla_config,
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
        onSave={(values) => {
          updateWorkflow({
            name: values.name,
            description: values.description,
            category: values.category,
          });
          setShowMetadataModal(false);
        }}
        form={metadataForm}
      />
    </Layout>
  );
}

// 主入口组件
export default function WorkflowDesigner({ workflowId }: WorkflowDesignerProps) {
  return <WorkflowDesignerInner workflowId={workflowId} />;
}
