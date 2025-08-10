"use client";

import { Plus, CheckCircle, Clock, Search, Trash2, Edit, Eye, TrendingUp, AlertTriangle, User, Tag as TagIcon, MoreHorizontal, FileText, UserPlus, RefreshCw, Download, X, Activity, CheckSquare } from "lucide-react";

import React, { useState, useEffect, useMemo } from "react";
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
  Row,
  Col,
  Statistic,
  App,
} from "antd";
import { TicketApi } from "../lib/ticket-api";
import { LoadingEmptyError } from "../components/ui/LoadingEmptyError";

const { Option } = Select;

// 定义页面使用的Ticket接口
interface Ticket {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  category?: string;
  assignee?: string;
  requester?: string;
  created_at: string;
  updated_at: string;
  sla_deadline?: string;
  tags?: string[];
  attachments?: number;
  comments?: number;
  ticket_number: string;
  requester_id: number;
  assignee_id?: number;
  tenant_id: number;
}

interface TicketFilters {
  status?: string;
  priority?: string;
  assignee?: string;
  keyword?: string;
  dateRange?: [string, string];
  category?: string;
}

const TicketManagementPage = () => {
  const { modal } = App.useApp();
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTicket, setEditingTicket] = useState<Ticket | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [filters, setFilters] = useState<TicketFilters>({});
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });
  const [stats, setStats] = useState({
    total: 0,
    open: 0,
    resolved: 0,
    highPriority: 0,
  });

  // 新增企业级功能状态管理
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [escalateModalVisible, setEscalateModalVisible] = useState(false);
  const [resolveModalVisible, setResolveModalVisible] = useState(false);
  const [activityModalVisible, setActivityModalVisible] = useState(false);
  const [selectedTicketForAction, setSelectedTicketForAction] =
    useState<Ticket | null>(null);
  const [ticketActivity, setTicketActivity] = useState<
    Array<{
      action: string;
      timestamp: string;
      user_id: number;
      details: string;
    }>
  >([]);
  const [userList] = useState([
    { id: 1, name: "张三", role: "技术支持" },
    { id: 2, name: "李四", role: "高级工程师" },
    { id: 3, name: "王五", role: "项目经理" },
  ]);

  // 模拟数据
  const mockTickets: Ticket[] = [
    {
      id: 1,
      title: "系统登录异常",
      description: "用户反馈无法正常登录系统，显示密码错误",
      status: "open",
      priority: "high",
      category: "系统问题",
      assignee: "张三",
      requester: "李四",
      created_at: "2024-01-15T10:30:00Z",
      updated_at: "2024-01-15T14:20:00Z",
      sla_deadline: "2024-01-16T10:30:00Z",
      tags: ["登录", "系统"],
      attachments: 2,
      comments: 5,
      ticket_number: "T001",
      requester_id: 1,
      assignee_id: 2,
      tenant_id: 1,
    },
    {
      id: 2,
      title: "打印机无法连接",
      description: "办公室打印机显示离线状态，无法打印文档",
      status: "in_progress",
      priority: "medium",
      category: "硬件问题",
      assignee: "王五",
      requester: "赵六",
      created_at: "2024-01-14T09:15:00Z",
      updated_at: "2024-01-15T11:30:00Z",
      sla_deadline: "2024-01-17T09:15:00Z",
      tags: ["打印机", "硬件"],
      attachments: 1,
      comments: 3,
      ticket_number: "T002",
      requester_id: 3,
      assignee_id: 4,
      tenant_id: 1,
    },
    {
      id: 3,
      title: "网络连接不稳定",
      description: "WiFi信号时强时弱，影响正常工作",
      status: "resolved",
      priority: "high",
      category: "网络问题",
      assignee: "张三",
      requester: "钱七",
      created_at: "2024-01-13T14:20:00Z",
      updated_at: "2024-01-14T16:45:00Z",
      sla_deadline: "2024-01-15T14:20:00Z",
      tags: ["网络", "WiFi"],
      attachments: 0,
      comments: 8,
      ticket_number: "T003",
      requester_id: 5,
      assignee_id: 2,
      tenant_id: 1,
    },
  ];

  useEffect(() => {
    loadTickets();
    loadStats();
  }, [pagination.current, pagination.pageSize, filters]);

  const loadTickets = async () => {
    setLoading(true);
    setError(null);
    try {
      // 模拟API调用
      await new Promise((resolve) => setTimeout(resolve, 500));
      setTickets(mockTickets);
      setPagination((prev) => ({ ...prev, total: mockTickets.length }));
    } catch (error) {
      console.error("加载工单失败:", error);
      setError(error instanceof Error ? error.message : "加载工单失败");
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      setStats({
        total: mockTickets.length,
        open: mockTickets.filter((t) => t.status === "open").length,
        resolved: mockTickets.filter((t) => t.status === "resolved").length,
        highPriority: mockTickets.filter((t) => t.priority === "high").length,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  };

  const handleCreateTicket = () => {
    setEditingTicket(null);
    setModalVisible(true);
  };

  const handleEditTicket = (record: Ticket) => {
    setEditingTicket(record);
    setModalVisible(true);
  };

  const handleDeleteTicket = async (id: number) => {
    try {
      await TicketApi.deleteTicket(id);
      message.success("工单删除成功");
      loadTickets();
    } catch {
      message.error("删除失败");
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要删除的工单");
      return;
    }

    modal.confirm({
      title: "确认删除",
      content: `确定要删除选中的 ${selectedRowKeys.length} 个工单吗？`,
      onOk: async () => {
        try {
          // 批量删除逻辑
          message.success("批量删除成功");
          setSelectedRowKeys([]);
          loadTickets();
        } catch {
          message.error("批量删除失败");
        }
      },
    });
  };

  const handleExport = () => {
    message.success("导出功能开发中");
  };

  // 新增企业级功能处理函数
  const handleAssignTicket = (ticket: Ticket) => {
    setSelectedTicketForAction(ticket);
    setAssignModalVisible(true);
  };

  const handleEscalateTicket = (ticket: Ticket) => {
    setSelectedTicketForAction(ticket);
    setEscalateModalVisible(true);
  };

  const handleResolveTicket = (ticket: Ticket) => {
    setSelectedTicketForAction(ticket);
    setResolveModalVisible(true);
  };

  const handleViewActivity = async (ticket: Ticket) => {
    setSelectedTicketForAction(ticket);
    try {
      const activity = await TicketApi.getTicketActivity(ticket.id);
      setTicketActivity(activity);
      setActivityModalVisible(true);
    } catch (error) {
      message.error("获取工单活动记录失败");
    }
  };

  const handleAssignSubmit = async (values: { assignee_id: number }) => {
    if (!selectedTicketForAction) return;

    try {
      await TicketApi.assignTicket(
        selectedTicketForAction.id,
        values.assignee_id
      );
      message.success("工单分配成功");
      setAssignModalVisible(false);
      loadTickets();
    } catch {
      message.error("工单分配失败");
    }
  };

  const handleEscalateSubmit = async (values: { reason: string }) => {
    if (!selectedTicketForAction) return;

    try {
      await TicketApi.escalateTicket(selectedTicketForAction.id, values.reason);
      message.success("工单升级成功");
      setEscalateModalVisible(false);
      loadTickets();
    } catch {
      message.error("工单升级失败");
    }
  };

  const handleResolveSubmit = async (values: { resolution: string }) => {
    if (!selectedTicketForAction) return;

    try {
      await TicketApi.resolveTicket(
        selectedTicketForAction.id,
        values.resolution
      );
      message.success("工单解决成功");
      setResolveModalVisible(false);
      loadTickets();
    } catch {
      message.error("工单解决失败");
    }
  };

  const handleCloseTicket = async (ticket: Ticket) => {
    modal.confirm({
      title: "确认关闭工单",
      content: "确定要关闭这个工单吗？",
      onOk: async () => {
        try {
          await TicketApi.closeTicket(ticket.id);
          message.success("工单关闭成功");
          loadTickets();
        } catch {
          message.error("工单关闭失败");
        }
      },
    });
  };

  const getStatusColor = (status: string) => {
    const colors = {
      open: "red",
      in_progress: "blue",
      resolved: "green",
      closed: "gray",
    };
    return colors[status as keyof typeof colors] || "default";
  };

  const getPriorityColor = (priority: string) => {
    const colors = {
      low: "green",
      medium: "orange",
      high: "red",
      critical: "purple",
    };
    return colors[priority as keyof typeof colors] || "default";
  };

  const getStatusText = (status: string) => {
    const texts = {
      open: "待处理",
      in_progress: "处理中",
      resolved: "已解决",
      closed: "已关闭",
    };
    return texts[status as keyof typeof texts] || status;
  };

  const getPriorityText = (priority: string) => {
    const texts = {
      low: "低",
      medium: "中",
      high: "高",
      critical: "紧急",
    };
    return texts[priority as keyof typeof texts] || priority;
  };

  const columns = [
    {
      title: "工单号",
      dataIndex: "id",
      key: "id",
      width: 100,
      render: (id: number) => (
        <span className="font-mono text-sm">
          #{String(id).padStart(5, "0")}
        </span>
      ),
    },
    {
      title: "标题",
      dataIndex: "title",
      key: "title",
      render: (title: string, record: Ticket) => (
        <div className="max-w-xs">
          <div className="font-medium text-gray-900 truncate">{title}</div>
          <div className="text-xs text-gray-500 mt-1">
            {record.category} • {record.requester}
          </div>
        </div>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 120,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>
      ),
      filters: [
        { text: "待处理", value: "open" },
        { text: "处理中", value: "in_progress" },
        { text: "已解决", value: "resolved" },
        { text: "已关闭", value: "closed" },
      ],
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 100,
      render: (priority: string) => (
        <Tag color={getPriorityColor(priority)}>
          {getPriorityText(priority)}
        </Tag>
      ),
    },
    {
      title: "处理人",
      dataIndex: "assignee",
      key: "assignee",
      width: 120,
      render: (assignee: string) => (
        <div className="flex items-center">
          <User className="w-4 h-4 mr-1 text-gray-400" />
          <span>{assignee}</span>
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
      sorter: true,
    },
    {
      title: "SLA",
      key: "sla",
      width: 120,
      render: (record: Ticket) => {
        if (!record.sla_deadline) {
          return <Tag color="default">无SLA</Tag>;
        }
        const deadline = new Date(record.sla_deadline);
        const now = new Date();
        const diff = deadline.getTime() - now.getTime();
        const hoursLeft = Math.floor(diff / (1000 * 60 * 60));

        if (hoursLeft < 0) {
          return <Tag color="red">已超时</Tag>;
        } else if (hoursLeft < 24) {
          return <Tag color="orange">{hoursLeft}小时</Tag>;
        } else {
          return <Tag color="green">正常</Tag>;
        }
      },
    },
    {
      title: "操作",
      key: "action",
      width: 120,
      render: (record: Ticket) => (
        <Dropdown
          menu={{
            items: [
              {
                key: "view",
                label: "查看详情",
                icon: <Eye className="w-4 h-4" />,
                onClick: () => window.open(`/tickets/${record.id}`, "_blank"),
              },
              {
                key: "activity",
                label: "查看活动",
                icon: <Activity className="w-4 h-4" />,
                onClick: () => handleViewActivity(record),
              },
              {
                type: "divider",
              },
              {
                key: "assign",
                label: "分配工单",
                icon: <UserPlus className="w-4 h-4" />,
                onClick: () => handleAssignTicket(record),
                disabled: record.status === "closed",
              },
              {
                key: "escalate",
                label: "升级工单",
                icon: <TrendingUp className="w-4 h-4" />,
                onClick: () => handleEscalateTicket(record),
                disabled:
                  record.status === "closed" || record.priority === "critical",
              },
              {
                key: "resolve",
                label: "解决工单",
                icon: <CheckSquare className="w-4 h-4" />,
                onClick: () => handleResolveTicket(record),
                disabled:
                  record.status === "closed" || record.status === "resolved",
              },
              {
                key: "close",
                label: "关闭工单",
                icon: <X className="w-4 h-4" />,
                onClick: () => handleCloseTicket(record),
                disabled: record.status === "closed",
              },
              {
                type: "divider",
              },
              {
                key: "edit",
                label: "编辑",
                icon: <Edit className="w-4 h-4" />,
                onClick: () => handleEditTicket(record),
              },
              {
                key: "delete",
                label: "删除",
                icon: <Trash2 className="w-4 h-4" />,
                danger: true,
                onClick: () => handleDeleteTicket(record.id),
              },
            ],
          }}
        >
          <Button type="text" icon={<MoreHorizontal className="w-4 h-4" />} />
        </Dropdown>
      ),
    },
  ];

  const rowSelection = {
    selectedRowKeys,
    onChange: (newSelectedRowKeys: React.Key[]) => {
      setSelectedRowKeys(newSelectedRowKeys);
    },
  };

  const filteredTickets = useMemo(() => {
    return tickets.filter((ticket) => {
      if (filters.status && ticket.status !== filters.status) return false;
      if (filters.priority && ticket.priority !== filters.priority)
        return false;
      if (
        filters.keyword &&
        !ticket.title.toLowerCase().includes(filters.keyword.toLowerCase())
      )
        return false;
      return true;
    });
  }, [tickets, filters]);

  return (
    <>
      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-stat-card">
            <Statistic
              title="总工单数"
              value={stats.total}
              prefix={<FileText className="w-5 h-5" />}
              valueStyle={{ color: "#3b82f6" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-stat-card">
            <Statistic
              title="待处理"
              value={stats.open}
              prefix={<Clock className="w-5 h-5" />}
              valueStyle={{ color: "#f59e0b" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-stat-card">
            <Statistic
              title="已解决"
              value={stats.resolved}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: "#22c55e" }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-stat-card">
            <Statistic
              title="高优先级"
              value={stats.highPriority}
              prefix={<AlertTriangle className="w-5 h-5" />}
              valueStyle={{ color: "#ef4444" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 工具栏 */}
      <Card className="mb-4">
        <Row gutter={[16, 16]} align="middle">
          <Col flex="auto">
            <Space size="middle">
              <Input
                placeholder="搜索工单..."
                prefix={<Search className="w-4 h-4" />}
                style={{ width: 300 }}
                value={filters.keyword}
                onChange={(e) =>
                  setFilters({ ...filters, keyword: e.target.value })
                }
              />
              <Select
                placeholder="状态筛选"
                style={{ width: 120 }}
                allowClear
                value={filters.status}
                onChange={(value) => setFilters({ ...filters, status: value })}
              >
                <Option value="open">待处理</Option>
                <Option value="in_progress">处理中</Option>
                <Option value="resolved">已解决</Option>
                <Option value="closed">已关闭</Option>
              </Select>
              <Select
                placeholder="优先级"
                style={{ width: 120 }}
                allowClear
                value={filters.priority}
                onChange={(value) =>
                  setFilters({ ...filters, priority: value })
                }
              >
                <Option value="low">低</Option>
                <Option value="medium">中</Option>
                <Option value="high">高</Option>
                <Option value="critical">紧急</Option>
              </Select>
              <Button
                icon={<RefreshCw className="w-4 h-4" />}
                onClick={loadTickets}
              >
                刷新
              </Button>
            </Space>
          </Col>
          <Col>
            <Space>
              {selectedRowKeys.length > 0 && (
                <Button
                  danger
                  icon={<Trash2 className="w-4 h-4" />}
                  onClick={handleBatchDelete}
                >
                  批量删除 ({selectedRowKeys.length})
                </Button>
              )}
              <Button
                icon={<Download className="w-4 h-4" />}
                onClick={handleExport}
              >
                导出
              </Button>
              <Button
                type="primary"
                icon={<Plus className="w-4 h-4" />}
                onClick={handleCreateTicket}
              >
                创建工单
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 工单表格 */}
      <Card>
        <LoadingEmptyError
          state={
            loading
              ? "loading"
              : filteredTickets.length === 0
              ? "empty"
              : "success"
          }
          loadingText="正在加载工单列表..."
          empty={{
            title: "暂无工单",
            description: "当前没有找到匹配的工单记录",
            actions: [
              {
                text: "创建工单",
                icon: <Plus className="w-4 h-4" />,
                onClick: handleCreateTicket,
                type: "primary",
              },
              {
                text: "刷新",
                icon: <RefreshCw className="w-4 h-4" />,
                onClick: loadTickets,
              },
            ],
          }}
          error={{
            title: "加载失败",
            description: "无法获取工单列表，请稍后重试",
            onRetry: loadTickets,
          }}
        >
          <Table
            columns={columns}
            dataSource={filteredTickets}
            rowKey="id"
            loading={false}
            rowSelection={rowSelection}
            pagination={{
              current: pagination.current,
              pageSize: pagination.pageSize,
              total: pagination.total,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total, range) =>
                `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
              onChange: (page, pageSize) => {
                setPagination({ ...pagination, current: page, pageSize });
              },
            }}
            scroll={{ x: 1200 }}
          />
        </LoadingEmptyError>
      </Card>

      {/* 创建/编辑工单模态框 */}
      <Modal
        title={editingTicket ? "编辑工单" : "创建工单"}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
        destroyOnHidden
        className="enterprise-modal"
      >
        <Form
          layout="vertical"
          initialValues={editingTicket || {}}
          className="enterprise-form"
          onFinish={async (values) => {
            try {
              if (editingTicket) {
                await TicketApi.updateTicket(editingTicket.id, values);
                message.success("工单更新成功");
              } else {
                await TicketApi.createTicket(values);
                message.success("工单创建成功");
              }
              setModalVisible(false);
              loadTickets();
            } catch {
              message.error("操作失败");
            }
          }}
        >
          <Form.Item
            name="title"
            label="标题"
            rules={[{ required: true, message: "请输入标题" }]}
          >
            <Input placeholder="请输入工单标题" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
            rules={[{ required: true, message: "请输入描述" }]}
          >
            <Input.TextArea rows={4} placeholder="请详细描述问题" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="priority"
                label="优先级"
                rules={[{ required: true, message: "请选择优先级" }]}
              >
                <Select placeholder="选择优先级">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">紧急</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="category"
                label="分类"
                rules={[{ required: true, message: "请选择分类" }]}
              >
                <Select placeholder="选择分类">
                  <Option value="系统问题">系统问题</Option>
                  <Option value="硬件问题">硬件问题</Option>
                  <Option value="网络问题">网络问题</Option>
                  <Option value="软件问题">软件问题</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="assignee" label="处理人">
            <Select placeholder="选择处理人" allowClear>
              <Option value="张三">张三</Option>
              <Option value="李四">李四</Option>
              <Option value="王五">王五</Option>
            </Select>
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingTicket ? "更新" : "创建"}
              </Button>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 分配工单模态框 */}
      <Modal
        title="分配工单"
        open={assignModalVisible}
        onCancel={() => setAssignModalVisible(false)}
        footer={null}
      >
        <Form onFinish={handleAssignSubmit} layout="vertical">
          <Form.Item
            name="assignee_id"
            label="分配给"
            rules={[{ required: true, message: "请选择处理人" }]}
          >
            <Select placeholder="选择处理人">
              {userList.map((user) => (
                <Option key={user.id} value={user.id}>
                  {user.name} - {user.role}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                分配
              </Button>
              <Button onClick={() => setAssignModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 升级工单模态框 */}
      <Modal
        title="升级工单"
        open={escalateModalVisible}
        onCancel={() => setEscalateModalVisible(false)}
        footer={null}
      >
        <Form onFinish={handleEscalateSubmit} layout="vertical">
          <Form.Item
            name="reason"
            label="升级原因"
            rules={[{ required: true, message: "请输入升级原因" }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="请详细说明升级原因，如：超出SLA时限、问题复杂度提升、需要上级处理等"
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" danger>
                升级
              </Button>
              <Button onClick={() => setEscalateModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 解决工单模态框 */}
      <Modal
        title="解决工单"
        open={resolveModalVisible}
        onCancel={() => setResolveModalVisible(false)}
        footer={null}
      >
        <Form onFinish={handleResolveSubmit} layout="vertical">
          <Form.Item
            name="resolution"
            label="解决方案"
            rules={[{ required: true, message: "请输入解决方案" }]}
          >
            <Input.TextArea
              rows={6}
              placeholder="请详细描述解决方案和处理过程，包括：&#10;1. 问题根本原因&#10;2. 采取的解决措施&#10;3. 验证结果&#10;4. 预防措施"
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                标记为已解决
              </Button>
              <Button onClick={() => setResolveModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 工单活动日志模态框 */}
      <Modal
        title={`工单活动记录 - ${selectedTicketForAction?.title}`}
        open={activityModalVisible}
        onCancel={() => setActivityModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setActivityModalVisible(false)}>
            关闭
          </Button>,
        ]}
        width={800}
      >
        <div className="space-y-4">
          {ticketActivity.map((activity, index) => (
            <div
              key={index}
              className="flex space-x-3 p-3 bg-gray-50 rounded-lg"
            >
              <div className="flex-shrink-0">
                <Activity className="w-5 h-5 text-blue-500" />
              </div>
              <div className="flex-1">
                <div className="flex items-center space-x-2 mb-1">
                  <span className="font-medium text-gray-900">
                    {activity.action}
                  </span>
                  <span className="text-xs text-gray-500">
                    {new Date(activity.timestamp).toLocaleString()}
                  </span>
                </div>
                <p className="text-sm text-gray-600">{activity.details}</p>
                <p className="text-xs text-gray-400 mt-1">
                  操作人: 用户 #{activity.user_id}
                </p>
              </div>
            </div>
          ))}
          {ticketActivity.length === 0 && (
            <div className="text-center py-8 text-gray-500">暂无活动记录</div>
          )}
        </div>
      </Modal>
    </>
  );
};

export default TicketManagementPage;
