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
  message,
  Spin,
} from "antd";
import {
  TrendingUp,
  Clock,
  AlertTriangle,
  CheckCircle,
  FileText,
  RefreshCw,
  Download,
} from "lucide-react";
import { dashboardService, DashboardOverviewResponse } from "@/lib/services/analytics-service";
import { ticketService } from "@/lib/services/ticket-service";

const { Option } = Select;
const { Text, Title } = Typography;
const { TabPane } = Tabs;

const TicketDashboardPage = () => {
  const [loading, setLoading] = useState(false);
  const [timeRange, setTimeRange] = useState("7d");
  const [selectedTeam, setSelectedTeam] = useState("all");
  const [activeTab, setActiveTab] = useState("overview");
  const [slaThreshold, setSlaThreshold] = useState(95);
  const [dashboardData, setDashboardData] = useState<DashboardOverviewResponse | null>(null);
  const [ticketStats, setTicketStats] = useState<any>(null);

  useEffect(() => {
    loadDashboardData();
  }, [timeRange, selectedTeam]);

  const loadDashboardData = async () => {
    setLoading(true);
    try {
      const [dashboardRes, statsRes] = await Promise.all([
        dashboardService.getOverview(),
        ticketService.getTicketStats(),
      ]);
      setDashboardData(dashboardRes);
      setTicketStats(statsRes);
    } catch (error) {
      console.error("Failed to load dashboard data:", error);
      message.error("加载仪表盘数据失败");
    } finally {
      setLoading(false);
    }
  };

  const handleExport = () => {
    message.info("导出功能开发中");
  };

  // 获取概览数据
  const getOverviewData = () => {
    if (dashboardData?.overview) return dashboardData.overview;
    // 不再回退到 ticketStats，完全依赖 API 数据
    return {
      totalTickets: 0,
      pendingTickets: 0,
      resolvedTickets: 0,
      avgResponseTime: 0,
      avgResolutionTime: 0,
      customerSatisfaction: 0,
      monthlyGrowth: 0,
      openTickets: 0,
      slaCompliance: 0,
    };
  };

  // 获取SLA数据
  const getSLAData = () => {
    return dashboardData?.sla_data || {
      compliance_rate: 0,
      response_time_compliance: 0,
      resolution_time_compliance: 0,
      at_risk_tickets: 0,
      breached_tickets: 0,
    };
  };

  const renderOverviewCards = () => {
    const overview = getOverviewData();
    const slaData = getSLAData();

    return (
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总工单数"
              value={overview.totalTickets}
              prefix={<FileText size={16} style={{ color: "#3b82f6" }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="待处理工单"
              value={overview.openTickets}
              styles={{ content: { color: "#faad14" } }}
              prefix={<Clock size={16} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="已解决工单"
              value={overview.resolvedTickets}
              styles={{ content: { color: "#52c41a" } }}
              prefix={<CheckCircle size={16} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="SLA合规率"
              value={slaData.compliance_rate}
              suffix="%"
              prefix={<TrendingUp size={16} style={{ color: "#3b82f6" }} />}
              styles={{ content: {
                color: slaData.compliance_rate >= 95 ? "#52c41a" : "#faad14",
              }}
            />
            <Progress
              percent={slaData.compliance_rate}
              showInfo={false}
              strokeColor={slaData.compliance_rate >= 95 ? "#52c41a" : "#faad14"}
              className="mt-2"
            />
          </Card>
        </Col>
      </Row>
    );
  };

  const renderSLAMetrics = () => {
    const slaData = getSLAData();
    const stats = ticketStats || { total: 0, high_priority: 0, urgent: 0 };

    return (
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} lg={8}>
          <Card title="SLA概览" className="h-full">
            <div className="text-center mb-4">
              <div className="text-3xl font-bold text-blue-600 mb-2">
                {stats.total - (slaData.breached_tickets || 0)}
              </div>
              <Text type="secondary">SLA合规工单数</Text>
            </div>
            <Progress
              percent={slaData.compliance_rate || 0}
              strokeColor="#52c41a"
              showInfo={false}
            />
            <div className="mt-4 space-y-2">
              <div className="flex justify-between text-sm">
                <span>总工单: {stats.total}</span>
                <span>已违规: {slaData.breached_tickets || 0}</span>
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="优先级分布" className="h-full">
            <div className="space-y-4">
              <div>
                <div className="flex justify-between mb-2">
                  <Text>紧急</Text>
                  <Text strong>{stats.urgent || 0}</Text>
                </div>
                <Progress
                  percent={stats.total > 0 ? ((stats.urgent || 0) / stats.total) * 100 : 0}
                  strokeColor="#ff4d4f"
                  showInfo={false}
                />
              </div>
              <div>
                <div className="flex justify-between mb-2">
                  <Text>高</Text>
                  <Text strong>{stats.high_priority || 0}</Text>
                </div>
                <Progress
                  percent={stats.total > 0 ? ((stats.high_priority || 0) / stats.total) * 100 : 0}
                  strokeColor="#faad14"
                  showInfo={false}
                />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="SLA告警" className="h-full">
            <div className="space-y-3">
              <Alert
                message="即将超时工单"
                description={`${slaData.at_risk_tickets || 0} 个工单即将超时`}
                type="warning"
                showIcon
                icon={<Clock size={16} />}
              />
              <Alert
                message="SLA合规状态"
                description={`当前合规率: ${slaData.compliance_rate?.toFixed(1) || 0}%`}
                type={slaData.compliance_rate >= 95 ? "success" : "warning"}
                showIcon
                icon={<CheckCircle size={16} />}
              />
            </div>
          </Card>
        </Col>
      </Row>
    );
  };

  const renderTeamPerformance = () => (
    <Card title="团队表现" className="mb-6">
      <div className="mb-4">
        <Space>
          <Text>筛选团队:</Text>
          <Select
            value={selectedTeam}
            onChange={setSelectedTeam}
            style={{ width: 120 }}
          >
            <Option value="all">全部</Option>
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
        dataSource={[]}
        columns={[
          {
            title: "团队成员",
            key: "member",
            render: () => (
              <div className="flex items-center">
                <Avatar size="small" className="mr-2">
                  -
                </Avatar>
                <div>
                  <div className="font-medium">暂无数据</div>
                </div>
              </div>
            ),
          },
          {
            title: "已分配工单",
            key: "assigned",
            render: () => <Badge count={0} showZero />,
          },
          {
            title: "已解决工单",
            key: "resolved",
            render: () => <Badge count={0} showZero />,
          },
          {
            title: "平均解决时间",
            key: "avgTime",
            render: () => <Text>-</Text>,
          },
          {
            title: "SLA合规率",
            key: "sla",
            render: () => <Text>-</Text>,
          },
        ]}
        pagination={false}
        size="small"
        locale={{ emptyText: "暂无团队数据" }}
      />
    </Card>
  );

  const renderRecentActivities = () => {
    const activities = dashboardData?.recent_activities || [];

    return (
      <Card title="最近活动" className="mb-6">
        {activities.length > 0 ? (
          <Timeline>
            {activities.map((activity) => (
              <Timeline.Item
                key={activity.id}
                color={
                  activity.type?.includes("resolved")
                    ? "green"
                    : activity.type?.includes("assigned")
                    ? "blue"
                    : activity.type?.includes("alert")
                    ? "red"
                    : "gray"
                }
              >
                <div className="flex justify-between items-start">
                  <div>
                    <Text strong>{activity.description || activity.type}</Text>
                    {activity.ticket_id && (
                      <div className="text-sm text-gray-500">
                        工单 #{activity.ticket_id}
                      </div>
                    )}
                  </div>
                  <div className="text-right">
                    <div className="text-sm text-gray-500">{activity.user}</div>
                    <div className="text-xs text-gray-400">
                      {activity.timestamp
                        ? new Date(activity.timestamp).toLocaleString("zh-CN")
                        : "-"}
                    </div>
                  </div>
                </div>
              </Timeline.Item>
            ))}
          </Timeline>
        ) : (
          <div className="text-center py-8 text-gray-500">暂无最近活动</div>
        )}
      </Card>
    );
  };

  const renderTrends = () => {
    const trendData = dashboardData?.ticket_trend || [];

    return (
      <Card title="趋势分析" className="mb-6">
        <div className="mb-4">
          <Space>
            <Text>时间范围:</Text>
            <Select
              value={timeRange}
              onChange={setTimeRange}
              style={{ width: 120 }}
            >
              <Option value="7d">最近7天</Option>
              <Option value="30d">最近30天</Option>
              <Option value="90d">最近90天</Option>
            </Select>
          </Space>
        </div>

        <Row gutter={16}>
          <Col span={12}>
            <Card title="工单量趋势" size="small">
              {trendData.length > 0 ? (
                <div className="h-32 flex items-center justify-center">
                  {/* 趋势图表占位 - 实际项目中应使用图表组件 */}
                  <Text type="secondary">
                    共 {trendData.length} 个数据点
                  </Text>
                </div>
              ) : (
                <div className="h-32 flex items-center justify-center text-gray-400">
                  暂无趋势数据
                </div>
              )}
            </Card>
          </Col>
          <Col span={12}>
            <Card title="解决率趋势" size="small">
              {trendData.length > 0 ? (
                <div className="h-32 flex items-center justify-center">
                  <Text type="secondary">
                    共 {trendData.length} 个数据点
                  </Text>
                </div>
              ) : (
                <div className="h-32 flex items-center justify-center text-gray-400">
                  暂无趋势数据
                </div>
              )}
            </Card>
          </Col>
        </Row>
      </Card>
    );
  };

  const renderAdvancedMetrics = () => {
    const satisfactionData = dashboardData?.satisfaction_data || {
      average_rating: 0,
      total_ratings: 0,
    };

    return (
      <Card title="高级指标" className="mb-6">
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={8}>
            <Card title="客户满意度" size="small">
              <div className="text-center">
                <div className="text-3xl font-bold text-blue-600 mb-2">
                  {satisfactionData.average_rating?.toFixed(1) || "N/A"}
                </div>
                <Text type="secondary">平均评分</Text>
                <Rate
                  disabled
                  value={satisfactionData.average_rating || 0}
                  className="mt-2"
                />
                <div className="text-sm text-gray-500 mt-2">
                  共 {satisfactionData.total_ratings || 0} 条评价
                </div>
              </div>
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card title="团队效率" size="small">
              <div className="text-center">
                <div className="text-3xl font-bold text-green-600 mb-2">-</div>
                <Text type="secondary">整体效率</Text>
                <Progress
                  percent={0}
                  strokeColor="#52c41a"
                  showInfo={false}
                />
              </div>
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card title="SLA阈值设置" size="small">
              <div className="p-4">
                <Text>SLA合规阈值: {slaThreshold}%</Text>
                <Rate
                  value={slaThreshold}
                  onChange={setSlaThreshold}
                  count={5}
                  style={{ marginTop: 8 }}
                />
              </div>
            </Card>
          </Col>
        </Row>
      </Card>
    );
  };

  return (
    <>
      <div className="mb-6 flex justify-between items-center">
        <div>
          <Title level={2}>工单仪表盘</Title>
          <p className="text-gray-600 mt-1">
            实时监控工单处理状态、SLA合规率和团队表现
          </p>
        </div>
        <Space>
          <Select value={timeRange} onChange={setTimeRange} style={{ width: 120 }}>
            <Option value="7d">最近7天</Option>
            <Option value="30d">最近30天</Option>
            <Option value="90d">最近90天</Option>
          </Select>
          <Button icon={<RefreshCw size={16} />} onClick={loadDashboardData}>
            刷新
          </Button>
          <Button icon={<Download size={16} />} onClick={handleExport}>
            导出报表
          </Button>
        </Space>
      </div>

      {loading ? (
        <div className="text-center py-16">
          <Spin size="large" />
          <div className="mt-4 text-gray-500">加载仪表盘数据...</div>
        </div>
      ) : (
        <div>
          <Tabs activeKey={activeTab} onChange={setActiveTab} className="mb-6">
            <TabPane tab="概览" key="overview">
              {renderOverviewCards()}
              {renderSLAMetrics()}
              {renderRecentActivities()}
            </TabPane>

            <TabPane tab="团队表现" key="performance">
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
