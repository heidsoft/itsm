'use client';

import React from 'react';
import Link from 'next/link';
import { Card, Tag, Button, Typography, Rate } from 'antd';
import { HardDrive, UserCog, ShieldCheck, Clock, ArrowRight } from 'lucide-react';
import { ServiceItem, ServiceCategory } from '@/types/service-catalog';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;

const categoryIcons: Record<string, typeof HardDrive> = {
  云资源服务: HardDrive,
  账号与权限: UserCog,
  安全服务: ShieldCheck,
  [ServiceCategory.IT_SERVICE]: HardDrive,
  [ServiceCategory.BUSINESS_SERVICE]: UserCog,
  [ServiceCategory.SUPPORT_SERVICE]: ShieldCheck,
};

interface ServiceItemCardProps {
  catalog: ServiceItem & {
    priority?: string;
    shortDescription?: string;
    sla_time?: string;
    estimated_time?: string;
    rating?: number;
  };
}

export const ServiceItemCard: React.FC<ServiceItemCardProps> = ({ catalog }) => {
  const { t } = useI18n();
  const categoryKey = String(catalog.category);
  const IconComponent = categoryIcons[categoryKey] || HardDrive;

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
      className="h-full rounded-lg shadow-sm border border-gray-200"
      variant="borderless"
      styles={{
        body: {
          padding: 24,
        },
      }}
    >
      <div className="flex items-start mb-4">
        <div className="w-12 h-12 rounded-lg bg-blue-50 flex items-center justify-center mr-4">
          <IconComponent size={24} className="text-blue-500" />
        </div>
        <div className="flex-1">
          <div className="flex justify-between items-start">
            <Title level={4} className="!m-0 !text-base">
              {catalog.name}
            </Title>
            <Tag color={getPriorityColor(catalog.priority || '')} className="!m-0">
              {catalog.priority || '—'}
            </Tag>
          </div>
          <Text type='secondary' className="!text-xs mt-1 block">
            {catalog.category}
          </Text>
        </div>
      </div>

      <Text className="mb-4 block min-h-[40px]">
        {catalog.shortDescription || (catalog as any).description || ''}
      </Text>

      <div className="flex justify-between items-center mt-4">
        <div className="flex items-center">
          <Clock size={14} className="mr-1 text-gray-500" />
          <Text type='secondary' className="!text-xs">
            {catalog.sla_time || catalog.estimated_time || ''}
          </Text>
        </div>
        <div>
          <Rate disabled defaultValue={catalog.rating || 4.5} count={5} className="!text-xs" />
        </div>
      </div>

      <div className="mt-4 pt-4 border-t border-gray-100">
        <Link href={`/service-catalog/request/${catalog.id}`} className="no-underline">
          <Button type='primary' block icon={<ArrowRight size={16} />}>
            {t('serviceCatalog.applyService')}
          </Button>
        </Link>
      </div>
    </Card>
  );
};
