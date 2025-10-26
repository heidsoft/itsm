'use client';

import {
  Pause,
  Play,
  Copy,
  AlertCircle,
  GitBranch,
  BarChart3,
  Activity,
  Users,
  Settings,
  CheckCircle,
  Edit,
  Eye,
  Trash2,
  Search,
  Plus,
} from 'lucide-react';
import React, { useState } from 'react';
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
  message,
  Badge,
  Progress,
  Alert,
} from 'antd';
const { Title, Text } = Typography;
const { Option } = Select;

// 工作流状态枚举
const WORKFLOW_STATUS = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  DRAFT: 'draft',
} as const;

// 工作流类型枚举
const WORKFLOW_TYPES = {
  INCIDENT: 'incident',
  SERVICE_REQUEST: 'service_request',
  CHANGE: 'change',
  PROBLEM: 'problem',
  APPROVAL: 'approval',
} as const;

// 工作流数据类型
interface Workflow {
  id: number;
  name: string;
  description: string;
  type: string;
  status: string;
  version: string;
  createdBy: string;
  createdAt: string;
  lastModified: string;
  stepsCount: number;
  activeInstances: number;
  completedInstances: number;
}

// 模拟工作流数据
const mockWorkflows: Workflow[] = [
  {
    id: 1,
    name: '事件处理流程',
    description: '标准事件处理和解决流程',
    type: WORKFLOW_TYPES.INCIDENT,
    status: WORKFLOW_STATUS.ACTIVE,
    version: 'v1.2',
    createdBy: '系统管理员',
    createdAt: '2024-01-10 14:30',
    lastModified: '2024-01-15 09:15',
    stepsCount: 5,
    activeInstances: 23,
    completedInstances: 156,
  },
  {
    id: 2,
    name: '服务请求审批流程',
    description: '服务请求的多级审批流程',
    type: WORKFLOW_TYPES.SERVICE_REQUEST,
    status: WORKFLOW_STATUS.ACTIVE,
    version: 'v2.0',
    createdBy: '流程管理员',
    createdAt: '2024-01-08 16:45',
    lastModified: '2024-01-14 11:20',
    stepsCount: 7,
    activeInstances: 12,
    completedInstances: 89,
  },
  {
    id: 3,
    name: '变更管理流程',
    description: 'IT变更的评估、审批和实施流程',
    type: WORKFLOW_TYPES.CHANGE,
    status: WORKFLOW_STATUS.DRAFT,
    version: 'v1.0',
    createdBy: '变更管理员',
    createdAt: '2024-01-12 10:00',
    lastModified: '2024-01-12 10:00',
    stepsCount: 8,
    activeInstances: 0,
    completedInstances: 0,
  },
  {
    id: 4,
    name: '问题调查流程',
    description: '根本原因分析和问题解决流程',
    type: WORKFLOW_TYPES.PROBLEM,
    status: WORKFLOW_STATUS.ACTIVE,
    version: 'v1.1',
    createdBy: '问题管理员',
    createdAt: '2024-01-05 11:30',
    lastModified: '2024-01-10 16:45',
    stepsCount: 6,
    activeInstances: 8,
    completedInstances: 45,
  },
  {
    id: 5,
    name: '采购审批流程',
    description: 'IT设备和服务采购的多级审批',
    type: WORKFLOW_TYPES.APPROVAL,
    status: WORKFLOW_STATUS.INACTIVE,
    version: 'v1.0',
    createdBy: '采购管理员',
    createdAt: '2024-01-01 09:00',
    lastModified: '2024-01-20 14:30',
    stepsCount: 4,
    activeInstances: 0,
    completedInstances: 25,
  },
];

