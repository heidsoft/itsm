'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { Timeline, Typography, Empty, Spin, Alert, Button, Tag } from 'antd';
import { History as HistoryIcon } from 'lucide-react';
import { useI18n } from '@/lib/i18n/useI18n';
import type { HistoryRecord, TargetType } from './types';

const { Text } = Typography;

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

  const defaultFormat = useCallback(
    (s: string) => (s ? new Date(s).toLocaleString(language === 'en-US' ? 'en-US' : 'zh-CN') : ''),
    [language]
  );
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
          message={error}
          type="error"
          showIcon
          closable
          className="mb-4"
          onClose={() => setError(null)}
          action={
            <Button size="small" type="link" onClick={() => void load()}>
              {t('common.retry')}
            </Button>
          }
        />
      )}

      {records.length > 0 ? (
        <Timeline>
          {records.map((r) => (
            <Timeline.Item key={r.id}>
              <div className="flex items-start justify-between">
                <div>
                  <Text strong>{r.user?.name || r.user?.username || t('detailTabs.system')}</Text>
                  {r.action && (
                    <Tag color="blue" className="ml-2">
                      {r.action}
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
                  {source === 'audit' && (r.method || r.path) && (
                    <div className="text-xs text-gray-400 mt-1">
                      {r.method} {r.path}{' '}
                      {r.statusCode !== undefined && (
                        <Tag color={r.statusCode >= 400 ? 'red' : 'default'}>{r.statusCode}</Tag>
                      )}
                    </div>
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
          ))}
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
