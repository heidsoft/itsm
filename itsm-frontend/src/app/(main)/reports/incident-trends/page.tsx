'use client';

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Card, Col, DatePicker, Empty, Row, Select, Spin, Statistic, Typography } from 'antd';
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { BarChart3, CheckCircle, Clock, RotateCcw, TrendingUp } from 'lucide-react';
import dayjs from 'dayjs';
import { IncidentAPI } from '@/lib/api/incident-api';
import type { IncidentReport, IncidentStatCount } from '@/lib/api/incident-api';
import {
  IncidentPriority,
  IncidentStatus,
  incidentPriorityLabel,
  incidentStatusLabel,
  isKnownIncidentPriority,
  isKnownIncidentStatus,
} from '@/constants/incident';

const { Title, Text } = Typography;

// 键由枚举计算而来，不再手抄字面量：本文件此前写着 urgent（工单词表），
// 而事件域合法取值是 critical，于是最高优先级分片一直落在默认灰。
const STATUS_COLORS: Record<IncidentStatus, string> = {
  [IncidentStatus.NEW]: '#1890ff',
  [IncidentStatus.ACKNOWLEDGED]: '#096dd9',
  [IncidentStatus.ASSIGNED]: '#13c2c2',
  [IncidentStatus.IN_PROGRESS]: '#faad14',
  [IncidentStatus.TRIAGED]: '#722ed1',
  [IncidentStatus.ESCALATED]: '#eb2f96',
  [IncidentStatus.ON_HOLD]: '#8c8c8c',
  [IncidentStatus.RESOLVED]: '#52c41a',
  [IncidentStatus.CLOSED]: '#d9d9d9',
  [IncidentStatus.CANCELLED]: '#bfbfbf',
};

const PRIORITY_COLORS: Record<IncidentPriority, string> = {
  [IncidentPriority.LOW]: '#52c41a',
  [IncidentPriority.MEDIUM]: '#faad14',
  [IncidentPriority.HIGH]: '#ff4d4f',
  [IncidentPriority.CRITICAL]: '#722ed1',
};

// 词表外的历史取值不猜测颜色，统一灰色并原样显示标签，避免把脏数据伪装成已知状态。
const UNKNOWN_COLOR = '#d9d9d9';

function statusColor(status: string): string {
  return isKnownIncidentStatus(status) ? STATUS_COLORS[status] : UNKNOWN_COLOR;
}

function priorityColor(priority: string): string {
  return isKnownIncidentPriority(priority) ? PRIORITY_COLORS[priority] : UNKNOWN_COLOR;
}

interface Slice {
  key: string;
  name: string;
  value: number;
  color: string;
}

// 后端已按词表顺序返回且只含真实存在的取值，前端不再二次聚合。
function toSlices(
  counts: IncidentStatCount[],
  labelOf: (value: string) => string,
  colorOf: (value: string) => string,
): Slice[] {
  return counts.map(entry => ({
    key: entry.value,
    name: labelOf(entry.value),
    value: entry.count,
    color: colorOf(entry.value),
  }));
}

const PRESET_OPTIONS = [
  { value: '7', label: '最近7天' },
  { value: '30', label: '最近30天' },
  { value: '90', label: '最近90天' },
];

interface WindowRequest {
  dateFrom: string;
  dateTo: string;
}

// 后端窗口两端必须成对提供，因此请求里始终带齐 dateFrom/dateTo。
function presetWindow(days: number): WindowRequest {
  const today = dayjs();
  return {
    dateFrom: today.subtract(days - 1, 'day').format('YYYY-MM-DD'),
    dateTo: today.format('YYYY-MM-DD'),
  };
}

