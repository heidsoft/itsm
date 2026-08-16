'use client';

/**
 * Ticket multi-level approval chain panel with workflow awareness.
 *
 * Compared to ApprovalTimeline, this component also displays:
 * 1. Workflow overview Steps — every node of the matched workflow is rendered as a Step.
 *    Each level is annotated with approval mode, approver type, and timeout.
 * 2. Approval Timeline (records grouped by level).
 * 3. Action area — when the current user is the pending approver, three buttons (approve/reject/delegate).
 * 4. Integrates TicketApprovalApi.getWorkflows / getApprovalRecords / submitApproval.
 *
 * If no workflow is bound for the ticket, the panel gracefully degrades to
 * Timeline + action area.
 */

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Card, Steps, Tag, Space, Typography, App, Empty, Divider, Skeleton, Alert } from 'antd';
import { CheckCircle, Timer, Users, User as UserIcon, GitBranch } from 'lucide-react';
import {
  TicketApprovalApi,
  type ApprovalWorkflow,
  type ApprovalNode,
  type ApprovalRecord,
} from '@/lib/api/ticket-approval-api';
import { ApprovalTimeline } from './ApprovalTimeline';
import type { ApprovalStep, ApprovalStepStatus } from './types';
import { useI18n } from '@/lib/i18n/useI18n';

const { Text } = Typography;

export interface ApprovalWorkflowPanelProps {
  ticketId: number;
  ticketType?: string;
  priority?: string;
  currentUserId?: number;
  isTicketFinal: boolean;
  onRefresh?: () => void;
  formatDateTime?: (s: string) => string;
}

interface LevelState {
  level: number;
  status: ApprovalStepStatus;
  records: ApprovalRecord[];
}

/**
 * Compute a single level's aggregate status:
 * - any rejected -> rejected
 * - any pending -> pending
 * - all approved -> approved
 * - any delegated (no others) -> delegated
 * - all timeout -> timeout
 * - no records -> pending (not started)
 */
function computeLevelStatus(records: ApprovalRecord[]): ApprovalStepStatus {
  if (records.length === 0) return 'pending';
  if (records.some((r) => r.status === 'rejected')) return 'rejected';
  if (records.some((r) => r.status === 'pending')) return 'pending';
  if (records.every((r) => r.status === 'approved')) return 'approved';
  if (records.some((r) => r.status === 'delegated')) return 'delegated';
  if (records.every((r) => r.status === 'timeout')) return 'timeout';
  return 'pending';
}

