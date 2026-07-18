'use client';

import React from 'react';
import { App, Button, Card, Tabs } from 'antd';
import { ArrowLeft, MessageSquare, Clock as HistoryIcon } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';
import IncidentDetail from '@/components/incident/IncidentDetail';
import {
  CommentPanel,
  HistoryTimeline,
  incidentCommentAdapter,
  fetchAuditLogHistory,
} from '@/components/business/detail-tabs';
import { useAuthStore } from '@/lib/store/auth-store';
import dayjs from 'dayjs';

const formatDateTime = (v?: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-');

// 动态路由参数类型
export default function IncidentDetailPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;
  const numericId = Number(id);
  const { user } = useAuthStore();

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
            返回列表
          </Button>
        </div>
        {/* 主详情组件保持不变 */}
        <IncidentDetail id={id} />

        {/* 追加：协作与历史 Tabs */}
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
                      评论
                    </span>
                  ),
                  children: (
                    <CommentPanel
                      targetType="incident"
                      targetId={numericId}
                      adapter={incidentCommentAdapter}
                      currentUserId={user?.id}
                      formatDateTime={formatDateTime}
                      showInternalToggle={false}
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
                      targetType="incident"
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
