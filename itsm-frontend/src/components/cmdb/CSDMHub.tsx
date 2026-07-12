'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { App, Button, Card, Col, Divider, Row, Space, Tag, Typography } from 'antd';
import {
  Cloud,
  Database,
  GitBranch,
  Layers3,
  Plus,
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

type WorkbenchAction = {
  title: string;
  description: string;
  href: string;
  primary?: boolean;
  icon: React.ReactNode;
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
        <Tag color="blue">分区</Tag>
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
      // Use Promise.allSettled so that a single failing endpoint (e.g. cloud accounts
      // / reconciliation when not yet wired in dev) does not block the dashboard
      // counts. Previously a `Promise.all` would short-circuit on the first rejection
      // and the catch block only flipped `loading` off without ever calling setState
      // for counts — leaving the UI stuck at zero.
      const results = await Promise.allSettled([
        CMDBApi.getCMDBStats(),
        CMDBApi.getCITypes(),
        CMDBApi.getCloudAccounts(),
        CMDBApi.getCloudServices(),
        CMDBApi.getDiscoverySources(),
        CMDBApi.getCloudResources(),
        CMDBApi.getReconciliationResults(),
      ]);
  
      const pick = <T,>(idx: number, fallback: T): T => {
        const r = results[idx];
        return r && r.status === 'fulfilled' ? ((r as PromiseFulfilledResult<T>).value as T) : fallback;
      };
  
      const stats = pick(0, {});
      const ciTypes = pick(1, []);
      const cloudAccounts = pick(2, []);
      const cloudServices = pick(3, []);
      const discoverySources = pick(4, []);
      const cloudResources = pick(5, []);
      const reconciliation = pick(6, {});
  
      const statsData = (stats as any)?.data ?? stats;
      const reconData = (reconciliation as any)?.data ?? reconciliation;
      const summary = (reconData as any)?.summary ?? {};
      const unboundResources = normalizeList<Record<string, unknown>>((reconData as any)?.unboundResources ?? (reconData as any)?.unboundResources);
      const orphanCIs = normalizeList<Record<string, unknown>>((reconData as any)?.orphanCIs ?? (reconData as any)?.orphanCis);
      const unlinkedCIs = normalizeList<Record<string, unknown>>((reconData as any)?.unlinkedCIs ?? (reconData as any)?.unlinkedCis);
  
      setState({
        loading: false,
        lastSyncedAt: new Date().toISOString(),
        counts: {
          totalCIs:
            statsData?.totalCount ?? statsData?.totalCount ?? statsData?.totalCIs ?? statsData?.totalCis ?? 0,
          ciTypes: Array.isArray(ciTypes) ? ciTypes.length : normalizeList(ciTypes).length,
          cloudAccounts: normalizeList(cloudAccounts).length,
          cloudServices: normalizeList(cloudServices).length,
          discoverySources: normalizeList(discoverySources).length,
          cloudResources: normalizeList(cloudResources).length,
          boundResources: Number(summary.boundResourceCount ?? summary.boundResourceCount ?? 0),
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
      title: '基础资料',
      description: '维护类型、云账号和发现源，保证后续录入和同步有标准可依。',
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
      title: '服务建模',
      description: '把配置项、服务目录和关系串起来，支撑故障、变更和影响分析。',
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
      title: '服务图谱',
      description: '从发现、绑定、关联到拓扑分析，形成可追溯的运行视图。',
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

  const governanceTotal =
    state.counts.unboundResources + state.counts.orphanCIs + state.counts.unlinkedCIs;

  const workbenchActions: WorkbenchAction[] = [
    {
      title: '录入配置项',
      description: '手工录入或从云资源带入基础信息。',
      href: '/cmdb/cis/create',
      primary: true,
      icon: <Plus className="h-5 w-5" />,
    },
    {
      title: '处理待绑定资源',
      description: `${state.counts.unboundResources} 个云资源还没有进入 CMDB。`,
      href: '/cmdb/reconciliation',
      primary: governanceTotal > 0,
      icon: <SlidersHorizontal className="h-5 w-5" />,
    },
    {
      title: '维护类型模板',
      description: '配置字段模板，让录入表单更标准。',
      href: '/admin/cmdb-types',
      icon: <Database className="h-5 w-5" />,
    },
    {
      title: '管理关系',
      description: '补齐依赖、托管、影响关系。',
      href: '/cmdb/relationships',
      icon: <GitBranch className="h-5 w-5" />,
    },
    {
      title: '接入云资源',
      description: '维护云账号、云服务目录和发现源。',
      href: '/cmdb/registry',
      icon: <Cloud className="h-5 w-5" />,
    },
    {
      title: '查看拓扑影响',
      description: '从 CI 出发查看上下游影响范围。',
      href: '/cmdb/topology',
      icon: <Layers3 className="h-5 w-5" />,
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
        title="配置管理数据库 (CMDB)"
        description="围绕配置项、云资源、关系拓扑和数据质量的日常工作台。"
        actions={
          <Space wrap>
            <Button icon={<RefreshCw className="h-4 w-4" />} loading={state.loading} onClick={load}>
              刷新总览
            </Button>
            <Button type="primary" icon={<Sparkles className="h-4 w-4" />} onClick={() => router.push('/cmdb/ci')}>
              配置项工作台
            </Button>
          </Space>
        }
      />

      <StatsOverview items={statsItems} />

      <Card loading={state.loading} className="border-slate-200 shadow-sm">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <Title level={4} className="!mb-0">
                当前最需要处理的事
              </Title>
              {governanceTotal > 0 ? <Tag color="orange">{governanceTotal} 个待治理项</Tag> : <Tag color="green">数据状态良好</Tag>}
            </div>
            <Text type="secondary" className="mt-2 block">
              优先处理未绑定云资源、孤儿 CI 和未关联 CI，避免工单、变更和故障分析引用到不可信数据。
            </Text>
          </div>
          <Space wrap>
            <Button type="primary" href="/cmdb/reconciliation">
              打开对账中心
            </Button>
            <Button href="/cmdb/cloud-resources">查看云资源</Button>
            <Button href="/reports/cmdb-quality">CMDB 质量报表</Button>
          </Space>
        </div>
      </Card>

      <Card
        title={
          <span className="flex items-center gap-2">
            <Sparkles className="h-4 w-4" />
            常用工作
          </span>
        }
        loading={state.loading}
      >
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {workbenchActions.map(action => (
            <a
              key={action.href}
              href={action.href}
              className="block rounded-lg border border-slate-200 p-4 text-inherit transition hover:border-blue-300 hover:bg-blue-50/40"
            >
              <div className="flex items-start gap-3">
                <div
                  className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${
                    action.primary ? 'bg-blue-600 text-white' : 'bg-slate-100 text-slate-700'
                  }`}
                >
                  {action.icon}
                </div>
                <div className="min-w-0">
                  <div className="flex items-center gap-2 font-medium text-slate-900">
                    {action.title}
                    {action.primary && <Tag color="blue">推荐</Tag>}
                  </div>
                  <div className="mt-1 text-sm text-slate-500">{action.description}</div>
                </div>
              </div>
            </a>
          ))}
        </div>
      </Card>

      <Card
        title={
          <span className="flex items-center gap-2">
            <Workflow className="h-4 w-4" />
            能力分区
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
            <Workflow className="h-4 w-4" />
            建模与治理流程
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

      <Card title="质量与现状">
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

      <Card title="日常巡检清单" loading={state.loading}>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <div className="rounded-lg border border-dashed border-slate-200 p-4">
            <div className="font-medium">1. 新资源是否已入库</div>
            <div className="mt-1 text-sm text-slate-500">
              还有 {state.counts.unboundResources} 个云资源待绑定到配置项。
            </div>
            <Button className="mt-3" size="small" href="/cmdb/reconciliation">
              去处理
            </Button>
          </div>
          <div className="rounded-lg border border-dashed border-slate-200 p-4">
            <div className="font-medium">2. 类型模板是否够用</div>
            <div className="mt-1 text-sm text-slate-500">
              当前有 {state.counts.ciTypes} 个 CI 类型，检查核心字段是否已标准化。
            </div>
            <Button className="mt-3" size="small" href="/admin/cmdb-types">
              维护模板
            </Button>
          </div>
          <div className="rounded-lg border border-dashed border-slate-200 p-4">
            <div className="font-medium">3. 关键 CI 是否有关联</div>
            <div className="mt-1 text-sm text-slate-500">
              还有 {state.counts.unlinkedCIs} 个 CI 缺少关系，影响故障和变更分析。
            </div>
            <Button className="mt-3" size="small" href="/cmdb/relationships">
              补关系
            </Button>
          </div>
          <div className="rounded-lg border border-dashed border-slate-200 p-4">
            <div className="font-medium">4. 数据质量是否可用</div>
            <div className="mt-1 text-sm text-slate-500">
              孤儿 CI 当前 {state.counts.orphanCIs} 个，建议纳入每周巡检。
            </div>
            <Button className="mt-3" size="small" href="/reports/cmdb-quality">
              看报表
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}
