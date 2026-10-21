'use client';

import React from 'react';
import { App, Button, Card, Tabs } from 'antd';
import { ArrowLeft, Link2, Clock as HistoryIcon } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';
import ProblemDetail from '@/components/problem/ProblemDetail';
import ProblemSLACard from '@/components/problem/ProblemSLACard';
import ProblemAssociationsTab from '@/components/problem/ProblemAssociationsTab';
import { HistoryTimeline, fetchAuditLogHistory, CommentPanel } from '@/components/business/detail-tabs';
import { problemCommentAdapter } from '@/components/business/detail-tabs/adapters/problem-comment-adapter';
import { MessageSquare } from 'lucide-react';
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

        {/* SLA 卡片：bound=false 如实显示「未绑定」，不伪装成正常 */}
        {Number.isFinite(numericId) && numericId > 0 && <ProblemSLACard problemId={numericId} />}

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
                  key: 'comments',
                  label: (
                    <span>
                      <MessageSquare size={14} className="inline mr-1" />
                      评论
                    </span>
                  ),
                  children: (
                    <CommentPanel
                      targetType="problem"
                      targetId={numericId}
                      adapter={problemCommentAdapter}
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