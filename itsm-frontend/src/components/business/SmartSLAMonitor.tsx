"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Progress,
  Alert,
  Space,
  Typography,
  Tag,
  Button,
  Tooltip,
  Badge,
  Timeline,
  Statistic,
  Row,
  Col,
} from "antd";
import {
  Clock,
  AlertTriangle,
  CheckCircle,
  TrendingUp,
  TrendingDown,
  Bell,
  Settings,
  Eye,
  Zap,
} from "lucide-react";

const { Title, Text } = Typography;

interface SLAMetrics {
  totalTickets: number;
  onTime: number;
  atRisk: number;
  breached: number;
  avgResponseTime: number;
  avgResolutionTime: number;
  slaCompliance: number;
}

interface SLATicket {
  id: number;
  ticketNumber: string;
  title: string;
  priority: string;
  slaDeadline: string;
  timeRemaining: number;
  status: "on_time" | "at_risk" | "breached";
  assignee: string;
  category: string;
}

interface SLAAlert {
  id: string;
  type: "warning" | "critical" | "info";
  message: string;
  ticketId: number;
  timeRemaining: number;
  priority: string;
}

export const SmartSLAMonitor: React.FC = () => {
  const [slaMetrics, setSlaMetrics] = useState<SLAMetrics | null>(null);
  const [atRiskTickets, setAtRiskTickets] = useState<SLATicket[]>([]);
  const [breachedTickets, setBreachedTickets] = useState<SLATicket[]>([]);
  const [alerts, setAlerts] = useState<SLAAlert[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadSLAData();
    const interval = setInterval(loadSLAData, 30000); // 30秒更新一次
    return () => clearInterval(interval);
  }, []);

  const loadSLAData = async () => {
    try {
      // 模拟API调用
      const mockMetrics: SLAMetrics = {
        totalTickets: 156,
        onTime: 128,
        atRisk: 18,
        breached: 10,
        avgResponseTime: 2.3,
        avgResolutionTime: 8.7,
        slaCompliance: 82.1,
      };

      const mockAtRisk: SLATicket[] = [
        {
          id: 1,
          ticketNumber: "T-2024-001",
          title: "数据库连接超时",
          priority: "high",
          slaDeadline: "2024-01-15 18:00:00",
          timeRemaining: 2.5,
          status: "at_risk",
          assignee: "张三",
          category: "数据库",
        },
        {
          id: 2,
          ticketNumber: "T-2024-002",
          title: "网络设备故障",
          priority: "urgent",
          slaDeadline: "2024-01-15 16:00:00",
          timeRemaining: 1.2,
          status: "at_risk",
          assignee: "李四",
          category: "网络",
        },
      ];

      const mockBreached: SLATicket[] = [
        {
          id: 3,
          ticketNumber: "T-2024-003",
          title: "系统登录异常",
          priority: "medium",
          slaDeadline: "2024-01-15 14:00:00",
          timeRemaining: -2.1,
          status: "breached",
          assignee: "王五",
          category: "系统",
        },
      ];

      const mockAlerts: SLAAlert[] = [
        {
          id: "1",
          type: "critical",
          message: "工单T-2024-002即将超时，剩余时间1.2小时",
          ticketId: 2,
          timeRemaining: 1.2,
          priority: "urgent",
        },
        {
          id: "2",
          type: "warning",
          message: "工单T-2024-001需要关注，剩余时间2.5小时",
          ticketId: 1,
          timeRemaining: 2.5,
          priority: "high",
        },
      ];

      setSlaMetrics(mockMetrics);
      setAtRiskTickets(mockAtRisk);
      setBreachedTickets(mockBreached);
      setAlerts(mockAlerts);
    } catch (error) {
      console.error("加载SLA数据失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case "on_time":
        return "green";
      case "at_risk":
        return "orange";
      case "breached":
        return "red";
      default:
        return "default";
    }
  };

  const getPriorityColor = (priority: string): string => {
    switch (priority) {
      case "urgent":
        return "red";
      case "high":
        return "orange";
      case "medium":
        return "blue";
      case "low":
        return "green";
      default:
        return "default";
    }
  };

  const getAlertIcon = (type: string) => {
    switch (type) {
      case "critical":
        return <AlertTriangle className="text-red-500" />;
      case "warning":
        return <AlertTriangle className="text-orange-500" />;
      case "info":
        return <Bell className="text-blue-500" />;
      default:
        return <Bell />;
    }
  };

  const formatTimeRemaining = (hours: number): string => {
    if (hours < 0) {
      return `超时 ${Math.abs(hours).toFixed(1)} 小时`;
    }
    if (hours < 1) {
      return `${Math.round(hours * 60)} 分钟`;
    }
    return `${hours.toFixed(1)} 小时`;
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
    <div className="space-y-6">
      {/* SLA概览指标 */}
      <Card
        title={
          <Space>
            <Clock className="text-blue-600" />
            <span>SLA概览</span>
          </Space>
        }
        extra={
          <Button icon={<Settings />} size="small">
            配置
          </Button>
        }
      >
        <Row gutter={16}>
          <Col span={6}>
            <Statistic
              title="总工单数"
              value={slaMetrics?.totalTickets}
              prefix={<Eye />}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="按时完成"
              value={slaMetrics?.onTime}
              valueStyle={{ color: "#3f8600" }}
              prefix={<CheckCircle />}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="有风险"
              value={slaMetrics?.atRisk}
              valueStyle={{ color: "#faad14" }}
              prefix={<AlertTriangle />}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="已超时"
              value={slaMetrics?.breached}
              valueStyle={{ color: "#cf1322" }}
              prefix={<AlertTriangle />}
            />
          </Col>
        </Row>

        <div className="mt-6">
          <div className="flex items-center justify-between mb-2">
            <Text>SLA合规率</Text>
            <Text strong>{slaMetrics?.slaCompliance}%</Text>
          </div>
          <Progress
            percent={slaMetrics?.slaCompliance || 0}
            status={
              slaMetrics && slaMetrics.slaCompliance < 90
                ? "exception"
                : "success"
            }
            strokeColor={{
              "0%": "#108ee9",
              "100%": "#87d068",
            }}
          />
        </div>

        <div className="mt-6 grid grid-cols-2 gap-4">
          <div>
            <Text type="secondary">平均响应时间</Text>
            <div className="flex items-center mt-1">
              <TrendingDown className="text-green-500 mr-2" />
              <Text strong>{slaMetrics?.avgResponseTime} 小时</Text>
            </div>
          </div>
          <div>
            <Text type="secondary">平均解决时间</Text>
            <div className="flex items-center mt-1">
              <TrendingUp className="text-blue-500 mr-2" />
              <Text strong>{slaMetrics?.avgResolutionTime} 小时</Text>
            </div>
          </div>
        </div>
      </Card>

      {/* 智能预警 */}
      {alerts.length > 0 && (
        <Card
          title={
            <Space>
              <Bell className="text-orange-600" />
              <span>智能预警</span>
              <Badge count={alerts.length} />
            </Space>
          }
        >
          <div className="space-y-3">
            {alerts.map((alert) => (
              <Alert
                key={alert.id}
                message={
                  <div className="flex items-center justify-between">
                    <span>{alert.message}</span>
                    <Space>
                      <Tag color={getPriorityColor(alert.priority)}>
                        {alert.priority.toUpperCase()}
                      </Tag>
                      <Text type="secondary">
                        剩余: {formatTimeRemaining(alert.timeRemaining)}
                      </Text>
                    </Space>
                  </div>
                }
                type={
                  alert.type === "critical"
                    ? "error"
                    : alert.type === "warning"
                    ? "warning"
                    : "info"
                }
                icon={getAlertIcon(alert.type)}
                action={
                  <Button size="small" type="link">
                    查看详情
                  </Button>
                }
                showIcon
              />
            ))}
          </div>
        </Card>
      )}

      {/* 有风险工单 */}
      {atRiskTickets.length > 0 && (
        <Card
          title={
            <Space>
              <AlertTriangle className="text-orange-600" />
              <span>有风险工单</span>
              <Badge count={atRiskTickets.length} />
            </Space>
          }
        >
          <div className="space-y-3">
            {atRiskTickets.map((ticket) => (
              <div
                key={ticket.id}
                className="p-3 border border-orange-200 rounded bg-orange-50"
              >
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <div className="flex items-center mb-2">
                      <Text strong className="mr-3">
                        {ticket.ticketNumber}
                      </Text>
                      <Tag color={getPriorityColor(ticket.priority)}>
                        {ticket.priority.toUpperCase()}
                      </Tag>
                      <Tag color="orange">有风险</Tag>
                    </div>
                    <Text className="block mb-1">{ticket.title}</Text>
                    <div className="flex items-center text-sm text-gray-500">
                      <span className="mr-4">分类: {ticket.category}</span>
                      <span className="mr-4">处理人: {ticket.assignee}</span>
                      <span>截止时间: {ticket.slaDeadline}</span>
                    </div>
                  </div>
                  <div className="text-right ml-4">
                    <div className="text-lg font-bold text-orange-600">
                      {formatTimeRemaining(ticket.timeRemaining)}
                    </div>
                    <Text type="secondary">剩余时间</Text>
                  </div>
                </div>
                <div className="mt-3 flex space-x-2">
                  <Button size="small" icon={<Eye />}>
                    查看详情
                  </Button>
                  <Button size="small" icon={<Zap />} type="primary">
                    立即处理
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* 已超时工单 */}
      {breachedTickets.length > 0 && (
        <Card
          title={
            <Space>
              <AlertTriangle className="text-red-600" />
              <span>已超时工单</span>
              <Badge count={breachedTickets.length} />
            </Space>
          }
        >
          <div className="space-y-3">
            {breachedTickets.map((ticket) => (
              <div
                key={ticket.id}
                className="p-3 border border-red-200 rounded bg-red-50"
              >
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <div className="flex items-center mb-2">
                      <Text strong className="mr-3">
                        {ticket.ticketNumber}
                      </Text>
                      <Tag color={getPriorityColor(ticket.priority)}>
                        {ticket.priority.toUpperCase()}
                      </Tag>
                      <Tag color="red">已超时</Tag>
                    </div>
                    <Text className="block mb-1">{ticket.title}</Text>
                    <div className="flex items-center text-sm text-gray-500">
                      <span className="mr-4">分类: {ticket.category}</span>
                      <span className="mr-4">处理人: {ticket.assignee}</span>
                      <span>截止时间: {ticket.slaDeadline}</span>
                    </div>
                  </div>
                  <div className="text-right ml-4">
                    <div className="text-lg font-bold text-red-600">
                      {formatTimeRemaining(ticket.timeRemaining)}
                    </div>
                    <Text type="secondary">超时时间</Text>
                  </div>
                </div>
                <div className="mt-3 flex space-x-2">
                  <Button size="small" icon={<Eye />}>
                    查看详情
                  </Button>
                  <Button size="small" icon={<Zap />} type="primary" danger>
                    紧急处理
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
};
