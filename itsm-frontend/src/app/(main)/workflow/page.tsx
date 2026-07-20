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
  Typography,
  Descriptions,
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
  Activity,
  ShieldCheck,
  Workflow,
} from 'lucide-react';

import BPMNDesigner from '@/components/workflow/BPMNDesigner';
import { FilterToolbarCard } from '@/components/ui/FilterToolbarCard';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview } from '@/components/ui/StatsOverview';
import { WorkflowAPI } from '@/lib/api/workflow-api';
import { WorkflowType, CreateWorkflowRequest, UpdateWorkflowRequest } from '@/types/workflow';
import { BatchActionBar } from '@/components/business/BatchActionBar';

import { useRouter } from 'next/navigation';
import { useI18n } from '@/lib/i18n';

const { Option } = Select;
const { Text } = Typography;

interface Workflow {
  id: string;
  name: string;
  description: string;
  category: string;
  version: string;
  status: 'draft' | 'active' | 'inactive' | 'archived';
  bpmnXml?: string;
  createdAt: string;
  updatedAt: string;
  instancesCount: number;
  runningInstances: number;
  createdBy: string;
}

const WorkflowManagementPage = () => {
  const { t } = useI18n();
  const router = useRouter();
  const { modal } = App.useApp();
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [designerVisible, setDesignerVisible] = useState(false);
  const [viewingWorkflow, setViewingWorkflow] = useState<Workflow | null>(null);

  const [editingWorkflow, setEditingWorkflow] = useState<Workflow | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
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

  // 分页状态
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });

  // 分页加载工作流列表
  const loadWorkflows = useCallback(
    async (page: number = 1, pageSize: number = 10) => {
      setLoading(true);
      try {
        const response = await WorkflowAPI.getWorkflows({
          page,
          pageSize,
        });
        // 适配后端返回格式
        const adaptedWorkflows: Workflow[] = (response.workflows || []).map((w: any) => ({
          id: String(w.key || w.code || w.id || ''), // 使用流程 key 作为 id，因为后端 API 需要 key
          name: w.name || w.code || 'Unknown',
          description: w.description || '',
          category: w.category || 'general',
          version: w.version || '1.0.0',
          status: w.status === 'active' || w.deployed ? 'active' : 'draft',
          bpmnXml: w.bpmnXml || w.xml || '',
          createdAt: w.createdAt || new Date().toISOString(),
          updatedAt: w.updatedAt || new Date().toISOString(),
          instancesCount: w.instancesCount || 0,
          runningInstances: w.runningInstances || 0,
          createdBy: w.createdBy || 'System',
        }));
        setWorkflows(adaptedWorkflows);
        // 更新分页信息
        setPagination(prev => ({
          ...prev,
          current: page,
          pageSize,
          total: response.total || 0,
        }));
      } catch (error) {
        console.error('Failed to load workflows:', error);
        // 显示实际错误信息以便调试
        const errorMessage = error instanceof Error ? error.message : String(error);
        message.error(t('workflow.loadFailed') + ': ' + errorMessage);
        setWorkflows([]); // 确保清空列表
      } finally {
        setLoading(false);
      }
    },
    [t]
  );

  // 分页/排序/筛选变化处理
  const handleTableChange = (page: number, pageSize: number) => {
    loadWorkflows(page, pageSize);
  };

  const loadStats = useCallback(async () => {
    // 从已加载的工作流数据中计算统计信息
    setStats(prev => {
      const active = workflows.filter(w => w.status === 'active').length;
      const draft = workflows.filter(w => w.status === 'draft').length;
      const inactive = workflows.filter(w => w.status === 'inactive').length;
      const running = workflows.reduce((sum, w) => sum + (w.runningInstances || 0), 0);
      const completed = workflows.reduce((sum, w) => sum + (w.instancesCount || 0), 0) - running;
      return {
        ...prev,
        total: pagination.total,
        active,
        draft,
        inactive,
        running: Math.max(0, running),
        completed: Math.max(0, completed),
        todayInstances: 0,
        avgExecutionTime: 0,
      };
    });
  }, [workflows, pagination.total]);

  // 当工作流列表更新时同步刷新统计
  useEffect(() => {
    loadStats();
  }, [workflows, pagination.total]);  

  // 初始加载工作流列表
  useEffect(() => {
    loadWorkflows(pagination.current, pagination.pageSize);
  }, []); // 只在组件挂载时加载一次

  // 当filters变化时重新加载数据（重置到第一页）
  useEffect(() => {
    loadWorkflows(1, pagination.pageSize);
  }, [filters]);  

  const handleCreateWorkflow = () => {
    setEditingWorkflow(null);
    setModalVisible(true);
  };

  const handleEditWorkflow = (workflow: Workflow) => {
    setEditingWorkflow(workflow);
    setModalVisible(true);
  };

  const handleViewWorkflow = (workflow: Workflow) => {
    setViewingWorkflow(workflow);
  };

  const workflowConsoleEntries = useMemo(
    () => [
      {
        key: 'dashboard',
        title: '监控仪表盘',
        description: '查看流程健康度、SLA 风险和运行趋势',
        icon: <BarChart3 className="h-5 w-5" />,
        path: '/workflow/dashboard',
        accent: '#1677ff',
      },
      {
        key: 'instances',
        title: '实例监控',
        description: '追踪运行中实例、异常实例和关键节点',
        icon: <Activity className="h-5 w-5" />,
        path: '/workflow/instances',
        accent: '#52c41a',
      },
      {
        key: 'versions',
        title: '版本治理',
        description: '管理历史版本、激活、回滚与比较',
        icon: <GitBranch className="h-5 w-5" />,
        path: '/workflow/versions',
        accent: '#722ed1',
      },
      {
        key: 'automation',
        title: '自动化规则',
        description: '配置自动分配、路由与升级规则',
        icon: <Workflow className="h-5 w-5" />,
        path: '/workflow/automation',
        accent: '#faad14',
      },
      {
        key: 'audit',
        title: '审计追踪',
        description: '查看流程动作、用户行为和时间线',
        icon: <ShieldCheck className="h-5 w-5" />,
        path: '/workflow/audit',
        accent: '#13c2c2',
      },
      {
        key: 'designer',
        title: '审批设计器',
        description: '快速进入工单审批与 BPMN 设计',
        icon: <Code className="h-5 w-5" />,
        path: '/workflow/ticket-approval',
        accent: '#eb2f96',
      },
    ],
    []
  );

  const handleDesignWorkflow = (workflow: Workflow) => {
    setEditingWorkflow(workflow);

    // 统一使用BPMN设计器，传递 key 而不是 id
    router.push(`/workflow/designer?id=${workflow.id}&key=${workflow.id}`);
  };

  const handleViewBPMN = (workflow: Workflow) => {
    if (!workflow.bpmnXml) {
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
            {workflow.bpmnXml}
          </pre>
        </div>
      ),
      okText: t('workflow.close'),
    });
  };

  const handleDeleteWorkflow = async (id: React.Key) => {
    modal.confirm({
      title: t('workflow.confirmDelete'),
      content: t('workflow.deleteConfirmation'),
      okText: t('workflow.confirmDelete'),
      okType: 'danger',
      cancelText: t('workflow.cancel'),
      onOk: async () => {
        try {
          await WorkflowAPI.deleteWorkflow(String(id));
          setWorkflows(prev => prev.filter(w => w.id !== String(id)));
          message.success(t('workflow.deleteSuccess'));
          loadStats();
        } catch {
          message.error(t('workflow.deleteFailed'));
        }
      },
    });
  };

  const handleDeployWorkflow = async (id: React.Key) => {
    modal.confirm({
      title: t('workflow.confirmDeploy'),
      content: t('workflow.deployConfirmation'),
      okText: t('workflow.confirmDeploy'),
      cancelText: t('workflow.cancel'),
      onOk: async () => {
        try {
          await WorkflowAPI.deployWorkflow(String(id));
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
      const result = await WorkflowAPI.cloneWorkflow(
        String(workflow.id),
        `${workflow.name} - ${t('workflow.copySuffix')}`
      );
      const newWorkflow: Workflow = {
        ...workflow,
        id: String(result.id || `${workflow.id}_copy_${Date.now()}`),
        name: result.name || '',
        status: 'draft',
        version: String(result.version) || '1.0.0',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        instancesCount: 0,
        runningInstances: 0,
        createdBy: t('workflow.currentUser'),
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
      const exportData = await WorkflowAPI.exportWorkflow(String(workflow.id));

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: 'application/json',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `workflow_${workflow.name}_${new Date().toISOString().split('T')[0]}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      message.success(t('workflow.exportSuccess', { name: workflow.name }));
    } catch {
      message.error(t('workflow.exportFailed'));
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
          id: String(importData.workflow.id || importData.workflow.key || Date.now()),
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
          status: 'draft',
          instancesCount: 0,
          runningInstances: 0,
          createdBy: t('workflow.currentUser'),
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

  const handleStopWorkflow = async (id: React.Key) => {
    modal.confirm({
      title: t('workflow.confirmDeactivate'),
      content: t('workflow.deactivateConfirmation'),
      okText: t('workflow.confirmDeactivate'),
      okType: 'danger',
      cancelText: t('workflow.cancel'),
      onOk: async () => {
        try {
          await WorkflowAPI.deactivateWorkflow(String(id));
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

  const handleActivateWorkflow = async (id: React.Key) => {
    try {
      await WorkflowAPI.activateWorkflow(String(id));
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
          for (const id of selectedRowKeys) {
            await WorkflowAPI.deleteWorkflow(String(id));
          }
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
          for (const id of selectedRowKeys) {
            await WorkflowAPI.activateWorkflow(String(id));
          }
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
          for (const id of selectedRowKeys) {
            await WorkflowAPI.deactivateWorkflow(String(id));
          }
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

  const getCategoryLabel = (category: string) => {
    const labels: Record<string, string> = {
      general: t('workflow.generalWorkflow'),
      approval: t('workflow.approvalProcess'),
      incident: t('workflow.incidentHandling'),
      change: t('workflow.changeManagement'),
      ticket: t('workflow.ticketApprovalProcess'),
    };
    return labels[category] || category;
  };

  const normalizeCategory = (category: string) => {
    const normalized: Record<string, string> = {
      [t('workflow.generalWorkflow')]: 'general',
      [t('workflow.approvalProcess')]: 'approval',
      [t('workflow.incidentHandling')]: 'incident',
      [t('workflow.changeManagement')]: 'change',
      [t('workflow.ticketApprovalProcess')]: 'ticket',
    };
    return normalized[category] || category;
  };

  const workflowCategoryOptions = [
    { value: 'general', label: t('workflow.generalWorkflow') },
    { value: 'approval', label: t('workflow.approvalProcess') },
    { value: 'incident', label: t('workflow.incidentHandling') },
    { value: 'change', label: t('workflow.changeManagement') },
    { value: 'ticket', label: t('workflow.ticketApprovalProcess') },
  ];

  const columns = [
    {
      title: t('workflow.name'),
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: Workflow) => (
        <div>
          <div className="font-medium text-gray-900">{name}</div>
          <div className="text-sm text-gray-500">{record.description}</div>
        </div>
      ),
    },
    {
      title: t('workflow.category'),
      dataIndex: 'category',
      key: 'category',
      width: 120,
      render: (category: string) => <Tag color="blue">{getCategoryLabel(category)}</Tag>,
    },
    {
      title: t('workflow.version'),
      dataIndex: 'version',
      key: 'version',
      width: 100,
      render: (version: string) => <span className="font-mono text-sm">{version}</span>,
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
      key:'bpmnStatus',
      width: 120,
      render: (record: Workflow) => (
        <Tag color={record.bpmnXml ? 'green' : 'orange'}>
          {record.bpmnXml ? t('workflow.defined') : t('workflow.undefined')}
        </Tag>
      ),
    },
    {
      title: t('workflow.instanceStats'),
      key: 'instances',
      width: 150,
      render: (record: Workflow) => (
        <div className="text-sm">
          <div>
            {t('workflow.total')}: {record.instancesCount}
          </div>
          <div>
            {t('workflow.running')}: {record.runningInstances}
          </div>
        </div>
      ),
    },
    {
      title: t('workflow.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (date: string) => (
        <div className="text-sm">{new Date(date).toLocaleDateString()}</div>
      ),
    },
    {
      title: t('workflow.actions'),
      key: 'action',
      width: 280,
      render: (record: Workflow) => (
        <Space>
          {/* 快捷部署按钮 - 直接显示在操作栏 */}
          {record.status !== 'active' && (
            <Tooltip title={record.status === 'draft' ? '部署' : '重新部署'}>
              <Button
                type="primary"
                size="small"
                icon={<PlayCircle className="w-3 h-3" />}
                onClick={() => handleDeployWorkflow(record.id)}
              >
                部署
              </Button>
            </Tooltip>
          )}
          <Tooltip title={t('workflow.designWorkflow')}>
            <Button
              type="text"
              icon={<GitBranch className="w-4 h-4" />}
              onClick={() => handleDesignWorkflow(record)}
              aria-label="设计工作流"
            />
          </Tooltip>
          <Tooltip title={t('workflow.viewDetails')}>
            <Button
              type="text"
              icon={<Eye className="w-4 h-4" />}
              onClick={() => handleViewWorkflow(record)}
              aria-label="查看工作流"
            />
          </Tooltip>
          <Dropdown
            menu={{
              items: [
                {
                  key: 'edit',
                  label: t('workflow.edit'),
                  icon: <Edit className="w-4 h-4" />,
                  onClick: () => handleEditWorkflow(record),
                },
                {
                  key: 'deploy',
                  label: record.status === 'draft' ? t('workflow.deploy') : t('workflow.redeploy'),
                  icon: <PlayCircle className="w-4 h-4" />,
                  onClick: () => handleDeployWorkflow(record.id),
                  disabled: record.status === 'active',
                },
                {
                  key: 'activate',
                  label: t('workflow.activate'),
                  icon: <CheckCircle className="w-4 h-4" />,
                  onClick: () => handleActivateWorkflow(record.id),
                  style: {
                    display: record.status === 'inactive' ? 'block' : 'none',
                  },
                },
                {
                  key: 'stop',
                  label: t('workflow.deactivate'),
                  icon: <PauseCircle className="w-4 h-4" />,
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
                  icon: <Copy className="w-4 h-4" />,
                  onClick: () => handleCopyWorkflow(record),
                },
                {
                  key: 'export',
                  label: t('workflow.export'),
                  icon: <Download className="w-4 h-4" />,
                  onClick: () => handleExportWorkflow(record),
                },
                {
                  key: 'instances',
                  label: t('workflow.viewInstances'),
                  icon: <BarChart3 className="w-4 h-4" />,
                  onClick: () =>
                    window.open(`/workflow/instances?workflowId=${record.id}`, '_blank'),
                },
                {
                  key: 'versions',
                  label: t('workflow.versionManagement'),
                  icon: <GitBranch className="w-4 h-4" />,
                  onClick: () =>
                    window.open(`/workflow/versions?workflowId=${record.id}`, '_blank'),
                },
                {
                  key: 'viewBPMN',
                  label: t('workflow.viewBPMN'),
                  icon: <Code className="w-4 h-4" />,
                  onClick: () => handleViewBPMN(record),
                  disabled: !record.bpmnXml,
                },
                {
                  type: 'divider' as const,
                },
                {
                  key: 'delete',
                  label: t('workflow.delete'),
                  icon: <Trash2 className="w-4 h-4" />,
                  danger: true,
                  onClick: () => handleDeleteWorkflow(record.id),
                  disabled: record.status === 'active' && record.runningInstances > 0,
                },
              ].filter(item => item.style?.display !== 'none'), // 过滤隐藏项目
            }}
          >
            <Button
              type="text"
              icon={<MoreHorizontal className="w-4 h-4" />}
              aria-label="更多操作"
            />
          </Dropdown>
        </Space>
      ),
    },
  ];

  const filteredWorkflows = workflows.filter(workflow => {
    if (filters.status && workflow.status !== filters.status) return false;
    if (filters.category && normalizeCategory(workflow.category) !== filters.category) return false;
    if (filters.keyword && !workflow.name.toLowerCase().includes(filters.keyword.toLowerCase()))
      return false;
    return true;
  });

  const statsItems = useMemo(
    () => [
      {
        key: 'total',
        title: t('workflow.totalWorkflows'),
        value: stats.total,
        prefix: <FileText className="w-5 h-5" />,
        accentColor: '#1890ff',
        helper: `${t('workflow.active')} ${stats.active} | ${t('workflow.draft')} ${stats.draft} | ${t('workflow.inactive')} ${stats.inactive}`,
      },
      {
        key: 'running',
        title: t('workflow.runningInstances'),
        value: stats.running,
        prefix: <Clock className="w-5 h-5" />,
        accentColor: '#faad14',
        helper: t('workflow.todayNewInstances', { count: stats.todayInstances }),
      },
      {
        key: 'completed',
        title: t('workflow.completedInstances'),
        value: stats.completed,
        prefix: <CheckCircle className="w-5 h-5" />,
        accentColor: '#52c41a',
        helper: `${t('workflow.completionRate')} ${
          stats.completed + stats.running > 0
            ? Math.round((stats.completed / (stats.completed + stats.running)) * 100)
            : 0
        }%`,
      },
      {
        key: 'efficiency',
        title: t('workflow.avgExecutionTime'),
        value: stats.avgExecutionTime,
        suffix: t('workflow.minutes'),
        prefix: <BarChart3 className="w-5 h-5" />,
        accentColor: '#722ed1',
        helper:
          stats.avgExecutionTime < 60
            ? t('workflow.goodEfficiency')
            : t('workflow.optimizableSpace'),
      },
    ],
    [stats, t]
  );

  const headerActions = (
    <>
      <Dropdown
        menu={{
          items: [
            {
              key:'ticketApproval',
              label: t('workflow.ticketApprovalProcess'),
              icon: <GitBranch className="w-4 h-4" />,
              onClick: () => router.push('/workflow/designer'),
            },
            {
              key: 'general',
              label: t('workflow.generalWorkflow'),
              icon: <GitBranch className="w-4 h-4" />,
              onClick: handleCreateWorkflow,
            },
            {
              key: 'bpmn',
              label: t('workflow.bpmnWorkflow'),
              icon: <Code className="w-4 h-4" />,
              onClick: () => {
                setEditingWorkflow(null);
                setDesignerVisible(true);
              },
            },
          ],
        }}
        trigger={['click']}
      >
        <Button type="primary" icon={<Plus className="w-4 h-4" />}>
          {t('workflow.newWorkflow')} <MoreHorizontal className="w-3 h-3 ml-1" />
        </Button>
      </Dropdown>
      <Button
        type="default"
        icon={<PlayCircle className="w-4 h-4" />}
        onClick={() => router.push('/workflow/instances')}
      >
        发起流程
      </Button>
      <Button
        icon={<Upload className="w-4 h-4" />}
        onClick={handleImportWorkflow}
        aria-label="导入工作流"
      >
        {t('workflow.import')}
      </Button>
      <Button
        icon={<RefreshCw className="w-4 h-4" />}
        onClick={() => loadWorkflows(pagination.current, pagination.pageSize)}
      >
        {t('workflow.refresh')}
      </Button>
    </>
  );

  const filterToolbar = (
    <>
      <Input
        placeholder={t('workflow.searchPlaceholder')}
        prefix={<Search className="w-4 h-4" />}
        value={filters.keyword}
        onChange={e => setFilters(prev => ({ ...prev, keyword: e.target.value }))}
        allowClear
        className="min-w-[220px]"
      />
      <Select
        placeholder={t('workflow.statusFilter')}
        value={filters.status || undefined}
        onChange={value => setFilters(prev => ({ ...prev, status: value ?? '' }))}
        allowClear
        className="min-w-[160px]"
      >
        <Option value="draft">{t('workflow.draft')}</Option>
        <Option value="active">{t('workflow.activated')}</Option>
        <Option value="inactive">{t('workflow.deactivated')}</Option>
        <Option value="archived">{t('workflow.archived')}</Option>
      </Select>
      <Select
        placeholder={t('workflow.categoryFilter')}
        value={filters.category || undefined}
        onChange={value => setFilters(prev => ({ ...prev, category: value ?? '' }))}
        allowClear
        className="min-w-[180px]"
      >
        <Option value="general">{t('workflow.generalWorkflow')}</Option>
        <Option value="approval">{t('workflow.approvalProcess')}</Option>
        <Option value="incident">{t('workflow.incidentHandling')}</Option>
        <Option value="change">{t('workflow.changeManagement')}</Option>
      </Select>
    </>
  );

  return (
    <>
      <ManagementPageHeader
        title={t('workflow.workflowManagement')}
        description="统一管理工作流定义、运行状态、设计入口和批量操作，作为工作流控制台主入口。"
        actions={headerActions}
      />

      <Row gutter={[16, 16]} className="mb-6">
        {workflowConsoleEntries.map(entry => (
          <Col key={entry.key} xs={24} sm={12} lg={8}>
            <Card
              hoverable
              className="h-full border border-gray-200 shadow-sm"
              onClick={() => router.push(entry.path)}
              styles={{
                body: {
                  padding: 20,
                },
              }}
            >
              <Space align="start" size={14} className="w-full">
                <div
                  style={{
                    width: 42,
                    height: 42,
                    borderRadius: 12,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: entry.accent,
                    background: `${entry.accent}15`,
                    flex: '0 0 auto',
                  }}
                >
                  {entry.icon}
                </div>
                <div className="flex-1">
                  <div className="text-base font-semibold text-gray-900">{entry.title}</div>
                  <div className="mt-1 text-sm text-gray-500">{entry.description}</div>
                </div>
              </Space>
            </Card>
          </Col>
        ))}
      </Row>

      <StatsOverview items={statsItems} className="mb-6" />

      <FilterToolbarCard
        className="mb-6"
        filters={filterToolbar}
        actions={
          <Text type="secondary">
            共 {filteredWorkflows.length} / {pagination.total} 个工作流
          </Text>
        }
      />

      {/* 批量操作工具栏（统一 BatchActionBar） */}
      <BatchActionBar
        selectedCount={selectedRowKeys.length}
        itemLabel={t('workflow.workflowItem') || '工作流'}
        leftExtra={t('workflow.selectedWorkflows', { count: selectedRowKeys.length })}
        onClear={() => setSelectedRowKeys([])}
        actions={[
          {
            key: 'activate',
            label: t('workflow.batchActivate'),
            type: 'primary',
            onClick: handleBatchActivate,
          },
          {
            key: 'stop',
            label: t('workflow.batchDeactivate'),
            onClick: handleBatchStop,
          },
          {
            key: 'export',
            label: t('workflow.batchExport'),
            onClick: handleBatchExport,
          },
          {
            key: 'delete',
            label: t('workflow.batchDelete'),
            danger: true,
            // 已有 Modal.confirm 二次确认，工具栏无需再次弹 Popconfirm
            onClick: handleBatchDelete,
          },
        ]}
      />

      {/* 工作流表格 */}
      <Card className="enterprise-card">
        {filteredWorkflows.length === 0 && !loading ? (
          <LoadingEmptyError
            state="empty"
            empty={{
              title: '暂无匹配的工作流',
              description: '可以放宽筛选条件，或者直接创建新的工作流定义。',
              actionText: t('workflow.newWorkflow'),
              onAction: handleCreateWorkflow,
            }}
          />
        ) : (
          <Table
            columns={columns}
            dataSource={filteredWorkflows}
            rowKey="id"
            loading={loading}
            rowSelection={{
              selectedRowKeys,
              onChange: (selectedRowKeys: React.Key[]) => {
                setSelectedRowKeys(selectedRowKeys);
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
              current: pagination.current,
              pageSize: pagination.pageSize,
              total: pagination.total,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total, range) =>
                t('workflow.showTotal', { start: range[0], end: range[1], total }),
              onChange: handleTableChange,
            }}
            scroll={{ x: 1200 }}
          />
        )}
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
          layout="vertical"
          initialValues={editingWorkflow || {}}
          onFinish={async values => {
            try {
              if (editingWorkflow) {
                const updateData: UpdateWorkflowRequest = {
                  ...values,
                };
                await WorkflowAPI.updateWorkflow(String(editingWorkflow.id), updateData);
              } else {
                const createData: CreateWorkflowRequest = {
                  ...values,
                  code: values.code || `workflow_${Date.now()}`,
                };
                await WorkflowAPI.createWorkflow(createData);
              }
              message.success(
                editingWorkflow ? t('workflow.updateSuccess') : t('workflow.createWorkflowSuccess')
              );
              setModalVisible(false);
              loadWorkflows();
            } catch (error) {
              message.error(t('workflow.operationFailed') + (error as Error).message);
            }
          }}
        >
          <Form.Item
            name="name"
            label={t('workflow.workflowName')}
            rules={[{ required: true, message: t('workflow.workflowNameRequired') }]}
          >
            <Input placeholder={t('workflow.workflowNamePlaceholder')} />
          </Form.Item>

          <Form.Item name="description" label={t('workflow.workflowDescription')}>
            <Input.TextArea rows={3} placeholder={t('workflow.workflowDescriptionPlaceholder')} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="category"
                label={t('workflow.category')}
                rules={[{ required: true, message: t('workflow.categoryRequired') }]}
              >
                <Select placeholder={t('workflow.categoryPlaceholder')}>
                  {workflowCategoryOptions.map(option => (
                    <Option key={option.value} value={option.value}>
                      {option.label}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="status" label={t('workflow.status')}>
                <Select placeholder={t('workflow.statusPlaceholder')}>
                  <Option value="draft">{t('workflow.draft')}</Option>
                  <Option value="active">{t('workflow.activated')}</Option>
                  <Option value="inactive">{t('workflow.deactivated')}</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
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
        width="95%"
        style={{ top: 10 }}
        destroyOnHidden
      >
        <BPMNDesigner
          xml={editingWorkflow?.bpmnXml || ''}
          onSave={async (xml: string) => {
            try {
              if (editingWorkflow) {
                // 更新现有工作流
                const updateData: any = {
                  bpmnXml: xml,
                };
                await WorkflowAPI.updateWorkflow(String(editingWorkflow.id), updateData);
                message.success(t('workflow.saveSuccess'));
              } else {
                // 创建新工作流
                const createData: CreateWorkflowRequest = {
                  name: t('workflow.newBPMNWorkflow'),
                  description: t('workflow.bpmnWorkflowDescription'),
                  code: `workflow_${Date.now()}`,
                  type: WorkflowType.APPROVAL,
                  bpmnXml: xml,
                };
                await WorkflowAPI.createWorkflow(createData);
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
                const updateData: any = {
                  bpmnXml: xml,
                };
                await WorkflowAPI.updateWorkflow(String(editingWorkflow.id), updateData);
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
                  bpmnXml: xml,
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

      <Modal
        title={viewingWorkflow ? `${viewingWorkflow.name} · ${t('workflow.viewDetails')}` : t('workflow.viewDetails')}
        open={!!viewingWorkflow}
        onCancel={() => setViewingWorkflow(null)}
        footer={null}
        width={720}
        destroyOnHidden
      >
        {viewingWorkflow && (
          <Space orientation="vertical" size="large" style={{ width: '100%' }}>
            <Descriptions bordered size="small" column={2}>
              <Descriptions.Item label={t('workflow.name')} span={2}>
                {viewingWorkflow.name}
              </Descriptions.Item>
              <Descriptions.Item label={t('workflow.description')} span={2}>
                {viewingWorkflow.description || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('workflow.category')}>
                <Tag color="blue">{getCategoryLabel(viewingWorkflow.category)}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('workflow.status')}>
                <Tag color={getStatusColor(viewingWorkflow.status)}>{getStatusText(viewingWorkflow.status)}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('workflow.version')}>
                <span className="font-mono">{viewingWorkflow.version}</span>
              </Descriptions.Item>
              <Descriptions.Item label={t('workflow.bpmnStatus')}>
                <Tag color={viewingWorkflow.bpmnXml ? 'green' : 'orange'}>
                  {viewingWorkflow.bpmnXml ? t('workflow.defined') : t('workflow.undefined')}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('workflow.total')}>
                {viewingWorkflow.instancesCount}
              </Descriptions.Item>
              <Descriptions.Item label={t('workflow.running')}>
                {viewingWorkflow.runningInstances}
              </Descriptions.Item>
              <Descriptions.Item label={t('workflow.createdAt')}>
                {new Date(viewingWorkflow.createdAt).toLocaleString('zh-CN')}
              </Descriptions.Item>
              <Descriptions.Item label={t('workflow.updatedAt')}>
                {new Date(viewingWorkflow.updatedAt).toLocaleString('zh-CN')}
              </Descriptions.Item>
              <Descriptions.Item label="创建人" span={2}>
                {viewingWorkflow.createdBy}
              </Descriptions.Item>
            </Descriptions>

            <Space wrap>
              <Button type="primary" onClick={() => handleDesignWorkflow(viewingWorkflow)}>
                {t('workflow.designWorkflow')}
              </Button>
              <Button onClick={() => window.open(`/workflow/instances?workflowId=${viewingWorkflow.id}`, '_blank')}>
                {t('workflow.viewInstances')}
              </Button>
              <Button onClick={() => window.open(`/workflow/versions?workflowId=${viewingWorkflow.id}`, '_blank')}>
                {t('workflow.versionManagement')}
              </Button>
            </Space>
          </Space>
        )}
      </Modal>
    </>
  );
};

export default WorkflowManagementPage;
