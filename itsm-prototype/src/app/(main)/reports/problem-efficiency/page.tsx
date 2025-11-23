"use client";

import React, { useState, useEffect } from "react";
import { Card, Statistic, Row, Col, List, Tag, Typography } from "antd";
import { AlertTriangle, CheckCircle, Clock, XCircle } from "lucide-react";
import { mockProblemsData } from "@/app/lib/mock-data";

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
  const resolvedProblems = problemsByStatus["Resolved"] || 0;
  const inProgressProblems = problemsByStatus["Investigating"] || 0;
  const highPriorityProblems = problemsByPriority["High"] || 0;

  const getStatusColor = (status: string) => {
    switch (status) {
      case "Resolved":
        return "success";
      case "Investigating":
        return "processing";
      case "Known Error":
        return "warning";
      case "Closed":
        return "default";
      default:
        return "default";
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case "High":
        return "error";
      case "Medium":
        return "warning";
      case "Low":
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
          <p className="mt-4 text-gray-600">Page loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8">
        <Title level={2} className="mb-2">
          Problem Management Efficiency Report
        </Title>
        <Text type="secondary">This report shows efficiency metrics for the problem management process</Text>
      </header>

      {/* Statistics Cards */}
      <Row gutter={16} className="mb-8">
        <Col span={6}>
          <Card>
            <Statistic
              title="Total Problems"
              value={totalProblems}
              prefix={<AlertTriangle className="text-blue-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Resolved Problems"
              value={resolvedProblems}
              valueStyle={{ color: "#3f8600" }}
              prefix={<CheckCircle className="text-green-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Problems in Progress"
              value={inProgressProblems}
              valueStyle={{ color: "#1890ff" }}
              prefix={<Clock className="text-blue-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="High Priority Problems"
              value={highPriorityProblems}
              valueStyle={{ color: "#cf1322" }}
              prefix={<XCircle className="text-red-500" />}
            />
          </Card>
        </Col>
      </Row>

      {/* Problem Status Distribution */}
      <Row gutter={16} className="mb-8">
        <Col span={12}>
          <Card title="Problem Status Distribution" className="h-80">
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
          <Card title="Problem Priority Distribution" className="h-80">
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

      {/* Problem List */}
      <Card title="Problem Details List" className="mb-8">
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
                  <span>Assignee: {problem.assignee}</span>
                  <span>Created: {problem.createdAt}</span>
                </div>
              </div>
            </List.Item>
          )}
        />
      </Card>

      {/* Efficiency Metrics */}
      <Card title="Efficiency Metrics" className="mb-8">
        <Row gutter={16}>
          <Col span={8}>
            <div className="text-center">
              <div className="text-3xl font-bold text-green-600">
                {totalProblems > 0
                  ? ((resolvedProblems / totalProblems) * 100).toFixed(1)
                  : 0}
                %
              </div>
              <div className="text-gray-500">Problem Resolution Rate</div>
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
              <div className="text-gray-500">In Progress Ratio</div>
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
              <div className="text-gray-500">High Priority Ratio</div>
            </div>
          </Col>
        </Row>
      </Card>
    </div>
  );
};

export default ProblemEfficiencyPage;
