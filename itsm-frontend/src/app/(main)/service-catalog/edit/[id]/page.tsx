'use client';

import React, { useEffect, useState } from 'react';
import { Card, Form, Input, InputNumber, Switch, Button, App, Spin } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { useI18n } from '@/lib/i18n';

const { TextArea } = Input;

export default function EditServicePage() {
  const params = useParams();
  const router = useRouter();
  const { message: appMessage } = App.useApp();
  const { t } = useI18n();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);

  const serviceId = params.id as string;

  useEffect(() => {
    const loadService = async () => {
      try {
        const data = await ServiceCatalogApi.getService(serviceId);
        form.setFieldsValue({
          name: data.name,
          category: data.category,
          description: data.shortDescription || data.fullDescription,
          deliveryTime: data.availability?.responseTime,
          status: data.status === 'published' ? 'enabled' : 'disabled',
        });
      } catch (error) {
        appMessage.error(t('common.getFailed'));
        router.push('/service-catalog');
      } finally {
        setInitialLoading(false);
      }
    };
    loadService();
  }, [serviceId, form, appMessage, router, t]);

  const handleSubmit = async (values: any) => {
    try {
      setLoading(true);
      await ServiceCatalogApi.updateService(serviceId, {
        name: values.name,
        category: values.category,
        shortDescription: values.description,
        availability: { responseTime: values.deliveryTime },
        status: values.status ? 'published' : 'disabled',
      });
      appMessage.success(t('common.saveSuccess'));
      router.push('/service-catalog');
    } catch (error) {
      appMessage.error(t('common.saveFailed'));
    } finally {
      setLoading(false);
    }
  };

  if (initialLoading) {
    return <Spin className="flex justify-center py-12" />;
  }

  return (
    <div className="p-6 max-w-2xl">
      <Card title={t('service.editService')}>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item label={t('service.name')} name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label={t('service.category')} name="category">
            <Input />
          </Form.Item>
          <Form.Item label={t('service.description')} name="description">
            <TextArea rows={4} />
          </Form.Item>
          <Form.Item label={t('service.deliveryTime')} name="deliveryTime">
            <InputNumber min={1} className="w-full" />
          </Form.Item>
          <Form.Item label={t('service.status')} name="status" valuePropName="checked">
            <Switch checkedChildren="enabled" unCheckedChildren="disabled" />
          </Form.Item>
          <Form.Item>
            <div className="flex gap-4">
              <Button type="primary" htmlType="submit" loading={loading}>
                {t('common.save')}
              </Button>
              <Button onClick={() => router.push('/service-catalog')}>
                {t('common.cancel')}
              </Button>
            </div>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}