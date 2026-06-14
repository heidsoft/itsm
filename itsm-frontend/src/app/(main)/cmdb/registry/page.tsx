'use client';

import React from 'react';
import { App, Badge, Button, Card, Col, Row, Space, Table, Tabs, Tag, Typography } from 'antd';
import { Cloud, Database, RefreshCw, Server, Sparkles } from 'lucide-react';

import { CMDBApi } from '@/lib/api/cmdb-api';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview, type StatsOverviewItem } from '@/components/ui/StatsOverview';

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
  const [loading, setLoading] = React.useState(true);
  const [discoverySources, setDiscoverySources] = React.useState<any[]>([]);
  const [cloudServices, setCloudServices] = React.useState<any[]>([]);
  const [cloudAccounts, setCloudAccounts] = React.useState<any[]>([]);
  const [discoveryHistory, setDiscoveryHistory] = React.useState<any[]>([]);
  const [refreshAt, setRefreshAt] = React.useState<string | null>(null);

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const [sources, services, accounts, history] = await Promise.all([
        CMDBApi.getDiscoverySources(),
        CMDBApi.getCloudServices(),
        CMDBApi.getCloudAccounts(),
        CMDBApi.getDiscoveryHistory(),
      ]);

      setDiscoverySources(normalizeList(sources));
      setCloudServices(normalizeList(services));
      setCloudAccounts(normalizeList(accounts));
      setDiscoveryHistory(normalizeList(history));
      setRefreshAt(new Date().toISOString());
    } catch (error) {
      message.error('加载图谱注册中心失败');
    } finally {
      setLoading(false);
    }
  }, [message]);

  React.useEffect(() => {
    load();
  }, [load]);

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
      dataIndex: 'source_type',
      key: 'source_type',
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
      dataIndex: 'is_active',
      key: 'is_active',
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
      dataIndex: 'service_code',
      key: 'service_code',
    },
    {
      title: '服务名称',
      dataIndex: 'service_name',
      key: 'service_name',
    },
    {
      title: '资源类型',
      dataIndex: 'resource_type_name',
      key: 'resource_type_name',
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
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
      dataIndex: 'account_name',
      key: 'account_name',
    },
    {
      title: '账号ID',
      dataIndex: 'account_id',
      key: 'account_id',
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
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
      dataIndex: 'source_id',
      key: 'source_id',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (value: string) => <Tag color={value === 'success' ? 'green' : value === 'failed' ? 'red' : 'blue'}>{value || '-'}</Tag>,
    },
    {
      title: '开始时间',
      dataIndex: 'started_at',
      key: 'started_at',
      render: (value?: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-'),
    },
    {
      title: '结束时间',
      dataIndex: 'finished_at',
      key: 'finished_at',
      render: (value?: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-'),
    },
  ];

  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="Service Graph Registry"
        description="把发现源、云服务、云账号和发现历史放到同一张接入注册表里，为 Service Graph 提供统一入口。"
        actions={
          <Space wrap>
            <Button icon={<RefreshCw className="h-4 w-4" />} loading={loading} onClick={load}>
              刷新
            </Button>
            <Button type="primary" href="/cmdb/relationships" icon={<Database className="h-4 w-4" />}>
              去关系图谱
            </Button>
          </Space>
        }
      />

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
