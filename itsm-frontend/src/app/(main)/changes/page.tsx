'use client';

import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Button, message } from 'antd';
import { GitPullRequest, CheckCircle, Clock, Plus } from 'lucide-react';
import { useRouter } from 'next/navigation';
import ChangeList from '@/components/change/ChangeList';
import { ChangeApi } from '@/lib/api/change-api';
import { useI18n } from '@/lib/i18n/useI18n';
import { PageContainer } from '@/components/layout/PageContainer';

export default function ChangesPage() {
  const router = useRouter();
  const { t } = useI18n();
  const [stats, setStats] = useState({
    total: 0,
    pending: 0,
    inProgress: 0,
    completed: 0,
  });

  const fetchStats = async () => {
    try {
      const stats = await ChangeApi.getChangeStats();
      setStats({
        total: stats.total || 0,
        pending: stats.pending || 0,
        inProgress: stats.inProgress || 0,
        completed: stats.completed || 0,
      });
    } catch (error) {
      console.error('Failed to fetch change stats:', error);
      message.error(t('changes.getStatsFailed') || '获取变更统计失败，请稍后重试');
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  const statsContent = (
    <Row gutter={[16, 16]} className="mb-6">
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="总变更数"
            value={stats.total}
            prefix={<GitPullRequest className="text-blue-500 mr-2" />}
            styles={{ content: { color: '#1890ff' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="待审批"
            value={stats.pending}
            prefix={<Clock className="text-orange-500 mr-2" />}
            styles={{ content: { color: '#fa8c16' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="进行中"
            value={stats.inProgress}
            prefix={<GitPullRequest className="text-blue-500 mr-2" />}
            styles={{ content: { color: '#1890ff' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="已完成"
            value={stats.completed}
            prefix={<CheckCircle className="text-green-500 mr-2" />}
            styles={{ content: { color: '#52c41a' } }}
          />
        </Card>
      </Col>
    </Row>
  );

  return (
    <div className="p-6 min-h-screen" style={{ backgroundColor: 'var(--color-bg-secondary, #f9fafb)' }}>
      <PageContainer
        title="变更管理"
        description="管理IT基础架构和服务的变更请求，最小化变更风险"
        extra={
          <Button
            type="primary"
            icon={<Plus className="w-4 h-4" />}
            size="large"
            onClick={() => router.push('/changes/new')}
          >
            新建变更
          </Button>
        }
        showStats
        stats={statsContent}
      >
        <ChangeList showHeader={false} />
      </PageContainer>
    </div>
  );
}
