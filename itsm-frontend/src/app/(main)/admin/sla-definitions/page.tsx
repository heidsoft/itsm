'use client';

import {
  Timer,
  CheckCircle,
  Clock,
  Edit,
  Target,
  TrendingUp,
  Eye,
  Trash2,
  Plus,
  Search,
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
  message,
  Progress,
  List,
  Badge,
  Tag,
} from 'antd';
const { Title, Text } = Typography;
const { Option } = Select;

// SLA定义数据类型
interface SLADefinition {
  id: string;
  name: string;
  description: string;
  serviceType: string;
  priority: 'P1' | 'P2' | 'P3' | 'P4';
  responseTime: string;
  resolutionTime: string;
  availability: string;
  businessHours: string;
  escalationRules: string[];
  applicableServices: string[];
  status: 'active' | 'inactive' | 'draft';
  createdAt: string;
  updatedAt: string;
  createdBy: string;
}

// 模拟SLA定义数据
const mockSLADefinitions: SLADefinition[] = [
  {
    id: 'SLA-DEF-001',
    name: '关键业务系统SLA',
    description: '针对关键业务系统的服务级别协议，包括ERP、CRM等核心业务系统',
    serviceType: '关键业务系统',
    priority: 'P1',
    responseTime: '15分钟',
    resolutionTime: '4小时',
    availability: '99.9%',
    businessHours: '7x24',
    escalationRules: ['15分钟升级至高级工程师', '1小时升级至技术经理', '4小时升级至CTO'],
    applicableServices: ['ERP系统', 'CRM系统', '财务系统'],
    status: 'active',
    createdAt: '2024-01-15',
    updatedAt: '2024-06-01',
    createdBy: '张三',
  },
  {
    id: 'SLA-DEF-002',
    name: '一般业务系统SLA',
    description: '针对一般业务系统的服务级别协议，包括OA、HR等支撑系统',
    serviceType: '一般业务系统',
    priority: 'P2',
    responseTime: '30分钟',
    resolutionTime: '8小时',
    availability: '99.5%',
    businessHours: '工作时间',
    escalationRules: ['30分钟升级至高级工程师', '2小时升级至技术经理'],
    applicableServices: ['OA系统', 'HR系统', '采购系统'],
    status: 'active',
    createdAt: '2024-01-20',
    updatedAt: '2024-05-15',
    createdBy: '李四',
  },
  {
    id: 'SLA-DEF-003',
    name: '基础设施SLA',
    description: '针对基础设施服务的服务级别协议，包括网络、服务器等',
    serviceType: '基础设施',
    priority: 'P1',
    responseTime: '10分钟',
    resolutionTime: '2小时',
    availability: '99.95%',
    businessHours: '7x24',
    escalationRules: ['10分钟升级至网络工程师', '30分钟升级至基础设施经理'],
    applicableServices: ['网络服务', '服务器', '存储系统'],
    status: 'active',
    createdAt: '2024-02-01',
    updatedAt: '2024-06-10',
    createdBy: '王五',
  },
  {
    id: 'SLA-DEF-004',
    name: '开发测试环境SLA',
    description: '针对开发测试环境的服务级别协议',
    serviceType: '开发测试',
    priority: 'P3',
    responseTime: '2小时',
    resolutionTime: '1工作日',
    availability: '95%',
    businessHours: '工作时间',
    escalationRules: ['4小时升级至开发经理'],
    applicableServices: ['开发环境', '测试环境', 'CI/CD平台'],
    status: 'draft',
    createdAt: '2024-06-01',
    updatedAt: '2024-06-15',
    createdBy: '赵六',
  },
];

// 优先级配置
const PRIORITY_CONFIG = {
  P1: { label: 'P1 - 紧急', color: 'red' },
  P2: { label: 'P2 - 高', color: 'orange' },
  P3: { label: 'P3 - 中', color: 'yellow' },
  P4: { label: 'P4 - 低', color: 'green' },
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
    color: 'processing',
    icon: <Edit className='w-3 h-3' />,
  },
};

