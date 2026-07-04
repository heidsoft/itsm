'use client';

import React, { useState, useCallback, useEffect } from 'react';
import { App, Modal, Tag, Typography } from 'antd';
import { ApprovalChainStatsCards } from './components/ApprovalChainStats';
import { ApprovalChainFilters } from './components/ApprovalChainFilters';
import { ApprovalChainTable } from './components/ApprovalChainTable';
import { ApprovalChainModal } from './components/ApprovalChainModal';
import type {
  ApprovalChain,
  ApprovalChainFilters as Filters,
  ApprovalChainStats,
} from '@/types/approval-chain';
import { httpClient } from '@/lib/api/http-client';

const { Text } = Typography;

type BackendApprovalStep = {
  level?: number;
  approverId?: number;
  role?: string;
  name?: string;
  isRequired?: boolean;
};

type BackendApprovalChain = {
  id: number;
  name: string;
  description?: string;
  entityType?: string;
  chain?: BackendApprovalStep[];
  status?: string;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
};

type ApprovalChainSubmitData = {
  name?: string;
  description?: string;
  entityType?: string;
  isActive?: boolean;
  steps?: ApprovalChain['steps'];
};

const normalizeChain = (chain: BackendApprovalChain): ApprovalChain => ({
  id: chain.id,
  name: chain.name,
  description: chain.description,
  entityType: chain.entityType || 'ticket',
  status: chain.status || 'active',
  isActive: chain.status !== 'inactive',
  tenantId: chain.tenantId,
  createdAt: chain.createdAt,
  updatedAt: chain.updatedAt,
  steps: (chain.chain || []).map((step, index) => ({
    id: index + 1,
    chainId: chain.id,
    stepOrder: step.level || index + 1,
    stepName: step.name || `步骤 ${index + 1}`,
    approverType: step.role ? 'role' : 'user',
    approverId: step.approverId || 0,
    approverName: step.name || step.role || '',
    isRequired: step.isRequired !== false,
    createdAt: chain.createdAt,
    updatedAt: chain.updatedAt,
  })),
});

const toBackendPayload = (data: ApprovalChainSubmitData, fallback?: ApprovalChain) => ({
  name: data.name || fallback?.name || '',
  description: data.description || fallback?.description || '',
  entityType: data.entityType || fallback?.entityType || 'ticket',
  status: data.isActive ?? fallback?.isActive ? 'active' : 'inactive',
  chain: (data.steps || fallback?.steps || []).map((step, index) => ({
    level: index + 1,
    approver_id: step.approverId || undefined,
    role: step.approverType === 'role' ? step.approverName : '',
    name: step.approverName || step.stepName,
    isRequired: step.isRequired,
  })),
});

export default function ApprovalChainsPage() {
  const { message } = App.useApp();

  // 临时错误处理函数
  const handleError = (error: unknown, errorMessage?: string) => {
    if (error instanceof Error) {
      message.error(error.message);
    } else {
      message.error(errorMessage || '操作失败');
    }
  };

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

      setChains((response.data as unknown as BackendApprovalChain[]).map(normalizeChain));
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
    Modal.info({
      title: '审批链详情',
      width: 640,
      content: (
        <div className="space-y-3 pt-2">
          <p>
            <Text strong>名称：</Text>
            {chain.name}
          </p>
          <p>
            <Text strong>适用对象：</Text>
            <Tag>{chain.entityType || 'ticket'}</Tag>
          </p>
          <p>
            <Text strong>状态：</Text>
            <Tag color={chain.isActive ? 'green' : 'red'}>{chain.isActive ? '启用' : '停用'}</Tag>
          </p>
          <p>
            <Text strong>描述：</Text>
            {chain.description || '-'}
          </p>
          <div>
            <Text strong>审批步骤：</Text>
            <ol className="mt-2 list-decimal pl-5">
              {chain.steps.map(step => (
                <li key={`${chain.id}-${step.stepOrder}`}>
                  {step.stepName}：{step.approverName || '未指定审批人'}
                </li>
              ))}
            </ol>
          </div>
        </div>
      ),
    });
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
        await httpClient.put(`/api/v1/approval-chains/${chain.id}`, {
          ...toBackendPayload(chain, chain),
          status: chain.isActive ? 'inactive' : 'active',
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
          ...toBackendPayload(chain, chain),
          name: `${chain.name} - 副本`,
          status: 'inactive',
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
            await Promise.all(
              ids.map(id => httpClient.delete(`/api/v1/approval-chains/${String(id)}`))
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
    async (data: unknown) => {
      try {
        const payload = toBackendPayload(data as ApprovalChainSubmitData, editingChain || undefined);
        if (editingChain) {
          await httpClient.put(`/api/v1/approval-chains/${editingChain.id}`, payload);
          message.success('更新成功');
        } else {
          await httpClient.post('/api/v1/approval-chains', payload);
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
    <div className="space-y-6">
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
