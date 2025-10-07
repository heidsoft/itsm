'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Table,
  Card,
  Button,
  Input,
  Select,
  Space,
  Tag,
  Tooltip,
  Dropdown,
  Modal,
  message,
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
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TableProps } from 'antd/es/table';
import type { MenuProps } from 'antd';
import dayjs from 'dayjs';
import { useRouter } from 'next/navigation';

import { TicketAPI, Ticket, ListTicketsRequest } from '@/lib/api/ticket-api';
import { useTicketListStore } from '@/lib/stores/ticket-store';

const { Search } = Input;
const { Option } = Select;
const { RangePicker } = DatePicker;

interface TicketListProps {
  embedded?: boolean;
  showHeader?: boolean;
  pageSize?: number;
  filters?: Partial<ListTicketsRequest>;
  onTicketSelect?: (ticket: Ticket) => void;
}

// 工单状态配置
const TICKET_STATUS_CONFIG: Record<string, { color: string; text: string }> = {
  new: { color: 'blue', text: '新建' },
  open: { color: 'blue', text: '待处理' },
  in_progress: { color: 'orange', text: '处理中' },
  pending: { color: 'yellow', text: '等待中' },
  resolved: { color: 'green', text: '已解决' },
  closed: { color: 'default', text: '已关闭' },
  cancelled: { color: 'red', text: '已取消' },
};

// 优先级配置
const PRIORITY_CONFIG: Record<string, { color: string; text: string }> = {
  low: { color: 'green', text: '低' },
  medium: { color: 'orange', text: '中' },
  high: { color: 'red', text: '高' },
  urgent: { color: 'purple', text: '紧急' },
  critical: { color: 'purple', text: '紧急' },
};

// 工单类型配置
const TICKET_TYPE_CONFIG: Record<string, string> = {
  incident: '事件',
  request: '请求',
  problem: '问题',
  change: '变更',
  task: '任务',
};

