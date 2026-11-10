'use client';

import {
  CheckCircle,
  Clock,
  Edit,
  Eye,
  Layers,
  Plus,
  RefreshCw,
  Search,
  Target,
  Timer,
  Trash2,
} from 'lucide-react';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { SLAApi } from '@/lib/api/sla-api';
import { formatDateTime } from '@/lib/formatters';
import type { ColumnsType } from 'antd/es/table';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Statistic,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { SLADefinition, SLAFormValues } from './types';
import { summarizeSLADefinitions, transformSLA } from './utils';
import {
  SLA_PRIORITY_VALUES,
  SLA_SERVICE_TYPE_LABELS,
  formatDuration,
  formatEscalationLevel,
  getServiceTypeLabel,
  getPriorityColor,
  getPriorityLabel,
  matchesPriority,
} from '@/lib/utils/sla-format';

const { Title, Text } = Typography;

// 后端 common.MaxPageSize 为 100，单次取满并在超量时显式提示，
// 避免默认分页把列表静默截断成 20 条却仍然按当前页做筛选。
const FETCH_PAGE_SIZE = 100;

// formatters.formatDateTime 对缺失或非法时间返回空串，页面统一显示为 -
const displayDateTime = (value?: string) => formatDateTime(value) || '-';

const STATUS_CONFIG: Record<SLADefinition['status'], { label: string; color: string }> = {
  active: { label: 'Enabled', color: 'success' },
  inactive: { label: 'Disabled', color: 'default' },
};

