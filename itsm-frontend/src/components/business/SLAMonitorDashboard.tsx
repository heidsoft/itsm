'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  App,
  Badge,
  Button,
  Card,
  Col,
  Empty,
  Progress,
  Row,
  Select,
  Space,
  Spin,
  Statistic,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  Activity,
  AlertTriangle,
  Bell,
  CheckCircle,
  Clock,
  Maximize,
  Minimize,
  RotateCcw,
  Target,
  XCircle,
  Zap,
} from 'lucide-react';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import SLAApi, {
  type SLAAlertItem,
  type SLAMonitoringData,
  type SLAPerformanceRow,
  type SLATimeRemaining,
} from '@/lib/api/sla-api';

const { Title, Text } = Typography;

// 统计窗口。监控接口用 startTime/endTime（请求体），绩效接口用 startDate/endDate
// （查询参数），两者共用同一对 RFC3339 边界，保证大屏上下数字口径一致。
type WindowKey = '24h' | '7d' | '30d';

const WINDOWS: Record<WindowKey, { label: string; milliseconds: number }> = {
  '24h': { label: '最近 24 小时', milliseconds: 24 * 60 * 60 * 1000 },
  '7d': { label: '最近 7 天', milliseconds: 7 * 24 * 60 * 60 * 1000 },
  '30d': { label: '最近 30 天', milliseconds: 30 * 24 * 60 * 60 * 1000 },
};

// 分组行数量级远小于工单数；后端 pageSize 上限是 200。
const PERFORMANCE_PAGE_SIZE = 200;

const PRIORITY_LABELS: Record<string, string> = {
  critical: '紧急',
  urgent: '紧急',
  high: '高',
  medium: '中',
  low: '低',
};

const PRIORITY_COLORS: Record<string, string> = {
  critical: 'red',
  urgent: 'red',
  high: 'orange',
  medium: 'blue',
  low: 'default',
};

const SERVICE_TYPE_LABELS: Record<string, string> = {
  incident: '事件',
  problem: '问题',
  change: '变更',
  release: '发布',
  service_request: '服务请求',
  // 后端用固定 key 表示「工单未绑定 SLA 定义」，这部分工单不得被丢弃
  unassigned: '未绑定 SLA',
};

const ALERT_LEVEL_LABELS: Record<string, string> = {
  warning: '警告',
  severe: '严重',
  critical: '紧急',
};

const PRIORITY_FILTER_OPTIONS = Object.entries(PRIORITY_LABELS)
  .filter(([key]) => key !== 'urgent')
  .map(([value, label]) => ({ value, label }));

function resolveWindow(key: WindowKey): { startTime: string; endTime: string } {
  const end = new Date();
  const start = new Date(end.getTime() - WINDOWS[key].milliseconds);
  return { startTime: start.toISOString(), endTime: end.toISOString() };
}

/** 比率由后端算好（0-100，一位小数），前端只负责补上百分号 */
function formatRate(value: number): string {
  return `${(Number.isFinite(value) ? value : 0).toFixed(1)}%`;
}

/** 达成率必须带样本数；样本为 0 是「暂无数据」，不是 0% 合规 */
function renderAchievement(rate: number, samples: number): React.ReactNode {
  if (!samples) {
    return <Text type="secondary">暂无样本</Text>;
  }
  return <Text>{formatRate(rate)}</Text>;
}

function formatDuration(minutes: number): string {
  if (!Number.isFinite(minutes) || minutes <= 0) {
    return '0 分钟';
  }
  if (minutes < 60) {
    return `${minutes.toFixed(1)} 分钟`;
  }
  if (minutes < 60 * 24) {
    return `${(minutes / 60).toFixed(1)} 小时`;
  }
  return `${(minutes / (60 * 24)).toFixed(1)} 天`;
}

function formatDateTime(value?: string): string {
  if (!value) {
    return '—';
  }
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime())
    ? value
    : format(parsed, 'yyyy-MM-dd HH:mm', { locale: zhCN });
}

