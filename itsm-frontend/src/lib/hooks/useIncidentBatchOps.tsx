'use client';

import { useCallback, useState } from 'react';
import { App, Form } from 'antd';
import { UserPlus, CheckCircle, XCircle, Trash2 } from 'lucide-react';

import { IncidentAPI } from '@/lib/api/incident-api';
import { UserApi } from '@/lib/api/user-api';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';
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
      failPrefix = 'Partial failure'
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
          message.success(`${successMsg}：${successCount} items`);
        } else if (successCount === 0) {
          message.error(`${failPrefix}：${failedIds.length} All items failed`);
        } else {
          message.warning(`${failPrefix}：Success ${successCount} Item, failed ${failedIds.length} items`);
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
        const res = await UserApi.getUsers({ pageSize: BACKEND_MAX_PAGE_SIZE });
        setAssignUserOptions(
          (res.items || []).map(u => ({ label: u.name || u.username, value: u.id }))
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
      id => IncidentAPI.resolveIncident(id, { resolution: 'Batch solution' }),
      'Batch solution successful'
    );
  }, [selectedRowKeys, runBatch]);

  const handleBatchClose = useCallback(async () => {
    await runBatch(
      selectedRowKeys,
      id => IncidentAPI.closeIncident(id, { closeNotes: 'Batch Close' }),
      'Batch closing successful'
    );
  }, [selectedRowKeys, runBatch]);

  const handleBatchDelete = useCallback(async () => {
    await runBatch(selectedRowKeys, id => IncidentAPI.deleteIncident(id), 'Batch deletion successful');
  }, [selectedRowKeys, runBatch]);

  // Note: assign-confirm logic lives in the page (handleAssignConfirm) so it can
  // keep the modal open on validation failure. Do not re-add handleBatchAssign
  // here without removing the page-side duplicate.

  const batchActions: BatchAction[] = [
    {
      key: 'assign',
      label: 'Batch dispatch',
      icon: <UserPlus size={14} />,
      onClick: openAssignModal,
      type: 'primary' as const,
    },
    {
      key: 'resolve',
      label: 'Batch solution',
      icon: <CheckCircle size={14} />,
      onClick: handleBatchResolve,
    },
    {
      key: 'close',
      label: 'Batch Close',
      icon: <XCircle size={14} />,
      onClick: handleBatchClose,
    },
    {
      key: 'delete',
      label: 'Batch Delete',
      icon: <Trash2 size={14} />,
      danger: true,
      confirmTitle: `Confirm to delete the selected ${selectedRowKeys.length} events? This action is irreversible`,
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
