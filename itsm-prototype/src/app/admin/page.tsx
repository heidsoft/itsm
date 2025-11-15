"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import {
  Card,
  Row,
  Col,
  Statistic,
  Button,
  Typography,
  Tag,
  Space,
  Avatar,
  theme,
  List,
  Tooltip,
} from "antd";
import {
  Users,
  Shield,
  Workflow,
  Bell,
  BookOpen,
  Database,
  BarChart3,
  Settings,
  TrendingUp,
  AlertCircle,
  Activity,
  Clock,
  CheckCircle,
  XCircle,
  Zap,
  Globe,
  FileText,
  Calendar,
  Mail,
  ArrowUpRight,
  RefreshCw,
} from "lucide-react";

const { Title, Text, Paragraph } = Typography;

// 系统健康状态数据
const systemHealth = {
  overall: "excellent", // excellent, good, warning, critical
  uptime: "99.98%",
  lastUpdate: "2024-01-15 14:30:25",
  services: {
    database: "healthy",
    api: "healthy",
    cache: "healthy",
    queue: "warning",
  },
};

// 系统概览数据 - 增强版
const systemStats = [
  {
    title: "活跃用户",
    value: "1,234",
    change: "+12%",
    changeValue: "+132",
    icon: Users,
    color: "blue",
    trend: "up",
    description: "较上月新增132位用户",
  },
  {
    title: "运行中工作流",
    value: "45",
    change: "+6.7%",
    changeValue: "+3",
    icon: Workflow,
    color: "green",
    trend: "up",
    description: "本周新增3个工作流",
  },
  {
    title: "服务目录项",
    value: "89",
    change: "+5.9%",
    changeValue: "+5",
    icon: BookOpen,
    color: "purple",
    trend: "up",
    description: "新增5个服务项目",
  },
  {
    title: "系统告警",
    value: "2",
    change: "-33%",
    changeValue: "-1",
    icon: AlertCircle,
    color: "red",
    trend: "down",
    description: "较昨日减少1个告警",
  },
];

// 快速操作配置 - 重新分组
const quickActionGroups = {
  userManagement: {
    title: "用户与权限",
    description: "管理用户、角色和权限配置",
    actions: [
      {
        title: "用户管理",
        description: "管理系统用户账户",
        href: "/admin/users",
        icon: Users,
        color: "bg-blue-500",
        stats: "1,234 用户",
      },
      {
        title: "角色管理",
        description: "配置用户角色权限",
        href: "/admin/roles",
        icon: Shield,
        color: "bg-indigo-500",
        stats: "15 角色",
      },
      {
        title: "用户组管理",
        description: "管理用户组织结构",
        href: "/admin/groups",
        icon: Users,
        color: "bg-cyan-500",
        stats: "28 用户组",
      },
      {
        title: "权限配置",
        description: "配置系统权限矩阵",
        href: "/admin/permissions",
        icon: Shield,
        color: "bg-purple-500",
        stats: "156 权限",
      },
    ],
  },
  processConfig: {
    title: "流程配置",
    description: "配置业务流程和审批链",
    actions: [
      {
        title: "工作流配置",
        description: "设计和管理业务流程",
        href: "/admin/workflows",
        icon: Workflow,
        color: "bg-green-500",
        stats: "45 工作流",
      },
      {
        title: "审批链配置",
        description: "配置多级审批流程",
        href: "/admin/approval-chains",
        icon: FileText,
        color: "bg-emerald-500",
        stats: "12 审批链",
      },
      {
        title: "SLA定义",
        description: "设置服务级别协议",
        href: "/admin/sla-definitions",
        icon: Calendar,
        color: "bg-teal-500",
        stats: "8 SLA规则",
      },
      {
        title: "升级规则",
        description: "配置事件升级策略",
        href: "/admin/escalation-rules",
        icon: Zap,
        color: "bg-yellow-500",
        stats: "6 升级规则",
      },
    ],
  },
  systemConfig: {
    title: "系统配置",
    description: "管理系统全局设置和集成",
    actions: [
      {
        title: "服务目录",
        description: "管理服务目录和分类",
        href: "/admin/service-catalogs",
        icon: BookOpen,
        color: "bg-orange-500",
        stats: "89 服务项",
      },
      {
        title: "通知配置",
        description: "配置系统通知规则",
        href: "/admin/notifications",
        icon: Bell,
        color: "bg-red-500",
        stats: "24 通知规则",
      },
      {
        title: "邮件模板",
        description: "管理邮件通知模板",
        href: "/admin/email-templates",
        icon: Mail,
        color: "bg-pink-500",
        stats: "18 模板",
      },
      {
        title: "数据源配置",
        description: "配置外部数据源",
        href: "/admin/data-sources",
        icon: Database,
        color: "bg-slate-500",
        stats: "5 数据源",
      },
      {
        title: "集成配置",
        description: "管理第三方系统集成",
        href: "/admin/integrations",
        icon: Globe,
        color: "bg-violet-500",
        stats: "12 集成",
      },
      {
        title: "系统属性",
        description: "配置全局系统参数",
        href: "/admin/system-properties",
        icon: Settings,
        color: "bg-gray-500",
        stats: "67 属性",
      },
    ],
  },
};

