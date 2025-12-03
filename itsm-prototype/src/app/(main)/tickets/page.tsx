'use client';

import React, { useCallback, useState, useMemo } from 'react';
import { Modal, message, Tabs, Button, Space, Dropdown } from 'antd';
import type { TabsProps, MenuProps } from 'antd';
import { Plus, FileText, Download, Upload, Settings, MoreHorizontal } from 'lucide-react';
import { TicketStats } from '@/components/business/TicketStats';
import { TicketFilters } from '@/components/business/TicketFilters';
import { TicketTable } from '@/components/business/TicketTable';
import { TicketModal, TicketTemplateModal } from '@/components/business/TicketModal';
import { TicketAssociation } from '@/components/business/TicketAssociation';
import { SatisfactionDashboard } from '@/components/business/SatisfactionDashboard';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { LazyWrapper, LoadingSpinner } from '@/components/common/LazyComponents';
import { ErrorBoundary } from '@/components/common/ErrorBoundary';
import { BatchOperationConfirm } from '@/components/business/BatchOperationConfirm';
import {
  useTicketsQuery,
  useTicketStatsQuery,
  useCreateTicketMutation,
  useUpdateTicketMutation,
  useBatchDeleteTicketsMutation,
} from '@/lib/hooks/useTicketsQuery';
import { useTicketFilters } from '@/lib/hooks/useTicketFilters';
import { useTicketStore } from '@/app/lib/store/ticket-store';
import { useI18n } from '@/lib/i18n';
import { Ticket } from '@/lib/services/ticket-service';

