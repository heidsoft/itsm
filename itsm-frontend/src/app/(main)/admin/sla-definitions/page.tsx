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
  active: { label: '已启用', color: 'success' },
  inactive: { label: '已停用', color: 'default' },
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
      setLoadError(error instanceof Error ? error.message : '加载 SLA 定义失败');
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
        message.success(`SLA 定义「${values.name}」已更新`);
      } else {
        await SLAApi.createSLADefinition(payload);
        message.success(`SLA 定义「${values.name}」已创建`);
      }
      closeForm();
      await loadDefinitions();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存 SLA 定义失败');
    } finally {
      setSaving(false);
    }
  };

  const handleStatusToggle = async (sla: SLADefinition) => {
    const nextActive = sla.status !== 'active';
    setTogglingId(sla.id);
    try {
      await SLAApi.updateSLADefinition(Number(sla.id), { isActive: nextActive });
      message.success(nextActive ? `「${sla.name}」已启用` : `「${sla.name}」已停用`);
      await loadDefinitions();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '更新状态失败');
    } finally {
      setTogglingId(null);
    }
  };

  const handleDelete = async (sla: SLADefinition) => {
    try {
      await SLAApi.deleteSLADefinition(Number(sla.id));
      message.success(`SLA 定义「${sla.name}」已删除`);
      if (detailSLA?.id === sla.id) setDetailSLA(null);
      await loadDefinitions();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败');
    }
  };

  const columns: ColumnsType<SLADefinition> = [
    {
      title: 'SLA 定义',
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
              {record.description || '无描述'}
            </Text>
          </div>
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>
              ID: {record.id} · 类型: {getServiceTypeLabel(record.serviceType)}
            </Text>
          </div>
        </div>
      ),
    },
    {
      title: '服务时限',
      key: 'targets',
      render: (_, record) => (
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Timer className="w-3 h-3 text-blue-500" />
            <span className="text-xs">响应: {formatDuration(record.responseTimeMinutes)}</span>
          </div>
          <div className="flex items-center gap-2">
            <Target className="w-3 h-3 text-green-500" />
            <span className="text-xs">解决: {formatDuration(record.resolutionTimeMinutes)}</span>
          </div>
          <div className="flex items-center gap-2">
            <Clock className="w-3 h-3 text-orange-500" />
            <span className="text-xs">服务窗口: {record.businessHours}</span>
          </div>
        </div>
      ),
    },
    {
      title: '升级规则',
      key: 'escalation',
      render: (_, record) =>
        record.escalationLevels.length === 0 ? (
          <Text type="secondary">未配置</Text>
        ) : (
          <Space wrap size={4}>
            {record.escalationLevels.map(level => (
              <Tag key={level.level}>{formatEscalationLevel(level)}</Tag>
            ))}
          </Space>
        ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      align: 'center',
      render: (status: SLADefinition['status']) => (
        <Tag color={STATUS_CONFIG[status].color}>{STATUS_CONFIG[status].label}</Tag>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      align: 'center',
      render: (updatedAt: string) => <Text style={{ fontSize: 12 }}>{displayDateTime(updatedAt)}</Text>,
    },
    {
      title: '操作',
      key: 'actions',
      align: 'center',
      render: (_, record) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              aria-label="查看详情"
              type="text"
              icon={<Eye className="w-4 h-4" />}
              onClick={() => setDetailSLA(record)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              aria-label="编辑"
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => openEdit(record)}
            />
          </Tooltip>
          <Tooltip title={record.status === 'active' ? '停用' : '启用'}>
            <Button
              aria-label={record.status === 'active' ? '停用' : '启用'}
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
            title="确定要删除这个 SLA 定义吗？"
            description={`删除「${record.name}」后无法恢复，引用它的工单将失去 SLA 保障。`}
            onConfirm={() => handleDelete(record)}
            okText="确定删除"
            cancelText="取消"
            okType="danger"
          >
            <Button aria-label="删除" type="text" danger icon={<Trash2 className="w-4 h-4" />} />
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
          SLA 定义管理
        </Title>
        <Text type="secondary">定义和管理服务级别协议，确保服务质量标准</Text>
      </div>

      <Row gutter={[16, 16]} align="stretch">
        <Col xs={24} sm={12} lg={6} className="flex">
          <Card className="enterprise-card h-full w-full">
            <Statistic
              title="SLA 定义总数"
              value={stats.total}
              prefix={<Target className="w-5 h-5" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6} className="flex">
          <Card className="enterprise-card h-full w-full">
            <Statistic
              title="已启用"
              value={stats.active}
              prefix={<CheckCircle className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6} className="flex">
          <Card className="enterprise-card h-full w-full">
            <Statistic
              title="已停用"
              value={stats.inactive}
              prefix={<Clock className="w-5 h-5" />}
              styles={{ content: { color: '#8c8c8c' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6} className="flex">
          <Card className="enterprise-card h-full w-full">
            <Statistic
              title="覆盖服务类型"
              value={stats.serviceTypeCount}
              suffix="类"
              prefix={<Layers className="w-5 h-5" />}
              styles={{ content: { color: '#722ed1' } }}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={8}>
            <Input
              placeholder="搜索 SLA 名称或描述..."
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              aria-label="优先级筛选"
              value={priorityFilter}
              onChange={setPriorityFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: '全部优先级' },
                ...SLA_PRIORITY_VALUES.map(value => ({ value, label: getPriorityLabel(value) })),
              ]}
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              aria-label="状态筛选"
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: '全部状态' },
                ...Object.entries(STATUS_CONFIG).map(([value, config]) => ({
                  value,
                  label: config.label,
                })),
              ]}
            />
          </Col>
          <Col xs={24} md={4}>
            <Select
              aria-label="服务类型筛选"
              value={serviceTypeFilter}
              onChange={setServiceTypeFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: '全部类型' },
                ...serviceTypes.map(type => ({ value: type, label: getServiceTypeLabel(type) })),
              ]}
            />
          </Col>
          <Col xs={24} md={4} className="text-right">
            <Button type="primary" icon={<Plus className="w-4 h-4" />} onClick={openCreate}>
              新建 SLA
            </Button>
          </Col>
        </Row>
      </Card>

      {loadError && (
        <Alert
          type="error"
          showIcon
          title="SLA 定义加载失败"
          description={loadError}
          action={
            <Button size="small" icon={<RefreshCw className="w-4 h-4" />} onClick={loadDefinitions}>
              重新加载
            </Button>
          }
        />
      )}

      {!loadError && serverTotal > definitions.length && (
        <Alert
          type="warning"
          showIcon
          title={`服务端共有 ${serverTotal} 条 SLA 定义，当前仅加载前 ${definitions.length} 条`}
          description="筛选与统计只覆盖已加载的部分，请精简定义数量或对接分页查询。"
        />
      )}

      <Card className="enterprise-card">
        {!loadError && !loading && definitions.length === 0 ? (
          <Empty description="还没有 SLA 定义，点击右上角「新建 SLA」创建" />
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
                `筛选出 ${total} 条（服务端共 ${serverTotal} 条）`,
            }}
            className="enterprise-table"
          />
        )}
      </Card>

      <Modal
        title={
          <span>
            <Target className="w-4 h-4 mr-2" />
            {editingSLA ? '编辑 SLA 定义' : '新建 SLA 定义'}
          </span>
        }
        open={showFormModal}
        onOk={handleSave}
        onCancel={closeForm}
        width={720}
        confirmLoading={saving}
        okText="保存"
        cancelText="取消"
        destroyOnHidden
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="SLA 名称"
                name="name"
                rules={[{ required: true, message: '请输入 SLA 名称' }]}
              >
                <Input placeholder="例如：Incident-P1-高" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="服务类型"
                name="serviceType"
                rules={[{ required: true, message: '请选择服务类型' }]}
              >
                <Select
                  showSearch
                  optionFilterProp="label"
                  placeholder="选择服务类型"
                  options={Object.entries(SLA_SERVICE_TYPE_LABELS).map(([value, label]) => ({
                    value,
                    label,
                  }))}
                />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} placeholder="说明这条 SLA 适用于什么场景" />
          </Form.Item>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                label="优先级"
                name="priority"
                initialValue="medium"
                rules={[{ required: true, message: '请选择优先级' }]}
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
                label="响应时限（分钟）"
                name="responseTimeMinutes"
                initialValue={60}
                rules={[{ required: true, message: '请输入响应时限' }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} placeholder="60" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                label="解决时限（分钟）"
                name="resolutionTimeMinutes"
                initialValue={240}
                rules={[{ required: true, message: '请输入解决时限' }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} placeholder="240" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            label="创建后启用"
            name="isActive"
            valuePropName="checked"
            initialValue={true}
            extra="后端创建接口不传 isActive 会按默认值建成停用状态，因此这里显式提交。"
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={
          <div className="flex items-center gap-2">
            <Eye className="w-5 h-5" />
            <span>SLA 定义详情</span>
          </div>
        }
        open={Boolean(detailSLA)}
        onCancel={() => setDetailSLA(null)}
        width={760}
        footer={[
          <Button key="close" onClick={() => setDetailSLA(null)}>
            关闭
          </Button>,
        ]}
      >
        {detailSLA && (
          <div className="space-y-6">
            <div>
              <Title level={4}>{detailSLA.name}</Title>
              <Text type="secondary">{detailSLA.description || '无描述'}</Text>
            </div>

            <Descriptions
              bordered
              size="small"
              column={2}
              items={[
                {
                  key: 'priority',
                  label: '优先级',
                  children: (
                    <Tag color={getPriorityColor(detailSLA.priority)}>
                      {getPriorityLabel(detailSLA.priority)}
                    </Tag>
                  ),
                },
                {
                  key: 'serviceType',
                  label: '服务类型',
                  children: getServiceTypeLabel(detailSLA.serviceType),
                },
                {
                  key: 'response',
                  label: '响应时限',
                  children: formatDuration(detailSLA.responseTimeMinutes),
                },
                {
                  key: 'resolution',
                  label: '解决时限',
                  children: formatDuration(detailSLA.resolutionTimeMinutes),
                },
                { key: 'window', label: '服务窗口', children: detailSLA.businessHours },
                {
                  key: 'status',
                  label: '状态',
                  children: (
                    <Tag color={STATUS_CONFIG[detailSLA.status].color}>
                      {STATUS_CONFIG[detailSLA.status].label}
                    </Tag>
                  ),
                },
                { key: 'created', label: '创建时间', children: displayDateTime(detailSLA.createdAt) },
                { key: 'updated', label: '更新时间', children: displayDateTime(detailSLA.updatedAt) },
              ]}
            />

            <div>
              <Title level={5}>升级规则</Title>
              {detailSLA.escalationLevels.length === 0 ? (
                <Text type="secondary">该定义未配置升级级别。</Text>
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