const TicketList: React.FC<TicketListProps> = ({
  embedded = false,
  showHeader = true,
  filters: initialFilters = {},
  onTicketSelect,
}) => {
  const router = useRouter();
  const {
    tickets,
    loading,
    total,
    page,
    pageSize,
    filters,
    selectedTickets,
    fetchTickets,
    updateFilter,
    clearFilters,
    setPage,
    setPageSize,
    selectTicket,
    deselectTicket,
    deselectAll,
    deleteTicket,
    batchDeleteTickets,
  } = useTicketListStore();

  // 本地状态
  const [searchValue, setSearchValue] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [ticketToDelete, setTicketToDelete] = useState<Ticket | null>(null);

  // 初始化加载
  useEffect(() => {
    const mergedFilters = { ...initialFilters, ...filters };
    fetchTickets(mergedFilters);
  }, [fetchTickets, initialFilters, filters]);

  // 搜索处理
  const handleSearch = useCallback((value: string) => {
    setSearchValue(value);
    updateFilter('keyword', value || undefined);
    fetchTickets({ ...filters, keyword: value || undefined, page: 1 });
  }, [filters, updateFilter, fetchTickets]);

  // 过滤器变更处理
  const handleFilterChange = useCallback((key: keyof ListTicketsRequest, value: unknown) => {
    updateFilter(key, value);
    const newFilters = { ...filters, [key]: value, page: 1 };
    fetchTickets(newFilters);
  }, [filters, updateFilter, fetchTickets]);

  // 分页变更处理
  const handleTableChange: TableProps<Ticket>['onChange'] = useCallback((pagination) => {
    const newPage = pagination.current || 1;
    const newPageSize = pagination.pageSize || 20;
    
    setPage(newPage);
    setPageSize(newPageSize);
    
    fetchTickets({
      ...filters,
      page: newPage,
      page_size: newPageSize,
    });
  }, [filters, setPage, setPageSize, fetchTickets]);

  // 刷新数据
  const handleRefresh = useCallback(() => {
    fetchTickets(filters);
  }, [fetchTickets, filters]);

  // 清空过滤器
  const handleClearFilters = useCallback(() => {
    setSearchValue('');
    clearFilters();
    fetchTickets({});
  }, [clearFilters, fetchTickets]);

  // 删除工单
  const handleDelete = useCallback(async (ticket: Ticket) => {
    try {
      await deleteTicket(ticket.id);
      message.success('删除成功');
      setDeleteModalVisible(false);
      setTicketToDelete(null);
    } catch (error) {
      console.error('Delete ticket error:', error);
      message.error('删除失败');
    }
  }, [deleteTicket]);

  // 批量删除
  const handleBatchDelete = useCallback(async () => {
    if (selectedTickets.length === 0) {
      message.warning('请选择要删除的工单');
      return;
    }

    try {
      const ids = selectedTickets.map(ticket => ticket.id);
      await batchDeleteTickets(ids);
      message.success(`成功删除 ${selectedTickets.length} 个工单`);
      deselectAll();
    } catch (error) {
      console.error('Batch delete error:', error);
      message.error('批量删除失败');
    }
  }, [selectedTickets, batchDeleteTickets, deselectAll]);

  // 导出数据
  const handleExport = useCallback(async () => {
    try {
      const blob = await TicketAPI.exportTickets(filters);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `tickets_${dayjs().format('YYYY-MM-DD')}.xlsx`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      message.success('导出成功');
    } catch (error) {
      console.error('Export error:', error);
      message.error('导出失败');
    }
  }, [filters]);

  // 日期范围变更处理
  const handleDateRangeChange = useCallback((dates: [dayjs.Dayjs | null, dayjs.Dayjs | null] | null, field: 'created' | 'due') => {
    if (dates && dates[0] && dates[1]) {
      const startKey = field === 'created' ? 'created_after' : 'due_after';
      const endKey = field === 'created' ? 'created_before' : 'due_before';
      
      updateFilter(startKey as keyof ListTicketsRequest, dates[0].format('YYYY-MM-DD'));
      updateFilter(endKey as keyof ListTicketsRequest, dates[1].format('YYYY-MM-DD'));
      
      fetchTickets({
        ...filters,
        [startKey]: dates[0].format('YYYY-MM-DD'),
        [endKey]: dates[1].format('YYYY-MM-DD'),
        page: 1,
      });
    } else {
      const startKey = field === 'created' ? 'created_after' : 'due_after';
      const endKey = field === 'created' ? 'created_before' : 'due_before';
      
      updateFilter(startKey as keyof ListTicketsRequest, undefined);
      updateFilter(endKey as keyof ListTicketsRequest, undefined);
      
      const newFilters = { ...filters };
      delete newFilters[startKey as keyof ListTicketsRequest];
      delete newFilters[endKey as keyof ListTicketsRequest];
      
      fetchTickets({ ...newFilters, page: 1 });
    }
  }, [filters, updateFilter, fetchTickets]);

  // 表格列定义
  const columns: ColumnsType<Ticket> = useMemo(() => [
    {
      title: '工单号',
      dataIndex: 'ticket_number',
      key: 'ticket_number',
      width: 120,
      fixed: 'left',
      render: (ticketNumber: string, record: Ticket) => (
        <Button
          type="link"
          onClick={() => {
            if (onTicketSelect) {
              onTicketSelect(record);
            } else {
              router.push(`/tickets/${record.id}`);
            }
          }}
        >
          {ticketNumber}
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
      render: (title: string) => (
        <Tooltip placement="topLeft" title={title}>
          {title}
        </Tooltip>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const config = TICKET_STATUS_CONFIG[status] || { color: 'default', text: status };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: string) => {
        const config = PRIORITY_CONFIG[priority] || { color: 'default', text: priority };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => {
        const typeName = TICKET_TYPE_CONFIG[type] || type;
        return <Tag>{typeName}</Tag>;
      },
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: 100,
      render: (source: string) => <Tag color="blue">{source}</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (createdAt: string) => dayjs(createdAt).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 160,
      render: (updatedAt: string) => dayjs(updatedAt).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      fixed: 'right',
      render: (_, record: Ticket) => {
        const items: MenuProps['items'] = [
          {
            key: 'view',
            icon: <EyeOutlined />,
            label: '查看详情',
            onClick: () => router.push(`/tickets/${record.id}`),
          },
          {
            key: 'edit',
            icon: <EditOutlined />,
            label: '编辑',
            onClick: () => router.push(`/tickets/${record.id}/edit`),
          },
          {
            type: 'divider',
          },
          {
            key: 'delete',
            icon: <DeleteOutlined />,
            label: '删除',
            danger: true,
            onClick: () => {
              setTicketToDelete(record);
              setDeleteModalVisible(true);
            },
          },
        ];

        return (
          <Dropdown menu={{ items }} trigger={['click']}>
            <Button type="text" icon={<MoreOutlined />} />
          </Dropdown>
        );
      },
    },
  ], [onTicketSelect, router]);

  // 行选择配置
  const rowSelection: TableProps<Ticket>['rowSelection'] = useMemo(() => ({
    selectedRowKeys: selectedTickets.map(ticket => ticket.id),
    onChange: (selectedRowKeys: React.Key[], selectedRows: Ticket[]) => {
      deselectAll();
      selectedRows.forEach(ticket => selectTicket(ticket));
    },
    onSelectAll: (selected: boolean, selectedRows: Ticket[], changeRows: Ticket[]) => {
      if (selected) {
        changeRows.forEach(ticket => selectTicket(ticket));
      } else {
        changeRows.forEach(ticket => deselectTicket(ticket));
      }
    },
  }), [selectedTickets, selectTicket, deselectTicket, deselectAll]);

  return (
    <div className="ticket-list">
      {showHeader && (
        <Card>
          <Row gutter={[16, 16]} align="middle">
            <Col flex="auto">
              <Space size="middle">
                <Search
                  placeholder="搜索工单标题、描述或工单号"
                  value={searchValue}
                  onChange={(e) => setSearchValue(e.target.value)}
                  onSearch={handleSearch}
                  style={{ width: 300 }}
                  allowClear
                />
                <Button
                  icon={<FilterOutlined />}
                  onClick={() => setShowFilters(!showFilters)}
                >
                  过滤器
                </Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={handleRefresh}
                  loading={loading}
                >
                  刷新
                </Button>
              </Space>
            </Col>
            <Col>
              <Space>
                {selectedTickets.length > 0 && (
                  <Button
                    danger
                    icon={<DeleteOutlined />}
                    onClick={handleBatchDelete}
                  >
                    批量删除 ({selectedTickets.length})
                  </Button>
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
                  onClick={() => router.push('/tickets/create')}
                >
                  创建工单
                </Button>
              </Space>
            </Col>
          </Row>

          {showFilters && (
            <>
              <Divider />
              <Row gutter={[16, 16]}>
                <Col span={6}>
                  <Select
                    placeholder="状态"
                    value={filters.status}
                    onChange={(value) => handleFilterChange('status', value)}
                    allowClear
                    style={{ width: '100%' }}
                  >
                    {Object.entries(TICKET_STATUS_CONFIG).map(([key, config]) => (
                      <Option key={key} value={key}>
                        <Tag color={config.color}>{config.text}</Tag>
                      </Option>
                    ))}
                  </Select>
                </Col>
                <Col span={6}>
                  <Select
                    placeholder="优先级"
                    value={filters.priority}
                    onChange={(value) => handleFilterChange('priority', value)}
                    allowClear
                    style={{ width: '100%' }}
                  >
                    {Object.entries(PRIORITY_CONFIG).map(([key, config]) => (
                      <Option key={key} value={key}>
                        <Tag color={config.color}>{config.text}</Tag>
                      </Option>
                    ))}
                  </Select>
                </Col>
                <Col span={6}>
                  <Select
                    placeholder="类型"
                    value={filters.type}
                    onChange={(value) => handleFilterChange('type', value)}
                    allowClear
                    style={{ width: '100%' }}
                  >
                    {Object.entries(TICKET_TYPE_CONFIG).map(([key, text]) => (
                      <Option key={key} value={key}>
                        {text}
                      </Option>
                    ))}
                  </Select>
                </Col>
                <Col span={6}>
                  <RangePicker
                    placeholder={['开始日期', '结束日期']}
                    onChange={(dates) => handleDateRangeChange(dates, 'created')}
                    style={{ width: '100%' }}
                  />
                </Col>
              </Row>
              <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
                <Col>
                  <Button onClick={handleClearFilters}>
                    清空过滤器
                  </Button>
                </Col>
              </Row>
            </>
          )}
        </Card>
      )}

      <Card style={{ marginTop: showHeader ? 16 : 0 }}>
        <Table<Ticket>
          columns={columns}
          dataSource={tickets}
          rowKey="id"
          rowSelection={embedded ? undefined : rowSelection}
          loading={loading}
          pagination={{
            current: page,
            pageSize: pageSize,
            total: total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
            pageSizeOptions: ['10', '20', '50', '100'],
          }}
          onChange={handleTableChange}
          scroll={{ x: 1200 }}
          size="middle"
        />
      </Card>

      {/* 删除确认对话框 */}
      <Modal
        title="确认删除"
        open={deleteModalVisible}
        onOk={() => ticketToDelete && handleDelete(ticketToDelete)}
        onCancel={() => {
          setDeleteModalVisible(false);
          setTicketToDelete(null);
        }}
        okText="确认"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <p>
          <ExclamationCircleOutlined style={{ color: '#ff4d4f', marginRight: 8 }} />
          确定要删除工单 <strong>{ticketToDelete?.ticket_number}</strong> 吗？此操作不可撤销。
        </p>
      </Modal>
    </div>
  );
};

export default TicketList;