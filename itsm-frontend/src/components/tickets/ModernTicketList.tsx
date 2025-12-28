import React, { useEffect, useState } from 'react';
import { 
  ResponsiveLayout, 
  PageHeader, 
  Card, 
  Grid, 
  Stack, 
  LoadingState, 
  EmptyState 
} from '../layout/ResponsiveLayout';
import { useModernTicketStore, useModernTicketSelectors } from '@/lib/stores/modern-ticket-store';
import { ticketUtils } from '@/lib/api/ticket';
import { cn } from '@/lib/utils';

interface TicketFiltersProps {
  onFilterChange?: (filters: any) => void;
}

function TicketFilters({ onFilterChange }: TicketFiltersProps) {
  const { filters, setFilters, clearFilters } = useModernTicketStore();
  const { getActiveFiltersCount } = useModernTicketSelectors();
  
  const activeFiltersCount = getActiveFiltersCount();

  const statusOptions = [
    { value: 'new', label: '新建', color: 'blue' },
    { value: 'open', label: '开放', color: 'cyan' },
    { value: 'in_progress', label: '处理中', color: 'orange' },
    { value: 'pending', label: '待处理', color: 'yellow' },
    { value: 'resolved', label: '已解决', color: 'green' },
    { value: 'closed', label: '已关闭', color: 'gray' },
    { value: 'cancelled', label: '已取消', color: 'red' },
  ];

  const priorityOptions = [
    { value: 'low', label: '低', color: 'green' },
    { value: 'normal', label: '普通', color: 'blue' },
    { value: 'high', label: '高', color: 'orange' },
    { value: 'urgent', label: '紧急', color: 'red' },
    { value: 'critical', label: '关键', color: 'purple' },
  ];

  const handleStatusChange = (status: string, checked: boolean) => {
    const currentStatus = filters.status || [];
    const newStatus = checked 
      ? [...currentStatus, status]
      : currentStatus.filter(s => s !== status);
    
    setFilters({ status: newStatus.length > 0 ? newStatus : undefined });
  };

  const handlePriorityChange = (priority: string, checked: boolean) => {
    const currentPriority = filters.priority || [];
    const newPriority = checked
      ? [...currentPriority, priority]
      : currentPriority.filter(p => p !== priority);
    
    setFilters({ priority: newPriority.length > 0 ? newPriority : undefined });
  };

  return (
    <Card
      title="筛选条件"
      headerActions={
        activeFiltersCount > 0 && (
          <button
            onClick={clearFilters}
            className="text-sm text-blue-600 hover:text-blue-800"
          >
            清除筛选 ({activeFiltersCount})
          </button>
        )
      }
      className="mb-6"
    >
      <Stack spacing="lg">
        {/* Keywords Search */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            关键字搜索
          </label>
          <input
            type="text"
            value={filters.keywords || ''}
            onChange={(e) => setFilters({ keywords: e.target.value || undefined })}
            placeholder="搜索标题或描述..."
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Status Filter */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            状态
          </label>
          <div className="flex flex-wrap gap-2">
            {statusOptions.map(option => (
              <label
                key={option.value}
                className="inline-flex items-center cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={filters.status?.includes(option.value) || false}
                  onChange={(e) => handleStatusChange(option.value, e.target.checked)}
                  className="sr-only"
                />
                <span className={cn(
                  'px-3 py-1 rounded-full text-xs font-medium border-2 transition-colors',
                  filters.status?.includes(option.value)
                    ? `bg-${option.color}-100 text-${option.color}-800 border-${option.color}-300`
                    : 'bg-gray-100 text-gray-600 border-gray-200 hover:border-gray-300'
                )}>
                  {option.label}
                </span>
              </label>
            ))}
          </div>
        </div>

        {/* Priority Filter */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            优先级
          </label>
          <div className="flex flex-wrap gap-2">
            {priorityOptions.map(option => (
              <label
                key={option.value}
                className="inline-flex items-center cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={filters.priority?.includes(option.value) || false}
                  onChange={(e) => handlePriorityChange(option.value, e.target.checked)}
                  className="sr-only"
                />
                <span className={cn(
                  'px-3 py-1 rounded-full text-xs font-medium border-2 transition-colors',
                  filters.priority?.includes(option.value)
                    ? `bg-${option.color}-100 text-${option.color}-800 border-${option.color}-300`
                    : 'bg-gray-100 text-gray-600 border-gray-200 hover:border-gray-300'
                )}>
                  {option.label}
                </span>
              </label>
            ))}
          </div>
        </div>

        {/* Category Filter */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            分类
          </label>
          <select
            value={filters.category || ''}
            onChange={(e) => setFilters({ category: e.target.value || undefined })}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="">所有分类</option>
            <option value="incident">故障</option>
            <option value="service_request">服务请求</option>
            <option value="problem">问题</option>
            <option value="change">变更</option>
          </select>
        </div>
      </Stack>
    </Card>
  );
}

interface TicketCardProps {
  ticket: any;
  selected?: boolean;
  onSelect?: (selected: boolean) => void;
  onClick?: () => void;
}

function TicketCard({ ticket, selected, onSelect, onClick }: TicketCardProps) {
  const { isTicketLoading, getTicketError } = useModernTicketSelectors();
  const loading = isTicketLoading(ticket.id);
  const error = getTicketError(ticket.id);

  const statusColor = ticketUtils.getStatusColor(ticket.status);
  const priorityColor = ticketUtils.getPriorityColor(ticket.priority);

  return (
    <Card
      className={cn(
        'cursor-pointer transition-all hover:shadow-md',
        selected && 'ring-2 ring-blue-500',
        loading && 'opacity-50'
      )}
      onClick={() => onClick?.()}
    >
      <Stack spacing="sm">
        {/* Header with checkbox and ticket number */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <input
              type="checkbox"
              checked={selected || false}
              onChange={(e) => {
                e.stopPropagation();
                onSelect?.(e.target.checked);
              }}
              className="h-4 w-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
            />
            <span className="text-sm font-mono text-gray-500">
              {ticketUtils.formatTicketNumber(ticket)}
            </span>
          </div>
          
          <div className="flex items-center space-x-2">
            <span className={cn(
              'inline-flex items-center px-2 py-1 rounded-full text-xs font-medium',
              `bg-${statusColor}-100 text-${statusColor}-800`
            )}>
              {ticket.status}
            </span>
            <span className={cn(
              'inline-flex items-center px-2 py-1 rounded-full text-xs font-medium',
              `bg-${priorityColor}-100 text-${priorityColor}-800`
            )}>
              {ticket.priority}
            </span>
          </div>
        </div>

        {/* Title */}
        <h3 className="font-medium text-gray-900 line-clamp-2">
          {ticket.title}
        </h3>

        {/* Meta information */}
        <div className="flex items-center justify-between text-sm text-gray-500">
          <div className="flex items-center space-x-4">
            <span>分类: {ticket.category}</span>
            {ticket.assigned_to && (
              <span>负责人: {ticket.assigned_to}</span>
            )}
          </div>
          <span>
            {ticketUtils.formatRelativeTime(ticket.created_at)}
          </span>
        </div>

        {/* Error state */}
        {error && (
          <div className="text-xs text-red-600 bg-red-50 px-2 py-1 rounded">
            {error}
          </div>
        )}

        {/* Loading overlay */}
        {loading && (
          <div className="absolute inset-0 bg-white bg-opacity-50 flex items-center justify-center">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600" />
          </div>
        )}
      </Stack>
    </Card>
  );
}

interface TicketListActionsProps {
  selectedCount: number;
  onCreateTicket?: () => void;
  onBulkAction?: (action: string) => void;
}

function TicketListActions({ selectedCount, onCreateTicket, onBulkAction }: TicketListActionsProps) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center space-x-3">
        <button
          onClick={onCreateTicket}
          className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-colors"
        >
          创建工单
        </button>
        
        {selectedCount > 0 && (
          <div className="flex items-center space-x-2">
            <span className="text-sm text-gray-500">
              已选择 {selectedCount} 个工单
            </span>
            <select
              onChange={(e) => {
                if (e.target.value) {
                  onBulkAction?.(e.target.value);
                  e.target.value = '';
                }
              }}
              className="px-3 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">批量操作...</option>
              <option value="assign">批量分配</option>
              <option value="status">批量更新状态</option>
              <option value="delete">批量删除</option>
            </select>
          </div>
        )}
      </div>

      <div className="flex items-center space-x-3">
        <select className="px-3 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
          <option>排序: 创建时间</option>
          <option>排序: 更新时间</option>
          <option>排序: 优先级</option>
          <option>排序: 状态</option>
        </select>

        <select className="px-3 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
          <option>每页显示: 20</option>
          <option>每页显示: 50</option>
          <option>每页显示: 100</option>
        </select>
      </div>
    </div>
  );
}

export function ModernTicketList() {
  const {
    tickets,
    loading,
    errors,
    selectedTickets,
    pagination,
    fetchTickets,
    selectTicket,
    deselectTicket,
    selectAllTickets,
    clearSelection,
  } = useModernTicketStore();

  const { getVisibleTickets, getSelectionStats } = useModernTicketSelectors();
  const visibleTickets = getVisibleTickets();
  const selectionStats = getSelectionStats();

  useEffect(() => {
    fetchTickets();
  }, [fetchTickets]);

  const handleTicketSelect = (ticketId: string, selected: boolean) => {
    if (selected) {
      selectTicket(ticketId);
    } else {
      deselectTicket(ticketId);
    }
  };

  const handleSelectAll = () => {
    if (selectionStats.isAllSelected) {
      clearSelection();
    } else {
      selectAllTickets();
    }
  };

  const sidebar = <TicketFilters />;

  const header = (
    <PageHeader
      title="工单管理"
      subtitle={`共 ${pagination.totalCount} 个工单`}
      breadcrumbs={[
        { label: '首页', href: '/' },
        { label: '工单管理' }
      ]}
      actions={
        <TicketListActions
          selectedCount={selectionStats.selectedCount}
          onCreateTicket={() => console.log('Create ticket')}
          onBulkAction={(action) => console.log('Bulk action:', action)}
        />
      }
    />
  );

  return (
    <ResponsiveLayout
      header={header}
      sidebar={sidebar}
      sidebarWidth="normal"
    >
      <Stack spacing="lg">
        {/* Header actions and bulk selection */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <label className="inline-flex items-center">
              <input
                type="checkbox"
                checked={selectionStats.isAllSelected}
                ref={(el) => {
                  if (el) el.indeterminate = selectionStats.hasSelection && !selectionStats.isAllSelected;
                }}
                onChange={handleSelectAll}
                className="h-4 w-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm text-gray-700">
                全选
              </span>
            </label>
            
            {selectionStats.hasSelection && (
              <span className="text-sm text-gray-500">
                已选择 {selectionStats.selectedCount} / {selectionStats.totalCount}
              </span>
            )}
          </div>

          <div className="text-sm text-gray-500">
            第 {pagination.page} 页 / 共 {Math.ceil(pagination.totalCount / pagination.pageSize)} 页
          </div>
        </div>

        {/* Error state */}
        {errors.tickets && (
          <Card variant="outlined" className="border-red-200 bg-red-50">
            <div className="text-red-800">
              <strong>加载失败:</strong> {errors.tickets}
            </div>
          </Card>
        )}

        {/* Content */}
        <LoadingState
          isLoading={loading.tickets && visibleTickets.length === 0}
          skeleton={
            <Grid cols={3}>
              {[...Array(6)].map((_, i) => (
                <Card key={i}>
                  <Stack spacing="sm">
                    <div className="h-4 bg-gray-200 rounded w-1/3"></div>
                    <div className="h-6 bg-gray-200 rounded w-3/4"></div>
                    <div className="h-4 bg-gray-200 rounded w-1/2"></div>
                  </Stack>
                </Card>
              ))}
            </Grid>
          }
        >
          {visibleTickets.length > 0 ? (
            <Grid cols={3}>
              {visibleTickets.map(ticket => (
                <TicketCard
                  key={ticket.id}
                  ticket={ticket}
                  selected={selectedTickets.includes(ticket.id)}
                  onSelect={(selected) => handleTicketSelect(ticket.id, selected)}
                  onClick={() => console.log('Open ticket:', ticket.id)}
                />
              ))}
            </Grid>
          ) : (
            <EmptyState
              title="暂无工单"
              description="创建您的第一个工单或调整筛选条件"
              icon={
                <svg className="h-12 w-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              }
              action={
                <button className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors">
                  创建工单
                </button>
              }
            />
          )}
        </LoadingState>

        {/* Pagination */}
        {pagination.totalCount > pagination.pageSize && (
          <div className="flex items-center justify-between border-t border-gray-200 bg-white px-4 py-3 sm:px-6">
            <div className="flex flex-1 justify-between sm:hidden">
              <button
                disabled={!pagination.hasPrevious}
                className="relative inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
              >
                上一页
              </button>
              <button
                disabled={!pagination.hasNext}
                className="relative ml-3 inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
              >
                下一页
              </button>
            </div>
            
            <div className="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
              <div>
                <p className="text-sm text-gray-700">
                  显示第 <span className="font-medium">{(pagination.page - 1) * pagination.pageSize + 1}</span> 到{' '}
                  <span className="font-medium">
                    {Math.min(pagination.page * pagination.pageSize, pagination.totalCount)}
                  </span>{' '}
                  条，共 <span className="font-medium">{pagination.totalCount}</span> 条结果
                </p>
              </div>
              <div>
                <nav className="isolate inline-flex -space-x-px rounded-md shadow-sm">
                  <button
                    disabled={!pagination.hasPrevious}
                    className="relative inline-flex items-center rounded-l-md px-2 py-2 text-gray-400 ring-1 ring-inset ring-gray-300 hover:bg-gray-50 focus:z-20 focus:outline-offset-0 disabled:opacity-50"
                  >
                    上一页
                  </button>
                  <button
                    disabled={!pagination.hasNext}
                    className="relative inline-flex items-center rounded-r-md px-2 py-2 text-gray-400 ring-1 ring-inset ring-gray-300 hover:bg-gray-50 focus:z-20 focus:outline-offset-0 disabled:opacity-50"
                  >
                    下一页
                  </button>
                </nav>
              </div>
            </div>
          </div>
        )}
      </Stack>
    </ResponsiveLayout>
  );
}

export default ModernTicketList;