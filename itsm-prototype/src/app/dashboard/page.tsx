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
                  </div>
                </Card>
              </Col>

              {/* SLA 合规性 */}
              <Col xs={24} lg={12}>
                <Card 
                  title={(
                    <div className="flex items-center space-x-3">
                      <div className="w-10 h-10 bg-green-50 rounded-lg flex items-center justify-center">
                        <CheckCircle size={20} className="text-green-600" />
                      </div>
                      <div>
                        <Title level={4} className="mb-0 text-gray-800">SLA 合规性</Title>
                        <Text className="text-xs text-gray-500">服务水平协议</Text>
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
                  <div className="space-y-6">
                    <div className="relative p-5 bg-gradient-to-r from-blue-50 to-blue-100 rounded-xl border border-blue-100 hover:shadow-sm transition-all duration-200">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-4">
                          <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center">
                            <Activity size={24} className="text-blue-600" />
                          </div>
                          <div>
                            <Text className="font-semibold text-gray-700 block">解决时间</Text>
                            <Text className="text-xs text-gray-500">&lt; 24小时</Text>
                          </div>
                        </div>
                        <div className="text-right">
                          <div className="text-3xl font-bold text-blue-600">95.2</div>
                          <Text className="text-sm text-gray-500">%</Text>
                        </div>
                      </div>
                      <div className="mt-3">
                        <Progress 
                          percent={95.2} 
                          strokeColor="#1890ff" 
                          trailColor="#f0f0f0"
                          size={6}
                          showInfo={false}
                        />
                      </div>
                    </div>
                    
                    <div className="relative p-5 bg-gradient-to-r from-purple-50 to-purple-100 rounded-xl border border-purple-100 hover:shadow-sm transition-all duration-200">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-4">
                          <div className="w-12 h-12 bg-purple-100 rounded-full flex items-center justify-center">
                            <CheckCircle size={24} className="text-purple-600" />
                          </div>
                          <div>
                            <Text className="font-semibold text-gray-700 block">系统可用性</Text>
                            <Text className="text-xs text-gray-500">7×24小时</Text>
                          </div>
                        </div>
                        <div className="text-right">
                          <div className="text-3xl font-bold text-purple-600">99.9</div>
                          <Text className="text-sm text-gray-500">%</Text>
                        </div>
                      </div>
                      <div className="mt-3">
                        <Progress 
                          percent={99.9} 
                          strokeColor="#722ed1" 
                          trailColor="#f0f0f0"
                          size={6}
                          showInfo={false}
                        />
                      </div>
                    </div>
                  </div>
                </Card>
              </Col>
            </Row>
          </div>
        </div>
      </LoadingStateManager>
    </ErrorBoundary>
  );
}