export const ApprovalWorkflowPanel: React.FC<ApprovalWorkflowPanelProps> = ({
  ticketId,
  ticketType,
  priority,
  currentUserId,
  isTicketFinal,
  onRefresh,
  formatDateTime,
}) => {
  const { t } = useI18n();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [workflow, setWorkflow] = useState<ApprovalWorkflow | null>(null);
  const [records, setRecords] = useState<ApprovalRecord[]>([]);

  const modeLabels = useMemo(
    () =>
      ({
        sequential: t('detailTabs.modeSequential'),
        parallel: t('detailTabs.modeParallel'),
        any: t('detailTabs.modeAny'),
        all: t('detailTabs.modeAll'),
      }) as Record<ApprovalNode['approvalMode'], string>,
    [t],
  );

  const modeColors: Record<ApprovalNode['approvalMode'], string> = useMemo(
    () => ({
      sequential: 'blue',
      parallel: 'purple',
      any: 'green',
      all: 'orange',
    }),
    [],
  );

  const approverTypeLabels = useMemo(
    () =>
      ({
        user: t('detailTabs.approverTypeUser'),
        role: t('detailTabs.approverTypeRole'),
        department: t('detailTabs.approverTypeDepartment'),
        dynamic: t('detailTabs.approverTypeDynamic'),
      }) as Record<ApprovalNode['approverType'], string>,
    [t],
  );

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      const [wfRes, recRes] = await Promise.all([
        TicketApprovalApi.getWorkflows({
          ticketType,
          priority,
          isActive: true,
          page: 1,
          pageSize: 50,
        }).catch(() => ({ items: [] as ApprovalWorkflow[], total: 0, page: 1, pageSize: 50 })),
        TicketApprovalApi.getApprovalRecords({ ticketId, page: 1, pageSize: 100 }).catch(() => ({
          items: [] as ApprovalRecord[],
          total: 0,
          page: 1,
          pageSize: 100,
        })),
      ]);

      setRecords(recRes.items || []);

      const wfIdFromRecord = (recRes.items || [])[0]?.workflowId;
      let matched: ApprovalWorkflow | null = null;
      if (wfIdFromRecord) {
        matched = (wfRes.items || []).find((w) => w.id === wfIdFromRecord) || null;
      }
      if (!matched && (ticketType || priority)) {
        matched =
          (wfRes.items || []).find(
            (w) =>
              (!w.ticketType || !ticketType || w.ticketType === ticketType) &&
              (!w.priority || !priority || w.priority === priority),
          ) || null;
      }
      setWorkflow(matched);
    } catch (e) {
      console.warn('load enhanced approval failed', e);
      message.error(t('detailTabs.workflowLoadFailed'));
    } finally {
      setLoading(false);
    }
  }, [ticketId, ticketType, priority, message, t]);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  const levelStates: LevelState[] = useMemo(() => {
    if (!workflow || !workflow.nodes || workflow.nodes.length === 0) {
      const map = new Map<number, ApprovalRecord[]>();
      for (const r of records) {
        const list = map.get(r.currentLevel) || [];
        list.push(r);
        map.set(r.currentLevel, list);
      }
      const levels = Array.from(map.keys()).sort((a, b) => a - b);
      return levels.map((lv) => ({
        level: lv,
        status: computeLevelStatus(map.get(lv) || []),
        records: map.get(lv) || [],
      }));
    }
    return workflow.nodes
      .slice()
      .sort((a, b) => a.level - b.level)
      .map((node) => {
        const rec = records.filter((r) => r.currentLevel === node.level);
        return {
          level: node.level,
          status: computeLevelStatus(rec),
          records: rec,
        };
      });
  }, [workflow, records]);

  const currentLevel = useMemo(() => {
    const pending = levelStates.find((l) => l.status === 'pending' && l.records.length > 0);
    return pending?.level;
  }, [levelStates]);

  const myApprovalId = useMemo(() => {
    if (!currentUserId || isTicketFinal) return null;
    const myRec = records.find(
      (r) => r.status === 'pending' && r.approverId === currentUserId,
    );
    return myRec?.id ?? null;
  }, [records, currentUserId, isTicketFinal]);

  const timelineSteps: ApprovalStep[] = useMemo(() => {
    return records
      .slice()
      .sort((a, b) => a.currentLevel - b.currentLevel || a.id - b.id)
      .map((r) => ({
        id: r.id,
        level: r.currentLevel,
        status: r.status,
        approverId: r.approverId,
        approverName: r.approverName,
        comment: r.comment,
        processedAt: r.processedAt,
        createdAt: r.createdAt,
      }));
  }, [records]);

  const submitApproval = async (
    action: 'approve' | 'reject' | 'delegate',
    payload: { comment: string; delegateToUserId?: number },
  ) => {
    if (!myApprovalId) {
      message.warning(t('detailTabs.noActionableApproval'));
      throw new Error('no pending approval');
    }
    await TicketApprovalApi.submitApproval({
      ticketId,
      approvalId: myApprovalId,
      action,
      comment: payload.comment,
      delegateToUserId: payload.delegateToUserId,
    });
    await loadAll();
    onRefresh?.();
  };

  const stepStatus = (
    s: ApprovalStepStatus,
  ): 'wait' | 'process' | 'finish' | 'error' => {
    switch (s) {
      case 'approved':
        return 'finish';
      case 'rejected':
      case 'timeout':
        return 'error';
      case 'pending':
        return 'process';
      case 'delegated':
      case 'skipped':
        return 'wait';
      default:
        return 'wait';
    }
  };

  const iconFor = (s: ApprovalStepStatus) => {
    if (s === 'approved') return <CheckCircle size={16} />;
    if (s === 'pending') return <Timer size={16} />;
    return undefined;
  };

  if (loading) {
    return (
      <div className="p-6">
        <Skeleton active paragraph={{ rows: 4 }} />
      </div>
    );
  }

  const hasWorkflowMeta = !!workflow && workflow.nodes && workflow.nodes.length > 0;
  const hasRecords = records.length > 0;

  return (
    <div className="p-6 space-y-4">
      {hasWorkflowMeta && workflow && (
        <Card
          size="small"
          title={
            <Space size={6}>
              <GitBranch size={14} />
              <Text strong>{workflow.name}</Text>
              <Tag color="blue">
                {t('detailTabs.levelCount', { count: workflow.nodes.length })}
              </Tag>
              {workflow.description && (
                <Text type="secondary" className="text-xs">
                  {workflow.description}
                </Text>
              )}
            </Space>
          }
        >
          <Steps
            direction="horizontal"
            size="small"
            current={levelStates.findIndex((l) => l.status === 'pending')}
            items={levelStates.map((l) => {
              const node = workflow.nodes.find((n) => n.level === l.level);
              return {
                title: node?.name || t('detailTabs.levelLabel', { level: l.level }),
                status: stepStatus(l.status),
                icon: iconFor(l.status),
                description: node ? (
                  <Space orientation="vertical" size={0} className="text-xs">
                    <Tag color={modeColors[node.approvalMode]}>
                      {modeLabels[node.approvalMode]}
                    </Tag>
                    <span className="text-gray-500">
                      {node.approverType === 'user' ? (
                        <UserIcon size={10} className="inline mr-0.5" />
                      ) : (
                        <Users size={10} className="inline mr-0.5" />
                      )}
                      {approverTypeLabels[node.approverType]}
                      {node.approverNames && node.approverNames.length > 0
                        ? `:${node.approverNames.slice(0, 2).join(',')}${node.approverNames.length > 2 ? '...' : ''}`
                        : ''}
                    </span>
                    {node.timeoutHours && (
                      <span className="text-gray-400">
                        {t('detailTabs.timeoutHours', { hours: node.timeoutHours })}
                      </span>
                    )}
                  </Space>
                ) : (
                  undefined
                ),
              };
            })}
          />
          {currentLevel !== undefined && (
            <div className="mt-3">
              <Alert
                type="info"
                showIcon
                message={
                  <Text>
                    {t('detailTabs.currentLevelIntro')}{' '}
                    <Text strong>{t('detailTabs.levelLabel', { level: currentLevel })}</Text>
                    {myApprovalId && (
                      <Tag color="orange" className="ml-2">
                        {t('detailTabs.youNeedToApprove')}
                      </Tag>
                    )}
                  </Text>
                }
              />
            </div>
          )}
        </Card>
      )}

      {!hasWorkflowMeta && hasRecords && (
        <Alert
          type="warning"
          showIcon
          message={t('detailTabs.workflowNotMatched')}
          className="text-xs"
        />
      )}

      {!hasRecords && !hasWorkflowMeta && (
        <Empty description={t('detailTabs.noApprovalWorkflow')} />
      )}

      {hasRecords && (
        <>
          <Divider style={{ margin: '8px 0' }} />
          <ApprovalTimeline
            approvals={timelineSteps}
            currentLevel={currentLevel}
            workflowName={workflow?.name}
            canApprove={!!myApprovalId && !isTicketFinal}
            showApprovalActions
            onApprove={(p) => submitApproval('approve', p)}
            onReject={(p) => submitApproval('reject', p)}
            onDelegate={(p) => submitApproval('delegate', p)}
            formatDateTime={formatDateTime}
          />
        </>
      )}
    </div>
  );
};

export default ApprovalWorkflowPanel;