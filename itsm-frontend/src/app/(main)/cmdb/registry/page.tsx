'use client';

import React from 'react';
import { Alert, App, Badge, Button, Card, Col, Row, Space, Table, Tabs, Tag, Typography } from 'antd';
import { Cloud, Database, RefreshCw, Server, Sparkles } from 'lucide-react';

import type { CMDBRuntimeCapability } from '@/lib/api/cmdb-api';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview, type StatsOverviewItem } from '@/components/ui/StatsOverview';
import {
  useCapabilitiesQuery,
  useCloudAccountsQuery,
  useCloudServicesQuery,
  useDiscoveryHistoryQuery,
  useDiscoverySourcesQuery,
} from '@/lib/hooks/useCMDB';

const { Text } = Typography;

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

  // React Query：5 路数据并行加载（替代手写 Promise.all + setState）
  const sourcesQuery = useDiscoverySourcesQuery();
  const servicesQuery = useCloudServicesQuery();
  const accountsQuery = useCloudAccountsQuery();
  const historyQuery = useDiscoveryHistoryQuery();
  const capabilitiesQuery = useCapabilitiesQuery();

  const queries = [sourcesQuery, servicesQuery, accountsQuery, historyQuery, capabilitiesQuery];
  const loading = queries.some(q => q.isLoading);
  const fetching = queries.some(q => q.isFetching);
  const hasError = queries.some(q => q.isError);

  React.useEffect(() => {
    if (hasError) {
      message.error('加载图谱注册中心失败');
    }
  }, [hasError, message]);

  const handleRefresh = () => {
    queries.forEach(q => q.refetch());
  };

  const discoverySources = normalizeList<Record<string, unknown>>(sourcesQuery.data);
  const cloudServices = normalizeList<Record<string, unknown>>(servicesQuery.data);
  const cloudAccounts = normalizeList<Record<string, unknown>>(accountsQuery.data);
  const discoveryHistory = normalizeList<Record<string, unknown>>(historyQuery.data);

  const discoveryCapability: CMDBRuntimeCapability | null =
    capabilitiesQuery.data?.items.find(item => item.key === 'cmdbDiscovery') ?? null;

  // 最后刷新时间取数据最近一次成功落地的时刻
  const refreshAt = queries.some(q => q.dataUpdatedAt > 0)
    ? new Date(Math.max(...queries.map(q => q.dataUpdatedAt))).toISOString()
    : null;

  const statsItems: StatsOverviewItem[] = [
    {
      key: 'sources',
      title: '发现源',
      value: discoverySources.length,
      prefix: <Sparkles className="text-cyan-500 mr-2" />,
      accentColor: '#13c2c2',
    },
    {
      key: 'services',
      title: '云服务',
      value: cloudServices.length,
      prefix: <Database className="text-blue-500 mr-2" />,
      accentColor: '#1890ff',
    },
    {
      key: 'accounts',
      title: '云账号',
      value: cloudAccounts.length,
      prefix: <Cloud className="text-green-500 mr-2" />,
      accentColor: '#52c41a',
    },
    {
      key: 'jobs',
      title: '发现记录',
      value: discoveryHistory.length,
      prefix: <Server className="text-orange-500 mr-2" />,
      accentColor: '#fa8c16',
    },
  ];

  const sourceColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '类型',
      dataIndex: 'sourceType',
      key: 'sourceType',
      render: (value: string) => <Tag color="blue">{value || '-'}</Tag>,
    },
    {
      title: '提供方',
      dataIndex: 'provider',
      key: 'provider',
      render: (value?: string) => value || '-',
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (value: boolean) => <Badge status={value ? 'success' : 'default'} text={value ? '启用' : '停用'} />,
    },
    {
      title: '说明',
      dataIndex: 'description',
      key: 'description',
      render: (value?: string) => value || '-',
    },
  ];

  const serviceColumns = [
    {
      title: '厂商',
      dataIndex: 'provider',
      key: 'provider',
    },
    {
      title: '服务代码',
      dataIndex: 'serviceCode',
      key: 'serviceCode',
    },
    {
      title: '服务名称',
      dataIndex: 'serviceName',
      key: 'serviceName',
    },
    {
      title: '资源类型',
      dataIndex: 'resourceTypeName',
      key: 'resourceTypeName',
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (value: boolean) => <Tag color={value ? 'green' : 'default'}>{value ? '启用' : '停用'}</Tag>,
    },
  ];

  const accountColumns = [
    {
      title: '厂商',
      dataIndex: 'provider',
      key: 'provider',
    },
    {
      title: '账号名称',
      dataIndex: 'accountName',
      key: 'accountName',
    },
    {
      title: '账号ID',
      dataIndex: 'accountId',
      key: 'accountId',
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (value: boolean) => <Tag color={value ? 'green' : 'default'}>{value ? '启用' : '停用'}</Tag>,
    },
  ];

  const historyColumns = [
    {
      title: '任务ID',
      dataIndex: 'id',
      key: 'id',
    },
    {
      title: '源ID',
      dataIndex: 'sourceId',
      key: 'sourceId',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (value: string) => <Tag color={value === 'success' ? 'green' : value === 'failed' ? 'red' : 'blue'}>{value || '-'}</Tag>,
    },
    {
      title: '开始时间',
      dataIndex: 'startedAt',
      key: 'startedAt',
      render: (value?: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-'),
    },
    {
      title: '结束时间',
      dataIndex: 'finishedAt',
      key: 'finishedAt',
      render: (value?: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-'),
    },
  ];

  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="Service Graph Registry"
        description="集中管理发现源、云服务和云账号的接入信息及历史记录。"
        actions={
          <Space wrap>
            <Button icon={<RefreshCw className="h-4 w-4" />} loading={fetching} onClick={handleRefresh}>
              刷新
            </Button>
            <Button type="primary" href="/cmdb/relationships" icon={<Database className="h-4 w-4" />}>
              去关系图谱
            </Button>
          </Space>
        }
      />

      {discoveryCapability?.state !== 'ready' ? (
        <Alert
          type="warning"
          showIcon
          message="云资源自动发现尚未就绪"
          description={
            discoveryCapability
              ? `当前状态：${discoveryCapability.state}；缺失条件：${discoveryCapability.missingRequirements.join('、') || '租户配置'}`
              : '无法读取后端能力状态；后端将按未就绪状态拒绝自动发现请求。'
          }
        />
      ) : null}

      <StatsOverview items={statsItems} />

      <Card loading={loading} className="shadow-sm">
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Card size="small" className="h-full bg-slate-50">
              <Text type="secondary">最后刷新</Text>
              <div className="mt-2 text-lg font-semibold">
                {refreshAt ? new Date(refreshAt).toLocaleString('zh-CN') : '-'}
              </div>
            </Card>
          </Col>
          <Col xs={24} md={12}>
            <Card size="small" className="h-full bg-slate-50">
              <Text type="secondary">注册中心说明</Text>
              <div className="mt-2">
                先定义发现源和云服务，再把云账号与 CI、关系和对账串起来，才能形成可持续的图谱治理闭环。
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
              label: '发现源',
              children: <Table rowKey="id" columns={sourceColumns} dataSource={discoverySources} pagination={{ pageSize: 8 }} />,
            },
            {
              key: 'services',
              label: '云服务',
              children: <Table rowKey="id" columns={serviceColumns} dataSource={cloudServices} pagination={{ pageSize: 8 }} />,
            },
            {
              key: 'accounts',
              label: '云账号',
              children: <Table rowKey="id" columns={accountColumns} dataSource={cloudAccounts} pagination={{ pageSize: 8 }} />,
            },
            {
              key: 'history',
              label: '发现历史',
              children: <Table rowKey="id" columns={historyColumns} dataSource={discoveryHistory} pagination={{ pageSize: 8 }} />,
            },
          ]}
        />
      </Card>
    </div>
  );
}
