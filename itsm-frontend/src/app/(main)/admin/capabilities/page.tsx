'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Empty,
  Input,
  Row,
  Segmented,
  Space,
  Statistic,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { AlertTriangle, Ban, FlaskConical, RefreshCw, Search, ShieldCheck } from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import { getCapabilities } from '@/lib/api/capability-api';
import type { ProductCapabilityState } from '@/lib/api/capability-api';

const { Text } = Typography;

/**
 * 能力矩阵控制台。
 *
 * 引入原因：矩阵是产品对客户的成熟度承诺（GA / Pilot / Disabled），此前只有
 * `GET /api/v1/capabilities` 一个 JSON 面——运营与售前要看「哪些能力还不能承诺」
 * 只能 curl，于是「预览态」在交付现场既不透明也不可核对。这个页面把三件事
 * 摆到同一屏：成熟度、部署/探测是否就绪、以及每一项**为什么还不是 GA**。
 *
 * 页面只读。成熟度是代码里的发布决策（handlers/capability/handler.go 的 registry），
 * 允许在界面上改等于让一次误点改变对外承诺，因此这里刻意不提供任何写操作。
 */

type MaturityFilter = 'all' | 'ga' | 'pilot' | 'unready';

const MATURITY_META: Record<
  ProductCapabilityState['maturity'],
  { label: string; color: string }
> = {
  ga: { label: 'GA', color: 'green' },
  pilot: { label: 'Preview', color: 'gold' },
  disabled: { label: 'Not included', color: 'default' },
};

const FILTER_OPTIONS: { label: string; value: MaturityFilter }[] = [
  { label: 'All', value: 'all' },
  { label: 'GA', value: 'ga' },
  { label: 'Preview', value: 'pilot' },
  { label: 'Not Ready', value: 'unready' },
];

/** 部署就绪 + 探测健康才算「此刻可用」。探测缺失（无信号）不算失败。 */
function isUnready(item: ProductCapabilityState): boolean {
  if (!item.deploymentReady) return true;
  return item.health ? !item.health.ok : false;
}

