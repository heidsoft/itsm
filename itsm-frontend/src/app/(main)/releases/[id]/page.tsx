'use client';

import React from 'react';
import { App, Card } from 'antd';
import { useParams } from 'next/navigation';
import ReleaseDetail from '@/components/release/ReleaseDetail';
import { HistoryTimeline, fetchAuditLogHistory } from '@/components/business/detail-tabs';
import dayjs from 'dayjs';

const formatDateTime = (v?: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-');

export default function ReleaseDetailPage() {
  const params = useParams();
  const id = params?.id as string;
  const numericId = Number(id);

  return (
    <App>
      <ReleaseDetail />

      {/* 追加：历史 Card（评论/附件/审批 API 未就绪，先占位历史） */}
      {Number.isFinite(numericId) && numericId > 0 && (
        <div style={{ padding: '0 24px 24px' }}>
          <Card
            title="历史（审计日志）"
            className="mt-4 rounded-lg shadow-sm border border-gray-200"
          >
            <HistoryTimeline
              targetType="release"
              targetId={numericId}
              fetchAuditLog={fetchAuditLogHistory}
              formatDateTime={formatDateTime}
            />
          </Card>
        </div>
      )}
    </App>
  );
}
