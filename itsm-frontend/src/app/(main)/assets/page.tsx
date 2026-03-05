'use client';

import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Typography, Button, Space } from 'antd';
import { Package, CheckCircle, Clock, AlertTriangle, Plus } from 'lucide-react';
import { useRouter } from 'next/navigation';
import AssetList from '@/components/asset/AssetList';

const { Title, Text } = Typography;

export default function AssetsPage() {
  const router = useRouter();
  const [stats, setStats] = useState({
    totalAssets: 0,
    inUse: 0,
    available: 0,
    maintenance: 0,
  });

  // Fetch stats
  const fetchStats = async () => {
    try {
      // Mock data - in real app would call API
      setStats({
        totalAssets: 256,
        inUse: 180,
        available: 52,
        maintenance: 24,
      });
    } catch (error) {
      console.error('Failed to fetch asset stats:', error);
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
          <Title level={2} style={{ marginBottom: 4 }}>
            资产管理
          </Title>
          <Text type="secondary">
            管理企业IT资产，包括硬件、软件、云资源和许可证
          </Text>
        </div>
        <Button
          type="primary"
          icon={<Plus className="w-4 h-4" />}
          size="large"
          onClick={() => router.push('/assets/new')}
        >
          新增资产
        </Button>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="资产总数"
              value={stats.totalAssets}
              prefix={<Package className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="使用中"
              value={stats.inUse}
              prefix={<CheckCircle className="text-green-500 mr-2" />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="可用"
              value={stats.available}
              prefix={<Package className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="维护中"
              value={stats.maintenance}
              prefix={<Clock className="text-orange-500 mr-2" />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 资产列表 */}
      <AssetList />
    </div>
  );
}
