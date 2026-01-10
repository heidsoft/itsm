'use client';

import React, { useMemo, useCallback, useEffect } from 'react';
import { Tag, Avatar, Typography, Space, Tooltip, Badge, Button } from 'antd';
import {
  UserOutlined,
  CalendarOutlined,
  FlagOutlined,
  EyeOutlined,
  EditOutlined,
  DeleteOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useRouter } from 'next/navigation';
import { motion } from 'framer-motion';

import { EnhancedTable, EnhancedTableColumn, EnhancedTableAction } from '../ui/EnhancedTable';
import type { Ticket } from '@/app/lib/api-config';
import { useTicketListStore } from '@/lib/stores/ticket-store';

const { Text } = Typography;

interface OptimizedTicketListProps {
  embedded?: boolean;
  showHeader?: boolean;
  pageSize?: number;
  filters?: Record<string, any>;
  onTicketSelect?: (ticket: Ticket) => void;
  className?: string;
}

// 工单状态配置
const TICKET_STATUS_CONFIG: Record<string, { color: string; text: string; icon?: React.ReactNode }> = {
  new: { color: 'blue', text: '新建', icon: <FlagOutlined /> },
  open: { color: 'processing', text: '待处理', icon: <ClockCircleOutlined /> },
  in_progress: { color: 'orange', text: '处理中', icon: <ClockCircleOutlined spin /> },
  pending: { color: 'warning', text: '待确认', icon: <ClockCircleOutlined /> },
  resolved: { color: 'success', text: '已解决', icon: <FlagOutlined /> },
  closed: { color: 'default', text: '已关闭', icon: <FlagOutlined /> },
  cancelled: { color: 'error', text: '已取消', icon: <FlagOutlined /> },
};

// 优先级配置
const PRIORITY_CONFIG: Record<string, { color: string; text: string; level: number }> = {
  low: { color: 'green', text: '低', level: 1 },
  normal: { color: 'blue', text: '正常', level: 2 },
  high: { color: 'orange', text: '高', level: 3 },
  urgent: { color: 'red', text: '紧急', level: 4 },
  critical: { color: 'magenta', text: '严重', level: 5 },
};

