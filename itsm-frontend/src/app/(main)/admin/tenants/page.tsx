'use client';

import {
  Building2,
  AlertCircle,
  CheckCircle,
  Clock,
  Plus,
  Search,
  Edit,
  Trash2,
  Eye,
  Users,
  Calendar,
  PauseCircle,
  PlayCircle,
} from 'lucide-react';

import React, { useState, useEffect } from 'react';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
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
  App,
  Tag,
  DatePicker,
  Drawer,
  Descriptions,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { TenantAPI } from '@/lib/api/tenant-api';
import type { TenantInitializationStatus } from '@/lib/api/api-config';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;

type TranslatorFn = (key: string, params?: Record<string, string | number>) => string;

const getTenantStatusConfig = (t: TranslatorFn) => ({
  active: {
    label: t('tenants.statusLabels.active'),
    color: 'success',
    icon: CheckCircle,
  },
  suspended: {
    label: t('tenants.statusLabels.suspended'),
    color: 'warning',
    icon: AlertCircle,
  },
  expired: { label: t('tenants.statusLabels.expired'), color: 'error', icon: AlertCircle },
  deleted: { label: t('tenants.statusLabels.deleted'), color: 'default', icon: AlertCircle },
});

const getTenantTypeConfig = (t: TranslatorFn) => ({
  standard: { label: t('tenants.typeLabels.standard'), color: 'blue' },
  internal: { label: t('tenants.typeLabels.internal'), color: 'cyan' },
  saasCustomer: { label: t('tenants.typeLabels.saasCustomer'), color: 'green' },
  mspProvider: { label: t('tenants.typeLabels.mspProvider'), color: 'gold' },
  mspCustomer: { label: t('tenants.typeLabels.mspCustomer'), color: 'purple' },
  msp: { label: t('tenants.typeLabels.msp'), color: 'orange' },
  customer: { label: t('tenants.typeLabels.customer'), color: 'default' },
});

type TenantStatusKey = 'active' | 'suspended' | 'expired' | 'deleted';
type TenantTypeKey = 'standard' | 'internal' | 'saasCustomer' | 'mspProvider' | 'mspCustomer' | 'msp' | 'customer';

type Tenant = {
  id: number;
  name: string;
  code: string;
  domain?: string;
  type: TenantTypeKey;
  status: TenantStatusKey;
  userCount?: number;
  ticketCount?: number;
  expiresAt?: string;
};

type InitStatusEntry = TenantInitializationStatus | 'loading' | 'error';

type TenantFormValues = {
  name: string;
  code: string;
  domain?: string;
  type: string;
  status: string;
  expiresAt?: Dayjs;
};

