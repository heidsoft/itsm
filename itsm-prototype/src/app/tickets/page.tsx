"use client";

import {
  Plus,
  Edit,
  Eye,
  FileText,
  RefreshCw,
  Download,
  Activity,
  AlertTriangle,
  Settings,
  Zap,
  BookOpen,
  Shield,
  Workflow,
  Clock,
  CheckCircle,
  TrendingUp,
  Users,
  MoreHorizontal,
} from "lucide-react";

import React, { useState, useEffect, useMemo, useCallback } from "react";
import {
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
  App,
  Tooltip,
  Badge,
  Typography,
  Avatar,
  DatePicker,
  Dropdown,
  Card,
} from "antd";
import {
  ticketService,
  Ticket,
  TicketStatus,
  TicketPriority,
  TicketType,
} from "../lib/services/ticket-service";
import { LoadingEmptyError } from "../components/ui/LoadingEmptyError";
import { TicketAssociation } from "../components/TicketAssociation";
import { SatisfactionDashboard } from "../components/SatisfactionDashboard";

const { Option } = Select;
const { Text } = Typography;

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
    sla: "4小时",
    icon: <Zap size={20} />,
  },
  {
    id: 4,
    name: "网络连接问题",
    type: TicketType.INCIDENT,
    category: "网络服务",
    priority: TicketPriority.HIGH,
    description: "网络连接不稳定，影响工作",
    estimatedTime: "3小时",
    sla: "4小时",
    icon: <Workflow size={20} />,
  },
];