// 工作流类型配置
const WORKFLOW_TYPE_CONFIG = {
  [WORKFLOW_TYPES.INCIDENT]: {
    label: '事件管理',
    color: 'red',
    icon: <AlertCircle className='w-3 h-3' />,
  },
  [WORKFLOW_TYPES.SERVICE_REQUEST]: {
    label: '服务请求',
    color: 'blue',
    icon: <Users className='w-3 h-3' />,
  },
  [WORKFLOW_TYPES.CHANGE]: {
    label: '变更管理',
    color: 'orange',
    icon: <GitBranch className='w-3 h-3' />,
  },
  [WORKFLOW_TYPES.PROBLEM]: {
    label: '问题管理',
    color: 'purple',
    icon: <Settings className='w-3 h-3' />,
  },
  [WORKFLOW_TYPES.APPROVAL]: {
    label: '审批流程',
    color: 'green',
    icon: <CheckCircle className='w-3 h-3' />,
  },
};

// 工作流状态配置
const STATUS_CONFIG = {
  [WORKFLOW_STATUS.ACTIVE]: {
    label: '已启用',
    color: 'success',
    icon: <Play className='w-3 h-3' />,
  },
  [WORKFLOW_STATUS.INACTIVE]: {
    label: '已停用',
    color: 'default',
    icon: <Pause className='w-3 h-3' />,
  },
  [WORKFLOW_STATUS.DRAFT]: {
    label: '草稿',
    color: 'processing',
    icon: <Edit className='w-3 h-3' />,
  },
};

