'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Select,
  Button,
  Space,
  Typography,
  Statistic,
  Progress,
  Table,
  Tag,
  DatePicker,
  Form,
  Checkbox,
  Alert,
  Divider,
  Tooltip,
} from 'antd';
import {
  TrendingUp,
  TrendingDown,
  BarChart3,
  PieChart,
  Activity,
  Target,
  Users,
  Clock,
  CheckCircle,
  AlertTriangle,
  Zap,
  Filter,
  Download,
} from 'lucide-react';
import { TicketAnalyticsApi, type AnalyticsConfig, type AnalyticsResponse } from '@/lib/api/ticket-analytics-api';
import ReportsCharts from './ReportsCharts';
import { format, subDays } from 'date-fns';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;
const { Option } = Select;

interface AdvancedAnalyticsProps {
  tenantId?: number;
}

interface KPIData {
  title: string;
  value: number | string;
  trend: 'up' | 'down' | 'stable';
  trendValue?: number;
  icon: React.ReactNode;
  color: string;
  format?: 'number' | 'percentage' | 'duration';
}

interface TrendData {
  period: string;
  value: number;
  comparison?: number;
}

interface TopPerformerData {
  name: string;
  value: number;
  efficiency?: number;
  tickets?: number;
}

const AdvancedAnalytics: React.FC<AdvancedAnalyticsProps> = ({ tenantId }) => {
  const [loading, setLoading] = useState(false);
  const [kpiData, setKpiData] = useState<KPIData[]>([]);
  const [trendData, setTrendData] = useState<TrendData[]>([]);
  const [topPerformers, setTopPerformers] = useState<TopPerformerData[]>([]);
  const [analyticsConfig, setAnalyticsConfig] = useState<Partial<AnalyticsConfig>>({
    dimensions: ['created_date'],
    metrics: ['count'],
    chart_type: 'line',
  });
  const [customTimeRange, setCustomTimeRange] = useState<[string, string]>([
    format(subDays(new Date(), 30), 'yyyy-MM-dd'),
    format(new Date(), 'yyyy-MM-dd')
  ]);

  // 预定义时间范围
  const timeRanges = {
    '7days': '最近7天',
    '30days': '最近30天',
    '90days': '最近90天',
    '6months': '最近6个月',
    '1year': '最近1年',
    'custom': '自定义',
  };

  // 加载KPI数据
  const loadKPIData = async () => {
    try {
      setLoading(true);
      
      // 模拟KPI数据
      const mockKPI: KPIData[] = [
        {
          title: '工单总量',
          value: 2847,
          trend: 'up',
          trendValue: 12.5,
          icon: <BarChart3 className="w-5 h-5" />,
          color: '#1890ff',
          format: 'number',
        },
        {
          title: '解决率',
          value: 87.3,
          trend: 'up',
          trendValue: 3.2,
          icon: <Target className="w-5 h-5" />,
          color: '#52c41a',
          format: 'percentage',
        },
        {
          title: '平均响应时间',
          value: 2.8,
          trend: 'down',
          trendValue: -15.6,
          icon: <Clock className="w-5 h-5" />,
          color: '#faad14',
          format: 'duration',
        },
        {
          title: 'SLA合规率',
          value: 94.2,
          trend: 'stable',
          trendValue: 0.0,
          icon: <CheckCircle className="w-5 h-5" />,
          color: '#722ed1',
          format: 'percentage',
        },
      ];

      setKpiData(mockKPI);

      // 模拟趋势数据
      const mockTrends: TrendData[] = [
        { period: '周一', value: 45, comparison: 38 },
        { period: '周二', value: 52, comparison: 48 },
        { period: '周三', value: 38, comparison: 42 },
        { period: '周四', value: 61, comparison: 55 },
        { period: '周五', value: 73, comparison: 68 },
        { period: '周六', value: 31, comparison: 28 },
        { period: '周日', value: 24, comparison: 22 },
      ];

      setTrendData(mockTrends);

      // 模拟Top Performer数据
      const mockTopPerformers: TopPerformerData[] = [
        { name: '张三', value: 127, efficiency: 94.5, tickets: 127 },
        { name: '李四', value: 115, efficiency: 91.2, tickets: 115 },
        { name: '王五', value: 108, efficiency: 89.8, tickets: 108 },
        { name: '赵六', value: 96, efficiency: 87.3, tickets: 96 },
        { name: '钱七', value: 89, efficiency: 85.6, tickets: 89 },
      ];

      setTopPerformers(mockTopPerformers);
    } catch (error) {
      console.error('Load KPI data failed:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadKPIData();
  }, []);

  // 渲染KPI卡片
  const renderKPI = (kpi: KPIData) => {
    const formatValue = (value: number | string, format?: string) => {
      if (typeof value === 'string') return value;
      
      switch (format) {
        case 'percentage':
          return `${value.toFixed(1)}%`;
        case 'duration':
          return `${value.toFixed(1)}h`;
        default:
          return value.toLocaleString();
      }
    };

    const getTrendIcon = () => {
      switch (kpi.trend) {
        case 'up':
          return <TrendingUp className="w-4 h-4 text-green-500" />;
        case 'down':
          return <TrendingDown className="w-4 h-4 text-red-500" />;
        default:
          return <div className="w-4 h-4 bg-gray-400 rounded-full" />;
      }
    };

    const getTrendColor = () => {
      switch (kpi.trend) {
        case 'up':
          return 'text-green-500';
        case 'down':
          return 'text-red-500';
        default:
          return 'text-gray-500';
      }
    };

    return (
      <Card className="kpi-card">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center space-x-2">
            <div style={{ color: kpi.color }}>
              {kpi.icon}
            </div>
            <Text type="secondary">{kpi.title}</Text>
          </div>
          <div className="flex items-center space-x-1">
            {getTrendIcon()}
            <Text className={`text-sm ${getTrendColor()}`}>
              {kpi.trendValue && kpi.trendValue !== 0 && (
                <span>
                  {kpi.trendValue > 0 ? '+' : ''}{kpi.trendValue}%
                </span>
              )}
            </Text>
          </div>
        </div>
        <div className="text-2xl font-bold" style={{ color: kpi.color }}>
          {formatValue(kpi.value, kpi.format)}
        </div>
      </Card>
    );
  };

  // Top Performer表格列
  const performerColumns = [
    {
      title: '排名',
      dataIndex: 'rank',
      key: 'rank',
      width: 60,
      render: (_: any, __: any, index: number) => (
        <div className={`text-center font-bold ${
          index === 0 ? 'text-yellow-500' :
          index === 1 ? 'text-gray-400' :
          index === 2 ? 'text-orange-600' : 'text-gray-600'
        }`}>
          #{index + 1}
        </div>
      ),
    },
    {
      title: '姓名',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <Space>
          <Users className="w-4 h-4 text-blue-500" />
          {name}
        </Space>
      ),
    },
    {
      title: '处理工单',
      dataIndex: 'tickets',
      key: 'tickets',
      render: (tickets: number) => (
        <Text strong>{tickets}</Text>
      ),
    },
    {
      title: '效率评分',
      dataIndex: 'efficiency',
      key: 'efficiency',
      render: (efficiency: number) => (
        <div className="flex items-center space-x-2">
          <Progress
            percent={efficiency}
            size="small"
            strokeColor={
              efficiency >= 90 ? '#52c41a' :
              efficiency >= 80 ? '#faad14' : '#ff4d4f'
            }
            showInfo={false}
            style={{ width: 60 }}
          />
          <Text>{efficiency.toFixed(1)}</Text>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* KPI卡片区域 */}
      <Row gutter={[16, 16]}>
        {kpiData.map((kpi, index) => (
          <Col xs={24} sm={12} md={6} key={index}>
            {renderKPI(kpi)}
          </Col>
        ))}
      </Row>

      {/* 控制面板 */}
      <Card title="数据分析控制台" extra={
        <Button icon={<Download />} onClick={() => message.info('导出功能开发中')}>
          导出报告
        </Button>
      }>
        <Form layout="inline">
          <Form.Item label="时间范围">
            <Select
              style={{ width: 150 }}
              value="30days"
              onChange={(value) => {
                if (value !== 'custom') {
                  const days = value === '7days' ? 7 : value === '30days' ? 30 : value === '90days' ? 90 : value === '6months' ? 180 : 365;
                  setCustomTimeRange([
                    format(subDays(new Date(), days), 'yyyy-MM-dd'),
                    format(new Date(), 'yyyy-MM-dd')
                  ]);
                }
              }}
            >
              {Object.entries(timeRanges).map(([key, label]) => (
                <Option key={key} value={key}>{label}</Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item label="分析维度">
            <Select
              style={{ width: 200 }}
              value={analyticsConfig.dimensions?.[0]}
              onChange={(value) => setAnalyticsConfig(prev => ({
                ...prev,
                dimensions: [value]
              }))}
            >
              <Option value="created_date">创建日期</Option>
              <Option value="status">工单状态</Option>
              <Option value="priority">优先级</Option>
              <Option value="category">工单分类</Option>
              <Option value="assignee">处理人</Option>
            </Select>
          </Form.Item>

          <Form.Item label="分析指标">
            <Select
              style={{ width: 200 }}
              value={analyticsConfig.metrics?.[0]}
              onChange={(value) => setAnalyticsConfig(prev => ({
                ...prev,
                metrics: [value]
              }))}
            >
              <Option value="count">工单数量</Option>
              <Option value="avg_response_time">平均响应时间</Option>
              <Option value="avg_resolution_time">平均解决时间</Option>
              <Option value="sla_compliance_rate">SLA合规率</Option>
            </Select>
          </Form.Item>

          <Form.Item>
            <Button type="primary" icon={<Activity />} onClick={loadKPIData}>
              刷新数据
            </Button>
          </Form.Item>
        </Form>
      </Card>

      {/* 图表和表格区域 */}
      <Row gutter={[16, 16]}>
        {/* 趋势图表 */}
        <Col xs={24} lg={16}>
          <Card 
            title={
              <Space>
                <TrendingUp className="w-5 h-5 text-blue-600" />
                <span>趋势分析</span>
              </Space>
            }
            extra={
              <Select defaultValue="week" style={{ width: 120 }}>
                <Option value="week">本周</Option>
                <Option value="month">本月</Option>
                <Option value="quarter">本季度</Option>
              </Select>
            }
          >
            <div style={{ height: 400 }}>
              {/* 这里应该集成真实的图表组件 */}
              <div className="flex items-center justify-center h-full bg-gray-50 rounded">
                <div className="text-center">
                  <Activity className="w-12 h-12 text-gray-400 mx-auto mb-2" />
                  <Text type="secondary">趋势图表加载中...</Text>
                  <div className="mt-4 space-y-2">
                    {trendData.map((trend, index) => (
                      <div key={index} className="flex justify-between items-center">
                        <Text className="w-12">{trend.period}</Text>
                        <div className="flex items-center space-x-2">
                          <Progress 
                            percent={(trend.value / 100) * 100} 
                            size="small" 
                            style={{ width: 80 }}
                            showInfo={false}
                          />
                          <Text className="w-12 text-right">{trend.value}</Text>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </Card>
        </Col>

        {/* Top Performer */}
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                <Zap className="w-5 h-5 text-orange-600" />
                <span>表现排行</span>
              </Space>
            }
          >
            <Table
              columns={performerColumns}
              dataSource={topPerformers}
              pagination={false}
              size="small"
              rowKey="name"
            />
          </Card>
        </Col>
      </Row>

      {/* 预警和洞察 */}
      <Alert
        message="数据洞察"
        description={
          <div className="space-y-2">
            <div>
              <AlertTriangle className="inline w-4 h-4 text-yellow-500 mr-1" />
              <span>本周工单量较上周增长12.5%，建议关注资源分配。</span>
            </div>
            <div>
              <CheckCircle className="inline w-4 h-4 text-green-500 mr-1" />
              <span>SLA合规率保持稳定，团队表现良好。</span>
            </div>
            <div>
              <TrendingDown className="inline w-4 h-4 text-red-500 mr-1" />
              <span>平均响应时间下降15.6%，处理效率提升明显。</span>
            </div>
          </div>
        }
        type="info"
        showIcon
        className="insights-alert"
      />
    </div>
  );
};

export default AdvancedAnalytics;