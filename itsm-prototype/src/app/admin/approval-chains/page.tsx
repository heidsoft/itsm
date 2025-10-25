'use client';

import {
  FileText,
  Pause,
  Play,
  Copy,
  UserCheck,
  GitBranch,
  ArrowRight,
  Activity,
  Workflow,
  Timer,
  Settings,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
  Users,
  Eye,
  Edit,
  Plus,
  Search,
  Trash2,
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
  Modal,
  Form,
  Row,
  Col,
  Statistic,
  Badge,
  Tooltip,
  Popconfirm,
  Steps,
  Descriptions,
  Drawer,
  App,
  Tag,
} from 'antd';
const { Title, Text } = Typography;
const { Option } = Select;
const { Step } = Steps;

// 审批链状态枚举
const APPROVAL_CHAIN_STATUS = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  DRAFT: 'draft',
} as const;

// 审批类型枚举
const APPROVAL_TYPES = {
  SEQUENTIAL: 'sequential', // 顺序审批
  PARALLEL: 'parallel', // 并行审批
  CONDITIONAL: 'conditional', // 条件审批
  ESCALATION: 'escalation', // 升级审批
} as const;

// 审批节点类型
const NODE_TYPES = {
  USER: 'user', // 用户审批
  ROLE: 'role', // 角色审批
  GROUP: 'group', // 用户组审批
  AUTO: 'auto', // 自动审批
} as const;

// 模拟审批链数据
const mockApprovalChains = [
  {
    id: 1,
    name: '服务请求审批链',
    description: '标准服务请求的多级审批流程',
    type: APPROVAL_TYPES.SEQUENTIAL,
    status: APPROVAL_CHAIN_STATUS.ACTIVE,
    category: '服务管理',
    createdBy: '系统管理员',
    createdAt: '2024-01-10 14:30',
    lastModified: '2024-01-15 09:15',
    nodesCount: 4,
    usageCount: 156,
    avgApprovalTime: '2.5小时',
    nodes: [
      {
        id: 1,
        name: '直属主管审批',
        type: NODE_TYPES.ROLE,
        approver: '直属主管',
        condition: '金额 < 5000',
        timeout: 24,
        isRequired: true,
      },
      {
        id: 2,
        name: '部门经理审批',
        type: NODE_TYPES.ROLE,
        approver: '部门经理',
        condition: '金额 >= 5000',
        timeout: 48,
        isRequired: true,
      },
      {
        id: 3,
        name: 'IT部门审批',
        type: NODE_TYPES.GROUP,
        approver: 'IT部门',
        condition: '涉及IT资源',
        timeout: 72,
        isRequired: false,
      },
      {
        id: 4,
        name: '财务审批',
        type: NODE_TYPES.ROLE,
        approver: '财务经理',
        condition: '金额 >= 10000',
        timeout: 48,
        isRequired: true,
      },
    ],
  },
  {
    id: 2,
    name: '变更管理审批链',
    description: 'IT变更的风险评估和审批流程',
    type: APPROVAL_TYPES.CONDITIONAL,
    status: APPROVAL_CHAIN_STATUS.ACTIVE,
    category: '变更管理',
    createdBy: '变更管理员',
    createdAt: '2024-01-08 16:45',
    lastModified: '2024-01-14 11:20',
    nodesCount: 5,
    usageCount: 89,
    avgApprovalTime: '4.2小时',
    nodes: [
      {
        id: 1,
        name: '变更发起人确认',
        type: NODE_TYPES.USER,
        approver: '变更发起人',
        condition: '自动',
        timeout: 12,
        isRequired: true,
      },
      {
        id: 2,
        name: '技术负责人审批',
        type: NODE_TYPES.ROLE,
        approver: '技术负责人',
        condition: '技术变更',
        timeout: 24,
        isRequired: true,
      },
      {
        id: 3,
        name: 'CAB委员会审批',
        type: NODE_TYPES.GROUP,
        approver: 'CAB委员会',
        condition: '高风险变更',
        timeout: 72,
        isRequired: true,
      },
    ],
  },
  {
    id: 3,
    name: '采购申请审批链',
    description: '设备和软件采购的审批流程',
    type: APPROVAL_TYPES.PARALLEL,
    status: APPROVAL_CHAIN_STATUS.DRAFT,
    category: '采购管理',
    createdBy: '采购管理员',
    createdAt: '2024-01-12 10:00',
    lastModified: '2024-01-12 10:00',
    nodesCount: 3,
    usageCount: 0,
    avgApprovalTime: '0小时',
    nodes: [
      {
        id: 1,
        name: '预算审批',
        type: NODE_TYPES.ROLE,
        approver: '财务经理',
        condition: '并行',
        timeout: 48,
        isRequired: true,
      },
      {
        id: 2,
        name: '技术审批',
        type: NODE_TYPES.ROLE,
        approver: '技术经理',
        condition: '并行',
        timeout: 48,
        isRequired: true,
      },
    ],
  },
];

