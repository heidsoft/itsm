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
import {
  Card,
  Steps,
  Tag,
  Space,
  Typography,
  App,
  Empty,
  Divider,
  Skeleton,
  Alert,
  Button,
  Tooltip,
} from 'antd';
import {
  CheckCircle,
  Timer,
  Users,
  User as UserIcon,
  GitBranch,
  ArrowRight,
  CircleAlert,
  Workflow as WorkflowIcon,
  ExternalLink,
} from 'lucide-react';
import {
  TicketApprovalApi,
  type ApprovalWorkflow,
  type ApprovalNode,
  type ApprovalRecord,
} from '@/lib/api/ticket-approval-api';
import { TicketWorkflowStateApi } from '@/lib/api/ticket-workflow-state-api';
import type { BpmnProcessState, NextActivityInfo } from '@/types/ticket-workflow-state';
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
  const [bpmnState, setBpmnState] = useState<BpmnProcessState | null>(null);

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
      const [wfRes, recRes, bpmn] = await Promise.all([
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
        // BPMN 状态拉取：失败静默返回 null，调用方继续走原有 V1 逻辑。
        TicketWorkflowStateApi.tryGetStateV2(ticketId),
      ]);

      setRecords(recRes.items || []);
      setBpmnState(bpmn);

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

  // BPMN 节点类型 -> 中文 / 颜色，仅展示用，不参与逻辑。
  const bpmnTypeLabels = useMemo(
    () =>
      ({
        startEvent: t('detailTabs.bpmnTypeStartEvent') || '开始',
        endEvent: t('detailTabs.bpmnTypeEndEvent') || '结束',
        userTask: t('detailTabs.bpmnTypeUserTask') || '用户任务',
        serviceTask: t('detailTabs.bpmnTypeServiceTask') || '服务任务',
        exclusiveGateway: t('detailTabs.bpmnTypeExclusiveGateway') || '排他网关',
        parallelGateway: t('detailTabs.bpmnTypeParallelGateway') || '并行网关',
        inclusiveGateway: t('detailTabs.bpmnTypeInclusiveGateway') || '包容网关',
      }) as Record<string, string>,
    [t],
  );

  const bpmnTypeColors: Record<string, string> = useMemo(
    () => ({
      startEvent: 'green',
      endEvent: 'default',
      userTask: 'blue',
      serviceTask: 'purple',
      exclusiveGateway: 'orange',
      parallelGateway: 'cyan',
      inclusiveGateway: 'gold',
    }),
    [],
  );

  // "查看完整流程图" 跳转地址：与后端 ProcessTriggerService 约定的 businessKey 协议一致。
  const openProcessDiagramUrl = useCallback(
    (processInstanceId: string) => {
      const url = `/workflow/instances?businessKey=ticket:${ticketId}&instanceId=${encodeURIComponent(
        processInstanceId,
      )}`;
      if (typeof window !== 'undefined') {
        window.open(url, '_blank', 'noopener,noreferrer');
      }
    },
    [ticketId],
  );

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
      {/* === BPMN 状态叠加层（工单三件套改造）===
         - 优先级高于原 V1 Steps；不可用时（null / not_started）退化到下方原逻辑。
         - 不影响原有 approve/reject 提交路径（仍然走 submitApproval -> BPMN bridge）。 */}
      {bpmnState && bpmnState.bpmnStatus === 'running' && bpmnState.currentActivityId && (
        <Card
          size="small"
          data-testid="bpmn-running-card"
          title={
            <Space size={6}>
              <WorkflowIcon size={14} />
              <Text strong>{bpmnState.processDefinitionName || bpmnState.processDefinitionKey}</Text>
              <Tag color="processing">
                {t('detailTabs.bpmnStatusRunning') || '进行中'}
              </Tag>
            </Space>
          }
          extra={
            <Button
              type="link"
              size="small"
              icon={<ExternalLink size={12} />}
              onClick={() => openProcessDiagramUrl(bpmnState.processInstanceId)}
            >
              {t('detailTabs.viewProcessDiagram') || '查看完整流程图'}
            </Button>
          }
        >
          <Space orientation="vertical" size={8} className="w-full">
            <div>
              <Text type="secondary" className="text-xs">
                {t('detailTabs.bpmnCurrentNode') || '当前节点'}
              </Text>
              <div className="flex items-center gap-2 mt-1">
                <Tag
                  color={bpmnTypeColors[bpmnState.currentActivityType || ''] || 'blue'}
                  className="mr-0"
                >
                  {bpmnTypeLabels[bpmnState.currentActivityType || ''] ||
                    bpmnState.currentActivityType}
                </Tag>
                <Text strong>{bpmnState.currentActivityName}</Text>
              </div>
              {bpmnState.currentAssignees && bpmnState.currentAssignees.length > 0 && (
                <Space size={4} className="mt-2" wrap>
                  <Text type="secondary" className="text-xs">
                    {t('detailTabs.bpmnAssignees') || '处理人'}
                  </Text>
                  {bpmnState.currentAssignees.map((u) => (
                    <Tooltip key={u.id} title={u.fullName || u.username}>
                      <Tag color="geekblue" icon={<UserIcon size={10} />}>
                        {u.fullName || u.username}
                      </Tag>
                    </Tooltip>
                  ))}
                </Space>
              )}
            </div>
            {bpmnState.nextActivities && bpmnState.nextActivities.length > 0 && (
              <div>
                <Text type="secondary" className="text-xs">
                  {t('detailTabs.bpmnNextNode') || '下一步'}
                </Text>
                <Space orientation="vertical" size={4} className="mt-1 w-full">
                  {bpmnState.nextActivities.map((nx: NextActivityInfo) => (
                    <Space key={nx.activityId} size={6} className="text-xs">
                      <ArrowRight size={12} className="text-gray-400" />
                      <Tag
                        color={bpmnTypeColors[nx.activityType] || 'blue'}
                        className="mr-0"
                      >
                        {bpmnTypeLabels[nx.activityType] || nx.activityType}
                      </Tag>
                      <Text>{nx.activityName}</Text>
                      {nx.isGateway && (
                        <Tag color="orange" className="ml-1">
                          {t('detailTabs.bpmnIsGateway') || '网关分支'}
                        </Tag>
                      )}
                    </Space>
                  ))}
                </Space>
              </div>
            )}
          </Space>
        </Card>
      )}

      {bpmnState && bpmnState.bpmnStatus === 'completed' && (
        <Alert
          type="success"
          showIcon
          icon={<CheckCircle size={16} />}
          message={
            <Text>
              <Tag color="success" className="mr-2">
                {t('detailTabs.bpmnStatusCompleted') || '流程已完成'}
              </Tag>
              {bpmnState.processDefinitionName || bpmnState.processDefinitionKey}
            </Text>
          }
          action={
            <Button
              type="link"
              size="small"
              icon={<ExternalLink size={12} />}
              onClick={() => openProcessDiagramUrl(bpmnState.processInstanceId)}
            >
              {t('detailTabs.viewProcessDiagram') || '查看完整流程图'}
            </Button>
          }
        />
      )}

      {bpmnState && bpmnState.bpmnStatus === 'terminated' && (
        <Alert
          type="error"
          showIcon
          icon={<CircleAlert size={16} />}
          message={t('detailTabs.bpmnStatusTerminated') || '流程已终止'}
        />
      )}

      {bpmnState && bpmnState.bpmnStatus === 'not_started' && (
        <Alert
          type="info"
          showIcon
          message={
            t('detailTabs.bpmnNotStartedHint') ||
            '该工单未绑定 BPMN 流程，使用简化审批模式。'
          }
          className="text-xs"
        />
      )}

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