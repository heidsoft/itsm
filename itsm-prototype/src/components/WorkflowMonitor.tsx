"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Table,
  Tag,
  Progress,
  Statistic,
  Row,
  Col,
  Alert,
  Button,
  Space,
} from "antd";
import {
  PlayCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  WarningOutlined,
  ReloadOutlined,
} from "@ant-design/icons";

interface WorkflowMetrics {
  totalInstances: number;
  activeInstances: number;
  completedInstances: number;
  failedInstances: number;
  averageDuration: number;
  successRate: number;
  throughput: number;
}

interface StepMetrics {
  stepId: string;
  stepName: string;
  executions: number;
  successCount: number;
  failureCount: number;
  averageDuration: number;
  bottleneckScore: number;
}

interface WorkflowInstance {
  id: string;
  workflowName: string;
  status: "running" | "completed" | "failed" | "paused";
  currentStep: string;
  progress: number;
  startTime: string;
  estimatedEndTime?: string;
  assignee?: string;
  priority: "low" | "medium" | "high" | "critical";
}

interface Alert {
  id: string;
  type: "warning" | "error" | "info";
  message: string;
  timestamp: string;
  acknowledged: boolean;
  resolved: boolean;
}

const WorkflowMonitor: React.FC = () => {
  const [metrics, setMetrics] = useState<WorkflowMetrics>({
    totalInstances: 0,
    activeInstances: 0,
    completedInstances: 0,
    failedInstances: 0,
    averageDuration: 0,
    successRate: 0,
    throughput: 0,
  });

  const [stepMetrics, setStepMetrics] = useState<StepMetrics[]>([]);
  const [instances, setInstances] = useState<WorkflowInstance[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(false);

  // 模拟数据加载
  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      // TODO: 调用API获取实际数据
      // 模拟数据
      setMetrics({
        totalInstances: 156,
        activeInstances: 23,
        completedInstances: 128,
        failedInstances: 5,
        averageDuration: 2.5,
        successRate: 96.2,
        throughput: 12.8,
      });

      setStepMetrics([
        {
          stepId: "step_1",
          stepName: "需求确认",
          executions: 156,
          successCount: 150,
          failureCount: 6,
          averageDuration: 1.2,
          bottleneckScore: 0.3,
        },
        {
          stepId: "step_2",
          stepName: "技术评审",
          executions: 150,
          successCount: 145,
          failureCount: 5,
          averageDuration: 3.8,
          bottleneckScore: 0.8,
        },
        {
          stepId: "step_3",
          stepName: "实施部署",
          executions: 145,
          successCount: 140,
          failureCount: 5,
          averageDuration: 4.2,
          bottleneckScore: 0.9,
        },
      ]);

      setInstances([
        {
          id: "inst_001",
          workflowName: "变更审批流程",
          status: "running",
          currentStep: "技术评审",
          progress: 65,
          startTime: "2024-01-15 09:00:00",
          estimatedEndTime: "2024-01-17 17:00:00",
          assignee: "张三",
          priority: "high",
        },
        {
          id: "inst_002",
          workflowName: "问题处理流程",
          status: "completed",
          currentStep: "问题解决",
          progress: 100,
          startTime: "2024-01-14 14:30:00",
          assignee: "李四",
          priority: "medium",
        },
      ]);

      setAlerts([
        {
          id: "alert_001",
          type: "warning",
          message: '工作流"变更审批流程"执行时间超过预期',
          timestamp: "2024-01-15 16:30:00",
          acknowledged: false,
          resolved: false,
        },
        {
          id: "alert_002",
          type: "error",
          message: '步骤"技术评审"失败率异常升高',
          timestamp: "2024-01-15 15:45:00",
          acknowledged: true,
          resolved: false,
        },
      ]);
    } catch (error) {
      console.error("加载数据失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "running":
        return "processing";
      case "completed":
        return "success";
      case "failed":
        return "error";
      case "paused":
        return "warning";
      default:
        return "default";
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case "low":
        return "blue";
      case "medium":
        return "green";
      case "high":
        return "orange";
      case "critical":
        return "red";
      default:
        return "default";
    }
  };

  const getBottleneckColor = (score: number) => {
    if (score >= 0.8) return "red";
    if (score >= 0.6) return "orange";
    if (score >= 0.4) return "yellow";
    return "green";
  };

  const instanceColumns = [
    {
      title: "工作流名称",
      dataIndex: "workflowName",
      key: "workflowName",
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>
          {status === "running" && <PlayCircleOutlined />}
          {status === "completed" && <CheckCircleOutlined />}
          {status === "failed" && <CloseCircleOutlined />}
          {status === "paused" && <ClockCircleOutlined />}
          {status}
        </Tag>
      ),
    },
    {
      title: "当前步骤",
      dataIndex: "currentStep",
      key: "currentStep",
    },
    {
      title: "进度",
      dataIndex: "progress",
      key: "progress",
      render: (progress: number) => (
        <Progress percent={progress} size="small" />
      ),
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      render: (priority: string) => (
        <Tag color={getPriorityColor(priority)}>{priority}</Tag>
      ),
    },
    {
      title: "负责人",
      dataIndex: "assignee",
      key: "assignee",
    },
    {
      title: "开始时间",
      dataIndex: "startTime",
      key: "startTime",
    },
  ];

  const stepColumns = [
    {
      title: "步骤名称",
      dataIndex: "stepName",
      key: "stepName",
    },
    {
      title: "执行次数",
      dataIndex: "executions",
      key: "executions",
    },
    {
      title: "成功率",
      key: "successRate",
      render: (record: StepMetrics) => {
        const rate = ((record.successCount / record.executions) * 100).toFixed(
          1
        );
        return `${rate}%`;
      },
    },
    {
      title: "平均耗时(小时)",
      dataIndex: "averageDuration",
      key: "averageDuration",
    },
    {
      title: "瓶颈指数",
      dataIndex: "bottleneckScore",
      key: "bottleneckScore",
      render: (score: number) => (
        <Tag color={getBottleneckColor(score)}>{(score * 100).toFixed(0)}%</Tag>
      ),
    },
  ];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">工作流监控仪表板</h1>
        <Button icon={<ReloadOutlined />} onClick={loadData} loading={loading}>
          刷新数据
        </Button>
      </div>

      {/* 关键指标 */}
      <Row gutter={16} className="mb-6">
        <Col span={6}>
          <Card>
            <Statistic
              title="总实例数"
              value={metrics.totalInstances}
              prefix={<PlayCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="活跃实例"
              value={metrics.activeInstances}
              valueStyle={{ color: "#1890ff" }}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="成功率"
              value={metrics.successRate}
              suffix="%"
              valueStyle={{ color: "#52c41a" }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="平均耗时"
              value={metrics.averageDuration}
              suffix="小时"
              valueStyle={{ color: "#fa8c16" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 告警信息 */}
      {alerts.length > 0 && (
        <Card title="告警信息" className="mb-6">
          <Space direction="vertical" className="w-full">
            {alerts.map((alert) => (
              <Alert
                key={alert.id}
                message={alert.message}
                type={alert.type}
                showIcon
                icon={
                  alert.type === "warning" ? <WarningOutlined /> : undefined
                }
                action={
                  <Space>
                    {!alert.acknowledged && (
                      <Button size="small" type="link">
                        确认
                      </Button>
                    )}
                    {!alert.resolved && (
                      <Button size="small" type="link">
                        解决
                      </Button>
                    )}
                  </Space>
                }
              />
            ))}
          </Space>
        </Card>
      )}

      <Row gutter={16}>
        {/* 工作流实例 */}
        <Col span={16}>
          <Card title="工作流实例" className="mb-6">
            <Table
              dataSource={instances}
              columns={instanceColumns}
              rowKey="id"
              pagination={false}
              size="small"
            />
          </Card>
        </Col>

        {/* 步骤性能 */}
        <Col span={8}>
          <Card title="步骤性能分析" className="mb-6">
            <Table
              dataSource={stepMetrics}
              columns={stepColumns}
              rowKey="stepId"
              pagination={false}
              size="small"
            />
          </Card>
        </Col>
      </Row>

      {/* 性能趋势图 */}
      <Card title="性能趋势" className="mb-6">
        <div className="text-center text-gray-500 py-8">
          {/* TODO: 集成图表库显示性能趋势 */}
          性能趋势图表将在这里显示
        </div>
      </Card>
    </div>
  );
};

export default WorkflowMonitor;