export default function TenantManagement() {
  const { message } = App.useApp();
  const { t } = useI18n();
  const TENANT_STATUS = getTenantStatusConfig(t);
  const TENANT_TYPES = getTenantTypeConfig(t);
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(null);
  const [viewOnly, setViewOnly] = useState(false);
  const [form] = Form.useForm();
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [typeFilter, setTypeFilter] = useState('all');
  // 初始化状态按需加载并缓存：状态接口逐组件验证较重，不在列表加载时强拉，
  // 而是后台限流补齐当前页，单行也可手动查询。
  const [initStatuses, setInitStatuses] = useState<Record<number, InitStatusEntry>>({});
  const [initDrawerTenant, setInitDrawerTenant] = useState<Tenant | null>(null);
  const [replaying, setReplaying] = useState(false);
  const [stats, setStats] = useState({
    total: 0,
    active: 0,
    suspended: 0,
    expired: 0,
  });

  // 加载租户数据
  const loadTenants = async () => {
    setLoading(true);
    try {
      const response = await TenantAPI.getTenants({
        search: searchTerm || undefined,
        status: statusFilter !== 'all' ? statusFilter : undefined,
        type: typeFilter !== 'all' ? typeFilter : undefined,
      });

      setTenants(response.items as Tenant[]);
      // 后台串行补齐当前页初始化状态：不阻塞表格渲染，单元格按需点亮
      void loadInitStatusesFor(response.items as Tenant[]);

      // 计算统计数据
      const total = response.items.length;
      const active = response.items.filter(
        (item: { status: string }) => item.status === 'active'
      ).length;
      const suspended = response.items.filter(
        (item: { status: string }) => item.status === 'suspended'
      ).length;
      const expired = response.items.filter(
        (item: { status: string }) => item.status === 'expired'
      ).length;

      setStats({
        total,
        active,
        suspended,
        expired,
      });
    } catch (error) {
      message.error(t('tenants.loadFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 初始化加载数据
  useEffect(() => {
    loadTenants();
  }, [searchTerm, statusFilter, typeFilter]);

  // 处理保存租户
  const handleSaveTenant = async () => {
    try {
      const values = (await form.validateFields()) as TenantFormValues;
      const payload = {
        name: values.name,
        domain: values.domain,
        type: values.type,
        status: values.status,
        expiresAt: values.expiresAt ? values.expiresAt.toISOString() : undefined,
      };

      if (selectedTenant) {
        // 更新租户
        await TenantAPI.updateTenant(selectedTenant.id, payload);
        message.success(t('tenants.updateSuccess'));
      } else {
        // 创建租户
        await TenantAPI.createTenant({
          ...payload,
          code: values.code,
        });
        message.success(t('tenants.createSuccess'));
      }

      setShowModal(false);
      form.resetFields();
      setSelectedTenant(null);
      loadTenants(); // 重新加载数据
    } catch (error) {
      message.error(t('tenants.saveFailed'));
    }
  };

  const openTenantModal = (tenant: Tenant | null, readonly = false) => {
    setSelectedTenant(tenant);
    setViewOnly(readonly);
    if (tenant) {
      form.setFieldsValue({
        ...tenant,
        expiresAt: tenant.expiresAt ? dayjs(tenant.expiresAt) : undefined,
      });
    } else {
      form.resetFields();
    }
    setShowModal(true);
  };

  // 处理删除租户
  const handleDeleteTenant = async (id: number) => {
    try {
      await TenantAPI.deleteTenant(id);
      message.success(t('tenants.deleteSuccess'));
      loadTenants(); // 重新加载数据
    } catch (error) {
      message.error(t('tenants.deleteFailed'));
    }
  };

  const handleChangeTenantStatus = async (tenant: Tenant, status: TenantStatusKey) => {
    setLoading(true);
    try {
      await TenantAPI.updateTenant(tenant.id, { status });
      message.success(t('tenants.statusUpdateSuccess', { status: TENANT_STATUS[status].label }));
      loadTenants();
    } catch (error) {
      message.error(t('tenants.statusUpdateFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 查询并缓存单个租户的初始化状态（只读）
  const loadInitStatus = async (tenantId: number) => {
    setInitStatuses(prev => ({ ...prev, [tenantId]: 'loading' }));
    try {
      const status = await TenantAPI.getInitializationStatus(tenantId);
      setInitStatuses(prev => ({ ...prev, [tenantId]: status }));
    } catch {
      setInitStatuses(prev => ({ ...prev, [tenantId]: 'error' }));
    }
  };

  // 后台限流补齐当前页状态：状态接口逐组件验证较重，逐个串行拉取避免压垮后端
  const loadInitStatusesFor = async (list: Tenant[]) => {
    for (const tenant of list) {
      if (initStatuses[tenant.id] === undefined) {
        await loadInitStatus(tenant.id);
      }
    }
  };

  const openInitDrawer = async (record: Tenant) => {
    setInitDrawerTenant(record);
    await loadInitStatus(record.id);
  };

  const handleReplayInstallation = async () => {
    const entry = initDrawerTenant ? initStatuses[initDrawerTenant.id] : undefined;
    if (!entry || typeof entry === 'string' || !entry.commandId) {
      return;
    }
    setReplaying(true);
    try {
      await TenantAPI.replayInitializationCommand(entry.commandId);
      message.success(t('tenants.replaySuccess'));
      await loadInitStatus(initDrawerTenant!.id);
    } catch (error) {
      message.error(t('tenants.replayFailed'));
    } finally {
      setReplaying(false);
    }
  };

  // 表格列定义
  const columns: ColumnsType<Tenant> = [
    {
      title: t('tenants.columns.info'),
      key: 'info',
      width: 200,
      render: (_: unknown, record: Tenant) => (
        <div className="flex items-center">
          <div className="flex-shrink-0 h-10 w-10 rounded-full bg-blue-100 flex items-center justify-center">
            <Building2 className="h-5 w-5 text-blue-600" />
          </div>
          <div className="ml-4">
            <div className="text-sm font-medium text-gray-900">{record.name}</div>
            <div className="text-sm text-gray-500">
              {record.code} • {record.domain || ''}
            </div>
          </div>
        </div>
      ),
    },
    {
      title: t('tenants.columns.typeStatus'),
      key: 'type-status',
      width: 120,
      render: (_: unknown, record: Tenant) => (
        <div className="space-y-1">
          <Tag color={TENANT_TYPES[record.type]?.color || 'default'}>
            {TENANT_TYPES[record.type]?.label || record.type}
          </Tag>
          <div className="flex items-center">
            <Tag color={TENANT_STATUS[record.status]?.color || 'default'}>
              {TENANT_STATUS[record.status]?.label || record.status}
            </Tag>
          </div>
        </div>
      ),
    },
    {
      title: t('tenants.columns.usage'),
      key: 'usage',
      width: 130,
      render: (_: unknown, record: Tenant) => (
        <div className="space-y-1">
          <div className="flex items-center">
            <Users className="w-4 h-4 mr-1 text-gray-400" />
            <span>{record.userCount || 0} {t('tenants.usage.users')}</span>
          </div>
          <div className="text-xs text-gray-500">{record.ticketCount || 0} {t('tenants.usage.tickets')}</div>
        </div>
      ),
    },
    {
      title: t('tenants.columns.expires'),
      key: 'expires',
      dataIndex: 'expiresAt',
      width: 150,
      render: (expiresAt: string) => (
        <div className="flex items-center">
          <Calendar className="w-4 h-4 mr-1 text-gray-400" />
          {expiresAt ? new Date(expiresAt).toLocaleDateString() : t('tenants.usage.none')}
        </div>
      ),
    },
    {
      title: t('tenants.columns.initialization'),
      key: 'initialization',
      width: 150,
      render: (_: unknown, record: Tenant) => {
        const entry = initStatuses[record.id];
        if (entry === 'loading') {
          return <Text type="secondary">{t('tenants.initialization.querying')}</Text>;
        }
        if (entry === 'error') {
          return (
            <Button type="link" size="small" onClick={() => void loadInitStatus(record.id)}>
              {t('tenants.initialization.retry')}
            </Button>
          );
        }
        if (!entry) {
          return (
            <Button type="link" size="small" onClick={() => void openInitDrawer(record)}>
              {t('tenants.initialization.query')}
            </Button>
          );
        }
        const failed = entry.components.filter(item => !item.verified);
        return (
          <div className="space-y-1">
            <Tag color={entry.ready ? 'success' : 'error'}>
              {entry.ready ? t('tenants.initialization.baselineReady') : t('tenants.initialization.itemsNotReady', { count: failed.length })}
            </Tag>
            <Button type="link" size="small" onClick={() => void openInitDrawer(record)}>
              {t('tenants.initialization.details')}
            </Button>
          </div>
        );
      },
    },
    {
      title: t('tenants.columns.actions'),
      key: 'actions',
      width: 120,
      render: (_: unknown, record: Tenant) => (
        <Space size="small">
          <Tooltip title={t('tenants.actions.edit')}>
            <Button
              aria-label={t('tenants.actions.edit')}
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => openTenantModal(record)}
            />
          </Tooltip>
          <Tooltip title={t('tenants.actions.view')}>
            <Button
              aria-label={t('tenants.actions.view')}
              type="text"
              icon={<Eye className="w-4 h-4" />}
              onClick={() => openTenantModal(record, true)}
            />
          </Tooltip>
          {record.status === 'active' ? (
            record.code === 'default' ? (
              <Tooltip title={t('tenants.actions.cannotPauseDefault')}>
                <Button aria-label={t('tenants.actions.cannotPauseDefault')} type="text" disabled icon={<PauseCircle className="w-4 h-4" />} />
              </Tooltip>
            ) : (
              <Tooltip title={t('tenants.actions.pause')}>
                <Button
                  aria-label={t('tenants.actions.pause')}
                  type="text"
                  icon={<PauseCircle className="w-4 h-4" />}
                  onClick={() => handleChangeTenantStatus(record, 'suspended')}
                />
              </Tooltip>
            )
          ) : (
            <Tooltip title={t('tenants.actions.resume')}>
              <Button
                aria-label={t('tenants.actions.resume')}
                type="text"
                icon={<PlayCircle className="w-4 h-4" />}
                onClick={() => handleChangeTenantStatus(record, 'active')}
              />
            </Tooltip>
          )}
          {record.code === 'default' ? (
            <Tooltip title={t('tenants.actions.cannotDeleteDefault')}>
              <Button aria-label={t('tenants.actions.cannotDeleteDefault')} type="text" danger disabled icon={<Trash2 className="w-4 h-4" />} />
            </Tooltip>
          ) : (
            <Popconfirm
              title={t('tenants.confirmDelete')}
              description={t('tenants.deleteWarning')}
              onConfirm={() => handleDeleteTenant(record.id)}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
            >
              <Tooltip title={t('tenants.actions.delete')}>
                <Button aria-label={t('tenants.actions.delete')} type="text" danger icon={<Trash2 className="w-4 h-4" />} />
              </Tooltip>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title level={2} className="!mb-2">
          <Building2 className="inline-block w-6 h-6 mr-2" />
          {t('tenants.title')}
        </Title>
        <Text type="secondary">{t('tenants.description')}</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title={t('tenants.totalTenants')}
              value={stats.total}
              prefix={<Building2 className="w-5 h-5" />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title={t('tenants.activeTenants')}
              value={stats.active}
              prefix={<CheckCircle className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title={t('tenants.suspendedTenants')}
              value={stats.suspended}
              prefix={<AlertCircle className="w-5 h-5" />}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title={t('tenants.expiredTenants')}
              value={stats.expired}
              prefix={<Clock className="w-5 h-5" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索和过滤 */}
      <Card>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={12} lg={8}>
            <Input
              placeholder={t('tenants.searchPlaceholder')}
              prefix={<Search className="w-4 h-4 text-gray-400" />}
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={8} lg={4}>
            <Select
              placeholder={t('tenants.filterStatus')}
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: t('tenants.allStatus') },
                { value: 'active', label: t('tenants.statusLabels.active') },
                { value: 'suspended', label: t('tenants.statusLabels.suspended') },
                { value: 'expired', label: t('tenants.statusLabels.expired') },
                { value: 'deleted', label: t('tenants.statusLabels.deleted') },
              ]}
            />
          </Col>
          <Col xs={24} md={8} lg={4}>
            <Select
              placeholder={t('tenants.filterType')}
              value={typeFilter}
              onChange={setTypeFilter}
              style={{ width: '100%' }}
              options={[
                { value: 'all', label: t('tenants.allTypes') },
                { value: 'standard', label: t('tenants.typeLabels.standard') },
                { value: 'internal', label: t('tenants.typeLabels.internal') },
                { value: 'saas_customer', label: t('tenants.typeLabels.saasCustomer') },
                { value: 'msp_provider', label: t('tenants.typeLabels.mspProvider') },
                { value: 'msp_customer', label: t('tenants.typeLabels.mspCustomer') },
              ]}
            />
          </Col>
          <Col xs={24} md={4} lg={8} className="text-right">
            <Button
              type="primary"
              icon={<Plus className="w-4 h-4" />}
              onClick={() => {
                openTenantModal(null);
              }}
            >
              {t('tenants.create')}
            </Button>
          </Col>
        </Row>
      </Card>

      {/* 租户列表 */}
      <Card className="enterprise-card">
        <Table<Tenant>
          columns={columns}
          dataSource={tenants}
          rowKey="id"
          loading={loading}
          scroll={{ x: 920 }}
          pagination={{
            total: tenants.length,
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: total => t('tenants.pagination.total', { total }),
          }}
          className="enterprise-table"
        />
      </Card>

      {/* 产品基线安装状态抽屉 */}
      <Drawer
        title={initDrawerTenant ? t('tenants.initialization.drawerTitleWithTenant', { name: initDrawerTenant.name }) : t('tenants.initialization.drawerTitle')}
        open={Boolean(initDrawerTenant)}
        onClose={() => setInitDrawerTenant(null)}
        width={560}
        extra={
          initDrawerTenant ? (
            <Space>
              <Button
                size="small"
                onClick={() => void loadInitStatus(initDrawerTenant.id)}
              >
                {t('tenants.initialization.refresh')}
              </Button>
              {(() => {
                const entry = initStatuses[initDrawerTenant.id];
                const replayable =
                  entry &&
                  typeof entry !== 'string' &&
                  entry.commandId !== undefined &&
                  entry.commandStatus === 'dead_letter';
                return replayable ? (
                  <Button
                    size="small"
                    type="primary"
                    loading={replaying}
                    onClick={() => void handleReplayInstallation()}
                  >
                    {t('tenants.initialization.replayInstall')}
                  </Button>
                ) : null;
              })()}
            </Space>
          ) : null
        }
      >
        {(() => {
          const entry = initDrawerTenant ? initStatuses[initDrawerTenant.id] : undefined;
          if (!entry) {
            return <Text type="secondary">{t('tenants.initialization.notQueried')}</Text>;
          }
          if (entry === 'loading') {
            return <Text type="secondary">{t('tenants.initialization.querying')}</Text>;
          }
          if (entry === 'error') {
            return <Text type="danger">{t('tenants.initialization.queryFailed')}</Text>;
          }
          return (
            <div className="space-y-4">
              <Descriptions column={1} size="small" bordered>
                <Descriptions.Item label={t('tenants.initialization.baselineReadyLabel')}>
                  <Tag color={entry.ready ? 'success' : 'error'}>{entry.ready ? t('tenants.initialization.yes') : t('tenants.initialization.no')}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('tenants.initialization.installCommand')}>
                  {entry.commandStatus}
                  {entry.commandAttempts > 0 ? t('tenants.initialization.attemptCount', { count: entry.commandAttempts }) : ''}
                </Descriptions.Item>
                <Descriptions.Item label={t('tenants.initialization.templateVersion')}>{entry.templateVersion}</Descriptions.Item>
                {entry.recordedVersion ? (
                  <Descriptions.Item label={t('tenants.initialization.recordedVersion')}>{entry.recordedVersion}</Descriptions.Item>
                ) : null}
                {entry.commandError ? (
                  <Descriptions.Item label={t('tenants.initialization.commandError')}>
                    <Text type="danger">{entry.commandError}</Text>
                  </Descriptions.Item>
                ) : null}
              </Descriptions>
              <Table<{ component: string; verified: boolean; error?: string }>
                rowKey="component"
                size="small"
                pagination={false}
                dataSource={entry.components}
                columns={[
                  { title: t('tenants.initialization.component'), dataIndex: 'component' },
                  {
                    title: t('tenants.initialization.status'),
                    dataIndex: 'verified',
                    width: 90,
                    render: (verified: boolean) => (
                      <Tag color={verified ? 'success' : 'error'}>{verified ? t('tenants.initialization.ready') : t('tenants.initialization.notReady')}</Tag>
                    ),
                  },
                  {
                    title: t('tenants.initialization.gap'),
                    dataIndex: 'error',
                    render: (error?: string) => error || '—',
                  },
                ]}
              />
            </div>
          );
        })()}
      </Drawer>

      {/* 租户编辑模态框 */}
      <Modal
        title={
          <span>
            {selectedTenant ? (
              <>
                <Edit className="w-4 h-4 mr-2" />
                {viewOnly ? t('tenants.view') : t('tenants.edit')}
              </>
            ) : (
              <>
                <Plus className="w-4 h-4 mr-2" />
                {t('tenants.create')}
              </>
            )}
          </span>
        }
        open={showModal}
        onOk={viewOnly ? undefined : handleSaveTenant}
        onCancel={() => {
          setShowModal(false);
          setSelectedTenant(null);
          setViewOnly(false);
          form.resetFields();
        }}
        width={600}
        confirmLoading={loading}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        footer={
          viewOnly
            ? [
                <Button
                  key="close"
                  onClick={() => {
                    setShowModal(false);
                    setSelectedTenant(null);
                    setViewOnly(false);
                    form.resetFields();
                  }}
                >
                  {t('common.close')}
                </Button>,
              ]
            : undefined
        }
      >
        <Form form={form} layout="vertical" className="mt-4" disabled={viewOnly} initialValues={{ type: 'standard', status: 'active' }}>
          <Form.Item
            label={t('tenants.tenantName')}
            name="name"
            rules={[{ required: true, message: t('tenants.form.nameRequired') }]}
          >
            <Input placeholder={t('tenants.form.namePlaceholder')} />
          </Form.Item>

          <Form.Item
            label={t('tenants.tenantCode')}
            name="code"
            rules={[
              { required: true, message: t('tenants.form.codeRequired') },
              {
                pattern: /^[a-zA-Z0-9][a-zA-Z0-9_-]*$/,
                message: t('tenants.form.codePattern'),
              },
            ]}
          >
            <Input
              disabled={!!selectedTenant}
              placeholder={t('tenants.form.codePlaceholder')}
            />
          </Form.Item>

          <Form.Item label={t('tenants.domain')} name="domain">
            <Input placeholder={t('tenants.form.domainPlaceholder')} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label={t('tenants.tenantType')}
                name="type"
                rules={[{ required: true, message: t('tenants.form.typeRequired') }]}
              >
                <Select placeholder={t('tenants.form.typePlaceholder')} options={[
                  { value: 'standard', label: t('tenants.typeLabels.standard') },
                  { value: 'internal', label: t('tenants.typeLabels.internal') },
                  { value: 'saas_customer', label: t('tenants.typeLabels.saasCustomer') },
                  { value: 'msp_provider', label: t('tenants.typeLabels.mspProvider') },
                  { value: 'msp_customer', label: t('tenants.typeLabels.mspCustomer') },
                ]} />
              </Form.Item>
            </Col>

            <Col span={12}>
              <Form.Item
                label={t('tenants.status')}
                name="status"
                rules={[{ required: true, message: t('tenants.form.statusRequired') }]}
              >
                <Select placeholder={t('tenants.form.statusPlaceholder')} options={[
                  { value: 'active', label: t('tenants.statusLabels.active') },
                  { value: 'suspended', label: t('tenants.statusLabels.suspended') },
                  { value: 'expired', label: t('tenants.statusLabels.expired') },
                  { value: 'deleted', label: t('tenants.statusLabels.deleted') },
                ]} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item label={t('tenants.expiresAt')} name="expiresAt">
            <DatePicker style={{ width: '100%' }} placeholder={t('tenants.form.expiresAtPlaceholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
