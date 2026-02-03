'use client';

import React from 'react';
import { Card, Col, Row, Statistic } from 'antd';
import { HardDrive, UserCog, ShieldCheck } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

interface ServiceCatalogStatsProps {
  stats: {
    total: number;
    cloudServices: number;
    accountServices: number;
    securityServices: number;
  };
}

export const ServiceCatalogStats: React.FC<ServiceCatalogStatsProps> = ({ stats }) => {
  const { t } = useI18n();
  return (
    <div className="mb-6">
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
            <Statistic
              title={t('serviceCatalog.totalServices')}
              value={stats.total}
              prefix={<HardDrive size={16} className="text-blue-500" />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
            <Statistic
              title={t('serviceCatalog.cloudResources')}
              value={stats.cloudServices}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<HardDrive size={16} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
            <Statistic
              title={t('serviceCatalog.accountPermissions')}
              value={stats.accountServices}
              styles={{ content: { color: '#722ed1' } }}
              prefix={<UserCog size={16} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
            <Statistic
              title={t('serviceCatalog.securityServices')}
              value={stats.securityServices}
              styles={{ content: { color: '#52c41a' } }}
              prefix={<ShieldCheck size={16} />}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};