// 最近活动数据
const recentActivities = [
  {
    id: 1,
    type: "user_created",
    title: "新用户注册",
    description: "张三 加入了系统",
    time: "2分钟前",
    icon: Users,
    color: "bg-blue-100 text-blue-600",
  },
  {
    id: 2,
    type: "workflow_updated",
    title: "工作流更新",
    description: "事件管理流程已更新",
    time: "1小时前",
    icon: Workflow,
    color: "bg-green-100 text-green-600",
  },
  {
    id: 3,
    type: "role_assigned",
    title: "角色分配",
    description: "为李四分配了管理员角色",
    time: "3小时前",
    icon: Shield,
    color: "bg-purple-100 text-purple-600",
  },
  {
    id: 4,
    type: "service_added",
    title: "服务目录更新",
    description: "新增云存储服务项",
    time: "5小时前",
    icon: BookOpen,
    color: "bg-orange-100 text-orange-600",
  },
  {
    id: 5,
    type: "notification_sent",
    title: "通知发送",
    description: "系统维护通知已发送",
    time: "1天前",
    icon: Bell,
    color: "bg-yellow-100 text-yellow-600",
  },
];

// 系统健康状态组件
const SystemHealthCard = () => {
  const { token } = theme.useToken();

  const getHealthStatus = (status: string) => {
    switch (status) {
      case "excellent":
        return {
          type: "success" as const,
          text: "优秀",
          color: token.colorSuccess,
        };
      case "good":
        return { type: "info" as const, text: "良好", color: token.colorInfo };
      case "warning":
        return {
          type: "warning" as const,
          text: "警告",
          color: token.colorWarning,
        };
      case "critical":
        return {
          type: "error" as const,
          text: "严重",
          color: token.colorError,
        };
      default:
        return {
          type: "default" as const,
          text: "未知",
          color: token.colorTextSecondary,
        };
    }
  };

  const getHealthIcon = (status: string) => {
    switch (status) {
      case "excellent":
      case "good":
        return CheckCircle;
      case "warning":
        return AlertCircle;
      case "critical":
        return XCircle;
      default:
        return Activity;
    }
  };

  const HealthIcon = getHealthIcon(systemHealth.overall);
  const healthStatus = getHealthStatus(systemHealth.overall);

  const serviceList = Object.entries(systemHealth.services).map(
    ([service, status]) => ({
      name:
        service === "database"
          ? "数据库"
          : service === "api"
          ? "API服务"
          : service === "cache"
          ? "缓存"
          : "消息队列",
      status,
      icon: getHealthIcon(status),
    })
  );

  return (
    <Card
      title={
        <Space>
          <Activity className="w-5 h-5" />
          系统健康状态
        </Space>
      }
      extra={
        <Button
          type="text"
          icon={<RefreshCw className="w-4 h-4" />}
          size="small"
        />
      }
    >
      <div style={{ marginBottom: token.marginLG }}>
        <Space align="center" size="large">
          <Avatar
            size={48}
            style={{ backgroundColor: healthStatus.color, border: "none" }}
            icon={<HealthIcon className="w-6 h-6" />}
          />
          <div>
            <Title level={3} style={{ margin: 0, color: healthStatus.color }}>
              {healthStatus.text}
            </Title>
            <Text type="secondary">系统运行时间: {systemHealth.uptime}</Text>
          </div>
        </Space>
      </div>

      <List
        grid={{ column: 2, gutter: 16 }}
        dataSource={serviceList}
        renderItem={(item) => {
          const ServiceIcon = item.icon;
          const serviceStatus = getHealthStatus(item.status);
          return (
            <List.Item>
              <Space align="center">
                <ServiceIcon
                  className="w-4 h-4"
                  style={{ color: serviceStatus.color }}
                />
                <Text>{item.name}</Text>
                <Tag color={serviceStatus.type}>{serviceStatus.text}</Tag>
              </Space>
            </List.Item>
          );
        }}
      />

      <div
        style={{
          marginTop: token.marginLG,
          paddingTop: token.paddingSM,
          borderTop: `1px solid ${token.colorBorder}`,
        }}
      >
        <Text type="secondary" style={{ fontSize: token.fontSizeSM }}>
          最后更新: {systemHealth.lastUpdate}
        </Text>
      </div>
    </Card>
  );
};

