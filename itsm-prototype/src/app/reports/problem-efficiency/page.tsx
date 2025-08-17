"use client";

import React, { useState, useEffect } from "react";
import { Card, Statistic, Row, Col, List, Tag, Typography } from "antd";
import { AlertTriangle, CheckCircle, Clock, XCircle } from "lucide-react";
import { mockProblemsData } from "../../lib/mock-data";

const { Title, Text } = Typography;

const ProblemEfficiencyPage = () => {
  const [isClient, setIsClient] = useState(false);

  useEffect(() => {
    setIsClient(true);
  }, []);

  // Aggregate data for charts
  const problemsByStatus = mockProblemsData.reduce(
    (acc: Record<string, number>, problem) => {
      acc[problem.status] = (acc[problem.status] || 0) + 1;
      return acc;
    },
    {}
  );

  const problemsByPriority = mockProblemsData.reduce(
    (acc: Record<string, number>, problem) => {
      acc[problem.priority] = (acc[problem.priority] || 0) + 1;
      return acc;
    },
    {}
  );

  const totalProblems = mockProblemsData.length;
  const resolvedProblems = problemsByStatus["已解决"] || 0;
  const inProgressProblems = problemsByStatus["调查中"] || 0;
  const highPriorityProblems = problemsByPriority["高"] || 0;

  const getStatusColor = (status: string) => {
    switch (status) {
      case "已解决":
        return "success";
      case "调查中":
        return "processing";
      case "已知错误":
        return "warning";
      case "已关闭":
        return "default";
      default:
        return "default";
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case "高":
        return "error";
      case "中":
        return "warning";
      case "低":
        return "default";
      default:
        return "default";
    }
  };

  if (!isClient) {
    return (
      <div className="p-10 bg-gray-50 min-h-full flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">页面加载中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8">
        <Title level={2} className="mb-2">
          问题管理效率报告
        </Title>
        <Text type="secondary">此报告显示问题管理流程的效率指标</Text>
      </header>

      {/* 统计卡片 */}
      <Row gutter={16} className="mb-8">
        <Col span={6}>
          <Card>
            <Statistic
              title="总问题数"
              value={totalProblems}
              prefix={<AlertTriangle className="text-blue-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已解决问题"
              value={resolvedProblems}
              valueStyle={{ color: "#3f8600" }}
              prefix={<CheckCircle className="text-green-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="处理中问题"
              value={inProgressProblems}
              valueStyle={{ color: "#1890ff" }}
              prefix={<Clock className="text-blue-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="高优先级问题"
              value={highPriorityProblems}
              valueStyle={{ color: "#cf1322" }}
              prefix={<XCircle className="text-red-500" />}
            />
          </Card>
        </Col>
      </Row>

      {/* 问题状态分布 */}
      <Row gutter={16} className="mb-8">
        <Col span={12}>
          <Card title="问题状态分布" className="h-80">
            <div className="space-y-4">
              {Object.entries(problemsByStatus).map(([status, count]) => (
                <div key={status} className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Tag color={getStatusColor(status)}>{status}</Tag>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-lg font-semibold">{count}</span>
                    <span className="text-sm text-gray-500">
                      ({((count / totalProblems) * 100).toFixed(1)}%)
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </Col>
        <Col span={12}>
          <Card title="问题优先级分布" className="h-80">
            <div className="space-y-4">
              {Object.entries(problemsByPriority).map(([priority, count]) => (
                <div
                  key={priority}
                  className="flex items-center justify-between"
                >
                  <div className="flex items-center gap-2">
                    <Tag color={getPriorityColor(priority)}>{priority}</Tag>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-lg font-semibold">{count}</span>
                    <span className="text-sm text-gray-500">
                      ({((count / totalProblems) * 100).toFixed(1)}%)
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>

      {/* 问题列表 */}
      <Card title="问题详情列表" className="mb-8">
        <List
          dataSource={mockProblemsData}
          renderItem={(problem) => (
            <List.Item>
              <div className="w-full">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-3">
                    <span className="font-medium text-blue-600">
                      {problem.id}
                    </span>
                    <span className="font-medium">{problem.title}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Tag color={getStatusColor(problem.status)}>
                      {problem.status}
                    </Tag>
                    <Tag color={getPriorityColor(problem.priority)}>
                      {problem.priority}
                    </Tag>
                  </div>
                </div>
                <div className="flex items-center justify-between text-sm text-gray-500">
                  <span>处理人: {problem.assignee}</span>
                  <span>创建时间: {problem.createdAt}</span>
                </div>
              </div>
            </List.Item>
          )}
        />
      </Card>

      {/* 效率指标 */}
      <Card title="效率指标" className="mb-8">
        <Row gutter={16}>
          <Col span={8}>
            <div className="text-center">
              <div className="text-3xl font-bold text-green-600">
                {totalProblems > 0
                  ? ((resolvedProblems / totalProblems) * 100).toFixed(1)
                  : 0}
                %
              </div>
              <div className="text-gray-500">问题解决率</div>
            </div>
          </Col>
          <Col span={8}>
            <div className="text-center">
              <div className="text-3xl font-bold text-blue-600">
                {totalProblems > 0
                  ? ((inProgressProblems / totalProblems) * 100).toFixed(1)
                  : 0}
                %
              </div>
              <div className="text-gray-500">处理中比例</div>
            </div>
          </Col>
          <Col span={8}>
            <div className="text-center">
              <div className="text-3xl font-bold text-red-600">
                {totalProblems > 0
                  ? ((highPriorityProblems / totalProblems) * 100).toFixed(1)
                  : 0}
                %
              </div>
              <div className="text-gray-500">高优先级比例</div>
            </div>
          </Col>
        </Row>
      </Card>
    </div>
  );
};

export default ProblemEfficiencyPage;