const WorkflowManagement = () => {
  const [workflows, setWorkflows] = useState<Workflow[]>(mockWorkflows);
  const [searchTerm, setSearchTerm] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showDetailModal, setShowDetailModal] = useState(false);
  const [selectedWorkflow, setSelectedWorkflow] = useState<Workflow | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  // 统计信息
  const stats = {
    total: workflows.length,
    active: workflows.filter(w => w.status === WORKFLOW_STATUS.ACTIVE).length,
    draft: workflows.filter(w => w.status === WORKFLOW_STATUS.DRAFT).length,
    totalInstances: workflows.reduce((sum, w) => sum + w.activeInstances, 0),
    avgSteps: Math.round(workflows.reduce((sum, w) => sum + w.stepsCount, 0) / workflows.length),
  };

  // 获取所有工作流类型
  const workflowTypes = Array.from(new Set(workflows.map(workflow => workflow.type)));

  // 过滤工作流
  const filteredWorkflows = workflows.filter(workflow => {
    const matchesSearch =
      workflow.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      workflow.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesType = typeFilter === 'all' || workflow.type === typeFilter;
    const matchesStatus = statusFilter === 'all' || workflow.status === statusFilter;

    return matchesSearch && matchesType && matchesStatus;
  });

  // 处理工作流状态切换
  const handleStatusToggle = (workflowId: number) => {
    setWorkflows(prev =>
      prev.map(workflow => {
        if (workflow.id === workflowId) {
          const newStatus =
            workflow.status === WORKFLOW_STATUS.ACTIVE
              ? WORKFLOW_STATUS.INACTIVE
              : WORKFLOW_STATUS.ACTIVE;
          return {
            ...workflow,
            status: newStatus,
            lastModified: new Date().toLocaleString('zh-CN'),
          };
        }
        return workflow;
      })
    );
    message.success('工作流状态已更新');
  };

  // 处理工作流复制
  const handleDuplicate = (workflow: Workflow) => {
    const newWorkflow: Workflow = {
      ...workflow,
      id: Math.max(...workflows.map(w => w.id)) + 1,
      name: `${workflow.name} (副本)`,
      status: WORKFLOW_STATUS.DRAFT,
      version: 'v1.0',
      createdAt: new Date().toLocaleString('zh-CN'),
      lastModified: new Date().toLocaleString('zh-CN'),
      activeInstances: 0,
      completedInstances: 0,
    };
    setWorkflows(prev => [newWorkflow, ...prev]);
    message.success('工作流已复制');
  };

  // 处理工作流删除
  const handleDelete = (workflowId: number) => {
    setWorkflows(prev => prev.filter(w => w.id !== workflowId));
    message.success('工作流已删除');
  };

  // 批量删除
  const handleBatchDelete = () => {
    setWorkflows(prev => prev.filter(workflow => !selectedRowKeys.includes(workflow.id)));
    setSelectedRowKeys([]);
    message.success(`已删除 ${selectedRowKeys.length} 个工作流`);
  };

  // 查看详情
  const handleViewDetail = (workflow: Workflow) => {
    setSelectedWorkflow(workflow);
    setShowDetailModal(true);
  };

  // 保存工作流
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      await new Promise(resolve => setTimeout(resolve, 1000));

      if (selectedWorkflow) {
        // 编辑
        setWorkflows(prev =>
          prev.map(workflow =>
            workflow.id === selectedWorkflow.id
              ? {
                  ...workflow,
                  ...values,
                  lastModified: new Date().toLocaleString('zh-CN'),
                }
              : workflow
          )
        );
        message.success('工作流更新成功');
      } else {
        // 新建
        const newWorkflow: Workflow = {
          id: Math.max(...workflows.map(w => w.id)) + 1,
          ...values,
          status: WORKFLOW_STATUS.DRAFT,
          version: 'v1.0',
          createdBy: '当前用户',
          createdAt: new Date().toLocaleString('zh-CN'),
          lastModified: new Date().toLocaleString('zh-CN'),
          stepsCount: 0,
          activeInstances: 0,
          completedInstances: 0,
        };
        setWorkflows(prev => [newWorkflow, ...prev]);
        message.success('工作流创建成功');
      }

      setShowCreateModal(false);
      setSelectedWorkflow(null);
      form.resetFields();
    } catch (error) {
      message.error('保存失败，请检查必填项');
    } finally {
      setLoading(false);
    }
  };

  // 行选择配置
  const rowSelection = {
    selectedRowKeys,
    onChange: setSelectedRowKeys,
    getCheckboxProps: (record: Workflow) => ({
      disabled: record.status === WORKFLOW_STATUS.ACTIVE, // 活跃工作流不能批量删除
    }),
  };

  // 表格列定义
  const columns = [
    {
      title: '工作流信息',
      dataIndex: 'name',
      key: 'name',
      render: (_: unknown, record: Workflow) => (
        <div>
          <div className='flex items-center gap-2'>
            <Text strong>{record.name}</Text>
            <Badge count={record.version} color='blue' />
          </div>
          <Text type='secondary' className='text-sm'>
            {record.description}
          </Text>
          <div className='flex items-center gap-4 mt-1'>
            <span className='text-xs text-gray-500'>创建者: {record.createdBy}</span>
            <span className='text-xs text-gray-500'>步骤: {record.stepsCount}</span>
          </div>
        </div>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      align: 'center' as const,
      render: (type: string) => (
        <Tag
          color={WORKFLOW_TYPE_CONFIG[type as keyof typeof WORKFLOW_TYPE_CONFIG]?.color}
          icon={WORKFLOW_TYPE_CONFIG[type as keyof typeof WORKFLOW_TYPE_CONFIG]?.icon}
        >
          {WORKFLOW_TYPE_CONFIG[type as keyof typeof WORKFLOW_TYPE_CONFIG]?.label}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      align: 'center' as const,
      render: (status: string) => (
        <Tag
          color={STATUS_CONFIG[status as keyof typeof STATUS_CONFIG]?.color}
          icon={STATUS_CONFIG[status as keyof typeof STATUS_CONFIG]?.icon}
        >
          {STATUS_CONFIG[status as keyof typeof STATUS_CONFIG]?.label}
        </Tag>
      ),
    },
    {
      title: '实例统计',
      key: 'instances',
      align: 'center' as const,
      render: (_: unknown, record: Workflow) => (
        <div className='text-center'>
          <div className='text-lg font-bold text-blue-600'>{record.activeInstances}</div>
          <div className='text-xs text-gray-500'>活跃实例</div>
          <div className='text-xs text-gray-500'>已完成: {record.completedInstances}</div>
        </div>
      ),
    },
    {
      title: '效率指标',
      key: 'efficiency',
      align: 'center' as const,
      render: (_: unknown, record: Workflow) => {
        const completionRate =
          record.completedInstances > 0
            ? Math.round(
                (record.completedInstances / (record.completedInstances + record.activeInstances)) *
                  100
              )
            : 0;
        return (
          <div className='text-center'>
            <Progress
              type='circle'
              size={50}
              percent={completionRate}
              format={percent => `${percent}%`}
            />
            <div className='text-xs text-gray-500 mt-1'>完成率</div>
          </div>
        );
      },
    },
    {
      title: '最后修改',
      dataIndex: 'lastModified',
      key: 'lastModified',
      align: 'center' as const,
      render: (date: string) => (
        <div className='text-center'>
          <div className='text-sm'>{date.split(' ')[0]}</div>
          <div className='text-xs text-gray-500'>{date.split(' ')[1]}</div>
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      align: 'center' as const,
      render: (_: unknown, record: Workflow) => (
        <Space>
          <Tooltip title='查看详情'>
            <Button
              type='text'
              icon={<Eye className='w-4 h-4' />}
              onClick={() => handleViewDetail(record)}
            />
          </Tooltip>
          <Tooltip title='编辑'>
            <Button
              type='text'
              icon={<Edit className='w-4 h-4' />}
              onClick={() => {
                setSelectedWorkflow(record);
                form.setFieldsValue(record);
                setShowCreateModal(true);
              }}
            />
          </Tooltip>
          <Tooltip title='复制'>
            <Button
              type='text'
              icon={<Copy className='w-4 h-4' />}
              onClick={() => handleDuplicate(record)}
            />
          </Tooltip>
          <Tooltip title={record.status === WORKFLOW_STATUS.ACTIVE ? '停用' : '启用'}>
            <Button
              type='text'
              icon={
                record.status === WORKFLOW_STATUS.ACTIVE ? (
                  <Pause className='w-4 h-4' />
                ) : (
                  <Play className='w-4 h-4' />
                )
              }
              onClick={() => handleStatusToggle(record.id)}
            />
          </Tooltip>
          <Popconfirm
            title='确定要删除这个工作流吗？'
            description='删除后无法恢复，相关的实例将被停止。'
            onConfirm={() => handleDelete(record.id)}
            okText='确定删除'
            cancelText='取消'
            okType='danger'
          >
            <Button type='text' danger icon={<Trash2 className='w-4 h-4' />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className='p-6'>
      {/* 页面标题 */}
      <div className='mb-6'>
        <Title level={2} className='!mb-2'>
          <GitBranch className='inline-block w-6 h-6 mr-2' />
          工作流管理
        </Title>
        <Text type='secondary'>设计和管理业务流程，配置审批节点和自动化规则</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className='mb-6'>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='工作流总数'
              value={stats.total}
              prefix={<GitBranch className='w-5 h-5' />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='已启用'
              value={stats.active}
              prefix={<Play className='w-5 h-5' />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='活跃实例'
              value={stats.totalInstances}
              prefix={<Activity className='w-5 h-5' />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='平均步骤数'
              value={stats.avgSteps}
              prefix={<BarChart3 className='w-5 h-5' />}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索和过滤 */}
      <Card className='mb-6'>
        <Row gutter={[16, 16]} align='middle'>
          <Col xs={24} md={8}>
            <Input
              placeholder='搜索工作流名称或描述...'
              prefix={<Search className='w-4 h-4 text-gray-400' />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder='工作流类型'
              value={typeFilter}
              onChange={setTypeFilter}
              style={{ width: '100%' }}
            >
              <Option value='all'>全部类型</Option>
              {Object.entries(WORKFLOW_TYPE_CONFIG).map(([key, config]) => (
                <Option key={key} value={key}>
                  {config.label}
                </Option>
              ))}
            </Select>
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder='状态'
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: '100%' }}
            >
              <Option value='all'>全部状态</Option>
              {Object.entries(STATUS_CONFIG).map(([key, config]) => (
                <Option key={key} value={key}>
                  {config.label}
                </Option>
              ))}
            </Select>
          </Col>
          <Col xs={24} md={8} className='text-right'>
            <Space>
              {selectedRowKeys.length > 0 && (
                <Popconfirm
                  title='确定要批量删除选中的工作流吗？'
                  onConfirm={handleBatchDelete}
                  okText='确定删除'
                  cancelText='取消'
                  okType='danger'
                >
                  <Button danger icon={<Trash2 className='w-4 h-4' />}>
                    批量删除 ({selectedRowKeys.length})
                  </Button>
                </Popconfirm>
              )}
              <Button
                type='primary'
                icon={<Plus className='w-4 h-4' />}
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
      <Card className='enterprise-card'>
        <Table
          columns={columns}
          dataSource={filteredWorkflows}
          rowKey='id'
          rowSelection={rowSelection}
          pagination={{
            total: filteredWorkflows.length,
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: total => `共 ${total} 条记录`,
          }}
          className='enterprise-table'
        />
      </Card>

      {/* 创建/编辑模态框 */}
      <Modal
        title={
          <span>
            <GitBranch className='w-4 h-4 mr-2' />
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
        okText='保存'
        cancelText='取消'
      >
        <Alert
          message='工作流设计器正在开发中'
          description='当前仅支持基本信息编辑，完整的可视化工作流设计器即将发布！'
          type='info'
          showIcon
          className='mb-4'
        />
        <Form form={form} layout='vertical' className='mt-4'>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label='工作流名称'
                name='name'
                rules={[{ required: true, message: '请输入工作流名称' }]}
              >
                <Input placeholder='请输入工作流名称' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label='工作流类型'
                name='type'
                rules={[{ required: true, message: '请选择工作流类型' }]}
              >
                <Select placeholder='选择工作流类型'>
                  {Object.entries(WORKFLOW_TYPE_CONFIG).map(([key, config]) => (
                    <Option key={key} value={key}>
                      {config.label}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            label='描述'
            name='description'
            rules={[{ required: true, message: '请输入工作流描述' }]}
          >
            <Input.TextArea rows={3} placeholder='请输入工作流描述' />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label='版本' name='version' initialValue='v1.0'>
                <Input placeholder='如: v1.0' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label='状态' name='status' initialValue={WORKFLOW_STATUS.DRAFT}>
                <Select>
                  {Object.entries(STATUS_CONFIG).map(([key, config]) => (
                    <Option key={key} value={key}>
                      {config.label}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      {/* 详情模态框 */}
      <Modal
        title={
          <div className='flex items-center gap-2'>
            <Eye className='w-5 h-5' />
            <span>工作流详情</span>
          </div>
        }
        open={showDetailModal}
        onCancel={() => setShowDetailModal(false)}
        width={800}
        footer={[
          <Button key='close' onClick={() => setShowDetailModal(false)}>
            关闭
          </Button>,
        ]}
      >
        {selectedWorkflow && (
          <div className='space-y-6'>
            <div>
              <Title level={4}>{selectedWorkflow.name}</Title>
              <Text type='secondary'>{selectedWorkflow.description}</Text>
            </div>

            <Row gutter={16}>
              <Col span={8}>
                <Card size='small'>
                  <Statistic
                    title='活跃实例'
                    value={selectedWorkflow.activeInstances}
                    prefix={<Activity className='w-4 h-4' />}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card size='small'>
                  <Statistic
                    title='已完成实例'
                    value={selectedWorkflow.completedInstances}
                    prefix={<CheckCircle className='w-4 h-4' />}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card size='small'>
                  <Statistic
                    title='步骤数'
                    value={selectedWorkflow.stepsCount}
                    prefix={<BarChart3 className='w-4 h-4' />}
                  />
                </Card>
              </Col>
            </Row>

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
                  <Text>{selectedWorkflow.version}</Text>
                </Col>
                <Col span={12}>
                  <Text strong>创建者：</Text>
                  <Text>{selectedWorkflow.createdBy}</Text>
                </Col>
                <Col span={12}>
                  <Text strong>创建时间：</Text>
                  <Text>{selectedWorkflow.createdAt}</Text>
                </Col>
                <Col span={12}>
                  <Text strong>最后修改：</Text>
                  <Text>{selectedWorkflow.lastModified}</Text>
                </Col>
              </Row>
            </div>

            <Alert
              message='工作流设计器预览'
              description='可视化工作流设计器和实例管理功能正在开发中，将在下个版本中提供完整的流程设计和监控能力。'
              type='warning'
              showIcon
            />
          </div>
        )}
      </Modal>
    </div>
  );
};

export default WorkflowManagement;