const SLADefinitionManagement = () => {
  const [slaDefinitions, setSlaDefinitions] = useState(mockSLADefinitions);
  const [searchTerm, setSearchTerm] = useState('');
  const [priorityFilter, setPriorityFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [serviceTypeFilter, setServiceTypeFilter] = useState('all');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedSLA, setSelectedSLA] = useState<SLADefinition | null>(null);
  const [showDetailModal, setShowDetailModal] = useState(false);
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  // 统计信息
  const stats = {
    total: slaDefinitions.length,
    active: slaDefinitions.filter(sla => sla.status === 'active').length,
    draft: slaDefinitions.filter(sla => sla.status === 'draft').length,
    avgAvailability: (
      slaDefinitions.reduce((sum, sla) => sum + parseFloat(sla.availability), 0) /
      slaDefinitions.length
    ).toFixed(1),
  };

  // 获取所有服务类型
  const serviceTypes = Array.from(new Set(slaDefinitions.map(sla => sla.serviceType)));

  // 过滤SLA定义
  const filteredSLAs = slaDefinitions.filter(sla => {
    const matchesSearch =
      sla.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      sla.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesPriority = priorityFilter === 'all' || sla.priority === priorityFilter;
    const matchesStatus = statusFilter === 'all' || sla.status === statusFilter;
    const matchesServiceType = serviceTypeFilter === 'all' || sla.serviceType === serviceTypeFilter;

    return matchesSearch && matchesPriority && matchesStatus && matchesServiceType;
  });

  // 处理状态切换
  const handleStatusToggle = (slaId: string) => {
    setSlaDefinitions(prev =>
      prev.map(sla => {
        if (sla.id === slaId) {
          const newStatus = sla.status === 'active' ? 'inactive' : 'active';
          return { ...sla, status: newStatus };
        }
        return sla;
      })
    );
    message.success('SLA状态已更新');
  };

  // 处理删除
  const handleDelete = (slaId: string) => {
    setSlaDefinitions(prev => prev.filter(sla => sla.id !== slaId));
    message.success('SLA定义已删除');
  };

  // 查看详情
  const handleViewDetail = (sla: SLADefinition) => {
    setSelectedSLA(sla);
    setShowDetailModal(true);
  };

  // 保存SLA定义
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      await new Promise(resolve => setTimeout(resolve, 1000));

      if (selectedSLA) {
        // 编辑
        setSlaDefinitions(prev =>
          prev.map(sla =>
            sla.id === selectedSLA.id
              ? {
                  ...sla,
                  ...values,
                  updatedAt: new Date().toLocaleDateString(),
                }
              : sla
          )
        );
        message.success('SLA定义更新成功');
      } else {
        // 新建
        const newSLA: SLADefinition = {
          id: `SLA-DEF-${String(slaDefinitions.length + 1).padStart(3, '0')}`,
          ...values,
          status: 'draft',
          createdBy: '当前用户',
          createdAt: new Date().toLocaleDateString(),
          updatedAt: new Date().toLocaleDateString(),
          escalationRules: [],
          applicableServices: [],
        };
        setSlaDefinitions(prev => [newSLA, ...prev]);
        message.success('SLA定义创建成功');
      }

      setShowCreateModal(false);
      setSelectedSLA(null);
      form.resetFields();
    } catch (error) {
      message.error('保存失败，请检查必填项');
    } finally {
      setLoading(false);
    }
  };

  // 表格列定义
  const columns = [
    {
      title: 'SLA定义',
      dataIndex: 'name',
      key: 'name',
      render: (_: unknown, record: SLADefinition) => (
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
          </div>
        </div>
      ),
    },
    {
      title: '服务指标',
      key: 'metrics',
      render: (_: unknown, record: SLADefinition) => (
        <div className='space-y-1'>
          <div className='flex items-center gap-2'>
            <Timer className='w-3 h-3 text-blue-500' />
            <span className='text-xs'>响应: {record.responseTime}</span>
          </div>
          <div className='flex items-center gap-2'>
            <Target className='w-3 h-3 text-green-500' />
            <span className='text-xs'>解决: {record.resolutionTime}</span>
          </div>
          <div className='flex items-center gap-2'>
            <TrendingUp className='w-3 h-3 text-purple-500' />
            <span className='text-xs'>可用性: {record.availability}</span>
          </div>
          <div className='flex items-center gap-2'>
            <Clock className='w-3 h-3 text-orange-500' />
            <span className='text-xs'>{record.businessHours}</span>
          </div>
        </div>
      ),
    },
    {
      title: '适用服务',
      dataIndex: 'applicableServices',
      key: 'applicableServices',
      render: (services: string[]) => (
        <div className='space-y-1'>
          {services.slice(0, 2).map(service => (
            <Tag key={service}>{service}</Tag>
          ))}
          {services.length > 2 && (
            <Text type='secondary' className='text-xs'>
              +{services.length - 2} 更多
            </Text>
          )}
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
      key: 'updateInfo',
      align: 'center' as const,
      render: (_: unknown, record: SLADefinition) => (
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
      render: (_: unknown, record: SLADefinition) => (
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
                setSelectedSLA(record);
                form.setFieldsValue(record);
                setShowCreateModal(true);
              }}
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
            title='确定要删除这个SLA定义吗？'
            description='删除后无法恢复，相关的服务将失去SLA保障。'
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
          <Target className='inline-block w-6 h-6 mr-2' />
          SLA定义管理
        </Title>
        <Text type='secondary'>定义和管理服务级别协议，确保服务质量标准</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className='mb-6'>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='SLA定义总数'
              value={stats.total}
              prefix={<Target className='w-5 h-5' />}
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
              title='平均可用性'
              value={stats.avgAvailability}
              suffix='%'
              prefix={<TrendingUp className='w-5 h-5' />}
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
              placeholder='搜索SLA定义名称或描述...'
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
          <Col xs={24} md={4} className='text-right'>
            <Button
              type='primary'
              icon={<Plus className='w-4 h-4' />}
              onClick={() => {
                setSelectedSLA(null);
                form.resetFields();
                setShowCreateModal(true);
              }}
            >
              新建SLA
            </Button>
          </Col>
        </Row>
      </Card>

      {/* SLA定义列表 */}
      <Card className='enterprise-card'>
        <Table
          columns={columns}
          dataSource={filteredSLAs}
          rowKey='id'
          pagination={{
            total: filteredSLAs.length,
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
            <Target className='w-4 h-4 mr-2' />
            {selectedSLA ? '编辑SLA定义' : '新建SLA定义'}
          </span>
        }
        open={showCreateModal}
        onOk={handleSave}
        onCancel={() => {
          setShowCreateModal(false);
          setSelectedSLA(null);
          form.resetFields();
        }}
        width={800}
        confirmLoading={loading}
        okText='保存'
        cancelText='取消'
      >
        <Form form={form} layout='vertical' className='mt-4'>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label='SLA名称'
                name='name'
                rules={[{ required: true, message: '请输入SLA名称' }]}
              >
                <Input placeholder='请输入SLA名称' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label='服务类型'
                name='serviceType'
                rules={[{ required: true, message: '请选择服务类型' }]}
              >
                <Select showSearch placeholder='选择或输入服务类型' mode='tags'>
                  {serviceTypes.map(type => (
                    <Option key={type} value={type}>
                      {type}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            label='描述'
            name='description'
            rules={[{ required: true, message: '请输入SLA描述' }]}
          >
            <Input.TextArea rows={3} placeholder='请输入SLA描述' />
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
                label='响应时间'
                name='responseTime'
                rules={[{ required: true, message: '请输入响应时间' }]}
              >
                <Input placeholder='如: 30分钟' />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label='解决时间'
                name='resolutionTime'
                rules={[{ required: true, message: '请输入解决时间' }]}
              >
                <Input placeholder='如: 4小时' />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label='可用性'
                name='availability'
                rules={[{ required: true, message: '请输入可用性' }]}
              >
                <Input placeholder='如: 99.9%' />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label='业务时间'
                name='businessHours'
                rules={[{ required: true, message: '请输入业务时间' }]}
              >
                <Select>
                  <Option value='7x24'>7x24小时</Option>
                  <Option value='工作时间'>工作时间</Option>
                  <Option value='5x8'>5x8小时</Option>
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
        </Form>
      </Modal>

      {/* 详情模态框 */}
      <Modal
        title={
          <div className='flex items-center gap-2'>
            <Eye className='w-5 h-5' />
            <span>SLA定义详情</span>
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
        {selectedSLA && (
          <div className='space-y-6'>
            <div>
              <Title level={4}>{selectedSLA.name}</Title>
              <Text type='secondary'>{selectedSLA.description}</Text>
            </div>

            <Row gutter={16}>
              <Col span={8}>
                <Card size='small'>
                  <Statistic
                    title='响应时间'
                    value={selectedSLA.responseTime}
                    prefix={<Timer className='w-4 h-4' />}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card size='small'>
                  <Statistic
                    title='解决时间'
                    value={selectedSLA.resolutionTime}
                    prefix={<Target className='w-4 h-4' />}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card size='small'>
                  <Statistic
                    title='可用性'
                    value={selectedSLA.availability}
                    prefix={<TrendingUp className='w-4 h-4' />}
                  />
                </Card>
              </Col>
            </Row>

            <div>
              <Title level={5}>升级规则</Title>
              <List
                size='small'
                dataSource={selectedSLA.escalationRules}
                renderItem={(rule, index) => (
                  <List.Item>
                    <Badge count={index + 1} color='blue' />
                    <span className='ml-2'>{rule}</span>
                  </List.Item>
                )}
              />
            </div>

            <div>
              <Title level={5}>适用服务</Title>
              <div className='flex flex-wrap gap-2'>
                {selectedSLA.applicableServices.map(service => (
                  <Tag key={service} color='blue'>
                    {service}
                  </Tag>
                ))}
              </div>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default SLADefinitionManagement;
