'use client';

import React, { useState, useEffect, useMemo, useCallback } from 'react';
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
  message,
  Spin,
} from 'antd';
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
import { RotateCcw, FileSpreadsheet, TrendingUp, TrendingDown } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import type { TicketAnalyticsResponse } from '@/lib/services/analytics-service';
import { ticketAnalyticsService } from '@/lib/services/analytics-service';
import { ticketService, TicketStatsResponse } from '@/lib/services/ticket-service';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;

const TicketAnalytics: React.FC = () => {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(30, 'day'),
    dayjs(),
  ]);
  const [analyticsData, setAnalyticsData] = useState<TicketAnalyticsResponse | null>(null);
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

  // 获取数据
  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [analyticsRes, statsRes] = await Promise.all([
        ticketAnalyticsService.getAnalytics({
          dateFrom: dateRange[0].format('YYYY-MM-DD'),
          dateTo: dateRange[1].format('YYYY-MM-DD'),
        }),
        ticketService.getTicketStats(),
      ]);

      // 适配后端返回格式到前端期望的格式
      const adaptedData: TicketAnalyticsResponse = {
        totalTickets: statsRes.total || analyticsRes.totalTickets || 0,
        openTickets: statsRes.open || analyticsRes.openTickets || 0,
        resolvedTickets: statsRes.resolved || analyticsRes.resolvedTickets || 0,
        closedTickets: analyticsRes.closedTickets || 0,
        overdueTickets: statsRes.overdue || analyticsRes.overdueTickets || 0,
        dailyTrend: analyticsRes.dailyTrend || [],
        statusDistribution: analyticsRes.statusDistribution || [],
        priorityDistribution: analyticsRes.priorityDistribution || [],
        typeDistribution: analyticsRes.typeDistribution || [],
        processingTimeStats: analyticsRes.processingTimeStats || {
          avgProcessingTime: 0,
          avgResolutionTime: 0,
          slaComplianceRate: 0,
        },
        teamPerformance: analyticsRes.teamPerformance || [],
        hotCategories: analyticsRes.hotCategories || [],
      };

      setAnalyticsData(adaptedData);
    } catch (error) {
      console.error('Failed to fetch analytics data:', error);
      message.error('获取分析数据失败');
      setAnalyticsData(getInitialData());
    } finally {
      setLoading(false);
    }
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
      message.error('导出失败，请重试');
    } finally {
      setExporting(false);
    }
  };

  // 计算百分比
  const calculatePercentage = (value: number, total: number) => {
    return total > 0 ? (value / total) * 100 : 0;
  };

  // 团队表现表格列
  const teamColumns = [
    {
      title: '处理人',
      dataIndex:'assigneeName',
      key:'assigneeName',
    },
    {
      title: '处理工单数',
      dataIndex:'totalHandled',
      key:'totalHandled',
      sorter: (a: any, b: any) => a.totalHandled - b.totalHandled,
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
  const categoryColumns = [
    {
      title: '类别',
      dataIndex: 'category',
      key: 'category',
    },
    {
      title: '工单数量',
      dataIndex: 'count',
      key: 'count',
      sorter: (a: any, b: any) => a.count - b.count,
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

  if (!analyticsData) {
    return (
      <div className="p-6 flex justify-center items-center min-h-[400px]">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* 页面标题和工具栏 */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <Title level={2}>工单分析</Title>
          <Space>
            <RangePicker
              value={dateRange}
              onChange={(dates: any) => {
                if (dates && dates.length === 2) {
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

      <Spin spinning={loading}>
        <Tabs
          activeKey={activeTab} onChange={setActiveTab}
          items={[

          ]}
        />
      </Spin>
    </div>
  );
};

export default TicketAnalytics;
