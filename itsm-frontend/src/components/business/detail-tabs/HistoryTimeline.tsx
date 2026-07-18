'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { Timeline, Typography, Empty, Spin, Alert, Button, Tag } from 'antd';
import { History as HistoryIcon } from 'lucide-react';
import type { HistoryRecord, TargetType } from './types';

const { Text } = Typography;

export interface HistoryTimelineProps {
  targetType: TargetType;
  targetId: number | string;
  /**
   * 模式 A：使用模块自有 /:id/history 端点
   */
  fetchHistory?: (targetId: number | string) => Promise<HistoryRecord[]>;
  /**
   * 模式 B：走 audit-logs 兜底
   */
  fetchAuditLog?: (targetType: TargetType, targetId: number | string) => Promise<HistoryRecord[]>;
  formatDateTime?: (dateString: string) => string;
}

const defaultFormat = (s: string) => (s ? new Date(s).toLocaleString('zh-CN') : '');

export const HistoryTimeline: React.FC<HistoryTimelineProps> = ({
  targetType,
  targetId,
  fetchHistory,
  fetchAuditLog,
  formatDateTime = defaultFormat,
}) => {
  const [records, setRecords] = useState<HistoryRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [source, setSource] = useState<'native' | 'audit'>('native');

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
          // 主源失败自动降级到 audit-log
          if (!fetchAuditLog) throw e;
        }
      }
      if (fetchAuditLog) {
        const data = await fetchAuditLog(targetType, targetId);
        setRecords(data || []);
        setSource('audit');
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载历史失败');
    } finally {
      setLoading(false);
    }
  }, [targetType, targetId, fetchHistory, fetchAuditLog]);

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
              重试
            </Button>
          }
        />
      )}

      {source === 'audit' && (
        <Alert
          type="info"
          showIcon
          className="mb-4"
          message="展示的是审计日志（通用来源）；模块专属历史 API 就绪后会自动切换。"
        />
      )}

      {records.length > 0 ? (
        <Timeline>
          {records.map((r) => (
            <Timeline.Item key={r.id}>
              <div className="flex items-start justify-between">
                <div>
                  <Text strong>{r.user?.name || r.user?.username || '系统'}</Text>
                  {r.action && (
                    <Tag color="blue" className="ml-2">
                      {r.action}
                    </Tag>
                  )}
                  {r.fieldName && (
                    <div className="text-sm text-gray-600 mt-1">
                      修改了 <Text code>{r.fieldName}</Text>
                    </div>
                  )}
                  {r.changeReason && (
                    <div className="text-sm text-gray-500 mt-1">原因: {r.changeReason}</div>
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
                  <div className="text-sm text-gray-500">{formatDateTime(r.createdAt)}</div>
                  {r.oldValue !== undefined && r.newValue !== undefined && (
                    <div className="text-xs text-gray-400">
                      {String(r.oldValue) || '(空)'} → {String(r.newValue) || '(空)'}
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
          description="暂无历史记录"
        />
      )}
    </div>
  );
};

export default HistoryTimeline;
