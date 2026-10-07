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
import React, { useState, useEffect } from 'react';
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
import { UsageGuideCard } from '@/components/common/UsageGuideCard';
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
    label: '工单流程',
    color: 'cyan',
    icon: <FileText className="w-3 h-3" />,
  },
  [WORKFLOW_TYPES.INCIDENT]: {
    label: '事件管理',
    color: 'red',
    icon: <AlertCircle className="w-3 h-3" />,
  },
  [WORKFLOW_TYPES.SERVICE_REQUEST]: {
    label: '服务请求',
    color: 'blue',
    icon: <Users className="w-3 h-3" />,
  },
  [WORKFLOW_TYPES.CHANGE]: {
    label: '变更管理',
    color: 'orange',
    icon: <GitBranch className="w-3 h-3" />,
  },
  [WORKFLOW_TYPES.PROBLEM]: {
    label: '问题管理',
    color: 'purple',
    icon: <Settings className="w-3 h-3" />,
  },
  [WORKFLOW_TYPES.APPROVAL]: {
    label: '审批流程',
    color: 'green',
    icon: <CheckCircle className="w-3 h-3" />,
  },
};

// 工作流状态配置
const STATUS_CONFIG = {
  [WORKFLOW_STATUS.ACTIVE]: {
    label: '已启用',
    color: 'success',
    icon: <Play className="w-3 h-3" />,
  },
  [WORKFLOW_STATUS.INACTIVE]: {
    label: '已停用',
    color: 'default',
    icon: <Pause className="w-3 h-3" />,
  },
  [WORKFLOW_STATUS.DRAFT]: {
    label: '草稿',
    color: 'processing',
    icon: <Edit className="w-3 h-3" />,
  },
};

