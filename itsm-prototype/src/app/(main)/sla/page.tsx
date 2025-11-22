'use client';

import React, { useState, useCallback, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Button,
  Space,
  message,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  Table,
  Tag,
  Tooltip,
  Progress,
  Alert,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  ClockCircleOutlined,
  WarningOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import {
  slaService,
  SLADefinition,
  SLAInstance,
  SLAStats,
  SLAType,
  SLAPriority,
  SLAStatus,
  EscalationLevel,
} from '../../lib/services/sla-service';

const { Option } = Select;
const { TextArea } = Input;

export default function SLAManagementPage() {
  // 状态管理
  const [slaDefinitions, setSlaDefinitions] = useState<SLADefinition[]>([]);
  const [slaInstances, setSlaInstances] = useState<SLAInstance[]>([]);
  const [stats, setStats] = useState<SLAStats>({
    totalInstances: 0,
    activeInstances: 0,
    warningInstances: 0,
    breachedInstances: 0,
    resolvedInstances: 0,
    complianceRate: 0,
    averageResolutionTime: 0,
    breachRate: 0,
    escalationRate: 0,
  });
  const [loading, setLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(false);

  // 模态框状态
  const [modalVisible, setModalVisible] = useState(false);
  const [editingSLA, setEditingSLA] = useState<SLADefinition | null>(null);
  const [form] = Form.useForm();

  // 分页状态
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });

  // 加载SLA定义
  const loadSLADefinitions = useCallback(async () => {
    try {
      setLoading(true);
      const response = await slaService.getSLADefinitions({
        page: pagination.current,
        pageSize: pagination.pageSize,
      });
      setSlaDefinitions(response.items);
      setPagination(prev => ({ ...prev, total: response.total }));
    } catch (error) {
      message.error('加载SLA定义失败');
      console.error('Load SLA definitions error:', error);
    } finally {
      setLoading(false);
    }
  }, [pagination.current, pagination.pageSize]);

  // 加载SLA统计
  const loadSLAStats = useCallback(async () => {
    try {
      setStatsLoading(true);
      const statsData = await slaService.getSLAStats();
      setStats(statsData);
    } catch (error) {
      message.error('加载SLA统计失败');
      console.error('Load SLA stats error:', error);
    } finally {
      setStatsLoading(false);
    }
  }, []);

  // 加载SLA实例
  const loadSLAInstances = useCallback(async () => {
    try {
      const response = await slaService.getSLAInstances({
        page: 1,
        pageSize: 10,
      });
      setSlaInstances(response.items);
    } catch (error) {
      message.error('加载SLA实例失败');
      console.error('Load SLA instances error:', error);
    }
  }, []);

  // 初始化加载
  useEffect(() => {
    loadSLADefinitions();
    loadSLAStats();
    loadSLAInstances();
  }, [loadSLADefinitions, loadSLAStats, loadSLAInstances]);

  // 创建SLA
  const handleCreateSLA = useCallback(() => {
    setEditingSLA(null);
    form.resetFields();
    setModalVisible(true);
  }, [form]);

  // 编辑SLA
  const handleEditSLA = useCallback(
    (sla: SLADefinition) => {
      setEditingSLA(sla);
      form.setFieldsValue({
        ...sla,
        escalationRules: sla.escalationRules.map(rule => ({
          ...rule,
          actions: rule.actions.map(action => ({
            ...action,
            target: action.target,
          })),
        })),
      });
      setModalVisible(true);
    },
    [form]
  );

  // 删除SLA
  const handleDeleteSLA = useCallback(
    async (id: number) => {
      try {
        await slaService.deleteSLADefinition(id);
        message.success('SLA定义删除成功');
        loadSLADefinitions();
      } catch (error) {
        message.error('删除SLA定义失败');
        console.error('Delete SLA error:', error);
      }
    },
    [loadSLADefinitions]
  );

  // 提交表单
  const handleSubmit = useCallback(
    async (values: any) => {
      try {
        if (editingSLA) {
          await slaService.updateSLADefinition(editingSLA.id, values);
          message.success('SLA定义更新成功');
        } else {
          await slaService.createSLADefinition(values);
          message.success('SLA定义创建成功');
        }
        setModalVisible(false);
        loadSLADefinitions();
      } catch (error) {
        message.error(editingSLA ? '更新SLA定义失败' : '创建SLA定义失败');
        console.error('Submit SLA error:', error);
      }
    },
    [editingSLA, loadSLADefinitions]
  );

  // 获取SLA状态颜色
  const getSLAStatusColor = (status: SLAStatus) => {
    switch (status) {
      case SLAStatus.ACTIVE:
        return 'green';
      case SLAStatus.INACTIVE:
        return 'default';
      case SLAStatus.SUSPENDED:
        return 'orange';
      case SLAStatus.EXPIRED:
        return 'red';
      default:
        return 'default';
    }
  };

  // 获取SLA类型标签
  const getSLATypeLabel = (type: SLAType) => {
    switch (type) {
      case SLAType.RESPONSE_TIME:
        return '响应时间';
      case SLAType.RESOLUTION_TIME:
        return '解决时间';
      case SLAType.AVAILABILITY:
        return '可用性';
      case SLAType.PERFORMANCE:
        return '性能';
      default:
        return type;
    }
  };

  // 获取优先级颜色
  const getPriorityColor = (priority: SLAPriority) => {
    switch (priority) {
      case SLAPriority.CRITICAL:
        return 'red';
      case SLAPriority.HIGH:
        return 'orange';
      case SLAPriority.MEDIUM:
        return 'blue';
      case SLAPriority.LOW:
        return 'green';
      default:
        return 'default';
    }
  };

  // SLA定义表格列
  const slaColumns = [
    {
      title: 'SLA名称',
      dataIndex: 'name',
      key: 'name',
      sorter: true,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: SLAType) => <Tag color='blue'>{getSLATypeLabel(type)}</Tag>,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      render: (priority: SLAPriority) => (
        <Tag color={getPriorityColor(priority)}>{priority.toUpperCase()}</Tag>
      ),
    },
    {
      title: '目标时间',
      dataIndex: 'targetTime',
      key: 'targetTime',
      render: (time: number) => `${time}分钟`,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: SLAStatus) => (
        <Tag color={getSLAStatusColor(status)}>{status.toUpperCase()}</Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: SLADefinition) => (
        <Space size='small'>
          <Tooltip title='查看详情'>
            <Button icon={<EyeOutlined />} size='small' />
          </Tooltip>
          <Tooltip title='编辑'>
            <Button icon={<EditOutlined />} size='small' onClick={() => handleEditSLA(record)} />
          </Tooltip>
          <Tooltip title='删除'>
            <Button
              icon={<DeleteOutlined />}
              size='small'
              danger
              onClick={() => handleDeleteSLA(record.id)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  // SLA实例表格列
  const instanceColumns = [
    {
      title: '工单号',
      dataIndex: 'ticketNumber',
      key: 'ticketNumber',
    },
    {
      title: 'SLA名称',
      dataIndex: ['slaDefinition', 'name'],
      key: 'slaName',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        let color = 'default';
        let icon = null;
        switch (status) {
          case 'active':
            color = 'green';
            icon = <ClockCircleOutlined />;
            break;
          case 'warning':
            color = 'orange';
            icon = <WarningOutlined />;
            break;
          case 'breached':
            color = 'red';
            icon = <ExclamationCircleOutlined />;
            break;
          case 'resolved':
            color = 'blue';
            icon = <CheckCircleOutlined />;
            break;
        }
        return (
          <Tag color={color} icon={icon}>
            {status.toUpperCase()}
          </Tag>
        );
      },
    },
    {
      title: '剩余时间',
      dataIndex: 'remainingTime',
      key: 'remainingTime',
      render: (time: number) => {
        const hours = Math.floor(time / 60);
        const minutes = time % 60;
        return `${hours}小时${minutes}分钟`;
      },
    },
    {
      title: '升级级别',
      dataIndex: 'currentLevel',
      key: 'currentLevel',
      render: (level: EscalationLevel) => <Tag color='purple'>{level.toUpperCase()}</Tag>,
    },
  ];

  return (
    <div className='space-y-6'>
      {/* SLA统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title='总SLA实例'
              value={stats.totalInstances}
              prefix={<ClockCircleOutlined />}
              loading={statsLoading}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title='活跃实例'
              value={stats.activeInstances}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#3f8600' }}
              loading={statsLoading}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title='预警实例'
              value={stats.warningInstances}
              prefix={<WarningOutlined />}
              valueStyle={{ color: '#faad14' }}
              loading={statsLoading}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title='违约实例'
              value={stats.breachedInstances}
              prefix={<ExclamationCircleOutlined />}
              valueStyle={{ color: '#cf1322' }}
              loading={statsLoading}
            />
          </Card>
        </Col>
      </Row>

      {/* SLA合规率 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} md={12}>
          <Card title='SLA合规率'>
            <div className='text-center'>
              <Progress
                type='circle'
                percent={Math.round(stats.complianceRate * 100)}
                format={percent => `${percent}%`}
                strokeColor={{
                  '0%': '#108ee9',
                  '100%': '#87d068',
                }}
              />
              <div className='mt-4'>
                <p className='text-gray-600'>平均解决时间: {stats.averageResolutionTime}分钟</p>
                <p className='text-gray-600'>违约率: {(stats.breachRate * 100).toFixed(1)}%</p>
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} md={12}>
          <Card title='SLA预警'>
            {stats.warningInstances > 0 ? (
              <Alert
                message='SLA预警'
                description={`当前有 ${stats.warningInstances} 个SLA实例处于预警状态，请及时处理。`}
                type='warning'
                showIcon
              />
            ) : (
              <Alert
                message='SLA状态正常'
                description='所有SLA实例都在正常范围内。'
                type='success'
                showIcon
              />
            )}
          </Card>
        </Col>
      </Row>

      {/* SLA定义列表 */}
      <Card title='SLA定义' extra={<Button onClick={loadSLADefinitions}>刷新</Button>}>
        <Table
          columns={slaColumns}
          dataSource={slaDefinitions}
          rowKey='id'
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            onChange: (page, pageSize) => {
              setPagination(prev => ({ ...prev, current: page, pageSize: pageSize || 20 }));
            },
          }}
        />
      </Card>

      {/* SLA实例列表 */}
      <Card title='当前SLA实例'>
        <Table
          columns={instanceColumns}
          dataSource={slaInstances}
          rowKey='id'
          pagination={false}
          size='small'
        />
      </Card>

      {/* SLA创建/编辑模态框 */}
      <Modal
        title={editingSLA ? '编辑SLA定义' : '创建SLA定义'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={800}
        destroyOnClose
      >
        <Form form={form} layout='vertical' onFinish={handleSubmit}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='name'
                label='SLA名称'
                rules={[{ required: true, message: '请输入SLA名称' }]}
              >
                <Input placeholder='例如：高优先级工单SLA' />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='type'
                label='SLA类型'
                rules={[{ required: true, message: '请选择SLA类型' }]}
              >
                <Select placeholder='选择SLA类型'>
                  <Option value={SLAType.RESPONSE_TIME}>响应时间</Option>
                  <Option value={SLAType.RESOLUTION_TIME}>解决时间</Option>
                  <Option value={SLAType.AVAILABILITY}>可用性</Option>
                  <Option value={SLAType.PERFORMANCE}>性能</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='priority'
                label='优先级'
                rules={[{ required: true, message: '请选择优先级' }]}
              >
                <Select placeholder='选择优先级'>
                  <Option value={SLAPriority.CRITICAL}>关键</Option>
                  <Option value={SLAPriority.HIGH}>高</Option>
                  <Option value={SLAPriority.MEDIUM}>中</Option>
                  <Option value={SLAPriority.LOW}>低</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='targetTime'
                label='目标时间（分钟）'
                rules={[{ required: true, message: '请输入目标时间' }]}
              >
                <Input type='number' placeholder='例如：120' />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name='warningTime'
            label='预警时间（分钟）'
            rules={[{ required: true, message: '请输入预警时间' }]}
          >
            <Input type='number' placeholder='例如：90' />
          </Form.Item>

          <Form.Item name='description' label='描述'>
            <TextArea rows={3} placeholder='SLA描述信息' />
          </Form.Item>

          <Form.Item name='isDefault' label='设为默认' valuePropName='checked'>
            <Switch />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type='primary' htmlType='submit'>
                {editingSLA ? '更新' : '创建'}
              </Button>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
