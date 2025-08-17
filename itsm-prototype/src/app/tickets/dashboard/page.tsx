"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Row,
  Col,
  Statistic,
  Progress,
  Table,
  Tag,
  Typography,
  Space,
  Button,
  Select,
  DatePicker,
  Tabs,
  List,
  Avatar,
  Badge,
  Tooltip,
  Alert,
  Divider,
  Timeline,
  Steps,
  Descriptions,
  Rate,
  Slider,
  Switch,
  Radio,
  Checkbox,
  Input,
  InputNumber,
  Form,
  Modal,
  message,
  notification,
  Spin,
  Empty,
  Result,
  Skeleton,
  Drawer,
  Popconfirm,
  Transfer,
  TreeSelect,
  Cascader,
  Mentions,
  AutoComplete,
  Divider as AntdDivider,
} from "antd";
import {
  TrendingUp,
  TrendingDown,
  Clock,
  AlertTriangle,
  CheckCircle,
  FileText,
  Users,
  BarChart3,
  PieChart,
  Activity,
  Target,
  Shield,
  Zap,
  Settings,
  RefreshCw,
  Download,
  Filter,
  Search,
  Eye,
  Edit,
  Delete,
  Plus,
  MoreHorizontal,
  ArrowUp,
  ArrowDown,
  Minus,
  Star,
  Heart,
  ThumbsUp,
  ThumbsDown,
  MessageSquare,
  Share2,
  Lock,
  Unlock,
  EyeOff,
  EyeOn,
  ArrowRight,
  ArrowLeft,
  RotateCcw,
  Save,
  Play,
  Pause,
  Stop,
  SkipBack,
  SkipForward,
  Repeat,
  Shuffle,
  Volume2,
  VolumeX,
  Mic,
  MicOff,
  Video,
  VideoOff,
  Phone,
  PhoneOff,
  Mail,
  Send,
  Trash2,
  Archive,
  Tag as TagIcon,
  Hash,
  AtSign,
  HashIcon,
  HashTag,
  HashTagIcon,
  HashTagIcon2,
  HashTagIcon3,
  HashTagIcon4,
  HashTagIcon5,
  HashTagIcon6,
  HashTagIcon7,
  HashTagIcon8,
  HashTagIcon9,
  HashTagIcon10,
  Calendar,
  Clock as ClockIcon,
  User,
  UserCheck,
  UserX,
  UserPlus,
  UserMinus,
  UserEdit,
  UserCog,
  UserShield,
  UserVoice,
  UserSearch,
  UserStar,
  UserHeart,
  UserCheckCircle,
  UserXCircle,
  UserClock,
  UserAlert,
  UserSettings,
  UserCog2,
  UserCog3,
  UserCog4,
  UserCog5,
  UserCog6,
  UserCog7,
  UserCog8,
  UserCog9,
  UserCog10,
} from "lucide-react";
// AppLayout is handled by layout.tsx

const { Option } = Select;
const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;
const { Step } = Steps;
const { RangePicker } = DatePicker;
const { TextArea } = Input;

