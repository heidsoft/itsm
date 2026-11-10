'use client';

import {
  Pause,
  Play,
  AlertCircle,
  GitBranch,
  Users,
  Settings,
  CheckCircle,
  Edit,
  Eye,
  Trash2,
  Search,
  Plus,
  FileText,
  MoreHorizontal,
  Copy,
} from 'lucide-react';
import React, { useState, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Space,
  Typography,
  Tag,
  Modal,
  Form,
  Row,
  Col,
  Statistic,
  Tooltip,
  Popconfirm,
  App,
  Empty,
  Dropdown,
} from 'antd';
import type { MenuProps } from 'antd';
import { WorkflowAPI } from '@/lib/api/workflow-api';
import { WorkflowType } from '@/types/workflow';
import { UsageGuideCard } from '@/components/common/UsageGuideCard';
import BPMNDiagramPreview from '@/components/workflow/BPMNDiagramPreview';
import WorkflowTemplateCatalog from '@/components/workflow/WorkflowTemplateCatalog';
const { Title, Text } = Typography;

// 工作流状态枚举
const WORKFLOW_STATUS = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  DRAFT: 'draft',
} as const;

// 工作流类型枚举
const WORKFLOW_TYPES = {
  TICKET: 'ticket',
  INCIDENT: 'incident',
  SERVICE_REQUEST: 'service_request',
  CHANGE: 'change',
  PROBLEM: 'problem',
  APPROVAL: 'approval',
} as const;

// 工作流数据类型（id 使用 process key，与后端 BPMN 路由契约一致）
interface Workflow {
  id: string;
  name: string;
  description: string;
  type: string;
  status: string;
  version: string;
  createdBy: string;
  createdAt: string;
  lastModified: string;
}


// 工作流类型配置
const WORKFLOW_TYPE_CONFIG = {
  [WORKFLOW_TYPES.TICKET]: {
    label: 'Ticket Process',
    color: 'cyan',
    icon: <FileText className="w-3 h-3" />,
  },
  [WORKFLOW_TYPES.INCIDENT]: {
    label: 'Incidents',
    color: 'red',
    icon: <AlertCircle className="w-3 h-3" />,
  },
  [WORKFLOW_TYPES.SERVICE_REQUEST]: {
    label: 'Service Request',
    color: 'blue',
    icon: <Users className="w-3 h-3" />,
  },
  [WORKFLOW_TYPES.CHANGE]: {
    label: 'Changes',
    color: 'orange',
    icon: <GitBranch className="w-3 h-3" />,
  },
  [WORKFLOW_TYPES.PROBLEM]: {
    label: 'Problems',
    color: 'purple',
    icon: <Settings className="w-3 h-3" />,
  },
  [WORKFLOW_TYPES.APPROVAL]: {
    label: 'Workflow',
    color: 'green',
    icon: <CheckCircle className="w-3 h-3" />,
  },
};

// 工作流状态配置
const STATUS_CONFIG = {
  [WORKFLOW_STATUS.ACTIVE]: {
    label: 'Enabled',
    color: 'success',
    icon: <Play className="w-3 h-3" />,
  },
  [WORKFLOW_STATUS.INACTIVE]: {
    label: 'Disabled',
    color: 'default',
    icon: <Pause className="w-3 h-3" />,
  },
  [WORKFLOW_STATUS.DRAFT]: {
    label: 'Draft',
    color: 'processing',
    icon: <Edit className="w-3 h-3" />,
  },
};

