"use client";

import React, { useState } from "react";
// AppLayout is handled by layout.tsx
import { AIMetrics } from "../components/AIMetrics";
import {
  Card,
  Row,
  Col,
  Statistic,
  Progress,
  Alert,
  List,
  Tag,
  Timeline,
  Button,
} from "antd";
import {
  Users,
  Clock,
  CheckCircle,
  Activity,
  BarChart3,
  FileText,
  AlertTriangle,
  Settings,
} from "lucide-react";

export default function DashboardPage() {
  const [systemAlerts] = useState([
    {
      type: "warning",
      message: "系统负载较高，建议检查服务器状态",
      time: "2分钟前",
    },
    {
      type: "info",
      message: "数据库备份已完成",
      time: "5分钟前",
    },
  ]);

  const [recentTickets] = useState([
    {
      id: "T-2024-001",
      title: "网络连接异常",
      priority: "high",
      status: "processing",
      assignee: "张三",
      time: "10分钟前",
    },
    {
      id: "T-2024-002",
      title: "软件安装失败",
      priority: "medium",
      status: "pending",
      assignee: "李四",
      time: "30分钟前",
    },
    {
      id: "T-2024-003",
      title: "权限申请",
      priority: "low",
      status: "resolved",
      assignee: "王五",
      time: "1小时前",
    },
  ]);

  const [recentActivities] = useState([
    {
      operator: "张三",
      action: "处理了工单",
      target: "T-2024-001",
      time: "10分钟前",
    },
    {
      operator: "系统",
      action: "自动分配工单",
      target: "T-2024-002",
      time: "30分钟前",
    },
    {
      operator: "李四",
      action: "更新了配置",
      target: "数据库配置",
      time: "1小时前",
    },
  ]);

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case "high":
        return "red";
      case "medium":
        return "orange";
      case "low":
        return "blue";
      default:
        return "default";
    }
  };

  return (
    <>
      {/* 系统状态 */}
      {systemAlerts.length > 0 && (
        <Alert
          message="系统状态"
          description="当前系统运行状态良好，但有需要注意的事项"
          type="info"
          showIcon
          className="mb-6"
        />
      )}

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总工单数"
              value={1128}
              prefix={<FileText size={20} className="text-blue-500" />}
              valueStyle={{ color: "#3f8600" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="待处理事件"
              value={93}
              prefix={<AlertTriangle size={20} className="text-red-500" />}
              valueStyle={{ color: "#cf1322" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="活跃用户"
              value={256}
              prefix={<Users size={20} className="text-green-500" />}
              valueStyle={{ color: "#1890ff" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="平均响应时间"
              value="2.5"
              suffix="小时"
              prefix={<Clock size={20} className="text-purple-500" />}
              valueStyle={{ color: "#722ed1" }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        {/* 最近工单 */}
        <Col xs={24} lg={12}>
          <Card title="最近工单" className="h-full">
            <List
              dataSource={recentTickets}
              renderItem={(item) => (
                <List.Item>
                  <div className="flex items-center justify-between w-full">
                    <div className="flex items-center space-x-3">
                      <div className="flex flex-col">
                        <span className="font-medium text-gray-900">
                          {item.title}
                        </span>
                        <span className="text-sm text-gray-500">
                          {item.id} • {item.assignee}
                        </span>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <Tag color={getPriorityColor(item.priority)}>
                        {item.priority}
                      </Tag>
                      <span className="text-xs text-gray-400">{item.time}</span>
                    </div>
                  </div>
                </List.Item>
              )}
            />
          </Card>
        </Col>

        {/* 最近活动 */}
        <Col xs={24} lg={12}>
          <Card title="最近活动" className="h-full">
            <Timeline
              items={recentActivities.map((activity) => ({
                children: (
                  <div className="flex items-center space-x-2">
                    <span className="font-medium text-gray-900">
                      {activity.operator}
                    </span>
                    <span className="text-gray-600">{activity.action}</span>
                    <span className="text-blue-600">{activity.target}</span>
                    <span className="text-xs text-gray-400">
                      {activity.time}
                    </span>
                  </div>
                ),
              }))}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} className="mt-6">
        {/* 系统性能 */}
        <Col xs={24} lg={12}>
          <Card title="系统性能">
            <div className="space-y-4">
              <div>
                <div className="flex justify-between items-center mb-2">
                  <span className="text-sm font-medium">CPU 使用率</span>
                  <span className="text-sm text-gray-500">65%</span>
                </div>
                <Progress percent={65} strokeColor="#52c41a" />
              </div>
              <div>
                <div className="flex justify-between items-center mb-2">
                  <span className="text-sm font-medium">内存使用率</span>
                  <span className="text-sm text-gray-500">78%</span>
                </div>
                <Progress percent={78} strokeColor="#1890ff" />
              </div>
              <div>
                <div className="flex justify-between items-center mb-2">
                  <span className="text-sm font-medium">磁盘使用率</span>
                  <span className="text-sm text-gray-500">45%</span>
                </div>
                <Progress percent={45} strokeColor="#722ed1" />
              </div>
            </div>
          </Card>
        </Col>

        {/* SLA监控 */}
        <Col xs={24} lg={12}>
          <Card title="SLA监控">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <CheckCircle size={16} className="text-green-500" />
                  <span className="text-sm">响应时间 SLA</span>
                </div>
                <span className="text-sm font-medium text-green-600">
                  98.5%
                </span>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <Activity size={16} className="text-blue-500" />
                  <span className="text-sm">解决时间 SLA</span>
                </div>
                <span className="text-sm font-medium text-blue-600">95.2%</span>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <BarChart3 size={16} className="text-purple-500" />
                  <span className="text-sm">可用性 SLA</span>
                </div>
                <span className="text-sm font-medium text-purple-600">
                  99.9%
                </span>
              </div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* AI 使用指标 */}
      <Row gutter={[16, 16]} className="mt-6">
        <Col xs={24}>
          <AIMetrics />
        </Col>
      </Row>

      {/* 快速操作 */}
      <div className="mt-6 p-4 bg-gray-50 rounded-lg">
        <h3 className="text-lg font-medium mb-4">快速操作</h3>
        <div className="flex flex-wrap gap-3">
          <Button type="primary" icon={<FileText size={16} />}>
            创建工单
          </Button>
          <Button icon={<AlertTriangle size={16} />}>报告事件</Button>
          <Button icon={<Settings size={16} />}>系统设置</Button>
          <Button icon={<BarChart3 size={16} />}>查看报表</Button>
        </div>
      </div>
    </>
  );
}
