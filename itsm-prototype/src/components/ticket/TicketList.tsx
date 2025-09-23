'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Table,
  Card,
  Button,
  Input,
  Select,
  Space,
  Tag,
  Avatar,
  Tooltip,
  Dropdown,
  Modal,
  message,
  Badge,
  DatePicker,
  Row,
  Col,
  Divider,
} from 'antd';
import {
  FilterOutlined,
  PlusOutlined,
  ReloadOutlined,
  ExportOutlined,
  EyeOutlined,
  EditOutlined,
  DeleteOutlined,
  MoreOutlined,
  UserOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TableProps } from 'antd/es/table';
import type { MenuProps } from 'antd';
import dayjs from 'dayjs';
import { useRouter } from 'next/navigation';

import type { 
  Ticket, 
  TicketStatus, 
  TicketPriority, 
  TicketType,
  TicketFilters
} from '@/types/ticket';
import { TicketAPI } from '@/lib/api/ticket-api';
import { useTicketListStore } from '@/lib/stores/ticket-store';

const { Search } = Input;
const { Option } = Select;
const { RangePicker } = DatePicker;

interface TicketListProps {
  embedded?: boolean;
  showHeader?: boolean;
  pageSize?: number;
  filters?: Partial<TicketFilters>;
  onTicketSelect?: (ticket: Ticket) => void;
}

