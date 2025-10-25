'use client';

import {
  Copy,
  Zap,
  ArrowUp,
  Mail,
  Phone,
  Megaphone,
  Activity,
  Timer,
  CheckCircle,
  Clock,
  Edit,
  MessageSquare,
  Bell,
  Eye,
  Plus,
  Search,
  Trash2,
  Target,
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
  Tooltip,
  Popconfirm,
  Descriptions,
  Timeline,
  Drawer,
  App,
  Tag,
} from 'antd';
const { Title, Text } = Typography;
const { Option } = Select;

// 升级规则的数据类型
interface EscalationRule {
  id: string;
  name: string;
  description: string;
  triggerCondition: string;
  priority: 'P1' | 'P2' | 'P3' | 'P4';
  serviceType: string;
  escalationLevels: EscalationLevel[];
  status: 'active' | 'inactive' | 'draft';
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  usageCount: number;
  lastTriggered: string;
}

interface EscalationLevel {
  level: number;
  timeThreshold: string;
  escalateTo: string;
  notificationMethod: string[];
  action: string;
}

// 模拟升级规则数据
const mockEscalationRules: EscalationRule[] = [
  {
    id: 'ESC-001',
    name: 'P1事件升级规则',
    description: '针对P1级别紧急事件的自动升级规则，确保关键问题得到及时处理',
    triggerCondition: '工单状态为"进行中"且优先级为P1',
    priority: 'P1',
    serviceType: '关键业务系统',
    escalationLevels: [
      {
        level: 1,
        timeThreshold: '15分钟',
        escalateTo: '高级工程师',
        notificationMethod: ['邮件', '短信', '企业微信'],
        action: '自动分配给高级工程师组',
      },
      {
        level: 2,
        timeThreshold: '1小时',
        escalateTo: '技术经理',
        notificationMethod: ['邮件', '短信', '电话'],
        action: '通知技术经理并创建紧急会议',
      },
      {
        level: 3,
        timeThreshold: '4小时',
        escalateTo: 'CTO',
        notificationMethod: ['邮件', '电话'],
        action: '升级至CTO并启动应急响应流程',
      },
    ],
    status: 'active',
    createdAt: '2024-01-15',
    updatedAt: '2024-06-01',
    createdBy: '张三',
    usageCount: 45,
    lastTriggered: '2024-01-20 15:30',
  },
  {
    id: 'ESC-002',
    name: 'P2事件升级规则',
    description: '针对P2级别高优先级事件的升级规则',
    triggerCondition: '工单状态为"进行中"且优先级为P2',
    priority: 'P2',
    serviceType: '一般业务系统',
    escalationLevels: [
      {
        level: 1,
        timeThreshold: '30分钟',
        escalateTo: '高级工程师',
        notificationMethod: ['邮件', '企业微信'],
        action: '自动分配给高级工程师组',
      },
      {
        level: 2,
        timeThreshold: '2小时',
        escalateTo: '技术经理',
        notificationMethod: ['邮件', '短信'],
        action: '通知技术经理',
      },
    ],
    status: 'active',
    createdAt: '2024-01-20',
    updatedAt: '2024-05-15',
    createdBy: '李四',
    usageCount: 28,
    lastTriggered: '2024-01-18 09:15',
  },
  {
    id: 'ESC-003',
    name: '服务请求超时升级',
    description: '服务请求长时间未处理的升级规则',
    triggerCondition: '服务请求状态为"待处理"超过预定时间',
    priority: 'P3',
    serviceType: '服务请求',
    escalationLevels: [
      {
        level: 1,
        timeThreshold: '24小时',
        escalateTo: '服务台主管',
        notificationMethod: ['邮件'],
        action: '通知服务台主管',
      },
    ],
    status: 'draft',
    createdAt: '2024-01-25',
    updatedAt: '2024-01-25',
    createdBy: '王五',
    usageCount: 0,
    lastTriggered: '从未触发',
  },
];

// 优先级配置
const PRIORITY_CONFIG = {
  P1: { label: 'P1 - 紧急', color: 'error', bgColor: '#fff2f0' },
  P2: { label: 'P2 - 高', color: 'warning', bgColor: '#fff7e6' },
  P3: { label: 'P3 - 中', color: 'processing', bgColor: '#f0f9ff' },
  P4: { label: 'P4 - 低', color: 'default', bgColor: '#fafafa' },
};

// 状态配置
const STATUS_CONFIG = {
  active: {
    label: '已启用',
    color: 'success',
    icon: <CheckCircle className='w-3 h-3' />,
  },
  inactive: {
    label: '已停用',
    color: 'default',
    icon: <Clock className='w-3 h-3' />,
  },
  draft: {
    label: '草稿',
    color: 'warning',
    icon: <Edit className='w-3 h-3' />,
  },
};

