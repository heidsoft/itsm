'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Table,
  Select,
  DatePicker,
  Button,
  Space,
  Typography,
  Tag,
  Progress,
  Tabs,
  Tooltip,
  App,
  Spin,
  Empty,
  Alert,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  BarChart,
  Bar,
  LineChart,
  Line,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as RechartsTooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { RotateCcw, FileSpreadsheet } from 'lucide-react';
import dayjs from 'dayjs';
import type { TicketAnalyticsResponse } from '@/lib/services/analytics-service';
import { ticketAnalyticsService } from '@/lib/services/analytics-service';
import { ticketService } from '@/lib/services/ticket-service';
import { isPermissionDeniedError, serverErrorMessage } from '@/lib/api/http-client';
import { TicketDeepAnalytics } from '@/components/business/TicketDeepAnalytics';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;

type LoadNotice = {
  fatal: boolean;
  denied: boolean;
  // 两个数据源权限不同，提示里必须说清缺的是哪一块、要哪个权限码，
  // 否则使用者只能看到「获取分析数据失败」。
  source: string;
  permission: string;
  detail: string;
};

const TicketAnalytics: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const { message } = App.useApp();
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(30, 'day'),
    dayjs(),
  ]);
  const [analyticsData, setAnalyticsData] = useState<TicketAnalyticsResponse | null>(null);
  // GET /tickets/stats 只要 ticket:read，GET /analytics/tickets 要 report:read。
  // 之前用 Promise.all + 全 0 兜底：坐席点进来（菜单码曾错挂 ticket:read）就被 403
  // 打回一屏 0，把「没有权限」渲染成「这个租户没有工单」。
  const [loadNotice, setLoadNotice] = useState<LoadNotice | null>(null);
  const [activeTab, setActiveTab] = useState('overview');

  function getInitialData(): TicketAnalyticsResponse {
    return {
      totalTickets: 0,
      openTickets: 0,
      resolvedTickets: 0,
      closedTickets: 0,
      overdueTickets: 0,
      dailyTrend: [],
      statusDistribution: [],
      priorityDistribution: [],
      typeDistribution: [],
      processingTimeStats: {
        avgProcessingTime: 0,
        avgResolutionTime: 0,
        slaComplianceRate: 0,
      },
      teamPerformance: [],
      hotCategories: [],
    };
  }

  const fetchData = useCallback(async () => {
    setLoading(true);
    const [analyticsResult, statsResult] = await Promise.allSettled([
      ticketAnalyticsService.getAnalytics({
        dateFrom: dateRange[0].format('YYYY-MM-DD'),
        dateTo: dateRange[1].format('YYYY-MM-DD'),
      }),
      ticketService.getTicketStats(),
    ]);

    const analytics = analyticsResult.status === 'fulfilled' ? analyticsResult.value : null;
    const stats = statsResult.status === 'fulfilled' ? statsResult.value : null;

    // 两边都失败才是整页失败：此时不渲染任何数字，避免 0 冒充统计结果。
    if (!analytics && !stats) {
      const reasons = [analyticsResult, statsResult]
        .filter((r): r is PromiseRejectedResult => r.status === 'rejected')
        .map((r) => r.reason);
      const deniedError = reasons.find(isPermissionDeniedError);
      const error = deniedError ?? reasons[0];
      setAnalyticsData(null);
      setLoadNotice({
        fatal: true,
        denied: Boolean(deniedError),
        source: '工单分析数据',
        permission: 'report:read / ticket:read',
        detail: serverErrorMessage(error, '后端未能返回工单统计。'),
      });
      setLoading(false);
      return;
    }

    // stats 是租户全量、不随日期筛选变化；analytics 是所选窗口的明细。
    // 口径差异由下方提示说明，不再用 || 把 0 当成缺失值。
    const adaptedData: TicketAnalyticsResponse = {
      totalTickets: stats?.total ?? analytics?.totalTickets ?? 0,
      openTickets: stats?.open ?? analytics?.openTickets ?? 0,
      resolvedTickets: stats?.resolved ?? analytics?.resolvedTickets ?? 0,
      closedTickets: stats?.closed ?? analytics?.closedTickets ?? 0,
      overdueTickets: stats?.overdue ?? analytics?.overdueTickets ?? 0,
      dailyTrend: analytics?.dailyTrend ?? [],
      statusDistribution: analytics?.statusDistribution ?? [],
      priorityDistribution: analytics?.priorityDistribution ?? [],
      typeDistribution: analytics?.typeDistribution ?? [],
      processingTimeStats: analytics?.processingTimeStats ?? getInitialData().processingTimeStats,
      teamPerformance: analytics?.teamPerformance ?? [],
      hotCategories: analytics?.hotCategories ?? [],
    };

    setAnalyticsData(adaptedData);
    if (analytics && stats) {
      setLoadNotice(null);
    } else {
      // analytics 缺失 → 日期窗口内的趋势/分布看不了（report:read）；
      // stats 缺失 → 只剩窗口口径（ticket:read）。两者文案与权限码不同，分开给。
      const missing = analytics
        ? { source: '工单总量', permission: 'ticket:read' }
        : { source: '日期窗口内的趋势与分布', permission: 'report:read' };
      const failed = analytics ? statsResult : analyticsResult;
      const error = failed.status === 'rejected' ? failed.reason : undefined;
      setLoadNotice({
        fatal: false,
        denied: isPermissionDeniedError(error),
        source: missing.source,
        permission: missing.permission,
        detail: serverErrorMessage(error, '后端未能返回该数据源。'),
      });
    }
    setLoading(false);
  }, [dateRange]);

  useEffect(() => {
    fetchData();
  }, [dateRange, fetchData]);

  // 导出数据
  const handleExport = async (format: 'csv' | 'excel' | 'pdf' = 'excel') => {
    setExporting(true);
    try {
      const blob = await ticketAnalyticsService.exportAnalytics({
        dateFrom: dateRange[0].format('YYYY-MM-DD'),
        dateTo: dateRange[1].format('YYYY-MM-DD'),
        format,
      });

      // 创建下载链接
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `工单分析_${dateRange[0].format('YYYYMMDD')}_${dateRange[1].format('YYYYMMDD')}.${format === 'excel' ? 'xlsx' : format}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      message.success('导出成功');
    } catch (error) {
      console.error('Export failed:', error);
      // 服务端给出的业务文案要透出来，否则参数错误、权限不足和后端故障都只剩
      // 「导出失败，请重试」，运维和用户都无从判断。
      message.error(error instanceof Error && error.message ? error.message : '导出失败，请重试');
    } finally {
      setExporting(false);
    }
  };

  // 计算百分比
  const calculatePercentage = (value: number, total: number) => {
    return total > 0 ? (value / total) * 100 : 0;
  };

  // 团队表现表格列
  const teamColumns: ColumnsType<TicketAnalyticsResponse['teamPerformance'][number]> = [
    {
      title: '处理人',
      dataIndex:'assigneeName',
      key:'assigneeName',
    },
    {
      title: '处理工单数',
      dataIndex:'totalHandled',
      key:'totalHandled',
      sorter: (a, b) => a.totalHandled - b.totalHandled,
    },
    {
      title: '已解决',
      dataIndex: 'resolved',
      key: 'resolved',
    },
    {
      title: '平均响应时间(小时)',
      dataIndex:'avgResponseTime',
      key:'avgResponseTime',
      render: (time: number) => time?.toFixed(1) || '-',
    },
    {
      title: '平均解决时间(小时)',
      dataIndex:'avgResolutionTime',
      key:'avgResolutionTime',
      render: (time: number) => time?.toFixed(1) || '-',
    },
  ];

  // 热门类别表格列
  const categoryColumns: ColumnsType<TicketAnalyticsResponse['hotCategories'][number]> = [
    {
      title: '类别',
      dataIndex: 'category',
      key: 'category',
    },
    {
      title: '工单数量',
      dataIndex: 'count',
      key: 'count',
      sorter: (a, b) => a.count - b.count,
    },
    {
      title: '趋势',
      dataIndex: 'trend',
      key: 'trend',
      render: (trend: string) => (
        <Tag color={trend === 'up' ? 'green' : trend === 'down' ? 'red' : 'default'}>
          {trend === 'up' ? '上升' : trend === 'down' ? '下降' : '持平'}
        </Tag>
      ),
    },
  ];

  const noticeAlert = loadNotice && (
    <Alert
      className="mb-4"
      type={loadNotice.denied ? 'warning' : 'error'}
      showIcon
      title={loadNotice.denied ? `${loadNotice.source}没有访问权限` : `${loadNotice.source}加载失败`}
      description={
        loadNotice.denied
          ? `${loadNotice.detail}。该数据源需要 ${loadNotice.permission}，请联系管理员为当前角色开放该权限；缺少的数据块不等于租户里没有工单。`
          : `${loadNotice.detail}。请稍后重试；若持续失败，请联系管理员查看后端日志。`
      }
      action={
        <Button size="small" icon={<RotateCcw />} onClick={fetchData} loading={loading}>
          重试
        </Button>
      }
    />
  );

  // 整页失败：不渲染 KPI 卡片和图表，0 会被读成「这个租户没有工单」。
  if (loadNotice?.fatal) {
    return (
      <div className="p-6">
        <Title level={2} className="mb-4">
          工单分析
        </Title>
        {noticeAlert}
      </div>
    );
  }

  if (!analyticsData) {
    return (
      <div className="p-6 flex justify-center items-center min-h-[400px]">
        <Spin size="large" />
      </div>
    );
  }

  // 饼图/bar 图 tooltip 占比分母（early-return 之后 analyticsData 必非空，
  // 普通计算即可，避免在条件分支后才调用 Hook）
  const statusTotal = (analyticsData.statusDistribution ?? []).reduce((acc, s) => acc + s.value, 0);
  const priorityTotal = (analyticsData.priorityDistribution ?? []).reduce((acc, p) => acc + p.value, 0);

  // 总览 tab：KPI + 趋势 + 分布图
  const overviewTab = (
    <Space orientation="vertical" size="middle" className="w-full">
      <Row gutter={[16, 16]}>
        <Col xs={12} sm={8} lg={4}>
          <Card size="small">
            <Statistic title="工单总数" value={analyticsData.totalTickets} />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card size="small">
            <Statistic title="待处理" value={analyticsData.openTickets} />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card size="small">
            <Statistic title="已解决" value={analyticsData.resolvedTickets} />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card size="small">
            <Statistic title="已关闭" value={analyticsData.closedTickets} />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card size="small">
            <Statistic title="已超期" value={analyticsData.overdueTickets} styles={{ content: { color: analyticsData.overdueTickets > 0 ? '#cf1322' : undefined } }} />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={4}>
          <Card size="small">
            <div className="flex flex-col items-center">
              <Text type="secondary" className="mb-1">SLA 达标率</Text>
              <Progress type="circle" size={64} percent={Math.round(analyticsData.processingTimeStats.slaComplianceRate)} />
            </div>
          </Card>
        </Col>
      </Row>

      <Card title="工单趋势（每日新建/解决）" size="small">
        {analyticsData.dailyTrend.length > 0 ? (
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={analyticsData.dailyTrend}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" tickFormatter={(d: string) => dayjs(d).format('MM-DD')} fontSize={12} />
              <YAxis allowDecimals={false} />
              <RechartsTooltip
                labelFormatter={(d: unknown) => dayjs(String(d)).format('YYYY-MM-DD')}
                formatter={(value: unknown, name: unknown) => [Number(value) || 0, name === 'created' ? '新建' : name === 'resolved' ? '已解决' : String(name)]}
              />
              <Legend formatter={(v: string) => (v === 'created' ? '新建' : v === 'resolved' ? '已解决' : v)} />
              <Line type="monotone" dataKey="created" stroke="#1890ff" strokeWidth={2} dot={false} />
              <Line type="monotone" dataKey="resolved" stroke="#52c41a" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <Empty description="所选时间范围内暂无趋势数据" />
        )}
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="状态分布" size="small">
            {analyticsData.statusDistribution.length > 0 ? (
              <ResponsiveContainer width="100%" height={280}>
                <PieChart>
                  <Pie
                    data={analyticsData.statusDistribution}
                    dataKey="value"
                    nameKey="name"
                    cx="50%"
                    cy="50%"
                    outerRadius={90}
                    label
                  >
                    {analyticsData.statusDistribution.map((entry) => (
                      <Cell key={entry.name} fill={entry.color} />
                    ))}
                  </Pie>
                  <RechartsTooltip
                    formatter={(value: unknown, name: unknown) => [
                      `${value} (${calculatePercentage(Number(value) || 0, statusTotal).toFixed(1)}%)`,
                      String(name),
                    ]}
                  />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <Empty description="暂无状态分布数据" />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="优先级分布" size="small">
            {analyticsData.priorityDistribution.length > 0 ? (
              <ResponsiveContainer width="100%" height={280}>
                <BarChart data={analyticsData.priorityDistribution}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" fontSize={12} />
                  <YAxis allowDecimals={false} />
                  <RechartsTooltip
                    formatter={(value: unknown, name: unknown) => [
                      `${value} (${calculatePercentage(Number(value) || 0, priorityTotal).toFixed(1)}%)`,
                      String(name),
                    ]}
                  />
                  <Bar dataKey="value" name="工单数" radius={[4, 4, 0, 0]}>
                    {analyticsData.priorityDistribution.map((entry) => (
                      <Cell key={entry.name} fill={entry.color} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <Empty description="暂无优先级分布数据" />
            )}
          </Card>
        </Col>
      </Row>

      <Card title="处理效率" size="small">
        <Row gutter={[16, 16]}>
          <Col xs={12} lg={6}>
            <Statistic
              title="平均响应时间（小时）"
              value={analyticsData.processingTimeStats.avgProcessingTime || 0}
              precision={1}
            />
          </Col>
          <Col xs={12} lg={6}>
            <Statistic
              title="平均解决时间（小时）"
              value={analyticsData.processingTimeStats.avgResolutionTime || 0}
              precision={1}
            />
          </Col>
          <Col xs={12} lg={6}>
            <Statistic
              title="解决率"
              value={calculatePercentage(analyticsData.resolvedTickets, analyticsData.totalTickets)}
              precision={1}
              suffix="%"
            />
          </Col>
          <Col xs={12} lg={6}>
            <Statistic
              title="超期率"
              value={calculatePercentage(analyticsData.overdueTickets, analyticsData.totalTickets)}
              precision={1}
              suffix="%"
            />
          </Col>
        </Row>
        {(analyticsData.typeDistribution.length === 0) && (
          <Text type="secondary" className="mt-2 block">
            注：后端当前未提供类型分布与处理时长明细，相关指标暂为空/0，不区分于真实零值。
          </Text>
        )}
      </Card>
    </Space>
  );

  return (
    <div className="p-6">
      {/* 页面标题和工具栏 */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <Title level={2}>工单分析</Title>
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
            <Button icon={<RotateCcw />} onClick={fetchData} loading={loading}>
              刷新数据
            </Button>
            <Button
              icon={<FileSpreadsheet />}
              onClick={() => handleExport('excel')}
              loading={exporting}
            >
              导出报表
            </Button>
          </Space>
        </div>
      </div>

      {noticeAlert}

      <Spin spinning={loading}>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            { key: 'overview', label: '总览', children: overviewTab },
            {
              key: 'team',
              label: '团队表现',
              children: (
                <Card size="small">
                  <Table
                    columns={teamColumns}
                    dataSource={analyticsData.teamPerformance}
                    rowKey="assigneeName"
                    pagination={false}
                    size="small"
                    locale={{ emptyText: <Empty description="暂无团队表现数据" /> }}
                  />
                </Card>
              ),
            },
            {
              key: 'category',
              label: '热门类别',
              children: (
                <Card size="small">
                  <Table
                    columns={categoryColumns}
                    dataSource={analyticsData.hotCategories}
                    rowKey="category"
                    pagination={false}
                    size="small"
                    locale={{ emptyText: <Empty description="暂无热门类别数据" /> }}
                  />
                </Card>
              ),
            },
            {
              key: 'deep',
              label: '深度分析',
              children: <TicketDeepAnalytics />,
            },
          ]}
        />
      </Spin>
    </div>
  );
};

export default TicketAnalytics;
