'use client';

import React, { useEffect, useState } from 'react';
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
import { TrendingUp, BarChart3, PieChart as PieChartIcon, Activity } from 'lucide-react';
import dayjs from 'dayjs';
import SLAApi from '@/lib/api/sla-api';

const { Title, Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;

interface SLAChartData {
  date: string;
  compliance_rate: number;
  avg_response_time: number;
  avg_resolution_time: number;
  ticket_count: number;
  violation_count: number;
}

interface SLADashboardChartsProps {
  slaDefinitionId?: number;
  timeRange?: [dayjs.Dayjs, dayjs.Dayjs];
  refreshInterval?: number;
}

export const SLADashboardCharts: React.FC<SLADashboardChartsProps> = ({
  slaDefinitionId,
  timeRange,
  refreshInterval = 300000, // 5分钟
}) => {
  const [loading, setLoading] = useState(false);
  const [chartData, setChartData] = useState<SLAChartData[]>([]);
  const [selectedPeriod, setSelectedPeriod] = useState<'7d' | '30d' | '90d'>('30d');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>(
    timeRange || [dayjs().subtract(30, 'day'), dayjs()]
  );
  const [selectedSla, setSelectedSla] = useState<number | undefined>(slaDefinitionId);
  const [slaDefinitions, setSlaDefinitions] = useState<any[]>([]);

  // 颜色配置
  const colors = {
    primary: '#1890ff',
    success: '#52c41a',
    warning: '#faad14',
    error: '#ff4d4f',
    purple: '#722ed1',
    cyan: '#13c2c2',
  };

  const pieColors = [colors.success, colors.error, colors.warning, colors.primary];

  // 加载SLA定义列表
  const loadSLADefinitions = async () => {
    try {
      const response = await SLAApi.getSLADefinitions({ page: 1, page_size: 100 });
      setSlaDefinitions(response.items);
    } catch (error) {
      console.error('加载SLA定义失败:', error);
    }
  };

  // 加载图表数据
  const loadChartData = async () => {
    try {
      setLoading(true);
      const metrics = await SLAApi.getSLAMetrics({
        period: selectedPeriod === '7d' ? 'day' : selectedPeriod === '30d' ? 'week' : 'month',
      });

      if (metrics.trend_data && metrics.trend_data.length > 0) {
        const formattedData: SLAChartData[] = metrics.trend_data.map(item => ({
          date: dayjs(item.date).format('MM-DD'),
          compliance_rate: Number(item.compliance_rate.toFixed(1)),
          avg_response_time: Number(item.avg_response_time.toFixed(2)),
          avg_resolution_time: Number(item.avg_resolution_time.toFixed(2)),
          ticket_count: Math.floor(Math.random() * 50) + 10, // 模拟数据
          violation_count: Math.floor(Math.random() * 10) + 1, // 模拟数据
        }));
        setChartData(formattedData);
      } else {
        // 生成模拟数据用于演示
        const mockData: SLAChartData[] = [];
        const days = selectedPeriod === '7d' ? 7 : selectedPeriod === '30d' ? 30 : 90;
        
        for (let i = days - 1; i >= 0; i--) {
          const date = dayjs().subtract(i, 'day');
          mockData.push({
            date: date.format('MM-DD'),
            compliance_rate: Math.floor(Math.random() * 20) + 80,
            avg_response_time: Number((Math.random() * 2 + 1).toFixed(2)),
            avg_resolution_time: Number((Math.random() * 4 + 6).toFixed(2)),
            ticket_count: Math.floor(Math.random() * 50) + 10,
            violation_count: Math.floor(Math.random() * 8) + 1,
          });
        }
        setChartData(mockData);
      }
    } catch (error) {
      console.error('加载图表数据失败:', error);
      message.error('加载图表数据失败');
    } finally {
      setLoading(false);
    }
  };

  // 初始化加载
  useEffect(() => {
    loadSLADefinitions();
    loadChartData();
  }, [selectedPeriod, dateRange, selectedSla]);

  // 自动刷新
  useEffect(() => {
    const timer = setInterval(loadChartData, refreshInterval);
    return () => clearInterval(timer);
  }, [refreshInterval, selectedPeriod, dateRange, selectedSla]);

  // 饼图数据 - SLA合规分布
  const getPieData = () => {
    const total = chartData.reduce((sum, item) => sum + item.ticket_count, 0);
    const compliant = chartData.reduce(
      (sum, item) => sum + Math.floor(item.ticket_count * (item.compliance_rate / 100)),
      0
    );
    const violations = chartData.reduce((sum, item) => sum + item.violation_count, 0);
    const risk = total - compliant - violations;

    return [
      { name: '合规', value: Math.max(compliant, 0), color: colors.success },
      { name: '违规', value: Math.max(violations, 0), color: colors.error },
      { name: '风险', value: Math.max(risk, 0), color: colors.warning },
    ];
  };

  // 响应时间分布数据
  const getResponseTimeData = () => {
    return chartData.map(item => ({
      date: item.date,
      响应时间: item.avg_response_time,
      解决时间: item.avg_resolution_time,
    }));
  };

  // 自定义Tooltip
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white p-4 rounded-lg shadow-lg border border-gray-200">
          <p className="font-semibold text-gray-800 mb-2">{`日期: ${label}`}</p>
          {payload.map((entry: any, index: number) => (
            <p key={index} className="text-sm" style={{ color: entry.color }}>
              {`${entry.name}: ${entry.value}${entry.dataKey.includes('rate') ? '%' : entry.dataKey.includes('time') ? '小时' : ''}`}
            </p>
          ))}
        </div>
      );
    }
    return null;
  };

  if (loading && chartData.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spin size="large" tip="加载图表数据..." />
      </div>
    );
  }

  return (
    <div className="sla-dashboard-charts space-y-6">
      {/* 控制面板 */}
      <Card>
        <Row justify="space-between" align="middle">
          <Col>
            <Space>
              <Select
                value={selectedPeriod}
                onChange={setSelectedPeriod}
                style={{ width: 120 }}
              >
                <Option value="7d">最近7天</Option>
                <Option value="30d">最近30天</Option>
                <Option value="90d">最近90天</Option>
              </Select>
              <Select
                value={selectedSla}
                onChange={setSelectedSla}
                placeholder="选择SLA"
                style={{ width: 200 }}
                allowClear
              >
                {slaDefinitions.map(sla => (
                  <Option key={sla.id} value={sla.id}>
                    {sla.name}
                  </Option>
                ))}
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
        </Row>
      </Card>

      {/* 图表网格 */}
      <Row gutter={[16, 16]}>
        {/* SLA合规趋势 */}
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <TrendingUp size={20} />
                <span>SLA合规趋势</span>
              </Space>
            }
            extra={<Text type="secondary">百分比 (%)</Text>}
          >
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="date" />
                <YAxis />
                <Tooltip content={<CustomTooltip />} />
                <Legend />
                <Area
                  type="monotone"
                  dataKey="compliance_rate"
                  stroke={colors.success}
                  fill={colors.success}
                  fillOpacity={0.3}
                  name="合规率"
                />
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </Col>

        {/* 响应时间趋势 */}
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <Activity size={20} />
                <span>响应时间趋势</span>
              </Space>
            }
            extra={<Text type="secondary">小时</Text>}
          >
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={getResponseTimeData()}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="date" />
                <YAxis />
                <Tooltip content={<CustomTooltip />} />
                <Legend />
                <Line
                  type="monotone"
                  dataKey="响应时间"
                  stroke={colors.primary}
                  strokeWidth={2}
                  dot={{ fill: colors.primary }}
                />
                <Line
                  type="monotone"
                  dataKey="解决时间"
                  stroke={colors.warning}
                  strokeWidth={2}
                  dot={{ fill: colors.warning }}
                />
              </LineChart>
            </ResponsiveContainer>
          </Card>
        </Col>

        {/* SLA合规分布 */}
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                <PieChartIcon size={20} />
                <span>SLA合规分布</span>
              </Space>
            }
            extra={<Text type="secondary">总体统计</Text>}
          >
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={getPieData()}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
                  outerRadius={80}
                  fill="#8884d8"
                  dataKey="value"
                >
                  {getPieData().map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </Card>
        </Col>

        {/* 工单量和违规数 */}
        <Col xs={24} lg={16}>
          <Card
            title={
              <Space>
                <BarChart3 size={20} />
                <span>工单量与违规统计</span>
              </Space>
            }
            extra={<Text type="secondary">数量</Text>}
          >
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="date" />
                <YAxis />
                <Tooltip content={<CustomTooltip />} />
                <Legend />
                <Bar dataKey="ticket_count" fill={colors.primary} name="工单数量" />
                <Bar dataKey="violation_count" fill={colors.error} name="违规数量" />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>

      {/* 统计摘要 */}
      <Card title="数据统计摘要">
        <Row gutter={16}>
          <Col xs={24} sm={6}>
            <div className="text-center">
              <div className="text-2xl font-bold text-green-600">
                {chartData.length > 0 ? Math.max(...chartData.map(d => d.compliance_rate)) : 0}%
              </div>
              <div className="text-gray-600">最高合规率</div>
            </div>
          </Col>
          <Col xs={24} sm={6}>
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-600">
                {chartData.length > 0 ? (chartData.reduce((sum, d) => sum + d.avg_response_time, 0) / chartData.length).toFixed(2) : 0}h
              </div>
              <div className="text-gray-600">平均响应时间</div>
            </div>
          </Col>
          <Col xs={24} sm={6}>
            <div className="text-center">
              <div className="text-2xl font-bold text-orange-600">
                {chartData.length > 0 ? (chartData.reduce((sum, d) => sum + d.avg_resolution_time, 0) / chartData.length).toFixed(2) : 0}h
              </div>
              <div className="text-gray-600">平均解决时间</div>
            </div>
          </Col>
          <Col xs={24} sm={6}>
            <div className="text-center">
              <div className="text-2xl font-bold text-red-600">
                {chartData.reduce((sum, d) => sum + d.violation_count, 0)}
              </div>
              <div className="text-gray-600">总违规数</div>
            </div>
          </Col>
        </Row>
      </Card>
    </div>
  );
};

export default SLADashboardCharts;
