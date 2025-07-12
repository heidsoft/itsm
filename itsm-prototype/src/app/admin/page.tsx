"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
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
  const getHealthColor = (status: string) => {
    switch (status) {
      case "excellent":
        return "text-green-600 bg-green-100";
      case "good":
        return "text-blue-600 bg-blue-100";
      case "warning":
        return "text-yellow-600 bg-yellow-100";
      case "critical":
        return "text-red-600 bg-red-100";
      default:
        return "text-gray-600 bg-gray-100";
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

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900">系统健康状态</h3>
        <button className="p-2 hover:bg-gray-100 rounded-lg transition-colors">
          <RefreshCw className="w-4 h-4 text-gray-500" />
        </button>
      </div>

      <div className="flex items-center space-x-3 mb-4">
        <div
          className={`p-2 rounded-lg ${getHealthColor(systemHealth.overall)}`}
        >
          <HealthIcon className="w-5 h-5" />
        </div>
        <div>
          <p className="text-2xl font-bold text-gray-900 capitalize">
            {systemHealth.overall === "excellent"
              ? "优秀"
              : systemHealth.overall === "good"
              ? "良好"
              : systemHealth.overall === "warning"
              ? "警告"
              : "严重"}
          </p>
          <p className="text-sm text-gray-600">
            系统运行时间: {systemHealth.uptime}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        {Object.entries(systemHealth.services).map(([service, status]) => {
          const ServiceIcon = getHealthIcon(status);
          return (
            <div key={service} className="flex items-center space-x-2">
              <ServiceIcon
                className={`w-4 h-4 ${getHealthColor(status).split(" ")[0]}`}
              />
              <span className="text-sm text-gray-700 capitalize">
                {service === "database"
                  ? "数据库"
                  : service === "api"
                  ? "API服务"
                  : service === "cache"
                  ? "缓存"
                  : "消息队列"}
              </span>
            </div>
          );
        })}
      </div>

      <div className="mt-4 pt-4 border-t border-gray-200">
        <p className="text-xs text-gray-500">
          最后更新: {systemHealth.lastUpdate}
        </p>
      </div>
    </div>
  );
};

// 增强的统计卡片组件
const EnhancedStatCard = ({ stat }: { stat: (typeof systemStats)[0] }) => {
  const Icon = stat.icon;
  const isPositive = stat.trend === "up";

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6 hover:shadow-md transition-all duration-300 group">
      <div className="flex items-center justify-between mb-4">
        <div
          className={`p-3 rounded-lg bg-${stat.color}-100 group-hover:scale-110 transition-transform`}
        >
          <Icon className={`w-6 h-6 text-${stat.color}-600`} />
        </div>
        <div
          className={`flex items-center space-x-1 text-sm font-medium ${
            isPositive ? "text-green-600" : "text-red-600"
          }`}
        >
          <TrendingUp className={`w-4 h-4 ${isPositive ? "" : "rotate-180"}`} />
          <span>{stat.change}</span>
        </div>
      </div>

      <div>
        <p className="text-sm font-medium text-gray-600 mb-1">{stat.title}</p>
        <p className="text-3xl font-bold text-gray-900 mb-2">{stat.value}</p>
        <p className="text-sm text-gray-500">{stat.description}</p>
      </div>
    </div>
  );
};

// 快速操作组件
const QuickActionGroup = ({
  group,
  actions,
}: {
  group: any;
  actions: any[];
}) => {
  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
      <div className="mb-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-2">
          {group.title}
        </h3>
        <p className="text-sm text-gray-600">{group.description}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {actions.map((action, index) => {
          const Icon = action.icon;
          return (
            <Link
              key={index}
              href={action.href}
              className="group p-4 rounded-lg border border-gray-200 hover:border-blue-300 hover:shadow-md transition-all duration-200"
            >
              <div className="flex items-start space-x-3">
                <div
                  className={`${action.color} p-2 rounded-lg group-hover:scale-110 transition-transform`}
                >
                  <Icon className="w-5 h-5 text-white" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <h4 className="text-sm font-semibold text-gray-900 group-hover:text-blue-600 transition-colors">
                      {action.title}
                    </h4>
                    <ArrowUpRight className="w-4 h-4 text-gray-400 group-hover:text-blue-600 transition-colors" />
                  </div>
                  <p className="text-xs text-gray-600 mt-1 line-clamp-2">
                    {action.description}
                  </p>
                  <p className="text-xs text-gray-500 mt-2 font-medium">
                    {action.stats}
                  </p>
                </div>
              </div>
            </Link>
          );
        })}
      </div>
    </div>
  );
};

