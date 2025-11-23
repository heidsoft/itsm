'use client';

import React from 'react';
import { Card, Col, Row, Statistic } from 'antd';
import { BookOpen } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

interface KnowledgeBaseStatsProps {
  stats: {
    total: number;
    accountManagement: number;
    troubleshooting: number;
    network: number;
  };
}

export const KnowledgeBaseStats: React.FC<KnowledgeBaseStatsProps> = ({ stats }) => {
  const { t } = useI18n();
  return (
    <div style={{ marginBottom: 24 }}>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('knowledgeBase.totalArticles')}
              value={stats.total}
              prefix={<BookOpen size={16} style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('knowledgeBase.accountManagement')}
              value={stats.accountManagement}
              valueStyle={{ color: '#1890ff' }}
              prefix={<BookOpen size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('knowledgeBase.troubleshooting')}
              value={stats.troubleshooting}
              valueStyle={{ color: '#ff4d4f' }}
              prefix={<BookOpen size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('knowledgeBase.network')}
              value={stats.network}
              valueStyle={{ color: '#52c41a' }}
              prefix={<BookOpen size={16} />}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};