// 增强的统计卡片组件
const EnhancedStatCard = ({ stat }: { stat: (typeof systemStats)[0] }) => {
  const { token } = theme.useToken();
  const Icon = stat.icon;
  const isPositive = stat.trend === "up";

  const getColorByName = (colorName: string) => {
    switch (colorName) {
      case "blue":
        return token.colorPrimary;
      case "green":
        return token.colorSuccess;
      case "purple":
        return "#722ed1";
      case "red":
        return token.colorError;
      default:
        return token.colorPrimary;
    }
  };

  const trendColor = isPositive ? token.colorSuccess : token.colorError;

  return (
    <Card hoverable style={{ height: "100%" }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          marginBottom: token.marginLG,
        }}
      >
        <Avatar
          size={48}
          style={{
            backgroundColor: getColorByName(stat.color),
            border: "none",
          }}
          icon={<Icon className="w-6 h-6" />}
        />
        <Space
          align="center"
          style={{
            color: trendColor,
            fontSize: token.fontSizeSM,
            fontWeight: 600,
          }}
        >
          <TrendingUp className={`w-4 h-4 ${isPositive ? "" : "rotate-180"}`} />
          <span>{stat.change}</span>
        </Space>
      </div>

      <Statistic
        title={stat.title}
        value={stat.value}
        valueStyle={{ color: token.colorText, fontSize: 28, fontWeight: 700 }}
      />
      <Paragraph
        type="secondary"
        style={{ marginTop: token.marginSM, marginBottom: 0 }}
      >
        {stat.description}
      </Paragraph>
    </Card>
  );
};

