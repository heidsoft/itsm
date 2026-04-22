'use client';

import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Button, message } from 'antd';
import { Bug, CheckCircle, Clock, AlertTriangle, Plus } from 'lucide-react';
import { useRouter } from 'next/navigation';
import ProblemList from '@/components/problem/ProblemList';
import { ProblemApi } from '@/lib/api/problem-api';
import { useI18n } from '@/lib/i18n/useI18n';
import { PageContainer } from '@/components/layout/PageContainer';

export default function ProblemListPage() {
  const router = useRouter();
  const { t } = useI18n();
  const [stats, setStats] = useState({
    total: 0,
    open: 0,
    inProgress: 0,
    resolved: 0,
  });

  const fetchStats = async () => {
    try {
      const stats = await ProblemApi.getProblemStats();
      setStats({
        total: stats.total || 0,
        open: stats.open || 0,
        inProgress: stats.in_progress || 0,
        resolved: stats.resolved || 0,
      });
    } catch (error) {
      console.error('Failed to fetch problem stats:', error);
      message.error(t('problems.getStatsFailed') || '获取问题统计失败，请稍后重试');
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
            title="总问题数"
            value={stats.total}
            prefix={<Bug className="text-blue-500 mr-2" />}
            styles={{ content: { color: '#1890ff' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="待处理"
            value={stats.open}
            prefix={<AlertTriangle className="text-red-500 mr-2" />}
            styles={{ content: { color: '#ff4d4f' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="调查中"
            value={stats.inProgress}
            prefix={<Clock className="text-orange-500 mr-2" />}
            styles={{ content: { color: '#fa8c16' } }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card className="rounded-lg shadow-sm">
          <Statistic
            title="已解决"
            value={stats.resolved}
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
        title="问题管理"
        description="识别、分析和消除事件发生的根本原因"
        extra={
          <Button
            type="primary"
            icon={<Plus className="w-4 h-4" />}
            size="large"
            onClick={() => router.push('/problems/new')}
          >
            新建问题
          </Button>
        }
        showStats
        stats={statsContent}
      >
        <ProblemList showHeader={false} />
      </PageContainer>
    </div>
  );
}
