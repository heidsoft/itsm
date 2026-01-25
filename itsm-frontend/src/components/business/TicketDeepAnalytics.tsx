// @ts-nocheck
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
} from 'antd';
import {
  BarChartOutlined,
  LineChartOutlined,
  PieChartOutlined,
  TableOutlined,
  DownloadOutlined,
  SettingOutlined,
  FilterOutlined,
  ReloadOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import {
  LineChart,
  Line,
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
  AreaChart,
  Area,
} from 'recharts';
import { CheckCircleOutlined } from '@ant-design/icons';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;
const { Option } = Select;
const { CheckboxGroup } = Checkbox;

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
  [key: string]: any;
}

interface AnalyticsConfig {
  dimensions: string[];
  metrics: string[];
  chartType: 'line' | 'bar' | 'pie' | 'area' | 'table';
  timeRange: [string, string];
  filters: Record<string, any>;
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
      label: '状态',
      type: 'category',
      options: ['open', 'in_progress', 'resolved', 'closed'],
    },
    {
      key: 'priority',
      label: '优先级',
      type: 'category',
      options: ['low', 'medium', 'high', 'urgent'],
    },
    {
      key: 'type',
      label: '类型',
      type: 'category',
      options: ['incident', 'service_request', 'problem', 'change'],
    },
    { key: 'category', label: '分类', type: 'category' },
    { key: 'assignee', label: '处理人', type: 'category' },
    { key: 'department', label: '部门', type: 'category' },
    { key: 'created_date', label: '创建日期', type: 'time' },
    { key: 'resolved_date', label: '解决日期', type: 'time' },
  ];

  // 可用指标
  const availableMetrics: AnalyticsMetric[] = [
    { key: 'count', label: '数量', type: 'count' },
    { key: 'response_time', label: '平均响应时间', type: 'avg', field: 'response_time' },
    { key: 'resolution_time', label: '平均解决时间', type: 'avg', field: 'resolution_time' },
    { key: 'sla_compliance', label: 'SLA合规率', type: 'avg', field: 'sla_compliance' },
    { key: 'customer_satisfaction', label: '客户满意度', type: 'avg', field: 'rating' },
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
          avg_response_time: 0,
          avg_resolution_time: 0,
          sla_compliance: 0,
          customer_satisfaction: 0,
        });
        antMessage.info('未找到符合条件的数据');
      }
    } catch (error) {
      console.error('Failed to load analytics data:', error);
      antMessage.error('加载分析数据失败');
      // 出错时重置为空状态
      setChartData([]);
      setSummaryData({
        total: 0,
        resolved: 0,
        avg_response_time: 0,
        avg_resolution_time: 0,
        sla_compliance: 0,
        customer_satisfaction: 0,
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
      antMessage.success('配置已保存');
      loadAnalyticsData();
    } catch (error) {
      console.error('Failed to save config:', error);
    }
  }, [form, config, onConfigChange, antMessage, loadAnalyticsData]);

  // 导出数据
  const handleExport = useCallback(() => {
    // 注意：导出功能需要后端支持导出API
    antMessage.info('导出功能开发中');
  }, [antMessage]);

  // 渲染图表
  const renderChart = () => {
    if (chartData.length === 0) {
      return <Empty description='暂无数据' />;
    }

    const COLORS = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1', '#13c2c2'];

    switch (config.chartType) {
      case 'line':
        return (
          <ResponsiveContainer width='100%' height={400}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray='3 3' />
              <XAxis dataKey='name' />
              <YAxis />
              <Tooltip />
              <Legend />
              <Line type='monotone' dataKey='value' stroke='#1890ff' strokeWidth={2} />
            </LineChart>
          </ResponsiveContainer>
        );
      case 'bar':
        return (
          <ResponsiveContainer width='100%' height={400}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray='3 3' />
              <XAxis dataKey='name' />
              <YAxis />
              <Tooltip />
              <Legend />
              <Bar dataKey='value' fill='#1890ff' />
            </BarChart>
          </ResponsiveContainer>
        );
      case 'pie':
        return (
          <ResponsiveContainer width='100%' height={400}>
            <PieChart>
              <Pie
                data={chartData}
                cx='50%'
                cy='50%'
                labelLine={false}
                label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                outerRadius={120}
                fill='#8884d8'
                dataKey='value'
              >
                {chartData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        );
      case 'area':
        return (
          <ResponsiveContainer width='100%' height={400}>
            <AreaChart data={chartData}>
              <CartesianGrid strokeDasharray='3 3' />
              <XAxis dataKey='name' />
              <YAxis />
              <Tooltip />
              <Legend />
              <Area
                type='monotone'
                dataKey='value'
                stroke='#1890ff'
                fill='#1890ff'
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
              { title: '名称', dataIndex: 'name', key: 'name' },
              { title: '数值', dataIndex: 'value', key: 'value' },
              ...(config.metrics.includes('response_time')
                ? [{ title: '平均响应时间', dataIndex: 'avg_time', key: 'avg_time' }]
                : []),
            ]}
            rowKey='name'
            pagination={false}
          />
        );
      default:
        return null;
    }
  };

  return (
    <div className='space-y-4'>
      {/* 工具栏 */}
      <Card>
        <div className='flex items-center justify-between mb-4'>
          <Title level={5} style={{ margin: 0 }}>
            深度数据分析
          </Title>
          <Space>
            <Button
              icon={<SettingOutlined />}
              onClick={() => {
                form.setFieldsValue(config);
                setConfigModalVisible(true);
              }}
            >
              配置
            </Button>
            {canExport && (
              <Button icon={<DownloadOutlined />} onClick={handleExport}>
                导出
              </Button>
            )}
            <Button icon={<ReloadOutlined />} onClick={loadAnalyticsData} loading={loading}>
              刷新
            </Button>
          </Space>
        </div>

        {/* 时间范围和快速筛选 */}
        <Space className='mb-4' wrap>
          <RangePicker
            value={[
              config.timeRange[0] ? new Date(config.timeRange[0]) : null,
              config.timeRange[1] ? new Date(config.timeRange[1]) : null,
            ]}
            onChange={dates => {
              if (dates) {
                setConfig({
                  ...config,
                  timeRange: [format(dates[0]!, 'yyyy-MM-dd'), format(dates[1]!, 'yyyy-MM-dd')],
                });
              }
            }}
          />
          <Select
            value={config.chartType}
            onChange={value => setConfig({ ...config, chartType: value })}
            style={{ width: 120 }}
          >
            <Option value='line'>
              <LineChartOutlined /> 折线图
            </Option>
            <Option value='bar'>
              <BarChartOutlined /> 柱状图
            </Option>
            <Option value='pie'>
              <PieChartOutlined /> 饼图
            </Option>
            <Option value='area'>
              <AreaChart /> 面积图
            </Option>
            <Option value='table'>
              <TableOutlined /> 表格
            </Option>
          </Select>
        </Space>
      </Card>

      {/* 汇总统计 */}
      <Row gutter={16}>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title='总工单数'
              value={summaryData.total || 0}
              prefix={<BarChartOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title='已解决'
              value={summaryData.resolved || 0}
              valueStyle={{ color: '#3f8600' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title='平均响应时间'
              value={summaryData.avg_response_time || 0}
              precision={1}
              suffix='小时'
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title='SLA合规率'
              value={summaryData.sla_compliance || 0}
              precision={1}
              suffix='%'
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 图表 */}
      <Card title='数据分析图表' loading={loading}>
        {renderChart()}
      </Card>

      {/* 配置模态框 */}
      <Modal
        title='分析配置'
        open={configModalVisible}
        onOk={handleSaveConfig}
        onCancel={() => setConfigModalVisible(false)}
        width={800}
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='dimensions'
            label='分析维度'
            rules={[{ required: true, message: '请至少选择一个维度' }]}
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
            name='metrics'
            label='分析指标'
            rules={[{ required: true, message: '请至少选择一个指标' }]}
          >
            <CheckboxGroup>
              {availableMetrics.map(metric => (
                <Checkbox key={metric.key} value={metric.key}>
                  {metric.label}
                </Checkbox>
              ))}
            </CheckboxGroup>
          </Form.Item>
          <Form.Item name='chartType' label='图表类型'>
            <Radio.Group>
              <Radio value='line'>折线图</Radio>
              <Radio value='bar'>柱状图</Radio>
              <Radio value='pie'>饼图</Radio>
              <Radio value='area'>面积图</Radio>
              <Radio value='table'>表格</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item name='groupBy' label='分组方式'>
            <Select placeholder='请选择分组方式' allowClear>
              {availableDimensions
                .filter(d => d.type === 'category')
                .map(dim => (
                  <Option key={dim.key} value={dim.key}>
                    {dim.label}
                  </Option>
                ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};
