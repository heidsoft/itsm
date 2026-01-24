'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Modal,
  Form,
  message,
  Tag,
  Space,
  Dropdown,
  Tooltip,
  Row,
  Col,
  Statistic,
  Alert,
  App,
  Skeleton,
} from 'antd';
import {
  Plus,
  Search,
  MoreHorizontal,
  Edit,
  Eye,
  Trash2,
  PlayCircle,
  PauseCircle,
  Copy,
  Download,
  Upload,
  BarChart3,
  RefreshCw,
  FileText,
  GitBranch,
  Clock,
  CheckCircle,
  Code,
} from 'lucide-react';

import BPMNDesigner from '@/components/workflow/BPMNDesigner';
import { WorkflowAPI } from '@/lib/api/workflow-api';
import { WorkflowType } from '@/types/workflow';

import { useRouter } from 'next/navigation';
import { useI18n } from '@/lib/i18n';

const { Option } = Select;

interface Workflow {
  id: number;
  name: string;
  description: string;
  category: string;
  version: string;
  status: 'draft' | 'active' | 'inactive' | 'archived';
  bpmn_xml?: string;
  created_at: string;
  updated_at: string;
  instances_count: number;
  running_instances: number;
  created_by: string;
}

const WorkflowManagementPageSkeleton: React.FC = () => (
  <div>
    <Skeleton active paragraph={{ rows: 4 }} />
    <Skeleton active paragraph={{ rows: 2 }} style={{ marginTop: 24 }} />
    <Skeleton active paragraph={{ rows: 8 }} style={{ marginTop: 24 }} />
  </div>
);

