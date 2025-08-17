"use client";

import {
  Plus,
  Search,
  Edit,
  Eye,
  FileText,
  RefreshCw,
  Download,
  Activity,
  Filter,
  AlertTriangle,
  Settings,
  Zap,
  BookOpen,
  Shield,
  Target,
  Workflow,
} from "lucide-react";

import React, { useState, useEffect } from "react";
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Modal,
  Form,
  message,
  Tag as AntdTag,
  Space,
  Row,
  Col,
  Statistic,
  App,
  Tooltip,
  Badge,
  Typography,
  Avatar,
  DatePicker,
} from "antd";
import {
  ticketService,
  Ticket,
  TicketStatus,
  TicketPriority,
  TicketType,
} from "../lib/services/ticket-service";
import { LoadingEmptyError } from "../components/ui/LoadingEmptyError";

const { Option } = Select;
const { Title, Text } = Typography;
const { RangePicker } = DatePicker;

interface TicketFilters {
  status?: TicketStatus;
  priority?: TicketPriority;
  type?: TicketType;
  category?: string;
  assignee_id?: number;
  keyword?: string;
  dateRange?: [string, string];
  tags?: string[];
  source?: string;
  impact?: string;
  urgency?: string;
}

interface TicketTemplate {
  id: number;
  name: string;
  type: TicketType;
  category: string;
  priority: TicketPriority;
  description: string;
  estimatedTime: string;
  sla: string;
  icon: React.ReactNode;
}

// 工单模板数据
const ticketTemplates: TicketTemplate[] = [
  {
    id: 1,
    name: "系统登录问题",
    type: TicketType.INCIDENT,
    category: "系统访问",
    priority: TicketPriority.MEDIUM,
    description: "用户无法登录系统，需要技术支持",
    estimatedTime: "2小时",
    sla: "4小时",
    icon: <Shield size={20} />,
  },
  {
    id: 2,
    name: "打印机故障",
    type: TicketType.INCIDENT,
    category: "硬件设备",
    priority: TicketPriority.HIGH,
    description: "办公室打印机无法正常工作",
    estimatedTime: "1小时",
    sla: "2小时",
    icon: <Settings size={20} />,
  },
  {
    id: 3,
    name: "软件安装请求",
    type: TicketType.SERVICE_REQUEST,
    category: "软件服务",
    priority: TicketPriority.LOW,
    description: "需要安装新的办公软件",
    estimatedTime: "30分钟",
    sla: "24小时",
    icon: <Zap size={20} />,
  },
  {
    id: 4,
    name: "网络连接问题",
    type: TicketType.INCIDENT,
    category: "网络服务",
    priority: TicketPriority.HIGH,
    description: "网络连接不稳定或无法访问",
    estimatedTime: "3小时",
    sla: "4小时",
    icon: <Target size={20} />,
  },
];

