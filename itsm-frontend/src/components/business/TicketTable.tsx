'use client';

import React, { useCallback, useState } from 'react';
import { Button, Space, Badge, Table } from 'antd';
import { AlertTriangle } from 'lucide-react';
import {
  Ticket,
  TicketStatus,
  TicketPriority,
  TicketType,
} from '../../lib/services/ticket-service';
import { VirtualizedTicketList } from './VirtualizedTicketList';
import { LazyWrapper, LoadingSpinner } from '../common/LazyComponents';
import { TicketListSkeleton } from './TicketListSkeleton';
import { LoadingEmptyError } from '../ui/LoadingEmptyError';

interface TicketTableProps {
  tickets: Ticket[];
  loading: boolean;
  pagination: {
    current: number;
    pageSize: number;
    total: number;
  };
  selectedRowKeys: React.Key[];
  onPaginationChange: (page: number, pageSize: number) => void;
  onRowSelectionChange: (keys: React.Key[]) => void;
  onEditTicket: (ticket: Ticket) => void;
  onViewActivity: (ticket: Ticket) => void;
  onBatchDelete: () => void;
  // onCreateTicket 已移除，统一使用页面顶部的创建按钮
}

// 性能配置
const PERFORMANCE_CONFIG = {
  // 虚拟滚动阈值
  VIRTUAL_SCROLL_THRESHOLD: 100,
  // 虚拟滚动行高
  VIRTUAL_ITEM_HEIGHT: 80,
  // 虚拟滚动容器高度
  VIRTUAL_CONTAINER_HEIGHT: 600,
  // 批量操作阈值
  BATCH_OPERATION_THRESHOLD: 50,
};

