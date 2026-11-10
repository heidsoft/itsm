'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Card,
  Row,
  Col,
  Select,
  DatePicker,
  Button,
  Space,
  Typography,
  Statistic,
  Table,
  Tag,
  Tabs,
  Form,
  Input,
  Switch,
  Radio,
  Checkbox,
  message,
  App,
  Divider,
  Empty,
  Spin,
  Modal,
  Tooltip as AntTooltip,
} from 'antd';
import {
  LineChart as RechartsLineChart,
  Line,
  BarChart as RechartsBarChart,
  Bar,
  PieChart as RechartsPieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  AreaChart,
  Area,
} from 'recharts';
import { Filter, Download, Settings, RotateCcw, CheckCircle, BarChart3, Table2 } from 'lucide-react';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import dayjs from 'dayjs';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;
const CheckboxGroup = Checkbox.Group;

interface AnalyticsDimension {
  key: string;
  label: string;
  type: 'category' | 'time' | 'numeric';
  options?: string[];
}

interface AnalyticsMetric {
  key: string;
  label: string;
  type: 'count' | 'sum' | 'avg' | 'max' | 'min';
  field?: string;
}

interface ChartData {
  name: string;
  value: number;
  [key: string]: unknown;
}

interface AnalyticsConfig {
  dimensions: string[];
  metrics: string[];
  chartType: 'line' | 'bar' | 'pie' | 'area' | 'table';
  timeRange: [string, string];
  filters: Record<string, unknown>;
  groupBy?: string;
}

interface TicketDeepAnalyticsProps {
  defaultConfig?: Partial<AnalyticsConfig>;
  onConfigChange?: (config: AnalyticsConfig) => void;
  canExport?: boolean;
}

