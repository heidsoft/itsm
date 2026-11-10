'use client';

import React, { useState, useCallback, useEffect, useRef } from 'react';
import { Alert, App, Tag, Typography } from 'antd';
import { ApprovalChainStatsCards } from './components/ApprovalChainStats';
import { ApprovalChainFilters } from './components/ApprovalChainFilters';
import { ApprovalChainTable } from './components/ApprovalChainTable';
import { ApprovalChainModal } from './components/ApprovalChainModal';
import type {
  ApprovalChain,
  ApprovalChainFilters as Filters,
  ApprovalChainStats,
  ApprovalStep,
  ApprovalType,
  FallbackAction,
} from '@/types/approval-chain';
import {
  APPROVAL_TYPE_LABELS,
  FALLBACK_ACTION_LABELS,
  entityTypeLabel,
} from '@/types/approval-chain';
import { httpClient } from '@/lib/api/http-client';
import { UserApi } from '@/lib/api/user-api';
import { RoleAPI } from '@/lib/api/role-api';

const { Text } = Typography;

type BackendApprovalStep = {
  level?: number;
  approverId?: number;
  role?: string;
  name?: string;
  isRequired?: boolean;
  approvalType?: ApprovalType;
  threshold?: number;
  fallbackAction?: FallbackAction;
  fallbackApproverId?: number;
  fallbackRole?: string;
  conditionPriorities?: string[];
  conditionAmountMin?: number;
  conditionAmountMax?: number;
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
    level: step.level || index + 1,
    stepOrder: index + 1,
    stepName: step.name || `Step ${index + 1}`,
    approverType: step.role ? 'role' : 'user',
    approverId: step.approverId || 0,
    approverName: step.role || '',
    isRequired: step.isRequired !== false,
    approvalType: step.approvalType,
    threshold: step.threshold,
    fallbackAction: step.fallbackAction,
    fallbackApproverId: step.fallbackApproverId,
    fallbackRole: step.fallbackRole,
    conditionPriorities: step.conditionPriorities,
    conditionAmountMin: step.conditionAmountMin,
    conditionAmountMax: step.conditionAmountMax,
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
    // 同层级步骤由后端合并为一个会签/或签层，level 必须按用户配置保留
    level: step.level || index + 1,
    approverId: step.approverType === 'user' && step.approverId > 0 ? step.approverId : undefined,
    role: step.approverType === 'role' ? step.approverName : '',
    name: step.stepName || `Step ${index + 1}`,
    isRequired: step.isRequired !== false,
    approvalType: step.approvalType,
    threshold: step.threshold,
    fallbackAction: step.fallbackAction,
    fallbackApproverId: step.fallbackApproverId,
    fallbackRole: step.fallbackRole,
    conditionPriorities: step.conditionPriorities,
    conditionAmountMin: step.conditionAmountMin,
    conditionAmountMax: step.conditionAmountMax,
  })),
});

