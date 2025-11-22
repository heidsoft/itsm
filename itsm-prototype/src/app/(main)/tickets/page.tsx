'use client';

import React, { useCallback, useState } from 'react';
import { Modal, message } from 'antd';
import { TicketStats } from '@/components/business/TicketStats';
import { TicketFilters } from '@/components/business/TicketFilters';
import { OptimizedTicketList } from '@/components/business/OptimizedTicketList';
import { TicketModal, TicketTemplateModal } from '@/components/business/TicketModal';
import { TicketAssociation } from '@/app/components/TicketAssociation';
import { SatisfactionDashboard } from '@/components/business/SatisfactionDashboard';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { LazyWrapper, LoadingSpinner } from '@/components/common/LazyComponents';
import {
  useTicketsQuery,
  useTicketStatsQuery,
  useCreateTicketMutation,
  useUpdateTicketMutation,
  useBatchDeleteTicketsMutation,
} from '@/lib/hooks/useTicketsQuery';
import { useTicketFilters } from '@/lib/hooks/useTicketFilters';
import { useTicketStore } from '@/lib/stores/ticket-store';

export default function TicketsPage() {
  // 状态管理
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });

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

  // 计算数据
  const tickets = ticketsData?.tickets || [];
  const stats = statsData || {
    total: 0,
    open: 0,
    resolved: 0,
    highPriority: 0,
  };

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
    (ticket: any) => {
      setEditingTicket(ticket);
      setModalVisible(true);
    },
    [setEditingTicket, setModalVisible]
  );

  const handleViewActivity = useCallback((ticket: any) => {
    console.log('View activity log:', ticket);
  }, []);

  const handleBatchDelete = useCallback(async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('Please select tickets to delete');
      return;
    }

    Modal.confirm({
      title: 'Confirm Delete',
      content: `Are you sure you want to delete the selected ${selectedRowKeys.length} tickets? This action cannot be undone.`,
      okText: 'Confirm Delete',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: async () => {
        try {
          await batchDeleteMutation.mutateAsync(selectedRowKeys as string[]);
          setSelectedRowKeys([]);
        } catch (error) {
          console.error('Delete failed:', error);
        }
      },
    });
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
      message.success('数据刷新成功');
    } catch (error) {
      console.error('Refresh failed:', error);
      message.error('数据刷新失败，请重试');
    }
  }, [refetchTickets, refetchStats]);

  const handleModalSubmit = useCallback(
    async (values: any) => {
      try {
        if (editingTicket) {
          await updateTicketMutation.mutateAsync({
            id: editingTicket.id,
            data: values,
          });
        } else {
          await createTicketMutation.mutateAsync(values);
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

  // 错误状态
  if (ticketsError || statsError) {
    const error = ticketsError || statsError;
    return (
      <LoadingEmptyError
        state='error'
        error={{
          title: 'Loading Failed',
          description: error?.message || 'Unknown error occurred',
          actionText: 'Retry',
          onAction: handleRefresh,
        }}
      />
    );
  }

  return (
    <div className='space-y-4'>
      {/* 统计卡片 - 懒加载 */}
      <LazyWrapper fallback={<LoadingSpinner />}>
        <TicketStats stats={stats} />
      </LazyWrapper>

      {/* 筛选器 - 懒加载 */}
      <LazyWrapper fallback={<LoadingSpinner />}>
        <TicketFilters
          filters={componentFilters}
          onFilterChange={handleFilterChange}
          onSearch={handleSearch}
          onRefresh={handleRefresh}
          loading={ticketsLoading}
        />
      </LazyWrapper>

      {/* 优化的工单列表 */}
      <OptimizedTicketList
        tickets={tickets}
        loading={ticketsLoading}
        pagination={pagination}
        selectedRowKeys={selectedRowKeys}
        onPaginationChange={handlePaginationChange}
        onRowSelectionChange={handleRowSelectionChange}
        onEditTicket={handleEditTicket}
        onViewActivity={handleViewActivity}
        onCreateTicket={handleCreateTicket}
        onBatchDelete={handleBatchDelete}
      />

      {/* 工单关联 - 懒加载 */}
      <LazyWrapper fallback={<LoadingSpinner />}>
        <TicketAssociation />
      </LazyWrapper>

      {/* 满意度仪表板 - 懒加载 */}
      <LazyWrapper fallback={<LoadingSpinner />}>
        <SatisfactionDashboard />
      </LazyWrapper>

      {/* 模态框 */}
      <TicketModal
        visible={modalVisible}
        editingTicket={editingTicket}
        onCancel={handleModalCancel}
        onSubmit={handleModalSubmit}
        loading={createTicketMutation.isPending || updateTicketMutation.isPending}
      />

      <TicketTemplateModal
        visible={templateModalVisible}
        onCancel={() => setTemplateModalVisible(false)}
      />
    </div>
  );
}
