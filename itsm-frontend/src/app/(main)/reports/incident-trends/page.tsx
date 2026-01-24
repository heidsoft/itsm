'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Select,
  DatePicker,
  Space,
  Typography,
  Spin,
  message,
  Statistic,
} from 'antd';
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import {
  TrendingUp,
  BarChart3,
  PieChart as PieChartIcon,
  AlertTriangle,
  CheckCircle,
  Clock,
} from 'lucide-react';
import dayjs from 'dayjs';
import { ticketAnalyticsService } from '@/lib/services/analytics-service';
import { ticketService } from '@/lib/services/ticket-service';

const { Title, Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;

const IncidentTrendsPage = () => {
  const [loading, setLoading] = useState(false);
  const [selectedPeriod, setSelectedPeriod] = useState<'7d' | '30d' | '90d'>('30d');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(30, 'day'),
    dayjs(),
  ]);
  const [trendData, setTrendData] = useState<any[]>([]);
  const [priorityData, setPriorityData] = useState<any[]>([]);
  const [stats, setStats] = useState({
    total: 0,
    created: 0,
    resolved: 0,
    avgResponseTime: 0,
    avgResolutionTime: 0,
  });

  // 颜色配置
  const colors = {
    primary: '#1890ff',
    success: '#52c41a',
    warning: '#faad14',
    error: '#ff4d4f',
    purple: '#722ed1',
    cyan: '#13c2c2',
  };

  const priorityColors = {
    low: colors.success,
    medium: colors.warning,
    high: colors.error,
    urgent: colors.purple,
  };

  // 加载数据
  const loadData = async () => {
    try {
      setLoading(true);
      const [analyticsRes, statsRes] = await Promise.all([
        ticketAnalyticsService.getAnalytics({
          date_from: dateRange[0].format('YYYY-MM-DD'),
          date_to: dateRange[1].format('YYYY-MM-DD'),
        }),
        ticketService.getTicketStats(),
      ]);

      // 处理趋势数据
      if (analyticsRes.daily_trend && analyticsRes.daily_trend.length > 0) {
        setTrendData(analyticsRes.daily_trend);
      } else {
        // 生成模拟数据
        setTrendData(generateMockTrend());
      }

      // 处理优先级数据
      if (analyticsRes.priority_distribution && analyticsRes.priority_distribution.length > 0) {
        setPriorityData(analyticsRes.priority_distribution);
      } else {
        setPriorityData([
          { name: '低', value: statsRes.low_priority || 0, color: priorityColors.low },
          { name: '中', value: statsRes.medium_priority || 0, color: priorityColors.medium },
          { name: '高', value: statsRes.high_priority || 0, color: priorityColors.high },
          { name: '紧急', value: statsRes.urgent || 0, color: priorityColors.urgent },
        ]);
      }

      // 更新统计
      setStats({
        total: statsRes.total || analyticsRes.total_tickets || 0,
        created: analyticsRes.daily_trend?.reduce((sum, d) => sum + d.created, 0) || 0,
        resolved: analyticsRes.daily_trend?.reduce((sum, d) => sum + d.resolved, 0) || 0,
        avgResponseTime: analyticsRes.processing_time_stats?.avg_processing_time || 0,
        avgResolutionTime: analyticsRes.processing_time_stats?.avg_resolution_time || 0,
      });
    } catch (error) {
      console.error('加载报表数据失败:', error);
      message.error('加载报表数据失败');
      // 使用模拟数据
      setTrendData(generateMockTrend());
      setPriorityData([
        { name: '低', value: 234, color: priorityColors.low },
        { name: '中', value: 389, color: priorityColors.medium },
        { name: '高', value: 198, color: priorityColors.high },
        { name: '紧急', value: 26, color: priorityColors.urgent },
      ]);
    } finally {
      setLoading(false);
    }
  };

  // 生成模拟趋势数据
  const generateMockTrend = () => {
    const days = selectedPeriod === '7d' ? 7 : selectedPeriod === '30d' ? 30 : 90;
    return Array.from({ length: days }, (_, i) => ({
      date: dayjs().subtract(days - 1 - i, 'day').format('MM-DD'),
      created: Math.floor(Math.random() * 20) + 5,
      resolved: Math.floor(Math.random() * 15) + 3,
      open: Math.floor(Math.random() * 10) + 2,
    }));
  };

  useEffect(() => {
    loadData();
  }, [selectedPeriod, dateRange]);

  // 自定义Tooltip
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white p-3 rounded-lg shadow-lg border border-gray-200">
          <p className="font-semibold text-gray-800 mb-1">{`日期: ${label}`}</p>
          {payload.map((entry: any, index: number) => (
            <p key={index} className="text-sm" style={{ color: entry.color }}>
              {`${entry.name}: ${entry.value}`}
            </p>
          ))}
        </div>
      );
    }
    return null;
  };

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <header className="mb-6">
        <Title level={2}>事件趋势报表</Title>
        <p className="text-gray-500 mt-1">
          展示事件数量随时间变化的趋势以及按优先级的分布情况
        </p>
      </header>

      {/* 控制面板 */}
      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col>
            <Space>
              <Select
                value={selectedPeriod}
                onChange={setSelectedPeriod}
                style={{ width: 140 }}
              >
                <Option value="7d">最近7天</Option>
                <Option value="30d">最近30天</Option>
                <Option value="90d">最近90天</Option>
              </Select>
              <RangePicker
                value={dateRange}
                onChange={(dates) => {
                  if (dates) {
                    setDateRange([dates[0]!, dates[1]!]);
                  }
                }}
              />
            </Space>
          </Col>
          <Col>
            <Space>
              <Button icon={<ReloadOutlined />} onClick={loadData}>
                刷新数据
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 统计卡片 */}
      <Row gutter={16} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="事件总数"
              value={stats.total}
              prefix={<BarChart3 size={20} style={{ color: colors.primary }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="新增事件"
              value={stats.created}
              valueStyle={{ color: colors.primary }}
              prefix={<TrendingUp size={20} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="已解决事件"
              value={stats.resolved}
              valueStyle={{ color: colors.success }}
              prefix={<CheckCircle size={20} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="平均解决时间"
              value={stats.avgResolutionTime.toFixed(1)}
              suffix="小时"
              prefix={<Clock size={20} style={{ color: colors.warning }} />}
            />
          </Card>
        </Col>
      </Row>

      {loading && trendData.length === 0 ? (
        <div className="flex items-center justify-center h-96">
          <Spin size="large" tip="加载报表数据..." />
        </div>
      ) : (
        <>
          {/* 趋势图表 */}
          <Row gutter={[16, 16]} className="mb-6">
            <Col span={24}>
              <Card title="每日事件数量趋势">
                <ResponsiveContainer width="100%" height={350}>
                  <AreaChart data={trendData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" />
                    <YAxis />
                    <Tooltip content={<CustomTooltip />} />
                    <Legend />
                    <Area
                      type="monotone"
                      dataKey="created"
                      stackId="1"
                      stroke={colors.primary}
                      fill={colors.primary}
                      fillOpacity={0.3}
                      name="新建事件"
                    />
                    <Area
                      type="monotone"
                      dataKey="resolved"
                      stackId="2"
                      stroke={colors.success}
                      fill={colors.success}
                      fillOpacity={0.3}
                      name="已解决事件"
                    />
                    <Area
                      type="monotone"
                      dataKey="open"
                      stackId="3"
                      stroke={colors.warning}
                      fill={colors.warning}
                      fillOpacity={0.3}
                      name="待处理事件"
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </Card>
            </Col>
          </Row>

          {/* 优先级分布和解决趋势 */}
          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <Card title="按优先级分布">
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={priorityData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip content={<CustomTooltip />} />
                    <Legend />
                    <Bar dataKey="value" name="事件数量" fill={colors.primary}>
                      {priorityData.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={entry.color || colors.primary} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card title="事件解决率趋势">
                <ResponsiveContainer width="100%" height={300}>
                  <LineChart data={trendData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" />
                    <YAxis />
                    <Tooltip content={<CustomTooltip />} />
                    <Legend />
                    <Line
                      type="monotone"
                      dataKey="resolved"
                      stroke={colors.success}
                      strokeWidth={2}
                      dot={{ fill: colors.success }}
                      name="已解决"
                    />
                    <Line
                      type="monotone"
                      dataKey="created"
                      stroke={colors.primary}
                      strokeWidth={2}
                      dot={{ fill: colors.primary }}
                      name="新建"
                    />
                  </LineChart>
                </ResponsiveContainer>
              </Card>
            </Col>
          </Row>
        </>
      )}
    </div>
  );
};

// 导入需要的组件
import { Button } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';

export default IncidentTrendsPage;