const WorkflowManagementPage = () => {
  const { t } = useI18n();
  const router = useRouter();
  const { modal } = App.useApp();
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [designerVisible, setDesignerVisible] = useState(false);

  const [editingWorkflow, setEditingWorkflow] = useState<Workflow | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [filters, setFilters] = useState({
    status: '',
    category: '',
    keyword: '',
  });
  const [stats, setStats] = useState({
    total: 0,
    active: 0,
    running: 0,
    completed: 0,
    draft: 0,
    inactive: 0,
    todayInstances: 0,
    avgExecutionTime: 0,
  });

  // 模拟数据 - 使用useMemo避免每次渲染时重新创建
  const mockWorkflows = useMemo<Workflow[]>(
    () => [
      {
        id: 1,
        name: t('workflow.ticketApprovalProcess'),
        description: t('workflow.ticketApprovalDescription'),
        category: t('workflow.approvalProcess'),
        version: '1.0.0',
        status: 'active',
        bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
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
    <bpmn:userTask id="UserTask_1" name="提交工单">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_2" name="审核工单">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_3" name="处理工单">
      <bpmn:incoming>Flow_3</bpmn:incoming>
      <bpmn:outgoing>Flow_4</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_1" name="结束">
      <bpmn:incoming>Flow_4</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="UserTask_1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="UserTask_1" targetRef="UserTask_2" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="UserTask_2" targetRef="UserTask_3" />
    <bpmn:sequenceFlow id="Flow_4" sourceRef="UserTask_3" targetRef="EndEvent_1" />
  </bpmn:process>
</bpmn:definitions>`,
        created_at: '2024-01-15T10:30:00Z',
        updated_at: '2024-01-15T14:20:00Z',
        instances_count: 156,
        running_instances: 23,
        created_by: '张三',
      },
      {
        id: 2,
        name: t('workflow.incidentHandlingProcess'),
        description: t('workflow.incidentHandlingDescription'),
        category: t('workflow.incidentHandling'),
        version: '2.1.0',
        status: 'active',
        bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_3" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_3" isExecutable="true">
    <bpmn:startEvent id="StartEvent_3" name="事件报告">
      <bpmn:outgoing>Flow_11</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="UserTask_7" name="事件分类">
      <bpmn:incoming>Flow_11</bpmn:incoming>
      <bpmn:outgoing>Flow_12</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:exclusiveGateway id="Gateway_2" name="优先级判断">
      <bpmn:incoming>Flow_12</bpmn:incoming>
      <bpmn:outgoing>Flow_13</bpmn:outgoing>
      <bpmn:outgoing>Flow_14</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:userTask id="UserTask_8" name="紧急处理">
      <bpmn:incoming>Flow_13</bpmn:incoming>
      <bpmn:outgoing>Flow_15</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_9" name="常规处理">
      <bpmn:incoming>Flow_14</bpmn:incoming>
      <bpmn:outgoing>Flow_16</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_3" name="事件解决">
      <bpmn:incoming>Flow_15</bpmn:incoming>
      <bpmn:incoming>Flow_16</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_11" sourceRef="StartEvent_3" targetRef="UserTask_7" />
    <bpmn:sequenceFlow id="Flow_12" sourceRef="UserTask_7" targetRef="Gateway_2" />
    <bpmn:sequenceFlow id="Flow_13" sourceRef="Gateway_2" targetRef="UserTask_8" />
    <bpmn:sequenceFlow id="Flow_14" sourceRef="Gateway_2" targetRef="UserTask_9" />
    <bpmn:sequenceFlow id="Flow_15" sourceRef="UserTask_8" targetRef="EndEvent_3" />
    <bpmn:sequenceFlow id="Flow_16" sourceRef="UserTask_9" targetRef="EndEvent_3" />
  </bpmn:process>
</bpmn:definitions>`,
        created_at: '2024-01-14T09:15:00Z',
        updated_at: '2024-01-15T11:45:00Z',
        instances_count: 89,
        running_instances: 12,
        created_by: '李四',
      },
      {
        id: 3,
        name: t('workflow.changeManagementProcess'),
        description: t('workflow.changeManagementDescription'),
        category: t('workflow.changeManagement'),
        version: '1.5.0',
        status: 'draft',
        bpmn_xml: `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                   xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" 
                   xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" 
                   xmlns:di="http://www.omg.org/spec/DD/20100524/DI" 
                   id="Definitions_4" 
                   targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_4" isExecutable="true">
    <bpmn:startEvent id="StartEvent_4" name="变更申请">
      <bpmn:outgoing>Flow_17</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="UserTask_10" name="变更评估">
      <bpmn:incoming>Flow_17</bpmn:incoming>
      <bpmn:outgoing>Flow_18</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_11" name="变更审批">
      <bpmn:incoming>Flow_18</bpmn:incoming>
      <bpmn:outgoing>Flow_19</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_12" name="变更实施">
      <bpmn:incoming>Flow_19</bpmn:incoming>
      <bpmn:outgoing>Flow_20</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:userTask id="UserTask_13" name="变更验证">
      <bpmn:incoming>Flow_20</bpmn:incoming>
      <bpmn:outgoing>Flow_21</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="EndEvent_4" name="变更完成">
      <bpmn:incoming>Flow_21</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_17" sourceRef="StartEvent_4" targetRef="UserTask_10" />
    <bpmn:sequenceFlow id="Flow_18" sourceRef="UserTask_10" targetRef="UserTask_11" />
    <bpmn:sequenceFlow id="Flow_19" sourceRef="UserTask_11" targetRef="UserTask_12" />
    <bpmn:sequenceFlow id="Flow_20" sourceRef="UserTask_12" targetRef="UserTask_13" />
    <bpmn:sequenceFlow id="Flow_21" sourceRef="UserTask_13" targetRef="EndEvent_4" />
  </bpmn:process>
</bpmn:definitions>`,
        created_at: '2024-01-13T16:20:00Z',
        updated_at: '2024-01-14T10:30:00Z',
        instances_count: 45,
        running_instances: 8,
        created_by: '王五',
      },
    ],
    [t]
  );

  const loadWorkflows = useCallback(async () => {
    setLoading(true);
    try {
      const response = await WorkflowAPI.getWorkflows({
        page: 1,
        pageSize: 100,
      });
      // 适配后端返回格式
      const adaptedWorkflows: Workflow[] = (response.workflows || []).map((w: any) => ({
        id: w.id || 0,
        name: w.name || w.code || 'Unknown',
        description: w.description || '',
        category: w.category || 'general',
        version: w.version || '1.0.0',
        status: (w.status === 'active' || w.deployed) ? 'active' : 'draft',
        bpmn_xml: w.bpmn_xml || w.xml || '',
        created_at: w.createdAt || w.created_at || new Date().toISOString(),
        updated_at: w.updatedAt || w.updated_at || new Date().toISOString(),
        instances_count: w.instances_count || 0,
        running_instances: w.running_instances || 0,
        created_by: w.createdBy || w.created_by || 'System',
      }));
      setWorkflows(adaptedWorkflows);
      // 基于实际数据计算统计
      setStats(prev => ({
        ...prev,
        total: adaptedWorkflows.length,
        active: adaptedWorkflows.filter(w => w.status === 'active').length,
        draft: adaptedWorkflows.filter(w => w.status === 'draft').length,
        inactive: adaptedWorkflows.filter(w => w.status === 'inactive').length,
        running: adaptedWorkflows.reduce((sum, w) => sum + w.running_instances, 0),
        completed: adaptedWorkflows.reduce(
          (sum, w) => sum + (w.instances_count - w.running_instances),
          0
        ),
      }));
    } catch {
      // 如果 API 失败，使用模拟数据作为降级
      console.warn('Failed to load workflows, using mock data');
      setWorkflows(mockWorkflows);
      message.error(t('workflow.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [mockWorkflows, t]);

  const loadStats = useCallback(async () => {
    // 统计已经在 loadWorkflows 中一起计算了
    // 这里保持空实现作为占位符
  }, []);

  useEffect(() => {
    loadWorkflows();
    loadStats();
  }, [loadWorkflows, loadStats]);

  // 当filters变化时重新加载数据
  useEffect(() => {
    loadWorkflows();
  }, [filters, loadWorkflows]);

  const handleCreateWorkflow = () => {
    setEditingWorkflow(null);
    setModalVisible(true);
  };

  const handleEditWorkflow = (workflow: Workflow) => {
    setEditingWorkflow(workflow);
    setModalVisible(true);
  };

  const handleDesignWorkflow = (workflow: Workflow) => {
    setEditingWorkflow(workflow);

    // 根据工作流类型选择对应的设计器
    if (workflow.category === 'ticket_approval' || workflow.category === t('workflow.approvalProcess') || workflow.name.includes(t('workflow.approvalProcess'))) {
      // 工单审批流程使用专门的设计器
      router.push(`/workflow/ticket-approval?id=${workflow.id}`);
    } else {
      // 通用工作流使用BPMN设计器
      setDesignerVisible(true);
    }
  };

  const handleViewBPMN = (workflow: Workflow) => {
    if (!workflow.bpmn_xml) {
      message.warning(t('workflow.noBPMNDefinition'));
      return;
    }

    // 显示BPMN XML内容
    Modal.info({
      title: `${workflow.name} - ${t('workflow.bpmnXml')}`,
      width: 800,
      content: (
        <div style={{ maxHeight: '500px', overflow: 'auto' }}>
          <pre
            style={{
              background: '#f5f5f5',
              padding: '16px',
              borderRadius: '4px',
              fontSize: '12px',
              lineHeight: '1.4',
            }}
          >
            {workflow.bpmn_xml}
          </pre>
        </div>
      ),
      okText: t('workflow.close'),
    });
  };

  const handleDeleteWorkflow = async (id: number) => {
    modal.confirm({
      title: t('workflow.confirmDelete'),
      content: t('workflow.deleteConfirmation'),
      okText: t('workflow.confirmDelete'),
      okType: 'danger',
      cancelText: t('workflow.cancel'),
      onOk: async () => {
        try {
          // 模拟删除API调用
          await new Promise(resolve => setTimeout(resolve, 1000));
          setWorkflows(prev => prev.filter(w => w.id !== id));
          message.success(t('workflow.deleteSuccess'));
          loadStats();
        } catch {
          message.error(t('workflow.deleteFailed'));
        }
      },
    });
  };

  const handleDeployWorkflow = async (id: number) => {
    modal.confirm({
      title: t('workflow.confirmDeploy'),
      content: t('workflow.deployConfirmation'),
      okText: t('workflow.confirmDeploy'),
      cancelText: t('workflow.cancel'),
      onOk: async () => {
        try {
          // 模拟部署API调用
          await new Promise(resolve => setTimeout(resolve, 1000));
          setWorkflows(prev =>
            prev.map(w => (w.id === id ? { ...w, status: 'active' as const } : w))
          );
          message.success(t('workflow.deploySuccess'));
          loadStats();
        } catch {
          message.error(t('workflow.deployFailed'));
        }
      },
    });
  };

  const handleCopyWorkflow = async (workflow: Workflow) => {
    try {
      // 模拟复制API调用
      await new Promise(resolve => setTimeout(resolve, 500));
      const newWorkflow: Workflow = {
        ...workflow,
        id: Date.now(), // 临时ID
        name: `${workflow.name} - ${t('workflow.copySuffix')}`,
        status: 'draft',
        version: '1.0.0',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        instances_count: 0,
        running_instances: 0,
        created_by: t('workflow.currentUser'),
      };
      setWorkflows(prev => [newWorkflow, ...prev]);
      message.success(t('workflow.copySuccess', { name: workflow.name }));
      loadStats();
    } catch {
      message.error(t('workflow.copyFailed'));
    }
  };

  const handleExportWorkflow = async (workflow: Workflow) => {
    try {
      // 模拟导出API调用
      await new Promise(resolve => setTimeout(resolve, 500));

      const exportData = {
        workflow: workflow,
        exportTime: new Date().toISOString(),
        version: '1.0',
      };

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: 'application/json',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `workflow_batch_export_${new Date().toISOString().split('T')[0]}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      message.success(t('workflow.batchExportSuccess', { count: selectedRowKeys.length }));
      setSelectedRowKeys([]);
    } catch {
      message.error(t('workflow.batchExportFailed'));
    }
  };

  const handleImportWorkflow = () => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json';
    input.onchange = async e => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file) return;

      try {
        const text = await file.text();
        const importData = JSON.parse(text);

        if (!importData.workflow) {
          throw new Error(t('workflow.invalidWorkflowFile'));
        }

        const newWorkflow: Workflow = {
          ...importData.workflow,
          id: Date.now(),
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          status: 'draft',
          instances_count: 0,
          running_instances: 0,
          created_by: t('workflow.currentUser'),
        };

        setWorkflows(prev => [newWorkflow, ...prev]);
        message.success(t('workflow.importSuccess', { name: newWorkflow.name }));
        loadStats();
      } catch (error) {
        message.error(t('workflow.importFailed') + (error as Error).message);
      }
    };
    input.click();
  };

  const handleStopWorkflow = async (id: number) => {
    modal.confirm({
      title: t('workflow.confirmDeactivate'),
      content: t('workflow.deactivateConfirmation'),
      okText: t('workflow.confirmDeactivate'),
      okType: 'danger',
      cancelText: t('workflow.cancel'),
      onOk: async () => {
        try {
          await new Promise(resolve => setTimeout(resolve, 1000));
          setWorkflows(prev =>
            prev.map(w => (w.id === id ? { ...w, status: 'inactive' as const } : w))
          );
          message.success(t('workflow.deactivateSuccess'));
          loadStats();
        } catch {
          message.error(t('workflow.deactivateFailed'));
        }
      },
    });
  };

  const handleActivateWorkflow = async (id: number) => {
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      setWorkflows(prev => prev.map(w => (w.id === id ? { ...w, status: 'active' as const } : w)));
      message.success(t('workflow.activateSuccess'));
      loadStats();
    } catch {
      message.error(t('workflow.activateFailed'));
    }
  };

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('workflow.selectWorkflowsToDelete'));
      return;
    }

    modal.confirm({
      title: t('workflow.batchDeleteConfirmation'),
      content: t('workflow.batchDeleteContent', { count: selectedRowKeys.length }),
      okText: t('workflow.batchDeleteOk'),
      okType: 'danger',
      cancelText: t('workflow.cancel'),
      onOk: async () => {
        try {
          await new Promise(resolve => setTimeout(resolve, 1000));
          setWorkflows(prev => prev.filter(w => !selectedRowKeys.includes(w.id)));
          setSelectedRowKeys([]);
          message.success(t('workflow.batchDeleteSuccess', { count: selectedRowKeys.length }));
          loadStats();
        } catch {
          message.error(t('workflow.batchDeleteFailed'));
        }
      },
    });
  };

  const handleBatchActivate = () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('workflow.selectWorkflowsToActivate'));
      return;
    }

    modal.confirm({
      title: t('workflow.batchActivateConfirmation'),
      content: t('workflow.batchActivateContent', { count: selectedRowKeys.length }),
      okText: t('workflow.batchActivateOk'),
      cancelText: t('workflow.cancel'),
      onOk: async () => {
        try {
          await new Promise(resolve => setTimeout(resolve, 1000));
          setWorkflows(prev =>
            prev.map(w =>
              selectedRowKeys.includes(w.id) ? { ...w, status: 'active' as const } : w
            )
          );
          setSelectedRowKeys([]);
          message.success(t('workflow.batchActivateSuccess', { count: selectedRowKeys.length }));
          loadStats();
        } catch {
          message.error(t('workflow.batchActivateFailed'));
        }
      },
    });
  };

  const handleBatchStop = () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('workflow.selectWorkflowsToDeactivate'));
      return;
    }

    modal.confirm({
      title: t('workflow.batchDeactivateConfirmation'),
      content: t('workflow.batchDeactivateContent', { count: selectedRowKeys.length }),
      okText: t('workflow.batchDeactivateOk'),
      okType: 'danger',
      cancelText: t('workflow.cancel'),
      onOk: async () => {
        try {
          await new Promise(resolve => setTimeout(resolve, 1000));
          setWorkflows(prev =>
            prev.map(w =>
              selectedRowKeys.includes(w.id) ? { ...w, status: 'inactive' as const } : w
            )
          );
          setSelectedRowKeys([]);
          message.success(t('workflow.batchDeactivateSuccess', { count: selectedRowKeys.length }));
          loadStats();
        } catch {
          message.error(t('workflow.batchDeactivateFailed'));
        }
      },
    });
  };

  const handleBatchExport = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('workflow.selectWorkflowsToExport'));
      return;
    }

    try {
      const selectedWorkflows = workflows.filter(w => selectedRowKeys.includes(w.id));
      const exportData = {
        workflows: selectedWorkflows,
        exportTime: new Date().toISOString(),
        version: '1.0',
        count: selectedWorkflows.length,
      };

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: 'application/json',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `workflow_batch_export_${new Date().toISOString().split('T')[0]}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      message.success(t('workflow.batchExportSuccess', { count: selectedRowKeys.length }));
      setSelectedRowKeys([]);
    } catch {
      message.error(t('workflow.batchExportFailed'));
    }
  };

  const getStatusColor = (status: string) => {
    const colors = {
      draft: 'orange',
      active: 'green',
      inactive: 'gray',
      archived: 'red',
    };
    return colors[status as keyof typeof colors] || 'default';
  };

  const getStatusText = (status: string) => {
    const texts = {
      draft: t('workflow.draft'),
      active: t('workflow.activated'),
      inactive: t('workflow.deactivated'),
      archived: t('workflow.archived'),
    };
    return texts[status as keyof typeof texts] || status;
  };

  const columns = [
    {
      title: t('workflow.name'),
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: Workflow) => (
        <div>
          <div className='font-medium text-gray-900'>{name}</div>
          <div className='text-sm text-gray-500'>{record.description}</div>
        </div>
      ),
    },
    {
      title: t('workflow.category'),
      dataIndex: 'category',
      key: 'category',
      width: 120,
      render: (category: string) => <Tag color='blue'>{category}</Tag>,
    },
    {
      title: t('workflow.version'),
      dataIndex: 'version',
      key: 'version',
      width: 100,
      render: (version: string) => <span className='font-mono text-sm'>{version}</span>,
    },
    {
      title: t('workflow.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>,
    },
    {
      title: t('workflow.bpmnStatus'),
      key: 'bpmn_status',
      width: 120,
      render: (record: Workflow) => (
        <Tag color={record.bpmn_xml ? 'green' : 'orange'}>
          {record.bpmn_xml ? t('workflow.defined') : t('workflow.undefined')}
        </Tag>
      ),
    },
    {
      title: t('workflow.instanceStats'),
      key: 'instances',
      width: 150,
      render: (record: Workflow) => (
        <div className='text-sm'>
          <div>{t('workflow.total')}: {record.instances_count}</div>
          <div>{t('workflow.running')}: {record.running_instances}</div>
        </div>
      ),
    },
    {
      title: t('workflow.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (date: string) => (
        <div className='text-sm'>{new Date(date).toLocaleDateString()}</div>
      ),
    },
    {
      title: t('workflow.actions'),
      key: 'action',
      width: 200,
      render: (record: Workflow) => (
        <Space>
          <Tooltip title={t('workflow.designWorkflow')}>
            <Button
              type='text'
              icon={<GitBranch className='w-4 h-4' />}
              onClick={() => handleDesignWorkflow(record)}
            />
          </Tooltip>
          <Tooltip title={t('workflow.viewDetails')}>
            <Button
              type='text'
              icon={<Eye className='w-4 h-4' />}
              onClick={() => window.open(`/workflow/${record.id}`, '_blank')}
            />
          </Tooltip>
          <Dropdown
            menu={{
              items: [
                {
                  key: 'edit',
                  label: t('workflow.edit'),
                  icon: <Edit className='w-4 h-4' />,
                  onClick: () => handleEditWorkflow(record),
                },
                {
                  key: 'deploy',
                  label: record.status === 'draft' ? t('workflow.deploy') : t('workflow.redeploy'),
                  icon: <PlayCircle className='w-4 h-4' />,
                  onClick: () => handleDeployWorkflow(record.id),
                  disabled: record.status === 'active',
                },
                {
                  key: 'activate',
                  label: t('workflow.activate'),
                  icon: <CheckCircle className='w-4 h-4' />,
                  onClick: () => handleActivateWorkflow(record.id),
                  style: {
                    display: record.status === 'inactive' ? 'block' : 'none',
                  },
                },
                {
                  key: 'stop',
                  label: t('workflow.deactivate'),
                  icon: <PauseCircle className='w-4 h-4' />,
                  onClick: () => handleStopWorkflow(record.id),
                  style: {
                    display: record.status === 'active' ? 'block' : 'none',
                  },
                },
                {
                  type: 'divider' as const,
                },
                {
                  key: 'copy',
                  label: t('workflow.copy'),
                  icon: <Copy className='w-4 h-4' />,
                  onClick: () => handleCopyWorkflow(record),
                },
                {
                  key: 'export',
                  label: t('workflow.export'),
                  icon: <Download className='w-4 h-4' />,
                  onClick: () => handleExportWorkflow(record),
                },
                {
                  key: 'instances',
                  label: t('workflow.viewInstances'),
                  icon: <BarChart3 className='w-4 h-4' />,
                  onClick: () =>
                    window.open(`/workflow/instances?workflowId=${record.id}`, '_blank'),
                },
                {
                  key: 'versions',
                  label: t('workflow.versionManagement'),
                  icon: <GitBranch className='w-4 h-4' />,
                  onClick: () =>
                    window.open(`/workflow/versions?workflowId=${record.id}`, '_blank'),
                },
                {
                  key: 'viewBPMN',
                  label: t('workflow.viewBPMN'),
                  icon: <Code className='w-4 h-4' />,
                  onClick: () => handleViewBPMN(record),
                  disabled: !record.bpmn_xml,
                },
                {
                  type: 'divider' as const,
                },
                {
                  key: 'delete',
                  label: t('workflow.delete'),
                  icon: <Trash2 className='w-4 h-4' />,
                  danger: true,
                  onClick: () => handleDeleteWorkflow(record.id),
                  disabled: record.status === 'active' && record.running_instances > 0,
                },
              ].filter(item => item.style?.display !== 'none'), // 过滤隐藏项目
            }}
          >
            <Button type='text' icon={<MoreHorizontal className='w-4 h-4' />} />
          </Dropdown>
        </Space>
      ),
    },
  ];

  const filteredWorkflows = workflows.filter(workflow => {
    if (filters.status && workflow.status !== filters.status) return false;
    if (filters.category && workflow.category !== filters.category) return false;
    if (filters.keyword && !workflow.name.toLowerCase().includes(filters.keyword.toLowerCase()))
      return false;
    return true;
  });

  return (
    <>
      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className='mb-6'>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title={t('workflow.totalWorkflows')}
              value={stats.total}
              prefix={<FileText className='w-5 h-5' />}
              valueStyle={{ color: '#1890ff' }}
            />
            <div className='mt-2 text-xs text-gray-500'>
              {t('workflow.active')} {stats.active} | {t('workflow.draft')} {stats.draft} |{' '}
              {t('workflow.inactive')} {stats.inactive}
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title={t('workflow.runningInstances')}
              value={stats.running}
              prefix={<Clock className='w-5 h-5' />}
              valueStyle={{ color: '#faad14' }}
            />
            <div className='mt-2 text-xs text-gray-500'>
              {t('workflow.todayNewInstances', { count: stats.todayInstances })}
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title={t('workflow.completedInstances')}
              value={stats.completed}
              prefix={<CheckCircle className='w-5 h-5' />}
              valueStyle={{ color: '#52c41a' }}
            />
            <div className='mt-2 text-xs text-gray-500'>
              {t('workflow.completionRate')}{' '}
              {stats.total > 0
                ? Math.round((stats.completed / (stats.completed + stats.running)) * 100)
                : 0}
              %
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title={t('workflow.avgExecutionTime')}
              value={stats.avgExecutionTime}
              suffix={t('workflow.minutes')}
              prefix={<BarChart3 className='w-5 h-5' />}
              valueStyle={{ color: '#722ed1' }}
            />
            <div className='mt-2 text-xs text-gray-500'>
              {stats.avgExecutionTime < 60 ? t('workflow.goodEfficiency') : t('workflow.optimizableSpace')}
            </div>
          </Card>
        </Col>
      </Row>

      {/* 工具栏 */}
      <Card className='enterprise-card mb-6'>
        <Row gutter={[16, 16]} align='middle'>
          <Col xs={24} sm={12} md={8}>
            <Input
              placeholder={t('workflow.searchPlaceholder')}
              prefix={<Search className='w-4 h-4' />}
              value={filters.keyword}
              onChange={e => setFilters(prev => ({ ...prev, keyword: e.target.value }))}
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder={t('workflow.statusFilter')}
              value={filters.status}
              onChange={value => setFilters(prev => ({ ...prev, status: value }))}
              allowClear
              style={{ width: '100%' }}
            >
              <Option value='draft'>{t('workflow.draft')}</Option>
              <Option value='active'>{t('workflow.activated')}</Option>
              <Option value='inactive'>{t('workflow.deactivated')}</Option>
              <Option value='archived'>{t('workflow.archived')}</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder={t('workflow.categoryFilter')}
              value={filters.category}
              onChange={value => setFilters(prev => ({ ...prev, category: value }))}
              allowClear
              style={{ width: '100%' }}
            >
              <Option value={t('workflow.approvalProcess')}>{t('workflow.approvalProcess')}</Option>
              <Option value={t('workflow.incidentHandling')}>{t('workflow.incidentHandling')}</Option>
              <Option value={t('workflow.changeManagement')}>{t('workflow.changeManagement')}</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={8}>
            <Space>
              <Dropdown
                menu={{
                  items: [
                    {
                      key: 'ticket_approval',
                      label: t('workflow.ticketApprovalProcess'),
                      icon: <GitBranch className='w-4 h-4' />,
                      onClick: () => router.push('/workflow/ticket-approval'),
                    },
                    {
                      key: 'general',
                      label: t('workflow.generalWorkflow'),
                      icon: <GitBranch className='w-4 h-4' />,
                      onClick: handleCreateWorkflow,
                    },
                    {
                      key: 'bpmn',
                      label: t('workflow.bpmnWorkflow'),
                      icon: <Code className='w-4 h-4' />,
                      onClick: () => {
                        setEditingWorkflow(null);
                        setDesignerVisible(true);
                      },
                    },
                  ],
                }}
                trigger={['click']}
              >
                <Button type='primary' icon={<Plus className='w-4 h-4' />}>
                  {t('workflow.newWorkflow')} <MoreHorizontal className='w-3 h-3 ml-1' />
                </Button>
              </Dropdown>
              <Button icon={<Upload className='w-4 h-4' />} onClick={handleImportWorkflow}>
                {t('workflow.import')}
              </Button>
              <Button icon={<RefreshCw className='w-4 h-4' />} onClick={loadWorkflows}>
                {t('workflow.refresh')}
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 批量操作工具栏 */}
      {selectedRowKeys.length > 0 && (
        <Card className='enterprise-card mb-4'>
          <Alert
            message={
              <Space>
                <span>{t('workflow.selectedWorkflows', { count: selectedRowKeys.length })}</span>
                <Button size='small' onClick={() => setSelectedRowKeys([])}>
                  {t('workflow.cancelSelection')}
                </Button>
              </Space>
            }
            type='info'
            action={
              <Space>
                <Button size='small' type='primary' onClick={handleBatchActivate}>
                  {t('workflow.batchActivate')}
                </Button>
                <Button size='small' onClick={handleBatchStop}>
                  {t('workflow.batchDeactivate')}
                </Button>
                <Button size='small' onClick={handleBatchExport}>
                  {t('workflow.batchExport')}
                </Button>
                <Button size='small' danger onClick={handleBatchDelete}>
                  {t('workflow.batchDelete')}
                </Button>
              </Space>
            }
          />
        </Card>
      )}

      {/* 工作流表格 */}
      <Card className='enterprise-card'>
        <Table
          columns={columns}
          dataSource={filteredWorkflows}
          rowKey='id'
          loading={loading}
          rowSelection={{
            selectedRowKeys,
            onChange: (selectedRowKeys: React.Key[]) => {
              setSelectedRowKeys(selectedRowKeys.map(key => Number(key)));
            },
            selections: [
              Table.SELECTION_ALL,
              Table.SELECTION_INVERT,
              Table.SELECTION_NONE,
              {
                key: 'select-active',
                text: t('workflow.selectActive'),
                onSelect: () => {
                  const activeKeys = filteredWorkflows
                    .filter(w => w.status === 'active')
                    .map(w => w.id);
                  setSelectedRowKeys(activeKeys);
                },
              },
              {
                key: 'select-draft',
                text: t('workflow.selectDraft'),
                onSelect: () => {
                  const draftKeys = filteredWorkflows
                    .filter(w => w.status === 'draft')
                    .map(w => w.id);
                  setSelectedRowKeys(draftKeys);
                },
              },
            ],
          }}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => t('workflow.showTotal', { start: range[0], end: range[1], total }),
          }}
          scroll={{ x: 1200 }}
        />
      </Card>

      {/* 创建/编辑工作流模态框 */}
      <Modal
        title={editingWorkflow ? t('workflow.editWorkflow') : t('workflow.newWorkflow')}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
        destroyOnHidden
      >
        <Form
          layout='vertical'
          initialValues={editingWorkflow || {}}
          onFinish={async () => {
            try {
              message.success(editingWorkflow ? t('workflow.updateSuccess') : t('workflow.createWorkflowSuccess'));
              setModalVisible(false);
              loadWorkflows();
            } catch {
              message.error(t('workflow.operationFailed'));
            }
          }}
        >
          <Form.Item
            name='name'
            label={t('workflow.workflowName')}
            rules={[{ required: true, message: t('workflow.workflowNameRequired') }]}
          >
            <Input placeholder={t('workflow.workflowNamePlaceholder')} />
          </Form.Item>

          <Form.Item name='description' label={t('workflow.workflowDescription')}>
            <Input.TextArea rows={3} placeholder={t('workflow.workflowDescriptionPlaceholder')} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='category'
                label={t('workflow.category')}
                rules={[{ required: true, message: t('workflow.categoryRequired') }]}
              >
                <Select placeholder={t('workflow.categoryPlaceholder')}>
                  <Option value={t('workflow.approvalProcess')}>{t('workflow.approvalProcess')}</Option>
                  <Option value={t('workflow.incidentHandling')}>{t('workflow.incidentHandling')}</Option>
                  <Option value={t('workflow.changeManagement')}>{t('workflow.changeManagement')}</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='status' label={t('workflow.status')}>
                <Select placeholder={t('workflow.statusPlaceholder')}>
                  <Option value='draft'>{t('workflow.draft')}</Option>
                  <Option value='active'>{t('workflow.activated')}</Option>
                  <Option value='inactive'>{t('workflow.deactivated')}</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item>
            <Space>
              <Button type='primary' htmlType='submit'>
                {editingWorkflow ? t('workflow.update') : t('workflow.create')}
              </Button>
              <Button onClick={() => setModalVisible(false)}>{t('workflow.cancel')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 工作流设计器 */}
      <Modal
        title={`${t('workflow.bpmnWorkflowDesigner')} - ${editingWorkflow?.name || t('workflow.newWorkflow')}`}
        open={designerVisible}
        onCancel={() => setDesignerVisible(false)}
        footer={null}
        width='95%'
        style={{ top: 10 }}
        destroyOnHidden
      >
        <BPMNDesigner
          xml={editingWorkflow?.bpmn_xml || ''}
          onSave={async (xml: string) => {
            try {
              if (editingWorkflow) {
                // 更新现有工作流
                await WorkflowAPI.updateWorkflow(String(editingWorkflow.id), {
                  bpmn_xml: xml,
                } as any);
                message.success(t('workflow.saveSuccess'));
              } else {
                // 创建新工作流
                await WorkflowAPI.createWorkflow({
                  name: t('workflow.newBPMNWorkflow'),
                  description: t('workflow.bpmnWorkflowDescription'),
                  code: `workflow_${Date.now()}`,
                  type: WorkflowType.APPROVAL,
                  bpmn_xml: xml,
                });
                message.success(t('workflow.createWorkflowSuccess'));
              }
              setDesignerVisible(false);
              loadWorkflows();
            } catch (error) {
              message.error(t('workflow.saveFailed') + (error as Error).message);
            }
          }}
          onDeploy={async (xml: string) => {
            try {
              if (editingWorkflow) {
                // 先保存BPMN XML
                await WorkflowAPI.updateWorkflow(String(editingWorkflow.id), {
                  bpmn_xml: xml,
                } as any);
                // 然后部署工作流
                await WorkflowAPI.deployWorkflow(String(editingWorkflow.id));
                message.success(t('workflow.deploySuccess'));
              } else {
                // 创建并部署新工作流
                const newWorkflow = await WorkflowAPI.createWorkflow({
                  name: t('workflow.newBPMNWorkflow'),
                  description: t('workflow.bpmnWorkflowDescription'),
                  code: `workflow_${Date.now()}`,
                  type: WorkflowType.APPROVAL,
                  bpmn_xml: xml,
                });
                await WorkflowAPI.deployWorkflow(String(newWorkflow.id));
                message.success(t('workflow.createAndDeploySuccess'));
              }
              setDesignerVisible(false);
              loadWorkflows();
            } catch (error) {
              message.error(t('workflow.deployFailed') + (error as Error).message);
            }
          }}
          height={700}
        />
      </Modal>
    </>
  );
};

export default WorkflowManagementPage;