// 审批类型配置
const APPROVAL_TYPE_CONFIG = {
  [APPROVAL_TYPES.SEQUENTIAL]: {
    label: '顺序审批',
    color: 'blue',
    icon: <ArrowRight className='w-4 h-4' />,
    description: '按顺序逐级审批',
  },
  [APPROVAL_TYPES.PARALLEL]: {
    label: '并行审批',
    color: 'green',
    icon: <GitBranch className='w-4 h-4' />,
    description: '多个审批人同时审批',
  },
  [APPROVAL_TYPES.CONDITIONAL]: {
    label: '条件审批',
    color: 'purple',
    icon: <Settings className='w-4 h-4' />,
    description: '根据条件动态审批',
  },
  [APPROVAL_TYPES.ESCALATION]: {
    label: '升级审批',
    color: 'orange',
    icon: <AlertTriangle className='w-4 h-4' />,
    description: '超时自动升级审批',
  },
};

// 状态配置
const STATUS_CONFIG = {
  [APPROVAL_CHAIN_STATUS.ACTIVE]: {
    label: '已启用',
    color: 'success',
    icon: <CheckCircle className='w-3 h-3' />,
  },
  [APPROVAL_CHAIN_STATUS.INACTIVE]: {
    label: '已停用',
    color: 'default',
    icon: <XCircle className='w-3 h-3' />,
  },
  [APPROVAL_CHAIN_STATUS.DRAFT]: {
    label: '草稿',
    color: 'warning',
    icon: <Clock className='w-3 h-3' />,
  },
};

// 节点类型配置
const NODE_TYPE_CONFIG = {
  [NODE_TYPES.USER]: {
    label: '用户审批',
    color: 'blue',
    icon: <UserCheck className='w-3 h-3' />,
  },
  [NODE_TYPES.ROLE]: {
    label: '角色审批',
    color: 'purple',
    icon: <Users className='w-3 h-3' />,
  },
  [NODE_TYPES.GROUP]: {
    label: '用户组审批',
    color: 'green',
    icon: <Users className='w-3 h-3' />,
  },
  [NODE_TYPES.AUTO]: {
    label: '自动审批',
    color: 'cyan',
    icon: <Settings className='w-3 h-3' />,
  },
};

