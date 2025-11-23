'use client';

import React from 'react';
import Link from 'next/link';
import { Card, Tag, Button, Typography, Rate } from 'antd';
import { HardDrive, UserCog, ShieldCheck, Clock, ArrowRight } from 'lucide-react';
import { ServiceCatalog } from '@/lib/api/service-catalog-api';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;

const categoryIcons = {
  云资源服务: HardDrive,
  账号与权限: UserCog,
  安全服务: ShieldCheck,
};

interface ServiceItemCardProps {
  catalog: ServiceCatalog;
}

export const ServiceItemCard: React.FC<ServiceItemCardProps> = ({ catalog }) => {
  const { t } = useI18n();
  const IconComponent = categoryIcons[catalog.category] || HardDrive;

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case '高':
        return 'red';
      case '中':
        return 'orange';
      case '低':
        return 'green';
      default:
        return 'default';
    }
  };

  return (
    <Card
      style={{ height: '100%' }}
      styles={{
        body: {
          padding: 24,
        },
      }}
    >
      <div style={{ display: 'flex', alignItems: 'flex-start', marginBottom: 16 }}>
        <div
          style={{
            width: 48,
            height: 48,
            borderRadius: 8,
            backgroundColor: '#e6f7ff',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginRight: 16,
          }}
        >
          <IconComponent size={24} style={{ color: '#1890ff' }} />
        </div>
        <div style={{ flex: 1 }}>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'flex-start',
            }}
          >
            <Title level={4} style={{ margin: 0, fontSize: 16 }}>
              {catalog.name}
            </Title>
            <Tag color={getPriorityColor(catalog.priority)} style={{ margin: 0 }}>
              {catalog.priority}
            </Tag>
          </div>
          <Text type='secondary' style={{ fontSize: 12, marginTop: 4, display: 'block' }}>
            {catalog.category}
          </Text>
        </div>
      </div>

      <Text style={{ marginBottom: 16, display: 'block', minHeight: 40 }}>
        {catalog.description}
      </Text>

      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginTop: 16,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <Clock size={14} style={{ marginRight: 4, color: '#666' }} />
          <Text type='secondary' style={{ fontSize: 12 }}>
            {catalog.sla_time || catalog.estimated_time}
          </Text>
        </div>
        <div>
          <Rate disabled defaultValue={catalog.rating || 4.5} count={5} style={{ fontSize: 12 }} />
        </div>
      </div>

      <div
        style={{
          marginTop: 16,
          paddingTop: 16,
          borderTop: '1px solid #f0f0f0',
        }}
      >
        <Link href={`/service-catalog/request/${catalog.id}`} style={{ textDecoration: 'none' }}>
          <Button type='primary' block icon={<ArrowRight size={16} />}>
            {t('serviceCatalog.applyService')}
          </Button>
        </Link>
      </div>
    </Card>
  );
};
