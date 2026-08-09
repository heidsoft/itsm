'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { App, Button, Card, Col, Row, Space, Tag, Typography } from 'antd';
import { Database, GitBranch, Plus, RefreshCw, Server, ShieldCheck } from 'lucide-react';

import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview, type StatsOverviewItem } from '@/components/ui/StatsOverview';
import { CMDBApi } from '@/lib/api/cmdb-api';
import { CMDB_GA_CAPABILITIES } from '@/lib/cmdb/cmdb-capabilities';

const { Paragraph, Text, Title } = Typography;

type CMDBOverview = {
  totalCIs: number;
  ciTypes: number;
  loading: boolean;
  error: boolean;
  refreshedAt: string | null;
};

const normalizeItems = (value: unknown): unknown[] => {
  if (Array.isArray(value)) return value;
  if (value && typeof value === 'object' && Array.isArray((value as { items?: unknown[] }).items)) {
    return (value as { items: unknown[] }).items;
  }
  return [];
};

export function CSDMHub() {
  const router = useRouter();
  const { message } = App.useApp();
  const [overview, setOverview] = React.useState<CMDBOverview>({
    totalCIs: 0,
    ciTypes: 0,
    loading: true,
    error: false,
    refreshedAt: null,
  });

  const load = React.useCallback(async () => {
    setOverview(previous => ({ ...previous, loading: true, error: false }));
    try {
      const [statsResponse, typesResponse] = await Promise.all([
        CMDBApi.getCMDBStats(),
        CMDBApi.getCITypes(),
      ]);
      const stats = (statsResponse as { data?: Record<string, unknown> })?.data ?? statsResponse;
      const statsRecord = (stats ?? {}) as Record<string, unknown>;

      setOverview({
        totalCIs: Number(statsRecord.totalCount ?? statsRecord.totalCIs ?? 0),
        ciTypes: normalizeItems(typesResponse).length,
        loading: false,
        error: false,
        refreshedAt: new Date().toISOString(),
      });
    } catch {
      setOverview(previous => ({ ...previous, loading: false, error: true }));
      message.error('CMDB 总览加载失败，请检查服务状态后重试');
    }
  }, [message]);

  React.useEffect(() => {
    void load();
  }, [load]);

  const statsItems: StatsOverviewItem[] = [
    {
      key: 'total-cis',
      title: '配置项总数',
      value: overview.totalCIs,
      prefix: <Database className="mr-2 text-blue-500" />,
      accentColor: '#1677ff',
    },
    {
      key: 'ci-types',
      title: 'CI 类型',
      value: overview.ciTypes,
      prefix: <Server className="mr-2 text-cyan-600" />,
      accentColor: '#08979c',
    },
    {
      key: 'ga-capabilities',
      title: '生产可用能力',
      value: CMDB_GA_CAPABILITIES.length,
      prefix: <ShieldCheck className="mr-2 text-green-600" />,
      accentColor: '#389e0d',
    },
  ];

  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="配置管理数据库 (CMDB)"
        description="面向生产运维的配置项、关系、拓扑和影响分析工作台。"
        actions={
          <Space wrap>
            <Button icon={<RefreshCw className="h-4 w-4" />} loading={overview.loading} onClick={load}>
              刷新
            </Button>
            <Button type="primary" icon={<Plus className="h-4 w-4" />} onClick={() => router.push('/cmdb/cis/create')}>
              录入配置项
            </Button>
          </Space>
        }
      />

      {overview.error && (
        <Card className="border-red-200 bg-red-50">
          <Text type="danger">总览数据加载失败。当前数值不是可信业务数据，请刷新或联系管理员。</Text>
        </Card>
      )}

      <StatsOverview items={statsItems} />

      <Card title="生产可用能力" loading={overview.loading}>
        <Row gutter={[16, 16]}>
          {CMDB_GA_CAPABILITIES.map(capability => (
            <Col key={capability.key} xs={24} md={12} xl={6}>
              <Card size="small" className="h-full border-slate-200">
                <div className="flex items-center justify-between gap-2">
                  <Title level={5} className="!mb-0">{capability.title}</Title>
                  <Tag color="green">GA</Tag>
                </div>
                <Paragraph type="secondary" className="!mb-4 !mt-3 min-h-11">
                  {capability.description}
                </Paragraph>
                <Button type="link" className="!px-0" onClick={() => router.push(capability.href)}>
                  进入
                </Button>
              </Card>
            </Col>
          ))}
        </Row>
      </Card>

      <Card title="推荐操作">
        <Space wrap size="middle">
          <Button type="primary" icon={<Server className="h-4 w-4" />} onClick={() => router.push('/cmdb/ci')}>
            配置项工作台
          </Button>
          <Button icon={<Database className="h-4 w-4" />} onClick={() => router.push('/admin/cmdb-types')}>
            维护类型模板
          </Button>
          <Button icon={<GitBranch className="h-4 w-4" />} onClick={() => router.push('/cmdb/relationships')}>
            维护关系
          </Button>
          <Button icon={<GitBranch className="h-4 w-4" />} onClick={() => router.push('/cmdb/topology')}>
            查看拓扑影响
          </Button>
        </Space>
        <Text type="secondary" className="mt-4 block">
          云资源自动发现和自动对账仍处于受控试点，完成真实连接、任务重试、审计和生产验收前不在正式入口展示。
        </Text>
        {overview.refreshedAt && (
          <Text type="secondary" className="mt-2 block text-xs">
            数据刷新时间：{new Date(overview.refreshedAt).toLocaleString('zh-CN')}
          </Text>
        )}
      </Card>
    </div>
  );
}