// 平均时长的单位由读数本身决定：修复前把后端的分钟当小时渲染，相差 60 倍。
function formatAvgResolution(report: IncidentReport): string {
  if (report.resolvedInWindow === 0) {
    // 后端对空集合返回 0，这里不能把它显示成「0 分钟」。
    return '窗口内无解决事件';
  }
  const minutes = report.avgResolutionMinutes;
  if (minutes < 60) {
    return `${minutes} 分钟`;
  }
  if (minutes < 60 * 48) {
    return `${(minutes / 60).toFixed(1)} 小时`;
  }
  return `${(minutes / (60 * 24)).toFixed(1)} 天`;
}

// X 轴只保留 MM-DD：90 天窗口下完整日期会互相压字。
function shortDate(value: string): string {
  return value.slice(5);
}

type LoadState =
  | { kind: 'loading' }
  | { kind: 'error'; reason: string }
  | { kind: 'ready'; report: IncidentReport };

const IncidentTrendsPage = () => {
  const [state, setState] = useState<LoadState>({ kind: 'loading' });
  const [windowRequest, setWindowRequest] = useState<WindowRequest>(() => presetWindow(30));
  const [presetDays, setPresetDays] = useState<string | 'custom'>('30');
  // 窗口连续变化时只接受最后一次请求的结果，避免旧响应覆盖新窗口。
  const requestIdRef = useRef(0);

  const loadReport = useCallback(async () => {
    const requestId = requestIdRef.current + 1;
    requestIdRef.current = requestId;
    setState({ kind: 'loading' });
    try {
      const report = await IncidentAPI.getIncidentReport(windowRequest);
      if (requestIdRef.current === requestId) {
        setState({ kind: 'ready', report });
      }
    } catch (err) {
      if (requestIdRef.current === requestId) {
        setState({ kind: 'error', reason: err instanceof Error ? err.message : '未知错误' });
      }
    }
  }, [windowRequest]);

  useEffect(() => {
    void loadReport();
  }, [loadReport]);

  const handlePresetChange = (value: string) => {
    setPresetDays(value);
    setWindowRequest(presetWindow(Number(value)));
  };

  const handleRangeChange = (dates: [dayjs.Dayjs | null, dayjs.Dayjs | null] | null) => {
    if (!dates || !dates[0] || !dates[1]) {
      return;
    }
    setPresetDays('custom');
    setWindowRequest({
      dateFrom: dates[0].format('YYYY-MM-DD'),
      dateTo: dates[1].format('YYYY-MM-DD'),
    });
  };

  const windowSummary =
    state.kind === 'ready'
      ? `统计窗口：${state.report.window.dateFrom} 至 ${state.report.window.dateTo}（${state.report.window.days} 天，日期按服务器时区整日边界）`
      : '统计窗口：事件趋势报表按创建时间与解决时间分别归日';

  if (state.kind === 'loading') {
    return (
      <div className="p-6 bg-gray-50 min-h-full">
        <Title level={2}>事件趋势报表</Title>
        <div className="flex items-center justify-center h-64">
          <Spin size="large" description="加载报表数据..." />
        </div>
      </div>
    );
  }

  if (state.kind === 'error') {
    // 失败态不得渲染全 0 卡片或空图表：那会把接口故障伪装成「窗口内没有事件」。
    return (
      <div className="p-6 bg-gray-50 min-h-full space-y-6">
        <Title level={2}>事件趋势报表</Title>
        <Alert
          type="error"
          showIcon
          title="事件趋势数据加载失败"
          description={`${state.reason}。请确认已登录且拥有事件读取权限，或调整窗口后重试。`}
          action={
            <Button size="small" icon={<RotateCcw />} onClick={loadReport}>
              重试
            </Button>
          }
        />
      </div>
    );
  }

  const { report } = state;
  const statusSlices = toSlices(report.byStatus, incidentStatusLabel, statusColor);
  const prioritySlices = toSlices(report.byPriority, incidentPriorityLabel, priorityColor);
  const isEmptyWindow = report.createdInWindow === 0 && report.resolvedInWindow === 0;

  const cards = [
    {
      title: '窗口内新建',
      value: report.createdInWindow,
      color: '#1890ff',
      icon: <TrendingUp size={20} style={{ color: '#1890ff' }} />,
    },
    {
      title: '窗口内解决',
      value: report.resolvedInWindow,
      color: '#52c41a',
      icon: <CheckCircle size={20} style={{ color: '#52c41a' }} />,
    },
    {
      title: '平均解决时长',
      value: formatAvgResolution(report),
      color: '#faad14',
      icon: <Clock size={20} style={{ color: '#faad14' }} />,
    },
    {
      title: '统计天数',
      value: report.window.days,
      color: '#722ed1',
      icon: <BarChart3 size={20} style={{ color: '#722ed1' }} />,
    },
  ];

  return (
    <div className="p-6 bg-gray-50 min-h-full space-y-6">
      <header>
        <Title level={2}>事件趋势报表</Title>
        <Text className="text-gray-500">{windowSummary}</Text>
      </header>

      <Card>
        <Row justify="space-between" align="middle" gutter={[16, 16]}>
          <Col>
            <div className="flex items-center gap-3">
              <Select
                value={presetDays}
                onChange={handlePresetChange}
                style={{ width: 140 }}
                options={[...PRESET_OPTIONS, { value: 'custom', label: '自定义' }]}
                aria-label="统计窗口"
              />
              <DatePicker.RangePicker
                value={[dayjs(windowRequest.dateFrom), dayjs(windowRequest.dateTo)]}
                onChange={handleRangeChange}
                allowClear={false}
              />
            </div>
          </Col>
          <Col>
            <Button icon={<RotateCcw />} onClick={loadReport}>
              刷新数据
            </Button>
          </Col>
        </Row>
      </Card>

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

      {isEmptyWindow ? (
        <Card>
          <Empty description="所选窗口内没有事件记录" />
        </Card>
      ) : (
        <>
          <Row gutter={[16, 16]}>
            <Col span={24}>
              <Card title="每日事件量趋势">
                <ResponsiveContainer width="100%" height={350}>
                  <AreaChart data={report.dailyTrend}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" tickFormatter={shortDate} minTickGap={16} />
                    <YAxis allowDecimals={false} />
                    <Tooltip />
                    <Legend />
                    {/* 两条曲线各自独立：created 按创建时间归日、resolved 按解决时间归日，
                        堆叠会把两者加成一条无意义的合计曲线。 */}
                    <Area
                      type="monotone"
                      dataKey="created"
                      stroke="#1890ff"
                      fill="#1890ff"
                      fillOpacity={0.3}
                      name="新建事件"
                    />
                    <Area
                      type="monotone"
                      dataKey="resolved"
                      stroke="#52c41a"
                      fill="#52c41a"
                      fillOpacity={0.3}
                      name="解决事件"
                    />
                  </AreaChart>
                </ResponsiveContainer>
                <Text type="secondary" className="mt-2 block">
                  窗口内每一天都会出现（无事件为 0）；新建按创建时间归日，解决按解决时间归日，
                  两者是不同口径，逐日不必相等。
                </Text>
              </Card>
            </Col>
          </Row>

          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <Card title="按优先级分布（窗口内新建）">
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={prioritySlices}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis allowDecimals={false} />
                    <Tooltip />
                    <Legend />
                    <Bar dataKey="value" name="事件数量" fill="#1890ff">
                      {prioritySlices.map(slice => (
                        <Cell key={slice.key} fill={slice.color} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card title="按状态分布（窗口内新建）">
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={statusSlices}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis allowDecimals={false} />
                    <Tooltip />
                    <Legend />
                    <Bar dataKey="value" name="事件数量" fill="#1890ff">
                      {statusSlices.map(slice => (
                        <Cell key={slice.key} fill={slice.color} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
                <Text type="secondary" className="mt-2 block">
                  状态分布反映窗口内新建事件的当前状态，之和恒等于「窗口内新建」。
                </Text>
              </Card>
            </Col>
          </Row>
        </>
      )}
    </div>
  );
};

export default IncidentTrendsPage;
