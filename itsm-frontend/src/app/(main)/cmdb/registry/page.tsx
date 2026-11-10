'use client';

import React from 'react';
import {
  Alert,
  App,
  Badge,
  Button,
  Card,
  Col,
  Form,
  Input,
  Modal,
  Popover,
  Row,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import { Cloud, Database, RefreshCw, Server, Sparkles } from 'lucide-react';

import type {
  CMDBRuntimeCapability,
  DiscoveryJob,
  DiscoveryScopeFailure,
  PendingDiscoveryAttribute,
  PendingDiscoveryResource,
  RegisterPendingDiscoveryResourceRequest,
} from '@/lib/api/cmdb-api';
import { ApiRequestError } from '@/lib/api/http-client';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview, type StatsOverviewItem } from '@/components/ui/StatsOverview';
import { formatDateTime } from '@/lib/formatters';
import {
  useCapabilitiesQuery,
  useCloudAccountsQuery,
  useCloudServicesQuery,
  useDiscoveryHistoryQuery,
  useDiscoverySourcesQuery,
  usePendingDiscoveryResourcesQuery,
  useRegisterPendingDiscoveryResourceMutation,
} from '@/lib/hooks/useCMDB';

const { Text } = Typography;

const ENVIRONMENT_OPTIONS = [
  { value: 'production', label: 'Production' },
  { value: 'staging', label: 'Pre-release' },
  { value: 'development', label: 'Development' },
];

// 必填属性的初始值：优先沿用云标签命中的现值，其次回落属性默认值。
// 两者都填过的属性即便云端本已满足也允许操作员覆盖——后端按「提交值优先」合并。
const initialAttrValue = (attr: PendingDiscoveryAttribute): string =>
  attr.currentValue || attr.defaultValue || '';

// 状态字符串由执行器落库，展示层只做翻译，不要用 success 之类的展示词反推后端状态。
const DISCOVERY_JOB_STATUS: Record<string, { label: string; color: string }> = {
  queued: { label: 'Queuing', color: 'default' },
  discovering: { label: 'Discovering', color: 'processing' },
  discovered: { label: 'Found', color: 'blue' },
  reconciling: { label: 'Reconciling', color: 'processing' },
  succeeded: { label: 'Success', color: 'green' },
  partial_failed: { label: 'Partial failure', color: 'orange' },
  failed: { label: 'Failed', color: 'red' },
  cancelled: { label: 'Cancelled', color: 'default' },
};

const discoveryJobStatusMeta = (status: string) =>
  DISCOVERY_JOB_STATUS[status] ?? { label: status || '-', color: 'default' };

const discoveryScopeFailures = (record: DiscoveryJob): DiscoveryScopeFailure[] =>
  record.summary?.failedScopeDetails ?? [];

interface CompletionFormValues {
  name?: string;
  environment?: string;
  assignedTo?: string;
  attributes?: Record<string, string>;
}

interface CompletionModalProps {
  resource: PendingDiscoveryResource | null;
  onClose: () => void;
}