// 快速操作组件
const QuickActionGroup = ({
  group,
  actions,
}: {
  group: {
    title: string;
    description: string;
    actions: Array<{
      title: string;
      description: string;
      href: string;
      icon: React.ComponentType<{ className?: string }>;
      color: string;
      stats: string;
    }>;
  };
  actions: Array<{
    title: string;
    description: string;
    href: string;
    icon: React.ComponentType<{ className?: string }>;
    color: string;
    stats: string;
  }>;
}) => {
  const { token } = theme.useToken();

  const getColorByClass = (colorClass: string) => {
    const colorMap: { [key: string]: string } = {
      "bg-blue-500": token.colorPrimary,
      "bg-indigo-500": "#6366f1",
      "bg-cyan-500": "#06b6d4",
      "bg-purple-500": "#722ed1",
      "bg-green-500": token.colorSuccess,
      "bg-emerald-500": "#10b981",
      "bg-teal-500": "#14b8a6",
      "bg-yellow-500": token.colorWarning,
      "bg-orange-500": "#f97316",
      "bg-red-500": token.colorError,
      "bg-pink-500": "#ec4899",
      "bg-slate-500": "#64748b",
      "bg-violet-500": "#8b5cf6",
      "bg-gray-500": token.colorTextSecondary,
    };
    return colorMap[colorClass] || token.colorPrimary;
  };

  return (
    <Card
      title={
        <div>
          <Title level={4} style={{ margin: 0 }}>
            {group.title}
          </Title>
          <Text type="secondary">{group.description}</Text>
        </div>
      }
    >
      <Row gutter={[16, 16]}>
        {actions.map((action, index) => {
          const Icon = action.icon;
          return (
            <Col xs={24} md={12} key={index}>
              <Link href={action.href} style={{ textDecoration: "none" }}>
                <Card
                  size="small"
                  hoverable
                  style={{ height: "100%" }}
                  styles={{ body: { padding: token.paddingMD } }}
                >
                  <Space align="start" style={{ width: "100%" }}>
                    <Avatar
                      size={40}
                      style={{
                        backgroundColor: getColorByClass(action.color),
                        border: "none",
                      }}
                      icon={<Icon className="w-5 h-5" />}
                    />
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div
                        style={{
                          display: "flex",
                          justifyContent: "space-between",
                          alignItems: "center",
                          marginBottom: 4,
                        }}
                      >
                        <Text strong style={{ color: token.colorText }}>
                          {action.title}
                        </Text>
                        <ArrowUpRight
                          className="w-4 h-4"
                          style={{ color: token.colorTextSecondary }}
                        />
                      </div>
                      <Tooltip title={action.description}>
                        <p
                          className="truncate"
                          style={{
                            fontSize: token.fontSizeSM,
                            margin: "4px 0",
                            lineHeight: 1.4,
                          }}
                        >
                          {action.description}
                        </p>
                      </Tooltip>
                      <Text
                        type="secondary"
                        style={{ fontSize: token.fontSizeSM, fontWeight: 500 }}
                      >
                        {action.stats}
                      </Text>
                    </div>
                  </Space>
                </Card>
              </Link>
            </Col>
          );
        })}
      </Row>
    </Card>
  );
};