// 模拟数据
const mockData = {
  overview: {
    totalTickets: 1247,
    openTickets: 89,
    resolvedTickets: 1158,
    avgResolutionTime: "8.7小时",
    slaCompliance: 94.2,
    customerSatisfaction: 4.3,
    teamPerformance: 87.5,
    monthlyGrowth: 12.5,
  },
  slaMetrics: {
    totalSLA: 1247,
    withinSLA: 1174,
    breachedSLA: 73,
    avgResponseTime: "2.3小时",
    avgResolutionTime: "8.7小时",
    criticalSLA: 15,
    highPrioritySLA: 42,
    mediumPrioritySLA: 156,
    lowPrioritySLA: 1034,
  },
  teamPerformance: [
    {
      id: 1,
      name: "张三",
      avatar: "张",
      role: "技术支持",
      ticketsAssigned: 45,
      ticketsResolved: 42,
      avgResolutionTime: "6.2小时",
      slaCompliance: 96.8,
      customerRating: 4.5,
      efficiency: 93.3,
    },
    {
      id: 2,
      name: "李四",
      avatar: "李",
      role: "高级工程师",
      ticketsAssigned: 38,
      ticketsResolved: 36,
      avgResolutionTime: "7.8小时",
      slaCompliance: 94.7,
      customerRating: 4.3,
      efficiency: 89.5,
    },
    {
      id: 3,
      name: "王五",
      avatar: "王",
      role: "项目经理",
      ticketsAssigned: 32,
      ticketsResolved: 30,
      avgResolutionTime: "9.1小时",
      slaCompliance: 93.8,
      customerRating: 4.4,
      efficiency: 87.2,
    },
  ],
  recentActivities: [
    {
      id: 1,
      action: "工单已解决",
      ticketNumber: "TICKET-001",
      user: "张三",
      timestamp: "2024-01-15 14:30",
      details: "系统登录问题已解决，用户反馈良好",
    },
    {
      id: 2,
      action: "工单已分配",
      ticketNumber: "TICKET-002",
      user: "李四",
      timestamp: "2024-01-15 14:25",
      details: "打印机故障工单已分配给硬件团队",
    },
    {
      id: 3,
      action: "SLA预警",
      ticketNumber: "TICKET-003",
      user: "系统",
      timestamp: "2024-01-15 14:20",
      details: "高优先级工单即将超时，请及时处理",
    },
  ],
  trends: {
    daily: [12, 19, 15, 25, 22, 30, 28, 35, 32, 40, 38, 45],
    weekly: [85, 92, 78, 105, 98, 115, 108],
    monthly: [320, 345, 298, 378, 356, 412, 389, 445, 423, 478, 456, 512],
  },
};