export const TicketTable: React.FC<TicketTableProps> = React.memo(
  ({
    tickets,
    loading,
    pagination,
    selectedRowKeys,
    onPaginationChange,
    onRowSelectionChange,
    onEditTicket,
    onViewActivity,
    onBatchDelete,
  }) => {
    // 自动启用虚拟滚动（当数据量大时）
    const [virtualScrollEnabled, setVirtualScrollEnabled] = useState(
      (tickets?.length || 0) > PERFORMANCE_CONFIG.VIRTUAL_SCROLL_THRESHOLD
    );

    // 自动启用虚拟滚动
    React.useEffect(() => {
      if (tickets && Array.isArray(tickets)) {
        setVirtualScrollEnabled(tickets.length > PERFORMANCE_CONFIG.VIRTUAL_SCROLL_THRESHOLD);
      }
    }, [tickets]);

    // 优化的批量删除处理
    const handleBatchDelete = useCallback(() => {
      if (!selectedRowKeys || selectedRowKeys.length === 0) {
        return;
      }

      // 大量数据时显示确认对话框
      if (
        selectedRowKeys &&
        selectedRowKeys.length > PERFORMANCE_CONFIG.BATCH_OPERATION_THRESHOLD
      ) {
        // 这里可以添加更复杂的确认逻辑
        console.log(`Batch deleting ${selectedRowKeys.length} tickets`);
      }

      onBatchDelete();
    }, [selectedRowKeys?.length, onBatchDelete]);

    // 优化的选择处理
    const handleRowSelectionChange = useCallback(
      (keys: React.Key[]) => {
        // 大量选择时使用防抖
        if (keys.length > PERFORMANCE_CONFIG.BATCH_OPERATION_THRESHOLD) {
          // 这里可以添加防抖逻辑
          console.log(`Selecting ${keys.length} tickets`);
        }
        onRowSelectionChange(keys);
      },
      [onRowSelectionChange]
    );

    // 渲染批量操作栏（仅显示批量操作相关功能）
    const renderActionBar = () => {
      // 如果没有选中项，不显示操作栏
      if (!selectedRowKeys || selectedRowKeys.length === 0) {
        return null;
      }

      return (
        <div className='flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6 p-4 bg-blue-50 rounded-lg border border-blue-200 shadow-sm'>
          <div className='flex items-center gap-3'>
            <Badge count={selectedRowKeys.length} showZero className='bg-blue-500' />
            <span className='text-sm font-medium text-gray-700'>
              已选择 <span className='text-blue-600 font-bold'>{selectedRowKeys.length}</span>{' '}
              个工单
            </span>
          </div>
          <div className='flex flex-wrap items-center gap-3'>
            <Button
              danger
              size='large'
              onClick={handleBatchDelete}
              className='rounded-lg hover:shadow-md transition-all duration-200'
              icon={<AlertTriangle size={16} />}
            >
              批量删除 ({selectedRowKeys.length})
            </Button>
          </div>
        </div>
      );
    };

    // 状态映射
    const statusMap: Record<string, string> = {
      new: '新建',
      open: '待处理',
      in_progress: '处理中',
      pending_approval: '待审批',
      resolved: '已解决',
      closed: '已关闭',
      cancelled: '已取消',
    };

    // 优先级映射
    const priorityMap: Record<string, string> = {
      low: '低',
      medium: '中',
      high: '高',
      urgent: '紧急',
      critical: '紧急',
    };

    // 类型映射
    const typeMap: Record<string, string> = {
      incident: '事件',
      service_request: '服务请求',
      problem: '问题',
      change: '变更',
    };

    const tableColumns = [
      {
        title: '工单信息',
        dataIndex: 'title',
        key: 'title',
        render: (text: string, record: Ticket) => (
          <div>
            <div className='font-medium'>{text}</div>
            <div className='text-sm text-gray-500'>
              #{record.id} • {record.category || '未分类'}
            </div>
          </div>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        render: (status: string) => statusMap[status] || status,
      },
      {
        title: '优先级',
        dataIndex: 'priority',
        key: 'priority',
        render: (priority: string) => priorityMap[priority] || priority,
      },
      {
        title: '类型',
        dataIndex: 'type',
        key: 'type',
        render: (type: string) => typeMap[type] || type || '未分类',
      },
      {
        title: '处理人',
        dataIndex: 'assignee',
        key: 'assignee',
        render: (assignee: { name: string }) => assignee?.name || '未分配',
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        key: 'created_at',
        render: (text: string) => new Date(text).toLocaleString('zh-CN'),
      },
      {
        title: '操作',
        key: 'actions',
        render: (_: any, record: Ticket) => (
          <Space size='small'>
            <Button type='text' size='small' onClick={() => onEditTicket(record)}>
              编辑
            </Button>
            <Button type='text' size='small' onClick={() => onViewActivity(record)}>
              查看
            </Button>
          </Space>
        ),
      },
    ];

    // 加载状态
    if (loading) {
      return (
        <div className='space-y-4'>
          {renderActionBar()}
          <TicketListSkeleton rows={pagination.pageSize} />
        </div>
      );
    }

    // 空状态
    if (!tickets || tickets.length === 0) {
      return (
        <div className='space-y-4'>
          {renderActionBar()}
          <LoadingEmptyError
            state='empty'
            empty={{
              title: '暂无工单',
              description: '当前没有工单数据，请使用页面顶部的"创建工单"按钮创建第一个工单',
              // 移除 actionText 和 onAction，引导用户使用页面顶部的创建按钮
            }}
          />
        </div>
      );
    }

    return (
      <div className='space-y-4'>
        {renderActionBar()}

        {/* 虚拟滚动列表 */}
        {virtualScrollEnabled ? (
          <LazyWrapper fallback={<LoadingSpinner />}>
            <VirtualizedTicketList
              tickets={tickets}
              loading={loading}
              selectedRowKeys={(selectedRowKeys || []) as string[]}
              onRowSelectionChange={handleRowSelectionChange}
              onEditTicket={onEditTicket}
              onViewActivity={onViewActivity}
              height={PERFORMANCE_CONFIG.VIRTUAL_CONTAINER_HEIGHT}
              itemHeight={PERFORMANCE_CONFIG.VIRTUAL_ITEM_HEIGHT}
            />
          </LazyWrapper>
        ) : (
          <Table
            columns={tableColumns}
            dataSource={tickets}
            rowKey='id'
            pagination={pagination}
            loading={loading}
            onRow={record => ({
              onClick: () => onEditTicket(record),
            })}
            rowSelection={{
              selectedRowKeys: selectedRowKeys || [],
              onChange: onRowSelectionChange,
            }}
          />
        )}

        {/* 分页信息 */}
        {!virtualScrollEnabled && pagination.total > 0 && (
          <div className='flex justify-between items-center p-4 bg-gray-50 rounded-lg'>
            <div className='text-sm text-gray-600'>
              显示第 {(pagination.current - 1) * pagination.pageSize + 1} 到{' '}
              {Math.min(pagination.current * pagination.pageSize, pagination.total)} 条，共{' '}
              {pagination.total} 条记录
            </div>
            <div className='text-sm text-gray-600'>
              第 {pagination.current} 页，共 {Math.ceil(pagination.total / pagination.pageSize)} 页
            </div>
          </div>
        )}
      </div>
    );
  }
);

TicketTable.displayName = 'TicketTable';