const TicketList: React.FC<TicketListProps> = ({
  embedded = false,
  showHeader = true,
  filters: initialFilters = {},
  onTicketSelect,
}) => {
  const router = useRouter();
  const ticketApi = new TicketAPI();
  const {
    tickets,
    loading,
    pagination,
    filters,
    setTickets,
    setLoading,
    setPagination,
    setFilters,
  } = useTicketListStore();

  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [searchText, setSearchText] = useState('');
  const [showFilters, setShowFilters] = useState(false);

  // 获取工单列表
  const fetchTickets = useCallback(async (params?: {
    page?: number;
    pageSize?: number;
    filters?: TicketFilters;
  }) => {
    try {
      setLoading(true);
      const response = await ticketApi.getTickets({
        page: params?.page || pagination.current,
        pageSize: params?.pageSize || pagination.pageSize,
        ...filters,
        ...initialFilters,
        ...params?.filters,
      });
      
      setTickets(response.tickets);
      setPagination({
        current: response.page,
        pageSize: response.pageSize,
        total: response.total,
      });
    } catch (error) {
      message.error('获取工单列表失败');
      console.error('Failed to fetch tickets:', error);
    } finally {
      setLoading(false);
    }
  }, [pagination, filters, initialFilters, setTickets, setLoading, setPagination]);

  // 初始化加载
  useEffect(() => {
    fetchTickets();
  }, []);

  // 搜索处理
  const handleSearch = (value: string) => {
    setSearchText(value);
    setFilters({ ...filters, search: value });
    fetchTickets({ page: 1, filters: { ...filters, search: value } });
  };

  // 筛选处理
  const handleFilterChange = (key: keyof TicketFilters, value: string | string[] | undefined) => {
    const newFilters = { ...filters, [key]: value };
    setFilters(newFilters);
    fetchTickets({ page: 1, filters: newFilters });
  };

  // 清除筛选
  const handleClearFilters = () => {
    setFilters({});
    setSearchText('');
    fetchTickets({ page: 1, filters: {} });
  };

  // 表格分页处理
  const handleTableChange: TableProps<Ticket>['onChange'] = (paginationInfo) => {
    fetchTickets({
      page: paginationInfo.current,
      pageSize: paginationInfo.pageSize,
    });
  };

  // 行选择处理
  const rowSelection = {
    selectedRowKeys,
    onChange: (newSelectedRowKeys: React.Key[]) => {
      setSelectedRowKeys(newSelectedRowKeys);
    },
    onSelectAll: (selected: boolean, selectedRows: Ticket[], changeRows: Ticket[]) => {
      console.log('Select all:', selected, selectedRows, changeRows);
    },
  };

  // 操作菜单
  const getActionMenu = (record: Ticket): MenuProps => ({
    items: [
      {
        key: 'view',
        icon: <EyeOutlined />,
        label: '查看详情',
        onClick: () => handleViewTicket(record),
      },
      {
        key: 'edit',
        icon: <EditOutlined />,
        label: '编辑工单',
        onClick: () => handleEditTicket(record),
      },
      {
        type: 'divider',
      },
      {
        key: 'assign',
        label: '分配工单',
        onClick: () => handleAssignTicket(record),
      },
      {
        key: 'status',
        label: '更新状态',
        children: [
          {
            key: 'in_progress',
            label: '处理中',
            onClick: () => handleUpdateStatus(record, 'in_progress'),
          },
          {
            key: 'resolved',
            label: '已解决',
            onClick: () => handleUpdateStatus(record, 'resolved'),
          },
          {
            key: 'closed',
            label: '已关闭',
            onClick: () => handleUpdateStatus(record, 'closed'),
          },
        ],
      },
      {
        type: 'divider',
      },
      {
        key: 'delete',
        icon: <DeleteOutlined />,
        label: '删除工单',
        danger: true,
        onClick: () => handleDeleteTicket(record),
      },
    ],
  });

  // 操作处理函数
  const handleViewTicket = (ticket: Ticket) => {
    if (onTicketSelect) {
      onTicketSelect(ticket);
    } else {
      router.push(`/tickets/${ticket.id}`);
    }
  };

  const handleEditTicket = (ticket: Ticket) => {
    router.push(`/tickets/${ticket.id}/edit`);
  };

  const handleAssignTicket = (_ticket: Ticket) => {
    // TODO: 实现分配工单逻辑
    message.info('分配工单功能开发中');
  };

  const handleUpdateStatus = async (ticket: Ticket, status: TicketStatus) => {
    try {
      await ticketApi.updateTicket(ticket.id, { status });
      message.success('状态更新成功');
      fetchTickets();
    } catch (_error) {
      message.error('状态更新失败');
    }
  };

  const handleDeleteTicket = (ticket: Ticket) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除工单 #${ticket.ticketNumber} 吗？`,
      icon: <ExclamationCircleOutlined />,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await ticketApi.deleteTicket(ticket.id);
          message.success('工单删除成功');
          fetchTickets();
        } catch (_error) {
          message.error('工单删除失败');
        }
      },
    });
  };

  // 批量操作
  const handleBatchAction = (action: string) => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要操作的工单');
      return;
    }
    
    // TODO: 实现批量操作逻辑
    message.info(`批量${action}功能开发中`);
  };

  // 导出数据
  const handleExport = () => {
    // TODO: 实现导出功能
    message.info('导出功能开发中');
  };

  // 状态标签渲染
  const renderStatusTag = (status: TicketStatus) => {
    const statusConfig = {
      open: { color: 'blue', text: '待处理' },
      in_progress: { color: 'orange', text: '处理中' },
      resolved: { color: 'green', text: '已解决' },
      closed: { color: 'default', text: '已关闭' },
      cancelled: { color: 'red', text: '已取消' },
    };
    
    const config = statusConfig[status];
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  // 优先级标签渲染
  const renderPriorityTag = (priority: TicketPriority) => {
    const priorityConfig = {
      low: { color: 'green', text: '低' },
      medium: { color: 'blue', text: '中' },
      high: { color: 'orange', text: '高' },
      urgent: { color: 'red', text: '紧急' },
      critical: { color: 'magenta', text: '严重' },
    };
    
    const config = priorityConfig[priority];
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  // 表格列定义
  const columns: ColumnsType<Ticket> = [
    {
      title: '工单号',
      dataIndex: 'ticketNumber',
      key: 'ticketNumber',
      width: 120,
      fixed: 'left',
      render: (text, record) => (
        <Button
          type="link"
          onClick={() => handleViewTicket(record)}
          style={{ padding: 0, height: 'auto' }}
        >
          #{text}
        </Button>
      ),
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: {
        showTitle: false,
      },
      render: (text) => (
        <Tooltip title={text}>
          <span>{text}</span>
        </Tooltip>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: renderStatusTag,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: renderPriorityTag,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: TicketType) => {
        const typeMap = {
          incident: '事件',
          request: '请求',
          problem: '问题',
          change: '变更',
        };
        return <Tag>{typeMap[type]}</Tag>;
      },
    },
    {
      title: '提交人',
      dataIndex: 'requester',
      key: 'requester',
      width: 120,
      render: (requester) => (
        <Space>
          <Avatar size="small" icon={<UserOutlined />} src={requester?.avatar} />
          <span>{requester?.fullName}</span>
        </Space>
      ),
    },
    {
      title: '分配人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 120,
      render: (assignee) => assignee ? (
        <Space>
          <Avatar size="small" icon={<UserOutlined />} src={assignee?.avatar} />
          <span>{assignee?.fullName}</span>
        </Space>
      ) : (
        <span style={{ color: '#999' }}>未分配</span>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (text) => (
        <Tooltip title={dayjs(text).format('YYYY-MM-DD HH:mm:ss')}>
          <Space>
            <ClockCircleOutlined />
            {dayjs(text).format('MM-DD HH:mm')}
          </Space>
        </Tooltip>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      fixed: 'right',
      render: (_, record) => (
        <Dropdown menu={getActionMenu(record)} trigger={['click']}>
          <Button type="text" icon={<MoreOutlined />} />
        </Dropdown>
      ),
    },
  ];

  // 筛选器组件
  const renderFilters = () => (
    <Card size="small" style={{ marginBottom: 16 }}>
      <Row gutter={[16, 16]}>
        <Col span={6}>
          <Select
            placeholder="状态"
            allowClear
            style={{ width: '100%' }}
            value={filters.status}
            onChange={(value) => handleFilterChange('status', value)}
          >
            <Option value="open">待处理</Option>
            <Option value="in_progress">处理中</Option>
            <Option value="resolved">已解决</Option>
            <Option value="closed">已关闭</Option>
            <Option value="cancelled">已取消</Option>
          </Select>
        </Col>
        <Col span={6}>
          <Select
            placeholder="优先级"
            allowClear
            style={{ width: '100%' }}
            value={filters.priority}
            onChange={(value) => handleFilterChange('priority', value)}
          >
            <Option value="low">低</Option>
            <Option value="medium">中</Option>
            <Option value="high">高</Option>
            <Option value="urgent">紧急</Option>
            <Option value="critical">严重</Option>
          </Select>
        </Col>
        <Col span={6}>
          <Select
            placeholder="类型"
            allowClear
            style={{ width: '100%' }}
            value={filters.type}
            onChange={(value) => handleFilterChange('type', value)}
          >
            <Option value="incident">事件</Option>
            <Option value="request">请求</Option>
            <Option value="problem">问题</Option>
            <Option value="change">变更</Option>
          </Select>
        </Col>
        <Col span={6}>
          <RangePicker
            style={{ width: '100%' }}
            placeholder={['开始日期', '结束日期']}
            onChange={(dates) => {
              if (dates) {
                handleFilterChange('createdAfter', dates[0]?.toISOString());
                handleFilterChange('createdBefore', dates[1]?.toISOString());
              } else {
                handleFilterChange('createdAfter', undefined);
                handleFilterChange('createdBefore', undefined);
              }
            }}
          />
        </Col>
      </Row>
      <Row style={{ marginTop: 16 }}>
        <Col>
          <Button onClick={handleClearFilters}>清除筛选</Button>
        </Col>
      </Row>
    </Card>
  );

  return (
    <div>
      {showHeader && (
        <Card>
          <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
            <Col>
              <Space>
                <Search
                  placeholder="搜索工单..."
                  allowClear
                  style={{ width: 300 }}
                  value={searchText}
                  onSearch={handleSearch}
                  onChange={(e) => setSearchText(e.target.value)}
                />
                <Button
                  icon={<FilterOutlined />}
                  onClick={() => setShowFilters(!showFilters)}
                >
                  筛选
                </Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => fetchTickets()}
                >
                  刷新
                </Button>
              </Space>
            </Col>
            <Col>
              <Space>
                {selectedRowKeys.length > 0 && (
                  <>
                    <Badge count={selectedRowKeys.length}>
                      <Button onClick={() => handleBatchAction('分配')}>
                        批量分配
                      </Button>
                    </Badge>
                    <Button onClick={() => handleBatchAction('关闭')}>
                      批量关闭
                    </Button>
                    <Divider type="vertical" />
                  </>
                )}
                <Button
                  icon={<ExportOutlined />}
                  onClick={handleExport}
                >
                  导出
                </Button>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => router.push('/tickets/new')}
                >
                  新建工单
                </Button>
              </Space>
            </Col>
          </Row>

          {showFilters && renderFilters()}
        </Card>
      )}

      <Card>
        <Table
          rowSelection={embedded ? undefined : rowSelection}
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
          }}
          onChange={handleTableChange}
          scroll={{ x: 1200 }}
          size={embedded ? 'small' : 'middle'}
        />
      </Card>
    </div>
  );
};

export default TicketList;