export default function OptimizedTicketList({
  embedded = false,
  showHeader = true,
  pageSize = 20,
  filters: externalFilters,
  onTicketSelect,
  className,
}: OptimizedTicketListProps) {
  const router = useRouter();
  const {
    tickets,
    loading,
    total,
    page,
    pageSize: storePageSize,
    selectedTickets,
    fetchTickets,
    batchDeleteTickets,
    setFilters,
    setPage,
    setPageSize,
    deselectAll,
    selectTicket,
  } = useTicketListStore();

  useEffect(() => {
    if (pageSize !== storePageSize) {
      setPageSize(pageSize);
    }
  }, [pageSize, setPageSize, storePageSize]);

  useEffect(() => {
    if (externalFilters && Object.keys(externalFilters).length > 0) {
      setFilters(externalFilters);
    }
    fetchTickets();
  }, [externalFilters, fetchTickets, setFilters]);

  // 处理搜索
  const handleSearch = useCallback((value: string) => {
    setFilters({ keyword: value });
    setPage(1);
    fetchTickets({ keyword: value, page: 1 });
  }, [fetchTickets, setFilters, setPage]);

  // 处理分页
  const handlePaginationChange = useCallback((page: number, size: number) => {
    setPage(page);
    setPageSize(size);
    fetchTickets({ page, page_size: size, size });
  }, [fetchTickets, setPage, setPageSize]);

  // 处理行选择
  const handleSelectionChange = useCallback((selectedRowKeys: React.Key[], selectedRows: Ticket[]) => {
    deselectAll();
    selectedRows.forEach((row) => selectTicket(row));
  }, [deselectAll, selectTicket]);

  // 表格列配置
  const columns: EnhancedTableColumn<Ticket>[] = useMemo(() => [
    {
      title: '工单号',
      dataIndex: 'ticket_number',
      key: 'ticket_number',
      width: 120,
      sortable: true,
      searchable: true,
      fixed: 'left',
      render: (ticketNumber: string, record: Ticket) => (
        <div className="flex flex-col">
          <Text 
            strong 
            className="text-blue-600 hover:text-blue-800 cursor-pointer transition-colors"
            onClick={() => onTicketSelect?.(record)}
          >
            {ticketNumber}
          </Text>
          <Text type="secondary" className="text-xs">
            {dayjs(record.created_at).format('MM-DD HH:mm')}
          </Text>
        </div>
      ),
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
      searchable: true,
      render: (title: string, record: Ticket) => (
        <div className="flex flex-col max-w-xs">
          <Text 
            className="font-medium cursor-pointer hover:text-blue-600 transition-colors"
            onClick={() => onTicketSelect?.(record)}
          >
            {title}
          </Text>
          {record.description && (
            <Text 
              type="secondary" 
              className="text-xs mt-1 line-clamp-2"
              ellipsis={{ tooltip: record.description }}
            >
              {record.description}
            </Text>
          )}
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      filterable: true,
      render: (status: string) => {
        const config = TICKET_STATUS_CONFIG[status] || { color: 'default', text: status };
        return (
          <motion.div
            initial={{ scale: 0.8, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ duration: 0.2 }}
          >
            <Tag 
              color={config.color} 
              icon={config.icon}
              className="flex items-center gap-1 px-2 py-1"
            >
              {config.text}
            </Tag>
          </motion.div>
        );
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      sortable: true,
      filterable: true,
      render: (priority: string) => {
        const config = PRIORITY_CONFIG[priority] || { color: 'default', text: priority, level: 0 };
        return (
          <Tooltip title={`优先级: ${config.text}`}>
            <Badge 
              color={config.color} 
              count={config.level}
              style={{ backgroundColor: config.color }}
            />
          </Tooltip>
        );
      },
    },
    {
      title: '分配人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 120,
      filterable: true,
      render: (assignee: any) => {
        if (!assignee) {
          return (
            <Text type="secondary" className="italic">
              未分配
            </Text>
          );
        }
        return (
          <div className="flex items-center space-x-2">
            <Avatar 
              size="small" 
              src={assignee.avatar} 
              icon={<UserOutlined />}
              className="flex-shrink-0"
            />
            <Text className="truncate">{assignee.name || assignee.username}</Text>
          </div>
        );
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 140,
      sortable: true,
      render: (createdAt: string) => (
        <div className="flex flex-col">
          <Text>{dayjs(createdAt).format('MM-DD HH:mm')}</Text>
          <Text type="secondary" className="text-xs">
            {dayjs(createdAt).fromNow()}
          </Text>
        </div>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 140,
      sortable: true,
      render: (updatedAt: string) => (
        <div className="flex flex-col">
          <Text>{dayjs(updatedAt).format('MM-DD HH:mm')}</Text>
          <Text type="secondary" className="text-xs">
            {dayjs(updatedAt).fromNow()}
          </Text>
        </div>
      ),
    },
  ], [onTicketSelect]);

  // 操作按钮配置
  const actions: EnhancedTableAction<Ticket>[] = useMemo(() => [
    {
      key: 'view',
      label: '查看',
      icon: <EyeOutlined />,
      onClick: (record) => {
        router.push(`/tickets/${record.id}`);
      },
    },
    {
      key: 'edit',
      label: '编辑',
      icon: <EditOutlined />,
      onClick: (record) => {
        router.push(`/tickets/${record.id}/edit`);
      },
    },
    {
      key: 'delete',
      label: '删除',
      icon: <DeleteOutlined />,
      danger: true,
      onClick: (record) => {
        // 这里可以添加确认对话框
        console.log('Delete ticket:', record.id);
      },
      disabled: (record) => record.status === 'closed',
    },
  ], [router]);

  // 批量操作配置
  const batchActions = useMemo(() => [
    {
      key: 'batchDelete',
      label: '批量删除',
      icon: <DeleteOutlined />,
      danger: true,
      onClick: (selectedRows: Ticket[]) => {
        const ids = selectedRows.map(row => row.id);
        batchDeleteTickets(ids);
      },
      disabled: (selectedRows: Ticket[]) => 
        selectedRows.some(row => row.status === 'closed'),
    },
    {
      key: 'batchAssign',
      label: '批量分配',
      icon: <UserOutlined />,
      onClick: (selectedRows: Ticket[]) => {
        console.log('Batch assign tickets:', selectedRows.map(row => row.id));
      },
    },
  ], [batchDeleteTickets]);

  // 过滤器配置
  const tableFilters = useMemo(() => [
    {
      key: 'status',
      label: '状态',
      options: Object.entries(TICKET_STATUS_CONFIG).map(([key, config]) => ({
        label: config.text,
        value: key,
      })),
      onChange: (values: string[]) => {
        console.log('Status filter changed:', values);
      },
    },
    {
      key: 'priority',
      label: '优先级',
      options: Object.entries(PRIORITY_CONFIG).map(([key, config]) => ({
        label: config.text,
        value: key,
      })),
      onChange: (values: string[]) => {
        console.log('Priority filter changed:', values);
      },
    },
  ], []);

  return (
    <div className={className}>
      <EnhancedTable
        data={tickets}
        loading={loading}
        total={total}
        columns={columns}
        rowKey="id"
        actions={actions}
        batchActions={batchActions}
        pagination={{
          current: page,
          pageSize: storePageSize,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: true,
          onChange: handlePaginationChange,
        }}
        selection={{
          selectedRowKeys: selectedTickets.map(ticket => ticket.id),
          onChange: handleSelectionChange,
        }}
        searchable={{
          placeholder: '搜索工单号、标题、描述...',
          onSearch: handleSearch,
          loading: loading,
        }}
        filters={tableFilters}
        toolbar={{
          title: embedded ? undefined : '工单列表',
          showRefresh: true,
          onRefresh: fetchTickets,
          showExport: true,
          onExport: () => console.log('Export tickets'),
          showSettings: true,
          onSettingsChange: (settings) => console.log('Settings changed:', settings),
          extra: !embedded && (
            <Button 
              type="primary" 
              onClick={() => router.push('/tickets/create')}
              className="bg-blue-600 hover:bg-blue-700"
            >
              创建工单
            </Button>
          ),
        }}
        size="middle"
        sticky={!embedded}
        onRowClick={onTicketSelect}
        scroll={{ x: 1200 }}
      />
    </div>
  );
}