/** 剩余时间来自后端绑定的解决截止时间；无截止时间时显示占位而不是 0 小时 */
function renderTimeRemaining(value?: SLATimeRemaining): React.ReactNode {
  if (!value) {
    return <Text type="secondary">—</Text>;
  }
  const absolute = Math.abs(value.hours);
  let hours = Math.floor(absolute);
  let minutes = Math.round((absolute - hours) * 60);
  if (minutes === 60) {
    hours += 1;
    minutes = 0;
  }
  const label = `${hours}小时${minutes}分钟`;
  const overdue = value.hours < 0;
  const color = overdue ? '#ff4d4f' : value.hours < 2 ? '#faad14' : '#52c41a';
  return (
    <Tooltip title={`截止时间 ${formatDateTime(value.deadline)}`}>
      <Text strong style={{ color }}>
        {overdue ? `已超时 ${label}` : `剩余 ${label}`}
      </Text>
    </Tooltip>
  );
}

interface SLAMonitorDashboardProps {
  autoRefresh?: boolean;
  refreshInterval?: number; // 秒
  onFullscreen?: (isFullscreen: boolean) => void;
}

/**
 * 绩效表格列定义。两个维度共用同一行契约，只有首列标题和字典不同。
 * 故意不加 sorter：排序在后端按全量分组完成，前端对当前页二次排序会冒充全量结果。
 */
function buildPerformanceColumns(
  firstTitle: string,
  labelOf: (key: string) => string,
): ColumnsType<SLAPerformanceRow> {
  const withSamples = (samples: number, cell: React.ReactNode) =>
    samples ? cell : <Text type="secondary">暂无样本</Text>;

  return [
    {
      title: firstTitle,
      dataIndex: 'key',
      key: 'key',
      width: 140,
      render: (key: string) => labelOf(key),
    },
    {
      title: '工单数',
      dataIndex: 'totalTickets',
      key: 'totalTickets',
      width: 90,
      render: (value: number) => value ?? 0,
    },
    {
      title: '已解决',
      dataIndex: 'resolvedTickets',
      key: 'resolvedTickets',
      width: 90,
      render: (value: number) => value ?? 0,
    },
    {
      title: '解决率',
      dataIndex: 'resolutionRate',
      key: 'resolutionRate',
      width: 100,
      render: (value: number, record) => withSamples(record.totalTickets, formatRate(value)),
    },
    {
      title: '违规工单',
      dataIndex: 'violatedTickets',
      key: 'violatedTickets',
      width: 100,
      render: (value: number) => (
        <span className={value > 0 ? 'text-red-600' : ''}>{value ?? 0}</span>
      ),
    },
    {
      title: 'SLA 合规率',
      dataIndex: 'complianceRate',
      key: 'complianceRate',
      width: 120,
      render: (value: number, record) => withSamples(record.totalTickets, formatRate(value)),
    },
    {
      title: '响应达成率',
      dataIndex: 'responseAchievementRate',
      key: 'responseAchievementRate',
      width: 130,
      render: (value: number, record) => (
        <Tooltip title={`样本 ${record.responseSamples}`}>
          <span>{withSamples(record.responseSamples, formatRate(value))}</span>
        </Tooltip>
      ),
    },
    {
      title: '解决达成率',
      dataIndex: 'resolutionAchievementRate',
      key: 'resolutionAchievementRate',
      width: 130,
      render: (value: number, record) => (
        <Tooltip title={`样本 ${record.resolutionSamples}`}>
          <span>{withSamples(record.resolutionSamples, formatRate(value))}</span>
        </Tooltip>
      ),
    },
    {
      title: '平均响应时长',
      dataIndex: 'averageResponseMinutes',
      key: 'averageResponseMinutes',
      width: 140,
      render: (value: number, record) =>
        record.responseSamples ? formatDuration(value) : <Text type="secondary">—</Text>,
    },
    {
      title: '平均解决时长',
      dataIndex: 'averageResolutionMinutes',
      key: 'averageResolutionMinutes',
      width: 140,
      render: (value: number, record) =>
        record.resolutionSamples ? formatDuration(value) : <Text type="secondary">—</Text>,
    },
  ];
}

