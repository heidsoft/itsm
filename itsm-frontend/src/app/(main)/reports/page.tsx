'use client';

import React, { useEffect, useMemo, useState } from 'react';
import dynamic from 'next/dynamic';
import { App, Button, Card, Select, Space, Tabs, Typography } from 'antd';
import { Activity, BarChart3, Download, FileText, TrendingUp } from 'lucide-react';
import { format, subDays } from 'date-fns';

import { FilterToolbarCard } from '@/components/ui/FilterToolbarCard';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview } from '@/components/ui/StatsOverview';
import {
  TicketAnalyticsApi,
  type AnalyticsConfig,
  type AnalyticsResponse,
} from '@/lib/api/ticket-analytics-api';

const ReportsCharts = dynamic(() => import('@/components/reports/ReportsCharts'), {
  ssr: false,
  loading: () => <div className="p-8 text-center text-slate-500">正在加载图表组件...</div>,
});

const ReportGenerator = dynamic(() => import('@/components/reports/ReportGenerator'), {
  ssr: false,
  loading: () => <div className="p-8 text-center text-slate-500">正在加载自定义分析器...</div>,
});

const AdvancedAnalytics = dynamic(() => import('@/components/reports/AdvancedAnalytics'), {
  ssr: false,
  loading: () => <div className="p-8 text-center text-slate-500">正在加载高级分析...</div>,
});

const RealTimeMonitoring = dynamic(() => import('@/components/reports/RealTimeMonitoring'), {
  ssr: false,
  loading: () => <div className="p-8 text-center text-slate-500">正在加载实时监控...</div>,
});

const { Text } = Typography;

const predefinedReports = {
  ticketTrends: {
    name: '工单趋势分析',
    dimensions: ['created_date'],
    metrics: ['count'],
    chartType: 'line' as const,
  },
  slaPerformance: {
    name: 'SLA 表现分析',
    dimensions: ['sla_status'],
    metrics: ['count', 'avg_response_time'],
    chartType: 'bar' as const,
  },
  priorityDistribution: {
    name: '优先级分布',
    dimensions: ['priority'],
    metrics: ['count'],
    chartType: 'pie' as const,
  },
  resolutionTime: {
    name: '解决时间分析',
    dimensions: ['resolution_category'],
    metrics: ['avg_resolution_time'],
    chartType: 'bar' as const,
  },
};

const emptyAnalytics: AnalyticsResponse = {
  data: [],
  summary: {
    total: 0,
    resolved: 0,
    avgResponseTime: 0,
    avgResolutionTime: 0,
    slaCompliance: 0,
    customerSatisfaction: 0,
  },
  generatedAt: new Date().toISOString(),
};

