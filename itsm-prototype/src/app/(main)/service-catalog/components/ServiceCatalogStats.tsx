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
    <div style={{ marginBottom: 24 }}>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('serviceCatalog.totalServices')}
              value={stats.total}
              prefix={<HardDrive size={16} style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('serviceCatalog.cloudResources')}
              value={stats.cloudServices}
              valueStyle={{ color: '#1890ff' }}
              prefix={<HardDrive size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('serviceCatalog.accountPermissions')}
              value={stats.accountServices}
              valueStyle={{ color: '#722ed1' }}
              prefix={<UserCog size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('serviceCatalog.securityServices')}
              value={stats.securityServices}
              valueStyle={{ color: '#52c41a' }}
              prefix={<ShieldCheck size={16} />}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};