// 通知方式配置
const NOTIFICATION_CONFIG = {
  邮件: { icon: <Mail className='w-3 h-3' />, color: 'blue' },
  短信: { icon: <MessageSquare className='w-3 h-3' />, color: 'green' },
  电话: { icon: <Phone className='w-3 h-3' />, color: 'red' },
  企业微信: { icon: <Bell className='w-3 h-3' />, color: 'cyan' },
  系统通知: { icon: <Megaphone className='w-3 h-3' />, color: 'purple' },
};

const EscalationRuleManagement = () => {
  const { message } = App.useApp();
  const [escalationRules, setEscalationRules] = useState(mockEscalationRules);
  const [searchTerm, setSearchTerm] = useState('');
  const [priorityFilter, setPriorityFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [serviceTypeFilter, setServiceTypeFilter] = useState('all');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedRule, setSelectedRule] = useState<EscalationRule | null>(null);
  const [showDetailDrawer, setShowDetailDrawer] = useState(false);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  // 统计信息
  const stats = {
    total: escalationRules.length,
    active: escalationRules.filter(rule => rule.status === 'active').length,
    draft: escalationRules.filter(rule => rule.status === 'draft').length,
    totalUsage: escalationRules.reduce((sum, rule) => sum + rule.usageCount, 0),
  };

  // 获取所有服务类型
  const serviceTypes = Array.from(new Set(escalationRules.map(rule => rule.serviceType)));

  // 过滤规则
  const filteredRules = escalationRules.filter(rule => {
    const matchesSearch =
      rule.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      rule.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesPriority = priorityFilter === 'all' || rule.priority === priorityFilter;
    const matchesStatus = statusFilter === 'all' || rule.status === statusFilter;
    const matchesServiceType =
      serviceTypeFilter === 'all' || rule.serviceType === serviceTypeFilter;

    return matchesSearch && matchesPriority && matchesStatus && matchesServiceType;
  });

  // 处理状态切换
  const handleStatusToggle = (ruleId: string) => {
    setEscalationRules(prev =>
      prev.map(rule => {
        if (rule.id === ruleId) {
          const newStatus = rule.status === 'active' ? 'inactive' : 'active';
          return { ...rule, status: newStatus };
        }
        return rule;
      })
    );
    message.success('升级规则状态已更新');
  };

  // 处理复制
  const handleDuplicate = (rule: EscalationRule) => {
    const newRule = {
      ...rule,
      id: `ESC-${String(escalationRules.length + 1).padStart(3, '0')}`,
      name: `${rule.name} (副本)`,
      status: 'draft' as const,
      createdAt: new Date().toLocaleDateString(),
      updatedAt: new Date().toLocaleDateString(),
      usageCount: 0,
      lastTriggered: '从未触发',
    };
    setEscalationRules(prev => [newRule, ...prev]);
    message.success('升级规则已复制');
  };

  // 处理删除
  const handleDelete = (ruleId: string) => {
    setEscalationRules(prev => prev.filter(rule => rule.id !== ruleId));
    message.success('升级规则已删除');
  };

  // 查看详情
  const handleViewDetail = (rule: EscalationRule) => {
    setSelectedRule(rule);
    setShowDetailDrawer(true);
  };

  // 保存规则
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      await new Promise(resolve => setTimeout(resolve, 1000));

      if (selectedRule) {
        // 编辑
        setEscalationRules(prev =>
          prev.map(rule =>
            rule.id === selectedRule.id
              ? {
                  ...rule,
                  ...values,
                  updatedAt: new Date().toLocaleDateString(),
                }
              : rule
          )
        );
        message.success('升级规则更新成功');
      } else {
        // 新建
        const newRule: EscalationRule = {
          id: `ESC-${String(escalationRules.length + 1).padStart(3, '0')}`,
          ...values,
          status: 'draft',
          createdBy: '当前用户',
          createdAt: new Date().toLocaleDateString(),
          updatedAt: new Date().toLocaleDateString(),
          usageCount: 0,
          lastTriggered: '从未触发',
          escalationLevels: [],
        };
        setEscalationRules(prev => [newRule, ...prev]);
        message.success('升级规则创建成功');
      }

      setShowCreateModal(false);
      setSelectedRule(null);
      form.resetFields();
    } catch (error) {
      message.error('保存失败，请检查必填项');
    } finally {
      setLoading(false);
    }
  };

  // 渲染升级级别
  const renderEscalationLevels = (levels: EscalationLevel[]) => {
    if (!levels || levels.length === 0) {
      return <Text type='secondary'>暂无升级级别</Text>;
    }

    return (
      <Timeline>
        {levels.map((level, index) => (
          <Timeline.Item
            key={level.level}
            color={index === 0 ? 'blue' : index === 1 ? 'orange' : 'red'}
            dot={<ArrowUp className='w-4 h-4' />}
          >
            <div className='space-y-2'>
              <div className='flex items-center gap-2'>
                <Text strong>级别 {level.level}</Text>
                <Tag color='blue'>{level.timeThreshold}</Tag>
              </div>
              <div className='text-sm text-gray-600'>
                <div>升级至: {level.escalateTo}</div>
                <div>操作: {level.action}</div>
                <div className='flex items-center gap-1 mt-1'>
                  <span>通知方式:</span>
                  {level.notificationMethod.map(method => (
                    <Tag
                      key={method}
                      size='small'
                      color={NOTIFICATION_CONFIG[method as keyof typeof NOTIFICATION_CONFIG]?.color}
                      icon={NOTIFICATION_CONFIG[method as keyof typeof NOTIFICATION_CONFIG]?.icon}
                    >
                      {method}
                    </Tag>
                  ))}
                </div>
              </div>
            </div>
          </Timeline.Item>
        ))}
      </Timeline>
    );
  };

  // 表格列定义
  const columns = [
    {
      title: '规则信息',
      dataIndex: 'name',
      key: 'name',
      render: (_: unknown, record: EscalationRule) => (
        <div>
          <div className='flex items-center gap-2'>
            <Text strong>{record.name}</Text>
            <Tag color={PRIORITY_CONFIG[record.priority]?.color}>
              {PRIORITY_CONFIG[record.priority]?.label}
            </Tag>
          </div>
          <Text type='secondary' className='text-sm'>
            {record.description}
          </Text>
          <div className='flex items-center gap-4 mt-1'>
            <span className='text-xs text-gray-500'>ID: {record.id}</span>
            <span className='text-xs text-gray-500'>类型: {record.serviceType}</span>
            <span className='text-xs text-gray-500'>级别: {record.escalationLevels.length}</span>
          </div>
        </div>
      ),
    },
    {
      title: '使用统计',
      dataIndex: 'usageCount',
      key: 'usageCount',
      align: 'center' as const,
      render: (_: unknown, record: EscalationRule) => (
        <div className='text-center'>
          <div className='text-lg font-bold text-orange-600'>{record.usageCount}</div>
          <div className='text-xs text-gray-500'>次触发</div>
          <div className='text-xs text-gray-500'>{record.lastTriggered}</div>
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
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      align: 'center' as const,
      render: (_: unknown, record: EscalationRule) => (
        <div className='text-center'>
          <div className='text-sm'>{record.updatedAt}</div>
          <div className='text-xs text-gray-500'>由 {record.createdBy}</div>
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      align: 'center' as const,
      render: (_: unknown, record: EscalationRule) => (
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
                setSelectedRule(record);
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
          <Tooltip title={record.status === 'active' ? '停用' : '启用'}>
            <Button
              type='text'
              icon={
                record.status === 'active' ? (
                  <Clock className='w-4 h-4' />
                ) : (
                  <CheckCircle className='w-4 h-4' />
                )
              }
              onClick={() => handleStatusToggle(record.id)}
            />
          </Tooltip>
          <Popconfirm
            title='确定要删除这个升级规则吗？'
            description='删除后无法恢复，正在使用的规则将停止工作。'
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
          <Zap className='inline-block w-6 h-6 mr-2' />
          升级规则管理
        </Title>
        <Text type='secondary'>配置事件和请求的自动升级策略，确保及时响应和处理</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className='mb-6'>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='升级规则总数'
              value={stats.total}
              prefix={<Zap className='w-5 h-5' />}
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
              prefix={<Edit className='w-5 h-5' />}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='总触发次数'
              value={stats.totalUsage}
              prefix={<Activity className='w-5 h-5' />}
              valueStyle={{ color: '#f5222d' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索和过滤 */}
      <Card className='mb-6'>
        <Row gutter={[16, 16]} align='middle'>
          <Col xs={24} md={6}>
            <Input
              placeholder='搜索规则名称或描述...'
              prefix={<Search className='w-4 h-4 text-gray-400' />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              placeholder='优先级'
              value={priorityFilter}
              onChange={setPriorityFilter}
              style={{ width: '100%' }}
            >
              <Option value='all'>全部优先级</Option>
              {Object.entries(PRIORITY_CONFIG).map(([key, config]) => (
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
              placeholder='服务类型'
              value={serviceTypeFilter}
              onChange={setServiceTypeFilter}
              style={{ width: '100%' }}
            >
              <Option value='all'>全部类型</Option>
              {serviceTypes.map(type => (
                <Option key={type} value={type}>
                  {type}
                </Option>
              ))}
            </Select>
          </Col>
          <Col xs={24} md={6} className='text-right'>
            <Button
              type='primary'
              icon={<Plus className='w-4 h-4' />}
              onClick={() => {
                setSelectedRule(null);
                form.resetFields();
                setShowCreateModal(true);
              }}
            >
              新建升级规则
            </Button>
          </Col>
        </Row>
      </Card>

      {/* 升级规则列表 */}
      <Card className='enterprise-card'>
        <Table
          columns={columns}
          dataSource={filteredRules}
          rowKey='id'
          pagination={{
            total: filteredRules.length,
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
            <Zap className='w-4 h-4 mr-2' />
            {selectedRule ? '编辑升级规则' : '新建升级规则'}
          </span>
        }
        open={showCreateModal}
        onOk={handleSave}
        onCancel={() => {
          setShowCreateModal(false);
          setSelectedRule(null);
          form.resetFields();
        }}
        width={700}
        confirmLoading={loading}
        okText='保存'
        cancelText='取消'
      >
        <Form form={form} layout='vertical' className='mt-4'>
          <Form.Item
            label='规则名称'
            name='name'
            rules={[{ required: true, message: '请输入规则名称' }]}
          >
            <Input placeholder='请输入规则名称' />
          </Form.Item>
          <Form.Item
            label='描述'
            name='description'
            rules={[{ required: true, message: '请输入规则描述' }]}
          >
            <Input.TextArea rows={3} placeholder='请输入规则描述' />
          </Form.Item>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label='优先级'
                name='priority'
                rules={[{ required: true, message: '请选择优先级' }]}
                initialValue='P3'
              >
                <Select>
                  {Object.entries(PRIORITY_CONFIG).map(([key, config]) => (
                    <Option key={key} value={key}>
                      {config.label}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label='服务类型'
                name='serviceType'
                rules={[{ required: true, message: '请输入服务类型' }]}
              >
                <Select showSearch placeholder='选择或输入服务类型' mode='combobox'>
                  {serviceTypes.map(type => (
                    <Option key={type} value={type}>
                      {type}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label='状态' name='status' initialValue='draft'>
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
          <Form.Item
            label='触发条件'
            name='triggerCondition'
            rules={[{ required: true, message: '请输入触发条件' }]}
          >
            <Input.TextArea rows={2} placeholder="例如：工单状态为'进行中'且优先级为P1" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title={
          <div className='flex items-center gap-2'>
            <Eye className='w-5 h-5' />
            <span>升级规则详情</span>
          </div>
        }
        placement='right'
        size='large'
        open={showDetailDrawer}
        onClose={() => setShowDetailDrawer(false)}
      >
        {selectedRule && (
          <div className='space-y-6'>
            <Descriptions title='基本信息' bordered column={1} size='small'>
              <Descriptions.Item label='规则ID'>{selectedRule.id}</Descriptions.Item>
              <Descriptions.Item label='名称'>{selectedRule.name}</Descriptions.Item>
              <Descriptions.Item label='描述'>{selectedRule.description}</Descriptions.Item>
              <Descriptions.Item label='优先级'>
                <Tag color={PRIORITY_CONFIG[selectedRule.priority]?.color}>
                  {PRIORITY_CONFIG[selectedRule.priority]?.label}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label='服务类型'>{selectedRule.serviceType}</Descriptions.Item>
              <Descriptions.Item label='触发条件'>
                {selectedRule.triggerCondition}
              </Descriptions.Item>
              <Descriptions.Item label='状态'>
                <Tag
                  color={STATUS_CONFIG[selectedRule.status as keyof typeof STATUS_CONFIG]?.color}
                  icon={STATUS_CONFIG[selectedRule.status as keyof typeof STATUS_CONFIG]?.icon}
                >
                  {STATUS_CONFIG[selectedRule.status as keyof typeof STATUS_CONFIG]?.label}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label='创建者'>{selectedRule.createdBy}</Descriptions.Item>
              <Descriptions.Item label='创建时间'>{selectedRule.createdAt}</Descriptions.Item>
              <Descriptions.Item label='最后修改'>{selectedRule.updatedAt}</Descriptions.Item>
            </Descriptions>

            <div>
              <Title level={4}>使用统计</Title>
              <Row gutter={16}>
                <Col span={8}>
                  <Statistic
                    title='触发次数'
                    value={selectedRule.usageCount}
                    prefix={<Activity className='w-4 h-4' />}
                  />
                </Col>
                <Col span={8}>
                  <Statistic
                    title='升级级别'
                    value={selectedRule.escalationLevels.length}
                    prefix={<Target className='w-4 h-4' />}
                  />
                </Col>
                <Col span={8}>
                  <Statistic
                    title='最后触发'
                    value={selectedRule.lastTriggered}
                    prefix={<Timer className='w-4 h-4' />}
                  />
                </Col>
              </Row>
            </div>

            <div>
              <Title level={4}>升级级别配置</Title>
              {renderEscalationLevels(selectedRule.escalationLevels)}
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default EscalationRuleManagement;
