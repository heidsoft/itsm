'use client';

import React, { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Breadcrumb,
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  App,
  Tooltip,
} from 'antd';
import { Search, Plus, Eye, RotateCcw, ShieldAlert } from 'lucide-react';
import dayjs from 'dayjs';

import { CMDBApi } from '@/lib/api/cmdb-api';
import CISearchSelect, { type CISelectOption } from '@/components/cmdb/CISearchSelect';
import { CMDB_KEYS, useCloudResourcesQuery, useCloudServicesQuery } from '@/lib/hooks/useCMDB';
import type { CloudResource } from '@/types/biz/cmdb';

// 与后端 cloud.KnownProviders 同源（后端 dto 枚举由守卫测试强制对齐）。
// 火山引擎与 k8s 的发现适配器已接入，缺这两项时前端根本选不到、能力等于零。
const providerOptions = [
  { value: 'aliyun', label: 'Aliyun' },
  { value: 'huawei', label: 'Huawei Cloud' },
  { value: 'tencent', label: 'Tencent Cloud' },
  { value: 'azure', label: 'Azure' },
  { value: 'aws', label: 'AWS' },
  { value: 'volcengine', label: 'Volcengine' },
  { value: 'k8s', label: 'Kubernetes' },
  { value: 'onprem', label: 'Private Cloud' },
];

const statusColors: Record<string, string> = {
  running: 'green',
  stopped: 'default',
  active: 'green',
  inactive: 'default',
  available: 'green',
  unavailable: 'red',
};

// 云资源状态中文映射
const cloudResourceStatusTextMap: Record<string, string> = {
  running: 'Running',
  stopped: 'Stopped',
  active: 'Active',
  inactive: 'Inactive',
  available: 'Available',
  unavailable: 'Not available',
  pending: 'In Progress',
  failed: 'Failed',
};

// 发现适配器写入 metadata 的字段名（snake_case）→ 展示标签。
// 各厂商字段并不完全一致，未登记的键按原文展示，避免静默丢字段。
const cloudMetadataLabels: Record<string, string> = {
  instance_type: 'Instance specification',
  instance_type_family: 'Specification family',
  cpu: 'vCPU',
  memory: 'Memory (MiB)',
  memory_gib: 'Memory (GiB)',
  os_type: 'System type',
  os_name: 'Operating System',
  host_name: 'hostname',
  hostname: 'hostname',
  image_id: 'Image ID',
  public_ip_address: 'Public IP',
  public_ip_addresses: 'Public IP',
  public_ip: 'Public IP',
  inner_ip_address: 'Intranet IP',
  private_ip_address: 'Intranet IP',
  private_ip_addresses: 'Intranet IP',
  private_ip: 'Intranet IP',
  primary_ip: 'Primary IP',
  eip_ip_address: 'Elastic IP',
  eip_bandwidth: 'Elastic IP Bandwidth (Mbps)',
  vpc_id: 'VPC',
  vswitch_id: 'switch',
  subnet_id: 'Subnet',
  security_groups: 'Security group',
  instance_charge_type: 'Billing method',
  charge_type: 'Billing method',
  expired_time: 'Expiration Date',
  start_time: 'Startup time',
  availability_zone: 'Availability Zone',
  architecture: 'Architecture',
  key_pair_name: 'Key pair',
  key_name: 'Key pair',
  serial_number: 'Serial Number',
  io_optimized: 'IO optimization',
  deletion_protection: 'Delete protection',
  spot_strategy: 'Preemption strategy',
  gpu_spec: 'GPU specifications',
  gpu_amount: 'Number of GPUs',
  instance_network_type: 'Network type',
  description: 'Description',
};

const cloudMetadataOrder = Object.keys(cloudMetadataLabels);

function formatMetadataValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return '-';
  if (Array.isArray(value)) {
    return value.length > 0 ? value.map(item => formatMetadataValue(item)).join('、') : '-';
  }
  if (typeof value === 'boolean') return value ? 'Yes' : 'No';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

function collectMetadataEntries(metadata?: Record<string, unknown>): Array<[string, unknown]> {
  if (!metadata) return [];
  const rank = (key: string) => {
    const index = cloudMetadataOrder.indexOf(key);
    return index === -1 ? cloudMetadataOrder.length : index;
  };
  return Object.entries(metadata)
    .filter(([, value]) => value !== null && value !== undefined && value !== '')
    .sort(([left], [right]) => rank(left) - rank(right));
}