export const TicketDeepAnalytics: React.FC<TicketDeepAnalyticsProps> = ({
  defaultConfig,
  onConfigChange,
  canExport = true,
}) => {
  const { message: antMessage } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [config, setConfig] = useState<AnalyticsConfig>({
    dimensions: ['status'],
    metrics: ['count'],
    chartType: 'bar',
    timeRange: [
      format(new Date(Date.now() - 30 * 24 * 60 * 60 * 1000), 'yyyy-MM-dd'),
      format(new Date(), 'yyyy-MM-dd'),
    ],
    filters: {},
    ...defaultConfig,
  });
  const [chartData, setChartData] = useState<ChartData[]>([]);
  const [summaryData, setSummaryData] = useState<Record<string, any>>({});
  const [configModalVisible, setConfigModalVisible] = useState(false);
  const [form] = Form.useForm();

  // 可用维度
  const availableDimensions: AnalyticsDimension[] = [
    {
      key: 'status',
      label: 'Status',
      type: 'category',
      options: ['open', 'in_progress', 'resolved', 'closed'],
    },
    {
      key: 'priority',
      label: 'Priority',
      type: 'category',
      options: ['low', 'medium', 'high', 'urgent'],
    },
    {
      key: 'type',
      label: 'Type',
      type: 'category',
      options: ['incident', 'service_request', 'problem', 'change'],
    },
    { key: 'category', label: 'Category', type: 'category' },
    { key: 'assignee', label: 'Assignee', type: 'category' },
    { key: 'department', label: 'Department', type: 'category' },
    { key:'createdDate', label: 'Creation date', type: 'time' },
    { key:'resolvedDate', label: 'Resolution date', type: 'time' },
  ];

  // 可用指标
  const availableMetrics: AnalyticsMetric[] = [
    { key: 'count', label: 'Count', type: 'count' },
    { key:'responseTime', label: 'Avg Response Time', type: 'avg', field: 'response_time' },
    { key:'resolutionTime', label: 'Avg Resolution Time', type: 'avg', field: 'resolution_time' },
    { key:'slaCompliance', label: 'SLA Compliance Rate', type: 'avg', field: 'sla_compliance' },
    { key:'customerSatisfaction', label: 'Customer Satisfaction', type: 'avg', field: 'rating' },
  ];

  // 加载分析数据
  const loadAnalyticsData = useCallback(async () => {
    try {
      setLoading(true);
      // 调用实际API
      const { TicketAnalyticsApi } = await import('@/lib/api/ticket-analytics-api');
      const data = await TicketAnalyticsApi.getDeepAnalytics(config);

      if (data && data.data && data.data.length > 0) {
        setChartData(data.data);
        setSummaryData(data.summary);
      } else {
        // 如果API返回空，显示空状态
        setChartData([]);
        setSummaryData({
          total: 0,
          resolved: 0,
          avgResponseTime: 0,
          avgResolutionTime: 0,
          slaCompliance: 0,
          customerSatisfaction: 0,
        });
        antMessage.info('No matching data found');
      }
    } catch (error) {
      antMessage.error('Failed to load analysis data');
      // 出错时重置为空状态
      setChartData([]);
      setSummaryData({
        total: 0,
        resolved: 0,
        avgResponseTime: 0,
        avgResolutionTime: 0,
        slaCompliance: 0,
        customerSatisfaction: 0,
      });
    } finally {
      setLoading(false);
    }
  }, [config, antMessage]);

  useEffect(() => {
    loadAnalyticsData();
  }, [loadAnalyticsData]);

  // 保存配置
  const handleSaveConfig = useCallback(async () => {
    try {
      const values = await form.validateFields();
      const newConfig: AnalyticsConfig = {
        ...config,
        ...values,
      };
      setConfig(newConfig);
      setConfigModalVisible(false);
      onConfigChange?.(newConfig);
      antMessage.success('Configuration saved');
      loadAnalyticsData();
    } catch (error) {
      antMessage.error('Failed to save configuration');
    }
  }, [form, config, onConfigChange, antMessage, loadAnalyticsData]);

  // 导出数据
  const handleExport = useCallback(async () => {
    try {
      // 调用真实导出API
      const { TicketAnalyticsApi } = await import('@/lib/api/ticket-analytics-api');
      const blob = await TicketAnalyticsApi.exportAnalytics(config, 'excel');
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `deep-analytics-${dayjs().format('YYYY-MM-DD')}.xlsx`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      antMessage.success('Exported');
    } catch (error) {
      console.error('Export failed:', error);
        antMessage.error('Export failed, please try again later.');
    }
  }, [config, antMessage]);

  // 渲染图表
  const renderChart = () => {
    if (chartData.length === 0) {
      return <Empty description="No data" />;
    }

    const COLORS = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1', '#13c2c2'];

    switch (config.chartType) {
      case 'line':
        return (
          <ResponsiveContainer width="100%" height={400}>
            <RechartsLineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="value" stroke="#1890ff" strokeWidth={2} />
            </RechartsLineChart>
          </ResponsiveContainer>
        );
      case 'bar':
        return (
          <ResponsiveContainer width="100%" height={400}>
            <RechartsBarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Bar dataKey="value" fill="#1890ff" />
            </RechartsBarChart>
          </ResponsiveContainer>
        );
      case 'pie':
        return (
          <ResponsiveContainer width="100%" height={400}>
            <RechartsPieChart>
              <Pie
                data={chartData}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={({ name, percent }) => `${name} ${((percent || 0) * 100).toFixed(0)}%`}
                outerRadius={120}
                fill="#8884d8"
                dataKey="value"
              >
                {chartData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip />
            </RechartsPieChart>
          </ResponsiveContainer>
        );
      case 'area':
        return (
          <ResponsiveContainer width="100%" height={400}>
            <AreaChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Area
                type="monotone"
                dataKey="value"
                stroke="#1890ff"
                fill="#1890ff"
                fillOpacity={0.6}
              />
            </AreaChart>
          </ResponsiveContainer>
        );
      case 'table':
        return (
          <Table
            dataSource={chartData}
            columns={[
              { title: 'Name', dataIndex: 'name', key: 'name' },
              { title: 'Value', dataIndex: 'value', key: 'value' },
              ...(config.metrics.includes('response_time')
                ? [{ title: 'Avg Response Time', dataIndex:'avgTime', key:'avgTime' }]
                : []),
            ]}
            rowKey="name"
            scroll={{ x: 'max-content' }}
            pagination={false}
          />
        );
      default:
        return null;
    }
  };

  return (
    <div className="space-y-4">
      {/* 工具栏 */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <Title level={5} style={{ margin: 0 }}>

            Deep Data Analysis
          </Title>
          <Space>
            <Button
              icon={<Settings />}
              onClick={() => {
                form.setFieldsValue(config);
                setConfigModalVisible(true);
              }}
            >

              Configure
            </Button>
            {canExport && (
              <Button icon={<Download />} onClick={handleExport}>

                Export
              </Button>
            )}
            <Button icon={<RotateCcw />} onClick={loadAnalyticsData} loading={loading}>

              Refresh
            </Button>
          </Space>
        </div>

        {/* 时间范围和快速筛选 */}
        <Space className="mb-4" wrap>
          <RangePicker
            value={[
              config.timeRange[0] ? dayjs(config.timeRange[0]) : null,
              config.timeRange[1] ? dayjs(config.timeRange[1]) : null,
            ]}
            onChange={dates => {
              if (dates) {
                setConfig({
                  ...config,
                  timeRange: [dates[0]!.format('YYYY-MM-DD'), dates[1]!.format('YYYY-MM-DD')],
                });
              }
            }}
          />
          <Select
            value={config.chartType}
            onChange={value => setConfig({ ...config, chartType: value })}
            style={{ width: 120 }}
            options={[
              { value: 'line', label: 'Line chart' },
              { value: 'bar', label: 'Bar chart' },
              { value: 'pie', label: 'Pie chart' },
              { value: 'area', label: 'Area Chart' },
              { value: 'table', label: 'Table' },
            ]}
          />
        </Space>
      </Card>

      {/* 汇总统计 */}
      <Row gutter={16}>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="Total Tickets"
              value={summaryData.total || 0}
              prefix={<BarChart3 />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="Resolved"
              value={summaryData.resolved || 0}
              styles={{ content: { color: '#3f8600' } }}
              prefix={<CheckCircle />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="Avg Response Time"
              value={summaryData.avgResponseTime || 0}
              precision={1}
              suffix="hours"
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="SLA Compliance Rate"
              value={summaryData.slaCompliance || 0}
              precision={1}
              suffix="%"
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 图表 */}
      <Card title="Data analysis chart" loading={loading}>
        {renderChart()}
      </Card>

      {/* 配置模态框 */}
      <Modal
        title="Analysis configuration"
        open={configModalVisible}
        onOk={handleSaveConfig}
        onCancel={() => setConfigModalVisible(false)}
        width={800}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="dimensions"
            label="Analysis Dimensions"
            rules={[{ required: true, message: 'Please select at least one dimension' }]}
          >
            <CheckboxGroup>
              {availableDimensions.map(dim => (
                <Checkbox key={dim.key} value={dim.key}>
                  {dim.label}
                </Checkbox>
              ))}
            </CheckboxGroup>
          </Form.Item>
          <Form.Item
            name="metrics"
            label="Analysis indicators"
            rules={[{ required: true, message: 'Please select at least one indicator' }]}
          >
            <CheckboxGroup>
              {availableMetrics.map(metric => (
                <Checkbox key={metric.key} value={metric.key}>
                  {metric.label}
                </Checkbox>
              ))}
            </CheckboxGroup>
          </Form.Item>
          <Form.Item name="chartType" label="Chart type">
            <Radio.Group>
              <Radio value="line">Line chart</Radio>
              <Radio value="bar">Bar chart</Radio>
              <Radio value="pie">Pie chart</Radio>
              <Radio value="area">Area Chart</Radio>
              <Radio value="table">Table</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item name="groupBy" label="Grouping method">
            <Select placeholder="Please select a grouping method" allowClear options={availableDimensions
              .filter(d => d.type === 'category')
              .map(dim => ({
                value: dim.key,
                label: dim.label,
              }))} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};
