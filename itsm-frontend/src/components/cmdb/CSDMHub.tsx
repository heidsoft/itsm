'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { App, Button, Card, Col, Divider, Row, Space, Tag, Typography } from 'antd';
import {
  ArrowRight,
  Cloud,
  Database,
  GitBranch,
  Layers3,
  RefreshCw,
  Shield,
  Server,
  SlidersHorizontal,
  Sparkles,
  Workflow,
} from 'lucide-react';

import { CMDBApi } from '@/lib/api/cmdb-api';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview, type StatsOverviewItem } from '@/components/ui/StatsOverview';

const { Text, Title } = Typography;

type CSDMCounts = {
  totalCIs: number;
  ciTypes: number;
  cloudAccounts: number;
  cloudServices: number;
  discoverySources: number;
  cloudResources: number;
  boundResources: number;
  unboundResources: number;
  orphanCIs: number;
  unlinkedCIs: number;
};

type CSDMState = {
  counts: CSDMCounts;
  lastSyncedAt: string | null;
  loading: boolean;
};

type HubCardProps = {
  title: string;
  description: string;
  accent: string;
  icon: React.ReactNode;
  metrics: Array<{ label: string; value: string | number; color?: string }>;
  actions: Array<{ label: string; href: string; icon?: React.ReactNode }>;
};

const normalizeList = <T,>(response: unknown): T[] => {
  if (Array.isArray(response)) return response as T[];
  if (response && typeof response === 'object') {
    const record = response as Record<string, unknown>;
    if (Array.isArray(record.data)) return record.data as T[];
    if (Array.isArray(record.items)) return record.items as T[];
  }
  return [];
};

function HubCard({ title, description, accent, icon, metrics, actions }: HubCardProps) {
  return (
    <Card
      className="h-full border-slate-200 shadow-sm"
      styles={{ body: { height: '100%' } }}
      style={{ borderTop: `3px solid ${accent}` }}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="mb-2 flex items-center gap-2">
            <div
              className="flex h-10 w-10 items-center justify-center rounded-xl text-white"
              style={{ background: accent }}
            >
              {icon}
            </div>
            <Title level={4} className="!mb-0">
              {title}
            </Title>
          </div>
          <Text type="secondary">{description}</Text>
        </div>
        <Tag color="blue">CSDM</Tag>
      </div>

      <div className="mt-5 grid gap-3 sm:grid-cols-3">
        {metrics.map(metric => (
          <div key={metric.label} className="rounded-lg bg-slate-50 px-3 py-2">
            <div className="text-xs text-slate-500">{metric.label}</div>
            <div className="mt-1 font-semibold text-slate-900">{metric.value}</div>
          </div>
        ))}
      </div>

      <Divider className="!my-4" />

      <div className="flex flex-wrap gap-2">
        {actions.map(action => (
          <Button key={action.href} href={action.href} icon={action.icon} type="default">
            {action.label}
          </Button>
        ))}
      </div>
    </Card>
  );
}

