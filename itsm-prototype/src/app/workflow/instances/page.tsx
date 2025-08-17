"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Tag,
  Space,
  Tooltip,
  Modal,
  Form,
  Badge,
  Row,
  Col,
  Statistic,
  Timeline,
  Descriptions,
  Progress,
  Alert,
  App,
} from "antd";
import {
  Search,
  Filter,
  PlayCircle,
  PauseCircle,
  StopCircle,
  Eye,
  Clock,
  User,
  Calendar,
  CheckCircle,
  AlertCircle,
  XCircle,
  RefreshCw,
  BarChart3,
  GitBranch,
  Activity,
  History,
} from "lucide-react";
// AppLayout is handled by parent layout
import {
  WorkflowAPI,
  WorkflowInstance,
  WorkflowTask,
} from "../../lib/workflow-api";

const { Option } = Select;

const WorkflowInstancesPage = () => {
  const { message } = App.useApp();
  const [instances, setInstances] = useState<WorkflowInstance[]>([]);
  const [tasks, setTasks] = useState<WorkflowTask[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedInstance, setSelectedInstance] =
    useState<WorkflowInstance | null>(null);
  const [filters, setFilters] = useState({
    status: "",
    workflow_id: "",
    business_key: "",
    started_by: "",
  });
  const [stats, setStats] = useState({
    total: 0,
    running: 0,
    completed: 0,
    suspended: 0,
    terminated: 0,
  });

  useEffect(() => {
    loadInstances();
    loadStats();
  }, [filters]);

  const loadInstances = async () => {
    setLoading(true);
    try {
      const response = await WorkflowAPI.listWorkflowInstances({
        page: 1,
        page_size: 50,
        ...filters,
      });
      setInstances(response.instances);
    } catch (error) {
      message.error("加载工作流实例失败");
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      // 模拟统计数据
      setStats({
        total: instances.length,
        running: instances.filter((i) => i.status === "running").length,
        completed: instances.filter((i) => i.status === "completed").length,
        suspended: instances.filter((i) => i.status === "suspended").length,
        terminated: instances.filter((i) => i.status === "terminated").length,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  };

  const handleViewDetail = async (instance: WorkflowInstance) => {
    setSelectedInstance(instance);
    setDetailVisible(true);

    try {
      const tasksResponse = await WorkflowAPI.listWorkflowTasks({
        instance_id: instance.id,
      });
      setTasks(tasksResponse.tasks);
    } catch (error) {
      message.error("加载任务列表失败");
    }
  };

  const handleSuspendInstance = async (instanceId: string) => {
    try {
      await WorkflowAPI.suspendWorkflow(instanceId, "手动暂停");
      message.success("实例暂停成功");
      loadInstances();
    } catch (error) {
      message.error("暂停失败");
    }
  };

  const handleResumeInstance = async (instanceId: string) => {
    try {
      await WorkflowAPI.resumeWorkflow(instanceId, "手动恢复");
      message.success("实例恢复成功");
      loadInstances();
    } catch (error) {
      message.error("恢复失败");
    }
  };

  const handleTerminateInstance = async (instanceId: string) => {
    try {
      await WorkflowAPI.terminateWorkflow(instanceId, "手动终止");
      message.success("实例终止成功");
      loadInstances();
    } catch (error) {
      message.error("终止失败");
    }
  };

  const getStatusColor = (status: string) => {
    const colors = {
      running: "green",
      completed: "blue",
      suspended: "orange",
      terminated: "red",
    };
    return colors[status as keyof typeof colors] || "default";
  };

  const getStatusText = (status: string) => {
    const texts = {
      running: "运行中",
      completed: "已完成",
      suspended: "已暂停",
      terminated: "已终止",
    };
    return texts[status as keyof typeof texts] || status;
  };

  const getPriorityColor = (priority: string) => {
    const colors = {
      low: "green",
      normal: "blue",
      high: "orange",
      critical: "red",
    };
    return colors[priority as keyof typeof colors] || "default";
  };

  const columns = [
    {
      title: "实例ID",
      dataIndex: "instance_id",
      key: "instance_id",
      width: 150,
      render: (instanceId: string) => (
        <span className="font-mono text-sm">{instanceId}</span>
      ),
    },
    {
      title: "业务键",
      dataIndex: "business_key",
      key: "business_key",
      width: 150,
      render: (businessKey: string) => (
        <span className="text-sm">{businessKey || "-"}</span>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>
      ),
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 100,
      render: (priority: string) => (
        <Tag color={getPriorityColor(priority)}>{priority}</Tag>
      ),
    },
    {
      title: "启动人",
      dataIndex: "started_by",
      key: "started_by",
      width: 120,
      render: (startedBy: string) => (
        <div className="flex items-center">
          <User className="w-4 h-4 mr-1" />
          <span>{startedBy}</span>
        </div>
      ),
    },
    {
      title: "启动时间",
      dataIndex: "started_at",
      key: "started_at",
      width: 150,
      render: (date: string) => (
        <div className="text-sm">{new Date(date).toLocaleString("zh-CN")}</div>
      ),
    },
    {
      title: "到期时间",
      dataIndex: "due_date",
      key: "due_date",
      width: 150,
      render: (date: string) => (
        <div className="text-sm">
          {date ? new Date(date).toLocaleString("zh-CN") : "-"}
        </div>
      ),
    },
    {
      title: "操作",
      key: "action",
      width: 200,
      render: (record: WorkflowInstance) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<Eye className="w-4 h-4" />}
              onClick={() => handleViewDetail(record)}
            />
          </Tooltip>
          {record.status === "running" && (
            <>
              <Tooltip title="暂停">
                <Button
                  type="text"
                  icon={<PauseCircle className="w-4 h-4" />}
                  onClick={() => handleSuspendInstance(record.instance_id)}
                />
              </Tooltip>
              <Tooltip title="终止">
                <Button
                  type="text"
                  danger
                  icon={<StopCircle className="w-4 h-4" />}
                  onClick={() => handleTerminateInstance(record.instance_id)}
                />
              </Tooltip>
            </>
          )}
          {record.status === "suspended" && (
            <Tooltip title="恢复">
              <Button
                type="text"
                icon={<PlayCircle className="w-4 h-4" />}
                onClick={() => handleResumeInstance(record.instance_id)}
              />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  const taskColumns = [
    {
      title: "任务名称",
      dataIndex: "name",
      key: "name",
      render: (name: string, record: WorkflowTask) => (
        <div>
          <div className="font-medium">{name}</div>
          <div className="text-sm text-gray-500">{record.activity_id}</div>
        </div>
      ),
    },
    {
      title: "类型",
      dataIndex: "type",
      key: "type",
      width: 120,
      render: (type: string) => <Tag color="blue">{type}</Tag>,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>
      ),
    },
    {
      title: "处理人",
      dataIndex: "assignee",
      key: "assignee",
      width: 120,
      render: (assignee: string) => <span>{assignee || "-"}</span>,
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 150,
      render: (date: string) => (
        <div className="text-sm">{new Date(date).toLocaleString("zh-CN")}</div>
      ),
    },
    {
      title: "到期时间",
      dataIndex: "due_date",
      key: "due_date",
      width: 150,
      render: (date: string) => (
        <div className="text-sm">
          {date ? new Date(date).toLocaleString("zh-CN") : "-"}
        </div>
      ),
    },
  ];

  return (
    <>
      {/* 页面头部 */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">工作流实例</h1>
        <p className="text-gray-600 mt-1">管理工作流实例的执行状态和生命周期</p>
      </div>
      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="总实例"
              value={stats.total}
              prefix={<Activity className="w-5 h-5" />}
              valueStyle={{ color: "#1890ff" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="运行中"
              value={stats.running}
              prefix={<PlayCircle className="w-5 h-5" />}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="已完成"
              value={stats.completed}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: "#1890ff" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="已暂停"
              value={stats.suspended}
              prefix={<PauseCircle className="w-5 h-5" />}
              valueStyle={{ color: "#faad14" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 工具栏 */}
      <Card className="enterprise-card mb-6">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} md={6}>
            <Input
              placeholder="搜索实例ID或业务键..."
              prefix={<Search className="w-4 h-4" />}
              value={filters.business_key}
              onChange={(e) =>
                setFilters((prev) => ({
                  ...prev,
                  business_key: e.target.value,
                }))
              }
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="状态筛选"
              value={filters.status}
              onChange={(value) =>
                setFilters((prev) => ({ ...prev, status: value }))
              }
              allowClear
              style={{ width: "100%" }}
            >
              <Option value="running">运行中</Option>
              <Option value="completed">已完成</Option>
              <Option value="suspended">已暂停</Option>
              <Option value="terminated">已终止</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="启动人筛选"
              value={filters.started_by}
              onChange={(value) =>
                setFilters((prev) => ({ ...prev, started_by: value }))
              }
              allowClear
              style={{ width: "100%" }}
            >
              <Option value="admin">管理员</Option>
              <Option value="user1">用户1</Option>
              <Option value="user2">用户2</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={10}>
            <Space>
              <Button
                icon={<RefreshCw className="w-4 h-4" />}
                onClick={loadInstances}
              >
                刷新
              </Button>
              <Button icon={<BarChart3 className="w-4 h-4" />}>统计报告</Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 实例表格 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={instances}
          rowKey="id"
          loading={loading}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          scroll={{ x: 1200 }}
        />
      </Card>

      {/* 实例详情模态框 */}
      <Modal
        title={`实例详情 - ${selectedInstance?.instance_id}`}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={800}
        destroyOnHidden
      >
        {selectedInstance && (
          <div className="space-y-6">
            {/* 基本信息 */}
            <Card title="基本信息" size="small">
              <Descriptions column={2}>
                <Descriptions.Item label="实例ID">
                  {selectedInstance.instance_id}
                </Descriptions.Item>
                <Descriptions.Item label="业务键">
                  {selectedInstance.business_key || "-"}
                </Descriptions.Item>
                <Descriptions.Item label="状态">
                  <Tag color={getStatusColor(selectedInstance.status)}>
                    {getStatusText(selectedInstance.status)}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="优先级">
                  <Tag color={getPriorityColor(selectedInstance.priority)}>
                    {selectedInstance.priority}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="启动人">
                  {selectedInstance.started_by}
                </Descriptions.Item>
                <Descriptions.Item label="启动时间">
                  {new Date(selectedInstance.started_at).toLocaleString(
                    "zh-CN"
                  )}
                </Descriptions.Item>
                {selectedInstance.completed_at && (
                  <Descriptions.Item label="完成时间">
                    {new Date(selectedInstance.completed_at).toLocaleString(
                      "zh-CN"
                    )}
                  </Descriptions.Item>
                )}
                {selectedInstance.due_date && (
                  <Descriptions.Item label="到期时间">
                    {new Date(selectedInstance.due_date).toLocaleString(
                      "zh-CN"
                    )}
                  </Descriptions.Item>
                )}
              </Descriptions>
            </Card>

            {/* 任务列表 */}
            <Card title="任务列表" size="small">
              <Table
                columns={taskColumns}
                dataSource={tasks}
                rowKey="id"
                pagination={false}
                size="small"
              />
            </Card>

            {/* 流程变量 */}
            {selectedInstance.variables &&
              Object.keys(selectedInstance.variables).length > 0 && (
                <Card title="流程变量" size="small">
                  <pre className="bg-gray-100 p-4 rounded text-sm overflow-auto">
                    {JSON.stringify(selectedInstance.variables, null, 2)}
                  </pre>
                </Card>
              )}
          </div>
        )}
      </Modal>
    </>
  );
};

export default WorkflowInstancesPage;
