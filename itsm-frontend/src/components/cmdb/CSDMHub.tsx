'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { App, Button, Card, Col, Row, Space, Tag, Typography } from 'antd';
import { Database, GitBranch, Plus, RefreshCw, Server, ShieldCheck } from 'lucide-react';

import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview, type StatsOverviewItem } from '@/components/ui/StatsOverview';
import { CMDBApi } from '@/lib/api/cmdb-api';
import { CMDB_CAPABILITIES } from '@/lib/cmdb/cmdb-capabilities';
import { useCapabilities } from '@/lib/hooks/useCapabilities';
import { useI18n } from '@/lib/i18n/useI18n';

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
  const { t } = useI18n();
  const { capabilities, allows } = useCapabilities();
  const availableCapabilities = CMDB_CAPABILITIES.filter(item => allows(item.capabilityKey));
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
      message.error(t('cmdb.hub.loadFailed'));
    }
  }, [message, t]);

  React.useEffect(() => {
    void load();
  }, [load]);

  const statsItems: StatsOverviewItem[] = [
    {
      key: 'total-cis',
      title: t('cmdb.totalCIs'),
      value: overview.totalCIs,
      prefix: <Database className="mr-2 text-blue-500" />,
      accentColor: '#1677ff',
    },
    {
      key: 'ci-types',
      title: t('cmdb.hub.ciTypes'),
      value: overview.ciTypes,
      prefix: <Server className="mr-2 text-cyan-600" />,
      accentColor: '#08979c',
    },
    {
      key: 'ga-capabilities',
      title: t('cmdb.hub.gaCapabilities'),
      value: availableCapabilities.filter(item =>
        capabilities.find(capability => capability.key === item.capabilityKey)?.maturity === 'ga'
      ).length,
      prefix: <ShieldCheck className="mr-2 text-green-600" />,
      accentColor: '#389e0d',
    },
  ];

  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title={t('cmdb.title')}
        description={t('cmdb.hub.description')}
        actions={
          <Space wrap>
            <Button icon={<RefreshCw className="h-4 w-4" />} loading={overview.loading} onClick={load}>
              {t('cmdb.refresh')}
            </Button>
            <Button type="primary" icon={<Plus className="h-4 w-4" />} onClick={() => router.push('/cmdb/cis/create')}>
              {t('cmdb.newCI')}
            </Button>
          </Space>
        }
      />

      {overview.error && (
        <Card className="border-red-200 bg-red-50">
          <Text type="danger">{t('cmdb.hub.errorBanner')}</Text>
        </Card>
      )}

      <StatsOverview items={statsItems} />

      <Card title={t('cmdb.hub.gaCapabilities')} loading={overview.loading}>
        <Row gutter={[16, 16]}>
          {availableCapabilities.map(capability => {
            const maturity = capabilities.find(item => item.key === capability.capabilityKey)?.maturity;
            return (
            <Col key={capability.key} xs={24} md={12} xl={6}>
              <Card size="small" className="h-full border-slate-200">
                <div className="flex items-center justify-between gap-2">
                  <Title level={5} className="!mb-0">{capability.title}</Title>
                  <Tag color={maturity === 'ga' ? 'green' : 'gold'}>{maturity === 'ga' ? t('cmdb.hub.gaBadge') : t('cmdb.hub.pilotBadge')}</Tag>
                </div>
                <Paragraph type="secondary" className="!mb-4 !mt-3 min-h-11">
                  {capability.description}
                </Paragraph>
                <Button type="link" className="!px-0" onClick={() => router.push(capability.href)}>
                  {t('cmdb.hub.enterButton')}
                </Button>
              </Card>
            </Col>
            );
          })}
        </Row>
      </Card>

      <Card title={t('cmdb.hub.recommendedActions')}>
        <Space wrap size="middle">
          <Button type="primary" icon={<Server className="h-4 w-4" />} onClick={() => router.push('/cmdb/cis')}>
            {t('cmdb.hub.ciWorkbench')}
          </Button>
          <Button icon={<Database className="h-4 w-4" />} onClick={() => router.push('/admin/cmdb-types')}>
            {t('cmdb.hub.maintainTypeTemplates')}
          </Button>
          <Button icon={<GitBranch className="h-4 w-4" />} onClick={() => router.push('/cmdb/relationships')}>
            {t('cmdb.hub.maintainRelationships')}
          </Button>
          <Button icon={<GitBranch className="h-4 w-4" />} onClick={() => router.push('/cmdb/topology')}>
            {t('cmdb.hub.viewTopology')}
          </Button>
        </Space>
        <Text type="secondary" className="mt-4 block">
          {t('cmdb.hub.autoDiscoverNote')}
        </Text>
        {overview.refreshedAt && (
          <Text type="secondary" className="mt-2 block text-xs">
            {t('cmdb.hub.refreshedAt', { time: new Date(overview.refreshedAt).toLocaleString() })}
          </Text>
        )}
      </Card>
    </div>
  );
}