const WorkflowManagement = () => {
  const { message, modal } = App.useApp();
  const router = useRouter();
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [debouncedSearchTerm, setDebouncedSearchTerm] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showDetailModal, setShowDetailModal] = useState(false);
  const [selectedWorkflow, setSelectedWorkflow] = useState<Workflow | null>(null);
  const [detailXml, setDetailXml] = useState('');
  const [detailXmlLoading, setDetailXmlLoading] = useState(false);
  const [detailXmlError, setDetailXmlError] = useState('');
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 「Total workflows 0」的假统计卡与空表格，再被真实数据覆盖。
  const [listLoading, setListLoading] = useState(true);
  // 复制/删除/批量删除/Save是提交动作，不应与列表加载共用一个标志：
  // 共用会让「Save中」把整个表格也转圈，反之亦然。
  const [actionLoading, setActionLoading] = useState(false);
  const [form] = Form.useForm();
  const detailXmlRequestRef = useRef(0);

  // 加载工作流数据
  const loadWorkflows = async () => {
    setListLoading(true);
    try {
      const response = await WorkflowAPI.getWorkflows({});
      const workflowList: Workflow[] = (response.workflows || []).map((w) => ({
        id: w.id || w.code || '',
        name: w.name || w.code || '',
        description: w.description || '',
        type: w.category || 'ticket',
        status: w.status || 'draft',
        version: String(w.version || '1'),
        createdBy: w.createdByName || '',
        createdAt: w.createdAt instanceof Date ? w.createdAt.toISOString() : (w.createdAt || ''),
        lastModified: w.updatedAt instanceof Date ? w.updatedAt.toISOString() : (w.updatedAt || w.createdAt || ''),
      }));
      setWorkflows(workflowList);
    } catch (error) {
      console.error('Failed to load workflows:', error);
      message.error('Failed to load workflow data');
    } finally {
      setListLoading(false);
    }
  };

  useEffect(() => {
    loadWorkflows();
  }, []);

  // 搜索输入防抖（300ms）
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearchTerm(searchTerm), 300);
    return () => clearTimeout(timer);
  }, [searchTerm]);

  // 统计信息
  const stats = {
    total: workflows.length,
    active: workflows.filter(w => w.status === WORKFLOW_STATUS.ACTIVE).length,
    draft: workflows.filter(w => w.status === WORKFLOW_STATUS.DRAFT).length,
    inactive: workflows.filter(w => w.status === WORKFLOW_STATUS.INACTIVE).length,
  };

  // 获取所有工作流类型
  const workflowTypes = Array.from(new Set(workflows.map(workflow => workflow.type)));

  // 过滤工作流
  const filteredWorkflows = workflows.filter(workflow => {
    const matchesSearch =
      workflow.name.toLowerCase().includes(debouncedSearchTerm.toLowerCase()) ||
      workflow.description.toLowerCase().includes(debouncedSearchTerm.toLowerCase());
    const matchesType = typeFilter === 'all' || workflow.type === typeFilter;
    const matchesStatus = statusFilter === 'all' || workflow.status === statusFilter;

    return matchesSearch && matchesType && matchesStatus;
  });

  // 处理工作流状态切换
  const handleStatusToggle = async (workflow: Workflow) => {
    const newStatus = workflow.status === WORKFLOW_STATUS.ACTIVE
      ? WORKFLOW_STATUS.INACTIVE
      : WORKFLOW_STATUS.ACTIVE;

    try {
      // 后端按 (key, version) 精确定位版本，必须回传列表里的真实版本号。
      if (newStatus === WORKFLOW_STATUS.ACTIVE) {
        await WorkflowAPI.activateWorkflow(workflow.id, workflow.version);
      } else {
        await WorkflowAPI.deactivateWorkflow(workflow.id, workflow.version);
      }

      setWorkflows(prev =>
        prev.map(w => {
          if (w.id === workflow.id) {
            return { ...w, status: newStatus };
          }
          return w;
        })
      );
      message.success('Workflow status updated');
    } catch (error) {
      console.error('Failed to toggle workflow status:', error);
      message.error('Failed to update status');
    }
  };

  // 处理工作流复制（调用后端 clone 接口，保留 bpmnXml）
  const handleDuplicate = async (workflow: Workflow) => {
    try {
      setActionLoading(true);
      await WorkflowAPI.cloneProcessDefinition(workflow.id, {
        newKey: `${workflow.id}_copy_${Date.now()}`,
        newName: `${workflow.name} (Copy)`,
      });
      message.success('Workflow copied');
      loadWorkflows();
    } catch (error) {
      console.error('Failed to duplicate workflow:', error);
      message.error('Failed to copy workflow');
    } finally {
      setActionLoading(false);
    }
  };

  // 处理工作流删除（id 为 process key，version 为列表返回的真实版本）
  const handleDelete = async (workflow: Workflow) => {
    try {
      setActionLoading(true);
      await WorkflowAPI.deleteWorkflow(workflow.id);
      message.success('Workflow deleted');
      loadWorkflows();
    } catch (error) {
      console.error('Failed to delete workflow:', error);
      message.error('Failed to delete workflow');
    } finally {
      setActionLoading(false);
    }
  };

  // 批量删除（循环调用真实 API，id 为 process key）
  const handleBatchDelete = async () => {
    try {
      setActionLoading(true);
      const selected = workflows.filter(w => selectedRowKeys.includes(w.id));
      await Promise.all(
        selected.map(workflow => WorkflowAPI.deleteWorkflow(workflow.id))
      );
      message.success(`Deleted ${selectedRowKeys.length} workflows`);
      setSelectedRowKeys([]);
      loadWorkflows();
    } catch (error) {
      console.error('Failed to batch delete workflows:', error);
      message.error('Batch delete failed');
    } finally {
      setActionLoading(false);
    }
  };

  // 查看详情
  const handleViewDetail = (workflow: Workflow) => {
    setSelectedWorkflow(workflow);
    setShowDetailModal(true);

    // 列表接口不返回 BPMN XML，预览按需用详情接口取当前定义；
    // 快速连续打开两个详情时只接受最后一次请求的结果。
    const requestId = ++detailXmlRequestRef.current;
    setDetailXml('');
    setDetailXmlError('');
    setDetailXmlLoading(true);
    WorkflowAPI.getWorkflow(workflow.id)
      .then(definition => {
        if (requestId !== detailXmlRequestRef.current) return;
        setDetailXml(definition.bpmnXml || '');
      })
      .catch(() => {
        if (requestId === detailXmlRequestRef.current) setDetailXmlError('Failed to load process definition and cannot preview');
      })
      .finally(() => {
        if (requestId === detailXmlRequestRef.current) setDetailXmlLoading(false);
      });
  };

  // Save工作流（创建或更新走真实 API）
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setActionLoading(true);

      if (selectedWorkflow) {
        await WorkflowAPI.updateWorkflow(
          selectedWorkflow.id,
          {
            name: values.name,
            description: values.description,
            category: values.type,
          },
          selectedWorkflow.version
        );
        message.success('Workflow updated successfully');
      } else {
        const created = await WorkflowAPI.createWorkflow({
          code: values.name,
          name: values.name,
          description: values.description,
          type: (values.type as WorkflowType) ?? WorkflowType.APPROVAL,
        });
        message.success('Workflow created successfully');

        if (created.id) {
          modal.confirm({
            title: 'Do you want to enter the designer orchestration process immediately?',
            content: 'The workflow has been created as a draft and needs to be orchestrated in the designer before it can be enabled.',
            okText: 'Enter the designer',
            cancelText: 'Talk to you later',
            onOk: () => {
              router.push(`/workflow/designer?id=${created.id}`);
            },
          });
        }
      }

      setShowCreateModal(false);
      setSelectedWorkflow(null);
      form.resetFields();
      loadWorkflows();
    } catch (error) {
      console.error('Failed to save workflow:', error);
      message.error('Save failed, please check required fields or backend availability');
    } finally {
      setActionLoading(false);
    }
  };

  // 行选择配置
  const rowSelection = {
    selectedRowKeys,
    onChange: (keys: React.Key[]) => setSelectedRowKeys(keys),
    getCheckboxProps: (record: Workflow) => ({
      disabled: record.status === WORKFLOW_STATUS.ACTIVE,
    }),
  };

  // 表格列定义
  const columns = [
    {
      title: 'Workflow information',
      dataIndex: 'name',
      key: 'name',
      width: 360,
      render: (_: unknown, record: Workflow) => (
        <div>
          <div className="flex items-center gap-2">
            <Text strong>{record.name}</Text>
            <Tag color="blue">v{record.version}</Tag>
          </div>
          {record.description && (
            <Text type="secondary" className="text-sm block mt-1">
              {record.description}
            </Text>
          )}
          {record.createdBy && (
            <div className="text-xs text-gray-500 mt-1">Created by: {record.createdBy}</div>
          )}
        </div>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      align: 'center' as const,
      width: 110,
      render: (type: string) => {
        const config = WORKFLOW_TYPE_CONFIG[type as keyof typeof WORKFLOW_TYPE_CONFIG];
        if (!config) return <Tag>{type}</Tag>;
        return (
          <Tag color={config.color} icon={config.icon}>
            {config.label}
          </Tag>
        );
      },
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      align: 'center' as const,
      width: 100,
      render: (status: string) => {
        const config = STATUS_CONFIG[status as keyof typeof STATUS_CONFIG];
        if (!config) return <Tag>{status}</Tag>;
        return (
          <Tag color={config.color} icon={config.icon}>
            {config.label}
          </Tag>
        );
      },
    },
    {
      title: 'Last Modified',
      dataIndex: 'lastModified',
      key: 'lastModified',
      align: 'center' as const,
      width: 160,
      render: (date: unknown) => {
        if (!date) return <span className="text-gray-400">-</span>;
        const dateStr = String(date);
        const datePart = dateStr.includes('T')
          ? dateStr.split('T')[0]
          : dateStr.split(' ')[0] || dateStr;
        const timePart = dateStr.includes('T')
          ? (dateStr.split('T')[1] || '').split(/[+Z]/)[0]
          : dateStr.split(' ')[1] || '';
        return (
          <div className="text-center">
            <div className="text-sm">{datePart}</div>
            {timePart && <div className="text-xs text-gray-500">{timePart}</div>}
          </div>
        );
      },
    },
    {
      title: 'Actions',
      key: 'actions',
      align: 'center' as const,
      width: 180,
      fixed: 'right' as const,
      render: (_: unknown, record: Workflow) => {
        const isDraft = record.status === WORKFLOW_STATUS.DRAFT;
        const moreItems: MenuProps['items'] = [
          {
            key: 'view',
            label: 'View Details',
            icon: <Eye className="w-4 h-4" />,
            onClick: () => handleViewDetail(record),
          },
          {
            key: 'edit',
            label: 'Edit metadata',
            icon: <Edit className="w-4 h-4" />,
            onClick: () => {
              setSelectedWorkflow(record);
              form.setFieldsValue(record);
              setShowCreateModal(true);
            },
          },
          {
            key: 'duplicate',
            label: 'Copy',
            icon: <Copy className="w-4 h-4" />,
            onClick: () => handleDuplicate(record),
          },
          { type: 'divider' },
          {
            key: 'delete',
            label: 'Delete',
            icon: <Trash2 className="w-4 h-4" />,
            danger: true,
            onClick: () => {
              modal.confirm({
                title: 'Are you sure you want to delete this workflow?',
                content: `Process definition "${record.name}" v${record.version} will be deleted. This cannot be undone.`,
                okText: 'Confirm deletion',
                cancelText: 'Cancel',
                okType: 'danger',
                onOk: () => handleDelete(record),
              });
            },
          },
        ];
        return (
          <Space>
            <Tooltip title="Design Process">
              <Button
                aria-label="Design Process"
                type="text"
                icon={<GitBranch className="w-4 h-4" />}
                onClick={() => router.push(`/workflow/designer?id=${encodeURIComponent(record.id)}`)}
              />
            </Tooltip>
            <Tooltip title={record.status === WORKFLOW_STATUS.ACTIVE ? 'Disable' : 'Enabled'}>
              <Button
                aria-label={record.status === WORKFLOW_STATUS.ACTIVE ? 'Disable' : 'Enabled'}
                type="text"
                icon={
                  record.status === WORKFLOW_STATUS.ACTIVE ? (
                    <Pause className="w-4 h-4" />
                  ) : (
                    <Play className="w-4 h-4" />
                  )
                }
                disabled={isDraft}
                onClick={() => handleStatusToggle(record)}
              />
            </Tooltip>
            <Dropdown menu={{ items: moreItems }} trigger={['click']}>
              <Tooltip title="More">
                <Button
                  type="text"
                  aria-label="More actions"
                  icon={<MoreHorizontal className="w-4 h-4" />}
                />
              </Tooltip>
            </Dropdown>
          </Space>
        );
      },
    },
  ];

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div>
        <Title level={2} className="!mb-2">
          <GitBranch className="inline-block w-6 h-6 mr-2" />

          Workflow Management
        </Title>
        <Text type="secondary">Design and manage business processes, configure approval nodes and automation rules</Text>
      </div>

      <UsageGuideCard
        style={{ marginBottom: 16 }}
        title="How to use workflow management"
        intro="This page manages process definition (create/copy/start/stop/delete), and node arrangement needs to be done in BPMN designer."
        steps={[
          'Create a new process: Click "Create Workflow" to fill in the name and description (save as a draft first), enter the designer to arrange the nodes and connections, and then return to this page to enable it. The draft cannot be activated until it is scheduled.',
          'Operation: Click "Design Process" to enter the designer; click "Copy" to generate a draft copy based on the existing process; it can be enabled/disabled/delete, and activated processes cannot be deleted in batches.',
          'If you want to import from a template (such as leave approval): After importing, go to the designer to confirm that the node connections are complete before saving.',
          'Click "Process Routing" to enter the binding page and associate the process with business scenarios such as work orders; the corresponding process instance will be started only when the business object meets the routing rules.',
          'Verification: After enabling it, create a matching business data, go to "Workflow Instance" to see the execution status and to-do tasks, and go to "Workflow Audit" to see the transfer records.',
        ]}
      />

      <WorkflowTemplateCatalog />

      {/* 统计卡片 */}
      {/* 统计卡片必须门控：还没拉到数据时渲染成「Total workflows 0」会被误读成「没有工作流」 */}
      {!listLoading && (
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={8}>
            <Card className="enterprise-card">
              <Statistic
                title="Total workflows"
                value={stats.total}
                prefix={<GitBranch className="w-5 h-5" />}
                styles={{ content: { color: '#1890ff' } }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card className="enterprise-card">
              <Statistic
                title="Enabled"
                value={stats.active}
                prefix={<Play className="w-5 h-5" />}
                styles={{ content: { color: '#52c41a' } }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card className="enterprise-card">
              <Statistic
                title="Draft"
                value={stats.draft}
                prefix={<Edit className="w-5 h-5" />}
                styles={{ content: { color: '#722ed1' } }}
              />
            </Card>
          </Col>
        </Row>
      )}

      {/* 搜索和过滤 */}
      <Card>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={8}>
            <Input
              placeholder="Search workflow name or description..."
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder="Workflow type"
              value={typeFilter}
              onChange={setTypeFilter}
              style={{ width: '100%' }}
              options={[{ value: 'all', label: 'All Types' }, ...Object.entries(WORKFLOW_TYPE_CONFIG).map(([key, config]) => ({ value: key, label: config.label }))]}
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder="Status"
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: '100%' }}
              options={[{ value: 'all', label: 'All Status' }, ...Object.entries(STATUS_CONFIG).map(([key, config]) => ({ value: key, label: config.label }))]}
            />
          </Col>
          <Col xs={24} md={8} className="text-right">
            <Space>
              {selectedRowKeys.length > 0 && (
                <Popconfirm
                  title="Are you sure you want to delete the selected workflows in batches?"
                  onConfirm={handleBatchDelete}
                  okText="Confirm deletion"
                  cancelText="Cancel"
                  okType="danger"
                >
                  <Button danger icon={<Trash2 className="w-4 h-4" />}>

                    Batch delete ({selectedRowKeys.length})
                  </Button>
                </Popconfirm>
              )}
              <Button
                icon={<Settings className="w-4 h-4" />}
                onClick={() => router.push('/admin/process-routing')}
              >

                Binding rules
              </Button>
              <Button
                type="primary"
                icon={<Plus className="w-4 h-4" />}
                onClick={() => {
                  setSelectedWorkflow(null);
                  form.resetFields();
                  setShowCreateModal(true);
                }}
              >

                Create Workflow
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 工作流列表 */}
      <Card className="enterprise-card">
        {filteredWorkflows.length === 0 && !listLoading ? (
          <Empty description="No workflows yet. Use the button in the upper right to create one." />
        ) : (
          <Table
            columns={columns}
            dataSource={filteredWorkflows}
            rowKey="id"
            rowSelection={rowSelection}
            loading={listLoading}
            scroll={{ x: 1200 }}
            pagination={{
              total: filteredWorkflows.length,
              pageSize: 10,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: total => `Total ${total} records`,
            }}
            className="enterprise-table"
          />
        )}
      </Card>

      {/* 创建/编辑模态框 */}
      <Modal
        title={
          <span>
            <GitBranch className="w-4 h-4 mr-2" />
            {selectedWorkflow ? 'Edit Workflow' : 'Create Workflow'}
          </span>
        }
        open={showCreateModal}
        onOk={handleSave}
        onCancel={() => {
          setShowCreateModal(false);
          setSelectedWorkflow(null);
          form.resetFields();
        }}
        width={600}
        confirmLoading={actionLoading}
        okText="Save"
        cancelText="Cancel"
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="Workflow Name"
                name="name"
                rules={[{ required: true, message: 'Please enter the workflow name' }]}
              >
                <Input placeholder="Please enter the workflow name" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="Workflow type"
                name="type"
                rules={[{ required: true, message: 'Please select a workflow type' }]}
              >
                <Select placeholder="Select workflow type" options={Object.entries(WORKFLOW_TYPE_CONFIG).map(([key, config]) => ({ value: key, label: config.label }))} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            label="Description"
            name="description"
            rules={[{ required: true, message: 'Please enter the workflow description' }]}
          >
            <Input.TextArea rows={3} placeholder="Please enter the workflow description" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情模态框 */}
      <Modal
        title={
          <div className="flex items-center gap-2">
            <Eye className="w-5 h-5" />
            <span>Workflow details</span>
          </div>
        }
        open={showDetailModal}
        onCancel={() => setShowDetailModal(false)}
        width={800}
        destroyOnHidden
        footer={[
          <Button key="close" onClick={() => setShowDetailModal(false)}>

            Close
          </Button>,
        ]}
      >
        {selectedWorkflow && (
          <div className="space-y-6">
            <div>
              <Title level={4}>{selectedWorkflow.name}</Title>
              <Text type="secondary">{selectedWorkflow.description}</Text>
            </div>

            <div>
              <Title level={5}>Basic Info</Title>
              <Row gutter={[16, 8]}>
                <Col span={12}>
                  <Text strong>Workflow type:</Text>
                  <Tag
                    color={
                      WORKFLOW_TYPE_CONFIG[
                        selectedWorkflow.type as keyof typeof WORKFLOW_TYPE_CONFIG
                      ]?.color
                    }
                    icon={
                      WORKFLOW_TYPE_CONFIG[
                        selectedWorkflow.type as keyof typeof WORKFLOW_TYPE_CONFIG
                      ]?.icon
                    }
                  >
                    {
                      WORKFLOW_TYPE_CONFIG[
                        selectedWorkflow.type as keyof typeof WORKFLOW_TYPE_CONFIG
                      ]?.label
                    }
                  </Tag>
                </Col>
                <Col span={12}>
                  <Text strong>Current status:</Text>
                  <Tag
                    color={
                      STATUS_CONFIG[selectedWorkflow.status as keyof typeof STATUS_CONFIG]?.color
                    }
                    icon={
                      STATUS_CONFIG[selectedWorkflow.status as keyof typeof STATUS_CONFIG]?.icon
                    }
                  >
                    {STATUS_CONFIG[selectedWorkflow.status as keyof typeof STATUS_CONFIG]?.label}
                  </Tag>
                </Col>
                <Col span={12}>
                  <Text strong>Version:</Text>
                  <Text>v{selectedWorkflow.version}</Text>
                </Col>
                <Col span={12}>
                  <Text strong>Process identification:</Text>
                  <Text copyable>{selectedWorkflow.id}</Text>
                </Col>
                <Col span={12}>
                  <Text strong>Creation time:</Text>
                  <Text>{selectedWorkflow.createdAt || '-'}</Text>
                </Col>
                <Col span={12}>
                  <Text strong>Last modified:</Text>
                  <Text>{selectedWorkflow.lastModified || '-'}</Text>
                </Col>
              </Row>
            </div>

            <div>
              <Title level={5}>Process preview</Title>
              {detailXmlLoading ? (
                <div className="flex h-[320px] items-center justify-center rounded border border-dashed border-gray-200">
                  <Text type="secondary">Loading process definition...</Text>
                </div>
              ) : detailXmlError ? (
                <Text type="danger">{detailXmlError}</Text>
              ) : (
                <BPMNDiagramPreview xml={detailXml} height={320} />
              )}
            </div>

            <div className="mt-6">
              <Button
                type="primary"
                icon={<GitBranch />}
                onClick={() => {
                  router.push(`/workflow/designer?id=${selectedWorkflow.id}`);
                  setShowDetailModal(false);
                }}
              >

                Open Workflow Designer
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default WorkflowManagement;
