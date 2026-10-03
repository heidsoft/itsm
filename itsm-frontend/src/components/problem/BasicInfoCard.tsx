'use client';

import React from 'react';
import { Card, Descriptions, Divider, Typography } from 'antd';
import dayjs from 'dayjs';
import { useI18n } from '@/lib/i18n/useI18n';
import { problemPriorityLabel, problemStatusLabel } from '@/constants/problem';
import type { Problem } from '@/lib/api/problem-api';

const { Title, Paragraph } = Typography;

interface BasicInfoCardProps {
  data: Problem;
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

  // assigneeId/assigneeName/createdByName 是后端 omitempty 指针字段，
  // 只有 undefined 才回退到 ID；ID 本身总有值，不参与兜底。
  const reporterDisplay = data.createdByName ?? data.createdBy;
  const assigneeDisplay =
    data.assigneeId === undefined ? '-' : (data.assigneeName ?? data.assigneeId);
  // 后端 rootCause/impact 是非 omitempty 的 string，未填写时返回 ""，
  // 所以 ?? 永远不会触发，必须显式判空。
  const hasRootCause = data.rootCause.trim() !== '';
  const hasImpact = data.impact.trim() !== '';

  const formatDate = (dateStr: string): string => {
    if (!dateStr) return '-';
    try {
      return dayjs(dateStr).format('YYYY-MM-DD HH:mm:ss');
    } catch {
      return String(dateStr);
    }
  };

  // 取值标签由 @/constants/problem 单点负责。卡片此前查的是另一份 i18n 词典，
  // 同一个 critical 在列表显示「极高」、在详情卡片显示「紧急」；词典里的状态键
  // 还写着 inProgress，而后端存量值是 in_progress，于是原样漏出英文。
  const statusLabel = data.status === '' ? '-' : problemStatusLabel(data.status);
  const priorityLabel = data.priority === '' ? '-' : problemPriorityLabel(data.priority);

  const category = data.category === '' ? '-' : data.category;
  const description = data.description === '' ? '-' : data.description;

  return (
    <Card styles={{ body: { padding: '16px 24px' } }}>
      <Descriptions column={2}>
        <Descriptions.Item label={t('problem.problemId')}>{data.id}</Descriptions.Item>
        <Descriptions.Item label={t('problem.status')}>
          <span
            style={{
              padding: '2px 8px',
              borderRadius: '4px',
              backgroundColor:
                data.status === 'resolved'
                  ? '#f6ffed'
                  : data.status === 'open'
                    ? '#fff7e6'
                    : '#e6f7ff',
              color:
                data.status === 'resolved'
                  ? '#52c41a'
                  : data.status === 'open'
                    ? '#fa8c16'
                    : '#1890ff',
            }}
          >
            {statusLabel}
          </span>
        </Descriptions.Item>
        <Descriptions.Item label={t('problem.reporterId')}>{reporterDisplay}</Descriptions.Item>
        <Descriptions.Item label={t('problem.assigneeId')}>{assigneeDisplay}</Descriptions.Item>
        <Descriptions.Item label={t('problem.priority')}>
          <span
            style={{
              padding: '2px 8px',
              borderRadius: '4px',
              backgroundColor:
                data.priority === 'critical'
                  ? '#fff2f0'
                  : data.priority === 'high'
                    ? '#fff7e6'
                    : '#e6f7ff',
              color:
                data.priority === 'critical'
                  ? '#ff4d4f'
                  : data.priority === 'high'
                    ? '#fa8c16'
                    : '#1890ff',
            }}
          >
            {priorityLabel}
          </span>
        </Descriptions.Item>
        <Descriptions.Item label={t('problem.category')}>{category}</Descriptions.Item>
        <Descriptions.Item label={t('problem.createdAt')}>
          {formatDate(data.createdAt)}
        </Descriptions.Item>
        <Descriptions.Item label={t('problem.updatedAt')}>
          {formatDate(data.updatedAt)}
        </Descriptions.Item>
      </Descriptions>

      <Divider />

      <Title level={5}>{t('problem.description')}</Title>
      <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{description}</Paragraph>

      <Divider />

      <Title level={5}>{t('problem.rootCause')}</Title>
      <Paragraph style={{ whiteSpace: 'pre-wrap', color: hasRootCause ? '#333' : '#999' }}>
        {hasRootCause ? data.rootCause : t('problem.noAnalysis')}
      </Paragraph>

      <Divider />

      <Title level={5}>{t('problem.impact')}</Title>
      <Paragraph style={{ whiteSpace: 'pre-wrap', color: hasImpact ? '#333' : '#999' }}>
        {hasImpact ? data.impact : t('problem.noDescription')}
      </Paragraph>
    </Card>
  );
};

export default BasicInfoCard;