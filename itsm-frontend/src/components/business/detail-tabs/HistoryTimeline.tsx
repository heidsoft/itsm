'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { Timeline, Typography, Empty, Spin, Alert, Button, Tag } from 'antd';
import { History as HistoryIcon } from 'lucide-react';
import { useI18n } from '@/lib/i18n/useI18n';
import { formatDateTime as sharedFormatDateTime } from '@/lib/formatters';
import type { HistoryRecord, TargetType } from './types';

const { Text } = Typography;

// 后端历史接口的 action 是稳定英文枚举（create/update/status_change…），
// 直接渲染会和中文 details 混排（如「update + 更新工单」）。
// 这里统一做展示层归一：已知 action 转本地化文案，未知值原样保留。
const ACTION_LABELS: Record<string, { zh: string; en: string }> = {
  create: { zh: 'Create', en: 'Create' },
  created: { zh: 'Create', en: 'Create' },
  update: { zh: 'Update', en: 'Update' },
  updated: { zh: 'Update', en: 'Update' },
  delete: { zh: 'Delete', en: 'Delete' },
  deleted: { zh: 'Delete', en: 'Delete' },
  status_change: { zh: 'Status change', en: 'Status Change' },
  status_changed: { zh: 'Status change', en: 'Status Change' },
  change_status: { zh: 'Status change', en: 'Status Change' },
  assign: { zh: 'Assign', en: 'Assign' },
  assigned: { zh: 'Assign', en: 'Assign' },
  unassign: { zh: 'Unassign', en: 'Unassign' },
  comment: { zh: 'Comments', en: 'Comment' },
  comment_added: { zh: 'Comments', en: 'Comment' },
  escalate: { zh: 'Escalate', en: 'Escalate' },
  escalated: { zh: 'Escalate', en: 'Escalate' },
  resolve: { zh: 'Resolve', en: 'Resolve' },
  resolved: { zh: 'Resolve', en: 'Resolve' },
  close: { zh: 'Close', en: 'Close' },
  closed: { zh: 'Close', en: 'Close' },
  reopen: { zh: 'Reopen', en: 'Reopen' },
  reopened: { zh: 'Reopen', en: 'Reopen' },
  approve: { zh: 'ApprovalApprove', en: 'Approve' },
  approved: { zh: 'ApprovalApprove', en: 'Approve' },
  reject: { zh: 'Rejected', en: 'Reject' },
  rejected: { zh: 'Rejected', en: 'Reject' },
  login_success: { zh: 'Login Successful', en: 'Login Success' },
  login_failed: { zh: 'Login Failed', en: 'Login Failed' },
  search: { zh: 'Query', en: 'Search' },
  export: { zh: 'Export', en: 'Export' },
  import: { zh: 'Import', en: 'Import' },
};

function formatActionLabel(action: string | undefined, language: string): string | null {
  if (!action) return null;
  const entry = ACTION_LABELS[action.trim().toLowerCase()];
  if (!entry) return action;
  return language === 'en-US' ? entry.en : entry.zh;
}

export interface HistoryTimelineProps {
  targetType: TargetType;
  targetId: number | string;
  fetchHistory?: (targetId: number | string) => Promise<HistoryRecord[]>;
  fetchAuditLog?: (targetType: TargetType, targetId: number | string) => Promise<HistoryRecord[]>;
  formatDateTime?: (dateString: string) => string;
}

export const HistoryTimeline: React.FC<HistoryTimelineProps> = ({
  targetType,
  targetId,
  fetchHistory,
  fetchAuditLog,
  formatDateTime,
}) => {
  const { t, language } = useI18n();
  const [records, setRecords] = useState<HistoryRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [source, setSource] = useState<'native' | 'audit'>('native');

  const defaultFormat = useCallback((s: string) => sharedFormatDateTime(s), []);
  const fmt = formatDateTime ?? defaultFormat;

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      if (fetchHistory) {
        try {
          const data = await fetchHistory(targetId);
          setRecords(data || []);
          setSource('native');
          return;
        } catch (e) {
          if (!fetchAuditLog) throw e;
        }
      }
      if (fetchAuditLog) {
        const data = await fetchAuditLog(targetType, targetId);
        setRecords(data || []);
        setSource('audit');
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : t('detailTabs.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [targetType, targetId, fetchHistory, fetchAuditLog, t]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) {
    return (
      <div className="p-6 text-center">
        <Spin />
      </div>
    );
  }

  return (
    <div className="p-6">
      {error && (
        <Alert
          title={error}
          type="error"
          showIcon
          closable={{ onClose: () => setError(null) }}
          className="mb-4"
          action={
            <Button size="small" type="link" onClick={() => void load()}>
              {t('common.retry')}
            </Button>
          }
        />
      )}

      {records.length > 0 ? (
        <Timeline>
          {records.map((r) => {
            const actionLabel = formatActionLabel(r.action, language);
            return (
              <Timeline.Item key={r.id}>
                <div className="flex items-start justify-between">
                  <div>
                    <Text strong>{r.user?.name || r.user?.username || t('detailTabs.system')}</Text>
                    {actionLabel && (
                      <Tag color="blue" className="ml-2">
                        {actionLabel}
                      </Tag>
                    )}
                    {r.details && (
                      <div className="text-sm text-gray-600 mt-1">{r.details}</div>
                    )}
                    {r.fieldName && (
                      <div className="text-sm text-gray-600 mt-1">
                        {t('detailTabs.fieldChanged')} <Text code>{r.fieldName}</Text>
                      </div>
                    )}
                    {r.changeReason && (
                      <div className="text-sm text-gray-500 mt-1">{t('detailTabs.changeReason')}：{r.changeReason}</div>
                    )}
                  </div>
                  <div className="text-right ml-4 shrink-0">
                    <div className="text-sm text-gray-500">{fmt(r.createdAt)}</div>
                    {r.oldValue !== undefined && r.newValue !== undefined && (
                      <div className="text-xs text-gray-400">
                        {String(r.oldValue) || t('detailTabs.emptyValue')} → {String(r.newValue) || t('detailTabs.emptyValue')}
                      </div>
                    )}
                  </div>
                </div>
              </Timeline.Item>
            );
          })}
        </Timeline>
      ) : (
        <Empty
          image={<HistoryIcon size={48} className="text-gray-300 mx-auto" />}
          description={t('detailTabs.noHistory')}
        />
      )}
    </div>
  );
};

export default HistoryTimeline;