export function CSDMHub() {
  const router = useRouter();
  const { message } = App.useApp();
  const [state, setState] = React.useState<CSDMState>({
    counts: {
      totalCIs: 0,
      ciTypes: 0,
      cloudAccounts: 0,
      cloudServices: 0,
      discoverySources: 0,
      cloudResources: 0,
      boundResources: 0,
      unboundResources: 0,
      orphanCIs: 0,
      unlinkedCIs: 0,
    },
    lastSyncedAt: null,
    loading: true,
  });

  const load = React.useCallback(async () => {
    setState(prev => ({ ...prev, loading: true }));
    try {
      const [
        stats,
        ciTypes,
        cloudAccounts,
        cloudServices,
        discoverySources,
        cloudResources,
        reconciliation,
      ] = await Promise.all([
        CMDBApi.getCMDBStats(),
        CMDBApi.getCITypes(),
        CMDBApi.getCloudAccounts(),
        CMDBApi.getCloudServices(),
        CMDBApi.getDiscoverySources(),
        CMDBApi.getCloudResources(),
        CMDBApi.getReconciliationResults(),
      ]);

      const statsData = (stats as any)?.data ?? stats;
      const reconData = (reconciliation as any)?.data ?? reconciliation;
      const summary = (reconData?.summary ?? {}) as Record<string, unknown>;
      const unboundResources = normalizeList<Record<string, unknown>>(reconData?.unboundResources ?? reconData?.unbound_resources);
      const orphanCIs = normalizeList<Record<string, unknown>>(reconData?.orphanCIs ?? reconData?.orphan_cis);
      const unlinkedCIs = normalizeList<Record<string, unknown>>(reconData?.unlinkedCIs ?? reconData?.unlinked_cis);

      setState({
        loading: false,
        lastSyncedAt: new Date().toISOString(),
        counts: {
          totalCIs:
            statsData?.totalCount ?? statsData?.total_count ?? statsData?.totalCIs ?? statsData?.total_cis ?? 0,
          ciTypes: Array.isArray(ciTypes) ? ciTypes.length : normalizeList(ciTypes).length,
          cloudAccounts: normalizeList(cloudAccounts).length,
          cloudServices: normalizeList(cloudServices).length,
          discoverySources: normalizeList(discoverySources).length,
          cloudResources: normalizeList(cloudResources).length,
          boundResources: Number(summary.boundResourceCount ?? summary.bound_resource_count ?? 0),
          unboundResources: unboundResources.length,
          orphanCIs: orphanCIs.length,
          unlinkedCIs: unlinkedCIs.length,
        },
      });
    } catch (error) {
      setState(prev => ({ ...prev, loading: false }));
      message.error('加载 CSDM 总览失败');
    }
  }, [message]);

  React.useEffect(() => {
    load();
  }, [load]);

  const statsItems: StatsOverviewItem[] = [
    {
      key: 'total-cis',
      title: '配置项总数',
      value: state.counts.totalCIs,
      prefix: <Database className="text-blue-500 mr-2" />,
      accentColor: '#1890ff',
    },
    {
      key: 'connectors',
      title: '连接器',
      value: state.counts.cloudServices,
      prefix: <Cloud className="text-cyan-500 mr-2" />,
      accentColor: '#13c2c2',
    },
    {
      key: 'graph',
      title: '图谱绑定',
      value: state.counts.boundResources,
      prefix: <GitBranch className="text-green-500 mr-2" />,
      accentColor: '#52c41a',
    },
    {
      key: 'quality',
      title: '待治理项',
      value: state.counts.unboundResources + state.counts.orphanCIs + state.counts.unlinkedCIs,
      prefix: <Shield className="text-orange-500 mr-2" />,
      accentColor: '#fa8c16',
    },
  ];

  const hubCards: HubCardProps[] = [
    {
      title: 'Foundation Data',
      description: '先建基础数据，再让服务、关系和发现自动落到统一模型里。',
      accent: '#1890ff',
      icon: <Database className="h-5 w-5" />,
      metrics: [
        { label: 'CI 类型', value: state.counts.ciTypes, color: 'blue' },
        { label: '云账号', value: state.counts.cloudAccounts, color: 'geekblue' },
        { label: '发现源', value: state.counts.discoverySources, color: 'cyan' },
      ],
      actions: [
        { label: 'CI 类型管理', href: '/admin/cmdb-types', icon: <Database className="h-4 w-4" /> },
        { label: '云服务目录', href: '/cmdb/cloud-services', icon: <Cloud className="h-4 w-4" /> },
        { label: '云账号管理', href: '/cmdb/cloud-accounts', icon: <Shield className="h-4 w-4" /> },
      ],
    },
    {
      title: 'Service Model',
      description: '把配置项提升为业务服务、应用服务和技术服务的可管理对象。',
      accent: '#722ed1',
      icon: <Layers3 className="h-5 w-5" />,
      metrics: [
        { label: '配置项', value: state.counts.totalCIs, color: 'purple' },
        { label: '服务目录', value: state.counts.cloudServices, color: 'magenta' },
        { label: '关系层', value: '已拆分', color: 'gold' },
      ],
      actions: [
        { label: '配置项工作台', href: '/cmdb/ci', icon: <Server className="h-4 w-4" /> },
        { label: '关系管理', href: '/cmdb/relationships', icon: <GitBranch className="h-4 w-4" /> },
        { label: '服务目录', href: '/service-catalog', icon: <Workflow className="h-4 w-4" /> },
      ],
    },
    {
      title: 'Service Graph',
      description: '从发现、绑定、关联到影响分析，形成可追溯的服务图谱。',
      accent: '#13c2c2',
      icon: <GitBranch className="h-5 w-5" />,
      metrics: [
        { label: '云资源', value: state.counts.cloudResources, color: 'cyan' },
        { label: '绑定数', value: state.counts.boundResources, color: 'green' },
        { label: '拓扑', value: '可视化', color: 'gold' },
      ],
      actions: [
        { label: '图谱注册中心', href: '/cmdb/registry', icon: <Sparkles className="h-4 w-4" /> },
        { label: '拓扑视图', href: '/cmdb/topology', icon: <GitBranch className="h-4 w-4" /> },
        { label: '云资源列表', href: '/cmdb/cloud-resources', icon: <Cloud className="h-4 w-4" /> },
        { label: '对账中心', href: '/cmdb/reconciliation', icon: <SlidersHorizontal className="h-4 w-4" /> },
      ],
    },
  ];

  const pipeline = [
    {
      title: '连接器接入',
      description: '把外部云、发现源和导入源先纳入统一入口。',
      action: '/cmdb/cloud-services',
      icon: <Cloud className="h-5 w-5" />,
    },
    {
      title: '标准化建模',
      description: '用 CI 类型和属性模式描述对象，而不是堆字段。',
      action: '/admin/cmdb-types',
      icon: <Database className="h-5 w-5" />,
    },
    {
      title: '服务图谱构建',
      description: '通过关系、拓扑和影响分析把 CI 组成服务图谱。',
      action: '/cmdb/topology',
      icon: <GitBranch className="h-5 w-5" />,
    },
    {
      title: '质量治理闭环',
      description: '对账、绑定和孤儿治理，持续压低脏数据。',
      action: '/cmdb/reconciliation',
      icon: <Shield className="h-5 w-5" />,
    },
  ];

  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="CSDM / Service Graph CMDB"
        description="不是资产表，而是围绕 Foundation Data、Service Model、Service Graph 和 Quality Governance 的控制台。"
        actions={
          <Space wrap>
            <Button icon={<RefreshCw className="h-4 w-4" />} loading={state.loading} onClick={load}>
              刷新总览
            </Button>
            <Button type="primary" icon={<Sparkles className="h-4 w-4" />} onClick={() => router.push('/cmdb/ci')}>
              进入配置项工作台
            </Button>
          </Space>
        }
      />

      <StatsOverview items={statsItems} />

      <Card
        title={
          <span className="flex items-center gap-2">
            <Workflow className="h-4 w-4" />
            CSDM 分层
          </span>
        }
        loading={state.loading}
      >
        <div className="grid gap-4 lg:grid-cols-3">
          {hubCards.map(card => (
            <HubCard
              key={card.title}
              title={card.title}
              description={card.description}
              accent={card.accent}
              icon={card.icon}
              metrics={card.metrics}
              actions={card.actions}
            />
          ))}
        </div>
      </Card>

      <Card
        title={
          <span className="flex items-center gap-2">
            <ArrowRight className="h-4 w-4" />
            Service Graph 流程
          </span>
        }
      >
        <Row gutter={[16, 16]}>
          {pipeline.map((step, index) => (
            <Col key={step.title} xs={24} md={12} xl={6}>
              <Card
                size="small"
                className="h-full"
                style={{ borderTop: index === 0 ? '3px solid #13c2c2' : '3px solid #1890ff' }}
              >
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-slate-900 text-white">
                    {step.icon}
                  </div>
                  <div>
                    <div className="text-xs text-slate-400">0{index + 1}</div>
                    <Title level={5} className="!mb-0">
                      {step.title}
                    </Title>
                  </div>
                </div>
                <Text type="secondary" className="mt-3 block">
                  {step.description}
                </Text>
                <Button className="mt-4" type="link" href={step.action}>
                  打开
                </Button>
              </Card>
            </Col>
          ))}
        </Row>
      </Card>

      <Card
        title={
          <span className="flex items-center gap-2">
            <SlidersHorizontal className="h-4 w-4" />
            质量与现状
          </span>
        }
      >
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Card size="small" className="h-full bg-slate-50">
              <div className="mb-2 text-sm text-slate-500">当前治理项</div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Text>待绑定资源</Text>
                  <Tag color="orange">{state.counts.unboundResources}</Tag>
                </div>
                <div className="flex items-center justify-between">
                  <Text>孤儿 CI</Text>
                  <Tag color="red">{state.counts.orphanCIs}</Tag>
                </div>
                <div className="flex items-center justify-between">
                  <Text>未关联 CI</Text>
                  <Tag color="blue">{state.counts.unlinkedCIs}</Tag>
                </div>
              </div>
            </Card>
          </Col>
          <Col xs={24} md={12}>
            <Card size="small" className="h-full bg-slate-50">
              <div className="mb-2 text-sm text-slate-500">最近同步</div>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <Text>最后刷新</Text>
                  <Text strong>{state.lastSyncedAt ? new Date(state.lastSyncedAt).toLocaleString('zh-CN') : '-'}</Text>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button href="/cmdb/reconciliation" type="primary">
                    打开对账中心
                  </Button>
                  <Button href="/cmdb/cloud-resources">查看云资源</Button>
                  <Button href="/cmdb/topology">查看拓扑</Button>
                </div>
              </div>
            </Card>
          </Col>
        </Row>
      </Card>

      <Card title="下一步建议" loading={state.loading}>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <div className="rounded-lg border border-dashed border-slate-200 p-4">
            <div className="font-medium">1. 建模优先</div>
            <div className="mt-1 text-sm text-slate-500">先把 CI 类型和属性模式做成标准模板。</div>
          </div>
          <div className="rounded-lg border border-dashed border-slate-200 p-4">
            <div className="font-medium">2. 服务优先</div>
            <div className="mt-1 text-sm text-slate-500">把业务服务、应用服务、技术服务串起来。</div>
          </div>
          <div className="rounded-lg border border-dashed border-slate-200 p-4">
            <div className="font-medium">3. 关系优先</div>
            <div className="mt-1 text-sm text-slate-500">关系和拓扑是 Service Graph 的核心，不是附属字段。</div>
          </div>
          <div className="rounded-lg border border-dashed border-slate-200 p-4">
            <div className="font-medium">4. 质量优先</div>
            <div className="mt-1 text-sm text-slate-500">对账、绑定、孤儿治理要有闭环，不要只看导入结果。</div>
          </div>
        </div>
      </Card>
    </div>
  );
}
