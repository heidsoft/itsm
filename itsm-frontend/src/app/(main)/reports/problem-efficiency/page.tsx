'use client';

import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Empty,
  Progress,
  Row,
  Spin,
  Statistic,
  Tag,
  Typography,
} from 'antd';
import { AlertTriangle, CheckCircle, Clock, RotateCcw, XCircle } from 'lucide-react';
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
import { ProblemApi } from '@/lib/api/problem-api';
import type { Problem, ProblemStatsResponse } from '@/lib/api/problem-api';
import {
  ProblemPriority,
  ProblemStatus,
  isKnownProblemPriority,
  isKnownProblemStatus,
  problemPriorityLabel,
  problemStatusLabel,
} from '@/constants/problem';

const { Title, Text } = Typography;

// 键用枚举计算而来，不再手抄字面量：此前这里写着 inProgress，
// 后端存量值是 in_progress，该状态的饼图分片因此一直是默认灰。
const STATUS_COLORS: Record<ProblemStatus, string> = {
  [ProblemStatus.OPEN]: '#1890ff',
  [ProblemStatus.INVESTIGATING]: '#722ed1',
  [ProblemStatus.IDENTIFIED]: '#fa8c16',
  [ProblemStatus.IN_PROGRESS]: '#faad14',
  [ProblemStatus.RESOLVED]: '#52c41a',
  [ProblemStatus.CLOSED]: '#d9d9d9',
};

const PRIORITY_COLORS: Record<ProblemPriority, string> = {
  [ProblemPriority.LOW]: '#52c41a',
  [ProblemPriority.MEDIUM]: '#faad14',
  [ProblemPriority.HIGH]: '#ff4d4f',
  [ProblemPriority.CRITICAL]: '#722ed1',
};

// 词表外的历史取值不猜测颜色，统一灰色并原样显示标签，避免把脏数据伪装成已知状态。
const UNKNOWN_COLOR = '#d9d9d9';

function statusColor(status: string): string {
  return isKnownProblemStatus(status) ? STATUS_COLORS[status] : UNKNOWN_COLOR;
}

function priorityColor(priority: string): string {
  return isKnownProblemPriority(priority) ? PRIORITY_COLORS[priority] : UNKNOWN_COLOR;
}

interface Slice {
  key: string;
  name: string;
  value: number;
  color: string;
}

// 后端已按词表顺序返回且只含真实存在的取值，前端不再二次聚合。
function toStatusSlices(stats: ProblemStatsResponse): Slice[] {
  return stats.byStatus.map(entry => ({
    key: entry.status,
    name: problemStatusLabel(entry.status),
    value: entry.count,
    color: statusColor(entry.status),
  }));
}

function toPrioritySlices(stats: ProblemStatsResponse): Slice[] {
  return stats.byPriority.map(entry => ({
    key: entry.priority,
    name: problemPriorityLabel(entry.priority),
    value: entry.count,
    color: priorityColor(entry.priority),
  }));
}

const STATUS_TAG_COLORS: Record<ProblemStatus, string> = {
  [ProblemStatus.OPEN]: 'processing',
  [ProblemStatus.INVESTIGATING]: 'processing',
  [ProblemStatus.IDENTIFIED]: 'warning',
  [ProblemStatus.IN_PROGRESS]: 'processing',
  [ProblemStatus.RESOLVED]: 'success',
  [ProblemStatus.CLOSED]: 'default',
};

const PRIORITY_TAG_COLORS: Record<ProblemPriority, string> = {
  [ProblemPriority.LOW]: 'green',
  [ProblemPriority.MEDIUM]: 'orange',
  [ProblemPriority.HIGH]: 'red',
  [ProblemPriority.CRITICAL]: 'red',
};

function statusTagColor(status: string): string {
  return isKnownProblemStatus(status) ? STATUS_TAG_COLORS[status] : 'default';
}

function priorityTagColor(priority: string): string {
  return isKnownProblemPriority(priority) ? PRIORITY_TAG_COLORS[priority] : 'default';
}

type LoadState =
  | { kind: 'loading' }
  | { kind: 'error'; reason: string }
  | { kind: 'ready'; stats: ProblemStatsResponse; recent: Problem[] };

const RECENT_PAGE_SIZE = 10;

