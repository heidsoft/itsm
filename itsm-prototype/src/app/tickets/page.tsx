"use client";

import {FileText,
  RefreshCw,
  Download,
  Activity,
  Zap,
  Workflow} from 'lucide-react';

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
import TicketFilters, { TicketFilterState } from "../components/TicketFilters";

const { Option } = Select;
const { Text } = Typography;

interface TicketQueryFilters {
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

// Removed unused TicketTemplate interface to fix lint errors

// Ticket template data
const ticketTemplates = [
  {
    id: 1,
    name: "System Login Issue",
    type: TicketType.INCIDENT,
    category: "System Access",
    priority: TicketPriority.MEDIUM,
    description: "User unable to login to system, technical support needed",
    estimatedTime: "2 hours",
    sla: "4 hours",
    icon: <Shield size={20} />,
  },
  {
    id: 2,
    name: "Printer Malfunction",
    type: TicketType.INCIDENT,
    category: "Hardware Equipment",
    priority: TicketPriority.HIGH,
    description: "Office printer not working properly",
    estimatedTime: "1 hour",
    sla: "2 hours",
    icon: <Settings size={20} />,
  },
  {
    id: 3,
    name: "Software Installation Request",
    type: TicketType.SERVICE_REQUEST,
    category: "Software Services",
    priority: TicketPriority.LOW,
    description: "Need to install new office software",
    estimatedTime: "30 minutes",
    sla: "4 hours",
    icon: <Zap size={20} />,
  },
  {
    id: 4,
    name: "Network Connection Issue",
    type: TicketType.INCIDENT,
    category: "Network Services",
    priority: TicketPriority.HIGH,
    description: "Network connection unstable, affecting work",
    estimatedTime: "3 hours",
    sla: "4 hours",
    icon: <Workflow size={20} />,
  },
];

// Optimized Tickets page component
export default function TicketsPage() {
  // Core page state
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 });
  const [filters, setFilters] = useState<Partial<TicketQueryFilters>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [editingTicket, setEditingTicket] = useState<Ticket | null>(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [templateModalVisible, setTemplateModalVisible] = useState(false);
  const userList = [
    { id: 1, name: "Alice", role: "IT Support", avatar: "A" },
    { id: 2, name: "Bob", role: "Network Admin", avatar: "B" },
    { id: 3, name: "Charlie", role: "Service Desk", avatar: "C" },
  ];

  // New TDD filter UI state (keeps component and domain filters in sync)
  const [componentFilters, setComponentFilters] = useState<TicketFilterState>({
    status: "all",
    priority: "all",
    keyword: "",
    dateStart: "",
    dateEnd: "",
    sortBy: "createdAt_desc",
  });

  // Map TicketFilters UI state to domain filters used by services
  const mapComponentToDomain = useCallback((cf: TicketFilterState): Partial<TicketQueryFilters> => {
    const statusMap: Record<TicketFilterState["status"], TicketStatus | undefined> = {
      all: undefined,
      open: TicketStatus.OPEN,
      in_progress: TicketStatus.IN_PROGRESS,
      resolved: TicketStatus.RESOLVED,
      closed: TicketStatus.CLOSED,
    };
    const priorityMap: Record<TicketFilterState["priority"], TicketPriority | undefined> = {
      all: undefined,
      p1: TicketPriority.URGENT,
      p2: TicketPriority.HIGH,
      p3: TicketPriority.MEDIUM,
      p4: TicketPriority.LOW,
    };
    const dateRange = cf.dateStart && cf.dateEnd ? [cf.dateStart, cf.dateEnd] as [string, string] : undefined;
    return {
      status: statusMap[cf.status],
      priority: priorityMap[cf.priority],
      keyword: cf.keyword || undefined,
      dateRange,
    };
  }, []);

  const mapDomainToComponent = useCallback((df: Partial<TicketQueryFilters>): TicketFilterState => {
    const statusReverseMap: Record<string, TicketFilterState["status"]> = {
      [TicketStatus.OPEN]: "open",
      [TicketStatus.IN_PROGRESS]: "in_progress",
      [TicketStatus.RESOLVED]: "resolved",
      [TicketStatus.CLOSED]: "closed",
    };
    const priorityReverseMap: Record<string, TicketFilterState["priority"]> = {
      [TicketPriority.URGENT]: "p1",
      [TicketPriority.HIGH]: "p2",
      [TicketPriority.MEDIUM]: "p3",
      [TicketPriority.LOW]: "p4",
    };
    return {
      status: df.status ? statusReverseMap[df.status] : "all",
      priority: df.priority ? priorityReverseMap[df.priority] : "all",
      keyword: df.keyword || "",
      dateStart: df.dateRange?.[0] || "",
      dateEnd: df.dateRange?.[1] || "",
      sortBy: "createdAt_desc",
    };
  }, []);
  
  // Statistics data aligned to usage in UI
  const [stats, setStats] = useState({
    total: 0,
    open: 0,
    resolved: 0,
    highPriority: 0,
  });

  // Use useCallback to optimize functions, avoiding unnecessary recreation
  const fetchTickets = useCallback(async () => {
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
      console.error("Failed to load tickets:", error);
      setError(error instanceof Error ? error.message : "Failed to load tickets");
    } finally {
      setLoading(false);
    }
  }, [pagination.current, pagination.pageSize, filters]);

  // Public wrapper for fetching tickets
  const loadTickets = useCallback(async () => {
    await fetchTickets();
  }, [fetchTickets]);

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
      console.error("Failed to load statistics:", error);
    }
  }, []);

  useEffect(() => {
    loadTickets();
    loadStats();
  }, [loadTickets, loadStats]);

  // Use useCallback to optimize event handling functions
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
      message.warning("Please select tickets to delete");
      return;
    }

    Modal.confirm({
      title: "Confirm Delete",
      content: `Are you sure you want to delete the selected ${selectedRowKeys.length} tickets? This action cannot be undone.`,
      okText: "Confirm Delete",
      okType: "danger",
      cancelText: "Cancel",
      onOk: async () => {
        try {
          await Promise.all(
            selectedRowKeys.map((id: React.Key) => ticketService.deleteTicket(Number(id)))
          );
          message.success("Deleted successfully");
          setSelectedRowKeys([]);
          loadTickets();
        } catch (error) {
          console.error("Delete failed:", error);
          message.error("Delete failed, please try again");
        }
      },
    });
  }, [selectedRowKeys, loadTickets]);

  const handleViewActivity = useCallback((ticket: Ticket) => {
     console.log("View activity log:", ticket);
   }, []);

  const handleSearch = useCallback((value: string) => {
    setFilters((prev) => ({ ...prev, keyword: value }));
    setPagination((prev) => ({ ...prev, current: 1 }));
    setComponentFilters((prev) => ({ ...prev, keyword: value }));
  }, []);

  const handleFilterChange = useCallback((
    key: keyof TicketQueryFilters,
    value: string | number | string[] | undefined
  ) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
    setPagination((prev) => ({ ...prev, current: 1 }));
    // Sync TDD UI component state for status/priority
    if (key === "status" && typeof value === "string") {
      const next = mapDomainToComponent({ status: value as TicketStatus });
      setComponentFilters((prev) => ({ ...prev, status: next.status }));
    }
    if (key === "priority" && typeof value === "string") {
      const next = mapDomainToComponent({ priority: value as TicketPriority });
      setComponentFilters((prev) => ({ ...prev, priority: next.priority }));
    }
  }, [mapDomainToComponent]);

  const handleRefresh = useCallback(() => {
    loadTickets();
    loadStats();
  }, [loadTickets, loadStats]);



  // Use useMemo to optimize statistics card rendering
  const renderStatsCards = useMemo(() => (
    <div className="mb-6">
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="border-0 shadow-sm hover:shadow-md transition-all duration-300 bg-gradient-to-br from-blue-50 to-blue-100">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-600 mb-1">Total Tickets</div>
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
                <div className="text-sm text-gray-600 mb-1">Open</div>
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
                <div className="text-sm text-gray-600 mb-1">Resolved</div>
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
                <div className="text-sm text-gray-600 mb-1">High Priority</div>
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

  // Use useMemo to optimize filter rendering
  const renderFilters = useMemo(() => (
    <Card className="mb-6 border-0 shadow-sm bg-gradient-to-r from-gray-50 to-white">
      <div className="p-2">
        <Row gutter={[16, 16]} align="middle">
          {/* 新增：TDD筛选组件，逐步替换AntD筛选区域 */}
          <Col span={24}>
            <div className="mb-3">
              <TicketFilters
                value={componentFilters}
                onChange={(next) => {
                  setComponentFilters(next);
                  const domain = mapComponentToDomain(next);
                  setFilters((prev) => ({ ...prev, ...domain }));
                  setPagination((prev) => ({ ...prev, current: 1 }));
                }}
              />
            </div>
          </Col>
          <Col xs={24} sm={12} lg={8}>
            <div className="relative">
              <Input.Search
                placeholder="Search ticket title, ID or description..."
                allowClear
                onSearch={handleSearch}
                size="large"
                className="rounded-lg border-gray-200 hover:border-blue-400 focus:border-blue-500 transition-colors"
                data-testid="input-搜索工单"
              />
            </div>
          </Col>
          <Col xs={24} sm={12} lg={4}>
            <Select
              placeholder="Status Filter"
              size="large"
              allowClear
              value={filters.status}
              onChange={(value) => handleFilterChange("status", value)}
              className="w-full rounded-lg"
              style={{ borderRadius: '8px' }}
              data-testid="select-状态筛选"
            >
              <Option value={TicketStatus.OPEN}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-orange-500 rounded-full mr-2"></div>
                  Open
                </div>
              </Option>
              <Option value={TicketStatus.IN_PROGRESS}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-blue-500 rounded-full mr-2"></div>
                  In Progress
                </div>
              </Option>
              <Option value={TicketStatus.RESOLVED}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
                  Resolved
                </div>
              </Option>
              <Option value={TicketStatus.CLOSED}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-gray-500 rounded-full mr-2"></div>
                  Closed
                </div>
              </Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} lg={4}>
            <Select
              placeholder="Priority"
              size="large"
              allowClear
              value={filters.priority}
              onChange={(value) => handleFilterChange("priority", value)}
              className="w-full rounded-lg"
              style={{ borderRadius: '8px' }}
              data-testid="select-优先级筛选"
            >
              <Option value={TicketPriority.LOW}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
                  Low
                </div>
              </Option>
              <Option value={TicketPriority.MEDIUM}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-blue-500 rounded-full mr-2"></div>
                  Medium
                </div>
              </Option>
              <Option value={TicketPriority.HIGH}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-orange-500 rounded-full mr-2"></div>
                  High
                </div>
              </Option>
              <Option value={TicketPriority.URGENT}>
                <div className="flex items-center">
                  <div className="w-2 h-2 bg-red-500 rounded-full mr-2"></div>
                  Urgent
                </div>
              </Option>
            </Select>
          </Col>
          <Col xs={24} sm={12} lg={4}>
            <Select
              placeholder="Type"
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
                  Incident
                </div>
              </Option>
              <Option value={TicketType.SERVICE_REQUEST}>
                <div className="flex items-center">
                  <Settings size={14} className="text-blue-500 mr-2" />
                  Service Request
                </div>
              </Option>
              <Option value={TicketType.PROBLEM}>
                <div className="flex items-center">
                  <Zap size={14} className="text-orange-500 mr-2" />
                  Problem
                </div>
              </Option>
              <Option value={TicketType.CHANGE}>
                <div className="flex items-center">
                  <Workflow size={14} className="text-purple-500 mr-2" />
                  Change
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
              Refresh
            </Button>
          </Col>
        </Row>
      </div>
    </Card>
  ), [filters.status, filters.priority, filters.type, handleSearch, handleFilterChange, handleRefresh, loading]);

  // Define table columns before usage
  const columns = useMemo(() => [
    {
      title: "Ticket Information",
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
      title: "Status",
      key: "status",
      width: 120,
      render: (_: unknown, record: Ticket) => {
        const statusConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
          [TicketStatus.OPEN]: { color: "#fa8c16", text: "Open", backgroundColor: "#fff7e6" },
          [TicketStatus.IN_PROGRESS]: { color: "#1890ff", text: "In Progress", backgroundColor: "#e6f7ff" },
          [TicketStatus.PENDING]: { color: "#faad14", text: "Pending", backgroundColor: "#fffbe6" },
          [TicketStatus.RESOLVED]: { color: "#52c41a", text: "Resolved", backgroundColor: "#f6ffed" },
          [TicketStatus.CLOSED]: { color: "#00000073", text: "Closed", backgroundColor: "#fafafa" },
          [TicketStatus.CANCELLED]: { color: "#00000073", text: "Cancelled", backgroundColor: "#fafafa" },
        };
        const config = statusConfig[record.status];
        return (
          <span style={{ padding: "4px 12px", borderRadius: 16, fontSize: "small", fontWeight: 500, color: config.color, backgroundColor: config.backgroundColor }}>
            {config.text}
          </span>
        );
      },
    },
    {
      title: "Priority",
      key: "priority",
      width: 100,
      render: (_: unknown, record: Ticket) => {
        const priorityConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
          [TicketPriority.LOW]: { color: "#52c41a", text: "Low", backgroundColor: "#f6ffed" },
          [TicketPriority.MEDIUM]: { color: "#1890ff", text: "Medium", backgroundColor: "#e6f7ff" },
          [TicketPriority.HIGH]: { color: "#fa8c16", text: "High", backgroundColor: "#fff7e6" },
          [TicketPriority.URGENT]: { color: "#ff4d4f", text: "Urgent", backgroundColor: "#fff2f0" },
        };
        const config = priorityConfig[record.priority];
        return (
          <span style={{ padding: "4px 12px", borderRadius: 16, fontSize: "small", fontWeight: 500, color: config.color, backgroundColor: config.backgroundColor }}>
            {config.text}
          </span>
        );
      },
    },
    {
      title: "Type",
      key: "type",
      width: 120,
      render: (_: unknown, record: Ticket) => {
        const typeConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
          [TicketType.INCIDENT]: { color: "#ff4d4f", text: "Incident", backgroundColor: "#fff2f0" },
          [TicketType.SERVICE_REQUEST]: { color: "#1890ff", text: "Service Request", backgroundColor: "#e6f7ff" },
          [TicketType.PROBLEM]: { color: "#fa8c16", text: "Problem", backgroundColor: "#fff7e6" },
          [TicketType.CHANGE]: { color: "#722ed1", text: "Change", backgroundColor: "#f9f0ff" },
        };
        const config = typeConfig[record.type];
        return (
          <span style={{ padding: "4px 12px", borderRadius: 16, fontSize: "small", fontWeight: 500, color: config.color, backgroundColor: config.backgroundColor }}>
            {config.text}
          </span>
        );
      },
    },
    {
      title: "Assignee",
      key: "assignee",
      width: 120,
      render: (_: unknown, record: Ticket) => (
        <div style={{ display: "flex", alignItems: "center" }}>
          <Avatar size="small" style={{ backgroundColor: "#1890ff", marginRight: 8 }}>
            {record.assignee?.name?.[0] || "U"}
          </Avatar>
          <span style={{ fontSize: "small" }}>
            {record.assignee?.name || "Unassigned"}
          </span>
        </div>
      ),
    },
    {
      title: "Created Time",
      key: "created_at",
      width: 150,
      render: (_: unknown, record: Ticket) => (
        <div style={{ fontSize: "small", color: "#666" }}>
          {new Date(record.created_at).toLocaleDateString("zh-CN")}
        </div>
      ),
    },
    {
      title: "Actions",
      key: "actions",
      width: 200,
      render: (_: unknown, record: Ticket) => (
        <Space size="small">
          <Tooltip title="View Details">
            <Button type="text" size="small" icon={<Eye size={16} />} onClick={() => window.open(`/tickets/${record.id}`)} />
          </Tooltip>
          <Tooltip title="Edit">
            <Button type="text" size="small" icon={<Edit size={16} />} onClick={() => handleEditTicket(record)} />
          </Tooltip>
          <Tooltip title="View Activity Log">
            <Button type="text" size="small" icon={<Activity size={16} />} onClick={() => handleViewActivity(record)} />
          </Tooltip>
          <Dropdown
            menu={{
              items: [
                { key: "assign", label: "Assign Handler", icon: <Users size={16} /> },
                { key: "escalate", label: "Escalate Ticket", icon: <TrendingUp size={16} /> },
                { type: "divider" },
                { key: "delete", label: "Delete Ticket", icon: <AlertTriangle size={16} />, danger: true },
              ],
            }}
            trigger={["click"]}
          >
            <Button type="text" size="small" icon={<MoreHorizontal size={16} />} />
          </Dropdown>
        </Space>
      ),
    },
  ], [handleEditTicket, handleViewActivity]);

  // Use useMemo to optimize ticket list rendering
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
                Selected {selectedRowKeys.length} tickets
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
              Batch Delete ({selectedRowKeys.length})
            </Button>
          )}
          <Button 
            icon={<Download size={16} />} 
            size="large"
            className="rounded-lg border-gray-200 hover:border-blue-400 hover:text-blue-600 transition-all duration-200"
          >
            Export Data
          </Button>
          <Button
            type="primary"
            icon={<Plus size={18} />}
            size="large"
            onClick={handleCreateTicket}
            className="rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 border-0 shadow-md hover:shadow-lg transition-all duration-200"
          >
            Create Ticket
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
              `${range[0]}-${range[1]} of ${total} items`,
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

  /* Duplicate columns definition start
  // Use useMemo to optimize table column definition
  const columns = useMemo(() => [
    {
      title: "Ticket Information",
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
      title: "Status",
      key: "status",
      width: 120,
      render: (_: unknown, record: Ticket) => {
        const statusConfig: Record<
          string,
          { color: string; text: string; backgroundColor: string }
        > = {
          [TicketStatus.OPEN]: {
            color: "#fa8c16",
            text: "Open",
            backgroundColor: "#fff7e6",
          },
          [TicketStatus.IN_PROGRESS]: {
            color: "#1890ff",
            text: "In Progress",
            backgroundColor: "#e6f7ff",
          },
          [TicketStatus.PENDING]: {
            color: "#faad14",
            text: "Pending",
            backgroundColor: "#fffbe6",
          },
          [TicketStatus.RESOLVED]: {
            color: "#52c41a",
            text: "Resolved",
            backgroundColor: "#f6ffed",
          },
          [TicketStatus.CLOSED]: {
            color: "#00000073",
            text: "Closed",
            backgroundColor: "#fafafa",
          },
          [TicketStatus.CANCELLED]: {
            color: "#00000073",
            text: "Cancelled",
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
      title: "Priority",
      key: "priority",
      width: 100,
      render: (_: unknown, record: Ticket) => {
        const priorityConfig: Record<
          string,
          { color: string; text: string; backgroundColor: string }
        > = {
          [TicketPriority.LOW]: {
            color: "#52c41a",
            text: "Low",
            backgroundColor: "#f6ffed",
          },
          [TicketPriority.MEDIUM]: {
            color: "#1890ff",
            text: "Medium",
            backgroundColor: "#e6f7ff",
          },
          [TicketPriority.HIGH]: {
            color: "#fa8c16",
            text: "High",
            backgroundColor: "#fff7e6",
          },
          [TicketPriority.URGENT]: {
            color: "#ff4d4f",
            text: "Urgent",
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
      title: "Type",
      key: "type",
      width: 120,
      render: (_: unknown, record: Ticket) => {
        const typeConfig: Record<
          string,
          { color: string; text: string; backgroundColor: string }
        > = {
          [TicketType.INCIDENT]: {
            color: "#ff4d4f",
            text: "Incident",
            backgroundColor: "#fff2f0",
          },
          [TicketType.SERVICE_REQUEST]: {
            color: "#1890ff",
            text: "Service Request",
            backgroundColor: "#e6f7ff",
          },
          [TicketType.PROBLEM]: {
            color: "#fa8c16",
            text: "Problem",
            backgroundColor: "#fff7e6",
          },
          [TicketType.CHANGE]: {
            color: "#722ed1",
            text: "Change",
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
      title: "Assignee",
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
            {record.assignee?.name || "Unassigned"}
          </span>
        </div>
      ),
    },
    {
      title: "Created Time",
      key: "created_at",
      width: 150,
      render: (_: unknown, record: Ticket) => (
        <div style={{ fontSize: "small", color: "#666" }}>
          {new Date(record.created_at).toLocaleDateString("zh-CN")}
        </div>
      ),
    },
    {
      title: "Actions",
      key: "actions",
      width: 200,
      render: (_: unknown, record: Ticket) => (
        <Space size="small">
          <Tooltip title="View Details">
            <Button
              type="text"
              size="small"
              icon={<Eye size={16} />}
              onClick={() => window.open(`/tickets/${record.id}`)}
            />
          </Tooltip>
          <Tooltip title="Edit">
            <Button
              type="text"
              size="small"
              icon={<Edit size={16} />}
              onClick={() => handleEditTicket(record)}
            />
          </Tooltip>
          <Tooltip title="View Activity Log">
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
                  label: "Assign Handler",
                  icon: <Users size={16} />,
                },
                {
                  key: "escalate",
                  label: "Escalate Ticket",
                  icon: <TrendingUp size={16} />,
                },
                {
                  type: "divider",
                },
                {
                  key: "delete",
                  label: "Delete Ticket",
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
  ], [handleEditTicket, handleViewActivity]); */

  // UseLoadingEmptyError component handles loading, empty state and error state
  if (error) {
    return (
      <LoadingEmptyError
        state="error"
        error={{
          title: "Loading Failed",
          description: error,
          actionText: "Retry",
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

      {/* Ticket association and merge */}
      <div className="mt-8">
        <TicketAssociation />
      </div>

      {/* Satisfaction analysis dashboard */}
      <div className="mt-8">
        <SatisfactionDashboard />
      </div>

      {/* Create/Edit ticket modal */}
      <Modal
        title={
          <div className="flex items-center gap-3 pb-4 border-b border-gray-100">
            <div className="w-10 h-10 bg-gradient-to-r from-blue-500 to-blue-600 rounded-lg flex items-center justify-center shadow-md">
              {editingTicket ? <Edit size={20} className="text-white" /> : <Plus size={20} className="text-white" />}
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900">
                {editingTicket ? "Edit Ticket" : "Create Ticket"}
              </h3>
              <p className="text-sm text-gray-500">
                {editingTicket ? "Modify ticket information" : "Fill in ticket details"}
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
                label="Title"
                name="title"
                rules={[{ required: true, message: "Please enter ticket title" }]}
              >
                <Input placeholder="Please enter ticket title" size="large" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="Type"
                name="type"
                rules={[{ required: true, message: "Please select ticket type" }]}
              >
                <Select placeholder="Please select ticket type" size="large">
                  <Option value={TicketType.INCIDENT}>Incident</Option>
                  <Option value={TicketType.SERVICE_REQUEST}>Service Request</Option>
                  <Option value={TicketType.PROBLEM}>Problem</Option>
                  <Option value={TicketType.CHANGE}>Change</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                label="Category"
                name="category"
                rules={[{ required: true, message: "Please select ticket category" }]}
              >
                <Select placeholder="Please select ticket category" size="large">
                  <Option value="System Access">System Access</Option>
                  <Option value="Hardware Equipment">Hardware Equipment</Option>
                  <Option value="Software Services">Software Services</Option>
                  <Option value="Network Services">Network Services</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="Priority"
                name="priority"
                rules={[{ required: true, message: "Please select priority" }]}
              >
                <Select placeholder="Please select priority" size="large">
                  <Option value={TicketPriority.LOW}>Low</Option>
                  <Option value={TicketPriority.MEDIUM}>Medium</Option>
                  <Option value={TicketPriority.HIGH}>High</Option>
                  <Option value={TicketPriority.URGENT}>Urgent</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item label="Assignee" name="assignee_id">
                <Select placeholder="Please select assignee" allowClear size="large">
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
              <Form.Item label="Estimated Time" name="estimated_time">
                <DatePicker
                  showTime
                  placeholder="Please select estimated time"
                  size="large"
                  style={{ width: "100%" }}
                />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label="Description"
            name="description"
            rules={[{ required: true, message: "Please input ticket description" }]}
          >
            <Input.TextArea
              rows={6}
              placeholder="Please detailedly describe the content of the ticket, the problem, the expected result, etc..."
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <div className="flex justify-end gap-3 pt-6 border-t border-gray-100">
              <Button 
                onClick={() => setModalVisible(false)} 
                size="large"
                className="rounded-lg border-gray-200 hover:border-gray-300 transition-colors duration-200"
              >
                Cancel
              </Button>
              <Button 
                type="primary" 
                htmlType="submit" 
                size="large"
                className="rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 border-0 shadow-md hover:shadow-lg transition-all duration-200"
                icon={editingTicket ? <Edit size={16} /> : <Plus size={16} />}
              >
                {editingTicket ? "Update Ticket" : "Create Ticket"}
              </Button>
            </div>
          </Form.Item>
        </Form>
      </Modal>

      {/* Ticket template management modal */}
      <Modal
        title={
          <div className="flex items-center gap-3 pb-4 border-b border-gray-100">
            <div className="w-10 h-10 bg-gradient-to-r from-green-500 to-green-600 rounded-lg flex items-center justify-center shadow-md">
              <BookOpen size={20} className="text-white" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900">
                Ticket Template Management
              </h3>
              <p className="text-sm text-gray-500">
                Manage and configure ticket templates to improve work efficiency
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
                Manage all available ticket templates to improve ticket creation efficiency
              </Text>
            </div>
            <Button type="primary" icon={<Plus size={16} />}>
              New Template
            </Button>
          </div>

          <Table
            columns={[
              {
                title: "Template Name",
                dataIndex: "name",
                key: "name",
                render: (name: string) => (
                  <div style={{ fontWeight: "medium", color: "#000" }}>
                    {name}
                  </div>
                ),
              },
              {
                title: "Type",
                dataIndex: "type",
                key: "type",
                render: (type: TicketType) => {
                  const typeConfig: Record<
                    string,
                    { color: string; text: string }
                  > = {
                    [TicketType.INCIDENT]: { color: "red", text: "Incident" },
                    [TicketType.SERVICE_REQUEST]: {
                      color: "blue",
                      text: "Service Request",
                    },
                    [TicketType.PROBLEM]: { color: "orange", text: "Problem" },
                    [TicketType.CHANGE]: { color: "purple", text: "Change" },
                  };
                  const config = typeConfig[type];
                  return <AntdTag color={config.color}>{config.text}</AntdTag>;
                },
              },
              {
                title: "Category",
                dataIndex: "category",
                key: "category",
                render: (category: string) => (
                  <span style={{ color: "#666" }}>{category}</span>
                ),
              },
              {
                title: "Priority",
                dataIndex: "priority",
                key: "priority",
                render: (priority: TicketPriority) => {
                  const priorityConfig: Record<
                    string,
                    { color: string; text: string }
                  > = {
                    [TicketPriority.LOW]: { color: "green", text: "Low" },
                    [TicketPriority.MEDIUM]: { color: "blue", text: "Medium" },
                    [TicketPriority.HIGH]: { color: "orange", text: "High" },
                    [TicketPriority.URGENT]: { color: "red", text: "Urgent" },
                  };
                  const config = priorityConfig[priority];
                  return <AntdTag color={config.color}>{config.text}</AntdTag>;
                },
              },
              {
                title: "Actions",
                key: "actions",
                render: () => (
                  <Space>
                    <Button size="small" icon={<Edit size={14} />}>
                      Edit
                    </Button>
                    <Button size="small" danger>
                      Delete
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
}
