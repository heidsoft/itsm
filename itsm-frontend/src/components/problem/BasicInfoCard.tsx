'use client';

import React from 'react';
import { Card, Descriptions, Divider, Typography } from 'antd';
import dayjs from 'dayjs';
import { useI18n } from '@/lib/i18n/useI18n';

const { Title, Paragraph } = Typography;

interface BasicInfoCardProps {
  data: {
    id: number;
    description: string;
    status: string;
    priority: string;
    severity?: string;
    category?: string;
    rootCause?: string;
    impact?: string;
    reporterId?: number;
    createdBy?: number;
    assigneeId?: number;
    createdAt: string;
    updatedAt: string;
  };
}

/**
 * 基本信息卡片组件
 * 使用统一的 camelCase API 字段
 */
const BasicInfoCard: React.FC<BasicInfoCardProps> = ({ data }) => {
  const { t } = useI18n();

  if (!data) {
    return (
      <Card styles={{ body: { padding: '16px 24px' } }}>
        <div style={{ textAlign: 'center', color: '#999' }}>{t('problem.noData')}</div>
      </Card>
    );
  }

  const reporterId = data.reporterId ?? data.createdBy ?? '-';
  const assigneeId = data.assigneeId ?? '-';
  const createdAt = data.createdAt ?? '';
  const updatedAt = data.updatedAt ?? '';
  const noAnalysisText = t('problem.noAnalysis');
  const noDescriptionText = t('problem.noDescription');
  const rootCause = data.rootCause ?? noAnalysisText;
  const impact = data.impact ?? noDescriptionText;
  const priority = (data.priority ?? data.severity ?? '') as string;
  const category = data.category ?? '-';
  const description = data.description ?? '-';
  const status = (data.status ?? '') as string;

  const formatDate = (dateStr: string | number | undefined): string => {
    if (!dateStr) return '-';
    try {
      return dayjs(dateStr as string).format('YYYY-MM-DD HH:mm:ss');
    } catch {
      return String(dateStr);
    }
  };

  const getPriorityLabel = (p: string): string => {
    if (!p) return '-';
    const key = p.toLowerCase();
    const knownKeys = ['critical', 'high', 'medium', 'low'];
    if (knownKeys.includes(key)) {
      return t(`problem.priorityLabels.${key}`);
    }
    return p;
  };

  const getStatusLabel = (s: string): string => {
    if (!s) return '-';
    const knownKeys = ['open', 'investigating', 'identified', 'resolved', 'closed', 'inProgress'];
    if (knownKeys.includes(s)) {
      return t(`problem.statusLabels.${s}`);
    }
    return s;
  };

  return (
    <Card styles={{ body: { padding: '16px 24px' } }}>
      <Descriptions column={2}>
        <Descriptions.Item label={t('problem.problemId')}>{data.id ?? '-'}</Descriptions.Item>
        <Descriptions.Item label={t('problem.status')}>
          <span
            style={{
              padding: '2px 8px',
              borderRadius: '4px',
              backgroundColor:
                status === 'resolved' ? '#f6ffed' : status === 'open' ? '#fff7e6' : '#e6f7ff',
              color: status === 'resolved' ? '#52c41a' : status === 'open' ? '#fa8c16' : '#1890ff',
            }}
          >
            {getStatusLabel(status)}
          </span>
        </Descriptions.Item>
        <Descriptions.Item label={t('problem.reporterId')}>{reporterId}</Descriptions.Item>
        <Descriptions.Item label={t('problem.assigneeId')}>{assigneeId}</Descriptions.Item>
        <Descriptions.Item label={t('problem.priority')}>
          <span
            style={{
              padding: '2px 8px',
              borderRadius: '4px',
              backgroundColor:
                priority === 'critical' ? '#fff2f0' : priority === 'high' ? '#fff7e6' : '#e6f7ff',
              color:
                priority === 'critical' ? '#ff4d4f' : priority === 'high' ? '#fa8c16' : '#1890ff',
            }}
          >
            {getPriorityLabel(priority)}
          </span>
        </Descriptions.Item>
        <Descriptions.Item label={t('problem.category')}>{category}</Descriptions.Item>
        <Descriptions.Item label={t('problem.createdAt')}>{formatDate(createdAt)}</Descriptions.Item>
        <Descriptions.Item label={t('problem.updatedAt')}>{formatDate(updatedAt)}</Descriptions.Item>
      </Descriptions>

      <Divider />

      <Title level={5}>{t('problem.description')}</Title>
      <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{description}</Paragraph>

      <Divider />

      <Title level={5}>{t('problem.rootCause')}</Title>
      <Paragraph
        style={{ whiteSpace: 'pre-wrap', color: rootCause === noAnalysisText ? '#999' : '#333' }}
      >
        {rootCause}
      </Paragraph>

      <Divider />

      <Title level={5}>{t('problem.impact')}</Title>
      <Paragraph style={{ whiteSpace: 'pre-wrap', color: impact === noDescriptionText ? '#999' : '#333' }}>
        {impact}
      </Paragraph>
    </Card>
  );
};

export default BasicInfoCard;