const AdminDashboard = () => {
  const { token } = theme.useToken();
  const [currentTime, setCurrentTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  return (
    <div style={{ padding: token.paddingLG }}>
      {/* 页面头部 - 增强版 */}
      <Card
        style={{
          background: `linear-gradient(135deg, ${token.colorPrimary} 0%, #722ed1 100%)`,
          marginBottom: token.marginLG,
          border: "none",
        }}
        styles={{ body: { padding: token.paddingLG } }}
      >
        <Row justify="space-between" align="middle">
          <Col>
            <Title
              level={1}
              style={{
                color: "white",
                margin: 0,
                marginBottom: token.marginSM,
              }}
            >
              系统管理中心
            </Title>
            <Text
              style={{
                color: "rgba(255,255,255,0.9)",
                fontSize: token.fontSizeLG,
              }}
            >
              欢迎使用ITSM Pro企业级系统管理中心
            </Text>
            <br />
            <Text
              style={{
                color: "rgba(255,255,255,0.7)",
                fontSize: token.fontSizeSM,
                marginTop: token.marginXS,
              }}
            >
              这里您可以配置和管理整个ITSM平台的各项功能
            </Text>
          </Col>
          <Col style={{ textAlign: "right" }}>
            <div
              style={{
                color: "white",
                fontSize: 24,
                fontFamily: "monospace",
                marginBottom: 4,
              }}
            >
              {currentTime.toLocaleTimeString()}
            </div>
            <Text
              style={{
                color: "rgba(255,255,255,0.7)",
                fontSize: token.fontSizeSM,
              }}
            >
              {currentTime.toLocaleDateString("zh-CN", {
                year: "numeric",
                month: "long",
                day: "numeric",
                weekday: "long",
              })}
            </Text>
          </Col>
        </Row>
      </Card>

      {/* 系统概览统计 */}
      <div style={{ marginBottom: token.marginLG }}>
        <Title level={3} style={{ marginBottom: token.marginLG }}>
          <BarChart3
            className="w-5 h-5"
            style={{ marginRight: token.marginXS, color: token.colorPrimary }}
          />
          系统概览
        </Title>
        <Row gutter={[16, 16]}>
          {systemStats.map((stat, index) => (
            <Col xs={24} sm={12} lg={6} key={index}>
              <EnhancedStatCard stat={stat} />
            </Col>
          ))}
        </Row>
      </div>

      {/* 系统健康状态和最近活动 */}
      <Row gutter={[24, 24]} style={{ marginBottom: token.marginLG }}>
        <Col xs={24} lg={8}>
          <SystemHealthCard />
        </Col>

        {/* 最近活动 */}
        <Col xs={24} lg={16}>
          <Card
            title={
              <Space>
                <Activity className="w-5 h-5" />
                最近系统活动
              </Space>
            }
            extra={
              <Button type="link" size="small">
                查看全部
              </Button>
            }
            style={{ height: "100%" }}
          >
            <List
              dataSource={recentActivities}
              renderItem={(activity) => {
                const Icon = activity.icon;
                const getActivityColor = (colorClass: string) => {
                  if (colorClass.includes("blue")) return token.colorPrimary;
                  if (colorClass.includes("green")) return token.colorSuccess;
                  if (colorClass.includes("purple")) return "#722ed1";
                  if (colorClass.includes("orange")) return "#f97316";
                  if (colorClass.includes("yellow")) return token.colorWarning;
                  return token.colorPrimary;
                };

                return (
                  <List.Item
                    style={{
                      padding: `${token.paddingSM}px 0`,
                      borderBottom: `1px solid ${token.colorBorder}`,
                    }}
                  >
                    <List.Item.Meta
                      avatar={
                        <Avatar
                          style={{
                            backgroundColor: getActivityColor(activity.color),
                          }}
                          icon={<Icon className="w-4 h-4" />}
                        />
                      }
                      title={
                        <div
                          style={{
                            display: "flex",
                            justifyContent: "space-between",
                            alignItems: "center",
                          }}
                        >
                          <Text strong>{activity.title}</Text>
                          <Space
                            align="center"
                            style={{
                              color: token.colorTextSecondary,
                              fontSize: token.fontSizeSM,
                            }}
                          >
                            <Clock className="w-3 h-3" />
                            {activity.time}
                          </Space>
                        </div>
                      }
                      description={activity.description}
                    />
                  </List.Item>
                );
              }}
            />
          </Card>
        </Col>
      </Row>

      {/* 快速操作分组 */}
      <div style={{ marginBottom: token.marginLG }}>
        <Title level={3} style={{ marginBottom: token.marginLG }}>
          <Zap
            className="w-5 h-5"
            style={{ marginRight: token.marginXS, color: token.colorPrimary }}
          />
          快速操作
        </Title>
        <Space direction="vertical" size="large" style={{ width: "100%" }}>
          {Object.entries(quickActionGroups).map(([key, group]) => (
            <QuickActionGroup key={key} group={group} actions={group.actions} />
          ))}
        </Space>
      </div>

      {/* 系统信息和帮助 */}
      <Row gutter={[24, 24]}>
        {/* 系统信息 */}
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <Settings className="w-5 h-5" />
                系统信息
              </Space>
            }
            style={{ height: "100%" }}
          >
            <Space direction="vertical" size="middle" style={{ width: "100%" }}>
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <Text type="secondary">系统版本</Text>
                <Text strong>ITSM Pro v2.0.1</Text>
              </div>
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <Text type="secondary">数据库版本</Text>
                <Text strong>PostgreSQL 14.2</Text>
              </div>
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <Text type="secondary">许可证状态</Text>
                <Tag color="success">已激活</Tag>
              </div>
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <Text type="secondary">许可证到期</Text>
                <Text strong>2024-12-31</Text>
              </div>
            </Space>
          </Card>
        </Col>

        {/* 帮助和支持 */}
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <FileText className="w-5 h-5" />
                帮助和支持
              </Space>
            }
            style={{ height: "100%" }}
          >
            <List
              dataSource={[
                { title: "系统配置指南", href: "#" },
                { title: "API文档", href: "#" },
                { title: "技术支持", href: "#" },
                { title: "更新日志", href: "#" },
              ]}
              renderItem={(item) => (
                <List.Item>
                  <Link
                    href={item.href}
                    style={{ textDecoration: "none", width: "100%" }}
                  >
                    <div
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                        padding: `${token.paddingSM}px 0`,
                        width: "100%",
                      }}
                    >
                      <Text>{item.title}</Text>
                      <ArrowUpRight
                        className="w-4 h-4"
                        style={{ color: token.colorTextSecondary }}
                      />
                    </div>
                  </Link>
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default AdminDashboard;