interface AppliedFilters {
  provider?: string;
  serviceId?: number;
  region?: string;
}

export default function CloudResourcePage() {
  const router = useRouter();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();
  const [bindForm] = Form.useForm();

  // React Query：云资源（过滤 + 分页进 queryKey，条件变化自动重取）
  const [filters, setFilters] = useState<AppliedFilters>({});
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10 });

  const resourcesQuery = useCloudResourcesQuery({
    provider: filters.provider,
    serviceId: filters.serviceId,
    region: filters.region,
    page: pagination.current,
    pageSize: pagination.pageSize,
  });
  const servicesQuery = useCloudServicesQuery();

  const provider = Form.useWatch('provider', form);

  const serviceItems = servicesQuery.data?.items;
  const services = useMemo(() => serviceItems ?? [], [serviceItems]);
  const serviceMap = useMemo(() => {
    return new Map(services.map(service => [service.id, service]));
  }, [services]);

  // 后端信封只有一种形状：{items,total,page,pageSize,totalPages}
  const resources = resourcesQuery.data?.items ?? [];
  const total = resourcesQuery.data?.total ?? 0;

  const handleTableChange = (page: number, pageSize: number) => {
    setPagination({ current: page, pageSize });
  };

  // 查询：提交表单当前值（重置回第一页）
  const handleSearch = () => {
    const values = form.getFieldsValue();
    setFilters({
      provider: values.provider,
      serviceId: values.serviceId,
      region: values.region,
    });
    setPagination(p => ({ ...p, current: 1 }));
  };

  const [binding, setBinding] = useState<CloudResource | null>(null);
  const [bindCIOption, setBindCIOption] = useState<CISelectOption | undefined>(undefined);
  const [selectedRow, setSelectedRow] = useState<CloudResource | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const selectedMetadata = useMemo(
    () => collectMetadataEntries(selectedRow?.metadata),
    [selectedRow],
  );

  // 查看资源详情
  const handleViewDetail = (record: CloudResource) => {
    setSelectedRow(record);
    setDetailOpen(true);
  };

  // 绑定已有 CI：写回云资源信息后失效 CI / 云资源 / 对账缓存
  const bindMutation = useMutation({
    mutationFn: ({ ciId, data }: { ciId: string; data: Parameters<typeof CMDBApi.updateCI>[1] }) =>
      CMDBApi.updateCI(ciId, data),
    onSuccess: () => {
      message.success('Already bound to configuration item');
      setBinding(null);
      setBindCIOption(undefined);
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.cis() });
      queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'cloud-resources'] });
      queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'reconciliation'] });
    },
    onError: error => {
      if (error instanceof Error) {
        message.error(error.message || 'Binding failed');
      } else {
        message.error('Binding failed');
      }
    },
  });

  const handleBindExisting = async () => {
    if (!binding) return;
    try {
      const values = await bindForm.validateFields();
      const service = serviceMap.get(binding.serviceId);
      bindMutation.mutate({
        ciId: values.ciId,
        data: {
          cloudResourceRefId: binding.id,
          cloudProvider: service?.provider,
          cloudAccountId: String(binding.cloudAccountId),
          cloudRegion: binding.region,
          cloudZone: binding.zone,
          cloudResourceId: binding.resourceId,
          cloudResourceType: service?.resourceTypeCode,
          cloudMetadata: binding.metadata,
          cloudSyncStatus: 'success',
        },
      });
    } catch {
      // 表单校验失败，由 antd Form 自行提示
    }
  };

  const columns = [
    {
      title: 'Vendor',
      dataIndex: 'provider',
      width: 100,
      render: (value: string) => {
        const provider = providerOptions.find(p => p.value === value);
        return <Tag color='blue'>{provider?.label || value || '-'}</Tag>;
      },
    },
    {
      title: 'Service',
      width: 140,
      render: (_: unknown, record: CloudResource) => {
        const serviceId = record.serviceId;
        const service = serviceMap.get(serviceId);
        return service?.serviceName || '-';
      },
    },
    {
      title: 'Resource Type',
      width: 120,
      render: (_: unknown, record: CloudResource) => {
        const serviceId = record.serviceId;
        const service = serviceMap.get(serviceId);
        return service?.resourceTypeName || '-';
      },
    },
    {
      title: 'Resource ID',
      dataIndex: 'resourceId',
      width: 180,
      ellipsis: true,
    },
    {
      title: 'Resource name',
      dataIndex: 'resourceName',
      width: 160,
      ellipsis: true,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Region',
      dataIndex: 'region',
      width: 100,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Zone',
      dataIndex: 'zone',
      width: 100,
      render: (value?: string) => value || '-',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (value?: string) => (
        <Tag color={statusColors[value || ''] || 'default'}>{value || 'Unknown'}</Tag>
      ),
    },
    {
      title: 'Recently discovered',
      width: 150,
      render: (_: unknown, record: CloudResource) => {
        const value = record.lastSeenAt;
        return value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-';
      },
    },
    {
      title: 'Actions',
      key: 'action',
      width: 160,
      render: (_: unknown, record: CloudResource) => (
        <Space>
          <Tooltip title="View Details">
            <Button
              aria-label="View Details"
              type='text'
              icon={<Eye />}
              onClick={() => handleViewDetail(record)}
              size='small'
            />
          </Tooltip>
          <Button
            type='link'
            size='small'
            onClick={() => router.push(`/cmdb/cis/create?cloudResourceRefId=${record.id}`)}
          >

            Create new CI
          </Button>
          <Button
            type="link"
            size="small"
            onClick={() => {
              setBinding(record);
              setBindCIOption(undefined);
            }}
          >

            Binding
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <div className='mb-4'>
        <h1 className='text-2xl font-bold'>Cloud resource list</h1>
        <p className='text-gray-500 mt-1'>View the discovered cloud resources and create or bind the resources as CMDB configuration items.</p>
      </div>

      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: 'Home page' },
          { title: 'Configuration Management' },
          { title: <a onClick={() => router.push('/cmdb')}>CMDB</a> },
          { title: 'Cloud resource list' },
        ]}
      />

      {/* 搜索工具栏 */}
      <div className='mb-4 flex flex-wrap items-center gap-3'>
        <Form form={form} layout='inline' className='flex-wrap gap-2'>
          <Form.Item name='provider' className='!mb-0'>
            <Select placeholder="Cloud Provider" style={{ width: 140 }} allowClear options={providerOptions} />
          </Form.Item>
          <Form.Item name='serviceId' className='!mb-0'>
            <Select
              placeholder="Cloud services"
              style={{ width: 180 }}
              allowClear
              showSearch
              optionFilterProp='label'
              options={services
                .filter(service => !provider || service.provider === provider)
                .map(service => ({
                  value: service.id,
                  label: `${service.serviceName} (${service.resourceTypeName})`,
                }))}
            />
          </Form.Item>
          <Form.Item name='region' className='!mb-0'>
            <Input placeholder='Region' style={{ width: 120 }} allowClear />
          </Form.Item>
          <Form.Item className='!mb-0'>
            <Space>
              <Button type='primary' icon={<Search />} onClick={handleSearch}>

                Query
              </Button>
              <Button
                icon={<RotateCcw />}
                onClick={() => resourcesQuery.refetch()}
                loading={resourcesQuery.isFetching}
              >

                Refresh
              </Button>
            </Space>
          </Form.Item>
        </Form>
        <Space className='ml-auto'>
          <span className='text-sm text-gray-500'>Total {total}  resource</span>
          <Button icon={<ShieldAlert />} onClick={() => router.push('/cmdb/cloud-signals')}>

            Security and Alarms
          </Button>
        </Space>
      </div>

      <Table
        rowKey='id'
        loading={resourcesQuery.isLoading}
        dataSource={resources}
        columns={columns as unknown as React.ComponentProps<typeof Table>['columns']}
        pagination={{
          current: pagination.current,
          pageSize: pagination.pageSize,
          total: total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: total => `Total ${total} records`,
          pageSizeOptions: ['10', '20', '50', '100'],
          onChange: handleTableChange,
        }}
        scroll={{ x: 1200 }}
      />

      {/* 绑定已有配置项模态框 */}
      <Modal
        title="Bind existing configuration items"
        open={Boolean(binding)}
        onCancel={() => {
          setBinding(null);
          setBindCIOption(undefined);
        }}
        onOk={handleBindExisting}
        confirmLoading={bindMutation.isPending}
        okText="Binding"
        destroyOnHidden
        width={480}
      >
        <Form form={bindForm} layout='vertical'>
          <Form.Item
            name='ciId'
            label="Configuration item"
            rules={[{ required: true, message: 'Please select the configuration item to be bound' }]}
          >
            <CISearchSelect
              onChange={(value, option) => {
                bindForm.setFieldValue('ciId', value);
                setBindCIOption(option);
              }}
              style={{ width: '100%' }}
              placeholder="Enter name to search for configuration items"
            />
          </Form.Item>
          {binding && (
            <div className='p-3 bg-gray-50 rounded text-sm text-gray-600'>
              <div className='font-medium mb-1'>Resources will be bound:</div>
              <div>
                {binding.resourceName || binding.resourceId}
              </div>
              <div className='text-gray-400 mt-1'>
                {providerOptions.find(p => p.value === binding.provider)?.label} /{' '}
                {binding.region} / {binding.zone}
              </div>
            </div>
          )}
        </Form>
      </Modal>

      {/* 资源详情模态框 */}
      <Modal
        title="Cloud resource details"
        open={detailOpen}
        onCancel={() => {
          setDetailOpen(false);
          setSelectedRow(null);
        }}
        footer={[
          <Button key='close' onClick={() => setDetailOpen(false)}>

            Close
          </Button>,
          <Button
            key='signals'
            icon={<ShieldAlert />}
            onClick={() => {
              if (selectedRow?.resourceId) {
                router.push(
                  `/cmdb/cloud-signals?tab=vulnerabilities&instanceId=${encodeURIComponent(selectedRow.resourceId)}`,
                );
              }
            }}
          >

            Security and Alarms
          </Button>,
          <Button
            key='create'
            type='primary'
            icon={<Plus />}
            onClick={() => {
              if (selectedRow) {
                router.push(`/cmdb/cis/create?cloudResourceRefId=${selectedRow.id}`);
              }
            }}
          >

            Create new CI
          </Button>,
        ]}
        width={720}
      >
        {selectedRow && (
          <div className='space-y-3'>
            <div className='grid grid-cols-2 gap-3'>
              <div>
                <div className='text-sm text-gray-500'>Cloud Provider</div>
                <div>
                  {providerOptions.find(p => p.value === selectedRow.provider)?.label ||
                    selectedRow.provider ||
                    '-'}
                </div>
              </div>
              <div>
                <div className='text-sm text-gray-500'>Service Type</div>
                <div>{serviceMap.get(selectedRow.serviceId)?.serviceName || '-'}</div>
              </div>
              <div>
                <div className='text-sm text-gray-500'>Resource ID</div>
                <div className='font-mono text-sm'>{selectedRow.resourceId || '-'}</div>
              </div>
              <div>
                <div className='text-sm text-gray-500'>Resource name</div>
                <div>{selectedRow.resourceName || '-'}</div>
              </div>
              <div>
                <div className='text-sm text-gray-500'>Region</div>
                <div>{selectedRow.region || '-'}</div>
              </div>
              <div>
                <div className='text-sm text-gray-500'>Zone</div>
                <div>{selectedRow.zone || '-'}</div>
              </div>
              <div>
                <div className='text-sm text-gray-500'>Status</div>
                <Tag color={statusColors[selectedRow.status || ''] || 'default'}>
                  {cloudResourceStatusTextMap[selectedRow.status || ''] ||
                    selectedRow.status ||
                    'Unknown'}
                </Tag>
              </div>
              <div>
                <div className='text-sm text-gray-500'>Recently discovered</div>
                <div>
                  {selectedRow.lastSeenAt
                    ? dayjs(selectedRow.lastSeenAt).format('YYYY-MM-DD HH:mm:ss')
                    : '-'}
                </div>
              </div>
            </div>

            {selectedMetadata.length > 0 && (
              <div>
                <div className='text-sm text-gray-500 mb-2'>Instance details</div>
                <Descriptions bordered size='small' column={2}>
                  {selectedMetadata.map(([key, value]) => (
                    <Descriptions.Item key={key} label={cloudMetadataLabels[key] || key}>
                      {formatMetadataValue(value)}
                    </Descriptions.Item>
                  ))}
                </Descriptions>
              </div>
            )}

            {selectedRow.tags && Object.keys(selectedRow.tags).length > 0 && (
              <div>
                <div className='text-sm text-gray-500 mb-2'>Tags</div>
                <Space wrap>
                  {Object.entries(selectedRow.tags).map(([key, value]) => (
                    <Tag key={key}>
                      {key}
                      {value ? `=${value}` : ''}
                    </Tag>
                  ))}
                </Space>
              </div>
            )}
          </div>
        )}
      </Modal>
    </Card>
  );
}
