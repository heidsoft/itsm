'use client';

import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Typography, Button, Space } from 'antd';
import { Bug, CheckCircle, Clock, AlertTriangle, Plus } from 'lucide-react';
import { useRouter } from 'next/navigation';
import ProblemList from '@/components/problem/ProblemList';

const { Title, Text } = Typography;

export default function ProblemListPage() {
  const router = useRouter();
  const [stats, setStats] = useState({
    total: 0,
    open: 0,
    inProgress: 0,
    resolved: 0,
  });

  // Fetch stats
  const fetchStats = async () => {
    try {
      // Mock data - in real app would call API
      setStats({
        total: 28,
        open: 8,
        inProgress: 12,
        resolved: 8,
      });
    } catch (error) {
      console.error('Failed to fetch problem stats:', error);
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
            问题管理
          </Title>
          <Text type="secondary">
            识别、分析和消除事件发生的根本原因
          </Text>
        </div>
        <Button
          type="primary"
          icon={<Plus className="w-4 h-4" />}
          size="large"
          onClick={() => router.push('/problems/new')}
        >
          新建问题
        </Button>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="总问题数"
              value={stats.total}
              prefix={<Bug className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="待处理"
              value={stats.open}
              prefix={<AlertTriangle className="text-red-500 mr-2" />}
              valueStyle={{ color: '#ff4d4f' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="调查中"
              value={stats.inProgress}
              prefix={<Clock className="text-orange-500 mr-2" />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="已解决"
              value={stats.resolved}
              prefix={<CheckCircle className="text-green-500 mr-2" />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 问题列表 */}
      <ProblemList />
    </div>
  );
}