const WorkflowManagement = () => {
  const { message } = App.useApp();
  const router = useRouter();
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [debouncedSearchTerm, setDebouncedSearchTerm] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showDetailModal, setShowDetailModal] = useState(false);
  const [selectedWorkflow, setSelectedWorkflow] = useState<Workflow | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  // 加载工作流数据
  const loadWorkflows = async () => {
    setLoading(true);
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
      message.error('加载工作流数据失败');
    } finally {
      setLoading(false);
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
  const handleStatusToggle = async (workflowKey: string) => {
    const workflow = workflows.find(w => w.id === workflowKey);
    if (!workflow) return;

    const newStatus = workflow.status === WORKFLOW_STATUS.ACTIVE
      ? WORKFLOW_STATUS.INACTIVE
      : WORKFLOW_STATUS.ACTIVE;

    try {
      if (newStatus === WORKFLOW_STATUS.ACTIVE) {
        await WorkflowAPI.activateWorkflow(workflowKey);
      } else {
        await WorkflowAPI.deactivateWorkflow(workflowKey);
      }

      setWorkflows(prev =>
        prev.map(w => {
          if (w.id === workflowKey) {
            return { ...w, status: newStatus };
          }
          return w;
        })
      );
      message.success('工作流状态已更新');
    } catch (error) {
      console.error('Failed to toggle workflow status:', error);
      message.error('更新状态失败');
    }
  };

  // 处理工作流复制（调用后端 clone 接口，保留 bpmnXml）
  const handleDuplicate = async (workflow: Workflow) => {
    try {
      setLoading(true);
      await WorkflowAPI.cloneProcessDefinition(workflow.id, {
        newKey: `${workflow.id}_copy_${Date.now()}`,
        newName: `${workflow.name} (副本)`,
        description: workflow.description,
      });
      message.success('工作流已复制');
      loadWorkflows();
    } catch (error) {
      console.error('Failed to duplicate workflow:', error);
      message.error('复制工作流失败');
    } finally {
      setLoading(false);
    }
  };

  // 处理工作流删除（id 为 process key）
  const handleDelete = async (workflowKey: string) => {
    try {
      setLoading(true);
      await WorkflowAPI.deleteWorkflow(workflowKey);
      message.success('工作流已删除');
      loadWorkflows();
    } catch (error) {
      console.error('Failed to delete workflow:', error);
      message.error('删除工作流失败');
    } finally {
      setLoading(false);
    }
  };

  // 批量删除（循环调用真实 API，id 为 process key）
  const handleBatchDelete = async () => {
    try {
      setLoading(true);
      await Promise.all(
        selectedRowKeys.map(key => WorkflowAPI.deleteWorkflow(String(key)))
      );
      message.success(`已删除 ${selectedRowKeys.length} 个工作流`);
      setSelectedRowKeys([]);
      loadWorkflows();
    } catch (error) {
      console.error('Failed to batch delete workflows:', error);
      message.error('批量删除失败');
    } finally {
      setLoading(false);
    }
  };

  // 查看详情
  const handleViewDetail = (workflow: Workflow) => {
    setSelectedWorkflow(workflow);
    setShowDetailModal(true);
  };

  // 保存工作流（创建或更新走真实 API）
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      if (selectedWorkflow) {
        await WorkflowAPI.updateWorkflow(selectedWorkflow.id, {
          name: values.name,
          description: values.description,
          category: values.type,
        } as never);
        message.success('工作流更新成功');
      } else {
        const created = await WorkflowAPI.createWorkflow({
          code: values.name,
          name: values.name,
          description: values.description,
          type: (values.type as 'approval') || 'approval',
          status: 'draft',
        } as never);
        message.success('工作流创建成功');

        const createdId = (created as { id?: number | string } | undefined)?.id;
        if (createdId) {
          Modal.confirm({
            title: '是否立即进入设计器编排流程？',
            content: '工作流已创建为草稿，需要在设计器中编排节点后才能启用。',
            okText: '进入设计器',
            cancelText: '稍后再说',
            onOk: () => {
              router.push(`/workflow/designer?id=${createdId}`);
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
      message.error('保存失败，请检查必填项或后端可用性');
    } finally {
      setLoading(false);
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
      title: '工作流信息',
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
            <div className="text-xs text-gray-500 mt-1">创建者: {record.createdBy}</div>
          )}
        </div>
      ),
    },
    {
      title: '类型',
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
      title: '状态',
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
      title: '最后修改',
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
      title: '操作',
      key: 'actions',
      align: 'center' as const,
      width: 180,
      fixed: 'right' as const,
      render: (_: unknown, record: Workflow) => {
        const isDraft = record.status === WORKFLOW_STATUS.DRAFT;
        const moreItems: MenuProps['items'] = [
          {
            key: 'view',
            label: '查看详情',
            icon: <Eye className="w-4 h-4" />,
            onClick: () => handleViewDetail(record),
          },
          {
            key: 'edit',
            label: '编辑元数据',
            icon: <Edit className="w-4 h-4" />,
            onClick: () => {
              setSelectedWorkflow(record);
              form.setFieldsValue(record);
              setShowCreateModal(true);
            },
          },
          {
            key: 'duplicate',
            label: '复制',
            icon: <Copy className="w-4 h-4" />,
            onClick: () => handleDuplicate(record),
          },
          { type: 'divider' },
          {
            key: 'delete',
            label: '删除',
            icon: <Trash2 className="w-4 h-4" />,
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: '确定要删除这个工作流吗？',
                content: `将删除流程定义「${record.name}」，删除后无法恢复。`,
                okText: '确定删除',
                cancelText: '取消',
                okType: 'danger',
                onOk: () => handleDelete(record.id),
              });
            },
          },
        ];
        return (
          <Space>
            <Tooltip title="设计流程">
              <Button
                aria-label="设计流程"
                type="text"
                icon={<GitBranch className="w-4 h-4" />}
                onClick={() => router.push(`/workflow/designer?id=${encodeURIComponent(record.id)}`)}
              />
            </Tooltip>
            <Tooltip title={record.status === WORKFLOW_STATUS.ACTIVE ? '停用' : '启用'}>
              <Button
                aria-label={record.status === WORKFLOW_STATUS.ACTIVE ? '停用' : '启用'}
                type="text"
                icon={
                  record.status === WORKFLOW_STATUS.ACTIVE ? (
                    <Pause className="w-4 h-4" />
                  ) : (
                    <Play className="w-4 h-4" />
                  )
                }
                disabled={isDraft}
                onClick={() => handleStatusToggle(record.id)}
              />
            </Tooltip>
            <Dropdown menu={{ items: moreItems }} trigger={['click']}>
              <Tooltip title="更多">
                <Button
                  type="text"
                  aria-label="更多操作"
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
          工作流管理
        </Title>
        <Text type="secondary">设计和管理业务流程，配置审批节点和自动化规则</Text>
      </div>

      <UsageGuideCard
        style={{ marginBottom: 16 }}
        title="工作流管理怎么用"
        intro="本页管流程定义（新建/复制/启停/删除），节点编排要去 BPMN 设计器做。"
        steps={[
          '新建流程：点"创建工作流"填名称和说明（先存为草稿），进设计器排好节点和连线，再回本页启用。草稿没排好前不能启用。',
          '行操作：点"设计流程"进设计器；点"复制"基于现有流程生成草稿副本；可启用/停用/删除，启用中的流程不能批量删。',
          '想从模板导入（比如请假审批）：导入后去设计器确认节点连线完整再保存。',
          '点"流程路由"进绑定页，把流程关联到工单等业务场景；业务对象满足路由规则时才会启动对应的流程实例。',
          '验证：启用后建一条匹配的业务数据，去"工作流实例"看执行状态和待办任务，去"工作流审计"看流转记录。',
        ]}
      />

      <WorkflowTemplateCatalog />

      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card className="enterprise-card">
            <Statistic
              title="工作流总数"
              value={stats.total}
              prefix={<GitBranch className="w-5 h-5" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card className="enterprise-card">
            <Statistic
              title="已启用"
              value={stats.active}
              prefix={<Play className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card className="enterprise-card">
            <Statistic
              title="草稿"
              value={stats.draft}
              prefix={<Edit className="w-5 h-5" />}
              styles={{ content: { color: '#722ed1' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索和过滤 */}
      <Card>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={8}>
            <Input
              placeholder="搜索工作流名称或描述..."
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder="工作流类型"
              value={typeFilter}
              onChange={setTypeFilter}
              style={{ width: '100%' }}
              options={[{ value: 'all', label: '全部类型' }, ...Object.entries(WORKFLOW_TYPE_CONFIG).map(([key, config]) => ({ value: key, label: config.label }))]}
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder="状态"
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: '100%' }}
              options={[{ value: 'all', label: '全部状态' }, ...Object.entries(STATUS_CONFIG).map(([key, config]) => ({ value: key, label: config.label }))]}
            />
          </Col>
          <Col xs={24} md={8} className="text-right">
            <Space>
              {selectedRowKeys.length > 0 && (
                <Popconfirm
                  title="确定要批量删除选中的工作流吗？"
                  onConfirm={handleBatchDelete}
                  okText="确定删除"
                  cancelText="取消"
                  okType="danger"
                >
                  <Button danger icon={<Trash2 className="w-4 h-4" />}>
                    批量删除 ({selectedRowKeys.length})
                  </Button>
                </Popconfirm>
              )}
              <Button
                icon={<Settings className="w-4 h-4" />}
                onClick={() => router.push('/admin/process-routing')}
              >
                绑定规则
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
                创建工作流
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 工作流列表 */}
      <Card className="enterprise-card">
        {filteredWorkflows.length === 0 && !loading ? (
          <Empty description="暂无工作流，点击右上角按钮创建" />
        ) : (
          <Table
            columns={columns}
            dataSource={filteredWorkflows}
            rowKey="id"
            rowSelection={rowSelection}
            loading={loading}
            scroll={{ x: 1200 }}
            pagination={{
              total: filteredWorkflows.length,
              pageSize: 10,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: total => `共 ${total} 条记录`,
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
            {selectedWorkflow ? '编辑工作流' : '创建工作流'}
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
        confirmLoading={loading}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="工作流名称"
                name="name"
                rules={[{ required: true, message: '请输入工作流名称' }]}
              >
                <Input placeholder="请输入工作流名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="工作流类型"
                name="type"
                rules={[{ required: true, message: '请选择工作流类型' }]}
              >
                <Select placeholder="选择工作流类型" options={Object.entries(WORKFLOW_TYPE_CONFIG).map(([key, config]) => ({ value: key, label: config.label }))} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            label="描述"
            name="description"
            rules={[{ required: true, message: '请输入工作流描述' }]}
          >
            <Input.TextArea rows={3} placeholder="请输入工作流描述" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情模态框 */}
      <Modal
        title={
          <div className="flex items-center gap-2">
            <Eye className="w-5 h-5" />
            <span>工作流详情</span>
          </div>
        }
        open={showDetailModal}
        onCancel={() => setShowDetailModal(false)}
        width={800}
        footer={[
          <Button key="close" onClick={() => setShowDetailModal(false)}>
            关闭
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
              <Title level={5}>基本信息</Title>
              <Row gutter={[16, 8]}>
                <Col span={12}>
                  <Text strong>工作流类型：</Text>
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
                  <Text strong>当前状态：</Text>
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
                  <Text strong>版本：</Text>
                  <Text>v{selectedWorkflow.version}</Text>
                </Col>
                <Col span={12}>
                  <Text strong>流程标识：</Text>
                  <Text copyable>{selectedWorkflow.id}</Text>
                </Col>
                <Col span={12}>
                  <Text strong>创建时间：</Text>
                  <Text>{selectedWorkflow.createdAt || '-'}</Text>
                </Col>
                <Col span={12}>
                  <Text strong>最后修改：</Text>
                  <Text>{selectedWorkflow.lastModified || '-'}</Text>
                </Col>
              </Row>
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
                打开工作流设计器
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default WorkflowManagement;
