"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Row,
  Col,
  Statistic,
  Progress,
  Table,
  Typography,
  Space,
  Button,
  Select,
  Tabs,
  Avatar,
  Badge,
  Alert,
  Timeline,
  Rate,
  Slider,
  Radio,
  message,
  Spin,
} from "antd";
import {
  TrendingUp,
  Clock,
  AlertTriangle,
  CheckCircle,
  FileText,
  Target,
  RefreshCw,
  Download,
  Eye,
  Edit,
} from "lucide-react";
// AppLayout is handled by layout.tsx

const { Option } = Select;
const { Text } = Typography;
const { TabPane } = Tabs;

// Mock data
const mockData = {
  overview: {
    totalTickets: 1247,
    pendingTickets: 89,
    resolvedTickets: 1158,
    avgResponseTime: "2.3 hours",
    avgResolutionTime: "8.7 hours",
    customerSatisfaction: 4.2,
  },
  slaMetrics: {
    totalSLA: 1247,
    metSLA: 1175,
    breachedSLA: 72,
    avgResponseTime: "2.3 hours",
    avgResolutionTime: "8.7 hours",
    complianceRate: 94.2,
  },
  teamPerformance: [
    {
      id: 1,
      name: "Zhang San",
      avatar: "Z",
      role: "Technical Support",
      assignedTickets: 45,
      resolvedTickets: 42,
      avgResolutionTime: "6.2 hours",
      slaCompliance: 96.8,
      customerRating: 4.5,
      efficiency: 93.3,
    },
    {
      id: 2,
      name: "Li Si",
      avatar: "L",
      role: "Senior Engineer",
      assignedTickets: 38,
      resolvedTickets: 35,
      avgResolutionTime: "7.8 hours",
      slaCompliance: 92.1,
      customerRating: 4.3,
      efficiency: 92.1,
    },
    {
      id: 3,
      name: "Wang Wu",
      avatar: "W",
      role: "Project Manager",
      assignedTickets: 32,
      resolvedTickets: 29,
      avgResolutionTime: "9.1 hours",
      slaCompliance: 90.6,
      customerRating: 4.1,
      efficiency: 90.6,
    },
  ],
  recentActivities: [
    {
      id: 1,
      action: "Ticket resolved",
      timestamp: "2024-01-15 14:30",
      user: "Zhang San",
      ticketId: "TICKET-001",
      details: "System login issue resolved, user feedback positive",
    },
    {
      id: 2,
      action: "Ticket assigned",
      timestamp: "2024-01-15 13:45",
      user: "Li Si",
      ticketId: "TICKET-002",
      details: "Printer malfunction ticket assigned to hardware team",
    },
    {
      id: 3,
      action: "SLA warning",
      timestamp: "2024-01-15 13:00",
      user: "System",
      ticketId: "TICKET-003",
      details: "High priority ticket about to timeout, please handle promptly",
    },
  ],
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
      // Mock API call
      await new Promise((resolve) => setTimeout(resolve, 1000));
    } catch (error) {
      message.error("Failed to load dashboard data");
    } finally {
      setLoading(false);
    }
  };

    const renderOverviewCards = () => (
      <Row gutter={16} className="mb-6">
        <Col span={6}>
          <Card>
            <Statistic
              title="Total Tickets"
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
                  <div>Monthly Growth</div>
                </div>
              }
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Pending Tickets"
              value={mockData.overview.openTickets}
              valueStyle={{ color: "#faad14" }}
              prefix={<Clock size={16} />}
              suffix={
                <div className="text-xs text-gray-500 mt-1">
                  <div>Avg Response: 2.3h</div>
                  <div className="text-orange-500">SLA: 4h</div>
                </div>
              }
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Resolved Tickets"
              value={mockData.overview.resolvedTickets}
              valueStyle={{ color: "#52c41a" }}
              prefix={<CheckCircle size={16} />}
              suffix={
                <div className="text-xs text-gray-500 mt-1">
                  <div>Avg Resolution: {mockData.overview.avgResolutionTime}</div>
                  <div className="text-green-500">
                    Satisfaction: {mockData.overview.customerSatisfaction}/5
                  </div>
                </div>
              }
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="SLA Compliance"
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
          <Card title="SLA Overview" className="h-full">
            <div className="text-center mb-4">
              <div className="text-3xl font-bold text-blue-600 mb-2">
                {mockData.slaMetrics.withinSLA}
              </div>
              <Text type="secondary">SLA Compliant</Text>
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
                <span>Total Tickets: {mockData.slaMetrics.totalSLA}</span>
                <span>Breached: {mockData.slaMetrics.breachedSLA}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span>Avg Response: {mockData.slaMetrics.avgResponseTime}</span>
                <span>Avg Resolution: {mockData.slaMetrics.avgResolutionTime}</span>
              </div>
            </div>
          </Card>
        </Col>
        <Col span={8}>
          <Card title="Priority Distribution" className="h-full">
            <div className="space-y-4">
              <div>
                <div className="flex justify-between mb-2">
                  <Text>Critical</Text>
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
                  <Text>High</Text>
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
                  <Text>Medium</Text>
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
                  <Text>Low</Text>
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
          <Card title="SLA Alerts" className="h-full">
            <div className="space-y-3">
              <Alert
                message="Critical ticket about to timeout"
                description="TICKET-003 will timeout in 30 minutes"
                type="error"
                showIcon
                icon={<AlertTriangle size={16} />}
              />
              <Alert
                message="High priority ticket alert"
                description="2 high priority tickets will timeout in 2 hours"
                type="warning"
                showIcon
                icon={<Clock size={16} />}
              />
              <Alert
                message="SLA compliance is good"
                description="Current SLA compliance rate is 94.2%, above target"
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
      <Card title="Team Performance" className="mb-6">
        <div className="mb-4">
          <Space>
            <Text>Filter Team:</Text>
            <Select
              value={selectedTeam}
              onChange={setSelectedTeam}
              style={{ width: 120 }}
            >
              <Option value="all">All Teams</Option>
              <Option value="support">Technical Support</Option>
              <Option value="engineering">Engineering Team</Option>
              <Option value="management">Management Team</Option>
            </Select>
            <Button icon={<RefreshCw size={16} />} onClick={loadDashboardData}>
              Refresh
            </Button>
          </Space>
        </div>

        <Table
          dataSource={mockData.teamPerformance}
          columns={[
            {
              title: "Team Member",
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
              title: "Ticket Assignment",
              dataIndex: "ticketsAssigned",
              key: "ticketsAssigned",
              render: (value) => <Badge count={value} showZero />,
            },
            {
              title: "Ticket Resolution",
              dataIndex: "ticketsResolved",
              key: "ticketsResolved",
              render: (value) => <Badge count={value} showZero />,
            },
            {
              title: "Avg Resolution Time",
              dataIndex: "avgResolutionTime",
              key: "avgResolutionTime",
              render: (value) => <Text>{value}</Text>,
            },
            {
              title: "SLA Compliance",
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
              title: "Customer Rating",
              dataIndex: "customerRating",
              key: "customerRating",
              render: (value) => (
                <Rate disabled defaultValue={value} size="small" />
              ),
            },
            {
              title: "Efficiency Index",
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
              title: "Actions",
              key: "actions",
              render: () => (
                <Space>
                  <Button type="text" size="small" icon={<Eye size={14} />}>
                    View Details
                  </Button>
                  <Button type="text" size="small" icon={<Edit size={14} />}>
                    Edit
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
      <Card title="Recent Activities" className="mb-6">
        <Timeline>
          {mockData.recentActivities.map((activity) => (
            <Timeline.Item
              key={activity.id}
              color={
                activity.action.includes("resolved")
                  ? "green"
                  : activity.action.includes("assigned")
                  ? "blue"
                  : activity.action.includes("alert")
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
      <Card title="Trend Analysis" className="mb-6">
        <div className="mb-4">
          <Space>
            <Text>Time Range:</Text>
            <Radio.Group
              value={timeRange}
              onChange={(e) => setTimeRange(e.target.value)}
            >
              <Radio.Button value="7d">7 Days</Radio.Button>
              <Radio.Button value="30d">30 Days</Radio.Button>
              <Radio.Button value="90d">90 Days</Radio.Button>
            </Radio.Group>
          </Space>
        </div>

        <Row gutter={16}>
          <Col span={12}>
            <Card title="Ticket Volume Trend" size="small">
              <div className="h-32 flex items-center justify-center text-gray-500">
                Chart Area - Ticket Volume Trend
              </div>
            </Card>
          </Col>
          <Col span={12}>
            <Card title="Resolution Time Trend" size="small">
              <div className="h-32 flex items-center justify-center text-gray-500">
                Chart Area - Resolution Time Trend
              </div>
            </Card>
          </Col>
        </Row>
      </Card>
    );

    const renderAdvancedMetrics = () => (
      <Card title="Advanced Metrics" className="mb-6">
        <Row gutter={16}>
          <Col span={8}>
            <Card title="Customer Satisfaction Analysis" size="small">
              <div className="text-center">
                <div className="text-3xl font-bold text-blue-600 mb-2">
                  {mockData.overview.customerSatisfaction}
                </div>
                <Text type="secondary">Average Rating</Text>
                <Rate
                  disabled
                  defaultValue={mockData.overview.customerSatisfaction}
                  className="mt-2"
                />
              </div>
            </Card>
          </Col>
          <Col span={8}>
            <Card title="Team Efficiency" size="small">
              <div className="text-center">
                <div className="text-3xl font-bold text-green-600 mb-2">92.1%</div>
                <Text type="secondary">Overall Efficiency</Text>
                <Progress
                  percent={mockData.overview.teamPerformance}
                  strokeColor="#52c41a"
                  showInfo={false}
                />
              </div>
            </Card>
          </Col>
          <Col span={8}>
            <Card title="SLA Threshold Settings" size="small">
              <div className="p-4">
                <Text>SLA Compliance Threshold: {slaThreshold}%</Text>
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
            </Card>
          </Col>
        </Row>
      </Card>
    );

    return (
      <>
        {/* Page header actions */}
        <div className="mb-6 flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Ticket Dashboard</h1>
            <p className="text-gray-600 mt-1">Real-time monitoring of ticket processing status, SLA compliance and team performance</p>
          </div>
          <Space>
            <Select
              value={timeRange}
              onChange={setTimeRange}
              style={{ width: 120 }}
            >
              <Option value="7d">Last 7 Days</Option>
              <Option value="30d">Last 30 Days</Option>
              <Option value="90d">Last 90 Days</Option>
            </Select>
            <Button icon={<RefreshCw size={16} />} onClick={loadDashboardData}>
              Refresh
            </Button>
            <Button icon={<Download size={16} />}>Export Report</Button>
          </Space>
        </div>
        {loading ? (
          <div className="text-center py-16">
            <Spin size="large" />
            <div className="mt-4 text-gray-500">Loading dashboard data...</div>
          </div>
        ) : (
          <div>
            <Tabs activeKey={activeTab} onChange={setActiveTab} className="mb-6">
              <TabPane tab="Overview" key="overview">
                {renderOverviewCards()}
                {renderSLAMetrics()}
                {renderRecentActivities()}
              </TabPane>

              <TabPane tab="Team Performance" key="performance">
                {renderTeamPerformance()}
                {renderTrends()}
              </TabPane>

              <TabPane tab="SLA Monitoring" key="sla">
                {renderSLAMetrics()}
                {renderAdvancedMetrics()}
              </TabPane>

              <TabPane tab="Trend Analysis" key="trends">
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
