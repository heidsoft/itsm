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
} from '@/lib/services/sla-service';
import { useI18n } from '@/lib/i18n';

const { Option } = Select;
const { TextArea } = Input;

export default function SLAManagementPage() {
  const { t } = useI18n();
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
      message.error(t('sla.loadDefinitionsFailed'));
      console.error('Load SLA definitions error:', error);
    } finally {
      setLoading(false);
    }
  }, [pagination.current, pagination.pageSize, t]);

  // 加载SLA统计
  const loadSLAStats = useCallback(async () => {
    try {
      setStatsLoading(true);
      const statsData = await slaService.getSLAStats();
      setStats(statsData);
    } catch (error) {
      message.error(t('sla.loadStatsFailed'));
      console.error('Load SLA stats error:', error);
    } finally {
      setStatsLoading(false);
    }
  }, [t]);

  // 加载SLA实例
  const loadSLAInstances = useCallback(async () => {
    try {
      const response = await slaService.getSLAInstances({
        page: 1,
        pageSize: 10,
      });
      setSlaInstances(response.items);
    } catch (error) {
      message.error(t('sla.loadInstancesFailed'));
      console.error('Load SLA instances error:', error);
    }
  }, [t]);

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
        message.success(t('sla.deleteSuccess'));
        loadSLADefinitions();
      } catch (error) {
        message.error(t('sla.deleteFailed'));
        console.error('Delete SLA error:', error);
      }
    },
    [loadSLADefinitions, t]
  );

  // 提交表单
  const handleSubmit = useCallback(
    async (values: any) => {
      try {
        if (editingSLA) {
          await slaService.updateSLADefinition(editingSLA.id, values);
          message.success(t('sla.updateSuccess'));
        } else {
          await slaService.createSLADefinition(values);
          message.success(t('sla.createSuccess'));
        }
        setModalVisible(false);
        loadSLADefinitions();
      } catch (error) {
        message.error(editingSLA ? t('sla.updateFailed') : t('sla.createFailed'));
        console.error('Submit SLA error:', error);
      }
    },
    [editingSLA, loadSLADefinitions, t]
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
        return t('sla.responseTime');
      case SLAType.RESOLUTION_TIME:
        return t('sla.resolutionTime');
      case SLAType.AVAILABILITY:
        return t('sla.availability');
      case SLAType.PERFORMANCE:
        return t('sla.performance');
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
      title: t('sla.name'),
      dataIndex: 'name',
      key: 'name',
      sorter: true,
    },
    {
      title: t('sla.type'),
      dataIndex: 'type',
      key: 'type',
      render: (type: SLAType) => <Tag color='blue'>{getSLATypeLabel(type)}</Tag>,
    },
    {
      title: t('sla.priority'),
      dataIndex: 'priority',
      key: 'priority',
      render: (priority: SLAPriority) => (
        <Tag color={getPriorityColor(priority)}>{priority.toUpperCase()}</Tag>
      ),
    },
    {
      title: t('sla.targetTime'),
      dataIndex: 'targetTime',
      key: 'targetTime',
      render: (time: number) => `${time}${t('sla.minutes')}`,
    },
    {
      title: t('sla.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status: SLAStatus) => (
        <Tag color={getSLAStatusColor(status)}>{status.toUpperCase()}</Tag>
      ),
    },
    {
      title: t('sla.actions'),
      key: 'actions',
      render: (_: any, record: SLADefinition) => (
        <Space size='small'>
          <Tooltip title={t('sla.viewDetails')}>
            <Button icon={<EyeOutlined />} size='small' />
          </Tooltip>
          <Tooltip title={t('sla.edit')}>
            <Button icon={<EditOutlined />} size='small' onClick={() => handleEditSLA(record)} />
          </Tooltip>
          <Tooltip title={t('sla.delete')}>
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
      title: t('sla.ticketNumber'),
      dataIndex: 'ticketNumber',
      key: 'ticketNumber',
    },
    {
      title: t('sla.name'),
      dataIndex: ['slaDefinition', 'name'],
      key: 'slaName',
    },
    {
      title: t('sla.status'),
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
      title: t('sla.remainingTime'),
      dataIndex: 'remainingTime',
      key: 'remainingTime',
      render: (time: number) => {
        const hours = Math.floor(time / 60);
        const minutes = time % 60;
        return `${hours}${t('sla.hours')}${minutes}${t('sla.minutes')}`;
      },
    },
    {
      title: t('sla.escalationLevel'),
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
              title={t('sla.totalInstances')}
              value={stats.totalInstances}
              prefix={<ClockCircleOutlined />}
              loading={statsLoading}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title={t('sla.activeInstances')}
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
              title={t('sla.warningInstances')}
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
              title={t('sla.breachedInstances')}
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
          <Card title={t('sla.complianceRate')}>
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
                <p className='text-gray-600'>{t('sla.avgResolutionTime')}: {stats.averageResolutionTime}{t('sla.minutes')}</p>
                <p className='text-gray-600'>{t('sla.breachRate')}: {(stats.breachRate * 100).toFixed(1)}%</p>
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} md={12}>
          <Card title={t('sla.slaWarning')}>
            {stats.warningInstances > 0 ? (
              <Alert
                message={t('sla.slaWarning')}
                description={t('sla.slaWarningDescription', { count: stats.warningInstances })}
                type='warning'
                showIcon
              />
            ) : (
              <Alert
                message={t('sla.slaStatusNormal')}
                description={t('sla.slaStatusNormalDescription')}
                type='success'
                showIcon
              />
            )}
          </Card>
        </Col>
      </Row>

      {/* SLA定义列表 */}
      <Card title={t('sla.slaDefinitions')} extra={<Button onClick={loadSLADefinitions}>{t('sla.refresh')}</Button>}>
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
      <Card title={t('sla.currentInstances')}>
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
        title={editingSLA ? t('sla.editSla') : t('sla.createSla')}
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
                label={t('sla.name')}
                rules={[{ required: true, message: t('sla.nameRequired') }]}
              >
                <Input placeholder={t('sla.namePlaceholder')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='type'
                label={t('sla.type')}
                rules={[{ required: true, message: t('sla.typeRequired') }]}
              >
                <Select placeholder={t('sla.typePlaceholder')}>
                  <Option value={SLAType.RESPONSE_TIME}>{t('sla.responseTime')}</Option>
                  <Option value={SLAType.RESOLUTION_TIME}>{t('sla.resolutionTime')}</Option>
                  <Option value={SLAType.AVAILABILITY}>{t('sla.availability')}</Option>
                  <Option value={SLAType.PERFORMANCE}>{t('sla.performance')}</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='priority'
                label={t('sla.priority')}
                rules={[{ required: true, message: t('sla.priorityRequired') }]}
              >
                <Select placeholder={t('sla.priorityPlaceholder')}>
                  <Option value={SLAPriority.CRITICAL}>{t('sla.critical')}</Option>
                  <Option value={SLAPriority.HIGH}>{t('sla.high')}</Option>
                  <Option value={SLAPriority.MEDIUM}>{t('sla.medium')}</Option>
                  <Option value={SLAPriority.LOW}>{t('sla.low')}</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='targetTime'
                label={`${t('sla.targetTime')} (${t('sla.minutes')})`}
                rules={[{ required: true, message: t('sla.targetTimeRequired') }]}
              >
                <Input type='number' placeholder={t('sla.targetTimePlaceholder')} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name='warningTime'
            label={`${t('sla.warningTime')} (${t('sla.minutes')})`}
            rules={[{ required: true, message: t('sla.warningTimeRequired') }]}
          >
            <Input type='number' placeholder={t('sla.warningTimePlaceholder')} />
          </Form.Item>

          <Form.Item name='description' label={t('sla.description')}>
            <TextArea rows={3} placeholder={t('sla.descriptionPlaceholder')} />
          </Form.Item>

          <Form.Item name='isDefault' label={t('sla.isDefault')} valuePropName='checked'>
            <Switch />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type='primary' htmlType='submit'>
                {editingSLA ? t('sla.update') : t('sla.create')}
              </Button>
              <Button onClick={() => setModalVisible(false)}>{t('sla.cancel')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
