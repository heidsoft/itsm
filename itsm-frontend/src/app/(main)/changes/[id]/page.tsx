'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { App, Card, Tabs } from 'antd';
import { GitBranch, Clock as HistoryIcon } from 'lucide-react';
import { useParams } from 'next/navigation';
import ChangeDetail from '@/components/change/ChangeDetail';
import { ChangeApi, type ChangeApproval } from '@/lib/api/change-api';
import {
  ApprovalTimeline,
  HistoryTimeline,
  fetchAuditLogHistory,
  type ApprovalStep,
  type ApprovalStepStatus,
} from '@/components/business/detail-tabs';
import dayjs from 'dayjs';

const formatDateTime = (v?: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-');

/**
 * 将 change 的 status 映射到 ApprovalStepStatus
 * change 的审批 record.status 会用 change 的 workflow 状态，例如 approved/rejected/pending
 */
function mapApprovalStatus(status: string): ApprovalStepStatus {
  switch (status) {
    case 'approved':
      return 'approved';
    case 'rejected':
      return 'rejected';
    case 'pending':
      return 'pending';
    default:
      return 'pending';
  }
}

export default function ChangeDetailPage() {
  const params = useParams();
  const id = params?.id as string;
  const numericId = Number(id);

  const [approvals, setApprovals] = useState<ApprovalStep[]>([]);
  const [approvalLoading, setApprovalLoading] = useState(false);

  const loadApprovals = useCallback(async () => {
    if (!Number.isFinite(numericId) || numericId <= 0) return;
    setApprovalLoading(true);
    try {
      const records = await ChangeApi.getChangeApprovals(numericId);
      const steps: ApprovalStep[] = (records || []).map((r: ChangeApproval, idx: number) => ({
        id: r.id,
        level: idx + 1,
        status: mapApprovalStatus(r.status),
        approverId: r.approverId,
        approverName: r.approverName,
        comment: r.comment,
        processedAt: r.approvedAt,
        createdAt: r.createdAt,
      }));
      setApprovals(steps);
    } catch (e) {
      // 静默：Empty 态
      console.warn('load change approvals failed', e);
      setApprovals([]);
    } finally {
      setApprovalLoading(false);
    }
  }, [numericId]);

  useEffect(() => {
    void loadApprovals();
  }, [loadApprovals]);

  return (
    <App>
      <ChangeDetail />

      {/* 追加：审批链 + 历史 Tabs */}
      {Number.isFinite(numericId) && numericId > 0 && (
        <div style={{ padding: '0 24px 24px' }}>
          <Card className="mt-4 rounded-lg shadow-sm border border-gray-200">
            <Tabs
              defaultActiveKey="approvals"
              items={[
                {
                  key: 'approvals',
                  label: (
                    <span>
                      <GitBranch size={14} className="inline mr-1" />
                      审批时间线
                    </span>
                  ),
                  children: approvalLoading ? (
                    <div className="p-6 text-center">加载中...</div>
                  ) : (
                    <ApprovalTimeline
                      approvals={approvals}
                      canApprove={false}
                      showApprovalActions={false}
                      formatDateTime={formatDateTime}
                    />
                  ),
                },
                {
                  key: 'history',
                  label: (
                    <span>
                      <HistoryIcon size={14} className="inline mr-1" />
                      历史（审计日志）
                    </span>
                  ),
                  children: (
                    <HistoryTimeline
                      targetType="change"
                      targetId={numericId}
                      fetchAuditLog={fetchAuditLogHistory}
                      formatDateTime={formatDateTime}
                    />
                  ),
                },
              ]}
            />
          </Card>
        </div>
      )}
    </App>
  );
}