export default function ApprovalChainsPage() {
  const { message, modal } = App.useApp();

  // 临时错误处理函数
  const handleError = (error: unknown, errorMessage?: string) => {
    if (error instanceof Error && error.message) {
      message.error(error.message);
    } else {
      message.error(errorMessage ? errorMessage : 'Operation failed');
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

  // 加载审批链列表：httpClient.get 已拆掉 { code, message, data } 外壳，
  // 拿到的就是分页对象本身，直接读 items/total，不能再套一层 response.data。
  const loadChains = useCallback(async () => {
    try {
      setLoading(true);

      const query = new URLSearchParams({
        page: String(pagination.current),
        pageSize: String(pagination.pageSize),
      });
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          query.append(key, String(value));
        }
      });

      const response = await httpClient.get<{
        items?: BackendApprovalChain[];
        total?: number;
      }>(`/api/v1/approval-chains?${query.toString()}`);

      const items = response?.items ?? [];
      setChains(items.map(normalizeChain));
      setPagination(prev => ({
        ...prev,
        total: response?.total ?? items.length,
      }));
    } catch (error) {
      handleError(error, 'Failed to load approval chain list');
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
      handleError(error, 'Failed to load statistics');
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

  // 审批人/角色名称映射：仅在首次查看详情时拉取并缓存，避免列表页常态 N+1 请求
  const referenceRef = useRef<{ users: Record<number, string>; roles: Record<string, string> } | null>(
    null
  );

  const loadReferenceData = useCallback(async () => {
    if (referenceRef.current) return referenceRef.current;
    try {
      const [userResponse, roleResponse] = await Promise.all([
        UserApi.getUsers({ page: 1, pageSize: 200 }),
        RoleAPI.getRoles({ page: 1, pageSize: 200 }),
      ]);
      const users: Record<number, string> = {};
      for (const user of userResponse.items || []) {
        users[user.id] = user.name || user.username;
      }
      const roles: Record<string, string> = {};
      for (const role of roleResponse.items) {
        roles[role.code || role.name] = role.name;
      }
      referenceRef.current = { users, roles };
    } catch {
      // 名称解析失败时回退显示 ID / 角色 code，不阻断查看详情
      referenceRef.current = { users: {}, roles: {} };
    }
    return referenceRef.current;
  }, []);

  const approverLabel = useCallback(
    (
      step: ApprovalStep,
      maps: { users: Record<number, string>; roles: Record<string, string> }
    ): string => {
      if (step.approverType === 'role') {
        return maps.roles[step.approverName] || step.approverName || 'No approval role specified';
      }
      return maps.users[step.approverId] || (step.approverId ? `User #${step.approverId}` : 'No approver specified');
    },
    []
  );

  // 处理查看审批链
  const handleViewChain = useCallback(
    async (chain: ApprovalChain) => {
      const maps = await loadReferenceData();
      modal.info({
        title: 'Approval chain details',
        width: 640,
        content: (
          <div className="space-y-3 pt-2">
            <p>
              <Text strong>Name:</Text>
              {chain.name}
            </p>
            <p>
              <Text strong>Applicable objects:</Text>
              <Tag>{entityTypeLabel(chain.entityType)}</Tag>
            </p>
            <p>
              <Text strong>Status:</Text>
              <Tag color={chain.isActive ? 'green' : 'red'}>{chain.isActive ? 'Enabled' : 'Disable'}</Tag>
            </p>
            <p>
              <Text strong>Description:</Text>
              {chain.description || '-'}
            </p>
            <div>
              <Text strong>Approval steps:</Text>
              <ol className="mt-2 list-decimal pl-5">
                {(chain.steps ?? []).map(step => (
                  <li key={`${chain.id}-${step.stepOrder}`}>
                    {step.stepName}(level {step.level}）：{approverLabel(step, maps)}
                    {step.approvalType
                      ? ` · ${APPROVAL_TYPE_LABELS[step.approvalType as ApprovalType] ?? step.approvalType}`
                      : ''}
                    {step.threshold ? ` · Threshold ${step.threshold}` : ''}
                    {step.fallbackAction
                      ? ` · Tell the truth:${FALLBACK_ACTION_LABELS[step.fallbackAction as FallbackAction]}`
                      : ''}
                  </li>
                ))}
              </ol>
            </div>
          </div>
        ),
      });
    },
    [loadReferenceData, approverLabel, modal]
  );

  // 处理删除审批链
  const handleDeleteChain = useCallback(
    (chain: ApprovalChain) => {
      modal.confirm({
        title: 'Confirm Delete',
        content: `Are you sure you want to delete the approval chain"${chain.name}"? This operation cannot be undone.`,
        okText: 'Confirm Delete',
        okType: 'danger',
        cancelText: 'Cancel',
        onOk: async () => {
          try {
            await httpClient.delete(`/api/v1/approval-chains/${chain.id}`);
            message.success('Deleted');
            loadChains();
            loadStats();
          } catch (error) {
            handleError(error, 'Failed to delete approval chain');
          }
        },
      });
    },
    [loadChains, loadStats, modal, message]
  );

  // 处理切换状态
  const handleToggleStatus = useCallback(
    async (chain: ApprovalChain) => {
      try {
        await httpClient.put(`/api/v1/approval-chains/${chain.id}`, {
          ...toBackendPayload(chain, chain),
          status: chain.isActive ? 'inactive' : 'active',
        });
        message.success(`${chain.isActive ? 'Disable' : 'Enabled'} Success`);
        loadChains();
        loadStats();
      } catch (error) {
        handleError(error, 'Status switching failed');
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
          name: `${chain.name} - copy`,
          status: 'inactive',
        });
        message.success('Copy successful');
        loadChains();
        loadStats();
      } catch (error) {
        handleError(error, 'Failed to copy approval chain');
      }
    },
    [loadChains, loadStats]
  );

  // 处理批量删除：部分失败必须保留失败项供重试（Promise.all 会在首个失败时丢掉其余结果）
  const handleBatchDelete = useCallback(
    (ids: React.Key[]) => {
      modal.confirm({
        title: 'Confirm batch deletion',
        content: `Are you sure you want to delete the selected ${ids.length} Is there an approval chain? This action cannot be undone.`,
        okText: 'Confirm Delete',
        okType: 'danger',
        cancelText: 'Cancel',
        onOk: async () => {
          const results = await Promise.allSettled(
            ids.map(id =>
              httpClient
                .delete(`/api/v1/approval-chains/${String(id)}`)
                .then(() => ({ id, ok: true as const }))
                .catch((error: unknown) => ({ id, ok: false as const, error }))
            )
          );
          const failedIds: React.Key[] = [];
          let firstError: unknown = null;
          results.forEach((result, index) => {
            if (result.status === 'rejected') {
              failedIds.push(ids[index]);
              firstError = firstError ?? result.reason;
              return;
            }
            if (result.value.ok) return;
            // 用原始 ids 回填，避免从结果对象再取一次 key 造成类型扩散
            failedIds.push(ids[index]);
            firstError = firstError ?? result.value.error;
          });

          const successCount = ids.length - failedIds.length;
          if (successCount > 0) {
            message.success(`Deleted ${successCount} approval chain`);
          }
          if (failedIds.length > 0) {
            const reason =
              firstError instanceof Error && firstError.message ? `：${firstError.message}` : '';
            message.error(`${failedIds.length} Deletion of approval chain failed, selected items have been retained and can be tried again ${reason}`);
            // 仅保留失败项，成功项不再参与后续批量操作
            setSelectedRowKeys(failedIds);
          } else {
            setSelectedRowKeys([]);
          }
          loadChains();
          loadStats();
        },
      });
    },
    [loadChains, loadStats, modal, message]
  );

  // 处理模态框提交
  const handleModalSubmit = useCallback(
    async (data: unknown) => {
      try {
        const payload = toBackendPayload(data as ApprovalChainSubmitData, editingChain || undefined);
        if (editingChain) {
          await httpClient.put(`/api/v1/approval-chains/${editingChain.id}`, payload);
          message.success('Updated');
        } else {
          await httpClient.post('/api/v1/approval-chains', payload);
          message.success('Created');
        }

        setModalVisible(false);
        setEditingChain(null);
        loadChains();
        loadStats();
      } catch (error) {
        handleError(error, editingChain ? 'Failed to update approval chain' : 'Failed to create approval chain');
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
      {/* 定位说明：本页只配置 BPMN 审批节点的「审批人规则」，流程流转/分支/超时由流程设计器编排 */}
      <Alert
        type="info"
        showIcon
        title="This page configures the approver rules of the BPMN approval node (approver/countersign/or sign/confirm). Please arrange the flow, branches and timeout of the process in the &quot;Process Designer&quot;."
        className="mb-4"
      />

      {/* 统计卡片 */}
      <ApprovalChainStatsCards stats={stats} loading={statsLoading} />

      {/* 筛选器 */}
      <ApprovalChainFilters
        filters={filters}
        onFilterChange={handleFilterChange}
        onRefresh={handleRefresh}
        onCreate={handleCreateChain}
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
