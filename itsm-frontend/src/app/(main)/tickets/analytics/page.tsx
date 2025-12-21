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
  Alert,
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
  AreaChart,
  Area,
} from 'recharts';
import {
  RiseOutlined,
  FallOutlined,
  FileExcelOutlined,
  ReloadOutlined,
  FilterOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useRouter } from 'next/navigation';

const { Title, Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;
const { TabPane } = Tabs;

interface TicketAnalytics {
  // 基础统计
  totalTickets: number;
  openTickets: number;
  resolvedTickets: number;
  closedTickets: number;
  overdueTickets: number;

  // 趋势数据
  dailyTrend: Array<{
    date: string;
    created: number;
    resolved: number;
    open: number;
  }>;

  // 状态分布
  statusDistribution: Array<{
    name: string;
    value: number;
    color: string;
  }>;

  // 优先级分布
  priorityDistribution: Array<{
    name: string;
    value: number;
    color: string;
  }>;

  // 类型分布
  typeDistribution: Array<{
    name: string;
    value: number;
    count: number;
  }>;

  // 处理时间统计
  processingTimeStats: {
    avgProcessingTime: number;
    avgResolutionTime: number;
    slaComplianceRate: number;
  };

  // 团队表现
  teamPerformance: Array<{
    assigneeName: string;
    totalHandled: number;
    avgResolutionTime: number;
    satisfactionRate: number;
  }>;

  // 热门类别
  topCategories: Array<{
    category: string;
    count: number;
    percentage: number;
  }>;
}

const TicketAnalytics: React.FC = () => {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(30, 'day'),
    dayjs(),
  ]);
  const [analyticsData, setAnalyticsData] = useState<TicketAnalytics | null>(null);
  const [activeTab, setActiveTab] = useState('overview');

  // 模拟获取分析数据
  useEffect(() => {
    fetchAnalyticsData();
  }, [dateRange]);

  const fetchAnalyticsData = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));

      const mockData: TicketAnalytics = {
        totalTickets: 847,
        openTickets: 124,
        resolvedTickets: 623,
        closedTickets: 100,
        overdueTickets: 18,

        dailyTrend: Array.from({ length: 30 }, (_, i) => ({
          date: dayjs().subtract(29 - i, 'day').format('MM-DD'),
          created: Math.floor(Math.random() * 20) + 5,
          resolved: Math.floor(Math.random() * 15) + 3,
          open: Math.floor(Math.random() * 10) + 2,
        })),

        statusDistribution: [
          { name: '新建', value: 45, color: '#1890ff' },
          { name: '待处理', value: 78, color: '#1890ff' },
          { name: '处理中', value: 156, color: '#fa8c16' },
          { name: '等待中', value: 23, color: '#faad14' },
          { name: '已解决', value: 623, color: '#52c41a' },
          { name: '已关闭', value: 100, color: '#d9d9d9' },
        ],

        priorityDistribution: [
          { name: '低', value: 234, color: '#52c41a' },
          { name: '中', value: 389, color: '#fa8c16' },
          { name: '高', value: 198, color: '#ff4d4f' },
          { name: '紧急', value: 23, color: '#722ed1' },
          { name: '严重', value: 3, color: '#ff4d4f' },
        ],

        typeDistribution: [
          { name: '事件', value: 312, count: 312 },
          { name: '请求', value: 278, count: 278 },
          { name: '问题', value: 145, count: 145 },
          { name: '变更', value: 67, count: 67 },
          { name: '任务', value: 45, count: 45 },
        ],

        processingTimeStats: {
          avgProcessingTime: 4.2,
          avgResolutionTime: 24.6,
          slaComplianceRate: 87.3,
        },

        teamPerformance: [
          { assigneeName: '张三', totalHandled: 67, avgResolutionTime: 18.5, satisfactionRate: 4.6 },
          { assigneeName: '李四', totalHandled: 54, avgResolutionTime: 22.3, satisfactionRate: 4.4 },
          { assigneeName: '王五', totalHandled: 89, avgResolutionTime: 15.7, satisfactionRate: 4.8 },
          { assigneeName: '赵六', totalHandled: 43, avgResolutionTime: 28.9, satisfactionRate: 4.2 },
        ],

        topCategories: [
          { category: '系统故障', count: 234, percentage: 27.6 },
          { category: '账户管理', count: 189, percentage: 22.3 },
          { category: '网络问题', count: 156, percentage: 18.4 },
          { category: '硬件故障', count: 134, percentage: 15.8 },
          { category: '软件安装', count: 134, percentage: 15.9 },
        ],
      };

      setAnalyticsData(mockData);
    } catch (error) {
      console.error('Failed to fetch analytics data:', error);
    } finally {
      setLoading(false);
    }
  };

  // 导出数据
  const handleExport = () => {
    // 模拟导出功能
    console.log('Exporting analytics data...');
  };

  // 团队表现表格列
  const teamColumns = [
    {
      title: '处理人',
      dataIndex: 'assigneeName',
      key: 'assigneeName',
    },
    {
      title: '处理工单数',
      dataIndex: 'totalHandled',
      key: 'totalHandled',
      sorter: (a: any, b: any) => a.totalHandled - b.totalHandled,
    },
    {
      title: '平均解决时间(小时)',
      dataIndex: 'avgResolutionTime',
      key: 'avgResolutionTime',
      render: (time: number) => time.toFixed(1),
    },
    {
      title: '满意度评分',
      dataIndex: 'satisfactionRate',
      key: 'satisfactionRate',
      render: (rate: number) => (
        <div className="flex items-center">
          <span className="mr-2">{rate.toFixed(1)}</span>
          <Progress
            percent={rate * 20}
            size="small"
            format={() => ''}
            strokeColor={rate >= 4.5 ? '#52c41a' : rate >= 4.0 ? '#fa8c16' : '#ff4d4f'}
          />
        </div>
      ),
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
      title: '占比',
      dataIndex: 'percentage',
      key: 'percentage',
      render: (percentage: number) => (
        <div className="flex items-center">
          <span className="mr-2">{percentage.toFixed(1)}%</span>
          <Progress
            percent={percentage}
            size="small"
            format={() => ''}
            strokeColor="#1890ff"
          />
        </div>
      ),
    },
  ];

  if (!analyticsData) {
    return <div>加载中...</div>;
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
              onChange={setDateRange}
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

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        {/* 概览标签页 */}
        <TabPane tab="概览" key="overview">
          {/* 关键指标卡片 */}
          <Row gutter={[16, 16]} className="mb-6">
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic
                  title="总工单数"
                  value={analyticsData.totalTickets}
                  prefix={<RiseOutlined style={{ color: '#1890ff' }} />}
                  valueStyle={{ color: '#1890ff' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic
                  title="待处理工单"
                  value={analyticsData.openTickets}
                  prefix={<RiseOutlined style={{ color: '#fa8c16' }} />}
                  valueStyle={{ color: '#fa8c16' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic
                  title="已解决工单"
                  value={analyticsData.resolvedTickets}
                  prefix={<RiseOutlined style={{ color: '#52c41a' }} />}
                  valueStyle={{ color: '#52c41a' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic
                  title="超时工单"
                  value={analyticsData.overdueTickets}
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
                  <LineChart data={analyticsData.dailyTrend}>
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
                      data={analyticsData.statusDistribution}
                      cx="50%"
                      cy="50%"
                      outerRadius={80}
                      dataKey="value"
                      label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                    >
                      {analyticsData.statusDistribution.map((entry, index) => (
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
                  <BarChart data={analyticsData.priorityDistribution}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <RechartsTooltip />
                    <Bar dataKey="value" fill="#1890ff">
                      {analyticsData.priorityDistribution.map((entry, index) => (
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
                    <Tag color="blue">{analyticsData.processingTimeStats.avgProcessingTime}小时</Tag>
                  </div>
                  <div className="flex justify-between items-center">
                    <Text>平均解决时间:</Text>
                    <Tag color="orange">{analyticsData.processingTimeStats.avgResolutionTime}小时</Tag>
                  </div>
                  <div className="flex justify-between items-center">
                    <Text>SLA达成率:</Text>
                    <Tag color={analyticsData.processingTimeStats.slaComplianceRate >= 90 ? 'green' : 'red'}>
                      {analyticsData.processingTimeStats.slaComplianceRate}%
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
              dataSource={analyticsData.teamPerformance}
              rowKey="assigneeName"
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
                  dataSource={analyticsData.topCategories}
                  rowKey="category"
                  pagination={false}
                  size="middle"
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card title="类型分布">
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={analyticsData.typeDistribution}>
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
    </div>
  );
};

export default TicketAnalytics;