const TicketDashboardPage = () => {
  const [loading, setLoading] = useState(false);
  const [timeRange, setTimeRange] = useState("7d");
  const [selectedTeam, setSelectedTeam] = useState("all");
  const [activeTab, setActiveTab] = useState("overview");
  const [slaThreshold, setSlaThreshold] = useState(95);
  const [showAdvancedMetrics, setShowAdvancedMetrics] = useState(false);

  useEffect(() => {
    loadDashboardData();
  }, [timeRange, selectedTeam]);

  const loadDashboardData = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise((resolve) => setTimeout(resolve, 1000));
    } catch (error) {
      message.error("加载仪表板数据失败");
    } finally {
      setLoading(false);
    }
  };

  const renderOverviewCards = () => (
    <Row gutter={16} className="mb-6">
      <Col span={6}>
        <Card>
          <Statistic
            title="总工单数"
            value={mockData.overview.totalTickets}
            prefix={<FileText size={16} style={{ color: "#3b82f6" }} />}
            suffix={
              <div className="text-xs text-gray-500 mt-1">
                <div className="flex items-center">
                  <TrendingUp size={12} className="text-green-500 mr-1" />
                  <span className="text-green-500">
                    +{mockData.overview.monthlyGrowth}%
                  </span>
                </div>
                <div>本月增长</div>
              </div>
            }
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="待处理工单"
            value={mockData.overview.openTickets}
            valueStyle={{ color: "#faad14" }}
            prefix={<Clock size={16} />}
            suffix={
              <div className="text-xs text-gray-500 mt-1">
                <div>平均响应: 2.3h</div>
                <div className="text-orange-500">SLA: 4h</div>
              </div>
            }
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="已解决工单"
            value={mockData.overview.resolvedTickets}
            valueStyle={{ color: "#52c41a" }}
            prefix={<CheckCircle size={16} />}
            suffix={
              <div className="text-xs text-gray-500 mt-1">
                <div>平均解决: {mockData.overview.avgResolutionTime}</div>
                <div className="text-green-500">
                  满意度: {mockData.overview.customerSatisfaction}/5
                </div>
              </div>
            }
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="SLA合规率"
            value={mockData.overview.slaCompliance}
            valueStyle={{ color: "#52c41a" }}
            prefix={<Target size={16} />}
            suffix="%"
          />
          <Progress
            percent={mockData.overview.slaCompliance}
            strokeColor="#52c41a"
            showInfo={false}
            className="mt-2"
          />
        </Card>
      </Col>
    </Row>
  );

  const renderSLAMetrics = () => (
    <Row gutter={16} className="mb-6">
      <Col span={8}>
        <Card title="SLA概览" className="h-full">
          <div className="text-center mb-4">
            <div className="text-3xl font-bold text-blue-600 mb-2">
              {mockData.slaMetrics.withinSLA}
            </div>
            <Text type="secondary">符合SLA</Text>
          </div>
          <Progress
            percent={
              (mockData.slaMetrics.withinSLA / mockData.slaMetrics.totalSLA) *
              100
            }
            strokeColor="#52c41a"
            showInfo={false}
          />
          <div className="mt-4 space-y-2">
            <div className="flex justify-between text-sm">
              <span>总工单: {mockData.slaMetrics.totalSLA}</span>
              <span>违规: {mockData.slaMetrics.breachedSLA}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span>平均响应: {mockData.slaMetrics.avgResponseTime}</span>
              <span>平均解决: {mockData.slaMetrics.avgResolutionTime}</span>
            </div>
          </div>
        </Card>
      </Col>
      <Col span={8}>
        <Card title="优先级分布" className="h-full">
          <div className="space-y-4">
            <div>
              <div className="flex justify-between mb-2">
                <Text>紧急</Text>
                <Text strong>{mockData.slaMetrics.criticalSLA}</Text>
              </div>
              <Progress
                percent={
                  (mockData.slaMetrics.criticalSLA /
                    mockData.slaMetrics.totalSLA) *
                  100
                }
                strokeColor="#ff4d4f"
                showInfo={false}
              />
            </div>
            <div>
              <div className="flex justify-between mb-2">
                <Text>高</Text>
                <Text strong>{mockData.slaMetrics.highPrioritySLA}</Text>
              </div>
              <Progress
                percent={
                  (mockData.slaMetrics.highPrioritySLA /
                    mockData.slaMetrics.totalSLA) *
                  100
                }
                strokeColor="#faad14"
                showInfo={false}
              />
            </div>
            <div>
              <div className="flex justify-between mb-2">
                <Text>中</Text>
                <Text strong>{mockData.slaMetrics.mediumPrioritySLA}</Text>
              </div>
              <Progress
                percent={
                  (mockData.slaMetrics.mediumPrioritySLA /
                    mockData.slaMetrics.totalSLA) *
                  100
                }
                strokeColor="#1890ff"
                showInfo={false}
              />
            </div>
            <div>
              <div className="flex justify-between mb-2">
                <Text>低</Text>
                <Text strong>{mockData.slaMetrics.lowPrioritySLA}</Text>
              </div>
              <Progress
                percent={
                  (mockData.slaMetrics.lowPrioritySLA /
                    mockData.slaMetrics.totalSLA) *
                  100
                }
                strokeColor="#52c41a"
                showInfo={false}
              />
            </div>
          </div>
        </Card>
      </Col>
      <Col span={8}>
        <Card title="SLA预警" className="h-full">
          <div className="space-y-3">
            <Alert
              message="紧急工单即将超时"
              description="TICKET-003 将在30分钟内超时"
              type="error"
              showIcon
              icon={<AlertTriangle size={16} />}
            />
            <Alert
              message="高优先级工单预警"
              description="2个高优先级工单将在2小时内超时"
              type="warning"
              showIcon
              icon={<Clock size={16} />}
            />
            <Alert
              message="SLA合规率良好"
              description="当前SLA合规率为94.2%，高于目标值"
              type="success"
              showIcon
              icon={<CheckCircle size={16} />}
            />
          </div>
        </Card>
      </Col>
    </Row>
  );

  const renderTeamPerformance = () => (
    <Card title="团队绩效" className="mb-6">
      <div className="mb-4">
        <Space>
          <Text>筛选团队:</Text>
          <Select
            value={selectedTeam}
            onChange={setSelectedTeam}
            style={{ width: 120 }}
          >
            <Option value="all">全部团队</Option>
            <Option value="support">技术支持</Option>
            <Option value="engineering">工程团队</Option>
            <Option value="management">管理团队</Option>
          </Select>
          <Button icon={<RefreshCw size={16} />} onClick={loadDashboardData}>
            刷新
          </Button>
        </Space>
      </div>

      <Table
        dataSource={mockData.teamPerformance}
        columns={[
          {
            title: "团队成员",
            key: "member",
            render: (_, record) => (
              <div className="flex items-center">
                <Avatar size="small" className="mr-2">
                  {record.avatar}
                </Avatar>
                <div>
                  <div className="font-medium">{record.name}</div>
                  <div className="text-xs text-gray-500">{record.role}</div>
                </div>
              </div>
            ),
          },
          {
            title: "工单分配",
            dataIndex: "ticketsAssigned",
            key: "ticketsAssigned",
            render: (value) => <Badge count={value} showZero />,
          },
          {
            title: "工单解决",
            dataIndex: "ticketsResolved",
            key: "ticketsResolved",
            render: (value) => <Badge count={value} showZero />,
          },
          {
            title: "平均解决时间",
            dataIndex: "avgResolutionTime",
            key: "avgResolutionTime",
            render: (value) => <Text>{value}</Text>,
          },
          {
            title: "SLA合规率",
            dataIndex: "slaCompliance",
            key: "slaCompliance",
            render: (value) => (
              <div>
                <Text>{value}%</Text>
                <Progress
                  percent={value}
                  strokeColor="#52c41a"
                  showInfo={false}
                  size="small"
                />
              </div>
            ),
          },
          {
            title: "客户评分",
            dataIndex: "customerRating",
            key: "customerRating",
            render: (value) => (
              <Rate disabled defaultValue={value} size="small" />
            ),
          },
          {
            title: "效率指数",
            dataIndex: "efficiency",
            key: "efficiency",
            render: (value) => (
              <div>
                <Text>{value}%</Text>
                <Progress
                  percent={value}
                  strokeColor="#1890ff"
                  showInfo={false}
                  size="small"
                />
              </div>
            ),
          },
          {
            title: "操作",
            key: "actions",
            render: () => (
              <Space>
                <Button type="text" size="small" icon={<Eye size={14} />}>
                  查看详情
                </Button>
                <Button type="text" size="small" icon={<Edit size={14} />}>
                  编辑
                </Button>
              </Space>
            ),
          },
        ]}
        pagination={false}
        size="small"
      />
    </Card>
  );

  const renderRecentActivities = () => (
    <Card title="最近活动" className="mb-6">
      <Timeline>
        {mockData.recentActivities.map((activity) => (
          <Timeline.Item
            key={activity.id}
            color={
              activity.action.includes("解决")
                ? "green"
                : activity.action.includes("分配")
                ? "blue"
                : activity.action.includes("预警")
                ? "red"
                : "gray"
            }
          >
            <div className="flex justify-between items-start">
              <div>
                <Text strong>{activity.action}</Text>
                <div className="text-sm text-gray-500">
                  {activity.ticketNumber}
                </div>
                <div className="text-sm">{activity.details}</div>
              </div>
              <div className="text-right">
                <div className="text-sm text-gray-500">{activity.user}</div>
                <div className="text-xs text-gray-400">
                  {activity.timestamp}
                </div>
              </div>
            </div>
          </Timeline.Item>
        ))}
      </Timeline>
    </Card>
  );

  const renderTrends = () => (
    <Card title="趋势分析" className="mb-6">
      <div className="mb-4">
        <Space>
          <Text>时间范围:</Text>
          <Radio.Group
            value={timeRange}
            onChange={(e) => setTimeRange(e.target.value)}
          >
            <Radio.Button value="7d">7天</Radio.Button>
            <Radio.Button value="30d">30天</Radio.Button>
            <Radio.Button value="90d">90天</Radio.Button>
          </Radio.Group>
        </Space>
      </div>

      <Row gutter={16}>
        <Col span={12}>
          <Card title="工单数量趋势" size="small">
            <div className="h-32 flex items-center justify-center text-gray-400">
              图表区域 - 工单数量趋势
            </div>
          </Card>
        </Col>
        <Col span={12}>
          <Card title="解决时间趋势" size="small">
            <div className="h-32 flex items-center justify-center text-gray-400">
              图表区域 - 解决时间趋势
            </div>
          </Card>
        </Col>
      </Row>
    </Card>
  );

  const renderAdvancedMetrics = () => (
    <Card title="高级指标" className="mb-6">
      <Row gutter={16}>
        <Col span={8}>
          <Card title="客户满意度分析" size="small">
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-600 mb-2">
                {mockData.overview.customerSatisfaction}
              </div>
              <Text type="secondary">平均评分</Text>
              <Rate
                disabled
                defaultValue={mockData.overview.customerSatisfaction}
                className="mt-2"
              />
            </div>
          </Card>
        </Col>
        <Col span={8}>
          <Card title="团队效率" size="small">
            <div className="text-center">
              <div className="text-2xl font-bold text-green-600 mb-2">
                {mockData.overview.teamPerformance}%
              </div>
              <Text type="secondary">整体效率</Text>
              <Progress
                percent={mockData.overview.teamPerformance}
                strokeColor="#52c41a"
                showInfo={false}
              />
            </div>
          </Card>
        </Col>
        <Col span={8}>
          <Card title="SLA阈值设置" size="small">
            <div className="space-y-3">
              <div>
                <Text>SLA合规率阈值: {slaThreshold}%</Text>
                <Slider
                  value={slaThreshold}
                  onChange={setSlaThreshold}
                  min={80}
                  max={100}
                  marks={{
                    80: "80%",
                    90: "90%",
                    95: "95%",
                    100: "100%",
                  }}
                />
              </div>
              <Switch
                checked={showAdvancedMetrics}
                onChange={setShowAdvancedMetrics}
                checkedChildren="显示高级指标"
                unCheckedChildren="隐藏高级指标"
              />
            </div>
          </Card>
        </Col>
      </Row>
    </Card>
  );

  return (
    <>
      {/* 页面头部操作 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">工单仪表板</h1>
          <p className="text-gray-600 mt-1">实时监控工单处理状态、SLA合规率和团队绩效</p>
        </div>
        <Space>
          <Select
            value={timeRange}
            onChange={setTimeRange}
            style={{ width: 120 }}
          >
            <Option value="7d">最近7天</Option>
            <Option value="30d">最近30天</Option>
            <Option value="90d">最近90天</Option>
          </Select>
          <Button icon={<RefreshCw size={16} />} onClick={loadDashboardData}>
            刷新
          </Button>
          <Button icon={<Download size={16} />}>导出报告</Button>
        </Space>
      </div>
      {loading ? (
        <div className="text-center py-16">
          <Spin size="large" />
          <div className="mt-4 text-gray-500">正在加载仪表板数据...</div>
        </div>
      ) : (
        <div>
          <Tabs activeKey={activeTab} onChange={setActiveTab} className="mb-6">
            <TabPane tab="概览" key="overview">
              {renderOverviewCards()}
              {renderSLAMetrics()}
              {renderRecentActivities()}
            </TabPane>

            <TabPane tab="团队绩效" key="performance">
              {renderTeamPerformance()}
              {renderTrends()}
            </TabPane>

            <TabPane tab="SLA监控" key="sla">
              {renderSLAMetrics()}
              {renderAdvancedMetrics()}
            </TabPane>

            <TabPane tab="趋势分析" key="trends">
              {renderTrends()}
              {renderAdvancedMetrics()}
            </TabPane>
          </Tabs>
        </div>
      )}
    </>
  );
};

export default TicketDashboardPage;
