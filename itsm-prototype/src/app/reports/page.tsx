"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Space,
  Typography,
  Row,
  Col,
  Statistic,
  Button,
  Select,
  DatePicker,
  Tabs,
  Alert,
  Badge,
  Progress,
  Tag,
} from "antd";
import {
  BarChart3,
  TrendingDown,
  Download,
  Heart,
  Star,
  FileText,
  Workflow,
  Zap} from 'lucide-react';
import { SmartSLAMonitor } from "../components/SmartSLAMonitor";
import { PredictiveAnalytics } from "../components/PredictiveAnalytics";
import { SatisfactionDashboard } from "../components/SatisfactionDashboard";
import { TicketAssociation } from "../components/TicketAssociation";

const { Title, Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;
const { TabPane } = Tabs;

interface ReportMetrics {
  totalTickets: number;
  resolvedTickets: number;
  avgResolutionTime: number;
  slaCompliance: number;
  satisfactionScore: number;
  activeAgents: number;
  pendingTickets: number;
  urgentTickets: number;
}

export default function ReportsPage() {
  const [metrics, setMetrics] = useState<ReportMetrics | null>(null);
  const [timeRange, setTimeRange] = useState<[string, string] | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadReportMetrics();
  }, [timeRange]);

  const loadReportMetrics = async () => {
    try {
      // 模拟API调用
      const mockMetrics: ReportMetrics = {
        totalTickets: 1247,
        resolvedTickets: 1189,
        avgResolutionTime: 8.7,
        slaCompliance: 87.3,
        satisfactionScore: 4.2,
        activeAgents: 12,
        pendingTickets: 58,
        urgentTickets: 23,
      };

      setMetrics(mockMetrics);
    } catch (error) {
      console.error("Failed to load report data:", error);
    } finally {
      setLoading(false);
    }
  };

  const getTrendIcon = (current: number, previous: number) => {
    if (current > previous) return <TrendingUp className="text-green-500" />;
    if (current < previous) return <TrendingDown className="text-red-500" />;
    return <TrendingUp className="text-blue-500" />;
  };

  const getSLAStatus = (compliance: number) => {
    if (compliance >= 95) return { color: "success", text: "Excellent" };
    if (compliance >= 85) return { color: "normal", text: "Good" };
    if (compliance >= 75) return { color: "exception", text: "Needs Improvement" };
    return { color: "exception", text: "Critical" };
  };

  const getSatisfactionColor = (score: number) => {
    if (score >= 4.5) return "#52c41a";
    if (score >= 4.0) return "#1890ff";
    if (score >= 3.5) return "#faad14";
    return "#f5222d";
  };

  if (loading) {
    return (
      <Card>
        <div className="animate-pulse space-y-4">
          <div className="h-4 bg-gray-200 rounded w-1/3"></div>
          <div className="space-y-3">
            <div className="h-3 bg-gray-200 rounded"></div>
            <div className="h-3 bg-gray-200 rounded w-5/6"></div>
            <div className="h-3 bg-gray-200 rounded w-4/6"></div>
          </div>
        </div>
      </Card>
    );
  }

  return (
    <div className="max-w-7xl mx-auto p-6">
      {/* Page Title */}
      <div className="mb-6">
        <Title level={2}>
          <BarChart3 className="mr-3 text-blue-600" />
          Intelligent Report Center
        </Title>
        <Text type="secondary">
          Comprehensive ITSM data analysis and intelligent insights to help you optimize service quality and operational efficiency
        </Text>
      </div>

      {/* Time Range Selector */}
      <Card className="mb-6">
        <div className="flex items-center justify-between">
          <Space>
            <Calendar className="text-gray-500" />
            <Text strong>Report Time Range</Text>
          </Space>
          <Space>
            <RangePicker
              value={timeRange}
              onChange={(dates) => {
                if (dates) {
                  setTimeRange([
                    dates[0]?.toISOString() || "",
                    dates[1]?.toISOString() || "",
                  ]);
                }
              }}
            />
            <Select
              defaultValue="30d"
              style={{ width: 120 }}
              onChange={(value) => console.log("Quick select:", value)}
            >
              <Option value="7d">Last 7 Days</Option>
              <Option value="30d">Last 30 Days</Option>
              <Option value="90d">Last 90 Days</Option>
              <Option value="1y">Last Year</Option>
            </Select>
            <Button icon={<Download />} type="primary">
              Export Report
            </Button>
          </Space>
        </div>
      </Card>

      {/* Key Metrics Overview */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Total Tickets"
              value={metrics?.totalTickets}
              prefix={<FileText className="text-blue-500" />}
              suffix={
                <div className="text-xs text-gray-500">
                  <div>Resolved: {metrics?.resolvedTickets}</div>
                  <div>Pending: {metrics?.pendingTickets}</div>
                </div>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Average Resolution Time"
              value={metrics?.avgResolutionTime}
              prefix={<Clock className="text-green-500" />}
              suffix="hours"
            />
            <div className="mt-2">
              <Progress
                percent={Math.min(
                  ((metrics?.avgResolutionTime || 0) / 24) * 100,
                  100
                )}
                size="small"
                status={
                  metrics && metrics.avgResolutionTime <= 12
                    ? "success"
                    : "normal"
                }
              />
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="SLA Compliance Rate"
              value={metrics?.slaCompliance}
              prefix={<CheckCircle className="text-blue-500" />}
              suffix="%"
              valueStyle={{
                color:
                  getSLAStatus(metrics?.slaCompliance || 0).color === "success"
                    ? "#52c41a"
                    : "#1890ff",
              }}
            />
            <div className="mt-2">
              <Progress
                percent={metrics?.slaCompliance || 0}
                status={getSLAStatus(metrics?.slaCompliance || 0).color as "success" | "normal" | "exception"}
                size="small"
              />
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Satisfaction Score"
              value={metrics?.satisfactionScore}
              precision={1}
              prefix={
                <Star
                  style={{
                    color: getSatisfactionColor(
                      metrics?.satisfactionScore || 0
                    ),
                  }}
                />
              }
              suffix="/5.0"
            />
            <div className="mt-2">
              <Progress
                percent={((metrics?.satisfactionScore || 0) / 5) * 100}
                status="success"
                size="small"
                strokeColor={getSatisfactionColor(
                  metrics?.satisfactionScore || 0
                )}
              />
            </div>
          </Card>
        </Col>
      </Row>

      {/* Urgent Ticket Alert */}
      {metrics && metrics.urgentTickets > 0 && (
        <Alert
          message={`There are ${metrics.urgentTickets} urgent tickets that need immediate attention`}
          description="Please prioritize high-priority tickets to ensure SLA compliance"
          type="warning"
          showIcon
          icon={<AlertTriangle />}
          className="mb-6"
          action={
            <Button size="small" type="link">
              View Details
            </Button>
          }
        />
      )}

      {/* Main Report Content */}
      <Tabs
        defaultActiveKey="overview"
        size="large"
        type="card"
        items={[
          {
            key: "overview",
            label: (
              <Space>
                <BarChart3 size={16} />
                Overview
              </Space>
            ),
            children: (
              <div className="space-y-6">
                <Row gutter={[16, 16]}>
                  <Col xs={24} lg={12}>
                    <Card title="Ticket Status Distribution">
                      <div className="space-y-4">
                        <div className="flex items-center justify-between">
                          <Text>Resolved</Text>
                          <div className="flex items-center">
                            <Progress
                              percent={Math.round(
                                ((metrics?.resolvedTickets || 0) /
                                  (metrics?.totalTickets || 1)) *
                                  100
                              )}
                              size="small"
                              status="success"
                              className="mr-2"
                              style={{ width: 100 }}
                            />
                            <Text strong>{metrics?.resolvedTickets}</Text>
                          </div>
                        </div>
                        <div className="flex items-center justify-between">
                          <Text>Pending</Text>
                          <div className="flex items-center">
                            <Progress
                              percent={Math.round(
                                ((metrics?.pendingTickets || 0) /
                                  (metrics?.totalTickets || 1)) *
                                  100
                              )}
                              size="small"
                              status="normal"
                              className="mr-2"
                              style={{ width: 100 }}
                            />
                            <Text strong>{metrics?.pendingTickets}</Text>
                          </div>
                        </div>
                        <div className="flex items-center justify-between">
                          <Text>Urgent Tickets</Text>
                          <div className="flex items-center">
                            <Progress
                              percent={Math.round(
                                ((metrics?.urgentTickets || 0) /
                                  (metrics?.totalTickets || 1)) *
                                  100
                              )}
                              size="small"
                              status="exception"
                              className="mr-2"
                              style={{ width: 100 }}
                            />
                            <Text strong style={{ color: "#f5222d" }}>
                              {metrics?.urgentTickets}
                            </Text>
                          </div>
                        </div>
                      </div>
                    </Card>
                  </Col>
                  <Col xs={24} lg={12}>
                    <Card title="Service Quality Metrics">
                      <div className="space-y-4">
                        <div className="flex items-center justify-between">
                          <Text>SLA Compliance Rate</Text>
                          <div className="flex items-center">
                            <Tag
                              color={
                                getSLAStatus(metrics?.slaCompliance || 0)
                                  .color === "success"
                                  ? "green"
                                  : "blue"
                              }
                            >
                              {getSLAStatus(metrics?.slaCompliance || 0).text}
                            </Tag>
                            <Text strong className="ml-2">
                              {metrics?.slaCompliance}%
                            </Text>
                          </div>
                        </div>
                        <div className="flex items-center justify-between">
                          <Text>Average Resolution Time</Text>
                          <div className="flex items-center">
                            <Tag
                              color={
                                metrics && metrics.avgResolutionTime <= 12
                                  ? "green"
                                  : "orange"
                              }
                            >
                              {metrics && metrics.avgResolutionTime <= 12
                                ? "Excellent"
                                : "Needs Improvement"}
                            </Tag>
                            <Text strong className="ml-2">
                              {metrics?.avgResolutionTime}hours
                            </Text>
                          </div>
                        </div>
                        <div className="flex items-center justify-between">
                          <Text>Satisfaction Score</Text>
                          <div className="flex items-center">
                            <Tag
                              color={
                                metrics && metrics.satisfactionScore >= 4.0
                                  ? "green"
                                  : "orange"
                              }
                            >
                              {metrics && metrics.satisfactionScore >= 4.0
                                ? "Satisfied"
                                : "Needs Improvement"}
                            </Tag>
                            <Text strong className="ml-2">
                              {metrics?.satisfactionScore}/5.0
                            </Text>
                          </div>
                        </div>
                      </div>
                    </Card>
                  </Col>
                </Row>
              </div>
            ),
          },
          {
            key: "sla",
            label: (
              <Space>
                <Clock size={16} />
                SLA Monitoring
                <Badge count={metrics?.urgentTickets} />
              </Space>
            ),
            children: <SmartSLAMonitor />,
          },
          {
            key: "satisfaction",
            label: (
              <Space>
                <Heart size={16} />
                Satisfaction Analysis
              </Space>
            ),
            children: <SatisfactionDashboard />,
          },
          {
            key: "prediction",
            label: (
              <Space>
                <TrendingUp size={16} />
                Intelligent Prediction
              </Space>
            ),
            children: <PredictiveAnalytics />,
          },
          {
            key: "association",
            label: (
              <Space>
                <Workflow size={16} />
                Ticket Association
              </Space>
            ),
            children: <TicketAssociation />,
          },
        ]}
      />

      {/* Quick Actions */}
      <Card className="mt-6">
        <div className="flex items-center justify-between">
          <div>
            <Title level={5}>Quick Actions</Title>
            <Text type="secondary">Quick access to common report functions</Text>
          </div>
          <Space>
            <Button icon={<Download />}>Export Comprehensive Report</Button>
            <Button icon={<Eye />} type="primary">
              View Detailed Data
            </Button>
            <Button icon={<Zap />}>Generate Analysis Report</Button>
          </Space>
        </div>
      </Card>
    </div>
  );
}