const ApprovalChainManagement = () => {
  const { message } = App.useApp();
  const [approvalChains, setApprovalChains] = useState(mockApprovalChains);
  const [searchTerm, setSearchTerm] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [categoryFilter, setCategoryFilter] = useState('all');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedChain, setSelectedChain] = useState<any>(null);
  const [showDetailDrawer, setShowDetailDrawer] = useState(false);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  // 统计信息
  const stats = {
    total: approvalChains.length,
    active: approvalChains.filter(c => c.status === APPROVAL_CHAIN_STATUS.ACTIVE).length,
    draft: approvalChains.filter(c => c.status === APPROVAL_CHAIN_STATUS.DRAFT).length,
    totalUsage: approvalChains.reduce((sum, c) => sum + c.usageCount, 0),
  };

  // 获取所有分类
  const categories = Array.from(new Set(approvalChains.map(chain => chain.category)));

  // 过滤审批链
  const filteredChains = approvalChains.filter(chain => {
    const matchesSearch =
      chain.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      chain.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesType = typeFilter === 'all' || chain.type === typeFilter;
    const matchesStatus = statusFilter === 'all' || chain.status === statusFilter;
    const matchesCategory = categoryFilter === 'all' || chain.category === categoryFilter;

    return matchesSearch && matchesType && matchesStatus && matchesCategory;
  });

  // 处理状态切换
  const handleStatusToggle = (chainId: number) => {
    setApprovalChains(prev =>
      prev.map(chain => {
        if (chain.id === chainId) {
          const newStatus =
            chain.status === APPROVAL_CHAIN_STATUS.ACTIVE
              ? APPROVAL_CHAIN_STATUS.INACTIVE
              : APPROVAL_CHAIN_STATUS.ACTIVE;
          return { ...chain, status: newStatus };
        }
        return chain;
      })
    );
    message.success('审批链状态已更新');
  };

  // 处理复制
  const handleDuplicate = (chain: unknown) => {
    const newChain = {
      ...chain,
      id: Math.max(...approvalChains.map(c => c.id)) + 1,
      name: `${chain.name} (副本)`,
      status: APPROVAL_CHAIN_STATUS.DRAFT,
      createdAt: new Date().toLocaleString('zh-CN'),
      lastModified: new Date().toLocaleString('zh-CN'),
      usageCount: 0,
    };
    setApprovalChains(prev => [newChain, ...prev]);
    message.success('审批链已复制');
  };

  // 处理删除
  const handleDelete = (chainId: number) => {
    setApprovalChains(prev => prev.filter(c => c.id !== chainId));
    message.success('审批链已删除');
  };

  // 查看详情
  const handleViewDetail = (chain: unknown) => {
    setSelectedChain(chain);
    setShowDetailDrawer(true);
  };

  // 保存审批链
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      await new Promise(resolve => setTimeout(resolve, 1000));

      if (selectedChain) {
        // 编辑
        setApprovalChains(prev =>
          prev.map(chain =>
            chain.id === selectedChain.id
              ? {
                  ...chain,
                  ...values,
                  lastModified: new Date().toLocaleString('zh-CN'),
                }
              : chain
          )
        );
        message.success('审批链更新成功');
      } else {
        // 新建
        const newChain = {
          id: Math.max(...approvalChains.map(c => c.id)) + 1,
          ...values,
          status: APPROVAL_CHAIN_STATUS.DRAFT,
          createdBy: '当前用户',
          createdAt: new Date().toLocaleString('zh-CN'),
          lastModified: new Date().toLocaleString('zh-CN'),
          nodesCount: 0,
          usageCount: 0,
          avgApprovalTime: '0小时',
          nodes: [],
        };
        setApprovalChains(prev => [newChain, ...prev]);
        message.success('审批链创建成功');
      }

      setShowCreateModal(false);
      setSelectedChain(null);
      form.resetFields();
    } catch (error) {
      message.error('保存失败，请检查必填项');
    } finally {
      setLoading(false);
    }
  };

  // 渲染审批节点流程
  const renderApprovalFlow = (nodes: unknown[], type: string) => {
    if (!nodes || nodes.length === 0) {
      return <Text type='secondary'>暂无审批节点</Text>;
    }

    if (type === APPROVAL_TYPES.SEQUENTIAL) {
      return (
        <Steps direction='vertical' size='small'>
          {nodes.map((node, index) => (
            <Step
              key={node.id}
              title={
                <div className='flex items-center gap-2'>
                  <span>{node.name}</span>
                  <Tag
                    color={NODE_TYPE_CONFIG[node.type as keyof typeof NODE_TYPE_CONFIG]?.color}
                    size='small'
                  >
                    {NODE_TYPE_CONFIG[node.type as keyof typeof NODE_TYPE_CONFIG]?.label}
                  </Tag>
                  {node.isRequired && <Badge status='error' text='必须' />}
                </div>
              }
              description={
                <div className='text-xs text-gray-500'>
                  <div>审批人: {node.approver}</div>
                  <div>条件: {node.condition}</div>
                  <div>超时: {node.timeout}小时</div>
                </div>
              }
              status={index === 0 ? 'process' : 'wait'}
            />
          ))}
        </Steps>
      );
    }

    return (
      <div className='space-y-2'>
        {nodes.map(node => (
          <Card key={node.id} size='small' className='border-l-4 border-l-blue-500'>
            <div className='flex items-center justify-between'>
              <div className='flex items-center gap-2'>
                <span className='font-medium'>{node.name}</span>
                <Tag
                  color={NODE_TYPE_CONFIG[node.type as keyof typeof NODE_TYPE_CONFIG]?.color}
                  size='small'
                >
                  {NODE_TYPE_CONFIG[node.type as keyof typeof NODE_TYPE_CONFIG]?.label}
                </Tag>
              </div>
              <div className='text-xs text-gray-500'>{node.timeout}h</div>
            </div>
            <div className='text-xs text-gray-500 mt-1'>
              {node.approver} · {node.condition}
            </div>
          </Card>
        ))}
      </div>
    );
  };

  // 表格列定义
  const columns = [
    {
      title: '审批链信息',
      dataIndex: 'name',
      key: 'name',
      render: (_: unknown, record: unknown) => (
        <div>
          <div className='flex items-center gap-2'>
            <Text strong>{record.name}</Text>
            <Tag
              color={APPROVAL_TYPE_CONFIG[record.type as keyof typeof APPROVAL_TYPE_CONFIG]?.color}
              icon={APPROVAL_TYPE_CONFIG[record.type as keyof typeof APPROVAL_TYPE_CONFIG]?.icon}
            >
              {APPROVAL_TYPE_CONFIG[record.type as keyof typeof APPROVAL_TYPE_CONFIG]?.label}
            </Tag>
          </div>
          <Text type='secondary' className='text-sm'>
            {record.description}
          </Text>
          <div className='flex items-center gap-4 mt-1'>
            <span className='text-xs text-gray-500'>分类: {record.category}</span>
            <span className='text-xs text-gray-500'>节点: {record.nodesCount}</span>
          </div>
        </div>
      ),
    },
    {
      title: '使用统计',
      dataIndex: 'usageCount',
      key: 'usageCount',
      align: 'center' as const,
      render: (_: unknown, record: unknown) => (
        <div className='text-center'>
          <div className='text-lg font-bold text-blue-600'>{record.usageCount}</div>
          <div className='text-xs text-gray-500'>次使用</div>
          <div className='text-xs text-gray-500'>平均 {record.avgApprovalTime}</div>
        </div>
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
      title: '更新信息',
      dataIndex: 'lastModified',
      key: 'lastModified',
      align: 'center' as const,
      render: (_: unknown, record: unknown) => (
        <div className='text-center'>
          <div className='text-sm'>{record.lastModified}</div>
          <div className='text-xs text-gray-500'>由 {record.createdBy}</div>
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      align: 'center' as const,
      render: (_: unknown, record: unknown) => (
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
                setSelectedChain(record);
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
          <Tooltip title={record.status === APPROVAL_CHAIN_STATUS.ACTIVE ? '停用' : '启用'}>
            <Button
              type='text'
              icon={
                record.status === APPROVAL_CHAIN_STATUS.ACTIVE ? (
                  <Pause className='w-4 h-4' />
                ) : (
                  <Play className='w-4 h-4' />
                )
              }
              onClick={() => handleStatusToggle(record.id)}
            />
          </Tooltip>
          <Popconfirm
            title='确定要删除这个审批链吗？'
            description='删除后无法恢复，正在使用的流程将受到影响。'
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
          <FileText className='inline-block w-6 h-6 mr-2' />
          审批链配置
        </Title>
        <Text type='secondary'>设计和管理审批流程，配置多级审批节点和条件规则</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className='mb-6'>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='审批链总数'
              value={stats.total}
              prefix={<FileText className='w-5 h-5' />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='已启用'
              value={stats.active}
              prefix={<CheckCircle className='w-5 h-5' />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='草稿状态'
              value={stats.draft}
              prefix={<Clock className='w-5 h-5' />}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='总使用次数'
              value={stats.totalUsage}
              prefix={<Activity className='w-5 h-5' />}
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
              placeholder='搜索审批链名称或描述...'
              prefix={<Search className='w-4 h-4 text-gray-400' />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder='审批类型'
              value={typeFilter}
              onChange={setTypeFilter}
              style={{ width: '100%' }}
            >
              <Option value='all'>全部类型</Option>
              {Object.entries(APPROVAL_TYPE_CONFIG).map(([key, config]) => (
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
          <Col xs={24} md={4}>
            <Select
              placeholder='分类'
              value={categoryFilter}
              onChange={setCategoryFilter}
              style={{ width: '100%' }}
            >
              <Option value='all'>全部分类</Option>
              {categories.map(category => (
                <Option key={category} value={category}>
                  {category}
                </Option>
              ))}
            </Select>
          </Col>
          <Col xs={24} md={4} className='text-right'>
            <Button
              type='primary'
              icon={<Plus className='w-4 h-4' />}
              onClick={() => {
                setSelectedChain(null);
                form.resetFields();
                setShowCreateModal(true);
              }}
            >
              新建审批链
            </Button>
          </Col>
        </Row>
      </Card>

      {/* 审批链列表 */}
      <Card className='enterprise-card'>
        <Table
          columns={columns}
          dataSource={filteredChains}
          rowKey='id'
          pagination={{
            total: filteredChains.length,
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
            <FileText className='w-4 h-4 mr-2' />
            {selectedChain ? '编辑审批链' : '新建审批链'}
          </span>
        }
        open={showCreateModal}
        onOk={handleSave}
        onCancel={() => {
          setShowCreateModal(false);
          setSelectedChain(null);
          form.resetFields();
        }}
        width={600}
        confirmLoading={loading}
        okText='保存'
        cancelText='取消'
      >
        <Form form={form} layout='vertical' className='mt-4'>
          <Form.Item
            label='审批链名称'
            name='name'
            rules={[{ required: true, message: '请输入审批链名称' }]}
          >
            <Input placeholder='请输入审批链名称' />
          </Form.Item>
          <Form.Item
            label='描述'
            name='description'
            rules={[{ required: true, message: '请输入审批链描述' }]}
          >
            <Input.TextArea rows={3} placeholder='请输入审批链描述' />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label='审批类型'
                name='type'
                rules={[{ required: true, message: '请选择审批类型' }]}
                initialValue={APPROVAL_TYPES.SEQUENTIAL}
              >
                <Select>
                  {Object.entries(APPROVAL_TYPE_CONFIG).map(([key, config]) => (
                    <Option key={key} value={key}>
                      <div className='flex items-center gap-2'>
                        {config.icon}
                        <span>{config.label}</span>
                      </div>
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label='分类'
                name='category'
                rules={[{ required: true, message: '请输入分类' }]}
              >
                <Select showSearch placeholder='选择或输入分类' mode='combobox'>
                  {categories.map(category => (
                    <Option key={category} value={category}>
                      {category}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title={
          <div className='flex items-center gap-2'>
            <Eye className='w-5 h-5' />
            <span>审批链详情</span>
          </div>
        }
        placement='right'
        size='large'
        open={showDetailDrawer}
        onClose={() => setShowDetailDrawer(false)}
      >
        {selectedChain && (
          <div className='space-y-6'>
            <Descriptions title='基本信息' bordered column={1} size='small'>
              <Descriptions.Item label='名称'>{selectedChain.name}</Descriptions.Item>
              <Descriptions.Item label='描述'>{selectedChain.description}</Descriptions.Item>
              <Descriptions.Item label='类型'>
                <Tag
                  color={
                    APPROVAL_TYPE_CONFIG[selectedChain.type as keyof typeof APPROVAL_TYPE_CONFIG]
                      ?.color
                  }
                  icon={
                    APPROVAL_TYPE_CONFIG[selectedChain.type as keyof typeof APPROVAL_TYPE_CONFIG]
                      ?.icon
                  }
                >
                  {
                    APPROVAL_TYPE_CONFIG[selectedChain.type as keyof typeof APPROVAL_TYPE_CONFIG]
                      ?.label
                  }
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label='状态'>
                <Tag
                  color={STATUS_CONFIG[selectedChain.status as keyof typeof STATUS_CONFIG]?.color}
                  icon={STATUS_CONFIG[selectedChain.status as keyof typeof STATUS_CONFIG]?.icon}
                >
                  {STATUS_CONFIG[selectedChain.status as keyof typeof STATUS_CONFIG]?.label}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label='分类'>{selectedChain.category}</Descriptions.Item>
              <Descriptions.Item label='创建者'>{selectedChain.createdBy}</Descriptions.Item>
              <Descriptions.Item label='创建时间'>{selectedChain.createdAt}</Descriptions.Item>
              <Descriptions.Item label='最后修改'>{selectedChain.lastModified}</Descriptions.Item>
            </Descriptions>

            <div>
              <Title level={4}>使用统计</Title>
              <Row gutter={16}>
                <Col span={8}>
                  <Statistic
                    title='使用次数'
                    value={selectedChain.usageCount}
                    prefix={<Activity className='w-4 h-4' />}
                  />
                </Col>
                <Col span={8}>
                  <Statistic
                    title='节点数量'
                    value={selectedChain.nodesCount}
                    prefix={<Workflow className='w-4 h-4' />}
                  />
                </Col>
                <Col span={8}>
                  <Statistic
                    title='平均审批时间'
                    value={selectedChain.avgApprovalTime}
                    prefix={<Timer className='w-4 h-4' />}
                  />
                </Col>
              </Row>
            </div>

            <div>
              <Title level={4}>审批流程</Title>
              {renderApprovalFlow(selectedChain.nodes, selectedChain.type)}
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default ApprovalChainManagement;
