"use client";

import React, { useState, useEffect, useCallback, useMemo } from "react";
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Modal,
  Form,
  message,
  Tag,
  Space,
  Dropdown,
  Tooltip,
  Row,
  Col,
  Statistic,
  Alert,
  App,
} from "antd";
import {
  Plus,
  Search,
  MoreHorizontal,
  Edit,
  Eye,
  Trash2,
  PlayCircle,
  PauseCircle,
  Copy,
  Download,
  Upload,
  BarChart3,
  RefreshCw,
  FileText,
  GitBranch,
  Clock,
  CheckCircle,
} from "lucide-react";

import { WorkflowDesigner } from "../components/WorkflowDesigner";

import { useRouter } from "next/navigation";

const { Option } = Select;

interface Workflow {
  id: number;
  name: string;
  description: string;
  category: string;
  version: string;
  status: "draft" | "active" | "inactive" | "archived";
  created_at: string;
  updated_at: string;
  instances_count: number;
  running_instances: number;
  created_by: string;
}

const WorkflowManagementPage = () => {
  const router = useRouter();
  const { modal } = App.useApp();
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [designerVisible, setDesignerVisible] = useState(false);

  const [editingWorkflow, setEditingWorkflow] = useState<Workflow | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [filters, setFilters] = useState({
    status: "",
    category: "",
    keyword: "",
  });
  const [stats, setStats] = useState({
    total: 0,
    active: 0,
    running: 0,
    completed: 0,
    draft: 0,
    inactive: 0,
    todayInstances: 0,
    avgExecutionTime: 0,
  });

  // 模拟数据 - 使用useMemo避免每次渲染时重新创建
  const mockWorkflows = useMemo<Workflow[]>(
    () => [
      {
        id: 1,
        name: "工单审批流程",
        description: "标准工单审批流程，包含提交、审核、处理、验收等环节",
        category: "ticket_approval",
        version: "1.0.0",
        status: "active",
        created_at: "2024-01-15T10:30:00Z",
        updated_at: "2024-01-15T14:20:00Z",
        instances_count: 156,
        running_instances: 23,
        created_by: "张三",
      },
      {
        id: 2,
        name: "事件处理流程",
        description: "IT事件处理标准流程，支持自动分配和升级",
        category: "事件处理",
        version: "2.1.0",
        status: "active",
        created_at: "2024-01-14T09:15:00Z",
        updated_at: "2024-01-15T11:45:00Z",
        instances_count: 89,
        running_instances: 12,
        created_by: "李四",
      },
      {
        id: 3,
        name: "变更管理流程",
        description: "IT变更管理流程，包含变更申请、审批、实施、验证",
        category: "变更管理",
        version: "1.5.0",
        status: "draft",
        created_at: "2024-01-13T16:20:00Z",
        updated_at: "2024-01-14T10:30:00Z",
        instances_count: 45,
        running_instances: 8,
        created_by: "王五",
      },
    ],
    []
  );

  const loadWorkflows = useCallback(async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise((resolve) => setTimeout(resolve, 500));
      setWorkflows(mockWorkflows);
    } catch {
      message.error("加载工作流失败");
    } finally {
      setLoading(false);
    }
  }, [mockWorkflows]);

  const loadStats = useCallback(async () => {
    try {
      const currentWorkflows = workflows.length > 0 ? workflows : mockWorkflows;
      setStats({
        total: currentWorkflows.length,
        active: currentWorkflows.filter((w) => w.status === "active").length,
        draft: currentWorkflows.filter((w) => w.status === "draft").length,
        inactive: currentWorkflows.filter((w) => w.status === "inactive")
          .length,
        running: currentWorkflows.reduce(
          (sum, w) => sum + w.running_instances,
          0
        ),
        completed: currentWorkflows.reduce(
          (sum, w) => sum + (w.instances_count - w.running_instances),
          0
        ),
        todayInstances: Math.floor(Math.random() * 50) + 10, // 模拟今日实例
        avgExecutionTime: Math.floor(Math.random() * 120) + 30, // 模拟平均执行时间(分钟)
      });
    } catch {
      console.error("加载统计数据失败");
    }
  }, [workflows, mockWorkflows]);

  useEffect(() => {
    loadWorkflows();
    loadStats();
  }, [loadWorkflows, loadStats]);

  // 当filters变化时重新加载数据
  useEffect(() => {
    loadWorkflows();
  }, [filters, loadWorkflows]);

  const handleCreateWorkflow = () => {
    setEditingWorkflow(null);
    setModalVisible(true);
  };

  const handleEditWorkflow = (workflow: Workflow) => {
    setEditingWorkflow(workflow);
    setModalVisible(true);
  };

  const handleDesignWorkflow = (workflow: Workflow) => {
    setEditingWorkflow(workflow);

    // 根据工作流类型选择对应的设计器
    if (
      workflow.category === "ticket_approval" ||
      workflow.name.includes("审批")
    ) {
      // 工单审批流程使用专门的设计器
      router.push(`/workflow/ticket-approval?id=${workflow.id}`);
    } else {
      // 通用工作流使用BPMN设计器
      setDesignerVisible(true);
    }
  };

  const handleDeleteWorkflow = async (id: number) => {
    modal.confirm({
      title: "确认删除",
      content:
        "删除工作流将同时删除所有相关实例，此操作不可恢复。确定要删除吗？",
      okText: "确定删除",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        try {
          // 模拟删除API调用
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) => prev.filter((w) => w.id !== id));
          message.success("工作流删除成功");
          loadStats();
        } catch {
          message.error("删除失败");
        }
      },
    });
  };

  const handleDeployWorkflow = async (id: number) => {
    modal.confirm({
      title: "确认部署",
      content:
        "部署后工作流将变为激活状态，可以创建新的流程实例。确定要部署吗？",
      okText: "确定部署",
      cancelText: "取消",
      onOk: async () => {
        try {
          // 模拟部署API调用
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) =>
            prev.map((w) =>
              w.id === id ? { ...w, status: "active" as const } : w
            )
          );
          message.success("工作流部署成功");
          loadStats();
        } catch {
          message.error("部署失败");
        }
      },
    });
  };

  const handleCopyWorkflow = async (workflow: Workflow) => {
    try {
      // 模拟复制API调用
      await new Promise((resolve) => setTimeout(resolve, 500));
      const newWorkflow: Workflow = {
        ...workflow,
        id: Date.now(), // 临时ID
        name: `${workflow.name} - 副本`,
        status: "draft",
        version: "1.0.0",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        instances_count: 0,
        running_instances: 0,
        created_by: "当前用户",
      };
      setWorkflows((prev) => [newWorkflow, ...prev]);
      message.success(`工作流 "${workflow.name}" 复制成功`);
      loadStats();
    } catch {
      message.error("复制失败");
    }
  };

  const handleExportWorkflow = async (workflow: Workflow) => {
    try {
      // 模拟导出API调用
      await new Promise((resolve) => setTimeout(resolve, 500));

      const exportData = {
        workflow: workflow,
        exportTime: new Date().toISOString(),
        version: "1.0",
      };

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `workflow_${workflow.name}_${workflow.version}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      message.success(`工作流 "${workflow.name}" 导出成功`);
    } catch {
      message.error("导出失败");
    }
  };

  const handleImportWorkflow = () => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".json";
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file) return;

      try {
        const text = await file.text();
        const importData = JSON.parse(text);

        if (!importData.workflow) {
          throw new Error("无效的工作流文件");
        }

        const newWorkflow: Workflow = {
          ...importData.workflow,
          id: Date.now(),
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          status: "draft",
          instances_count: 0,
          running_instances: 0,
          created_by: "当前用户",
        };

        setWorkflows((prev) => [newWorkflow, ...prev]);
        message.success(`工作流 "${newWorkflow.name}" 导入成功`);
        loadStats();
      } catch (error) {
        message.error("导入失败：" + (error as Error).message);
      }
    };
    input.click();
  };

  const handleStopWorkflow = async (id: number) => {
    modal.confirm({
      title: "确认停用",
      content:
        "停用工作流后将无法创建新的流程实例，但已有实例会继续运行。确定要停用吗？",
      okText: "确定停用",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        try {
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) =>
            prev.map((w) =>
              w.id === id ? { ...w, status: "inactive" as const } : w
            )
          );
          message.success("工作流已停用");
          loadStats();
        } catch {
          message.error("停用失败");
        }
      },
    });
  };

  const handleActivateWorkflow = async (id: number) => {
    try {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      setWorkflows((prev) =>
        prev.map((w) => (w.id === id ? { ...w, status: "active" as const } : w))
      );
      message.success("工作流已激活");
      loadStats();
    } catch {
      message.error("激活失败");
    }
  };

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要删除的工作流");
      return;
    }

    modal.confirm({
      title: "批量删除确认",
      content: `确定要删除选中的 ${selectedRowKeys.length} 个工作流吗？此操作不可恢复。`,
      okText: "确定删除",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        try {
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) =>
            prev.filter((w) => !selectedRowKeys.includes(w.id))
          );
          setSelectedRowKeys([]);
          message.success(`成功删除 ${selectedRowKeys.length} 个工作流`);
          loadStats();
        } catch {
          message.error("批量删除失败");
        }
      },
    });
  };

  const handleBatchActivate = () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要激活的工作流");
      return;
    }

    modal.confirm({
      title: "批量激活确认",
      content: `确定要激活选中的 ${selectedRowKeys.length} 个工作流吗？`,
      okText: "确定激活",
      cancelText: "取消",
      onOk: async () => {
        try {
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) =>
            prev.map((w) =>
              selectedRowKeys.includes(w.id)
                ? { ...w, status: "active" as const }
                : w
            )
          );
          setSelectedRowKeys([]);
          message.success(`成功激活 ${selectedRowKeys.length} 个工作流`);
          loadStats();
        } catch {
          message.error("批量激活失败");
        }
      },
    });
  };

  const handleBatchStop = () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要停用的工作流");
      return;
    }

    modal.confirm({
      title: "批量停用确认",
      content: `确定要停用选中的 ${selectedRowKeys.length} 个工作流吗？`,
      okText: "确定停用",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        try {
          await new Promise((resolve) => setTimeout(resolve, 1000));
          setWorkflows((prev) =>
            prev.map((w) =>
              selectedRowKeys.includes(w.id)
                ? { ...w, status: "inactive" as const }
                : w
            )
          );
          setSelectedRowKeys([]);
          message.success(`成功停用 ${selectedRowKeys.length} 个工作流`);
          loadStats();
        } catch {
          message.error("批量停用失败");
        }
      },
    });
  };

  const handleBatchExport = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要导出的工作流");
      return;
    }

    try {
      const selectedWorkflows = workflows.filter((w) =>
        selectedRowKeys.includes(w.id)
      );
      const exportData = {
        workflows: selectedWorkflows,
        exportTime: new Date().toISOString(),
        version: "1.0",
        count: selectedWorkflows.length,
      };

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `workflows_batch_export_${
        new Date().toISOString().split("T")[0]
      }.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      message.success(`成功导出 ${selectedRowKeys.length} 个工作流`);
      setSelectedRowKeys([]);
    } catch {
      message.error("批量导出失败");
    }
  };

  const getStatusColor = (status: string) => {
    const colors = {
      draft: "orange",
      active: "green",
      inactive: "gray",
      archived: "red",
    };
    return colors[status as keyof typeof colors] || "default";
  };

  const getStatusText = (status: string) => {
    const texts = {
      draft: "草稿",
      active: "激活",
      inactive: "停用",
      archived: "归档",
    };
    return texts[status as keyof typeof texts] || status;
  };

  const columns = [
    {
      title: "工作流名称",
      dataIndex: "name",
      key: "name",
      render: (name: string, record: Workflow) => (
        <div>
          <div className="font-medium text-gray-900">{name}</div>
          <div className="text-sm text-gray-500">{record.description}</div>
        </div>
      ),
    },
    {
      title: "分类",
      dataIndex: "category",
      key: "category",
      width: 120,
      render: (category: string) => <Tag color="blue">{category}</Tag>,
    },
    {
      title: "版本",
      dataIndex: "version",
      key: "version",
      width: 100,
      render: (version: string) => (
        <span className="font-mono text-sm">{version}</span>
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
      title: "实例统计",
      key: "instances",
      width: 150,
      render: (record: Workflow) => (
        <div className="text-sm">
          <div>总数: {record.instances_count}</div>
          <div>运行中: {record.running_instances}</div>
        </div>
      ),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 150,
      render: (date: string) => (
        <div className="text-sm">
          {new Date(date).toLocaleDateString("zh-CN")}
        </div>
      ),
    },
    {
      title: "操作",
      key: "action",
      width: 200,
      render: (record: Workflow) => (
        <Space>
          <Tooltip title="设计工作流">
            <Button
              type="text"
              icon={<GitBranch className="w-4 h-4" />}
              onClick={() => handleDesignWorkflow(record)}
            />
          </Tooltip>
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<Eye className="w-4 h-4" />}
              onClick={() => window.open(`/workflow/${record.id}`, "_blank")}
            />
          </Tooltip>
          <Dropdown
            menu={{
              items: [
                {
                  key: "edit",
                  label: "编辑",
                  icon: <Edit className="w-4 h-4" />,
                  onClick: () => handleEditWorkflow(record),
                },
                {
                  key: "deploy",
                  label: record.status === "draft" ? "部署" : "重新部署",
                  icon: <PlayCircle className="w-4 h-4" />,
                  onClick: () => handleDeployWorkflow(record.id),
                  disabled: record.status === "active",
                },
                {
                  key: "activate",
                  label: "激活",
                  icon: <CheckCircle className="w-4 h-4" />,
                  onClick: () => handleActivateWorkflow(record.id),
                  style: {
                    display: record.status === "inactive" ? "block" : "none",
                  },
                },
                {
                  key: "stop",
                  label: "停用",
                  icon: <PauseCircle className="w-4 h-4" />,
                  onClick: () => handleStopWorkflow(record.id),
                  style: {
                    display: record.status === "active" ? "block" : "none",
                  },
                },
                {
                  type: "divider" as const,
                },
                {
                  key: "copy",
                  label: "复制",
                  icon: <Copy className="w-4 h-4" />,
                  onClick: () => handleCopyWorkflow(record),
                },
                {
                  key: "export",
                  label: "导出",
                  icon: <Download className="w-4 h-4" />,
                  onClick: () => handleExportWorkflow(record),
                },
                {
                  key: "instances",
                  label: "查看实例",
                  icon: <BarChart3 className="w-4 h-4" />,
                  onClick: () =>
                    window.open(
                      `/workflow/instances?workflowId=${record.id}`,
                      "_blank"
                    ),
                },
                {
                  key: "versions",
                  label: "版本管理",
                  icon: <GitBranch className="w-4 h-4" />,
                  onClick: () =>
                    window.open(
                      `/workflow/versions?workflowId=${record.id}`,
                      "_blank"
                    ),
                },
                {
                  type: "divider" as const,
                },
                {
                  key: "delete",
                  label: "删除",
                  icon: <Trash2 className="w-4 h-4" />,
                  danger: true,
                  onClick: () => handleDeleteWorkflow(record.id),
                  disabled:
                    record.status === "active" && record.running_instances > 0,
                },
              ].filter((item) => item.style?.display !== "none"), // 过滤隐藏项目
            }}
          >
            <Button type="text" icon={<MoreHorizontal className="w-4 h-4" />} />
          </Dropdown>
        </Space>
      ),
    },
  ];

  const filteredWorkflows = workflows.filter((workflow) => {
    if (filters.status && workflow.status !== filters.status) return false;
    if (filters.category && workflow.category !== filters.category)
      return false;
    if (
      filters.keyword &&
      !workflow.name.toLowerCase().includes(filters.keyword.toLowerCase())
    )
      return false;
    return true;
  });

  return (
    <>
      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="总工作流"
              value={stats.total}
              prefix={<FileText className="w-5 h-5" />}
              valueStyle={{ color: "#1890ff" }}
            />
            <div className="mt-2 text-xs text-gray-500">
              激活 {stats.active} | 草稿 {stats.draft} | 停用 {stats.inactive}
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="运行中实例"
              value={stats.running}
              prefix={<Clock className="w-5 h-5" />}
              valueStyle={{ color: "#faad14" }}
            />
            <div className="mt-2 text-xs text-gray-500">
              今日新增 {stats.todayInstances} 个实例
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="已完成实例"
              value={stats.completed}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: "#52c41a" }}
            />
            <div className="mt-2 text-xs text-gray-500">
              完成率{" "}
              {stats.total > 0
                ? Math.round(
                    (stats.completed / (stats.completed + stats.running)) * 100
                  )
                : 0}
              %
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="平均执行时间"
              value={stats.avgExecutionTime}
              suffix="分钟"
              prefix={<BarChart3 className="w-5 h-5" />}
              valueStyle={{ color: "#722ed1" }}
            />
            <div className="mt-2 text-xs text-gray-500">
              {stats.avgExecutionTime < 60 ? "执行效率良好" : "可优化空间"}
            </div>
          </Card>
        </Col>
      </Row>

      {/* 工具栏 */}
      <Card className="enterprise-card mb-6">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} md={8}>
            <Input
              placeholder="搜索工作流..."
              prefix={<Search className="w-4 h-4" />}
              value={filters.keyword}
              onChange={(e) =>
                setFilters((prev) => ({ ...prev, keyword: e.target.value }))
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
              <Option value="draft">草稿</Option>
              <Option value="active">激活</Option>
              <Option value="inactive">停用</Option>
              <Option value="archived">归档</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="分类筛选"
              value={filters.category}
              onChange={(value) =>
                setFilters((prev) => ({ ...prev, category: value }))
              }
              allowClear
              style={{ width: "100%" }}
            >
              <Option value="审批流程">审批流程</Option>
              <Option value="事件处理">事件处理</Option>
              <Option value="变更管理">变更管理</Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} md={8}>
            <Space>
              <Dropdown
                menu={{
                  items: [
                    {
                      key: "ticket_approval",
                      label: "工单审批流程",
                      icon: <GitBranch className="w-4 h-4" />,
                      onClick: () => router.push("/workflow/ticket-approval"),
                    },
                    {
                      key: "general",
                      label: "通用工作流",
                      icon: <GitBranch className="w-4 h-4" />,
                      onClick: handleCreateWorkflow,
                    },
                  ],
                }}
                trigger={["click"]}
              >
                <Button type="primary" icon={<Plus className="w-4 h-4" />}>
                  新建工作流 <MoreHorizontal className="w-3 h-3 ml-1" />
                </Button>
              </Dropdown>
              <Button
                icon={<Upload className="w-4 h-4" />}
                onClick={handleImportWorkflow}
              >
                导入
              </Button>
              <Button
                icon={<RefreshCw className="w-4 h-4" />}
                onClick={loadWorkflows}
              >
                刷新
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 批量操作工具栏 */}
      {selectedRowKeys.length > 0 && (
        <Card className="enterprise-card mb-4">
          <Alert
            message={
              <Space>
                <span>已选择 {selectedRowKeys.length} 个工作流</span>
                <Button size="small" onClick={() => setSelectedRowKeys([])}>
                  取消选择
                </Button>
              </Space>
            }
            type="info"
            action={
              <Space>
                <Button
                  size="small"
                  type="primary"
                  onClick={handleBatchActivate}
                >
                  批量激活
                </Button>
                <Button size="small" onClick={handleBatchStop}>
                  批量停用
                </Button>
                <Button size="small" onClick={handleBatchExport}>
                  批量导出
                </Button>
                <Button size="small" danger onClick={handleBatchDelete}>
                  批量删除
                </Button>
              </Space>
            }
          />
        </Card>
      )}

      {/* 工作流表格 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={filteredWorkflows}
          rowKey="id"
          loading={loading}
          rowSelection={{
            selectedRowKeys,
            onChange: (selectedRowKeys: React.Key[]) => {
              setSelectedRowKeys(selectedRowKeys.map((key) => Number(key)));
            },
            selections: [
              Table.SELECTION_ALL,
              Table.SELECTION_INVERT,
              Table.SELECTION_NONE,
              {
                key: "select-active",
                text: "选择激活的",
                onSelect: () => {
                  const activeKeys = filteredWorkflows
                    .filter((w) => w.status === "active")
                    .map((w) => w.id);
                  setSelectedRowKeys(activeKeys);
                },
              },
              {
                key: "select-draft",
                text: "选择草稿",
                onSelect: () => {
                  const draftKeys = filteredWorkflows
                    .filter((w) => w.status === "draft")
                    .map((w) => w.id);
                  setSelectedRowKeys(draftKeys);
                },
              },
            ],
          }}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          scroll={{ x: 1200 }}
        />
      </Card>

      {/* 创建/编辑工作流模态框 */}
      <Modal
        title={editingWorkflow ? "编辑工作流" : "新建工作流"}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
        destroyOnHidden
      >
        <Form
          layout="vertical"
          initialValues={editingWorkflow || {}}
          onFinish={async () => {
            try {
              message.success(
                editingWorkflow ? "工作流更新成功" : "工作流创建成功"
              );
              setModalVisible(false);
              loadWorkflows();
            } catch {
              message.error("操作失败");
            }
          }}
        >
          <Form.Item
            name="name"
            label="工作流名称"
            rules={[{ required: true, message: "请输入工作流名称" }]}
          >
            <Input placeholder="请输入工作流名称" />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入工作流描述" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="category"
                label="分类"
                rules={[{ required: true, message: "请选择分类" }]}
              >
                <Select placeholder="选择分类">
                  <Option value="审批流程">审批流程</Option>
                  <Option value="事件处理">事件处理</Option>
                  <Option value="变更管理">变更管理</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="status" label="状态">
                <Select placeholder="选择状态">
                  <Option value="draft">草稿</Option>
                  <Option value="active">激活</Option>
                  <Option value="inactive">停用</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingWorkflow ? "更新" : "创建"}
              </Button>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 工作流设计器 */}
      <Modal
        title={`工作流设计器 - ${editingWorkflow?.name || "新建工作流"}`}
        open={designerVisible}
        onCancel={() => setDesignerVisible(false)}
        footer={null}
        width="90%"
        style={{ top: 20 }}
        destroyOnHidden
      >
        <WorkflowDesigner
          workflow={editingWorkflow}
          onSave={() => {
            message.success("工作流保存成功");
            setDesignerVisible(false);
            loadWorkflows();
          }}
          onCancel={() => setDesignerVisible(false)}
        />
      </Modal>
    </>
  );
};

export default WorkflowManagementPage;
