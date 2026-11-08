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
  Modal,
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
import { splitPilotReason } from '@/lib/api/pilot-reason';
import type { ProductCapabilityState } from '@/lib/api/capability-api';

const { Paragraph, Text } = Typography;

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
  pilot: { label: '预览', color: 'gold' },
  disabled: { label: '未包含', color: 'default' },
};

const FILTER_OPTIONS: { label: string; value: MaturityFilter }[] = [
  { label: '全部', value: 'all' },
  { label: 'GA', value: 'ga' },
  { label: '预览', value: 'pilot' },
  { label: '未就绪', value: 'unready' },
];

/** 部署就绪 + 探测健康才算「此刻可用」。探测缺失（无信号）不算失败。 */
function isUnready(item: ProductCapabilityState): boolean {
  if (!item.deploymentReady) return true;
  return item.health ? !item.health.ok : false;
}

/**
 * 表格里的预览态摘要。三件事必须同时可核对，且不能把千字散文平铺进单元格：
 *   1. degradedReason —— 此刻为什么不可用（排障信号，会变）；
 *   2. 剩余阻断       —— 为什么它还不是 GA（发布决策）；
 *   3. 晋升条件       —— 还差哪块可验证证据，读者要能据此推进。
 * 完整原文不丢，收进详情面板；解析不出晋升条件时如实标「契约不完整」，
 * 不用「暂无数据」把缺契约伪装成正常状态。
 */
function PilotReasonCell({
  item,
  onOpen,
}: {
  item: ProductCapabilityState;
  onOpen: (key: string) => void;
}) {
  const parts = splitPilotReason(item.pilotReason);
  if (!item.degradedReason && !parts) return <Text type='secondary'>—</Text>;
  return (
    <Space orientation='vertical' size={2} style={{ maxWidth: 460 }}>
      {item.degradedReason && <Text type='danger'>{item.degradedReason}</Text>}
      {parts && !parts.structured && (
        <>
          <Text type='warning'>未声明晋升条件（契约不完整）</Text>
          <Text type='secondary' ellipsis={{ tooltip: false }}>
            {parts.full}
          </Text>
        </>
      )}
      {parts?.structured && (
        <>
          <Space size={4} align='start'>
            <Tag color='gold'>剩余阻断</Tag>
            <Text type='secondary' ellipsis={{ tooltip: false }}>
              {parts.blockers}
            </Text>
          </Space>
          <Space size={4} align='start'>
            <Tag color='blue'>晋升条件</Tag>
            <Text ellipsis={{ tooltip: false }}>{parts.criteria}</Text>
          </Space>
          <Button
            type='link'
            size='small'
            style={{ padding: 0, height: 'auto' }}
            onClick={() => onOpen(item.key)}
          >
            查看完整理由
          </Button>
        </>
      )}
    </Space>
  );
}

/** 详情面板按三段回看接口原文，避免「切分后看不到依据」。 */
function PilotReasonDetail({ item }: { item: ProductCapabilityState }) {
  const parts = splitPilotReason(item.pilotReason);
  if (!parts) return <Text type='secondary'>该条目未下发预览态理由</Text>;
  return (
    <Space orientation='vertical' size={12} style={{ width: '100%' }}>
      <Text type='secondary'>
        对账基线 {item.acceptanceVersion || '未声明'} · 成熟度：预览（Pilot）
        {item.degradedReason ? ` · 此刻不可用：${item.degradedReason}` : ''}
      </Text>
      <div>
        <Text strong>已闭环进展</Text>
        <Paragraph style={{ marginTop: 4, marginBottom: 0 }}>
          {parts.progress || '（理由未区分进展与阻断，见下方剩余阻断）'}
        </Paragraph>
      </div>
      <div>
        <Text strong type='danger'>
          剩余阻断
        </Text>
        <Paragraph style={{ marginTop: 4, marginBottom: 0 }}>{parts.blockers}</Paragraph>
      </div>
      <div>
        <Text strong>晋升条件</Text>
        <Paragraph style={{ marginTop: 4, marginBottom: 0 }}>
          {parts.criteria || '未声明（契约不完整）'}
        </Paragraph>
      </div>
      <div>
        <Text type='secondary'>接口原文</Text>
        <Paragraph style={{ marginTop: 4, marginBottom: 0 }}>{parts.full}</Paragraph>
      </div>
    </Space>
  );
}