export default function CapabilitiesAdminPage() {
  const { message } = App.useApp();
  const [items, setItems] = useState<ProductCapabilityState[]>([]);
  const [acceptanceVersion, setAcceptanceVersion] = useState('');
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<MaturityFilter>('all');
  const [keyword, setKeyword] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getCapabilities();
      setItems(data.items ?? []);
      setAcceptanceVersion(data.acceptanceVersion ?? '');
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to load capability matrix');
      setItems([]);
      setAcceptanceVersion('');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    void load();
  }, [load]);

  const summary = useMemo(() => {
    const ga = items.filter(item => item.maturity === 'ga').length;
    const pilot = items.filter(item => item.maturity === 'pilot').length;
    const disabled = items.filter(item => item.maturity === 'disabled').length;
    const unready = items.filter(isUnready).length;
    return { total: items.length, ga, pilot, disabled, unready };
  }, [items]);

  // 矩阵按租户下发：没有租户上下文时 allowedActions 为空、tenantReady=false，
  // 此时页面只能反映部署级事实，必须说清楚，否则会被当成「全员无权限」。
  const tenantMissing = items.length > 0 && items.every(item => !item.tenantReady);

  const visible = useMemo(() => {
    const needle = keyword.trim().toLowerCase();
    return items.filter(item => {
      if (filter === 'ga' && item.maturity !== 'ga') return false;
      if (filter === 'pilot' && item.maturity !== 'pilot') return false;
      if (filter === 'unready' && !isUnready(item)) return false;
      if (!needle) return true;
      return item.key.toLowerCase().includes(needle);
    });
  }, [items, filter, keyword]);

  const columns: ColumnsType<ProductCapabilityState> = [
    {
      title: 'Capabilities',
      dataIndex: 'key',
      width: 260,
      render: (key: string, item) => (
        <Space orientation='vertical' size={0}>
          <Text strong>{key}</Text>
          {item.dependencies.length > 0 && (
            <Text type='secondary' style={{ fontSize: 12 }}>

              Dependencies:{item.dependencies.join(' / ')}
            </Text>
          )}
        </Space>
      ),
    },
    {
      title: 'Maturity',
      dataIndex: 'maturity',
      width: 100,
      render: (maturity: ProductCapabilityState['maturity']) => (
        <Tag color={MATURITY_META[maturity].color}>{MATURITY_META[maturity].label}</Tag>
      ),
    },
    {
      title: 'Ready to deploy',
      dataIndex: 'deploymentReady',
      width: 110,
      render: (ready: boolean) =>
        ready ? <Tag color='green'>Ready</Tag> : <Tag color='orange'>Not Ready</Tag>,
    },
    {
      title: 'Detection',
      dataIndex: 'health',
      width: 150,
      render: (_: unknown, item) => {
        if (!item.health) {
          // 没有探测维度不等于健康，也不等于故障：如实显示「无信号」。
          return <Text type='secondary'>No signal</Text>;
        }
        return item.health.ok ? (
          <Tag color='green'>OK</Tag>
        ) : (
          <Tooltip title={item.health.reason || 'Detection failed'}>
            <Tag color='red'>Error</Tag>
          </Tooltip>
        );
      },
    },
    {
      title: 'Available actions',
      dataIndex: 'allowedActions',
      width: 140,
      render: (actions: string[]) =>
        actions.length > 0 ? (
          <Space size={4} wrap>
            {actions.map(action => (
              <Tag key={action}>{action}</Tag>
            ))}
          </Space>
        ) : (
          <Text type='secondary'>—</Text>
        ),
    },
    {
      title: 'Description',
      key: 'reason',
      render: (_: unknown, item) => {
        // 两件事必须同时可见，不能二选一：
        //   degradedReason = 此刻为什么不可用（排障信号，会变）；
        //   pilotReason    = 为什么它还不是 GA（发布决策，不会自己消失）。
        // 此前只显示前者，于是「部署未就绪」的 Pilot 能力在页面上看不到任何
        // 晋升线索——运维修好 READY 变量后仍不知道这块能力还差什么。
        if (!item.degradedReason && !item.pilotReason) {
          return <Text type='secondary'>—</Text>;
        }
        return (
          <Space orientation='vertical' size={2}>
            {item.degradedReason && <Text type='danger'>{item.degradedReason}</Text>}
            {item.pilotReason && <Text type='secondary'>{item.pilotReason}</Text>}
          </Space>
        );
      },
    },
  ];

  return (
    <PageContainer
      header={{ title: 'Capability Matrix' }}
      extra={
        <Button icon={<RefreshCw size={16} />} onClick={() => void load()} loading={loading}>

          Refresh
        </Button>
      }
    >
      <Space orientation='vertical' size={16} style={{ width: '100%' }}>
        <Row gutter={16}>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic title="Total number of abilities" value={summary.total} prefix={<ShieldCheck size={16} />} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic title="GA (can commit)" value={summary.ga} styles={{ content: { color: '#3f8600' } }} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic
                title="Preview (Pilot)"
                value={summary.pilot}
                styles={{ content: { color: '#d48806' } }}
                prefix={<FlaskConical size={16} />}
              />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size='small'>
              <Statistic
                title="Not available at this time"
                value={summary.unready}
                styles={{ content: { color: summary.unready > 0 ? '#cf1322' : undefined } }}
                prefix={summary.unready > 0 ? <AlertTriangle size={16} /> : <Ban size={16} />}
              />
            </Card>
          </Col>
        </Row>

        <Space wrap size={6}>
          <Text type='secondary'>Contract acceptance baseline</Text>
          <Tag color={acceptanceVersion ? 'blue' : 'default'}>{acceptanceVersion || 'Not declared'}</Tag>
          <Tooltip title="The date of the last reconciliation of the capability matrix, not the release version. It must not be earlier than the latest released version; whether a single capability can be promised depends on maturity, deployment readiness and description.">
            <Text type='secondary' style={{ cursor: 'help' }}>

              Description
            </Text>
          </Tooltip>
        </Space>

        {tenantMissing && (
          <Alert
            type='warning'
            showIcon
            title="There is no tenant context for the current session"
            description="The matrix only reflects deployment-level facts: tenantReady is false and allowedActions is empty, which does not mean a lack of capabilities."
          />
        )}

        {summary.unready > 0 && (
          <Alert
            type='error'
            showIcon
            title={`${summary.unready} Ability is not available at this time`}
            description="The deployment dependency is not ready or the detection failed. Switch to the &quot;Not Ready&quot; filter to see the specific reasons. After repairing, refresh this page to review."
          />
        )}

        <Space wrap>
          <Segmented<MaturityFilter>
            options={FILTER_OPTIONS}
            value={filter}
            onChange={value => setFilter(value)}
          />
          <Input
            allowClear
            prefix={<Search size={14} />}
            placeholder="Search by capability key, such as connector.feishu"
            style={{ width: 280 }}
            value={keyword}
            onChange={event => setKeyword(event.target.value)}
          />
        </Space>

        <Table<ProductCapabilityState>
          rowKey='key'
          size='small'
          loading={loading}
          columns={columns}
          dataSource={visible}
          pagination={{ pageSize: 20, hideOnSinglePage: true }}
          locale={{
            emptyText: (
              <Empty
                description={
                  items.length === 0 ? 'The capability matrix is empty, please confirm that the backend /api/v1/capabilities is reachable' : 'No ability matching the filter criteria'
                }
              />
            ),
          }}
        />
      </Space>
    </PageContainer>
  );
}
