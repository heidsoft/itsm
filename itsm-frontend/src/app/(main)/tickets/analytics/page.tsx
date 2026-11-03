'use client';

import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Empty,
  Progress,
  Row,
  Space,
  Spin,
  Statistic,
  Tabs,
  Tag,
  Typography,
  DatePicker,
} from 'antd';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip as RechartsTooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { RotateCcw } from 'lucide-react';
import dayjs from 'dayjs';
import { TicketApi } from '@/lib/api/ticket-api';
import type { TicketStatsResponse } from '@/lib/api/ticket-api';
import { isPermissionDeniedError, serverErrorMessage } from '@/lib/api/http-client';
import {
  TicketPriority,
  TicketPriorityConfig,
  TicketStatus,
  TicketStatusConfig,
} from '@/constants/taxonomy';
import { TicketDeepAnalytics } from '@/components/business/TicketDeepAnalytics';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;

// 图表配色按枚举键取，不按下标轮转：下标轮转会随租户词表外的历史取值漂移。
const STATUS_COLORS: Record<TicketStatus, string> = {
  [TicketStatus.NEW]: '#1890ff',
  [TicketStatus.OPEN]: '#fa8c16',
  [TicketStatus.ASSIGNED]: '#2f54eb',
  [TicketStatus.IN_PROGRESS]: '#13c2c2',
  [TicketStatus.PENDING]: '#faad14',
  [TicketStatus.RESOLVED]: '#52c41a',
  [TicketStatus.CLOSED]: '#722ed1',
  [TicketStatus.CANCELLED]: '#8c8c8c',
  [TicketStatus.APPROVED]: '#389e0d',
  [TicketStatus.REJECTED]: '#ff4d4f',
};

const PRIORITY_COLORS: Record<TicketPriority, string> = {
  [TicketPriority.LOW]: '#52c41a',
  [TicketPriority.MEDIUM]: '#faad14',
  [TicketPriority.HIGH]: '#fa541c',
  [TicketPriority.URGENT]: '#ff4d4f',
  [TicketPriority.CRITICAL]: '#cf1322',
};

const UNKNOWN_COLOR = '#d9d9d9';

const SCOPE_LABELS: Record<TicketStatsResponse['scope'], string> = {
  all: '租户全量工单',
  department: '本部门可见工单',
  owner: '本人创建或受理的工单',
};

interface Slice {
  key: string;
  name: string;
  value: number;
  color: string;
}

function statusLabel(status: string): string {
  return (Object.values(TicketStatus) as string[]).includes(status)
    ? TicketStatusConfig[status as TicketStatus].label
    : status;
}

function priorityLabel(priority: string): string {
  return (Object.values(TicketPriority) as string[]).includes(priority)
    ? TicketPriorityConfig[priority as TicketPriority].label
    : priority;
}

// 后端已按词表顺序返回且只含真实存在的取值，前端不再二次聚合、补零或排序。
function toStatusSlices(byStatus: TicketStatsResponse['byStatus']): Slice[] {
  return byStatus.map((entry) => ({
    key: entry.status,
    name: statusLabel(entry.status),
    value: entry.count,
    color:
      (Object.values(TicketStatus) as string[]).includes(entry.status)
        ? STATUS_COLORS[entry.status as TicketStatus]
        : UNKNOWN_COLOR,
  }));
}

function toPrioritySlices(byPriority: TicketStatsResponse['byPriority']): Slice[] {
  return byPriority.map((entry) => ({
    key: entry.priority,
    name: priorityLabel(entry.priority),
    value: entry.count,
    color:
      (Object.values(TicketPriority) as string[]).includes(entry.priority)
        ? PRIORITY_COLORS[entry.priority as TicketPriority]
        : UNKNOWN_COLOR,
  }));
}

function percentage(value: number, total: number): number {
  return total > 0 ? (value / total) * 100 : 0;
}

type LoadState =
  | { kind: 'loading' }
  | { kind: 'error'; denied: boolean; reason: string }
  | { kind: 'ready'; stats: TicketStatsResponse };

