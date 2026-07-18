'use client';

/**
 * 工单多级审批链增强面板
 *
 * 相比 ApprovalTimeline 的增强：
 * 1. 顶部展示"工作流全景 Steps"——把匹配到的 workflow 的 nodes[] 展开为 Steps
 *    每级标注：审批模式(sequential/parallel/any/all)、审批人类型、超时时长
 * 2. 中部展示"审批记录 Timeline"——现有 records，按 level 分组显示
 * 3. 底部展示"操作区"——当前用户是本级审批人时，展示 通过/拒绝/委派 三按钮
 * 4. 内部集成 TicketApprovalApi 的 getWorkflows / getApprovalRecords / submitApproval
 *
 * 若后端没有为该工单绑定 workflow（getWorkflows 未匹配），会自动降级为
 * 仅显示 Timeline + 操作区，与 Iter-1 行为一致。
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

const { Text } = Typography;

const modeLabels: Record<ApprovalNode['approvalMode'], string> = {
  sequential: '串行审批',
  parallel: '并行审批',
  any: '任一通过',
  all: '全员通过',
};

const modeColors: Record<ApprovalNode['approvalMode'], string> = {
  sequential: 'blue',
  parallel: 'purple',
  any: 'green',
  all: 'orange',
};

const approverTypeLabels: Record<ApprovalNode['approverType'], string> = {
  user: '指定用户',
  role: '角色',
  department: '部门',
  dynamic: '动态',
};

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
 * 计算某一级的状态
 * - 有 approved 且没有 pending → approved
 * - 有 rejected → rejected
 * - 有 pending → pending（当前级）
 * - 有 delegated 但没有其他 → delegated
 * - 全部 timeout → timeout
 * - 无记录 → pending（未开始）
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
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [workflow, setWorkflow] = useState<ApprovalWorkflow | null>(null);
  const [records, setRecords] = useState<ApprovalRecord[]>([]);

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      // 并行加载工作流列表 + 审批记录
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

      // 优先用 records 里的 workflowId 匹配（更精确，因为工单实际用了哪个）
      const wfIdFromRecord = (recRes.items || [])[0]?.workflowId;
      let matched: ApprovalWorkflow | null = null;
      if (wfIdFromRecord) {
        matched = (wfRes.items || []).find((w) => w.id === wfIdFromRecord) || null;
      }
      // 如未匹配（record 为空 or workflow 已删除），再按 ticketType/priority 兜底
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
      message.error('审批链加载失败');
    } finally {
      setLoading(false);
    }
  }, [ticketId, ticketType, priority, message]);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  // ==================== 计算全景 Steps ====================
  const levelStates: LevelState[] = useMemo(() => {
    if (!workflow || !workflow.nodes || workflow.nodes.length === 0) {
      // 无 workflow 定义：按 records 的 currentLevel 分组
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
    // 有 workflow：按 nodes 顺序，每级填充对应 records
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

  // 当前 pending 级次
  const currentLevel = useMemo(() => {
    const pending = levelStates.find((l) => l.status === 'pending' && l.records.length > 0);
    return pending?.level;
  }, [levelStates]);

  // 当前用户在本级的 pending 审批记录 id
  const myApprovalId = useMemo(() => {
    if (!currentUserId || isTicketFinal) return null;
    const myRec = records.find(
      (r) => r.status === 'pending' && r.approverId === currentUserId,
    );
    return myRec?.id ?? null;
  }, [records, currentUserId, isTicketFinal]);

  // ==================== Timeline 用的扁平 steps ====================
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

  // ==================== 提交审批 ====================
  const submitApproval = async (
    action: 'approve' | 'reject' | 'delegate',
    payload: { comment: string; delegateToUserId?: number },
  ) => {
    if (!myApprovalId) {
      message.warning('没有可操作的审批节点');
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

  // ==================== 渲染 ====================
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
      {/* 工作流全景 */}
      {hasWorkflowMeta && workflow && (
        <Card
          size="small"
          title={
            <Space size={6}>
              <GitBranch size={14} />
              <Text strong>{workflow.name}</Text>
              <Tag color="blue">{workflow.nodes.length} 级审批</Tag>
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
                title: node?.name || `第 ${l.level} 级`,
                status: stepStatus(l.status),
                icon: iconFor(l.status),
                description: node ? (
                  <Space direction="vertical" size={0} className="text-xs">
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
                        ? `：${node.approverNames.slice(0, 2).join(',')}${node.approverNames.length > 2 ? '...' : ''}`
                        : ''}
                    </span>
                    {node.timeoutHours && (
                      <span className="text-gray-400">超时 {node.timeoutHours}h</span>
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
                    当前进行到 <Text strong>第 {currentLevel} 级</Text>
                    {myApprovalId && (
                      <Tag color="orange" className="ml-2">
                        本级需要您审批
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
          message="未找到匹配的审批工作流定义，仅显示审批记录"
          className="text-xs"
        />
      )}

      {!hasRecords && !hasWorkflowMeta && (
        <Empty description="该工单未走审批流程" />
      )}

      {/* 审批记录 Timeline + 操作区 */}
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
