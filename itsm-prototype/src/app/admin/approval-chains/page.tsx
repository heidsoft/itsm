'use client';

import React, { useState, useCallback, useEffect } from 'react';
import { Button, message, Modal } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { ApprovalChainStatsCards } from './components/ApprovalChainStats';
import { ApprovalChainFilters } from './components/ApprovalChainFilters';
import { ApprovalChainTable } from './components/ApprovalChainTable';
import { ApprovalChainModal } from './components/ApprovalChainModal';
import {
  ApprovalChain,
  ApprovalChainFilters as Filters,
  ApprovalChainStats,
} from '@/types/approval-chain';
import { httpClient } from '@/lib/api/http-client';
// import { handleError } from '@/lib/error-handler';

// 临时错误处理函数
const handleError = (error: unknown, message?: string) => {
  console.error(message || '操作失败', error);
  if (error instanceof Error) {
    message.error(error.message);
  } else {
    message.error(message || '操作失败');
  }
};

export default function ApprovalChainsPage() {
  // 状态管理
  const [chains, setChains] = useState<ApprovalChain[]>([]);
  const [stats, setStats] = useState<ApprovalChainStats>({
    total: 0,
    active: 0,
    inactive: 0,
    totalSteps: 0,
    avgStepsPerChain: 0,
  });
  const [loading, setLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(false);

  // 筛选和分页
  const [filters, setFilters] = useState<Filters>({});
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });

  // 选择状态
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 模态框状态
  const [modalVisible, setModalVisible] = useState(false);
  const [editingChain, setEditingChain] = useState<ApprovalChain | null>(null);

  // 加载审批链列表
  const loadChains = useCallback(async () => {
    try {
      setLoading(true);

      const params = {
        page: pagination.current,
        page_size: pagination.pageSize,
        ...filters,
      };

      const response = await httpClient.getPaginated<ApprovalChain>(
        '/api/v1/approval-chains',
        params
      );

      setChains(response.data);
      setPagination(prev => ({
        ...prev,
        total: response.total || response.data.length,
      }));
    } catch (error) {
      handleError(error, '加载审批链列表失败');
    } finally {
      setLoading(false);
    }
  }, [pagination.current, pagination.pageSize, filters]);

  // 加载统计数据
  const loadStats = useCallback(async () => {
    try {
      setStatsLoading(true);

      const response = await httpClient.get<ApprovalChainStats>('/api/v1/approval-chains/stats');
      setStats(response);
    } catch (error) {
      handleError(error, '加载统计数据失败');
    } finally {
      setStatsLoading(false);
    }
  }, []);

  // 初始加载
  useEffect(() => {
    loadChains();
    loadStats();
  }, [loadChains, loadStats]);

  // 处理筛选变化
  const handleFilterChange = useCallback((newFilters: Filters) => {
    setFilters(newFilters);
    setPagination(prev => ({ ...prev, current: 1 }));
  }, []);

  // 处理分页变化
  const handlePaginationChange = useCallback((page: number, pageSize: number) => {
    setPagination(prev => ({ ...prev, current: page, pageSize }));
  }, []);

  // 处理行选择变化
  const handleRowSelectionChange = useCallback(
    (selectedRowKeys: React.Key[], selectedRows: ApprovalChain[]) => {
      setSelectedRowKeys(selectedRowKeys);
    },
    []
  );

  // 处理创建审批链
  const handleCreateChain = useCallback(() => {
    setEditingChain(null);
    setModalVisible(true);
  }, []);

  // 处理编辑审批链
  const handleEditChain = useCallback((chain: ApprovalChain) => {
    setEditingChain(chain);
    setModalVisible(true);
  }, []);

  // 处理查看审批链
  const handleViewChain = useCallback((chain: ApprovalChain) => {
    // 这里可以打开详情页面或模态框
    message.info(`查看审批链: ${chain.name}`);
  }, []);

  // 处理删除审批链
  const handleDeleteChain = useCallback(
    (chain: ApprovalChain) => {
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除审批链"${chain.name}"吗？此操作不可撤销。`,
        okText: '确认删除',
        okType: 'danger',
        cancelText: '取消',
        onOk: async () => {
          try {
            await httpClient.delete(`/api/v1/approval-chains/${chain.id}`);
            message.success('删除成功');
            loadChains();
            loadStats();
          } catch (error) {
            handleError(error, '删除审批链失败');
          }
        },
      });
    },
    [loadChains, loadStats]
  );

  // 处理切换状态
  const handleToggleStatus = useCallback(
    async (chain: ApprovalChain) => {
      try {
        await httpClient.patch(`/api/v1/approval-chains/${chain.id}`, {
          isActive: !chain.isActive,
        });
        message.success(`${chain.isActive ? '停用' : '启用'}成功`);
        loadChains();
        loadStats();
      } catch (error) {
        handleError(error, '状态切换失败');
      }
    },
    [loadChains, loadStats]
  );

  // 处理复制审批链
  const handleCopyChain = useCallback(
    async (chain: ApprovalChain) => {
      try {
        await httpClient.post('/api/v1/approval-chains', {
          name: `${chain.name} - 副本`,
          description: chain.description,
          isActive: false,
          steps: chain.steps.map(step => ({
            stepOrder: step.stepOrder,
            stepName: step.stepName,
            approverType: step.approverType,
            approverId: step.approverId,
            approverName: step.approverName,
            isRequired: step.isRequired,
            timeoutHours: step.timeoutHours,
            conditions: step.conditions,
          })),
        });
        message.success('复制成功');
        loadChains();
        loadStats();
      } catch (error) {
        handleError(error, '复制审批链失败');
      }
    },
    [loadChains, loadStats]
  );

  // 处理批量删除
  const handleBatchDelete = useCallback(
    (ids: React.Key[]) => {
      Modal.confirm({
        title: '确认批量删除',
        content: `确定要删除选中的 ${ids.length} 个审批链吗？此操作不可撤销。`,
        okText: '确认删除',
        okType: 'danger',
        cancelText: '取消',
        onOk: async () => {
          try {
            await httpClient.batchOperation(
              '/api/v1/approval-chains/batch',
              'delete',
              ids as number[]
            );
            message.success('批量删除成功');
            setSelectedRowKeys([]);
            loadChains();
            loadStats();
          } catch (error) {
            handleError(error, '批量删除失败');
          }
        },
      });
    },
    [loadChains, loadStats]
  );

  // 处理模态框提交
  const handleModalSubmit = useCallback(
    async (data: any) => {
      try {
        if (editingChain) {
          await httpClient.put(`/api/v1/approval-chains/${editingChain.id}`, data);
          message.success('更新成功');
        } else {
          await httpClient.post('/api/v1/approval-chains', data);
          message.success('创建成功');
        }

        setModalVisible(false);
        setEditingChain(null);
        loadChains();
        loadStats();
      } catch (error) {
        handleError(error, editingChain ? '更新审批链失败' : '创建审批链失败');
      }
    },
    [editingChain, loadChains, loadStats]
  );

  // 处理模态框取消
  const handleModalCancel = useCallback(() => {
    setModalVisible(false);
    setEditingChain(null);
  }, []);

  // 处理刷新
  const handleRefresh = useCallback(() => {
    loadChains();
    loadStats();
  }, [loadChains, loadStats]);

  return (
    <div className='space-y-6'>
      {/* 统计卡片 */}
      <ApprovalChainStatsCards stats={stats} loading={statsLoading} />

      {/* 筛选器 */}
      <ApprovalChainFilters
        filters={filters}
        onFilterChange={handleFilterChange}
        onRefresh={handleRefresh}
        loading={loading}
      />

      {/* 表格 */}
      <ApprovalChainTable
        dataSource={chains}
        loading={loading}
        pagination={pagination}
        selectedRowKeys={selectedRowKeys}
        onPaginationChange={handlePaginationChange}
        onRowSelectionChange={handleRowSelectionChange}
        onEdit={handleEditChain}
        onView={handleViewChain}
        onDelete={handleDeleteChain}
        onToggleStatus={handleToggleStatus}
        onCopy={handleCopyChain}
        onBatchDelete={handleBatchDelete}
      />

      {/* 模态框 */}
      <ApprovalChainModal
        visible={modalVisible}
        editingChain={editingChain}
        onCancel={handleModalCancel}
        onSubmit={handleModalSubmit}
        loading={loading}
      />
    </div>
  );
}
