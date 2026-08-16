'use client';

import React, { useState, useEffect } from 'react';
import { Card, Descriptions, Tag, Button, Skeleton, Result, Space, App } from 'antd';
import { ArrowLeft } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import { TicketApi } from '@/lib/api/ticket-api';
import { useI18n } from '@/lib/i18n/useI18n';

const ImprovementDetailPage = () => {
  const { t } = useI18n();
  const params = useParams() as { id?: string };
  const id = params?.id;
  const router = useRouter();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [improvement, setImprovement] = useState<{
    id: string | number;
    title: string;
    description?: string;
    status?: string;
    priority?: string;
    assignee?: { name?: string };
    createdAt?: string;
    updatedAt?: string;
  } | null>(null);

  useEffect(() => {
    if (id) loadDetail();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  const loadDetail = async () => {
    if (!id) return;
    // 接受 ticketNumber（IMP-xxx 或数字）或纯数字 id
    const numericId = parseInt(id, 10);
    if (!isNaN(numericId) && numericId > 0 && /^\d+$/.test(id)) {
      setLoading(true);
      try {
        const data = await TicketApi.getTicket(numericId);
        setImprovement(data as unknown as typeof improvement);
      } catch (err) {
        console.error('Load improvement failed:', err);
        message.error(t('improvements.loadFailed'));
        setError(t('improvements.notFound'));
      } finally {
        setLoading(false);
      }
      return;
    }
    setError(t('improvements.invalidId'));
    setLoading(false);
  };

  if (loading) {
    return (
      <div className="p-6">
        <Card>
          <Skeleton active />
        </Card>
      </div>
    );
  }

  if (!improvement) {
    return (
      <div className="p-6">
        <Card>
          <Result
            status="404"
            title="404"
            subTitle={error || t('improvements.notFoundHint')}
            extra={
              <Button type="primary" onClick={() => router.push('/improvements')}>
                {t('common.back')}
              </Button>
            }
          />
        </Card>
      </div>
    );
  }

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
        <Button
          icon={<ArrowLeft />}
          onClick={() => router.push('/improvements')}
          type="text"
        >
          {t('common.back')}
        </Button>

        <Card>
          <Space orientation="vertical" size="small" style={{ width: '100%' }}>
            <h2 className="text-2xl font-bold text-gray-800">{improvement.title}</h2>
            <Space>
              <Tag color="blue">{improvement.status || t('improvements.defaultStatus')}</Tag>
              <Tag>{improvement.priority || t('improvements.defaultPriority')}</Tag>
            </Space>
          </Space>

          <Descriptions bordered column={2} style={{ marginTop: 24 }}>
            <Descriptions.Item label={t('improvements.fieldPlanId')}>
              {String(improvement.id)}
            </Descriptions.Item>
            <Descriptions.Item label={t('improvements.fieldTitle')}>
              {improvement.title}
            </Descriptions.Item>
            <Descriptions.Item label={t('improvements.fieldStatus')}>
              <Tag color="blue">{improvement.status || t('improvements.defaultStatus')}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('improvements.fieldPriority')}>
              {improvement.priority || t('improvements.defaultPriority')}
            </Descriptions.Item>
            <Descriptions.Item label={t('improvements.fieldOwner')}>
              {improvement.assignee?.name || t('improvements.unassigned')}
            </Descriptions.Item>
            <Descriptions.Item label={t('improvements.fieldCreatedAt')}>
              {improvement.createdAt || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('improvements.fieldUpdatedAt')} span={2}>
              {improvement.updatedAt || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('improvements.fieldDescription')} span={2}>
              {improvement.description || '-'}
            </Descriptions.Item>
          </Descriptions>
        </Card>
      </Space>
    </div>
  );
};

export default ImprovementDetailPage;