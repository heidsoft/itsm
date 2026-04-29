'use client';

import React, { useEffect, useState } from 'react';
import { Card, Descriptions, Tag, Button, Row, Col, App, Spin, Empty } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import type { ServiceItem } from '@/types/service-catalog';
import { useI18n } from '@/lib/i18n';

export default function ServiceDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { message } = App.useApp();
  const { t } = useI18n();
  const [service, setService] = useState<ServiceItem | null>(null);
  const [loading, setLoading] = useState(true);

  const serviceId = params.id as string;

  useEffect(() => {
    const loadService = async () => {
      try {
        setLoading(true);
        const data = await ServiceCatalogApi.getService(serviceId);
        setService(data);
      } catch (error) {
        message.error(t('common.getFailed'));
        router.push('/service-catalog');
      } finally {
        setLoading(false);
      }
    };
    loadService();
  }, [serviceId, message, router, t]);

  if (loading) {
    return <Spin className="flex justify-center py-12" />;
  }

  if (!service) {
    return <Empty description={t('service.notFound')} />;
  }

  return (
    <div className="p-6">
      <Card>
        <Descriptions title={service.name} bordered column={2}>
          <Descriptions.Item label={t('service.category')}>
            {service.category}
          </Descriptions.Item>
          <Descriptions.Item label={t('service.status')}>
            <Tag color={service.status === 'published' ? 'green' : 'default'}>
              {service.status}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('service.description')} span={2}>
            {service.shortDescription || service.fullDescription}
          </Descriptions.Item>
          {service.availability?.responseTime && (
            <Descriptions.Item label={t('service.deliveryTime')}>
              {service.availability.responseTime}
            </Descriptions.Item>
          )}
        </Descriptions>

        <div className="mt-6 flex gap-4">
          <Button type="primary" onClick={() => router.push(`/service-catalog/request/${serviceId}`)}>
            {t('service.request')}
          </Button>
          <Button onClick={() => router.push('/service-catalog')}>
            {t('common.back')}
          </Button>
        </div>
      </Card>
    </div>
  );
}
