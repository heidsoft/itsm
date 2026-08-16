'use client';

import React from 'react';
import { App, Button, Card, Tabs } from 'antd';
import { ArrowLeft, MessageSquare, Clock as HistoryIcon, Workflow } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';
import TicketDetail from '@/components/ticket/TicketDetail';
import {
  CommentPanel,
  HistoryTimeline,
  ApprovalWorkflowPanel,
  ticketCommentAdapter,
  fetchAuditLogHistory,
} from '@/components/business/detail-tabs';
import { useAuthStore } from '@/lib/store/auth-store';
import { useI18n } from '@/lib/i18n/useI18n';
import dayjs from 'dayjs';

const formatDateTime = (v?: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-');

export default function TicketDetailPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.ticketId as string;
  const numericId = Number(id);
  const { user } = useAuthStore();
  const { t } = useI18n();

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
        <TicketDetail id={id} />

        {Number.isFinite(numericId) && numericId > 0 && (
          <Card className="mt-4 rounded-lg shadow-sm border border-gray-200">
            <Tabs
              defaultActiveKey="comments"
              items={[
                {
                  key: 'comments',
                  label: (
                    <span>
                      <MessageSquare size={14} className="inline mr-1" />
                      {t('detailTabs.comments')}
                    </span>
                  ),
                  children: (
                    <CommentPanel
                      targetType="ticket"
                      targetId={numericId}
                      adapter={ticketCommentAdapter}
                      currentUserId={user?.id}
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
                      targetType="ticket"
                      targetId={numericId}
                      fetchAuditLog={fetchAuditLogHistory}
                      formatDateTime={formatDateTime}
                    />
                  ),
                },
                {
                  key: 'approvals',
                  label: (
                    <span>
                      <Workflow size={14} className="inline mr-1" />
                      {t('detailTabs.approvals')}
                    </span>
                  ),
                  children: (
                    <ApprovalWorkflowPanel
                      ticketId={numericId}
                      currentUserId={user?.id}
                      isTicketFinal={false}
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
