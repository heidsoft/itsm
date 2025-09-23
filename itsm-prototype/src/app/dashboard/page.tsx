"use client";

import React from "react";
import { Card, Row, Col, Typography, Space, Button, Progress } from 'antd';
import { 
  RefreshCw, 
  Settings, 
  FileText, 
  AlertTriangle, 
  Users, 
  Clock, 
  Activity, 
  CheckCircle, 
  TrendingUp, 
  TrendingDown
} from 'lucide-react';
import { useDashboardData } from "../hooks/useDashboardData";
import ErrorBoundary from "../components/common/ErrorBoundary";
import LoadingStateManager from "../components/common/LoadingStateManager";

const { Title, Text } = Typography;

export default function DashboardPage() {
  const { 
    data, 
    loading, 
    error, 
    refreshData, 
    lastUpdated,
    retryCount
  } = useDashboardData();

  // KPI数据
  const kpiData = [
    {
      title: "总工单数",
      value: data?.kpiData?.totalTickets?.value || 0,
      change: data?.kpiData?.totalTickets?.change || 0,
      trend: data?.kpiData?.totalTickets?.trend || 'up',
      icon: FileText,
      color: "#1890ff"
    },
    {
      title: "待处理事件",
      value: data?.kpiData?.pendingEvents?.value || 0,
      change: data?.kpiData?.pendingEvents?.change || 0,
      trend: data?.kpiData?.pendingEvents?.trend || 'down',
      icon: AlertTriangle,
      color: "#ff4d4f"
    },
    {
      title: "活跃用户",
      value: data?.kpiData?.activeUsers?.value || 0,
      change: data?.kpiData?.activeUsers?.change || 0,
      trend: data?.kpiData?.activeUsers?.trend || 'up',
      icon: Users,
      color: "#52c41a"
    },
    {
      title: "平均响应时间",
      value: data?.kpiData?.avgResponseTime?.value || 0,
      change: data?.kpiData?.avgResponseTime?.change || 0,
      trend: data?.kpiData?.avgResponseTime?.trend || 'down',
      icon: Clock,
      color: "#faad14",
      suffix: "min"
    }
  ];

  return (
    <ErrorBoundary>
      <LoadingStateManager
        loading={loading}
        error={error}
        onRetry={refreshData}
        retryCount={retryCount}
      >
        <div className="min-h-screen bg-gray-50">
          <div className="max-w-7xl mx-auto p-6 space-y-8">
            {/* 页面标题 */}
            <div className="flex items-center justify-between">
              <div>
                <Title level={2} className="mb-2 text-gray-800">
                  仪表盘
                </Title>
                <Text className="text-gray-500">
                  系统概览和关键指标监控
                  {lastUpdated && (
                    <span className="ml-2">
                      · 最后更新: {new Date(lastUpdated).toLocaleTimeString()}
                    </span>
                  )}
                </Text>
              </div>
              <Space>
                <Button 
                  icon={<RefreshCw size={16} />} 
                  onClick={refreshData}
                  loading={loading}
                >
                  刷新
                </Button>
                <Button 
                  icon={<Settings size={16} />} 
                  type="default"
                >
                  设置
                </Button>
              </Space>
            </div>

            {/* KPI 指标卡片 */}
            <Row gutter={[24, 24]}>
              {kpiData.map((kpi, index) => {
                const IconComponent = kpi.icon;
                return (
                  <Col xs={24} sm={12} lg={6} key={index}>
                    <Card 
                      className="h-full border-0 shadow-sm hover:shadow-md transition-all duration-300 bg-white"
                      bodyStyle={{ padding: '24px' }}
                    >
                      {/* 图标和标题 */}
                      <div className="flex items-center justify-between mb-4">
                        <div className="flex items-center space-x-3">
                          <div 
                            className="w-12 h-12 rounded-lg flex items-center justify-center"
                            style={{ backgroundColor: `${kpi.color}15` }}
                          >
                            <IconComponent 
                              size={24} 
                              style={{ color: kpi.color }}
                            />
                          </div>
                          <div>
                            <Text className="text-sm text-gray-500 font-medium block">
                              {kpi.title}
                            </Text>
                          </div>
                        </div>
                        {kpi.trend === 'up' ? (
                          <TrendingUp size={20} className="text-green-500" />
                        ) : (
                          <TrendingDown size={20} className="text-red-500" />
                        )}
                      </div>
                      
                      {/* 数值显示 */}
                      <div className="mb-4">
                        <div className="flex items-baseline space-x-2">
                          <span 
                            className="text-3xl font-bold"
                            style={{ color: kpi.color }}
                          >
                            {kpi.value.toLocaleString()}
                          </span>
                          {kpi.suffix && (
                            <span className="text-lg text-gray-400 font-medium">
                              {kpi.suffix}
                            </span>
                          )}
                        </div>
                      </div>
                      
                      {/* 变化百分比 */}
                      <div className="flex items-center justify-between">
                        <span 
                          className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-semibold ${
                            kpi.trend === 'up' 
                              ? 'bg-green-100 text-green-800' 
                              : 'bg-red-100 text-red-800'
                          }`}
                        >
                          {kpi.change > 0 ? '+' : ''}{kpi.change}%
                        </span>
                        <Text className="text-xs text-gray-400 font-medium">
                          较上月
                        </Text>
                      </div>
                    </Card>
                  </Col>
                );
              })}
            </Row>

            {/* 主要内容区域 */}
            <Row gutter={[32, 32]}>
              {/* 系统状态 */}
              <Col xs={24} lg={12}>
                <Card 
                  title={(
                    <div className="flex items-center justify-between">
                      <div className="flex items-center space-x-3">
                        <div className="w-10 h-10 bg-blue-50 rounded-lg flex items-center justify-center">
                          <Activity size={20} className="text-blue-600" />
                        </div>
                        <div>
                          <Title level={4} className="mb-0 text-gray-800">系统状态</Title>
                          <Text className="text-xs text-gray-500">实时监控</Text>
                        </div>
                      </div>
                      <div className="w-3 h-3 bg-green-400 rounded-full animate-pulse" />
                    </div>
                  )}
                  className="h-full border-0 shadow-sm hover:shadow-md transition-shadow duration-300"
                  styles={{
                    header: { 
                      borderBottom: '1px solid #f0f0f0',
                      paddingBottom: '16px',
                      marginBottom: '8px'
                    }
                  }}
                >
                  <div className="space-y-8">
                    <div className="relative">
                      <div className="flex justify-between items-center mb-3">
                        <div className="flex items-center space-x-2">
                          <div className="w-2 h-2 bg-blue-500 rounded-full" />
                          <Text className="font-semibold text-gray-700">CPU 使用率</Text>
                        </div>
                        <div className="text-right">
                          <Text className="text-2xl font-bold text-blue-600">65</Text>
                          <Text className="text-sm text-gray-400 ml-1">%</Text>
                        </div>
                      </div>
                      <Progress 
                        percent={65} 
                        strokeColor={{
                          '0%': '#1890ff',
                          '100%': '#40a9ff',
                        }}
                        trailColor="#f0f0f0"
                        size={8}
                        showInfo={false}
                        className="mb-2"
                      />
                      <Text className="text-xs text-gray-400">正常范围: 0-80%</Text>
                    </div>
                    
                    <div className="relative">
                      <div className="flex justify-between items-center mb-3">
                        <div className="flex items-center space-x-2">
                          <div className="w-2 h-2 bg-green-500 rounded-full" />
                          <Text className="font-semibold text-gray-700">内存使用率</Text>
                        </div>
                        <div className="text-right">
                          <Text className="text-2xl font-bold text-green-600">78</Text>
                          <Text className="text-sm text-gray-400 ml-1">%</Text>
                        </div>
                      </div>
                      <Progress 
                        percent={78} 
                        strokeColor={{
                          '0%': '#52c41a',
                          '100%': '#73d13d',
                        }}
                        trailColor="#f0f0f0"
                        size={8}
                        showInfo={false}
                        className="mb-2"
                      />
                      <Text className="text-xs text-gray-400">正常范围: 0-85%</Text>
                    </div>

                    <div className="relative">
                      <div className="flex justify-between items-center mb-3">
                        <div className="flex items-center space-x-2">
                          <div className="w-2 h-2 bg-orange-500 rounded-full" />
                          <Text className="font-semibold text-gray-700">磁盘使用率</Text>
                        </div>
                        <div className="text-right">
                          <Text className="text-2xl font-bold text-orange-600">45</Text>
                          <Text className="text-sm text-gray-400 ml-1">%</Text>
                        </div>
                      </div>
                      <Progress 
                        percent={45} 
                        strokeColor={{
                          '0%': '#fa8c16',
                          '100%': '#ffa940',
                        }}
                        trailColor="#f0f0f0"
                        size={8}
                        showInfo={false}
                        className="mb-2"
                      />
                      <Text className="text-xs text-gray-400">正常范围: 0-90%</Text>
                    </div>
                  </div>
                </Card>
              </Col>

              {/* 最近工单 */}
              <Col xs={24} lg={12}>
                <Card 
                  title={(
                    <div className="flex items-center justify-between">
                      <div className="flex items-center space-x-3">
                        <div className="w-10 h-10 bg-purple-50 rounded-lg flex items-center justify-center">
                          <FileText size={20} className="text-purple-600" />
                        </div>
                        <div>
                          <Title level={4} className="mb-0 text-gray-800">最近工单</Title>
                          <Text className="text-xs text-gray-500">最新5条</Text>
                        </div>
                      </div>
                      <Button 
                        type="link" 
                        size="small"
                        onClick={() => window.open('/tickets', '_blank')}
                      >
                        查看全部
                      </Button>
                    </div>
                  )}
                  className="h-full border-0 shadow-sm hover:shadow-md transition-shadow duration-300"
                  styles={{
                    header: { 
                      borderBottom: '1px solid #f0f0f0',
                      paddingBottom: '16px',
                      marginBottom: '8px'
                    }
                  }}
                >
                  <div className="space-y-4">
                    {data?.recentTickets?.slice(0, 5).map((ticket, index) => (
                      <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer">
                        <div className="flex items-center space-x-3">
                          <div className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold text-white ${
                            ticket.priority === 'high' ? 'bg-red-500' :
                            ticket.priority === 'medium' ? 'bg-orange-500' : 'bg-green-500'
                          }`}>
                            {ticket.priority === 'high' ? 'H' : ticket.priority === 'medium' ? 'M' : 'L'}
                          </div>
                          <div>
                            <Text className="font-semibold text-gray-800 block">{ticket.title}</Text>
                            <Text className="text-xs text-gray-500">{ticket.assignee} · {ticket.time}</Text>
                          </div>
                        </div>
                        <div className="text-right">
                          <div className={`inline-flex px-2 py-1 rounded-full text-xs font-medium ${
                            ticket.status === 'processing' ? 'bg-blue-100 text-blue-800' :
                            ticket.status === 'pending' ? 'bg-yellow-100 text-yellow-800' : 'bg-green-100 text-green-800'
                          }`}>
                            {ticket.status === 'processing' ? '处理中' : 
                             ticket.status === 'pending' ? '待处理' : '已解决'}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </Card>
              </Col>

              {/* 系统告警 */}
              <Col xs={24} lg={12}>
                <Card 
                  title={(
                    <div className="flex items-center justify-between">
                      <div className="flex items-center space-x-3">
                        <div className="w-10 h-10 bg-red-50 rounded-lg flex items-center justify-center">
                          <AlertTriangle size={20} className="text-red-600" />
                        </div>
                        <div>
                          <Title level={4} className="mb-0 text-gray-800">系统告警</Title>
                          <Text className="text-xs text-gray-500">需要关注</Text>
                        </div>
                      </div>
                      <div className="flex items-center space-x-2">
                        <div className="w-2 h-2 bg-red-400 rounded-full animate-pulse" />
                        <Text className="text-xs text-red-600 font-medium">
                          {data?.systemAlerts?.length || 0} 条告警
                        </Text>
                      </div>
                    </div>
                  )}
                  className="h-full border-0 shadow-sm hover:shadow-md transition-shadow duration-300"
                  styles={{
                    header: { 
                      borderBottom: '1px solid #f0f0f0',
                      paddingBottom: '16px',
                      marginBottom: '8px'
                    }
                  }}
                >
                  <div className="space-y-3">
                    {data?.systemAlerts?.map((alert, index) => (
                      <div key={index} className={`p-3 rounded-lg border-l-4 ${
                        alert.type === 'error' ? 'bg-red-50 border-red-500' :
                        alert.type === 'warning' ? 'bg-yellow-50 border-yellow-500' :
                        alert.type === 'info' ? 'bg-blue-50 border-blue-500' : 'bg-green-50 border-green-500'
                      }`}>
                        <div className="flex items-start justify-between">
                          <div className="flex-1">
                            <Text className="font-medium text-gray-800 block mb-1">
                              {alert.message}
                            </Text>
                            <Text className="text-xs text-gray-500">
                              {alert.time} · 严重程度: {
                                alert.severity === 'high' ? '高' :
                                alert.severity === 'medium' ? '中' : '低'
                              }
                            </Text>
                          </div>
                          <div className={`ml-3 px-2 py-1 rounded text-xs font-medium ${
                            alert.severity === 'high' ? 'bg-red-100 text-red-800' :
                            alert.severity === 'medium' ? 'bg-yellow-100 text-yellow-800' : 'bg-green-100 text-green-800'
                          }`}>
                            {alert.severity === 'high' ? '高' : alert.severity === 'medium' ? '中' : '低'}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </Card>
              </Col>

              {/* 最近活动 */}
              <Col xs={24} lg={12}>
                <Card 
                  title={(
                    <div className="flex items-center justify-between">
                      <div className="flex items-center space-x-3">
                        <div className="w-10 h-10 bg-indigo-50 rounded-lg flex items-center justify-center">
                          <Users size={20} className="text-indigo-600" />
                        </div>
                        <div>
                          <Title level={4} className="mb-0 text-gray-800">最近活动</Title>
                          <Text className="text-xs text-gray-500">团队动态</Text>
                        </div>
                      </div>
                    </div>
                  )}
                  className="h-full border-0 shadow-sm hover:shadow-md transition-shadow duration-300"
                  styles={{
                    header: { 
                      borderBottom: '1px solid #f0f0f0',
                      paddingBottom: '16px',
                      marginBottom: '8px'
                    }
                  }}
                >
                  <div className="space-y-4">
                    {data?.recentActivities?.map((activity, index) => (
                      <div key={index} className="flex items-center space-x-3 p-2 hover:bg-gray-50 rounded-lg transition-colors">
                        <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-white text-sm font-bold">
                          {activity.avatar}
                        </div>
                        <div className="flex-1">
                          <Text className="text-sm text-gray-800">
                            <span className="font-semibold">{activity.operator}</span>
                            <span className="mx-1">{activity.action}</span>
                            <span className="font-medium">{activity.target}</span>
                          </Text>
                          <Text className="text-xs text-gray-500 block">{activity.time}</Text>
                        </div>
                      </div>
                    ))}
                  </div>
                </Card>
              </Col>
            </Row>

            {/* 快速操作区域 */}
            <Row gutter={[24, 24]}>
              <Col span={24}>
                <Card 
                  title={(
                    <div className="flex items-center space-x-3">
                      <div className="w-10 h-10 bg-gradient-to-br from-green-500 to-blue-600 rounded-lg flex items-center justify-center">
                        <CheckCircle size={20} className="text-white" />
                      </div>
                      <div>
                        <Title level={4} className="mb-0 text-gray-800">快速操作</Title>
                        <Text className="text-xs text-gray-500">常用功能入口</Text>
                      </div>
                    </div>
                  )}
                  className="border-0 shadow-sm"
                >
                  <Row gutter={[16, 16]}>
                    <Col xs={12} sm={8} md={6} lg={4}>
                      <Button 
                        type="default" 
                        size="large" 
                        block
                        icon={<FileText size={18} />}
                        className="h-16 flex flex-col items-center justify-center border-dashed hover:border-blue-500 hover:text-blue-600 transition-all"
                        onClick={() => window.open('/tickets/create', '_blank')}
                      >
                        <span className="text-xs mt-1">创建工单</span>
                      </Button>
                    </Col>
                    <Col xs={12} sm={8} md={6} lg={4}>
                      <Button 
                        type="default" 
                        size="large" 
                        block
                        icon={<AlertTriangle size={18} />}
                        className="h-16 flex flex-col items-center justify-center border-dashed hover:border-red-500 hover:text-red-600 transition-all"
                        onClick={() => window.open('/incidents/new', '_blank')}
                      >
                        <span className="text-xs mt-1">报告事件</span>
                      </Button>
                    </Col>
                    <Col xs={12} sm={8} md={6} lg={4}>
                      <Button 
                        type="default" 
                        size="large" 
                        block
                        icon={<Users size={18} />}
                        className="h-16 flex flex-col items-center justify-center border-dashed hover:border-purple-500 hover:text-purple-600 transition-all"
                        onClick={() => window.open('/admin', '_blank')}
                      >
                        <span className="text-xs mt-1">用户管理</span>
                      </Button>
                    </Col>
                    <Col xs={12} sm={8} md={6} lg={4}>
                      <Button 
                        type="default" 
                        size="large" 
                        block
                        icon={<Activity size={18} />}
                        className="h-16 flex flex-col items-center justify-center border-dashed hover:border-green-500 hover:text-green-600 transition-all"
                        onClick={() => window.open('/cmdb', '_blank')}
                      >
                        <span className="text-xs mt-1">配置管理</span>
                      </Button>
                    </Col>
                    <Col xs={12} sm={8} md={6} lg={4}>
                      <Button 
                        type="default" 
                        size="large" 
                        block
                        icon={<Clock size={18} />}
                        className="h-16 flex flex-col items-center justify-center border-dashed hover:border-orange-500 hover:text-orange-600 transition-all"
                        onClick={() => window.open('/changes', '_blank')}
                      >
                        <span className="text-xs mt-1">变更管理</span>
                      </Button>
                    </Col>
                    <Col xs={12} sm={8} md={6} lg={4}>
                      <Button 
                        type="default" 
                        size="large" 
                        block
                        icon={<Settings size={18} />}
                        className="h-16 flex flex-col items-center justify-center border-dashed hover:border-gray-500 hover:text-gray-600 transition-all"
                        onClick={refreshData}
                      >
                        <span className="text-xs mt-1">刷新数据</span>
                      </Button>
                    </Col>
                  </Row>
                </Card>
              </Col>
            </Row>
          </div>
        </div>
      </LoadingStateManager>
    </ErrorBoundary>
  );
}