// 优化后的Tickets页面组件
const TicketsPageComponent: React.FC = React.memo(() => {
  const { modal } = App.useApp();
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [templateModalVisible, setTemplateModalVisible] = useState(false);
  const [editingTicket, setEditingTicket] = useState<Ticket | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 筛选和分页状态
  const [filters, setFilters] = useState<TicketFilters>({});
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });

  // 统计数据
  const [stats, setStats] = useState({
    total: 0,
    open: 0,
    resolved: 0,
    highPriority: 0,
  });

  const [userList] = useState([
    { id: 1, name: "张三", role: "技术支持", avatar: "张" },
    { id: 2, name: "李四", role: "高级工程师", avatar: "李" },
    { id: 3, name: "王五", role: "项目经理", avatar: "王" },
  ]);

  // 使用useCallback优化函数，避免不必要的重新创建
  const loadTickets = useCallback(async () => {
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
  }, [pagination.current, pagination.pageSize, filters]);

  const loadStats = useCallback(async () => {
    try {
      const response = await ticketService.getTicketStats();
      setStats({
        total: response.total,
        open: response.open,
        resolved: response.resolved,
        highPriority: response.high_priority,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  }, []);

  useEffect(() => {
    loadTickets();
    loadStats();
  }, [loadTickets, loadStats]);

  // 使用useCallback优化事件处理函数
  const handleCreateTicket = useCallback(() => {
    setEditingTicket(null);
    setModalVisible(true);
  }, []);

  const handleEditTicket = useCallback((ticket: Ticket) => {
     setEditingTicket(ticket);
     setModalVisible(true);
   }, []);

  const handleBatchDelete = useCallback(async () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要删除的工单");
      return;
    }

    modal.confirm({
      title: "确认删除",
      content: `确定要删除选中的 ${selectedRowKeys.length} 个工单吗？此操作不可撤销。`,
      okText: "确认删除",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        try {
          await Promise.all(
            selectedRowKeys.map((id) => ticketService.deleteTicket(Number(id)))
          );
          message.success("删除成功");
          setSelectedRowKeys([]);
          loadTickets();
        } catch (error) {
          console.error("删除失败:", error);
          message.error("删除失败，请重试");
        }
      },
    });
  }, [selectedRowKeys, modal, loadTickets]);

  const handleViewActivity = useCallback((ticket: Ticket) => {
     console.log("查看活动日志:", ticket);
   }, []);

  const handleSearch = useCallback((value: string) => {
    setFilters((prev) => ({ ...prev, keyword: value }));
    setPagination((prev) => ({ ...prev, current: 1 }));
  }, []);

  const handleFilterChange = useCallback((
    key: keyof TicketFilters,
    value: string | number | string[] | undefined
  ) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
    setPagination((prev) => ({ ...prev, current: 1 }));
  }, []);

  const handleRefresh = useCallback(() => {
    loadTickets();
    loadStats();
  }, [loadTickets, loadStats]);



  // 使用useMemo优化统计卡片渲染
  const renderStatsCards = useMemo(() => (
    <div className="mb-6">
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="border-0 shadow-sm hover:shadow-md transition-all duration-300 bg-gradient-to-br from-blue-50 to-blue-100">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-600 mb-1">总工单数</div>
                <div className="text-2xl font-bold text-blue-600">{stats.total}</div>
              </div>
              <div className="w-12 h-12 bg-blue-500 rounded-lg flex items-center justify-center">
                <FileText size={24} className="text-white" />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="border-0 shadow-sm hover:shadow-md transition-all duration-300 bg-gradient-to-br from-orange-50 to-orange-100">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-600 mb-1">待处理</div>
                <div className="text-2xl font-bold text-orange-600">{stats.open}</div>
              </div>
              <div className="w-12 h-12 bg-orange-500 rounded-lg flex items-center justify-center">
                <Clock size={24} className="text-white" />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="border-0 shadow-sm hover:shadow-md transition-all duration-300 bg-gradient-to-br from-green-50 to-green-100">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-600 mb-1">已解决</div>
                <div className="text-2xl font-bold text-green-600">{stats.resolved}</div>
              </div>
              <div className="w-12 h-12 bg-green-500 rounded-lg flex items-center justify-center">
                <CheckCircle size={24} className="text-white" />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="border-0 shadow-sm hover:shadow-md transition-all duration-300 bg-gradient-to-br from-red-50 to-red-100">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-600 mb-1">高优先级</div>
                <div className="text-2xl font-bold text-red-600">{stats.highPriority}</div>
              </div>
              <div className="w-12 h-12 bg-red-500 rounded-lg flex items-center justify-center">
                <AlertTriangle size={24} className="text-white" />
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  ), [stats]);

  // 使用useMemo优化筛选器渲染
  const renderFilters = useMemo(() => (
    <Card className="mb-6 border-0 shadow-sm bg-gradient-to-r from-gray-50 to-white">
      <div className="p-2">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} lg={8}>
            <div className="relative">
              <Input.Search
                placeholder="搜索工单标题、ID或描述..."
                allowClear
                onSearch={handleSearch}
                size="large"
                className="rounded-lg border-gray-200 hover:border-blue-400 focus:border-blue-500 transition-colors"
              />
            </div>
          </Col>
          <Col xs={24} sm={12} lg={4}>
            <Select
              placeholder="状态筛选"
              size="large"
              allowClear
              value={filters.status}
              onChange={(value) => handleFilterChange("status", value)}
              className="w-full rounded-lg"
              style={{ borderRadius: '8px' }}
            >
              <Option value={TicketStatus.OPEN}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-orange-500 rounded-full mr-2"></div>
                  待处理
                </div>
              </Option>
              <Option value={TicketStatus.IN_PROGRESS}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-blue-500 rounded-full mr-2"></div>
                  处理中
                </div>
              </Option>
              <Option value={TicketStatus.RESOLVED}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
                  已解决
                </div>
              </Option>
              <Option value={TicketStatus.CLOSED}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-gray-500 rounded-full mr-2"></div>
                  已关闭
                </div>
              </Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} lg={4}>
            <Select
              placeholder="优先级"
              size="large"
              allowClear
              value={filters.priority}
              onChange={(value) => handleFilterChange("priority", value)}
              className="w-full rounded-lg"
              style={{ borderRadius: '8px' }}
            >
              <Option value={TicketPriority.LOW}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
                  低
                </div>
              </Option>
              <Option value={TicketPriority.MEDIUM}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-blue-500 rounded-full mr-2"></div>
                  中
                </div>
              </Option>
              <Option value={TicketPriority.HIGH}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-orange-500 rounded-full mr-2"></div>
                  高
                </div>
              </Option>
              <Option value={TicketPriority.URGENT}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-red-500 rounded-full mr-2"></div>
                  紧急
                </div>
              </Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} lg={4}>
            <Select
              placeholder="类型"
              size="large"
              allowClear
              value={filters.type}
              onChange={(value) => handleFilterChange("type", value)}
              className="w-full rounded-lg"
              style={{ borderRadius: '8px' }}
            >
              <Option value={TicketType.INCIDENT}>
                <div className="flex items-center">
                  <AlertTriangle size={14} className="text-red-500 mr-2" />
                  事件
                </div>
              </Option>
              <Option value={TicketType.SERVICE_REQUEST}>
                <div className="flex items-center">
                  <Settings size={14} className="text-blue-500 mr-2" />
                  服务请求
                </div>
              </Option>
              <Option value={TicketType.PROBLEM}>
                <div className="flex items-center">
                  <Zap size={14} className="text-orange-500 mr-2" />
                  问题
                </div>
              </Option>
              <Option value={TicketType.CHANGE}>
                <div className="flex items-center">
                  <Workflow size={14} className="text-purple-500 mr-2" />
                  变更
                </div>
              </Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} lg={4}>
            <Button
              icon={<RefreshCw size={18} />}
              onClick={handleRefresh}
              loading={loading}
              size="large"
              className="w-full rounded-lg border-gray-200 hover:border-blue-400 hover:text-blue-600 transition-all duration-200"
            >
              刷新
            </Button>
          </Col>
        </Row>
      </div>
    </Card>
  ), [filters.status, filters.priority, filters.type, handleSearch, handleFilterChange, handleRefresh, loading]);

  // 使用useMemo优化工单列表渲染
  const renderTicketList = useMemo(() => (
    <div>
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6 p-4 bg-white rounded-lg border border-gray-100 shadow-sm">
        <div className="flex items-center gap-3">
          {selectedRowKeys.length > 0 && (
            <div className="flex items-center gap-2">
              <Badge
                count={selectedRowKeys.length}
                showZero
                className="bg-blue-500"
              />
              <span className="text-sm text-gray-600">
                已选择 {selectedRowKeys.length} 个工单
              </span>
            </div>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {selectedRowKeys.length > 0 && (
            <Button 
              danger 
              size="large" 
              onClick={handleBatchDelete}
              className="rounded-lg hover:shadow-md transition-all duration-200"
              icon={<AlertTriangle size={16} />}
            >
              批量删除 ({selectedRowKeys.length})
            </Button>
          )}
          <Button 
            icon={<Download size={16} />} 
            size="large"
            className="rounded-lg border-gray-200 hover:border-blue-400 hover:text-blue-600 transition-all duration-200"
          >
            导出数据
          </Button>
          <Button
            type="primary"
            icon={<Plus size={18} />}
            size="large"
            onClick={handleCreateTicket}
            className="rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 border-0 shadow-md hover:shadow-lg transition-all duration-200"
          >
            创建工单
          </Button>
        </div>
      </div>

      <div className="bg-white rounded-lg border border-gray-100 shadow-sm overflow-hidden">
        <Table
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          columns={columns}
          dataSource={tickets}
          rowKey="id"
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
            onChange: (page, size) => {
              setPagination((prev) => ({
                ...prev,
                current: page,
                pageSize: size,
              }));
            },
            className: "px-6 py-4 border-t border-gray-100",
          }}
          scroll={{ x: 1200 }}
          className="[&_.ant-table-thead>tr>th]:bg-gray-50 [&_.ant-table-thead>tr>th]:border-b-2 [&_.ant-table-thead>tr>th]:border-gray-200 [&_.ant-table-thead>tr>th]:font-semibold [&_.ant-table-thead>tr>th]:text-gray-700 [&_.ant-table-tbody>tr]:hover:bg-blue-50 [&_.ant-table-tbody>tr>td]:border-b [&_.ant-table-tbody>tr>td]:border-gray-100 [&_.ant-table]:border-0"
        />
      </div>
    </div>
  ), [selectedRowKeys, handleBatchDelete, handleCreateTicket, columns, tickets, loading, pagination]);

  // 使用useMemo优化表格列定义
  const columns = useMemo(() => [
    {
      title: "工单信息",
      key: "ticket_info",
      width: 300,
      render: (_: unknown, record: Ticket) => (
        <div style={{ display: "flex", alignItems: "center" }}>
          <div
            style={{
              width: 40,
              height: 40,
              backgroundColor: "#e6f7ff",
              borderRadius: 8,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              marginRight: 12,
            }}
          >
            <FileText size={20} style={{ color: "#1890ff" }} />
          </div>
          <div>
            <div
              style={{ fontWeight: "medium", color: "#000", marginBottom: 4 }}
            >
              {record.title}
            </div>
            <div style={{ fontSize: "small", color: "#666" }}>
              #{record.id} • {record.category}
            </div>
          </div>
        </div>
      ),
    },
    {
      title: "状态",
      key: "status",
      width: 120,
      render: (_: unknown, record: Ticket) => {
        const statusConfig: Record<
          string,
          { color: string; text: string; backgroundColor: string }
        > = {
          [TicketStatus.OPEN]: {
            color: "#fa8c16",
            text: "待处理",
            backgroundColor: "#fff7e6",
          },
          [TicketStatus.IN_PROGRESS]: {
            color: "#1890ff",
            text: "处理中",
            backgroundColor: "#e6f7ff",
          },
          [TicketStatus.PENDING]: {
            color: "#faad14",
            text: "等待中",
            backgroundColor: "#fffbe6",
          },
          [TicketStatus.RESOLVED]: {
            color: "#52c41a",
            text: "已解决",
            backgroundColor: "#f6ffed",
          },
          [TicketStatus.CLOSED]: {
            color: "#00000073",
            text: "已关闭",
            backgroundColor: "#fafafa",
          },
          [TicketStatus.CANCELLED]: {
            color: "#00000073",
            text: "已取消",
            backgroundColor: "#fafafa",
          },
        };
        const config = statusConfig[record.status];
        return (
          <span
            style={{
              padding: "4px 12px",
              borderRadius: 16,
              fontSize: "small",
              fontWeight: 500,
              color: config.color,
              backgroundColor: config.backgroundColor,
            }}
          >
            {config.text}
          </span>
        );
      },
    },
    {
      title: "优先级",
      key: "priority",
      width: 100,
      render: (_: unknown, record: Ticket) => {
        const priorityConfig: Record<
          string,
          { color: string; text: string; backgroundColor: string }
        > = {
          [TicketPriority.LOW]: {
            color: "#52c41a",
            text: "低",
            backgroundColor: "#f6ffed",
          },
          [TicketPriority.MEDIUM]: {
            color: "#1890ff",
            text: "中",
            backgroundColor: "#e6f7ff",
          },
          [TicketPriority.HIGH]: {
            color: "#fa8c16",
            text: "高",
            backgroundColor: "#fff7e6",
          },
          [TicketPriority.URGENT]: {
            color: "#ff4d4f",
            text: "紧急",
            backgroundColor: "#fff2f0",
          },
        };
        const config = priorityConfig[record.priority];
        return (
          <span
            style={{
              padding: "4px 12px",
              borderRadius: 16,
              fontSize: "small",
              fontWeight: 500,
              color: config.color,
              backgroundColor: config.backgroundColor,
            }}
          >
            {config.text}
          </span>
        );
      },
    },
    {
      title: "类型",
      key: "type",
      width: 120,
      render: (_: unknown, record: Ticket) => {
        const typeConfig: Record<
          string,
          { color: string; text: string; backgroundColor: string }
        > = {
          [TicketType.INCIDENT]: {
            color: "#ff4d4f",
            text: "事件",
            backgroundColor: "#fff2f0",
          },
          [TicketType.SERVICE_REQUEST]: {
            color: "#1890ff",
            text: "服务请求",
            backgroundColor: "#e6f7ff",
          },
          [TicketType.PROBLEM]: {
            color: "#fa8c16",
            text: "问题",
            backgroundColor: "#fff7e6",
          },
          [TicketType.CHANGE]: {
            color: "#722ed1",
            text: "变更",
            backgroundColor: "#f9f0ff",
          },
        };
        const config = typeConfig[record.type];
        return (
          <span
            style={{
              padding: "4px 12px",
              borderRadius: 16,
              fontSize: "small",
              fontWeight: 500,
              color: config.color,
              backgroundColor: config.backgroundColor,
            }}
          >
            {config.text}
          </span>
        );
      },
    },
    {
      title: "处理人",
      key: "assignee",
      width: 120,
      render: (_: unknown, record: Ticket) => (
        <div style={{ display: "flex", alignItems: "center" }}>
          <Avatar
            size="small"
            style={{ backgroundColor: "#1890ff", marginRight: 8 }}
          >
            {record.assignee?.name?.[0] || "U"}
          </Avatar>
          <span style={{ fontSize: "small" }}>
            {record.assignee?.name || "未分配"}
          </span>
        </div>
      ),
    },
    {
      title: "创建时间",
      key: "created_at",
      width: 150,
      render: (_: unknown, record: Ticket) => (
        <div style={{ fontSize: "small", color: "#666" }}>
          {new Date(record.created_at).toLocaleDateString("zh-CN")}
        </div>
      ),
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
              icon={<Eye size={16} />}
              onClick={() => window.open(`/tickets/${record.id}`)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              size="small"
              icon={<Edit size={16} />}
              onClick={() => handleEditTicket(record)}
            />
          </Tooltip>
          <Tooltip title="查看活动日志">
            <Button
              type="text"
              size="small"
              icon={<Activity size={16} />}
              onClick={() => handleViewActivity(record)}
            />
          </Tooltip>
          <Dropdown
            menu={{
              items: [
                {
                  key: "assign",
                  label: "分配处理人",
                  icon: <Users size={16} />,
                },
                {
                  key: "escalate",
                  label: "升级工单",
                  icon: <TrendingUp size={16} />,
                },
                {
                  type: "divider",
                },
                {
                  key: "delete",
                  label: "删除工单",
                  icon: <AlertTriangle size={16} />,
                  danger: true,
                },
              ],
            }}
            trigger={["click"]}
          >
            <Button
              type="text"
              size="small"
              icon={<MoreHorizontal size={16} />}
            />
          </Dropdown>
        </Space>
      ),
    },
  ], [handleEditTicket, handleViewActivity]);

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
    <div>
      {renderStatsCards}
      {renderFilters}
      {renderTicketList}

      {/* 工单关联与合并 */}
      <div className="mt-8">
        <TicketAssociation />
      </div>

      {/* 满意度分析看板 */}
      <div className="mt-8">
        <SatisfactionDashboard />
      </div>

      {/* 创建/编辑工单模态框 */}
      <Modal
        title={
          <div className="flex items-center gap-3 pb-4 border-b border-gray-100">
            <div className="w-10 h-10 bg-gradient-to-r from-blue-500 to-blue-600 rounded-lg flex items-center justify-center shadow-md">
              {editingTicket ? <Edit size={20} className="text-white" /> : <Plus size={20} className="text-white" />}
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900">
                {editingTicket ? "编辑工单" : "创建工单"}
              </h3>
              <p className="text-sm text-gray-500">
                {editingTicket ? "修改工单信息" : "填写工单详细信息"}
              </p>
            </div>
          </div>
        }
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={900}
        className="[&_.ant-modal-content]:rounded-xl [&_.ant-modal-content]:shadow-2xl [&_.ant-modal-header]:border-0 [&_.ant-modal-header]:pb-0 [&_.ant-modal-body]:pt-6"
      >
        <Form layout="vertical" style={{ marginTop: 20 }}>
          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                label="标题"
                name="title"
                rules={[{ required: true, message: "请输入工单标题" }]}
              >
                <Input placeholder="请输入工单标题" size="large" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="类型"
                name="type"
                rules={[{ required: true, message: "请选择工单类型" }]}
              >
                <Select placeholder="请选择工单类型" size="large">
                  <Option value={TicketType.INCIDENT}>事件</Option>
                  <Option value={TicketType.SERVICE_REQUEST}>服务请求</Option>
                  <Option value={TicketType.PROBLEM}>问题</Option>
                  <Option value={TicketType.CHANGE}>变更</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                label="分类"
                name="category"
                rules={[{ required: true, message: "请选择工单分类" }]}
              >
                <Select placeholder="请选择工单分类" size="large">
                  <Option value="系统访问">系统访问</Option>
                  <Option value="硬件设备">硬件设备</Option>
                  <Option value="软件服务">软件服务</Option>
                  <Option value="网络服务">网络服务</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="优先级"
                name="priority"
                rules={[{ required: true, message: "请选择优先级" }]}
              >
                <Select placeholder="请选择优先级" size="large">
                  <Option value={TicketPriority.LOW}>低</Option>
                  <Option value={TicketPriority.MEDIUM}>中</Option>
                  <Option value={TicketPriority.HIGH}>高</Option>
                  <Option value={TicketPriority.URGENT}>紧急</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item label="处理人" name="assignee_id">
                <Select placeholder="请选择处理人" allowClear size="large">
                  {userList.map((user) => (
                    <Option key={user.id} value={user.id}>
                      <div style={{ display: "flex", alignItems: "center" }}>
                        <Avatar
                          size="small"
                          style={{ backgroundColor: "#1890ff", marginRight: 8 }}
                        >
                          {user.avatar}
                        </Avatar>
                        <span>{user.name}</span>
                        <span style={{ color: "#666", marginLeft: 8 }}>
                          ({user.role})
                        </span>
                      </div>
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="预计完成时间" name="estimated_time">
                <DatePicker
                  showTime
                  placeholder="请选择预计完成时间"
                  size="large"
                  style={{ width: "100%" }}
                />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label="描述"
            name="description"
            rules={[{ required: true, message: "请输入工单描述" }]}
          >
            <Input.TextArea
              rows={6}
              placeholder="请详细描述工单内容、问题现象、期望结果等..."
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <div className="flex justify-end gap-3 pt-6 border-t border-gray-100">
              <Button 
                onClick={() => setModalVisible(false)} 
                size="large"
                className="rounded-lg border-gray-200 hover:border-gray-300 transition-colors duration-200"
              >
                取消
              </Button>
              <Button 
                type="primary" 
                htmlType="submit" 
                size="large"
                className="rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 border-0 shadow-md hover:shadow-lg transition-all duration-200"
                icon={editingTicket ? <Edit size={16} /> : <Plus size={16} />}
              >
                {editingTicket ? "更新工单" : "创建工单"}
              </Button>
            </div>
          </Form.Item>
        </Form>
      </Modal>

      {/* 工单模板管理模态框 */}
      <Modal
        title={
          <div className="flex items-center gap-3 pb-4 border-b border-gray-100">
            <div className="w-10 h-10 bg-gradient-to-r from-green-500 to-green-600 rounded-lg flex items-center justify-center shadow-md">
              <BookOpen size={20} className="text-white" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900">
                工单模板管理
              </h3>
              <p className="text-sm text-gray-500">
                管理和配置工单模板，提升工作效率
              </p>
            </div>
          </div>
        }
        open={templateModalVisible}
        onCancel={() => setTemplateModalVisible(false)}
        footer={null}
        width={1100}
        className="[&_.ant-modal-content]:rounded-xl [&_.ant-modal-content]:shadow-2xl [&_.ant-modal-header]:border-0 [&_.ant-modal-header]:pb-0 [&_.ant-modal-body]:pt-6"
      >
        <div style={{ padding: "24px 0" }}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              marginBottom: 24,
            }}
          >
            <div>
              <Text style={{ color: "#666" }}>
                管理所有可用的工单模板，提升工单创建效率
              </Text>
            </div>
            <Button type="primary" icon={<Plus size={16} />}>
              新建模板
            </Button>
          </div>

          <Table
            columns={[
              {
                title: "模板名称",
                dataIndex: "name",
                key: "name",
                render: (name: string) => (
                  <div style={{ fontWeight: "medium", color: "#000" }}>
                    {name}
                  </div>
                ),
              },
              {
                title: "类型",
                dataIndex: "type",
                key: "type",
                render: (type: TicketType) => {
                  const typeConfig: Record<
                    string,
                    { color: string; text: string }
                  > = {
                    [TicketType.INCIDENT]: { color: "red", text: "事件" },
                    [TicketType.SERVICE_REQUEST]: {
                      color: "blue",
                      text: "服务请求",
                    },
                    [TicketType.PROBLEM]: { color: "orange", text: "问题" },
                    [TicketType.CHANGE]: { color: "purple", text: "变更" },
                  };
                  const config = typeConfig[type];
                  return <AntdTag color={config.color}>{config.text}</AntdTag>;
                },
              },
              {
                title: "分类",
                dataIndex: "category",
                key: "category",
                render: (category: string) => (
                  <span style={{ color: "#666" }}>{category}</span>
                ),
              },
              {
                title: "优先级",
                dataIndex: "priority",
                key: "priority",
                render: (priority: TicketPriority) => {
                  const priorityConfig: Record<
                    string,
                    { color: string; text: string }
                  > = {
                    [TicketPriority.LOW]: { color: "green", text: "低" },
                    [TicketPriority.MEDIUM]: { color: "blue", text: "中" },
                    [TicketPriority.HIGH]: { color: "orange", text: "高" },
                    [TicketPriority.URGENT]: { color: "red", text: "紧急" },
                  };
                  const config = priorityConfig[priority];
                  return <AntdTag color={config.color}>{config.text}</AntdTag>;
                },
              },
              {
                title: "操作",
                key: "actions",
                render: () => (
                  <Space>
                    <Button size="small" icon={<Edit size={14} />}>
                      编辑
                    </Button>
                    <Button size="small" danger>
                      删除
                    </Button>
                  </Space>
                ),
              },
            ]}
            dataSource={ticketTemplates}
            rowKey="id"
            pagination={false}
            rowClassName="hover:bg-gray-50 transition-colors"
          />
        </div>
      </Modal>
    </div>
  );
});

// 设置组件显示名称
TicketsPageComponent.displayName = 'TicketsPageComponent';

// 导出组件
export default TicketsPageComponent;
