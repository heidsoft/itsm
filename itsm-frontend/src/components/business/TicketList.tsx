'use client';
// @ts-nocheck

import React, { useState, useEffect, useMemo, useCallback } from 'react';
import {
  Table,
  Card,
  Button,
  Space,
  Tag,
  Input,
  Select,
  DatePicker,
  Modal,
  Tooltip,
  Dropdown,
  Menu,
  App,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  FilterOutlined,
  DownloadOutlined,
  MoreOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  UserOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import {
  ticketService,
  type Ticket,
  type TicketFilterParams,
} from '../../lib/services/ticket-service';
import { useAuthStore } from '@/lib/store/auth-store';
import type { PaginatedResponse } from '../../lib/services/api-service';

const { RangePicker } = DatePicker;
const { Option } = Select;

interface TicketListProps {
  onTicketSelect?: (ticket: Ticket) => void;
  onRefresh?: () => void;
}

export const TicketList: React.FC<TicketListProps> = ({ onTicketSelect, onRefresh }) => {
  const { message } = App.useApp();
  const { currentTenant } = useAuthStore();
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState<TicketFilterParams>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
  const [searchText, setSearchText] = useState('');

  // 获取工单列表
  const fetchTickets = useCallback(async () => {
    if (!currentTenant?.id) return;

    setLoading(true);
    try {
      const params: TicketFilterParams = {
        page: currentPage,
        size: pageSize,
        ...filters,
        search: searchText || undefined,
      };

      const response: PaginatedResponse<Ticket> = await ticketService.getTickets(params);
      setTickets(response.data);
      setTotal(response.total);
    } catch (error) {
      message.error('获取工单列表失败');
      console.error('获取工单列表失败:', error);
    } finally {
      setLoading(false);
    }
  }, [currentTenant?.id, currentPage, pageSize, filters, searchText]);

  // 监听参数变化，重新获取数据
  useEffect(() => {
    fetchTickets();
  }, [fetchTickets]);

  // 处理搜索
  const handleSearch = useCallback((value: string) => {
    setSearchText(value);
    setCurrentPage(1);
  }, []);

  // 处理筛选
  const handleFilterChange = useCallback((newFilters: Partial<TicketFilterParams>) => {
    setFilters(prev => ({ ...prev, ...newFilters }));
    setCurrentPage(1);
  }, []);

  // 处理分页
  const handleTableChange = useCallback((pagination: any) => {
    setCurrentPage(pagination.current);
    setPageSize(pagination.pageSize);
  }, []);

  // 获取状态颜色
  const getStatusColor = useCallback((status: string) => {
    switch (status) {
      case 'open':
        return 'blue';
      case 'in_progress':
        return 'orange';
      case 'pending':
        return 'yellow';
      case 'resolved':
        return 'green';
      case 'closed':
        return 'default';
      case 'cancelled':
        return 'red';
      default:
        return 'default';
    }
  }, []);

  // 获取优先级颜色
  const getPriorityColor = useCallback((priority: string) => {
    switch (priority) {
      case 'critical':
        return 'red';
      case 'high':
        return 'orange';
      case 'medium':
        return 'yellow';
      case 'low':
        return 'green';
      default:
        return 'default';
    }
  }, []);

  // 获取状态文本
  const getStatusText = useCallback((status: string) => {
    const statusMap: Record<string, string> = {
      open: '待处理',
      in_progress: '处理中',
      pending: '待确认',
      resolved: '已解决',
      closed: '已关闭',
      cancelled: '已取消',
    };
    return statusMap[status] || status;
  }, []);

  // 获取优先级文本
  const getPriorityText = useCallback((priority: string) => {
    const priorityMap: Record<string, string> = {
      critical: '紧急',
      high: '高',
      medium: '中',
      low: '低',
    };
    return priorityMap[priority] || priority;
  }, []);

  // 批量操作
  const handleBatchAction = useCallback(
    async (action: string) => {
      if (selectedRowKeys.length === 0) {
        message.warning('请选择要操作的工单');
        return;
      }

      try {
        switch (action) {
          case 'delete':
            Modal.confirm({
              title: '确认删除',
              content: `确定要删除选中的 ${selectedRowKeys.length} 个工单吗？`,
              onOk: async () => {
                await Promise.all(selectedRowKeys.map(id => ticketService.deleteTicket(id)));
                message.success('删除成功');
                setSelectedRowKeys([]);
                fetchTickets();
              },
            });
            break;
          case 'export':
            const blob = await ticketService.exportTickets({
              ...filters,
              search: searchText || undefined,
            });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `tickets_${new Date().toISOString().split('T')[0]}.csv`;
            a.click();
            window.URL.revokeObjectURL(url);
            message.success('导出成功');
            break;
        }
      } catch (error) {
        message.error('操作失败');
        console.error('批量操作失败:', error);
      }
    },
    [selectedRowKeys, filters, searchText, fetchTickets]
  );

  // 表格列定义
  const columns = useMemo(
    () => [
      {
        title: '工单号',
        dataIndex: 'ticket_number',
        key: 'ticket_number',
        width: 120,
        render: (text: string) => <span className='font-mono text-sm'>{text}</span>,
      },
      {
        title: '标题',
        dataIndex: 'title',
        key: 'title',
        render: (text: string, record: Ticket) => (
          <Tooltip title={text}>
            <Button
              type='link'
              className='p-0 h-auto text-left truncate'
              style={{ maxWidth: '100%' }}
              onClick={() => onTicketSelect?.(record)}
            >
              <span className='truncate'>{text}</span>
            </Button>
          </Tooltip>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (status: string) => (
          <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>
        ),
      },
      {
        title: '优先级',
        dataIndex: 'priority',
        key: 'priority',
        width: 80,
        render: (priority: string) => (
          <Tag color={getPriorityColor(priority)}>{getPriorityText(priority)}</Tag>
        ),
      },
      {
        title: '分类',
        dataIndex: 'category',
        key: 'category',
        width: 100,
      },
      {
        title: '处理人',
        dataIndex: 'assignee_name',
        key: 'assignee_name',
        width: 100,
        render: (text: string) => (
          <span>
            <UserOutlined className='mr-1' />
            {text || '未分配'}
          </span>
        ),
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        key: 'created_at',
        width: 150,
        render: (text: string) => (
          <span>
            <ClockCircleOutlined className='mr-1' />
            {new Date(text).toLocaleString()}
          </span>
        ),
      },
      {
        title: '操作',
        key: 'action',
        width: 120,
        render: (_, record: Ticket) => (
          <Dropdown
            overlay={
              <Menu>
                <Menu.Item
                  key='view'
                  icon={<EyeOutlined />}
                  onClick={() => onTicketSelect?.(record)}
                >
                  查看详情
                </Menu.Item>
                <Menu.Item
                  key='edit'
                  icon={<EditOutlined />}
                  onClick={() => onTicketSelect?.(record)}
                >
                  编辑
                </Menu.Item>
                <Menu.Item
                  key='delete'
                  icon={<DeleteOutlined />}
                  danger
                  onClick={() => {
                    Modal.confirm({
                      title: '确认删除',
                      content: '确定要删除这个工单吗？',
                      onOk: async () => {
                        try {
                          await ticketService.deleteTicket(record.id);
                          message.success('删除成功');
                          fetchTickets();
                        } catch (error) {
                          message.error('删除失败');
                        }
                      },
                    });
                  }}
                >
                  删除
                </Menu.Item>
              </Menu>
            }
          >
            <Button type='text' icon={<MoreOutlined />} />
          </Dropdown>
        ),
      },
    ],
    [getStatusColor, getPriorityColor, getStatusText, getPriorityText, onTicketSelect, fetchTickets]
  );

  // 行选择配置
  const rowSelection = {
    selectedRowKeys,
    onChange: (keys: React.Key[]) => setSelectedRowKeys(keys as string[]),
  };

  return (
    <Card
      title='工单列表'
      extra={
        <Space>
          <Button
            type='primary'
            icon={<PlusOutlined />}
            onClick={() => onTicketSelect?.({} as Ticket)}
          >
            新建工单
          </Button>
          {selectedRowKeys.length > 0 && (
            <>
              <Button icon={<DownloadOutlined />} onClick={() => handleBatchAction('export')}>
                导出选中
              </Button>
              <Button danger icon={<DeleteOutlined />} onClick={() => handleBatchAction('delete')}>
                删除选中
              </Button>
            </>
          )}
        </Space>
      }
    >
      {/* 搜索和筛选区域 */}
      <div className='mb-4 space-y-4'>
        <div className='flex items-center space-x-4'>
          <Input.Search
            placeholder='搜索工单标题或描述'
            allowClear
            style={{ width: 300 }}
            onSearch={handleSearch}
            prefix={<SearchOutlined />}
          />
          <Select
            placeholder='状态'
            allowClear
            style={{ width: 120 }}
            onChange={value => handleFilterChange({ status: value })}
          >
            <Option value='open'>待处理</Option>
            <Option value='in_progress'>处理中</Option>
            <Option value='pending'>待确认</Option>
            <Option value='resolved'>已解决</Option>
            <Option value='closed'>已关闭</Option>
            <Option value='cancelled'>已取消</Option>
          </Select>
          <Select
            placeholder='优先级'
            allowClear
            style={{ width: 120 }}
            onChange={value => handleFilterChange({ priority: value })}
          >
            <Option value='critical'>紧急</Option>
            <Option value='high'>高</Option>
            <Option value='medium'>中</Option>
            <Option value='low'>低</Option>
          </Select>
          <RangePicker
            placeholder={['开始日期', '结束日期']}
            onChange={dates => {
              if (dates) {
                handleFilterChange({
                  created_after: dates[0]?.toISOString(),
                  created_before: dates[1]?.toISOString(),
                });
              }
            }}
          />
        </div>
      </div>

      {/* 表格 */}
      <Table
        columns={columns}
        dataSource={tickets}
        rowKey='id'
        loading={loading}
        pagination={{
          current: currentPage,
          pageSize: pageSize,
          total: total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
        }}
        rowSelection={rowSelection}
        onChange={handleTableChange}
        scroll={{ x: 1200 }}
      />
    </Card>
  );
};
