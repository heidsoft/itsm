'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { Alert, Button, Card, Col, Empty, Row, Spin, Statistic, Tag, Typography } from 'antd';
import { RotateCcw } from 'lucide-react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { TicketApi } from '@/lib/api/ticket-api';
import type { TicketStatsResponse } from '@/lib/api/ticket-api';
import { TicketPriority, TicketPriorityConfig, TicketStatus, TicketStatusConfig } from '@/constants/taxonomy';

const { Title, Text } = Typography;

// 图表配色按枚举键取，不写死字面量、也不按下标轮转：
// 下标轮转会让「同状态不同租户」换色，也无法区分词表外的历史取值。
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

// 后端已按词表顺序返回且只含真实存在的取值，前端不再二次聚合或补零。
function toStatusSlices(byStatus: TicketStatsResponse['byStatus']): Slice[] {
  return byStatus.map(entry => ({
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
  return byPriority.map(entry => ({
    key: entry.priority,
    name: priorityLabel(entry.priority),
    value: entry.count,
    color:
      (Object.values(TicketPriority) as string[]).includes(entry.priority)
        ? PRIORITY_COLORS[entry.priority as TicketPriority]
        : UNKNOWN_COLOR,
  }));
}

function countOf(slices: Slice[], status: TicketStatus): number {
  return slices.find(slice => slice.key === status)?.value ?? 0;
}

type LoadState =
  | { kind: 'loading' }
  | { kind: 'error'; reason: string }
  | { kind: 'ready'; stats: TicketStatsResponse };

const TicketsReportPage = () => {
  const [state, setState] = useState<LoadState>({ kind: 'loading' });

  const loadData = useCallback(async () => {
    setState({ kind: 'loading' });
    try {
      // 权威来源是 GET /api/v1/tickets/stats 的租户全量分组计数。
      // 修复前这里用 listTickets({pageSize:200}) 在浏览器里自己数：200 页长之外
      // 的工单被静默截断后仍当成全量展示，而且超时卡片过滤 status === 'overdue'
      // ——工单状态机里没有这个取值，那张卡片永远显示 0。
      const stats = await TicketApi.getTicketStats();
      setState({ kind: 'ready', stats });
    } catch (err) {
      setState({ kind: 'error', reason: err instanceof Error ? err.message : '未知错误' });
    }
  }, []);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  if (state.kind === 'loading') {
    return (
      <div className="p-6 bg-gray-50 min-h-full">
        <Title level={2}>工单报表</Title>
        <div className="flex items-center justify-center h-64">
          <Spin description="加载工单统计..." />
        </div>
      </div>
    );
  }

  if (state.kind === 'error') {
    // 失败态不得渲染全 0 卡片或空图表：那会把接口故障伪装成「没有工单」。
    return (
      <div className="p-6 bg-gray-50 min-h-full">
        <Title level={2}>工单报表</Title>
        <Alert
          type="error"
          showIcon
          title="工单统计加载失败"
          description={`${state.reason}。请确认已登录且拥有工单读取权限，或稍后重试。`}
          action={
            <Button size="small" icon={<RotateCcw />} onClick={loadData}>
              重试
            </Button>
          }
        />
      </div>
    );
  }

  const { stats } = state;
  const statusSlices = toStatusSlices(stats.byStatus);
  const prioritySlices = toPrioritySlices(stats.byPriority);
  const total = stats.total;

  const cards = [
    { title: '工单总数', value: total, color: '#1890ff' },
    { title: statusLabel(TicketStatus.NEW), value: countOf(statusSlices, TicketStatus.NEW), color: STATUS_COLORS[TicketStatus.NEW] },
    { title: statusLabel(TicketStatus.OPEN), value: countOf(statusSlices, TicketStatus.OPEN), color: STATUS_COLORS[TicketStatus.OPEN] },
    { title: statusLabel(TicketStatus.IN_PROGRESS), value: countOf(statusSlices, TicketStatus.IN_PROGRESS), color: STATUS_COLORS[TicketStatus.IN_PROGRESS] },
    { title: statusLabel(TicketStatus.PENDING), value: countOf(statusSlices, TicketStatus.PENDING), color: STATUS_COLORS[TicketStatus.PENDING] },
    { title: statusLabel(TicketStatus.RESOLVED), value: countOf(statusSlices, TicketStatus.RESOLVED), color: STATUS_COLORS[TicketStatus.RESOLVED] },
    { title: statusLabel(TicketStatus.CLOSED), value: countOf(statusSlices, TicketStatus.CLOSED), color: STATUS_COLORS[TicketStatus.CLOSED] },
    // 超时来自 sla_states 的权威判定，不是工单状态。
    { title: 'SLA 超时', value: stats.overdue, color: '#ff4d4f' },
  ];

  return (
    <div className="p-6 bg-gray-50 min-h-full space-y-6">
      <header>
        <Title level={2}>工单报表</Title>
        <Text className="text-gray-500">
          统计口径为当前租户全量工单；分布数据来自后端分组计数，不是当前页列表。
        </Text>
      </header>

      <Row justify="end">
        <Button icon={<RotateCcw />} onClick={loadData}>
          刷新数据
        </Button>
      </Row>

      <Row gutter={[16, 16]}>
        {cards.map(card => (
          <Col xs={12} sm={8} lg={6} key={card.title}>
            <Card>
              <Statistic
                title={card.title}
                value={card.value}
                styles={{ content: { color: card.color } }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      {total === 0 ? (
        <Card>
          <Empty description="当前租户还没有工单" />
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}>
            <Card title="工单状态分布">
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={statusSlices}
                    cx="50%"
                    cy="50%"
                    outerRadius={100}
                    dataKey="value"
                    nameKey="name"
                    label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
                  >
                    {statusSlices.map(slice => (
                      <Cell key={slice.key} fill={slice.color} />
                    ))}
                  </Pie>
                  <Tooltip />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </Card>
          </Col>

          <Col xs={24} lg={12}>
            <Card title="工单优先级分布">
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={prioritySlices}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis allowDecimals={false} />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="value" name="工单数量" fill="#1890ff">
                    {prioritySlices.map(slice => (
                      <Cell key={slice.key} fill={slice.color} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </Card>
          </Col>
        </Row>
      )}

      {total > 0 && (
        <Card title="状态明细">
          <Row gutter={[16, 16]}>
            {statusSlices.map(slice => (
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
        </Card>
      )}
    </div>
  );
};

export default TicketsReportPage;
