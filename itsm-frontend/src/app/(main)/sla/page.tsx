'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Card, Row, Col, Statistic, Button, Space, Tabs, Spin } from 'antd';
import { useI18n } from '@/lib/i18n/useI18n';
import { SLAApi } from '@/lib/api/sla-api';

// 兼容旧前端 snake_case 和新前端 camelCase
interface SLAStatsResponse {
  totalDefinitions?: number;
  activeDefinitions?: number;
  totalViolations?: number;
  openViolations?: number;
  overallComplianceRate?: number;
}

interface SLAStats {
  totalDefinitions: number;
  activeDefinitions: number;
  totalViolations: number;
  openViolations: number;
  overallComplianceRate: number;
}

// 工具函数：提取兼容字段值
function getSLAStatsValue(data: SLAStatsResponse, camelKey: keyof SLAStatsResponse, snakeKey: string): number {
  return (data[camelKey] ?? data[snakeKey as keyof SLAStatsResponse] ?? 0) as number;
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

      // 映射后端 snake_case 字段到前端 camelCase
      const response = data as unknown as SLAStatsResponse;
      setStats({
        totalDefinitions: getSLAStatsValue(response, 'totalDefinitions', 'total_definitions'),
        activeDefinitions: getSLAStatsValue(response, 'activeDefinitions', 'active_definitions'),
        totalViolations: getSLAStatsValue(response, 'totalViolations', 'total_violations'),
        openViolations: getSLAStatsValue(response, 'openViolations', 'open_violations'),
        overallComplianceRate: getSLAStatsValue(response, 'overallComplianceRate', 'overall_compliance_rate'),
      });
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
          <Row gutter={[16, 16]} className="mb-6" align="stretch">
            <Col xs={24} sm={12} lg={6} className="flex">
              <Card className="h-full w-full">
                <Statistic
                  title="总SLA数量"
                  value={stats.totalDefinitions}
                  styles={{ content: { color: '#1890ff' } }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6} className="flex">
              <Card className="h-full w-full">
                <Statistic
                  title="生效中"
                  value={stats.activeDefinitions}
                  styles={{ content: { color: '#52c41a' } }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6} className="flex">
              <Card className="h-full w-full">
                <Statistic
                  title="违规数量"
                  value={stats.openViolations}
                  styles={{ content: { color: '#ff4d4f' } }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6} className="flex">
              <Card className="h-full w-full">
                <Statistic
                  title="合规率"
                  value={stats.overallComplianceRate}
                  suffix="%"
                  precision={1}
                  styles={{
                    content: { color: stats.overallComplianceRate >= 95 ? '#52c41a' : '#faad14' },
                  }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6} className="flex">
              <Card className="h-full w-full">
                <Statistic
                  title="总违规"
                  value={stats.totalViolations}
                  styles={{ content: { color: '#ff4d4f', fontSize: 14 } }}
                />
              </Card>
            </Col>
          </Row>

          <Card title="快速导航" className="mb-6">
            <Space wrap>
              <Button type="primary" onClick={() => router.push('/sla-dashboard')}>
                SLA仪表盘
              </Button>
              <Button onClick={() => router.push('/sla-monitor')}>SLA监控</Button>
              <Button onClick={() => router.push('/sla/definitions')}>SLA定义</Button>
              <Button onClick={() => router.push('/admin/sla-templates')}>
                预置SLA模板
              </Button>
              <Button onClick={() => router.push('/admin/escalation-matrices')}>
                升级矩阵
              </Button>
            </Space>
          </Card>
        </div>
      ),
    },
    {
      key: 'monitor',
      label: 'SLA监控',
      children: (
        <div className="space-y-4">
          <Card title="实时SLA监控">
            <p className="text-gray-500">监控正在运行的SLA实例和违规情况</p>
            <Row gutter={[16, 16]} className="mt-4" align="stretch">
              <Col xs={24} sm={12} lg={6} className="flex">
                <Card className="h-full w-full">
                  <Statistic
                    title="运行中实例"
                    value={0}
                    styles={{ content: { color: '#52c41a' } }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6} className="flex">
                <Card className="h-full w-full">
                  <Statistic
                    title="即将超时"
                    value={0}
                    styles={{ content: { color: '#faad14' } }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6} className="flex">
                <Card className="h-full w-full">
                  <Statistic title="已超时" value={0} styles={{ content: { color: '#f5222d' } }} />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6} className="flex">
                <Card className="h-full w-full">
                  <Statistic
                    title="完成率"
                    value={'0%'}
                    suffix=""
                    styles={{ content: { color: '#1890ff' } }}
                  />
                </Card>
              </Col>
            </Row>
          </Card>
        </div>
      ),
    },
    {
      key: 'definitions',
      label: 'SLA定义',
      children: (
        <Card title="SLA定义管理">
          <Button
            type="primary"
            className="mb-4"
            onClick={() => router.push('/admin/sla-definitions')}
          >
            前往SLA定义管理
          </Button>
          <p>点击上方按钮前往SLA定义管理页面创建和管理SLA定义。</p>
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
