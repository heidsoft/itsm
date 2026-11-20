'use client';

import React, { useCallback, useEffect, useRef, useState } from 'react';
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
import { useI18n } from '@/lib/i18n/useI18n';

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
  const { t } = useI18n();
  const params = useParams();
  const id = params?.id as string;
  const numericId = Number(id);

  const [approvals, setApprovals] = useState<ApprovalStep[]>([]);
  const [approvalLoading, setApprovalLoading] = useState(false);
  // 深链 /changes/1 → /changes/2 复用组件实例：旧 id 的审批请求若慢返回，
  // 会覆盖新 id 的审批链（且 approvals 不重置）。用 generation 守卫丢弃过期响应。
  const approvalSeqRef = useRef(0);

  const loadApprovals = useCallback(async () => {
    if (!Number.isFinite(numericId) || numericId <= 0) return;
    const seq = ++approvalSeqRef.current;
    setApprovalLoading(true);
    try {
      const records = await ChangeApi.getChangeApprovals(numericId);
      if (seq !== approvalSeqRef.current) return;
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
      if (seq !== approvalSeqRef.current) return;
      // 静默：Empty 态
      console.warn('load change approvals failed', e);
      setApprovals([]);
    } finally {
      if (seq === approvalSeqRef.current) setApprovalLoading(false);
    }
  }, [numericId]);

  // id 变化时先清空旧审批链，避免新 id 数据到达前展示上一个变更的审批记录。
  useEffect(() => {
    approvalSeqRef.current += 1;
    setApprovals([]);
    setApprovalLoading(false);
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
                      {t('detailTabs.approvals')}
                    </span>
                  ),
                  children: approvalLoading ? (
                    <div className="p-6 text-center">{t('common.loading')}</div>
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
                      {t('detailTabs.history')}
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