export default function TicketsPage() {
  const { t } = useI18n();
  // 状态管理
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });
  const [currentViewId, setCurrentViewId] = useState<number | undefined>();
  const [batchConfirmVisible, setBatchConfirmVisible] = useState(false);
  const [batchOperationType, setBatchOperationType] = useState<
    'delete' | 'assign' | 'updateStatus' | 'updatePriority' | 'export' | 'archive'
  >('delete');

  // 自定义hooks
  const { componentFilters, updateComponentFilters, mapComponentToDomain } = useTicketFilters();

  // Zustand store
  const {
    selectedRowKeys,
    modalVisible,
    editingTicket,
    templateModalVisible,
    setSelectedRowKeys,
    setModalVisible,
    setEditingTicket,
    setTemplateModalVisible,
  } = useTicketStore();

  // React Query hooks
  const {
    data: ticketsData,
    isLoading: ticketsLoading,
    error: ticketsError,
    refetch: refetchTickets,
  } = useTicketsQuery(mapComponentToDomain(componentFilters), pagination);

  const {
    data: statsData,
    isLoading: statsLoading,
    error: statsError,
    refetch: refetchStats,
  } = useTicketStatsQuery();

  const createTicketMutation = useCreateTicketMutation();
  const updateTicketMutation = useUpdateTicketMutation();
  const batchDeleteMutation = useBatchDeleteTicketsMutation();

  // 计算数据 - 确保tickets始终是数组
  const tickets = Array.isArray(ticketsData?.tickets) ? ticketsData.tickets : [];
  const stats = statsData || {
    total: 0,
    open: 0,
    resolved: 0,
    highPriority: 0,
  };

  // 获取选中的工单详情
  const selectedTickets = useMemo(() => {
    if (!selectedRowKeys || selectedRowKeys.length === 0) {
      return [];
    }
    return tickets.filter(ticket => selectedRowKeys.includes(String(ticket.id)));
  }, [tickets, selectedRowKeys]);

  // 更新分页
  React.useEffect(() => {
    if (ticketsData?.total !== undefined) {
      setPagination(prev => ({ ...prev, total: ticketsData.total }));
    }
  }, [ticketsData?.total]);

  // 事件处理器
  const handleCreateTicket = useCallback(() => {
    setEditingTicket(null);
    setModalVisible(true);
  }, [setEditingTicket, setModalVisible]);

  const handleEditTicket = useCallback(
    (ticket: Ticket) => {
      setEditingTicket(ticket);
      setModalVisible(true);
    },
    [setEditingTicket, setModalVisible]
  );

  const handleViewActivity = useCallback((ticket: Ticket) => {
    console.log('View activity log:', ticket);
  }, []);

  const handleBatchDelete = useCallback(() => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('tickets.deleteConfirmationWarning') || '请至少选择一个工单');
      return;
    }
    setBatchOperationType('delete');
    setBatchConfirmVisible(true);
  }, [selectedRowKeys.length, t]);

  // 执行批量删除
  const handleConfirmBatchDelete = useCallback(async () => {
    try {
      await batchDeleteMutation.mutateAsync(selectedRowKeys as string[]);
      const deletedCount = selectedRowKeys.length;
      const deletedIds = selectedRowKeys.slice(0, 5).join(', ');
      const moreText = selectedRowKeys.length > 5 ? ` 等 ${deletedCount} 个工单` : '';

      setSelectedRowKeys([]);
      setBatchConfirmVisible(false);

      // 增强的成功反馈
      message.success({
        content: `成功删除 ${deletedCount} 个工单${moreText}`,
        duration: 4,
      });
    } catch (error) {
      console.error('Delete failed:', error);
      const errorMessage = error instanceof Error ? error.message : '删除失败';
      message.error({
        content: `删除失败：${errorMessage}`,
        duration: 4,
      });
    }
  }, [selectedRowKeys, batchDeleteMutation, setSelectedRowKeys]);

  const handleSearch = useCallback(
    (keyword: string) => {
      updateComponentFilters({ keyword });
    },
    [updateComponentFilters]
  );

  const handleFilterChange = useCallback(
    (newFilters: any) => {
      updateComponentFilters(newFilters);
      setPagination(prev => ({ ...prev, current: 1 }));
    },
    [updateComponentFilters]
  );

  const handleRefresh = useCallback(async () => {
    // 使用 React Query 的 refetch 优雅地刷新数据
    try {
      await Promise.all([refetchTickets(), refetchStats()]);
      message.success(t('tickets.refreshSuccess'));
    } catch (error) {
      console.error('Refresh failed:', error);
      message.error(t('tickets.refreshFailed'));
    }
  }, [refetchTickets, refetchStats, t]);

  const handleModalSubmit = useCallback(
    async (values: Partial<Ticket>) => {
      try {
        if (editingTicket) {
          await updateTicketMutation.mutateAsync({
            id: String(editingTicket.id),
            data: values,
          });
        } else {
          await createTicketMutation.mutateAsync(values as any);
        }
        setModalVisible(false);
        setEditingTicket(null);
      } catch (error) {
        console.error('Submit failed:', error);
      }
    },
    [editingTicket, updateTicketMutation, createTicketMutation, setModalVisible, setEditingTicket]
  );

  const handleModalCancel = useCallback(() => {
    setModalVisible(false);
    setEditingTicket(null);
  }, [setModalVisible, setEditingTicket]);

  const handlePaginationChange = useCallback((page: number, pageSize: number) => {
    setPagination(prev => ({ ...prev, current: page, pageSize }));
  }, []);

  const handleRowSelectionChange = useCallback(
    (keys: React.Key[]) => {
      setSelectedRowKeys(keys as string[]);
    },
    [setSelectedRowKeys]
  );

  // 视图切换处理
  const handleViewChange = useCallback((view: any) => {
    setCurrentViewId(view?.id);
    // 视图切换时，筛选条件会通过onFiltersChange自动更新
  }, []);

  // 错误状态
  if (ticketsError || statsError) {
    const error = ticketsError || statsError;
    return (
      <LoadingEmptyError
        state='error'
        error={{
          title: t('tickets.loadingFailed'),
          description: error?.message || t('tickets.unknownError'),
          actionText: t('tickets.retry'),
          onAction: handleRefresh,
        }}
      />
    );
  }

  // 快速操作菜单（创建相关）
  const createActionMenu: MenuProps = {
    items: [
      {
        key: 'template',
        label: '使用模板',
        icon: <FileText size={16} />,
        onClick: () => setTemplateModalVisible(true),
      },
      {
        key: 'import',
        label: '导入工单',
        icon: <Upload size={16} />,
        onClick: () => message.info('导入功能开发中'),
      },
    ],
  };

  // 更多操作菜单（导出和设置）
  const moreActionMenu: MenuProps = {
    items: [
      {
        key: 'export',
        label: '导出数据',
        icon: <Download size={16} />,
        onClick: () => message.info('导出功能开发中'),
      },
      {
        type: 'divider',
      },
      {
        key: 'settings',
        label: '视图设置',
        icon: <Settings size={16} />,
        onClick: () => message.info('视图设置功能开发中'),
      },
      {
        key: 'columns',
        label: '列设置',
        icon: <Settings size={16} />,
        onClick: () => message.info('列设置功能开发中'),
      },
    ],
  };

  return (
    <ErrorBoundary>
      <div className='space-y-6'>
        {/* 页面标题区域 */}
        <div className='flex items-center justify-between'>
          <div>
            <h1 className='text-2xl font-bold text-gray-900'>工单管理</h1>
            <p className='text-sm text-gray-600 mt-1'>管理和跟踪所有工单，快速处理服务请求</p>
          </div>
        </div>

        {/* 快速操作栏 */}
        <div className='flex items-center justify-between gap-3 p-4 bg-white rounded-lg border border-gray-100 shadow-md hover:shadow-lg transition-shadow overflow-hidden'>
          {/* 左侧：主要操作 */}
          <div className='flex items-center gap-2 flex-shrink-0'>
            <Button
              type='primary'
              size='middle'
              icon={<Plus size={16} />}
              onClick={handleCreateTicket}
              className='rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 border-0 shadow-md hover:shadow-lg transition-all duration-200'
            >
              创建工单
            </Button>
            <Dropdown menu={createActionMenu} trigger={['click']}>
              <Button
                size='middle'
                icon={<FileText size={16} />}
                className='rounded-lg border-gray-200 hover:border-blue-400 hover:text-blue-600 transition-all duration-200'
              >
                更多创建
              </Button>
            </Dropdown>
          </div>
          {/* 右侧：辅助操作 */}
          <div className='flex items-center gap-2 flex-shrink-0'>
            <Dropdown menu={moreActionMenu} trigger={['click']}>
              <Button
                size='middle'
                icon={<MoreHorizontal size={16} />}
                className='rounded-lg border-gray-200 hover:border-blue-400 hover:text-blue-600 transition-all duration-200'
              >
                更多操作
              </Button>
            </Dropdown>
          </div>
        </div>

        {/* 统计卡片 - 懒加载 */}
        <LazyWrapper fallback={<LoadingSpinner />}>
          <ErrorBoundary>
            <TicketStats stats={stats} loading={statsLoading} />
          </ErrorBoundary>
        </LazyWrapper>

        {/* 筛选器 - 懒加载 */}
        <LazyWrapper fallback={<LoadingSpinner />}>
          <ErrorBoundary>
            <TicketFilters
              filters={componentFilters}
              onFilterChange={handleFilterChange}
              onSearch={handleSearch}
              onRefresh={handleRefresh}
              loading={ticketsLoading}
              currentViewId={currentViewId}
              onViewChange={handleViewChange}
            />
          </ErrorBoundary>
        </LazyWrapper>

        {/* 主要内容区域 */}
        <ErrorBoundary>
          <div className='bg-white rounded-lg border border-gray-100 shadow-md'>
            <Tabs
              defaultActiveKey='1'
              type='card'
              size='large'
              items={[
                {
                  key: '1',
                  label: t('tickets.list'),
                  children: (
                    <ErrorBoundary>
                      <TicketTable
                        tickets={tickets}
                        loading={ticketsLoading}
                        pagination={pagination}
                        selectedRowKeys={selectedRowKeys || []}
                        onPaginationChange={handlePaginationChange}
                        onRowSelectionChange={handleRowSelectionChange}
                        onEditTicket={handleEditTicket}
                        onViewActivity={handleViewActivity}
                        onBatchDelete={handleBatchDelete}
                      />
                    </ErrorBoundary>
                  ),
                },
                {
                  key: '2',
                  label: t('tickets.association'),
                  children: (
                    <ErrorBoundary>
                      <TicketAssociation />
                    </ErrorBoundary>
                  ),
                },
                {
                  key: '3',
                  label: t('tickets.satisfaction'),
                  children: (
                    <ErrorBoundary>
                      <SatisfactionDashboard />
                    </ErrorBoundary>
                  ),
                },
              ]}
            />
          </div>
        </ErrorBoundary>

        {/* 模态框 */}
        <ErrorBoundary>
          <TicketModal
            visible={modalVisible}
            editingTicket={editingTicket}
            onCancel={handleModalCancel}
            onSubmit={handleModalSubmit}
            loading={createTicketMutation.isPending || updateTicketMutation.isPending}
          />
        </ErrorBoundary>

        <ErrorBoundary>
          <TicketTemplateModal
            visible={templateModalVisible}
            onCancel={() => setTemplateModalVisible(false)}
          />
        </ErrorBoundary>

        {/* 批量操作确认对话框 */}
        <ErrorBoundary>
          <BatchOperationConfirm
            visible={batchConfirmVisible}
            operationType={batchOperationType}
            selectedCount={selectedRowKeys?.length || 0}
            selectedTickets={selectedTickets}
            onConfirm={handleConfirmBatchDelete}
            onCancel={() => setBatchConfirmVisible(false)}
            loading={batchDeleteMutation.isPending}
          />
        </ErrorBoundary>
      </div>
    </ErrorBoundary>
  );
}