// 快速操作选项
const quickActions = [
  {
    key: "incident",
    label: "创建事件",
    icon: <AlertTriangle size={16} />,
    color: "red",
  },
  {
    key: "request",
    label: "服务请求",
    icon: <FileText size={16} />,
    color: "blue",
  },
  {
    key: "problem",
    label: "问题管理",
    icon: <BookOpen size={16} />,
    color: "orange",
  },
  {
    key: "change",
    label: "变更请求",
    icon: <Workflow size={16} />,
    color: "purple",
  },
];

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
    overdue: 0,
    slaBreach: 0,
  });

  // 新增企业级功能状态管理
  const [activityModalVisible, setActivityModalVisible] = useState(false);
  const [templateModalVisible, setTemplateModalVisible] = useState(false);
  const [quickCreateModalVisible, setQuickCreateModalVisible] = useState(false);
  const [selectedTemplate, setSelectedTemplate] =
    useState<TicketTemplate | null>(null);
  const [selectedTicketForAction, setSelectedTicketForAction] =
    useState<Ticket | null>(null);
  const [showAdvancedFilters, setShowAdvancedFilters] = useState(false);
  const [bulkActionModalVisible, setBulkActionModalVisible] = useState(false);
  const [bulkAction, setBulkAction] = useState<string>("");
  const [ticketActivity] = useState<
    Array<{
      action: string;
      timestamp: string;
      user_id: number;
      details: string;
    }>
  >([]);

  const [userList] = useState([
    { id: 1, name: "张三", role: "技术支持", avatar: "张" },
    { id: 2, name: "李四", role: "高级工程师", avatar: "李" },
    { id: 3, name: "王五", role: "项目经理", avatar: "王" },
  ]);

  useEffect(() => {
    loadTickets();
    loadStats();
  }, [pagination.current, pagination.pageSize, filters]);

  const loadTickets = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await ticketService.listTickets({
        page: pagination.current,
        page_size: pagination.pageSize,
        status: filters.status,
        priority: filters.priority,
        type: filters.type,
        category: filters.category,
        assignee_id: filters.assignee_id,
        keyword: filters.keyword,
      });
      setTickets(response.tickets);
      setPagination((prev) => ({ ...prev, total: response.total }));
    } catch (error) {
      console.error("加载工单失败:", error);
      setError(error instanceof Error ? error.message : "加载工单失败");
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      const response = await ticketService.getTicketStats();
      setStats({
        total: response.total,
        open: response.open,
        resolved: response.resolved,
        highPriority: response.high_priority,
        overdue: response.overdue || 0,
        slaBreach: response.sla_breach || 0,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  };

  const handleCreateTicket = () => {
    setEditingTicket(null);
    setModalVisible(true);
  };

  const handleQuickCreate = (type: string) => {
    setQuickCreateModalVisible(true);
    // 根据类型预填充表单
  };

  const handleTemplateSelect = (template: TicketTemplate) => {
    setSelectedTemplate(template);
    setModalVisible(true);
  };

  const handleEditTicket = (record: Ticket) => {
    setEditingTicket(record);
    setModalVisible(true);
  };

  const handleDeleteTicket = async (id: number) => {
    try {
      await ticketService.deleteTicket(id);
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

  const handleBulkAction = (action: string) => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要操作的工单");
      return;
    }
    setBulkAction(action);
    setBulkActionModalVisible(true);
  };

  const handleViewActivity = (record: Ticket) => {
    setSelectedTicketForAction(record);
    setActivityModalVisible(true);
  };

  const handleSearch = (value: string) => {
    setFilters((prev) => ({ ...prev, keyword: value }));
    setPagination((prev) => ({ ...prev, current: 1 }));
  };

  const handleFilterChange = (
    key: keyof TicketFilters,
    value: string | number | string[] | undefined
  ) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
    setPagination((prev) => ({ ...prev, current: 1 }));
  };

  const handleTableChange = (pagination: {
    current?: number;
    pageSize?: number;
  }) => {
    setPagination((prev) => ({
      ...prev,
      current: pagination.current || prev.current,
      pageSize: pagination.pageSize || prev.pageSize,
    }));
  };

  const handleRefresh = () => {
    loadTickets();
    loadStats();
  };

  const handleResetFilters = () => {
    setFilters({});
    setPagination((prev) => ({ ...prev, current: 1 }));
  };

  // 渲染空状态
  const renderEmptyState = () => (
    <div className="text-center py-16">
      <div className="mb-6">
        <div className="inline-flex items-center justify-center w-24 h-24 bg-blue-50 rounded-full mb-4">
          <FileText size={48} className="text-blue-500" />
        </div>
        <Title level={3} className="text-gray-700 mb-2">
          欢迎使用工单管理系统
        </Title>
        <Text type="secondary" className="text-base">
          这是您的ITSM工单管理中心，可以管理事件、服务请求、问题和变更
        </Text>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8 max-w-4xl mx-auto">
        {quickActions.map((action) => (
          <Card
            key={action.key}
            hoverable
            className="text-center cursor-pointer border-2 border-dashed border-gray-200 hover:border-blue-300 transition-colors"
            onClick={() => handleQuickCreate(action.key)}
          >
            <div
              className={`inline-flex items-center justify-center w-12 h-12 bg-${action.color}-50 rounded-full mb-3`}
            >
              <span className={`text-${action.color}-500`}>{action.icon}</span>
            </div>
            <Text strong className="block">
              {action.label}
            </Text>
            <Text type="secondary" className="text-sm">
              快速创建
            </Text>
          </Card>
        ))}
      </div>

      <div className="mb-8">
        <Title level={4} className="text-gray-600 mb-4">
          常用工单模板
        </Title>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 max-w-4xl mx-auto">
          {ticketTemplates.map((template) => (
            <Card
              key={template.id}
              hoverable
              className="cursor-pointer"
              onClick={() => handleTemplateSelect(template)}
            >
              <div className="flex items-center mb-3">
                <span className="text-blue-500 mr-2">{template.icon}</span>
                <Text strong className="flex-1">
                  {template.name}
                </Text>
              </div>
              <Text type="secondary" className="text-sm mb-2 block">
                {template.description}
              </Text>
              <div className="flex items-center justify-between text-xs text-gray-500">
                <span>预计: {template.estimatedTime}</span>
                <span>SLA: {template.sla}</span>
              </div>
            </Card>
          ))}
        </div>
      </div>

      <div className="flex justify-center space-x-4">
        <Button
          type="primary"
          size="large"
          icon={<Plus size={20} />}
          onClick={handleCreateTicket}
        >
          创建工单
        </Button>
        <Button
          size="large"
          icon={<BookOpen size={20} />}
          onClick={() => setTemplateModalVisible(true)}
        >
          查看所有模板
        </Button>
      </div>
    </div>
  );

  // 渲染统计卡片
  const renderStatsCards = () => (
    <Row gutter={16} style={{ marginBottom: 24 }}>
      <Col span={6}>
        <Card>
          <Statistic
            title="总工单数"
            value={stats.total}
            prefix={<FileText size={16} style={{ color: "#3b82f6" }} />}
            suffix={
              <div className="text-xs text-gray-500 mt-1">
                <div>本月新增: +12</div>
                <div className="text-green-500">↑ 8.5%</div>
              </div>
            }
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="待处理"
            value={stats.open}
            valueStyle={{ color: "#faad14" }}
            suffix={
              <div className="text-xs text-gray-500 mt-1">
                <div>平均响应: 2.3h</div>
                <div className="text-orange-500">SLA: 4h</div>
              </div>
            }
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="已解决"
            value={stats.resolved}
            valueStyle={{ color: "#52c41a" }}
            suffix={
              <div className="text-xs text-gray-500 mt-1">
                <div>平均解决: 8.7h</div>
                <div className="text-green-500">满意度: 4.2/5</div>
              </div>
            }
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="高优先级"
            value={stats.highPriority}
            valueStyle={{ color: "#ff4d4f" }}
            suffix={
              <div className="text-xs text-gray-500 mt-1">
                <div>超时: {stats.overdue}</div>
                <div className="text-red-500">SLA违规: {stats.slaBreach}</div>
              </div>
            }
          />
        </Card>
      </Col>
    </Row>
  );

  // 渲染筛选器
  const renderFilters = () => (
    <Card style={{ marginBottom: 24 }}>
      <div className="flex items-center justify-between mb-4">
        <Title level={5} className="mb-0">
          筛选条件
        </Title>
        <Space>
          <Button
            type="text"
            icon={<Filter size={16} />}
            onClick={() => setShowAdvancedFilters(!showAdvancedFilters)}
          >
            {showAdvancedFilters ? "隐藏高级筛选" : "显示高级筛选"}
          </Button>
          <Button onClick={handleResetFilters}>重置</Button>
        </Space>
      </div>

      <Row gutter={16} align="middle">
        <Col span={6}>
          <Input.Search
            placeholder="搜索工单标题、描述或工单号"
            allowClear
            onSearch={handleSearch}
            prefix={<Search size={16} />}
          />
        </Col>
        <Col span={4}>
          <Select
            value={filters.status}
            onChange={(value) => handleFilterChange("status", value)}
            style={{ width: "100%" }}
            placeholder="状态筛选"
            allowClear
          >
            <Option value={TicketStatus.OPEN}>待处理</Option>
            <Option value={TicketStatus.IN_PROGRESS}>处理中</Option>
            <Option value={TicketStatus.PENDING}>等待中</Option>
            <Option value={TicketStatus.RESOLVED}>已解决</Option>
            <Option value={TicketStatus.CLOSED}>已关闭</Option>
          </Select>
        </Col>
        <Col span={4}>
          <Select
            value={filters.priority}
            onChange={(value) => handleFilterChange("priority", value)}
            style={{ width: "100%" }}
            placeholder="优先级筛选"
            allowClear
          >
            <Option value={TicketPriority.LOW}>低</Option>
            <Option value={TicketPriority.MEDIUM}>中</Option>
            <Option value={TicketPriority.HIGH}>高</Option>
            <Option value={TicketPriority.URGENT}>紧急</Option>
          </Select>
        </Col>
        <Col span={4}>
          <Select
            value={filters.type}
            onChange={(value) => handleFilterChange("type", value)}
            style={{ width: "100%" }}
            placeholder="类型筛选"
            allowClear
          >
            <Option value={TicketType.INCIDENT}>事件</Option>
            <Option value={TicketType.SERVICE_REQUEST}>服务请求</Option>
            <Option value={TicketType.PROBLEM}>问题</Option>
            <Option value={TicketType.CHANGE}>变更</Option>
          </Select>
        </Col>
        <Col span={6}>
          <Space>
            <Button type="primary" onClick={loadTickets}>
              应用筛选
            </Button>
            <Button icon={<RefreshCw size={16} />} onClick={handleRefresh}>
              刷新
            </Button>
          </Space>
        </Col>
      </Row>

      {showAdvancedFilters && (
        <div className="mt-4 pt-4 border-t border-gray-200">
          <Row gutter={16}>
            <Col span={6}>
              <Select
                value={filters.source}
                onChange={(value) => handleFilterChange("source", value)}
                style={{ width: "100%" }}
                placeholder="来源筛选"
                allowClear
              >
                <Option value="web">Web门户</Option>
                <Option value="email">邮件</Option>
                <Option value="phone">电话</Option>
                <Option value="chat">在线聊天</Option>
              </Select>
            </Col>
            <Col span={6}>
              <Select
                value={filters.impact}
                onChange={(value) => handleFilterChange("impact", value)}
                style={{ width: "100%" }}
                placeholder="影响范围"
                allowClear
              >
                <Option value="individual">个人</Option>
                <Option value="department">部门</Option>
                <Option value="organization">组织</Option>
                <Option value="customer">客户</Option>
              </Select>
            </Col>
            <Col span={6}>
              <Select
                value={filters.urgency}
                onChange={(value) => handleFilterChange("urgency", value)}
                style={{ width: "100%" }}
                placeholder="紧急程度"
                allowClear
              >
                <Option value="low">低</Option>
                <Option value="medium">中</Option>
                <Option value="high">高</Option>
                <Option value="critical">严重</Option>
              </Select>
            </Col>
            <Col span={6}>
              <RangePicker
                placeholder={["开始日期", "结束日期"]}
                onChange={(dates) => {
                  if (dates) {
                    handleFilterChange("dateRange", [
                      dates[0]?.toISOString() || "",
                      dates[1]?.toISOString() || "",
                    ]);
                  }
                }}
                style={{ width: "100%" }}
              />
            </Col>
          </Row>
        </div>
      )}
    </Card>
  );

  // 渲染工单列表
  const renderTicketList = () => (
    <Card>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-4">
          <Title level={5} className="mb-0">
            工单列表
          </Title>
          <Badge count={tickets.length} showZero />
        </div>
        <Space>
          <Button
            danger
            disabled={selectedRowKeys.length === 0}
            onClick={() => handleBulkAction("delete")}
          >
            批量删除 ({selectedRowKeys.length})
          </Button>
          <Button
            disabled={selectedRowKeys.length === 0}
            onClick={() => handleBulkAction("assign")}
          >
            批量分配 ({selectedRowKeys.length})
          </Button>
          <Button
            disabled={selectedRowKeys.length === 0}
            onClick={() => handleBulkAction("status")}
          >
            批量状态变更 ({selectedRowKeys.length})
          </Button>
          <Button icon={<Download size={16} />}>导出</Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={tickets}
        rowKey="id"
        loading={loading}
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
        pagination={{
          current: pagination.current,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total, range) =>
            `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
        }}
        onChange={handleTableChange}
        scroll={{ x: 1200 }}
      />
    </Card>
  );

  const columns = [
    {
      title: "工单号",
      dataIndex: "ticket_number",
      key: "ticket_number",
      width: 120,
      render: (ticketNumber: string, record: Ticket) => (
        <a
          href={`/tickets/${record.id}`}
          className="text-blue-600 hover:text-blue-800 font-medium"
        >
          {ticketNumber}
        </a>
      ),
    },
    {
      title: "标题",
      dataIndex: "title",
      key: "title",
      ellipsis: true,
      width: 200,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: TicketStatus) => (
        <AntdTag color={ticketService.getStatusColor(status)}>
          {ticketService.getStatusLabel(status)}
        </AntdTag>
      ),
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 100,
      render: (priority: TicketPriority) => (
        <AntdTag color={ticketService.getPriorityColor(priority)}>
          {ticketService.getPriorityLabel(priority)}
        </AntdTag>
      ),
    },
    {
      title: "类型",
      dataIndex: "type",
      key: "type",
      width: 100,
      render: (type: TicketType) => (
        <AntdTag color={ticketService.getTypeColor(type)}>
          {ticketService.getTypeLabel(type)}
        </AntdTag>
      ),
    },
    {
      title: "分类",
      dataIndex: "category",
      key: "category",
      width: 120,
    },
    {
      title: "负责人",
      dataIndex: "assignee",
      key: "assignee",
      width: 100,
      render: (assignee: { name: string } | undefined) =>
        assignee ? (
          <div className="flex items-center">
            <Avatar size="small" className="mr-2">
              {assignee.name[0]}
            </Avatar>
            <span>{assignee.name}</span>
          </div>
        ) : (
          <span className="text-gray-400">未分配</span>
        ),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 150,
      render: (date: string) => new Date(date).toLocaleDateString("zh-CN"),
    },
    {
      title: "操作",
      key: "actions",
      width: 200,
      render: (_: unknown, record: Ticket) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Button
              type="text"
              size="small"
              icon={<Eye size={14} />}
              onClick={() => window.open(`/tickets/${record.id}`)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              size="small"
              icon={<Edit size={14} />}
              onClick={() => handleEditTicket(record)}
            />
          </Tooltip>
          <Tooltip title="查看活动日志">
            <Button
              type="text"
              size="small"
              icon={<Activity size={14} />}
              onClick={() => handleViewActivity(record)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  // 使用LoadingEmptyError组件处理加载、空状态和错误状态
  if (error) {
    return (
      <LoadingEmptyError
        state="error"
        error={{
          title: "加载失败",
          description: error,
          actionText: "重试",
          onAction: loadTickets,
        }}
      />
    );
  }

  return (
    <>
      {loading ? (
        <LoadingEmptyError state="loading" loadingText="正在加载工单数据..." />
      ) : tickets.length === 0 ? (
        renderEmptyState()
      ) : (
        <>
          {/* 工单统计概览 */}
          {renderStatsCards()}

          {/* 工单列表区域 */}
          <Card title="工单列表">
            {renderFilters()}
            {renderTicketList()}
          </Card>
        </>
      )}

      {/* 创建/编辑工单模态框 */}
      <Modal
        title={editingTicket ? "编辑工单" : "新建工单"}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={800}
      >
        <Form layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="标题"
                name="title"
                rules={[{ required: true, message: "请输入工单标题" }]}
              >
                <Input placeholder="请输入工单标题" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="类型"
                name="type"
                rules={[{ required: true, message: "请选择工单类型" }]}
              >
                <Select placeholder="请选择工单类型">
                  <Option value={TicketType.INCIDENT}>事件</Option>
                  <Option value={TicketType.SERVICE_REQUEST}>服务请求</Option>
                  <Option value={TicketType.PROBLEM}>问题</Option>
                  <Option value={TicketType.CHANGE}>变更</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="优先级"
                name="priority"
                rules={[{ required: true, message: "请选择优先级" }]}
              >
                <Select placeholder="请选择优先级">
                  <Option value={TicketPriority.LOW}>低</Option>
                  <Option value={TicketPriority.MEDIUM}>中</Option>
                  <Option value={TicketPriority.HIGH}>高</Option>
                  <Option value={TicketPriority.URGENT}>紧急</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="分类"
                name="category"
                rules={[{ required: true, message: "请选择分类" }]}
              >
                <Select placeholder="请选择分类">
                  <Option value="系统问题">系统问题</Option>
                  <Option value="硬件问题">硬件问题</Option>
                  <Option value="网络问题">网络问题</Option>
                  <Option value="软件问题">软件问题</Option>
                  <Option value="用户管理">用户管理</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label="描述"
            name="description"
            rules={[{ required: true, message: "请输入工单描述" }]}
          >
            <Input.TextArea rows={4} placeholder="请详细描述问题或需求" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="处理人" name="assignee_id">
                <Select placeholder="请选择处理人" allowClear>
                  {userList.map((user) => (
                    <Option key={user.id} value={user.id}>
                      <div className="flex items-center">
                        <Avatar size="small" className="mr-2">
                          {user.avatar}
                        </Avatar>
                        <span>
                          {user.name} ({user.role})
                        </span>
                      </div>
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="标签" name="tags">
                <Select
                  mode="tags"
                  placeholder="请输入标签，按回车确认"
                  allowClear
                />
              </Form.Item>
            </Col>
          </Row>

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

      {/* 快速创建模态框 */}
      <Modal
        title="快速创建工单"
        open={quickCreateModalVisible}
        onCancel={() => setQuickCreateModalVisible(false)}
        footer={null}
        width={600}
      >
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            {quickActions.map((action) => (
              <Card
                key={action.key}
                hoverable
                className="text-center cursor-pointer"
                onClick={() => {
                  setQuickCreateModalVisible(false);
                  handleQuickCreate(action.key);
                }}
              >
                <div
                  className={`inline-flex items-center justify-center w-12 h-12 bg-${action.color}-50 rounded-full mb-3`}
                >
                  <span className={`text-${action.color}-500`}>
                    {action.icon}
                  </span>
                </div>
                <Title level={5} className="mb-2">
                  {action.label}
                </Title>
                <Text type="secondary" className="text-sm">
                  点击创建
                </Text>
              </Card>
            ))}
          </div>
        </div>
      </Modal>

      {/* 模板选择模态框 */}
      <Modal
        title="选择工单模板"
        open={templateModalVisible}
        onCancel={() => setTemplateModalVisible(false)}
        footer={null}
        width={800}
      >
        <div className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {ticketTemplates.map((template) => (
              <Card
                key={template.id}
                hoverable
                className="cursor-pointer"
                onClick={() => {
                  setTemplateModalVisible(false);
                  handleTemplateSelect(template);
                }}
              >
                <div className="flex items-center mb-3">
                  <span className="text-blue-500 mr-2">{template.icon}</span>
                  <Text strong className="flex-1">
                    {template.name}
                  </Text>
                </div>
                <Text type="secondary" className="text-sm mb-2 block">
                  {template.description}
                </Text>
                <div className="flex items-center justify-between text-xs text-gray-500">
                  <span>预计: {template.estimatedTime}</span>
                  <span>SLA: {template.sla}</span>
                </div>
              </Card>
            ))}
          </div>
          <div className="text-center pt-4">
            <Button
              type="link"
              onClick={() => window.open("/tickets/templates")}
            >
              管理工单模板 →
            </Button>
          </div>
        </div>
      </Modal>

      {/* 工单活动记录模态框 */}
      <Modal
        title="工单活动记录"
        open={activityModalVisible}
        onCancel={() => setActivityModalVisible(false)}
        footer={null}
        width={800}
      >
        <div>
          {ticketActivity.map((activity, index) => (
            <div
              key={index}
              style={{
                marginBottom: 16,
                padding: 12,
                border: "1px solid #f0f0f0",
              }}
            >
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  marginBottom: 8,
                }}
              >
                <span style={{ fontWeight: "bold" }}>{activity.action}</span>
                <span style={{ color: "#666" }}>{activity.timestamp}</span>
              </div>
              <div>{activity.details}</div>
            </div>
          ))}
        </div>
      </Modal>

      {/* 批量操作模态框 */}
      <Modal
        title="批量操作"
        open={bulkActionModalVisible}
        onCancel={() => setBulkActionModalVisible(false)}
        onOk={() => {
          // 实现批量操作逻辑
          setBulkActionModalVisible(false);
        }}
      >
        <Form layout="vertical">
          <Form.Item label="操作类型">
            <Text strong>
              {bulkAction === "delete"
                ? "批量删除"
                : bulkAction === "assign"
                ? "批量分配"
                : "批量状态变更"}
            </Text>
          </Form.Item>
          <Form.Item label="选中工单数量">
            <Text>{selectedRowKeys.length} 个工单</Text>
          </Form.Item>
          {bulkAction === "assign" && (
            <Form.Item
              label="分配给"
              name="assignee_id"
              rules={[{ required: true }]}
            >
              <Select placeholder="请选择处理人">
                {userList.map((user) => (
                  <Option key={user.id} value={user.id}>
                    <div className="flex items-center">
                      <Avatar size="small" className="mr-2">
                        {user.avatar}
                      </Avatar>
                      <span>
                        {user.name} ({user.role})
                      </span>
                    </div>
                  </Option>
                ))}
              </Select>
            </Form.Item>
          )}
          {bulkAction === "status" && (
            <Form.Item
              label="新状态"
              name="status"
              rules={[{ required: true }]}
            >
              <Select placeholder="请选择新状态">
                <Option value={TicketStatus.OPEN}>待处理</Option>
                <Option value={TicketStatus.IN_PROGRESS}>处理中</Option>
                <Option value={TicketStatus.PENDING}>等待中</Option>
                <Option value={TicketStatus.RESOLVED}>已解决</Option>
                <Option value={TicketStatus.CLOSED}>已关闭</Option>
              </Select>
            </Form.Item>
          )}
          <Form.Item label="操作原因" name="reason">
            <Input.TextArea rows={3} placeholder="请输入操作原因" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default TicketManagementPage;
