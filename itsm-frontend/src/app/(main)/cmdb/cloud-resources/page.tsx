'use client';

import React, { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Breadcrumb,
  Button,
  Card,
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
import { Search, Plus, Eye, RotateCcw } from 'lucide-react';
import dayjs from 'dayjs';

import { CMDBApi } from '@/lib/api/cmdb-api';
import CISearchSelect, { type CISelectOption } from '@/components/cmdb/CISearchSelect';
import { CMDB_KEYS, useCloudResourcesQuery, useCloudServicesQuery } from '@/lib/hooks/useCMDB';
import type { CloudResource } from '@/types/biz/cmdb';

const providerOptions = [
  { value: 'aliyun', label: '阿里云' },
  { value: 'huawei', label: '华为云' },
  { value: 'tencent', label: '腾讯云' },
  { value: 'azure', label: 'Azure' },
  { value: 'aws', label: 'AWS' },
  { value: 'onprem', label: '私有云' },
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
  running: '运行中',
  stopped: '已停止',
  active: '活跃',
  inactive: '未激活',
  available: '可用',
  unavailable: '不可用',
  pending: '处理中',
  failed: '失败',
};

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
    offset: (pagination.current - 1) * pagination.pageSize,
    limit: pagination.pageSize,
  });
  const servicesQuery = useCloudServicesQuery();

  const provider = Form.useWatch('provider', form);

  const services = servicesQuery.data ?? [];
  const serviceMap = useMemo(() => {
    return new Map(services.map(service => [service.id, service]));
  }, [services]);

  // API 返回兼容处理：可能是数组或 { items, total } 格式
  type ApiListResponse =
    | CloudResource[]
    | { items?: CloudResource[]; data?: CloudResource[]; total?: number };
  const response = resourcesQuery.data as ApiListResponse | undefined;
  const resources = useMemo(
    () =>
      Array.isArray(response) ? response : response?.items || response?.data || [],
    [response]
  );
  const total = Array.isArray(response)
    ? response.length
    : response?.total ?? resources.length;

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
      message.success('已绑定到配置项');
      setBinding(null);
      setBindCIOption(undefined);
      queryClient.invalidateQueries({ queryKey: CMDB_KEYS.cis() });
      queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'cloud-resources'] });
      queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'reconciliation'] });
    },
    onError: error => {
      if (error instanceof Error) {
        message.error(error.message || '绑定失败');
      } else {
        message.error('绑定失败');
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
      title: '厂商',
      dataIndex: 'provider',
      width: 100,
      render: (value: string) => {
        const provider = providerOptions.find(p => p.value === value);
        return <Tag color='blue'>{provider?.label || value || '-'}</Tag>;
      },
    },
    {
      title: '服务',
      width: 140,
      render: (_: unknown, record: CloudResource) => {
        const serviceId = record.serviceId;
        const service = serviceMap.get(serviceId);
        return service?.serviceName || '-';
      },
    },
    {
      title: '资源类型',
      width: 120,
      render: (_: unknown, record: CloudResource) => {
        const serviceId = record.serviceId;
        const service = serviceMap.get(serviceId);
        return service?.resourceTypeName || '-';
      },
    },
    {
      title: '资源ID',
      dataIndex: 'resourceId',
      width: 180,
      ellipsis: true,
    },
    {
      title: '资源名称',
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
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (value?: string) => (
        <Tag color={statusColors[value || ''] || 'default'}>{value || '未知'}</Tag>
      ),
    },
    {
      title: '最近发现',
      width: 150,
      render: (_: unknown, record: CloudResource) => {
        const value = record.lastSeenAt;
        return value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-';
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: CloudResource) => (
        <Space>
          <Tooltip title='查看详情'>
            <Button
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
            新建CI
          </Button>
          <Button
            type="link"
            size="small"
            onClick={() => {
              setBinding(record);
              setBindCIOption(undefined);
            }}
          >
            绑定
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <div className='mb-4'>
        <h1 className='text-2xl font-bold'>云资源列表</h1>
        <p className='text-gray-500 mt-1'>查看已发现的云资源，并将资源新建或绑定为 CMDB 配置项。</p>
      </div>

      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: '首页' },
          { title: '配置管理' },
          { title: <a onClick={() => router.push('/cmdb')}>CMDB</a> },
          { title: '云资源列表' },
        ]}
      />

      {/* 搜索工具栏 */}
      <div className='mb-4 flex flex-wrap items-center gap-3'>
        <Form form={form} layout='inline' className='flex-wrap gap-2'>
          <Form.Item name='provider' className='!mb-0'>
            <Select placeholder='云厂商' style={{ width: 140 }} allowClear options={providerOptions} />
          </Form.Item>
          <Form.Item name='serviceId' className='!mb-0'>
            <Select
              placeholder='云服务'
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
                查询
              </Button>
              <Button
                icon={<RotateCcw />}
                onClick={() => resourcesQuery.refetch()}
                loading={resourcesQuery.isFetching}
              >
                刷新
              </Button>
            </Space>
          </Form.Item>
        </Form>
        <span className='ml-auto text-sm text-gray-500'>共 {total} 条资源</span>
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
          showTotal: total => `共 ${total} 条记录`,
          pageSizeOptions: ['10', '20', '50', '100'],
          onChange: handleTableChange,
        }}
        scroll={{ x: 1200 }}
      />

      {/* 绑定已有配置项模态框 */}
      <Modal
        title='绑定已有配置项'
        open={Boolean(binding)}
        onCancel={() => {
          setBinding(null);
          setBindCIOption(undefined);
        }}
        onOk={handleBindExisting}
        confirmLoading={bindMutation.isPending}
        okText='绑定'
        destroyOnClose
        width={480}
      >
        <Form form={bindForm} layout='vertical'>
          <Form.Item
            name='ciId'
            label='配置项'
            rules={[{ required: true, message: '请选择要绑定的配置项' }]}
          >
            <CISearchSelect
              onChange={(value, option) => {
                bindForm.setFieldValue('ciId', value);
                setBindCIOption(option);
              }}
              style={{ width: '100%' }}
              placeholder='输入名称搜索配置项'
            />
          </Form.Item>
          {binding && (
            <div className='p-3 bg-gray-50 rounded text-sm text-gray-600'>
              <div className='font-medium mb-1'>将绑定资源：</div>
              <div>
                {binding.resourceName || binding.resourceId}
              </div>
              <div className='text-gray-400 mt-1'>
                {providerOptions.find(p => p.value === (binding as any).provider)?.label} /{' '}
                {binding.region} / {binding.zone}
              </div>
            </div>
          )}
        </Form>
      </Modal>

      {/* 资源详情模态框 */}
      <Modal
        title='云资源详情'
        open={detailOpen}
        onCancel={() => {
          setDetailOpen(false);
          setSelectedRow(null);
        }}
        footer={[
          <Button key='close' onClick={() => setDetailOpen(false)}>
            关闭
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
            新建CI
          </Button>,
        ]}
        width={560}
      >
        {selectedRow && (
          <div className='space-y-3'>
            <div className='grid grid-cols-2 gap-3'>
              <div>
                <div className='text-sm text-gray-500'>云厂商</div>
                <div>
                  {providerOptions.find(p => p.value === (selectedRow as any).provider)?.label ||
                    (selectedRow as any).provider ||
                    '-'}
                </div>
              </div>
              <div>
                <div className='text-sm text-gray-500'>服务类型</div>
                <div>{serviceMap.get(selectedRow.serviceId)?.serviceName || '-'}</div>
              </div>
              <div>
                <div className='text-sm text-gray-500'>资源ID</div>
                <div className='font-mono text-sm'>{selectedRow.resourceId || '-'}</div>
              </div>
              <div>
                <div className='text-sm text-gray-500'>资源名称</div>
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
                <div className='text-sm text-gray-500'>状态</div>
                <Tag color={statusColors[selectedRow.status || ''] || 'default'}>
                  {cloudResourceStatusTextMap[selectedRow.status || ''] ||
                    selectedRow.status ||
                    '未知'}
                </Tag>
              </div>
              <div>
                <div className='text-sm text-gray-500'>最近发现</div>
                <div>
                  {selectedRow.lastSeenAt
                    ? dayjs(selectedRow.lastSeenAt).format('YYYY-MM-DD HH:mm:ss')
                    : '-'}
                </div>
              </div>
            </div>
          </div>
        )}
      </Modal>
    </Card>
  );
}
