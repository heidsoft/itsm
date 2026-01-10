'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Typography,
  Tabs,
  Row,
  Col,
  Select,
  Button,
  Space,
  Alert,
  Statistic,
  Table,
  Tag,
  Progress,
  message,
  Spin,
  Empty,
} from 'antd';
import {
  BarChart3,
  TrendingUp,
  Download,
  FileText,
  PieChart,
  Activity,
  Clock,
  CheckCircle,
  Target,
} from 'lucide-react';
import { TicketAnalyticsApi, type AnalyticsConfig, type AnalyticsResponse } from '@/lib/api/ticket-analytics-api';
import ReportsCharts from '@/components/reports/ReportsCharts';
import ReportGenerator from '@/components/reports/ReportGenerator';
import AdvancedAnalytics from '@/components/reports/AdvancedAnalytics';
import RealTimeMonitoring from '@/components/reports/RealTimeMonitoring';
import { format, subDays } from 'date-fns';

const { Title, Text } = Typography;
const { Option } = Select;

const ReportsPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [analyticsData, setAnalyticsData] = useState<AnalyticsResponse | null>(null);
  const [timeRange, setTimeRange] = useState<[string, string]>([
    format(subDays(new Date(), 30), 'yyyy-MM-dd'),
    format(new Date(), 'yyyy-MM-dd')
  ]);
  const [activeTab, setActiveTab] = useState('overview');

  // 预定义的报告配置
  const predefinedReports = {
    ticket_trends: {
      name: '工单趋势分析',
      dimensions: ['created_date'],
      metrics: ['count'],
      chart_type: 'line' as const,
    },
    sla_performance: {
      name: 'SLA表现分析',
      dimensions: ['sla_status'],
      metrics: ['count', 'avg_response_time'],
      chart_type: 'bar' as const,
    },
    priority_distribution: {
      name: '优先级分布',
      dimensions: ['priority'],
      metrics: ['count'],
      chart_type: 'pie' as const,
    },
    resolution_time: {
      name: '解决时间分析',
      dimensions: ['resolution_category'],
      metrics: ['avg_resolution_time'],
      chart_type: 'bar' as const,
    },
  };

  // 获取数据
  const fetchAnalyticsData = async (config: Partial<AnalyticsConfig>) => {
    try {
      setLoading(true);
      const analyticsConfig: AnalyticsConfig = {
        dimensions: config.dimensions || ['created_date'],
        metrics: config.metrics || ['count'],
        chart_type: config.chart_type || 'bar',
        time_range: timeRange,
        filters: config.filters || {},
        group_by: config.group_by,
      };

      const response = await TicketAnalyticsApi.getDeepAnalytics(analyticsConfig);
      setAnalyticsData(response);
    } catch (error) {
      message.error('获取分析数据失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 导出报告
  const handleExport = async (fileFormat: 'excel' | 'pdf' | 'csv') => {
    try {
      const config: AnalyticsConfig = {
        dimensions: ['status'],
        metrics: ['count', 'avg_resolution_time'],
        chart_type: 'table',
        time_range: timeRange,
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
    } catch (error) {
      message.error('导出失败');
    }
  };

  useEffect(() => {
    // 默认加载工单趋势数据
    fetchAnalyticsData(predefinedReports.ticket_trends);
  }, [timeRange]);

  // 标签页内容
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
          {/* 快速统计 */}
          {analyticsData && (
            <Row gutter={[16, 16]}>
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="总工单数"
                    value={analyticsData.summary.total}
                    prefix={<FileText className="w-4 h-4" />}
                    valueStyle={{ color: '#1890ff' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="已解决"
                    value={analyticsData.summary.resolved}
                    prefix={<CheckCircle className="w-4 h-4" />}
                    valueStyle={{ color: '#52c41a' }}
                    suffix={
                      <Text type="secondary">
                        ({((analyticsData.summary.resolved / analyticsData.summary.total) * 100).toFixed(1)}%)
                      </Text>
                    }
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="平均响应时间"
                    value={analyticsData.summary.avg_response_time}
                    suffix="小时"
                    prefix={<Clock className="w-4 h-4" />}
                    valueStyle={{ color: '#fa8c16' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="SLA合规率"
                    value={analyticsData.summary.sla_compliance}
                    suffix="%"
                    prefix={<Target className="w-4 h-4" />}
                    valueStyle={{ 
                      color: analyticsData.summary.sla_compliance >= 95 ? '#52c41a' : '#f5222d' 
                    }}
                  />
                </Card>
              </Col>
            </Row>
          )}

          {/* 图表区域 */}
          <Card title="工单趋势分析" extra={
            <Space>
              <Select
                defaultValue="ticket_trends"
                style={{ width: 200 }}
                onChange={(value) => {
                  fetchAnalyticsData(predefinedReports[value as keyof typeof predefinedReports]);
                }}
              >
                {Object.entries(predefinedReports).map(([key, report]) => (
                  <Option key={key} value={key}>
                    {report.name}
                  </Option>
                ))}
              </Select>
              <Button 
                icon={<Download size={16} />}
                onClick={() => handleExport('excel')}
              >
                导出
              </Button>
            </Space>
          }>
            <ReportsCharts 
              data={analyticsData?.data || []} 
              loading={loading}
              chartType="line"
            />
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
      children: (
        <div className="space-y-6">
          <AdvancedAnalytics />
        </div>
      ),
    },
    {
      key: 'detailed',
      label: (
        <Space>
          <Activity size={16} />
          自定义分析
        </Space>
      ),
      children: (
        <div className="space-y-6">
          <Card title="自定义分析报告">
            <ReportGenerator 
              onGenerate={fetchAnalyticsData}
              loading={loading}
              timeRange={timeRange}
            />
          </Card>
        </div>
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
      children: (
        <div className="space-y-6">
          <RealTimeMonitoring autoRefresh={true} refreshInterval={30} />
        </div>
      ),
    },
    {
      key: 'export',
      label: (
        <Space>
          <Download size={16} />
          报告导出
        </Space>
      ),
      children: (
        <div className="space-y-6">
          <Card title="报告导出中心">
            <Space direction="vertical" size="large" style={{ width: '100%' }}>
              <div>
                <Title level={4}>选择导出格式</Title>
                <Space wrap>
                  <Button 
                    size="large"
                    onClick={() => handleExport('excel')}
                    icon={<Download size={16} />}
                  >
                    导出 Excel
                  </Button>
                  <Button 
                    size="large"
                    onClick={() => handleExport('pdf')}
                    icon={<Download size={16} />}
                  >
                    导出 PDF
                  </Button>
                  <Button 
                    size="large"
                    onClick={() => handleExport('csv')}
                    icon={<Download size={16} />}
                  >
                    导出 CSV
                  </Button>
                </Space>
              </div>

              <div>
                <Title level={4}>时间范围</Title>
                <Space direction="vertical">
                  <div>
                    <Text type="secondary">开始日期: {timeRange[0]}</Text>
                  </div>
                  <div>
                    <Text type="secondary">结束日期: {timeRange[1]}</Text>
                  </div>
                </Space>
              </div>
            </Space>
          </Card>
        </div>
      ),
    },
  ];

  return (
    <div className="p-6">
      <div className="mb-6">
        <Title level={2} className="mb-2">
          报表中心
        </Title>
        <Text type="secondary">
          全面的ITSM数据分析与报告中心，提供多维度数据洞察和决策支持
        </Text>
      </div>

      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabItems}
        />
      </Card>
    </div>
  );
};

export default ReportsPage;
