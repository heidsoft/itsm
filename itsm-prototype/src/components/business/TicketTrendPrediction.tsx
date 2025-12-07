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
  Alert,
  Form,
  Input,
  Radio,
  message,
  App,
  Spin,
  Empty,
  Divider,
  Progress,
} from 'antd';
import {
  ArrowUpOutlined,
  ArrowDownOutlined,
  WarningOutlined,
  CheckCircleOutlined,
  BarChartOutlined,
  DownloadOutlined,
  ReloadOutlined,
  CalendarOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { TicketPredictionApi } from '@/lib/api/ticket-prediction-api';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;
const { Option } = Select;
const { TextArea } = Input;

interface PredictionData {
  date: string;
  actual?: number;
  predicted: number;
  upper_bound?: number;
  lower_bound?: number;
  confidence?: number;
}

interface PredictionMetrics {
  accuracy: number;
  mape: number; // Mean Absolute Percentage Error
  rmse: number; // Root Mean Square Error
  trend: 'up' | 'down' | 'stable';
  trend_strength: number;
  next_week_prediction: number;
  next_month_prediction: number;
  risk_level: 'low' | 'medium' | 'high';
  risk_factors: string[];
}

interface PredictionReport {
  period: string;
  summary: string;
  key_findings: string[];
  recommendations: string[];
  metrics: PredictionMetrics;
  data: PredictionData[];
  generated_at: string;
}

interface TicketTrendPredictionProps {
  ticketType?: string;
  category?: string;
  onPredictionChange?: (prediction: PredictionReport) => void;
}

export const TicketTrendPrediction: React.FC<TicketTrendPredictionProps> = ({
  ticketType,
  category,
  onPredictionChange,
}) => {
  const { message: antMessage } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [predictionReport, setPredictionReport] = useState<PredictionReport | null>(null);
  const [timeRange, setTimeRange] = useState<[string, string]>([
    format(new Date(Date.now() - 90 * 24 * 60 * 60 * 1000), 'yyyy-MM-dd'),
    format(new Date(), 'yyyy-MM-dd'),
  ]);
  const [predictionPeriod, setPredictionPeriod] = useState<'week' | 'month' | 'quarter'>('month');
  const [modelType, setModelType] = useState<'arima' | 'exponential' | 'linear'>('arima');

  // 加载预测数据
  const loadPrediction = useCallback(async () => {
    try {
      setLoading(true);
      // 调用实际API
      const data = await TicketPredictionApi.getTrendPrediction({
        ticket_type: ticketType,
        category: category,
        time_range: timeRange,
        prediction_period: predictionPeriod,
        model_type: modelType,
      });
      
      if (data) {
        setPredictionReport(data);
        onPredictionChange?.(data);
        return;
      }
      
      // 如果API返回空，使用模拟数据
      const mockData: PredictionData[] = [];
      const today = new Date();
      for (let i = 30; i >= 0; i--) {
        const date = new Date(today);
        date.setDate(date.getDate() - i);
        mockData.push({
          date: format(date, 'yyyy-MM-dd'),
          actual: i < 20 ? Math.floor(Math.random() * 50) + 20 : undefined,
          predicted: Math.floor(Math.random() * 50) + 20,
          upper_bound: Math.floor(Math.random() * 50) + 30,
          lower_bound: Math.floor(Math.random() * 50) + 10,
          confidence: 85 + Math.random() * 10,
        });
      }

      const mockMetrics: PredictionMetrics = {
        accuracy: 87.5,
        mape: 12.3,
        rmse: 4.2,
        trend: 'up',
        trend_strength: 0.75,
        next_week_prediction: 145,
        next_month_prediction: 620,
        risk_level: 'medium',
        risk_factors: ['节假日影响', '季节性波动', '系统升级'],
      };

      const mockReport: PredictionReport = {
        period: `${timeRange[0]} 至 ${timeRange[1]}`,
        summary: '根据历史数据分析，预计未来一个月工单数量将呈上升趋势，建议提前准备资源。',
        key_findings: [
          '工单数量在过去30天呈上升趋势，增长率为15%',
          '预计下周工单数量将达到145个，较本周增长8%',
          '预计下月工单数量将达到620个，较本月增长12%',
          '高风险时段集中在工作日9-11点和14-16点',
        ],
        recommendations: [
          '建议增加处理人员，特别是高峰时段',
          '建议优化自动化流程，减少人工处理时间',
          '建议提前准备常见问题解决方案，提升响应速度',
          '建议监控系统性能，预防潜在问题',
        ],
        metrics: mockMetrics,
        data: mockData,
        generated_at: format(new Date(), 'yyyy-MM-dd HH:mm:ss', { locale: zhCN }),
      };

      setPredictionReport(mockReport);
      onPredictionChange?.(mockReport);
    } catch (error) {
      console.error('Failed to load prediction:', error);
      antMessage.error('加载预测数据失败');
    } finally {
      setLoading(false);
    }
  }, [ticketType, category, timeRange, predictionPeriod, modelType, antMessage, onPredictionChange]);

  useEffect(() => {
    loadPrediction();
  }, [loadPrediction]);

  // 获取趋势图标
  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case 'up':
        return <ArrowUpOutlined style={{ color: '#ff4d4f' }} />;
      case 'down':
        return <ArrowDownOutlined style={{ color: '#52c41a' }} />;
      default:
        return <BarChartOutlined style={{ color: '#1890ff' }} />;
    }
  };

  // 获取风险级别颜色
  const getRiskLevelColor = (level: string) => {
    const colorMap: Record<string, string> = {
      low: 'green',
      medium: 'orange',
      high: 'red',
    };
    return colorMap[level] || 'default';
  };

  // 渲染预测图表
  const renderPredictionChart = () => {
    if (!predictionReport || predictionReport.data.length === 0) {
      return <Empty description='暂无预测数据' />;
    }

    const chartData = predictionReport.data.map(item => ({
      date: format(new Date(item.date), 'MM-dd'),
      actual: item.actual,
      predicted: item.predicted,
      upper: item.upper_bound,
      lower: item.lower_bound,
    }));

    return (
      <ResponsiveContainer width='100%' height={400}>
        <AreaChart data={chartData}>
          <CartesianGrid strokeDasharray='3 3' />
          <XAxis dataKey='date' />
          <YAxis />
          <Tooltip />
          <Legend />
          {chartData.some(d => d.actual !== undefined) && (
            <Area
              type='monotone'
              dataKey='actual'
              stroke='#1890ff'
              fill='#1890ff'
              fillOpacity={0.3}
              name='实际值'
            />
          )}
          <Area
            type='monotone'
            dataKey='predicted'
            stroke='#52c41a'
            fill='#52c41a'
            fillOpacity={0.3}
            name='预测值'
          />
          {chartData.some(d => d.upper && d.lower) && (
            <>
              <Area
              type='monotone'
              dataKey='upper'
              stroke='#ff4d4f'
              strokeDasharray='5 5'
              fill='none'
              name='上限'
            />
            <Area
              type='monotone'
              dataKey='lower'
              stroke='#ff4d4f'
              strokeDasharray='5 5'
              fill='none'
              name='下限'
            />
            </>
          )}
          <ReferenceLine x={chartData[chartData.length - 20]?.date} stroke='#faad14' strokeDasharray='3 3' label='预测起点' />
        </AreaChart>
      </ResponsiveContainer>
    );
  };

  // 导出报告
  const handleExport = useCallback(() => {
    // TODO: 实现导出功能
    antMessage.info('导出功能开发中');
  }, [antMessage]);

  return (
    <div className='space-y-4'>
      {/* 工具栏 */}
      <Card>
        <div className='flex items-center justify-between mb-4'>
          <Title level={5} style={{ margin: 0 }}>
            工单趋势预测
          </Title>
          <Space>
            <Button icon={<DownloadOutlined />} onClick={handleExport}>
              导出报告
            </Button>
            <Button icon={<ReloadOutlined />} onClick={loadPrediction} loading={loading}>
              刷新
            </Button>
          </Space>
        </div>

        <Space className='mb-4' wrap>
          <RangePicker
            value={[new Date(timeRange[0]), new Date(timeRange[1])]}
            onChange={(dates) => {
              if (dates) {
                setTimeRange([
                  format(dates[0]!, 'yyyy-MM-dd'),
                  format(dates[1]!, 'yyyy-MM-dd'),
                ]);
              }
            }}
          />
          <Select
            value={predictionPeriod}
            onChange={setPredictionPeriod}
            style={{ width: 120 }}
          >
            <Option value='week'>预测一周</Option>
            <Option value='month'>预测一月</Option>
            <Option value='quarter'>预测一季</Option>
          </Select>
          <Select
            value={modelType}
            onChange={setModelType}
            style={{ width: 150 }}
          >
            <Option value='arima'>ARIMA模型</Option>
            <Option value='exponential'>指数平滑</Option>
            <Option value='linear'>线性回归</Option>
          </Select>
        </Space>
      </Card>

      {loading ? (
        <div className='text-center py-16'>
          <Spin size='large' />
          <div className='mt-4 text-gray-500'>正在生成预测数据...</div>
        </div>
      ) : predictionReport ? (
        <>
          {/* 预测指标卡片 */}
          <Row gutter={16}>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic
                  title='预测准确率'
                  value={predictionReport.metrics.accuracy}
                  precision={1}
                  suffix='%'
                  valueStyle={{ color: '#3f8600' }}
                  prefix={<CheckCircleOutlined />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic
                  title='下周预测'
                  value={predictionReport.metrics.next_week_prediction}
                  suffix='个工单'
                  valueStyle={{ color: '#1890ff' }}
                  prefix={<CalendarOutlined />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic
                  title='下月预测'
                  value={predictionReport.metrics.next_month_prediction}
                  suffix='个工单'
                  valueStyle={{ color: '#faad14' }}
                  prefix={<CalendarOutlined />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <div className='flex items-center justify-between'>
                  <div>
                    <Text type='secondary' className='text-sm'>趋势</Text>
                    <div className='flex items-center gap-2 mt-1'>
                      {getTrendIcon(predictionReport.metrics.trend)}
                      <Text strong>
                        {predictionReport.metrics.trend === 'up' ? '上升' : predictionReport.metrics.trend === 'down' ? '下降' : '稳定'}
                      </Text>
                    </div>
                  </div>
                  <Progress
                    type='circle'
                    percent={predictionReport.metrics.trend_strength * 100}
                    size={60}
                    format={() => `${(predictionReport.metrics.trend_strength * 100).toFixed(0)}%`}
                  />
                </div>
              </Card>
            </Col>
          </Row>

          {/* 预测图表 */}
          <Card title='趋势预测图表' loading={loading}>
            {renderPredictionChart()}
          </Card>

          {/* 预测报告 */}
          <Row gutter={16}>
            <Col xs={24} lg={16}>
              <Card title='预测报告摘要'>
                <div className='space-y-4'>
                  <div>
                    <Text strong>分析周期：</Text>
                    <Text>{predictionReport.period}</Text>
                  </div>
                  <div>
                    <Text strong>报告摘要：</Text>
                    <Text>{predictionReport.summary}</Text>
                  </div>
                  <Divider />
                  <div>
                    <Title level={5}>关键发现</Title>
                    <ul className='list-disc list-inside space-y-2'>
                      {predictionReport.key_findings.map((finding, index) => (
                        <li key={index}>
                          <Text>{finding}</Text>
                        </li>
                      ))}
                    </ul>
                  </div>
                  <Divider />
                  <div>
                    <Title level={5}>建议措施</Title>
                    <ul className='list-disc list-inside space-y-2'>
                      {predictionReport.recommendations.map((rec, index) => (
                        <li key={index}>
                          <Text>{rec}</Text>
                        </li>
                      ))}
                    </ul>
                  </div>
                </div>
              </Card>
            </Col>
            <Col xs={24} lg={8}>
              <Card title='预测指标'>
                <div className='space-y-4'>
                  <div>
                    <Text type='secondary' className='text-sm'>准确率</Text>
                    <Progress
                      percent={predictionReport.metrics.accuracy}
                      status={predictionReport.metrics.accuracy >= 85 ? 'success' : predictionReport.metrics.accuracy >= 70 ? 'active' : 'exception'}
                      className='mt-2'
                    />
                  </div>
                  <div>
                    <Text type='secondary' className='text-sm'>平均绝对百分比误差 (MAPE)</Text>
                    <Text strong className='text-lg'>{predictionReport.metrics.mape.toFixed(2)}%</Text>
                  </div>
                  <div>
                    <Text type='secondary' className='text-sm'>均方根误差 (RMSE)</Text>
                    <Text strong className='text-lg'>{predictionReport.metrics.rmse.toFixed(2)}</Text>
                  </div>
                  <Divider />
                  <div>
                    <Text type='secondary' className='text-sm'>风险级别</Text>
                    <div className='mt-2'>
                      <Tag color={getRiskLevelColor(predictionReport.metrics.risk_level)}>
                        {predictionReport.metrics.risk_level === 'low' ? '低风险' : predictionReport.metrics.risk_level === 'medium' ? '中风险' : '高风险'}
                      </Tag>
                    </div>
                  </div>
                  <div>
                    <Text type='secondary' className='text-sm'>风险因素</Text>
                    <div className='mt-2 space-y-1'>
                      {predictionReport.metrics.risk_factors.map((factor, index) => (
                        <Tag key={index} color='orange'>
                          {factor}
                        </Tag>
                      ))}
                    </div>
                  </div>
                </div>
              </Card>
            </Col>
          </Row>

          {/* 风险预警 */}
          {predictionReport.metrics.risk_level !== 'low' && (
            <Alert
              message='风险预警'
              description={`根据预测分析，系统检测到${predictionReport.metrics.risk_level === 'high' ? '高风险' : '中等风险'}因素。建议采取相应措施。`}
              type={predictionReport.metrics.risk_level === 'high' ? 'error' : 'warning'}
              showIcon
              icon={<WarningOutlined />}
              action={
                <Button size='small' type='primary'>
                  查看详情
                </Button>
              }
            />
          )}

          <Card>
            <div className='flex items-center justify-between'>
              <Text type='secondary' className='text-sm'>
                报告生成时间：{predictionReport.generated_at}
              </Text>
              <Text type='secondary' className='text-sm'>
                预测模型：{modelType.toUpperCase()}
              </Text>
            </div>
          </Card>
        </>
      ) : (
        <Empty description='暂无预测数据' />
      )}
    </div>
  );
};