const AdminDashboard = () => {
  const [currentTime, setCurrentTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  return (
    <div className="space-y-8">
      {/* 页面头部 - 增强版 */}
      <div className="bg-gradient-to-r from-blue-600 to-purple-600 rounded-xl p-8 text-white">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-4xl font-bold mb-2">系统管理中心</h1>
            <p className="text-blue-100 text-lg">
              欢迎使用ITSM Pro企业级系统管理中心
            </p>
            <p className="text-blue-200 text-sm mt-2">
              这里您可以配置和管理整个ITSM平台的各项功能
            </p>
          </div>
          <div className="text-right">
            <div className="text-2xl font-mono">
              {currentTime.toLocaleTimeString()}
            </div>
            <div className="text-blue-200 text-sm">
              {currentTime.toLocaleDateString("zh-CN", {
                year: "numeric",
                month: "long",
                day: "numeric",
                weekday: "long",
              })}
            </div>
          </div>
        </div>
      </div>

      {/* 系统概览统计 */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-6 flex items-center">
          <BarChart3 className="w-5 h-5 mr-2" />
          系统概览
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {systemStats.map((stat, index) => (
            <EnhancedStatCard key={index} stat={stat} />
          ))}
        </div>
      </div>

      {/* 系统健康状态和最近活动 */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <SystemHealthCard />

        {/* 最近活动 */}
        <div className="lg:col-span-2 bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-lg font-semibold text-gray-900 flex items-center">
              <Activity className="w-5 h-5 mr-2" />
              最近系统活动
            </h3>
            <Link
              href="#"
              className="text-sm text-blue-600 hover:text-blue-700 font-medium"
            >
              查看全部
            </Link>
          </div>

          <div className="space-y-4">
            {recentActivities.map((activity) => {
              const Icon = activity.icon;
              return (
                <div
                  key={activity.id}
                  className="flex items-start space-x-3 p-3 rounded-lg hover:bg-gray-50 transition-colors"
                >
                  <div className={`p-2 rounded-lg ${activity.color}`}>
                    <Icon className="w-4 h-4" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between">
                      <p className="text-sm font-medium text-gray-900">
                        {activity.title}
                      </p>
                      <p className="text-xs text-gray-500 flex items-center">
                        <Clock className="w-3 h-3 mr-1" />
                        {activity.time}
                      </p>
                    </div>
                    <p className="text-sm text-gray-600 mt-1">
                      {activity.description}
                    </p>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* 快速操作分组 */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-6 flex items-center">
          <Zap className="w-5 h-5 mr-2" />
          快速操作
        </h2>
        <div className="space-y-6">
          {Object.entries(quickActionGroups).map(([key, group]) => (
            <QuickActionGroup key={key} group={group} actions={group.actions} />
          ))}
        </div>
      </div>

      {/* 系统信息和帮助 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 系统信息 */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center">
            <Settings className="w-5 h-5 mr-2" />
            系统信息
          </h3>
          <div className="space-y-3">
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">系统版本</span>
              <span className="text-sm font-medium text-gray-900">
                ITSM Pro v2.0.1
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">数据库版本</span>
              <span className="text-sm font-medium text-gray-900">
                PostgreSQL 14.2
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">许可证状态</span>
              <span className="text-sm font-medium text-green-600">已激活</span>
            </div>
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">许可证到期</span>
              <span className="text-sm font-medium text-gray-900">
                2024-12-31
              </span>
            </div>
          </div>
        </div>

        {/* 帮助和支持 */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center">
            <FileText className="w-5 h-5 mr-2" />
            帮助和支持
          </h3>
          <div className="space-y-3">
            <Link
              href="#"
              className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <span className="text-sm text-gray-700">系统配置指南</span>
              <ArrowUpRight className="w-4 h-4 text-gray-400" />
            </Link>
            <Link
              href="#"
              className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <span className="text-sm text-gray-700">API文档</span>
              <ArrowUpRight className="w-4 h-4 text-gray-400" />
            </Link>
            <Link
              href="#"
              className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <span className="text-sm text-gray-700">技术支持</span>
              <ArrowUpRight className="w-4 h-4 text-gray-400" />
            </Link>
            <Link
              href="#"
              className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <span className="text-sm text-gray-700">更新日志</span>
              <ArrowUpRight className="w-4 h-4 text-gray-400" />
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AdminDashboard;
