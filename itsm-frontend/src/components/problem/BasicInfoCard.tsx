'use client';

import React from 'react';
import { Card, Descriptions, Divider, Typography } from 'antd';
import { ProblemPriorityLabels } from '@/constants/problem';
import dayjs from 'dayjs';

const { Title, Paragraph } = Typography;

interface BasicInfoCardProps {
   
  data: any;
}

/**
 * 基本信息卡片组件
 * 使用统一的 camelCase API 字段
 */
const BasicInfoCard: React.FC<BasicInfoCardProps> = ({ data }) => {
  if (!data) {
    return (
      <Card styles={{ body: { padding: '16px 24px' } }}>
        <div style={{ textAlign: 'center', color: '#999' }}>暂无数据</div>
      </Card>
    );
  }

  const reporterId = data.reporterId ?? data.createdBy ?? '-';
  const assigneeId = data.assigneeId ?? '-';
  const createdAt = data.createdAt ?? '';
  const updatedAt = data.updatedAt ?? '';
  const rootCause = data.rootCause ?? '暂无分析';
  const impact = data.impact ?? '暂无描述';
  const priority = (data.priority ?? data.severity ?? '') as string;
  const category = data.category ?? '-';
  const description = data.description ?? '-';
  const status = (data.status ?? '') as string;

  // 格式化时间
  const formatDate = (dateStr: string | number | undefined): string => {
    if (!dateStr) return '-';
    try {
      return dayjs(dateStr as string).format('YYYY-MM-DD HH:mm:ss');
    } catch {
      return String(dateStr);
    }
  };

  // 获取优先级标签
  const getPriorityLabel = (p: string): string => {
    if (!p) return '-';
    const labels: Record<string, string> = {
      critical: '紧急',
      high: '高',
      medium: '中',
      low: '低',
    };
    return labels[p.toLowerCase()] || p;
  };

  // 获取状态标签
  const getStatusLabel = (s: string): string => {
    if (!s) return '-';
    const labels: Record<string, string> = {
      open: '待处理',
      investigating: '调查中',
      identified: '已识别',
      resolved: '已解决',
      closed: '已关闭',
      inProgress: '处理中',
    };
    return labels[s] || s;
  };

  return (
    <Card styles={{ body: { padding: '16px 24px' } }}>
      <Descriptions column={2}>
        <Descriptions.Item label="问题ID">{data.id ?? '-'}</Descriptions.Item>
        <Descriptions.Item label="状态">
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
        <Descriptions.Item label="创建人ID">{reporterId}</Descriptions.Item>
        <Descriptions.Item label="负责人ID">{assigneeId}</Descriptions.Item>
        <Descriptions.Item label="优先级">
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
        <Descriptions.Item label="分类">{category}</Descriptions.Item>
        <Descriptions.Item label="创建时间">{formatDate(createdAt)}</Descriptions.Item>
        <Descriptions.Item label="更新时间">{formatDate(updatedAt)}</Descriptions.Item>
      </Descriptions>

      <Divider />

      <Title level={5}>描述</Title>
      <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{description}</Paragraph>

      <Divider />

      <Title level={5}>根本原因分析</Title>
      <Paragraph
        style={{ whiteSpace: 'pre-wrap', color: rootCause === '暂无分析' ? '#999' : '#333' }}
      >
        {rootCause}
      </Paragraph>

      <Divider />

      <Title level={5}>影响范围</Title>
      <Paragraph style={{ whiteSpace: 'pre-wrap', color: impact === '暂无描述' ? '#999' : '#333' }}>
        {impact}
      </Paragraph>
    </Card>
  );
};

export default BasicInfoCard;
