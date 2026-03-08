'use client';

import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Typography, Button, message } from 'antd';
import { GitPullRequest, CheckCircle, Clock, XCircle, Plus } from 'lucide-react';
import { useRouter } from 'next/navigation';
import ChangeList from '@/components/change/ChangeList';
import { ChangeApi } from '@/lib/api/change-api';

const { Title, Text } = Typography;

export default function ChangesPage() {
  const router = useRouter();
  const [stats, setStats] = useState({
    total: 0,
    pending: 0,
    inProgress: 0,
    completed: 0,
  });

  // Fetch stats
  const fetchStats = async () => {
    try {
      const stats = await ChangeApi.getChangeStats();
      setStats({
        total: stats.total || 0,
        pending: stats.pending || 0,
        inProgress: stats.in_progress || 0,
        completed: stats.completed || 0,
      });
    } catch (error) {
      console.error('Failed to fetch change stats:', error);
      message.error('获取变更统计失败，请稍后重试');
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      {/* 页面头部 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <Title level={2} style={{ marginBottom: 0 }}>
            变更管理
          </Title>
          <Text type="secondary">
            管理IT基础架构和服务的变更请求，最小化变更风险
          </Text>
        </div>
        <Button
          type="primary"
          icon={<Plus className="w-4 h-4" />}
          size="large"
          onClick={() => router.push('/changes/new')}
        >
          新建变更
        </Button>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="总变更数"
              value={stats.total}
              prefix={<GitPullRequest className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="待审批"
              value={stats.pending}
              prefix={<Clock className="text-orange-500 mr-2" />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="进行中"
              value={stats.inProgress}
              prefix={<GitPullRequest className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="已完成"
              value={stats.completed}
              prefix={<CheckCircle className="text-green-500 mr-2" />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 变更列表 */}
      <ChangeList showHeader={false} />
    </div>
  );
}
