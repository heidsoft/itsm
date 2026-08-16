'use client';

import React from 'react';
import { App, Button, Card, Tabs } from 'antd';
import { ArrowLeft, Link2, Clock as HistoryIcon } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';
import ProblemDetail from '@/components/problem/ProblemDetail';
import ProblemAssociationsTab from '@/components/problem/ProblemAssociationsTab';
import { HistoryTimeline, fetchAuditLogHistory } from '@/components/business/detail-tabs';
import dayjs from 'dayjs';
import { useI18n } from '@/lib/i18n/useI18n';

const formatDateTime = (v?: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-');

export default function ProblemDetailPage() {
  const { t } = useI18n();
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;
  const numericId = Number(id);

  return (
    <App>
      <div style={{ padding: 24 }}>
        <div style={{ marginBottom: 16 }}>
          <Button
            type="link"
            icon={<ArrowLeft />}
            onClick={() => router.back()}
            style={{ paddingLeft: 0, color: '#666' }}
          >
            {t('common.back')}
          </Button>
        </div>
        <ProblemDetail id={id} />

        {/* 追加：关联 + 历史 Tabs */}
        {Number.isFinite(numericId) && numericId > 0 && (
          <Card className="mt-4 rounded-lg shadow-sm border border-gray-200">
            <Tabs
              defaultActiveKey="associations"
              items={[
                {
                  key: 'associations',
                  label: (
                    <span>
                      <Link2 size={14} className="inline mr-1" />
                      {t('detailTabs.relations')}
                    </span>
                  ),
                  children: <ProblemAssociationsTab problemId={numericId} />,
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
                      targetType="problem"
                      targetId={numericId}
                      fetchAuditLog={fetchAuditLogHistory}
                      formatDateTime={formatDateTime}
                    />
                  ),
                },
              ]}
            />
          </Card>
        )}
      </div>
    </App>
  );
}