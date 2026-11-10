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
  all: 'Tenant full work order',
  department: 'Work orders are visible to this department',
  owner: 'Work order created or accepted by me',
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
        reason: serverErrorMessage(error, 'The backend failed to return work order statistics.'),
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

          Work order statistics
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

            Refresh Data
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
        title={state.denied ? 'No access rights to work order statistics' : 'Failed to load work order statistics'}
        description={
          state.denied
            ? `${state.reason}. The counters, distributions and trends of this page all come from GET /api/v1/tickets/stats, which requires ticket:read; please contact the administrator to enable this permission for the current role. Lack of data does not mean there are no work orders in the tenant.`
            : `${state.reason}. Please try again later; if it continues to fail, please contact the administrator to check the backend log.`
        }
        action={
          <Button size="small" icon={<RotateCcw />} onClick={fetchData}>

            Retry
          </Button>
        }
      />
    );
  }

  if (state.kind === 'loading') {
    return renderShell(
      <div className="flex min-h-[400px] items-center justify-center">
        <Spin size="large" description="Loading work order statistics..." />
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
  const scopeText = SCOPE_LABELS[stats.scope] ?? `Unrecognized data range ${stats.scope}`;
  const windowText = stats.window
    ? `${stats.window.dateFrom} to ${stats.window.dateTo}（${stats.window.days} days, by creation time)`
    : 'All time (by creation time)';

  const kpiCards = [
    { title: 'Total number of work orders', value: stats.total, color: '#1890ff' },
    { title: 'Pending', value: stats.open, color: STATUS_COLORS[TicketStatus.OPEN] },
    { title: 'In Progress', value: stats.inProgress, color: STATUS_COLORS[TicketStatus.IN_PROGRESS] },
    { title: 'hang', value: stats.pending, color: STATUS_COLORS[TicketStatus.PENDING] },
    { title: 'Resolved', value: stats.resolved, color: STATUS_COLORS[TicketStatus.RESOLVED] },
    { title: 'Closed', value: stats.closed, color: STATUS_COLORS[TicketStatus.CLOSED] },
    { title: 'High Priority', value: stats.highPriority, color: PRIORITY_COLORS[TicketPriority.HIGH] },
    {
      title: 'Expired',
      value: stats.overdue,
      color: stats.overdue > 0 ? '#cf1322' : undefined,
    },
  ];

  const overviewTab = (
    <Space orientation="vertical" size="middle" className="w-full">
      <div className="rounded-md border border-gray-200 bg-white px-4 py-3">
        <Text type="secondary">

          Statistical caliber:{scopeText};Time window {windowText}. All cards, distributions and trends on this page share the same query.
does not contain other data sources.
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
        title="Processing Efficiency and SLA"
        size="small"
        extra={<Text type="secondary">SLA readings are calculated based on the full amount of the tenant and do not follow the data range of the login user</Text>}
      >
        {stats.sla ? (
          <Row gutter={[16, 16]}>
            <Col xs={12} lg={6}>
              <div className="flex flex-col items-center">
                <Text type="secondary" className="mb-1">

                  SLA Compliance
                </Text>
                <Progress
                  type="circle"
                  size={72}
                  percent={Math.round(stats.sla.complianceRate)}
                />
                <Text type="secondary" className="mt-1 text-xs">

                  Sample: {stats.sla.sampledTickets} tickets; SLA breaches: {stats.sla.violatedTickets}
                </Text>
              </div>
            </Col>
            <Col xs={12} lg={6}>
              <Statistic
                title="Average response time (hours)"
                value={stats.sla.avgResponseMinutes / 60}
                precision={1}
              />
            </Col>
            <Col xs={12} lg={6}>
              <Statistic
                title="Average resolution time (hours)"
                value={stats.sla.avgResolutionHours}
                precision={1}
              />
            </Col>
            <Col xs={12} lg={6}>
              <Statistic
                title="Resolution rate"
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
                title="SLA compliance rate is temporarily unavailable"
                description="Not connected to the SLA timing service or this reading failed, so &quot;not obtained&quot; is displayed here instead of 0%. Please complete the definition and monitoring configuration in the SLA module first."
              />
            </Col>
            <Col xs={12} lg={6}>
              <Statistic
                title="Resolution rate"
                value={percentage(stats.resolved + stats.closed, stats.total)}
                precision={1}
                suffix="%"
              />
            </Col>
            <Col xs={12} lg={6}>
              <Statistic
                title="Overdue rate"
                value={percentage(stats.overdue, stats.total)}
                precision={1}
                suffix="%"
              />
            </Col>
          </Row>
        )}
      </Card>

      <Card title="Work order trend (newly created/resolved daily)" size="small">
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
                  name === 'created' ? 'Create' : name === 'resolved' ? 'Resolved' : String(name),
                ]}
              />
              <Legend formatter={(v: string) => (v === 'created' ? 'Create' : v === 'resolved' ? 'Resolved' : v)} />
              <Line type="monotone" dataKey="created" stroke="#1890ff" strokeWidth={2} dot={false} />
              <Line type="monotone" dataKey="resolved" stroke="#52c41a" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <Empty
            description={
              stats.dailyTrend === undefined
                ? 'This request did not return a trend sequence (window trend is not enabled on the backend)'
                : 'There are no new or resolved tickets in the selected time range'
            }
          />
        )}
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="Status distribution" size="small">
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
              <Empty description="There is no status distribution data in the selected range." />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="Priority distribution" size="small">
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
                  <Bar dataKey="value" name="Number of work orders" radius={[4, 4, 0, 0]}>
                    {prioritySlices.map((slice) => (
                      <Cell key={slice.key} fill={slice.color} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <Empty description="There is no priority distribution data in the selected range." />
            )}
          </Card>
        </Col>
      </Row>

      <Card title="Status details" size="small">
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
          <Empty description="There are no tickets in the selected range" />
        )}
      </Card>
    </Space>
  );

  return renderShell(
    <Tabs
      activeKey={activeTab}
      onChange={setActiveTab}
      items={[
        { key: 'overview', label: 'Overview', children: overviewTab },
        { key: 'deep', label: 'In-depth analysis', children: <TicketDeepAnalytics /> },
      ]}
    />
  );
};

export default TicketAnalytics;