export const SLAMonitorDashboard: React.FC<SLAMonitorDashboardProps> = ({
  autoRefresh = true,
  refreshInterval = 30,
  onFullscreen,
}) => {
  const { message: antMessage } = App.useApp();
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [monitoring, setMonitoring] = useState<SLAMonitoringData | null>(null);
  const [serviceTypeRows, setServiceTypeRows] = useState<SLAPerformanceRow[]>([]);
  const [priorityRows, setPriorityRows] = useState<SLAPerformanceRow[]>([]);
  const [serviceTypeTotal, setServiceTypeTotal] = useState(0);
  const [priorityTotal, setPriorityTotal] = useState(0);
  const [windowKey, setWindowKey] = useState<WindowKey>('24h');
  const [serviceTypeFilter, setServiceTypeFilter] = useState<string | undefined>();
  const [priorityFilter, setPriorityFilter] = useState<string | undefined>();
  const [knownServiceTypes, setKnownServiceTypes] = useState<string[]>([]);
  const [updatedAt, setUpdatedAt] = useState<Date | null>(null);

  const loadData = useCallback(async () => {
    const { startTime, endTime } = resolveWindow(windowKey);
    setLoading(true);
    try {
      const [monitoringData, serviceTypeResult, priorityResult] = await Promise.all([
        SLAApi.getSLAMonitoring({ startTime, endTime }),
        SLAApi.getSLAPerformance({
          dimension: 'serviceType',
          startDate: startTime,
          endDate: endTime,
          serviceType: serviceTypeFilter,
          priority: priorityFilter,
          page: 1,
          pageSize: PERFORMANCE_PAGE_SIZE,
        }),
        SLAApi.getSLAPerformance({
          dimension: 'priority',
          startDate: startTime,
          endDate: endTime,
          serviceType: serviceTypeFilter,
          priority: priorityFilter,
          page: 1,
          pageSize: PERFORMANCE_PAGE_SIZE,
        }),
      ]);

      setMonitoring(monitoringData);
      setServiceTypeRows(serviceTypeResult.items);
      setPriorityRows(priorityResult.items);
      setServiceTypeTotal(serviceTypeResult.total);
      setPriorityTotal(priorityResult.total);
      // 过滤生效后结果集会变小，只保留无过滤时读到的服务类型作为下拉选项。
      if (!serviceTypeFilter && !priorityFilter) {
        setKnownServiceTypes(serviceTypeResult.items.map(row => row.key));
      }
      setUpdatedAt(new Date());
      setLoadError(null);
    } catch (error) {
      // 保留上一次成功数据并标注其时间，同时显式暴露失败原因，不静默清空或伪造。
      setLoadError(error instanceof Error ? error.message : '加载SLA数据失败');
    } finally {
      setLoading(false);
    }
  }, [windowKey, serviceTypeFilter, priorityFilter]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  // 自动刷新
  useEffect(() => {
    if (!autoRefresh) {
      return;
    }
    const timer = setInterval(() => {
      void loadData();
    }, refreshInterval * 1000);
    return () => clearInterval(timer);
  }, [autoRefresh, refreshInterval, loadData]);

  const handleFullscreen = useCallback(() => {
    const element = document.documentElement;
    if (!isFullscreen) {
      element.requestFullscreen?.();
      setIsFullscreen(true);
      onFullscreen?.(true);
    } else {
      document.exitFullscreen?.();
      setIsFullscreen(false);
      onFullscreen?.(false);
    }
  }, [isFullscreen, onFullscreen]);

  const handleRefresh = useCallback(() => {
    void loadData();
    antMessage.success('数据已刷新');
  }, [loadData, antMessage]);

  const getComplianceColor = (rate: number) => {
    if (rate >= 95) return '#52c41a';
    if (rate >= 85) return '#faad14';
    return '#ff4d4f';
  };

  const serviceTypeOptions = useMemo(
    () => knownServiceTypes.map(key => ({ value: key, label: SERVICE_TYPE_LABELS[key] || key })),
    [knownServiceTypes],
  );

  const alertColumns: ColumnsType<SLAAlertItem> = useMemo(
    () => [
      {
        title: '工单编号',
        dataIndex: 'ticketNumber',
        key: 'ticketNumber',
        width: 160,
        render: (text: string, record) => (
          <Text strong style={{ color: '#1890ff' }}>{text || `#${record.ticketId}`}</Text>
        ),
      },
      {
        title: '工单标题',
        dataIndex: 'ticketTitle',
        key: 'ticketTitle',
        ellipsis: true,
        // 后端告警历史自带工单标题；缺失时回显工单号，不再伪造 "Ticket #x"
        render: (text: string, record) => text || `#${record.ticketId}`,
      },
      {
        title: '优先级',
        dataIndex: 'priority',
        key: 'priority',
        width: 100,
        render: (priority: string) => (
          <Tag color={PRIORITY_COLORS[priority] || 'default'}>
            {PRIORITY_LABELS[priority] || priority || '—'}
          </Tag>
        ),
      },
      {
        title: '告警级别',
        dataIndex: 'alertLevel',
        key: 'alertLevel',
        width: 110,
        render: (level: string) => (
          <Badge
            status={level === 'critical' ? 'error' : 'warning'}
            text={
              <Tag color={level === 'critical' ? 'red' : 'orange'}>
                {ALERT_LEVEL_LABELS[level] || level || '—'}
              </Tag>
            }
          />
        ),
      },
      {
        title: '告警规则',
        dataIndex: 'alertRuleName',
        key: 'alertRuleName',
        width: 180,
        ellipsis: true,
        render: (name: string, record) => (
          <Tooltip title={`阈值 ${record.thresholdPercentage}% · 实际 ${record.actualPercentage.toFixed(1)}%`}>
            {name || '—'}
          </Tooltip>
        ),
      },
      {
        title: '剩余时间',
        dataIndex: 'timeRemaining',
        key: 'timeRemaining',
        width: 150,
        render: (value?: SLATimeRemaining) => renderTimeRemaining(value),
      },
      {
        title: '触发时间',
        dataIndex: 'createdAt',
        key: 'createdAt',
        width: 160,
        render: (value: string) => formatDateTime(value),
      },
    ],
    [],
  );

  const serviceTypeColumns = useMemo(
    () => buildPerformanceColumns('服务类型', key => SERVICE_TYPE_LABELS[key] || key),
    [],
  );
  const priorityColumns = useMemo(
    () => buildPerformanceColumns('优先级', key => PRIORITY_LABELS[key] || key),
    [],
  );

  const cardStyle: React.CSSProperties = {
    height: '100%',
    backgroundColor: isFullscreen ? 'rgba(255,255,255,0.05)' : '#fff',
    border: isFullscreen ? '1px solid rgba(255,255,255,0.1)' : undefined,
  };

  const textStyle: React.CSSProperties = {
    color: isFullscreen ? 'rgba(255,255,255,0.8)' : undefined,
    fontSize: isFullscreen ? '18px' : '14px',
  };

  const footerStyle: React.CSSProperties = {
    color: isFullscreen ? 'rgba(255,255,255,0.7)' : undefined,
    fontSize: isFullscreen ? '16px' : '14px',
  };

  if (loading && !monitoring) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spin size="large" description="加载SLA监控数据..." />
      </div>
    );
  }

  return (
    <div
      className={`sla-monitor-dashboard ${isFullscreen ? 'fullscreen' : ''}`}
      style={{
        minHeight: isFullscreen ? '100vh' : 'auto',
        padding: isFullscreen ? '24px' : '16px',
        backgroundColor: isFullscreen ? '#0a0e27' : '#f0f2f5',
      }}
    >
      {/* 顶部工具栏：统计周期 + 绩效过滤条件 */}
      <div
        className="flex items-center justify-between mb-6"
        style={{
          backgroundColor: isFullscreen ? 'rgba(255,255,255,0.1)' : 'transparent',
          padding: '12px 16px',
          borderRadius: '8px',
        }}
      >
        <div className="flex flex-col gap-1">
          <Title
            level={2}
            style={{
              color: isFullscreen ? '#fff' : '#000',
              margin: 0,
              fontSize: isFullscreen ? '32px' : '24px',
            }}
          >
            <Activity className="inline-block mr-2" size={isFullscreen ? 32 : 24} />
            SLA实时监控大屏
          </Title>
          <Text
            style={{
              color: isFullscreen ? 'rgba(255,255,255,0.7)' : undefined,
              fontSize: isFullscreen ? '16px' : '14px',
            }}
          >
            {monitoring
              ? `统计周期 ${formatDateTime(monitoring.startTime)} ~ ${formatDateTime(monitoring.endTime)} · 最后更新 ${updatedAt ? format(updatedAt, 'HH:mm:ss') : '—'}`
              : '尚无成功加载的数据'}
          </Text>
        </div>
        <Space wrap size="small">
          <Select
            value={windowKey}
            onChange={setWindowKey}
            style={{ width: 140 }}
            aria-label="统计周期"
            options={(Object.keys(WINDOWS) as WindowKey[]).map(key => ({
              value: key,
              label: WINDOWS[key].label,
            }))}
          />
          <Select
            value={serviceTypeFilter}
            onChange={value => setServiceTypeFilter(value)}
            placeholder="全部服务类型"
            allowClear
            style={{ width: 160 }}
            aria-label="绩效过滤：服务类型"
            options={serviceTypeOptions}
          />
          <Select
            value={priorityFilter}
            onChange={value => setPriorityFilter(value)}
            placeholder="全部优先级"
            allowClear
            style={{ width: 130 }}
            aria-label="绩效过滤：优先级"
            options={PRIORITY_FILTER_OPTIONS}
          />
          <Button
            icon={<RotateCcw />}
            onClick={handleRefresh}
            loading={loading}
            size={isFullscreen ? 'large' : 'middle'}
            style={{
              backgroundColor: isFullscreen ? 'rgba(255,255,255,0.1)' : undefined,
              color: isFullscreen ? '#fff' : undefined,
              borderColor: isFullscreen ? 'rgba(255,255,255,0.3)' : undefined,
            }}
          >
            刷新
          </Button>
          <Button
            icon={isFullscreen ? <Minimize /> : <Maximize />}
            onClick={handleFullscreen}
            size={isFullscreen ? 'large' : 'middle'}
            style={{
              backgroundColor: isFullscreen ? 'rgba(255,255,255,0.1)' : undefined,
              color: isFullscreen ? '#fff' : undefined,
              borderColor: isFullscreen ? 'rgba(255,255,255,0.3)' : undefined,
            }}
          >
            {isFullscreen ? '退出全屏' : '全屏'}
          </Button>
        </Space>
      </div>

      {loadError ? (
        <Alert
          type="error"
          showIcon
          className="mb-6"
          message={`数据加载失败：${loadError}`}
          description={
            monitoring
              ? `当前展示的是 ${updatedAt ? format(updatedAt, 'yyyy-MM-dd HH:mm:ss') : '未知时间'} 成功加载的数据。`
              : '请重试刷新，或确认后端 SLA 监控接口是否可用。'
          }
          action={
            <Button size="small" onClick={handleRefresh} loading={loading}>
              重试
            </Button>
          }
        />
      ) : null}

      {monitoring?.truncated ? (
        <Alert
          type="warning"
          showIcon
          className="mb-6"
          message="窗口内工单数超过扫描上限，以下指标基于部分样本计算，不代表全量结果。"
        />
      ) : null}

      {/* 关键指标：四张卡片结构一致并撑满列高，保证视觉等高 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="sla-stat-card" style={cardStyle}>
            <Statistic
              title={<Text style={textStyle}>工单解决率</Text>}
              value={monitoring?.resolutionRate ?? 0}
              precision={1}
              suffix="%"
              styles={{
                content: {
                  color: isFullscreen ? '#fff' : getComplianceColor(monitoring?.resolutionRate ?? 0),
                  fontSize: isFullscreen ? '44px' : '30px',
                  fontWeight: 'bold',
                },
              }}
              prefix={<CheckCircle />}
            />
            <div className="mt-4">
              <Text style={footerStyle}>
                已解决 {monitoring?.resolvedTickets ?? 0} / {monitoring?.totalTickets ?? 0} 单
              </Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="sla-stat-card" style={cardStyle}>
            <Statistic
              title={<Text style={textStyle}>SLA 合规率</Text>}
              value={monitoring?.complianceRate ?? 0}
              precision={1}
              suffix="%"
              styles={{
                content: {
                  color: isFullscreen ? '#fff' : getComplianceColor(monitoring?.complianceRate ?? 0),
                  fontSize: isFullscreen ? '44px' : '30px',
                  fontWeight: 'bold',
                },
              }}
              prefix={<Target />}
            />
            <div className="mt-4">
              <Text style={footerStyle}>
                达标 {monitoring?.metSlaTickets ?? 0} · 违规 {monitoring?.violatedTickets ?? 0}（
                {formatRate(monitoring?.violationRate ?? 0)}）
              </Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="sla-stat-card" style={cardStyle}>
            <Statistic
              title={<Text style={textStyle}>活跃告警</Text>}
              value={monitoring?.activeAlerts ?? 0}
              styles={{
                content: {
                  color: (monitoring?.activeAlerts ?? 0) > 0 ? '#faad14' : '#52c41a',
                  fontSize: isFullscreen ? '44px' : '30px',
                  fontWeight: 'bold',
                },
              }}
              prefix={<Bell />}
            />
            <div className="mt-4">
              <Text style={footerStyle}>
                未解决违规 {monitoring?.activeViolations ?? 0} 条 · 告警规则{' '}
                {monitoring?.activeAlertRules ?? 0} 条
              </Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="sla-stat-card" style={cardStyle}>
            <Statistic
              title={<Text style={textStyle}>风险工单</Text>}
              value={monitoring?.atRiskTickets ?? 0}
              styles={{
                content: {
                  color: '#ff4d4f',
                  fontSize: isFullscreen ? '44px' : '30px',
                  fontWeight: 'bold',
                },
              }}
              prefix={<AlertTriangle />}
            />
            <div className="mt-4">
              <Text style={footerStyle}>
                启用 SLA 定义 {monitoring?.activeSlas ?? 0} 个
              </Text>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 达成率：样本数为 0 时明确显示暂无样本 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} lg={12}>
          <Card
            title={
              <div className="flex items-center gap-2">
                <Clock style={{ fontSize: '20px' }} />
                <span style={{ fontSize: isFullscreen ? '20px' : '16px' }}>响应达成率</span>
              </div>
            }
            style={cardStyle}
          >
            {monitoring && monitoring.responseTimeSamples > 0 ? (
              <div className="text-center">
                <Progress
                  type="dashboard"
                  percent={Number(monitoring.responseTimeCompliance.toFixed(1))}
                  strokeColor={getComplianceColor(monitoring.responseTimeCompliance)}
                  format={percent => `${(percent ?? 0).toFixed(1)}%`}
                  style={{ fontSize: isFullscreen ? '24px' : '18px' }}
                />
                <div className="mt-4">
                  <Text style={footerStyle}>
                    样本 {monitoring.responseTimeSamples} · 达标 {monitoring.responseTimeMet}
                  </Text>
                </div>
                <div>
                  <Text style={footerStyle}>
                    平均响应时长 {formatDuration(monitoring.averageResponseMinutes)}
                  </Text>
                </div>
              </div>
            ) : (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="窗口内没有同时具备响应截止时间和首次响应时间的工单，暂无响应达成率样本"
              />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title={
              <div className="flex items-center gap-2">
                <Zap style={{ fontSize: '20px' }} />
                <span style={{ fontSize: isFullscreen ? '20px' : '16px' }}>解决达成率</span>
              </div>
            }
            style={cardStyle}
          >
            {monitoring && monitoring.resolutionTimeSamples > 0 ? (
              <div className="text-center">
                <Progress
                  type="dashboard"
                  percent={Number(monitoring.resolutionTimeCompliance.toFixed(1))}
                  strokeColor={getComplianceColor(monitoring.resolutionTimeCompliance)}
                  format={percent => `${(percent ?? 0).toFixed(1)}%`}
                  style={{ fontSize: isFullscreen ? '24px' : '18px' }}
                />
                <div className="mt-4">
                  <Text style={footerStyle}>
                    样本 {monitoring.resolutionTimeSamples} · 达标 {monitoring.resolutionTimeMet}
                  </Text>
                </div>
                <div>
                  <Text style={footerStyle}>
                    平均解决时长 {formatDuration(monitoring.averageResolutionMinutes)}
                  </Text>
                </div>
              </div>
            ) : (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="窗口内没有同时具备解决截止时间和解决时间的工单，暂无解决达成率样本"
              />
            )}
          </Card>
        </Col>
      </Row>

      {/* 绩效表格：受顶部服务类型 / 优先级过滤条件约束 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} xl={12}>
          <Card
            title={
              <div className="flex items-center gap-2">
                <XCircle style={{ fontSize: '18px' }} />
                <span style={{ fontSize: isFullscreen ? '20px' : '16px' }}>各服务类型绩效</span>
              </div>
            }
            extra={<Text style={footerStyle}>{`共 ${serviceTypeTotal} 组`}</Text>}
            style={cardStyle}
          >
            <Table<SLAPerformanceRow>
              columns={serviceTypeColumns}
              dataSource={serviceTypeRows}
              rowKey="key"
              size="small"
              loading={loading}
              pagination={false}
              scroll={{ x: 'max-content' }}
              locale={{ emptyText: '所选周期与过滤条件下没有工单' }}
            />
            {serviceTypeTotal > serviceTypeRows.length ? (
              <Text type="secondary" className="mt-2 block">
                {`仅展示前 ${serviceTypeRows.length} 组，共 ${serviceTypeTotal} 组`}
              </Text>
            ) : null}
          </Card>
        </Col>
        <Col xs={24} xl={12}>
          <Card
            title={
              <div className="flex items-center gap-2">
                <XCircle style={{ fontSize: '18px' }} />
                <span style={{ fontSize: isFullscreen ? '20px' : '16px' }}>各优先级绩效</span>
              </div>
            }
            extra={<Text style={footerStyle}>{`共 ${priorityTotal} 组`}</Text>}
            style={cardStyle}
          >
            <Table<SLAPerformanceRow>
              columns={priorityColumns}
              dataSource={priorityRows}
              rowKey="key"
              size="small"
              loading={loading}
              pagination={false}
              scroll={{ x: 'max-content' }}
              locale={{ emptyText: '所选周期与过滤条件下没有工单' }}
            />
            {priorityTotal > priorityRows.length ? (
              <Text type="secondary" className="mt-2 block">
                {`仅展示前 ${priorityRows.length} 组，共 ${priorityTotal} 组`}
              </Text>
            ) : null}
          </Card>
        </Col>
      </Row>

      {/* 告警列表：数量取后端 activeAlerts，不再用列表长度冒充 */}
      <Card
        title={
          <div className="flex items-center gap-2">
            <Bell style={{ fontSize: '20px' }} />
            <span style={{ fontSize: isFullscreen ? '24px' : '18px' }}>SLA告警列表</span>
            <Badge count={monitoring?.activeAlerts ?? 0} showZero className="ml-2" />
          </div>
        }
        extra={<Text style={footerStyle}>{`明细 ${monitoring?.alerts.length ?? 0} 条`}</Text>}
        style={cardStyle}
        styles={{
          body: {
            maxHeight: isFullscreen ? '500px' : '400px',
            overflowY: 'auto',
          },
        }}
      >
        {monitoring && monitoring.alerts.length > 0 ? (
          <Table<SLAAlertItem>
            dataSource={monitoring.alerts}
            columns={alertColumns}
            rowKey="id"
            scroll={{ x: 'max-content' }}
            pagination={false}
            size={isFullscreen ? 'large' : 'middle'}
            rowClassName={record =>
              record.alertLevel === 'critical' ? 'sla-alert-critical' : 'sla-alert-warning'
            }
          />
        ) : (
          <div className="text-center py-12">
            <CheckCircle
              style={{
                fontSize: '48px',
                color: isFullscreen ? 'rgba(255,255,255,0.5)' : '#d9d9d9',
                marginBottom: '16px',
              }}
            />
            <Text style={{ color: isFullscreen ? 'rgba(255,255,255,0.7)' : undefined }}>
              {monitoring ? '所选周期内没有未解决的 SLA 告警' : '暂无告警数据'}
            </Text>
          </div>
        )}
      </Card>

      {/* 全屏样式 */}
      <style jsx global>{`
        .sla-monitor-dashboard.fullscreen {
          color: #fff;
        }
        .sla-monitor-dashboard.fullscreen .ant-card-head-title,
        .sla-monitor-dashboard.fullscreen .ant-statistic-title {
          color: rgba(255, 255, 255, 0.8) !important;
        }
        .sla-alert-critical {
          background-color: rgba(255, 77, 79, 0.1) !important;
        }
        .sla-alert-warning {
          background-color: rgba(250, 173, 20, 0.1) !important;
        }
        .sla-stat-card {
          transition: all 0.3s ease;
        }
        .sla-stat-card:hover {
          transform: translateY(-4px);
          box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
        }
      `}</style>
    </div>
  );
};