const TicketAnalytics: React.FC = () => {
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(29, 'day'),
    dayjs(),
  ]);
  const [state, setState] = useState<LoadState>({ kind: 'loading' });
  const [activeTab, setActiveTab] = useState('overview');
  // 改窗口会立刻重取；若旧请求后返回，会把上一个窗口的数字盖到新窗口上。
  const requestIdRef = useRef(0);

  const fetchData = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setState({ kind: 'loading' });
    try {
      const stats = await TicketApi.getTicketStats({
        dateFrom: dateRange[0].format('YYYY-MM-DD'),
        dateTo: dateRange[1].format('YYYY-MM-DD'),
      });
      if (requestId !== requestIdRef.current) return;
      setState({ kind: 'ready', stats });
    } catch (error) {
      if (requestId !== requestIdRef.current) return;
      setState({
        kind: 'error',
        denied: isPermissionDeniedError(error),
        reason: serverErrorMessage(error, '后端未能返回工单统计。'),
      });
    }
  }, [dateRange]);

  useEffect(() => {
    void fetchData();
  }, [fetchData]);

  const renderShell = (content: React.ReactNode) => (
    <div className="p-6">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <Title level={2} className="!mb-0">
          工单统计
        </Title>
        <Space>
          <RangePicker
            value={dateRange}
            onChange={(dates) => {
              if (dates && dates[0] && dates[1]) {
                setDateRange([dates[0], dates[1]]);
              }
            }}
            format="YYYY-MM-DD"
            allowClear={false}
          />
          <Button icon={<RotateCcw />} onClick={fetchData} loading={state.kind === 'loading'}>
            刷新数据
          </Button>
        </Space>
      </div>
      {content}
    </div>
  );

  // 失败态不渲染任何数字：一屏 0 会被读成「这个租户没有工单」。
  if (state.kind === 'error') {
    return renderShell(
      <Alert
        type={state.denied ? 'warning' : 'error'}
        showIcon
        title={state.denied ? '工单统计没有访问权限' : '工单统计加载失败'}
        description={
          state.denied
            ? `${state.reason}。该页面的计数器、分布与趋势都来自 GET /api/v1/tickets/stats，需要 ticket:read；请联系管理员为当前角色开放该权限。缺少数据不等于租户里没有工单。`
            : `${state.reason}。请稍后重试；若持续失败，请联系管理员查看后端日志。`
        }
        action={
          <Button size="small" icon={<RotateCcw />} onClick={fetchData}>
            重试
          </Button>
        }
      />
    );
  }

  if (state.kind === 'loading') {
    return renderShell(
      <div className="flex min-h-[400px] items-center justify-center">
        <Spin size="large" description="加载工单统计..." />
      </div>
    );
  }

  const { stats } = state;
  const statusSlices = toStatusSlices(stats.byStatus);
  const prioritySlices = toPrioritySlices(stats.byPriority);
  const statusTotal = statusSlices.reduce((acc, slice) => acc + slice.value, 0);
  const priorityTotal = prioritySlices.reduce((acc, slice) => acc + slice.value, 0);

  // 后端把 scope 锁定在 all/department/owner（handlers/ticket/stats_contract_test.go）。
  // 词表外的值如实显示原始值，不冒充「租户全量」，否则口径提示本身就是错的。
  const scopeText = SCOPE_LABELS[stats.scope] ?? `未识别的数据范围 ${stats.scope}`;
  const windowText = stats.window
    ? `${stats.window.dateFrom} 至 ${stats.window.dateTo}（${stats.window.days} 天，按创建时间）`
    : '全部时间（按创建时间）';

  const kpiCards = [
    { title: '工单总数', value: stats.total, color: '#1890ff' },
    { title: '待处理', value: stats.open, color: STATUS_COLORS[TicketStatus.OPEN] },
    { title: '处理中', value: stats.inProgress, color: STATUS_COLORS[TicketStatus.IN_PROGRESS] },
    { title: '挂起', value: stats.pending, color: STATUS_COLORS[TicketStatus.PENDING] },
    { title: '已解决', value: stats.resolved, color: STATUS_COLORS[TicketStatus.RESOLVED] },
    { title: '已关闭', value: stats.closed, color: STATUS_COLORS[TicketStatus.CLOSED] },
    { title: '高优先级', value: stats.highPriority, color: PRIORITY_COLORS[TicketPriority.HIGH] },
    {
      title: '已超期',
      value: stats.overdue,
      color: stats.overdue > 0 ? '#cf1322' : undefined,
    },
  ];

  const overviewTab = (
    <Space orientation="vertical" size="middle" className="w-full">
      <div className="rounded-md border border-gray-200 bg-white px-4 py-3">
        <Text type="secondary">
          统计口径：{scopeText}；时间窗口 {windowText}。本页所有卡片、分布与趋势共用这同一次查询，
          不含其他数据源。
        </Text>
      </div>

      <Row gutter={[16, 16]}>
        {kpiCards.map((card) => (
          <Col xs={12} sm={8} lg={6} xl={3} key={card.title}>
            <Card size="small">
              <Statistic title={card.title} value={card.value} styles={{ content: { color: card.color } }} />
            </Card>
          </Col>
        ))}
      </Row>

      <Card
        title="处理效率与 SLA"
        size="small"
        extra={<Text type="secondary">SLA 读数按租户全量计算，不跟随登录者的数据范围</Text>}
      >
        {stats.sla ? (
          <Row gutter={[16, 16]}>
            <Col xs={12} lg={6}>
              <div className="flex flex-col items-center">
                <Text type="secondary" className="mb-1">
                  SLA 达标率
                </Text>
                <Progress
                  type="circle"
                  size={72}
                  percent={Math.round(stats.sla.complianceRate)}
                />
                <Text type="secondary" className="mt-1 text-xs">
                  样本 {stats.sla.sampledTickets} 张，违约 {stats.sla.violatedTickets} 张
                </Text>
              </div>
            </Col>
            <Col xs={12} lg={6}>
              <Statistic
                title="平均响应时长（小时）"
                value={stats.sla.avgResponseMinutes / 60}
                precision={1}
              />
            </Col>
            <Col xs={12} lg={6}>
              <Statistic
                title="平均解决时长（小时）"
                value={stats.sla.avgResolutionHours}
                precision={1}
              />
            </Col>
            <Col xs={12} lg={6}>
              <Statistic
                title="解决率"
                value={percentage(stats.resolved + stats.closed, stats.total)}
                precision={1}
                suffix="%"
              />
            </Col>
          </Row>
        ) : (
          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <Alert
                type="info"
                showIcon
                title="SLA 达标率暂不可用"
                description="未接入 SLA 计时服务或本次读数失败，因此这里显示「没取到」而不是 0%。请先在 SLA 模块完成定义与监控配置。"
              />
            </Col>
            <Col xs={12} lg={6}>
              <Statistic
                title="解决率"
                value={percentage(stats.resolved + stats.closed, stats.total)}
                precision={1}
                suffix="%"
              />
            </Col>
            <Col xs={12} lg={6}>
              <Statistic
                title="超期率"
                value={percentage(stats.overdue, stats.total)}
                precision={1}
                suffix="%"
              />
            </Col>
          </Row>
        )}
      </Card>

      <Card title="工单趋势（每日新建/解决）" size="small">
        {stats.dailyTrend && stats.dailyTrend.length > 0 ? (
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={stats.dailyTrend}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="date"
                tickFormatter={(d: string) => dayjs(d).format('MM-DD')}
                fontSize={12}
              />
              <YAxis allowDecimals={false} />
              <RechartsTooltip
                labelFormatter={(d: unknown) => dayjs(String(d)).format('YYYY-MM-DD')}
                formatter={(value: unknown, name: unknown) => [
                  Number(value) || 0,
                  name === 'created' ? '新建' : name === 'resolved' ? '已解决' : String(name),
                ]}
              />
              <Legend formatter={(v: string) => (v === 'created' ? '新建' : v === 'resolved' ? '已解决' : v)} />
              <Line type="monotone" dataKey="created" stroke="#1890ff" strokeWidth={2} dot={false} />
              <Line type="monotone" dataKey="resolved" stroke="#52c41a" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <Empty
            description={
              stats.dailyTrend === undefined
                ? '本次请求未返回趋势序列（后端未启用窗口趋势）'
                : '所选时间范围内没有新建或解决的工单'
            }
          />
        )}
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="状态分布" size="small">
            {statusSlices.length > 0 ? (
              <ResponsiveContainer width="100%" height={280}>
                <PieChart>
                  <Pie
                    data={statusSlices}
                    dataKey="value"
                    nameKey="name"
                    cx="50%"
                    cy="50%"
                    outerRadius={90}
                    label
                  >
                    {statusSlices.map((slice) => (
                      <Cell key={slice.key} fill={slice.color} />
                    ))}
                  </Pie>
                  <RechartsTooltip
                    formatter={(value: unknown, name: unknown) => [
                      `${value} (${percentage(Number(value) || 0, statusTotal).toFixed(1)}%)`,
                      String(name),
                    ]}
                  />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <Empty description="所选范围内暂无状态分布数据" />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="优先级分布" size="small">
            {prioritySlices.length > 0 ? (
              <ResponsiveContainer width="100%" height={280}>
                <BarChart data={prioritySlices}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" fontSize={12} />
                  <YAxis allowDecimals={false} />
                  <RechartsTooltip
                    formatter={(value: unknown, name: unknown) => [
                      `${value} (${percentage(Number(value) || 0, priorityTotal).toFixed(1)}%)`,
                      String(name),
                    ]}
                  />
                  <Bar dataKey="value" name="工单数" radius={[4, 4, 0, 0]}>
                    {prioritySlices.map((slice) => (
                      <Cell key={slice.key} fill={slice.color} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <Empty description="所选范围内暂无优先级分布数据" />
            )}
          </Card>
        </Col>
      </Row>

      <Card title="状态明细" size="small">
        {statusSlices.length > 0 ? (
          <Row gutter={[16, 16]}>
            {statusSlices.map((slice) => (
              <Col xs={12} sm={8} md={6} key={slice.key}>
                <div className="flex items-center gap-2">
                  <Tag color={slice.color} className="m-0">
                    {slice.name}
                  </Tag>
                  <span className="text-lg font-semibold">{slice.value}</span>
                </div>
              </Col>
            ))}
          </Row>
        ) : (
          <Empty description="所选范围内没有工单" />
        )}
      </Card>
    </Space>
  );

  return renderShell(
    <Tabs
      activeKey={activeTab}
      onChange={setActiveTab}
      items={[
        { key: 'overview', label: '总览', children: overviewTab },
        { key: 'deep', label: '深度分析', children: <TicketDeepAnalytics /> },
      ]}
    />
  );
};

export default TicketAnalytics;