function CompletionModal({ resource, onClose }: CompletionModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<CompletionFormValues>();
  const registerMutation = useRegisterPendingDiscoveryResourceMutation();
  const [submitting, setSubmitting] = React.useState(false);

  React.useEffect(() => {
    if (!resource) return;
    form.resetFields();
    const attributes: Record<string, string> = {};
    resource.requiredAttributes.forEach(attr => {
      const value = initialAttrValue(attr);
      if (value) attributes[attr.name] = value;
    });
    form.setFieldsValue({
      name: resource.resourceName,
      attributes,
    });
  }, [resource, form]);

  const handleClose = () => {
    if (submitting) return;
    onClose();
  };

  const handleSubmit = async () => {
    if (!resource) return;
    let values: CompletionFormValues;
    try {
      values = await form.validateFields();
    } catch {
      return; // 校验失败由表单就地提示
    }

    const attributes: Record<string, unknown> = {};
    Object.entries(values.attributes ?? {}).forEach(([key, value]) => {
      // 只提交非空值：空串会覆盖云端已推断的标签，破坏后端合并语义。
      if (typeof value === 'string' && value.trim() !== '') attributes[key] = value.trim();
    });

    const payload: RegisterPendingDiscoveryResourceRequest = { attributes };
    if (values.name?.trim()) payload.name = values.name.trim();
    if (values.environment) payload.environment = values.environment;
    if (values.assignedTo?.trim()) payload.assignedTo = values.assignedTo.trim();

    setSubmitting(true);
    try {
      const ci = await registerMutation.mutateAsync({ id: resource.id, data: payload });
      message.success(`Account has been established:${ci.name}`);
      onClose();
    } catch (error) {
      if (error instanceof ApiRequestError) {
        if (error.status === 400 || error.code === 4000) {
          // 必填/属性校验失败：保留弹窗与已填内容，让操作员补齐。
          message.error(error.message || 'Required attributes are not completed, please check and try again');
        } else if (error.status === 409 || error.code === 4090) {
          message.warning('The cloud resource has been accounted for and the list will be automatically refreshed later.');
          onClose();
        } else if (error.status === 503 || error.code === 5003) {
          message.warning('Manual completion is temporarily unavailable: it was found that the writeback service is not installed');
        } else if (error.status === 404 || error.code === 4004) {
          message.error('The cloud resource does not exist or does not belong to the current tenant');
          onClose();
        } else {
          message.error(error.message || 'Failed to create additional account');
        }
      } else {
        message.error(error instanceof Error ? error.message : 'Failed to create additional account');
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={resource ? `Supplementary entry and account creation:${resource.resourceId}` : 'Supplementary entry and account creation'}
      open={!!resource}
      okText="Create account"
      confirmLoading={submitting}
      cancelText="Cancel"
      onCancel={handleClose}
      onOk={handleSubmit}
      destroyOnHidden
    >
      {resource ? (
        <>
          <Text type="secondary">

            Target CI type:{resource.targetCiType};Missing required fields:
            {resource.missingAttributeNames.join('、') || '(Prefilled, can be overridden)'}
          </Text>
          <Form form={form} layout="vertical" className="mt-4" disabled={submitting}>
            <Form.Item name="name" label="CI name">
              <Input placeholder="Leave blank to use the cloud resource name." allowClear />
            </Form.Item>
            <Form.Item name="environment" label="Environment">
              <Select
                placeholder="Leave blank to use label inference"
                options={ENVIRONMENT_OPTIONS}
                allowClear
              />
            </Form.Item>
            <Form.Item name="assignedTo" label="Assignee">
              <Input placeholder="Leave blank to use label inference" allowClear />
            </Form.Item>
            {resource.requiredAttributes.map(attr => (
              <Form.Item
                key={attr.name}
                name={['attributes', attr.name]}
                label={attr.displayName || attr.name}
                tooltip={attr.helpText || undefined}
                rules={[{ required: true, message: `${attr.displayName || attr.name} is required` }]}
              >
                {attr.enumValues && attr.enumValues.length > 0 ? (
                  <Select
                    placeholder={attr.placeholder || 'Please select'}
                    options={attr.enumValues.map(value => ({ value, label: value }))}
                    allowClear
                  />
                ) : (
                  <Input placeholder={attr.placeholder || 'Please input'} allowClear />
                )}
              </Form.Item>
            ))}
          </Form>
        </>
      ) : null}
    </Modal>
  );
}

const normalizeList = <T,>(response: unknown): T[] => {
  if (Array.isArray(response)) return response as T[];
  if (response && typeof response === 'object') {
    const record = response as Record<string, unknown>;
    if (Array.isArray(record.data)) return record.data as T[];
    if (Array.isArray(record.items)) return record.items as T[];
  }
  return [];
};

export default function ServiceGraphRegistryPage() {
  const { message } = App.useApp();

  const [pendingPage, setPendingPage] = React.useState(1);
  const [pendingPageSize, setPendingPageSize] = React.useState(8);
  const [completionTarget, setCompletionTarget] = React.useState<PendingDiscoveryResource | null>(
    null,
  );

  // React Query：5 路数据并行加载（替代手写 Promise.all + setState）
  const sourcesQuery = useDiscoverySourcesQuery();
  const servicesQuery = useCloudServicesQuery();
  const accountsQuery = useCloudAccountsQuery();
  const historyQuery = useDiscoveryHistoryQuery();
  const capabilitiesQuery = useCapabilitiesQuery();
  const pendingQuery = usePendingDiscoveryResourcesQuery({
    page: pendingPage,
    pageSize: pendingPageSize,
  });

  const queries = [sourcesQuery, servicesQuery, accountsQuery, historyQuery, capabilitiesQuery];
  const loading = queries.some(q => q.isLoading);
  const fetching = queries.some(q => q.isFetching) || pendingQuery.isFetching;
  const hasError = queries.some(q => q.isError);

  React.useEffect(() => {
    if (hasError) {
      message.error('Failed to load map registration center');
    }
  }, [hasError, message]);

  const handleRefresh = () => {
    queries.forEach(q => q.refetch());
    pendingQuery.refetch();
  };

  const discoverySources = normalizeList<Record<string, unknown>>(sourcesQuery.data);
  const cloudServices = servicesQuery.data?.items ?? [];
  const cloudAccounts = accountsQuery.data?.items ?? [];
  const discoveryHistory = historyQuery.data?.items ?? [];
  const pendingResources = pendingQuery.data?.items ?? [];

  const discoveryCapability: CMDBRuntimeCapability | null =
    capabilitiesQuery.data?.items.find(item => item.key === 'cmdbDiscovery') ?? null;

  // 最后刷新时间取数据最近一次成功落地的时刻
  const refreshAt = queries.some(q => q.dataUpdatedAt > 0)
    ? new Date(Math.max(...queries.map(q => q.dataUpdatedAt))).toISOString()
    : null;

  const statsItems: StatsOverviewItem[] = [
    {
      key: 'sources',
      title: 'Discovery Source',
      value: discoverySources.length,
      prefix: <Sparkles className="text-cyan-500 mr-2" />,
      accentColor: '#13c2c2',
    },
    {
      key: 'services',
      title: 'Cloud services',
      value: cloudServices.length,
      prefix: <Database className="text-blue-500 mr-2" />,
      accentColor: '#1890ff',
    },
    {
      key: 'accounts',
      title: 'Cloud account',
      value: cloudAccounts.length,
      prefix: <Cloud className="text-green-500 mr-2" />,
      accentColor: '#52c41a',
    },
    {
      key: 'jobs',
      title: 'Record found',
      value: discoveryHistory.length,
      prefix: <Server className="text-orange-500 mr-2" />,
      accentColor: '#fa8c16',
    },
    {
      key: 'pending',
      title: 'Resources to be established',
      value: pendingQuery.data?.total ?? 0,
      prefix: <Cloud className="text-rose-500 mr-2" />,
      accentColor: '#f5222d',
    },
  ];

  const sourceColumns = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'Type',
      dataIndex: 'sourceType',
      key: 'sourceType',
      render: (value: string) => <Tag color="blue">{value || '-'}</Tag>,
    },
    {
      title: 'Provider',
      dataIndex: 'provider',
      key: 'provider',
      render: (value?: string) => value || '-',
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (value: boolean) => <Badge status={value ? 'success' : 'default'} text={value ? 'Enabled' : 'Disable'} />,
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      render: (value?: string) => value || '-',
    },
  ];

  const serviceColumns = [
    {
      title: 'Vendor',
      dataIndex: 'provider',
      key: 'provider',
    },
    {
      title: 'Service code',
      dataIndex: 'serviceCode',
      key: 'serviceCode',
    },
    {
      title: 'Service Name',
      dataIndex: 'serviceName',
      key: 'serviceName',
    },
    {
      title: 'Resource Type',
      dataIndex: 'resourceTypeName',
      key: 'resourceTypeName',
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (value: boolean) => <Tag color={value ? 'green' : 'default'}>{value ? 'Enabled' : 'Disable'}</Tag>,
    },
  ];

  const accountColumns = [
    {
      title: 'Vendor',
      dataIndex: 'provider',
      key: 'provider',
    },
    {
      title: 'Account name',
      dataIndex: 'accountName',
      key: 'accountName',
    },
    {
      title: 'Account ID',
      dataIndex: 'accountId',
      key: 'accountId',
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (value: boolean) => <Tag color={value ? 'green' : 'default'}>{value ? 'Enabled' : 'Disable'}</Tag>,
    },
  ];

  const historyColumns = [
    {
      title: 'Task ID',
      dataIndex: 'id',
      key: 'id',
    },
    {
      title: 'Source ID',
      dataIndex: 'sourceId',
      key: 'sourceId',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (value: string) => {
        const meta = discoveryJobStatusMeta(value);
        return <Tag color={meta.color}>{meta.label}</Tag>;
      },
    },
    {
      title: 'Failure reason',
      key: 'failure',
      render: (_: unknown, record: DiscoveryJob) => {
        const failures = discoveryScopeFailures(record);
        if (failures.length > 0) {
          return (
            <Popover
              title="Failure range"
              content={
                <div className="max-w-md space-y-2">
                  {failures.map((item, index) => (
                    <div key={`${item.serviceCode}-${item.region}-${index}`} className="text-xs">
                      <div className="font-medium">
                        {item.serviceCode}/{item.region}
                        <span className="ml-1 text-gray-500">{item.errorCode || 'scope_failed'}</span>
                      </div>
                      <div className="break-all text-gray-600">{item.message || '-'}</div>
                    </div>
                  ))}
                </div>
              }
            >
              <Tag color="red" className="cursor-pointer">
                {failures.length}  range failed
              </Tag>
            </Popover>
          );
        }
        if (record.errorMessage) {
          return <Text type="danger">{record.errorMessage}</Text>;
        }
        if (record.errorCode) {
          return <Text type="danger">{record.errorCode}</Text>;
        }
        return '-';
      },
    },
    {
      title: 'Start Time',
      dataIndex: 'startedAt',
      key: 'startedAt',
      render: (value?: string) => formatDateTime(value) || '-',
    },
    {
      title: 'End Time',
      dataIndex: 'finishedAt',
      key: 'finishedAt',
      render: (value?: string) => formatDateTime(value) || '-',
    },
  ];

  const pendingColumns = [
    {
      title: 'Resource ID',
      dataIndex: 'resourceId',
      key: 'resourceId',
    },
    {
      title: 'Name',
      dataIndex: 'resourceName',
      key: 'resourceName',
      render: (value: string) => value || '-',
    },
    {
      title: 'Vendor',
      dataIndex: 'provider',
      key: 'provider',
    },
    {
      title: 'Area',
      dataIndex: 'region',
      key: 'region',
      render: (value: string) => value || '-',
    },
    {
      title: 'Target CI type',
      dataIndex: 'targetCiType',
      key: 'targetCiType',
      render: (value: string) => <Tag color="geekblue">{value || '-'}</Tag>,
    },
    {
      title: 'Missing required fields',
      dataIndex: 'missingAttributeNames',
      key: 'missingAttributeNames',
      render: (values: string[]) =>
        values?.length ? (
          <Space size={4} wrap>
            {values.map(name => (
              <Tag key={name} color="red">
                {name}
              </Tag>
            ))}
          </Space>
        ) : (
          <Text type="secondary">Prefilled</Text>
        ),
    },
    {
      title: 'Last found',
      dataIndex: 'lastSeenAt',
      key: 'lastSeenAt',
      render: (value?: string) => formatDateTime(value) || '-',
    },
    {
      title: 'Actions',
      key: 'action',
      render: (_: unknown, record: PendingDiscoveryResource) => (
        <Button type="link" onClick={() => setCompletionTarget(record)}>

          Supplementary entry and account creation
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="Service Graph Registry"
        description="Centrally manage access information and history of discovery sources, cloud services, and cloud accounts."
        actions={
          <Space wrap>
            <Button icon={<RefreshCw className="h-4 w-4" />} loading={fetching} onClick={handleRefresh}>

              Refresh
            </Button>
            <Button type="primary" href="/cmdb/relationships" icon={<Database className="h-4 w-4" />}>

              Remove the relationship graph
            </Button>
          </Space>
        }
      />

      {discoveryCapability?.state !== 'ready' ? (
        <Alert
          type="warning"
          showIcon
          title="Cloud resource auto-discovery is not ready yet"
          description={
            discoveryCapability
              ? `Current status:${discoveryCapability.state};Missing condition:${discoveryCapability.missingRequirements.join('、') || 'Tenant configuration'}`
              : 'Unable to read backend capability status; backend will reject autodiscover request as not ready.'
          }
        />
      ) : null}

      <StatsOverview items={statsItems} />

      <Card loading={loading} className="shadow-sm">
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Card size="small" className="h-full bg-slate-50">
              <Text type="secondary">Last refreshed</Text>
              <div className="mt-2 text-lg font-semibold">
                {formatDateTime(refreshAt) || '-'}
              </div>
            </Card>
          </Col>
          <Col xs={24} md={12}>
            <Card size="small" className="h-full bg-slate-50">
              <Text type="secondary">Registration center description</Text>
              <div className="mt-2">

                Only by first defining discovery sources and cloud services, and then linking cloud accounts with CIs, relationships, and reconciliations can a sustainable graph governance closed loop be formed.
              </div>
            </Card>
          </Col>
        </Row>
      </Card>

      <Card>
        <Tabs
          items={[
            {
              key: 'sources',
              label: 'Discovery Source',
              children: <Table rowKey="id" columns={sourceColumns} dataSource={discoverySources} pagination={{ pageSize: 8 }} />,
            },
            {
              key: 'services',
              label: 'Cloud services',
              children: <Table rowKey="id" columns={serviceColumns} dataSource={cloudServices} pagination={{ pageSize: 8 }} />,
            },
            {
              key: 'accounts',
              label: 'Cloud account',
              children: <Table rowKey="id" columns={accountColumns} dataSource={cloudAccounts} pagination={{ pageSize: 8 }} />,
            },
            {
              key: 'history',
              label: 'Discovery history',
              children: <Table rowKey="id" columns={historyColumns} dataSource={discoveryHistory} pagination={{ pageSize: 8 }} />,
            },
            {
              key: 'pending',
              label: `Resources to be established ${pendingQuery.data?.total ? ` (${pendingQuery.data.total})` : ''}`,
              children: (
                <Table
                  rowKey="id"
                  columns={pendingColumns}
                  dataSource={pendingResources}
                  loading={pendingQuery.isLoading || pendingQuery.isFetching}
                  locale={{ emptyText: 'There are no cloud resources that require manual completion' }}
                  pagination={{
                    current: pendingPage,
                    pageSize: pendingPageSize,
                    total: pendingQuery.data?.total ?? 0,
                    showSizeChanger: true,
                    onChange: (page, pageSize) => {
                      setPendingPage(page);
                      setPendingPageSize(pageSize ?? 8);
                    },
                  }}
                />
              ),
            },
          ]}
        />
      </Card>

      <CompletionModal resource={completionTarget} onClose={() => setCompletionTarget(null)} />
    </div>
  );
}