export default function CapabilitiesAdminPage() {
  const { message } = App.useApp();
  const [items, setItems] = useState<ProductCapabilityState[]>([]);
  const [acceptanceVersion, setAcceptanceVersion] = useState('');
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 「能力总数 0」的假统计卡与空表格，再被真实数据覆盖。
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<MaturityFilter>('all');
  const [keyword, setKeyword] = useState('');
  const [detailKey, setDetailKey] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getCapabilities();
      setItems(data.items ?? []);
      setAcceptanceVersion(data.acceptanceVersion ?? '');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载能力矩阵失败');
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
    // 有多少预览条目真的写出了晋升条件：分母是预览总数，缺口必须可见，
    // 否则「预览态可核对」只是一句注释。
    const criteriaDeclared = items.filter(
      item => item.maturity === 'pilot' && splitPilotReason(item.pilotReason)?.structured === true
    ).length;
    return { total: items.length, ga, pilot, disabled, unready, criteriaDeclared };
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
      // 预览态要能按「还差什么」检索：拿到一句「真实账号验收」也应能定位
      // 到对应能力，而不必先猜出能力键。
      return `${item.key} ${item.pilotReason ?? ''} ${item.degradedReason ?? ''}`
        .toLowerCase()
        .includes(needle);
    });
  }, [items, filter, keyword]);

  const detailItem = detailKey ? (items.find(item => item.key === detailKey) ?? null) : null;

  const columns: ColumnsType<ProductCapabilityState> = [
    {
      title: '能力',
      dataIndex: 'key',
      width: 260,
      render: (key: string, item) => (
        <Space orientation='vertical' size={0}>
          <Text strong>{key}</Text>
          {item.dependencies.length > 0 && (
            <Text type='secondary' style={{ fontSize: 12 }}>
              依赖：{item.dependencies.join(' / ')}
            </Text>
          )}
        </Space>
      ),
    },
    {
      title: '成熟度',
      dataIndex: 'maturity',
      width: 100,
      render: (maturity: ProductCapabilityState['maturity']) => (
        <Tag color={MATURITY_META[maturity].color}>{MATURITY_META[maturity].label}</Tag>
      ),
    },
    {
      title: '部署就绪',
      dataIndex: 'deploymentReady',
      width: 110,
      render: (ready: boolean) =>
        ready ? <Tag color='green'>就绪</Tag> : <Tag color='orange'>未就绪</Tag>,
    },
    {
      title: '探测',
      dataIndex: 'health',
      width: 150,
      render: (_: unknown, item) => {
        if (!item.health) {
          // 没有探测维度不等于健康，也不等于故障：如实显示「无信号」。
          return <Text type='secondary'>无信号</Text>;
        }
        return item.health.ok ? (
          <Tag color='green'>正常</Tag>
        ) : (
          <Tooltip title={item.health.reason || '探测未通过'}>
            <Tag color='red'>异常</Tag>
          </Tooltip>
        );
      },
    },
    {
      title: '可用动作',
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
      title: '说明',
      key: 'reason',
      render: (_: unknown, item) => <PilotReasonCell item={item} onOpen={setDetailKey} />,
    },
  ];

  return (
    <PageContainer
      header={{ title: '能力矩阵' }}
      extra={
        <Button icon={<RefreshCw size={16} />} onClick={() => void load()} loading={loading}>
          刷新
        </Button>
      }
    >
      <Space orientation='vertical' size={16} style={{ width: '100%' }}>
        {/* 统计卡必须门控：还没拉到数据时渲染成「能力总数 0」会被误读成「矩阵为空」 */}
        {!loading && (
          <Row gutter={16}>
            <Col xs={12} md={6}>
              <Card size='small'>
                <Statistic title='能力总数' value={summary.total} prefix={<ShieldCheck size={16} />} />
              </Card>
            </Col>
            <Col xs={12} md={6}>
              <Card size='small'>
                <Statistic title='GA（可承诺）' value={summary.ga} styles={{ content: { color: '#3f8600' } }} />
              </Card>
            </Col>
            <Col xs={12} md={6}>
              <Card size='small'>
                <Statistic
                  title='预览（Pilot）'
                  value={summary.pilot}
                  styles={{ content: { color: '#d48806' } }}
                  prefix={<FlaskConical size={16} />}
                />
              </Card>
            </Col>
            <Col xs={12} md={6}>
              <Card size='small'>
                <Statistic
                  title='此刻不可用'
                  value={summary.unready}
                  styles={{ content: { color: summary.unready > 0 ? '#cf1322' : undefined } }}
                  prefix={summary.unready > 0 ? <AlertTriangle size={16} /> : <Ban size={16} />}
                />
              </Card>
            </Col>
          </Row>
        )}

        <Space wrap size={6}>
          <Text type='secondary'>契约验收基线</Text>
          <Tag color={acceptanceVersion ? 'blue' : 'default'}>{acceptanceVersion || '未声明'}</Tag>
          <Tooltip title='预览能力中写出「晋升条件」的条数 / 预览总数。小于分母说明有条目无法据此推进晋升，需回后端补齐理由。'>
            <span>
              <Tag color={summary.criteriaDeclared === summary.pilot ? 'green' : 'orange'}>
                晋升条件 {summary.criteriaDeclared}/{summary.pilot}
              </Tag>
            </span>
          </Tooltip>
          <Tooltip title='能力矩阵最近一次对账日期，不是发布版本。它不得早于最新发布版本；单条能力是否可承诺仍看成熟度、部署就绪与说明。'>
            <Text type='secondary' style={{ cursor: 'help' }}>
              说明
            </Text>
          </Tooltip>
        </Space>

        {tenantMissing && (
          <Alert
            type='warning'
            showIcon
            title='当前会话没有租户上下文'
            description='矩阵只反映部署级事实：tenantReady 为 false、allowedActions 为空，不代表能力缺失。'
          />
        )}

        {summary.unready > 0 && (
          <Alert
            type='error'
            showIcon
            title={`${summary.unready} 项能力此刻不可用`}
            description='部署依赖未就绪或探测失败。切到「未就绪」筛选可看到具体原因，修好后刷新本页即可复核。'
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
            placeholder='按能力键或预览理由搜索，如 connector.feishu'
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
                  items.length === 0 ? '能力矩阵为空，请确认后端 /api/v1/capabilities 可达' : '没有符合筛选条件的能力'
                }
              />
            ),
          }}
        />

        <Modal
          open={detailItem !== null}
          onCancel={() => setDetailKey(null)}
          footer={null}
          width={760}
          title={`预览态核对 · ${detailItem?.key ?? ''}`}
        >
          {detailItem && <PilotReasonDetail item={detailItem} />}
        </Modal>
      </Space>
    </PageContainer>
  );
}