const SLADefinitionManagement = () => {
  const { message } = App.useApp();
  const [form] = Form.useForm<SLAFormValues>();

  const [definitions, setDefinitions] = useState<SLADefinition[]>([]);
  const [serverTotal, setServerTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [togglingId, setTogglingId] = useState<string | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [searchTerm, setSearchTerm] = useState('');
  const [priorityFilter, setPriorityFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [serviceTypeFilter, setServiceTypeFilter] = useState('all');

  const [editingSLA, setEditingSLA] = useState<SLADefinition | null>(null);
  const [showFormModal, setShowFormModal] = useState(false);
  const [detailSLA, setDetailSLA] = useState<SLADefinition | null>(null);

  const loadDefinitions = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const { items, total } = await SLAApi.getSLADefinitions({ page: 1, pageSize: FETCH_PAGE_SIZE });
      setDefinitions(items.map(transformSLA));
      setServerTotal(total);
    } catch (error) {
      setDefinitions([]);
      setServerTotal(0);
      setLoadError(error instanceof Error ? error.message : 'Failed to load SLA definition');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadDefinitions();
  }, [loadDefinitions]);

  const stats = useMemo(() => summarizeSLADefinitions(definitions), [definitions]);

  const serviceTypes = useMemo(
    () => Array.from(new Set(definitions.map(sla => sla.serviceType).filter(Boolean))),
    [definitions]
  );

  const filteredSLAs = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return definitions.filter(sla => {
      if (term && !`${sla.name} ${sla.description}`.toLowerCase().includes(term)) return false;
      if (!matchesPriority(sla.priority, priorityFilter)) return false;
      if (statusFilter !== 'all' && sla.status !== statusFilter) return false;
      if (serviceTypeFilter !== 'all' && sla.serviceType !== serviceTypeFilter) return false;
      return true;
    });
  }, [definitions, searchTerm, priorityFilter, statusFilter, serviceTypeFilter]);

  const openCreate = () => {
    setEditingSLA(null);
    form.resetFields();
    setShowFormModal(true);
  };

  const openEdit = (sla: SLADefinition) => {
    setEditingSLA(sla);
    form.setFieldsValue({
      name: sla.name,
      description: sla.description,
      serviceType: sla.serviceType,
      priority: sla.priority,
      responseTimeMinutes: sla.responseTimeMinutes,
      resolutionTimeMinutes: sla.resolutionTimeMinutes,
      isActive: sla.status === 'active',
    });
    setShowFormModal(true);
  };

  const closeForm = () => {
    setShowFormModal(false);
    setEditingSLA(null);
    form.resetFields();
  };

  const handleSave = async () => {
    let values: SLAFormValues;
    try {
      values = await form.validateFields();
    } catch {
      return;
    }

    // 表单字段与后端请求体同名同单位（分钟），不再做小时/分钟换算或可用性兜底。
    const payload = {
      name: values.name,
      description: values.description,
      serviceType: values.serviceType,
      priority: values.priority,
      responseTime: values.responseTimeMinutes,
      resolutionTime: values.resolutionTimeMinutes,
      isActive: values.isActive,
    };

    setSaving(true);
    try {
      if (editingSLA) {
        await SLAApi.updateSLADefinition(Number(editingSLA.id), payload);
        message.success(`SLA definition "${values.name}"Updated`);
      } else {
        await SLAApi.createSLADefinition(payload);
        message.success(`SLA definition "${values.name}"Already created`);
      }
      closeForm();
      await loadDefinitions();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to save SLA definition');
    } finally {
      setSaving(false);
    }
  };

  const handleStatusToggle = async (sla: SLADefinition) => {
    const nextActive = sla.status !== 'active';
    setTogglingId(sla.id);
    try {
      await SLAApi.updateSLADefinition(Number(sla.id), { isActive: nextActive });
      message.success(nextActive ? `「${sla.name}"Enabled"` : `「${sla.name}"Deactivated`);
      await loadDefinitions();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to update status');
    } finally {
      setTogglingId(null);
    }
  };

  const handleDelete = async (sla: SLADefinition) => {
    try {
      await SLAApi.deleteSLADefinition(Number(sla.id));
      message.success(`SLA definition "${sla.name}"Deleted`);
      if (detailSLA?.id === sla.id) setDetailSLA(null);
      await loadDefinitions();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Delete failed');
    }
  };

  const columns: ColumnsType<SLADefinition> = [
    {
      title: 'SLA Definition',
      dataIndex: 'name',
      key: 'name',
      render: (_, record) => (
        <div>
          <Space>
            <Text strong>{record.name}</Text>
            <Tag color={getPriorityColor(record.priority)}>{getPriorityLabel(record.priority)}</Tag>
          </Space>
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {record.description || 'No description'}
            </Text>
          </div>
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>
              ID: {record.id}  · Type: {getServiceTypeLabel(record.serviceType)}
            </Text>
          </div>
        </div>
      ),
    },
    {
      title: 'Service time limit',
      key: 'targets',
      render: (_, record) => (
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Timer className="w-3 h-3 text-blue-500" />
            <span className="text-xs">Response: {formatDuration(record.responseTimeMinutes)}</span>
          </div>
          <div className="flex items-center gap-2">
            <Target className="w-3 h-3 text-green-500" />
            <span className="text-xs">Solution: {formatDuration(record.resolutionTimeMinutes)}</span>
          </div>
          <div className="flex items-center gap-2">
            <Clock className="w-3 h-3 text-orange-500" />
            <span className="text-xs">Service window: {record.businessHours}</span>
          </div>
        </div>
      ),
    },
    {
      title: 'EscalationRule',
      key: 'escalation',
      render: (_, record) =>
        record.escalationLevels.length === 0 ? (
          <Text type="secondary">Not configured</Text>
        ) : (
          <Space wrap size={4}>
            {record.escalationLevels.map(level => (
              <Tag key={level.level}>{formatEscalationLevel(level)}</Tag>
            ))}
          </Space>
        ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      align: 'center',
      render: (status: SLADefinition['status']) => (
        <Tag color={STATUS_CONFIG[status].color}>{STATUS_CONFIG[status].label}</Tag>
      ),
    },
    {
      title: 'Updated At',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      align: 'center',
      render: (updatedAt: string) => <Text style={{ fontSize: 12 }}>{displayDateTime(updatedAt)}</Text>,
    },
    {
      title: 'Actions',
      key: 'actions',
      align: 'center',
      render: (_, record) => (
        <Space>
          <Tooltip title="View Details">
            <Button
              aria-label="View Details"
              type="text"
              icon={<Eye className="w-4 h-4" />}
              onClick={() => setDetailSLA(record)}
            />
          </Tooltip>
          <Tooltip title="Edit">
            <Button
              aria-label="Edit"
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => openEdit(record)}
            />
          </Tooltip>
          <Tooltip title={record.status === 'active' ? 'Disable' : 'Enabled'}>
            <Button
              aria-label={record.status === 'active' ? 'Disable' : 'Enabled'}
              type="text"
              loading={togglingId === record.id}
              icon={
                record.status === 'active' ? (
                  <Clock className="w-4 h-4" />
                ) : (
                  <CheckCircle className="w-4 h-4" />
                )
              }
              onClick={() => handleStatusToggle(record)}
            />
          </Tooltip>
          <Popconfirm
            title="Are you sure you want to delete this SLA definition?"
            description={`Delete "${record.name} cannot be recovered, and the work orders that reference it will lose SLA guarantee.`}
            onConfirm={() => handleDelete(record)}
            okText="Confirm deletion"
            cancelText="Cancel"
            okType="danger"
          >
            <Button aria-label="Delete" type="text" danger icon={<Trash2 className="w-4 h-4" />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title level={2} className="!mb-2">
          <Target className="inline-block w-6 h-6 mr-2" />

          SLA Definition Management
        </Title>
        <Text type="secondary">Define and manage service level agreements to ensure service quality standards</Text>
      </div>

      {/* 统计卡必须门控：还没拉到数据时渲染成「SLA 定义总数 0」会被误读成「没有定义任何 SLA」 */}
      {!loading && (
      <Row gutter={[16, 16]} align="stretch">
        <Col xs={24} sm={12} lg={6} className="flex">
          <Card className="enterprise-card h-full w-full">
            <Statistic
              title="Total number of SLA definitions"
              value={stats.total}
              prefix={<Target className="w-5 h-5" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6} className="flex">
          <Card className="enterprise-card h-full w-full">
            <Statistic
              title="Enabled"
              value={stats.active}
              prefix={<CheckCircle className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6} className="flex">
          <Card className="enterprise-card h-full w-full">
            <Statistic
              title="Disabled"
              value={stats.inactive}
              prefix={<Clock className="w-5 h-5" />}
              styles={{ content: { color: '#8c8c8c' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6} className="flex">
          <Card className="enterprise-card h-full w-full">
            <Statistic
              title="Override service type"
              value={stats.serviceTypeCount}
              suffix="Class"
              prefix={<Layers className="w-5 h-5" />}
              styles={{ content: { color: '#722ed1' } }}
            />
          </Card>
        </Col>
      </Row>
      )}

      <Card>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={8}>
            <Input
              placeholder="Searching for SLA name or description..."
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              aria-label="Priority Filter"
              value={priorityFilter}
              onChange={setPriorityFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: 'All priorities' },
                ...SLA_PRIORITY_VALUES.map(value => ({ value, label: getPriorityLabel(value) })),
              ]}
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              aria-label="Status"
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: 'All Status' },
                ...Object.entries(STATUS_CONFIG).map(([value, config]) => ({
                  value,
                  label: config.label,
                })),
              ]}
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              aria-label="Service type filter"
              value={serviceTypeFilter}
              onChange={setServiceTypeFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: 'All Types' },
                ...serviceTypes.map(type => ({ value: type, label: getServiceTypeLabel(type) })),
              ]}
            />
          </Col>
          <Col xs={24} md={4} className="text-right">
            <Button type="primary" icon={<Plus className="w-4 h-4" />} onClick={openCreate}>

              Create new SLA
            </Button>
          </Col>
        </Row>
      </Card>

      {loadError && (
        <Alert
          type="error"
          showIcon
          title="SLA definition loading failed"
          description={loadError}
          action={
            <Button size="small" icon={<RefreshCw className="w-4 h-4" />} onClick={loadDefinitions}>

              Reload
            </Button>
          }
        />
      )}

      {!loadError && serverTotal > definitions.length && (
        <Alert
          type="warning"
          showIcon
          title={`Shared by the server ${serverTotal} SLA definitions, currently only loaded ${definitions.length} items`}
          description="Filtering and statistics only cover the loaded part, please simplify the number of definitions or connect to paging query."
        />
      )}

      <Card className="enterprise-card">
        {!loadError && !loading && definitions.length === 0 ? (
          <Empty description="There is no SLA definition yet. Click &quot;New SLA&quot; in the upper right corner to create it." />
        ) : (
          <Table<SLADefinition>
            columns={columns}
            dataSource={filteredSLAs}
            rowKey="id"
            loading={loading}
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: total =>
                `Filter out ${total} (total number of servers) ${serverTotal} Article)`,
            }}
            className="enterprise-table"
          />
        )}
      </Card>

      <Modal
        title={
          <span>
            <Target className="w-4 h-4 mr-2" />
            {editingSLA ? 'Edit SLA Definition' : 'New SLA definition'}
          </span>
        }
        open={showFormModal}
        onOk={handleSave}
        onCancel={closeForm}
        width={720}
        confirmLoading={saving}
        okText="Save"
        cancelText="Cancel"
        destroyOnHidden
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="SLA Name"
                name="name"
                rules={[{ required: true, message: 'Please enter SLA name' }]}
              >
                <Input placeholder="Example: Incident-P1-High" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="Service Type"
                name="serviceType"
                rules={[{ required: true, message: 'Please select service type' }]}
              >
                <Select
                  showSearch
                  optionFilterProp="label"
                  placeholder="Select service type"
                  options={Object.entries(SLA_SERVICE_TYPE_LABELS).map(([value, label]) => ({
                    value,
                    label,
                  }))}
                />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item label="Description" name="description">
            <Input.TextArea rows={3} placeholder="Explain what scenarios this SLA applies to" />
          </Form.Item>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="Priority"
                name="priority"
                initialValue="medium"
                rules={[{ required: true, message: 'Please select priority' }]}
              >
                <Select
                  options={SLA_PRIORITY_VALUES.map(value => ({
                    value,
                    label: getPriorityLabel(value),
                  }))}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="Response time limit (minutes)"
                name="responseTimeMinutes"
                initialValue={60}
                rules={[{ required: true, message: 'Please enter response time limit' }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} placeholder="60" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="Resolution time limit (minutes)"
                name="resolutionTimeMinutes"
                initialValue={240}
                rules={[{ required: true, message: 'Please enter the resolution time limit' }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} placeholder="240" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            label="Enable after creation"
            name="isActive"
            valuePropName="checked"
            initialValue={true}
            extra="If the backend creation interface does not pass isActive, it will be disabled by default, so it is submitted explicitly here."
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={
          <div className="flex items-center gap-2">
            <Eye className="w-5 h-5" />
            <span>SLA definition details</span>
          </div>
        }
        open={Boolean(detailSLA)}
        onCancel={() => setDetailSLA(null)}
        width={760}
        footer={[
          <Button key="close" onClick={() => setDetailSLA(null)}>

            Close
          </Button>,
        ]}
      >
        {detailSLA && (
          <div className="space-y-6">
            <div>
              <Title level={4}>{detailSLA.name}</Title>
              <Text type="secondary">{detailSLA.description || 'No description'}</Text>
            </div>

            <Descriptions
              bordered
              size="small"
              column={2}
              items={[
                {
                  key: 'priority',
                  label: 'Priority',
                  children: (
                    <Tag color={getPriorityColor(detailSLA.priority)}>
                      {getPriorityLabel(detailSLA.priority)}
                    </Tag>
                  ),
                },
                {
                  key: 'serviceType',
                  label: 'Service Type',
                  children: getServiceTypeLabel(detailSLA.serviceType),
                },
                {
                  key: 'response',
                  label: 'Response time limit',
                  children: formatDuration(detailSLA.responseTimeMinutes),
                },
                {
                  key: 'resolution',
                  label: 'Resolution time limit',
                  children: formatDuration(detailSLA.resolutionTimeMinutes),
                },
                { key: 'window', label: 'Service window', children: detailSLA.businessHours },
                {
                  key: 'status',
                  label: 'Status',
                  children: (
                    <Tag color={STATUS_CONFIG[detailSLA.status].color}>
                      {STATUS_CONFIG[detailSLA.status].label}
                    </Tag>
                  ),
                },
                { key: 'created', label: 'Created', children: displayDateTime(detailSLA.createdAt) },
                { key: 'updated', label: 'Updated At', children: displayDateTime(detailSLA.updatedAt) },
              ]}
            />

            <div>
              <Title level={5}>EscalationRule</Title>
              {detailSLA.escalationLevels.length === 0 ? (
                <Text type="secondary">The definition does not have an upgrade level configured.</Text>
              ) : (
                <Space wrap size={8}>
                  {detailSLA.escalationLevels.map(level => (
                    <Tag key={level.level} color="blue">
                      {formatEscalationLevel(level)}
                    </Tag>
                  ))}
                </Space>
              )}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default SLADefinitionManagement;
