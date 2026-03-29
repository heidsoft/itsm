'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Card, Row, Col, Statistic, Button, Space, Tabs, Spin } from 'antd';
import { useI18n } from '@/lib/i18n/useI18n';
import { SLAApi } from '@/lib/api/sla-api';

interface SLAStats {
  totalDefinitions: number;
  activeDefinitions: number;
  totalViolations: number;
  openViolations: number;
  overallComplianceRate: number;
}

export default function SLAPage() {
  const router = useRouter();
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<SLAStats>({
    totalDefinitions: 0,
    activeDefinitions: 0,
    totalViolations: 0,
    openViolations: 0,
    overallComplianceRate: 0,
  });

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const data = await SLAApi.getSLAStats();
      setStats(data);
    } catch (error) {
      console.error('Failed to load SLA stats:', error);
    } finally {
      setLoading(false);
    }
  };

  const tabItems = [
    {
      key: 'dashboard',
      label: 'SLA仪表盘',
      children: (
        <div>
          <Row gutter={[16, 16]} className="mb-6">
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title="总SLA数量"
                  value={stats.totalDefinitions}
                  valueStyle={{ color: '#1890ff' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title="生效中"
                  value={stats.activeDefinitions}
                  valueStyle={{ color: '#52c41a' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title="违规数量"
                  value={stats.openViolations}
                  valueStyle={{ color: '#ff4d4f' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title="合规率"
                  value={stats.overallComplianceRate}
                  suffix="%"
                  precision={1}
                  valueStyle={{ color: stats.overallComplianceRate >= 95 ? '#52c41a' : '#faad14' }}
                />
                <Statistic
                  title="总违规"
                  value={stats.totalViolations}
                  valueStyle={{ color: '#ff4d4f', fontSize: 14 }}
                />
              </Card>
            </Col>
          </Row>

          <Card title="快速导航" className="mb-6">
            <Space wrap>
              <Button type="primary" onClick={() => router.push('/sla-dashboard')}>
                SLA仪表盘
              </Button>
              <Button onClick={() => router.push('/sla-monitor')}>
                SLA监控
              </Button>
              <Button onClick={() => router.push('/sla/definitions')}>
                SLA定义
              </Button>
            </Space>
          </Card>
        </div>
      ),
    },
    {
      key: 'definitions',
      label: 'SLA定义',
      children: (
        <Card title="SLA定义管理">
          <Button type="primary" className="mb-4">
            新建SLA
          </Button>
          <p>暂无SLA定义，请创建第一个SLA定义。</p>
        </Card>
      ),
    },
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">SLA服务级别管理</h1>
      <Tabs items={tabItems} defaultActiveKey="dashboard" />
    </div>
  );
}
