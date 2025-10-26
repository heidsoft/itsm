'use client';

import React, { useMemo, useCallback, useState } from 'react';
import { Button, Space, Tooltip, Dropdown, Badge, Switch } from 'antd';
import { Download, Plus, AlertTriangle, Settings, Zap } from 'lucide-react';
import {
  Ticket,
  TicketStatus,
  TicketPriority,
  TicketType,
} from '../../lib/services/ticket-service';
import { VirtualizedTicketList } from './VirtualizedTicketList';
import { LazyWrapper, LoadingSpinner } from '../common/LazyComponents';

interface OptimizedTicketListProps {
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
  onCreateTicket: () => void;
  onBatchDelete: () => void;
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

export const OptimizedTicketList: React.FC<OptimizedTicketListProps> = React.memo(
  ({
    tickets,
    loading,
    pagination,
    selectedRowKeys,
    onPaginationChange,
    onRowSelectionChange,
    onEditTicket,
    onViewActivity,
    onCreateTicket,
    onBatchDelete,
  }) => {
    // 性能模式状态
    const [performanceMode, setPerformanceMode] = useState(true);
    const [virtualScrollEnabled, setVirtualScrollEnabled] = useState(
      tickets.length > PERFORMANCE_CONFIG.VIRTUAL_SCROLL_THRESHOLD
    );

    // 自动启用虚拟滚动
    React.useEffect(() => {
      setVirtualScrollEnabled(tickets.length > PERFORMANCE_CONFIG.VIRTUAL_SCROLL_THRESHOLD);
    }, [tickets.length]);

    // 性能统计
    const performanceStats = useMemo(() => {
      const startTime = performance.now();
      const stats = {
        totalTickets: tickets.length,
        selectedTickets: selectedRowKeys.length,
        virtualScrollEnabled,
        performanceMode,
        renderTime: 0,
      };
      stats.renderTime = performance.now() - startTime;
      return stats;
    }, [tickets.length, selectedRowKeys.length, virtualScrollEnabled, performanceMode]);

    // 优化的批量删除处理
    const handleBatchDelete = useCallback(() => {
      if (selectedRowKeys.length === 0) {
        return;
      }

      // 大量数据时显示确认对话框
      if (selectedRowKeys.length > PERFORMANCE_CONFIG.BATCH_OPERATION_THRESHOLD) {
        // 这里可以添加更复杂的确认逻辑
        console.log(`Batch deleting ${selectedRowKeys.length} tickets`);
      }

      onBatchDelete();
    }, [selectedRowKeys.length, onBatchDelete]);

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

    // 渲染性能信息
    const renderPerformanceInfo = () => {
      if (!performanceMode) return null;

      return (
        <div className='mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg'>
          <div className='flex items-center justify-between'>
            <div className='text-sm text-blue-700'>
              <span className='font-medium'>Performance Mode:</span>
              <span className='ml-2'>
                {virtualScrollEnabled ? 'Virtual Scroll' : 'Standard List'} |{tickets.length}{' '}
                tickets |{selectedRowKeys.length} selected | Render:{' '}
                {performanceStats.renderTime.toFixed(2)}ms
              </span>
            </div>
            <div className='flex items-center gap-2'>
              <Switch size='small' checked={performanceMode} onChange={setPerformanceMode} />
              <span className='text-xs text-blue-600'>Performance Mode</span>
            </div>
          </div>
        </div>
      );
    };

    // 渲染操作栏
    const renderActionBar = () => (
      <div className='flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6 p-4 bg-white rounded-lg border border-gray-100 shadow-sm'>
        <div className='flex items-center gap-3'>
          {selectedRowKeys.length > 0 && (
            <div className='flex items-center gap-2'>
              <Badge count={selectedRowKeys.length} showZero className='bg-blue-500' />
              <span className='text-sm text-gray-600'>
                Selected {selectedRowKeys.length} tickets
              </span>
            </div>
          )}
        </div>
        <div className='flex flex-wrap items-center gap-3'>
          {selectedRowKeys.length > 0 && (
            <Button
              danger
              size='large'
              onClick={handleBatchDelete}
              className='rounded-lg hover:shadow-md transition-all duration-200'
              icon={<AlertTriangle size={16} />}
            >
              Batch Delete ({selectedRowKeys.length})
            </Button>
          )}
          <Button
            icon={<Download size={16} />}
            size='large'
            className='rounded-lg border-gray-200 hover:border-blue-400 hover:text-blue-600 transition-all duration-200'
          >
            Export Data
          </Button>
          <Button
            type='primary'
            icon={<Plus size={18} />}
            size='large'
            onClick={onCreateTicket}
            className='rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 border-0 shadow-md hover:shadow-lg transition-all duration-200'
          >
            Create Ticket
          </Button>
        </div>
      </div>
    );

    // 渲染加载状态
    if (loading) {
      return (
        <div className='space-y-4'>
          {renderPerformanceInfo()}
          {renderActionBar()}
          <div className='flex items-center justify-center h-64'>
            <LoadingSpinner size='large' />
          </div>
        </div>
      );
    }

    // 渲染空状态
    if (tickets.length === 0) {
      return (
        <div className='space-y-4'>
          {renderPerformanceInfo()}
          {renderActionBar()}
          <div className='flex items-center justify-center h-64'>
            <div className='text-center'>
              <div className='text-gray-400 mb-4'>
                <Settings size={48} className='mx-auto' />
              </div>
              <div className='text-gray-600'>No tickets found</div>
            </div>
          </div>
        </div>
      );
    }

    return (
      <div className='space-y-4'>
        {renderPerformanceInfo()}
        {renderActionBar()}

        {/* 虚拟滚动列表 */}
        {virtualScrollEnabled ? (
          <LazyWrapper fallback={<LoadingSpinner />}>
            <VirtualizedTicketList
              tickets={tickets}
              loading={loading}
              selectedRowKeys={selectedRowKeys as string[]}
              onRowSelectionChange={handleRowSelectionChange}
              onEditTicket={onEditTicket}
              onViewActivity={onViewActivity}
              height={PERFORMANCE_CONFIG.VIRTUAL_CONTAINER_HEIGHT}
              itemHeight={PERFORMANCE_CONFIG.VIRTUAL_ITEM_HEIGHT}
            />
          </LazyWrapper>
        ) : (
          <div className='bg-white rounded-lg border border-gray-100 shadow-sm overflow-hidden'>
            <div className='p-4 text-center text-gray-500'>
              Standard list view for {tickets.length} tickets
              <Button
                type='link'
                size='small'
                onClick={() => setVirtualScrollEnabled(true)}
                className='ml-2'
              >
                Enable Virtual Scroll
              </Button>
            </div>
          </div>
        )}

        {/* 分页信息 */}
        {!virtualScrollEnabled && (
          <div className='flex justify-between items-center p-4 bg-gray-50 rounded-lg'>
            <div className='text-sm text-gray-600'>
              Showing {(pagination.current - 1) * pagination.pageSize + 1} to{' '}
              {Math.min(pagination.current * pagination.pageSize, pagination.total)} of{' '}
              {pagination.total} entries
            </div>
            <div className='text-sm text-gray-600'>
              Page {pagination.current} of {Math.ceil(pagination.total / pagination.pageSize)}
            </div>
          </div>
        )}
      </div>
    );
  }
);

OptimizedTicketList.displayName = 'OptimizedTicketList';
