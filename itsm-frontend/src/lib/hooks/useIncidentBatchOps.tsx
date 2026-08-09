'use client';

import { useCallback, useState } from 'react';
import { App, Form } from 'antd';
import { UserPlus, CheckCircle, XCircle, Trash2 } from 'lucide-react';

import { IncidentAPI } from '@/lib/api/incident-api';
import { UserApi } from '@/lib/api/user-api';
import type { BatchAction } from '@/components/business/BatchActionBar';

export interface UseIncidentBatchOpsOptions {
  /** Called after a successful batch run to refresh the list. */
  readonly onAfterBatch: () => Promise<void> | void;
}

export interface UseIncidentBatchOpsResult {
  readonly selectedRowKeys: React.Key[];
  readonly setSelectedRowKeys: (keys: React.Key[]) => void;
  readonly batchLoading: boolean;
  readonly assignModalOpen: boolean;
  readonly openAssignModal: () => Promise<void>;
  readonly closeAssignModal: () => void;
  readonly assignForm: ReturnType<typeof Form.useForm<{ assigneeId: number }>>[0];
  readonly batchActions: BatchAction[];
  readonly runBatch: (
    ids: readonly React.Key[],
    handler: (id: number) => Promise<unknown>,
    successMsg: string,
    failPrefix?: string
  ) => Promise<{ successCount: number; failedIds: React.Key[] }>;
  readonly assignUserOptions: { label: string; value: number }[];
}

/**
 * Owns the row-selection state and the multi-action batch pipeline for the
 * incidents page. Includes the assign-modal state and the lazy user list
 * fetch the modal needs.
 */
export function useIncidentBatchOps({
  onAfterBatch,
}: UseIncidentBatchOpsOptions): UseIncidentBatchOpsResult {
  const { message } = App.useApp();
  const [assignForm] = Form.useForm<{ assigneeId: number }>();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [batchLoading, setBatchLoading] = useState(false);
  const [assignModalOpen, setAssignModalOpen] = useState(false);
  const [assignUserOptions, setAssignUserOptions] = useState<{ label: string; value: number }[]>(
    []
  );

  // H6: limit concurrent in-flight requests to avoid exhausting the browser's
  // per-origin connection pool (~6) and triggering false rate-limit failures.
  // H2: return per-id success/failure detail so the caller can keep failed
  // rows selected for retry instead of nuking the whole selection.
  const BATCH_CONCURRENCY = 5;

  const runBatch = useCallback(
    async (
      ids: readonly React.Key[],
      handler: (id: number) => Promise<unknown>,
      successMsg: string,
      failPrefix = '部分失败'
    ): Promise<{ successCount: number; failedIds: React.Key[] }> => {
      if (ids.length === 0) {
        return { successCount: 0, failedIds: [] };
      }
      setBatchLoading(true);
      const failedIds: React.Key[] = [];
      let successCount = 0;

      try {
        // Simple pool: keep at most BATCH_CONCURRENCY requests in flight.
        let cursor = 0;
        const workers = Array.from(
          { length: Math.min(BATCH_CONCURRENCY, ids.length) },
          async () => {
            while (cursor < ids.length) {
              const idx = cursor++;
              const id = ids[idx];
              if (id === undefined) return;
              try {
                await handler(Number(id));
                successCount++;
              } catch {
                failedIds.push(id);
              }
            }
          }
        );
        await Promise.allSettled(workers);

        if (failedIds.length === 0) {
          message.success(`${successMsg}：${successCount} 项`);
        } else if (successCount === 0) {
          message.error(`${failPrefix}：${failedIds.length} 项全部失败`);
        } else {
          message.warning(`${failPrefix}：成功 ${successCount} 项，失败 ${failedIds.length} 项`);
        }
        // Keep failed ids selected so the operator can retry; clear only on
        // full success. Use a Set for O(1) membership when the failed list is
        // large.
        if (failedIds.length === 0) {
          setSelectedRowKeys([]);
        } else {
          const failedSet = new Set(failedIds);
          setSelectedRowKeys(prev => prev.filter(k => failedSet.has(k)));
        }
        await onAfterBatch();
        return { successCount, failedIds };
      } finally {
        setBatchLoading(false);
      }
    },
    [message, onAfterBatch]
  );

  const openAssignModal = useCallback(async () => {
    assignForm.resetFields();
    setAssignModalOpen(true);
    if (assignUserOptions.length === 0) {
      try {
        const res = await UserApi.getUsers({ pageSize: 100 });
        setAssignUserOptions(
          (res.users || []).map(u => ({ label: u.name || u.username, value: u.id }))
        );
      } catch (error) {
        console.warn('load users for assign failed', error);
      }
    }
  }, [assignForm, assignUserOptions.length]);

  const closeAssignModal = useCallback(() => {
    setAssignModalOpen(false);
  }, []);

  const handleBatchResolve = useCallback(async () => {
    await runBatch(
      selectedRowKeys,
      id => IncidentAPI.resolveIncident(id, { resolution: '批量解决' }),
      '批量解决成功'
    );
  }, [selectedRowKeys, runBatch]);

  const handleBatchClose = useCallback(async () => {
    await runBatch(
      selectedRowKeys,
      id => IncidentAPI.closeIncident(id, { closeNotes: '批量关闭' }),
      '批量关闭成功'
    );
  }, [selectedRowKeys, runBatch]);

  const handleBatchDelete = useCallback(async () => {
    await runBatch(selectedRowKeys, id => IncidentAPI.deleteIncident(id), '批量删除成功');
  }, [selectedRowKeys, runBatch]);

  // Note: assign-confirm logic lives in the page (handleAssignConfirm) so it can
  // keep the modal open on validation failure. Do not re-add handleBatchAssign
  // here without removing the page-side duplicate.

  const batchActions: BatchAction[] = [
    {
      key: 'assign',
      label: '批量分派',
      icon: <UserPlus size={14} />,
      onClick: openAssignModal,
      type: 'primary' as const,
    },
    {
      key: 'resolve',
      label: '批量解决',
      icon: <CheckCircle size={14} />,
      onClick: handleBatchResolve,
    },
    {
      key: 'close',
      label: '批量关闭',
      icon: <XCircle size={14} />,
      onClick: handleBatchClose,
    },
    {
      key: 'delete',
      label: '批量删除',
      icon: <Trash2 size={14} />,
      danger: true,
      confirmTitle: `确定删除选中的 ${selectedRowKeys.length} 个事件？此操作不可撤销`,
      onClick: handleBatchDelete,
      overflow: true,
    },
  ];

  return {
    selectedRowKeys,
    setSelectedRowKeys,
    batchLoading,
    assignModalOpen,
    openAssignModal,
    closeAssignModal,
    assignForm,
    batchActions,
    runBatch,
    assignUserOptions,
  };
}