const ReportsPage: React.FC = () => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState('overview');
  const [selectedReportKey, setSelectedReportKey] =
    useState<keyof typeof predefinedReports>('ticketTrends');
  const [analyticsData, setAnalyticsData] = useState<AnalyticsResponse>(emptyAnalytics);
  const [timeRange, setTimeRange] = useState<[string, string]>([
    format(subDays(new Date(), 30), 'yyyy-MM-dd'),
    format(new Date(), 'yyyy-MM-dd'),
  ]);

  const selectedReport = predefinedReports[selectedReportKey];

  const fetchAnalyticsData = async (config: Partial<AnalyticsConfig>) => {
    try {
      setLoading(true);
      setError(null);
      const analyticsConfig: AnalyticsConfig = {
        dimensions: config.dimensions || ['created_date'],
        metrics: config.metrics || ['count'],
        chartType: config.chartType || 'bar',
        timeRange: timeRange,
        filters: config.filters || {},
        groupBy: config.groupBy,
      };

      const response = await TicketAnalyticsApi.getDeepAnalytics(analyticsConfig);
      setAnalyticsData({
        data: Array.isArray(response?.data) ? response.data : [],
        summary: response?.summary || emptyAnalytics.summary,
        generatedAt: response?.generatedAt || new Date().toISOString(),
      });
    } catch (fetchError) {
      setAnalyticsData(emptyAnalytics);
      setError('报表数据暂时不可用，已切换到安全空态。');
      message.error('获取分析数据失败');
      console.error(fetchError);
    } finally {
      setLoading(false);
    }
  };

  const handleExport = async (fileFormat: 'excel' | 'pdf' | 'csv') => {
    try {
      const config: AnalyticsConfig = {
        dimensions: selectedReport.dimensions,
        metrics: selectedReport.metrics,
        chartType: 'table',
        timeRange: timeRange,
        filters: {},
      };

      const blob = await TicketAnalyticsApi.exportAnalytics(config, fileFormat);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `report-${format(new Date(), 'yyyy-MM-dd')}.${fileFormat}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      message.success(`报告导出成功 (${fileFormat.toUpperCase()})`);
    } catch (exportError) {
      console.error('Export failed:', exportError);
      message.warning('导出功能暂时不可用，请稍后重试');
    }
  };

  useEffect(() => {
    fetchAnalyticsData(predefinedReports[selectedReportKey]);
  }, [selectedReportKey, timeRange]);

  const statsItems = useMemo(
    () => [
      {
        key: 'total',
        title: '总工单数',
        value: analyticsData.summary.total,
        prefix: <FileText className="h-5 w-5" />,
        accentColor: '#1677ff',
      },
      {
        key: 'resolved',
        title: '已解决',
        value: analyticsData.summary.resolved,
        prefix: <Activity className="h-5 w-5" />,
        accentColor: '#52c41a',
        helper:
          analyticsData.summary.total > 0
            ? `占比 ${((analyticsData.summary.resolved / analyticsData.summary.total) * 100).toFixed(1)}%`
            : '暂无已解决数据',
      },
      {
        key: 'response',
        title: '平均响应时间',
        value: analyticsData.summary.avgResponseTime,
        suffix: '小时',
        prefix: <TrendingUp className="h-5 w-5" />,
        accentColor: '#fa8c16',
      },
      {
        key: 'sla',
        title: 'SLA 合规率',
        value: analyticsData.summary.slaCompliance,
        suffix: '%',
        prefix: <BarChart3 className="h-5 w-5" />,
        accentColor: analyticsData.summary.slaCompliance >= 95 ? '#52c41a' : '#faad14',
      },
    ],
    [analyticsData]
  );

  const tabItems = [
    {
      key: 'overview',
      label: (
        <Space>
          <BarChart3 size={16} />
          概览仪表板
        </Space>
      ),
      children: (
        <div className="space-y-6">
          <StatsOverview items={statsItems} />
          <Card title={selectedReport.name}>
            <LoadingEmptyError
              state={loading ? 'loading' : analyticsData.data.length === 0 ? 'empty' : 'success'}
              loadingText="正在加载报表数据..."
              empty={{
                title: '暂无可展示的报表数据',
                description: error || '请切换报表类型或稍后重试。',
                actionText: '重新加载',
                onAction: () => fetchAnalyticsData(selectedReport),
              }}
            >
              <ReportsCharts
                data={analyticsData.data}
                loading={loading}
                chartType={selectedReport.chartType}
              />
            </LoadingEmptyError>
          </Card>
        </div>
      ),
    },
    {
      key: 'advanced',
      label: (
        <Space>
          <TrendingUp size={16} />
          高级分析
        </Space>
      ),
      children: <AdvancedAnalytics />,
    },
    {
      key: 'custom',
      label: (
        <Space>
          <Activity size={16} />
          自定义分析
        </Space>
      ),
      children: (
        <Card title="自定义分析报告">
          <ReportGenerator
            onGenerate={fetchAnalyticsData}
            loading={loading}
            timeRange={timeRange}
          />
        </Card>
      ),
    },
    {
      key: 'realtime',
      label: (
        <Space>
          <BarChart3 size={16} />
          实时监控
        </Space>
      ),
      children: <RealTimeMonitoring autoRefresh refreshInterval={30} />,
    },
  ];

  return (
    <div className="space-y-6 p-6">
      <ManagementPageHeader
        title="报表中心"
        description="统一查看 ITSM 趋势、SLA 表现和自定义分析结果。"
      />

      <FilterToolbarCard
        filters={
          <>
            <Select
              value={selectedReportKey}
              style={{ width: 220 }}
              onChange={value => setSelectedReportKey(value)}
              options={Object.entries(predefinedReports).map(([key, report]) => ({
                value: key,
                label: report.name,
              }))}
            />
            <Button
              onClick={() =>
                setTimeRange([
                  format(subDays(new Date(), 7), 'yyyy-MM-dd'),
                  format(new Date(), 'yyyy-MM-dd'),
                ])
              }
            >
              最近 7 天
            </Button>
            <Button
              onClick={() =>
                setTimeRange([
                  format(subDays(new Date(), 30), 'yyyy-MM-dd'),
                  format(new Date(), 'yyyy-MM-dd'),
                ])
              }
            >
              最近 30 天
            </Button>
          </>
        }
        actions={
          <>
            <Text type="secondary">
              时间范围：{timeRange[0]} 至 {timeRange[1]}
            </Text>
            <Button icon={<Download size={16} />} onClick={() => handleExport('excel')}>
              导出 Excel
            </Button>
          </>
        }
      />

      <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} />
    </div>
  );
};

export default ReportsPage;
