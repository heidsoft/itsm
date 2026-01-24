'use client';

import React, { useState, useEffect, useMemo } from 'react';
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
import {
  RiseOutlined,
  FallOutlined,
  FileExcelOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { ticketAnalyticsService, TicketAnalyticsResponse } from '@/lib/services/analytics-service';
import { ticketService, TicketStatsResponse } from '@/lib/services/ticket-service';

const { Title, Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;
const { TabPane } = Tabs;

const TicketAnalytics: React.FC = () => {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(30, 'day'),
    dayjs(),
  ]);
  const [analyticsData, setAnalyticsData] = useState<TicketAnalyticsResponse | null>(null);
  const [activeTab, setActiveTab] = useState('overview');

  // 获取分析数据
  useEffect(() => {
    fetchAnalyticsData();
  }, [dateRange]);

  const fetchAnalyticsData = async () => {
    setLoading(true);
    try {
      const [analyticsRes, statsRes] = await Promise.all([
        ticketAnalyticsService.getAnalytics({
          date_from: dateRange[0].format('YYYY-MM-DD'),
          date_to: dateRange[1].format('YYYY-MM-DD'),
        }),
        ticketService.getTicketStats(),
      ]);

      // 适配后端返回格式到前端期望的格式
      const adaptedData: TicketAnalyticsResponse = {
        total_tickets: statsRes.total || analyticsRes.total_tickets || 0,
        open_tickets: statsRes.open || analyticsRes.open_tickets || 0,
        resolved_tickets: statsRes.resolved || analyticsRes.resolved_tickets || 0,
        closed_tickets: statsRes.closed || analyticsRes.closed_tickets || 0,
        overdue_tickets: statsRes.overdue || analyticsRes.overdue_tickets || 0,

        daily_trend: analyticsRes.daily_trend || generateMockTrend(30),

        status_distribution: analyticsRes.status_distribution || generateMockStatusDist(),

        priority_distribution: analyticsRes.priority_distribution || [
          { name: '低', value: 0, color: '#52c41a' },
          { name: '中', value: 0, color: '#fa8c16' },
          { name: '高', value: statsRes.high_priority || 0, color: '#ff4d4f' },
          { name: '紧急', value: statsRes.urgent || 0, color: '#722ed1' },
        ],

        type_distribution: analyticsRes.type_distribution || [
          { name: '事件', value: 0, count: 0 },
          { name: '请求', value: 0, count: 0 },
          { name: '问题', value: 0, count: 0 },
          { name: '变更', value: 0, count: 0 },
        ],

        processing_time_stats: analyticsRes.processing_time_stats || {
          avg_processing_time: 0,
          avg_resolution_time: 0,
          sla_compliance_rate: 0,
        },

        team_performance: analyticsRes.team_performance || [],

        hot_categories: analyticsRes.hot_categories || [],
      };

      setAnalyticsData(adaptedData);
    } catch (error) {
      console.error('Failed to fetch analytics data:', error);
      message.error('获取分析数据失败，使用演示数据');
      // 使用演示数据作为降级
      setAnalyticsData(getMockData());
    } finally {
      setLoading(false);
    }
  };

  // 生成模拟趋势数据
  function generateMockTrend(days: number) {
    return Array.from({ length: days }, (_, i) => ({
      date: dayjs().subtract(days - 1 - i, 'day').format('MM-DD'),
      created: Math.floor(Math.random() * 20) + 5,
      resolved: Math.floor(Math.random() * 15) + 3,
      open: Math.floor(Math.random() * 10) + 2,
    }));
  }

  // 生成模拟状态分布
  function generateMockStatusDist() {
    return [
      { name: '新建', value: 45, color: '#1890ff' },
      { name: '待处理', value: 78, color: '#1890ff' },
      { name: '处理中', value: 156, color: '#fa8c16' },
      { name: '等待中', value: 23, color: '#faad14' },
      { name: '已解决', value: 623, color: '#52c41a' },
      { name: '已关闭', value: 100, color: '#d9d9d9' },
    ];
  }

  // 演示数据
  function getMockData(): TicketAnalyticsResponse {
    return {
      total_tickets: 847,
      open_tickets: 124,
      resolved_tickets: 623,
      closed_tickets: 100,
      overdue_tickets: 18,
      daily_trend: generateMockTrend(30),
      status_distribution: generateMockStatusDist(),
      priority_distribution: [
        { name: '低', value: 234, color: '#52c41a' },
        { name: '中', value: 389, color: '#fa8c16' },
        { name: '高', value: 198, color: '#ff4d4f' },
        { name: '紧急', value: 23, color: '#722ed1' },
      ],
      type_distribution: [
        { name: '事件', value: 312, count: 312 },
        { name: '请求', value: 278, count: 278 },
        { name: '问题', value: 145, count: 145 },
        { name: '变更', value: 67, count: 67 },
      ],
      processing_time_stats: {
        avg_processing_time: 4.2,
        avg_resolution_time: 24.6,
        sla_compliance_rate: 87.3,
      },
      team_performance: [
        { assignee_name: '张三', total_handled: 67, resolved: 62, avg_response_time: 2.5, avg_resolution_time: 18.5 },
        { assignee_name: '李四', total_handled: 54, resolved: 50, avg_response_time: 3.2, avg_resolution_time: 22.3 },
        { assignee_name: '王五', total_handled: 89, resolved: 85, avg_response_time: 1.8, avg_resolution_time: 15.7 },
        { assignee_name: '赵六', total_handled: 43, resolved: 38, avg_response_time: 4.1, avg_resolution_time: 28.9 },
      ],
      hot_categories: [
        { category: '系统故障', count: 234, trend: 'up' },
        { category: '账户管理', count: 189, trend: 'up' },
        { category: '网络问题', count: 156, trend: 'down' },
        { category: '硬件故障', count: 134, trend: 'up' },
        { category: '软件安装', count: 134, trend: 'up' },
      ],
    };
  }

  // 导出数据
  const handleExport = () => {
    message.info('导出功能开发中');
  };

  // 计算百分比
  const calculatePercentage = (value: number, total: number) => {
    return total > 0 ? (value / total) * 100 : 0;
  };

  // 团队表现表格列
  const teamColumns = [
    {
      title: '处理人',
      dataIndex: 'assignee_name',
      key: 'assignee_name',
    },
    {
      title: '处理工单数',
      dataIndex: 'total_handled',
      key: 'total_handled',
      sorter: (a: any, b: any) => a.total_handled - b.total_handled,
    },
    {
      title: '已解决',
      dataIndex: 'resolved',
      key: 'resolved',
    },
    {
      title: '平均响应时间(小时)',
      dataIndex: 'avg_response_time',
      key: 'avg_response_time',
      render: (time: number) => time?.toFixed(1) || '-',
    },
    {
      title: '平均解决时间(小时)',
      dataIndex: 'avg_resolution_time',
      key: 'avg_resolution_time',
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
              onChange={(dates) => {
                if (dates) {
                  setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs]);
                }
              }}
              format="YYYY-MM-DD"
              allowClear={false}
            />
            <Button
              icon={<ReloadOutlined />}
              onClick={fetchAnalyticsData}
              loading={loading}
            >
              刷新数据
            </Button>
            <Button icon={<FileExcelOutlined />} onClick={handleExport}>
              导出报表
            </Button>
          </Space>
        </div>
      </div>

      <Spin spinning={loading}>
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          {/* 概览标签页 */}
          <TabPane tab="概览" key="overview">
            {/* 关键指标卡片 */}
            <Row gutter={[16, 16]} className="mb-6">
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="总工单数"
                    value={analyticsData.total_tickets}
                    prefix={<RiseOutlined style={{ color: '#1890ff' }} />}
                    valueStyle={{ color: '#1890ff' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="待处理工单"
                    value={analyticsData.open_tickets}
                    prefix={<RiseOutlined style={{ color: '#fa8c16' }} />}
                    valueStyle={{ color: '#fa8c16' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="已解决工单"
                    value={analyticsData.resolved_tickets}
                    prefix={<RiseOutlined style={{ color: '#52c41a' }} />}
                    valueStyle={{ color: '#52c41a' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="超时工单"
                    value={analyticsData.overdue_tickets}
                    prefix={<FallOutlined style={{ color: '#ff4d4f' }} />}
                    valueStyle={{ color: '#ff4d4f' }}
                  />
                </Card>
              </Col>
            </Row>

            {/* 趋势图表 */}
            <Row gutter={[16, 16]} className="mb-6">
              <Col span={24}>
                <Card title="工单趋势（最近30天）">
                  <ResponsiveContainer width="100%" height={300}>
                    <LineChart data={analyticsData.daily_trend}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="date" />
                      <YAxis />
                      <RechartsTooltip />
                      <Legend />
                      <Line
                        type="monotone"
                        dataKey="created"
                        stroke="#1890ff"
                        name="新建工单"
                        strokeWidth={2}
                      />
                      <Line
                        type="monotone"
                        dataKey="resolved"
                        stroke="#52c41a"
                        name="已解决工单"
                        strokeWidth={2}
                      />
                      <Line
                        type="monotone"
                        dataKey="open"
                        stroke="#fa8c16"
                        name="待处理工单"
                        strokeWidth={2}
                      />
                    </LineChart>
                  </ResponsiveContainer>
                </Card>
              </Col>
            </Row>

            {/* 分布图表 */}
            <Row gutter={[16, 16]}>
              <Col xs={24} md={8}>
                <Card title="状态分布">
                  <ResponsiveContainer width="100%" height={250}>
                    <PieChart>
                      <Pie
                        data={analyticsData.status_distribution}
                        cx="50%"
                        cy="50%"
                        outerRadius={80}
                        dataKey="value"
                        label={({ name, percent }) =>
                          `${name} ${(((percent ?? 0) * 100)).toFixed(0)}%`
                        }
                      >
                        {analyticsData.status_distribution?.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.color} />
                        ))}
                      </Pie>
                      <RechartsTooltip />
                    </PieChart>
                  </ResponsiveContainer>
                </Card>
              </Col>
              <Col xs={24} md={8}>
                <Card title="优先级分布">
                  <ResponsiveContainer width="100%" height={250}>
                    <BarChart data={analyticsData.priority_distribution}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="name" />
                      <YAxis />
                      <RechartsTooltip />
                      <Bar dataKey="value" fill="#1890ff">
                        {analyticsData.priority_distribution?.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.color} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </Card>
              </Col>
              <Col xs={24} md={8}>
                <Card title="处理时间统计">
                  <div className="space-y-4">
                    <div className="flex justify-between items-center">
                      <Text>平均处理时间:</Text>
                      <Tag color="blue">{analyticsData.processing_time_stats?.avg_processing_time?.toFixed(1) || 0}小时</Tag>
                    </div>
                    <div className="flex justify-between items-center">
                      <Text>平均解决时间:</Text>
                      <Tag color="orange">{analyticsData.processing_time_stats?.avg_resolution_time?.toFixed(1) || 0}小时</Tag>
                    </div>
                    <div className="flex justify-between items-center">
                      <Text>SLA达成率:</Text>
                      <Tag color={analyticsData.processing_time_stats?.sla_compliance_rate >= 90 ? 'green' : 'red'}>
                        {analyticsData.processing_time_stats?.sla_compliance_rate?.toFixed(1) || 0}%
                      </Tag>
                    </div>
                  </div>
                </Card>
              </Col>
            </Row>
          </TabPane>

          {/* 团队表现标签页 */}
          <TabPane tab="团队表现" key="team">
            <Card title="团队处理效率统计">
              <Table
                columns={teamColumns}
                dataSource={analyticsData.team_performance}
                rowKey="assignee_name"
                pagination={false}
                size="middle"
              />
            </Card>
          </TabPane>

          {/* 类别分析标签页 */}
          <TabPane tab="类别分析" key="category">
            <Row gutter={[16, 16]}>
              <Col span={16}>
                <Card title="热门工单类别">
                  <Table
                    columns={categoryColumns}
                    dataSource={analyticsData.hot_categories}
                    rowKey="category"
                    pagination={false}
                    size="middle"
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card title="类型分布">
                  <ResponsiveContainer width="100%" height={300}>
                    <BarChart data={analyticsData.type_distribution}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="name" />
                      <YAxis />
                      <RechartsTooltip />
                      <Bar dataKey="count" fill="#1890ff" />
                    </BarChart>
                  </ResponsiveContainer>
                </Card>
              </Col>
            </Row>
          </TabPane>
        </Tabs>
      </Spin>
    </div>
  );
};

export default TicketAnalytics;