const ProblemEfficiencyPage = () => {
  const [state, setState] = useState<LoadState>({ kind: 'loading' });

  const loadData = useCallback(async () => {
    setState({ kind: 'loading' });
    try {
      // 分布的权威来源是 GET /api/v1/problems/stats 的租户全量分组计数。
      // 修复前这里拉 listProblems({page:1,pageSize:100}) 在浏览器里自己数：第 100 条
      // 之后的问题被静默截断后仍当成全量读数，而 identified 从来不在任何单值桶里，
      // 页面因此无法自证总数对得上。列表只用于「最新问题列表」，按后端倒序取一页即可。
      const [stats, recent] = await Promise.all([
        ProblemApi.getProblemStats(),
        ProblemApi.getProblems({ page: 1, pageSize: RECENT_PAGE_SIZE }),
      ]);
      setState({ kind: 'ready', stats, recent: recent.items });
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
        <Title level={2}>问题管理效率报表</Title>
        <div className="flex items-center justify-center h-64">
          <Spin size="large" description="加载报表数据..." />
        </div>
      </div>
    );
  }

  if (state.kind === 'error') {
    // 失败态不得渲染全 0 卡片或空图表：那会把接口故障伪装成「没有问题记录」。
    return (
      <div className="p-6 bg-gray-50 min-h-full">
        <Title level={2}>问题管理效率报表</Title>
        <Alert
          type="error"
          showIcon
          title="问题统计加载失败"
          description={`${state.reason}。请确认已登录且拥有问题读取权限，或稍后重试。`}
          action={
            <Button size="small" icon={<RotateCcw />} onClick={loadData}>
              重试
            </Button>
          }
        />
      </div>
    );
  }

  const { stats, recent } = state;
  const statusSlices = toStatusSlices(stats);
  const prioritySlices = toPrioritySlices(stats);
  const total = stats.total;

  const cards = [
    {
      title: '问题总数',
      value: total,
      color: '#1890ff',
      icon: <AlertTriangle size={20} style={{ color: '#1890ff' }} />,
    },
    {
      title: '已解决问题',
      value: stats.resolved,
      color: '#52c41a',
      icon: <CheckCircle size={20} />,
    },
    {
      // 折叠口径：investigating 与 in_progress 同进本卡片，与后端 stats 一致。
      title: problemStatusLabel(ProblemStatus.IN_PROGRESS),
      value: stats.inProgress,
      color: '#faad14',
      icon: <Clock size={20} />,
    },
    {
      // 折叠口径：high 与 critical 同进本卡片。
      title: '高优先级',
      value: stats.highPriority,
      color: '#ff4d4f',
      icon: <XCircle size={20} />,
    },
  ];

  // 占比的分母是租户全量问题数，total 为 0 时整块效率指标已被空态替换，
  // 因此这里只在有分母时计算，修复前的 (0/0)*100 会把 NaN 写成读数。
  const resolutionPercent = (stats.resolved / total) * 100;
  const metrics = [
    {
      title: '解决率',
      label: '已解决',
      part: stats.resolved,
      percent: resolutionPercent,
      // 解决率低于 70% 用警示色，与修复前的判定阈值保持一致。
      color: resolutionPercent < 70 ? '#faad14' : '#52c41a',
    },
    {
      title: '处理中比例',
      label: problemStatusLabel(ProblemStatus.IN_PROGRESS),
      part: stats.inProgress,
      percent: (stats.inProgress / total) * 100,
      color: '#1890ff',
    },
    {
      title: '高优先级占比',
      label: '高优先级',
      part: stats.highPriority,
      percent: (stats.highPriority / total) * 100,
      color: '#ff4d4f',
    },
  ];

  return (
    <div className="p-6 bg-gray-50 min-h-full space-y-6">
      <header>
        <Title level={2}>问题管理效率报表</Title>
        <Text className="text-gray-500">
          统计口径为当前租户全量问题记录；分布数据来自后端分组计数，不是当前页列表。
        </Text>
      </header>

      <Row justify="end">
        <Button icon={<RotateCcw />} onClick={loadData}>
          刷新数据
        </Button>
      </Row>

      <Row gutter={[16, 16]}>
        {cards.map(card => (
          <Col xs={24} sm={12} lg={6} key={card.title}>
            <Card>
              <Statistic
                title={card.title}
                value={card.value}
                prefix={card.icon}
                styles={{ content: { color: card.color } }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      {total === 0 ? (
        <Card>
          <Empty description="当前租户还没有问题记录" />
        </Card>
      ) : (
        <>
          <Row gutter={[16, 16]}>
            {metrics.map(metric => (
              <Col xs={24} lg={8} key={metric.title}>
                <Card title={metric.title}>
                  <div className="text-center py-4">
                    <div className="text-4xl font-bold mb-2" style={{ color: metric.color }}>
                      {metric.percent.toFixed(1)}%
                    </div>
                    <Progress
                      percent={metric.percent}
                      strokeColor={metric.color}
                      showInfo={false}
                    />
                    <Text type="secondary" className="mt-2 block">
                      {metric.label} {metric.part} / 总数 {total}
                    </Text>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>

          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <Card title="问题状态分布">
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
              <Card title="问题优先级分布">
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={prioritySlices}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis allowDecimals={false} />
                    <Tooltip />
                    <Legend />
                    <Bar dataKey="value" name="问题数量" fill="#1890ff">
                      {prioritySlices.map(slice => (
                        <Cell key={slice.key} fill={slice.color} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </Card>
            </Col>
          </Row>

          {recent.length > 0 && (
            <Card title="最新问题列表">
              {/* antd v6 已弃用 List（运行时告警），这里用与本表其余部分一致的
                  Tailwind 结构渲染，行为与视觉不变。 */}
              <div className="flex flex-col">
                {recent.map(problem => (
                  <div
                    key={problem.id}
                    className="flex flex-col gap-2 border-b border-gray-100 py-3 last:border-b-0 md:flex-row md:items-center md:justify-between"
                  >
                    <div className="flex items-center gap-3">
                      <span className="font-medium text-blue-600">#{problem.id}</span>
                      <span className="font-medium">{problem.title}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Tag color={statusTagColor(problem.status)}>
                        {problemStatusLabel(problem.status)}
                      </Tag>
                      <Tag color={priorityTagColor(problem.priority)}>
                        {problemPriorityLabel(problem.priority)}
                      </Tag>
                    </div>
                    <div className="flex items-center gap-4 text-sm text-gray-500">
                      <span>处理人: {problem.assigneeName ?? '未分配'}</span>
                      <span>创建时间: {new Date(problem.createdAt).toLocaleDateString()}</span>
                    </div>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </>
      )}
    </div>
  );
};

export default ProblemEfficiencyPage;
