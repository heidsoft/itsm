import { useCallback, useRef, useState } from 'react';

export interface BatchDecisionPayload {
  action: 'approve' | 'reject';
  comment: string;
}

export interface BatchDecisionResult {
  total: number;
  successCount: number;
  failedIds: number[];
  errors: Record<number, string>;
}

type SubmitFn = (taskId: number, payload: BatchDecisionPayload) => Promise<void>;

// 浏览器单 origin 连接池约 6，批量 fan-out 必须限流，否则数百条同时 fire
// 会耗尽连接、触发 429 被误判为业务失败（Pattern E）。
const BATCH_CONCURRENCY = 5;

/**
 * 批量审批决策执行器（Pattern D/E）。
 *
 * 用 Promise.allSettled 语义收集 per-id 明细：部分失败时返回 failedIds/errors，
 * 由调用方保留失败项选中以便重试，绝不因个别失败丢弃整批结果。
 * runBatch 身份稳定（空 deps + ref），可安全放进调用方 effect deps。
 */
export function useBatchTaskDecision(submit: SubmitFn) {
  const submitRef = useRef(submit);
  submitRef.current = submit;
  const [running, setRunning] = useState(false);

  const runBatch = useCallback(
    async (taskIds: number[], payload: BatchDecisionPayload): Promise<BatchDecisionResult> => {
      setRunning(true);
      const errors: Record<number, string> = {};
      const failedIds: number[] = [];
      let successCount = 0;
      const queue = [...taskIds];

      const worker = async () => {
        for (;;) {
          const id = queue.shift();
          if (id === undefined) {
            return;
          }
          try {
            await submitRef.current(id, payload);
            successCount += 1;
          } catch (e) {
            failedIds.push(id);
            errors[id] = e instanceof Error ? e.message : String(e);
          }
        }
      };

      try {
        const workers = Array.from({ length: Math.min(BATCH_CONCURRENCY, taskIds.length) }, () => worker());
        await Promise.all(workers);
      } finally {
        setRunning(false);
      }

      return { total: taskIds.length, successCount, failedIds, errors };
    },
    []
  );

  return { running, runBatch };
}